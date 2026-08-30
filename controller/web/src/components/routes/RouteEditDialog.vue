<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { showToast } from '@/helper/toast'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import ChipInput from '@/components/common/ChipInput.vue'
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { SelectField } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { api } from '@/api'
import ResourceSourceTab from '@/components/ResourceSourceTab.vue'
import type { RouteRule } from '@/api'
import { useStatusStore } from '@/store/status'
import { RULE_FIELDS, RULE_GROUPS } from '@/views/routeFields'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    id?: string
    isEdit?: boolean
  }>(),
  {
    id: '',
    isEdit: false,
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'saved', result: unknown): void
}>()

const statusStore = useStatusStore()

const outboundTags = ref<string[]>([])
const inboundTags = ref<string[]>([])

const srcTab = ref<InstanceType<typeof ResourceSourceTab>>()
const sourceJson = ref('{}')
const saving = ref(false)
const activeTab = ref('form')

const actionOptions = [
  { value: 'route', label: '出站（route）' },
  { value: 'direct', label: '直连（direct）' },
  { value: 'bypass', label: '绕过（bypass）' },
  { value: 'reject', label: '拒绝（reject）' },
  { value: 'hijack-dns', label: 'DNS 劫持（hijack-dns）' },
  { value: 'sniff', label: '协议嗅探（sniff）' },
  { value: 'resolve', label: 'DNS 解析（resolve）' },
  { value: 'route-options', label: '路由选项（route-options）' },
] as const

const snifferOptions = [
  'tls',
  'http',
  'quic',
  'dns',
  'stun',
  'bittorrent',
  'dtls',
  'ssh',
  'rdp',
  'ntp',
]

const ruleForm = reactive<Record<string, any>>({
  ruleType: 'default',
  mode: 'and',
  rulesJson: '',
  action: 'route',
  outbound: '',
  sniffers: [] as string[],
  resolve_server: '',
})

for (const f of RULE_FIELDS) {
  if (f.type === 'bool') ruleForm[f.key] = false
  else if (f.type === 'select' || f.type === 'string') ruleForm[f.key] = ''
  else ruleForm[f.key] = []
}

const isRouteAction = computed(() => ruleForm.action === 'route')

