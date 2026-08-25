<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import type { Component } from 'vue'
import {
  ArrowDownTrayIcon,
  ArrowUpTrayIcon,
  Bars3Icon,
  BeakerIcon,
  BoltIcon,
  BookmarkIcon,
  ClipboardDocumentListIcon,
  CodeBracketIcon,
  Cog6ToothIcon,
  GlobeAltIcon,
  MapIcon,
  ServerStackIcon,
  ShieldCheckIcon,
  UserGroupIcon
} from '@heroicons/vue/24/outline'
import { api } from './api'
import { useStatusStore } from './stores/status'
import { useThemeStore } from './stores/theme'
import { useRuntimeStore } from './stores/runtime'
import { useLogsStore } from './stores/logs'
import { showToast } from '@/helper/toast'
import { isSidebarCollapsed } from '@/stores/settings'
import {  isMiddleScreen } from '@/helper/utils'

import ConfirmDialogHost from './components/common/ConfirmDialogHost.vue'
import SideBar from '@/components/sidebar/SideBar.vue'

const sidebarLayoutCollapsed = ref(isSidebarCollapsed.value)
const syncSidebarLayoutState = () => {
  sidebarLayoutCollapsed.value = isSidebarCollapsed.value
}

watch(isSidebarCollapsed, (value) => {
  if (value) {
    sidebarLayoutCollapsed.value = true
  }
})

watch(
  isMiddleScreen,
  (value) => {
    if (!value) {
      sidebarLayoutCollapsed.value = isSidebarCollapsed.value
    }
  },
  { immediate: true },
)

const route = useRoute()
const statusStore = useStatusStore()
const themeStore = useThemeStore()
const runtimeStore = useRuntimeStore()
const logsStore = useLogsStore()
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
// 折叠态（图标栏）：仅移动端收起时；桌面始终完整菜单
const menuCollapsed = computed(() => isMobile.value && !mobileExpanded.value)
// 移动端展开态：覆盖内容区的抽屉样式
const asideOverlay = computed(() => isMobile.value && mobileExpanded.value)

// 导航激活指示条（zashboard 风格：弹性滑动过渡）
// 定位：指示条与菜单项在同一滚动内容流（nav-inner）内，transform = 激活项 rect - 容器 rect，
// 差值在滚动内容内恒定（滚动时两者同步偏移），天然对齐、无需滚动监听。
const navRef = ref<HTMLElement>()
const indicatorStyle = ref({
  height: '0px',
  opacity: '0',
  transform: 'translate3d(0, 0, 0)',
  width: '0px'
})
let navResizeObserver: ResizeObserver | undefined

const syncTabIndicator = () => {
  const nav = navRef.value
  if (!nav) return
  const activeTab = nav.querySelector<HTMLElement>('.nav-item.active')
  if (!activeTab) return
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
  },
  { immediate: true }
)
watch(
  [menuCollapsed, mobileExpanded],
  async () => {
    await nextTick()
    syncTabIndicator()
  }
)

onMounted(() => {
  statusStore.refresh()
  runtimeStore.start()
  logsStore.start()
  syncTabIndicator()
  navResizeObserver = new ResizeObserver(() => syncTabIndicator())
  if (navRef.value) navResizeObserver.observe(navRef.value)
})
onBeforeUnmount(() => {
  mql.removeEventListener('change', onMqlChange)
  navResizeObserver?.disconnect()
  runtimeStore.stop()
})

// 重载 sing-box（左下角全局悬浮按钮，触发方式由 settings.reload 配置）
const reloadSingBox = async () => {
  reloading.value = true
  try {
    await api.reload()
    showToast('已触发 sing-box 重载', 'success')
  } catch (e) {
    showToast((e as Error).message || '重载失败', 'error')
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
  <div
    ref="app"
    id="app-content"
    class="bg-base-200 home-page flex size-full"
    :class="sidebarLayoutCollapsed ? 'sidebar-collapsed' : 'sidebar-expanded'"
  >
    <!-- 侧边栏：桌面固定 210px；移动端折叠为 64px 图标栏，点击汉堡按钮伸展为完整菜单（抽屉） -->
    <!-- <div
      v-if="!isMiddleScreen"
      class="relative z-40 flex-none overflow-visible transition-none"
      :class="sidebarLayoutCollapsed ? 'w-18' : 'w-64'"
    > -->
    <RouterView />
    <div
      ref="toast"
      class="toast-sm toast toast-end toast-top z-100000 max-w-80 text-sm md:max-w-96 md:translate-y-8"
    />
      <!-- <SideBar
        class="absolute inset-y-0 left-0"
        @transitionend="syncSidebarLayoutState"
      /> -->
    </div>

  <!-- </div> -->
</template>

