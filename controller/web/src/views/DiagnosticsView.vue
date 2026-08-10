<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'

const loading = ref(false)
const items = ref<Array<{ level: string; message: string }>>([])

const load = async () => {
  loading.value = true
  try {
    const data = await api.diagnostics()
    items.value = data.diagnostics || []
  } catch (e) {
    ElMessage.error((e as Error).message || '诊断失败')
  } finally {
    loading.value = false
  }
}

const byLevel = (level: string) => items.value.filter((i) => i.level === level)

const alertType = (level: string) => (level === 'error' ? 'error' : level === 'warning' ? 'warning' : 'info')

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="toolbar">
      <el-button type="primary" :loading="loading" @click="load">重新诊断</el-button>
      <el-tag v-if="items.length" :type="byLevel('error').length ? 'danger' : byLevel('warning').length ? 'warning' : 'success'">
        错误 {{ byLevel('error').length }} · 警告 {{ byLevel('warning').length }} · 信息 {{ byLevel('info').length }}
      </el-tag>
      <span class="hint">静态分析当前配置：重复 tag、悬空引用、端口冲突、未使用资源。深度校验由写操作管线的 sing-box 干跑负责</span>
    </div>

    <div v-if="!items.length" class="empty">
      <el-empty description="配置未发现问题" />
    </div>

    <template v-else>
      <div v-for="level in ['error', 'warning', 'info']" :key="level">
        <el-alert
          v-for="(item, i) in byLevel(level)"
          :key="`${level}-${i}`"
          :type="alertType(level)"
          :closable="false"
          :title="item.message"
          style="margin-bottom: 8px"
        />
      </div>
    </template>
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
.empty {
  padding: 40px 0;
}
</style>
