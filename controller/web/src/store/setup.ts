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
// 切换后端的唯一写入口。切换本身只是改一个 uuid,但后续的一切(会话重启、探测、
// 提示)都挂在 activeBackend 的 watch 上,散着写 activeUuid 就没有地方能收口。
export const setActiveBackend = (uuid: string) => {
  activeUuid.value = uuid
}
export const addBackend = (backend: Omit<Backend, 'uuid'>) => {
  const currentEnd = backendList.value.find((end) => {
    return isEqual(omit(end, 'uuid'), backend)
  })

  if (currentEnd) {
    setActiveBackend(currentEnd.uuid)
    return currentEnd.uuid
  }

  const id = uuid()

  backendList.value.push({
    ...backend,
    uuid: id,
  })
  setActiveBackend(id)
  return id
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