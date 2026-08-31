// 组装层 · inbounds 门面。持有 inbounds 统一状态与渲染派生，提供入站管理操作。
import { api, type Inbound, type UserMeta } from '@/api'
import { toSearchRegex } from '@/helper/search'
import { showToast } from '@/helper/toast'
import { useStatusStore } from '@/store/status'
import { computed, ref } from 'vue'

export const SS_METHODS = [
  'none',
  'aes-128-gcm',
  'aes-192-gcm',
  'aes-256-gcm',
  'chacha20-ietf-poly1305',
  'xchacha20-ietf-poly1305',
  '2022-blake3-aes-128-gcm',
  '2022-blake3-aes-256-gcm',
  '2022-blake3-chacha20-poly1305',
]
export const SS_DEFAULT_METHOD = 'chacha20-ietf-poly1305'
export const SS_DEFAULT_LISTEN = '::'
export const SS_DEFAULT_PORT = 23010
export const TLS_VERSIONS = ['1.0', '1.1', '1.2', '1.3']

export const inbounds = ref<Inbound[]>([])
export const inboundTypes = ref<string[]>([])
export const loading = ref(false)
export const inboundsFilter = ref('')
export const inboundsTabShow = ref('form')
export const poolUsers = ref<UserMeta[]>([])
export const certProviders = ref<string[]>([])

export const renderInbounds = computed(() => {
  const searchRegex = toSearchRegex(inboundsFilter.value)
  if (!searchRegex) {
    return inbounds.value
  }

  return inbounds.value.filter((row) => {
    const targets = [
      row.tag,
      row.type,
      row.listen || '',
      row.listen_port != null ? String(row.listen_port) : '',
      row.method || '',
    ]
    return searchRegex.testAny(targets)
  })
})

export const fetchInbounds = async () => {
  loading.value = true
  try {
    inbounds.value = await api.inbounds()
  } catch (e) {
    showToast((e as Error).message || '加载 inbounds 失败', 'error')
  } finally {
    loading.value = false
  }
}

export const fetchInboundTypes = async () => {
  try {
    const t = await api.types()
    const implemented = ['mixed', 'tuic', 'shadowsocks']
    const all = t.inbounds || []
    inboundTypes.value = implemented.filter((x) => all.includes(x))
  } catch {
    inboundTypes.value = []
  }
}

export const fetchPoolUsers = async () => {
  try {
    poolUsers.value = await api.users()
  } catch {
    poolUsers.value = []
  }
}

export const fetchCertProviders = async () => {
  try {
    const cert = await api.certificate()
    const list = cert?.providers || []
    certProviders.value = list
      .map((p: { id: string; provider: Record<string, any> }) => p.provider?.tag)
      .filter((t: unknown): t is string => typeof t === 'string' && t.length > 0)
  } catch {
    certProviders.value = []
  }
}

export const fetchDefaults = async (): Promise<string | undefined> => {
  try {
    const s = await api.settings()
    return s.defaults?.listen
  } catch {
    return undefined
  }
}

export const createInbound = async (data: Inbound) => {
  const statusStore = useStatusStore()
  try {
    const res = await api.createInbound(data)
    handleResult(res)
    await Promise.all([fetchInbounds(), statusStore.refresh()])
    return true
  } catch (e) {
    showToast((e as Error).message || '创建 Inbound 失败', 'error')
    return false
  }
}

export const updateInbound = async (tag: string, data: Inbound) => {
  const statusStore = useStatusStore()
  try {
    const res = await api.updateInbound(tag, data)
    handleResult(res)
    await Promise.all([fetchInbounds(), statusStore.refresh()])
    return true
  } catch (e) {
    showToast((e as Error).message || '更新 Inbound 失败', 'error')
    return false
  }
}

export const deleteInbound = async (tag: string) => {
  const statusStore = useStatusStore()
  try {
    await api.deleteInbound(tag)
    showToast('删除成功', 'success')
    await Promise.all([fetchInbounds(), statusStore.refresh()])
    return true
  } catch (e) {
    showToast((e as Error).message || '删除失败', 'error')
    return false
  }
}

export const allocateAvailablePort = async (notify = true): Promise<number | undefined> => {
  try {
    const res = await api.availablePort()
    if (notify) showToast(`已分配可用端口 ${res.port}`, 'success')
    return res.port
  } catch (e) {
    if (notify) showToast((e as Error).message || '端口分配失败', 'error')
    return undefined
  }
}

export const generateSsPassword = async (notify = true): Promise<string> => {
  try {
    const pwd = await api.genPassword()
    if (notify) showToast('已生成随机密码', 'success')
    return pwd
  } catch {
    const bytes = new Uint8Array(16)
    crypto.getRandomValues(bytes)
    const pwd = btoa(String.fromCharCode(...bytes))
    if (notify) showToast('后端生成失败，已用浏览器随机数生成', 'warning')
    return pwd
  }
}

export const syncBoundUsers = async (tag: string, targetUserNames: string[]) => {
  if (!tag) return
  const current = new Set(
    poolUsers.value.filter((u) => u.bound_inbounds?.includes(tag)).map((u) => u.name)
  )
  const target = new Set(targetUserNames)
  for (const u of poolUsers.value) {
    const has = current.has(u.name)
    const want = target.has(u.name)
    if (has !== want) {
      const binds = new Set(u.bound_inbounds || [])
      if (want) binds.add(tag)
      else binds.delete(tag)
      await api.updateUser(u.name, { ...u, bound_inbounds: [...binds] })
    }
  }
}

const handleResult = (res: unknown) => {
  const r = (res ?? {}) as { reload_error?: string; message?: string }
  if (r.reload_error) {
    showToast(r.message || `配置已保存，但实例重载失败：${r.reload_error}`, 'warning')
  } else {
    showToast('保存成功', 'success')
  }
}
