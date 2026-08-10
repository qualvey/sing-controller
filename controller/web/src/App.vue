<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  Connection,
  Iphone,
  Guide,
  CollectionTag,
  Monitor,
  CircleCheck,
  Setting,
  Document,
  DataAnalysis,
  Expand
} from '@element-plus/icons-vue'
import { api } from './api'
import { useStatusStore } from './stores/status'
import { useThemeStore } from './stores/theme'

const route = useRoute()
const statusStore = useStatusStore()
const themeStore = useThemeStore()
const reloading = ref(false)
// 移动端：侧边栏展开状态（折叠图标栏 ↔ 完整菜单）
const mobileExpanded = ref(false)

// 响应式：窄屏（<768px）自动折叠侧边栏为图标栏
const isMobile = ref(window.matchMedia('(max-width: 768px)').matches)
const mql = window.matchMedia('(max-width: 768px)')
const onMqlChange = (e: MediaQueryListEvent) => {
  isMobile.value = e.matches
  if (!e.matches) mobileExpanded.value = false
}
mql.addEventListener('change', onMqlChange)

const asideWidth = computed(() => (isMobile.value ? (mobileExpanded.value ? '210px' : '64px') : '210px'))
// 移动端折叠态：菜单 collapse（图标栏）；展开时恢复完整菜单
const menuCollapse = computed(() => isMobile.value && !mobileExpanded.value)

const menuItems = [
  { path: '/inbounds', label: 'Inbounds', icon: Connection },
  { path: '/outbounds', label: 'Outbounds', icon: Iphone },
  { path: '/routes', label: 'Routes', icon: Guide },
  { path: '/rule-sets', label: '规则集', icon: CollectionTag },
  { path: '/dns', label: 'DNS', icon: Monitor },
  { path: '/certificate', label: '证书', icon: CircleCheck },
  { path: '/diagnostics', label: '诊断', icon: DataAnalysis },
  { path: '/config', label: 'Config', icon: Document },
  { path: '/settings', label: 'Settings', icon: Setting }
]

onMounted(() => {
  statusStore.refresh()
  syncTabIndicator()
  navResizeObserver = new ResizeObserver(() => syncTabIndicator())
  if (navRef.value) navResizeObserver.observe(navRef.value)
})
onBeforeUnmount(() => {
  mql.removeEventListener('change', onMqlChange)
  navResizeObserver?.disconnect()
})

// 导航激活指示条（zashboard 风格：弹性滑动过渡）
const navRef = ref<HTMLElement>()
const indicatorReady = ref(false)
const indicatorStyle = ref({
  height: '0px',
  opacity: '0',
  transform: 'translate3d(0, 0, 0)',
  width: '0px'
})
let navResizeObserver: ResizeObserver | undefined

let indicatorRetried = false
const syncTabIndicator = () => {
  const nav = navRef.value
  if (!nav) return
  // 在原生容器内查找激活菜单项（el-menu 是组件实例，无 querySelector）
  const activeTab = nav.querySelector<HTMLElement>('.el-menu-item.is-active')
  if (!activeTab) {
    // 时序边缘（deep-link 整页加载时 active 可能晚一拍）：延迟重试一次
    if (!indicatorRetried) {
      indicatorRetried = true
      setTimeout(() => {
        indicatorRetried = false
        syncTabIndicator()
      }, 80)
    }
    return
  }
  const navRect = nav.getBoundingClientRect()
  const tabRect = activeTab.getBoundingClientRect()
  indicatorStyle.value = {
    height: `${tabRect.height}px`,
    opacity: '1',
    transform: `translate3d(${tabRect.left - navRect.left}px, ${tabRect.top - navRect.top}px, 0)`,
    width: `${tabRect.width}px`
  }
}

// 路由切换 / 折叠状态变化时重算指示条位置
watch(
  () => route.path,
  async () => {
    await nextTick()
    syncTabIndicator()
    requestAnimationFrame(() => {
      indicatorReady.value = true
    })
  },
  { immediate: true }
)
watch(
  [menuCollapse, mobileExpanded],
  async () => {
    await nextTick()
    syncTabIndicator()
  }
)

