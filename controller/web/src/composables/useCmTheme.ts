import { Compartment } from '@codemirror/state'
import { oneDark } from '@codemirror/theme-one-dark'
import { EditorView } from '@codemirror/view'
import { watch } from 'vue'
import { useThemeStore } from '@/store/theme'

export const editorUiTheme = (dark: boolean) =>
  EditorView.theme(
    dark
      ? {
          '&': {
            height: '100%',
            fontSize: '13px',
          },
          '.cm-scroller': {
            fontFamily:
              "ui-monospace, 'JetBrains Mono', 'Cascadia Code', 'Fira Code', Consolas, monospace",
            fontVariantLigatures: 'normal',
            lineHeight: '1.7',
          },
          '.cm-gutters': {
            backgroundColor: '#1e1e1f',
            color: '#636d83',
            borderRight: '1px solid rgba(255, 255, 255, 0.08)',
          },
          '.cm-activeLineGutter': {
            backgroundColor: 'rgba(255, 255, 255, 0.06)',
            color: '#abb2bf',
          },
          '.cm-activeLine': {
            backgroundColor: 'rgba(255, 255, 255, 0.03)',
          },
          '.cm-panels': {
            backgroundColor: '#1e1e1f',
            color: '#abb2bf',
          },
          '.cm-panels-top': {
            borderBottom: '1px solid rgba(255, 255, 255, 0.08)',
          },
          '.cm-search': {
            backgroundColor: '#1e1e1f',
            color: '#abb2bf',
            padding: '6px 10px',
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
          },
          '.cm-search input, .cm-search select': {
            backgroundColor: '#2b2b2d',
            color: '#abb2bf',
            border: '1px solid rgba(255, 255, 255, 0.15)',
            borderRadius: '4px',
            padding: '3px 8px',
            outline: 'none',
          },
          '.cm-search input:focus': {
            borderColor: '#61afef',
          },
          '.cm-search button': {
            backgroundColor: '#2b2b2d',
            color: '#abb2bf',
            border: '1px solid rgba(255, 255, 255, 0.15)',
            borderRadius: '4px',
            padding: '3px 10px',
            cursor: 'pointer',
          },
          '.cm-search button:hover': {
            backgroundColor: '#3e4451',
          },
          '.cm-search .cm-textfield': {
            backgroundColor: '#2b2b2d',
          },
          '&.cm-focused .cm-matchingBracket, &.cm-focused .cm-nonmatchingBracket': {
            backgroundColor: '#3e4451',
          },
          '.cm-cursor': {
            borderLeftColor: '#abb2bf',
          },
          '.cm-selectionBackground, &.cm-focused .cm-selectionBackground, ::selection': {
            backgroundColor: 'rgba(62, 68, 81, 0.6)',
          },
          '.cm-foldGutter span': {
            cursor: 'pointer',
            opacity: '0.65',
            transition: 'opacity 0.15s ease, color 0.15s ease',
          },
          '.cm-foldGutter span:hover': {
            opacity: '1',
            color: '#61afef',
          },
          '.cm-foldPlaceholder': {
            backgroundColor: 'rgba(255, 255, 255, 0.1)',
            border: '1px solid rgba(255, 255, 255, 0.2)',
            color: '#abb2bf',
            borderRadius: '3px',
            padding: '0 5px',
            margin: '0 2px',
            cursor: 'pointer',
          },

          // --- 现代 VS Code 风格补全浮窗 (Dark) ---
          '.cm-tooltip': {
            backgroundColor: '#1f2024',
            color: '#abb2bf',
            border: '1px solid rgba(255, 255, 255, 0.12)',
            borderRadius: '8px',
            boxShadow: '0 12px 32px -4px rgba(0, 0, 0, 0.6), 0 4px 12px -2px rgba(0, 0, 0, 0.4)',
          },
          '.cm-tooltip.cm-tooltip-autocomplete': {
            fontFamily: "ui-monospace, 'JetBrains Mono', 'Cascadia Code', 'Fira Code', Consolas, monospace",
            fontSize: '12px',
            padding: '4px',
            minWidth: '280px',
            maxWidth: '560px',
            overflow: 'hidden',
          },
          '.cm-tooltip.cm-tooltip-autocomplete > ul': {
            fontFamily: "ui-monospace, 'JetBrains Mono', 'Cascadia Code', 'Fira Code', Consolas, monospace",
            maxHeight: '230px',
            padding: '0',
            margin: '0',
          },
          '.cm-tooltip.cm-tooltip-autocomplete > ul > li': {
            padding: '4px 8px',
            borderRadius: '5px',
            margin: '1px 0',
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            lineHeight: '1.45',
            cursor: 'pointer',
            transition: 'background-color 0.1s ease',
          },
          '.cm-tooltip-autocomplete > ul > li[aria-selected]': {
            backgroundColor: '#094771',
            color: '#ffffff',
          },
          '.cm-completionLabel': {
            fontWeight: '500',
            color: '#e5eaf3',
          },
          '.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionLabel': {
            color: '#ffffff',
            fontWeight: '600',
          },
          '.cm-completionMatchedText': {
            color: '#61afef',
            fontWeight: '700',
            textDecoration: 'none',
          },
          '.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionMatchedText': {
            color: '#93c5fd',
            textDecoration: 'underline',
          },
          '.cm-completionDetail': {
            fontStyle: 'normal',
            fontSize: '11px',
            marginLeft: 'auto',
            paddingLeft: '12px',
            color: '#858b98',
            fontFamily: "ui-monospace, 'JetBrains Mono', Consolas, monospace",
          },
          '.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionDetail': {
            color: '#cbd5e1',
          },
          // 补全类型徽章（解决 □ 方块问题，赋予专业 IDE 标识）
          '.cm-completionIcon': {
            width: '18px',
            height: '18px',
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: '10px',
            fontWeight: '700',
            borderRadius: '4px',
            marginRight: '2px',
            opacity: '1',
            boxSizing: 'border-box',
            flexShrink: '0',
          },
          '.cm-completionIcon:after': {
            content: '""',
          },
          '.cm-completionIcon-property': {
            backgroundColor: 'rgba(97, 175, 239, 0.16)',
            color: '#61afef',
            border: '1px solid rgba(97, 175, 239, 0.35)',
          },
          '.cm-completionIcon-property:after': {
            content: '"p"',
            fontFamily: "ui-monospace, 'JetBrains Mono', monospace",
            textTransform: 'uppercase',
          },
          '.cm-completionIcon-enum': {
            backgroundColor: 'rgba(229, 192, 123, 0.16)',
            color: '#e5c07b',
            border: '1px solid rgba(229, 192, 123, 0.35)',
          },
          '.cm-completionIcon-enum:after': {
            content: '"e"',
            fontFamily: "ui-monospace, 'JetBrains Mono', monospace",
            textTransform: 'uppercase',
          },
          '.cm-completionIcon-keyword': {
            backgroundColor: 'rgba(198, 120, 221, 0.16)',
            color: '#c678dd',
            border: '1px solid rgba(198, 120, 221, 0.35)',
          },
          '.cm-completionIcon-keyword:after': {
            content: '"k"',
            fontFamily: "ui-monospace, 'JetBrains Mono', monospace",
            textTransform: 'uppercase',
          },
          '.cm-completionIcon-constant': {
            backgroundColor: 'rgba(209, 154, 102, 0.16)',
            color: '#d19a66',
            border: '1px solid rgba(209, 154, 102, 0.35)',
          },
          '.cm-completionIcon-constant:after': {
            content: '"c"',
            fontFamily: "ui-monospace, 'JetBrains Mono', monospace",
            textTransform: 'uppercase',
          },
          '.cm-completionIcon-type': {
            backgroundColor: 'rgba(86, 182, 194, 0.16)',
            color: '#56b6c2',
            border: '1px solid rgba(86, 182, 194, 0.35)',
          },
          '.cm-completionIcon-type:after': {
            content: '"t"',
            fontFamily: "ui-monospace, 'JetBrains Mono', monospace",
            textTransform: 'uppercase',
          },
          '.cm-tooltip.cm-completionInfo': {
            backgroundColor: '#1f2024',
            color: '#abb2bf',
            border: '1px solid rgba(255, 255, 255, 0.12)',
            borderRadius: '8px',
            boxShadow: '0 12px 32px -4px rgba(0, 0, 0, 0.5)',
            padding: '8px 12px',
            fontSize: '12px',
            lineHeight: '1.5',
            fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif",
            maxWidth: '380px',
          },
        }
      : {
          // --- Light Mode ---
          '&': {
            height: '100%',
            fontSize: '13px',
          },
          '.cm-scroller': {
            fontFamily:
              "ui-monospace, 'JetBrains Mono', 'Cascadia Code', 'Fira Code', Consolas, monospace",
            fontVariantLigatures: 'normal',
            lineHeight: '1.7',
          },
          '.cm-gutters': {
            backgroundColor: '#f8f9fa',
            color: '#909399',
            borderRight: '1px solid rgba(0, 0, 0, 0.08)',
          },
          '.cm-activeLineGutter': {
            backgroundColor: '#edf0f5',
            color: '#606266',
          },
          '.cm-activeLine': {
            backgroundColor: 'rgba(0, 0, 0, 0.02)',
          },
          '.cm-panels': {
            backgroundColor: '#ffffff',
            color: '#303133',
          },
          '.cm-panels-top': {
            borderBottom: '1px solid rgba(0, 0, 0, 0.08)',
          },
          '.cm-search': {
            backgroundColor: '#ffffff',
            color: '#303133',
            padding: '6px 10px',
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
          },
          '.cm-search input, .cm-search select': {
            backgroundColor: '#ffffff',
            color: '#303133',
            border: '1px solid rgba(0, 0, 0, 0.15)',
            borderRadius: '4px',
            padding: '3px 8px',
            outline: 'none',
          },
          '.cm-search input:focus': {
            borderColor: '#3b82f6',
          },
          '.cm-search button': {
            backgroundColor: '#f3f4f6',
            color: '#303133',
            border: '1px solid rgba(0, 0, 0, 0.15)',
            borderRadius: '4px',
            padding: '3px 10px',
            cursor: 'pointer',
          },
          '.cm-search button:hover': {
            backgroundColor: '#e5e7eb',
          },
          '.cm-search .cm-textfield': {
            backgroundColor: '#ffffff',
          },
          '&.cm-focused .cm-matchingBracket, &.cm-focused .cm-nonmatchingBracket': {
            backgroundColor: '#e5e7eb',
          },
          '.cm-cursor': {
            borderLeftColor: '#303133',
          },
          '.cm-selectionBackground, &.cm-focused .cm-selectionBackground, ::selection': {
            backgroundColor: 'rgba(59, 130, 246, 0.2)',
          },
          '.cm-foldGutter span': {
            cursor: 'pointer',
            opacity: '0.65',
            transition: 'opacity 0.15s ease, color 0.15s ease',
          },
          '.cm-foldGutter span:hover': {
            opacity: '1',
            color: '#3b82f6',
          },
          '.cm-foldPlaceholder': {
            backgroundColor: 'rgba(0, 0, 0, 0.05)',
            border: '1px solid rgba(0, 0, 0, 0.15)',
            color: '#606266',
            borderRadius: '3px',
            padding: '0 5px',
            margin: '0 2px',
            cursor: 'pointer',
          },

          // --- 现代 VS Code 风格补全浮窗 (Light) ---
          '.cm-tooltip': {
            backgroundColor: '#ffffff',
            color: '#303133',
            border: '1px solid rgba(0, 0, 0, 0.12)',
            borderRadius: '8px',
            boxShadow: '0 12px 32px -4px rgba(0, 0, 0, 0.15), 0 4px 12px -2px rgba(0, 0, 0, 0.08)',
          },
          '.cm-tooltip.cm-tooltip-autocomplete': {
            fontFamily: "ui-monospace, 'JetBrains Mono', 'Cascadia Code', 'Fira Code', Consolas, monospace",
            fontSize: '12px',
            padding: '4px',
            minWidth: '280px',
            maxWidth: '560px',
            overflow: 'hidden',
          },
          '.cm-tooltip.cm-tooltip-autocomplete > ul': {
            fontFamily: "ui-monospace, 'JetBrains Mono', 'Cascadia Code', 'Fira Code', Consolas, monospace",
            maxHeight: '230px',
            padding: '0',
            margin: '0',
          },
          '.cm-tooltip.cm-tooltip-autocomplete > ul > li': {
            padding: '4px 8px',
            borderRadius: '5px',
            margin: '1px 0',
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            lineHeight: '1.45',
            cursor: 'pointer',
            transition: 'background-color 0.1s ease',
          },
          '.cm-tooltip-autocomplete > ul > li[aria-selected]': {
            backgroundColor: '#0066b8',
            color: '#ffffff',
          },
          '.cm-completionLabel': {
            fontWeight: '500',
            color: '#1f2937',
          },
          '.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionLabel': {
            color: '#ffffff',
            fontWeight: '600',
          },
          '.cm-completionMatchedText': {
            color: '#2563eb',
            fontWeight: '700',
            textDecoration: 'none',
          },
          '.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionMatchedText': {
            color: '#dbeafe',
            textDecoration: 'underline',
          },
          '.cm-completionDetail': {
            fontStyle: 'normal',
            fontSize: '11px',
            marginLeft: 'auto',
            paddingLeft: '12px',
            color: '#6b7280',
            fontFamily: "ui-monospace, 'JetBrains Mono', Consolas, monospace",
          },
          '.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionDetail': {
            color: '#f3f4f6',
          },
          // 补全图标 Badge
          '.cm-completionIcon': {
            width: '18px',
            height: '18px',
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: '10px',
            fontWeight: '700',
            borderRadius: '4px',
            marginRight: '2px',
            opacity: '1',
            boxSizing: 'border-box',
            flexShrink: '0',
          },
          '.cm-completionIcon:after': {
            content: '""',
          },
          '.cm-completionIcon-property': {
            backgroundColor: 'rgba(37, 99, 235, 0.1)',
            color: '#2563eb',
            border: '1px solid rgba(37, 99, 235, 0.25)',
          },
          '.cm-completionIcon-property:after': {
            content: '"p"',
            fontFamily: "ui-monospace, 'JetBrains Mono', monospace",
            textTransform: 'uppercase',
          },
          '.cm-completionIcon-enum': {
            backgroundColor: 'rgba(217, 119, 6, 0.1)',
            color: '#d97706',
            border: '1px solid rgba(217, 119, 6, 0.25)',
          },
          '.cm-completionIcon-enum:after': {
            content: '"e"',
            fontFamily: "ui-monospace, 'JetBrains Mono', monospace",
            textTransform: 'uppercase',
          },
          '.cm-completionIcon-keyword': {
            backgroundColor: 'rgba(147, 51, 234, 0.1)',
            color: '#9333ea',
            border: '1px solid rgba(147, 51, 234, 0.25)',
          },
          '.cm-completionIcon-keyword:after': {
            content: '"k"',
            fontFamily: "ui-monospace, 'JetBrains Mono', monospace",
            textTransform: 'uppercase',
          },
          '.cm-completionIcon-constant': {
            backgroundColor: 'rgba(234, 88, 12, 0.1)',
            color: '#ea580c',
            border: '1px solid rgba(234, 88, 12, 0.25)',
          },
          '.cm-completionIcon-constant:after': {
            content: '"c"',
            fontFamily: "ui-monospace, 'JetBrains Mono', monospace",
            textTransform: 'uppercase',
          },
          '.cm-completionIcon-type': {
            backgroundColor: 'rgba(13, 148, 136, 0.1)',
            color: '#0d9488',
            border: '1px solid rgba(13, 148, 136, 0.25)',
          },
          '.cm-completionIcon-type:after': {
            content: '"t"',
            fontFamily: "ui-monospace, 'JetBrains Mono', monospace",
            textTransform: 'uppercase',
          },
          '.cm-tooltip.cm-completionInfo': {
            backgroundColor: '#ffffff',
            color: '#374151',
            border: '1px solid rgba(0, 0, 0, 0.12)',
            borderRadius: '8px',
            boxShadow: '0 12px 32px -4px rgba(0, 0, 0, 0.15)',
            padding: '8px 12px',
            fontSize: '12px',
            lineHeight: '1.5',
            fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif",
            maxWidth: '380px',
          },
        },
  )

// CodeMirror 语法高亮主题：跟随全局日/夜模式（dark → oneDark，light → 默认亮色）
export function useCmTheme() {
  const themeStore = useThemeStore()
  const comp = new Compartment()

  const extFor = (dark: boolean) => (dark ? [oneDark, editorUiTheme(true)] : [editorUiTheme(false)])

  const ext = comp.of(extFor(themeStore.mode === 'dark'))

  const sync = (view: EditorView) => {
    view.dispatch({ effects: comp.reconfigure(extFor(themeStore.mode === 'dark')) })
  }

  const watchTheme = (getView: () => EditorView | undefined) => {
    watch(
      () => themeStore.mode,
      () => {
        const view = getView()
        if (view) sync(view)
      }
    )
  }

  return { comp, ext, sync, watchTheme }
}
