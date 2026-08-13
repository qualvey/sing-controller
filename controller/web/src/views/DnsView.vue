<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { showToast } from '@/helper/toast'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { PlusIcon, RefreshCw } from 'lucide-vue-next'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import ChipInput from '@/components/common/ChipInput.vue'
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { SelectField, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { api } from '../api'
import { useStatusStore } from '../stores/status'
import SourcePane from '../components/SourcePane.vue'
import ResourceSourceTab from '../components/ResourceSourceTab.vue'
import {
  DNS_RULE_ACTIONS,
  DNS_RULE_FIELDS,
  DNS_RULE_GROUPS,
  DNS_RULE_SUMMARY_ORDER,
  DNS_RULE_TYPES,
  LOGICAL_MODES
} from './dnsRuleFields'

const statusStore = useStatusStore()

const activeTab = ref('servers')
const outerTab = ref('form')
const loading = ref(false)
const saving = ref(false)
const servers = ref<Array<Record<string, any>>>([])
const rules = ref<Array<{ id: string; rule: Record<string, any> }>>([])
const outboundTags = ref<string[]>([])
const inboundTags = ref<string[]>([])
const dnsTags = ref<string[]>([])

const serverDialogVisible = ref(false)
const srcTab = ref<InstanceType<typeof ResourceSourceTab>>()
const sourceJson = ref('{}')
const isEditServer = ref(false)
const editingServerTag = ref('')
const ruleDialogVisible = ref(false)
const isEditRule = ref(false)
const editingRuleId = ref('')

// ---------- DNS transport 类型与字段 ----------
const DNS_TYPES = ['local', 'udp', 'tcp', 'tls', 'https', 'quic', 'h3', 'fakeip', 'hosts', 'dhcp', 'mdns'] as const

function serverFieldKeys(type: string): string[] {
  switch (type) {
    case 'udp':
    case 'tcp':
      return ['server', 'server_port']
    case 'tls':
    case 'quic':
      return ['server', 'server_port', 'tls_server_name']
    case 'https':
    case 'h3':
      return ['server', 'server_port', 'tls_server_name', 'path']
    case 'fakeip':
      return ['inet4_range', 'inet6_range']
    case 'hosts':
      return ['path']
    case 'dhcp':
    case 'mdns':
      return ['interface']
    default:
      return []
  }
}

const serverForm = reactive<Record<string, any>>({
  type: 'udp',
  tag: '',
  server: '',
  server_port: 53,
  detour: '',
  tls_server_name: '',
  path: '',
  inet4_range: '',
  inet6_range: '',
  interface: '',
  predefined_json: '',
  // DialerOptions 常用（option/outbound.go AbstractDialerOptions）
  bind_interface: '',
  connect_timeout: '',
  routing_mark: 0,
  reuse_addr: false,
  udp_fragment: false,
  network_strategy: '',
  network_type: [] as string[],
  // domain_resolver
  dr_server: '',
  dr_timeout: '',
  dr_strategy: '',
  dr_disable_cache: false,
  dr_disable_optimistic_cache: false,
  dr_rewrite_ttl: 0,
  dr_client_subnet: ''
})

const resetServerForm = () => {
  serverForm.type = 'udp'
  serverForm.tag = ''
  serverForm.server = ''
  serverForm.server_port = 53
  serverForm.detour = ''
  serverForm.tls_server_name = ''
  serverForm.path = ''
  serverForm.inet4_range = ''
  serverForm.inet6_range = ''
  serverForm.interface = ''
  serverForm.predefined_json = ''
  serverForm.bind_interface = ''
  serverForm.connect_timeout = ''
  serverForm.routing_mark = 0
  serverForm.reuse_addr = false
  serverForm.udp_fragment = false
  serverForm.network_strategy = ''
  serverForm.network_type = []
  serverForm.dr_server = ''
  serverForm.dr_timeout = ''
  serverForm.dr_strategy = ''
  serverForm.dr_disable_cache = false
  serverForm.dr_disable_optimistic_cache = false
  serverForm.dr_rewrite_ttl = 0
  serverForm.dr_client_subnet = ''
}

function buildServer(): Record<string, any> {
  const s: Record<string, any> = { type: serverForm.type, tag: serverForm.tag.trim() }
  if (serverForm.detour) s.detour = serverForm.detour
  // DialerOptions 常用字段
  if (serverForm.bind_interface.trim()) s.bind_interface = serverForm.bind_interface.trim()
  if (serverForm.connect_timeout.trim()) s.connect_timeout = serverForm.connect_timeout.trim()
  if (serverForm.routing_mark) s.routing_mark = Number(serverForm.routing_mark)
  if (serverForm.reuse_addr) s.reuse_addr = true
  if (serverForm.udp_fragment) s.udp_fragment = true
  if (serverForm.network_strategy) s.network_strategy = serverForm.network_strategy
  if (serverForm.network_type.length) s.network_type = [...serverForm.network_type]
  // domain_resolver：server 必填才创建对象
  if (serverForm.dr_server) {
    const dr: Record<string, any> = { server: serverForm.dr_server }
    if (serverForm.dr_timeout.trim()) dr.timeout = serverForm.dr_timeout.trim()
    if (serverForm.dr_strategy) dr.strategy = serverForm.dr_strategy
    if (serverForm.dr_disable_cache) dr.disable_cache = true
    if (serverForm.dr_disable_optimistic_cache) dr.disable_optimistic_cache = true
    if (serverForm.dr_rewrite_ttl) dr.rewrite_ttl = Number(serverForm.dr_rewrite_ttl)
    if (serverForm.dr_client_subnet.trim()) dr.client_subnet = serverForm.dr_client_subnet.trim()
    s.domain_resolver = dr
  }
  const keys = serverFieldKeys(serverForm.type)
  if (keys.includes('server') && serverForm.server.trim()) s.server = serverForm.server.trim()
  if (keys.includes('server_port') && serverForm.server_port) s.server_port = Number(serverForm.server_port)
  if (keys.includes('tls_server_name') && serverForm.tls_server_name.trim()) {
    s.tls = { enabled: true, server_name: serverForm.tls_server_name.trim() }
  }
  if (keys.includes('path') && serverForm.path.trim()) s.path = serverForm.path.trim()
  if (keys.includes('inet4_range') && serverForm.inet4_range.trim()) s.inet4_range = serverForm.inet4_range.trim()
  if (keys.includes('inet6_range') && serverForm.inet6_range.trim()) s.inet6_range = serverForm.inet6_range.trim()
  if (keys.includes('interface') && serverForm.interface.trim()) s.interface = serverForm.interface.trim()
  // hosts.predefined（复杂 map，字段级 JSON 输入）
  if (serverForm.predefined_json.trim()) {
    let predefined: unknown
    try {
      predefined = JSON.parse(serverForm.predefined_json.trim())
    } catch (e) {
      throw new Error(`predefined JSON 格式错误：${(e as Error).message}`)
    }
    s.predefined = predefined
  }
  return s
}

function openCreateServer() {
  isEditServer.value = false
  editingServerTag.value = ''
  resetServerForm()
  serverDialogVisible.value = true
}

function openEditServer(row: Record<string, any>) {
  isEditServer.value = true
  editingServerTag.value = String(row.tag || '')
  resetServerForm()
  serverForm.type = String(row.type || 'udp')
  serverForm.tag = String(row.tag || '')
  if (row.server != null) serverForm.server = String(row.server)
  if (row.server_port != null) serverForm.server_port = Number(row.server_port)
  if (row.detour != null) serverForm.detour = String(row.detour)
  if (row.tls && typeof row.tls === 'object') {
    serverForm.tls_server_name = String((row.tls as any).server_name || '')
  }
  if (row.path != null) serverForm.path = String(row.path)
  if (row.inet4_range != null) serverForm.inet4_range = String(row.inet4_range)
  if (row.inet6_range != null) serverForm.inet6_range = String(row.inet6_range)
  if (row.interface != null) serverForm.interface = String(row.interface)
  if (row.predefined != null) serverForm.predefined_json = JSON.stringify(row.predefined, null, 2)
  // DialerOptions
  if (row.bind_interface != null) serverForm.bind_interface = String(row.bind_interface)
  if (row.connect_timeout != null) serverForm.connect_timeout = String(row.connect_timeout)
  if (row.routing_mark != null) serverForm.routing_mark = Number(row.routing_mark)
  serverForm.reuse_addr = row.reuse_addr === true
  serverForm.udp_fragment = row.udp_fragment === true
  if (row.network_strategy != null) serverForm.network_strategy = String(row.network_strategy)
  serverForm.network_type = Array.isArray(row.network_type) ? row.network_type.map(String) : row.network_type ? [String(row.network_type)] : []
  // domain_resolver
  const dr = row.domain_resolver
  if (dr && typeof dr === 'object') {
    serverForm.dr_server = typeof dr.server === 'string' ? dr.server : ''
    if (dr.timeout != null) serverForm.dr_timeout = String(dr.timeout)
    if (dr.strategy != null) serverForm.dr_strategy = String(dr.strategy)
    serverForm.dr_disable_cache = dr.disable_cache === true
    serverForm.dr_disable_optimistic_cache = dr.disable_optimistic_cache === true
    if (dr.rewrite_ttl != null) serverForm.dr_rewrite_ttl = Number(dr.rewrite_ttl)
    if (dr.client_subnet != null) serverForm.dr_client_subnet = String(dr.client_subnet)
  }
  sourceJson.value = JSON.stringify(row, null, 2)
  serverDialogVisible.value = true
}

const saveServer = async () => {
  if (!serverForm.tag.trim()) {
    showToast('请填写 tag', 'warning')
    return
  }
  saving.value = true
  try {
    const payload = buildServer()
    if (isEditServer.value) {
      await api.updateDnsServer(editingServerTag.value, payload)
    } else {
      await api.createDnsServer(payload)
    }
    showToast('保存成功', 'success')
    serverDialogVisible.value = false
    await loadDns()
    await statusStore.refresh()
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
  } finally {
    saving.value = false
  }
}

const removeServer = async (row: Record<string, any>) => {
  const first = await showConfirmDialog({
    title: '删除确认',
    message: `确定删除 DNS server「${row.tag}」？该操作不可恢复。`,
    confirmText: '删除',
    confirmButtonClass: 'btn-error'
  })
  if (!first.confirmed) return
  try {
    await api.deleteDnsServer(String(row.tag))
    showToast('删除成功', 'success')
    await loadDns()
    await statusStore.refresh()
  } catch (e) {
    const err = e as Error & { references?: string[] }
    if (err.references?.length) {
      const second = await showConfirmDialog({
        title: '被引用确认',
        message: `DNS server「${row.tag}」被引用：${err.references.join('、')}\n删除后将自动清除这些引用。确认删除？`,
        confirmText: '确认删除',
        confirmButtonClass: 'btn-error'
      })
      if (!second.confirmed) return
      try {
        await api.deleteDnsServer(String(row.tag), true)
        showToast('删除成功（已清除引用）', 'success')
        await loadDns()
        await statusStore.refresh()
      } catch (e2) {
        showToast((e2 as Error).message || '删除失败', 'error')
      }
      return
    }
    showToast(err.message || '删除失败', 'error')
  }
}

// ---------- DNS 规则 ----------
const ruleForm = reactive<Record<string, any>>({
  ruleType: 'default',
  mode: 'and',
  rulesJson: '',
  action: 'route',
  server: '',
  speculative: false,
  evaluate_tag: '',
  evaluate_speculative: false,
  reject_method: '',
  actionParamsJson: ''
})
for (const f of DNS_RULE_FIELDS) {
  if (f.type === 'bool') ruleForm[f.key] = false
  else if (f.type === 'string' || f.type === 'select' || f.type === 'json') ruleForm[f.key] = ''
  else ruleForm[f.key] = []
}

const resetRuleForm = () => {
  ruleForm.ruleType = 'default'
  ruleForm.mode = 'and'
  ruleForm.rulesJson = ''
  ruleForm.action = 'route'
  ruleForm.server = ''
  ruleForm.speculative = false
  ruleForm.evaluate_tag = ''
  ruleForm.evaluate_speculative = false
  ruleForm.reject_method = ''
  ruleForm.actionParamsJson = ''
  for (const f of DNS_RULE_FIELDS) {
    if (f.type === 'bool') ruleForm[f.key] = false
    else if (f.type === 'string' || f.type === 'select' || f.type === 'json') ruleForm[f.key] = ''
    else ruleForm[f.key] = []
  }
}

function parseJsonField(text: string, label: string): unknown {
  if (!text.trim()) return undefined
  try {
    return JSON.parse(text.trim())
  } catch (e) {
    throw new Error(`${label} JSON 格式错误：${(e as Error).message}`)
  }
}

function buildDnsRule(): Record<string, any> {
  const rule: Record<string, any> = {}
  // logical 类型：mode + 嵌套子规则（JSON）+ 共用 action
  if (ruleForm.ruleType === 'logical') {
    rule.type = 'logical'
    rule.mode = ruleForm.mode || 'and'
    let nested: unknown
    try {
      nested = JSON.parse(String(ruleForm.rulesJson).trim() || '[]')
    } catch (e) {
      throw new Error(`子规则 JSON 格式错误：${(e as Error).message}`)
    }
    if (!Array.isArray(nested)) throw new Error('子规则必须是数组')
    rule.rules = nested
    if (ruleForm.invert === true) rule.invert = true
    buildDnsAction(rule)
    return rule
  }
  for (const f of DNS_RULE_FIELDS) {
    const v = ruleForm[f.key]
    if (f.type === 'bool') {
      if (v === true) rule[f.key] = true
      continue
    }
    if (f.type === 'string') {
      if (typeof v === 'string' && v.trim()) rule[f.key] = v.trim()
      continue
    }
    if (f.type === 'select') {
      if (v !== '' && v != null) rule[f.key] = f.key === 'ip_version' ? Number(v) : v
      continue
    }
    if (f.type === 'json') {
      const parsed = parseJsonField(String(v ?? ''), f.label)
      if (parsed !== undefined) rule[f.key] = parsed
      continue
    }
    if (!Array.isArray(v) || !v.length) continue
    rule[f.key] =
      f.type === 'uint-list' || f.type === 'int-list'
        ? (v as string[]).map((x) => Number(x))
        : [...(v as string[])]
  }
  buildDnsAction(rule)
  return rule
}

// action 与参数（route=server 字段模型；其余动作参数各有 JSON 本体）——default/logical 共用
function buildDnsAction(rule: Record<string, any>) {
  const action = ruleForm.action
  if (action === 'reject') {
    rule.action = 'reject'
    if (String(ruleForm.reject_method).trim()) rule.method = String(ruleForm.reject_method).trim()
    if (ruleForm.server) rule.server = String(ruleForm.server)
  } else if (action === 'respond') {
    rule.action = 'respond'
    if (ruleForm.server) rule.server = String(ruleForm.server)
  } else if (action === 'evaluate') {
    rule.action = 'evaluate'
    if (ruleForm.server) rule.server = String(ruleForm.server)
    if (String(ruleForm.evaluate_tag).trim()) rule.tag = String(ruleForm.evaluate_tag).trim()
    if (ruleForm.evaluate_speculative) rule.speculative = true
  } else if (action === 'route-options' || action === 'predefined') {
    rule.action = action
    const parsed = parseJsonField(String(ruleForm.actionParamsJson), '动作参数')
    if (parsed !== undefined) {
      if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
        throw new Error('动作参数必须为 JSON 对象')
      }
      for (const k of Object.keys(parsed)) rule[k] = (parsed as Record<string, unknown>)[k]
    }
  } else {
    // route（默认）
    if (ruleForm.server) rule.server = String(ruleForm.server)
    if (ruleForm.speculative) rule.speculative = true
  }
}