// 重载 sing-box（左下角全局悬浮按钮，触发方式由 settings.reload 配置）
const reloadSingBox = async () => {
  reloading.value = true
  try {
    await api.reload()
    ElMessage.success('已触发 sing-box 重载')
  } catch (e) {
    ElMessage.error((e as Error).message || '重载失败')
  } finally {
    reloading.value = false
  }
}

const onNavSelect = () => {
  // 移动端：选中路由后自动折叠回图标栏
  if (isMobile.value) mobileExpanded.value = false
}
</script>

<template>
  <el-container class="app-root">
    <!-- 侧边栏：桌面固定 210px；移动端折叠为 64px 图标栏，点击汉堡按钮伸展为完整菜单 -->
    <el-aside :width="asideWidth" class="app-aside" :class="{ 'aside-expanded': isMobile && mobileExpanded }">
      <div v-if="!isMobile || mobileExpanded" class="logo">sing-box <span class="logo-sub">WebUI</span></div>
      <div v-else class="logo logo-mini">SB</div>
      <div ref="navRef" class="relative min-h-0 flex-1">
        <!-- 激活指示条（zashboard 风格滑动动效） -->
        <div
          aria-hidden="true"
          class="sidebar-tab-indicator"
          :class="{ 'sidebar-tab-indicator-ready': indicatorReady }"
          :style="indicatorStyle"
        />
        <el-menu
          :default-active="route.path"
          router
          class="app-menu"
          :collapse="menuCollapse"
          :collapse-transition="false"
          @select="onNavSelect"
        >
          <el-menu-item v-for="item in menuItems" :key="item.path" :index="item.path">
            <el-icon><component :is="item.icon" /></el-icon>
            <template #title>{{ item.label }}</template>
          </el-menu-item>
        </el-menu>
      </div>
    </el-aside>

    <!-- 移动端展开遮罩：点击收起侧边栏 -->
    <div v-if="isMobile && mobileExpanded" class="aside-mask" @click="mobileExpanded = false" />

    <el-container class="app-body">
      <el-header class="app-header">
        <div class="header-left">
          <!-- 移动端：展开侧边栏（自身伸展，非抽屉） -->
          <button v-if="isMobile" class="icon-btn" @click="mobileExpanded = true">
            <el-icon :size="18"><Expand /></el-icon>
          </button>
          <span class="dot" />
          <span>sing-box-controller</span>
          <template v-if="!isMobile">
            <el-divider direction="vertical" />
            <span>Inbounds: {{ statusStore.status?.inbounds ?? '—' }}</span>
            <el-divider direction="vertical" />
            <span>Outbounds: {{ statusStore.status?.outbounds ?? '—' }}</span>
            <el-divider direction="vertical" />
            <span>Rules: {{ statusStore.status?.rules ?? '—' }}</span>
            <el-divider direction="vertical" />
            <span>min_port: {{ statusStore.status?.min_port ?? '—' }}</span>
          </template>
        </div>
        <div v-if="statusStore.status?.config_path && !isMobile" class="header-right" :title="statusStore.status.config_path">
          主配置: {{ statusStore.status.config_path }}
        </div>
        <el-tooltip :content="themeStore.mode === 'dark' ? '切换到日间模式' : '切换到夜间模式'" placement="bottom">
          <button class="theme-toggle" @click="themeStore.toggle()">
            {{ themeStore.mode === 'dark' ? '☀️' : '🌙' }}
          </button>
        </el-tooltip>
      </el-header>

      <el-main class="app-main">
        <router-view />
      </el-main>
    </el-container>

    <!-- 移动端：侧边栏伸展遮罩点击收起（已由 aside-mask 处理） -->

    <!-- 全局悬浮：重载 sing-box（左下角） -->
    <el-tooltip content="重载 sing-box 配置（SIGHUP）" placement="right">
      <button class="reload-fab" :class="{ spinning: reloading }" :disabled="reloading" @click="reloadSingBox">
        ↻
      </button>
    </el-tooltip>
  </el-container>
</template>

