<template>
  <div class="core-control-card base-container w-full p-4 sm:p-5 rounded-3xl border border-base-content/5 shadow-sm">
    <!-- 顶部标题栏 -->
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-3">
        <!-- 终端控制台图标 -->
        <div class="size-11 rounded-2xl bg-base-300/80 border border-base-content/10 flex items-center justify-center text-base-content shadow-sm">
          <svg
            class="size-5.5"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <rect
              x="3"
              y="4"
              width="18"
              height="16"
              rx="3.5"
            />
            <line
              x1="8"
              y1="12"
              x2="16"
              y2="12"
            />
          </svg>
        </div>
        <div class="text-base sm:text-lg font-bold text-base-content tracking-tight">
          {{ t('coreControl.title') || 'sing-box 控制台' }}
        </div>
      </div>

      <!-- 运行状态胶囊 -->
      <div
        class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold border select-none transition-colors"
        :class="statusCapsuleClass"
      >
        <span
          class="size-2 rounded-full"
          :class="statusDotClass"
        />
        <span>{{ statusText }}</span>
      </div>
    </div>

    <!-- 核心操作面板（卡片列表） -->
    <div class="bg-base-200/60 rounded-2xl border border-base-content/5 overflow-hidden shadow-sm">
      <!-- 动作 1: 重载配置 -->
      <div
        class="flex items-center justify-between px-5 py-4 cursor-pointer hover:bg-base-content/[0.04] active:bg-base-content/[0.08] transition-colors select-none group"
        :class="{ 'opacity-70 pointer-events-none': isReloading }"
        @click="handleReloadConfig"
      >
        <span class="text-[15px] sm:text-base font-semibold text-base-content/90 group-hover:text-base-content transition-colors">
          {{ t('coreControl.reloadConfig') || '重载配置' }}
        </span>
        <div class="text-sky-400 p-1 flex items-center justify-center">
          <svg
            class="size-6 transition-transform"
            :class="{ 'animate-spin': isReloading }"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
            <path d="M3 3v5h5" />
            <path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16" />
            <path d="M21 21v-5h-5" />
          </svg>
        </div>
      </div>

      <!-- 分隔线 -->
      <div class="h-px bg-base-content/[0.06] mx-5" />

      <!-- 动作 2: 重启核心 -->
      <div
        class="flex items-center justify-between px-5 py-4 cursor-pointer hover:bg-base-content/[0.04] active:bg-base-content/[0.08] transition-colors select-none group"
        :class="{ 'opacity-70 pointer-events-none': isRestarting }"
        @click="handleRestartCore"
      >
        <span class="text-[15px] sm:text-base font-semibold text-base-content/90 group-hover:text-base-content transition-colors">
          {{ t('coreControl.restartCore') || '重启核心' }}
        </span>
        <div class="text-orange-400 p-1 flex items-center justify-center">
          <svg
            class="size-6 transition-transform"
            :class="{ 'animate-spin': isRestarting }"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
            <path d="M3 3v5h5" />
            <path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16" />
            <path d="M21 21v-5h-5" />
          </svg>
        </div>
      </div>

      <!-- 分隔线 -->
      <div class="h-px bg-base-content/[0.06] mx-5" />

      <!-- 动作 3: 配置管理 -->
      <div
        class="flex items-center justify-between px-5 py-4 cursor-pointer hover:bg-base-content/[0.04] active:bg-base-content/[0.08] transition-colors select-none group"
        @click="openProfileModal"
      >
        <span class="text-[15px] sm:text-base font-semibold text-base-content/90 group-hover:text-base-content transition-colors">
          {{ t('coreControl.profileManager') || '配置管理' }}
        </span>
        <div class="text-base-content/75 group-hover:text-base-content p-1 flex items-center justify-center transition-colors">
          <svg
            class="size-6"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z" />
            <circle
              cx="12"
              cy="12"
              r="3"
            />
          </svg>
        </div>
      </div>

      <!-- 动作 4: 系统代理 (Desktop 原生环境扩展) -->
      <template v-if="isDesktop">
        <div class="h-px bg-base-content/[0.06] mx-5" />
        <div
          class="flex items-center justify-between px-5 py-4 cursor-pointer hover:bg-base-content/[0.04] active:bg-base-content/[0.08] transition-colors select-none group"
          :class="{ 'opacity-70 pointer-events-none': isTogglingProxy }"
          @click="handleToggleSysProxy"
        >
          <div class="flex items-center gap-2.5">
            <span class="text-[15px] sm:text-base font-semibold text-base-content/90 group-hover:text-base-content transition-colors">
              系统代理
            </span>
            <span
              class="badge badge-xs font-normal"
              :class="isSysProxyOn ? 'badge-success text-white' : 'badge-ghost text-base-content/50'"
            >
              {{ isSysProxyOn ? '已开启' : '未设置' }}
            </span>
          </div>
          <div class="p-1 flex items-center justify-center">
            <span
              v-if="isTogglingProxy"
              class="loading loading-spinner loading-xs text-primary"
            />
            <GlobeAltIcon
              v-else
              class="size-6 transition-colors"
              :class="isSysProxyOn ? 'text-success' : 'text-base-content/50'"
            />
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { GlobeAltIcon } from '@heroicons/vue/24/outline'
import { desktopApi } from '@/api/desktop'
import { reloadConfigsAPI } from '@/assembly/config'
import { restartCoreAPI } from '@/assembly/version'
import { reloadAll } from '@/composables/backendActions'
import {
  loadProfilesData,
  openProfileModal,
} from '@/composables/profiles'
import { showNotification } from '@/helper/notification'
import { notifyRequestError } from '@/helper/requestError'
import { useStatusStore } from '@/store/status'

