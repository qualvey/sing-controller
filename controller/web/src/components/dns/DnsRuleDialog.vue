<template>
  <DialogWrapper
    v-model="isDnsRuleModalOpen"
    :title="isRuleEditMode ? '编辑 DNS 规则' : '新建 DNS 规则'"
    box-class="max-w-[760px]"
    :body-class="activeTab === 'source' ? 'overflow-hidden' : 'overflow-y-auto'"
    @update:model-value="(val: boolean) => (!val ? closeDnsRuleModal() : null)"
  >
    <TabsRoot v-model="activeTab">
      <TabsList class="self-start shrink-0">
        <TabsTrigger value="form">表单</TabsTrigger>
        <TabsTrigger value="source">源码</TabsTrigger>
      </TabsList>

      <TabsContent value="form">
        <div ref="formContentRef">
          <div class="mb-4">
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              @click="fillFromJson"
            >
              <ClipboardPasteIcon class="h-3.5 w-3.5" />
              从 JSON 填充（粘贴解析）
            </button>
            <span class="ml-2 text-xs text-base-content/50">
              粘贴完整 DNS Rule JSON，自动解析并填充下方字段
            </span>
          </div>

          <div
            class="grid gap-x-4 gap-y-5 pt-2 text-sm"
            style="grid-template-columns: 140px minmax(0, 1fr)"
          >
            <label class="text-base-content/70 pt-2">类型</label>
            <SelectField
              v-model="ruleForm.ruleType"
              :options="DNS_RULE_TYPES"
            />

            <!-- 逻辑组合 -->
            <template v-if="ruleForm.ruleType === 'logical'">
              <label class="text-base-content/70 pt-2">
                mode <span class="text-destructive">*</span>
              </label>
              <SelectField
                v-model="ruleForm.mode"
                :options="LOGICAL_MODES"
              />

              <label class="text-base-content/70 pt-2">invert</label>
              <Switch
                v-model="ruleForm.invert"
                class="self-center"
              />

              <label class="text-base-content/70 pt-2">
                子规则 (JSON) <span class="text-destructive">*</span>
              </label>
              <div class="flex flex-col gap-1">
                <textarea
                  v-model="ruleForm.rulesJson"
                  rows="8"
                  class="textarea textarea-bordered font-mono text-xs w-full"
                  placeholder='[{"rule_set": "gfw", "invert": true}, {"clash_mode": "direct"}]'
                />
                <p class="text-base-content/50 text-xs">
                  嵌套子规则为 DNSRule 数组（可再嵌套 logical）；每个子规则也可带 server 等动作
                </p>
              </div>
            </template>

            <!-- 普通规则匹配字段 -->
            <template v-if="ruleForm.ruleType === 'default'">
              <div
                class="flex items-center gap-2 text-[13px] font-semibold text-base-content/90"
                style="grid-column: 1 / -1"
              >
                <span class="border-base-content/10 h-px flex-1 border-t" />
                匹配条件
                <span class="border-base-content/10 h-px flex-1 border-t" />
              </div>

              <template
                v-for="g in DNS_RULE_GROUPS"
                :key="g"
              >
                <div
                  class="flex items-center gap-2 text-xs font-semibold text-base-content/50"
                  style="grid-column: 1 / -1"
                >
                  <span class="border-base-content/10 h-px flex-1 border-t" />
                  {{ g }}
                  <span class="border-base-content/10 h-px flex-1 border-t" />
                </div>
                <template
                  v-for="f in DNS_RULE_FIELDS.filter((x) => x.group === g)"
                  :key="f.key"
                >
                  <label class="text-base-content/70 pt-2">{{ f.label }}</label>
                  <ChipInput
                    v-if="f.type === 'string-list' || f.type === 'uint-list' || f.type === 'int-list'"
                    v-model="ruleForm[f.key]"
                    :placeholder="f.placeholder || '输入后回车添加，可多值'"
                    :suggestions="f.options ?? (f.key === 'inbound' ? inboundTags : [])"
                  />
                  <SelectField
                    v-else-if="f.type === 'select'"
                    v-model="ruleForm[f.key]"
                    :options="f.options ?? []"
                    :placeholder="f.placeholder || '请选择'"
                  />
                  <Switch
                    v-else-if="f.type === 'bool'"
                    v-model="ruleForm[f.key]"
                    class="self-center"
                  />
                  <textarea
                    v-else-if="f.type === 'json'"
                    v-model="ruleForm[f.key]"
                    rows="3"
                    class="textarea textarea-bordered font-mono text-xs w-full"
                    :placeholder="f.placeholder || 'JSON 对象，如 {\x22key\x22: \x22value\x22}'"
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
            </template>

            <!-- 动作配置 -->
            <div
              class="flex items-center gap-2 text-[13px] font-semibold text-base-content/90"
              style="grid-column: 1 / -1"
            >
              <span class="border-base-content/10 h-px flex-1 border-t" />
              动作与服务器
              <span class="border-base-content/10 h-px flex-1 border-t" />
            </div>

            <label class="text-base-content/70 pt-2">action</label>
            <SelectField
              v-model="ruleForm.action"
              :options="DNS_RULE_ACTIONS"
            />

            <!-- action=route（默认走指定 server） -->
            <template v-if="ruleForm.action === 'route'">
              <label class="text-base-content/70 pt-2">
                server <span class="text-destructive">*</span>
              </label>
              <SelectField
                v-model="ruleForm.server"
                :options="dnsTags"
                placeholder="选择 DNS server"
              />

              <label class="text-base-content/70 pt-2">speculative</label>
              <Switch
                v-model="ruleForm.speculative"
                class="self-center"
              />
            </template>

            <!-- action=reject -->
            <template v-else-if="ruleForm.action === 'reject'">
              <label class="text-base-content/70 pt-2">method</label>
              <SelectField
                v-model="ruleForm.reject_method"
                :options="['default', 'drop']"
                placeholder="default / drop"
              />

              <label class="text-base-content/70 pt-2">server</label>
              <SelectField
                v-model="ruleForm.server"
                :options="dnsTags"
                placeholder="留空或选 DNS server"
              />
            </template>

            <!-- action=respond -->
            <template v-else-if="ruleForm.action === 'respond'">
              <label class="text-base-content/70 pt-2">server</label>
              <SelectField
                v-model="ruleForm.server"
                :options="dnsTags"
                placeholder="留空或选 DNS server"
              />
            </template>

            <!-- action=evaluate -->
            <template v-else-if="ruleForm.action === 'evaluate'">
              <label class="text-base-content/70 pt-2">tag</label>
              <input
                v-model="ruleForm.evaluate_tag"
                type="text"
                class="input input-bordered input-sm w-full"
                placeholder="评估规则标签"
              />

              <label class="text-base-content/70 pt-2">speculative</label>
              <Switch
                v-model="ruleForm.evaluate_speculative"
                class="self-center"
              />

              <label class="text-base-content/70 pt-2">server</label>
              <SelectField
                v-model="ruleForm.server"
                :options="dnsTags"
                placeholder="留空或选 DNS server"
              />
            </template>

            <!-- action=route-options 或 predefined -->
            <template v-else-if="ruleForm.action === 'route-options' || ruleForm.action === 'predefined'">
              <label class="text-base-content/70 pt-2">动作参数 (JSON)</label>
              <div class="flex flex-col gap-1">
                <textarea
                  v-model="ruleForm.actionParamsJson"
                  rows="4"
                  class="textarea textarea-bordered font-mono text-xs w-full"
                  :placeholder="actionPlaceholder"
                />
                <p class="text-base-content/50 text-xs">
                  如 route-options 附加参数或 predefined 静态回答配置
                </p>
              </div>
            </template>
          </div>

          <p
            v-if="validateMsg"
            class="mt-3 text-[0.8rem] font-medium text-destructive"
          >
            {{ validateMsg }}
          </p>
        </div>
      </TabsContent>

      <TabsContent value="source">
        <ResourceSourceTab
          ref="srcTab"
          :initial="sourceJson"
          :height="sourceEditorHeight"
        />
      </TabsContent>
    </TabsRoot>

    <template #footer>
      <div class="flex items-center justify-between w-full">
        <div>
          <button
            v-if="isRuleEditMode"
            type="button"
            class="btn btn-ghost btn-sm text-error hover:bg-error/10"
            :disabled="saving"
            @click="handleDelete"
          >
            {{ $t('delete') || '删除' }}
          </button>
        </div>
        <div class="flex justify-end gap-2">
          <button
            type="button"
            class="btn btn-ghost btn-sm"
            @click="closeDnsRuleModal"
          >
            {{ $t('cancel') || '取消' }}
          </button>
          <button
            type="button"
            class="btn btn-primary btn-sm"
            :disabled="saving"
            @click="save"
          >
            {{ $t('save') || '保存' }}
          </button>
        </div>
      </div>
    </template>
  </DialogWrapper>

  <!-- 从 JSON 填充弹窗 -->
  <DialogWrapper
    v-model="jsonDialog"
    title="从 JSON 填充"
    box-class="max-w-[560px]"
  >
    <textarea
      v-model="jsonText"
      rows="10"
      class="textarea textarea-bordered w-full font-mono text-xs"
      placeholder='{"rule_set":["geosite-cn"],"server":"local-dns"}'
    />
    <p class="mt-1 text-xs text-base-content/50">
      解析成功后填充表单字段（请检查解析结果）
    </p>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <button
          type="button"
          class="btn btn-ghost btn-sm"
          @click="jsonDialog = false"
        >
          取消
        </button>
        <button
          type="button"
          class="btn btn-primary btn-sm"
          @click="doFillFromJson"
        >
          解析并填充
        </button>
      </div>
    </template>
  </DialogWrapper>
