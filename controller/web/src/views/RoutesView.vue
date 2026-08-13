<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { showToast } from '@/helper/toast'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { PlusIcon, RefreshCw } from 'lucide-vue-next'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import ChipInput from '@/components/common/ChipInput.vue'
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { SelectRoot, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { api } from '../api'
import SourcePane from '../components/SourcePane.vue'
import ResourceSourceTab from '../components/ResourceSourceTab.vue'
import type { RouteInfo, RouteRule } from '../api'
import { useStatusStore } from '../stores/status'
import { RULE_FIELDS, RULE_GROUPS, RULE_SUMMARY_ORDER } from './routeFields'

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

const ruleForm = reactive<Record<string, any>>({
  ruleType: 'default',
  mode: 'and',
  rulesJson: '',
  action: 'route',
  outbound: '',
  sniffers: [] as string[],
  resolve_server: ''
})
// 从 sing-box RawDefaultRule 整理的字段表初始化表单
for (const f of RULE_FIELDS) {
  if (f.type === 'bool') ruleForm[f.key] = false
  else if (f.type === 'select' || f.type === 'string') ruleForm[f.key] = ''
  else ruleForm[f.key] = []
}

const isRouteAction = computed(() => ruleForm.action === 'route')

const rows = computed<any[]>(() => routes.value.map((r) => ({ id: r.id, ...r.rule })))

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
    showToast((e as Error).message || '加载路由失败', 'error')
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
    showToast((e as Error).message || '加载标签列表失败', 'error')
  }
}

const handleResult = (res: unknown) => {
  const r = (res ?? {}) as { reload_error?: string; message?: string }
  if (r.reload_error) {
    showToast(r.message || `配置已保存，但实例重载失败：${r.reload_error}`, 'warning')
  } else {
    showToast('保存成功', 'success')
  }
}

