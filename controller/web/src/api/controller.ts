import axios, { type AxiosError } from 'axios'

export interface CoreStatusInfo {
  alive: boolean
  type?: string
  version?: string
  latency_ms?: number
  error?: string
}

export interface StatusInfo {
  config_path: string
  config_exists?: boolean
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
  core?: CoreStatusInfo
  core_alive?: boolean
  core_version?: string
  core_type?: string
  core_error?: string
  tun_access_denied?: boolean
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
  attach_groups?: string[]
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
  routes: Array<{ id: string; rule: RouteRule; disabled?: boolean }>
  final: string
}

export interface TypesInfo {
  inbounds: string[]
  outbounds: string[]
  endpoints: string[]
  services: string[]
}

export interface UserMeta {
  name: string
  uuid?: string
  password?: string
  flow?: string
  bound_inbounds?: string[]
}

export interface DiagnosticItem {
  level: 'error' | 'warning' | 'info'
  message: string
}

export interface DnsInfo {
  servers: Array<Record<string, unknown>>
  rules: Array<{ id: string; rule: RouteRule }>
  options: {
    final?: string
    strategy?: string
    timeout?: string
    disable_cache?: boolean
    independent_cache?: boolean
    reverse_mapping?: boolean
    client_subnet?: string
  }
}

export interface CertConfigInfo {
  certificate: Record<string, unknown> | null
  providers: Array<{ id: string; provider: Record<string, unknown> }>
}

const http = axios.create({
  // 与 vite base 对齐：根路径部署 -> '/api'；子路径部署（VITE_BASE_URL=/webui/）-> '/webui/api'
  baseURL: import.meta.env.BASE_URL + 'api',
  timeout: 15000,
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
  },
)

/**
 * sing-controller 专有配置管理 REST API 客户端
 * 严格对齐 API.md 契约规范
 */
