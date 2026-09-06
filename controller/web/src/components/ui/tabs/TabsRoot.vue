<script setup lang="ts">
import { computed, provide, ref, type HTMLAttributes } from 'vue'
import { tabsContextKey } from './tabsContext'
import { cn } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    modelValue?: string | number
    defaultValue?: string | number
    class?: HTMLAttributes['class']
  }>(),
  {
    modelValue: undefined,
    defaultValue: undefined,
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: string | number): void
}>()

const innerValue = ref<string | number | undefined>(props.defaultValue)

const currentTab = computed(() => {
  return props.modelValue !== undefined ? props.modelValue : innerValue.value
})

const changeTab = (val: string | number) => {
  innerValue.value = val
  emit('update:modelValue', val)
}

provide(tabsContextKey, {
  modelValue: currentTab,
  changeTab,
})
</script>

<template>
  <div :class="cn('w-full', props.class)">
    <slot />
  </div>
</template>
