import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api'
import type { StatusInfo } from '../api'

export const isCoreReloading = ref(false)
let reloadTimer: ReturnType<typeof setTimeout> | null = null

export function markCoreReloading(durationMs = 4000) {
  isCoreReloading.value = true
  if (reloadTimer) {
    clearTimeout(reloadTimer)
  }
  reloadTimer = setTimeout(() => {
    isCoreReloading.value = false
    reloadTimer = null
  }, durationMs)
}

export function clearCoreReloading() {
  isCoreReloading.value = false
  if (reloadTimer) {
    clearTimeout(reloadTimer)
    reloadTimer = null
  }
}

export const useStatusStore = defineStore('status', {
  state: () => ({
    status: null as StatusInfo | null,
  }),
  getters: {
    isReloading: () => isCoreReloading.value,
  },
  actions: {
    markReloading(durationMs = 4000) {
      markCoreReloading(durationMs)
    },
    clearReloading() {
      clearCoreReloading()
    },
    // 每次写操作成功后调用刷新；后端暂不可达时静默保留旧状态
    async refresh() {
      try {
        const s = await api.status()
        this.status = s
        if (s?.core_alive || s?.core?.alive) {
          clearCoreReloading()
        }
      } catch {
        // ignore
      }
    },
  },
})
