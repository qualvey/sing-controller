import { ref } from 'vue'

export const isProfileModalOpen = ref(false)

export const openProfileModal = () => {
  isProfileModalOpen.value = true
}

export const closeProfileModal = () => {
  isProfileModalOpen.value = false
}
