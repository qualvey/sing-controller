<template>
  <!-- 连不上后端时,面板上的每一页都是空的 —— 与其让用户对着空页面猜,
       不如直接把「哪个后端、为什么、怎么办」摆在最前面。 -->
  <Transition name="connection-error">
    <div
      v-if="visible && activeBackend"
      class="bg-base-200/80 fixed inset-0 z-[100000] flex items-center justify-center overflow-auto p-4 backdrop-blur-sm"
    >
      <div
        class="border-base-border bg-base-100 flex w-[420px] max-w-full flex-col gap-3 rounded-xl border px-6 py-5 shadow-xl"
      >
        <div class="flex items-start gap-3">
          <span
            class="bg-error/10 text-error flex h-10 w-10 flex-none items-center justify-center rounded-lg"
          >
            <ExclamationTriangleIcon class="h-5 w-5" />
          </span>
          <div class="min-w-0 flex-1">
            <h1 class="text-base font-semibold leading-tight">
              {{
                isCoreOffline
                  ? 'sing-box 核心未运行'
                  : isWsError
                    ? 'WebSocket 连接失败'
                    : $t('backendUnreachable')
              }}
            </h1>
            <div class="text-base-content/60 mt-0.5 truncate text-sm">
              {{ isWsError ? '反向代理未配置 WebSocket / Upgrade 转发' : label }}
            </div>
            <div
              v-if="url !== label"
              class="text-base-content/40 truncate text-xs"
            >
              {{ url }}
            </div>
          </div>
        </div>

        <!-- 状态指示栏：核心离线 -->
        <div
          v-if="isCoreOffline"
          class="bg-base-200/60 flex items-center gap-2 rounded-lg px-3 py-1.5 text-xs"
        >
          <span class="inline-flex items-center gap-1 text-success">
            <span class="inline-block h-2 w-2 rounded-full bg-success"></span>
            Controller: 在线
          </span>
          <span class="text-base-content/20">|</span>
          <span class="inline-flex items-center gap-1 text-error">
            <span class="inline-block h-2 w-2 rounded-full bg-error"></span>
            sing-box 核心: 离线
          </span>
        </div>

        <!-- 状态指示栏：WebSocket 握手失败 -->
        <div
          v-else-if="isWsError"
          class="bg-base-200/60 flex items-center gap-2 rounded-lg px-3 py-1.5 text-xs"
        >
          <span class="inline-flex items-center gap-1 text-success">
            <span class="inline-block h-2 w-2 rounded-full bg-success"></span>
            Controller: 在线
          </span>
          <span class="text-base-content/20">|</span>
          <span class="inline-flex items-center gap-1 text-success">
            <span class="inline-block h-2 w-2 rounded-full bg-success"></span>
            sing-box 核心: 在线
          </span>
          <span class="text-base-content/20">|</span>
          <span class="inline-flex items-center gap-1 text-error">
            <span class="inline-block h-2 w-2 rounded-full bg-error"></span>
            WebSocket: 异常
          </span>
        </div>

        <div class="bg-error/10 text-error rounded-lg px-3 py-2 text-xs leading-5 break-all">
          {{ displayErrorDetail }}
        </div>

        <!-- WebSocket 异常时展示 Nginx / 反代配置指导与一键复制 -->
        <div
          v-if="isWsError"
          class="border-base-border bg-base-200/60 flex flex-col gap-2 rounded-lg border p-3 text-xs"
        >
          <div class="flex items-center justify-between">
            <span class="font-medium text-base-content/80">Nginx 反代配置参考：</span>
            <button
              class="btn btn-ghost btn-xs text-primary h-6 min-h-0 px-2"
              @click="copyNginxConfig"
            >
              {{ copiedNginx ? '已复制 ✓' : '复制配置' }}
            </button>
          </div>
          <pre class="bg-base-300/80 text-base-content/90 rounded p-2 text-[11px] font-mono leading-relaxed overflow-x-auto select-all"><code>proxy_http_version 1.1;
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection "upgrade";</code></pre>
          <div class="text-base-content/60 text-[11px] leading-relaxed">
            提示：HTTP 请求正常但 WebSocket 无法升级握手。实时流量、连接图表与日志均依赖 WebSocket。
          </div>
        </div>

        <div class="flex flex-col gap-2">
          <div class="flex gap-2">
            <!-- 核心离线时提供一键重启/重载核心 -->
            <button
              v-if="isCoreOffline"
              class="btn btn-primary btn-sm flex-1"
              :disabled="isReloading || isRetrying"
              @click="reloadCore"
            >
              <span
                v-if="isReloading"
                class="loading loading-spinner loading-xs"
              ></span>
              <ArrowPathIcon
                v-else
                class="h-4 w-4"
              />
              {{ isReloading ? '正在重载核心...' : '重载核心' }}
            </button>

            <button
              class="btn btn-sm flex-1"
              :class="{ 'btn-primary': !isCoreOffline, 'btn-outline': isCoreOffline }"
              :disabled="isRetrying || isReloading"
              @click="retry"
            >
              <span
                v-if="isRetrying"
                class="loading loading-spinner loading-xs"
              ></span>
              {{ isRetrying ? $t('backendConnecting') : $t('retry') }}
            </button>
            <button
              class="btn btn-sm flex-1"
              @click="editActiveBackend"
            >
              {{ $t('editBackendTitle') }}
            </button>
          </div>

          <button
            class="btn btn-ghost btn-xs w-full text-base-content/70"
            @click="openBackendManager()"
          >
            {{ $t('manageBackends') }}
          </button>
        </div>

        <!-- 快速切换到其他后端 -->
        <div
          v-if="otherBackends.length > 0"
          class="border-base-border border-t pt-2.5"
        >
          <div class="text-base-content/60 mb-1.5 text-xs font-medium">快速切换到其他后端：</div>
          <div class="flex max-h-32 flex-col gap-1 overflow-y-auto">
            <button
              v-for="backend in otherBackends"
              :key="backend.uuid"
              class="hover:bg-base-200 flex items-center justify-between rounded-md px-2 py-1.5 text-left text-xs transition-colors"
              @click="switchBackend(backend.uuid)"
            >
              <span class="flex items-center gap-1.5 truncate">
                <ServerIcon class="h-3.5 w-3.5 opacity-60 flex-none" />
                <span class="truncate">{{ getLabelFromBackend(backend) }}</span>
              </span>
              <ChevronRightIcon class="h-3 w-3 opacity-40 flex-none" />
            </button>
          </div>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { api } from '@/api'
