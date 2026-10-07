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
          {{ t('coreControl.title') }}
        </div>
      </div>
    </div>

    <div class="core-control-grid grid grid-cols-2 gap-3">
      <section class="core-control-actions bg-base-200/30 col-span-2 flex flex-col gap-3 rounded-xl p-4">
        <div class="text-base-content/60 text-xs font-semibold tracking-wider uppercase">
          {{ t('coreControl.actions') }}
        </div>

        <button
          type="button"
          class="group flex w-full items-center justify-between gap-3 rounded-xl border border-base-content/5 bg-base-100/40 px-3.5 py-3 text-left text-sm font-semibold text-base-content/90 transition-colors hover:bg-base-content/[0.05] active:bg-base-content/[0.08] disabled:pointer-events-none disabled:opacity-60"
          :disabled="isReloading"
          @click="handleReloadConfig"
        >
          <span>{{ t('coreControl.reloadConfig') }}</span>
          <svg
            class="size-5 text-sky-400 transition-transform"
            :class="{ 'animate-spin': isReloading }"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
            <path d="M3 3v5h5" />
            <path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16" />
            <path d="M21 21v-5h-5" />
          </svg>
        </button>

        <button
          type="button"
          class="group flex w-full items-center justify-between gap-3 rounded-xl border border-base-content/5 bg-base-100/40 px-3.5 py-3 text-left text-sm font-semibold text-base-content/90 transition-colors hover:bg-base-content/[0.05] active:bg-base-content/[0.08] disabled:pointer-events-none disabled:opacity-60"
          :disabled="isRestarting"
          @click="handleRestartCore"
        >
          <span>{{ t('coreControl.restartCore') }}</span>
          <svg
            class="size-5 text-orange-400 transition-transform"
            :class="{ 'animate-spin': isRestarting }"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
            <path d="M3 3v5h5" />
            <path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16" />
            <path d="M21 21v-5h-5" />
          </svg>
        </button>

        <button
          type="button"
          class="group flex w-full items-center justify-between gap-3 rounded-xl border border-base-content/5 bg-base-100/40 px-3.5 py-3 text-left text-sm font-semibold text-base-content/90 transition-colors hover:bg-base-content/[0.05] active:bg-base-content/[0.08]"
          @click="openProfileModal('providers')"
        >
          <span>{{ t('coreControl.providerManager') }}</span>
          <svg
            class="size-5 text-base-content/60 transition-colors group-hover:text-base-content"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M12 3v12" />
            <path d="m7 10 5 5 5-5" />
            <path d="M5 20h14" />
          </svg>
        </button>
      </section>

      <section class="bg-base-200/30 flex min-w-0 flex-col gap-3 rounded-xl p-4">
        <div class="flex items-center justify-between gap-2">
          <div class="text-base-content/60 text-xs font-semibold tracking-wider uppercase">
            {{ t('coreControl.runtime') }}
          </div>
          <div
            class="inline-flex shrink-0 items-center gap-1.5 rounded-full border px-2 py-1 text-[10px] font-semibold transition-colors"
            :class="statusCapsuleClass"
          >
            <span class="size-1.5 rounded-full" :class="statusDotClass" />
            <span>{{ statusText }}</span>
          </div>
        </div>
        <div class="flex flex-col gap-1">
          <div class="text-base-content/50 text-xs">{{ t('coreControl.uptime') }}</div>
          <div class="text-2xl font-extralight tabular-nums sm:text-3xl">
            {{ formattedUptime }}
          </div>
        </div>
      </section>

      <section class="bg-base-200/30 flex min-w-0 flex-col gap-3 rounded-xl p-4">
        <div class="text-base-content/60 text-xs font-semibold tracking-wider uppercase">
          {{ t('coreControl.configurationOverview') }}
        </div>
        <div class="grid grid-cols-3 gap-2">
          <div class="min-w-0">
            <div class="text-2xl font-extralight tabular-nums">{{ statusStore.status?.inbounds ?? '—' }}</div>
            <div class="text-base-content/50 truncate text-xs">{{ t('inbounds') }}</div>
          </div>
          <div class="min-w-0">
            <div class="text-2xl font-extralight tabular-nums">{{ statusStore.status?.outbounds ?? '—' }}</div>
            <div class="text-base-content/50 truncate text-xs">{{ t('outbound') }}</div>
          </div>
          <div class="min-w-0">
            <div class="text-2xl font-extralight tabular-nums">{{ statusStore.status?.rules ?? '—' }}</div>
            <div class="text-base-content/50 truncate text-xs">{{ t('rules') }}</div>
          </div>
        </div>
        <button
          type="button"
          class="group mt-auto flex w-full items-center justify-between gap-3 rounded-xl border border-base-content/5 bg-base-100/40 px-3.5 py-3 text-left text-sm font-semibold text-base-content/90 transition-colors hover:bg-base-content/[0.05] active:bg-base-content/[0.08]"
          @click="openProfileModal('profiles')"
        >
          <span>{{ t('coreControl.profileManager') }}</span>
          <svg
            class="size-5 text-base-content/60 transition-colors group-hover:text-base-content"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M9 5h10v14H9z" />
            <path d="M5 8v13h10" />
            <path d="M12 9h4M12 13h4" />
          </svg>
        </button>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { controllerApi, type CoreSupervisorStatus } from '@/api/controller'
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
const coreSupervisorStatus = ref<CoreSupervisorStatus | null>(null)
const coreStatusReceivedAt = ref(0)
const now = ref(Date.now())
let clockTimer: ReturnType<typeof setInterval> | null = null
let coreStatusTimer: ReturnType<typeof setInterval> | null = null

