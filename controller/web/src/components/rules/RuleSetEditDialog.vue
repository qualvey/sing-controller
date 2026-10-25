<template>
  <DialogWrapper :model-value="modelValue" :title="isEdit ? '编辑规则集' : '新建规则集'" box-class="max-w-[680px]"
    :body-class="activeTab === 'source' ? 'overflow-hidden' : 'overflow-y-auto'"
    @update:model-value="emit('update:modelValue', $event)">
    <TabsRoot v-model="activeTab">
      <TabsList class="self-start shrink-0">
        <TabsTrigger value="form">表单</TabsTrigger>
        <TabsTrigger value="source">源码</TabsTrigger>
      </TabsList>

      <TabsContent value="form">
        <div ref="formContentRef">
          <div class="mb-4">
            <button type="button" class="btn btn-ghost btn-xs" @click="fillFromJson">
              <ClipboardPasteIcon class="h-3.5 w-3.5" />
              从 JSON 填充（粘贴解析）
            </button>
            <span class="ml-2 text-xs text-base-content/50">
              粘贴完整 RuleSet JSON，自动解析并填充下方字段
            </span>
          </div>

          <div class="grid gap-x-4 gap-y-4 pt-2 text-sm" style="grid-template-columns: 140px minmax(0, 1fr)">
            <label class="text-base-content/70 pt-2">type <span class="text-destructive">*</span></label>
            <SelectField v-model="form.type" :options="RULE_SET_TYPES" placeholder="选择类型" />

            <label class="text-base-content/70 pt-2">tag <span class="text-destructive">*</span></label>
            <div class="flex flex-col gap-1.5">
              <div class="flex flex-wrap gap-1.5">
                <span v-for="t in form.tag" :key="t" class="badge badge-primary badge-outline gap-1">
                  {{ t }}
                  <button type="button" class="cursor-pointer text-xs opacity-70 hover:opacity-100"
                    @click="removeTag(t)">
                    ×
                  </button>
                </span>
              </div>
              <input v-model="tagInput" type="text" class="input input-bordered input-sm w-full"
                :placeholder="form.type === 'inline' ? '单个 tag（回车添加）' : 'tag（回车/逗号添加，多个）'"
                @keydown.enter.prevent="addTag" @blur="addTag" />
            </div>

            <template v-if="form.type === 'inline'">
              <label class="text-base-content/70 pt-2">rules (JSON)</label>
              <div class="flex flex-col gap-1">
                <textarea v-model="form.rulesJson" rows="8" class="textarea textarea-bordered font-mono text-xs w-full"
                  placeholder='[{"domain_suffix": ".cn"}, {"ip_cidr": ["10.0.0.0/8"]}]' />
                <p class="text-base-content/50 text-xs">
                  HeadlessRule 数组（无 action 的匹配规则）；多个 tag 时 path/url 需含 {tag} 占位符
                </p>
              </div>
            </template>

            <template v-else-if="form.type === 'local'">
              <label class="text-base-content/70 pt-2">format</label>
              <SelectField v-model="form.format" :options="FORMAT_OPTIONS" />
              <label class="text-base-content/70 pt-2">path <span class="text-destructive">*</span></label>
              <input v-model="form.path" type="text" class="input input-bordered input-sm w-full font-mono"
                placeholder="规则集文件路径（.json→source，.srs→binary，多 tag 时含 {tag}）" />
            </template>

            <template v-else>
              <label class="text-base-content/70 pt-2">format</label>
              <SelectField v-model="form.format" :options="FORMAT_OPTIONS" />
              <label class="text-base-content/70 pt-2">url <span class="text-destructive">*</span></label>
              <input v-model="form.url" type="text" class="input input-bordered input-sm w-full font-mono"
                placeholder="https://example.com/rule-set.json（多 tag 时含 {tag}）" />
              <label class="text-base-content/70 pt-2">http_client</label>
              <div class="flex flex-col gap-1">
                <input v-model="form.http_client" list="rule-set-http-clients" type="text"
                  class="input input-bordered input-sm w-full font-mono" placeholder="可选，例如 Proxy" />
                <datalist id="rule-set-http-clients">
                  <option v-for="tag in httpClientTags" :key="tag" :value="tag" />
                </datalist>
                <p class="text-base-content/50 text-xs">引用 sing-box 顶层 http_clients[].tag</p>
              </div>
              <label class="text-base-content/70 pt-2">initial_path</label>
              <input v-model="form.initial_path" type="text" class="input input-bordered input-sm w-full font-mono"
                placeholder="可选预置缓存文件路径" />
              <label class="text-base-content/70 pt-2">update_interval</label>
              <input v-model="form.update_interval" type="text" class="input input-bordered input-sm w-full"
                placeholder="如 24h、1d" />
            </template>
          </div>

          <p v-if="validateMsg" class="mt-3 text-[0.8rem] font-medium text-destructive">
            {{ validateMsg }}
          </p>
          <p v-else-if="repositoryNeedsNewCore" class="mt-3 text-xs text-warning">
            当前 sing-box 版本（{{ version || '未知' }}）不支持多 tag 远程规则集，请升级到 1.4.0 或更高版本。
          </p>
        </div>
      </TabsContent>

      <TabsContent value="source">
        <ResourceSourceTab ref="srcTab" :initial="sourceJson" :height="sourceEditorHeight" />
      </TabsContent>
    </TabsRoot>

    <template #footer>
      <div class="flex items-center justify-between w-full">
        <div>
          <button v-if="isEdit && ruleSetData?.id" type="button"
            class="btn btn-ghost btn-sm text-error hover:bg-error/10" :disabled="saving" @click="handleDelete">
            {{ $t('delete') || '删除' }}
          </button>
        </div>
        <div class="flex gap-2">
          <button type="button" class="btn btn-ghost btn-sm" @click="emit('update:modelValue', false)">
            {{ $t('cancel') || '取消' }}
          </button>
          <button type="button" class="btn btn-primary btn-sm" :disabled="saving" @click="save">
            {{ $t('save') || '保存' }}
          </button>
        </div>
      </div>
    </template>
  </DialogWrapper>

  <!-- 从 JSON 填充弹窗 -->
  <DialogWrapper v-model="jsonDialog" title="从 JSON 填充" box-class="max-w-[560px]">
    <textarea v-model="jsonText" rows="10" class="textarea textarea-bordered w-full font-mono text-xs"
      placeholder='{"type":"remote","tag":"geosite-cn","format":"binary","url":"https://example.com/geosite-cn.srs"}' />
    <p class="mt-1 text-xs text-base-content/50">
      解析成功后填充表单字段（请检查解析结果）
    </p>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <button type="button" class="btn btn-ghost btn-sm" @click="jsonDialog = false">
          取消
        </button>
        <button type="button" class="btn btn-primary btn-sm" @click="doFillFromJson">
          解析并填充
        </button>
      </div>
    </template>
  </DialogWrapper>
