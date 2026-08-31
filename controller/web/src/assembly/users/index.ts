// 组装层 · users 门面。持有 users / inbounds 统一状态与渲染派生，提供用户管理操作。
import { api, type Inbound, type UserMeta } from '@/api'
import { toSearchRegex } from '@/helper/search'
import { showToast } from '@/helper/toast'
import { computed, ref } from 'vue'

export const loading = ref(false)
export const users = ref<UserMeta[]>([])
export const inbounds = ref<Inbound[]>([])
export const usersFilter = ref('')

// 支持 users 的入站类型（与后端一致）
export const USER_TYPES = [
  'vless',
  'vmess',
  'trojan',
  'tuic',
  'hysteria2',
  'hysteria',
  'shadowsocks',
  'anytls',
  'shadowtls',
]

export const bindableInbounds = computed(() => {
  return inbounds.value.filter((x) => USER_TYPES.includes(x.type))
})

export const renderUsers = computed(() => {
  const searchRegex = toSearchRegex(usersFilter.value)
  if (!searchRegex) {
    return users.value
  }

  return users.value.filter((user) => {
    const searchTargets = [
      user.name,
      user.uuid || '',
      user.flow || '',
      ...(user.bound_inbounds || []),
    ]
    return searchRegex.testAny(searchTargets)
  })
})

export const fetchUsers = async () => {
  loading.value = true
  try {
    const [u, i] = await Promise.all([api.users(), api.inbounds()])
    users.value = u
    inbounds.value = i
  } catch (e) {
    showToast((e as Error).message || '加载用户失败', 'error')
  } finally {
    loading.value = false
  }
}

export const saveUser = async (user: UserMeta, isEdit: boolean, originalName?: string) => {
  try {
    if (isEdit && originalName) {
      await api.updateUser(originalName, user)
    } else {
      await api.createUser(user)
    }
    showToast('已保存（绑定入站的 users 已同步）', 'success')
    await fetchUsers()
    return true
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
    return false
  }
}

export const deleteUser = async (name: string) => {
  try {
    await api.deleteUser(name)
    showToast('已删除', 'success')
    await fetchUsers()
    return true
  } catch (e) {
    showToast((e as Error).message || '删除失败', 'error')
    return false
  }
}

// crypto.randomUUID 需要安全上下文（https/localhost）——局域网 http 下不可用，
// 用 getRandomValues（无此限制）实现 UUID v4
export const generateUuid = () => {
  const bytes = crypto.getRandomValues(new Uint8Array(16))
  bytes[6] = (bytes[6] & 0x0f) | 0x40
  bytes[8] = (bytes[8] & 0x3f) | 0x80
  const hex = [...bytes].map((b) => b.toString(16).padStart(2, '0')).join('')
  return (
    hex.slice(0, 8) +
    '-' +
    hex.slice(8, 12) +
    '-' +
    hex.slice(12, 16) +
    '-' +
    hex.slice(16, 20) +
    '-' +
    hex.slice(20)
  )
}

// 生成随机密码：优先后端（POST /api/tools/password），失败浏览器端兜底
export const generatePassword = async (notify = false): Promise<string> => {
  try {
    const pwd = await api.genPassword()
    if (notify) showToast('已生成随机密码', 'success')
    return pwd
  } catch {
    const bytes = crypto.getRandomValues(new Uint8Array(16))
    const pwd = btoa(String.fromCharCode(...bytes))
    if (notify) showToast('后端生成失败，已用浏览器随机数生成', 'warning')
    return pwd
  }
}
