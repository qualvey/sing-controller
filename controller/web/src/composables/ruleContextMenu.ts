import { ref } from 'vue'
import type { Rule } from '@/types'
import { api } from '@/api'

export interface RuleContextMenuTarget {
  rule: Rule
  index: number // 1-based index
}

export const isRuleContextMenuVisible = ref(false)
export const ruleContextMenuPos = ref({ x: 0, y: 0 })
export const ruleContextMenuTarget = ref<RuleContextMenuTarget | null>(null)
export const isRuleActionsSheet = ref(false)

export const isRouteEditModalOpen = ref(false)
export const isRouteEditMode = ref(false)
export const routeEditTargetId = ref('')

export const openRuleContextMenu = (e: MouseEvent, target: RuleContextMenuTarget) => {
  e.preventDefault()
  e.stopPropagation()

  isRuleActionsSheet.value = false
  ruleContextMenuTarget.value = target

  const menuWidth = 160
  const menuHeight = 160
  const padding = 8

  let x = e.clientX
  let y = e.clientY

  if (x + menuWidth > window.innerWidth - padding) {
    x = Math.max(padding, window.innerWidth - menuWidth - padding)
  }
  if (y + menuHeight > window.innerHeight - padding) {
    y = Math.max(padding, window.innerHeight - menuHeight - padding)
  }

  ruleContextMenuPos.value = { x, y }
  isRuleContextMenuVisible.value = true
}

export const openRuleActions = (target: RuleContextMenuTarget) => {
  isRuleActionsSheet.value = true
  ruleContextMenuTarget.value = target
  isRuleContextMenuVisible.value = true
}

export const closeRuleContextMenu = () => {
  isRuleContextMenuVisible.value = false
  ruleContextMenuTarget.value = null
  isRuleActionsSheet.value = false
}

export const openCreateRouteModal = () => {
  closeRuleContextMenu()
  isRouteEditMode.value = false
  routeEditTargetId.value = ''
  isRouteEditModalOpen.value = true
}

export const openEditRouteModal = async (targetIndex: number) => {
  closeRuleContextMenu()
  isRouteEditMode.value = true
  try {
    const data = await api.routes()
    const matchRoute = data.routes[targetIndex - 1]
    if (matchRoute && matchRoute.id) {
      routeEditTargetId.value = matchRoute.id
    } else {
      routeEditTargetId.value = ''
    }
  } catch {
    routeEditTargetId.value = ''
  }
  isRouteEditModalOpen.value = true
}

export const closeRouteEditModal = () => {
  isRouteEditModalOpen.value = false
  isRouteEditMode.value = false
  routeEditTargetId.value = ''
}
