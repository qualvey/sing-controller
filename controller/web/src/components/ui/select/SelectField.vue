<template>
  <SelectInput
    v-model="model"
    :options="normalizedOptions"
    :placeholder="placeholder"
    :disabled="disabled"
    :panel-class="panelClass"
    :class="['select-bordered select-sm w-full min-w-0', $attrs.class]"
    v-bind="filteredAttrs"
    @change="emit('change', $event)"
  >
    <template
      v-if="$slots.value"
      #value="slotProps"
    >
      <slot
        name="value"
        v-bind="slotProps"
      />
    </template>
    <template
      v-if="$slots.option"
      #option="slotProps"
    >
      <slot
        name="option"
        v-bind="slotProps"
      />
    </template>
  </SelectInput>
</template>

<script setup lang="ts" generic="T = any">
import SelectInput, { type SelectOption as CommonSelectOption } from '@/components/common/SelectInput.vue'
import { computed, useAttrs } from 'vue'

export type SelectOption<V = any> =
  | string
  | number
  | {
      value: V
      label?: string
      disabled?: boolean
      group?: string
      theme?: string
    }

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    options?: readonly (
      | string
      | number
      | {
          value: any
          label?: string
          disabled?: boolean
          group?: string
          theme?: string
        }
    )[]
    placeholder?: string
    disabled?: boolean
    panelClass?: string
  }>(),
  {
    options: () => [],
    placeholder: '',
    disabled: false,
    panelClass: '',
  },
)

const emit = defineEmits<{
  (e: 'change', value: T): void
}>()

const model = defineModel<T>()
const attrs = useAttrs()

const filteredAttrs = computed(() => {
  const rest = { ...attrs }
  delete rest.class
  return rest
})

const normalizedOptions = computed<CommonSelectOption<any>[]>(() => {
  if (!props.options) return []
  return props.options.map((opt) => {
    if (typeof opt === 'string' || typeof opt === 'number') {
      const valStr = String(opt)
      return {
        value: opt,
        label: valStr === '' ? (props.placeholder || '默认 (省略)') : valStr,
      }
    }
    return {
      value: opt.value,
      label: opt.label ?? (opt.value === '' ? (props.placeholder || '默认') : String(opt.value ?? '')),
      disabled: opt.disabled,
      group: opt.group,
      theme: opt.theme,
    }
  })
})
</script>