function fieldOptions(f: (typeof RULE_FIELDS)[number]): string[] {
  if (f.key === 'inbound') return inboundTags.value
  return f.options ?? []
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

function fillAction(rowRec: Record<string, unknown>) {
  const action = typeof rowRec.action === 'string' && rowRec.action ? rowRec.action : 'route'
  ruleForm.action = action
  ruleForm.outbound = typeof rowRec.outbound === 'string' ? rowRec.outbound : ''
  const sniffers = rowRec.sniffer
  ruleForm.sniffers = Array.isArray(sniffers)
    ? sniffers.map(String)
    : typeof sniffers === 'string'
      ? [sniffers]
      : []
  ruleForm.resolve_server = typeof rowRec.server === 'string' ? rowRec.server : ''
}

const loadRuleData = async (ruleId: string) => {
  if (!ruleId) {
    resetForm()
    sourceJson.value = '{}'
    return
  }
  try {
    let rowRec: Record<string, unknown> = {}
    try {
      const res = await api.getRoute(ruleId)
      rowRec = (res.rule || {}) as Record<string, unknown>
    } catch {
      const all = await api.routes()
      const matched = all.routes.find((item) => item.id === ruleId)
      if (matched) {
        rowRec = (matched.rule || {}) as Record<string, unknown>
      }
    }
    const { id: _omit, ...ruleBody } = rowRec
    sourceJson.value = JSON.stringify(ruleBody, null, 2)
    if (rowRec.type === 'logical') {
      ruleForm.ruleType = 'logical'
      ruleForm.mode = rowRec.mode === 'or' ? 'or' : 'and'
      ruleForm.rulesJson = Array.isArray(rowRec.rules)
        ? JSON.stringify(rowRec.rules, null, 2)
        : ''
      ruleForm.invert = rowRec.invert === true
      fillAction(rowRec)
      return
    }
    ruleForm.ruleType = 'default'
    ruleForm.invert = rowRec.invert === true
    for (const f of RULE_FIELDS) {
      const v = rowRec[f.key]
      if (f.type === 'bool') ruleForm[f.key] = v === true
      else if (f.type === 'select' || f.type === 'string')
        ruleForm[f.key] = v == null ? '' : String(v)
      else ruleForm[f.key] = v == null ? [] : Array.isArray(v) ? v.map(String) : [String(v)]
    }
    fillAction(rowRec)
  } catch (e) {
    showToast((e as Error).message || '获取规则详情失败', 'error')
  }
}

watch(
  () => [props.modelValue, props.id, props.isEdit],
  ([visible, id, isEdit]) => {
    if (visible) {
      activeTab.value = 'form'
      void loadTags()
      if (isEdit && id) {
        void loadRuleData(String(id))
      } else {
        resetForm()
        sourceJson.value = '{}'
      }
    }
  },
  { immediate: true }
)

function buildRule(): RouteRule {
  const rule: RouteRule = {}
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

const handleResult = (res: unknown) => {
  const r = (res ?? {}) as { reload_error?: string; message?: string }
  if (r.reload_error) {
    showToast(r.message || `配置已保存，但实例重载失败：${r.reload_error}`, 'warning')
  } else {
    showToast('保存成功', 'success')
  }
}

const save = async () => {
  if (isRouteAction.value && !String(ruleForm.outbound)) {
    showToast('出站（outbound）必选', 'warning')
    return
  }
  saving.value = true
  try {
    const rule = srcTab.value?.isDirty() ? JSON.parse(srcTab.value.getText()) : buildRule()
    const res = props.isEdit && props.id
      ? await api.updateRoute(props.id, rule)
      : await api.createRoute(rule)
    handleResult(res)
    emit('update:modelValue', false)
    emit('saved', res)
    await statusStore.refresh()
  } catch (e) {
    showToast((e as Error).message || '操作失败', 'error')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  void loadTags()
})
</script>

<template>
  <DialogWrapper
    :model-value="modelValue"
    :title="isEdit ? '编辑规则' : '新建规则'"
    box-class="max-w-[760px]"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <TabsRoot v-model="activeTab">
      <TabsList>
        <TabsTrigger value="form">表单</TabsTrigger>
        <TabsTrigger value="source">源码</TabsTrigger>
      </TabsList>

      <TabsContent value="form">
        <div class="grid gap-x-4 gap-y-5 pt-2" style="grid-template-columns: 130px minmax(0, 1fr)">
          <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">类型</label>
          <SelectField
            v-model="ruleForm.ruleType"
            :options="[
              { value: 'default', label: '普通规则（匹配字段）' },
              { value: 'logical', label: '逻辑组合（and/or 嵌套子规则）' },
            ]"
          />

          <!-- 逻辑组合区域 -->
          <div class="col-span-2 collapse-row" :class="{ 'is-open': ruleForm.ruleType === 'logical' }">
            <div class="collapse-inner">
              <div class="grid gap-x-4 gap-y-4 pt-2" style="grid-template-columns: 130px minmax(0, 1fr)">
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">mode <span class="text-destructive">*</span></label>
                <SelectField v-model="ruleForm.mode" :options="['and', 'or']" />
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">invert</label>
                <Switch v-model="ruleForm.invert" class="self-center" />
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">子规则 (JSON) <span class="text-destructive">*</span></label>
                <div class="flex flex-col gap-1">
                  <textarea
                    v-model="ruleForm.rulesJson"
                    rows="8"
                    class="textarea textarea-bordered w-full font-mono text-xs"
                    placeholder='[{"rule_set": "gfw"}, {"clash_mode": "direct"}]'
                  />
                  <p class="text-xs text-[#909399]">嵌套子规则为 Rule 数组（可再嵌套 logical）；每个子规则也可带 action</p>
                </div>
              </div>
            </div>
          </div>

          <!-- 普通规则匹配字段区域 -->
          <div class="col-span-2 collapse-row" :class="{ 'is-open': ruleForm.ruleType === 'default' }">
            <div class="collapse-inner">
              <div class="grid gap-x-4 gap-y-4 pt-2" style="grid-template-columns: 130px minmax(0, 1fr)">
                <div
                  class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]"
                  style="grid-column: 1 / -1"
                >
                  <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />匹配条件<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
                </div>
                <template v-for="g in RULE_GROUPS" :key="g">
                  <div
                    class="flex items-center gap-2 text-xs font-semibold text-[#909399]"
                    style="grid-column: 1 / -1"
                  >
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
                    <SelectField
                      v-else-if="f.type === 'select'"
                      v-model="ruleForm[f.key]"
                      :options="fieldOptions(f)"
                      :placeholder="f.placeholder || '请选择'"
                    />
                    <Switch
                      v-else-if="f.type === 'bool'"
                      v-model="ruleForm[f.key]"
                      class="self-center"
                    />
                    <input
                      v-else
                      v-model="ruleForm[f.key]"
                      type="text"
                      class="input input-bordered input-sm w-full"
                      :placeholder="f.placeholder || '请输入'"
                    />
                  </template>
                </template>
              </div>
            </div>
          </div>

          <!-- 动作配置区域 -->
          <div
            class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]"
            style="grid-column: 1 / -1"
          >
            <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />动作<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
          </div>
          <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">action</label>
          <SelectField v-model="ruleForm.action" :options="actionOptions" />

          <template v-if="isRouteAction">
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">outbound <span class="text-destructive">*</span></label>
            <SelectField v-model="ruleForm.outbound" :options="outboundTags" placeholder="选择出站" />
          </template>
          <template v-else-if="ruleForm.action === 'sniff'">
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">sniffer</label>
            <ChipInput
              v-model="ruleForm.sniffers as string[]"
              placeholder="嗅探协议（可多选）"
              :suggestions="snifferOptions"
            />
          </template>
          <template v-else-if="ruleForm.action === 'resolve'">
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">DNS server</label>
            <input
              v-model="ruleForm.resolve_server"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="DNS 服务器 tag（可选，留空用默认）"
            />
          </template>

          <div
            class="rounded-md border border-[#409eff]/30 bg-[#e8f3ff] p-3 text-xs leading-relaxed text-[#1890ff] dark:bg-[rgba(64,158,255,0.16)] dark:text-[#66b1ff]"
            style="grid-column: 1 / -1"
          >
            route=默认出站（选 outbound）；direct=直连；bypass=绕过；reject=拒绝；hijack-dns=DNS 劫持；sniff=协议嗅探；resolve=DNS 解析；route-options=路由选项（参数写在附加字段）
          </div>
        </div>
      </TabsContent>

      <TabsContent value="source">
        <ResourceSourceTab ref="srcTab" :initial="sourceJson" />
      </TabsContent>
    </TabsRoot>

    <div class="mt-5 flex justify-end gap-2">
      <button class="btn btn-ghost btn-sm" @click="emit('update:modelValue', false)">取消</button>
      <button class="btn btn-primary btn-sm" :disabled="saving" @click="save">保存</button>
    </div>
  </DialogWrapper>
</template>

<style scoped>
.collapse-row {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}
.collapse-row.is-open {
  grid-template-rows: 1fr;
}
.collapse-inner {
  overflow: hidden;
  min-height: 0;
}
</style>
