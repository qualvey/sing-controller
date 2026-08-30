<script setup lang="ts">
import { computed, type HTMLAttributes } from 'vue'
import { CheckboxRoot, CheckboxIndicator, type CheckboxRootProps } from 'reka-ui'
import { Check } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

const props = defineProps<
  CheckboxRootProps & {
    modelValue?: boolean | 'indeterminate'
    class?: HTMLAttributes['class']
  }
>()

const emit = defineEmits<{
  (e: 'update:checked', payload: boolean | 'indeterminate'): void
  (e: 'update:modelValue', payload: boolean | 'indeterminate'): void
}>()

const isChecked = computed({
  get: () => (props.modelValue !== undefined ? props.modelValue : props.checked),
  set: (val: boolean | 'indeterminate') => {
    emit('update:checked', val)
    emit('update:modelValue', val)
  }
})
</script>

<template>
  <CheckboxRoot
    v-model:checked="isChecked"
    :disabled="props.disabled"
    :required="props.required"
    :name="props.name"
    :value="props.value"
    :as="props.as"
    :as-child="props.asChild"
    :class="cn(
      'peer size-4 shrink-0 rounded-sm border border-base-300 dark:border-base-content/30 shadow-xs transition-shadow focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:border-primary data-[state=checked]:bg-primary data-[state=checked]:text-primary-content',
      props.class
    )"
  >
    <CheckboxIndicator class="flex h-full w-full items-center justify-center text-current">
      <Check class="size-3.5" />
    </CheckboxIndicator>
  </CheckboxRoot>
</template>
