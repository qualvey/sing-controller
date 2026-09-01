<template>
  <div class="border-base-content/10 bg-base-100/60 backdrop-blur-xs flex flex-col gap-3 rounded-2xl border p-3.5 sm:p-4 shadow-xs">
    <!-- 头部工具栏 -->
    <div
      v-if="!hideHeader"
      class="flex flex-wrap items-center justify-between gap-2 border-b border-base-content/5 pb-3"
    >
      <div class="flex items-center gap-2">
        <slot name="icon" />
        <span class="text-sm sm:text-base font-semibold text-base-content/90">{{ title }}</span>
        <span
          v-if="count !== undefined"
          class="badge badge-sm badge-neutral badge-outline font-mono"
        >
          {{ count }}
        </span>
      </div>

      <div class="flex items-center gap-2">
        <slot name="actions" />
      </div>
    </div>

    <!-- 卡片网格列表 -->
    <div
      v-if="totalCount > 0"
      class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3"
    >
      <slot />
    </div>

    <!-- 空状态 -->
    <div
      v-else
      class="text-base-content/50 border-base-content/5 bg-base-200/30 flex flex-col items-center justify-center rounded-xl border py-14 text-center text-sm"
    >
      <slot name="empty">
        <p class="mb-3 text-sm text-base-content/60">{{ emptyText || '暂无数据' }}</p>
        <slot name="empty-actions" />
      </slot>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title?: string
  count?: number
  totalCount: number
  emptyText?: string
  hideHeader?: boolean
}>()
</script>
