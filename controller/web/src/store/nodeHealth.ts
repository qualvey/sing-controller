import { api, type NodeHealthStatus } from '@/api/controller'
import { ref } from 'vue'

export const nodeHealthEnabled = ref(false)
export const nodeHealthScanning = ref(false)
export const nodeHealthStatuses = ref<Record<string, NodeHealthStatus>>({})
let refreshInFlight = false

export const refreshNodeHealth = async () => {
  if (refreshInFlight) return
  refreshInFlight = true
  try {
    const snapshot = await api.nodeHealth()
    nodeHealthEnabled.value = snapshot.enabled
    nodeHealthScanning.value = snapshot.scanning || snapshot.scan_queued
    nodeHealthStatuses.value = snapshot.statuses
  } catch {
    nodeHealthEnabled.value = false
    nodeHealthScanning.value = false
    nodeHealthStatuses.value = {}
  } finally {
    refreshInFlight = false
  }
}
