<template>
  <DialogWrapper
    v-model="isDnsRuleModalOpen"
    :title="isRuleEditMode ? '编辑 DNS 规则' : '新建 DNS 规则'"
    box-class="max-w-[760px]"
    @update:model-value="(val) => (!val ? closeDnsRuleModal() : null)"
  >
    <TabsRoot v-model="activeTab">
      <TabsList>
        <TabsTrigger value="form">表单</TabsTrigger>
        <TabsTrigger value="source">源码</TabsTrigger>
      </TabsList>

      <TabsContent value="form">
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
                <input
                  v-else-if="f.type === 'string'"
                  v-model="ruleForm[f.key]"
                  type="text"
                  class="input input-bordered input-sm w-full"
                  :placeholder="f.placeholder || '请输入'"
                />
                <textarea
                  v-else
                  v-model="ruleForm[f.key]"
                  rows="3"
                  class="textarea textarea-bordered font-mono text-xs w-full"
                  :placeholder="f.placeholder || '字段 JSON 对象'"
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
            动作
            <span class="border-base-content/10 h-px flex-1 border-t" />
          </div>

          <label class="text-base-content/70 pt-2">action</label>
          <SelectField
            v-model="ruleForm.action"
            :options="DNS_RULE_ACTIONS"
          />

          <template v-if="ruleForm.action === 'route'">
            <label class="text-base-content/70 pt-2">server</label>
            <SelectField
              v-model="ruleForm.server"
              :options="dnsTags"
              placeholder="选择 DNS server"
            />

            <label class="text-base-content/70 pt-2">speculative</label>
            <div class="flex items-center gap-2">
              <Switch v-model="ruleForm.speculative" />
              <span class="text-base-content/50 text-xs">先返回假结果再修正</span>
            </div>
          </template>

          <template v-else-if="ruleForm.action === 'evaluate'">
            <label class="text-base-content/70 pt-2">server</label>
            <SelectField
              v-model="ruleForm.server"
              :options="dnsTags"
              placeholder="选择 DNS server"
            />

            <label class="text-base-content/70 pt-2">tag</label>
            <input
              v-model="ruleForm.evaluate_tag"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="评估结果写入的 tag（可选）"
            />

            <label class="text-base-content/70 pt-2">speculative</label>
            <Switch
              v-model="ruleForm.evaluate_speculative"
              class="self-center"
            />
          </template>

          <template v-else-if="ruleForm.action === 'reject'">
            <label class="text-base-content/70 pt-2">method</label>
            <input
              v-model="ruleForm.reject_method"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="拒绝方法（默认 drop，如 reject）"
            />
          </template>

          <p
            v-else-if="ruleForm.action === 'respond'"
            class="text-base-content/50 text-xs"
            style="grid-column: 2"
          >
            respond 无参数；应答内容用匹配条件的 response_* 字段
          </p>

          <template v-else>
            <label class="text-base-content/70 pt-2">动作参数 (JSON)</label>
            <textarea
              v-model="ruleForm.actionParamsJson"
              rows="5"
              class="textarea textarea-bordered font-mono text-xs w-full"
              :placeholder="actionPlaceholder"
            />
          </template>
        </div>
      </TabsContent>

      <TabsContent value="source">
        <ResourceSourceTab
          ref="srcTab"
          :initial="sourceJson"
        />
      </TabsContent>
    </TabsRoot>

    <div class="mt-5 flex justify-end gap-2">
      <button
        type="button"
        class="btn btn-ghost btn-sm"
        @click="closeDnsRuleModal"
      >
        取消
      </button>
      <button
        type="button"
        class="btn btn-primary btn-sm"
        :disabled="saving"
        @click="save"
      >
        保存
      </button>
    </div>
  </DialogWrapper>
</template>

<script setup lang="ts">
import {
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
import { computed, reactive, ref, watch } from 'vue'

const activeTab = ref('form')
const srcTab = ref<InstanceType<typeof ResourceSourceTab>>()
const sourceJson = ref('{}')

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

watch(
  () => [isDnsRuleModalOpen.value, editingRule.value],
  ([visible]) => {
    if (!visible) return
    activeTab.value = 'form'
    resetRuleForm()
    const item = editingRule.value
    if (item && item.rule) {
      const r = item.rule
      sourceJson.value = JSON.stringify(r, null, 2)
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
    } else {
      sourceJson.value = '{}'
    }
  },
  { immediate: true },
)

const save = async () => {
  try {
    const payload = srcTab.value?.isDirty()
      ? JSON.parse(srcTab.value.getText())
      : buildDnsRule()
    const id = isRuleEditMode.value ? String(editingRule.value?.id || '') : ''
    const ok = await saveDnsRule(id, payload, isRuleEditMode.value)
    if (ok) closeDnsRuleModal()
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
  }
}
</script>
