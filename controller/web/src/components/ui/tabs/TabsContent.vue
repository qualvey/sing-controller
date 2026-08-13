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
    :class="cn('mt-2 outline-none focus-visible:ring-ring/50 focus-visible:ring-[3px]', props.class)"
  >
    <slot />
  </TabsContent>
</template>
