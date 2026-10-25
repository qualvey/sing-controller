<template>
  <div
    class="border-base-content/10 bg-base-100 flex flex-col overflow-hidden transition-all"
    :class="
      isFullscreen
        ? 'fixed inset-0 z-50 h-screen w-screen rounded-none shadow-2xl'
        : 'h-[calc(100vh-180px)] min-h-[520px] rounded-xl border shadow-xs'
    "
  >
    <!-- 文件标签栏 -->
    <div class="border-base-content/10 bg-base-200/70 flex items-center justify-between border-b px-3 py-1.5 text-xs select-none">
      <div class="flex items-center gap-2">
        <span class="bg-base-100 border-base-content/10 text-primary flex items-center gap-2 rounded-md border px-2.5 py-1 font-medium shadow-2xs">
          <span class="font-mono font-bold text-xs">{ }</span>
          <span class="font-mono">config.json</span>
          <span
            v-if="dirty"
            class="badge badge-error badge-xs p-0.5 text-[10px] font-semibold text-white animate-pulse"
            title="存在未保存的更改"
          >
            ●
          </span>
        </span>
        <span
          v-if="dirty"
          class="text-error/80 text-[11px] hidden sm:inline"
        >
          (未保存)
        </span>
      </div>

      <div class="flex items-center gap-2">
        <div
          class="text-base-content/50 font-mono text-[11px] truncate max-w-[200px] sm:max-w-[340px] md:max-w-[500px]"
          :title="configPath"
        >
          {{ configPath || '未加载' }}
        </div>
        <div class="h-3 w-px bg-base-content/20 hidden sm:block"></div>
        <!-- 全屏切换按钮 -->
        <button
          type="button"
          class="btn btn-ghost btn-xs btn-square text-base-content/70 hover:text-base-content"
          :title="isFullscreen ? '退出全屏 (Esc)' : '全屏编辑'"
          @click="toggleFullscreen"
        >
          <ArrowsPointingInIcon
            v-if="isFullscreen"
            class="h-4 w-4"
          />
          <ArrowsPointingOutIcon
            v-else
            class="h-4 w-4"
          />
        </button>
      </div>
    </div>

    <!-- 工具栏 -->
    <div class="border-base-content/10 bg-base-100 flex flex-wrap items-center justify-between gap-2 border-b px-3 py-1.5">
      <!-- 左侧操作组：刷新、格式化、查找、大纲折叠 -->
      <div class="flex flex-wrap items-center gap-1 sm:gap-1.5">
        <button
          type="button"
          class="btn btn-ghost btn-xs gap-1"
          :disabled="loading"
          title="重新加载磁盘配置文件"
          @click="loadConfig"
        >
          <ArrowPathIcon
            class="h-3.5 w-3.5"
            :class="{ 'animate-spin': loading }"
          />
          <span>刷新</span>
        </button>

        <button
          type="button"
          class="btn btn-ghost btn-xs gap-1"
          title="格式化排版 (Ctrl+Shift+F)"
          @click="format"
        >
          <SparklesIcon class="h-3.5 w-3.5" />
          <span>格式化</span>
        </button>

        <button
          type="button"
          class="btn btn-ghost btn-xs gap-1"
          title="查找 / 替换 (Ctrl+F)"
          @click="openSearch"
        >
          <MagnifyingGlassIcon class="h-3.5 w-3.5" />
          <span>查找</span>
        </button>

        <div class="h-3.5 w-px bg-base-content/15 mx-0.5"></div>

        <button
          type="button"
          class="btn btn-ghost btn-xs gap-1"
          title="折叠至一级大纲 (只折叠顶层 log, dns, inbounds...)"
          @click="foldLevel1"
        >
          <ChevronUpDownIcon class="h-3.5 w-3.5" />
          <span>折叠1级</span>
        </button>

        <button
          type="button"
          class="btn btn-ghost btn-xs gap-1"
          title="展开全部代码块 (Ctrl+Alt+])"
          @click="unfoldAllBlocks"
        >
          <ChevronDownIcon class="h-3.5 w-3.5" />
          <span>展开全部</span>
        </button>
      </div>

      <!-- 右侧操作组：快捷键帮助、放弃修改、保存配置 -->
      <div class="flex items-center gap-1.5">
        <!-- 快捷键指南下拉菜单 -->
        <div class="dropdown dropdown-end">
          <button
            tabindex="0"
            type="button"
            class="btn btn-ghost btn-xs gap-1 text-base-content/60 hover:text-base-content"
            title="查看快捷键"
          >
            <QuestionMarkCircleIcon class="h-3.5 w-3.5" />
            <span class="hidden md:inline">快捷键</span>
          </button>
          <div
            tabindex="0"
            class="dropdown-content bg-base-100 border-base-content/10 text-base-content z-30 w-72 rounded-xl border p-3 shadow-xl text-xs space-y-2.5"
          >
            <div class="font-semibold text-xs pb-1.5 border-b border-base-content/10 flex items-center justify-between">
              <span>编辑器快捷键</span>
              <span class="text-[10px] font-mono text-base-content/50">JSONC</span>
            </div>
            <div class="space-y-1.5 text-base-content/80">
              <div class="flex items-center justify-between">
                <span>保存配置</span>
                <kbd class="kbd kbd-xs font-mono">Ctrl + S</kbd>
              </div>
              <div class="flex items-center justify-between">
                <span>格式化代码</span>
                <kbd class="kbd kbd-xs font-mono">Ctrl + Shift + F</kbd>
              </div>
              <div class="flex items-center justify-between">
                <span>查找 / 替换</span>
                <kbd class="kbd kbd-xs font-mono">Ctrl + F</kbd>
              </div>
              <div class="flex items-center justify-between">
                <span>折叠1级大纲</span>
                <span class="text-[11px] text-base-content/60">工具栏点击</span>
              </div>
              <div class="flex items-center justify-between">
                <span>展开全部</span>
                <kbd class="kbd kbd-xs font-mono">Ctrl + Alt + ]</kbd>
              </div>
              <div class="flex items-center justify-between">
                <span>单行注释</span>
                <kbd class="kbd kbd-xs font-mono">Ctrl + /</kbd>
              </div>
              <div class="flex items-center justify-between">
                <span>块注释</span>
                <kbd class="kbd kbd-xs font-mono">Shift + Alt + A</kbd>
              </div>
              <div class="flex items-center justify-between">
                <span>补全建议</span>
                <kbd class="kbd kbd-xs font-mono">Tab / Enter</kbd>
              </div>
            </div>
            <div class="pt-1.5 border-t border-base-content/10 text-[11px] text-base-content/50 leading-relaxed">
              支持 JSONC 标准（行/块注释与尾逗号），保存时自动进行严格的 sing-box 格式校验。
            </div>
          </div>
        </div>

        <!-- 放弃修改（仅在 dirty 时显示） -->
        <button
          v-if="dirty"
          type="button"
          class="btn btn-ghost btn-xs text-error/80 hover:text-error gap-1 transition-all"
          title="放弃未保存的修改，还原到磁盘文件"
          @click="discardChanges"
        >
          <ArrowUturnLeftIcon class="h-3.5 w-3.5" />
          <span>放弃修改</span>
        </button>

        <!-- 保存配置主按钮 -->
        <button
          type="button"
          class="btn btn-primary btn-xs gap-1.5 shadow-2xs font-medium"
          :class="{ 'btn-active shadow-sm ring-1 ring-primary/40': dirty }"
          :disabled="saving"
          title="保存配置到磁盘 (Ctrl+S)"
          @click="save"
        >
          <CheckIcon
            v-if="!saving"
            class="h-3.5 w-3.5"
          />
          <ArrowPathIcon
            v-else
            class="h-3.5 w-3.5 animate-spin"
          />
          <span>保存配置</span>
        </button>
      </div>
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
        <span class="hidden sm:inline">UTF-8</span>
        <span class="hidden sm:inline">LF</span>
        <span class="hidden md:inline">空格: 2</span>
        <!-- 语法校验指示 -->
        <div class="flex items-center gap-1.5">
          <span
            v-if="lintErrorsCount === 0"
            class="text-success/90 flex items-center gap-1"
          >
            <span class="h-1.5 w-1.5 rounded-full bg-success"></span>
            <span>语法正常</span>
          </span>
          <span
            v-else
            class="text-error flex items-center gap-1 font-semibold"
            :title="lintErrorMessage"
          >
            <span class="h-1.5 w-1.5 rounded-full bg-error animate-pulse"></span>
            <span>{{ lintErrorsCount }} 处错误</span>
          </span>
        </div>
      </div>

      <div class="flex items-center gap-3">
        <span v-if="selected" class="text-primary font-medium">已选 {{ selected }}</span>
        <span>{{ docLines }} 行</span>
        <span>Ln {{ cursor.line }}, Col {{ cursor.col }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { api } from '@/api'
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
import { foldEffect, foldKeymap, syntaxTree, unfoldAll } from '@codemirror/language'
import { lintGutter } from '@codemirror/lint'
import { openSearchPanel } from '@codemirror/search'
import { Compartment, EditorState, Prec } from '@codemirror/state'
import { oneDark } from '@codemirror/theme-one-dark'
import { EditorView, keymap } from '@codemirror/view'
import { editorUiTheme } from '@/composables/useCmTheme'
import {
  ArrowPathIcon,
  ArrowsPointingInIcon,
  ArrowsPointingOutIcon,
  ArrowUturnLeftIcon,
  CheckIcon,
  ChevronDownIcon,
  ChevronUpDownIcon,
  MagnifyingGlassIcon,
  QuestionMarkCircleIcon,
  SparklesIcon,
} from '@heroicons/vue/24/outline'
import { basicSetup } from 'codemirror'
import { parseTree, printParseErrorCode } from 'jsonc-parser'
import type { ParseError } from 'jsonc-parser'
import { onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'

const statusStore = useStatusStore()
const themeStore = useThemeStore()

const editorHost = ref<HTMLElement>()
const editor = shallowRef<EditorView>()
const loading = ref(false)
const saving = ref(false)
const dirty = ref(false)
const configPath = ref('')
const isFullscreen = ref(false)
const lintErrorsCount = ref(0)
const lintErrorMessage = ref('')

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

const checkSyntax = (text: string) => {
  if (!text.trim()) {
    lintErrorsCount.value = 0
    lintErrorMessage.value = ''
    return
  }
  const parseErrors: ParseError[] = []
  parseTree(text, parseErrors, { allowTrailingComma: true })
  lintErrorsCount.value = parseErrors.length
  if (parseErrors.length > 0) {
    lintErrorMessage.value = printParseErrorCode(parseErrors[0].error)
  } else {
    lintErrorMessage.value = ''
  }
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
    let text = await api.configRaw()
    // 核心未运行时 controller 可能返回空内容，或因配置尚未加载返回结构化数据。
    // 始终给 CodeMirror 一个字符串，避免异常响应让编辑器保持空白。
    if (typeof text !== 'string') {
      text = JSON.stringify(text, null, 2)
    }
    if (!text || text.trim() === '' || text.trim() === '{}') {
      try {
        const fallback = await api.config()
        if (fallback && typeof fallback === 'object' && Object.keys(fallback).length > 0) {
          text = JSON.stringify(fallback, null, 2)
        }
      } catch {
        // 核心断开时回退接口也可能不可用，继续使用明确的空 JSON 文档。
      }
    }
    if (!text || text.trim() === '') text = '{}\n'
    configPath.value = statusStore.status?.config_path || ''
    editor.value?.dispatch({
      changes: { from: 0, to: editor.value.state.doc.length, insert: text },
    })
    if (editor.value) {
      updateStats(editor.value)
      checkSyntax(text)
    }
    dirty.value = false
  } catch (e) {
    showToast((e as Error).message || '加载配置失败', 'error')
  } finally {
    loading.value = false
  }
}

const discardChanges = async () => {
  await loadConfig()
  showToast('已放弃更改并重新载入', 'info')
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

const foldLevel1 = () => {
  const ed = editor.value
  if (!ed) return

  unfoldAll(ed)

  const tree = syntaxTree(ed.state)
  const effects: any[] = []

  let rootNode = tree.topNode
  let rootObj = rootNode.firstChild
  while (rootObj && rootObj.name !== 'Object') {
    rootObj = rootObj.nextSibling
  }

  if (rootObj) {
    for (let cur = rootObj.firstChild; cur; cur = cur.nextSibling) {
      if (cur.name === 'Prop') {
        for (let val = cur.firstChild; val; val = val.nextSibling) {
          if (val.name === 'Object' || val.name === 'Array') {
            const startLine = ed.state.doc.lineAt(val.from).number
            const endLine = ed.state.doc.lineAt(val.to).number
            if (endLine > startLine && val.to > val.from + 2) {
              effects.push(foldEffect.of({ from: val.from + 1, to: val.to - 1 }))
            }
          }
        }
      }
    }
  }

  if (effects.length === 0) {
    const text = ed.state.doc.toString()
    const parsed = parseTree(text, [], { allowTrailingComma: true })
    if (parsed && parsed.type === 'object' && parsed.children) {
      for (const prop of parsed.children) {
        if (prop.type === 'property' && prop.children && prop.children[1]) {
          const val = prop.children[1]
          if (val.type === 'object' || val.type === 'array') {
            const startLine = ed.state.doc.lineAt(val.offset).number
            const endLine = ed.state.doc.lineAt(val.offset + val.length).number
            if (endLine > startLine && val.length > 2) {
              effects.push(foldEffect.of({ from: val.offset + 1, to: val.offset + val.length - 1 }))
            }
          }
        }
      }
    }
  }

  if (effects.length > 0) {
    ed.dispatch({ effects })
    showToast(`已折叠 ${effects.length} 个一级配置大纲`, 'info')
  } else {
    showToast('未找到可折叠的一级配置项', 'info')
  }
}

const unfoldAllBlocks = () => {
  if (editor.value) {
    unfoldAll(editor.value)
    showToast('已展开全部代码块', 'info')
  }
}

const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
}

onMounted(() => {
  if (!editorHost.value) return
  const state = EditorState.create({
    doc: '{}',
    extensions: [
      basicSetup,
      jsonc(),
      themeExtension(),
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
        if (u.docChanged) {
          dirty.value = true
          checkSyntax(u.state.doc.toString())
        }
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
    if (e.key === 'Escape' && isFullscreen.value) {
      isFullscreen.value = false
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
