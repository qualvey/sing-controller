// JSONC 编辑器支持：语法高亮（Lezer grammar）+ lint + 格式化 + 解析
// 格式化引擎用微软 jsonc-parser —— 与 VSCode 格式化 JSON/JSONC 完全同款：
// 支持 // 与 /* */ 注释、尾逗号，格式化后原样保留注释。
import { LRLanguage, LanguageSupport, indentNodeProp, continuedIndent, foldNodeProp, foldInside } from '@codemirror/language'
import { styleTags, tags as t } from '@lezer/highlight'
import { linter } from '@codemirror/lint'
import type { Diagnostic } from '@codemirror/lint'
import type { EditorView } from '@codemirror/view'
import { applyEdits, format as formatJsonc, getNodeValue, parseTree, printParseErrorCode } from 'jsonc-parser'
import type { ParseError } from 'jsonc-parser'
import { parser } from './jsonc.grammar'

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
        'Object Array': foldInside
      })
    ]
  }),
  languageData: {
    closeBrackets: { brackets: ['[', '{', '"'] },
    indentOnInput: /^\s*[\}\]]$/
  }
})

export function jsonc(): LanguageSupport {
  return new LanguageSupport(jsoncLanguage)
}

/** lint：jsonc-parser 语法诊断；注释/尾逗号不报错，真正错误信息与 VSCode 一致 */
export function jsoncLinter() {
  return linter((view: EditorView): Diagnostic[] => {
    const errors: ParseError[] = []
    parseTree(view.state.doc.toString(), errors, { allowTrailingComma: true })
    return errors.map((e) => ({
      from: e.offset,
      to: e.offset + Math.max(e.length, 1),
      severity: 'error',
      message: printParseErrorCode(e.error)
    }))
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
