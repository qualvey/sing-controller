<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { api } from '../api'
import SourcePane from '../components/SourcePane.vue'
import ResourceSourceTab from '../components/ResourceSourceTab.vue'
import type { RouteInfo, RouteRule } from '../api'
import { useStatusStore } from '../stores/status'
import { RULE_FIELDS, RULE_FIELD_KEYS, RULE_GROUPS, RULE_SUMMARY_ORDER } from './routeFields'

const statusStore = useStatusStore()

const loading = ref(false)
const outerTab = ref('form')
const saving = ref(false)
const savingFinal = ref(false)
const routes = ref<RouteInfo['routes']>([])
const finalTag = ref('')
const outboundTags = ref<string[]>([])
const inboundTags = ref<string[]>([])

const routeDialogVisible = ref(false)
const srcTab = ref<InstanceType<typeof ResourceSourceTab>>()
const sourceJson = ref('{}')
const isEdit = ref(false)
const editingId = ref('')
const ruleFormRef = ref<FormInstance>()

// sing-box 1.14 RuleAction：route(默认出站)/direct/bypass/reject/hijack-dns/sniff/resolve/route-options
const actionOptions = [
  { value: 'route', label: '出站（route）' },
  { value: 'direct', label: '直连（direct）' },
  { value: 'bypass', label: '绕过（bypass）' },
  { value: 'reject', label: '拒绝（reject）' },
  { value: 'hijack-dns', label: 'DNS 劫持（hijack-dns）' },
  { value: 'sniff', label: '协议嗅探（sniff）' },
  { value: 'resolve', label: 'DNS 解析（resolve）' },
  { value: 'route-options', label: '路由选项（route-options）' }
] as const

const snifferOptions = ['tls', 'http', 'quic', 'dns', 'stun', 'bittorrent', 'dtls', 'ssh', 'rdp', 'ntp']

const ruleForm = reactive<Record<string, unknown>>({
  ruleType: 'default',
  mode: 'and',
  rulesJson: '',
  action: 'route',
  outbound: '',
  sniffers: [] as string[],
  resolve_server: '',
})
// 从 sing-box RawDefaultRule 整理的字段表初始化表单
for (const f of RULE_FIELDS) {
  if (f.type === 'bool') ruleForm[f.key] = false
  else if (f.type === 'select' || f.type === 'string') ruleForm[f.key] = ''
  else ruleForm[f.key] = []
}

const isRouteAction = computed(() => ruleForm.action === 'route')

const ruleRules = computed<FormRules>(() => {
  const rules: FormRules = {}
  if (isRouteAction.value) {
    rules.outbound = [{ required: true, message: '出站（outbound）必选', trigger: 'change' }]
  }
  return rules
})

const rows = computed(() => routes.value.map((r) => ({ id: r.id, ...r.rule })))

function fmtList(v: unknown): string {
  if (Array.isArray(v)) return v.join(', ')
  return v == null ? '—' : String(v)
}

// 字段可选值：inbound 动态取自入站列表，其余用字段表枚举/空（allow-create 可输入）
function fieldOptions(f: (typeof RULE_FIELDS)[number]): string[] {
  if (f.key === 'inbound') return inboundTags.value
  return f.options ?? []
}

// 列表摘要：logical 显示组合信息；default 按优先级取有值字段 + 其余字段计数
function ruleSummary(row: Record<string, unknown>): { items: Array<{ k: string; v: string }>; otherCount: number } {
  if (row.type === 'logical') {
    const count = Array.isArray(row.rules) ? row.rules.length : 0
    return { items: [{ k: 'logical', v: `${String(row.mode || 'and')} · ${count} 条子规则` }], otherCount: 0 }
  }
  const items = RULE_SUMMARY_ORDER.filter((k) => row[k] != null).map((k) => ({ k, v: fmtList(row[k]) }))
  const otherCount = RULE_FIELDS.filter((f) => !RULE_SUMMARY_ORDER.includes(f.key) && row[f.key] != null).length
  return { items, otherCount }
}

