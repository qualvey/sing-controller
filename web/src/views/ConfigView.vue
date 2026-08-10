<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import { useStatusStore } from '../stores/status'

const statusStore = useStatusStore()

const cfgText = ref('')
const loading = ref(false)
const saving = ref(false)
const opLoading = ref('')

const loadConfig = async () => {
  loading.value = true
  try {
    cfgText.value = JSON.stringify(await api.config(), null, 2)
  } catch (e) {
    ElMessage.error((e as Error).message || '加载配置失败')
  } finally {
    loading.value = false
  }
}

const handleResult = (res: unknown) => {
  const r = (res ?? {}) as { reload_error?: string; message?: string }
  if (r.reload_error) {
    ElMessage.warning(r.message || `操作已提交，但实例重载失败：${r.reload_error}`)
  } else {
    ElMessage.success('操作成功')
  }
}

const save = async () => {
  let parsed: unknown
  try {
    parsed = JSON.parse(cfgText.value)
  } catch (e) {
    ElMessage.error(`JSON 格式错误：${(e as Error).message}`)
    return
  }
  if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
    ElMessage.error('配置必须是 JSON 对象')
    return
  }
  saving.value = true
  try {
    handleResult(await api.saveConfig(parsed))
    await loadConfig()
    await statusStore.refresh()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

const runOp = async (op: 'start' | 'stop' | 'reload') => {
  opLoading.value = op
  try {
    const res =
      op === 'start' ? await api.instanceStart() : op === 'stop' ? await api.instanceStop() : await api.reload()
    handleResult(res)
    await statusStore.refresh()
  } catch (e) {
    ElMessage.error((e as Error).message || '操作失败')
  } finally {
    opLoading.value = ''
  }
}

onMounted(() => {
  loadConfig()
  statusStore.refresh()
})
</script>

<template>
  <div class="page">
    <div class="toolbar">
      <el-button type="success" :loading="opLoading === 'start'" :disabled="!!statusStore.status?.running" @click="runOp('start')">
        启动实例
      </el-button>
      <el-button type="danger" :loading="opLoading === 'stop'" :disabled="!statusStore.status?.running" @click="runOp('stop')">
        停止实例
      </el-button>
      <el-button :loading="opLoading === 'reload'" @click="runOp('reload')">重载配置</el-button>
      <el-button :loading="loading" @click="loadConfig">刷新</el-button>
      <span class="spacer" />
      <el-button type="primary" :loading="saving" @click="save">保存配置（PUT /api/config）</el-button>
    </div>
    <el-input
      v-model="cfgText"
      type="textarea"
      :rows="26"
      class="mono-editor"
      spellcheck="false"
      placeholder="加载配置中..."
    />
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}
.spacer {
  flex: 1;
}
</style>
