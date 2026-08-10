// 日志状态（对接 sing-box service API 的 SubscribeLog 增量流）
// 数据结构参考 zashboard（seq/time/level/message），渲染用其 LogsCard/HighlightText
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { subscribeLogs } from '../api/singbox'
import { LogLevel, type Log } from '@/gen/daemon/started_service_pb'

const LEVEL_LABEL: Record<number, string> = {
  [LogLevel.PANIC]: 'PANIC',
  [LogLevel.FATAL]: 'FATAL',
  [LogLevel.ERROR]: 'ERROR',
  [LogLevel.WARN]: 'WARN',
  [LogLevel.INFO]: 'INFO',
  [LogLevel.DEBUG]: 'DEBUG',
  [LogLevel.TRACE]: 'TRACE'
}

export interface LogEntry {
  seq: number
  time: string
  level: LogLevel
  levelLabel: string
  message: string
  /** 渲染时注入当前搜索词（HighlightText 用） */
  filter: string
}

const MAX_LOGS = 2000

export const useLogsStore = defineStore('logs', () => {
  const logs = ref<LogEntry[]>([])
  const connected = ref(false)
  const failed = ref(false)
  const paused = ref(false)
  const closed = ref(false)
  const filter = ref('')
  const levelFilter = ref<LogLevel | 'all'>('all')

  let seq = 0
  const abort = new AbortController()
  let started = false

  function pushMessage(level: LogLevel, message: string) {
    logs.value.push({
      seq: ++seq,
      time: new Date().toLocaleTimeString(),
      level,
      levelLabel: LEVEL_LABEL[level] || String(level),
      message,
      filter: filter.value
    })
    if (logs.value.length > MAX_LOGS) logs.value = logs.value.slice(-MAX_LOGS)
  }

  function handleLog(log: Log) {
    if (closed.value || paused.value) return
    if (log.reset) {
      logs.value = []
      seq = 0
    }
    for (const m of log.messages) {
      pushMessage(m.level, m.message)
    }
    connected.value = true
  }

  async function run() {
    let delay = 1000
    while (!closed.value) {
      try {
        for await (const log of subscribeLogs(abort.signal)) {
          handleLog(log)
          delay = 1000
        }
      } catch {
        connected.value = false
        failed.value = true
      }
      if (closed.value) break
      await new Promise((r) => setTimeout(r, delay))
      delay = Math.min(delay * 2, 30000)
    }
  }

  function start() {
    if (started) return
    started = true
    void run()
  }

  function stop() {
    closed.value = true
    abort.abort()
  }

  const filteredLogs = computed(() => {
    let list = levelFilter.value === 'all' ? logs.value : logs.value.filter((l) => l.level === levelFilter.value)
    if (filter.value.trim()) {
      const q = filter.value.toLowerCase()
      list = list.filter((l) => l.message.toLowerCase().includes(q))
    }
    return list
  })

  return {
    logs,
    connected,
    failed,
    paused,
    filter,
    levelFilter,
    filteredLogs,
    start,
    stop,
    togglePause: () => (paused.value = !paused.value),
    clear: () => {
      logs.value = []
      seq = 0
    }
  }
})
