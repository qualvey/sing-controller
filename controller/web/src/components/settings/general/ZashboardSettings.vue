<template>
  <div class="relative flex flex-col text-sm">
    <div class="flex items-center justify-between gap-2 px-1 py-1">
      <div class="flex items-center gap-2">
        <div class="indicator">
          <span
            v-if="updateStore.isUpdateAvailable"
            class="indicator-item top-1 -right-1 flex"
            title="发现新版本"
          >
            <span class="bg-secondary absolute h-2 w-2 animate-ping rounded-full"></span>
            <span class="bg-secondary h-2 w-2 rounded-full"></span>
          </span>
          <a
            href="https://github.com/qualvey/sing-controller"
            target="_blank"
            rel="noopener noreferrer"
            class="text-lg font-semibold hover:text-primary transition-colors flex items-center gap-1.5"
          >
            sing-controller
            <span class="text-sm font-normal opacity-60">
              {{ updateStore.currentVersion }}
              <span
                v-if="updateStore.platformArch"
                class="badge badge-ghost badge-sm text-[10px] font-mono ml-1"
              >
                {{ updateStore.platformArch }}
              </span>
              <span
                v-if="commitId"
                class="text-xs opacity-70 ml-0.5"
              >
                ({{ commitId }})
              </span>
            </span>
          </a>
        </div>

        <!-- 发现新版本徽章提示 -->
        <a
          v-if="updateStore.isUpdateAvailable && updateStore.latestRelease"
          :href="updateStore.latestRelease.html_url"
          target="_blank"
          rel="noopener noreferrer"
          class="badge badge-secondary badge-sm gap-1 hover:brightness-110 cursor-pointer transition-all"
        >
          新版本 {{ updateStore.latestVersion }}
        </a>
      </div>

      <!-- 检查更新按钮 -->
      <button
        type="button"
        class="btn btn-ghost btn-xs gap-1 text-xs text-base-content/70 hover:text-base-content"
        :disabled="updateStore.isChecking"
        @click="handleCheckUpdate"
      >
        <ArrowPathIcon
          class="h-3.5 w-3.5"
          :class="{ 'animate-spin': updateStore.isChecking }"
        />
        <span>{{ updateStore.isChecking ? '正在检查...' : '检查更新' }}</span>
      </button>
    </div>

    <StyleSettings />
    <GeneralSettings />
  </div>
</template>

<script setup lang="ts">
import { showToast } from '@/helper/toast'
import { useUpdateStore } from '@/store/update'
import { ArrowPathIcon } from '@heroicons/vue/24/outline'
import { onMounted } from 'vue'
import GeneralSettings from './GeneralSettings.vue'
import StyleSettings from './StyleSettings.vue'

const commitId = __COMMIT_ID__
const updateStore = useUpdateStore()

onMounted(() => {
  // 静默检查更新（带 1 小时本地缓存）
  updateStore.checkUpdate(false).catch(() => {})
})

const handleCheckUpdate = async () => {
  try {
    const hasUpdate = await updateStore.checkUpdate(true)
    if (hasUpdate) {
      showToast(`发现新版本 ${updateStore.latestVersion}！`, 'info')
    } else if (updateStore.checkError) {
      showToast(updateStore.checkError, 'error')
    } else {
      showToast('当前已是最新版本', 'success')
    }
  } catch (err) {
    showToast((err as Error).message || '检查更新失败', 'error')
  }
}
</script>

