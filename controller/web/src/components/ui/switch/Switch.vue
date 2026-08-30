<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  modelValue?: boolean
  checked?: boolean
  disabled?: boolean
  class?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'update:checked', value: boolean): void
  (e: 'change', value: Event): void
}>()

const isChecked = computed({
  get: () => (props.modelValue !== undefined ? props.modelValue : !!props.checked),
  set: (val: boolean) => {
    emit('update:modelValue', val)
    emit('update:checked', val)
  }
})
</script>

<template>
  <input
    v-model="isChecked"
    type="checkbox"
    :disabled="props.disabled"
    class="toggle toggle-primary toggle-sm"
    :class="props.class"
  />
</template>
