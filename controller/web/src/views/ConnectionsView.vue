<script setup lang="ts">
import { computed, ref } from 'vue'
import { showToast } from '@/helper/toast'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { PauseIcon, PlayIcon, TrashIcon, Search, FileTextIcon, XIcon } from 'lucide-vue-next'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
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

// 表头点击切换排序
const sortableCols = ['target', 'network', 'outbound', 'up', 'down', 'time'] as const
const toggleSort = (key: (typeof sortableCols)[number]) => {
  if (sortKey.value === key) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortKey.value = key
    sortDir.value = 'desc'
  }
}
const sortArrow = (key: string) => (sortKey.value === key ? (sortDir.value === 'asc' ? ' ↑' : ' ↓') : '')

const closeOne = async (id: string) => {
  try {
    await runtime.closeOne(id)
  } catch (e) {
    showToast((e as Error).message || '断开失败', 'error')
  }
}

const closeAll = async () => {
  const { confirmed } = await showConfirmDialog({
    title: '断开全部',
    message: `确认断开全部 ${runtime.connections.length} 条连接？`,
    confirmButtonClass: 'btn-error'
  })
  if (!confirmed) return
  try {
    await runtime.closeAll()
  } catch (e) {
    showToast((e as Error).message || '断开失败', 'error')
  }
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- 统计条 -->
    <div class="flex flex-wrap items-center gap-4 rounded-lg border border-[#e4e7ed] bg-white px-4 py-3 dark:border-[#303030] dark:bg-[#1d1e1f]">
      <div class="flex items-baseline gap-1.5">
        <span class="text-xs text-[#606266] dark:text-[#a6b0bf]">上传</span>
        <span class="text-sm font-semibold tabular-nums text-green-600 dark:text-green-400">{{ formatSpeed(runtime.upSpeed) }}</span>
      </div>
      <div class="flex items-baseline gap-1.5">
        <span class="text-xs text-[#606266] dark:text-[#a6b0bf]">下载</span>
        <span class="text-sm font-semibold tabular-nums text-blue-600 dark:text-blue-400">{{ formatSpeed(runtime.downSpeed) }}</span>
      </div>
      <span class="h-3.5 w-px bg-[#e4e7ed] dark:bg-[#303030]" />
      <span class="text-xs text-[#606266] dark:text-[#a6b0bf]">连接数</span>
      <span class="text-sm font-semibold tabular-nums">{{ runtime.connections.length }}</span>

      <label class="relative ml-2">
        <Search class="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-[#909399]" />
        <input
          v-model="search"
          type="text"
          class="input input-bordered input-sm w-60 pl-9"
          placeholder="搜索 目标/IP/出站/规则"
        />
      </label>

      <span class="ml-auto flex items-center gap-2">
        <span class="flex items-center gap-1 text-xs" :class="runtime.connected ? 'text-[#606266] dark:text-[#a6b0bf]' : 'text-red-500'">
          <span class="inline-block h-2 w-2 animate-pulse rounded-full" :class="runtime.connected ? 'bg-green-500' : 'bg-red-500'" />
          {{ runtime.connected ? '实时推送中' : 'service API 不可用' }}
        </span>
        <button class="btn btn-ghost btn-sm" @click="runtime.togglePause()">
          <PauseIcon v-if="!runtime.paused" class="h-4 w-4" />
          <PlayIcon v-else class="h-4 w-4" />
          {{ runtime.paused ? '恢复' : '暂停' }}
        </button>
        <button class="btn btn-ghost btn-sm" :disabled="!runtime.connections.length" @click="closeAll">
          <XIcon class="h-4 w-4" />
          断开全部
        </button>
      </span>
    </div>

    <!-- 连接表 -->
    <div class="overflow-hidden rounded-lg border border-[#e4e7ed] bg-white dark:border-[#303030] dark:bg-[#1d1e1f]">
      <div class="max-h-[calc(100vh-220px)] overflow-auto">
        <table class="table table-sm w-full">
          <thead class="sticky top-0 z-10 bg-[#f5f7fa] dark:bg-[#141414]">
            <tr>
              <th class="cursor-pointer select-none" @click="toggleSort('target')">目标{{ sortArrow('target') }}</th>
              <th class="w-20 cursor-pointer select-none" @click="toggleSort('network')">网络{{ sortArrow('network') }}</th>
              <th class="w-[110px] cursor-pointer select-none" @click="toggleSort('outbound')">出站{{ sortArrow('outbound') }}</th>
              <th class="w-[110px]">规则</th>
              <th class="w-24 cursor-pointer select-none text-right" @click="toggleSort('up')">上传{{ sortArrow('up') }}</th>
              <th class="w-24 cursor-pointer select-none text-right" @click="toggleSort('down')">下载{{ sortArrow('down') }}</th>
              <th class="w-20 cursor-pointer select-none text-right" @click="toggleSort('time')">时长{{ sortArrow('time') }}</th>
              <th class="w-14 text-center">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in filtered" :key="c.id" class="cursor-pointer hover:bg-black/5 dark:hover:bg-white/5" @click="detail = c">
              <td>
                <div class="flex flex-col">
                  <span class="text-[13px] font-medium">{{ targetOf(c) }}</span>
                  <span class="text-xs text-[#606266] dark:text-[#a6b0bf]">
                    {{ c.source }} → {{ c.destination }}
                    <template v-if="c.processInfo?.processPath"> · {{ c.processInfo.processPath.split(/[\\/]/).pop() }}</template>
                  </span>
                </div>
              </td>
              <td><span class="text-xs">{{ c.network }}</span></td>
              <td><span class="text-xs">{{ outboundOf(c) }}</span></td>
              <td><span class="text-xs">{{ c.rule || '—' }}</span></td>
              <td class="text-right"><span class="text-xs tabular-nums text-green-600 dark:text-green-400">{{ formatBytes(c.uplinkTotal) }}</span></td>
              <td class="text-right"><span class="text-xs tabular-nums text-blue-600 dark:text-blue-400">{{ formatBytes(c.downlinkTotal) }}</span></td>
              <td class="text-right"><span class="text-xs tabular-nums">{{ formatDuration(durationOf(c)) }}</span></td>
              <td class="text-center">
                <button class="btn btn-ghost btn-xs text-error" title="断开" @click.stop="closeOne(c.id)">
                  <TrashIcon class="h-3.5 w-3.5" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-if="!filtered.length" class="py-10 text-center text-sm text-[#606266] dark:text-[#a6b0bf]">
        {{ search ? '无匹配连接' : '暂无活动连接' }}
      </div>
    </div>

    <!-- 连接详情弹窗 -->
    <DialogWrapper :model-value="detail !== null" title="连接详情" box-class="max-w-[380px]" @update:model-value="(v: boolean | undefined) => { if (!v) detail = null }">
      <template v-if="detail">
        <div class="mb-4 flex items-center gap-2">
          <FileTextIcon class="h-4 w-4 text-[#409eff]" />
          <span class="text-sm font-semibold">{{ targetOf(detail) }}</span>
        </div>
        <dl class="flex flex-col gap-2 text-sm">
          <div class="flex justify-between gap-3"><dt class="shrink-0 text-[#909399]">ID</dt><dd class="break-all text-right font-mono text-xs">{{ detail.id }}</dd></div>
          <div class="flex justify-between gap-3"><dt class="shrink-0 text-[#909399]">网络</dt><dd>{{ detail.network }}</dd></div>
          <div class="flex justify-between gap-3"><dt class="shrink-0 text-[#909399]">入站</dt><dd>{{ detail.inbound }} ({{ detail.inboundType }})</dd></div>
          <div class="flex justify-between gap-3"><dt class="shrink-0 text-[#909399]">来源</dt><dd class="break-all text-right">{{ detail.source }}</dd></div>
          <div class="flex justify-between gap-3"><dt class="shrink-0 text-[#909399]">目标</dt><dd class="break-all text-right">{{ detail.destination }}</dd></div>
          <div class="flex justify-between gap-3"><dt class="shrink-0 text-[#909399]">域名</dt><dd class="break-all text-right">{{ detail.domain || '—' }}</dd></div>
          <div class="flex justify-between gap-3"><dt class="shrink-0 text-[#909399]">协议</dt><dd>{{ detail.protocol || '—' }}</dd></div>
          <div class="flex justify-between gap-3"><dt class="shrink-0 text-[#909399]">用户</dt><dd>{{ detail.user || '—' }}</dd></div>
          <div class="flex justify-between gap-3"><dt class="shrink-0 text-[#909399]">出站</dt><dd class="break-all text-right">{{ outboundOf(detail) }}</dd></div>
          <div class="flex justify-between gap-3"><dt class="shrink-0 text-[#909399]">出站类型</dt><dd>{{ detail.outboundType || '—' }}</dd></div>
          <div class="flex justify-between gap-3"><dt class="shrink-0 text-[#909399]">规则</dt><dd class="break-all text-right">{{ detail.rule || '—' }}</dd></div>
          <div class="flex justify-between gap-3">
            <dt class="shrink-0 text-[#909399]">链</dt>
            <dd class="flex flex-col gap-0.5 text-right">
              <span v-for="(ch, i) in detail.chainList || []" :key="i" class="text-xs">{{ ch }}</span>
              <span v-if="!detail.chainList?.length">—</span>
            </dd>
          </div>
          <div class="flex justify-between gap-3">
            <dt class="shrink-0 text-[#909399]">进程</dt>
            <dd v-if="detail.processInfo" class="flex flex-col gap-0.5 text-right text-xs">
              <span>{{ detail.processInfo.processPath || '—' }}</span>
              <span class="text-[#606266] dark:text-[#a6b0bf]">PID {{ detail.processInfo.processId }} · {{ detail.processInfo.userName }}</span>
            </dd>
            <dd v-else>—</dd>
          </div>
          <div class="flex justify-between gap-3"><dt class="shrink-0 text-[#909399]">上传</dt><dd>{{ formatBytes(detail.uplinkTotal) }}</dd></div>
          <div class="flex justify-between gap-3"><dt class="shrink-0 text-[#909399]">下载</dt><dd>{{ formatBytes(detail.downlinkTotal) }}</dd></div>
          <div class="flex justify-between gap-3"><dt class="shrink-0 text-[#909399]">时长</dt><dd>{{ formatDuration(durationOf(detail)) }}</dd></div>
        </dl>
        <button class="btn btn-error btn-sm mt-4 w-full" @click="closeOne(detail.id); detail = null">
          <TrashIcon class="h-4 w-4" />
          断开此连接
        </button>
      </template>
    </DialogWrapper>
  </div>
</template>
