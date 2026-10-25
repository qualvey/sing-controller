<template>
  <DialogWrapper
    v-model="isOpen"
    :title="t('tunModal.title')"
    :closable="false"
    box-class="max-w-lg"
  >
    <div class="p-6 space-y-6">
      <!-- 警示卡片 -->
      <div class="relative overflow-hidden rounded-2xl bg-amber-500/10 border border-amber-500/20 p-5">
        <div class="flex items-start gap-4">
          <div class="flex-shrink-0 w-12 h-12 rounded-xl bg-amber-500/20 border border-amber-500/30 flex items-center justify-center text-amber-500 text-2xl shadow-lg shadow-amber-500/10">
            <span class="icon-[solar--shield-warning-bold-duotone]"></span>
          </div>
          <div class="flex-1 min-w-0">
            <h4 class="text-base font-semibold text-white tracking-tight flex items-center gap-2">
              {{ t('tunModal.headline') }}
            </h4>
            <p class="mt-1 text-xs leading-relaxed text-zinc-400">
              {{ t('tunModal.description') }}
            </p>
          </div>
        </div>
      </div>

      <!-- 选择方案对比说明 -->
      <div class="space-y-3">
        <div class="text-xs font-medium text-zinc-400 px-1">
          {{ t('tunModal.chooseAction') }}
        </div>

        <!-- 方案 1: UAC 提权 -->
        <button
          @click="handleElevate"
          :disabled="isElevating || isDisabling"
          class="w-full text-left p-4 rounded-xl border border-primary/30 bg-primary/10 hover:bg-primary/20 hover:border-primary/50 transition-all duration-200 group relative overflow-hidden"
        >
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <div class="w-9 h-9 rounded-lg bg-primary/20 flex items-center justify-center text-primary text-xl group-hover:scale-110 transition-transform">
                <span v-if="!isElevating" class="icon-[solar--key-bold-duotone]"></span>
                <span v-else class="icon-[solar--spinner-bold] animate-spin"></span>
              </div>
              <div>
                <div class="text-sm font-semibold text-zinc-100 group-hover:text-primary transition-colors">
                  {{ t('tunModal.elevateBtn') }}
                </div>
                <div class="text-[11px] text-zinc-400 mt-0.5">
                  {{ t('tunModal.elevateDesc') }}
                </div>
              </div>
            </div>
            <span class="icon-[solar--arrow-right-linear] text-zinc-500 group-hover:text-primary group-hover:translate-x-1 transition-all"></span>
          </div>
        </button>

        <!-- 方案 2: 禁用 TUN 模式 -->
        <button
          @click="handleDisableTun"
          :disabled="isElevating || isDisabling"
          class="w-full text-left p-4 rounded-xl border border-white/10 bg-white/[0.03] hover:bg-white/[0.08] hover:border-white/20 transition-all duration-200 group"
        >
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <div class="w-9 h-9 rounded-lg bg-white/10 flex items-center justify-center text-zinc-300 text-xl group-hover:scale-110 transition-transform">
                <span v-if="!isDisabling" class="icon-[solar--close-circle-bold-duotone]"></span>
                <span v-else class="icon-[solar--spinner-bold] animate-spin"></span>
              </div>
              <div>
                <div class="text-sm font-semibold text-zinc-200">
                  {{ t('tunModal.disableTunBtn') }}
                </div>
                <div class="text-[11px] text-zinc-400 mt-0.5">
                  {{ t('tunModal.disableTunDesc') }}
                </div>
              </div>
            </div>
            <span class="icon-[solar--arrow-right-linear] text-zinc-500 group-hover:text-zinc-300 group-hover:translate-x-1 transition-all"></span>
          </div>
        </button>
      </div>

      <!-- 底部状态说明 -->
      <div class="text-center">
        <p class="text-[11px] text-zinc-500">
          {{ t('tunModal.tip') }}
        </p>
      </div>
    </div>
  </DialogWrapper>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import { useStatusStore } from '@/store/status'
import { controllerApi } from '@/api/controller'
import { showToast } from '@/helper/toast'
import { startBackendSession, stopBackendSession } from '@/assembly/session'
import { getRequestErrorMessage } from '@/helper/requestError'

const { t } = useI18n()
const statusStore = useStatusStore()

const isElevating = ref(false)
const isDisabling = ref(false)

const waitForElevatedController = async () => {
  const deadline = Date.now() + 20_000

  while (Date.now() < deadline) {
    await new Promise((resolve) => setTimeout(resolve, 500))
    try {
      // The old controller goes away after returning the elevation response.
      // Keep probing the same endpoint until the elevated replacement is up.
      const status = await controllerApi.status()
      statusStore.status = status
      if (status.tun_access_denied === false) {
        // 页面资源来自旧 Controller。旧进程退出后虽然新进程已接管端口，
        // 当前 DOM/长连接仍属于旧会话；重新加载可确保 API、WebSocket 和
        // 内嵌静态资源全部从管理员进程重新建立。
        window.location.reload()
        return true
      }
    } catch {
      // Expected while the old process is shutting down.
    }
  }

  return false
}

const isOpen = computed({
  get: () => Boolean(statusStore.status?.tun_access_denied),
  set: () => {}
})

const handleElevate = async () => {
  isElevating.value = true
  stopBackendSession()
  try {
    await controllerApi.elevateProcess()
    showToast(t('tunModal.elevateSuccessNotify'), 'success')
    if (!(await waitForElevatedController())) {
      showToast('管理员 controller 未能重新连接，请检查 UAC 是否确认', 'error')
    }
    isElevating.value = false
  } catch (err: unknown) {
    const msg = getRequestErrorMessage(err) || '提权请求失败'
    showToast(msg, 'error')
    startBackendSession()
    isElevating.value = false
  }
}

const handleDisableTun = async () => {
  isDisabling.value = true
  try {
    await controllerApi.disableTun()
    showToast(t('tunModal.disableTunSuccessNotify'), 'success')
    await statusStore.refresh()
    setTimeout(() => {
      startBackendSession()
    }, 600)
  } catch (err: unknown) {
    const msg = (err as Error)?.message || '禁用 TUN 模式失败'
    showToast(msg, 'error')
  } finally {
    isDisabling.value = false
  }
}
</script>
