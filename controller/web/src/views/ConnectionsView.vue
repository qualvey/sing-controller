<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Close, VideoPause, VideoPlay, Search, Document } from '@element-plus/icons-vue'
import { useRuntimeStore } from '../stores/runtime'
import type { Connection } from '@/gen/daemon/started_service_pb'

const runtime = useRuntimeStore()

const search = ref('')
const sortKey = ref<'target' | 'network' | 'outbound' | 'up' | 'down' | 'time'>('time')
const sortDir = ref<'asc' | 'desc'>('desc')
const detail = ref<Connection | null>(null)

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

const targetOf = (c: Connection) => c.domain || c.destination || c.id.slice(0, 8)
const outboundOf = (c: Connection) => c.outbound || c.fromOutbound || c.chainList?.[0] || '—'
const durationOf = (c: Connection) => (c.createdAt === undefined ? 0 : (Date.now() - Number(c.createdAt)) / 1000)

const formatDuration = (sec: number) => {
  if (sec < 60) return `${Math.floor(sec)}s`
  if (sec < 3600) return `${Math.floor(sec / 60)}m ${Math.floor(sec % 60)}s`
  return `${Math.floor(sec / 3600)}h ${Math.floor((sec % 3600) / 60)}m`
}

const filtered = computed(() => {
  let list = runtime.connections
  const q = search.value.trim().toLowerCase()
  if (q) {
    list = list.filter(
      (c) =>
        targetOf(c).toLowerCase().includes(q) ||
        c.destination?.toLowerCase().includes(q) ||
        outboundOf(c).toLowerCase().includes(q) ||
        (c.rule || '').toLowerCase().includes(q) ||
        c.source?.toLowerCase().includes(q)
    )
  }
  const dir = sortDir.value === 'asc' ? 1 : -1
  return [...list].sort((a, b) => {
    switch (sortKey.value) {
      case 'target':
        return targetOf(a).localeCompare(targetOf(b)) * dir
      case 'network':
        return (a.network || '').localeCompare(b.network || '') * dir
      case 'outbound':
        return outboundOf(a).localeCompare(outboundOf(b)) * dir
      case 'up':
        return (Number(a.uplinkTotal) - Number(b.uplinkTotal)) * dir
      case 'down':
        return (Number(a.downlinkTotal) - Number(b.downlinkTotal)) * dir
      case 'time':
        return (durationOf(a) - durationOf(b)) * dir
    }
    return 0
  })
})

const closeOne = async (id: string) => {
  try {
    await runtime.closeOne(id)
  } catch (e) {
    ElMessage.error((e as Error).message || '断开失败')
  }
}

