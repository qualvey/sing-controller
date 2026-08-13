<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { EditorView, keymap } from '@codemirror/view'
import { EditorState } from '@codemirror/state'
import { basicSetup } from 'codemirror'
import { jsonc, jsoncLinter, formatJsoncDoc, parseJsonc } from '../editor/jsonc'
import { foldGutter } from '@codemirror/language'
import { defaultKeymap, indentWithTab } from '@codemirror/commands'
import { api } from '../api'
import { useCmTheme } from '../composables/useCmTheme'
import { showToast } from '../helper/toast'

const { ext: themeExt, watchTheme } = useCmTheme()

const props = defineProps<{
  /** 配置段点路径，如 'dns' / 'route' / 'route.rule_set' / 'inbounds' / 'certificate' */
  segment: string
}>()

const emit = defineEmits<{ saved: [] }>()

const editorHost = ref<HTMLElement>()
const editor = shallowRef<EditorView>()
const loading = ref(false)
const saving = ref(false)

function getPath(obj: unknown, path: string): unknown {
  return path.split('.').reduce<unknown>((o: unknown, k: string) => {
    if (o == null || typeof o !== 'object') return undefined
    return (o as Record<string, unknown>)[k]
  }, obj)
}

function setPath(obj: Record<string, any>, path: string, value: unknown): void {
  const keys = path.split('.')
  let cur: Record<string, any> = obj
  for (let i = 0; i < keys.length - 1; i++) {
    const k = keys[i]
    if (cur[k] == null || typeof cur[k] !== 'object' || Array.isArray(cur[k])) cur[k] = {}
    cur = cur[k]
  }
  const last = keys[keys.length - 1]
  if (value === null) {
    delete cur[last]
  } else {
    cur[last] = value
  }
}

// 从主配置 raw 中提取本段内容
const load = async () => {
  loading.value = true
  try {
    const raw = await api.configRaw()
    const parsed = parseJsonc(raw)
    if (!parsed.ok) throw new Error(parsed.message)
    const seg = getPath(parsed.value, props.segment)
    const text = seg == null ? 'null' : JSON.stringify(seg, null, 2)
    editor.value?.dispatch({ changes: { from: 0, to: editor.value.state.doc.length, insert: text } })
  } catch (e) {
    showToast((e as Error).message || '加载配置段失败', 'error')
  } finally {
    loading.value = false
  }
}

// 编辑本段 → 合并进主配置（整段替换，其余配置不动）→ 全量 raw 保存
const save = async () => {
  const text = editor.value?.state.doc.toString() ?? ''
  const parsed = parseJsonc(text)
  if (!parsed.ok) {
    showToast(`JSON 格式错误：${parsed.message}`, 'error')
    return
  }
  saving.value = true
  try {
    const raw = await api.configRaw()
    const root = parseJsonc(raw)
    if (!root.ok) throw new Error(root.message)
    const rootObj = (root.value ?? {}) as Record<string, any>
    if (typeof rootObj !== 'object' || rootObj === null || Array.isArray(rootObj)) {
      throw new Error('主配置不是 JSON 对象')
    }
    setPath(rootObj, props.segment, parsed.value)
    await api.saveConfigRaw(JSON.stringify(rootObj, null, 2))
    showToast(`「${props.segment}」已合并保存（其余配置不变）`, 'success')
    emit('saved')
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
  } finally {
    saving.value = false
  }
}

const format = () => {
  const ed = editor.value
  if (ed) formatJsoncDoc(ed)
}

watch(
  () => props.segment,
  () => load()
)

onMounted(() => {
  if (!editorHost.value) return
  const state = EditorState.create({
    doc: 'null',
    extensions: [
      basicSetup,
      jsonc(),
      themeExt,
      foldGutter(),
      jsoncLinter(),
      keymap.of([indentWithTab, { key: 'Mod-Shift-f', run: formatJsoncDoc }, ...defaultKeymap]),
      EditorView.theme({
        '&': { height: '100%', fontSize: '13px' },
        '.cm-scroller': {
          fontFamily: "ui-monospace, 'JetBrains Mono', 'Cascadia Code', Consolas, monospace",
          lineHeight: '1.6'
        }
      })
    ]
  })
  editor.value = new EditorView({ state, parent: editorHost.value })
  watchTheme(() => editor.value)
  load()
})

onBeforeUnmount(() => {
  editor.value?.destroy()
})
</script>

<template>
  <div class="flex flex-col gap-2">
    <div class="flex items-center gap-2">
      <span class="text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]">{{ segment }} 段</span>
      <button class="btn btn-ghost btn-xs" :disabled="loading" @click="load">刷新</button>
      <button class="btn btn-ghost btn-xs" @click="format">格式化</button>
      <span class="text-xs text-[#909399]">手动编辑该段 JSON（JSONC，支持注释）；保存时整段替换并入主配置，其余配置不变；写 null 删除该段</span>
      <span class="flex-1" />
      <button class="btn btn-primary btn-sm" :disabled="saving" @click="save">保存并合并</button>
    </div>
    <div ref="editorHost" class="h-[420px] overflow-hidden rounded-md border border-[#30363d] text-left" />
  </div>
</template>

<style scoped>
.source-editor :deep(.cm-editor) {
  height: 100%;
}
</style>
