import { api } from '@/api'
import { controllerApi } from '@/api/controller'
import {
  fetchClashVersion,
  getClashWsUrl,
  probeClashChannel,
  restartCoreAPI as clashRestartCoreAPI,
  upgradeCoreAPI,
  upgradeUIAPI,
} from '@/api/clash'
import { getSingboxClient } from '@/api/singbox/client'
import HonkLogo from '@/assets/images/honk.svg'
import MetacubexLogo from '@/assets/images/metacubex.jpg'
import { MIHOMO, MIHOMO_CHANNEL } from '@/constant'
import { probeWebSocket } from '@/helper/connectivity'
import { getRequestErrorMessage } from '@/helper/requestError'
import { autoUpgradeCore, autoUpgradeDashboard, checkUpgradeCore } from '@/store/settings'
import { activeBackend } from '@/store/setup'
import type { Backend } from '@/types'
import { computed, nextTick, ref } from 'vue'
import { can, channel, Channel, clashAvailable, core, Core, resetCore } from './backend'

export const version = ref()
export const isCoreUpdateAvailable = ref(false)
export const zashboardVersion = ref(__APP_VERSION__)

// 切后端时本来就要打一次 /version,顺手把它的结果暴露成连通性状态,
// 给切换提示用 —— 不额外发探测请求,量的也正是实际在用的那条 API。
export type BackendProbe = {
  uuid: string
  status: 'probing' | 'connected' | 'failed'
  // 拿到 /version 响应的耗时(ms),failed 时无意义。
  latency: number
  message: string
}

export const backendProbe = ref<BackendProbe | undefined>()

// 内核类型嗅探：优先识别 sing-box，支持 honk 与 mihomo
const detectCore = (versionString: string): Core => {
  if (!versionString) return Core.Unknown
  if (/\bsing-box\b/i.test(versionString) || /singbox/i.test(versionString)) return Core.Singbox
  if (/\bhonk\b/i.test(versionString)) return Core.Honk
  if (/\bmihomo\b/i.test(versionString) || /\bmeta\b/i.test(versionString)) return Core.Mihomo
  return Core.Singbox
}

// 内核品牌的展示信息(logo / 官网链接)。纯展示,不是能力门控,故允许 view 使用。
export const coreBrand = computed(() => {
  switch (core.value) {
    case Core.Honk:
      return { logo: HonkLogo, url: 'https://github.com/Glassyiris/honk' }
    case Core.Mihomo:
      return {
        logo: MetacubexLogo,
        url: MIHOMO_CHANNEL[mihomo.value?.[0] ?? MIHOMO.Meta].url,
      }
    case Core.Singbox:
    default:
      return {
        logo: MetacubexLogo,
        url: 'https://sing-box.sagernet.org',
      }
  }
})

export const mihomo = computed<[MIHOMO, string] | undefined>(() => {
  if (core.value !== Core.Mihomo) return undefined

  const match = /(alpha-smart|alpha|beta|meta)-?(\w+)/.exec(version.value)
  switch (match?.[1]) {
    case 'alpha':
      return [MIHOMO.Alpha, match[2] ?? version.value]
    case 'alpha-smart':
      return [MIHOMO.Smart, match[2] ?? version.value]
    case 'meta':
      return [MIHOMO.Meta, match[2] ?? version.value]
    default:
      return [MIHOMO.Meta, version.value]
  }
})

export const fetchVersionAPI = async (): Promise<{ data: { version: string } }> => {
  if (channel.value === Channel.Singbox) {
    // 1. 优先通过 controller /api/status 检查内核健康与版本
    try {
      const status = await api.status()
      if (status) {
        if (status.core && !status.core.alive) {
          throw new Error(status.core.error || 'sing-box 核心未运行')
        }
        if (status.core_alive === false) {
          throw new Error(status.core_error || 'sing-box 核心未运行')
        }
        if (status.core?.version || status.core_version) {
          return { data: { version: status.core?.version || status.core_version || 'sing-box' } }
        }
      }
    } catch (e) {
      if (e instanceof Error && (e.message.includes('核心未运行') || e.message.includes('offline'))) {
        throw e
      }
    }

    // 2. 备选尝试 direct gRPC StartedService.getVersion
    try {
      const client = getSingboxClient()
      if (client) {
        const resp = await client.client.getVersion({})
        if (resp && resp.version) {
          return { data: { version: resp.version } }
        }
      }
    } catch {
      // continue
    }

    // 3. 备选尝试 clash API /version (必须真实请求并抛错，不吞掉异常)
    return await fetchClashVersion()
  }

  return fetchClashVersion()
}

