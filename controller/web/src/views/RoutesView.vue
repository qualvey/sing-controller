<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { api } from '../api'
import type { RouteInfo, RouteRule } from '../api'
import { useStatusStore } from '../stores/status'

const statusStore = useStatusStore()

const loading = ref(false)
const saving = ref(false)
const savingFinal = ref(false)
const routes = ref<RouteInfo['routes']>([])
const finalTag = ref('')
const outboundTags = ref<string[]>([])
const inboundTags = ref<string[]>([])

const routeDialogVisible = ref(false)
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

const ruleForm = reactive({
  action: 'route' as string,
  inbound: [] as string[],
  network: [] as string[],
  outbound: '',
  sniffers: [] as string[],
  resolve_server: '',
  extraJson: ''
})

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
  ruleForm.action = 'route'
  ruleForm.inbound = []
  ruleForm.network = []
  ruleForm.outbound = ''
  ruleForm.sniffers = []
  ruleForm.resolve_server = ''
  ruleForm.extraJson = ''
}

const openCreate = () => {
  isEdit.value = false
  editingId.value = ''
  resetForm()
  routeDialogVisible.value = true
  ruleFormRef.value?.clearValidate()
}

const openEdit = (row: RouteRule) => {
  isEdit.value = true
  editingId.value = typeof row.id === 'string' ? row.id : ''
  // sing-box Listable 单值序列化为字符串，需兼容回填
  ruleForm.inbound = row.inbound == null ? [] : Array.isArray(row.inbound) ? row.inbound.map(String) : [String(row.inbound)]
  ruleForm.network = row.network == null ? [] : Array.isArray(row.network) ? row.network.map(String) : [String(row.network)]
  const action = typeof row.action === 'string' && row.action ? row.action : 'route'
  ruleForm.action = action
  ruleForm.outbound = typeof row.outbound === 'string' ? row.outbound : ''
  const sniffers = row.sniffer
  ruleForm.sniffers = Array.isArray(sniffers) ? sniffers.map(String) : typeof sniffers === 'string' ? [sniffers] : []
  ruleForm.resolve_server = typeof row.server === 'string' ? row.server : ''
  const extra: Record<string, unknown> = {}
  for (const k of Object.keys(row)) {
    if (
      k !== 'id' && k !== 'inbound' && k !== 'network' && k !== 'outbound' &&
      k !== 'action' && k !== 'sniffer' && k !== 'server'
    ) {
      extra[k] = row[k]
    }
  }
  ruleForm.extraJson = Object.keys(extra).length ? JSON.stringify(extra, null, 2) : ''
  routeDialogVisible.value = true
  ruleFormRef.value?.clearValidate()
}

function buildRule(): RouteRule {
  const rule: RouteRule = {}
  if (ruleForm.inbound.length) rule.inbound = [...ruleForm.inbound]
  if (ruleForm.network.length) rule.network = [...ruleForm.network]
  // action 模型：route(默认) → outbound；其他 → action 字段
  if (isRouteAction.value) {
    rule.outbound = ruleForm.outbound
  } else {
    rule.action = ruleForm.action
    if (ruleForm.action === 'sniff' && ruleForm.sniffers.length) {
      rule.sniffer = [...ruleForm.sniffers]
    }
    if (ruleForm.action === 'resolve' && ruleForm.resolve_server.trim()) {
      rule.server = ruleForm.resolve_server.trim()
    }
  }
  if (ruleForm.extraJson.trim()) {
    let extra: Record<string, unknown>
    try {
      extra = JSON.parse(ruleForm.extraJson.trim())
    } catch (e) {
      throw new Error(`附加字段 JSON 格式错误：${(e as Error).message}`)
    }
    if (typeof extra !== 'object' || extra === null || Array.isArray(extra)) {
      throw new Error('附加字段必须为 JSON 对象')
    }
    Object.assign(rule, extra)
    // 表单中的选择项优先
    if (ruleForm.inbound.length) rule.inbound = [...ruleForm.inbound]
    if (ruleForm.network.length) rule.network = [...ruleForm.network]
    if (isRouteAction.value) {
      rule.outbound = ruleForm.outbound
    } else {
      rule.action = ruleForm.action
    }
  }
  return rule
}

const save = async () => {
  const valid = await ruleFormRef.value?.validate().then(() => true).catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    const rule = buildRule()
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
      <el-table-column label="rule" min-width="220">
        <template #default="{ row }">
          <div class="rule-cell">
            <span v-if="row.inbound" class="rule-item"><b>inbound</b> {{ fmtList(row.inbound) }}</span>
            <span v-if="row.network" class="rule-item"><b>network</b> {{ fmtList(row.network) }}</span>
            <span v-if="!row.inbound && !row.network" class="rule-item muted">全部</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="action" min-width="150">
        <template #default="{ row }">{{ actionText(row) }}</template>
      </el-table-column>
      <el-table-column label="其他" min-width="160">
        <template #default="{ row }">
          <div class="rule-cell">
            <span v-if="row.domain_suffix" class="rule-item"><b>domain_suffix</b> {{ fmtList(row.domain_suffix) }}</span>
            <span v-if="row.ip_cidr" class="rule-item"><b>ip_cidr</b> {{ fmtList(row.ip_cidr) }}</span>
            <span v-if="row.port" class="rule-item"><b>port</b> {{ fmtList(row.port) }}</span>
            <span v-if="!row.domain_suffix && !row.ip_cidr && !row.port" class="rule-item muted">—</span>
          </div>
        </template>
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
      width="640px"
      :close-on-click-modal="false"
    >
      <el-form ref="ruleFormRef" :model="ruleForm" :rules="ruleRules" label-width="130px">
        <el-form-item label="inbound" prop="inbound">
          <el-select v-model="ruleForm.inbound" multiple style="width: 100%" placeholder="匹配的入站（可多选，留空匹配所有）">
            <el-option v-for="t in inboundTags" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>
        <el-form-item label="network" prop="network">
          <el-select v-model="ruleForm.network" multiple style="width: 100%" placeholder="tcp / udp（可多选）">
            <el-option label="tcp" value="tcp" />
            <el-option label="udp" value="udp" />
          </el-select>
        </el-form-item>
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
        <el-form-item label="其他字段 (JSON)">
          <el-input
            v-model="ruleForm.extraJson"
            type="textarea"
            :rows="6"
            class="mono"
            placeholder='{"domain_suffix": [".com"], "ip_cidr": ["0.0.0.0/8"]}'
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="routeDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
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
</style>
