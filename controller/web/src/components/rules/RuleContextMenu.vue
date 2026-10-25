<script setup lang="ts">
import { computed, inject, nextTick, onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  isRuleContextMenuVisible,
  ruleContextMenuPos,
  ruleContextMenuTarget,
  isRuleActionsSheet,
  closeRuleContextMenu,
  openEditRouteModal
} from '@/composables/ruleContextMenu'
import { fetchRules, reorderRules, rules, toggleRuleDisabled } from '@/assembly/rules'
import { proxyGroupList } from '@/assembly/proxies'
import { Channel, channel } from '@/assembly/backend'
import { api } from '@/api'
import { showToast } from '@/helper/toast'
import { showConfirmDialog } from '@/helper/confirmDialog'
import {
  PencilSquareIcon,
  TrashIcon,
  DocumentDuplicateIcon,
  NoSymbolIcon,
  CheckCircleIcon,
  ArrowUpIcon,
  ArrowDownIcon,
  BarsArrowUpIcon,
  BarsArrowDownIcon,
  ArrowsRightLeftIcon,
} from '@heroicons/vue/24/outline'
import { useDialogOpenState } from '@/composables/dialog'

const { t } = useI18n()
const expandedRule = inject<Ref<string | null>>('expandedRule', ref(null))
const canSpecify = computed(() =>
  !!target.value && proxyGroupList.value.includes(target.value.rule.proxy),
)

useDialogOpenState(isRuleContextMenuVisible)

const target = computed(() => ruleContextMenuTarget.value)

const menuStyle = computed(() => ({
  left: `${ruleContextMenuPos.value.x}px`,
  top: `${ruleContextMenuPos.value.y}px`
}))

const isDisabled = computed(() => {
  if (!target.value) return false
  const r = target.value.rule
  return r.extra ? r.extra.disabled : r.disabled
})

const canReorderInContext = computed(() => {
  return channel.value === Channel.Singbox && rules.value.length > 1
})

const isFirst = computed(() => {
  if (!target.value) return false
  return target.value.index <= 1
})

const isLast = computed(() => {
  if (!target.value) return false
  return target.value.index >= rules.value.length
})

const handleMoveUp = async () => {
  if (!target.value || isFirst.value) return
  const curIdx = target.value.index - 1
  closeRuleContextMenu()
  const newList = [...rules.value]
  const [item] = newList.splice(curIdx, 1)
  newList.splice(curIdx - 1, 0, item)
  try {
    await reorderRules(newList)
    showToast(t('reorderRulesSuccess') || '规则顺序已更新', 'success')
  } catch (e: any) {
    showToast(e.message || '移动失败', 'error')
    await fetchRules()
  }
}

const handleMoveDown = async () => {
  if (!target.value || isLast.value) return
  const curIdx = target.value.index - 1
  closeRuleContextMenu()
  const newList = [...rules.value]
  const [item] = newList.splice(curIdx, 1)
  newList.splice(curIdx + 1, 0, item)
  try {
    await reorderRules(newList)
    showToast(t('reorderRulesSuccess') || '规则顺序已更新', 'success')
  } catch (e: any) {
    showToast(e.message || '移动失败', 'error')
    await fetchRules()
  }
}

const handleMoveToTop = async () => {
  if (!target.value || isFirst.value) return
  const curIdx = target.value.index - 1
  closeRuleContextMenu()
  const newList = [...rules.value]
  const [item] = newList.splice(curIdx, 1)
  newList.unshift(item)
  try {
    await reorderRules(newList)
    showToast(t('reorderRulesSuccess') || '规则顺序已更新', 'success')
  } catch (e: any) {
    showToast(e.message || '置顶失败', 'error')
    await fetchRules()
  }
}

const handleMoveToBottom = async () => {
  if (!target.value || isLast.value) return
  const curIdx = target.value.index - 1
  closeRuleContextMenu()
  const newList = [...rules.value]
  const [item] = newList.splice(curIdx, 1)
  newList.push(item)
  try {
    await reorderRules(newList)
    showToast(t('reorderRulesSuccess') || '规则顺序已更新', 'success')
  } catch (e: any) {
    showToast(e.message || '置底失败', 'error')
    await fetchRules()
  }
}

const handleEditRule = () => {
  if (!target.value) return
  openEditRouteModal(target.value.index)
}

const handleSpecifyRule = async () => {
  if (!target.value || !canSpecify.value) return
  const { rule, index } = target.value
  expandedRule.value = `${index}-${rule.payload}`
  closeRuleContextMenu()
  await nextTick()
  document.getElementById(`rule-card-${index}`)?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
}

