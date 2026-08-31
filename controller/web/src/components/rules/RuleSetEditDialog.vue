<template>
  <DialogWrapper
    :model-value="modelValue"
    :title="isEdit ? '编辑规则集' : '新建规则集'"
    box-class="max-w-[640px]"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <TabsRoot v-model="activeTab">
      <TabsList>
        <TabsTrigger value="form">表单</TabsTrigger>
        <TabsTrigger value="source">源码</TabsTrigger>
      </TabsList>

      <TabsContent value="form">
        <div
          class="grid gap-x-4 gap-y-4 pt-2 text-sm"
          style="grid-template-columns: 140px minmax(0, 1fr)"
        >
          <label class="text-base-content/70 pt-2">type <span class="text-destructive">*</span></label>
          <SelectField
            v-model="form.type"
            :options="RULE_SET_TYPES"
            placeholder="选择类型"
          />

          <label class="text-base-content/70 pt-2">tag <span class="text-destructive">*</span></label>
          <div class="flex flex-col gap-1.5">
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="t in form.tag"
                :key="t"
                class="badge badge-primary badge-outline gap-1"
              >
                {{ t }}
                <button
                  type="button"
                  class="cursor-pointer text-xs opacity-70 hover:opacity-100"
                  @click="removeTag(t)"
                >
                  ×
                </button>
              </span>
            </div>
            <input
              v-model="tagInput"
              type="text"
              class="input input-bordered input-sm w-full"
              :placeholder="form.type === 'inline' ? '单个 tag（回车添加）' : 'tag（回车/逗号添加，多个）'"
              @keydown.enter.prevent="addTag"
              @blur="addTag"
            />
          </div>

          <template v-if="form.type === 'inline'">
            <label class="text-base-content/70 pt-2">rules (JSON)</label>
            <div class="flex flex-col gap-1">
              <textarea
                v-model="form.rulesJson"
                rows="8"
                class="textarea textarea-bordered font-mono text-xs w-full"
                placeholder='[{"domain_suffix": ".cn"}, {"ip_cidr": ["10.0.0.0/8"]}]'
              />
              <p class="text-base-content/50 text-xs">
                HeadlessRule 数组（无 action 的匹配规则）；多个 tag 时 path/url 需含 {tag} 占位符
              </p>
            </div>
          </template>

          <template v-else-if="form.type === 'local'">
            <label class="text-base-content/70 pt-2">format</label>
            <SelectField
              v-model="form.format"
              :options="FORMAT_OPTIONS"
            />
            <label class="text-base-content/70 pt-2">path <span class="text-destructive">*</span></label>
            <input
              v-model="form.path"
              type="text"
              class="input input-bordered input-sm w-full font-mono"
              placeholder="规则集文件路径（.json→source，.srs→binary，多 tag 时含 {tag}）"
            />
          </template>

          <template v-else>
            <label class="text-base-content/70 pt-2">format</label>
            <SelectField
              v-model="form.format"
              :options="FORMAT_OPTIONS"
            />
            <label class="text-base-content/70 pt-2">url <span class="text-destructive">*</span></label>
            <input
              v-model="form.url"
              type="text"
              class="input input-bordered input-sm w-full font-mono"
              placeholder="https://example.com/rule-set.json（多 tag 时含 {tag}）"
            />
            <label class="text-base-content/70 pt-2">initial_path</label>
            <input
              v-model="form.initial_path"
              type="text"
              class="input input-bordered input-sm w-full font-mono"
              placeholder="初始下载路径（可选）"
            />
            <label class="text-base-content/70 pt-2">update_interval</label>
            <input
              v-model="form.update_interval"
              type="text"
              class="input input-bordered input-sm w-full"
              placeholder="如 24h（可选）"
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
        @click="emit('update:modelValue', false)"
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
import { api } from '@/api'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import ResourceSourceTab from '@/components/ResourceSourceTab.vue'
import { SelectField } from '@/components/ui/select'
import { TabsContent, TabsList, TabsRoot, TabsTrigger } from '@/components/ui/tabs'
import { showToast } from '@/helper/toast'
import { useStatusStore } from '@/store/status'
import { reactive, ref, watch } from 'vue'

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

const RULE_SET_TYPES = ['inline', 'local', 'remote']
const FORMAT_OPTIONS = ['source', 'binary']

const form = reactive({
  type: 'remote',
  tag: [] as string[],
  format: 'source',
  path: '',
  url: '',
  initial_path: '',
  update_interval: '',
  rulesJson: '',
})

const tagInput = ref('')

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

const resetForm = () => {
  form.type = 'remote'
  form.tag = []
  form.format = 'source'
  form.path = ''
  form.url = ''
  form.initial_path = ''
  form.update_interval = ''
  form.rulesJson = ''
  tagInput.value = ''
}

function inferFormat(rs: Record<string, any>): string {
  if (typeof rs.format === 'string' && rs.format) return rs.format
  const src = String(rs.url || rs.path || '')
  return src.endsWith('.srs') ? 'binary' : 'source'
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
      if (form.initial_path.trim()) rs.initial_path = form.initial_path.trim()
      if (form.update_interval.trim()) rs.update_interval = form.update_interval.trim()
    }
  }
  return rs
}

watch(
  () => [props.modelValue, props.isEdit, props.ruleSetData],
  ([visible, isEdit, data]) => {
    if (!visible) return
    activeTab.value = 'form'
    resetForm()
    if (isEdit && data && data.rule_set) {
      const rs = data.rule_set
      sourceJson.value = JSON.stringify(rs, null, 2)
      form.type = typeof rs.type === 'string' && rs.type ? rs.type : 'inline'
      form.tag = rs.tag == null ? [] : Array.isArray(rs.tag) ? rs.tag.map(String) : [String(rs.tag)]
      form.format = inferFormat(rs)
      form.path = typeof rs.path === 'string' ? rs.path : ''
      form.url = typeof rs.url === 'string' ? rs.url : ''
      form.initial_path = typeof rs.initial_path === 'string' ? rs.initial_path : ''
      form.update_interval = rs.update_interval ? String(rs.update_interval) : ''
      if (Array.isArray(rs.rules)) {
        form.rulesJson = JSON.stringify(rs.rules, null, 2)
      }
    } else {
      sourceJson.value = '{}'
    }
  },
  { immediate: true },
)

const save = async () => {
  if (!form.tag.length) {
    showToast('请填写 tag', 'warning')
    return
  }
  if (form.type !== 'inline' && form.type === 'local' && !form.path.trim()) {
    showToast('local 规则集需要 path', 'warning')
    return
  }
  if (form.type === 'remote' && !form.url.trim()) {
    showToast('remote 规则集需要 url', 'warning')
    return
  }
  saving.value = true
  try {
    const payload = srcTab.value?.isDirty()
      ? JSON.parse(srcTab.value.getText())
      : buildRuleSet()
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
    showToast((e as Error).message || '保存失败', 'error')
  } finally {
    saving.value = false
  }
}
</script>