export const api = {
  // 健康检查与状态/配置
  healthz: () =>
    http.get<{ status: string }>('/healthz', { baseURL: window.location.origin }).then((r) => r.data),
  status: () => http.get<StatusInfo>('/status').then((r) => r.data),
  config: () => http.get<unknown>('/config').then((r) => r.data),
  saveConfig: (config: unknown) => http.put('/config', config).then((r) => r.data),
  configRaw: () => http.get<string>('/config/raw', { responseType: 'text' }).then((r) => r.data),
  saveConfigRaw: (text: string) =>
    http.put('/config/raw', text, { headers: { 'Content-Type': 'text/plain' } }).then((r) => r.data),
  schema: () => http.get<Record<string, unknown>>('/schema').then((r) => r.data),
  types: () => http.get<TypesInfo>('/types').then((r) => r.data),

  // Controller 设置
  settings: () => http.get<ControllerSettings>('/settings').then((r) => r.data),
  updateSettings: (s: ControllerSettings) => http.put('/settings', s).then((r) => r.data),
  availablePort: (start?: number) =>
    http
      .get<{ port: number; start: number }>('/ports/available', {
        params: start != null ? { start } : {},
      })
      .then((r) => r.data),

  // 系统与权限工具
  elevateProcess: () => http.post<{ message: string }>('/system/elevate').then((r) => r.data),
  disableTun: () => http.post<{ message: string }>('/system/disable-tun').then((r) => r.data),

  // 工具
  genUuid: () => http.post<{ uuid: string }>('/tools/uuid').then((r) => r.data.uuid),
  genRealityKeypair: () =>
    http.post<{ private_key: string; public_key: string }>('/tools/reality-keypair').then((r) => r.data),
  genPassword: () => http.post<{ password: string }>('/tools/password').then((r) => r.data.password),
  parseJson: (json: string) =>
    http.post<{ ok: boolean; data: unknown }>('/tools/parse-json', { json }).then((r) => r.data),
  parseNode: (text: string) =>
    http.post<{ ok: boolean; data: unknown }>('/tools/parse-node', { text }).then((r) => r.data),

  // Outbound CRUD
  outbounds: () => http.get<{ outbounds: Outbound[] }>('/outbounds').then((r) => r.data.outbounds),
  getOutbound: (tag: string) =>
    http.get<Outbound>(`/outbounds/${encodeURIComponent(tag)}`).then((r) => r.data),
  createOutbound: (o: Outbound, reload = true) =>
    http.post(`/outbounds${reload ? '?reload=true' : ''}`, o).then((r) => r.data),
  updateOutbound: (tag: string, o: Outbound, reload = true) =>
    http.put(`/outbounds/${encodeURIComponent(tag)}${reload ? '?reload=true' : ''}`, o).then((r) => r.data),
  deleteOutbound: (tag: string, force = false, reload = true) => {
    const params = new URLSearchParams()
    if (force) params.set('force', 'true')
    if (reload) params.set('reload', 'true')
    const qs = params.toString() ? `?${params.toString()}` : ''
    return http.delete(`/outbounds/${encodeURIComponent(tag)}${qs}`).then((r) => r.data)
  },

  // Inbound CRUD
  inbounds: () => http.get<{ inbounds: Inbound[] }>('/inbounds').then((r) => r.data.inbounds),
  getInbound: (tag: string) =>
    http.get<Inbound>(`/inbounds/${encodeURIComponent(tag)}`).then((r) => r.data),
  createInbound: (i: Inbound) => http.post('/inbounds', i).then((r) => r.data),
  updateInbound: (tag: string, i: Inbound) =>
    http.put(`/inbounds/${encodeURIComponent(tag)}`, i).then((r) => r.data),
  deleteInbound: (tag: string) =>
    http.delete(`/inbounds/${encodeURIComponent(tag)}`).then((r) => r.data),

  // Route 规则 CRUD
  routes: () => http.get<RouteInfo>('/routes').then((r) => r.data),
  getRoute: (id: string) =>
    http
      .get<{ id: string; rule: RouteRule; disabled?: boolean }>(`/routes/${encodeURIComponent(id)}`)
      .then((r) => r.data),
  createRoute: (rule: RouteRule) => http.post('/routes', rule).then((r) => r.data),
  updateRoute: (id: string, rule: RouteRule) =>
    http.put(`/routes/${encodeURIComponent(id)}`, rule).then((r) => r.data),
  deleteRoute: (id: string) =>
    http.delete(`/routes/${encodeURIComponent(id)}`).then((r) => r.data),
  toggleRouteDisabled: (id: string, disabled?: boolean) =>
    http.put(`/routes/${encodeURIComponent(id)}/toggle`, { disabled }).then((r) => r.data),

  // DNS 管理
  dns: () => http.get<DnsInfo>('/dns').then((r) => r.data),
  createDnsServer: (s: Record<string, unknown>) => http.post('/dns/servers', s).then((r) => r.data),
  updateDnsServer: (tag: string, s: Record<string, unknown>) =>
    http.put(`/dns/servers/${encodeURIComponent(tag)}`, s).then((r) => r.data),
  deleteDnsServer: (tag: string, force = false) =>
    http.delete(`/dns/servers/${encodeURIComponent(tag)}${force ? '?force=true' : ''}`).then((r) => r.data),
  createDnsRule: (rule: RouteRule) => http.post('/dns/rules', rule).then((r) => r.data),
  updateDnsRule: (id: string, rule: RouteRule) =>
    http.put(`/dns/rules/${encodeURIComponent(id)}`, rule).then((r) => r.data),
  deleteDnsRule: (id: string) =>
    http.delete(`/dns/rules/${encodeURIComponent(id)}`).then((r) => r.data),
  updateDnsOptions: (opts: Record<string, unknown>) =>
    http.put('/dns/options', opts).then((r) => r.data),
  dnsLookup: (domains: string[]) =>
    http
      .post<{ results: Record<string, string> }>('/dns/lookup', { domains })
      .then((r) => r.data.results),

  // 配置诊断
  diagnostics: () =>
    http.get<{ diagnostics: DiagnosticItem[] }>('/diagnostics').then((r) => r.data),

  // 重载 sing-box
  reload: () =>
    http.post<{ reloaded: boolean; mode: string }>('/reload').then((r) => r.data),

  // 规则集（route.rule_set 段）
  ruleSets: () =>
    http
      .get<{ rule_sets: Array<{ id: string; rule_set: Record<string, unknown> }> }>('/rule-sets')
      .then((r) => r.data),
  createRuleSet: (rs: Record<string, unknown>) => http.post('/rule-sets', rs).then((r) => r.data),
  updateRuleSet: (id: string, rs: Record<string, unknown>) =>
    http.put(`/rule-sets/${encodeURIComponent(id)}`, rs).then((r) => r.data),
  deleteRuleSet: (id: string, force = false) =>
    http.delete(`/rule-sets/${encodeURIComponent(id)}${force ? '?force=true' : ''}`).then((r) => r.data),

  // 证书管理
  certificate: () => http.get<CertConfigInfo>('/certificate').then((r) => r.data),
  saveCertificate: (c: Record<string, unknown> | null) => http.put('/certificate', c).then((r) => r.data),
  createCertProvider: (p: Record<string, unknown>) =>
    http.post('/certificate/providers', p).then((r) => r.data),

  updateCertProvider: (id: string, p: Record<string, unknown>) =>
    http.put(`/certificate/providers/${encodeURIComponent(id)}`, p).then((r) => r.data),
  deleteCertProvider: (id: string, force = false) =>
    http.delete(`/certificate/providers/${encodeURIComponent(id)}${force ? '?force=true' : ''}`).then((r) => r.data),

  // 用户池
  users: () => http.get<{ users: UserMeta[] }>('/users').then((r) => r.data.users),
  getUser: (name: string) => http.get<UserMeta>(`/users/${encodeURIComponent(name)}`).then((r) => r.data),
  createUser: (u: UserMeta) => http.post('/users', u).then((r) => r.data),
  updateUser: (name: string, u: UserMeta) =>
    http.put(`/users/${encodeURIComponent(name)}`, u).then((r) => r.data),
  deleteUser: (name: string) =>
    http.delete(`/users/${encodeURIComponent(name)}`).then((r) => r.data),

  // Core Supervisor (进程生命周期 & 日志)
  coreStatus: () => http.get<CoreSupervisorStatus>('/core/status').then((r) => r.data),
  coreStart: () => http.post<{ success: boolean; message: string }>('/core/start').then((r) => r.data),
  coreStop: () => http.post<{ success: boolean; message: string }>('/core/stop').then((r) => r.data),
  coreRestart: () => http.post<{ success: boolean; message: string }>('/core/restart').then((r) => r.data),
  coreLogs: (limit = 100) =>
    http.get<{ logs: CoreLogLine[]; count: number }>(`/core/logs?limit=${limit}`).then((r) => r.data),

  // Provider (节点源与订阅管理)
  providers: () => http.get<{ providers: NodeProvider[] }>('/providers').then((r) => r.data.providers),
  getProvider: (id: string) => http.get<NodeProvider>(`/providers/${encodeURIComponent(id)}`).then((r) => r.data),
  createProvider: (p: Partial<NodeProvider>) => http.post<NodeProvider>('/providers', p).then((r) => r.data),
  updateProvider: (id: string, p: Partial<NodeProvider>) =>
    http.put(`/providers/${encodeURIComponent(id)}`, p).then((r) => r.data),
  deleteProvider: (id: string) =>
    http.delete(`/providers/${encodeURIComponent(id)}`).then((r) => r.data),
  refreshProvider: (id: string) =>
    http.post<{ success: boolean; node_count: number }>(`/providers/${encodeURIComponent(id)}/refresh`).then((r) => r.data),
  getProviderNodes: (tag: string) =>
    http.get<{ tag: string; count: number; outbounds: any[] }>(`/providers/${encodeURIComponent(tag)}/nodes`).then((r) => r.data),

  // Profile (配置组方案)
  profiles: () => http.get<{ profiles: ConfigProfile[] }>('/profiles').then((r) => r.data.profiles),
  getProfile: (id: string) => http.get<ConfigProfile>(`/profiles/${encodeURIComponent(id)}`).then((r) => r.data),
  createProfile: (p: Partial<ConfigProfile>) => http.post<ConfigProfile>('/profiles', p).then((r) => r.data),
  updateProfile: (id: string, p: Partial<ConfigProfile>) =>
    http.put(`/profiles/${encodeURIComponent(id)}`, p).then((r) => r.data),
  deleteProfile: (id: string) =>
    http.delete(`/profiles/${encodeURIComponent(id)}`).then((r) => r.data),
  applyProfile: (id: string) =>
    http.post<{ success: boolean; message: string }>(`/profiles/${encodeURIComponent(id)}/apply`).then((r) => r.data),
}

export const controllerApi = api
export default api

export interface NodeProvider {
  id: string
  tag: string
  name: string
  type: 'remote' | 'file' | 'inline'
  url?: string
  format: 'auto' | 'sing-box' | 'clash' | 'base64'
  update_interval_min: number
  last_updated?: string
  last_etag?: string
  node_count: number
  enabled: boolean
  last_error?: string
  created_at?: string
  updated_at?: string
}

export interface ConfigProfile {
  id: string
  tag: string
  name: string
  description?: string
  provider_tags: string[]
  target_selectors: string[]
  auto_reload: boolean
  is_active: boolean
  created_at?: string
  updated_at?: string
}

export interface CoreSupervisorStatus {
  managed: boolean
  state: 'stopped' | 'starting' | 'running' | 'stopping' | 'crashed' | 'error' | 'unmanaged'
  pid?: number
  uptime_sec?: number
  started_at?: string
  stopped_at?: string
  exit_code?: number
  last_error?: string
  binary_path?: string
  config_path?: string
  restarts?: number
}

export interface CoreLogLine {
  timestamp: string
  source: 'stdout' | 'stderr' | 'system'
  message: string
}

