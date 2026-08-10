<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Close } from '@element-plus/icons-vue'
import {
  subscribeConnections,
  closeConnection,
  closeAllConnections
} from '../api/singbox'
import {
  ConnectionEventType,
  type Connection,
  type ConnectionEvents
} from '@/gen/daemon/started_service_pb'

// gRPC 事件流维护的连接状态（NEW/UPDATE/CLOSED 增量）
const connMap = ref<Map<string, Connection>>(new Map())
const upSpeed = ref(0)
const downSpeed = ref(0)
const connected = ref(false)
const failed = ref(false)
const closed = ref(false)
const abort = new AbortController()

let lastBatchAt = Date.now()

const connections = computed(() => Array.from(connMap.value.values()))

const formatBytes = (n: bigint | number | undefined) => {
  const v = Number(n ?? 0)
  if (!v || v < 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let x = v
  while (x >= 1024 && i < units.length - 1) {
    x /= 1024
    i++
  }
  return `${x.toFixed(x >= 100 || i === 0 ? 0 : 1)} ${units[i]}`
}

const formatSpeed = (n: number) => `${formatBytes(n)}/s`

const formatTime = (createdAt: bigint | undefined) => {
  if (createdAt === undefined) return '—'
  const sec = Math.max(0, (Date.now() - Number(createdAt)) / 1000)
  if (sec < 60) return `${Math.floor(sec)}s`
  if (sec < 3600) return `${Math.floor(sec / 60)}m ${Math.floor(sec % 60)}s`
  return `${Math.floor(sec / 3600)}h ${Math.floor((sec % 3600) / 60)}m`
}

const targetOf = (c: Connection) => {
  const host = c.domain || c.destination
  return host || c.id.slice(0, 8)
}

const outboundOf = (c: Connection) =>
  c.outbound || c.fromOutbound || c.chainList?.[0] || '—'

const handleBatch = (ev: ConnectionEvents) => {
  if (closed.value) return
  if (ev.reset) connMap.value = new Map()
  const now = Date.now()
  const dt = Math.max(0.3, (now - lastBatchAt) / 1000)
  lastBatchAt = now
  let up = 0
  let down = 0
  for (const e of ev.events) {
    switch (e.type) {
      case ConnectionEventType.CONNECTION_EVENT_NEW:
        if (e.connection) connMap.value.set(e.id, e.connection)
        break
      case ConnectionEventType.CONNECTION_EVENT_UPDATE:
        up += Number(e.uplinkDelta)
        down += Number(e.downlinkDelta)
        if (e.connection) connMap.value.set(e.id, e.connection)
        break
      case ConnectionEventType.CONNECTION_EVENT_CLOSED:
        connMap.value.delete(e.id)
        break
    }
  }
  upSpeed.value = up / dt
  downSpeed.value = down / dt
  connected.value = true
}

const run = async () => {
  try {
    for await (const ev of subscribeConnections(abort.signal)) {
      handleBatch(ev)
    }
  } catch {
    connected.value = false
    failed.value = true
  }
}

const closeOne = async (id: string) => {
  try {
    await closeConnection(id)
  } catch (e) {
    ElMessage.error((e as Error).message || '断开失败')
  }
}

const closeAll = async () => {
  try {
    await ElMessageBox.confirm(`确认断开全部 ${connections.value.length} 条连接？`, '断开全部', {
      type: 'warning'
    })
    await closeAllConnections()
  } catch {
    // 取消
  }
}

onMounted(() => {
  void run()
})
onBeforeUnmount(() => {
  closed.value = true
  abort.abort()
})
</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- 统计条 -->
    <div class="flex flex-wrap items-center gap-4 rounded-lg border border-[var(--el-border-color)] bg-[var(--el-bg-color)] px-4 py-3">
      <div class="flex items-baseline gap-1.5">
        <span class="text-xs text-[var(--el-text-color-secondary)]">上传</span>
        <span class="text-sm font-semibold text-green-600 dark:text-green-400">{{ formatSpeed(upSpeed) }}</span>
      </div>
      <div class="flex items-baseline gap-1.5">
        <span class="text-xs text-[var(--el-text-color-secondary)]">下载</span>
        <span class="text-sm font-semibold text-blue-600 dark:text-blue-400">{{ formatSpeed(downSpeed) }}</span>
      </div>
      <el-divider direction="vertical" />
      <span class="text-xs text-[var(--el-text-color-secondary)]">连接数</span>
      <span class="text-sm font-semibold">{{ connections.length }}</span>
      <span class="ml-auto flex items-center gap-2">
        <span class="flex items-center gap-1 text-xs" :class="connected ? 'text-[var(--el-text-color-secondary)]' : 'text-red-500'">
          <span class="inline-block h-2 w-2 animate-pulse rounded-full" :class="connected ? 'bg-green-500' : (failed ? 'bg-red-500' : 'bg-yellow-500')" />
          {{ connected ? '实时推送中' : (failed ? 'service API 不可用' : '连接中…') }}
        </span>
        <el-button size="small" :icon="Close" :disabled="!connections.length" @click="closeAll">断开全部</el-button>
      </span>
    </div>

    <!-- 连接表 -->
    <div class="rounded-lg border border-[var(--el-border-color)] bg-[var(--el-bg-color)]">
      <el-table :data="connections" size="small" height="calc(100vh - 220px)" style="width: 100%">
        <el-table-column label="目标" min-width="220">
          <template #default="{ row }">
            <div class="flex flex-col">
              <span class="text-[13px] font-medium">{{ targetOf(row) }}</span>
              <span class="text-xs text-[var(--el-text-color-secondary)]">
                {{ row.source }} → {{ row.destination }}
                <template v-if="row.processInfo?.processPath"> · {{ row.processInfo.processPath.split(/[\\/]/).pop() }}</template>
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="network" label="网络" width="80" />
        <el-table-column label="出站" min-width="110">
          <template #default="{ row }">
            <span class="text-xs">{{ outboundOf(row) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="规则" min-width="110">
          <template #default="{ row }">
            <span class="text-xs">{{ row.rule || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="上传" width="100" align="right">
          <template #default="{ row }">
            <span class="text-xs text-green-600 dark:text-green-400">{{ formatBytes(row.uplinkTotal) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="下载" width="100" align="right">
          <template #default="{ row }">
            <span class="text-xs text-blue-600 dark:text-blue-400">{{ formatBytes(row.downlinkTotal) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="时长" width="90" align="right">
          <template #default="{ row }">
            <span class="text-xs">{{ formatTime(row.createdAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="" width="60" align="center">
          <template #default="{ row }">
            <el-button size="small" text :icon="Delete" @click="closeOne(row.id)" />
          </template>
        </el-table-column>
        <template #empty>
          <div class="flex flex-col items-center gap-2 py-10">
            <span class="text-sm text-[var(--el-text-color-secondary)]">暂无活动连接</span>
          </div>
        </template>
      </el-table>
    </div>
  </div>
</template>
