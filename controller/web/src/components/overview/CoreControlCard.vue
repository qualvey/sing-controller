<template>
  <div class="core-control-card base-container w-full p-4">
    <!-- Header -->
    <div class="flex items-center justify-between border-b border-base-200/60 pb-3">
      <div class="flex items-center gap-2.5">
        <div class="p-2 rounded-xl bg-primary/10 text-primary">
          <CpuChipIcon class="size-5" />
        </div>
        <div>
          <div class="font-semibold text-sm flex items-center gap-2">
            <span>{{ t('coreControl.title') }}</span>
            <span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-success/15 text-success">
              <span class="size-1.5 rounded-full bg-success animate-pulse" />
              {{ t('coreControl.running') }}
            </span>
          </div>
          <div class="text-xs text-base-content/50 mt-0.5">
            {{ coreName }} {{ coreVersionText }}
          </div>
        </div>
      </div>

      <!-- 当前方案徽标 -->
      <div
        class="badge badge-lg gap-1.5 cursor-pointer bg-base-200 hover:bg-base-300 transition-all text-xs font-medium py-3 px-3 rounded-xl border-none"
        @click="openProfileModal"
      >
        <RectangleGroupIcon class="size-4 text-primary" />
        <span class="text-base-content/60">{{ t('coreControl.currentProfile') }}:</span>
        <span class="text-base-content font-semibold">{{ activeProfileName }}</span>
      </div>
    </div>

    <!-- Actions & Stats Grid -->
    <div
      class="grid gap-3 mt-3"
      :class="isDesktop ? 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-4' : 'grid-cols-1 sm:grid-cols-3'"
    >
      <!-- 动作 1: 重载配置 -->
      <div class="bg-base-200/30 hover:bg-base-200/50 transition-all rounded-xl p-3.5 flex flex-col justify-between gap-3">
        <div class="flex items-start justify-between">
          <div>
            <div class="text-xs font-semibold text-base-content/80">
              {{ t('coreControl.reloadConfig') }}
            </div>
            <div class="text-[11px] text-base-content/50 mt-0.5">
              {{ t('coreControl.reloadDesc') }}
            </div>
          </div>
          <div class="p-1.5 rounded-lg bg-info/10 text-info">
            <ArrowPathIcon class="size-4" />
          </div>
        </div>
        <button
          class="btn btn-sm btn-outline btn-info w-full gap-1.5 rounded-lg font-medium"
          :disabled="isReloading"
          @click="handleReloadConfig"
        >
          <span
            v-if="isReloading"
            class="loading loading-spinner loading-xs"
          />
          <ArrowPathIcon
            v-else
            class="size-4"
          />
          <span>{{ isReloading ? t('coreControl.reloading') : t('coreControl.doReload') }}</span>
        </button>
      </div>

      <!-- 动作 2: 重启内核 -->
      <div class="bg-base-200/30 hover:bg-base-200/50 transition-all rounded-xl p-3.5 flex flex-col justify-between gap-3">
        <div class="flex items-start justify-between">
          <div>
            <div class="text-xs font-semibold text-base-content/80">
              {{ t('coreControl.restartCore') }}
            </div>
            <div class="text-[11px] text-base-content/50 mt-0.5">
              {{ t('coreControl.restartDesc') }}
            </div>
          </div>
          <div class="p-1.5 rounded-lg bg-warning/10 text-warning">
            <ArrowPathRoundedSquareIcon class="size-4" />
          </div>
        </div>
        <button
          class="btn btn-sm btn-outline btn-warning w-full gap-1.5 rounded-lg font-medium"
          :disabled="isRestarting"
          @click="handleRestartCore"
        >
          <span
            v-if="isRestarting"
            class="loading loading-spinner loading-xs"
          />
          <ArrowPathRoundedSquareIcon
            v-else
            class="size-4"
          />
          <span>{{ isRestarting ? t('coreControl.restarting') : t('coreControl.doRestart') }}</span>
        </button>
      </div>

      <!-- 动作 3: 方案与订阅管理 -->
      <div class="bg-base-200/30 hover:bg-base-200/50 transition-all rounded-xl p-3.5 flex flex-col justify-between gap-3">
        <div class="flex items-start justify-between">
          <div>
            <div class="text-xs font-semibold text-base-content/80">
              {{ t('coreControl.profileManager') }}
            </div>
            <div class="text-[11px] text-base-content/50 mt-0.5">
              {{ t('coreControl.profileDesc') }}
            </div>
          </div>
          <div class="p-1.5 rounded-lg bg-primary/10 text-primary">
            <RectangleGroupIcon class="size-4" />
          </div>
        </div>
        <button
          class="btn btn-sm btn-primary w-full gap-1.5 rounded-lg font-medium"
          @click="openProfileModal"
        >
          <RectangleGroupIcon class="size-4" />
          <span>{{ t('coreControl.openManager') }}</span>
        </button>
      </div>

      <!-- 动作 4: 系统代理 (Desktop 原生能力) -->
      <div
        v-if="isDesktop"
        class="bg-base-200/30 hover:bg-base-200/50 transition-all rounded-xl p-3.5 flex flex-col justify-between gap-3"
      >
        <div class="flex items-start justify-between">
          <div>
            <div class="text-xs font-semibold text-base-content/80 flex items-center gap-1.5">
              <span>系统代理</span>
              <span
                class="badge badge-xs font-normal"
                :class="isSysProxyOn ? 'badge-success text-white' : 'badge-ghost text-base-content/50'"
              >
                {{ isSysProxyOn ? '已开启' : '未设置' }}
              </span>
            </div>
            <div class="text-[11px] text-base-content/50 mt-0.5">
              {{ isSysProxyOn ? '系统流量已接入 sing-box' : '一键配置操作系统代理' }}
            </div>
          </div>
          <div
            class="p-1.5 rounded-lg"
            :class="isSysProxyOn ? 'bg-success/15 text-success' : 'bg-base-300/60 text-base-content/50'"
          >
            <GlobeAltIcon class="size-4" />
          </div>
        </div>
        <button
          class="btn btn-sm w-full gap-1.5 rounded-lg font-medium transition-all"
          :class="isSysProxyOn ? 'btn-outline btn-error' : 'btn-success text-white'"
          :disabled="isTogglingProxy"
          @click="handleToggleSysProxy"
        >
          <span
            v-if="isTogglingProxy"
            class="loading loading-spinner loading-xs"
          />
          <GlobeAltIcon
            v-else
            class="size-4"
          />
          <span>{{ isSysProxyOn ? '关闭系统代理' : '开启系统代理' }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  CpuChipIcon,
  ArrowPathIcon,
  ArrowPathRoundedSquareIcon,
  RectangleGroupIcon,
  GlobeAltIcon,
} from '@heroicons/vue/24/outline'
import { desktopApi } from '@/api/desktop'
import { reloadConfigsAPI } from '@/assembly/config'
import { restartCoreAPI, version } from '@/assembly/version'
import { reloadAll } from '@/composables/backendActions'
import {
  profilesList,
  loadProfilesData,
  openProfileModal,
} from '@/composables/profiles'
import { showNotification } from '@/helper/notification'
import { notifyRequestError } from '@/helper/requestError'

const { t, te } = useI18n()

const isReloading = ref(false)
const isRestarting = ref(false)
const isDesktop = ref(desktopApi.isDesktop())
const isSysProxyOn = ref(false)
const isTogglingProxy = ref(false)
let unsubscribeProxyChange: (() => void) | null = null

const coreName = computed(() => 'sing-box')
const coreVersionText = computed(() => version.value || 'v1.11.x')

const activeProfileName = computed(() => {
  const active = profilesList.value.find((p) => p.is_active)
  return active?.name || (te('profiles.defaultProfileName') ? t('profiles.defaultProfileName') : '默认方案')
})

const handleReloadConfig = async () => {
  if (isReloading.value) return
  isReloading.value = true
  try {
    await reloadConfigsAPI()
    await reloadAll()
    showNotification({
      content: t('coreControl.reloadSuccess'),
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
    showNotification({
      content: t('coreControl.restartSuccess'),
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
