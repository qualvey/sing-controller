<script setup lang="ts">
// 通用 chip 输入（替代 el-select multiple+allow-create）：回车/逗号添加，chip 可删除
import { ref } from 'vue'

defineProps<{
  placeholder?: string
  /** 建议候选（可选，datalist 提示） */
  suggestions?: string[]
}>()

const model = defineModel<string[]>({ default: () => [] })
const input = ref('')

const add = () => {
  const parts = input.value
    .split(/[,，\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  for (const p of parts) {
    if (!model.value.includes(p)) model.value.push(p)
  }
  input.value = ''
}
const remove = (t: string) => {
  const i = model.value.indexOf(t)
  if (i >= 0) model.value.splice(i, 1)
}
</script>

<template>
  <div class="flex flex-col gap-1.5">
    <div v-if="model.length" class="flex flex-wrap gap-1.5">
      <span v-for="t in model" :key="t" class="badge badge-primary badge-outline gap-1">
        {{ t }}
        <button type="button" class="cursor-pointer text-xs opacity-70 hover:opacity-100" @click="remove(t)">×</button>
      </span>
    </div>
    <input
      v-model="input"
      type="text"
      class="input input-bordered input-sm w-full"
      :placeholder="placeholder || '输入后回车添加，可多值'"
      :list="suggestions?.length ? 'chip-suggest' : undefined"
      @keydown.enter.prevent="add"
      @blur="add"
    />
    <datalist v-if="suggestions?.length" id="chip-suggest">
      <option v-for="s in suggestions" :key="s" :value="s" />
    </datalist>
  </div>
</template>
