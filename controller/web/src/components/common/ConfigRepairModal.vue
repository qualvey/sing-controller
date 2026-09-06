<template>
  <Teleport to="body">
    <Transition name="repair-modal">
      <div
        v-if="modelValue"
        class="fixed inset-0 z-[100050] flex items-center justify-center p-2 sm:p-4 bg-black/60 backdrop-blur-sm overflow-hidden"
        @keydown.escape="handleClose"
      >
        <div
          class="border-base-border bg-base-100 flex flex-col w-[1100px] max-w-full h-[92vh] max-h-[900px] rounded-xl border shadow-2xl overflow-hidden"
        >
          <!-- 弹窗顶部标题栏 -->
          <div class="border-base-border bg-base-200/70 flex items-center justify-between border-b px-4 py-2.5 text-xs select-none">
            <div class="flex items-center gap-2 min-w-0">
              <span class="bg-warning/15 text-warning flex h-7 w-7 items-center justify-center rounded-lg">
                <WrenchScrewdriverIcon class="h-4 w-4" />
              </span>
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <h2 class="text-sm font-bold leading-none">修复主配置 (config.json)</h2>
                  <span class="inline-flex items-center gap-1 rounded bg-success/15 px-1.5 py-0.5 text-[10px] font-medium text-success">
                    <span class="h-1.5 w-1.5 rounded-full bg-success"></span>
                    Controller 在线
                  </span>
                  <span
                    v-if="dirty"
                    class="text-error font-medium text-[11px]"
                  >
                    ● 未保存
                  </span>
                </div>
                <div
                  class="text-base-content/50 font-mono text-[11px] truncate mt-0.5"
                  :title="configPath"
                >
                  {{ configPath || '未获取到路径' }}
                </div>
              </div>
            </div>
            <button
              type="button"
              class="btn btn-ghost btn-circle btn-xs text-base-content/60 hover:text-base-content"
              aria-label="关闭"
              @click="handleClose"
            >
              <XMarkIcon class="h-4 w-4" />
            </button>
          </div>

          <!-- 错误与崩溃信息辅助折叠栏 -->
          <div
            v-if="errorDetail || (recentLogs && recentLogs.length > 0)"
            class="border-base-border bg-error/5 border-b px-4 py-2 text-xs"
          >
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2 text-error font-medium">
                <ExclamationTriangleIcon class="h-4 w-4 shrink-0" />
                <span class="truncate">参考报错：{{ errorDetail || '核心异常退出' }}</span>
              </div>
              <button
                type="button"
                class="btn btn-ghost btn-xs text-error/80 hover:text-error h-6 min-h-0 px-2 font-normal"
                @click="showErrorDetail = !showErrorDetail"
              >
                {{ showErrorDetail ? '收起报错 ▲' : '查看详情 ▼' }}
              </button>
            </div>

            <!-- 展开的报错日志 -->
            <div
              v-if="showErrorDetail"
              class="mt-2 flex flex-col gap-1"
            >
              <div
                v-if="errorDetail"
                class="bg-error/10 text-error rounded p-2 text-[11px] font-mono leading-relaxed select-all break-all"
              >
                {{ errorDetail }}
              </div>
              <div
                v-if="recentLogs && recentLogs.length > 0"
                class="bg-base-300 text-base-content/90 max-h-32 overflow-y-auto rounded p-2 text-[11px] font-mono leading-relaxed select-all"
              >
                <div
                  v-for="(line, idx) in recentLogs"
                  :key="idx"
                  class="break-all"
                  :class="{
                    'text-error': line.source === 'stderr' || line.message.includes('panic') || line.message.includes('fatal') || line.message.includes('error'),
                    'text-warning': line.source === 'system',
                    'text-base-content/80': line.source === 'stdout',
                  }"
                >
                  <span class="opacity-40">[{{ formatLogTime(line.timestamp) }}]</span>
                  <span class="opacity-60 ml-1">[{{ line.source }}]</span>
                  {{ line.message }}
                </div>
              </div>
            </div>
          </div>

          <!-- 工具栏 -->
          <div class="border-base-border bg-base-100 flex flex-wrap items-center gap-2 border-b px-4 py-2 text-xs">
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
              重新载入
            </button>
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              @click="format"
            >
              格式化 (保留注释)
            </button>
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              @click="openSearch"
            >
              查找 / 替换
            </button>

            <div class="flex-1"></div>

            <span class="text-base-content/50 hidden md:inline text-[11px]">
              支持 JSONC（注释与尾逗号）；Ctrl+S 直接保存并重启核心
            </span>
          </div>

          <!-- 编辑器主体 -->
          <div
            ref="editorHost"
            class="flex-1 min-h-0 overflow-hidden"
          />

          <!-- 状态栏 -->
          <div class="border-base-border bg-base-200/50 text-base-content/60 flex items-center justify-between border-t px-4 py-1 font-mono text-[11px] select-none">
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

          <!-- 弹窗底部操作按钮 -->
          <div class="border-base-border bg-base-200/70 flex items-center justify-between border-t px-4 py-3">
            <button
              type="button"
              class="btn btn-ghost btn-sm"
              :disabled="saving || restarting"
              @click="handleClose"
            >
              关闭
            </button>

            <div class="flex items-center gap-2">
              <button
                type="button"
                class="btn btn-outline btn-sm"
                :disabled="saving || restarting || loading"
                @click="saveOnly"
              >
                <span
                  v-if="saving && !restarting"
                  class="loading loading-spinner loading-xs"
                ></span>
                仅保存配置
              </button>
              <button
                type="button"
                class="btn btn-primary btn-sm gap-1.5"
                :disabled="saving || restarting || loading"
                @click="saveAndRestart"
              >
                <span
                  v-if="restarting"
                  class="loading loading-spinner loading-xs"
                ></span>
                <PlayIcon
                  v-else
                  class="h-4 w-4"
                />
                {{ restarting ? '正在保存并重启...' : '保存并重启核心' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { api, type CoreLogLine } from '@/api'
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
import { foldGutter, foldKeymap } from '@codemirror/language'
import { lintGutter } from '@codemirror/lint'
import { openSearchPanel } from '@codemirror/search'
import { Compartment, EditorState, Prec } from '@codemirror/state'
import { oneDark } from '@codemirror/theme-one-dark'
import { editorUiTheme } from '@/composables/useCmTheme'
import {
  ArrowPathIcon,
  ExclamationTriangleIcon,
  PlayIcon,
  WrenchScrewdriverIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'
import { basicSetup } from 'codemirror'
import dayjs from 'dayjs'
import { computed, nextTick, onBeforeUnmount, ref, shallowRef, watch } from 'vue'

const props = defineProps<{
  modelValue: boolean
  errorDetail?: string
  recentLogs?: CoreLogLine[]
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'restarted'): void
}>()

const statusStore = useStatusStore()
const themeStore = useThemeStore()

const editorHost = ref<HTMLElement>()
const editor = shallowRef<EditorView>()
const loading = ref(false)
const saving = ref(false)
const restarting = ref(false)
const dirty = ref(false)
const showErrorDetail = ref(false)

const configPath = computed(() => statusStore.status?.config_path || '')

const cursor = ref({ line: 1, col: 1 })
const selected = ref(0)
const docLines = ref(1)

const formatLogTime = (ts: string) => {
  if (!ts) return ''
  return dayjs(ts).format('HH:mm:ss.SSS')
}

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

const openSearch = () => {
  editor.value && openSearchPanel(editor.value)
}

const validateAndGetText = (): string | null => {
  const text = editor.value?.state.doc.toString() ?? ''
  const parsed = parseJsonc(text)
  if (!parsed.ok) {
    showToast(`配置格式错误：${parsed.message}`, 'error')
    return null
  }
  if (typeof parsed.value !== 'object' || parsed.value === null || Array.isArray(parsed.value)) {
    showToast('配置必须是 JSON 对象', 'error')
    return null
  }
  return text
}

const saveOnly = async (): Promise<boolean> => {
  const text = validateAndGetText()
  if (text === null) return false

  saving.value = true
  try {
    const res = (await api.saveConfigRaw(text)) as { load_error?: string; message?: string }
    if (res.load_error) {
      showToast(res.message || `配置已保存，但加载校验有误：${res.load_error}`, 'warning')
    } else {
      showToast('配置已保存（已写入磁盘）', 'success')
    }
    dirty.value = false
    await statusStore.refresh()
    return true
  } catch (e) {
    showToast((e as Error).message || '保存配置失败', 'error')
    return false
  } finally {
    saving.value = false
  }
}

const saveAndRestart = async () => {
  const saved = await saveOnly()
  if (!saved) return

  restarting.value = true
  try {
    await api.coreRestart()
    showToast('配置保存成功，已发送核心重启指令', 'success')
    await statusStore.refresh()
    emit('restarted')
    emit('update:modelValue', false)
  } catch (e) {
    showToast((e as Error).message || '重启核心失败，请查看日志核对错误', 'error')
    await statusStore.refresh()
  } finally {
    restarting.value = false
  }
}

const handleClose = () => {
  if (saving.value || restarting.value) return
  if (dirty.value) {
    if (!confirm('存在未保存的配置更改，确定要关闭吗？')) {
      return
    }
  }
  emit('update:modelValue', false)
}

const initEditor = () => {
  if (!editorHost.value) return
  if (editor.value) {
    editor.value.destroy()
    editor.value = undefined
  }

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
              void saveAndRestart()
              return true
            },
          },
          { key: 'Tab', run: acceptCompletion },
          {
            key: 'Enter',
            run: (view) => {
              closeCompletion(view)
              return insertNewlineAndIndent(view)
            },
          },
          { key: 'Mod-Shift-f', run: formatJsoncDoc },
          { key: 'Mod-/', run: toggleComment },
          { key: 'Shift-Alt-a', run: toggleBlockComment },
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
  void loadConfig()
}

const handleGlobalKeyDown = (e: KeyboardEvent) => {
  if (!props.modelValue) return
  if ((e.ctrlKey || e.metaKey) && (e.key.toLowerCase() === 's' || e.code === 'KeyS')) {
    e.preventDefault()
    e.stopPropagation()
    void saveAndRestart()
  }
}

watch(
  () => props.modelValue,
  (val) => {
    if (val) {
      nextTick(() => {
        initEditor()
      })
      window.addEventListener('keydown', handleGlobalKeyDown, { capture: true })
    } else {
      window.removeEventListener('keydown', handleGlobalKeyDown, { capture: true } as any)
      if (editor.value) {
        editor.value.destroy()
        editor.value = undefined
      }
      dirty.value = false
    }
  },
  { immediate: true },
)

watch(
  () => themeStore.mode,
  () => {
    if (editor.value) syncEditorTheme(editor.value)
  },
)

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleGlobalKeyDown, { capture: true } as any)
  editor.value?.destroy()
})
</script>

<style scoped>
.repair-modal-enter-active,
.repair-modal-leave-active {
  transition: opacity 0.25s ease;
}

.repair-modal-enter-from,
.repair-modal-leave-to {
  opacity: 0;
}
</style>
