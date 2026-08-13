<script setup lang="ts">
// Logs 页：虚拟滚动 + 搜索高亮 + ANSI 渲染（组件搬运自 zashboard）
import { computed, ref, watch } from 'vue'
import { showToast } from '@/helper/toast'
import {
  ArrowDownTrayIcon,
  MagnifyingGlassIcon,
  PauseIcon,
  PlayIcon,
  TrashIcon
} from '@heroicons/vue/24/outline'
import SegmentedControl from '@/components/common/SegmentedControl.vue'
import VirtualScroller from '@/components/common/VirtualScroller.vue'
import LogsCard from '@/components/logs/LogsCard.vue'
import { useLogsStore } from '@/stores/logs'
import { LogLevel } from '@/gen/daemon/started_service_pb'

const logsStore = useLogsStore()

const LEVEL_OPTIONS = [
  { label: '全部', value: 'all' },
  { label: 'INFO', value: 'info' },
  { label: 'WARN', value: 'warn' },
  { label: 'ERROR', value: 'error' },
  { label: 'DEBUG', value: 'debug' }
]

// SegmentedControl 是字符串 v-model；store 用 LogLevel | 'all'
const LEVEL_MAP: Record<string, LogLevel | 'all'> = {
  all: 'all',
  info: LogLevel.INFO,
  warn: LogLevel.WARN,
  error: LogLevel.ERROR,
  debug: LogLevel.DEBUG
}
const LEVEL_STR = Object.fromEntries(Object.entries(LEVEL_MAP).map(([k, v]) => [String(v), k]))
const levelStr = ref(LEVEL_STR[String(logsStore.levelFilter)] ?? 'all')
watch(levelStr, (v) => {
  logsStore.levelFilter = LEVEL_MAP[v] ?? 'all'
})

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
  showToast('已清空', 'success')
}

const onConnectionClick = (id: string) => {
  // 跳转到 Connections 页（预留）
  void id
}
</script>

<template>
  <div class="flex h-full flex-col gap-4">
    <!-- 工具栏 -->
    <div class="flex flex-wrap items-center gap-3 rounded-lg border border-[#e4e7ed] bg-white px-4 py-3 dark:border-[#303030] dark:bg-[#1d1e1f]">
      <SegmentedControl v-model="levelStr" :options="LEVEL_OPTIONS" />
      <label class="relative">
        <MagnifyingGlassIcon class="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-[#909399]" />
        <input
          v-model="logsStore.filter"
          type="text"
          class="input input-sm input-bordered w-60 pl-9"
          placeholder="搜索日志（支持正则）"
        />
      </label>
      <span class="ml-auto flex items-center gap-2 text-xs text-[#606266] dark:text-[#a6b0bf]">
        <span class="inline-block h-2 w-2 rounded-full" :class="logsStore.connected ? 'bg-green-500' : (logsStore.failed ? 'bg-red-500' : 'bg-yellow-500')"></span>
        {{ logsStore.connected ? `${logsStore.logs.length} 条` : (logsStore.failed ? 'service API 不可用' : '连接中…') }}
        <button class="btn btn-ghost btn-xs" @click="logsStore.togglePause()">
          <PauseIcon v-if="!logsStore.paused" class="h-4 w-4" />
          <PlayIcon v-else class="h-4 w-4" />
          {{ logsStore.paused ? '恢复' : '暂停' }}
        </button>
        <button class="btn btn-ghost btn-xs" @click="clearLogs">
          <TrashIcon class="h-4 w-4" />
          清空
        </button>
        <button class="btn btn-ghost btn-xs" @click="downloadAll">
          <ArrowDownTrayIcon class="h-4 w-4" />
          下载
        </button>
      </span>
    </div>

    <!-- 虚拟滚动日志区（zashboard VirtualScroller） -->
    <div class="min-h-0 flex-1 overflow-hidden rounded-lg border border-[#e4e7ed] bg-white dark:border-[#303030] dark:bg-[#1d1e1f]">
      <VirtualScroller :data="renderLogs" :size="44" content-class="p-2">
        <template #default="{ item }: { item: any }">
          <LogsCard :log="item" @connection-click="onConnectionClick" />
        </template>
      </VirtualScroller>
    </div>
  </div>
</template>