// 规则动作展示：route → 出站 tag；其他 → action 名
function actionText(row: Record<string, unknown>): string {
  const action = typeof row.action === 'string' && row.action ? row.action : 'route'
  if (action === 'route') {
    return typeof row.outbound === 'string' && row.outbound ? `出站 → ${row.outbound}` : '出站'
  }
  const label = actionOptions.find((a) => a.value === action)?.label
  return label ? label.split('（')[0] : action
}

const loadRoutes = async () => {
  loading.value = true
  try {
    const data = await api.routes()
    routes.value = data.routes
    finalTag.value = data.final || ''
  } catch (e) {
    ElMessage.error((e as Error).message || '加载路由失败')
  } finally {
    loading.value = false
  }
}

const loadTags = async () => {
  try {
    const [obs, ibs] = await Promise.all([api.outbounds(), api.inbounds()])
    outboundTags.value = obs.map((o) => String(o.tag)).filter(Boolean)
    inboundTags.value = ibs.map((i) => String(i.tag)).filter(Boolean)
  } catch (e) {
    ElMessage.error((e as Error).message || '加载标签列表失败')
  }
}

const handleResult = (res: unknown) => {
  const r = (res ?? {}) as { reload_error?: string; message?: string }
  if (r.reload_error) {
    ElMessage.warning(r.message || `配置已保存，但实例重载失败：${r.reload_error}`)
  } else {
    ElMessage.success('保存成功')
  }
}

// 后端暂无 final 修改端点 → GET /api/config 全量 → 改 route.final → PUT /api/config 整体回写
const saveFinal = async () => {
  if (!finalTag.value) {
    ElMessage.warning('请先选择 final outbound')
    return
  }
  savingFinal.value = true
  try {
    const cfg = (await api.config()) as Record<string, any>
    if (typeof cfg.route !== 'object' || cfg.route === null || Array.isArray(cfg.route)) {
      cfg.route = {}
    }
    cfg.route.final = finalTag.value
    handleResult(await api.saveConfig(cfg))
    await loadRoutes()
    await statusStore.refresh()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存 final 失败')
  } finally {
    savingFinal.value = false
  }
}

const resetForm = () => {
  ruleForm.ruleType = 'default'
  ruleForm.mode = 'and'
  ruleForm.rulesJson = ''
  ruleForm.action = 'route'
  ruleForm.outbound = ''
  ruleForm.sniffers = []
  ruleForm.resolve_server = ''
  for (const f of RULE_FIELDS) {
    if (f.type === 'bool') ruleForm[f.key] = false
    else if (f.type === 'select' || f.type === 'string') ruleForm[f.key] = ''
    else ruleForm[f.key] = []
  }
}

const openCreate = () => {
  isEdit.value = false
  editingId.value = ''
  sourceJson.value = '{}'
  resetForm()
  routeDialogVisible.value = true
  ruleFormRef.value?.clearValidate()
}

