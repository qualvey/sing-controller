<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <p v-if="!!$slots.default" :class="cn('text-[0.8rem] font-medium text-destructive', props.class)">
    <slot />
  </p>
</template>
