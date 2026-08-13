<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { showToast } from '@/helper/toast'
import { api } from '../api'

const loading = ref(false)
const items = ref<Array<{ level: string; message: string }>>([])

const load = async () => {
  loading.value = true
  try {
    const data = await api.diagnostics()
    items.value = data.diagnostics || []
  } catch (e) {
    showToast((e as Error).message || '诊断失败', 'error')
  } finally {
    loading.value = false
  }
}

const byLevel = (level: string) => items.value.filter((i) => i.level === level)

onMounted(load)
</script>

<template>
  <div>
    <div class="mb-3.5 flex items-center gap-2.5">
      <button class="btn btn-primary btn-sm" :disabled="loading" @click="load">重新诊断</button>
      <span
        v-if="items.length"
        class="badge"
        :class="byLevel('error').length ? 'badge-error' : byLevel('warning').length ? 'badge-warning' : 'badge-success'"
      >
        错误 {{ byLevel('error').length }} · 警告 {{ byLevel('warning').length }} · 信息 {{ byLevel('info').length }}
      </span>
      <span class="text-xs text-[#909399]">静态分析当前配置：重复 tag、悬空引用、端口冲突、未使用资源。深度校验由写操作管线的 sing-box 干跑负责</span>
    </div>

    <div v-if="!items.length" class="py-10 text-center">
      <p class="text-sm text-[#909399]">配置未发现问题</p>
    </div>

    <template v-else>
      <div v-for="level in ['error', 'warning', 'info']" :key="level" class="mb-2 flex flex-col gap-2">
        <div
          v-for="(item, i) in byLevel(level)"
          :key="`${level}-${i}`"
          class="alert text-sm"
          :class="level === 'error' ? 'alert-error' : level === 'warning' ? 'alert-warning' : 'alert-info'"
        >
          <span>{{ item.message }}</span>
        </div>
      </div>
    </template>
  </div>
</template>
