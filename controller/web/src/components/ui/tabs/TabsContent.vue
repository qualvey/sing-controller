<script setup lang="ts">
import { computed, inject, type HTMLAttributes } from 'vue'
import { tabsContextKey } from './tabsContext'
import { cn } from '@/lib/utils'

const props = defineProps<{
  value: string | number
  class?: HTMLAttributes['class']
}>()

const context = inject(tabsContextKey, null)

const isActive = computed(() => context?.modelValue.value === props.value)
</script>

<template>
  <div
    v-show="isActive"
    role="tabpanel"
    :data-state="isActive ? 'active' : 'inactive'"
    :class="cn('tab-content-panel mt-2 min-h-[280px] outline-none focus-visible:ring-ring/50 focus-visible:ring-[3px]', props.class)"
  >
    <slot />
  </div>
</template>

<style scoped>
.tab-content-panel[data-state='active'] {
  animation: tabContentFadeIn 0.2s cubic-bezier(0.32, 0.72, 0, 1) forwards;
}

@keyframes tabContentFadeIn {
  from {
    opacity: 0;
    transform: translateY(3px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .tab-content-panel[data-state='active'] {
    animation: none;
  }
}
</style>