</template>

<script setup lang="ts">
import { api } from '@/api'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import ResourceSourceTab from '@/components/ResourceSourceTab.vue'
import { SelectField } from '@/components/ui/select'
import { TabsContent, TabsList, TabsRoot, TabsTrigger } from '@/components/ui/tabs'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { showToast } from '@/helper/toast'
import { useStatusStore } from '@/store/status'
import { hasRuleSetTemplate, supportsMultiTagRuleSet } from '@/assembly/rules/ruleSetRepository'
import { version } from '@/assembly/version'
import { ClipboardPasteIcon } from '@lucide/vue'
import { useResizeObserver } from '@vueuse/core'
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  modelValue: boolean
  isEdit: boolean
  ruleSetData?: { id: string; rule_set: Record<string, any> } | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'saved'): void
}>()

const statusStore = useStatusStore()

const activeTab = ref('form')
const srcTab = ref<InstanceType<typeof ResourceSourceTab>>()
const sourceJson = ref('{}')
const saving = ref(false)
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

const RULE_SET_TYPES = ['inline', 'local', 'remote']
const FORMAT_OPTIONS = ['binary', 'source']
// 1. 单一数据源：统一由函数返回默认结构
const getDefaultForm = () => ({
  type: 'remote',
  tag: [] as string[],
  format: 'binary',
  path: '',
  url: '',
  initial_path: '',
  update_interval: '',
  http_client: '',
  rulesJson: '',
})
const form = reactive(getDefaultForm())

