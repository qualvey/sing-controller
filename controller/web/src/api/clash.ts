// clash API 客户端（移植自 zashboard，MIT）
// 通过 controller 的 /api/clash/* 反向代理同源访问 sing-box clash API：
// secret 由 controller 注入，前端不接触。
import axios from 'axios'
import ReconnectingWebSocket from 'reconnectingwebsocket'
import { shallowRef } from 'vue'
import { probeWebSocket, type ProbeResult } from '@/helper/connectivity'

// ============ 类型（clash API 响应结构） ============

export interface HistoryItem {
  time: string
  delay: number
}

export interface ProxyItem {
  name: string
  type: string
  history: HistoryItem[]
  all?: string[]
  udp?: boolean
  xudp?: boolean
  now?: string
  fixed?: string
  icon?: string
  hidden?: boolean
  selectable?: boolean
  testUrl?: string
  'dialer-proxy'?: string
  'provider-name'?: string
}

export interface ProxyGroup extends ProxyItem {
  type: 'Selector' | 'URLTest' | 'Fallback' | 'LoadBalance' | string
  now?: string
  all: string[]
}

export interface RuleItem {
  type: string
  payload: string
  proxy: string
  size: number
  uuid?: string
}

export interface ConnectionMeta {
  network: string
  type?: string
  sourceIP: string
  sourcePort: string
  destinationIP: string
  destinationPort: string
  host: string
  sniffHost?: string
  dnsMode?: string
  process?: string
  processPath?: string
  inboundName?: string
  inboundUser?: string
  sourceGeoIP?: string
  destinationGeoIP?: string
  destinationIPASN?: string
}

export interface ConnectionItem {
  id: string
  download: number
  upload: number
  chains: string[]
  rule: string
  rulePayload: string
  start: string | number
  metadata: ConnectionMeta
}

export interface ConnectionsSnapshot {
  downloadTotal: number
  uploadTotal: number
  connections: ConnectionItem[]
}

export interface LogItem {
  type: 'debug' | 'info' | 'warning' | 'error'
  payload: string
  time?: string
}

export interface TrafficSnapshot {
  up: number
  down: number
}

export interface ClashConfig {
  port?: number
  'mixed-port'?: number
  'socks-port'?: number
  'redir-port'?: number
  'tproxy-port'?: number
  'allow-lan'?: boolean
  mode?: string
  'log-level'?: string
  ipv6?: boolean
  tun?: { enable: boolean }
}

export interface DNSQueryResult {
  Question?: { name: string; type: number }
  Answer?: { name: string; type: number; TTL: number; data: string }[]
  Server?: string
}

// ============ REST 客户端 ============

const http = axios.create({ baseURL: import.meta.env.BASE_URL + 'api/clash', timeout: 15000 })

export const fetchClashVersion = () => http.get<{ version: string }>('/version')

export const fetchProxies = () => http.get<{ proxies: Record<string, ProxyItem> }>('/proxies')

/** 选择代理组节点（PUT /proxies/{group} {name}） */
export const selectProxy = (group: string, name: string) =>
  http.put(`/proxies/${encodeURIComponent(group)}`, { name })

/** 单节点延迟测试 */
export const fetchProxyLatency = (name: string, url: string, timeout: number) =>
  http.get<{ delay: number }>(`/proxies/${encodeURIComponent(name)}/delay`, {
    params: { url, timeout }
  })

/** 整组延迟测试（返回 {节点: 延迟}） */
export const fetchGroupLatency = (group: string, url: string, timeout: number) =>
  http.get<Record<string, number>>(`/group/${encodeURIComponent(group)}/delay`, {
    params: { url, timeout }
  })

export const fetchRules = () => http.get<{ rules: RuleItem[] }>('/rules')

export const fetchConfigs = () => http.get<ClashConfig>('/configs')

export const patchConfigs = (patch: Record<string, string | number | boolean | object>) =>
  http.patch('/configs', patch)

export const flushFakeIP = () => http.post('/cache/fakeip/flush')
export const flushDNSCache = () => http.post('/cache/dns/flush')

