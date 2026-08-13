<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { showToast } from '../helper/toast'
import { EditorView, keymap } from '@codemirror/view'
import { Compartment, EditorState } from '@codemirror/state'
import { basicSetup } from 'codemirror'
import { openSearchPanel } from '@codemirror/search'
import { jsonc, jsoncLinter, formatJsoncDoc, parseJsonc } from '../editor/jsonc'
import { oneDark } from '@codemirror/theme-one-dark'
import { lintGutter } from '@codemirror/lint'
import { defaultKeymap, indentWithTab } from '@codemirror/commands'
import { foldGutter } from '@codemirror/language'
import { api } from '../api'
import { useStatusStore } from '../stores/status'
import { useThemeStore } from '../stores/theme'

const statusStore = useStatusStore()

const editorHost = ref<HTMLElement>()
const editor = shallowRef<EditorView>()
const loading = ref(false)
const saving = ref(false)
const dirty = ref(false)
const configPath = ref('')

// 状态栏数据
const cursor = ref({ line: 1, col: 1 })
const selected = ref(0)
const docLines = ref(1)

// 编辑器 UI 主题（行号/搜索面板/提示框等外壳）：dark 深色 / light 亮色
const uiTheme = (dark: boolean) =>
  EditorView.theme(
    dark
      ? {
          '&': {
            height: '100%',
            fontSize: '13px'
          },
          '.cm-scroller': {
            fontFamily: "ui-monospace, 'JetBrains Mono', 'Cascadia Code', 'Fira Code', Consolas, monospace",
            fontVariantLigatures: 'normal',
            lineHeight: '1.7'
          },
          '.cm-gutters': {
            backgroundColor: '#21252b',
            color: '#636d83',
            borderRight: '1px solid #2c313a'
          },
          '.cm-activeLineGutter': {
            backgroundColor: '#2c313a',
            color: '#abb2bf'
          },
          '.cm-activeLine': {
            backgroundColor: '#2c313a55'
          },
          '.cm-panels': {
            backgroundColor: '#21252b',
            color: '#abb2bf'
          },
          '.cm-panels-top': {
            borderBottom: '1px solid #30363d'
          },
          '.cm-search': {
            backgroundColor: '#21252b',
            color: '#abb2bf',
            padding: '6px 10px',
            display: 'flex',
            alignItems: 'center',
            gap: '6px'
          },
          '.cm-search input, .cm-search select': {
            backgroundColor: '#2c313a',
            color: '#abb2bf',
            border: '1px solid #3e4451',
            borderRadius: '3px',
            padding: '2px 6px',
            outline: 'none'
          },
          '.cm-search input:focus': {
            borderColor: '#61afef'
          },
          '.cm-search button': {
            backgroundColor: '#2c313a',
            color: '#abb2bf',
            border: '1px solid #3e4451',
            borderRadius: '3px',
            padding: '2px 8px',
            cursor: 'pointer'
          },
          '.cm-search button:hover': {
            backgroundColor: '#3e4451'
          },
          '.cm-search .cm-textfield': {
            backgroundColor: '#2c313a'
          },
          '.cm-tooltip': {
            backgroundColor: '#21252b',
            color: '#abb2bf',
            border: '1px solid #3e4451'
          },
          '.cm-tooltip-autocomplete > ul > li[aria-selected]': {
            backgroundColor: '#3e4451',
            color: '#abb2bf'
          },
          '.cm-tooltip.cm-tooltip-autocomplete ul li[aria-selected]': {
            backgroundColor: '#3e4451'
          },
          '&.cm-focused .cm-matchingBracket, &.cm-focused .cm-nonmatchingBracket': {
            backgroundColor: '#3e4451'
          },
          '.cm-cursor': {
            borderLeftColor: '#abb2bf'
          },
          '.cm-selectionBackground, &.cm-focused .cm-selectionBackground, ::selection': {
            backgroundColor: '#3e445199'
          }
        }
      : {
          '&': {
            height: '100%',
            fontSize: '13px'
          },
          '.cm-scroller': {
            fontFamily: "ui-monospace, 'JetBrains Mono', 'Cascadia Code', 'Fira Code', Consolas, monospace",
            fontVariantLigatures: 'normal',
            lineHeight: '1.7'
          },
          '.cm-gutters': {
            backgroundColor: '#f5f7fa',
            color: '#909399',
            borderRight: '1px solid #e4e7ed'
          },
          '.cm-activeLineGutter': {
            backgroundColor: '#eef1f5',
            color: '#606266'
          },
          '.cm-activeLine': {
            backgroundColor: '#f0f4f855'
          },
          '.cm-panels': {
            backgroundColor: '#ffffff',
            color: '#303133'
          },
          '.cm-panels-top': {
            borderBottom: '1px solid #e4e7ed'
          },
          '.cm-search': {
            backgroundColor: '#ffffff',
            color: '#303133',
            padding: '6px 10px',
            display: 'flex',
            alignItems: 'center',
            gap: '6px'
          },
          '.cm-search input, .cm-search select': {
            backgroundColor: '#fff',
            color: '#303133',
            border: '1px solid #dcdfe6',
            borderRadius: '3px',
            padding: '2px 6px',
            outline: 'none'
          },
          '.cm-search input:focus': {
            borderColor: '#409eff'
          },
          '.cm-search button': {
            backgroundColor: '#f5f7fa',
            color: '#303133',
            border: '1px solid #dcdfe6',
            borderRadius: '3px',
            padding: '2px 8px',
            cursor: 'pointer'
          },
          '.cm-search button:hover': {
            backgroundColor: '#e8eaed'
          },
          '.cm-search .cm-textfield': {
            backgroundColor: '#fff'
          },
          '.cm-tooltip': {
            backgroundColor: '#fff',
            color: '#303133',
            border: '1px solid #dcdfe6'
          },
          '.cm-tooltip-autocomplete > ul > li[aria-selected]': {
            backgroundColor: '#ecf5ff',
            color: '#303133'
          },
          '.cm-tooltip.cm-tooltip-autocomplete ul li[aria-selected]': {
            backgroundColor: '#ecf5ff'
          },
          '&.cm-focused .cm-matchingBracket, &.cm-focused .cm-nonmatchingBracket': {
            backgroundColor: '#e8eaed'
          },
          '.cm-cursor': {
            borderLeftColor: '#303133'
          },
          '.cm-selectionBackground, &.cm-focused .cm-selectionBackground, ::selection': {
            backgroundColor: '#c6e2ff99'
          }
        }
  )

