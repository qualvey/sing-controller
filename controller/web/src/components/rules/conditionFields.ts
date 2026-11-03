// 规则匹配条件（键值对）编辑器的共享元数据与工具函数。
// 路由规则与 DNS 规则都复用这套数据结构。

export type ConditionFieldType =
  | 'string-list'
  | 'uint-list'
  | 'int-list'
  | 'bool'
  | 'string'
  | 'select'
  | 'json'

export interface ConditionField {
  key: string
  label: string
  group: string
  type: ConditionFieldType
  /** 候选值；fixed 为 true 时表示完整枚举 */
  options?: string[]
  /** 动态选项来源（如 rule-set）：由对话框加载后通过 optionSources 传入 */
  optionsSource?: string
  /** 固定枚举：值渲染为下拉框（单选或多选由字段类型决定） */
  fixed?: boolean
  placeholder?: string
  /** 常用字段：始终保留一行，不可删除 */
  common?: boolean
}

export type ConditionValue = boolean | string | string[]

export interface ConditionRow {
  id: number
  key: string
  value: ConditionValue
}

let rowSeed = 0

export function nextRowId() {
  rowSeed += 1
  return rowSeed
}

export function isListField(field: ConditionField) {
  return field.type === 'string-list' || field.type === 'uint-list' || field.type === 'int-list'
}

/** 固定枚举字段（值用下拉框渲染） */
export function isEnumField(field: ConditionField) {
  return (
    (!!field.options?.length || !!field.optionsSource) &&
    (field.fixed === true || field.type === 'select')
  )
}

export function emptyFieldValue(field?: ConditionField): ConditionValue {
  if (!field) return ''
  if (field.type === 'bool') return false
  if (field.type === 'string' || field.type === 'select' || field.type === 'json') return ''
  return []
}

export function normalizeFieldValue(field: ConditionField, raw: unknown): ConditionValue {
  if (field.type === 'bool') return raw === true
  if (field.type === 'json') {
    if (raw == null || raw === '') return ''
    return typeof raw === 'string' ? raw : JSON.stringify(raw, null, 2)
  }
  if (field.type === 'string' || field.type === 'select') {
    return raw == null ? '' : String(raw)
  }
  return raw == null ? [] : Array.isArray(raw) ? raw.map(String) : [String(raw)]
}

export function hasFieldValue(field: ConditionField, value: ConditionValue): boolean {
  if (field.type === 'bool') return value === true
  if (isListField(field)) return Array.isArray(value) && value.length > 0
  return typeof value === 'string' && value.trim() !== ''
}

export function makeRow(field: ConditionField, raw?: unknown): ConditionRow {
  return {
    id: nextRowId(),
    key: field.key,
    value:
      raw === undefined ? emptyFieldValue(field) : normalizeFieldValue(field, raw),
  }
}

/**
 * 根据规则对象生成条件行。
 * 常用字段（inbound / user）始终保留；其余字段仅在规则中存在值时出现。
 */
export function createConditionRows(
  fields: ConditionField[],
  record?: Record<string, unknown>,
): ConditionRow[] {
  const rows: ConditionRow[] = []

  for (const field of fields) {
    if (!field.common) continue
    rows.push(makeRow(field, record && field.key in record ? record[field.key] : undefined))
  }

  if (!record) return rows

  for (const field of fields) {
    if (field.common) continue
    if (!(field.key in record)) continue
    const value = normalizeFieldValue(field, record[field.key])
    if (!hasFieldValue(field, value)) continue
    rows.push(makeRow(field, record[field.key]))
  }

  return rows
}
