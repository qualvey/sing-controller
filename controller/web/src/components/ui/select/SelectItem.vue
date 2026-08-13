<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { SelectItem, SelectItemIndicator, SelectItemText, type SelectItemProps, useForwardProps } from 'reka-ui'
import { Check } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

const props = defineProps<SelectItemProps & { class?: HTMLAttributes['class'] }>()

const forwarded = useForwardProps(props)
</script>

<template>
  <SelectItem
    v-bind="forwarded"
    :class="cn(
      'focus:bg-accent focus:text-accent-foreground relative flex w-full cursor-default items-center gap-2 rounded-sm py-1.5 pr-8 pl-2 text-sm outline-hidden select-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
      props.class
    )"
  >
    <span class="absolute right-2 flex size-3.5 items-center justify-center">
      <SelectItemIndicator>
        <Check class="size-4" />
      </SelectItemIndicator>
    </span>
    <!-- SelectItemText 必需：负责把选项注册到 optionsSet（SelectValue 显示选中文本依赖它） -->
    <SelectItemText><slot /></SelectItemText>
  </SelectItem>
</template>
