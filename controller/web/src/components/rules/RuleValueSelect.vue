<template>
  <div class="relative min-w-0">
    <button
      ref="triggerRef"
      type="button"
      class="select select-bordered select-sm w-full min-w-0 cursor-pointer text-left"
      :class="selectedValues.length ? '' : 'text-base-content/50'"
      :disabled="disabled"
      :title="selectedValues.join(', ')"
      :aria-expanded="isOpen"
      aria-haspopup="listbox"
      @click="toggle"
      @keydown="handleTriggerKeydown"
    >
      <span class="min-w-0 flex-1 truncate">
        {{ selectedValues.length ? selectedValues.join(', ') : placeholder || '请选择' }}
      </span>
      <ChevronDownIcon class="ml-1 h-4 w-4 shrink-0 opacity-50" />
    </button>
  </div>

  <Teleport
    v-if="isMounted"
    to="#app-content"
  >
    <Transition name="floating-menu">
      <div
        v-if="isOpen"
        ref="panelRef"
        class="floating-menu-panel !p-0"
        :style="panelStyle"
        role="listbox"
        :aria-multiselectable="multiple || undefined"
      >
        <div
          v-if="showSearch"
          class="border-base-content/10 bg-base-100 sticky top-0 z-10 border-b p-1.5"
        >
          <label class="input input-sm flex items-center gap-1.5">
            <SearchIcon class="h-4 w-4 shrink-0 opacity-50" />
            <input
              ref="searchRef"
              v-model="query"
              type="text"
              class="min-w-0 flex-1 bg-transparent outline-none"
              placeholder="搜索选项"
              @keydown="handleSearchKeydown"
            />
          </label>
        </div>

        <div class="max-h-64 overflow-y-auto p-1">
          <div
            v-for="(option, index) in filteredOptions"
            :key="option"
            role="option"
            :aria-selected="isSelected(option)"
            class="floating-menu-option gap-2"
            :class="[
              activeIndex === index && 'bg-base-200',
              isSelected(option) && 'text-primary font-medium',
            ]"
            @pointermove="activeIndex = index"
            @pointerdown.prevent
            @click="choose(option)"
          >
            <span class="min-w-0 flex-1 truncate">{{ option }}</span>
            <CheckIcon
              v-if="isSelected(option)"
              class="h-4 w-4 shrink-0 stroke-[2.5]"
            />
          </div>

          <div
            v-if="!filteredOptions.length"
            class="text-base-content/40 px-2 py-3 text-center text-xs"
          >
            {{ emptyText }}
          </div>
        </div>

        <div
          v-if="multiple && selectedValues.length"
          class="border-base-content/10 flex justify-end border-t p-1"
        >
          <button
            type="button"
            class="btn btn-ghost btn-xs text-base-content/60"
            @click="clear"
          >
            清空
          </button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useFloatingMenu } from '@/composables/floatingMenu'
import { CheckIcon } from '@heroicons/vue/24/outline'
import { ChevronDownIcon, SearchIcon } from '@lucide/vue'
import { computed, nextTick, onMounted, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue?: string | string[]
    options?: string[]
    /** 多选由字段类型决定：list 字段多选，select/string 字段单选 */
    multiple?: boolean
    placeholder?: string
    emptyText?: string
    disabled?: boolean
    /** 选项较多时显示搜索框 */
    searchable?: boolean
  }>(),
  {
    modelValue: '',
    options: () => [],
    multiple: false,
    placeholder: '',
    emptyText: '暂无可选项',
    disabled: false,
    searchable: true,
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: string | string[]): void
}>()

const triggerRef = ref<HTMLButtonElement>()
const panelRef = ref<HTMLDivElement>()
const searchRef = ref<HTMLInputElement>()
const isMounted = ref(false)
const isOpen = ref(false)
const query = ref('')
const activeIndex = ref(0)

const { panelStyle, remeasure } = useFloatingMenu(triggerRef, panelRef, isOpen, {
  minimumWidth: 220,
  maximumHeight: 360,
})

const selectedValues = computed(() => {
  const value = props.modelValue
  if (Array.isArray(value)) return value.filter(Boolean)
  return value ? [value] : []
})

// 保留当前值：即使它已不在配置的选项里（例如规则集被删除），也仍能显示与移除。
const allOptions = computed(() => {
  const merged = [...props.options]
  const seen = new Set(merged)
  for (const value of selectedValues.value) {
    if (!seen.has(value)) {
      merged.push(value)
      seen.add(value)
    }
  }
  return merged
})

const showSearch = computed(() => props.searchable && allOptions.value.length > 6)

const filteredOptions = computed(() => {
  const text = query.value.trim().toLowerCase()
  if (!text) return allOptions.value
  return allOptions.value.filter((option) => option.toLowerCase().includes(text))
})

const isSelected = (option: string) => selectedValues.value.includes(option)

const scrollActiveIntoView = () => {
  nextTick(() => {
    panelRef.value
      ?.querySelectorAll<HTMLElement>('[role="option"]')
      [activeIndex.value]?.scrollIntoView({ block: 'nearest' })
  })
}

const open = () => {
  if (props.disabled || isOpen.value) return
  query.value = ''
  activeIndex.value = 0
  isOpen.value = true
  nextTick(() => {
    if (showSearch.value) searchRef.value?.focus()
    remeasure()
  })
}

const close = () => {
  isOpen.value = false
  query.value = ''
}

const toggle = () => (isOpen.value ? close() : open())

function choose(option: string) {
  if (props.multiple) {
    const values = [...selectedValues.value]
    const index = values.indexOf(option)
    if (index >= 0) values.splice(index, 1)
    else values.push(option)
    emit('update:modelValue', values)
    return
  }

  emit('update:modelValue', isSelected(option) ? '' : option)
  close()
  triggerRef.value?.focus()
}

function clear() {
  emit('update:modelValue', props.multiple ? [] : '')
  if (!props.multiple) close()
}

function moveActive(direction: 1 | -1) {
  if (!filteredOptions.value.length) return
  activeIndex.value =
    (activeIndex.value + direction + filteredOptions.value.length) % filteredOptions.value.length
  scrollActiveIntoView()
}

function handleSearchKeydown(event: KeyboardEvent) {
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    moveActive(1)
    return
  }
  if (event.key === 'ArrowUp') {
    event.preventDefault()
    moveActive(-1)
    return
  }
  if (event.key === 'Enter') {
    event.preventDefault()
    const option = filteredOptions.value[activeIndex.value]
    if (option) choose(option)
    return
  }
  if (event.key === 'Escape') {
    event.preventDefault()
    close()
    triggerRef.value?.focus()
  }
}

function handleTriggerKeydown(event: KeyboardEvent) {
  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    if (!isOpen.value) open()
    else moveActive(event.key === 'ArrowUp' ? -1 : 1)
    return
  }
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    if (!isOpen.value) {
      open()
      return
    }
    const option = filteredOptions.value[activeIndex.value]
    if (option) choose(option)
    return
  }
  if (event.key === 'Escape') {
    event.preventDefault()
    close()
  }
}

watch(query, () => {
  activeIndex.value = 0
  remeasure()
})

watch(
  () => [props.modelValue, props.options],
  () => {
    if (isOpen.value) remeasure()
  },
  { deep: true },
)

onMounted(() => {
  isMounted.value = true
})
</script>
