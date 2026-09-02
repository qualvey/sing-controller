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
            <span>{{ t('coreControl.title', '内核与运行控制') }}</span>
            <span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-success/15 text-success">
              <span class="size-1.5 rounded-full bg-success animate-pulse" />
              {{ t('coreControl.running', '正常运行') }}
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
        <span class="text-base-content/60">{{ t('coreControl.currentProfile', '当前方案') }}:</span>
        <span class="text-base-content font-semibold">{{ activeProfileName }}</span>
      </div>
    </div>

    <!-- Actions & Stats Grid -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 mt-3">
      <!-- 动作 1: 重载配置 -->
      <div class="bg-base-200/30 hover:bg-base-200/50 transition-all rounded-xl p-3.5 flex flex-col justify-between gap-3">
        <div class="flex items-start justify-between">
          <div>
            <div class="text-xs font-semibold text-base-content/80">
              {{ t('coreControl.reloadConfig', '平滑重载配置') }}
            </div>
            <div class="text-[11px] text-base-content/50 mt-0.5">
              {{ t('coreControl.reloadDesc', '热加载规则与路由，不断开网络连接') }}
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
          <span>{{ isReloading ? t('coreControl.reloading', '重载中...') : t('coreControl.doReload', '执行重载') }}</span>
        </button>
      </div>

      <!-- 动作 2: 重启内核 -->
      <div class="bg-base-200/30 hover:bg-base-200/50 transition-all rounded-xl p-3.5 flex flex-col justify-between gap-3">
        <div class="flex items-start justify-between">
          <div>
            <div class="text-xs font-semibold text-base-content/80">
              {{ t('coreControl.restartCore', '重启内核进程') }}
            </div>
            <div class="text-[11px] text-base-content/50 mt-0.5">
              {{ t('coreControl.restartDesc', '由 Supervisor 重新拉起 sing-box 进程') }}
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
          <span>{{ isRestarting ? t('coreControl.restarting', '重启中...') : t('coreControl.doRestart', '执行重启') }}</span>
        </button>
      </div>

      <!-- 动作 3: 方案与订阅管理 -->
      <div class="bg-base-200/30 hover:bg-base-200/50 transition-all rounded-xl p-3.5 flex flex-col justify-between gap-3">
        <div class="flex items-start justify-between">
          <div>
            <div class="text-xs font-semibold text-base-content/80">
              {{ t('coreControl.profileManager', '配置组与订阅源') }}
            </div>
            <div class="text-[11px] text-base-content/50 mt-0.5">
              {{ t('coreControl.profileDesc', '管理多套配置方案及外部节点订阅') }}
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
          <span>{{ t('coreControl.openManager', '管理方案 / 订阅') }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  CpuChipIcon,
  ArrowPathIcon,
  ArrowPathRoundedSquareIcon,
  RectangleGroupIcon,
} from '@heroicons/vue/24/outline'
import { reloadConfigsAPI } from '@/assembly/config'
import { restartCoreAPI, version } from '@/assembly/version'
import {
  profilesList,
  loadProfilesData,
  openProfileModal,
} from '@/composables/profiles'
import { showNotification } from '@/helper/notification'
import { notifyRequestError } from '@/helper/requestError'

const { t } = useI18n()

const isReloading = ref(false)
const isRestarting = ref(false)

const coreName = computed(() => 'sing-box')
const coreVersionText = computed(() => version.value || 'v1.11.x')

const activeProfileName = computed(() => {
  const active = profilesList.value.find((p) => p.is_active)
  return active?.name || t('profiles.defaultProfileName', '默认方案')
})

const handleReloadConfig = async () => {
  if (isReloading.value) return
  isReloading.value = true
  try {
    await reloadConfigsAPI()
    showNotification({
      content: t('coreControl.reloadSuccess', '配置重载成功'),
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
    showNotification({
      content: t('coreControl.restartSuccess', '内核重启命令已发送'),
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
})
</script>