export const queryDNS = (params: { name: string; type: string }) =>
  http.get<DNSQueryResult>('/dns/query', { params })

export const disconnectConnection = (id: string) => http.delete(`/connections/${id}`)
export const disconnectAll = () => http.delete('/connections')

// ============ WebSocket 流（自动重连） ============

const wsBase = () => {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  return `${proto}://${location.host}${import.meta.env.BASE_URL}api/clash`
}

/**
 * 订阅流式数据（connections / logs / traffic）
 * 返回 shallowRef 持有最新快照；close() 关闭连接
 */
export const createClashStream = <T>(url: string) => {
  const data = shallowRef<T>()
  const ws = new ReconnectingWebSocket(`${wsBase()}/${url}`)
  ws.onmessage = (ev) => {
    data.value = JSON.parse(ev.data as string) as T
  }
  return {
    data,
    close: () => ws.close()
  }
}

// 便捷订阅：实时连接 / 日志 / 流量
export const subscribeConnections = () => createClashStream<ConnectionsSnapshot>('connections')
export const subscribeLogs = () => createClashStream<LogItem>('logs')
export const subscribeTraffic = () => createClashStream<TrafficSnapshot>('traffic')
// api 层 · Clash 通道(REST / WebSocket)的纯请求函数。
//
// 「Clash 通道」上跑着三种方言(mihomo / sing-box / honk),本文件按方言分区:
//   1. 通用         —— 三种方言都提供
//   2. mihomo 专属  —— mihomo(含 smart 分支)的扩展端点,sing-box 与 honk 没有
//   3. sing-box 的 Clash 兼容 API 专属 —— 仅 sing-box 提供的端点
// honk 实现的是通用分区的子集(没有 /upgrade/ui),故不单列分区,差异见能力表。
// sing-box API(gRPC)是另一条通道,不在这里,见 api/singbox/。
//
// 新增端点时请放进对应分区。是否向用户暴露由 assembly/backend.ts 的能力表决定,
// 本层不做任何后端判断。
import { getUrlFromBackend } from '@/helper/utils'
import { activeBackend } from '@/store/setup'
import type {
  Backend,
  Config,
  DNSQuery,
  NodeRank,
  Proxy,
  ProxyProvider,
  Rule,
  RuleProvider,
} from '@/types'
import { debounce } from 'lodash'
import { debug } from 'three/tsl'


// ==========================================================================
// 两方言共用
// ==========================================================================

// export const fetchClashVersion = () => axios.get<{ version: string }>('/version')

export const fetchProxiesAPI = () => {
  return axios.get<{ proxies: Record<string, Proxy> }>('/proxies')
}

export const selectProxyAPI = (proxyGroup: string, name: string) => {
  return axios.put(`/proxies/${encodeURIComponent(proxyGroup)}`, { name })
}

export const deleteFixedProxyAPI = (proxyGroup: string) => {
  return axios.delete(`/proxies/${encodeURIComponent(proxyGroup)}`)
}

// export const fetchProxyLatencyAPI = (proxyName: string, url: string, timeout: number) => {
//   debugger
//   return axios.get<{ delay: number }>(`/proxies/${encodeURIComponent(proxyName)}/delay`, {
//     params: {
//       url,
//       timeout,
//     },
//   })
// }
export const fetchProxyLatencyAPI = (name: string, url: string, timeout: number) => {

  return http.get<{ delay: number }>(`/proxies/${encodeURIComponent(name)}/delay`, {
    params: { url, timeout }
  })
}

// provider 节点可能不在全局 /proxies 映射中(或与其他 provider 的同名节点冲突),
// 已知所属 provider 时用该端点测指定节点;与 /proxies/{name}/delay 共用内核的
// getProxyDelay,同样返回 { delay }
export const fetchProxyProviderLatencyAPI = (
  proxyName: string,
  url: string,
  timeout: number,
) => {

  return axios.get<{ delay: number }>(
    `/proxies/${encodeURIComponent(proxyName)}/delay`,
    {
      params: {
        url,
        timeout,
      },
    },
  )
}

