import { Compartment } from '@codemirror/state'
import { oneDark } from '@codemirror/theme-one-dark'
import type { EditorView } from '@codemirror/view'
import { watch } from 'vue'
import { useThemeStore } from '../stores/theme'

// CodeMirror 语法高亮主题：跟随全局日/夜模式（dark → oneDark，light → 默认亮色）
export function useCmTheme() {
  const themeStore = useThemeStore()
  const comp = new Compartment()

  const extFor = (dark: boolean) => (dark ? oneDark : [])

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
