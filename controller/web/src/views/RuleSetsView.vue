<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import SourcePane from '../components/SourcePane.vue'
import { useStatusStore } from '../stores/status'

const statusStore = useStatusStore()

const loading = ref(false)
const outerTab = ref('form')
const saving = ref(false)
const items = ref<Array<{ id: string; rule_set: Record<string, any> }>>([])

const dialogVisible = ref(false)
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

const resetForm = () => {
  form.type = 'remote'
  form.tag = []
  form.format = 'source'
  form.path = ''
  form.url = ''
  form.initial_path = ''
  form.update_interval = ''
  form.rulesJson = ''
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
  resetForm()
  dialogVisible.value = true
}

function openEdit(row: { id: string; rule_set: Record<string, any> }) {
  isEdit.value = true
  editingId.value = row.id
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
    ElMessage.warning('请填写 tag')
    return
  }
  if (form.type !== 'inline' && form.type === 'local' && !form.path.trim()) {
    ElMessage.warning('local 规则集需要 path')
    return
  }
  if (form.type === 'remote' && !form.url.trim()) {
    ElMessage.warning('remote 规则集需要 url')
    return
  }
  saving.value = true
  try {
    const payload = buildRuleSet()
    if (isEdit.value) {
      await api.updateRuleSet(editingId.value, payload)
    } else {
      await api.createRuleSet(payload)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    await load()
    await statusStore.refresh()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

const remove = async (row: { id: string; rule_set: Record<string, any> }) => {
  try {
    await ElMessageBox.confirm('确定删除该规则集？被引用的规则将失效。', '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  try {
    await api.deleteRuleSet(row.id)
    ElMessage.success('删除成功')
    await load()
    await statusStore.refresh()
  } catch (e) {
    const err = e as Error & { references?: string[] }
    if (err.references?.length) {
      try {
        await ElMessageBox.confirm(
          `该规则集被引用：${err.references.join('、')}\\n删除后将自动从这些规则中移除引用。确认删除？`,
          '被引用确认',
          { type: 'warning', confirmButtonText: '确认删除', cancelButtonText: '取消' }
        )
      } catch {
        return
      }
      try {
        await api.deleteRuleSet(row.id, true)
        ElMessage.success('删除成功（已移除引用）')
        await load()
        await statusStore.refresh()
      } catch (e2) {
        ElMessage.error((e2 as Error).message || '删除失败')
      }
      return
    }
    ElMessage.error(err.message || '删除失败')
  }
}

const load = async () => {
  loading.value = true
  try {
    const data = await api.ruleSets()
    items.value = data.rule_sets || []
  } catch (e) {
    ElMessage.error((e as Error).message || '加载规则集失败')
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
  <div class="page">
    <el-tabs v-model="outerTab">
      <el-tab-pane label="表单" name="form">
    <div class="toolbar">
      <el-button type="primary" @click="openCreate">新建规则集</el-button>
      <el-button :loading="loading" @click="load">刷新</el-button>
      <span class="hint">规则集定义在 route.rule_set 段（sing-box 1.14）：inline 内联 / local 本地文件 / remote 远程 URL；规则通过 rule_set 字段引用</span>
    </div>

    <el-table :data="items" v-loading="loading" border stripe>
      <el-table-column label="tag(s)" min-width="140">
        <template #default="{ row }">{{ fmtList(row.rule_set.tag) }}</template>
      </el-table-column>
      <el-table-column label="type" width="80">
        <template #default="{ row }">{{ row.rule_set.type || 'inline' }}</template>
      </el-table-column>
      <el-table-column label="format" width="80">
        <template #default="{ row }">{{ inferFormat(row.rule_set) }}</template>
      </el-table-column>
      <el-table-column label="来源" min-width="220">
        <template #default="{ row }">
          <span v-if="row.rule_set.url" class="mono">{{ row.rule_set.url }}</span>
          <span v-else-if="row.rule_set.path" class="mono">{{ row.rule_set.path }}</span>
          <span v-else-if="Array.isArray(row.rule_set.rules)">{{ row.rule_set.rules.length }} 条内联规则</span>
          <span v-else class="muted">—</span>
        </template>
      </el-table-column>
      <el-table-column label="update_interval" width="130">
        <template #default="{ row }">{{ row.rule_set.update_interval || '—' }}</template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" link @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" link @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑规则集' : '新建规则集'" width="620px" :close-on-click-modal="false">
      <el-form label-width="130px">
        <el-form-item label="type" required>
          <el-select v-model="form.type" style="width: 100%">
            <el-option v-for="t in RULE_SET_TYPES" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>
        <el-form-item label="tag" required>
          <el-select v-model="form.tag" multiple filterable allow-create default-first-option style="width: 100%" placeholder="规则集 tag（inline 仅支持单个）">
            <el-option v-for="t in form.tag" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>
        <template v-if="form.type === 'inline'">
          <el-form-item label="rules (JSON)">
            <el-input v-model="form.rulesJson" type="textarea" :rows="10" class="mono" placeholder='[{"domain_suffix": ".cn"}, {"ip_cidr": ["10.0.0.0/8"]}]' />
          </el-form-item>
          <span class="hint">HeadlessRule 数组（无 action 的匹配规则）；多个 tag 时 path/url 需含 {tag} 占位符</span>
        </template>
        <template v-else-if="form.type === 'local'">
          <el-form-item label="format">
            <el-select v-model="form.format" style="width: 100%">
              <el-option v-for="f in FORMAT_OPTIONS" :key="f" :label="f" :value="f" />
            </el-select>
          </el-form-item>
          <el-form-item label="path" required>
            <el-input v-model="form.path" placeholder="规则集文件路径（.json→source，.srs→binary，多 tag 时含 {tag}）" />
          </el-form-item>
        </template>
        <template v-else>
          <el-form-item label="format">
            <el-select v-model="form.format" style="width: 100%">
              <el-option v-for="f in FORMAT_OPTIONS" :key="f" :label="f" :value="f" />
            </el-select>
          </el-form-item>
          <el-form-item label="url" required>
            <el-input v-model="form.url" placeholder="https://example.com/rule-set.json（多 tag 时含 {tag}）" />
          </el-form-item>
          <el-form-item label="initial_path">
            <el-input v-model="form.initial_path" placeholder="初始下载路径（可选）" />
          </el-form-item>
          <el-form-item label="update_interval">
            <el-input v-model="form.update_interval" placeholder="如 24h（可选）" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
      </el-tab-pane>
      <el-tab-pane label="源码" name="source">
        <SourcePane segment="route.rule_set" @saved="load" />
      </el-tab-pane>
    </el-tabs>
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
.mono {
  font-family: ui-monospace, Consolas, monospace;
  font-size: 12px;
}
.muted {
  color: #c0c4cc;
}
</style>
