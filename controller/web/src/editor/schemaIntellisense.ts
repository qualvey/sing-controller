import type { Completion, CompletionContext, CompletionResult } from '@codemirror/autocomplete'
import type { EditorView, Tooltip } from '@codemirror/view'
import { getLocation, getNodeValue, parseTree } from 'jsonc-parser'
import type { Node } from 'jsonc-parser'
import rawSchema from '../../schema.json'

interface SchemaObject {
  $ref?: string
  type?: string | string[]
  properties?: Record<string, SchemaObject>
  items?: SchemaObject | SchemaObject[]
  oneOf?: SchemaObject[]
  anyOf?: SchemaObject[]
  allOf?: SchemaObject[]
  enum?: (string | number | boolean)[]
  const?: string | number | boolean
  description?: string
  default?: unknown
  minimum?: number
  maximum?: number
  examples?: unknown[]
  [key: string]: unknown
}

const schema = rawSchema as SchemaObject

/** 解析 $ref 引用（如 #/$defs/Inbound） */
function resolveRef(ref: string): SchemaObject | null {
  if (!ref.startsWith('#/')) return null
  const parts = ref.slice(2).split('/')
  let cur: any = schema
  for (const part of parts) {
    if (cur && typeof cur === 'object' && part in cur) {
      cur = cur[part]
    } else {
      return null
    }
  }
  return cur as SchemaObject
}

/** 展开 $ref 和 allOf 合并 */
function expandSchema(s: SchemaObject | undefined): SchemaObject | null {
  if (!s) return null
  if (s.$ref) {
    const resolved = resolveRef(s.$ref)
    return resolved ? expandSchema(resolved) : null
  }
  return s
}

/** 根据当前 AST 节点获取同级对象值（用于 discriminator 识别，如 type: 'vless'） */
function getSiblingObjectValue(rootNode: Node, pos: number): Record<string, any> | undefined {
  let target: Node | undefined
  const find = (node: Node) => {
    if (node.offset <= pos && pos <= node.offset + node.length) {
      if (node.type === 'object') {
        target = node
      }
      if (node.children) {
        for (const child of node.children) find(child)
      }
    }
  }
  find(rootNode)
  if (target) {
    return getNodeValue(target) as Record<string, any>
  }
  return undefined
}

/** 获取同级对象中已有的所有属性名 */
function getExistingKeys(rootNode: Node, pos: number): Set<string> {
  const keys = new Set<string>()
  let target: Node | undefined
  const find = (node: Node) => {
    if (node.offset <= pos && pos <= node.offset + node.length) {
      if (node.type === 'object') {
        target = node
      }
      if (node.children) {
        for (const child of node.children) find(child)
      }
    }
  }
  find(rootNode)
  if (target && target.children) {
    for (const prop of target.children) {
      if (prop.type === 'property' && prop.children && prop.children[0]) {
        keys.add(String(getNodeValue(prop.children[0])))
      }
    }
  }
  return keys
}

/** 沿着 JSON Path 解析目标 Schema */
function resolveSchemaAtPath(
  currentSchema: SchemaObject | null,
  path: (string | number)[],
  currentObj?: Record<string, any>
): SchemaObject | null {
  if (!currentSchema) return null
  const s = expandSchema(currentSchema)
  if (!s) return null

  if (path.length === 0) {
    return s
  }

  const [head, ...tail] = path

  // 处理 oneOf / anyOf 变体（多态协议识别）
  const variants = s.oneOf || s.anyOf
  if (variants && variants.length > 0) {
    // 若当前已知 type，尝试匹配指定变体
    if (currentObj && typeof currentObj.type === 'string') {
      for (const v of variants) {
        const expanded = expandSchema(v)
        const typeProp = expanded?.properties?.type
        if (typeProp && (typeProp.const === currentObj.type || typeProp.enum?.includes(currentObj.type))) {
          return resolveSchemaAtPath(expanded, path, currentObj)
        }
      }
    }
    // 否则尝试遍历所有变体寻找命中的属性
    if (typeof head === 'string') {
      for (const v of variants) {
        const expanded = expandSchema(v)
        if (expanded?.properties?.[head]) {
          return resolveSchemaAtPath(expanded.properties[head], tail, currentObj)
        }
      }
    }
  }

  // 数组索引
  if (typeof head === 'number') {
    if (s.items) {
      const itemsSchema = Array.isArray(s.items) ? s.items[head] || s.items[0] : s.items
      return resolveSchemaAtPath(expandSchema(itemsSchema), tail, currentObj)
    }
    return null
  }

  // 对象属性
  if (typeof head === 'string') {
    if (s.properties && s.properties[head]) {
      return resolveSchemaAtPath(expandSchema(s.properties[head]), tail, currentObj)
    }
  }

  return null
}