function openCreateRule() {
  isEditRule.value = false
  editingRuleId.value = ''
  sourceJson.value = '{}'
  resetRuleForm()
  ruleDialogVisible.value = true
}

function openEditRule(row: { id: string; rule: Record<string, any> }) {
  isEditRule.value = true
  editingRuleId.value = row.id
  sourceJson.value = JSON.stringify(row.rule, null, 2)
  resetRuleForm()
  const r = row.rule
  // logical 类型（多态）
  if (r.type === 'logical') {
    ruleForm.ruleType = 'logical'
    ruleForm.mode = r.mode === 'or' ? 'or' : 'and'
    ruleForm.rulesJson = Array.isArray(r.rules) ? JSON.stringify(r.rules, null, 2) : ''
    ruleForm.invert = r.invert === true
    fillDnsAction(r)
    ruleDialogVisible.value = true
    return
  }
  for (const f of DNS_RULE_FIELDS) {
    const v = r[f.key]
    if (f.type === 'bool') ruleForm[f.key] = v === true
    else if (f.type === 'json') ruleForm[f.key] = v == null ? '' : JSON.stringify(v, null, 2)
    else if (f.type === 'string' || f.type === 'select') ruleForm[f.key] = v == null ? '' : String(v)
    else ruleForm[f.key] = v == null ? [] : Array.isArray(v) ? v.map(String) : [String(v)]
  }
  fillDnsAction(r)
  ruleDialogVisible.value = true
}

