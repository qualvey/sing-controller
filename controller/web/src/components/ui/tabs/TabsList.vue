<script setup lang="ts">
import { cn } from '@/lib/utils'
import { useMutationObserver, useResizeObserver } from '@vueuse/core'
import { TabsList, type TabsListProps, useForwardProps } from 'reka-ui'
import { computed, nextTick, onMounted, ref, type ComponentPublicInstance, type HTMLAttributes } from 'vue'

const props = defineProps<TabsListProps & { class?: HTMLAttributes['class'] }>()
const forwarded = useForwardProps(props)

const listRef = ref<ComponentPublicInstance | HTMLElement | null>(null)
const indicatorLeft = ref(0)
const indicatorWidth = ref(0)
const ready = ref(false)

const indicatorStyle = computed(() => ({
  transform: `translateX(${indicatorLeft.value}px)`,
  width: `${indicatorWidth.value}px`,
}))

function getRootElement(): HTMLElement | null {
  if (!listRef.value) return null
  if ('$el' in listRef.value && listRef.value.$el instanceof HTMLElement) {
    return listRef.value.$el
  }
  if (listRef.value instanceof HTMLElement) {
    return listRef.value
  }
  return null
}

function updateIndicator() {
  const root = getRootElement()
  if (!root) return
  const activeTrigger = root.querySelector('[data-state="active"]') as HTMLElement | null
  if (!activeTrigger) return

  indicatorLeft.value = activeTrigger.offsetLeft
  indicatorWidth.value = activeTrigger.offsetWidth
  if (!ready.value && indicatorWidth.value > 0) {
    requestAnimationFrame(() => {
      ready.value = true
    })
  }
}

onMounted(() => {
  void nextTick(() => {
    updateIndicator()
  })
  const root = getRootElement()
  if (root) {
    useMutationObserver(
      root,
      () => {
        updateIndicator()
      },
      {
        attributes: true,
        attributeFilter: ['data-state'],
        subtree: true,
      },
    )
    useResizeObserver(root, () => {
      updateIndicator()
    })
  }
})
</script>

<template>
  <TabsList
    ref="listRef"
    v-bind="forwarded"
    :class="cn(
      'bg-muted text-muted-foreground relative inline-flex h-9 w-max items-center justify-center rounded-lg p-1',
      props.class
    )"
  >
    <!-- 滑动背景指示条 -->
    <div
      class="tab-sliding-indicator bg-background pointer-events-none absolute top-1 bottom-1 left-0 rounded-md shadow-xs"
      :class="ready ? 'tab-indicator-ready' : 'opacity-0'"
      :style="indicatorStyle"
    />
    <slot />
  </TabsList>
</template>

<style scoped>
.tab-indicator-ready {
  transition: transform 0.25s cubic-bezier(0.32, 0.72, 0, 1), width 0.2s cubic-bezier(0.32, 0.72, 0, 1);
}
@media (prefers-reduced-motion: reduce) {
  .tab-indicator-ready {
    transition: none;
  }
}
</style>