// 主题 compartment：dark → [oneDark, 深色 UI]；light → [亮色 UI]
const themeStore = useThemeStore()
const themeComp = new Compartment()
const themeExtension = () =>
  themeComp.of(themeStore.mode === 'dark' ? [oneDark, uiTheme(true)] : [uiTheme(false)])
const syncEditorTheme = (view: EditorView) => {
  view.dispatch({ effects: themeComp.reconfigure(themeStore.mode === 'dark' ? [oneDark, uiTheme(true)] : [uiTheme(false)]) })
}

const updateStats = (view: EditorView) => {
  const pos = view.state.selection.main.head
  const line = view.state.doc.lineAt(pos)
  cursor.value = { line: line.number, col: pos - line.from + 1 }
  const sel = view.state.selection.main
  selected.value = Math.max(sel.from, sel.to) - Math.min(sel.from, sel.to)
  docLines.value = view.state.doc.lines
}

// 原样读取主配置文件（保留注释/格式/字段顺序）
const loadConfig = async () => {
  loading.value = true
  try {
    const text = await api.configRaw()
    configPath.value = statusStore.status?.config_path || ''
    editor.value?.dispatch({ changes: { from: 0, to: editor.value.state.doc.length, insert: text } })
    if (editor.value) updateStats(editor.value)
    dirty.value = false
  } catch (e) {
    showToast((e as Error).message || '加载配置失败', 'error')
  } finally {
    loading.value = false
  }
}

// 格式化：VSCode 同款 JSONC 格式化（jsonc-parser），保留注释与尾逗号
const format = () => {
  const ed = editor.value
  if (!ed) return
  if (formatJsoncDoc(ed)) {
    showToast('已格式化（保留注释）', 'success')
  }
}

// 原样保存：文本直传后端（sing-box 解析校验 + 原子写盘，注释/格式保留）
const save = async () => {
  const text = editor.value?.state.doc.toString() ?? ''
  const parsed = parseJsonc(text)
  if (!parsed.ok) {
    showToast(`配置格式错误：${parsed.message}`, 'error')
    return
  }
  if (typeof parsed.value !== 'object' || parsed.value === null || Array.isArray(parsed.value)) {
    showToast('配置必须是 JSON 对象', 'error')
    return
  }
  saving.value = true
  try {
    const res = (await api.saveConfigRaw(text)) as { load_error?: string; message?: string }
    if (res.load_error) {
    showToast(res.message || `配置已保存，但加载新配置失败：${res.load_error}`, 'warning')
    } else {
    showToast('配置已保存（原样写盘，已通过 sing-box 校验）', 'success')
    }
    dirty.value = false
    await statusStore.refresh()
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
  } finally {
    saving.value = false
  }
}

const openSearch = () => {
  editor.value && openSearchPanel(editor.value)
}

