<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { type ButtonVariants, buttonVariants } from '.'
import { cn } from '@/lib/utils'

interface Props {
  as?: string
  variant?: ButtonVariants['variant']
  size?: ButtonVariants['size']
  class?: HTMLAttributes['class']
}

const props = withDefaults(defineProps<Props>(), {
  as: 'button',
  variant: 'default',
  size: 'default'
})
</script>

<template>
  <component :is="as" :class="cn(buttonVariants({ variant, size }), props.class)">
    <slot />
  </component>
</template>
