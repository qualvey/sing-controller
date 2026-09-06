<template>
  <div class="flex flex-col gap-1.5 w-full">
    <!-- 日志操作工具条 -->
    <div
      v-if="logs.length > 0"
      class="flex items-center justify-between px-1 text-[11px] text-base-content/60"
    >
      <div class="flex items-center gap-2">
        <span>共 {{ logs.length }} 条日志</span>
        <span
          v-if="fatalCount > 0"
          class="inline-flex items-center gap-1 rounded bg-error/20 px-1.5 py-0.2 text-[10px] font-bold text-error animate-pulse"
        >
          <ExclamationTriangleIcon class="h-3 w-3" />
          {{ fatalCount }} 处致命错误 (FATAL)
        </span>
      </div>
      <button
        type="button"
        class="btn btn-ghost btn-xs h-5 min-h-0 gap-1 px-1.5 text-[11px] hover:text-base-content"
        @click="copyLogs"
      >
        <ClipboardDocumentCheckIcon
          v-if="copied"
          class="h-3.5 w-3.5 text-success"
        />
        <ClipboardDocumentIcon
          v-else
          class="h-3.5 w-3.5"
        />
        {{ copied ? '已复制' : '复制全部日志' }}
      </button>
    </div>

    <!-- 日志显示主体区域 -->
    <div
      ref="containerRef"
      class="bg-base-300 text-base-content/90 overflow-y-auto rounded-lg p-2.5 font-mono text-[11px] leading-relaxed select-text"
      :style="{ maxHeight: maxHeight || '13rem' }"
    >
      <div
        v-if="logs.length === 0"
        class="py-3 text-center text-base-content/40"
      >
        暂无日志记录
      </div>
      <div
        v-for="(line, idx) in processedLogs"
        :key="idx"
        class="log-row rounded-sm px-1.5 py-0.5 transition-colors"
        :class="[
          line.isFatal
            ? 'border-l-4 border-error bg-error/20 text-error font-semibold my-1 shadow-sm'
            : line.isError
              ? 'border-l-2 border-error/70 bg-error/10 text-error/95 my-0.5'
              : line.isWarning
                ? 'border-l-2 border-warning/60 text-warning my-0.5'
                : 'text-base-content/85 hover:bg-base-200/40',
        ]"
      >
        <!-- 时间戳（仅在有效时展示，杜绝空 [] ） -->
        <span
          v-if="line.timeStr"
          class="opacity-40 mr-1.5 shrink-0"
        >[{{ line.timeStr }}]</span>

        <!-- 日志来源（如 stdout/stderr/systemd，有效时展示） -->
        <span
          v-if="line.sourceStr"
          class="opacity-60 mr-1.5 shrink-0"
        >[{{ line.sourceStr }}]</span>

        <!-- 显眼的 FATAL 提示角标 -->
        <span
          v-if="line.isFatal"
          class="inline-flex items-center gap-0.5 rounded bg-error px-1 py-0.2 mr-1.5 text-[10px] text-white font-bold tracking-wider"
        >
          <ExclamationTriangleIcon class="h-3 w-3 shrink-0" />
          FATAL
        </span>

        <!-- ANSI 丰富色彩切片渲染 -->
        <span class="break-all whitespace-pre-wrap">
          <span
            v-for="(seg, sIdx) in line.segments"
            :key="sIdx"
            :style="seg.style"
          >{{ seg.text }}</span>
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CoreLogLine } from '@/api'
import { parseAnsiText } from '@/helper/ansi'
import { showToast } from '@/helper/toast'
import {
  ClipboardDocumentCheckIcon,
  ClipboardDocumentIcon,
  ExclamationTriangleIcon,
} from '@heroicons/vue/24/outline'
import dayjs from 'dayjs'
import { computed, nextTick, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    logs: CoreLogLine[]
    maxHeight?: string
  }>(),
  {
    maxHeight: '13rem',
  }
)

const containerRef = ref<HTMLElement>()
const copied = ref(false)

const formatTime = (ts?: string) => {
  if (!ts || typeof ts !== 'string') return ''
  const d = dayjs(ts)
  return d.isValid() ? d.format('HH:mm:ss.SSS') : ''
}

const isFatalMessage = (msg: string) => {
  if (!msg) return false
  return /fatal|panic/i.test(msg)
}

const isErrorMessage = (msg: string, source?: string) => {
  if (!msg) return false
  if (source === 'stderr') return true
  return /error|failed|failure|exit-code|exited,\s*status=[^0]/i.test(msg)
}

const isWarningMessage = (msg: string, source?: string) => {
  if (!msg) return false
  if (source === 'system') return true
  return /warn(?:ing)?/i.test(msg)
}

const processedLogs = computed(() => {
  return props.logs.map((item) => {
    const rawMsg = item.message || (item as any).raw || ''
    const isFatal = isFatalMessage(rawMsg)
    const isError = !isFatal && isErrorMessage(rawMsg, item.source)
    const isWarning = !isFatal && !isError && isWarningMessage(rawMsg, item.source)

    return {
      timeStr: formatTime(item.timestamp),
      sourceStr: item.source ? String(item.source).trim() : '',
      isFatal,
      isError,
      isWarning,
      segments: parseAnsiText(rawMsg),
      raw: rawMsg,
    }
  })
})

const fatalCount = computed(() => {
  return processedLogs.value.filter((l) => l.isFatal).length
})

const copyLogs = async () => {
  if (!props.logs.length) return
  const fullText = props.logs
    .map((l) => {
      const parts: string[] = []
      if (l.timestamp) parts.push(`[${l.timestamp}]`)
      if (l.source) parts.push(`[${l.source}]`)
      parts.push(l.message || (l as any).raw || '')
      return parts.join(' ')
    })
    .join('\n')

  try {
    await navigator.clipboard.writeText(fullText)
    copied.value = true
    showToast('已复制日志到剪贴板', 'success')
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch {
    showToast('复制日志失败', 'error')
  }
}

watch(
  () => props.logs.length,
  () => {
    nextTick(() => {
      if (containerRef.value) {
        // 如果有致命错误行，优先滚动或者滚到底部
        containerRef.value.scrollTop = containerRef.value.scrollHeight
      }
    })
  }
)
</script>

<style scoped>
.log-row {
  word-break: break-all;
}
</style>
