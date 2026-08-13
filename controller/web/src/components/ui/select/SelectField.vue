<script setup lang="ts">
// radix SelectRoot 是多根组件（trigger + teleport 锚点 + 隐藏 select 共 3 节点），
// 直接放进 grid 布局会被展开成 3 个 grid item 导致错位 → 用单根 div 包装
import type { HTMLAttributes } from 'vue'
import { SelectRoot } from 'radix-vue'

const model = defineModel<any>()
const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div :class="['w-full min-w-0', props.class]">
    <SelectRoot v-model="model" v-bind="$attrs">
      <slot />
    </SelectRoot>
  </div>
</template>
