<script setup lang="ts">
import { computed, onMounted, ref, watch, type HTMLAttributes } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue?: boolean | 'indeterminate' | null
    checked?: boolean | 'indeterminate' | null
    disabled?: boolean
    required?: boolean
    name?: string
    value?: string
    class?: HTMLAttributes['class']
  }>(),
  {
    modelValue: undefined,
    checked: undefined,
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'update:checked', value: boolean): void
  (e: 'change', value: Event): void
}>()

const inputRef = ref<HTMLInputElement | null>(null)

const isChecked = computed({
  get: () => {
    if (props.modelValue !== undefined && props.modelValue !== null) {
      return props.modelValue === true
    }
    if (props.checked !== undefined && props.checked !== null) {
      return props.checked === true
    }
    return false
  },
  set: (val: boolean) => {
    emit('update:modelValue', val)
    emit('update:checked', val)
  }
})

const updateIndeterminate = () => {
  if (inputRef.value) {
    const val =
      props.modelValue !== undefined && props.modelValue !== null
        ? props.modelValue
        : props.checked
    inputRef.value.indeterminate = val === 'indeterminate'
  }
}

watch(() => [props.modelValue, props.checked], updateIndeterminate)
onMounted(updateIndeterminate)
</script>

<template>
  <input
    ref="inputRef"
    v-model="isChecked"
    type="checkbox"
    :disabled="props.disabled"
    :required="props.required"
    :name="props.name"
    :value="props.value"
    class="checkbox checkbox-primary checkbox-sm rounded shrink-0"
    :class="props.class"
  />
</template>