import { startBackendSession } from '@/assembly/session'
import { backendProbe } from '@/assembly/version'
import { ROUTE_NAME } from '@/constant'
import { describeConnectionError } from '@/helper/connectivity'
import { showToast } from '@/helper/toast'
import { getBackendProbeUrl, getLabelFromBackend } from '@/helper/utils'
import {
  activeBackend,
  activeUuid,
  backendList,
  backendManagerView,
  openBackendManager,
} from '@/store/setup'
import { useStatusStore } from '@/store/status'
import {
  ArrowPathIcon,
  ChevronRightIcon,
  ExclamationTriangleIcon,
  ServerIcon,
} from '@heroicons/vue/24/outline'
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'

const statusStore = useStatusStore()
const detail = ref('')
const isReloading = ref(false)
const copiedNginx = ref(false)

// 探测结果属于上一个后端时不认,免得刚切走就弹一个旧后端的失败。
const probe = computed(() =>
  backendProbe.value?.uuid === activeUuid.value ? backendProbe.value : undefined,
)
const isRetrying = computed(() => probe.value?.status === 'probing')

// 判断是否属于 Controller 存活但 sing-box 核心未运行 / 离线
const isCoreOffline = computed(() => {
  const msg = probe.value?.message || detail.value || ''
  return (
    msg.includes('核心未运行') ||
    msg.includes('core offline') ||
    statusStore.status?.core_alive === false ||
    (statusStore.status?.core && !statusStore.status.core.alive)
  )
})