// action 回填（default/logical 共用）
function fillDnsAction(r: Record<string, any>) {
  const action = typeof r.action === 'string' && r.action ? r.action : 'route'
  ruleForm.action = action
  ruleForm.server = typeof r.server === 'string' ? r.server : ''
  ruleForm.speculative = r.speculative === true
  ruleForm.evaluate_tag = typeof r.tag === 'string' ? r.tag : ''
  ruleForm.evaluate_speculative = r.speculative === true
  ruleForm.reject_method = typeof r.method === 'string' ? r.method : ''
  if (action === 'route-options' || action === 'predefined') {
    const known = new Set(['action', 'server', 'speculative', 'type', 'mode', 'rules', ...DNS_RULE_FIELDS.map((f) => f.key)])
    const extra: Record<string, any> = {}
    for (const k of Object.keys(r)) {
      if (!known.has(k)) extra[k] = r[k]
    }
    ruleForm.actionParamsJson = Object.keys(extra).length ? JSON.stringify(extra, null, 2) : ''
  }
}

const saveRule = async () => {
  saving.value = true
  try {
    const payload = srcTab.value?.isDirty() ? JSON.parse(srcTab.value.getText()) : buildDnsRule()
    if (isEditRule.value) {
      await api.updateDnsRule(editingRuleId.value, payload)
    } else {
      await api.createDnsRule(payload)
    }
    showToast('保存成功', 'success')
    ruleDialogVisible.value = false
    await loadDns()
    await statusStore.refresh()
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
  } finally {
    saving.value = false
  }
}

