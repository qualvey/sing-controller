import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { useStatusStore } from './status'
import {
  compareVersions,
  fetchLatestControllerRelease,
  type GitHubRelease,
} from '@/assembly/version'

const STORAGE_KEY_IGNORED = 'sing_controller_ignored_version'

export const useUpdateStore = defineStore('update', () => {
  const statusStore = useStatusStore()

  const isChecking = ref(false)
  const isUpdateAvailable = ref(false)
  const latestRelease = ref<GitHubRelease | null>(null)
  const lastCheckedAt = ref<number>(0)
  const checkError = ref<string | null>(null)

  const ignoredVersion = ref<string>(localStorage.getItem(STORAGE_KEY_IGNORED) || '')

  const currentVersion = computed(() => {
    return statusStore.status?.controller_version || __APP_VERSION__ || 'dev'
  })

  const platformArch = computed(() => {
    const os = statusStore.status?.go_os
    const arch = statusStore.status?.go_arch
    if (os && arch) {
      return `${os}/${arch}`
    }
    return ''
  })

  const latestVersion = computed(() => {
    return latestRelease.value?.tag_name || ''
  })

  const isIgnored = computed(() => {
    return !!latestVersion.value && ignoredVersion.value === latestVersion.value
  })

  const checkUpdate = async (force = false): Promise<boolean> => {
    if (isChecking.value) return isUpdateAvailable.value

    isChecking.value = true
    checkError.value = null

    try {
      const release = await fetchLatestControllerRelease(currentVersion.value, force)
      if (release) {
        latestRelease.value = release
        lastCheckedAt.value = Date.now()

        // 比较版本号：如果 latest > current
        const cmp = compareVersions(currentVersion.value, release.tag_name)
        const hasNew = cmp < 0
        isUpdateAvailable.value = hasNew && !isIgnored.value
        return isUpdateAvailable.value
      } else {
        isUpdateAvailable.value = false
        return false
      }
    } catch (e) {
      checkError.value = (e as Error).message || '检查更新失败'
      return false
    } finally {
      isChecking.value = false
    }
  }

  const ignoreVersion = (tag: string) => {
    ignoredVersion.value = tag
    localStorage.setItem(STORAGE_KEY_IGNORED, tag)
    if (latestVersion.value === tag) {
      isUpdateAvailable.value = false
    }
  }

  const resetIgnored = () => {
    ignoredVersion.value = ''
    localStorage.removeItem(STORAGE_KEY_IGNORED)
    if (latestRelease.value) {
      const cmp = compareVersions(currentVersion.value, latestRelease.value.tag_name)
      isUpdateAvailable.value = cmp < 0
    }
  }

  return {
    isChecking,
    isUpdateAvailable,
    latestRelease,
    lastCheckedAt,
    checkError,
    ignoredVersion,
    currentVersion,
    platformArch,
    latestVersion,
    isIgnored,
    checkUpdate,
    ignoreVersion,
    resetIgnored,
  }
})