const closeAll = async () => {
  try {
    await ElMessageBox.confirm(`确认断开全部 ${runtime.connections.length} 条连接？`, '断开全部', {
      type: 'warning'
    })
    await runtime.closeAll()
  } catch {
    // 取消
  }
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- 统计条 -->
    <div class="flex flex-wrap items-center gap-4 rounded-lg border border-[var(--el-border-color)] bg-[var(--el-bg-color)] px-4 py-3">
      <div class="flex items-baseline gap-1.5">
        <span class="text-xs text-[var(--el-text-color-secondary)]">上传</span>
        <span class="text-sm font-semibold tabular-nums text-green-600 dark:text-green-400">{{ formatSpeed(runtime.upSpeed) }}</span>
      </div>
      <div class="flex items-baseline gap-1.5">
        <span class="text-xs text-[var(--el-text-color-secondary)]">下载</span>
        <span class="text-sm font-semibold tabular-nums text-blue-600 dark:text-blue-400">{{ formatSpeed(runtime.downSpeed) }}</span>
      </div>
      <el-divider direction="vertical" />
      <span class="text-xs text-[var(--el-text-color-secondary)]">连接数</span>
      <span class="text-sm font-semibold tabular-nums">{{ runtime.connections.length }}</span>

      <el-input
        v-model="search"
        size="small"
        placeholder="搜索 目标/IP/出站/规则"
        :prefix-icon="Search"
        clearable
        style="width: 240px"
        class="ml-2"
      />

      <span class="ml-auto flex items-center gap-2">
        <span class="flex items-center gap-1 text-xs" :class="runtime.connected ? 'text-[var(--el-text-color-secondary)]' : 'text-red-500'">
          <span class="inline-block h-2 w-2 animate-pulse rounded-full" :class="runtime.connected ? 'bg-green-500' : 'bg-red-500'" />
          {{ runtime.connected ? '实时推送中' : 'service API 不可用' }}
        </span>
        <el-button size="small" :icon="runtime.paused ? VideoPlay : VideoPause" @click="runtime.togglePause()">
          {{ runtime.paused ? '恢复' : '暂停' }}
        </el-button>
        <el-button size="small" :icon="Close" :disabled="!runtime.connections.length" @click="closeAll">断开全部</el-button>
      </span>
    </div>

    <!-- 连接表 -->
    <div class="rounded-lg border border-[var(--el-border-color)] bg-[var(--el-bg-color)]">
      <el-table :data="filtered" size="small" height="calc(100vh - 220px)" style="width: 100%" @row-click="(row: Connection) => (detail = row)">
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
        <el-table-column label="网络" width="80" sortable :sort-by="(r: Connection) => r.network" :sort-orders="['ascending', 'descending']">
          <template #default="{ row }">
            <span class="text-xs">{{ row.network }}</span>
          </template>
        </el-table-column>
        <el-table-column label="出站" min-width="110" sortable :sort-by="(r: Connection) => outboundOf(r)">
          <template #default="{ row }">
            <span class="text-xs">{{ outboundOf(row) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="规则" min-width="110">
          <template #default="{ row }">
            <span class="text-xs">{{ row.rule || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="上传" width="100" align="right" sortable :sort-by="(r: Connection) => Number(r.uplinkTotal)">
          <template #default="{ row }">
            <span class="text-xs tabular-nums text-green-600 dark:text-green-400">{{ formatBytes(row.uplinkTotal) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="下载" width="100" align="right" sortable :sort-by="(r: Connection) => Number(r.downlinkTotal)">
          <template #default="{ row }">
            <span class="text-xs tabular-nums text-blue-600 dark:text-blue-400">{{ formatBytes(row.downlinkTotal) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="时长" width="90" align="right" sortable :sort-by="(r: Connection) => durationOf(r)">
          <template #default="{ row }">
            <span class="text-xs tabular-nums">{{ formatDuration(durationOf(row)) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="" width="60" align="center">
          <template #default="{ row }">
            <el-button size="small" text :icon="Delete" @click.stop="closeOne(row.id)" />
          </template>
        </el-table-column>
        <template #empty>
          <div class="flex flex-col items-center gap-2 py-10">
            <span class="text-sm text-[var(--el-text-color-secondary)]">{{ search ? '无匹配连接' : '暂无活动连接' }}</span>
          </div>
        </template>
      </el-table>
    </div>

    <!-- 连接详情抽屉 -->
    <el-drawer :model-value="detail !== null" size="380px" :with-header="false" @update:model-value="(v: boolean) => { if (!v) detail = null }">
      <template v-if="detail">
        <div class="mb-4 flex items-center gap-2">
          <el-icon :size="16" class="text-[var(--el-color-primary)]"><Document /></el-icon>
          <span class="text-sm font-semibold">{{ targetOf(detail) }}</span>
        </div>
        <el-descriptions :column="1" size="small" border>
          <el-descriptions-item label="ID">{{ detail.id }}</el-descriptions-item>
          <el-descriptions-item label="网络">{{ detail.network }}</el-descriptions-item>
          <el-descriptions-item label="入站">{{ detail.inbound }} ({{ detail.inboundType }})</el-descriptions-item>
          <el-descriptions-item label="来源">{{ detail.source }}</el-descriptions-item>
          <el-descriptions-item label="目标">{{ detail.destination }}</el-descriptions-item>
          <el-descriptions-item label="域名">{{ detail.domain || '—' }}</el-descriptions-item>
          <el-descriptions-item label="协议">{{ detail.protocol || '—' }}</el-descriptions-item>
          <el-descriptions-item label="用户">{{ detail.user || '—' }}</el-descriptions-item>
          <el-descriptions-item label="出站">{{ outboundOf(detail) }}</el-descriptions-item>
          <el-descriptions-item label="出站类型">{{ detail.outboundType || '—' }}</el-descriptions-item>
          <el-descriptions-item label="规则">{{ detail.rule || '—' }}</el-descriptions-item>
          <el-descriptions-item label="链">
            <div class="flex flex-col gap-0.5">
              <span v-for="(ch, i) in detail.chainList || []" :key="i" class="text-xs">{{ ch }}</span>
              <span v-if="!detail.chainList?.length">—</span>
            </div>
          </el-descriptions-item>
          <el-descriptions-item label="进程">
            <template v-if="detail.processInfo">
              <div class="flex flex-col gap-0.5 text-xs">
                <span>{{ detail.processInfo.processPath || '—' }}</span>
                <span class="text-[var(--el-text-color-secondary)]">PID {{ detail.processInfo.processId }} · {{ detail.processInfo.userName }}</span>
              </div>
            </template>
            <span v-else>—</span>
          </el-descriptions-item>
          <el-descriptions-item label="上传">{{ formatBytes(detail.uplinkTotal) }}</el-descriptions-item>
          <el-descriptions-item label="下载">{{ formatBytes(detail.downlinkTotal) }}</el-descriptions-item>
          <el-descriptions-item label="时长">{{ formatDuration(durationOf(detail)) }}</el-descriptions-item>
        </el-descriptions>
        <el-button class="mt-4 w-full" type="danger" plain :icon="Delete" @click="closeOne(detail.id); detail = null">
          断开此连接
        </el-button>
      </template>
    </el-drawer>
  </div>
</template>
