// 组装层 · DNS 业务门面。统一持有 DNS 状态（servers, rules, options）并提供 CRUD 契约。
import { api } from '@/api'
import { toSearchRegex } from '@/helper/search'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { showToast } from '@/helper/toast'
import { useStatusStore } from '@/store/status'
import { computed, reactive, ref } from 'vue'

export const DNS_TYPES = [
  'local',
  'udp',
  'tcp',
  'tls',
  'https',
  'quic',
  'h3',
  'fakeip',
  'hosts',
  'dhcp',
  'mdns',
] as const

export const STRATEGY_OPTIONS = ['ipv4_only', 'ipv6_only', 'prefer_ipv4', 'prefer_ipv6']

export const servers = ref<Array<Record<string, any>>>([])
export const rules = ref<Array<{ id: string; rule: Record<string, any> }>>([])
export const outboundTags = ref<string[]>([])
export const inboundTags = ref<string[]>([])
export const loading = ref(false)
export const saving = ref(false)
export const dnsTabShow = ref('servers')
export const dnsFilter = ref('')

export const optsForm = reactive({
  final: '',
  strategy: '',
  timeout: '',
  disable_cache: false,
  independent_cache: false,
  reverse_mapping: false,
})

export const dnsTags = computed(() => servers.value.map((s) => String(s.tag)).filter(Boolean))

export const renderServers = computed(() => {
  const searchRegex = toSearchRegex(dnsFilter.value)
  if (!searchRegex) return servers.value

  return servers.value.filter((row) => {
    const targets = [
      row.tag,
      row.type,
      row.server,
      row.detour,
      row.interface,
      row.inet4_range,
      row.inet6_range,
    ]
    return searchRegex.testAny(targets)
  })
})

export const renderRules = computed(() => {
  const searchRegex = toSearchRegex(dnsFilter.value)
  if (!searchRegex) return rules.value

  return rules.value.filter((item) => {
    const r = item.rule || {}
    const targets = [
      item.id,
      r.type,
      r.action,
      r.server,
      r.tag,
      Array.isArray(r.domain) ? r.domain.join(' ') : r.domain,
      Array.isArray(r.rule_set) ? r.rule_set.join(' ') : r.rule_set,
      Array.isArray(r.ip_cidr) ? r.ip_cidr.join(' ') : r.ip_cidr,
    ]
    return searchRegex.testAny(targets)
  })
})

export function serverFieldKeys(type: string): string[] {
  switch (type) {
    case 'udp':
    case 'tcp':
      return ['server', 'server_port']
    case 'tls':
    case 'quic':
      return ['server', 'server_port', 'tls_server_name']
    case 'https':
    case 'h3':
      return ['server', 'server_port', 'tls_server_name', 'path']
    case 'fakeip':
      return ['inet4_range', 'inet6_range']
    case 'hosts':
      return ['path']
    case 'dhcp':
    case 'mdns':
      return ['interface']
    default:
      return []
  }
}

export const fetchDns = async () => {
  loading.value = true
  try {
    const data = await api.dns()
    servers.value = Array.isArray(data.servers) ? data.servers : []
    rules.value = Array.isArray(data.rules) ? data.rules : []
    const opts = data.options || {}
    optsForm.final = String(opts.final || '')
    optsForm.strategy = String(opts.strategy || '')
    optsForm.timeout = opts.timeout ? String(opts.timeout) : ''
    optsForm.disable_cache = !!opts.disable_cache
    optsForm.independent_cache = !!opts.independent_cache
    optsForm.reverse_mapping = !!opts.reverse_mapping
  } catch (e) {
    showToast((e as Error).message || '加载 DNS 配置失败', 'error')
  } finally {
    loading.value = false
  }
}

export const fetchTags = async () => {
  try {
    const [obs, ibs] = await Promise.all([api.outbounds(), api.inbounds()])
    outboundTags.value = obs.map((o) => String(o.tag)).filter(Boolean)
    inboundTags.value = ibs.map((i) => String(i.tag)).filter(Boolean)
  } catch {
    // 标签加载失败不阻断流程
  }
}

export const saveDnsServer = async (tag: string, payload: Record<string, any>, isEdit: boolean) => {
  const statusStore = useStatusStore()
  saving.value = true
  try {
    if (isEdit) {
      await api.updateDnsServer(tag, payload)
    } else {
      await api.createDnsServer(payload)
    }
    showToast('保存成功', 'success')
    await Promise.all([fetchDns(), statusStore.refresh()])
    return true
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
    return false
  } finally {
    saving.value = false
  }
}

export const deleteDnsServer = async (tag: string) => {
  const statusStore = useStatusStore()
  const first = await showConfirmDialog({
    title: '删除确认',
    message: `确定删除 DNS server「${tag}」？该操作不可恢复。`,
    confirmText: '删除',
    confirmButtonClass: 'btn-error',
  })
  if (!first.confirmed) return false

  try {
    await api.deleteDnsServer(tag)
    showToast('删除成功', 'success')
    await Promise.all([fetchDns(), statusStore.refresh()])
    return true
  } catch (e) {
    const err = e as Error & { references?: string[] }
    if (err.references?.length) {
      const second = await showConfirmDialog({
        title: '被引用确认',
        message: `DNS server「${tag}」被引用：${err.references.join('、')}\n删除后将自动清除这些引用。确认删除？`,
        confirmText: '确认删除',
        confirmButtonClass: 'btn-error',
      })
      if (!second.confirmed) return false
      try {
        await api.deleteDnsServer(tag, true)
        showToast('删除成功（已清除引用）', 'success')
        await Promise.all([fetchDns(), statusStore.refresh()])
        return true
      } catch (e2) {
        showToast((e2 as Error).message || '删除失败', 'error')
        return false
      }
    }
    showToast(err.message || '删除失败', 'error')
    return false
  }
}

export const saveDnsRule = async (id: string, payload: Record<string, any>, isEdit: boolean) => {
  const statusStore = useStatusStore()
  saving.value = true
  try {
    if (isEdit) {
      await api.updateDnsRule(id, payload)
    } else {
      await api.createDnsRule(payload)
    }
    showToast('保存成功', 'success')
    await Promise.all([fetchDns(), statusStore.refresh()])
    return true
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
    return false
  } finally {
    saving.value = false
  }
}

export const deleteDnsRule = async (id: string) => {
  const statusStore = useStatusStore()
  const { confirmed } = await showConfirmDialog({
    title: '删除确认',
    message: '确定删除这条 DNS 规则？该操作不可恢复。',
    confirmText: '删除',
    confirmButtonClass: 'btn-error',
  })
  if (!confirmed) return false

  try {
    await api.deleteDnsRule(id)
    showToast('删除成功', 'success')
    await Promise.all([fetchDns(), statusStore.refresh()])
    return true
  } catch (e) {
    showToast((e as Error).message || '删除失败', 'error')
    return false
  }
}

export const saveDnsOptions = async () => {
  const statusStore = useStatusStore()
  saving.value = true
  try {
    const patch: Record<string, unknown> = {}
    patch.final = optsForm.final || ''
    if (optsForm.strategy) patch.strategy = optsForm.strategy
    if (optsForm.timeout.trim()) patch.timeout = optsForm.timeout.trim()
    patch.disable_cache = optsForm.disable_cache
    patch.independent_cache = optsForm.independent_cache
    patch.reverse_mapping = optsForm.reverse_mapping
    await api.updateDnsOptions(patch)
    showToast('DNS 选项已保存', 'success')
    await Promise.all([fetchDns(), statusStore.refresh()])
    return true
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
    return false
  } finally {
    saving.value = false
  }
}
