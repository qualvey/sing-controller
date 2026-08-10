<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { subscribeLogs } from '../api/singbox'
import { LogLevel, type Log } from '@/gen/daemon/started_service_pb'

const MAX_LOGS = 500
const logs = ref<{ level: LogLevel; message: string; time: string }[]>([])
const connected = ref(false)
const closed = ref(false)
const autoScroll = ref(true)
const listRef = ref<HTMLElement>()
const filter = ref<LogLevel | 'all'>('all')

const LEVEL_META: Record<number, { label: string; cls: string }> = {
  [LogLevel.PANIC]: { label: 'PANIC', cls: 'text-red-500 font-bold' },
  [LogLevel.FATAL]: { label: 'FATAL', cls: 'text-red-500 font-bold' },
  [LogLevel.ERROR]: { label: 'ERROR', cls: 'text-red-500' },
  [LogLevel.WARN]: { label: 'WARN', cls: 'text-yellow-500' },
  [LogLevel.INFO]: { label: 'INFO', cls: 'text-blue-500' },
  [LogLevel.DEBUG]: { label: 'DEBUG', cls: 'text-gray-500 dark:text-gray-400' },
  [LogLevel.TRACE]: { label: 'TRACE', cls: 'text-gray-400 dark:text-gray-500' }
}

const levelLabel = (l: LogLevel) => LEVEL_META[l]?.label || String(l)
const levelCls = (l: LogLevel) => LEVEL_META[l]?.cls || ''

const filtered = computed(() =>
  filter.value === 'all' ? logs.value : logs.value.filter((l) => l.level === filter.value)
)

const handleLogs = (log: Log) => {
  if (closed.value) return
  if (log.reset) logs.value = []
  for (const m of log.messages) {
    logs.value.push({
      level: m.level,
      message: m.message,
      time: new Date().toLocaleTimeString()
    })
  }
  if (logs.value.length > MAX_LOGS) {
    logs.value = logs.value.slice(-MAX_LOGS)
  }
  connected.value = true
  if (autoScroll.value) {
    void nextTick(() => {
      const el = listRef.value
      if (el) el.scrollTop = el.scrollHeight
    })
  }
}

const run = async () => {
  try {
    for await (const log of subscribeLogs()) {
      handleLogs(log)
    }
  } catch {
    connected.value = false
  }
}

const onScroll = () => {
  const el = listRef.value
  if (!el) return
  autoScroll.value = el.scrollHeight - el.scrollTop - el.clientHeight < 40
}

onMounted(() => {
  void run()
})
onBeforeUnmount(() => {
  closed.value = true
})
</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- 工具栏 -->
    <div class="flex flex-wrap items-center gap-3 rounded-lg border border-[var(--el-border-color)] bg-[var(--el-bg-color)] px-4 py-3">
      <el-radio-group v-model="filter" size="small">
        <el-radio-button value="all">全部</el-radio-button>
        <el-radio-button value="info">INFO</el-radio-button>
        <el-radio-button value="warn">WARN</el-radio-button>
        <el-radio-button value="error">ERROR</el-radio-button>
        <el-radio-button value="debug">DEBUG</el-radio-button>
      </el-radio-group>
      <span class="ml-auto flex items-center gap-2 text-xs text-[var(--el-text-color-secondary)]">
        <span class="inline-block h-2 w-2 rounded-full" :class="connected ? 'bg-green-500' : 'bg-red-500'"></span>
        {{ connected ? '实时日志' : 'service API 不可用' }}
        <el-switch v-model="autoScroll" size="small" active-text="自动滚动" />
      </span>
    </div>

    <!-- 日志区 -->
    <div class="overflow-hidden rounded-lg border border-[var(--el-border-color)] bg-[var(--el-bg-color)]">
      <div
        ref="listRef"
        class="h-[calc(100vh-220px)] overflow-y-auto p-3 font-mono text-[12.5px] leading-relaxed"
        @scroll.passive="onScroll"
      >
        <div
          v-for="(l, i) in filtered"
          :key="i"
          class="flex gap-2 whitespace-pre-wrap break-all px-1 hover:bg-[var(--el-fill-color-light)]"
        >
          <span class="shrink-0 text-[var(--el-text-color-placeholder)]">{{ l.time }}</span>
          <span class="w-12 shrink-0 text-right" :class="levelCls(l.level)">{{ levelLabel(l.level) }}</span>
          <span class="text-[var(--el-text-color-primary)]">{{ l.message }}</span>
        </div>
        <div v-if="!connected" class="py-8 text-center text-sm text-[var(--el-text-color-secondary)]">
          service API 未配置或不可用（sing-box 需启用 services[type=api]）
        </div>
        <div v-else-if="!filtered.length" class="py-8 text-center text-sm text-[var(--el-text-color-secondary)]">
          暂无日志
        </div>
      </div>
    </div>
  </div>
</template>
