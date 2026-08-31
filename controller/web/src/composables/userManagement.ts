import type { UserMeta } from '@/api'
import { computed, ref } from 'vue'

export const isUserModalOpen = ref(false)
export const editingUser = ref<UserMeta | null>(null)

export const isEditMode = computed(() => !!editingUser.value)

export const openCreateUserModal = () => {
  editingUser.value = null
  isUserModalOpen.value = true
}

export const openEditUserModal = (user: UserMeta) => {
  editingUser.value = { ...user }
  isUserModalOpen.value = true
}

export const closeUserModal = () => {
  isUserModalOpen.value = false
  editingUser.value = null
}