const probeBackend = async (backend: Backend) => {
  const startAt = Date.now()
  let data

  try {
    ; ({ data } = await fetchVersionAPI())
  } catch (e) {
    if (activeBackend.value?.uuid === backend.uuid) {
      backendProbe.value = {
        uuid: backend.uuid,
        status: 'failed',
        latency: 0,
        message: getRequestErrorMessage(e),
      }
    }
    throw e
  }

  // 探测期间用户可能又切了后端,过期结果直接丢弃。
  if (activeBackend.value?.uuid !== backend.uuid) return

  if (backend.type === 'clash') {
    // Clash 模式：验证 WebSocket 连通性（防止 Nginx 等反向代理漏配 Upgrade 导致实时流静默失效）
    const wsUrl = getClashWsUrl(backend, 'traffic')
    const wsProbe = await probeWebSocket(wsUrl, 3000)
    if (!wsProbe.ok) {
      clashAvailable.value = false
      if (activeBackend.value?.uuid === backend.uuid) {
        backendProbe.value = {
          uuid: backend.uuid,
          status: 'failed',
          latency: 0,
          message: wsProbe.message || 'WebSocket 握手失败（反向代理未配置 Upgrade 支持）',
        }
      }
      return
    }
    clashAvailable.value = true
  } else {
    // sing-box gRPC 模式：异步静默探测 clash_api 是否可用（作为可选增强功能，不阻断主链路）
    probeClashChannel(backend, 2000)
      .then((res) => {
        clashAvailable.value = res.ok
      })
      .catch(() => {
        clashAvailable.value = false
      })
  }

  version.value = data?.version || ''
  core.value = detectCore(version.value)
  backendProbe.value = {
    uuid: backend.uuid,
    status: 'connected',
    latency: Date.now() - startAt,
    message: '',
  }


  if (!can('coreUpdateCheck') || !checkUpgradeCore.value || backend.disableUpgradeCore) return

  isCoreUpdateAvailable.value = await fetchBackendUpdateAvailableAPI()

  if (isCoreUpdateAvailable.value && autoUpgradeCore.value) {
    // 自动升级不是用户点的,失败静默
    upgradeCoreAPI('auto').catch(() => { })
  }
}

// 当前后端的内核探测。core 未就绪前依赖它的判断都不可信,
// 需要等结论的调用方(如登录后的设置同步)用 coreReady() 等待。
let probe: Promise<void> = Promise.resolve()

export const coreReady = async () => {
  // 先让会话的 watcher 跑完,确保拿到的是新后端的探测,而非上一次的残留。
  await nextTick()
  await probe
}

// 由 assembly/session 在每次会话开始时调用:先把上一个后端的结论清干净,
// 再对当前后端重新探测。返回的 promise 只给 coreReady 用,调用方不必等。
export const probeActiveBackend = () => {
  const backend = activeBackend.value

  resetCore()
  version.value = ''
  isCoreUpdateAvailable.value = false
  backendProbe.value = backend
    ? { uuid: backend.uuid, status: 'probing', latency: 0, message: '' }
    : undefined

  probe = backend ? probeBackend(backend).catch(() => { }) : Promise.resolve()
  return probe
}

const CACHE_DURATION = 1000 * 60 * 60

interface CacheEntry<T> {
  timestamp: number
  version: string
  data: T
}

async function fetchWithLocalCache<T>(url: string, version: string): Promise<T> {
  const cacheKey = 'cache/' + url
  const cacheRaw = localStorage.getItem(cacheKey)

  if (cacheRaw) {
    try {
      const cache: CacheEntry<T> = JSON.parse(cacheRaw)
      const now = Date.now()

      if (now - cache.timestamp < CACHE_DURATION && cache.version === version) {
        return cache.data
      } else {
        localStorage.removeItem(cacheKey)
      }
    } catch (e) {
      console.warn('Failed to parse cache for', url, e)
    }
  }

  const response = await fetch(url)
  if (!response.ok) {
    throw new Error(`Fetch failed: ${response.status} ${response.statusText}`)
  }

  const data: T = await response.json()
  const newCache: CacheEntry<T> = {
    timestamp: Date.now(),
    version,
    data,
  }

  localStorage.setItem(cacheKey, JSON.stringify(newCache))
  return data
}

export const fetchIsUIUpdateAvailable = async () => {
  const { tag_name } = await fetchWithLocalCache<{ tag_name: string }>(
    'https://api.github.com/repos/Zephyruso/zashboard/releases/latest',
    zashboardVersion.value,
  )

  return Boolean(tag_name && tag_name !== `v${zashboardVersion.value}`)
}

const check = async (url: string, versionNumber: string) => {
  const { assets } = await fetchWithLocalCache<{ assets: { name: string }[] }>(url, versionNumber)
  const alreadyLatest = assets.some(({ name }) => name.includes(versionNumber))

  return !alreadyLatest
}

export const fetchBackendUpdateAvailableAPI = async () => {
  return await check(
    MIHOMO_CHANNEL[mihomo.value?.[0] ?? MIHOMO.Meta].check_update_url,
    mihomo.value?.[1] ?? version.value,
  )
}

// 仪表盘(UI)更新检查,迁自 composables/settings.ts 的 useSettings。
export const isUIUpdateAvailable = ref(false)

export const checkUIUpdate = async () => {
  isUIUpdateAvailable.value = await fetchIsUIUpdateAvailable()
  if (isUIUpdateAvailable.value && autoUpgradeDashboard.value) {
    // 自动升级不是用户点的,失败静默
    upgradeUIAPI().catch(() => { })
  }
}

export const restartCoreAPI = async () => {
  if (activeBackend.value?.type === 'controller' || channel.value === Channel.Singbox) {
    return controllerApi.coreRestart()
  }
  return clashRestartCoreAPI()
}

export { upgradeCoreAPI, upgradeUIAPI }
