<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed, useSlots } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
const slots = useSlots()

// 仅当插槽有实际内容（非空文本）时渲染（vee-validate 空错误不显示）
const hasContent = computed(() => {
  const children = slots.default?.() ?? []
  return children.some((c) => {
    const t = (c as { children?: unknown })?.children
    return typeof t === 'string' ? t.trim() !== '' : true
  })
})
</script>

<template>
  <p v-if="hasContent" :class="cn('text-[0.8rem] font-medium text-destructive', props.class)">
    <slot />
  </p>
</template>