const openEdit = (row: RouteRule) => {
  isEdit.value = true
  editingId.value = typeof row.id === 'string' ? row.id : ''
  const rowRec = row as Record<string, unknown>
  const { id: _omit, ...ruleBody } = rowRec
  sourceJson.value = JSON.stringify(ruleBody, null, 2)
  // logical 类型（多态）：mode + 嵌套子规则 + 共用 action
  if (rowRec.type === 'logical') {
    ruleForm.ruleType = 'logical'
    ruleForm.mode = rowRec.mode === 'or' ? 'or' : 'and'
    ruleForm.rulesJson = Array.isArray(rowRec.rules) ? JSON.stringify(rowRec.rules, null, 2) : ''
    ruleForm.invert = rowRec.invert === true
    fillAction(rowRec)
    routeDialogVisible.value = true
    ruleFormRef.value?.clearValidate()
    return
  }
  // 全字段回填（sing-box Listable 单值序列化为字符串，需兼容）
  for (const f of RULE_FIELDS) {
    const v = rowRec[f.key]
    if (f.type === 'bool') ruleForm[f.key] = v === true
    else if (f.type === 'select' || f.type === 'string') ruleForm[f.key] = v == null ? '' : String(v)
    else ruleForm[f.key] = v == null ? [] : Array.isArray(v) ? v.map(String) : [String(v)]
  }
  fillAction(rowRec)
  const extra: Record<string, unknown> = {}
  for (const k of Object.keys(rowRec)) {
    if (k !== 'id' && !RULE_FIELD_KEYS.includes(k) && !['action', 'outbound', 'sniffer', 'server', 'type', 'mode', 'rules'].includes(k)) {
      extra[k] = rowRec[k]
    }
  }
  routeDialogVisible.value = true
  ruleFormRef.value?.clearValidate()
}

// action 回填（default/logical 共用）
function fillAction(rowRec: Record<string, unknown>) {
  const action = typeof rowRec.action === 'string' && rowRec.action ? rowRec.action : 'route'
  ruleForm.action = action
  ruleForm.outbound = typeof rowRec.outbound === 'string' ? rowRec.outbound : ''
  const sniffers = rowRec.sniffer
  ruleForm.sniffers = Array.isArray(sniffers) ? sniffers.map(String) : typeof sniffers === 'string' ? [sniffers] : []
  ruleForm.resolve_server = typeof rowRec.server === 'string' ? rowRec.server : ''
}

function buildRule(): RouteRule {
  const rule: RouteRule = {}
  // logical 类型：mode + 嵌套子规则（JSON）+ 共用 action
  if (ruleForm.ruleType === 'logical') {
    rule.type = 'logical'
    rule.mode = String(ruleForm.mode || 'and')
    let nested: unknown
    try {
      nested = JSON.parse(String(ruleForm.rulesJson).trim() || '[]')
    } catch (e) {
      throw new Error(`子规则 JSON 格式错误：${(e as Error).message}`)
    }
    if (!Array.isArray(nested)) throw new Error('子规则必须是数组')
    rule.rules = nested
    if (ruleForm.invert === true) rule.invert = true
    buildAction(rule)
    return rule
  }
  // 匹配字段：字段表驱动，空值不写入
  for (const f of RULE_FIELDS) {
    const v = ruleForm[f.key]
    if (f.type === 'bool') {
      if (v === true) rule[f.key] = true
      continue
    }
    if (f.type === 'select') {
      if (v !== '' && v != null) rule[f.key] = f.key === 'ip_version' ? Number(v) : v
      continue
    }
    if (f.type === 'string') {
      if (typeof v === 'string' && v.trim()) rule[f.key] = v.trim()
      continue
    }
    if (!Array.isArray(v) || !v.length) continue
    rule[f.key] =
      f.type === 'uint-list' || f.type === 'int-list'
        ? (v as string[]).map((x) => Number(x))
        : [...(v as string[])]
  }
  buildAction(rule)
  return rule
}

// action 构建（default/logical 共用）
// 注意：sing-box 接受 action=bypass/direct 等与 outbound 并存的写法（实测），
// 编辑保存时必须保留 outbound 字段，否则静默丢数据
function buildAction(rule: RouteRule) {
  if (isRouteAction.value) {
    rule.outbound = ruleForm.outbound as string
  } else {
    rule.action = ruleForm.action as string
    if (String(ruleForm.outbound)) rule.outbound = ruleForm.outbound as string
    if (ruleForm.action === 'sniff' && (ruleForm.sniffers as string[]).length) {
      rule.sniffer = [...(ruleForm.sniffers as string[])]
    }
    if (ruleForm.action === 'resolve' && String(ruleForm.resolve_server).trim()) {
      rule.server = String(ruleForm.resolve_server).trim()
    }
  }
}

