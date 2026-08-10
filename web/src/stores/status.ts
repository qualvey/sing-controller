import { defineStore } from 'pinia'
import { api } from '../api'
import type { StatusInfo } from '../api'

export const useStatusStore = defineStore('status', {
  state: () => ({
    status: null as StatusInfo | null
  }),
  actions: {
    // 每次写操作成功后调用刷新；后端暂不可达时静默保留旧状态
    async refresh() {
      try {
        this.status = await api.status()
      } catch {
        // ignore
      }
    }
  }
})
