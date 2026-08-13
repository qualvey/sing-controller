<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'
import { Slot } from 'reka-ui'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <Slot :class="cn('flex w-full flex-col gap-2', props.class)">
    <slot />
  </Slot>
</template>
