<script setup lang="ts">
// Logs 页：虚拟滚动 + 搜索高亮 + ANSI 渲染（组件搬运自 zashboard）
import { computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, VideoPause, VideoPlay, Delete, Download } from '@element-plus/icons-vue'
import VirtualScroller from '@/components/common/VirtualScroller.vue'
import LogsCard from '@/components/logs/LogsCard.vue'
import { useLogsStore } from '@/stores/logs'
import { LogLevel } from '@/gen/daemon/started_service_pb'

const logsStore = useLogsStore()


const LEVEL_OPTIONS = [
  { label: '全部', value: 'all' },
  { label: 'INFO', value: LogLevel.INFO },
  { label: 'WARN', value: LogLevel.WARN },
  { label: 'ERROR', value: LogLevel.ERROR },
  { label: 'DEBUG', value: LogLevel.DEBUG }
]

// 注入搜索词到每条日志（HighlightText 高亮用）
const renderLogs = computed(() =>
  logsStore.filteredLogs.map((l) => ({ ...l, filter: logsStore.filter }))
)

const downloadAll = () => {
  const text = logsStore.logs
    .map((l) => `${l.seq}\t${l.time}\t${l.levelLabel}\t${l.message}`)
    .join('\n')
  const blob = new Blob([text], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `sing-box-logs-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.log`
  a.click()
  URL.revokeObjectURL(url)
}

const clearLogs = () => {
  logsStore.clear()
  ElMessage.success('已清空')
}

const onConnectionClick = (id: string) => {
  // 跳转到 Connections 页（预留）
  void id
}
</script>

<template>
  <div class="flex h-full flex-col gap-4">
    <!-- 工具栏 -->
    <div class="flex flex-wrap items-center gap-3 rounded-lg border border-[var(--el-border-color)] bg-[var(--el-bg-color)] px-4 py-3">
      <el-radio-group v-model="logsStore.levelFilter" size="small">
        <el-radio-button v-for="opt in LEVEL_OPTIONS" :key="String(opt.value)" :value="opt.value">
          {{ opt.label }}
        </el-radio-button>
      </el-radio-group>
      <el-input
        v-model="logsStore.filter"
        size="small"
        placeholder="搜索日志（支持正则）"
        :prefix-icon="Search"
        clearable
        style="width: 240px"
      />
      <span class="ml-auto flex items-center gap-2 text-xs text-[var(--el-text-color-secondary)]">
        <span class="inline-block h-2 w-2 rounded-full" :class="logsStore.connected ? 'bg-green-500' : (logsStore.failed ? 'bg-red-500' : 'bg-yellow-500')"></span>
        {{ logsStore.connected ? `${logsStore.logs.length} 条` : (logsStore.failed ? 'service API 不可用' : '连接中…') }}
        <el-button size="small" :icon="logsStore.paused ? VideoPlay : VideoPause" @click="logsStore.togglePause()">
          {{ logsStore.paused ? '恢复' : '暂停' }}
        </el-button>
        <el-button size="small" :icon="Delete" @click="clearLogs">清空</el-button>
        <el-button size="small" :icon="Download" @click="downloadAll">下载</el-button>
      </span>
    </div>

    <!-- 虚拟滚动日志区（zashboard VirtualScroller） -->
    <div class="min-h-0 flex-1 overflow-hidden rounded-lg border border-[var(--el-border-color)] bg-[var(--el-bg-color)]">
      <VirtualScroller :data="renderLogs" :size="44" content-class="p-2">
        <template #default="{ item }: { item: any }">
          <LogsCard :log="item" @connection-click="onConnectionClick" />
        </template>
      </VirtualScroller>
    </div>
  </div>
</template>
