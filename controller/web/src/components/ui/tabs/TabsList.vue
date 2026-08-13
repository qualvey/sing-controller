<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { TabsList, type TabsListProps, useForwardProps } from 'radix-vue'
import { cn } from '@/lib/utils'

const props = defineProps<TabsListProps & { class?: HTMLAttributes['class'] }>()

const forwarded = useForwardProps(props)
</script>

<template>
  <TabsList
    v-bind="forwarded"
    :class="cn(
      'bg-muted text-muted-foreground inline-flex h-9 w-max items-center justify-center rounded-lg p-1',
      props.class
    )"
  >
    <slot />
  </TabsList>
</template>
