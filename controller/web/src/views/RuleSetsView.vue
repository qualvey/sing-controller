<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { showToast } from '@/helper/toast'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { PlusIcon, RefreshCw } from 'lucide-vue-next'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { SelectField, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'
import { api } from '../api'
import SourcePane from '../components/SourcePane.vue'
import ResourceSourceTab from '../components/ResourceSourceTab.vue'
import { useStatusStore } from '../stores/status'

const statusStore = useStatusStore()

const loading = ref(false)
const outerTab = ref('form')
const saving = ref(false)
const items = ref<Array<{ id: string; rule_set: Record<string, any> }>>([])

const dialogVisible = ref(false)
const srcTab = ref<InstanceType<typeof ResourceSourceTab>>()
const sourceJson = ref('{}')
const isEdit = ref(false)
const editingId = ref('')

// sing-box 1.14 RuleSet（option/rule_set.go）：inline 内联 / local 本地文件 / remote 远程 URL
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
  rulesJson: ''
})

// tag chip 输入（替代 el-select multiple+allow-create）：回车/逗号添加，chip 可删
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
    // 始终显式写 format（sing-box Marshal 会丢 format，保存时固定避免推断漂移）
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

function openCreate() {
  isEdit.value = false
  editingId.value = ''
  sourceJson.value = '{}'
  resetForm()
  dialogVisible.value = true
}

function openEdit(row: { id: string; rule_set: Record<string, any> }) {
  isEdit.value = true
  editingId.value = row.id
  sourceJson.value = JSON.stringify(row.rule_set, null, 2)
  resetForm()
  const rs = row.rule_set
  const type = typeof rs.type === 'string' && rs.type ? rs.type : 'inline'
  form.type = type
  form.tag = rs.tag == null ? [] : Array.isArray(rs.tag) ? rs.tag.map(String) : [String(rs.tag)]
  form.format = inferFormat(rs)
  form.path = typeof rs.path === 'string' ? rs.path : ''
  form.url = typeof rs.url === 'string' ? rs.url : ''
  form.initial_path = typeof rs.initial_path === 'string' ? rs.initial_path : ''
  form.update_interval = rs.update_interval ? String(rs.update_interval) : ''
  if (Array.isArray(rs.rules)) {
    form.rulesJson = JSON.stringify(rs.rules, null, 2)
  }
  dialogVisible.value = true
}

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
    const payload = srcTab.value?.isDirty() ? JSON.parse(srcTab.value.getText()) : buildRuleSet()
    if (isEdit.value) {
      await api.updateRuleSet(editingId.value, payload)
    } else {
      await api.createRuleSet(payload)
    }
    showToast('保存成功', 'success')
    dialogVisible.value = false
    await load()
    await statusStore.refresh()
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
  } finally {
    saving.value = false
  }
}

const remove = async (row: { id: string; rule_set: Record<string, any> }) => {
  const first = await showConfirmDialog({
    title: '删除确认',
    message: '确定删除该规则集？被引用的规则将失效。',
    confirmText: '删除',
    confirmButtonClass: 'btn-error'
  })
  if (!first.confirmed) return
  try {
    await api.deleteRuleSet(row.id)
    showToast('删除成功', 'success')
    await load()
    await statusStore.refresh()
  } catch (e) {
    const err = e as Error & { references?: string[] }
    if (err.references?.length) {
      const second = await showConfirmDialog({
        title: '被引用确认',
        message: `该规则集被引用：${err.references.join('、')}\n删除后将自动从这些规则中移除引用。确认删除？`,
        confirmText: '确认删除',
        confirmButtonClass: 'btn-error'
      })
      if (!second.confirmed) return
      try {
        await api.deleteRuleSet(row.id, true)
        showToast('删除成功（已移除引用）', 'success')
        await load()
        await statusStore.refresh()
      } catch (e2) {
        showToast((e2 as Error).message || '删除失败', 'error')
      }
      return
    }
    showToast(err.message || '删除失败', 'error')
  }
}

const load = async () => {
  loading.value = true
  try {
    const data = await api.ruleSets()
    items.value = data.rule_sets || []
  } catch (e) {
    showToast((e as Error).message || '加载规则集失败', 'error')
  } finally {
    loading.value = false
  }
}

function fmtList(v: unknown): string {
  if (Array.isArray(v)) return v.join(', ')
  return v == null ? '—' : String(v)
}

// sing-box RuleSet Marshal 会丢弃 format（实测）→ 从 url/path 扩展名推断（.srs→binary，其余 source）
function inferFormat(rs: Record<string, any>): string {
  if (typeof rs.format === 'string' && rs.format) return rs.format
  const src = String(rs.url || rs.path || '')
  return src.endsWith('.srs') ? 'binary' : 'source'
}

