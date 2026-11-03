<template>
  <div class="relative min-w-0">
    <button
      ref="triggerRef"
      type="button"
      class="select select-bordered select-sm w-full min-w-0 cursor-pointer text-left"
      :class="selectedField ? '' : 'text-base-content/50'"
      :disabled="disabled"
      :aria-expanded="isOpen"
      aria-haspopup="listbox"
      @click="toggle"
    >
      <span class="min-w-0 flex-1 truncate">
        {{ selectedField?.label || placeholder || '选择字段' }}
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
      >
        <div class="border-base-content/10 bg-base-100 sticky top-0 z-10 border-b p-1.5">
          <label class="input input-sm flex items-center gap-1.5">
            <SearchIcon class="h-4 w-4 shrink-0 opacity-50" />
            <input
              ref="searchRef"
              v-model="query"
              type="text"
              class="min-w-0 flex-1 bg-transparent outline-none"
              placeholder="搜索字段（支持模糊匹配）"
              @keydown="handleSearchKeydown"
            />
          </label>
        </div>

        <div class="max-h-64 overflow-y-auto p-1">
          <template
            v-for="item in displayItems"
            :key="item.type === 'group' ? `g-${item.label}` : item.field.key"
          >
            <div
              v-if="item.type === 'group'"
              class="text-base-content/45 px-2 pt-2 pb-1 text-xs font-medium"
            >
              {{ item.label }}
            </div>
            <div
              v-else
              role="option"
              :aria-selected="item.field.key === modelValue"
              class="floating-menu-option gap-2"
              :class="[
                activeIndex === item.index && 'bg-base-200',
                item.field.key === modelValue && 'text-primary font-medium',
              ]"
              @pointermove="activeIndex = item.index"
              @pointerdown.prevent
              @click="choose(item.field)"
            >
              <span class="min-w-0 flex-1 truncate">{{ item.field.label }}</span>
              <span
                v-if="query.trim()"
                class="text-base-content/40 shrink-0 text-[11px]"
              >
                {{ item.field.group }}
              </span>
              <span class="text-base-content/35 shrink-0 font-mono text-[11px]">
                {{ item.field.key }}
              </span>
            </div>
          </template>

          <div
            v-if="!optionCount"
            class="text-base-content/40 px-2 py-3 text-center text-xs"
          >
            没有匹配的字段
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useFloatingMenu } from '@/composables/floatingMenu'
import type { ConditionField } from './conditionFields'
import { ChevronDownIcon, SearchIcon } from '@lucide/vue'
import { computed, nextTick, onMounted, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    fields: ConditionField[]
    modelValue?: string
    /** 需要排除的 key（已使用的字段） */
    exclude?: string[]
    placeholder?: string
    disabled?: boolean
    /** 新添加的行挂载后自动展开搜索面板 */
    autoOpen?: boolean
  }>(),
  {
    modelValue: '',
    exclude: () => [],
    placeholder: '',
    disabled: false,
    autoOpen: false,
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'select', field: ConditionField): void
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

const selectedField = computed(() => props.fields.find((field) => field.key === props.modelValue))
const excludedKeys = computed(() => new Set(props.exclude.filter((key) => key !== props.modelValue)))

function scoreField(field: ConditionField, rawQuery: string) {
  const queryText = rawQuery.toLowerCase()
  if (!queryText) return 0

  const candidates = [field.label, field.key, field.group]
  let best = -1

  for (const candidate of candidates) {
    const text = candidate.toLowerCase()
    if (text === queryText) {
      best = Math.max(best, 1000)
      continue
    }
    if (text.startsWith(queryText)) {
      best = Math.max(best, 900 - text.length)
      continue
    }
    const index = text.indexOf(queryText)
    if (index >= 0) {
      best = Math.max(best, 800 - index - text.length * 0.01)
      continue
    }

    // 子序列模糊匹配：查询字符按顺序出现即可
    let cursor = 0
    let first = -1
    let gaps = 0
    for (const char of queryText) {
      const found = text.indexOf(char, cursor)
      if (found < 0) {
        cursor = -1
        break
      }
      if (first < 0) first = found
      gaps += found - cursor
      cursor = found + 1
    }
    if (cursor >= 0) best = Math.max(best, 400 - gaps - Math.max(first, 0))
  }

  return best
}

const filteredFields = computed(() => {
  const queryText = query.value.trim()
  const indexed = props.fields
    .map((field, index) => ({ field, index }))
    .filter(({ field }) => !excludedKeys.value.has(field.key))

  if (!queryText) return indexed

  return indexed
    .map((item) => ({ ...item, score: scoreField(item.field, queryText) }))
    .filter((item) => item.score >= 0)
    .sort((a, b) => b.score - a.score || a.index - b.index)
})

const optionCount = computed(() => filteredFields.value.length)

type DisplayItem =
  | { type: 'group'; label: string }
  | { type: 'option'; field: ConditionField; index: number }

const displayItems = computed<DisplayItem[]>(() => {
  const items: DisplayItem[] = []
  let lastGroup = ''

  filteredFields.value.forEach(({ field }, index) => {
    if (!query.value.trim() && field.group !== lastGroup) {
      lastGroup = field.group
      items.push({ type: 'group', label: field.group })
    }
    items.push({ type: 'option', field, index })
  })

  return items
})

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
    searchRef.value?.focus()
    remeasure()
  })
}

const close = () => {
  isOpen.value = false
  query.value = ''
}

const toggle = () => (isOpen.value ? close() : open())

function choose(field: ConditionField) {
  emit('update:modelValue', field.key)
  emit('select', field)
  close()
  triggerRef.value?.focus()
}

function moveActive(direction: 1 | -1) {
  if (!optionCount.value) return
  activeIndex.value =
    (activeIndex.value + direction + optionCount.value) % optionCount.value
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
    const field = filteredFields.value[activeIndex.value]?.field
    if (field) choose(field)
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

onMounted(() => {
  isMounted.value = true
  if (props.autoOpen) {
    nextTick(open)
  }
})
</script>
