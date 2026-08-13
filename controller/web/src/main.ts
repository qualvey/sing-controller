import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './style.css'
import App from './App.vue'
import router from './router'
import { useThemeStore } from './stores/theme'

const app = createApp(App)
app.use(createPinia())
app.use(router)
// 主题初始化（html.dark class + localStorage 恢复）
useThemeStore().init()
// 等 router 就绪再挂载：侧边栏激活态需要拿到正确的初始路由
// （否则整页刷新 deep-link 时高亮/指示条丢失）
router.isReady().then(() => app.mount('#app'))
