<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import { useStatusStore } from '../stores/status'
import {
  DNS_RULE_ACTIONS,
  DNS_RULE_FIELDS,
  DNS_RULE_GROUPS,
  DNS_RULE_SUMMARY_ORDER
} from './dnsRuleFields'

const statusStore = useStatusStore()

const activeTab = ref('servers')
const loading = ref(false)
const saving = ref(false)
const servers = ref<Array<Record<string, any>>>([])
const rules = ref<Array<{ id: string; rule: Record<string, any> }>>([])
const outboundTags = ref<string[]>([])
const inboundTags = ref<string[]>([])
const dnsTags = ref<string[]>([])

const serverDialogVisible = ref(false)
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
  rawJson: ''
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
  serverForm.rawJson = ''
}

function buildServer(): Record<string, any> {
  const s: Record<string, any> = { type: serverForm.type, tag: serverForm.tag.trim() }
  if (serverForm.detour) s.detour = serverForm.detour
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
  if (serverForm.rawJson.trim()) {
    let extra: Record<string, any>
    try {
      extra = JSON.parse(serverForm.rawJson.trim())
    } catch (e) {
      throw new Error(`附加字段 JSON 格式错误：${(e as Error).message}`)
    }
    for (const k of Object.keys(extra)) {
      if (!(k in s) && k !== 'type' && k !== 'tag') s[k] = extra[k]
    }
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
  const known = new Set(['type', 'tag', 'server', 'server_port', 'detour', 'path', 'inet4_range', 'inet6_range', 'interface', 'tls'])
  const extra: Record<string, any> = {}
  for (const k of Object.keys(row)) {
    if (!known.has(k)) extra[k] = row[k]
  }
  serverForm.rawJson = Object.keys(extra).length ? JSON.stringify(extra, null, 2) : ''
  serverDialogVisible.value = true
}

const saveServer = async () => {
  if (!serverForm.tag.trim()) {
    ElMessage.warning('请填写 tag')
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
    ElMessage.success('保存成功')
    serverDialogVisible.value = false
    await loadDns()
    await statusStore.refresh()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

const removeServer = async (row: Record<string, any>) => {
  try {
    await ElMessageBox.confirm(`确定删除 DNS server「${row.tag}」？该操作不可恢复。`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  try {
    await api.deleteDnsServer(String(row.tag))
    ElMessage.success('删除成功')
    await loadDns()
    await statusStore.refresh()
  } catch (e) {
    const err = e as Error & { references?: string[] }
    if (err.references?.length) {
      try {
        await ElMessageBox.confirm(
          `DNS server「${row.tag}」被引用：${err.references.join('、')}\\n删除后将自动清除这些引用。确认删除？`,
          '被引用确认',
          { type: 'warning', confirmButtonText: '确认删除', cancelButtonText: '取消' }
        )
      } catch {
        return
      }
      try {
        await api.deleteDnsServer(String(row.tag), true)
        ElMessage.success('删除成功（已清除引用）')
        await loadDns()
        await statusStore.refresh()
      } catch (e2) {
        ElMessage.error((e2 as Error).message || '删除失败')
      }
      return
    }
    ElMessage.error(err.message || '删除失败')
  }
}

// ---------- DNS 规则 ----------
// 规则表单：字段表驱动（含 json 类型字段的字段级 JSON 输入），无附加字段兜底
const ruleForm = reactive<Record<string, any>>({
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
  // action 与参数（route=server 字段模型；其余动作参数各有 JSON 本体）
  const action = ruleForm.action
  if (action === 'reject') {
    rule.action = 'reject'
    if (String(ruleForm.reject_method).trim()) rule.method = String(ruleForm.reject_method).trim()
  } else if (action === 'respond') {
    rule.action = 'respond'
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
  return rule
}

function openCreateRule() {
  isEditRule.value = false
  editingRuleId.value = ''
  resetRuleForm()
  ruleDialogVisible.value = true
}

function openEditRule(row: { id: string; rule: Record<string, any> }) {
  isEditRule.value = true
  editingRuleId.value = row.id
  resetRuleForm()
  const r = row.rule
  for (const f of DNS_RULE_FIELDS) {
    const v = r[f.key]
    if (f.type === 'bool') ruleForm[f.key] = v === true
    else if (f.type === 'json') ruleForm[f.key] = v == null ? '' : JSON.stringify(v, null, 2)
    else if (f.type === 'string' || f.type === 'select') ruleForm[f.key] = v == null ? '' : String(v)
    else ruleForm[f.key] = v == null ? [] : Array.isArray(v) ? v.map(String) : [String(v)]
  }
  const action = typeof r.action === 'string' && r.action ? r.action : 'route'
  ruleForm.action = action
  ruleForm.server = typeof r.server === 'string' ? r.server : ''
  ruleForm.speculative = r.speculative === true
  ruleForm.evaluate_tag = typeof r.tag === 'string' ? r.tag : ''
  ruleForm.evaluate_speculative = r.speculative === true
  ruleForm.reject_method = typeof r.method === 'string' ? r.method : ''
  // route-options / predefined 的动作参数（action 之外的顶层字段）
  if (action === 'route-options' || action === 'predefined') {
    const known = new Set(['action', 'server', 'speculative', ...DNS_RULE_FIELDS.map((f) => f.key)])
    const extra: Record<string, any> = {}
    for (const k of Object.keys(r)) {
      if (!known.has(k)) extra[k] = r[k]
    }
    ruleForm.actionParamsJson = Object.keys(extra).length ? JSON.stringify(extra, null, 2) : ''
  }
  ruleDialogVisible.value = true
}

const saveRule = async () => {
  saving.value = true
  try {
    const payload = buildDnsRule()
    if (isEditRule.value) {
      await api.updateDnsRule(editingRuleId.value, payload)
    } else {
      await api.createDnsRule(payload)
    }
    ElMessage.success('保存成功')
    ruleDialogVisible.value = false
    await loadDns()
    await statusStore.refresh()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

const removeRule = async (row: { id: string; rule: Record<string, any> }) => {
  try {
    await ElMessageBox.confirm('确定删除这条 DNS 规则？该操作不可恢复。', '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  try {
    await api.deleteDnsRule(row.id)
    ElMessage.success('删除成功')
    await loadDns()
    await statusStore.refresh()
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
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
    ElMessage.success('DNS 选项已保存')
    await loadDns()
    await statusStore.refresh()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
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
    ElMessage.error((e as Error).message || '加载 DNS 配置失败')
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

// 列表摘要：按源码字段表（RawDefaultDNSRule）优先级取有值字段 + 其余字段计数
function ruleSummary(r: Record<string, any>): { items: Array<{ k: string; v: string }>; otherCount: number } {
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
  <div class="page">
    <div class="toolbar">
      <el-button type="primary" @click="openCreateServer">新建 Server</el-button>
      <el-button type="primary" @click="openCreateRule">新建规则</el-button>
      <el-button :loading="loading" @click="loadDns">刷新</el-button>
      <span class="hint">DNS 管理：servers（传输类型多态）+ 规则（匹配 → server）+ 基础选项；保存走 sing-box 完整校验</span>
    </div>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="Servers" name="servers">
        <el-table :data="servers" v-loading="loading" border stripe>
          <el-table-column prop="tag" label="tag" min-width="120" />
          <el-table-column prop="type" label="type" width="90" />
          <el-table-column label="地址" min-width="180">
            <template #default="{ row }">
              <span v-if="row.server">{{ row.server }}<template v-if="row.server_port">:{{ row.server_port }}</template></span>
              <span v-else-if="row.inet4_range || row.inet6_range">{{ row.inet4_range || '' }} {{ row.inet6_range || '' }}</span>
              <span v-else class="muted">—</span>
            </template>
          </el-table-column>
          <el-table-column label="TLS" width="70">
            <template #default="{ row }">{{ row.tls?.enabled ? '✓' : '—' }}</template>
          </el-table-column>
          <el-table-column prop="detour" label="detour" width="110">
            <template #default="{ row }">{{ row.detour || '—' }}</template>
          </el-table-column>
          <el-table-column label="操作" width="140" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" link @click="openEditServer(row)">编辑</el-button>
              <el-button size="small" type="danger" link @click="removeServer(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="规则" name="rules">
        <el-table :data="rules" v-loading="loading" border stripe>
          <el-table-column type="index" label="#" width="56" />
          <el-table-column label="规则" min-width="300">
            <template #default="{ row }">
              <div class="rule-cell">
                <template v-if="ruleSummary(row.rule).items.length">
                  <span v-for="it in ruleSummary(row.rule).items" :key="it.k" class="rule-item">
                    <b>{{ it.k }}</b> {{ it.v }}
                  </span>
                  <span v-if="ruleSummary(row.rule).otherCount" class="rule-item muted">+{{ ruleSummary(row.rule).otherCount }} 项</span>
                </template>
                <span v-else class="rule-item muted">全部（无匹配条件）</span>
                <el-tag v-if="row.rule.invert" size="small" type="warning" style="margin-left: 6px">取反</el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="动作" min-width="140">
            <template #default="{ row }">{{ ruleActionText(row.rule) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="140" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" link @click="openEditRule(row)">编辑</el-button>
              <el-button size="small" type="danger" link @click="removeRule(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="选项" name="options">
        <el-form label-width="160px" style="max-width: 560px">
          <el-form-item label="final（默认 server）">
            <el-select v-model="optsForm.final" style="width: 100%" clearable placeholder="选择默认 DNS server（留空则不指定）">
              <el-option v-for="t in dnsTags" :key="t" :label="t" :value="t" />
            </el-select>
          </el-form-item>
          <el-form-item label="strategy（解析策略）">
            <el-select v-model="optsForm.strategy" style="width: 100%" clearable placeholder="不指定">
              <el-option v-for="s in STRATEGY_OPTIONS" :key="s" :label="s" :value="s" />
            </el-select>
          </el-form-item>
          <el-form-item label="timeout">
            <el-input v-model="optsForm.timeout" placeholder="如 5s（留空用默认）" />
          </el-form-item>
          <el-form-item label="disable_cache">
            <el-switch v-model="optsForm.disable_cache" />
            <span class="hint" style="margin-left: 8px">禁用 DNS 缓存</span>
          </el-form-item>
          <el-form-item label="independent_cache">
            <el-switch v-model="optsForm.independent_cache" />
          </el-form-item>
          <el-form-item label="reverse_mapping">
            <el-switch v-model="optsForm.reverse_mapping" />
            <span class="hint" style="margin-left: 8px">IP 反查域名</span>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="saving" @click="saveOptions">保存选项</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>
    </el-tabs>

    <!-- Server 编辑 -->
    <el-dialog v-model="serverDialogVisible" :title="isEditServer ? '编辑 DNS Server' : '新建 DNS Server'" width="560px" :close-on-click-modal="false">
      <el-form label-width="120px">
        <el-form-item label="type" required>
          <el-select v-model="serverForm.type" style="width: 100%">
            <el-option v-for="t in DNS_TYPES" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>
        <el-form-item label="tag" required>
          <el-input v-model="serverForm.tag" placeholder="唯一标识，如 local-dns / cf-doh" />
        </el-form-item>
        <template v-if="serverFieldKeys(serverForm.type).includes('server')">
          <el-form-item label="server">
            <el-input v-model="serverForm.server" placeholder="IP 或域名" />
          </el-form-item>
          <el-form-item label="server_port">
            <el-input-number v-model="serverForm.server_port" :min="1" :max="65535" />
          </el-form-item>
        </template>
        <el-form-item v-if="serverFieldKeys(serverForm.type).includes('tls_server_name')" label="TLS server_name">
          <el-input v-model="serverForm.tls_server_name" placeholder="如 dns.google（启用 TLS 并校验证书）" />
        </el-form-item>
        <el-form-item v-if="serverFieldKeys(serverForm.type).includes('path')" label="path">
          <el-input v-model="serverForm.path" placeholder="如 /dns-query" />
        </el-form-item>
        <el-form-item v-if="serverFieldKeys(serverForm.type).includes('inet4_range')" label="inet4_range">
          <el-input v-model="serverForm.inet4_range" placeholder="如 198.18.0.0/15" />
        </el-form-item>
        <el-form-item v-if="serverFieldKeys(serverForm.type).includes('inet6_range')" label="inet6_range">
          <el-input v-model="serverForm.inet6_range" placeholder="如 fc00::/18" />
        </el-form-item>
        <el-form-item v-if="serverFieldKeys(serverForm.type).includes('interface')" label="interface">
          <el-input v-model="serverForm.interface" placeholder="网卡名" />
        </el-form-item>
        <el-form-item label="detour">
          <el-select v-model="serverForm.detour" style="width: 100%" clearable placeholder="经出站代理（可选）">
            <el-option v-for="t in outboundTags" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>
        <el-form-item label="附加字段 (JSON)">
          <el-input v-model="serverForm.rawJson" type="textarea" :rows="4" class="mono" placeholder='{"client_subnet": "1.2.3.4/24"}' />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="serverDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveServer">保存</el-button>
      </template>
    </el-dialog>

    <!-- 规则编辑 -->
    <el-dialog v-model="ruleDialogVisible" :title="isEditRule ? '编辑 DNS 规则' : '新建 DNS 规则'" width="760px" :close-on-click-modal="false">
      <el-form label-width="150px">
        <el-tabs>
          <el-tab-pane label="匹配条件">
            <template v-for="g in DNS_RULE_GROUPS" :key="g">
              <el-divider content-position="left">{{ g }}</el-divider>
              <el-form-item v-for="f in DNS_RULE_FIELDS.filter((x) => x.group === g)" :key="f.key" :label="f.label">
                <el-select
                  v-if="f.type === 'string-list' || f.type === 'uint-list' || f.type === 'int-list'"
                  v-model="ruleForm[f.key]"
                  multiple
                  filterable
                  allow-create
                  default-first-option
                  style="width: 100%"
                  :placeholder="f.placeholder || '输入后回车添加，可多值'"
                >
                  <el-option v-for="o in f.options ?? (f.key === 'inbound' ? inboundTags : [])" :key="o" :label="o" :value="o" />
                </el-select>
                <el-select
                  v-else-if="f.type === 'select'"
                  v-model="ruleForm[f.key]"
                  style="width: 100%"
                  :placeholder="f.placeholder || '请选择'"
                >
                  <el-option v-for="o in f.options ?? []" :key="o" :label="o" :value="o" />
                </el-select>
                <el-switch v-else-if="f.type === 'bool'" v-model="ruleForm[f.key]" />
                <el-input v-else-if="f.type === 'string'" v-model="ruleForm[f.key]" :placeholder="f.placeholder || '请输入'" />
                <el-input
                  v-else
                  v-model="ruleForm[f.key]"
                  type="textarea"
                  :rows="3"
                  class="mono"
                  :placeholder="f.placeholder || '字段 JSON 对象'"
                />
              </el-form-item>
            </template>
          </el-tab-pane>
          <el-tab-pane label="动作">
            <el-form-item label="action">
              <el-select v-model="ruleForm.action" style="width: 100%">
                <el-option v-for="a in DNS_RULE_ACTIONS" :key="a.value" :label="a.label" :value="a.value" />
              </el-select>
            </el-form-item>
            <template v-if="ruleForm.action === 'route'">
              <el-form-item label="server">
                <el-select v-model="ruleForm.server" style="width: 100%" clearable placeholder="选择 DNS server">
                  <el-option v-for="t in dnsTags" :key="t" :label="t" :value="t" />
                </el-select>
              </el-form-item>
              <el-form-item label="speculative">
                <el-switch v-model="ruleForm.speculative" />
                <span class="hint" style="margin-left: 8px">先返回假结果再修正</span>
              </el-form-item>
            </template>
            <template v-else-if="ruleForm.action === 'evaluate'">
              <el-form-item label="server">
                <el-select v-model="ruleForm.server" style="width: 100%" clearable placeholder="选择 DNS server">
                  <el-option v-for="t in dnsTags" :key="t" :label="t" :value="t" />
                </el-select>
              </el-form-item>
              <el-form-item label="tag">
                <el-input v-model="ruleForm.evaluate_tag" placeholder="评估结果写入的 tag（可选）" />
              </el-form-item>
              <el-form-item label="speculative">
                <el-switch v-model="ruleForm.evaluate_speculative" />
              </el-form-item>
            </template>
            <el-form-item v-else-if="ruleForm.action === 'reject'" label="method">
              <el-input v-model="ruleForm.reject_method" placeholder="拒绝方法（默认 drop，如 reject）" />
            </el-form-item>
            <el-alert
              v-else-if="ruleForm.action === 'respond'"
              type="info"
              :closable="false"
              title="respond 无参数；应答内容用匹配条件的 response_* 字段"
            />
            <el-form-item v-else label="动作参数 (JSON)">
              <el-input
                v-model="ruleForm.actionParamsJson"
                type="textarea"
                :rows="6"
                class="mono"
                :placeholder="ruleForm.action === 'route-options' ? 'route-options 参数对象' : PREDEFINED_PLACEHOLDER"
              />
            </el-form-item>
          </el-tab-pane>
        </el-tabs>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveRule">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}
.hint {
  font-size: 12px;
  color: #909399;
}
.muted {
  color: #c0c4cc;
}
.rule-text {
  font-size: 13px;
}
.rule-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.rule-item {
  font-size: 13px;
}
.rule-item b {
  color: #909399;
  font-weight: 500;
  margin-right: 4px;
}
.rule-item.muted {
  color: #c0c4cc;
}
</style>
