import type { UserMeta } from '@/api'
import { computed, ref } from 'vue'

export const isUserModalOpen = ref(false)
export const isUserDrawerOpen = ref(false)
export const editingUser = ref<UserMeta | null>(null)
export const onUserSavedCallback = ref<((user: UserMeta) => void) | null>(null)

export const isEditMode = computed(() => !!editingUser.value)

export const openCreateUserModal = (onSaved?: (user: UserMeta) => void) => {
  editingUser.value = null
  onUserSavedCallback.value = onSaved || null
  isUserModalOpen.value = true
}

export const openEditUserModal = (user: UserMeta, onSaved?: (user: UserMeta) => void) => {
  editingUser.value = { ...user }
  onUserSavedCallback.value = onSaved || null
  isUserModalOpen.value = true
}

export const closeUserModal = () => {
  isUserModalOpen.value = false
  editingUser.value = null
  onUserSavedCallback.value = null
}


export const openUserDrawer = () => {
  isUserDrawerOpen.value = true
}

export const closeUserDrawer = () => {
  isUserDrawerOpen.value = false
}

