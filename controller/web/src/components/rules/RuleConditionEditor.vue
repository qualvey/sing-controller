<template>
  <div class="flex flex-col gap-2">
    <div class="flex flex-col gap-2">
      <div
        v-for="row in rows"
        :key="row.id"
        class="grid items-start gap-2"
        style="grid-template-columns: minmax(0, 8.5rem) minmax(0, 1fr) auto"
      >
        <!-- 左：字段类型 -->
        <div
          v-if="fieldOf(row)?.common"
          class="border-base-content/10 bg-base-200/40 flex h-8 min-w-0 items-center gap-1.5 rounded-md border px-2 text-xs"
          :title="fieldOf(row)?.label"
        >
          <span class="min-w-0 flex-1 truncate">{{ fieldOf(row)?.label }}</span>
          <span class="badge badge-ghost badge-xs shrink-0">常用</span>
        </div>
        <RuleFieldPicker
          v-else
          :fields="fields"
          :model-value="row.key"
          :exclude="usedKeys(row)"
          :auto-open="autoOpenRowId === row.id"
          @update:model-value="changeField(row, $event)"
        />

        <!-- 右：字段值 -->
        <div class="min-w-0 pt-0">
          <template v-if="!fieldOf(row)">
            <div class="text-base-content/40 flex h-8 items-center text-xs">
              请选择字段
            </div>
          </template>

          <template v-else-if="fieldOf(row)!.type === 'bool'">
            <label class="flex h-8 items-center gap-2">
              <Checkbox
                :model-value="row.value === true"
                @update:model-value="row.value = $event"
              />
              <span class="text-base-content/60 text-xs">启用</span>
            </label>
          </template>

          <template v-else-if="isEnumField(fieldOf(row)!)">
            <RuleValueSelect
              :model-value="row.value as string | string[]"
              :options="fieldOptions(fieldOf(row)!)"
              :multiple="isListField(fieldOf(row)!)"
              :placeholder="fieldOf(row)!.placeholder || '请选择'"
              :empty-text="emptyOptionText(fieldOf(row)!)"
              @update:model-value="row.value = $event"
            />
          </template>

          <template v-else-if="isListField(fieldOf(row)!)">
            <ChipInput
              v-model="row.value as string[]"
              :placeholder="fieldOf(row)!.placeholder || '输入后回车添加，可多值'"
              :suggestions="suggestionsFor(fieldOf(row)!)"
            />
          </template>

          <template v-else-if="fieldOf(row)!.type === 'json'">
            <textarea
              v-model="row.value as string"
              rows="3"
              class="textarea textarea-bordered w-full font-mono text-xs"
              :placeholder="fieldOf(row)!.placeholder || 'JSON 内容'"
            />
          </template>

          <template v-else>
            <input
              v-model="row.value as string"
              type="text"
              class="input input-bordered input-sm w-full"
              :placeholder="fieldOf(row)!.placeholder || '请输入'"
            />
          </template>
        </div>

        <!-- 减号：移除键值对（常用字段不可移除） -->
        <button
          type="button"
          class="btn btn-circle btn-ghost btn-xs mt-1 shrink-0 text-base-content/45 hover:text-error"
          :class="{ invisible: fieldOf(row)?.common }"
          :disabled="fieldOf(row)?.common"
          :aria-label="`移除 ${fieldOf(row)?.label || '字段'}`"
          :title="fieldOf(row)?.common ? '常用字段不可移除' : '移除此键值对'"
          @click="removeRow(row)"
        >
          <MinusIcon class="h-4 w-4" />
        </button>
      </div>
    </div>

    <button
      type="button"
      class="btn btn-ghost btn-sm text-primary w-fit gap-1"
      @click="addRow"
    >
      <PlusIcon class="h-4 w-4" />
      {{ addLabel }}
    </button>
  </div>
</template>

<script setup lang="ts">
import ChipInput from '@/components/common/ChipInput.vue'
import { MinusIcon, PlusIcon } from '@lucide/vue'
import { computed, ref } from 'vue'
import RuleFieldPicker from './RuleFieldPicker.vue'
import RuleValueSelect from './RuleValueSelect.vue'
import {
  emptyFieldValue,
  isEnumField,
  isListField,
  nextRowId,
  type ConditionField,
  type ConditionRow,
} from './conditionFields'

const props = withDefaults(
  defineProps<{
    fields: ConditionField[]
    suggestions?: Record<string, string[]>
    /** 动态选项来源，键为字段的 optionsSource */
    optionSources?: Record<string, string[]>
    addLabel?: string
  }>(),
  {
    suggestions: () => ({}),
    optionSources: () => ({}),
    addLabel: '添加匹配条件',
  },
)

const rows = defineModel<ConditionRow[]>({ required: true })
const autoOpenRowId = ref<number | null>(null)

const fieldMap = computed(() => new Map(props.fields.map((field) => [field.key, field])))

function fieldOf(row: ConditionRow) {
  return fieldMap.value.get(row.key)
}

function usedKeys(current: ConditionRow) {
  return rows.value.filter((row) => row.id !== current.id).map((row) => row.key)
}

function changeField(row: ConditionRow, key: string) {
  const field = fieldMap.value.get(key)
  if (!field) return
  if (row.key === key) return
  row.key = key
  row.value = emptyFieldValue(field)
}

function removeRow(row: ConditionRow) {
  const field = fieldOf(row)
  if (field?.common) return
  const index = rows.value.findIndex((item) => item.id === row.id)
  if (index >= 0) rows.value.splice(index, 1)
}

function addRow() {
  const id = nextRowId()
  rows.value.push({ id, key: '', value: '' })
  autoOpenRowId.value = id
}

function fieldOptions(field: ConditionField) {
  if (field.optionsSource) return props.optionSources[field.optionsSource] ?? []
  return field.options ?? []
}

function emptyOptionText(field: ConditionField) {
  if (field.optionsSource === 'rule-set') return '暂无规则集（请先在规则集中配置）'
  return '暂无可选项'
}

function suggestionsFor(field: ConditionField) {
  return [...new Set([...(props.suggestions[field.key] ?? []), ...(field.options ?? [])])]
}
</script>