</template>

<script setup lang="ts">
import { api } from '@/api'
import {
  deleteDnsRule,
  dnsTags,
  inboundTags,
  saveDnsRule,
  saving,
} from '@/assembly/dns'
import ChipInput from '@/components/common/ChipInput.vue'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import ResourceSourceTab from '@/components/ResourceSourceTab.vue'
import { SelectField } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { TabsContent, TabsList, TabsRoot, TabsTrigger } from '@/components/ui/tabs'
import {
  closeDnsRuleModal,
  editingRule,
  isDnsRuleModalOpen,
  isRuleEditMode,
} from '@/composables/dnsManagement'
import { showToast } from '@/helper/toast'
import {
  DNS_RULE_ACTIONS,
  DNS_RULE_FIELDS,
  DNS_RULE_GROUPS,
  DNS_RULE_TYPES,
  LOGICAL_MODES,
} from '@/views/dnsRuleFields'
import { ClipboardPasteIcon } from '@lucide/vue'
import { useResizeObserver } from '@vueuse/core'
import { computed, reactive, ref, watch } from 'vue'

const activeTab = ref('form')
const srcTab = ref<InstanceType<typeof ResourceSourceTab>>()
const sourceJson = ref('{}')
const validateMsg = ref('')
const formContentRef = ref<HTMLElement>()
const sharedTabHeight = ref<number>(460)

