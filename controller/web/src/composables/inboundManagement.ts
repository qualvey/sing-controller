import { computed, ref } from 'vue'

export const isInboundModalOpen = ref(false)
export const editingInboundTag = ref('')

export const isEditMode = computed(() => !!editingInboundTag.value)

export const openCreateInboundModal = () => {
  editingInboundTag.value = ''
  isInboundModalOpen.value = true
}

export const openEditInboundModal = (tag: string) => {
  editingInboundTag.value = tag
  isInboundModalOpen.value = true
}

export const closeInboundModal = () => {
  isInboundModalOpen.value = false
  editingInboundTag.value = ''
}
