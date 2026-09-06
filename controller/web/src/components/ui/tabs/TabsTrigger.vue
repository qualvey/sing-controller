<script setup lang="ts">
import { computed, inject, type HTMLAttributes } from 'vue'
import { tabsContextKey } from './tabsContext'
import { cn } from '@/lib/utils'

const props = defineProps<{
  value: string | number
  disabled?: boolean
  class?: HTMLAttributes['class']
}>()

const context = inject(tabsContextKey, null)

const isActive = computed(() => context?.modelValue.value === props.value)

const handleClick = () => {
  if (props.disabled) return
  context?.changeTab(props.value)
}
</script>

<template>
  <button
    type="button"
    role="tab"
    :disabled="props.disabled"
    :data-state="isActive ? 'active' : 'inactive'"
    :aria-selected="isActive"
    :class="cn(
      'relative z-1 inline-flex items-center justify-center gap-1.5 rounded-[0.5rem] px-3.5 py-1 text-sm font-medium whitespace-nowrap transition-colors duration-150 outline-none select-none disabled:pointer-events-none disabled:opacity-50 cursor-pointer',
      'text-base-content/60 hover:text-base-content/85 data-[state=active]:text-base-content data-[state=active]:font-semibold',
      props.class
    )"
    @click="handleClick"
  >
    <slot />
  </button>
</template>
