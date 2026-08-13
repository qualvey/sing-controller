<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'
import Label from '@/components/ui/label/Label.vue'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <Label :class="cn('text-[#606266] dark:text-[#a6b0bf]', props.class)">
    <slot />
  </Label>
</template>
