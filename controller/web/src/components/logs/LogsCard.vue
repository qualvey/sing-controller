<script setup lang="ts">
// 日志卡片（搬运自 zashboard，裁剪依赖）
// 结构：序号 + 级别 + 时间 / payload（ANSI 渲染 + 搜索高亮）
import HighlightText from '@/components/common/HighlightText.vue'
import { LogLevel } from '@/gen/daemon/started_service_pb'
import type { LogEntry } from '@/stores/logs'
import { computed } from 'vue'

const props = defineProps<{
  log: LogEntry
}>()

const seqWithPadding = computed(() => props.log.seq.toString().padStart(3, '0'))

const colorMapForType: Record<number, string> = {
  [LogLevel.PANIC]: 'text-red-500',
  [LogLevel.FATAL]: 'text-red-500',
  [LogLevel.ERROR]: 'text-red-500',
  [LogLevel.WARN]: 'text-yellow-500',
  [LogLevel.INFO]: 'text-blue-500',
  [LogLevel.DEBUG]: 'text-emerald-500',
  [LogLevel.TRACE]: 'text-gray-400'
}

const levelClass = computed(() => colorMapForType[props.log.level] || '')
</script>

<template>
  <div class="flex flex-col gap-1 px-3 py-1.5 text-[12.5px] transition-colors hover:bg-black/5 dark:hover:bg-white/5">
    <div class="flex items-center gap-2">
      <span class="text-xs text-[#909399] tabular-nums dark:text-[#636d83]">{{ seqWithPadding }}</span>
      <span class="text-[11px] tracking-wide uppercase" :class="levelClass">{{ log.levelLabel }}</span>
      <div class="flex-1" />
      <span class="text-xs text-[#909399] tabular-nums dark:text-[#636d83]">{{ log.time }}</span>
    </div>
    <div class="w-full break-words leading-relaxed">
      <HighlightText :text="log.message" :filter="log.filter" ansi />
    </div>
  </div>
</template>
