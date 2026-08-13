<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import {
  SelectContent,
  type SelectContentEmits,
  type SelectContentProps,
  useForwardPropsEmits
} from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<SelectContentProps & { class?: HTMLAttributes['class'] }>()
const emits = defineEmits<SelectContentEmits>()

const forwarded = useForwardPropsEmits(props, emits)
</script>

<template>
  <!-- force-mount：保持 SelectItem 注册（SelectValue 显示选中文本依赖 optionsSet），
      关闭态用 data-state 隐藏 -->
  <SelectContent
    v-bind="forwarded"
    force-mount
    :class="cn(
      'bg-popover text-popover-foreground data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 relative z-50 max-h-(--radix-select-content-available-height) min-w-[8rem] origin-(--radix-select-content-transform-origin) overflow-x-hidden overflow-y-auto rounded-md border shadow-md data-[state=closed]:hidden',
      props.class
    )"
  >
    <slot />
  </SelectContent>
</template>