const handleDeleteRule = async () => {
  if (!target.value) return
  const targetIndex = target.value.index
  closeRuleContextMenu()

  const { confirmed } = await showConfirmDialog({
    title: t('confirmDelete') || '删除确认',
    message: t('confirmDeleteRule') || `确定删除第 ${targetIndex} 条路由规则？该操作不可恢复。`,
    confirmText: t('delete') || '删除',
    confirmButtonClass: 'btn-error'
  })
  if (!confirmed) return

  try {
    const data = await api.routes()
    const matchRoute = data.routes[targetIndex - 1]
    if (!matchRoute || !matchRoute.id) {
      throw new Error('未找到对应规则 ID')
    }
    await api.deleteRoute(matchRoute.id)
    showToast(t('deleteSuccess') || '删除成功', 'success')
    await fetchRules()
  } catch (e) {
    showToast((e as Error).message || '删除失败', 'error')
  }
}

const handleToggleDisabled = async () => {
  if (!target.value) return
  const rule = target.value.rule
  const willBeDisabled = !isDisabled.value
  closeRuleContextMenu()
  try {
    await toggleRuleDisabled(rule, willBeDisabled)
    await fetchRules()
  } catch (e) {
    showToast((e as Error).message || '操作失败', 'error')
  }
}

const handleCopyPayload = async () => {
  if (!target.value) return
  const text = target.value.rule.payload || target.value.rule.type
  closeRuleContextMenu()
  try {
    await navigator.clipboard.writeText(text)
    showToast(t('copySuccess') || '复制成功', 'success')
  } catch {
    const textArea = document.createElement('textarea')
    textArea.value = text
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
  if (!isRuleContextMenuVisible.value) return
  const el = document.getElementById(
    isRuleActionsSheet.value ? 'rule-actions-sheet' : 'rule-context-menu',
  )
  if (el && !el.contains(e.target as Node)) {
    closeRuleContextMenu()
  }
}

const handleKeyDown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && isRuleContextMenuVisible.value) {
    closeRuleContextMenu()
  }
}

const handleScroll = () => {
  if (isRuleContextMenuVisible.value && !isRuleActionsSheet.value) {
    closeRuleContextMenu()
  }
}

