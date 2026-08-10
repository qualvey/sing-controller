// 全局运行时状态：单例订阅 connections 流（Proxies 组速率 / Connections 页共用）
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { subscribeConnections, closeConnection, closeAllConnections } from '../api/singbox'
import { ConnectionEventType, type Connection, type ConnectionEvents } from '@/gen/daemon/started_service_pb'

export const useRuntimeStore = defineStore('runtime', () => {
  const connMap = ref<Map<string, Connection>>(new Map())
  const upSpeed = ref(0)
  const downSpeed = ref(0)
  // 每个出站的实时下载速率（组卡片显示用）
  const groupSpeed = ref<Record<string, number>>({})
  const connected = ref(false)
  const paused = ref(false)
  const closed = ref(false)

  let lastBatchAt = Date.now()
  let pendingEvents: ConnectionEvents[] = []

  function applyBatch(ev: ConnectionEvents) {
    if (ev.reset) {
      connMap.value = new Map()
      groupSpeed.value = {}
    }
    const now = Date.now()
    const dt = Math.max(0.3, (now - lastBatchAt) / 1000)
    lastBatchAt = now
    let up = 0
    let down = 0
    const speedAcc: Record<string, number> = {}
    for (const e of ev.events) {
      switch (e.type) {
        case ConnectionEventType.CONNECTION_EVENT_NEW:
          if (e.connection) connMap.value.set(e.id, e.connection)
          break
        case ConnectionEventType.CONNECTION_EVENT_UPDATE: {
          up += Number(e.uplinkDelta)
          down += Number(e.downlinkDelta)
          const conn = e.connection
          if (conn) connMap.value.set(e.id, conn)
          // 按出站累计速率（优先 outbound 字段，回退 chainList 首元素）
          const ob = conn?.outbound || conn?.fromOutbound || conn?.chainList?.[0]
          if (ob) speedAcc[ob] = (speedAcc[ob] || 0) + Number(e.downlinkDelta)
          break
        }
        case ConnectionEventType.CONNECTION_EVENT_CLOSED:
          connMap.value.delete(e.id)
          break
      }
    }
    upSpeed.value = up / dt
    downSpeed.value = down / dt
    const next: Record<string, number> = {}
    for (const [k, v] of Object.entries(speedAcc)) next[k] = v / dt
    groupSpeed.value = next
    connected.value = true
  }

  function handleBatch(ev: ConnectionEvents) {
    if (paused.value) {
      pendingEvents.push(ev)
      return
    }
    applyBatch(ev)
  }

  async function run() {
    let delay = 1000
    while (!closed.value) {
      try {
        for await (const ev of subscribeConnections(abort.signal)) {
          handleBatch(ev)
          delay = 1000
        }
      } catch {
        connected.value = false
      }
      if (closed.value) break
      await new Promise((r) => setTimeout(r, delay))
      delay = Math.min(delay * 2, 30000)
    }
  }

  const abort = new AbortController()
  let started = false

  function start() {
    if (started) return
    started = true
    void run()
  }

  function stop() {
    closed.value = true
    abort.abort()
  }

  function togglePause() {
    paused.value = !paused.value
    if (!paused.value && pendingEvents.length) {
      const batch = pendingEvents
      pendingEvents = []
      batch.forEach(applyBatch)
    }
  }

  const connections = computed(() => Array.from(connMap.value.values()))

  /** 某出站（组 tag）的实时下载速率 */
  function groupDownloadSpeed(outbound: string): number {
    return groupSpeed.value[outbound] || 0
  }

  return {
    connections,
    connMap,
    upSpeed,
    downSpeed,
    groupSpeed,
    connected,
    paused,
    start,
    stop,
    togglePause,
    groupDownloadSpeed,
    closeOne: (id: string) => closeConnection(id),
    closeAll: () => closeAllConnections()
  }
})