/** 收集 Schema 支持的所有属性候选 */
function collectPropertyCompletions(
  targetSchema: SchemaObject | null,
  existingKeys: Set<string>,
  currentObj?: Record<string, any>
): Completion[] {
  if (!targetSchema) return []
  const s = expandSchema(targetSchema)
  if (!s) return []

  const results: Map<string, Completion> = new Map()

  const addFromProps = (props: Record<string, SchemaObject>) => {
    for (const [key, propSchema] of Object.entries(props)) {
      if (existingKeys.has(key)) continue
      const expanded = expandSchema(propSchema)
      let typeLabel = typeof expanded?.type === 'string' ? expanded.type : 'any'
      if (expanded?.enum) {
        typeLabel = expanded.enum.map((e) => `"${e}"`).join(' | ')
      } else if (expanded?.$ref) {
        typeLabel = expanded.$ref.replace('#/$defs/', '')
      }

      results.set(key, {
        label: `"${key}"`,
        displayLabel: key,
        type: 'property',
        detail: `: ${typeLabel}`,
        info: expanded?.description || undefined,
        apply: `"${key}": `,
        boost: key === 'type' ? 10 : key === 'tag' ? 8 : 0,
      })
    }
  }

  // 1. 直接 properties
  if (s.properties) {
    addFromProps(s.properties)
  }

  // 2. oneOf / anyOf 变体
  const variants = s.oneOf || s.anyOf
  if (variants) {
    // 如果已知 type，仅添加该 type 专属属性
    if (currentObj && typeof currentObj.type === 'string') {
      for (const v of variants) {
        const exp = expandSchema(v)
        if (exp?.properties?.type && (exp.properties.type.const === currentObj.type || exp.properties.type.enum?.includes(currentObj.type))) {
          if (exp.properties) addFromProps(exp.properties)
          break
        }
      }
    } else {
      // 未知 type，提取所有变体的公共 / 聚合属性
      for (const v of variants) {
        const exp = expandSchema(v)
        if (exp?.properties) addFromProps(exp.properties)
      }
    }
  }

  return Array.from(results.values())
}

/** 收集属性值候选（如枚举 enum / const / boolean） */
function collectValueCompletions(targetSchema: SchemaObject | null, currentObj?: Record<string, any>): Completion[] {
  if (!targetSchema) return []
  const s = expandSchema(targetSchema)
  if (!s) return []

  const list: Completion[] = []

  // 1. 枚举类型
  if (s.enum) {
    for (const val of s.enum) {
      const isStr = typeof val === 'string'
      list.push({
        label: isStr ? `"${val}"` : String(val),
        type: 'enum',
        detail: 'enum',
        apply: isStr ? `"${val}"` : String(val),
      })
    }
  }

  // 2. const 常量
  if (s.const !== undefined) {
    const isStr = typeof s.const === 'string'
    list.push({
      label: isStr ? `"${s.const}"` : String(s.const),
      type: 'constant',
      apply: isStr ? `"${s.const}"` : String(s.const),
    })
  }

  // 3. 布尔值
  if (s.type === 'boolean' || (Array.isArray(s.type) && s.type.includes('boolean'))) {
    list.push({ label: 'true', type: 'keyword', apply: 'true' })
    list.push({ label: 'false', type: 'keyword', apply: 'false' })
  }

  // 4. oneOf / anyOf 变体中的 const（例如 inbounds/outbounds 的 type 字段）
  const variants = s.oneOf || s.anyOf
  if (variants) {
    for (const v of variants) {
      const exp = expandSchema(v)
      if (exp) {
        list.push(...collectValueCompletions(exp, currentObj))
      }
    }
  }

  return list
}

