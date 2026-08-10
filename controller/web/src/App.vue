<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
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
const drawerVisible = ref(false)

// 响应式：窄屏（<768px）自动折叠侧边栏为图标栏
const isMobile = ref(window.matchMedia('(max-width: 768px)').matches)
const mql = window.matchMedia('(max-width: 768px)')
const onMqlChange = (e: MediaQueryListEvent) => {
  isMobile.value = e.matches
  if (!e.matches) drawerVisible.value = false
}
mql.addEventListener('change', onMqlChange)

const asideWidth = computed(() => (isMobile.value ? '64px' : '210px'))

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
})
onBeforeUnmount(() => {
  mql.removeEventListener('change', onMqlChange)
})

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

const onDrawerSelect = () => {
  drawerVisible.value = false
}
</script>

<template>
  <el-container class="app-root">
    <!-- 桌面端：固定侧边栏（窄屏自动折叠为图标栏） -->
    <el-aside :width="asideWidth" class="app-aside">
      <div v-if="!isMobile" class="logo">sing-box <span class="logo-sub">WebUI</span></div>
      <div v-else class="logo logo-mini">SB</div>
      <el-menu :default-active="route.path" router class="app-menu" :collapse="isMobile" :collapse-transition="false">
        <el-menu-item v-for="item in menuItems" :key="item.path" :index="item.path">
          <el-icon><component :is="item.icon" /></el-icon>
          <template #title>{{ item.label }}</template>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container class="app-body">
      <el-header class="app-header">
        <div class="header-left">
          <!-- 移动端：展开抽屉菜单 -->
          <button v-if="isMobile" class="icon-btn" @click="drawerVisible = true">
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

    <!-- 移动端：抽屉导航 -->
    <el-drawer v-model="drawerVisible" direction="ltr" size="220px" :with-header="false" class="nav-drawer">
      <div class="drawer-logo">sing-box <span class="logo-sub">WebUI</span></div>
      <el-menu :default-active="route.path" router class="app-menu drawer-menu" @select="onDrawerSelect">
        <el-menu-item v-for="item in menuItems" :key="item.path" :index="item.path">
          <el-icon><component :is="item.icon" /></el-icon>
          <template #title>{{ item.label }}</template>
        </el-menu-item>
      </el-menu>
    </el-drawer>

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
}
.app-menu :deep(.el-menu-item) {
  color: #a6b0bf;
}
.app-menu :deep(.el-menu-item:hover) {
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
}
.app-menu :deep(.el-menu-item.is-active) {
  color: #fff;
  background: #1890ff;
}
/* 折叠态菜单项居中 */
.app-menu:not(.el-menu--collapse) {
  width: 100%;
}
.app-menu.el-menu--collapse {
  width: 64px;
}
.app-body {
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

/* 抽屉导航 */
.drawer-logo {
  color: #fff;
  font-size: 16px;
  font-weight: 700;
  padding: 8px 4px 12px;
  letter-spacing: 1px;
}
.nav-drawer :deep(.el-drawer__body) {
  background: #001529;
  padding: 8px;
}
.drawer-menu {
  --el-menu-bg-color: transparent;
}
.drawer-menu :deep(.el-menu-item) {
  color: #a6b0bf;
}
.drawer-menu :deep(.el-menu-item:hover) {
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
}
.drawer-menu :deep(.el-menu-item.is-active) {
  color: #fff;
  background: #1890ff;
}

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
