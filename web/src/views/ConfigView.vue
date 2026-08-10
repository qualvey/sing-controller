<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import { useStatusStore } from '../stores/status'

const statusStore = useStatusStore()

const cfgText = ref('')
const loading = ref(false)
const saving = ref(false)

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
    const res = (await api.saveConfig(parsed)) as { load_error?: string; message?: string }
    if (res.load_error) {
      ElMessage.warning(res.message || `配置已保存，但加载新配置失败：${res.load_error}`)
    } else {
      ElMessage.success('配置已保存（已通过 sing-box 校验）')
    }
    await loadConfig()
    await statusStore.refresh()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
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
      <el-button :loading="loading" @click="loadConfig">刷新</el-button>
      <span class="hint">直接编辑 sing-box 主配置 JSON，保存时后端会做完整校验（未知字段/非法类型/实例化预检）</span>
      <span class="spacer" />
      <el-button type="primary" :loading="saving" @click="save">保存配置</el-button>
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
.hint {
  font-size: 12px;
  color: #909399;
}
.spacer {
  flex: 1;
}
</style>