const refreshCoreStatus = async () => {
  try {
    coreSupervisorStatus.value = await controllerApi.coreStatus()
    coreStatusReceivedAt.value = Date.now()
  } catch {
    // The overview remains usable when the supervisor endpoint is unavailable.
  }
}

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
      return t('coreControl.stopped')
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

const uptimeSeconds = computed(() => {
  const uptime = coreSupervisorStatus.value?.uptime_sec
  if (uptime === undefined || statusState.value !== 'running') return null
  return uptime + Math.max(0, Math.floor((now.value - coreStatusReceivedAt.value) / 1000))
})

const formattedUptime = computed(() => {
  if (uptimeSeconds.value === null) return '—'

  let seconds = uptimeSeconds.value
  const days = Math.floor(seconds / 86_400)
  seconds %= 86_400
  const hours = Math.floor(seconds / 3_600)
  seconds %= 3_600
  const minutes = Math.floor(seconds / 60)
  seconds %= 60

  if (days > 0) return `${days}${t('coreControl.dayShort')} ${hours}${t('coreControl.hourShort')}`
  if (hours > 0) return `${hours}${t('coreControl.hourShort')} ${minutes}${t('coreControl.minuteShort')}`
  if (minutes > 0) return `${minutes}${t('coreControl.minuteShort')} ${seconds}${t('coreControl.secondShort')}`
  return `${seconds}${t('coreControl.secondShort')}`
})

const handleReloadConfig = async () => {
  if (isReloading.value) return
  isReloading.value = true
  try {
    await reloadConfigsAPI()
    await reloadAll()
    await statusStore.refresh()
    await refreshCoreStatus()
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
    await refreshCoreStatus()
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

onMounted(() => {
  loadProfilesData()
  void refreshCoreStatus()
  clockTimer = setInterval(() => {
    now.value = Date.now()
  }, 1000)
  coreStatusTimer = setInterval(() => {
    void refreshCoreStatus()
  }, 30_000)
})

onUnmounted(() => {
  if (clockTimer) clearInterval(clockTimer)
  if (coreStatusTimer) clearInterval(coreStatusTimer)
})
</script>

<style scoped>
.core-control-card {
  container-type: inline-size;
  user-select: none;
}

@container (min-width: 768px) {
  .core-control-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .core-control-actions {
    grid-column: span 1 / span 1;
  }
}
</style>