onMounted(load)
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
          <button class="btn btn-primary btn-sm" @click="openCreate">
            <PlusIcon class="h-4 w-4" />
            新建规则集
          </button>
          <button class="btn btn-ghost btn-sm" :disabled="loading" @click="load">
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
          <span class="text-xs text-[#909399]">规则集定义在 route.rule_set 段（sing-box 1.14）：inline 内联 / local 本地文件 / remote 远程 URL；规则通过 rule_set 字段引用</span>
        </div>

        <div class="overflow-hidden rounded-lg border border-[#e4e7ed] bg-white dark:border-[#303030] dark:bg-[#1d1e1f]">
          <div v-if="loading && !items.length" class="p-10 text-center text-sm text-[#909399]">加载中…</div>
          <table v-else class="table table-sm w-full">
            <thead>
              <tr>
                <th class="w-[140px]">tag(s)</th>
                <th class="w-20">type</th>
                <th class="w-20">format</th>
                <th>来源</th>
                <th class="w-[130px]">update_interval</th>
                <th class="w-[130px] text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in items" :key="row.id">
                <td class="font-mono text-xs">{{ fmtList(row.rule_set.tag) }}</td>
                <td class="text-xs">{{ row.rule_set.type || 'inline' }}</td>
                <td class="text-xs">{{ inferFormat(row.rule_set) }}</td>
                <td>
                  <span v-if="row.rule_set.url" class="font-mono text-xs">{{ row.rule_set.url }}</span>
                  <span v-else-if="row.rule_set.path" class="font-mono text-xs">{{ row.rule_set.path }}</span>
                  <span v-else-if="Array.isArray(row.rule_set.rules)" class="text-xs">{{ row.rule_set.rules.length }} 条内联规则</span>
                  <span v-else class="text-xs text-[#c0c4cc]">—</span>
                </td>
                <td class="text-xs">{{ row.rule_set.update_interval || '—' }}</td>
                <td class="text-right">
                  <button class="btn btn-ghost btn-xs text-primary" @click="openEdit(row)">编辑</button>
                  <button class="btn btn-ghost btn-xs text-error" @click="remove(row)">删除</button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!loading && !items.length" class="py-8 text-center text-sm text-[#606266] dark:text-[#a6b0bf]">暂无规则集</div>
        </div>
      </TabsContent>

      <TabsContent value="source">
        <SourcePane segment="route.rule_set" @saved="load" />
      </TabsContent>
    </TabsRoot>

    <!-- 新建/编辑弹窗 -->
    <DialogWrapper v-model="dialogVisible" :title="isEdit ? '编辑规则集' : '新建规则集'" box-class="max-w-[620px]">
      <TabsRoot :model-value="'form'">
        <TabsList>
          <TabsTrigger value="form">表单</TabsTrigger>
          <TabsTrigger value="source">源码</TabsTrigger>
        </TabsList>
        <TabsContent value="form">
          <div class="grid gap-x-4 gap-y-5" style="grid-template-columns: 130px minmax(0, 1fr)">
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">type <span class="text-destructive">*</span></label>
            <SelectField v-model="form.type">
              <SelectTrigger><SelectValue placeholder="选择类型" /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="t in RULE_SET_TYPES" :key="t" :value="t">{{ t }}</SelectItem>
              </SelectContent>
            </SelectField>

            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tag <span class="text-destructive">*</span></label>
            <div class="flex flex-col gap-1.5">
              <div class="flex flex-wrap gap-1.5">
                <span v-for="t in form.tag" :key="t" class="badge badge-primary badge-outline gap-1">
                  {{ t }}
                  <button type="button" class="cursor-pointer text-xs opacity-70 hover:opacity-100" @click="removeTag(t)">×</button>
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
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">rules (JSON)</label>
              <div class="flex flex-col gap-1">
                <textarea
                  v-model="form.rulesJson"
                  rows="10"
                  class="textarea textarea-bordered w-full font-mono text-xs"
                  placeholder='[{"domain_suffix": ".cn"}, {"ip_cidr": ["10.0.0.0/8"]}]'
                />
                <p class="text-xs text-[#909399]">HeadlessRule 数组（无 action 的匹配规则）；多个 tag 时 path/url 需含 {tag} 占位符</p>
              </div>
            </template>
            <template v-else-if="form.type === 'local'">
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">format</label>
              <SelectField v-model="form.format">
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="f in FORMAT_OPTIONS" :key="f" :value="f">{{ f }}</SelectItem>
                </SelectContent>
              </SelectField>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">path <span class="text-destructive">*</span></label>
              <input v-model="form.path" type="text" class="input input-bordered input-sm w-full" placeholder="规则集文件路径（.json→source，.srs→binary，多 tag 时含 {tag}）" />
            </template>
            <template v-else>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">format</label>
              <SelectField v-model="form.format">
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="f in FORMAT_OPTIONS" :key="f" :value="f">{{ f }}</SelectItem>
                </SelectContent>
              </SelectField>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">url <span class="text-destructive">*</span></label>
              <input v-model="form.url" type="text" class="input input-bordered input-sm w-full" placeholder="https://example.com/rule-set.json（多 tag 时含 {tag}）" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">initial_path</label>
              <input v-model="form.initial_path" type="text" class="input input-bordered input-sm w-full" placeholder="初始下载路径（可选）" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">update_interval</label>
              <input v-model="form.update_interval" type="text" class="input input-bordered input-sm w-full" placeholder="如 24h（可选）" />
            </template>
          </div>
        </TabsContent>
        <TabsContent value="source">
          <ResourceSourceTab ref="srcTab" :initial="sourceJson" />
        </TabsContent>
      </TabsRoot>
      <div class="mt-5 flex justify-end gap-2">
        <button class="btn btn-ghost btn-sm" @click="dialogVisible = false">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="saving" @click="save">保存</button>
      </div>
    </DialogWrapper>
  </div>
</template>
