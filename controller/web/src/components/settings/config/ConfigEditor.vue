<template>
  <div class="border-base-content/10 bg-base-100 flex h-[calc(100vh-180px)] min-h-[500px] flex-col overflow-hidden rounded-xl border shadow-xs">
    <!-- 文件标签栏 -->
    <div class="border-base-content/10 bg-base-200/60 flex items-center justify-between border-b px-3 py-1.5 text-xs select-none">
      <div class="flex items-center gap-2">
        <span class="bg-base-100 border-base-content/10 text-primary flex items-center gap-1.5 rounded-md border px-2.5 py-1 font-medium shadow-2xs">
          <span class="font-mono font-bold text-xs">{ }</span>
          <span>config.json</span>
          <span
            v-if="dirty"
            class="text-error text-[10px]"
            title="未保存的更改"
          >
            ●
          </span>
        </span>
      </div>
      <div
        class="text-base-content/50 font-mono text-[11px] truncate max-w-[240px] md:max-w-[480px]"
        :title="configPath"
      >
        {{ configPath || '未加载' }}
      </div>
    </div>

    <!-- 工具栏 -->
    <div class="border-base-content/10 bg-base-100 flex flex-wrap items-center gap-2 border-b px-3 py-2">
      <button
        type="button"
        class="btn btn-ghost btn-xs gap-1"
        :disabled="loading"
        @click="loadConfig"
      >
        <ArrowPathIcon
          class="h-3.5 w-3.5"
          :class="{ 'animate-spin': loading }"
        />
        刷新
      </button>
      <button
        type="button"
        class="btn btn-ghost btn-xs"
        @click="format"
      >
        格式化
      </button>
      <button
        type="button"
        class="btn btn-ghost btn-xs"
        @click="openSearch"
      >
        查找 / 替换
      </button>
      <button
        type="button"
        class="btn btn-ghost btn-xs"
        title="折叠全部代码块 (Ctrl+Alt+[)"
        @click="foldAllBlocks"
      >
        折叠全部
      </button>
      <button
        type="button"
        class="btn btn-ghost btn-xs"
        title="展开全部代码块 (Ctrl+Alt+])"
        @click="unfoldAllBlocks"
      >
        展开全部
      </button>
      <div class="flex-1"></div>
      <span class="text-base-content/50 hidden text-xs lg:inline">
        原样编辑主配置（JSONC：支持注释与尾逗号；Ctrl+S 保存、Ctrl+/ 注释、Ctrl+F 查找、Ctrl+Shift+F 格式化）
      </span>
      <button
        type="button"
        class="btn btn-outline btn-primary btn-xs gap-1"
        :disabled="reloading"
        title="热重载配置到内核（不中断已有网络连接）"
        @click="reloadConfig"
      >
        <ArrowPathIcon
          class="h-3.5 w-3.5"
          :class="{ 'animate-spin': reloading }"
        />
        重载配置
      </button>
      <button
        type="button"
        class="btn btn-primary btn-xs"
        :disabled="saving"
        @click="save"
      >
        保存配置
      </button>
    </div>

    <!-- 编辑器主体 -->
    <div
      ref="editorHost"
      class="editor-host flex-1 min-h-0 overflow-hidden"
    />

    <!-- 状态栏 -->
    <div class="border-base-content/10 bg-base-200/60 text-base-content/60 flex items-center justify-between border-t px-3 py-1 font-mono text-[11px] select-none">
      <div class="flex items-center gap-3">
        <span class="bg-primary/10 text-primary rounded px-1.5 py-0.5 font-semibold">JSONC</span>
        <span>UTF-8</span>
        <span>LF</span>
        <span>空格: 2</span>
      </div>
      <div class="flex items-center gap-3">
        <span v-if="selected">已选 {{ selected }}</span>
        <span>{{ docLines }} 行</span>
        <span>Ln {{ cursor.line }}, Col {{ cursor.col }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { api } from '@/api'
import { reloadConfigsAPI } from '@/assembly/config'
import { formatJsoncDoc, jsonc, jsoncLinter, parseJsonc } from '@/editor/jsonc'
import { showToast } from '@/helper/toast'
import { useStatusStore } from '@/store/status'
import { useThemeStore } from '@/store/theme'
import {
  acceptCompletion,
  closeCompletion,
  moveCompletionSelection,
  startCompletion,
} from '@codemirror/autocomplete'
import {
  defaultKeymap,
  indentWithTab,
  insertNewlineAndIndent,
  toggleBlockComment,
  toggleComment,
} from '@codemirror/commands'
import { foldAll, foldGutter, foldKeymap, unfoldAll } from '@codemirror/language'
import { lintGutter } from '@codemirror/lint'
import { openSearchPanel } from '@codemirror/search'
import { Compartment, EditorState, Prec } from '@codemirror/state'
import { oneDark } from '@codemirror/theme-one-dark'
import { EditorView, keymap } from '@codemirror/view'
import { editorUiTheme } from '@/composables/useCmTheme'
import { ArrowPathIcon } from '@heroicons/vue/24/outline'
import { basicSetup } from 'codemirror'
import { onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'

const statusStore = useStatusStore()
const themeStore = useThemeStore()

const editorHost = ref<HTMLElement>()
const editor = shallowRef<EditorView>()
const loading = ref(false)
const saving = ref(false)
const reloading = ref(false)
const dirty = ref(false)
const configPath = ref('')

const cursor = ref({ line: 1, col: 1 })
const selected = ref(0)
const docLines = ref(1)

const themeComp = new Compartment()
const themeExtension = () =>
  themeComp.of(themeStore.mode === 'dark' ? [oneDark, editorUiTheme(true)] : [editorUiTheme(false)])

const syncEditorTheme = (view: EditorView) => {
  view.dispatch({
    effects: themeComp.reconfigure(
      themeStore.mode === 'dark' ? [oneDark, editorUiTheme(true)] : [editorUiTheme(false)],
    ),
  })
}

const updateStats = (view: EditorView) => {
  const pos = view.state.selection.main.head
  const line = view.state.doc.lineAt(pos)
  cursor.value = { line: line.number, col: pos - line.from + 1 }
  const sel = view.state.selection.main
  selected.value = Math.max(sel.from, sel.to) - Math.min(sel.from, sel.to)
  docLines.value = view.state.doc.lines
}

const loadConfig = async () => {
  loading.value = true
  try {
    const text = await api.configRaw()
    configPath.value = statusStore.status?.config_path || ''
    editor.value?.dispatch({
      changes: { from: 0, to: editor.value.state.doc.length, insert: text },
    })
    if (editor.value) updateStats(editor.value)
    dirty.value = false
  } catch (e) {
    showToast((e as Error).message || '加载配置失败', 'error')
  } finally {
    loading.value = false
  }
}

const format = () => {
  const ed = editor.value
  if (!ed) return
  if (formatJsoncDoc(ed)) {
    showToast('已格式化（保留注释）', 'success')
  }
}

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

const reloadConfig = async () => {
  reloading.value = true
  try {
    await reloadConfigsAPI()
    showToast('配置重载成功', 'success')
    await statusStore.refresh()
  } catch (e) {
    showToast((e as Error).message || '重载配置失败', 'error')
  } finally {
    reloading.value = false
  }
}

const foldAllBlocks = () => {
  if (editor.value) foldAll(editor.value)
}

const unfoldAllBlocks = () => {
  if (editor.value) unfoldAll(editor.value)
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
      Prec.highest(
        keymap.of([
          {
            key: 'Mod-s',
            run: () => {
              void save()
              return true
            },
          },
          {
            key: 'Tab',
            run: acceptCompletion,
          },
          {
            key: 'Enter',
            run: (view) => {
              closeCompletion(view)
              return insertNewlineAndIndent(view)
            },
          },
          {
            key: 'Mod-Shift-f',
            run: (view) => {
              formatJsoncDoc(view)
              return true
            },
          },
          {
            key: 'Mod-/',
            run: toggleComment,
          },
          {
            key: 'Shift-Alt-a',
            run: toggleBlockComment,
          },
        ]),
      ),
      keymap.of([
        { key: 'Ctrl-Space', run: startCompletion },
        { key: 'Escape', run: closeCompletion },
        { key: 'ArrowDown', run: moveCompletionSelection(true) },
        { key: 'ArrowUp', run: moveCompletionSelection(false) },
        { key: 'PageDown', run: moveCompletionSelection(true, 'page') },
        { key: 'PageUp', run: moveCompletionSelection(false, 'page') },
        indentWithTab,
        ...foldKeymap,
        ...defaultKeymap,
      ]),
      EditorView.lineWrapping,
      EditorView.updateListener.of((u) => {
        if (u.docChanged || u.selectionSet || u.focusChanged) updateStats(u.view)
        if (u.docChanged) dirty.value = true
      }),
    ] as any,
  })
  editor.value = new EditorView({ state, parent: editorHost.value })

  const handleGlobalKeyDown = (e: KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && (e.key.toLowerCase() === 's' || e.code === 'KeyS')) {
      e.preventDefault()
      e.stopPropagation()
      void save()
    }
  }
  window.addEventListener('keydown', handleGlobalKeyDown, { capture: true })

  watch(
    () => themeStore.mode,
    () => {
      if (editor.value) syncEditorTheme(editor.value)
    },
  )
  void loadConfig()
  void statusStore.refresh()

  onBeforeUnmount(() => {
    window.removeEventListener('keydown', handleGlobalKeyDown, { capture: true } as any)
    editor.value?.destroy()
  })
})
</script>
