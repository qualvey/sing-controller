<script setup lang="ts">
// 原生 select（daisyUI 样式）：点击显示/选中隐藏由浏览器原生保证，零依赖零坑
// 替代 reka-ui Select（该组件在当前 vue 版本下存在多项交互问题）
import type { HTMLAttributes } from 'vue'

export type SelectOption = string | { value: string | number; label: string }

const model = defineModel<string | number | null>()
defineProps<{
  options?: readonly SelectOption[]
  placeholder?: string
  disabled?: boolean
  class?: HTMLAttributes['class']
}>()
</script>

<template>
  <select
    v-model="model"
    v-bind="$attrs"
    :disabled="disabled"
    :class="['select select-bordered select-sm w-full min-w-0', $attrs.class]"
  >
    <option v-if="placeholder" value="" disabled>{{ placeholder }}</option>
    <option
      v-for="o in options"
      :key="typeof o === 'string' ? o : o.value"
      :value="typeof o === 'string' ? o : o.value"
    >
      {{ typeof o === 'string' ? o : o.label }}
    </option>
  </select>
</template>
