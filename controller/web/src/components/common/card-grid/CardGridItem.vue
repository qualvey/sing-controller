<template>
  <div
    class="bg-base-200/70 hover:bg-base-200 border-base-content/5 hover:border-base-content/15 group relative flex flex-col justify-between gap-2.5 rounded-xl border p-3 transition-all duration-200 hover:shadow-sm"
    :class="clickable ? 'cursor-pointer' : ''"
    @click="$emit('click', $event)"
  >
    <!-- 头部区域：左侧标题+副标，右侧徽章 -->
    <div class="flex items-start justify-between gap-2">
      <div class="flex flex-1 min-w-0 items-start gap-2">
        <slot name="icon" />
        <div class="flex flex-col min-w-0 flex-1">
          <div class="flex items-center gap-1.5 min-w-0">
            <slot name="title">
              <span class="truncate font-semibold text-sm text-base-content/95">{{ title }}</span>
            </slot>
          </div>
          <div
            v-if="subtitle || $slots.subtitle"
            class="text-xs text-base-content/55 truncate"
          >
            <slot name="subtitle">
              {{ subtitle }}
            </slot>
          </div>
        </div>
      </div>

      <!-- 右侧徽章/Pill -->
      <div
        v-if="$slots.badge || badge"
        class="shrink-0"
      >
        <slot name="badge">
          <span class="badge badge-sm badge-neutral font-mono text-[11px] font-medium shadow-xs">
            {{ badge }}
          </span>
        </slot>
      </div>
    </div>

    <!-- 中部内容区域（可选） -->
    <div
      v-if="$slots.default"
      class="text-xs text-base-content/75 flex flex-col gap-1 py-0.5"
    >
      <slot />
    </div>

    <!-- 底部区域：左侧标签组，右侧操作工具栏 -->
    <div
      v-if="$slots.footer || $slots.tags || $slots.actions"
      class="flex items-center justify-between gap-2 pt-1 border-t border-base-content/5 mt-auto"
    >
      <div class="flex flex-1 flex-wrap items-center gap-1 min-w-0">
        <slot name="tags" />
        <slot name="footer" />
      </div>

      <div
        v-if="$slots.actions"
        class="flex items-center gap-1 shrink-0"
        @click.stop
      >
        <slot name="actions" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title?: string
  subtitle?: string
  badge?: string | number
  clickable?: boolean
}>()

defineEmits<{
  (e: 'click', event: MouseEvent): void
}>()
</script>
