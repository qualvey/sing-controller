<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { SelectTrigger, type SelectTriggerProps, useForwardProps } from 'radix-vue'
import { ChevronDown } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

const props = defineProps<SelectTriggerProps & { class?: HTMLAttributes['class'] }>()

const forwarded = useForwardProps(props)
</script>

<template>
  <SelectTrigger
    v-bind="forwarded"
    :class="cn(
      'border-input data-[placeholder]:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 flex h-9 w-full items-center justify-between gap-2 rounded-md border bg-transparent px-3 py-2 text-sm whitespace-nowrap shadow-xs transition-[color,box-shadow] outline-none focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50',
      props.class
    )"
  >
    <slot />
    <ChevronDown class="size-4 opacity-50" />
  </SelectTrigger>
</template>