const removeRule = async (row: { id: string; rule: Record<string, any> }) => {
  const { confirmed } = await showConfirmDialog({
    title: '删除确认',
    message: '确定删除这条 DNS 规则？该操作不可恢复。',
    confirmText: '删除',
    confirmButtonClass: 'btn-error'
  })
  if (!confirmed) return
  try {
    await api.deleteDnsRule(row.id)
    showToast('删除成功', 'success')
    await loadDns()
    await statusStore.refresh()
  } catch (e) {
    showToast((e as Error).message || '删除失败', 'error')
  }
}

const PREDEFINED_PLACEHOLDER = '{"rcode": "NXDOMAIN", "answer": []}'
// ---------- DNS 选项 ----------
const optsForm = reactive({
  final: '',
  strategy: '',
  timeout: '',
  disable_cache: false,
  independent_cache: false,
  reverse_mapping: false
})
const STRATEGY_OPTIONS = ['ipv4_only', 'ipv6_only', 'prefer_ipv4', 'prefer_ipv6']

const saveOptions = async () => {
  saving.value = true
  try {
    const patch: Record<string, unknown> = {}
    patch.final = optsForm.final || ''
    if (optsForm.strategy) patch.strategy = optsForm.strategy
    if (optsForm.timeout.trim()) patch.timeout = optsForm.timeout.trim()
    patch.disable_cache = optsForm.disable_cache
    patch.independent_cache = optsForm.independent_cache
    patch.reverse_mapping = optsForm.reverse_mapping
    await api.updateDnsOptions(patch)
    showToast('DNS 选项已保存', 'success')
    await loadDns()
    await statusStore.refresh()
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
  } finally {
    saving.value = false
  }
}

// ---------- 加载 ----------
const loadDns = async () => {
  loading.value = true
  try {
    const data = await api.dns()
    servers.value = Array.isArray(data.servers) ? data.servers : []
    rules.value = Array.isArray(data.rules) ? data.rules : []
    const opts = data.options || {}
    optsForm.final = String(opts.final || '')
    optsForm.strategy = String(opts.strategy || '')
    optsForm.timeout = opts.timeout ? String(opts.timeout) : ''
    optsForm.disable_cache = !!opts.disable_cache
    optsForm.independent_cache = !!opts.independent_cache
    optsForm.reverse_mapping = !!opts.reverse_mapping
    dnsTags.value = servers.value.map((s) => String(s.tag)).filter(Boolean)
  } catch (e) {
    showToast((e as Error).message || '加载 DNS 配置失败', 'error')
  } finally {
    loading.value = false
  }
}

const loadTags = async () => {
  try {
    const [obs, ibs] = await Promise.all([api.outbounds(), api.inbounds()])
    outboundTags.value = obs.map((o) => String(o.tag)).filter(Boolean)
    inboundTags.value = ibs.map((i) => String(i.tag)).filter(Boolean)
  } catch {
    // 标签列表加载失败不阻塞主页面
  }
}