const { t } = useI18n()
const statusStore = useStatusStore()

const isReloading = ref(false)
const isRestarting = ref(false)
const isDesktop = ref(desktopApi.isDesktop())
const isSysProxyOn = ref(false)
const isTogglingProxy = ref(false)
let unsubscribeProxyChange: (() => void) | null = null

// 核心状态计算
const statusState = computed<'running' | 'reloading' | 'stopped'>(() => {
  if (isReloading.value || isRestarting.value || statusStore.isReloading) {
    return 'reloading'
  }
  if (statusStore.status?.core_alive !== undefined) {
    return statusStore.status.core_alive ? 'running' : 'stopped'
  }
  return 'running'
})

const statusText = computed(() => {
  switch (statusState.value) {
    case 'reloading':
      return t('coreControl.reloading') || '正在重载'
    case 'stopped':
      return '运行异常'
    case 'running':
    default:
      return t('coreControl.running') || '正常运行'
  }
})

const statusCapsuleClass = computed(() => {
  switch (statusState.value) {
    case 'reloading':
      return 'bg-amber-950/40 text-amber-400 border-amber-500/20'
    case 'stopped':
      return 'bg-rose-950/40 text-rose-400 border-rose-500/20'
    case 'running':
    default:
      return 'bg-[#132a1c] text-[#22c55e] border-emerald-500/20'
  }
})

const statusDotClass = computed(() => {
  switch (statusState.value) {
    case 'reloading':
      return 'bg-amber-400 animate-ping'
    case 'stopped':
      return 'bg-rose-500'
    case 'running':
    default:
      return 'bg-[#22c55e]'
  }
})

const handleReloadConfig = async () => {
  if (isReloading.value) return
  isReloading.value = true
  try {
    await reloadConfigsAPI()
    await reloadAll()
    await statusStore.refresh()
    showNotification({
      content: t('coreControl.reloadSuccess') || '重载配置成功',
      type: 'alert-success',
    })
  } catch (err: unknown) {
    notifyRequestError(err)
  } finally {
    isReloading.value = false
  }
}

const handleRestartCore = async () => {
  if (isRestarting.value) return
  isRestarting.value = true
  try {
    await restartCoreAPI()
    await reloadAll()
    await statusStore.refresh()
    showNotification({
      content: t('coreControl.restartSuccess') || '重启核心成功',
      type: 'alert-success',
    })
  } catch (err: unknown) {
    notifyRequestError(err)
  } finally {
    isRestarting.value = false
  }
}

const handleToggleSysProxy = async () => {
  if (isTogglingProxy.value) return
  isTogglingProxy.value = true
  const target = !isSysProxyOn.value
  try {
    await desktopApi.setSystemProxy(target)
    isSysProxyOn.value = target
    showNotification({
      content: target ? '系统代理已开启' : '系统代理已关闭',
      type: 'alert-success',
    })
  } catch (err: unknown) {
    notifyRequestError(err)
  } finally {
    isTogglingProxy.value = false
  }
}

onMounted(async () => {
  loadProfilesData()
  if (isDesktop.value) {
    isSysProxyOn.value = await desktopApi.getSystemProxy()
    unsubscribeProxyChange = desktopApi.onSystemProxyChanged((enabled) => {
      isSysProxyOn.value = enabled
    })
  }
})

onUnmounted(() => {
  if (unsubscribeProxyChange) {
    unsubscribeProxyChange()
    unsubscribeProxyChange = null
  }
})
</script>

<style scoped>
.core-control-card {
  user-select: none;
}
</style>
