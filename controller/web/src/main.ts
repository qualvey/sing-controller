import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import './style.css'
import App from './App.vue'
import router from './router'
import { useThemeStore } from './stores/theme'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(ElementPlus, { locale: zhCn })
// 主题初始化（html.dark class + localStorage 恢复）
useThemeStore().init()
// 等 router 就绪再挂载：el-menu 的 default-active 需要拿到正确的初始路由
// （否则整页刷新 deep-link 时高亮/指示条丢失）
router.isReady().then(() => app.mount('#app'))
