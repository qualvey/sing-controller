<template>
  <Teleport to="body">
    <Transition name="fade-scale">
      <div
        v-if="isProviderContextMenuVisible && target"
        id="provider-context-menu"
        class="fixed z-[9999] min-w-[150px] rounded-lg border border-base-300/80 bg-base-100/95 p-1 text-sm shadow-xl backdrop-blur-md dark:border-base-content/10"
        :style="menuStyle"
        @contextmenu.prevent.stop
      >
        <div class="text-base-content/50 max-w-[220px] truncate px-2.5 py-1.5 text-[11px] font-medium">
          {{ target.name }} · {{ target.type || 'RuleSet' }}
        </div>
        <div class="bg-base-200 dark:bg-base-300 my-0.5 h-px" />
        <ul class="flex flex-col gap-0.5">
          <li v-if="target.vehicleType !== 'Inline'">
            <button
              class="hover:bg-primary/10 hover:text-primary text-base-content flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors"
              @click="handleUpdate"
            >
              <ArrowPathIcon class="h-4 w-4 shrink-0" />
              <span>{{ t('update') || '更新规则集' }}</span>
            </button>
          </li>
          <li>
            <button
              class="hover:bg-primary/10 hover:text-primary text-base-content flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors"
              @click="handleEdit"
            >
              <PencilSquareIcon class="h-4 w-4 shrink-0" />
              <span>{{ t('edit') || '编辑规则集' }}</span>
            </button>
          </li>
          <li>
            <button
              class="hover:bg-primary/10 hover:text-primary text-base-content flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors"
              @click="handleCopyName"
            >
              <DocumentDuplicateIcon class="h-4 w-4 shrink-0" />
              <span>{{ t('copy') || '复制名称' }}</span>
            </button>
          </li>
          <div class="bg-base-200 dark:bg-base-300 my-0.5 h-px" />
          <li>
            <button
              class="hover:bg-error/10 text-error flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors"
              @click="handleDelete"
            >
              <TrashIcon class="h-4 w-4 shrink-0" />
              <span>{{ t('delete') || '删除规则集' }}</span>
            </button>
          </li>
        </ul>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { api } from '@/api'
import { fetchRules, updateRuleProviderAPI } from '@/assembly/rules'
import {
  closeProviderContextMenu,
  isProviderContextMenuVisible,
  openEditRuleSetByName,
  providerContextMenuPos,
  providerContextMenuTarget,
} from '@/composables/ruleSetManagement'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { showToast } from '@/helper/toast'
import { useStatusStore } from '@/store/status'
import {
  ArrowPathIcon,
  DocumentDuplicateIcon,
  PencilSquareIcon,
  TrashIcon,
} from '@heroicons/vue/24/outline'
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const statusStore = useStatusStore()

const target = computed(() => providerContextMenuTarget.value)

const menuStyle = computed(() => ({
  left: `${providerContextMenuPos.value.x}px`,
  top: `${providerContextMenuPos.value.y}px`,
}))

const handleUpdate = async () => {
  if (!target.value) return
  const name = target.value.name
  closeProviderContextMenu()
  try {
    await updateRuleProviderAPI(name)
    showToast('更新已触发', 'success')
    await fetchRules()
  } catch (e) {
    showToast((e as Error).message || '更新失败', 'error')
  }
}

const handleEdit = () => {
  if (!target.value) return
  const name = target.value.name
  closeProviderContextMenu()
  openEditRuleSetByName(name)
}

const handleCopyName = async () => {
  if (!target.value) return
  const text = target.value.name
  closeProviderContextMenu()
  try {
    await navigator.clipboard.writeText(text)
    showToast('复制成功', 'success')
  } catch {
    showToast('复制失败', 'error')
  }
}

const handleDelete = async () => {
  if (!target.value) return
  const name = target.value.name
  closeProviderContextMenu()

  const first = await showConfirmDialog({
    title: t('confirmDelete') || '删除确认',
    message: `确定删除规则集「${name}」？被引用的规则将失效。`,
    confirmText: t('delete') || '删除',
    confirmButtonClass: 'btn-error',
  })
  if (!first.confirmed) return

  try {
    // 优先通过 api.ruleSets 匹配到准确 ID
    let deleteId = name
    try {
      const res = await api.ruleSets()
      const match = (res.rule_sets || []).find((item) => {
        const rs = item.rule_set || {}
        const tag = Array.isArray(rs.tag) ? rs.tag : [rs.tag]
        return item.id === name || tag.includes(name)
      })
      if (match) deleteId = match.id
    } catch {
      // ignore
    }

    await api.deleteRuleSet(deleteId)
    showToast('删除成功', 'success')
    await Promise.all([fetchRules(), statusStore.refresh()])
  } catch (e) {
    const err = e as Error & { references?: string[] }
    if (err.references?.length) {
      const second = await showConfirmDialog({
        title: '被引用确认',
        message: `该规则集被引用：${err.references.join('、')}\n删除后将自动从这些规则中移除引用。确认删除？`,
        confirmText: '确认删除',
        confirmButtonClass: 'btn-error',
      })
      if (!second.confirmed) return
      try {
        await api.deleteRuleSet(name, true)
        showToast('删除成功（已移除引用）', 'success')
        await Promise.all([fetchRules(), statusStore.refresh()])
      } catch (e2) {
        showToast((e2 as Error).message || '删除失败', 'error')
      }
      return
    }
    showToast(err.message || '删除失败', 'error')
  }
}

const handleGlobalPointerDown = (e: MouseEvent | TouchEvent) => {
  if (!isProviderContextMenuVisible.value) return
  const el = document.getElementById('provider-context-menu')
  if (el && !el.contains(e.target as Node)) {
    closeProviderContextMenu()
  }
}

const handleKeyDown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && isProviderContextMenuVisible.value) {
    closeProviderContextMenu()
  }
}

const handleScroll = () => {
  if (isProviderContextMenuVisible.value) {
    closeProviderContextMenu()
  }
}

watch(isProviderContextMenuVisible, (visible) => {
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
