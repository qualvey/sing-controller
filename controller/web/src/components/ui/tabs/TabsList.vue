<script setup lang="ts">
import { cn } from '@/lib/utils'
import { useMutationObserver, useResizeObserver } from '@vueuse/core'
import { computed, nextTick, onMounted, ref, type HTMLAttributes } from 'vue'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()

const listRef = ref<HTMLElement | null>(null)
const indicatorLeft = ref(0)
const indicatorWidth = ref(0)
const ready = ref(false)

const indicatorStyle = computed(() => ({
  transform: `translateX(${indicatorLeft.value}px)`,
  width: `${indicatorWidth.value}px`,
}))

function updateIndicator() {
  const root = listRef.value
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
  const root = listRef.value
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
  <div
    ref="listRef"
    role="tablist"
    :class="cn(
      'bg-base-300/50 relative inline-flex self-start h-9 items-center justify-center rounded-[0.625rem] p-[3px]',
      props.class
    )"
  >
    <!-- 滑动背景指示条 -->
    <div
      class="tab-sliding-indicator bg-base-100 ring-base-content/5 pointer-events-none absolute top-[3px] bottom-[3px] left-0 rounded-[0.5rem] shadow-sm ring-1"
      :class="ready ? 'tab-indicator-ready' : 'opacity-0'"
      :style="indicatorStyle"
    />
    <slot />
  </div>
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
