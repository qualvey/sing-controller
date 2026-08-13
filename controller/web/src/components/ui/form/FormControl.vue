<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

// 普通 div 包裹（不能用 reka-ui Slot：它不渲染包裹元素、class 合并到子元素，
// 会把表单内容展开成多个 grid item 导致排版错乱）
const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div :class="cn('flex w-full flex-col gap-2', props.class)">
    <slot />
  </div>
</template>
