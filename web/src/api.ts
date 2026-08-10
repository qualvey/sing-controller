import axios from 'axios'
import type { AxiosError } from 'axios'

export interface StatusInfo {
  config_path: string
  controller_config: string
  min_port: number
  defaults: {
    inbound_type: string
    outbound_type: string
    listen: string
    listen_port: number
  }
  inbounds: number
  outbounds: number
  rules: number
}

export interface ControllerSettings {
  config: string
  min_port: number
  defaults: {
    inbound_type: string
    outbound_type: string
    listen: string
    listen_port: number
  }
}

export interface Outbound {
  type: string
  tag: string
  server?: string
  server_port?: number
  [key: string]: unknown
}

export interface Inbound {
  type: string
  tag: string
  listen?: string
  listen_port?: number
  [key: string]: unknown
}

export interface RouteRule {
  id?: string
  inbound?: string[]
  network?: string[]
  outbound?: string
  [key: string]: unknown
}

export interface RouteInfo {
  routes: Array<{ id: string; rule: RouteRule }>
  final: string
}

export interface TypesInfo {
  inbounds: string[]
  outbounds: string[]
  endpoints: string[]
  services: string[]
}

const http = axios.create({
  baseURL: '/api',
  timeout: 15000
})

// 非 2xx 响应统一 reject，并提取后端 error 字段，调用处 catch 后展示
http.interceptors.response.use(
  (res) => res,
  (err: AxiosError<{ error?: string }>) => {
    const msg = err.response?.data?.error || err.message || '请求失败'
    return Promise.reject(new Error(msg))
  }
)

export const api = {
  // 状态与配置
  status: () => http.get<StatusInfo>('/status').then((r) => r.data),
  config: () => http.get<unknown>('/config').then((r) => r.data),
  saveConfig: (config: unknown) => http.put('/config', config).then((r) => r.data),
  types: () => http.get<TypesInfo>('/types').then((r) => r.data),

  // controller 设置
  settings: () => http.get<ControllerSettings>('/settings').then((r) => r.data),
  updateSettings: (s: ControllerSettings) => http.put('/settings', s).then((r) => r.data),
  availablePort: (start?: number) =>
    http
      .get<{ port: number; start: number }>('/ports/available', {
        params: start != null ? { start } : {}
      })
      .then((r) => r.data),

  // 工具
  genUuid: () => http.post<{ uuid: string }>('/tools/uuid').then((r) => r.data.uuid),
  genRealityKeypair: () =>
    http.post<{ private_key: string; public_key: string }>('/tools/reality-keypair').then((r) => r.data),
  parseJson: (json: string) => http.post<{ ok: boolean; data: unknown }>('/tools/parse-json', { json }).then((r) => r.data),

  // Outbound CRUD
  outbounds: () => http.get<{ outbounds: Outbound[] }>('/outbounds').then((r) => r.data.outbounds),
  getOutbound: (tag: string) => http.get<Outbound>(`/outbounds/${encodeURIComponent(tag)}`).then((r) => r.data),
  createOutbound: (o: Outbound) => http.post('/outbounds', o).then((r) => r.data),
  updateOutbound: (tag: string, o: Outbound) => http.put(`/outbounds/${encodeURIComponent(tag)}`, o).then((r) => r.data),
  deleteOutbound: (tag: string) => http.delete(`/outbounds/${encodeURIComponent(tag)}`).then((r) => r.data),

  // Inbound CRUD
  inbounds: () => http.get<{ inbounds: Inbound[] }>('/inbounds').then((r) => r.data.inbounds),
  getInbound: (tag: string) => http.get<Inbound>(`/inbounds/${encodeURIComponent(tag)}`).then((r) => r.data),
  createInbound: (i: Inbound) => http.post('/inbounds', i).then((r) => r.data),
  updateInbound: (tag: string, i: Inbound) => http.put(`/inbounds/${encodeURIComponent(tag)}`, i).then((r) => r.data),
  deleteInbound: (tag: string) => http.delete(`/inbounds/${encodeURIComponent(tag)}`).then((r) => r.data),

  // Route 规则 CRUD
  routes: () => http.get<RouteInfo>('/routes').then((r) => r.data),
  createRoute: (rule: RouteRule) => http.post('/routes', rule).then((r) => r.data),
  updateRoute: (id: string, rule: RouteRule) => http.put(`/routes/${encodeURIComponent(id)}`, rule).then((r) => r.data),
  deleteRoute: (id: string) => http.delete(`/routes/${encodeURIComponent(id)}`).then((r) => r.data)
}
