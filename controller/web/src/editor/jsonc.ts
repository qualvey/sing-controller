// JSONC 编辑器支持：语法高亮（Lezer grammar）+ lint + 格式化 + 解析
// 格式化引擎用微软 jsonc-parser —— 与 VSCode 格式化 JSON/JSONC 完全同款：
// 支持 // 与 /* */ 注释、尾逗号，格式化后原样保留注释。
import { LRLanguage, LanguageSupport, indentNodeProp, continuedIndent, foldNodeProp } from '@codemirror/language'
import { styleTags, tags as t } from '@lezer/highlight'
import { linter } from '@codemirror/lint'
import type { Diagnostic } from '@codemirror/lint'
import type { EditorView } from '@codemirror/view'
import { applyEdits, findNodeAtLocation, format as formatJsonc, getNodeValue, parseTree, printParseErrorCode } from 'jsonc-parser'
import type { JSONPath, Node, ParseError } from 'jsonc-parser'
import { parser } from './jsonc.grammar'
import { getSingBoxValidator } from './schemaValidator'

import { autocompletion } from '@codemirror/autocomplete'
import { hoverTooltip } from '@codemirror/view'
import { jsoncCompletionSource, jsoncHoverTooltip } from './schemaIntellisense'

export const jsoncLanguage = LRLanguage.define({
  name: 'jsonc',
  parser: parser.configure({
    props: [
      styleTags({
        String: t.string,
        Number: t.number,
        PropertyName: t.propertyName,
        True: t.bool,
        False: t.bool,
        Null: t.null,
        LineComment: t.comment,
        BlockComment: t.comment
      }),
      indentNodeProp.add({
        Object: continuedIndent({ except: /^\s*\}/ }),
        Array: continuedIndent({ except: /^\s*\]/ })
      }),
      foldNodeProp.add({
        'Object Array': (node) => {
          if (node.to <= node.from + 2) return null
          return { from: node.from + 1, to: node.to - 1 }
        },
        BlockComment: (node) => {
          if (node.to <= node.from + 4) return null
          return { from: node.from + 2, to: node.to - 2 }
        }
      })
    ]
  }),
  languageData: {
    closeBrackets: { brackets: ['[', '{', '"'] },
    indentOnInput: /^\s*[\}\]]$/,
    commentTokens: { line: '//', block: { open: '/*', close: '*/' } }
  }
})

export function jsonc(): LanguageSupport {
  return new LanguageSupport(jsoncLanguage, [
    autocompletion({
      override: [jsoncCompletionSource],
      defaultKeymap: false
    }),
    hoverTooltip(jsoncHoverTooltip)
  ])
}

export { jsoncCompletionSource, jsoncHoverTooltip }

function parseInstancePath(path: string): JSONPath {
  if (!path || path === '/') return []
  return path
    .slice(1)
    .split('/')
    .map((seg) => {
      const decoded = seg.replace(/~1/g, '/').replace(/~0/g, '~')
      return /^\d+$/.test(decoded) ? Number(decoded) : decoded
    })
}

/** lint：jsonc-parser 语法诊断 + sing-box JSON Schema 语义校验 */
export function jsoncLinter() {
  return linter((view: EditorView): Diagnostic[] => {
    const text = view.state.doc.toString()
    if (!text.trim()) return []

    const parseErrors: ParseError[] = []
    const rootNode = parseTree(text, parseErrors, { allowTrailingComma: true })

    // 1. 语法错误优先报告
    if (parseErrors.length > 0) {
      return parseErrors.map((e) => ({
        from: e.offset,
        to: Math.min(e.offset + Math.max(e.length, 1), text.length),
        severity: 'error',
        message: printParseErrorCode(e.error)
      }))
    }

    if (!rootNode) return []

    // 2. sing-box JSON Schema 语义校验
    const validator = getSingBoxValidator()
    if (!validator) return []

    const jsonValue = getNodeValue(rootNode)
    const valid = validator(jsonValue)
    if (valid || !validator.errors) return []

    const diagnostics: Diagnostic[] = []
    for (const err of validator.errors) {
      const pathSegments = parseInstancePath(err.instancePath)
      let targetNode: Node | undefined = findNodeAtLocation(rootNode, pathSegments)

      // 如果是缺少必填字段，定位到对象内部或属性名
      if (!targetNode && pathSegments.length > 0) {
        targetNode = findNodeAtLocation(rootNode, pathSegments.slice(0, -1))
      }
      if (!targetNode) {
        targetNode = rootNode
      }

      let from = targetNode.offset
      let to = targetNode.offset + targetNode.length

      // 避免跨越整个庞大对象高亮
      if (to - from > 200 && targetNode.children && targetNode.children.length > 0) {
        from = targetNode.children[0].offset
        to = from + Math.min(targetNode.children[0].length, 50)
      }

      diagnostics.push({
        from: Math.max(0, Math.min(from, text.length)),
        to: Math.max(from + 1, Math.min(to, text.length)),
        severity: 'error',
        message: `[Schema] ${err.instancePath || '/'}: ${err.message || '配置不符合 sing-box Schema'}`
      })
    }

    return diagnostics
  })
}

/** 解析 JSONC 文本为对象（保存用）：语法错误返回失败信息，合法则返回解析值（注释被忽略、尾逗号容忍） */
export function parseJsonc(text: string): { ok: true; value: unknown } | { ok: false; message: string } {
  const errors: ParseError[] = []
  const root = parseTree(text, errors, { allowTrailingComma: true })
  if (errors.length) return { ok: false, message: printParseErrorCode(errors[0].error) }
  return { ok: true, value: root ? getNodeValue(root) : null }
}

/** 整文档格式化（VSCode 同款：tabSize=2 空格缩进，保留注释）。返回是否有改动 */
export function formatJsoncDoc(view: EditorView): boolean {
  const text = view.state.doc.toString()
  const edits = formatJsonc(text, undefined, { tabSize: 2, insertSpaces: true })
  if (!edits.length) return false
  view.dispatch({ changes: { from: 0, to: text.length, insert: applyEdits(text, edits) } })
  return true
}