const tagInput = ref('')
const httpClientTags = ref<string[]>([])

const repositoryNeedsNewCore = computed(() => {
  if (form.type !== 'remote') return false
  return (form.tag.length > 1 || hasRuleSetTemplate(form.url)) && !supportsMultiTagRuleSet(version.value)
})

const addTag = () => {
  const parts = tagInput.value
    .split(/[,，\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  for (const p of parts) {
    if (!form.tag.includes(p)) form.tag.push(p)
  }
  tagInput.value = ''
}

const removeTag = (t: string) => {
  form.tag = form.tag.filter((x) => x !== t)
}

// 3. 重置时直接 Object.assign，无需重复赋值
const resetForm = () => {
  validateMsg.value = ''
  Object.assign(form, getDefaultForm())
  tagInput.value = ''
}

function inferFormat(rs: Record<string, any>): string {
  if (typeof rs.format === 'string' && rs.format) return rs.format
  const src = String(rs.url || rs.path || '')
  return src.endsWith('.srs') ? 'binary' : 'source'
}

function fillRuleSetForm(rs: Record<string, any>) {
  resetForm()
  form.type = typeof rs.type === 'string' && rs.type ? rs.type : 'inline'
  form.tag = rs.tag == null ? [] : Array.isArray(rs.tag) ? rs.tag.map(String) : [String(rs.tag)]
  form.format = inferFormat(rs)
  form.path = typeof rs.path === 'string' ? rs.path : ''
  form.url = typeof rs.url === 'string' ? rs.url : ''
  form.initial_path = typeof rs.initial_path === 'string' ? rs.initial_path : ''
  form.update_interval = rs.update_interval ? String(rs.update_interval) : ''
  form.http_client = typeof rs.http_client === 'string' ? rs.http_client : ''
  if (Array.isArray(rs.rules)) {
    form.rulesJson = JSON.stringify(rs.rules, null, 2)
  }
}

function buildRuleSet(): Record<string, any> {
  const rs: Record<string, any> = {}
  if (form.type !== 'inline') rs.type = form.type
  if (form.tag.length) rs.tag = form.tag.length === 1 ? form.tag[0] : [...form.tag]
  if (form.type === 'inline') {
    if (form.rulesJson.trim()) {
      let rules: unknown
      try {
        rules = JSON.parse(form.rulesJson.trim())
      } catch (e) {
        throw new Error(`rules JSON 格式错误：${(e as Error).message}`)
      }
      if (!Array.isArray(rules)) throw new Error('rules 必须是数组')
      rs.rules = rules
    }
  } else {
    rs.format = form.format || inferFormat({ url: form.url, path: form.path })
    if (form.type === 'local') {
      if (form.path.trim()) rs.path = form.path.trim()
    } else {
      if (form.url.trim()) rs.url = form.url.trim()
      if (form.http_client.trim()) rs.http_client = form.http_client.trim()
      if (form.initial_path.trim()) rs.initial_path = form.initial_path.trim()
      if (form.update_interval.trim()) rs.update_interval = form.update_interval.trim()
    }
  }
  return rs
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
    fillRuleSetForm(parsed)
    jsonDialog.value = false
    showToast('已从 JSON 填充，请检查表单', 'success')
  } catch (e) {
    showToast((e as Error).message || '解析失败', 'error')
  }
}

const loadHttpClientTags = async () => {
  try {
    const config = (await api.config()) as { http_clients?: Array<{ tag?: unknown }> } | null
    httpClientTags.value = Array.isArray(config?.http_clients)
      ? config.http_clients.map((item) => String(item?.tag ?? '')).filter(Boolean)
      : []
  } catch {
    httpClientTags.value = []
  }
}

watch(
  [() => props.modelValue, () => props.isEdit, () => props.ruleSetData],
  ([visible, isEdit, data]) => {
    if (!visible) return
    activeTab.value = 'form'
    validateMsg.value = ''
    resetForm()
    if (isEdit && data && data.rule_set) {
      const rs = data.rule_set
      sourceJson.value = JSON.stringify(rs, null, 2)
      fillRuleSetForm(rs)
    } else {
      sourceJson.value = '{}'
    }
    loadHttpClientTags()
  },
  { immediate: true },
)

watch(activeTab, (tab) => {
  if (tab === 'source' && !srcTab.value?.isDirty()) {
    try {
      sourceJson.value = JSON.stringify(buildRuleSet(), null, 2)
    } catch {
      // ignore
    }
  }
})

const save = async () => {
  validateMsg.value = ''
  if (!form.tag.length) {
    validateMsg.value = '请填写 tag'
    showToast('请填写 tag', 'warning')
    return
  }
  if (form.type !== 'inline' && form.type === 'local' && !form.path.trim()) {
    validateMsg.value = 'local 规则集需要 path'
    showToast('local 规则集需要 path', 'warning')
    return
  }
  if (form.type === 'remote' && !form.url.trim()) {
    validateMsg.value = 'remote 规则集需要 url'
    showToast('remote 规则集需要 url', 'warning')
    return
  }
  if (form.type === 'remote' && form.tag.length > 1 && !hasRuleSetTemplate(form.url)) {
    validateMsg.value = '多个 tag 的远程规则集 URL 必须包含 {tag}'
    showToast(validateMsg.value, 'warning')
    return
  }
  if (repositoryNeedsNewCore.value) {
    validateMsg.value = '多 tag 远程规则集需要 sing-box 1.4.0 或更高版本'
    showToast(validateMsg.value, 'warning')
    return
  }
  saving.value = true
  try {
    const payload = srcTab.value?.isDirty()
      ? JSON.parse(srcTab.value.getText())
      : buildRuleSet()
    const payloadTags = Array.isArray(payload?.tag) ? payload.tag : payload?.tag ? [payload.tag] : []
    if (
      payload?.type === 'remote' &&
      payloadTags.length > 1 &&
      !hasRuleSetTemplate(payload?.url)
    ) {
      throw new Error('多个 tag 的远程规则集 URL 必须包含 {tag}')
    }
    if (
      payload?.type === 'remote' &&
      (payloadTags.length > 1 || hasRuleSetTemplate(payload?.url)) &&
      !supportsMultiTagRuleSet(version.value)
    ) {
      throw new Error('多 tag 远程规则集需要 sing-box 1.4.0 或更高版本')
    }
    if (props.isEdit && props.ruleSetData?.id) {
      await api.updateRuleSet(props.ruleSetData.id, payload)
    } else {
      await api.createRuleSet(payload)
    }
    showToast('保存成功', 'success')
    emit('update:modelValue', false)
    emit('saved')
    await statusStore.refresh()
  } catch (e) {
    validateMsg.value = (e as Error).message || '保存失败'
    showToast((e as Error).message || '保存失败', 'error')
  } finally {
    saving.value = false
  }
}

const handleDelete = async () => {
  if (!props.isEdit || !props.ruleSetData?.id) return
  const { confirmed } = await showConfirmDialog({
    title: t('delete') || '删除规则集',
    message: '确定删除该规则集？该操作不可恢复。',
    confirmText: t('delete') || '删除',
    confirmButtonClass: 'btn-error',
  })
  if (!confirmed) return
  saving.value = true
  try {
    await api.deleteRuleSet(props.ruleSetData.id)
    showToast('删除成功', 'success')
    emit('update:modelValue', false)
    emit('saved')
    await statusStore.refresh()
  } catch (e) {
    showToast((e as Error).message || '删除失败', 'error')
  } finally {
    saving.value = false
  }
}
</script>