function fmtList(v: unknown): string {
  if (Array.isArray(v)) return v.join(', ')
  return v == null ? '—' : String(v)
}

// 列表摘要：logical 显示组合信息；default 按源码字段表优先级取有值字段 + 其余字段计数
function ruleSummary(r: Record<string, any>): { items: Array<{ k: string; v: string }>; otherCount: number } {
  if (r.type === 'logical') {
    const count = Array.isArray(r.rules) ? r.rules.length : 0
    return { items: [{ k: 'logical', v: `${r.mode || 'and'} · ${count} 条子规则` }], otherCount: 0 }
  }
  const items = DNS_RULE_SUMMARY_ORDER.filter((k) => r[k] != null).map((k) => ({ k, v: fmtList(r[k]) }))
  const otherCount = DNS_RULE_FIELDS.filter((f) => !DNS_RULE_SUMMARY_ORDER.includes(f.key) && r[f.key] != null).length
  return { items, otherCount }
}

function ruleActionText(r: Record<string, any>): string {
  const action = typeof r.action === 'string' && r.action ? r.action : 'route'
  if (action === 'route') return r.server ? `→ ${String(r.server)}` : r.outbound ? `→ ${fmtList(r.outbound)}` : '→ (未指定 server)'
  return action
}

onMounted(() => {
  loadTags()
  loadDns()
})
</script>