const save = async () => {
  const valid = await ruleFormRef.value?.validate().then(() => true).catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    const rule = srcTab.value?.isDirty() ? JSON.parse(srcTab.value.getText()) : buildRule()
    const res = isEdit.value ? await api.updateRoute(editingId.value, rule) : await api.createRoute(rule)
    handleResult(res)
    routeDialogVisible.value = false
    await Promise.all([loadRoutes(), statusStore.refresh()])
  } catch (e) {
    ElMessage.error((e as Error).message || '操作失败')
  } finally {
    saving.value = false
  }
}

const remove = async (row: RouteRule) => {
  try {
    await ElMessageBox.confirm('确定删除这条路由规则？该操作不可恢复。', '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  const id = typeof row.id === 'string' ? row.id : ''
  if (!id) return
  try {
    await api.deleteRoute(id)
    ElMessage.success('删除成功')
    await Promise.all([loadRoutes(), statusStore.refresh()])
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}

onMounted(() => {
  loadTags()
  loadRoutes()
})
</script>

<template>
  <div class="page">
    <el-tabs v-model="outerTab">
      <el-tab-pane label="表单" name="form">
    <div class="final-bar">
      <span class="final-label">route.final</span>
      <el-select v-model="finalTag" style="width: 240px" placeholder="选择 final outbound" :disabled="!outboundTags.length">
        <el-option v-for="t in outboundTags" :key="t" :label="t" :value="t" />
      </el-select>
      <el-button type="primary" :loading="savingFinal" :disabled="!outboundTags.length" @click="saveFinal">
        保存 final
      </el-button>
      <span class="hint">未匹配任何规则时的默认出站（整体读取/回写配置）</span>
    </div>

    <div class="toolbar">
      <el-button type="primary" @click="openCreate">新建规则</el-button>
      <el-button :loading="loading" @click="loadRoutes">刷新</el-button>
      <span class="hint">规则模型：rule(匹配条件) → action。默认 action 为出站（route）；reject 拒绝、bypass 绕过、direct 直连</span>
    </div>

    <el-table :data="rows" v-loading="loading" border stripe>
      <el-table-column type="index" label="#" width="56" />
      <el-table-column label="rule" min-width="300">
        <template #default="{ row }">
          <div class="rule-cell">
            <template v-if="ruleSummary(row).items.length">
              <span v-for="it in ruleSummary(row).items" :key="it.k" class="rule-item">
                <b>{{ it.k }}</b> {{ it.v }}
              </span>
              <span v-if="ruleSummary(row).otherCount" class="rule-item muted">+{{ ruleSummary(row).otherCount }} 项</span>
            </template>
            <span v-else class="rule-item muted">全部（无匹配条件）</span>
            <span v-if="row.invert" class="rule-item tag-invert">[取反]</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="action" min-width="150">
        <template #default="{ row }">{{ actionText(row) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" link @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" link @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog
      v-model="routeDialogVisible"
      :title="isEdit ? '编辑规则' : '新建规则'"
      width="760px"
      :close-on-click-modal="false"
    >
      <el-tabs>
        <el-tab-pane label="表单">
      <el-form ref="ruleFormRef" :model="ruleForm" :rules="ruleRules" label-width="130px">
        <el-form-item label="类型">
          <el-select v-model="ruleForm.ruleType" style="width: 100%">
            <el-option label="普通规则（匹配字段）" value="default" />
            <el-option label="逻辑组合（and/or 嵌套子规则）" value="logical" />
          </el-select>
        </el-form-item>
        <template v-if="ruleForm.ruleType === 'logical'">
          <el-form-item label="mode" required>
            <el-select v-model="ruleForm.mode" style="width: 100%">
              <el-option label="and" value="and" />
              <el-option label="or" value="or" />
            </el-select>
          </el-form-item>
          <el-form-item label="invert">
            <el-switch v-model="ruleForm.invert" />
          </el-form-item>
          <el-form-item label="子规则 (JSON)" required>
            <el-input v-model="ruleForm.rulesJson" type="textarea" :rows="8" class="mono" placeholder='[{"rule_set": "gfw"}, {"clash_mode": "direct"}]' />
          </el-form-item>
          <span class="hint">嵌套子规则为 Rule 数组（可再嵌套 logical）；每个子规则也可带 action</span>
        </template>
        <el-tabs>
          <el-tab-pane v-if="ruleForm.ruleType === 'default'" label="匹配条件">
            <template v-for="g in RULE_GROUPS" :key="g">
              <el-divider content-position="left">{{ g }}</el-divider>
              <el-form-item
                v-for="f in RULE_FIELDS.filter((x) => x.group === g)"
                :key="f.key"
                :label="f.label"
                :prop="f.key"
              >
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
                  <el-option v-for="o in fieldOptions(f)" :key="o" :label="o" :value="o" />
                </el-select>
                <el-select
                  v-else-if="f.type === 'select'"
                  v-model="ruleForm[f.key]"
                  style="width: 100%"
                  :placeholder="f.placeholder || '请选择'"
                >
                  <el-option v-for="o in fieldOptions(f)" :key="o" :label="o" :value="o" />
                </el-select>
                <el-switch v-else-if="f.type === 'bool'" v-model="ruleForm[f.key]" />
                <el-input v-else v-model="ruleForm[f.key]" :placeholder="f.placeholder || '请输入'" />
              </el-form-item>
            </template>
          </el-tab-pane>
          <el-tab-pane label="动作">
            <el-form-item label="action" prop="action">
              <el-select v-model="ruleForm.action" style="width: 100%">
                <el-option v-for="a in actionOptions" :key="a.value" :label="a.label" :value="a.value" />
              </el-select>
            </el-form-item>
            <el-form-item v-if="isRouteAction" label="outbound" prop="outbound">
              <el-select v-model="ruleForm.outbound" style="width: 100%" placeholder="选择出站">
                <el-option v-for="t in outboundTags" :key="t" :label="t" :value="t" />
              </el-select>
            </el-form-item>
            <el-form-item v-else-if="ruleForm.action === 'sniff'" label="sniffer">
              <el-select v-model="ruleForm.sniffers" multiple style="width: 100%" placeholder="嗅探协议（可多选）">
                <el-option v-for="s in snifferOptions" :key="s" :label="s" :value="s" />
              </el-select>
            </el-form-item>
            <el-form-item v-else-if="ruleForm.action === 'resolve'" label="DNS server">
              <el-input v-model="ruleForm.resolve_server" placeholder="DNS 服务器 tag（可选，留空用默认）" />
            </el-form-item>
            <el-alert
              type="info"
              :closable="false"
              title="动作说明"
              description="route=默认出站（选 outbound）；direct=直连；bypass=绕过；reject=拒绝；hijack-dns=DNS 劫持；sniff=协议嗅探；resolve=DNS 解析；route-options=路由选项（参数写在附加字段）"
            />
          </el-tab-pane>
        </el-tabs>
      </el-form>
      </el-tab-pane>
      <el-tab-pane label="源码">
        <ResourceSourceTab ref="srcTab" :initial="sourceJson" />
      </el-tab-pane>
    </el-tabs>
      <template #footer>
        <el-button @click="routeDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
      </el-tab-pane>
      <el-tab-pane label="源码" name="source">
        <SourcePane segment="route" @saved="loadRoutes" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.final-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 12px 14px;
  margin-bottom: 14px;
}
.final-label {
  font-weight: 600;
  color: #303133;
}
.hint {
  color: #909399;
  font-size: 12px;
}
.toolbar {
  margin-bottom: 14px;
  display: flex;
  gap: 10px;
  align-items: center;
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
.rule-item.tag-invert {
  color: #e6a23c;
  font-weight: 600;
}
</style>
