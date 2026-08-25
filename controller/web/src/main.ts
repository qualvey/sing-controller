import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './style.css'
import App from './App.vue'
import router from './router'
import { useThemeStore } from './stores/theme'
import '@/helper/dayjs'
import '@/api/http'
import '@/helper/dayjs'
import 'tippy.js/animations/scale.css'
import 'tippy.js/dist/tippy.css'
import { loadFonts } from './assets/load-fonts'
import './assets/main.css'
import { applyCustomThemes, applyKsuTheme } from './helper'
import { i18n } from './i18n'
const isEdge = /Edg\//.test(navigator.userAgent)

if (isEdge) {
  const originalReplaceState = history.replaceState
  history.replaceState = function (...args) {
    if (document.visibilityState === 'hidden') return
    return originalReplaceState.apply(this, args)
  }
}

applyCustomThemes()
applyKsuTheme()
loadFonts()
const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(i18n)

// 主题初始化（html.dark class + localStorage 恢复）
useThemeStore().init()
// 等 router 就绪再挂载：侧边栏激活态需要拿到正确的初始路由
// （否则整页刷新 deep-link 时高亮/指示条丢失）
router.isReady().then(() => app.mount('#app'))
