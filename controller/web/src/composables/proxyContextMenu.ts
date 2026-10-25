import { ref } from 'vue'

export interface ProxyContextMenuTarget {
  name: string
  groupName?: string
  isGroup?: boolean
}

export const isProxyContextMenuVisible = ref(false)
export const proxyContextMenuPos = ref({ x: 0, y: 0 })
export const proxyContextMenuTarget = ref<ProxyContextMenuTarget | null>(null)

export const isProxyEditModalOpen = ref(false)
export const isProxyEditModalIsEdit = ref(false)
export const proxyEditTargetTag = ref('')

export const openProxyContextMenu = (e: MouseEvent, target: ProxyContextMenuTarget) => {
  e.preventDefault()
  e.stopPropagation()

  proxyContextMenuTarget.value = target

  // 视口防溢出计算
  const menuWidth = 160
  const menuHeight = 140
  const padding = 8

  let x = e.clientX
  let y = e.clientY

  if (x + menuWidth > window.innerWidth - padding) {
    x = Math.max(padding, window.innerWidth - menuWidth - padding)
  }
  if (y + menuHeight > window.innerHeight - padding) {
    y = Math.max(padding, window.innerHeight - menuHeight - padding)
  }

  proxyContextMenuPos.value = { x, y }
  isProxyContextMenuVisible.value = true
}

export const closeProxyContextMenu = () => {
  isProxyContextMenuVisible.value = false
  proxyContextMenuTarget.value = null
}

export const openProxyEditModal = (tag: string) => {
  closeProxyContextMenu()
  proxyEditTargetTag.value = tag
  isProxyEditModalIsEdit.value = true
  isProxyEditModalOpen.value = true
}

export const openCreateProxyModal = () => {
  closeProxyContextMenu()
  proxyEditTargetTag.value = ''
  isProxyEditModalIsEdit.value = false
  isProxyEditModalOpen.value = true
}

export const closeProxyEditModal = () => {
  isProxyEditModalOpen.value = false
  proxyEditTargetTag.value = ''
  isProxyEditModalIsEdit.value = false
}
