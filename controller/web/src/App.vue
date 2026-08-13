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
import { showToast } from './helper/toast'
import ConfirmDialogHost from './components/common/ConfirmDialogHost.vue'

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

const menuItems: Array<{ path: string; label: string; icon: Component }> = [
  { path: '/inbounds', label: 'Inbounds', icon: ArrowDownTrayIcon },
  { path: '/proxies', label: 'Proxies', icon: ServerStackIcon },
  { path: '/connections', label: 'Connections', icon: BoltIcon },
  { path: '/logs', label: 'Logs', icon: ClipboardDocumentListIcon },
  { path: '/users', label: 'Users', icon: UserGroupIcon },
  { path: '/outbounds', label: 'Outbounds', icon: ArrowUpTrayIcon },
  { path: '/routes', label: 'Routes', icon: MapIcon },
  { path: '/rule-sets', label: '规则集', icon: BookmarkIcon },
  { path: '/dns', label: 'DNS', icon: GlobeAltIcon },
  { path: '/certificate', label: '证书', icon: ShieldCheckIcon },
  { path: '/diagnostics', label: '诊断', icon: BeakerIcon },
  { path: '/config', label: 'Config', icon: CodeBracketIcon },
  { path: '/settings', label: 'Settings', icon: Cog6ToothIcon }
]

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
  <div class="flex h-screen">
    <!-- 侧边栏：桌面固定 210px；移动端折叠为 64px 图标栏，点击汉堡按钮伸展为完整菜单（抽屉） -->
    <aside
      class="flex shrink-0 flex-col border-r border-[#e4e7ed] bg-white transition-[width] duration-200 dark:border-none dark:bg-[#001529]"
      :class="{
        'fixed inset-y-0 left-0 z-[2100] shadow-[4px_0_16px_rgba(0,0,0,0.25)]': asideOverlay
      }"
      :style="{ width: asideWidth }"
    >
      <div v-if="!isMobile || mobileExpanded" class="whitespace-nowrap px-4 pb-3.5 pt-5 text-lg font-bold tracking-wide text-[#303133] dark:text-white">
        sing-box <span class="text-[#1890ff]">WebUI</span>
      </div>
      <div v-else class="pb-3 pt-[18px] text-center text-sm font-bold text-[#303133] dark:text-white">SB</div>

      <div class="min-h-0 flex-1 overflow-y-auto">
        <nav ref="navRef" class="relative">
          <!-- 激活指示条（zashboard 风格滑动动效） -->
          <div
            aria-hidden="true"
            class="sidebar-tab-indicator pointer-events-none absolute left-0 top-0 z-0 rounded-lg bg-[rgba(24,144,255,0.1)] shadow-[inset_0_0_0_1px_rgba(24,144,255,0.18)] dark:bg-[rgba(255,255,255,0.14)] dark:shadow-[inset_0_0_0_1px_rgba(255,255,255,0.06)]"
            :style="indicatorStyle"
          />
          <div class="flex flex-col gap-0.5 px-1.5 py-1">
            <router-link
              v-for="item in menuItems"
              :key="item.path"
              :to="item.path"
              class="nav-item relative z-[1] flex items-center whitespace-nowrap rounded-lg text-sm text-[#606266] no-underline transition-colors hover:bg-black/[0.06] hover:text-[#303133] dark:text-[#a6b0bf] dark:hover:bg-white/[0.06] dark:hover:text-white"
              :class="[
                menuCollapsed ? 'justify-center gap-0 px-0 py-2.5' : 'justify-start gap-2.5 px-3 py-2.5',
                // active 类供 syncTabIndicator 定位指示条（.nav-item.active），勿删
                { 'active text-[#1890ff] dark:text-white': route.path === item.path }
              ]"
              @click="onNavSelect"
            >
              <component :is="item.icon" class="nav-icon size-[18px] shrink-0" />
              <span v-show="!menuCollapsed">{{ item.label }}</span>
            </router-link>
          </div>
        </nav>
      </div>
    </aside>

    <!-- 移动端展开遮罩：点击收起侧边栏 -->
    <div v-if="asideOverlay" class="fixed inset-0 z-[2090] bg-black/45" @click="mobileExpanded = false" />

    <div id="app-content" class="flex min-w-0 flex-1 flex-col">
      <header class="flex min-h-14 shrink-0 items-center justify-between border-b border-[#e4e7ed] bg-white px-3 dark:border-[#303030] dark:bg-[#1d1e1f]">
        <div class="flex min-w-0 items-center gap-2 overflow-hidden text-[13px] text-[#606266] dark:text-[#cfd3dc]">
          <!-- 移动端：展开侧边栏（自身伸展，非抽屉） -->
          <button v-if="isMobile" class="icon-btn flex size-[30px] shrink-0 items-center justify-center rounded p-0 text-inherit hover:bg-black/[0.06] dark:hover:bg-white/[0.08]" title="展开菜单" @click="mobileExpanded = true">
            <Bars3Icon class="h-[18px] w-[18px]" />
          </button>
          <span class="dot inline-block size-2.5 shrink-0 rounded-full bg-[#1890ff]" />
          <span>sing-box-controller</span>
          <template v-if="!isMobile">
            <span class="divider-v h-3.5 w-px shrink-0 bg-[#e4e7ed] dark:bg-[#303030]" />
            <span>Inbounds: {{ statusStore.status?.inbounds ?? '—' }}</span>
            <span class="divider-v h-3.5 w-px shrink-0 bg-[#e4e7ed] dark:bg-[#303030]" />
            <span>Outbounds: {{ statusStore.status?.outbounds ?? '—' }}</span>
            <span class="divider-v h-3.5 w-px shrink-0 bg-[#e4e7ed] dark:bg-[#303030]" />
            <span>Rules: {{ statusStore.status?.rules ?? '—' }}</span>
            <span class="divider-v h-3.5 w-px shrink-0 bg-[#e4e7ed] dark:bg-[#303030]" />
            <span>min_port: {{ statusStore.status?.min_port ?? '—' }}</span>
          </template>
        </div>
        <div class="flex min-w-0 items-center gap-2">
          <div
            v-if="statusStore.status?.config_path && !isMobile"
            class="max-w-[40vw] truncate text-xs text-[#909399]"
            :title="statusStore.status.config_path"
          >
            主配置: {{ statusStore.status.config_path }}
          </div>
          <button
            class="ml-3 flex size-8 shrink-0 cursor-pointer items-center justify-center rounded-full border border-[#dcdfe6] bg-white text-base transition-colors hover:border-[#409eff] dark:border-[#4c4d4f] dark:bg-[#1d1e1f]"
            :title="themeStore.mode === 'dark' ? '切换到日间模式' : '切换到夜间模式'"
            @click="themeStore.toggle()"
          >
            {{ themeStore.mode === 'dark' ? '☀️' : '🌙' }}
          </button>
        </div>
      </header>

      <main class="flex-1 overflow-auto bg-[#f5f7fa] p-4 max-md:p-2.5 dark:bg-[#141414]">
        <router-view />
      </main>
    </div>

    <!-- 全局确认对话框宿主（替代 ElMessageBox，见 helper/confirmDialog.ts） -->
    <ConfirmDialogHost />

    <!-- 全局悬浮：重载 sing-box（左下角） -->
    <button
      class="fixed bottom-[18px] left-[18px] z-[2000] flex size-11 cursor-pointer items-center justify-center rounded-full border-none bg-[#409eff] text-[22px] leading-none text-white shadow-[0_4px_12px_rgba(64,158,255,0.4)] transition-colors hover:bg-[#66b1ff] disabled:cursor-not-allowed disabled:opacity-70"
      :class="{ 'animate-spin': reloading }"
      :disabled="reloading"
      title="重载 sing-box 配置（SIGHUP）"
      @click="reloadSingBox"
    >
      ↻
    </button>
  </div>
</template>

<style scoped>
/* 指示条滑动/尺寸过渡：弹性曲线是刻意设计（zashboard 风格），
   用 Tailwind arbitrary 写太长，保留为 scoped 规则；其余样式已 Tailwind 化 */
.sidebar-tab-indicator {
  transition:
    transform 0.55s cubic-bezier(0.22, 1.5, 0.36, 1),
    width 0.32s cubic-bezier(0.32, 0.72, 0, 1),
    height 0.32s cubic-bezier(0.32, 0.72, 0, 1),
    opacity 0.15s ease-out;
}
@media (prefers-reduced-motion: reduce) {
  .sidebar-tab-indicator {
    transition: none;
  }
}
</style>
