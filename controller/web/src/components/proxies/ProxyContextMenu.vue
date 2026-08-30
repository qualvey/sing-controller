<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  isProxyContextMenuVisible,
  proxyContextMenuPos,
  proxyContextMenuTarget,
  closeProxyContextMenu,
  openProxyEditModal
} from '@/composables/proxyContextMenu'
import { proxyLatencyTest, getTestUrl } from '@/assembly/proxies'
import { showToast } from '@/helper/toast'
import { BoltIcon, PencilSquareIcon, DocumentDuplicateIcon } from '@heroicons/vue/24/outline'

const { t } = useI18n()

const target = computed(() => proxyContextMenuTarget.value)

const menuStyle = computed(() => ({
  left: `${proxyContextMenuPos.value.x}px`,
  top: `${proxyContextMenuPos.value.y}px`
}))

const handleTestLatency = async () => {
  if (!target.value) return
  const { name, groupName } = target.value
  closeProxyContextMenu()
  try {
    await proxyLatencyTest(name, getTestUrl(groupName), undefined)
  } catch {
    // latency test failure is handled in assembly/proxies
  }
}

const handleEditNode = () => {
  if (!target.value) return
  openProxyEditModal(target.value.name)
}

const handleCopyName = async () => {
  if (!target.value) return
  const name = target.value.name
  closeProxyContextMenu()
  try {
    await navigator.clipboard.writeText(name)
    showToast(t('copySuccess') || '复制成功', 'success')
  } catch {
    // 降级处理
    const textArea = document.createElement('textarea')
    textArea.value = name
    document.body.appendChild(textArea)
    textArea.select()
    try {
      document.execCommand('copy')
      showToast(t('copySuccess') || '复制成功', 'success')
    } catch {
      showToast('复制失败', 'error')
    }
    document.body.removeChild(textArea)
  }
}

const handleGlobalPointerDown = (e: MouseEvent | TouchEvent) => {
  if (!isProxyContextMenuVisible.value) return
  const el = document.getElementById('proxy-context-menu')
  if (el && !el.contains(e.target as Node)) {
    closeProxyContextMenu()
  }
}

const handleKeyDown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && isProxyContextMenuVisible.value) {
    closeProxyContextMenu()
  }
}

const handleScroll = () => {
  if (isProxyContextMenuVisible.value) {
    closeProxyContextMenu()
  }
}

watch(isProxyContextMenuVisible, (visible) => {
  if (visible) {
    window.addEventListener('scroll', handleScroll, { capture: true, passive: true })
  } else {
    window.removeEventListener('scroll', handleScroll, { capture: true })
  }
})

onMounted(() => {
  window.addEventListener('pointerdown', handleGlobalPointerDown)
  window.addEventListener('keydown', handleKeyDown)
})

onBeforeUnmount(() => {
  window.removeEventListener('pointerdown', handleGlobalPointerDown)
  window.removeEventListener('keydown', handleKeyDown)
  window.removeEventListener('scroll', handleScroll, { capture: true })
})
</script>

<template>
  <Teleport to="body">
    <Transition name="fade-scale">
      <div
        v-if="isProxyContextMenuVisible && target"
        id="proxy-context-menu"
        class="fixed z-[9999] min-w-[150px] rounded-lg border border-base-300/80 bg-base-100/95 p-1 text-sm shadow-xl backdrop-blur-md dark:border-base-content/10"
        :style="menuStyle"
        @contextmenu.prevent.stop
      >
        <div class="px-2.5 py-1.5 text-[11px] font-medium text-base-content/50 truncate max-w-[200px]">
          {{ target.name }}
        </div>
        <div class="h-px bg-base-200 dark:bg-base-300 my-0.5" />
        <ul class="flex flex-col gap-0.5">
          <li>
            <button
              class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-base-content hover:bg-primary/10 hover:text-primary transition-colors text-left"
              @click="handleTestLatency"
            >
              <BoltIcon class="h-4 w-4 shrink-0" />
              <span>{{ t('testLatency') }}</span>
            </button>
          </li>
          <li>
            <button
              class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-base-content hover:bg-primary/10 hover:text-primary transition-colors text-left"
              @click="handleEditNode"
            >
              <PencilSquareIcon class="h-4 w-4 shrink-0" />
              <span>{{ t('editNode') }}</span>
            </button>
          </li>
          <li>
            <button
              class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-base-content hover:bg-primary/10 hover:text-primary transition-colors text-left"
              @click="handleCopyName"
            >
              <DocumentDuplicateIcon class="h-4 w-4 shrink-0" />
              <span>{{ t('copyNodeName') }}</span>
            </button>
          </li>
        </ul>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.fade-scale-enter-active,
.fade-scale-leave-active {
  transition: all 0.12s ease-out;
}

.fade-scale-enter-from,
.fade-scale-leave-to {
  opacity: 0;
  transform: scale(0.95);
}
</style>