watch(isRuleContextMenuVisible, (visible) => {
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
        v-if="isRuleContextMenuVisible && target"
        v-show="!isRuleActionsSheet"
        id="rule-context-menu"
        class="fixed z-[9999] min-w-[150px] rounded-lg border border-base-300/80 bg-base-100/95 p-1 text-sm shadow-xl backdrop-blur-md dark:border-base-content/10"
        :style="menuStyle"
        @contextmenu.prevent.stop
      >
        <div class="px-2.5 py-1.5 text-[11px] font-medium text-base-content/50 truncate max-w-[220px]">
          #{{ target.index }} · {{ target.rule.type }}{{ target.rule.payload ? ` : ${target.rule.payload}` : '' }}
        </div>
        <div class="h-px bg-base-200 dark:bg-base-300 my-0.5" />
        <ul class="flex flex-col gap-0.5">
          <li>
            <button
              class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-base-content hover:bg-primary/10 hover:text-primary transition-colors text-left"
              @click="handleEditRule"
            >
              <PencilSquareIcon class="h-4 w-4 shrink-0" />
              <span>{{ t('editRule') || '编辑规则' }}</span>
            </button>
          </li>
          <li>
            <button
              class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-base-content hover:bg-primary/10 hover:text-primary transition-colors text-left"
              @click="handleToggleDisabled"
            >
              <CheckCircleIcon v-if="isDisabled" class="h-4 w-4 shrink-0 text-success" />
              <NoSymbolIcon v-else class="h-4 w-4 shrink-0 text-warning" />
              <span>{{ isDisabled ? (t('enableRule') || '启用规则') : (t('disableRule') || '禁用规则') }}</span>
            </button>
          </li>
          <li>
            <button
              class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-base-content hover:bg-primary/10 hover:text-primary transition-colors text-left"
              @click="handleCopyPayload"
            >
              <DocumentDuplicateIcon class="h-4 w-4 shrink-0" />
              <span>{{ t('copyRule') || '复制规则内容' }}</span>
            </button>
          </li>
          <template v-if="canReorderInContext">
            <div class="h-px bg-base-200 dark:bg-base-300 my-0.5" />
            <li v-if="!isFirst">
              <button
                class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-base-content hover:bg-primary/10 hover:text-primary transition-colors text-left"
                @click="handleMoveToTop"
              >
                <BarsArrowUpIcon class="h-4 w-4 shrink-0 opacity-70" />
                <span>{{ t('moveRuleToTop') || '置顶规则' }}</span>
              </button>
            </li>
            <li v-if="!isFirst">
              <button
                class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-base-content hover:bg-primary/10 hover:text-primary transition-colors text-left"
                @click="handleMoveUp"
              >
                <ArrowUpIcon class="h-4 w-4 shrink-0 opacity-70" />
                <span>{{ t('moveRuleUp') || '上移规则' }}</span>
              </button>
            </li>
            <li v-if="!isLast">
              <button
                class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-base-content hover:bg-primary/10 hover:text-primary transition-colors text-left"
                @click="handleMoveDown"
              >
                <ArrowDownIcon class="h-4 w-4 shrink-0 opacity-70" />
                <span>{{ t('moveRuleDown') || '下移规则' }}</span>
              </button>
            </li>
            <li v-if="!isLast">
              <button
                class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-base-content hover:bg-primary/10 hover:text-primary transition-colors text-left"
                @click="handleMoveToBottom"
              >
                <BarsArrowDownIcon class="h-4 w-4 shrink-0 opacity-70" />
                <span>{{ t('moveRuleToBottom') || '置底规则' }}</span>
              </button>
            </li>
          </template>
          <div class="h-px bg-base-200 dark:bg-base-300 my-0.5" />
          <li>
            <button
              class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-error hover:bg-error/10 transition-colors text-left"
              @click="handleDeleteRule"
            >
              <TrashIcon class="h-4 w-4 shrink-0" />
              <span>{{ t('deleteRule') || '删除规则' }}</span>
            </button>
          </li>
        </ul>
      </div>
    </Transition>

    <Transition name="rule-actions-sheet">
      <div
        v-if="isRuleContextMenuVisible && isRuleActionsSheet && target"
        id="rule-actions-sheet"
        class="fixed inset-0 z-[9999] flex items-end bg-black/40"
        role="presentation"
        @click="closeRuleContextMenu"
      >
        <section
          class="rule-actions-panel bg-base-100 border-base-content/10 w-full overflow-y-auto rounded-t-2xl border-t px-4 pt-3 shadow-2xl"
          role="dialog"
          aria-modal="true"
          :aria-label="$t('ruleActions')"
          @click.stop
        >
          <div class="bg-base-content/20 mx-auto mb-3 h-1 w-10 rounded-full" />
          <div class="text-base-content/55 mb-2 truncate px-1 text-xs">
            #{{ target.index }} · {{ target.rule.type }}{{ target.rule.payload ? ` : ${target.rule.payload}` : '' }}
          </div>
          <div class="flex flex-col pb-[calc(0.75rem+env(safe-area-inset-bottom))]">
            <button
              type="button"
              class="rule-action-button"
              @click="handleEditRule"
            >
              <PencilSquareIcon class="h-5 w-5" />
              <span>{{ t('editRule') }}</span>
            </button>
            <button
              v-if="canSpecify"
              type="button"
              class="rule-action-button"
              @click="handleSpecifyRule"
            >
              <ArrowsRightLeftIcon class="h-5 w-5" />
              <span>{{ t('specifyRule') }}</span>
            </button>
            <button
              type="button"
              class="rule-action-button"
              @click="handleToggleDisabled"
            >
              <CheckCircleIcon v-if="isDisabled" class="h-5 w-5 text-success" />
              <NoSymbolIcon v-else class="h-5 w-5 text-warning" />
              <span>{{ isDisabled ? t('enableRule') : t('disableRule') }}</span>
            </button>
            <button
              type="button"
              class="rule-action-button text-error"
              @click="handleDeleteRule"
            >
              <TrashIcon class="h-5 w-5" />
              <span>{{ t('deleteRule') }}</span>
            </button>
          </div>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.rule-actions-panel {
  max-height: min(75dvh, 34rem);
  padding-top: max(0.75rem, env(safe-area-inset-top));
}

.rule-action-button {
  display: flex;
  min-height: 3rem;
  align-items: center;
  gap: 0.75rem;
  border-radius: 0.5rem;
  padding-inline: 0.5rem;
  text-align: left;
  font-size: 0.875rem;
  line-height: 1.25rem;
}

.rule-action-button:active {
  background-color: var(--color-base-200);
}

.fade-scale-enter-active,
.fade-scale-leave-active {
  transition: all 0.12s ease-out;
}

.fade-scale-enter-from,
.fade-scale-leave-to {
  opacity: 0;
  transform: scale(0.95);
}

.rule-actions-sheet-enter-active,
.rule-actions-sheet-leave-active {
  transition: opacity 0.18s ease-out;
}

.rule-actions-sheet-enter-active .rule-actions-panel,
.rule-actions-sheet-leave-active .rule-actions-panel {
  transition: transform 0.22s cubic-bezier(0.32, 0.72, 0, 1);
}

.rule-actions-sheet-enter-from,
.rule-actions-sheet-leave-to {
  opacity: 0;
}

.rule-actions-sheet-enter-from .rule-actions-panel,
.rule-actions-sheet-leave-to .rule-actions-panel {
  transform: translateY(100%);
}
</style>
