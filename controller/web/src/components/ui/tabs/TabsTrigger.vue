<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { TabsTrigger, type TabsTriggerProps, useForwardProps } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<TabsTriggerProps & { class?: HTMLAttributes['class'] }>()

const forwarded = useForwardProps(props)
</script>

<template>
  <TabsTrigger
    v-bind="forwarded"
    :class="cn(
      'relative z-1 inline-flex items-center justify-center gap-1.5 rounded-[0.5rem] px-3.5 py-1 text-sm font-medium whitespace-nowrap transition-colors duration-150 outline-none select-none disabled:pointer-events-none disabled:opacity-50',
      'text-base-content/60 hover:text-base-content/85 data-[state=active]:text-base-content data-[state=active]:font-semibold',
      props.class
    )"
  >
    <slot />
  </TabsTrigger>
</template>