<template>
  <div>
    <TabsRoot v-model="outerTab">
      <TabsList>
        <TabsTrigger value="form">表单</TabsTrigger>
        <TabsTrigger value="source">源码</TabsTrigger>
      </TabsList>

      <TabsContent value="form">
        <div class="mb-3.5 flex flex-wrap items-center gap-2.5">
          <button class="btn btn-primary btn-sm" @click="openCreateServer">
            <PlusIcon class="h-4 w-4" />
            新建 Server
          </button>
          <button class="btn btn-primary btn-sm" @click="openCreateRule">
            <PlusIcon class="h-4 w-4" />
            新建规则
          </button>
          <button class="btn btn-ghost btn-sm" :disabled="loading" @click="loadDns">
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
          <span class="text-xs text-[#909399]">DNS 管理：servers（传输类型多态）+ 规则（匹配 → server）+ 基础选项；保存走 sing-box 完整校验</span>
        </div>

        <TabsRoot v-model="activeTab">
          <TabsList>
            <TabsTrigger value="servers">Servers</TabsTrigger>
            <TabsTrigger value="rules">规则</TabsTrigger>
            <TabsTrigger value="options">选项</TabsTrigger>
          </TabsList>

          <TabsContent value="servers">
            <div class="overflow-hidden rounded-lg border border-[#e4e7ed] bg-white dark:border-[#303030] dark:bg-[#1d1e1f]">
              <div v-if="loading && !servers.length" class="p-10 text-center text-sm text-[#909399]">加载中…</div>
              <table v-else class="table table-sm w-full">
                <thead>
                  <tr>
                    <th class="w-[120px]">tag</th>
                    <th class="w-20">type</th>
                    <th>地址</th>
                    <th class="w-14">TLS</th>
                    <th class="w-[110px]">detour</th>
                    <th class="w-[130px] text-right">操作</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in servers" :key="row.tag">
                    <td class="text-xs font-medium">{{ row.tag }}</td>
                    <td class="text-xs">{{ row.type }}</td>
                    <td class="text-xs">
                      <span v-if="row.server">{{ row.server }}<template v-if="row.server_port">:{{ row.server_port }}</template></span>
                      <span v-else-if="row.inet4_range || row.inet6_range">{{ row.inet4_range || '' }} {{ row.inet6_range || '' }}</span>
                      <span v-else class="text-[#c0c4cc]">—</span>
                    </td>
                    <td class="text-xs">{{ row.tls?.enabled ? '✓' : '—' }}</td>
                    <td class="text-xs">{{ row.detour || '—' }}</td>
                    <td class="text-right">
                      <button class="btn btn-ghost btn-xs text-primary" @click="openEditServer(row)">编辑</button>
                      <button class="btn btn-ghost btn-xs text-error" @click="removeServer(row)">删除</button>
                    </td>
                  </tr>
                </tbody>
              </table>
              <div v-if="!loading && !servers.length" class="py-8 text-center text-sm text-[#606266] dark:text-[#a6b0bf]">暂无 DNS server</div>
            </div>
          </TabsContent>

          <TabsContent value="rules">
            <div class="overflow-hidden rounded-lg border border-[#e4e7ed] bg-white dark:border-[#303030] dark:bg-[#1d1e1f]">
              <div v-if="loading && !rules.length" class="p-10 text-center text-sm text-[#909399]">加载中…</div>
              <table v-else class="table table-sm w-full">
                <thead>
                  <tr>
                    <th class="w-14">#</th>
                    <th>规则</th>
                    <th class="w-[140px]">动作</th>
                    <th class="w-[130px] text-right">操作</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, idx) in rules" :key="row.id">
                    <td class="text-xs text-[#909399]">{{ idx + 1 }}</td>
                    <td>
                      <div class="flex flex-col gap-0.5">
                        <template v-if="ruleSummary(row.rule).items.length">
                          <span v-for="it in ruleSummary(row.rule).items" :key="it.k" class="text-[13px]">
                            <b class="mr-1 font-medium text-[#909399]">{{ it.k }}</b> {{ it.v }}
                          </span>
                          <span v-if="ruleSummary(row.rule).otherCount" class="text-[13px] text-[#c0c4cc]">+{{ ruleSummary(row.rule).otherCount }} 项</span>
                        </template>
                        <span v-else class="text-[13px] text-[#c0c4cc]">全部（无匹配条件）</span>
                        <span v-if="row.rule.invert" class="badge badge-warning badge-outline w-fit text-xs">取反</span>
                      </div>
                    </td>
                    <td class="text-xs">{{ ruleActionText(row.rule) }}</td>
                    <td class="text-right">
                      <button class="btn btn-ghost btn-xs text-primary" @click="openEditRule(row)">编辑</button>
                      <button class="btn btn-ghost btn-xs text-error" @click="removeRule(row)">删除</button>
                    </td>
                  </tr>
                </tbody>
              </table>
              <div v-if="!loading && !rules.length" class="py-8 text-center text-sm text-[#606266] dark:text-[#a6b0bf]">暂无 DNS 规则</div>
            </div>
          </TabsContent>

          <TabsContent value="options">
            <div class="grid max-w-[560px] gap-x-4 gap-y-5" style="grid-template-columns: 160px minmax(0, 1fr)">
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">final（默认 server）</label>
              <SelectField v-model="optsForm.final">
                <SelectTrigger><SelectValue placeholder="选择默认 DNS server（留空则不指定）" /></SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="t in dnsTags" :key="t" :value="t">{{ t }}</SelectItem>
                </SelectContent>
              </SelectField>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">strategy（解析策略）</label>
              <SelectField v-model="optsForm.strategy">
                <SelectTrigger><SelectValue placeholder="不指定" /></SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="s in STRATEGY_OPTIONS" :key="s" :value="s">{{ s }}</SelectItem>
                </SelectContent>
              </SelectField>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">timeout</label>
              <input v-model="optsForm.timeout" type="text" class="input input-bordered input-sm w-full" placeholder="如 5s（留空用默认）" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">disable_cache</label>
              <div class="flex items-center gap-2">
                <Switch v-model="optsForm.disable_cache" />
                <span class="text-xs text-[#909399]">禁用 DNS 缓存</span>
              </div>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">independent_cache</label>
              <Switch v-model="optsForm.independent_cache" class="mt-1.5" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">reverse_mapping</label>
              <div class="flex items-center gap-2">
                <Switch v-model="optsForm.reverse_mapping" />
                <span class="text-xs text-[#909399]">IP 反查域名</span>
              </div>
              <span />
              <button class="btn btn-primary btn-sm w-fit" :disabled="saving" @click="saveOptions">保存选项</button>
            </div>
          </TabsContent>
        </TabsRoot>

        <!-- Server 编辑弹窗 -->
        <DialogWrapper v-model="serverDialogVisible" :title="isEditServer ? '编辑 DNS Server' : '新建 DNS Server'" box-class="max-w-[560px]">
          <TabsRoot :model-value="'form'">
            <TabsList>
              <TabsTrigger value="form">表单</TabsTrigger>
              <TabsTrigger value="source">源码</TabsTrigger>
            </TabsList>
            <TabsContent value="form">
              <div class="grid gap-x-4 gap-y-5" style="grid-template-columns: 120px minmax(0, 1fr)">
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">type <span class="text-destructive">*</span></label>
                <SelectField v-model="serverForm.type">
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="t in DNS_TYPES" :key="t" :value="t">{{ t }}</SelectItem>
                  </SelectContent>
                </SelectField>
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tag <span class="text-destructive">*</span></label>
                <input v-model="serverForm.tag" type="text" class="input input-bordered input-sm w-full" placeholder="唯一标识，如 local-dns / cf-doh" />
                <template v-if="serverFieldKeys(serverForm.type).includes('server')">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">server</label>
                  <input v-model="serverForm.server" type="text" class="input input-bordered input-sm w-full" placeholder="IP 或域名" />
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">server_port</label>
                  <input v-model.number="serverForm.server_port" type="number" min="1" max="65535" class="input input-bordered input-sm w-40" />
                </template>
                <template v-if="serverFieldKeys(serverForm.type).includes('tls_server_name')">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">TLS server_name</label>
                  <input v-model="serverForm.tls_server_name" type="text" class="input input-bordered input-sm w-full" placeholder="如 dns.google（启用 TLS 并校验证书）" />
                </template>
                <template v-if="serverFieldKeys(serverForm.type).includes('path')">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">path</label>
                  <input v-model="serverForm.path" type="text" class="input input-bordered input-sm w-full" placeholder="如 /dns-query" />
                </template>
                <template v-if="serverFieldKeys(serverForm.type).includes('inet4_range')">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">inet4_range</label>
                  <input v-model="serverForm.inet4_range" type="text" class="input input-bordered input-sm w-full font-mono" placeholder="如 198.18.0.0/15" />
                </template>
                <template v-if="serverFieldKeys(serverForm.type).includes('inet6_range')">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">inet6_range</label>
                  <input v-model="serverForm.inet6_range" type="text" class="input input-bordered input-sm w-full font-mono" placeholder="如 fc00::/18" />
                </template>
                <template v-if="serverFieldKeys(serverForm.type).includes('interface')">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">interface</label>
                  <input v-model="serverForm.interface" type="text" class="input input-bordered input-sm w-full" placeholder="网卡名" />
                </template>
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">detour</label>
                <SelectField v-model="serverForm.detour">
                  <SelectTrigger><SelectValue placeholder="经出站代理（可选）" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="t in outboundTags" :key="t" :value="t">{{ t }}</SelectItem>
                  </SelectContent>
                </SelectField>

                <div class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]" style="grid-column: 1 / -1">
                  <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />拨号选项（DialerOptions）<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
                </div>
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">bind_interface</label>
                <input v-model="serverForm.bind_interface" type="text" class="input input-bordered input-sm w-full" placeholder="绑定网卡名（可选）" />
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">connect_timeout</label>
                <input v-model="serverForm.connect_timeout" type="text" class="input input-bordered input-sm w-full" placeholder="如 5s（可选）" />
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">routing_mark</label>
                <input v-model.number="serverForm.routing_mark" type="number" min="0" class="input input-bordered input-sm w-40" />
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">reuse_addr</label>
                <Switch v-model="serverForm.reuse_addr" class="mt-1.5" />
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">udp_fragment</label>
                <Switch v-model="serverForm.udp_fragment" class="mt-1.5" />
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">network_strategy</label>
                <SelectField v-model="serverForm.network_strategy">
                  <SelectTrigger><SelectValue placeholder="不指定" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="s in STRATEGY_OPTIONS" :key="s" :value="s">{{ s }}</SelectItem>
                  </SelectContent>
                </SelectField>
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">network_type</label>
                <ChipInput v-model="serverForm.network_type" :suggestions="['wifi', 'cellular', 'ethernet', 'other']" placeholder="wifi/cellular/ethernet/other" />

                <div class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]" style="grid-column: 1 / -1">
                  <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />domain_resolver（server 为域名的必填）<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
                </div>
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">resolver server</label>
                <div class="flex flex-col gap-1">
                  <SelectField v-model="serverForm.dr_server">
                    <SelectTrigger><SelectValue placeholder="选择 DNS server（如 local/ali）" /></SelectTrigger>
                    <SelectContent>
                      <SelectItem v-for="t in dnsTags" :key="t" :value="t">{{ t }}</SelectItem>
                    </SelectContent>
                  </SelectField>
                  <p class="text-xs text-[#606266] dark:text-[#a6b0bf]">server/url 是域名时，用该 resolver 解析域名的 IP（避免自举）</p>
                </div>
                <template v-if="serverForm.dr_server">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">resolver timeout</label>
                  <input v-model="serverForm.dr_timeout" type="text" class="input input-bordered input-sm w-full" placeholder="如 5s" />
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">resolver strategy</label>
                  <SelectField v-model="serverForm.dr_strategy">
                    <SelectTrigger><SelectValue placeholder="不指定" /></SelectTrigger>
                    <SelectContent>
                      <SelectItem v-for="s in STRATEGY_OPTIONS" :key="s" :value="s">{{ s }}</SelectItem>
                    </SelectContent>
                  </SelectField>
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">resolver disable_cache</label>
                  <Switch v-model="serverForm.dr_disable_cache" class="mt-1.5" />
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">disable_optimistic_cache</label>
                  <Switch v-model="serverForm.dr_disable_optimistic_cache" class="mt-1.5" />
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">rewrite_ttl</label>
                  <input v-model.number="serverForm.dr_rewrite_ttl" type="number" min="0" class="input input-bordered input-sm w-40" />
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">client_subnet</label>
                  <input v-model="serverForm.dr_client_subnet" type="text" class="input input-bordered input-sm w-full font-mono" placeholder="如 1.2.3.4/24" />
                </template>
                <template v-if="serverForm.type === 'hosts'">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">predefined (JSON)</label>
                  <textarea v-model="serverForm.predefined_json" rows="4" class="textarea textarea-bordered w-full font-mono text-xs" placeholder='{"example.com": "1.2.3.4"}' />
                </template>
              </div>
            </TabsContent>
            <TabsContent value="source">
              <ResourceSourceTab ref="srcTab" :initial="sourceJson" />
            </TabsContent>
          </TabsRoot>
          <div class="mt-5 flex justify-end gap-2">
            <button class="btn btn-ghost btn-sm" @click="serverDialogVisible = false">取消</button>
            <button class="btn btn-primary btn-sm" :disabled="saving" @click="saveServer">保存</button>
          </div>
        </DialogWrapper>

        <!-- 规则编辑弹窗 -->
        <DialogWrapper v-model="ruleDialogVisible" :title="isEditRule ? '编辑 DNS 规则' : '新建 DNS 规则'" box-class="max-w-[760px]">
          <TabsRoot :model-value="'form'">
            <TabsList>
              <TabsTrigger value="form">表单</TabsTrigger>
              <TabsTrigger value="source">源码</TabsTrigger>
            </TabsList>
            <TabsContent value="form">
              <div class="grid gap-x-4 gap-y-5" style="grid-template-columns: 150px minmax(0, 1fr)">
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">类型</label>
                <SelectField v-model="ruleForm.ruleType">
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="t in DNS_RULE_TYPES" :key="t.value" :value="t.value">{{ t.label }}</SelectItem>
                  </SelectContent>
                </SelectField>

                <template v-if="ruleForm.ruleType === 'logical'">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">mode <span class="text-destructive">*</span></label>
                  <SelectField v-model="ruleForm.mode">
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem v-for="m in LOGICAL_MODES" :key="m" :value="m">{{ m }}</SelectItem>
                    </SelectContent>
                  </SelectField>
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">invert</label>
                  <Switch v-model="ruleForm.invert" class="mt-1.5" />
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">子规则 (JSON) <span class="text-destructive">*</span></label>
                  <div class="flex flex-col gap-1">
                    <textarea v-model="ruleForm.rulesJson" rows="10" class="textarea textarea-bordered w-full font-mono text-xs" placeholder='[{"rule_set": "gfw", "invert": true}, {"clash_mode": "direct"}]' />
                    <p class="text-xs text-[#909399]">嵌套子规则为 DNSRule 数组（可再嵌套 logical）；每个子规则也可带 server 等动作</p>
                  </div>
                </template>

                <template v-if="ruleForm.ruleType === 'default'">
                  <div class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]" style="grid-column: 1 / -1">
                    <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />匹配条件<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
                  </div>
                  <template v-for="g in DNS_RULE_GROUPS" :key="g">
                    <div class="flex items-center gap-2 text-xs font-semibold text-[#909399]" style="grid-column: 1 / -1">
                      <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />{{ g }}<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
                    </div>
                    <template v-for="f in DNS_RULE_FIELDS.filter((x) => x.group === g)" :key="f.key">
                      <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">{{ f.label }}</label>
                      <ChipInput
                        v-if="f.type === 'string-list' || f.type === 'uint-list' || f.type === 'int-list'"
                        v-model="ruleForm[f.key]"
                        :placeholder="f.placeholder || '输入后回车添加，可多值'"
                        :suggestions="f.options ?? (f.key === 'inbound' ? inboundTags : [])"
                      />
                      <SelectField v-else-if="f.type === 'select'" v-model="ruleForm[f.key]">
                        <SelectTrigger><SelectValue :placeholder="f.placeholder || '请选择'" /></SelectTrigger>
                        <SelectContent>
                          <SelectItem v-for="o in f.options ?? []" :key="o" :value="o">{{ o }}</SelectItem>
                        </SelectContent>
                      </SelectField>
                      <Switch v-else-if="f.type === 'bool'" v-model="ruleForm[f.key]" class="mt-1.5" />
                      <input v-else-if="f.type === 'string'" v-model="ruleForm[f.key]" type="text" class="input input-bordered input-sm w-full" :placeholder="f.placeholder || '请输入'" />
                      <textarea v-else v-model="ruleForm[f.key]" rows="3" class="textarea textarea-bordered w-full font-mono text-xs" :placeholder="f.placeholder || '字段 JSON 对象'" />
                    </template>
                  </template>
                </template>

                <div class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]" style="grid-column: 1 / -1">
                  <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />动作<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
                </div>
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">action</label>
                <SelectField v-model="ruleForm.action">
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="a in DNS_RULE_ACTIONS" :key="a.value" :value="a.value">{{ a.label }}</SelectItem>
                  </SelectContent>
                </SelectField>
                <template v-if="ruleForm.action === 'route'">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">server</label>
                  <SelectField v-model="ruleForm.server">
                    <SelectTrigger><SelectValue placeholder="选择 DNS server" /></SelectTrigger>
                    <SelectContent>
                      <SelectItem v-for="t in dnsTags" :key="t" :value="t">{{ t }}</SelectItem>
                    </SelectContent>
                  </SelectField>
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">speculative</label>
                  <div class="flex items-center gap-2">
                    <Switch v-model="ruleForm.speculative" />
                    <span class="text-xs text-[#909399]">先返回假结果再修正</span>
                  </div>
                </template>
                <template v-else-if="ruleForm.action === 'evaluate'">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">server</label>
                  <SelectField v-model="ruleForm.server">
                    <SelectTrigger><SelectValue placeholder="选择 DNS server" /></SelectTrigger>
                    <SelectContent>
                      <SelectItem v-for="t in dnsTags" :key="t" :value="t">{{ t }}</SelectItem>
                    </SelectContent>
                  </SelectField>
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tag</label>
                  <input v-model="ruleForm.evaluate_tag" type="text" class="input input-bordered input-sm w-full" placeholder="评估结果写入的 tag（可选）" />
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">speculative</label>
                  <Switch v-model="ruleForm.evaluate_speculative" class="mt-1.5" />
                </template>
                <template v-else-if="ruleForm.action === 'reject'">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">method</label>
                  <input v-model="ruleForm.reject_method" type="text" class="input input-bordered input-sm w-full" placeholder="拒绝方法（默认 drop，如 reject）" />
                </template>
                <p v-else-if="ruleForm.action === 'respond'" class="text-xs text-[#606266] dark:text-[#a6b0bf]" style="grid-column: 2">
                  respond 无参数；应答内容用匹配条件的 response_* 字段
                </p>
                <template v-else>
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">动作参数 (JSON)</label>
                  <textarea
                    v-model="ruleForm.actionParamsJson"
                    rows="6"
                    class="textarea textarea-bordered w-full font-mono text-xs"
                    :placeholder="ruleForm.action === 'route-options' ? 'route-options 参数对象' : PREDEFINED_PLACEHOLDER"
                  />
                </template>
              </div>
            </TabsContent>
            <TabsContent value="source">
              <ResourceSourceTab ref="srcTab" :initial="sourceJson" />
            </TabsContent>
          </TabsRoot>
          <div class="mt-5 flex justify-end gap-2">
            <button class="btn btn-ghost btn-sm" @click="ruleDialogVisible = false">取消</button>
            <button class="btn btn-primary btn-sm" :disabled="saving" @click="saveRule">保存</button>
          </div>
        </DialogWrapper>
      </TabsContent>

      <TabsContent value="source">
        <SourcePane segment="dns" @saved="loadDns" />
      </TabsContent>
    </TabsRoot>
  </div>
</template>
