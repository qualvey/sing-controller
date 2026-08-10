<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'
import { ElMessage } from 'element-plus'
import { EditorView, keymap } from '@codemirror/view'
import { EditorState } from '@codemirror/state'
import { basicSetup } from 'codemirror'
import { json, jsonParseLinter } from '@codemirror/lang-json'
import { oneDark } from '@codemirror/theme-one-dark'
import { linter, lintGutter } from '@codemirror/lint'
import { defaultKeymap, indentWithTab } from '@codemirror/commands'
import { foldGutter } from '@codemirror/language'
import { api } from '../api'
import { useStatusStore } from '../stores/status'

const statusStore = useStatusStore()

const editorHost = ref<HTMLElement>()
const editor = shallowRef<EditorView>()
const loading = ref(false)
const saving = ref(false)

const loadConfig = async () => {
  loading.value = true
  try {
    const text = JSON.stringify(await api.config(), null, 2)
    editor.value?.dispatch({ changes: { from: 0, to: editor.value.state.doc.length, insert: text } })
  } catch (e) {
    ElMessage.error((e as Error).message || '加载配置失败')
  } finally {
    loading.value = false
  }
}

// 格式化：JSON.stringify 重排当前文档
const format = () => {
  try {
    const text = editor.value?.state.doc.toString() ?? ''
    const parsed = JSON.parse(text)
    editor.value?.dispatch({ changes: { from: 0, to: editor.value.state.doc.length, insert: JSON.stringify(parsed, null, 2) } })
    ElMessage.success('已格式化')
  } catch (e) {
    ElMessage.error(`JSON 格式错误：${(e as Error).message}`)
  }
}

const save = async () => {
  const text = editor.value?.state.doc.toString() ?? ''
  let parsed: unknown
  try {
    parsed = JSON.parse(text)
  } catch (e) {
    ElMessage.error(`JSON 格式错误：${(e as Error).message}`)
    return
  }
  if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
    ElMessage.error('配置必须是 JSON 对象')
    return
  }
  saving.value = true
  try {
    const res = (await api.saveConfig(parsed)) as { load_error?: string; message?: string }
    if (res.load_error) {
      ElMessage.warning(res.message || `配置已保存，但加载新配置失败：${res.load_error}`)
    } else {
      ElMessage.success('配置已保存（已通过 sing-box 校验）')
    }
    await statusStore.refresh()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  if (!editorHost.value) return
  const state = EditorState.create({
    doc: '{}',
    extensions: [
      basicSetup,
      json(),
      oneDark,
      foldGutter(),
      lintGutter(),
      linter(jsonParseLinter()),
      keymap.of([indentWithTab, ...defaultKeymap]),
      EditorView.lineWrapping
    ]
  })
  editor.value = new EditorView({ state, parent: editorHost.value })
  loadConfig()
  statusStore.refresh()
})

onBeforeUnmount(() => {
  editor.value?.destroy()
})
</script>

<template>
  <div class="page">
    <div class="toolbar">
      <el-button :loading="loading" @click="loadConfig">刷新</el-button>
      <el-button @click="format">格式化</el-button>
      <span class="hint">直接编辑 sing-box 主配置 JSON（CodeMirror：实时 JSON 校验、折叠、Tab 缩进）；保存时后端做完整校验</span>
      <span class="spacer" />
      <el-button type="primary" :loading="saving" @click="save">保存配置</el-button>
    </div>
    <div ref="editorHost" class="editor-host" />
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}
.hint {
  font-size: 12px;
  color: #909399;
}
.spacer {
  flex: 1;
}
.editor-host {
  border: 1px solid #30363d;
  border-radius: 6px;
  overflow: hidden;
  height: calc(100vh - 170px);
  text-align: left;
}
.editor-host :deep(.cm-editor) {
  height: 100%;
  font-size: 13px;
}
</style>
