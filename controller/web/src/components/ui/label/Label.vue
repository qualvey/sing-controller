<script setup lang="ts">
// 原生 label（reka-ui 的 Label 组件在当前 vue 版本下渲染异常：内容丢失/宽度错乱）
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <label :class="cn('flex items-center gap-2 text-sm leading-none font-medium select-none', props.class)">
    <slot />
  </label>
</template>
