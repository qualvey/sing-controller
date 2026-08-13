<script setup lang="ts">
// 原生 label（reka-ui 的 Label 组件在当前 vue 版本下渲染异常：内容丢失/宽度错乱）
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <label :class="cn('text-[#606266] dark:text-[#a6b0bf]', props.class)">
    <slot />
  </label>
</template>
