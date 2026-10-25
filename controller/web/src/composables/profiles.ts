import {
  controllerApi,
  type ConfigProfile,
  type NodeProvider,
} from '@/api/controller'
import { startBackendSession } from '@/assembly/session'
import { showNotification } from '@/helper/notification'
import { useStatusStore } from '@/store/status'
import { ref } from 'vue'

export const isProfileModalOpen = ref(false)
// 由错误/诊断界面请求直接打开某个远程配置源的编辑器。
export const providerEditRequest = ref<NodeProvider | null>(null)

export const openProfileModal = (tab?: 'profiles' | 'providers') => {
  if (tab) activeProfilesTab.value = tab
  isProfileModalOpen.value = true
}

export const closeProfileModal = () => {
  isProfileModalOpen.value = false
}

export const openProviderEditor = (provider: NodeProvider) => {
  activeProfilesTab.value = 'providers'
  providerEditRequest.value = provider
  isProfileModalOpen.value = true
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
    showNotification({ content: msg, type: 'alert-error' })
  } finally {
    isProfilesLoading.value = false
  }
}

export const applyProfile = async (id: string) => {
  isApplyingProfileId.value = id
  try {
    await controllerApi.applyProfile(id)
    showNotification({ content: '配置方案已应用，sing-box 内核正在启动...', type: 'alert-success' })
    await loadProfilesData()
    const statusStore = useStatusStore()
    await statusStore.refresh()
    setTimeout(() => {
      startBackendSession()
    }, 600)
  } catch (err: unknown) {
    const msg = (err as any)?.response?.data?.message || (err as Error)?.message || '应用方案失败'
    showNotification({ content: msg, type: 'alert-error' })
  } finally {
    isApplyingProfileId.value = null
  }
}

export const deleteProfile = async (id: string) => {
  try {
    await controllerApi.deleteProfile(id)
    showNotification({ content: '配置方案已删除', type: 'alert-success' })
    await loadProfilesData()
  } catch (err: unknown) {
    const msg = (err as any)?.response?.data?.message || (err as Error)?.message || '删除方案失败'
    showNotification({ content: msg, type: 'alert-error' })
  }
}

export const refreshProvider = async (id: string) => {
  isRefreshingProviderId.value = id
  try {
    await controllerApi.refreshProvider(id)
    showNotification({ content: '订阅源已成功更新', type: 'alert-success' })
    await loadProfilesData()
    const statusStore = useStatusStore()
    await statusStore.refresh()

    if (statusStore.status?.core_alive) {
      setTimeout(() => {
        startBackendSession()
      }, 600)
    }
  } catch (err: unknown) {
    const msg = (err as any)?.response?.data?.message || (err as Error)?.message || '刷新订阅失败'
    showNotification({ content: msg, type: 'alert-error' })
  } finally {
    isRefreshingProviderId.value = null
  }
}

export const deleteProvider = async (id: string) => {
  try {
    await controllerApi.deleteProvider(id)
    showNotification({ content: '订阅源已删除', type: 'alert-success' })
    await loadProfilesData()
  } catch (err: unknown) {
    const msg = (err as any)?.response?.data?.message || (err as Error)?.message || '删除订阅源失败'
    showNotification({ content: msg, type: 'alert-error' })
  }
}
