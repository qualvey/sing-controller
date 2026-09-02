import {
  controllerApi,
  type ConfigProfile,
  type NodeProvider,
} from '@/api/controller'
import { showNotification } from '@/helper/notification'
import { ref } from 'vue'

export const isProfileModalOpen = ref(false)

export const openProfileModal = () => {
  isProfileModalOpen.value = true
}

export const closeProfileModal = () => {
  isProfileModalOpen.value = false
}

export const activeProfilesTab = ref<'profiles' | 'providers'>('profiles')
export const profilesList = ref<ConfigProfile[]>([])
export const providersList = ref<NodeProvider[]>([])
export const isProfilesLoading = ref(false)
export const isApplyingProfileId = ref<string | null>(null)
export const isRefreshingProviderId = ref<string | null>(null)

export const loadProfilesData = async () => {
  isProfilesLoading.value = true
  try {
    const [profs, provs] = await Promise.all([
      controllerApi.profiles(),
      controllerApi.providers(),
    ])
    profilesList.value = profs || []
    providersList.value = provs || []
  } catch (err: unknown) {
    const msg = (err as any)?.response?.data?.message || (err as Error)?.message || '加载配置与订阅失败'
    showNotification(msg, 'alert-error')
  } finally {
    isProfilesLoading.value = false
  }
}

export const applyProfile = async (id: string) => {
  isApplyingProfileId.value = id
  try {
    await controllerApi.applyProfile(id)
    showNotification('配置方案已应用', 'alert-success')
    await loadProfilesData()
  } catch (err: unknown) {
    const msg = (err as any)?.response?.data?.message || (err as Error)?.message || '应用方案失败'
    showNotification(msg, 'alert-error')
  } finally {
    isApplyingProfileId.value = null
  }
}

export const deleteProfile = async (id: string) => {
  try {
    await controllerApi.deleteProfile(id)
    showNotification('配置方案已删除', 'alert-success')
    await loadProfilesData()
  } catch (err: unknown) {
    const msg = (err as any)?.response?.data?.message || (err as Error)?.message || '删除方案失败'
    showNotification(msg, 'alert-error')
  }
}

export const refreshProvider = async (id: string) => {
  isRefreshingProviderId.value = id
  try {
    await controllerApi.refreshProvider(id)
    showNotification('订阅源已成功更新', 'alert-success')
    await loadProfilesData()
  } catch (err: unknown) {
    const msg = (err as any)?.response?.data?.message || (err as Error)?.message || '刷新订阅失败'
    showNotification(msg, 'alert-error')
  } finally {
    isRefreshingProviderId.value = null
  }
}

export const deleteProvider = async (id: string) => {
  try {
    await controllerApi.deleteProvider(id)
    showNotification('订阅源已删除', 'alert-success')
    await loadProfilesData()
  } catch (err: unknown) {
    const msg = (err as any)?.response?.data?.message || (err as Error)?.message || '删除订阅源失败'
    showNotification(msg, 'alert-error')
  }
}