const jsonDialog = ref(false)
const jsonText = ref('')

useResizeObserver(formContentRef, (entries) => {
  const entry = entries[0]
  if (entry && activeTab.value === 'form' && entry.contentRect.height > 60) {
    sharedTabHeight.value = Math.max(300, Math.round(entry.contentRect.height))
  }
})

const sourceEditorHeight = computed(() => {
  if (typeof window === 'undefined') return 440
  const maxAvailable = Math.max(300, Math.floor(window.innerHeight * 0.85 - 170))
  return Math.min(sharedTabHeight.value, maxAvailable)
})

const actionPlaceholder = computed(() =>
  ruleForm.action === 'route-options'
    ? 'route-options 参数对象'
    : '{"rcode": "NXDOMAIN", "answer": []}',
)

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
  actionParamsJson: '',
})

for (const f of DNS_RULE_FIELDS) {
  if (f.type === 'bool') ruleForm[f.key] = false
  else if (f.type === 'string' || f.type === 'select' || f.type === 'json') ruleForm[f.key] = ''
  else ruleForm[f.key] = []
}

const resetRuleForm = () => {
  validateMsg.value = ''
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
    if (ruleForm.server) rule.server = String(ruleForm.server)
    if (ruleForm.speculative) rule.speculative = true
  }
}

