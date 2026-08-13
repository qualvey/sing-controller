<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { CheckboxRoot, CheckboxIndicator, useForwardProps, type CheckboxRootProps } from 'radix-vue'
import { Check } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

const props = defineProps<CheckboxRootProps & { class?: HTMLAttributes['class'] }>()

const forwarded = useForwardProps(props)
</script>

<template>
  <CheckboxRoot
    v-bind="forwarded"
    :class="cn(
      'peer size-4 shrink-0 rounded-sm border border-input shadow-xs transition-shadow focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] outline-none disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:border-primary data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground',
      props.class
    )"
  >
    <CheckboxIndicator class="flex h-full w-full items-center justify-center text-current">
      <Check class="size-3.5" />
    </CheckboxIndicator>
  </CheckboxRoot>
</template>
