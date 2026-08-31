import type { Backend } from '@/types'
import { useStorage } from '@vueuse/core'
import { isEqual, omit } from 'lodash'
import { v4 as uuid } from 'uuid'
import { computed, ref } from 'vue'
import { sourceIPLabelList } from './settings'

// 旧版本的后端结构:没有 `type` 字段,且 sing-box 以附属通道 `singboxChannel` 存在。
type LegacySingboxChannel = {
  protocol?: string
  host?: string
  port?: string
  secret?: string
}
type LegacyBackend = Partial<Backend> & { singboxChannel?: LegacySingboxChannel }

// 一次性迁移:补全 `type`;把旧的 singboxChannel 拆分为独立的 sing-box 后端。
const migrateBackendList = (list: LegacyBackend[]): Backend[] => {
  const migrated: Backend[] = []

  for (const item of list) {
    const channel = item.singboxChannel
    const base = omit(item, 'singboxChannel') as Backend

    migrated.push({
      ...base,
      type: base.type ?? 'clash',
    })

    if (channel?.host) {
      migrated.push({
        type: 'singbox',
        protocol: channel.protocol || 'http',
        uuid: uuid(),
        label: base.label ? `${base.label} (sing-box)` : undefined,
      })
    }
  }

  return migrated
}


export const backendList = ref<Backend[]>([
  { type: 'singbox', protocol: 'sing-box', uuid: '123', label: 'sing-box' }
]);

export const showBackendSettingsDialog = ref(false)

export const toggleBackendSettingsDialog = () => {
  showBackendSettingsDialog.value = !showBackendSettingsDialog.value
}
export const activeUuid = ref('123')
export const activeBackend = computed(() =>
  backendList.value.find((backend) => backend.uuid === activeUuid.value)
)


export type BackendManagerView =
  { mode: 'list' } | { mode: 'create' } | { mode: 'edit'; uuid: string }
export const backendManagerView = ref<BackendManagerView | null>(null)

export const openBackendManager = (view: BackendManagerView = { mode: 'list' }) => {
  backendManagerView.value = view
}
export const closeBackendManager = () => {
  backendManagerView.value = null
}