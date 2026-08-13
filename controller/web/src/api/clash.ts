// clash API 客户端（移植自 zashboard，MIT）
// 通过 controller 的 /api/clash/* 反向代理同源访问 sing-box clash API：
// secret 由 controller 注入，前端不接触。
import axios from 'axios'
import ReconnectingWebSocket from 'reconnectingwebsocket'
import { shallowRef } from 'vue'

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