export const fetchProxyGroupLatencyAPI = (proxyName: string, url: string, timeout: number) => {
  return axios.get<Record<string, number>>(`/group/${encodeURIComponent(proxyName)}/delay`, {
    params: {
      url,
      timeout,
    },
  })
}

export const fetchProxyProviderAPI = () => {
  return axios.get<{ providers: Record<string, ProxyProvider> }>('/providers/proxies')
}

export const updateProxyProviderAPI = (name: string) => {
  return axios.put(`/providers/proxies/${encodeURIComponent(name)}`)
}

export const proxyProviderHealthCheckAPI = (name: string) => {
  return axios.get<Record<string, number>>(
    `/providers/proxies/${encodeURIComponent(name)}/healthcheck`,
    {
      timeout: 15000,
    },
  )
}

export const fetchRulesAPI = () => {
  return axios.get<{ rules: Rule[] }>('/rules')
}

export const fetchRuleProvidersAPI = () => {
  return axios.get<{ providers: Record<string, RuleProvider> }>('/providers/rules')
}

export const updateRuleProviderAPI = (name: string) => {
  return axios.put(`/providers/rules/${encodeURIComponent(name)}`)
}

export const disconnectClashByIdAPI = (id: string) => {
  return axios.delete(`/connections/${id}`)
}

export const disconnectAllClashAPI = () => {
  return axios.delete('/connections')
}

export const getConfigsAPI = () => {
  return axios.get<Config>('/configs')
}

export const patchConfigsAPI = (configs: Record<string, string | boolean | object | number>) => {
  return axios.patch('/configs', configs)
}

export const flushFakeIPAPI = () => {
  return axios.post('/cache/fakeip/flush')
}

export const flushDNSCacheAPI = () => {
  return axios.post('/cache/dns/flush')
}

export const queryDNSAPI = (params: { name: string; type: string }) => {
  return axios.get<DNSQuery>('/dns/query', {
    params,
  })
}

// 面板自升级。mihomo 与 sing-box 的 Clash 兼容 API 都提供,honk 没有(见 dashboardUpgrade)。
export const upgradeUIAPI = () => {
  return axios.post('/upgrade/ui')
}

export const getClashWsUrl = (backend?: Omit<Backend, 'uuid'>, endpoint: string = 'traffic') => {
  const isHttps = backend?.protocol === 'https' || (!backend?.protocol && location.protocol === 'https:')
  const proto = isHttps ? 'wss:' : 'ws:'
  const ep = endpoint.startsWith('/') ? endpoint.slice(1) : endpoint

  const baseUrl = import.meta.env.BASE_URL || '/'
  const cleanBase = baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl
  const apiPath = `${cleanBase}/api/clash`

  return `${proto}//${location.host}${apiPath}/${ep}`
}

export const createClashWebSocket = <T>(url: string, searchParams?: Record<string, string>) => {
  const backend = activeBackend.value
  const rawUrl = getClashWsUrl(backend, url)
  const resurl = new URL(rawUrl)

  if (searchParams) {
    Object.entries(searchParams).forEach(([key, value]) => {
      resurl.searchParams.append(key, value)
    })
  }


  const data = shallowRef<T>()
  const websocket = new ReconnectingWebSocket(resurl.toString())

  const close = () => {
    websocket.close()
  }

  const messageHandler = ({ data: message }: { data: string }) => {
    try {
      data.value = JSON.parse(message)
    } catch {
      // ignore
    }
  }

  websocket.onmessage = url === 'logs' ? messageHandler : debounce(messageHandler, 100)

  return {
    data,
    close,
  }
}

