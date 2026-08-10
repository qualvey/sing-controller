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

const ruleForm = reactive({
  inbound: [] as string[],
  network: [] as string[],
  outbound: '',
  extraJson: ''
})

const ruleRules: FormRules = {
  outbound: [{ required: true, message: 'outbound 必选', trigger: 'change' }]
}

const rows = computed(() => routes.value.map((r) => ({ id: r.id, ...r.rule })))

function fmtList(v: unknown): string {
  if (Array.isArray(v)) return v.join(', ')
  return v == null ? '—' : String(v)
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

const openCreate = () => {
  isEdit.value = false
  editingId.value = ''
  ruleForm.inbound = []
  ruleForm.network = []
  ruleForm.outbound = ''
  ruleForm.extraJson = ''
  routeDialogVisible.value = true
  ruleFormRef.value?.clearValidate()
}

const openEdit = (row: RouteRule) => {
  isEdit.value = true
  editingId.value = typeof row.id === 'string' ? row.id : ''
  ruleForm.inbound = Array.isArray(row.inbound) ? row.inbound.map(String) : []
  ruleForm.network = Array.isArray(row.network) ? row.network.map(String) : []
  ruleForm.outbound = typeof row.outbound === 'string' ? row.outbound : ''
  const extra: Record<string, unknown> = {}
  for (const k of Object.keys(row)) {
    if (k !== 'id' && k !== 'inbound' && k !== 'network' && k !== 'outbound') {
      extra[k] = row[k]
    }
  }
  ruleForm.extraJson = Object.keys(extra).length ? JSON.stringify(extra, null, 2) : ''
  routeDialogVisible.value = true
  ruleFormRef.value?.clearValidate()
}

function buildRule(): RouteRule {
  const rule: RouteRule = { outbound: ruleForm.outbound }
  if (ruleForm.inbound.length) rule.inbound = [...ruleForm.inbound]
  if (ruleForm.network.length) rule.network = [...ruleForm.network]
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
    rule.outbound = ruleForm.outbound
    if (ruleForm.inbound.length) rule.inbound = [...ruleForm.inbound]
    if (ruleForm.network.length) rule.network = [...ruleForm.network]
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
      <span class="hint">说明：通过整体读取/回写配置（GET/PUT /api/config）修改 route.final</span>
    </div>

    <div class="toolbar">
      <el-button type="primary" @click="openCreate">新建规则</el-button>
      <el-button :loading="loading" @click="loadRoutes">刷新</el-button>
    </div>

    <el-table :data="rows" v-loading="loading" border stripe>
      <el-table-column label="inbound" min-width="150">
        <template #default="{ row }">{{ fmtList(row.inbound) }}</template>
      </el-table-column>
      <el-table-column label="network" width="120">
        <template #default="{ row }">{{ fmtList(row.network) }}</template>
      </el-table-column>
      <el-table-column label="outbound" min-width="150">
        <template #default="{ row }">{{ row.outbound ?? '—' }}</template>
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
        <el-form-item label="outbound" prop="outbound">
          <el-select v-model="ruleForm.outbound" style="width: 100%" placeholder="选择出站">
            <el-option v-for="t in outboundTags" :key="t" :label="t" :value="t" />
          </el-select>
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
}
</style>
