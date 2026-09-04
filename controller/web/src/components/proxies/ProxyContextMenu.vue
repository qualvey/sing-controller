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
import { proxyLatencyTest, proxyGroupLatencyTest, getTestUrl, fetchProxies } from '@/assembly/proxies'
import { api } from '@/api'
import { showToast } from '@/helper/toast'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { useStatusStore } from '@/store/status'
import { BoltIcon, PencilSquareIcon, DocumentDuplicateIcon, TrashIcon, StarIcon } from '@heroicons/vue/24/outline'

const { t } = useI18n()
const statusStore = useStatusStore()

const target = computed(() => proxyContextMenuTarget.value)

const menuStyle = computed(() => ({
  left: `${proxyContextMenuPos.value.x}px`,
  top: `${proxyContextMenuPos.value.y}px`
}))

const handleSetAsDefault = async () => {
  if (!target.value || !target.value.groupName) return
  const { name, groupName } = target.value
  closeProxyContextMenu()

  try {
    const group = await api.getOutbound(groupName)
    const outboundsList = (group as any)?.outbounds
    if (!group || !Array.isArray(outboundsList)) {
      showToast('当前分组类型不支持调整节点排序', 'warning')
      return
    }
    const filtered = outboundsList.filter((t: string) => t !== name)
    const newOutbounds = [name, ...filtered]
    ;(group as any).outbounds = newOutbounds

    await api.updateOutbound(groupName, group as any)
    showToast(`已将「${name}」置为分组「${groupName}」的首选默认节点`, 'success')
    await fetchProxies(true)
  } catch (e) {
    showToast((e as Error).message || '设置默认节点失败', 'error')
  }
}

const handleTestLatency = async () => {
  if (!target.value) return
  const { name, groupName, isGroup } = target.value
  closeProxyContextMenu()
  try {
    if (isGroup) {
      await proxyGroupLatencyTest(name)
    } else {
      await proxyLatencyTest(name, getTestUrl(groupName))
    }
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

const handleDeleteNode = async () => {
  if (!target.value) return
  const { name, isGroup } = target.value
  closeProxyContextMenu()

  const itemType = isGroup ? '分组' : '节点'
  const first = await showConfirmDialog({
    title: t('confirmDelete') || '删除确认',
    message: `确定删除${itemType}「${name}」？该操作不可恢复。`,
    confirmText: t('delete') || '删除',
    confirmButtonClass: 'btn-error'
  })
  if (!first.confirmed) return

  try {
    await api.deleteOutbound(name)
    showToast(`${itemType}删除成功`, 'success')
    await fetchProxies(true)
    try {
      await statusStore.refresh()
    } catch {
      // ignore
    }
  } catch (e) {
    const err = e as Error & { references?: string[] }
    if (err.references?.length) {
      const second = await showConfirmDialog({
        title: '被引用确认',
        message: `该${itemType}被以下对象引用：${err.references.join('、')}\n强制删除将自动从相关配置中移除引用。确认删除？`,
        confirmText: '强制删除',
        confirmButtonClass: 'btn-error'
      })
      if (!second.confirmed) return
      try {
        await api.deleteOutbound(name, true)
        showToast(`${itemType}删除成功（已移除引用）`, 'success')
        await fetchProxies(true)
        try {
          await statusStore.refresh()
        } catch {
          // ignore
        }
      } catch (e2) {
        showToast((e2 as Error).message || '删除失败', 'error')
      }
      return
    }
    showToast(err.message || '删除失败', 'error')
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
          {{ target.isGroup ? `[分组] ${target.name}` : target.name }}
        </div>
        <div class="h-px bg-base-200 dark:bg-base-300 my-0.5" />
        <ul class="flex flex-col gap-0.5">
          <li>
            <button
              class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-base-content hover:bg-primary/10 hover:text-primary transition-colors text-left"
              @click="handleTestLatency"
            >
              <BoltIcon class="h-4 w-4 shrink-0" />
              <span>{{ target.isGroup ? '测试整组延迟' : t('testLatency') }}</span>
            </button>
          </li>
          <li>
            <button
              class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-base-content hover:bg-primary/10 hover:text-primary transition-colors text-left"
              @click="handleEditNode"
            >
              <PencilSquareIcon class="h-4 w-4 shrink-0" />
              <span>{{ target.isGroup ? '编辑分组' : t('editNode') }}</span>
            </button>
          </li>
          <li>
            <button
              class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-base-content hover:bg-primary/10 hover:text-primary transition-colors text-left"
              @click="handleCopyName"
            >
              <DocumentDuplicateIcon class="h-4 w-4 shrink-0" />
              <span>{{ target.isGroup ? '复制分组名称' : t('copyNodeName') }}</span>
            </button>
          </li>
          <li v-if="target.groupName && !target.isGroup">
            <button
              class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-base-content hover:bg-primary/10 hover:text-primary transition-colors text-left"
              @click="handleSetAsDefault"
            >
              <StarIcon class="h-4 w-4 shrink-0 text-amber-500" />
              <span>设为分组默认</span>
            </button>
          </li>
          <div class="h-px bg-base-200 dark:bg-base-300 my-0.5" />
          <li>
            <button
              class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-error hover:bg-error/10 transition-colors text-left"
              @click="handleDeleteNode"
            >
              <TrashIcon class="h-4 w-4 shrink-0" />
              <span>{{ target.isGroup ? '删除分组' : t('deleteNode') }}</span>
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