/** CodeMirror 6 补全源 */
export function jsoncCompletionSource(context: CompletionContext): CompletionResult | null {
  const text = context.state.doc.toString()
  const pos = context.pos
  const loc = getLocation(text, pos)

  const rootNode = parseTree(text, [], { allowTrailingComma: true })
  if (!rootNode) return null

  const currentObj = getSiblingObjectValue(rootNode, pos)
  const existingKeys = getExistingKeys(rootNode, pos)

  // 1. 属性名补全 (isAtPropertyKey)
  if (loc.isAtPropertyKey) {
    const parentPath = loc.path.slice(0, -1)
    const parentSchema = resolveSchemaAtPath(schema, parentPath, currentObj)
    const completions = collectPropertyCompletions(parentSchema, existingKeys, currentObj)
    if (completions.length === 0) return null

    // 计算补全替换范围
    const word = context.matchBefore(/[\w"-]*/)
    return {
      from: word ? word.from : pos,
      options: completions,
      validFor: /^[\w"-]*$/,
    }
  }

  // 2. 属性值补全 (枚举、常量、类型)
  const propSchema = resolveSchemaAtPath(schema, loc.path, currentObj)
  const valCompletions = collectValueCompletions(propSchema, currentObj)
  if (valCompletions.length > 0) {
    const word = context.matchBefore(/[\w"-]*/)
    return {
      from: word ? word.from : pos,
      options: valCompletions,
      validFor: /^[\w"-]*$/,
    }
  }

  return null
}

/** CodeMirror 6 悬浮提示 (Hover Tooltip) */
export function jsoncHoverTooltip(view: EditorView, pos: number, side: number): Tooltip | null {
  const text = view.state.doc.toString()
  const rootNode = parseTree(text, [], { allowTrailingComma: true })
  if (!rootNode) return null

  let propNode: Node | undefined
  const find = (node: Node) => {
    if (node.offset <= pos && pos <= node.offset + node.length) {
      if (node.type === 'property' && node.children && node.children[0]) {
        const keyNode = node.children[0]
        if (keyNode.offset <= pos && pos <= keyNode.offset + keyNode.length) {
          propNode = node
        }
      }
      if (node.children) {
        for (const child of node.children) find(child)
      }
    }
  }
  find(rootNode)
  if (!propNode || !propNode.children || !propNode.children[0]) return null

  const keyNode = propNode.children[0]
  const keyName = String(getNodeValue(keyNode))
  const loc = getLocation(text, keyNode.offset)
  const currentObj = getSiblingObjectValue(rootNode, keyNode.offset)
  const propSchema = resolveSchemaAtPath(schema, loc.path, currentObj)

  if (!propSchema && !keyName) return null
  const expanded = propSchema ? expandSchema(propSchema) : null

  const description = expanded?.description || 'sing-box 配置项'
  const typeText = typeof expanded?.type === 'string' ? expanded.type : expanded?.$ref ? expanded.$ref.replace('#/$defs/', '') : 'any'
  const defaultVal = expanded?.default !== undefined ? JSON.stringify(expanded.default) : null

  return {
    pos: keyNode.offset,
    end: keyNode.offset + keyNode.length,
    above: true,
    create() {
      const dom = document.createElement('div')
      dom.className = 'cm-schema-tooltip'
      dom.style.cssText = `
        padding: 8px 12px;
        background: #1e1e2e;
        color: #cdd6f4;
        border: 1px solid rgba(255, 255, 255, 0.12);
        border-radius: 8px;
        font-size: 12px;
        line-height: 1.5;
        max-width: 420px;
        box-shadow: 0 8px 24px rgba(0, 0, 0, 0.35);
        font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      `

      let html = `<div style="display: flex; align-items: center; gap: 8px; margin-bottom: 4px;">
        <span style="font-weight: 700; color: #89b4fa;">${keyName}</span>
        <span style="color: #a6adc8; font-size: 11px;">: ${typeText}</span>
      </div>`

      if (description) {
        html += `<div style="color: #bac2de; margin-bottom: 4px; font-family: sans-serif; font-size: 12px;">${description}</div>`
      }

      if (defaultVal !== null) {
        html += `<div style="font-size: 11px; color: #f9e2af;">默认值: <code style="background: rgba(255,255,255,0.08); padding: 1px 4px; border-radius: 4px;">${defaultVal}</code></div>`
      }

      if (expanded?.enum) {
        html += `<div style="font-size: 11px; color: #fab387; margin-top: 2px;">可选值: ${expanded.enum.map((e) => `<code>${e}</code>`).join(', ')}</div>`
      }

      dom.innerHTML = html
      return { dom }
    },
  }
}
