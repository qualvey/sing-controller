<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { EditorView, keymap } from '@codemirror/view'
import { EditorState } from '@codemirror/state'
import { basicSetup } from 'codemirror'
import { jsonc, jsoncLinter } from '../editor/jsonc'
import { defaultKeymap, indentWithTab, insertNewlineAndIndent } from '@codemirror/commands'
import { acceptCompletion } from '@codemirror/autocomplete'
import { foldGutter } from '@codemirror/language'
import { useCmTheme } from '../composables/useCmTheme'

const { ext: themeExt, watchTheme } = useCmTheme()

// 编辑 dialog 内的「源码」tab：显示/编辑单个资源对象 JSON（JSONC）
// 父组件保存时：isDirty() 为 true 则用 getText() 解析作为提交体，否则用表单构建结果
const props = withDefaults(
  defineProps<{
    initial: string
    height?: number | string
    minHeight?: number | string
  }>(),
  {
    minHeight: 240,
  }
)
const emit = defineEmits<{ update: [] }>()

const editorHost = ref<HTMLElement>()
const editor = shallowRef<EditorView>()
const dirty = ref(false)
// 程序化替换（initial 变更）期间禁止标记 dirty：否则 watch 自身 dispatch 会触发
// updateListener → docChanged → dirty=true 永久卡住，后续切换编辑对象时源码不刷新
// （表现为"源码永远指向第一个项目"，且保存时误用上一项的源码覆盖当前项）
let applyingInitial = false

const resolvedHeight = computed(() => {
  if (props.height !== undefined && props.height !== null && props.height !== '') {
    return typeof props.height === 'number' ? `${props.height}px` : String(props.height)
  }
  return '440px'
})

const resolvedMinHeight = computed(() => {
  if (props.minHeight !== undefined && props.minHeight !== null && props.minHeight !== '') {
    return typeof props.minHeight === 'number' ? `${props.minHeight}px` : String(props.minHeight)
  }
  return '240px'
})

watch(
  () => props.initial,
  (v) => {
    // 切换编辑对象：丢弃上一项的编辑状态，强制刷新为当前项源码
    dirty.value = false
    if (!editor.value) return
    applyingInitial = true
    editor.value.dispatch({ changes: { from: 0, to: editor.value.state.doc.length, insert: v } })
    applyingInitial = false
  }
)

onMounted(() => {
  if (!editorHost.value) return
  const state = EditorState.create({
    doc: props.initial,
    extensions: [
      basicSetup,
      jsonc(),
      themeExt,
      foldGutter(),
      jsoncLinter(),
      keymap.of([
        { key: 'Tab', run: acceptCompletion },
        indentWithTab,
        { key: 'Enter', run: insertNewlineAndIndent },
        ...defaultKeymap
      ]),
      EditorView.updateListener.of((u) => {
        if (u.docChanged && !applyingInitial && !dirty.value) {
          dirty.value = true
          emit('update')
        }
      }),
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
})

onBeforeUnmount(() => {
  editor.value?.destroy()
})

function getText(): string {
  return editor.value?.state.doc.toString() ?? props.initial
}

function isDirty(): boolean {
  return dirty.value
}

defineExpose({ getText, isDirty })
</script>

<template>
  <div
    class="flex flex-col gap-1.5 w-full"
    :style="{ height: resolvedHeight, minHeight: resolvedMinHeight }"
  >
    <div v-if="dirty" class="text-xs text-warning leading-relaxed shrink-0">
      {{ $t('sourceDirtyHint') }}
    </div>
    <div
      ref="editorHost"
      class="border border-base-300 rounded-md overflow-hidden text-left bg-base-100 flex-1 min-h-0 w-full"
    />
  </div>
</template>

<style scoped>
:deep(.cm-editor) {
  height: 100%;
}
:deep(.cm-scroller) {
  overflow: auto;
}
</style>