<style scoped>
.app-root {
  height: 100vh;
}
.app-aside {
  background: #001529;
  display: flex;
  flex-direction: column;
  transition: width 0.2s;
}
.logo {
  color: #fff;
  font-size: 18px;
  font-weight: 700;
  padding: 20px 16px 14px;
  letter-spacing: 1px;
  white-space: nowrap;
}
.logo-mini {
  font-size: 14px;
  padding: 18px 0 12px;
  text-align: center;
}
.app-menu {
  border-right: none;
  background: transparent;
  flex: 1;
  position: relative;
  z-index: 1;
}
.app-menu :deep(.el-menu-item) {
  color: #a6b0bf;
  position: relative;
  z-index: 1;
  margin: 2px 6px;
  border-radius: 8px;
}
.app-menu :deep(.el-menu-item:hover) {
  background: rgba(255, 255, 255, 0.06);
  color: #fff;
}
.app-menu :deep(.el-menu-item.is-active) {
  color: #fff;
  background: transparent;
}
/* 折叠态：去掉边距，图标垂直居中 */
.app-menu.el-menu--collapse :deep(.el-menu-item) {
  margin: 2px 0;
  border-radius: 8px;
}

/* zashboard 风格：导航激活指示条（弹性滑动过渡） */
.sidebar-tab-indicator {
  position: absolute;
  left: 6px;
  top: 2px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.14);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.06);
  will-change: transform, width, height;
  pointer-events: none;
  z-index: 0;
}
.sidebar-tab-indicator-ready {
  transition:
    transform 0.55s cubic-bezier(0.22, 1.5, 0.36, 1),
    width 0.32s cubic-bezier(0.32, 0.72, 0, 1),
    height 0.32s cubic-bezier(0.32, 0.72, 0, 1),
    opacity 0.15s ease-out;
}
@media (prefers-reduced-motion: reduce) {
  .sidebar-tab-indicator-ready {
    transition: none;
  }
}
/* 折叠态菜单项居中 */
.app-menu:not(.el-menu--collapse) {
  width: 100%;
}
.app-menu.el-menu--collapse {
  width: 64px;
}.app-body {
  min-width: 0;
}
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  padding: 0 12px;
}
html.dark .app-header {
  background: #1d1e1f;
  border-bottom-color: #303030;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #606266;
  min-width: 0;
  overflow: hidden;
}
html.dark .header-left {
  color: #cfd3dc;
}
.header-right {
  font-size: 12px;
  color: #909399;
  max-width: 40%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
/* 移动端图标按钮 */
.icon-btn {
  width: 30px;
  height: 30px;
  border: none;
  background: transparent;
  color: inherit;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  padding: 0;
}
.icon-btn:hover {
  background: rgba(0, 0, 0, 0.06);
}
html.dark .icon-btn:hover {
  background: rgba(255, 255, 255, 0.08);
}

/* 主题切换按钮 */
.theme-toggle {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 1px solid #dcdfe6;
  background: #fff;
  font-size: 16px;
  cursor: pointer;
  margin-left: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s, border-color 0.2s;
  flex-shrink: 0;
}
.theme-toggle:hover {
  border-color: #409eff;
}
html.dark .theme-toggle {
  background: #1d1e1f;
  border-color: #4c4d4f;
}
.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #1890ff;
  display: inline-block;
  flex-shrink: 0;
}
.app-main {
  background: #f5f7fa;
  padding: 16px;
}
html.dark .app-main {
  background: #141414;
}
/* 窄屏 main 减少内边距 */
@media (max-width: 768px) {
  .app-main {
    padding: 10px;
  }
}

/* 侧边栏伸展态：覆盖内容区，遮罩可点收 */
.app-aside.aside-expanded {
  position: fixed;
  left: 0;
  top: 0;
  bottom: 0;
  z-index: 2100;
  box-shadow: 4px 0 16px rgba(0, 0, 0, 0.25);
}
.aside-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  z-index: 2090;
}

/* 抽屉导航（已移除，保留深色菜单通用样式） */

/* 左下角悬浮重载按钮 */
.reload-fab {
  position: fixed;
  left: 18px;
  bottom: 18px;
  z-index: 2000;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: none;
  background: #409eff;
  color: #fff;
  font-size: 22px;
  line-height: 1;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.4);
  transition: background 0.2s, transform 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}
.reload-fab:hover {
  background: #66b1ff;
}
.reload-fab:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}
.reload-fab.spinning {
  animation: fab-spin 1s linear infinite;
}
@keyframes fab-spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