// 连通性探测。包含 HTTP /version 与 WebSocket /traffic 两层探测，
// 确保即使 HTTP 能通，也能及时捕获 Nginx 等反向代理漏配 Upgrade 头的异常状态。
export const probeClashChannel = async (
  backend: Backend,
  timeout: number,
  signal?: AbortSignal,
): Promise<ProbeResult> => {
  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), timeout)
  const onAbort = () => controller.abort()

  signal?.addEventListener('abort', onAbort, { once: true })

  const startAt = Date.now()
  const latency = () => Date.now() - startAt

  try {
    const res = await fetch(`${getUrlFromBackend(backend)}/version`, {
      method: 'GET',
      // headers: {
      //   Authorization: `Bearer ${backend.password}`,
      // },
      signal: controller.signal,
    })

    if (!res.ok) {
      return {
        ok: false,
        latency: latency(),
        kind: res.status === 401 ? 'unauthorized' : 'http',
        message: `HTTP ${res.status}`,
      }
    }

    // 2. HTTP 探测通过后，进一步探测 WebSocket 连通性
    const wsUrl = getClashWsUrl(backend, 'traffic')
    const wsProbe = await probeWebSocket(wsUrl, Math.min(timeout, 3000), controller.signal)
    if (!wsProbe.ok) {
      return {
        ok: false,
        latency: latency(),
        kind: 'websocket',
        message: wsProbe.message || 'WebSocket 握手失败（反向代理未配置 Upgrade 支持）',
      }
    }

    return { ok: true, latency: latency() }
  } catch (e) {
    // 外部取消(切走了 / 组件卸载)不是失败,但调用方已经不看结果了,归入超时即可。
    return {
      ok: false,
      latency: latency(),
      kind: controller.signal.aborted ? 'timeout' : 'network',
      message: e instanceof Error ? e.message : String(e),
    }
  } finally {
    clearTimeout(timeoutId)
    signal?.removeEventListener('abort', onAbort)
  }
}


// ==========================================================================
// mihomo 专属(sing-box 官方版的 Clash 兼容 API 不提供)
// ==========================================================================

// smart 内核的节点权重。是否暴露由数据决定(proxy.type === 'smart'),不走能力表。
export const fetchSmartWeightsAPI = () => {
  return axios.get<{
    message: string
    weights: Record<string, NodeRank[]>
  }>(`/group/weights`)
}

// deprecated
export const fetchSmartGroupWeightsAPI = (proxyName: string) => {
  return axios.get<{
    message: string
    weights: NodeRank[]
  }>(`/group/${encodeURIComponent(proxyName)}/weights`)
}

export const flushSmartGroupWeightsAPI = () => {
  return axios.post(`/cache/smart/flush`)
}

// 按索引批量切换规则启用状态;sing-box 侧走 toggleRuleDisabledSingBoxAPI。
export const toggleRuleDisabledAPI = (data: Record<number, boolean>) => {
  return axios.patch(`/rules/disable`, data)
}

export const blockConnectionByIdAPI = (id: string) => {
  return axios.delete(`/connections/smart/${id}`)
}

export const reloadConfigsAPI = () => {
  return axios.put('/configs?reload=true', { path: '', payload: '' })
}

export const updateConfigsAPI = (
  config: { path?: string; payload?: string },
  force: boolean = false,
) => {
  return axios.put(`/configs${force ? '?force=true' : ''}`, {
    path: config.path || '',
    payload: config.payload || '',
  })
}

export const updateGeoDataAPI = () => {
  return axios.post('/configs/geo')
}

export const upgradeCoreAPI = (type: 'release' | 'alpha' | 'auto') => {
  const url = type === 'auto' ? '/upgrade' : `/upgrade?channel=${type}`

  return axios.post(url)
}

export const restartCoreAPI = () => {
  return axios.post('/restart')
}

// 面板设置同步。/storage/zashboard 是 mihomo 扩展。
export const getStorageAPI = () => {
  return axios.get<Record<string, unknown>>(`/storage/zashboard`)
}

export const setStorageAPI = (value: Record<string, string>) => {
  return axios.put(`/storage/zashboard`, value)
}

export const deleteStorageAPI = () => {
  return axios.delete(`/storage/zashboard`)
}

// ==========================================================================
// sing-box 的 Clash 兼容 API 专属
// ==========================================================================

// sing-box 的规则带稳定 uuid,按 uuid 切换启用状态;mihomo 走 PATCH /rules/disable。
// 两者的选择由响应数据(rule.uuid 是否存在)决定,见 assembly/rules。
export const toggleRuleDisabledSingBoxAPI = (uuid: string) => {
  return axios.put(`/rules/${encodeURIComponent(uuid)}`)
}
