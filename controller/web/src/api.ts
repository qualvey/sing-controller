import axios from 'axios'
import type { AxiosError } from 'axios'

export interface StatusInfo {
  config_path: string
  controller_config: string
  listen: string
  log_level: string
  min_port: number
  defaults: {
    inbound_type: string
    outbound_type: string
    listen: string
    listen_port: number
    attach_to_selector?: boolean
    proxy_selector?: string
  }
  inbounds: number
  outbounds: number
  rules: number
}

export interface ControllerSettings {
  config: string
  listen: string
  log?: {
    level: string
  }
  min_port: number
  reload?: {
    mode: string
    service?: string
    pid_file?: string
    hook?: string
    after_save?: boolean
  }
  defaults: {
    inbound_type: string
    outbound_type: string
    listen: string
    listen_port: number
    attach_to_selector?: boolean
    proxy_selector?: string
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

// 非 2xx 响应统一 reject，并提取后端 error 字段；409 引用冲突时附带 references 列表（前端弹确认框用）
http.interceptors.response.use(
  (res) => res,
  (err: AxiosError<{ error?: string; references?: string[] }>) => {
    const data = err.response?.data
    const msg = data?.error || err.message || '请求失败'
    const e = new Error(msg) as Error & { references?: string[] }
    if (data?.references?.length) e.references = data.references
    return Promise.reject(e)
  }
)

export interface UserMeta {
  name: string
  uuid?: string
  password?: string
  flow?: string
  bound_inbounds?: string[]
}

export const api = {
  // 状态与配置
  status: () => http.get<StatusInfo>('/status').then((r) => r.data),
  config: () => http.get<unknown>('/config').then((r) => r.data),
  saveConfig: (config: unknown) => http.put('/config', config).then((r) => r.data),
  configRaw: () => http.get<string>('/config/raw', { responseType: 'text' }).then((r) => r.data),
  saveConfigRaw: (text: string) =>
    http.put('/config/raw', text, { headers: { 'Content-Type': 'text/plain' } }).then((r) => r.data),
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
  deleteOutbound: (tag: string, force = false) =>
    http.delete(`/outbounds/${encodeURIComponent(tag)}${force ? '?force=true' : ''}`).then((r) => r.data),

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
  deleteRoute: (id: string) => http.delete(`/routes/${encodeURIComponent(id)}`).then((r) => r.data),

  // DNS 管理
  dns: () => http.get('/dns').then((r) => r.data),
  createDnsServer: (s: Record<string, unknown>) => http.post('/dns/servers', s).then((r) => r.data),
  updateDnsServer: (tag: string, s: Record<string, unknown>) =>
    http.put(`/dns/servers/${encodeURIComponent(tag)}`, s).then((r) => r.data),
  deleteDnsServer: (tag: string, force = false) =>
    http.delete(`/dns/servers/${encodeURIComponent(tag)}${force ? '?force=true' : ''}`).then((r) => r.data),
  createDnsRule: (rule: RouteRule) => http.post('/dns/rules', rule).then((r) => r.data),
  updateDnsRule: (id: string, rule: RouteRule) => http.put(`/dns/rules/${encodeURIComponent(id)}`, rule).then((r) => r.data),
  deleteDnsRule: (id: string) => http.delete(`/dns/rules/${encodeURIComponent(id)}`).then((r) => r.data),
  updateDnsOptions: (o: Record<string, unknown>) => http.put('/dns/options', o).then((r) => r.data),

  // 重载 sing-box
  reload: () => http.post('/reload').then((r) => r.data),
  users: () => http.get<{ users: UserMeta[] }>('/users').then((r) => r.data.users || []),
  createUser: (u: UserMeta) => http.post('/users', u).then((r) => r.data),
  updateUser: (name: string, u: UserMeta) => http.put(`/users/${encodeURIComponent(name)}`, u).then((r) => r.data),
  deleteUser: (name: string) => http.delete(`/users/${encodeURIComponent(name)}`).then((r) => r.data),

  // 诊断
  diagnostics: () => http.get<{ diagnostics: Array<{ level: string; message: string }> }>('/diagnostics').then((r) => r.data),

  // 规则集（route.rule_set 段）
  ruleSets: () => http.get<{ rule_sets: Array<{ id: string; rule_set: Record<string, any> }> }>('/rule-sets').then((r) => r.data),
  createRuleSet: (rs: Record<string, unknown>) => http.post('/rule-sets', rs).then((r) => r.data),
  updateRuleSet: (id: string, rs: Record<string, unknown>) =>
    http.put(`/rule-sets/${encodeURIComponent(id)}`, rs).then((r) => r.data),
  deleteRuleSet: (id: string, force = false) =>
    http.delete(`/rule-sets/${encodeURIComponent(id)}${force ? '?force=true' : ''}`).then((r) => r.data),

  // 证书（certificate 段 + certificate_providers）
  certificate: () => http.get<{ certificate: unknown; providers: Array<{ id: string; provider: Record<string, any> }> }>('/certificate').then((r) => r.data),
  saveCertificate: (cert: unknown) => http.put('/certificate', cert).then((r) => r.data),
  createCertProvider: (p: Record<string, unknown>) => http.post('/certificate/providers', p).then((r) => r.data),
  updateCertProvider: (id: string, p: Record<string, unknown>) =>
    http.put(`/certificate/providers/${encodeURIComponent(id)}`, p).then((r) => r.data),
  deleteCertProvider: (id: string, force = false) =>
    http.delete(`/certificate/providers/${encodeURIComponent(id)}${force ? '?force=true' : ''}`).then((r) => r.data)
}
