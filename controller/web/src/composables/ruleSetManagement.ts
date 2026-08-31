import { computed, ref } from 'vue'
import { api } from '@/api'
import { showToast } from '@/helper/toast'

export const isRuleSetModalOpen = ref(false)
export const editingRuleSetData = ref<{ id: string; rule_set: Record<string, any> } | null>(null)
export const isRuleSetEditMode = computed(() => !!editingRuleSetData.value)

export const openCreateRuleSetModal = () => {
  editingRuleSetData.value = null
  isRuleSetModalOpen.value = true
}

export const openEditRuleSetModal = (data: { id: string; rule_set: Record<string, any> }) => {
  editingRuleSetData.value = { id: data.id, rule_set: { ...data.rule_set } }
  isRuleSetModalOpen.value = true
}

export const openEditRuleSetByName = async (name: string) => {
  try {
    const res = await api.ruleSets()
    const match = (res.rule_sets || []).find((item) => {
      const rs = item.rule_set || {}
      const tag = Array.isArray(rs.tag) ? rs.tag : [rs.tag]
      return item.id === name || tag.includes(name)
    })
    if (match) {
      openEditRuleSetModal(match)
    } else {
      // 如果没有匹配到完整对象，提供一个初始 tag
      openEditRuleSetModal({
        id: name,
        rule_set: { tag: name, type: 'remote', format: 'binary' }
      })
    }
  } catch (e) {
    showToast((e as Error).message || '获取规则集信息失败', 'error')
  }
}

export const closeRuleSetModal = () => {
  isRuleSetModalOpen.value = false
  editingRuleSetData.value = null
}

// Provider 右键菜单状态
export const isProviderContextMenuVisible = ref(false)
export const providerContextMenuPos = ref({ x: 0, y: 0 })
export const providerContextMenuTarget = ref<{ name: string; type: string; vehicleType?: string } | null>(null)

export const openProviderContextMenu = (
  e: MouseEvent,
  target: { name: string; type: string; vehicleType?: string },
) => {
  e.preventDefault()
  e.stopPropagation()

  providerContextMenuTarget.value = target

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

  providerContextMenuPos.value = { x, y }
  isProviderContextMenuVisible.value = true
}

export const closeProviderContextMenu = () => {
  isProviderContextMenuVisible.value = false
  providerContextMenuTarget.value = null
}