function fillDnsAction(r: Record<string, any>) {
  const action = typeof r.action === 'string' && r.action ? r.action : 'route'
  ruleForm.action = action
  ruleForm.server = typeof r.server === 'string' ? r.server : ''
  ruleForm.speculative = r.speculative === true
  ruleForm.evaluate_tag = typeof r.tag === 'string' ? r.tag : ''
  ruleForm.evaluate_speculative = r.speculative === true
  ruleForm.reject_method = typeof r.method === 'string' ? r.method : ''
  if (action === 'route-options' || action === 'predefined') {
    const known = new Set([
      'action',
      'server',
      'speculative',
      'type',
      'mode',
      'rules',
      ...DNS_RULE_FIELDS.map((f) => f.key),
    ])
    const extra: Record<string, any> = {}
    for (const k of Object.keys(r)) {
      if (!known.has(k)) extra[k] = r[k]
    }
    ruleForm.actionParamsJson = Object.keys(extra).length ? JSON.stringify(extra, null, 2) : ''
  }
}

function fillForm(r: Record<string, any>) {
  resetRuleForm()
  if (r.type === 'logical') {
    ruleForm.ruleType = 'logical'
    ruleForm.mode = r.mode === 'or' ? 'or' : 'and'
    ruleForm.rulesJson = Array.isArray(r.rules) ? JSON.stringify(r.rules, null, 2) : ''
    ruleForm.invert = r.invert === true
    fillDnsAction(r)
    return
  }
  for (const f of DNS_RULE_FIELDS) {
    const v = r[f.key]
    if (f.type === 'bool') ruleForm[f.key] = v === true
    else if (f.type === 'json') ruleForm[f.key] = v == null ? '' : JSON.stringify(v, null, 2)
    else if (f.type === 'string' || f.type === 'select')
      ruleForm[f.key] = v == null ? '' : String(v)
    else ruleForm[f.key] = v == null ? [] : Array.isArray(v) ? v.map(String) : [String(v)]
  }
  fillDnsAction(r)
}

const fillFromJson = () => {
  jsonText.value = ''
  jsonDialog.value = true
}

const doFillFromJson = async () => {
  try {
    let parsed: Record<string, any>
    try {
      const res = await api.parseJson(jsonText.value)
      parsed = (res?.data || res) as Record<string, any>
    } catch {
      parsed = JSON.parse(jsonText.value)
    }
    if (!parsed || typeof parsed !== 'object') {
      throw new Error('解析内容必须为 JSON 对象')
    }
    fillForm(parsed)
    jsonDialog.value = false
    showToast('已从 JSON 填充，请检查表单', 'success')
  } catch (e) {
    showToast((e as Error).message || '解析失败', 'error')
  }
}

watch(
  () => [isDnsRuleModalOpen.value, editingRule.value],
  ([visible]) => {
    if (!visible) return
    activeTab.value = 'form'
    validateMsg.value = ''
    resetRuleForm()
    const item = editingRule.value
    if (item && item.rule) {
      const r = item.rule
      sourceJson.value = JSON.stringify(r, null, 2)
      fillForm(r)
    } else {
      sourceJson.value = '{}'
    }
  },
  { immediate: true },
)

watch(activeTab, (tab) => {
  if (tab === 'source' && !srcTab.value?.isDirty()) {
    try {
      sourceJson.value = JSON.stringify(buildDnsRule(), null, 2)
    } catch {
      // ignore
    }
  }
})

const save = async () => {
  validateMsg.value = ''
  try {
    const payload = srcTab.value?.isDirty()
      ? JSON.parse(srcTab.value.getText())
      : buildDnsRule()
    const id = isRuleEditMode.value ? String(editingRule.value?.id || '') : ''
    const ok = await saveDnsRule(id, payload, isRuleEditMode.value)
    if (ok) closeDnsRuleModal()
  } catch (e) {
    validateMsg.value = (e as Error).message || '保存失败'
    showToast((e as Error).message || '保存失败', 'error')
  }
}

const handleDelete = async () => {
  if (!isRuleEditMode.value || !editingRule.value?.id) return
  const ok = await deleteDnsRule(editingRule.value.id)
  if (ok) closeDnsRuleModal()
}
</script>