// 后端暂无 final 修改端点 → GET /api/config 全量 → 改 route.final → PUT /api/config 整体回写
const saveFinal = async () => {
  if (!finalTag.value) {
    showToast('请先选择 final outbound', 'warning')
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
    showToast((e as Error).message || '保存 final 失败', 'error')
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
  routeDialogVisible.value = true
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
  // 校验：route 动作必须选 outbound
  if (isRouteAction.value && !String(ruleForm.outbound)) {
    showToast('出站（outbound）必选', 'warning')
    return
  }
  saving.value = true
  try {
    const rule = srcTab.value?.isDirty() ? JSON.parse(srcTab.value.getText()) : buildRule()
    const res = isEdit.value ? await api.updateRoute(editingId.value, rule) : await api.createRoute(rule)
    handleResult(res)
    routeDialogVisible.value = false
    await Promise.all([loadRoutes(), statusStore.refresh()])
  } catch (e) {
    showToast((e as Error).message || '操作失败', 'error')
  } finally {
    saving.value = false
  }
}

const remove = async (row: RouteRule) => {
  const { confirmed } = await showConfirmDialog({
    title: '删除确认',
    message: '确定删除这条路由规则？该操作不可恢复。',
    confirmText: '删除',
    confirmButtonClass: 'btn-error'
  })
  if (!confirmed) return
  const id = typeof row.id === 'string' ? row.id : ''
  if (!id) return
  try {
    await api.deleteRoute(id)
    showToast('删除成功', 'success')
    await Promise.all([loadRoutes(), statusStore.refresh()])
  } catch (e) {
    showToast((e as Error).message || '删除失败', 'error')
  }
}

onMounted(() => {
  loadTags()
  loadRoutes()
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
        <div class="mb-3.5 flex items-center gap-2.5 rounded-lg border border-[#e4e7ed] bg-white px-4 py-3 dark:border-[#303030] dark:bg-[#1d1e1f]">
          <span class="font-semibold text-[#303133] dark:text-[#e5eaf3]">route.final</span>
          <SelectRoot v-model="finalTag" class="w-60" :disabled="!outboundTags.length">
            <SelectTrigger class="w-60"><SelectValue placeholder="选择 final outbound" /></SelectTrigger>
            <SelectContent>
              <SelectItem v-for="t in outboundTags" :key="t" :value="t">{{ t }}</SelectItem>
            </SelectContent>
          </SelectRoot>
          <button class="btn btn-primary btn-sm" :disabled="savingFinal || !outboundTags.length" @click="saveFinal">保存 final</button>
          <span class="text-xs text-[#909399]">未匹配任何规则时的默认出站（整体读取/回写配置）</span>
        </div>

        <div class="mb-3.5 flex flex-wrap items-center gap-2.5">
          <button class="btn btn-primary btn-sm" @click="openCreate">
            <PlusIcon class="h-4 w-4" />
            新建规则
          </button>
          <button class="btn btn-ghost btn-sm" :disabled="loading" @click="loadRoutes">
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
          <span class="text-xs text-[#909399]">规则模型：rule(匹配条件) → action。默认 action 为出站（route）；reject 拒绝、bypass 绕过、direct 直连</span>
        </div>

        <div class="overflow-hidden rounded-lg border border-[#e4e7ed] bg-white dark:border-[#303030] dark:bg-[#1d1e1f]">
          <div v-if="loading && !rows.length" class="p-10 text-center text-sm text-[#909399]">加载中…</div>
          <table v-else class="table table-sm w-full">
            <thead>
              <tr>
                <th class="w-14">#</th>
                <th>rule</th>
                <th class="w-[150px]">action</th>
                <th class="w-[130px] text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, idx) in rows" :key="row.id">
                <td class="text-xs text-[#909399]">{{ idx + 1 }}</td>
                <td>
                  <div class="flex flex-col gap-0.5">
                    <template v-if="ruleSummary(row).items.length">
                      <span v-for="it in ruleSummary(row).items" :key="it.k" class="text-[13px]">
                        <b class="mr-1 font-medium text-[#909399]">{{ it.k }}</b> {{ it.v }}
                      </span>
                      <span v-if="ruleSummary(row).otherCount" class="text-[13px] text-[#c0c4cc]">+{{ ruleSummary(row).otherCount }} 项</span>
                    </template>
                    <span v-else class="text-[13px] text-[#c0c4cc]">全部（无匹配条件）</span>
                    <span v-if="row.invert" class="text-[13px] font-semibold text-[#e6a23c]">[取反]</span>
                  </div>
                </td>
                <td class="text-xs">{{ actionText(row) }}</td>
                <td class="text-right">
                  <button class="btn btn-ghost btn-xs text-primary" @click="openEdit(row)">编辑</button>
                  <button class="btn btn-ghost btn-xs text-error" @click="remove(row)">删除</button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!loading && !rows.length" class="py-8 text-center text-sm text-[#606266] dark:text-[#a6b0bf]">暂无规则</div>
        </div>
      </TabsContent>

      <TabsContent value="source">
        <SourcePane segment="route" @saved="loadRoutes" />
      </TabsContent>
    </TabsRoot>

    <!-- 新建/编辑弹窗 -->
    <DialogWrapper v-model="routeDialogVisible" :title="isEdit ? '编辑规则' : '新建规则'" box-class="max-w-[760px]">
      <TabsRoot :model-value="'form'">
        <TabsList>
          <TabsTrigger value="form">表单</TabsTrigger>
          <TabsTrigger value="source">源码</TabsTrigger>
        </TabsList>
        <TabsContent value="form">
          <div class="grid gap-x-4 gap-y-5" style="grid-template-columns: 130px minmax(0, 1fr)">
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">类型</label>
            <SelectRoot v-model="ruleForm.ruleType">
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="default">普通规则（匹配字段）</SelectItem>
                <SelectItem value="logical">逻辑组合（and/or 嵌套子规则）</SelectItem>
              </SelectContent>
            </SelectRoot>

            <template v-if="ruleForm.ruleType === 'logical'">
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">mode <span class="text-destructive">*</span></label>
              <SelectRoot v-model="ruleForm.mode">
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="and">and</SelectItem>
                  <SelectItem value="or">or</SelectItem>
                </SelectContent>
              </SelectRoot>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">invert</label>
              <Switch v-model="ruleForm.invert" class="mt-1.5" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">子规则 (JSON) <span class="text-destructive">*</span></label>
              <div class="flex flex-col gap-1">
                <textarea v-model="ruleForm.rulesJson" rows="8" class="textarea textarea-bordered w-full font-mono text-xs" placeholder='[{"rule_set": "gfw"}, {"clash_mode": "direct"}]' />
                <p class="text-xs text-[#909399]">嵌套子规则为 Rule 数组（可再嵌套 logical）；每个子规则也可带 action</p>
              </div>
            </template>

            <template v-if="ruleForm.ruleType === 'default'">
              <div class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]" style="grid-column: 1 / -1">
                <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />匹配条件<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
              </div>
              <template v-for="g in RULE_GROUPS" :key="g">
                <div class="flex items-center gap-2 text-xs font-semibold text-[#909399]" style="grid-column: 1 / -1">
                  <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />{{ g }}<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
                </div>
                <template v-for="f in RULE_FIELDS.filter((x) => x.group === g)" :key="f.key">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">{{ f.label }}</label>
                  <ChipInput
                    v-if="f.type === 'string-list' || f.type === 'uint-list' || f.type === 'int-list'"
                    v-model="ruleForm[f.key] as string[]"
                    :placeholder="f.placeholder || '输入后回车添加，可多值'"
                    :suggestions="fieldOptions(f)"
                  />
                  <SelectRoot v-else-if="f.type === 'select'" v-model="ruleForm[f.key]">
                    <SelectTrigger><SelectValue :placeholder="f.placeholder || '请选择'" /></SelectTrigger>
                    <SelectContent>
                      <SelectItem v-for="o in fieldOptions(f)" :key="o" :value="o">{{ o }}</SelectItem>
                    </SelectContent>
                  </SelectRoot>
                  <Switch v-else-if="f.type === 'bool'" v-model="ruleForm[f.key]" class="mt-1.5" />
                  <input v-else v-model="ruleForm[f.key]" type="text" class="input input-bordered input-sm w-full" :placeholder="f.placeholder || '请输入'" />
                </template>
              </template>
            </template>

            <div class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]" style="grid-column: 1 / -1">
              <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />动作<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
            </div>
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">action</label>
            <SelectRoot v-model="ruleForm.action">
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="a in actionOptions" :key="a.value" :value="a.value">{{ a.label }}</SelectItem>
              </SelectContent>
            </SelectRoot>
            <template v-if="isRouteAction">
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">outbound <span class="text-destructive">*</span></label>
              <SelectRoot v-model="ruleForm.outbound">
                <SelectTrigger><SelectValue placeholder="选择出站" /></SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="t in outboundTags" :key="t" :value="t">{{ t }}</SelectItem>
                </SelectContent>
              </SelectRoot>
            </template>
            <template v-else-if="ruleForm.action === 'sniff'">
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">sniffer</label>
              <ChipInput v-model="ruleForm.sniffers as string[]" placeholder="嗅探协议（可多选）" :suggestions="snifferOptions" />
            </template>
            <template v-else-if="ruleForm.action === 'resolve'">
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">DNS server</label>
              <input v-model="ruleForm.resolve_server" type="text" class="input input-bordered input-sm w-full" placeholder="DNS 服务器 tag（可选，留空用默认）" />
            </template>
            <div class="rounded-md border border-[#409eff]/30 bg-[#e8f3ff] p-3 text-xs leading-relaxed text-[#1890ff] dark:bg-[rgba(64,158,255,0.16)] dark:text-[#66b1ff]" style="grid-column: 1 / -1">
              route=默认出站（选 outbound）；direct=直连；bypass=绕过；reject=拒绝；hijack-dns=DNS 劫持；sniff=协议嗅探；resolve=DNS 解析；route-options=路由选项（参数写在附加字段）
            </div>
          </div>
        </TabsContent>
        <TabsContent value="source">
          <ResourceSourceTab ref="srcTab" :initial="sourceJson" />
        </TabsContent>
      </TabsRoot>
      <div class="mt-5 flex justify-end gap-2">
        <button class="btn btn-ghost btn-sm" @click="routeDialogVisible = false">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="saving" @click="save">保存</button>
      </div>
    </DialogWrapper>
  </div>
</template>