// 判断是否属于 HTTP 正常但 WebSocket 握手失败
const isWsError = computed(() => {
  const msg = probe.value?.message || detail.value || ''
  return (
    !isCoreOffline.value &&
    (msg.includes('WebSocket') ||
      msg.includes('websocket') ||
      msg.includes('Upgrade') ||
      msg.includes('upgrade') ||
      msg.includes('反代') ||
      msg.includes('反向代理'))
  )
})

const displayErrorDetail = computed(() => {
  if (isCoreOffline.value) {
    return (
      statusStore.status?.core_error ||
      statusStore.status?.core?.error ||
      detail.value ||
      'sing-box 核心进程已停止或未启动'
    )
  }
  if (isWsError.value) {
    return detail.value || 'WebSocket 连接失败，请检查反向代理 WebSocket (Upgrade) 转发配置'
  }
  return detail.value || '无法连接到控制服务'
})

const copyNginxConfig = async () => {
  const config = `proxy_http_version 1.1;\nproxy_set_header Upgrade $http_upgrade;\nproxy_set_header Connection "upgrade";`
  try {
    await navigator.clipboard.writeText(config)
    copiedNginx.value = true
    showToast('已复制 Nginx WebSocket 配置', 'success')
    setTimeout(() => {
      copiedNginx.value = false
    }, 2500)
  } catch {
    showToast('复制失败，请手动选择复制', 'error')
  }
}

// 失败一次就一直摆着,直到真的连上或换了后端 —— 点重试的瞬间探测状态会回到
// probing,若跟着它走,页面会先整个消失再弹回来,像是自己好了。
const failed = ref(false)

watch(
  [() => probe.value?.status, () => statusStore.status?.core_alive, () => statusStore.status?.core?.alive],
  ([status, coreAlive, coreObjAlive]) => {
    if (status === 'failed' || coreAlive === false || coreObjAlive === false) {
      failed.value = true
    } else if (status === 'connected' || status === undefined) {
      failed.value = false
    }
  },
  { immediate: true },
)

// 管理面板自己会实时探测,盖在它上面只会挡路;
// Setup 页本来就是登录后端的地方,不必再盖一层。
const route = useRoute()
const visible = computed(
  () => failed.value && backendManagerView.value === null && route.name !== ROUTE_NAME.setup,
)

const label = computed(() => (activeBackend.value ? getLabelFromBackend(activeBackend.value) : ''))
const url = computed(() => (activeBackend.value ? getBackendProbeUrl(activeBackend.value) : ''))

const otherBackends = computed(() =>
  backendList.value.filter((backend) => backend.uuid !== activeUuid.value),
)

// 会话探测拿到的是 axios / gRPC 的原始 message,"Network Error" 这类不透明说法
// 要再诊断一次才有信息量(CORS?混合内容?服务没起?)。
watch(
  probe,
  async (value) => {
    // 重试期间(probing)留着上一轮的原因不动:清空只会让横幅退回一句废话,
    // 而这一轮多半还是同一个原因。
    if (!value || value.status === 'connected') {
      detail.value = ''
      return
    }
    if (value.status !== 'failed') return

    const target = value.uuid
    const described = await describeConnectionError(value.message, url.value)

    if (activeUuid.value === target) {
      detail.value = described
    }
  },
  { immediate: true },
)

const retry = () => {
  void statusStore.refresh()
  startBackendSession()
}

const reloadCore = async () => {
  if (isReloading.value) return
  isReloading.value = true
  try {
    const res = await api.reload()
    showToast(res.reloaded ? 'sing-box 核心重载成功' : '已发送重载指令', 'success')
    await statusStore.refresh()
    startBackendSession()
  } catch (e) {
    showToast((e as Error).message || '重载核心失败', 'error')
  } finally {
    isReloading.value = false
  }
}

const switchBackend = (uuid: string) => {
  activeUuid.value = uuid
  startBackendSession()
}

const editActiveBackend = () => {
  if (!activeUuid.value) return
  openBackendManager({ mode: 'edit', uuid: activeUuid.value })
}
</script>

<style scoped>
.connection-error-enter-active,
.connection-error-leave-active {
  transition: opacity 0.2s ease;
}

.connection-error-enter-from,
.connection-error-leave-to {
  opacity: 0;
}
</style>