onMounted(() => {
  if (!editorHost.value) return
  const state = EditorState.create({
    doc: '{}',
    extensions: [
      basicSetup,
      jsonc(),
      themeExtension(),
      foldGutter(),
      lintGutter(),
      jsoncLinter(),
      keymap.of([indentWithTab, { key: 'Mod-Shift-f', run: formatJsoncDoc }, ...defaultKeymap]),
      EditorView.lineWrapping,
      EditorView.updateListener.of((u) => {
        if (u.docChanged || u.selectionSet || u.focusChanged) updateStats(u.view)
        if (u.docChanged) dirty.value = true
      })
    ]
  })
  editor.value = new EditorView({ state, parent: editorHost.value })
  // 主题切换联动
  watch(
    () => themeStore.mode,
    () => {
      if (editor.value) syncEditorTheme(editor.value)
    }
  )
  loadConfig()
  statusStore.refresh()
})

onBeforeUnmount(() => {
  editor.value?.destroy()
})
</script>

<template>
  <div class="page">
    <div class="editor-shell">
      <!-- 文件标签栏 -->
      <div class="editor-tabs">
        <div class="editor-tab active">
          <span class="tab-icon">{ }</span>
          <span>config.json</span>
          <span v-if="dirty" class="tab-dirty">●</span>
        </div>
        <div class="tab-path" :title="configPath">{{ configPath || '未加载' }}</div>
      </div>

      <!-- 工具栏 -->
      <div class="editor-toolbar">
        <button class="btn btn-ghost btn-xs" :disabled="loading" @click="loadConfig">刷新</button>
        <button class="btn btn-ghost btn-xs" @click="format">格式化</button>
        <button class="btn btn-ghost btn-xs" @click="openSearch">查找/替换</button>
        <span class="toolbar-spacer" />
        <span class="hint">原样编辑主配置（JSONC：注释/尾逗号保留；Ctrl+F 搜索、Ctrl+Shift+F 格式化）</span>
        <button class="btn btn-primary btn-xs" :disabled="saving" @click="save">保存配置</button>
      </div>

      <!-- 编辑器主体 -->
      <div ref="editorHost" class="editor-host" />

      <!-- 状态栏 -->
      <div class="editor-statusbar">
        <span class="sb-item sb-lang">JSONC</span>
        <span class="sb-item">UTF-8</span>
        <span class="sb-item">LF</span>
        <span class="sb-item">空格: 2</span>
        <span class="sb-spacer" />
        <span v-if="selected" class="sb-item">已选 {{ selected }}</span>
        <span class="sb-item">{{ docLines }} 行</span>
        <span class="sb-item">Ln {{ cursor.line }}, Col {{ cursor.col }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.editor-shell {
  border: 1px solid var(--cfg-shell-border);
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  height: calc(100vh - 130px);
  background: var(--cfg-shell-bg);
  box-shadow: 0 4px 16px var(--cfg-shadow);
}

/* 文件标签栏（VSCode 风格） */
.editor-tabs {
  display: flex;
  align-items: center;
  background: var(--cfg-tabbar-bg);
  border-bottom: 1px solid var(--cfg-shell-border);
  height: 36px;
}
.editor-tab {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 100%;
  padding: 0 14px;
  background: var(--cfg-tab-bg);
  color: var(--cfg-tab-color);
  font-size: 13px;
  border-right: 1px solid var(--cfg-shell-border);
  cursor: default;
  user-select: none;
}
.editor-tab .tab-icon {
  color: var(--cfg-dirty);
  font-weight: 600;
}
.tab-dirty {
  color: var(--cfg-dirty);
  font-size: 10px;
}
.tab-path {
  flex: 1;
  padding: 0 12px;
  font-size: 12px;
  color: var(--cfg-text-dim);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 工具栏 */
.editor-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  background: var(--cfg-toolbar-bg);
  border-bottom: 1px solid var(--cfg-shell-border);
}
.toolbar-spacer {
  flex: 1;
}
.hint {
  font-size: 12px;
  color: var(--cfg-text-dim);
}

/* 编辑器主体 */
.editor-host {
  flex: 1;
  overflow: hidden;
  text-align: left;
}
.editor-host :deep(.cm-editor) {
  height: 100%;
}
.editor-host :deep(.cm-editor.cm-focused) {
  outline: none;
}

/* 状态栏（VSCode 风格） */
.editor-statusbar {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 0 12px;
  height: 24px;
  background: var(--cfg-statusbar-bg);
  border-top: 1px solid var(--cfg-shell-border);
  font-size: 12px;
  color: var(--cfg-text-dim);
  user-select: none;
}
.sb-item {
  white-space: nowrap;
}
.sb-lang {
  color: var(--cfg-accent);
  font-weight: 600;
}
.sb-spacer {
  flex: 1;
}
</style>
