<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { TabsContent, type TabsContentProps, useForwardProps } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<TabsContentProps & { class?: HTMLAttributes['class'] }>()

const forwarded = useForwardProps(props)
</script>

<template>
  <TabsContent
    v-bind="forwarded"
    :class="cn('tab-content-panel mt-2 min-h-[280px] outline-none focus-visible:ring-ring/50 focus-visible:ring-[3px]', props.class)"
  >
    <slot />
  </TabsContent>
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
