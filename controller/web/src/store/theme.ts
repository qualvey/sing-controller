import { defineStore } from 'pinia'
import { ref } from 'vue'

export type ThemeMode = 'light' | 'dark'

export const useThemeStore = defineStore('theme', () => {
  const mode = ref<ThemeMode>('dark')

  const apply = (m: ThemeMode) => {
    mode.value = m
    document.documentElement.classList.toggle('dark', m === 'dark')
    try {
      localStorage.setItem('theme', m)
    } catch {
      // 隐私模式等场景忽略
    }
  }

  // 启动时恢复：localStorage 优先，默认暗色（sing-box 工具风格）
  const init = () => {
    let saved: ThemeMode | null = null
    try {
      const v = localStorage.getItem('theme')
      if (v === 'light' || v === 'dark') saved = v
    } catch {
      // ignore
    }
    apply(saved ?? 'dark')
  }

  const toggle = () => apply(mode.value === 'dark' ? 'light' : 'dark')

  return { mode, apply, init, toggle }
})
