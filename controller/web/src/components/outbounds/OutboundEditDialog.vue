<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useResizeObserver } from '@vueuse/core'
import { showToast } from '@/helper/toast'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { ClipboardPasteIcon, KeyRoundIcon, LoaderIcon } from '@lucide/vue'
import { EyeIcon, EyeSlashIcon } from '@heroicons/vue/24/outline'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import ChipInput from '@/components/common/ChipInput.vue'
import GroupMemberPicker from '@/components/outbounds/GroupMemberPicker.vue'
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { SelectField } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { api } from '@/api'
import ResourceSourceTab from '@/components/ResourceSourceTab.vue'
import type { Outbound } from '@/api'
import { proxyMap, fetchProxies } from '@/assembly/proxies'
import { useStatusStore } from '@/store/status'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    tag?: string
    isEdit?: boolean
  }>(),
  {
    tag: '',
    isEdit: false
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'saved', result: unknown): void
}>()

const statusStore = useStatusStore()

const outbounds = ref<Outbound[]>([])
const outboundTypes = ref<string[]>([])

const srcTab = ref<InstanceType<typeof ResourceSourceTab>>()
const sourceJson = ref('{}')
const saving = ref(false)
const generating = ref(false)
const showPassword = ref(false)
const validateMsg = ref('')
const activeTab = ref('form')
const formContentRef = ref<HTMLElement>()
const sharedTabHeight = ref<number>(440)

useResizeObserver(formContentRef, (entries) => {
  const entry = entries[0]
  if (entry && activeTab.value === 'form' && entry.contentRect.height > 60) {
    sharedTabHeight.value = Math.max(300, Math.round(entry.contentRect.height))
  }
})

// 确保在「源码」Tab 与「表单」Tab 尺寸高度无缝一致，杜绝任何缩小与跳动
const sourceEditorHeight = computed(() => {
  if (typeof window === 'undefined') return 440
  // 弹窗最大可用净高度（85vh 减去头部45 + 底部49 + Padding32 + Tabs36）
  const maxAvailable = Math.max(300, Math.floor(window.innerHeight * 0.85 - 170))
  // 严格贴合表单已撑开的实际高度，若表单超高处于滚动状态则填满当前视口，平滑无跳动
  return Math.min(sharedTabHeight.value, maxAvailable)
})

interface OutboundForm {
  type: string
  tag: string
  server: string
  server_port?: number
  uuid: string
  flow: string
  network: string
  password: string
  congestion_control: string
  udp_relay_mode: string
  zero_rtt_handshake: boolean
  tls: {
    enabled: boolean
    server_name: string
    utls: { enabled: boolean; fingerprint: string }
    reality: { enabled: boolean; public_key: string; short_id: string }
    alpn: string[]
  }
  outbounds: string[]
  default_out: string
  url: string
  interval: string
  tolerance?: number
  interrupt_exist_connections: boolean
  attach_groups: string[]
}

const form = reactive<OutboundForm>({
  type: '',
  tag: '',
  server: '',
  server_port: undefined,
  uuid: '',
  flow: '',
  network: 'none',
  password: '',
  congestion_control: 'bbr',
  udp_relay_mode: 'native',
  zero_rtt_handshake: false,
  tls: {
    enabled: false,
    server_name: '',
    utls: { enabled: true, fingerprint: 'chrome' },
    reality: { enabled: false, public_key: '', short_id: '' },
    alpn: ['h3']
  },
  outbounds: [],
  default_out: '',
  url: '',
  interval: '',
  tolerance: undefined,
  interrupt_exist_connections: false,
  attach_groups: []
})

const networkOptions = computed(() => [
  { value: 'none', label: t('outboundsDialog.networkNone') },
  { value: 'tcp', label: 'tcp' },
  { value: 'udp', label: 'udp' },
  { value: 'ws', label: 'ws' },
  { value: 'grpc', label: 'grpc' },
  { value: 'h2', label: 'h2' },
  { value: 'httpupgrade', label: 'httpupgrade' },
  { value: 'xhttp', label: 'xhttp' }
])
const fingerprintOptions = ['chrome', 'firefox', 'edge', 'ios', 'android', 'random']
const congestionOptions = ['bbr', 'cubic', 'new_reno']
const udpRelayOptions = ['native', 'quic']
const alpnOptions = ['h3', 'h2', 'http/1.1']

const isVless = computed(() => form.type === 'vless')
const isTuic = computed(() => form.type === 'tuic')
const isSelector = computed(() => form.type === 'selector')
const isUrlTest = computed(() => form.type === 'urltest')

const groupCandidates = computed(() =>
  outbounds.value.map((o) => String(o.tag)).filter((t) => t && t !== form.tag)
)

// 当前可加入的代理分组（selector 或 urltest，且排除自身）
const availableProxyGroups = computed(() =>
  outbounds.value
    .filter((o) => (o.type === 'selector' || o.type === 'urltest') && o.tag && o.tag !== form.tag)
    .map((o) => String(o.tag))
)

// 默认自动勾选策略：首个或名为 Proxy / 节点选择 的常用分组，默认优先给首个 selector
function getDefaultSelectedGroups(allOutbounds: Outbound[]): string[] {
  const groups = allOutbounds.filter(
    (o) => (o.type === 'selector' || o.type === 'urltest') && o.tag
  )
  if (!groups.length) return []

  const commonNames = ['Proxy', '节点选择', 'PROXY', 'proxy', '节点']
  const matchedCommon = groups.find((g) => commonNames.includes(String(g.tag)))
  if (matchedCommon && matchedCommon.tag) {
    return [String(matchedCommon.tag)]
  }

  const firstSelector = groups.find((g) => g.type === 'selector')
  if (firstSelector && firstSelector.tag) {
    return [String(firstSelector.tag)]
  }

  if (groups[0]?.tag) {
    return [String(groups[0].tag)]
  }
  return []
}

const networkOptionsAll = computed(() => {
  const opts = networkOptions.value
  const has = opts.some((o) => (typeof o === 'string' ? o : o.value) === form.network)
  return form.network && !has ? [...opts, { value: form.network, label: form.network }] : opts
})

const flowOptions = computed(() => [
  { value: 'none', label: t('outboundsDialog.flowNone') },
  { value: 'xtls-rprx-vision', label: 'xtls-rprx-vision' },
  { value: 'xtls-rprx-vision-udp443', label: 'xtls-rprx-vision-udp443' }
])

function str(v: unknown): string {
  return typeof v === 'string' ? v : ''
}

function num(v: unknown): number | undefined {
  return typeof v === 'number' ? v : undefined
}

function validateForm(): string {
  if (!form.tag.trim()) return t('outboundsDialog.tagRequired')
  if (!isSelector.value && !isUrlTest.value) {
    if (!form.server.trim()) return t('outboundsDialog.serverRequired')
    if (typeof form.server_port !== 'number' || form.server_port < 1 || form.server_port > 65535) {
      return t('outboundsDialog.serverPortRequired')
    }
  }
  if (isVless.value || isTuic.value) {
    if (!form.uuid.trim()) return t('outboundsDialog.uuidRequired')
    if (isTuic.value && !form.password) return t('outboundsDialog.passwordRequired')
  }
  if (isSelector.value || isUrlTest.value) {
    if (!form.outbounds.length) return t('outboundsDialog.memberRequired')
  }
  if (form.tls.enabled && !form.tls.server_name.trim()) return t('outboundsDialog.serverNameRequired')
  if (form.tls.reality.enabled && !form.tls.reality.public_key.trim()) return t('outboundsDialog.realityKeyRequired')
  return ''
}

function resetForm(type: string) {
  form.type = type
  form.tag = ''
  form.server = ''
  form.server_port = undefined
  form.uuid = ''
  form.flow = ''
  form.network = 'none'
  form.password = ''
  form.congestion_control = 'bbr'
  form.udp_relay_mode = 'native'
  form.zero_rtt_handshake = false
  form.tls = {
    enabled: false,
    server_name: '',
    utls: { enabled: true, fingerprint: 'chrome' },
    reality: { enabled: false, public_key: '', short_id: '' },
    alpn: ['h3']
  }
  form.outbounds = []
  form.default_out = ''
  form.url = ''
  form.interval = ''
  form.tolerance = undefined
  form.interrupt_exist_connections = false
  form.attach_groups = []
}

function fillForm(obj: Outbound) {
  const tls = (obj.tls ?? {}) as Record<string, unknown>
  const utls = (tls.utls ?? {}) as Record<string, unknown>
  const reality = (tls.reality ?? {}) as Record<string, unknown>
  const transport = (obj.transport ?? {}) as Record<string, unknown>
  form.type = str(obj.type)
  form.tag = str(obj.tag)
  form.server = str(obj.server)
  form.server_port = num(obj.server_port)
  form.uuid = str(obj.uuid)
  form.flow = str(obj.flow)
  let net = ''
  if (Array.isArray(obj.network)) {
    net = obj.network.length === 1 ? String(obj.network[0]) : 'none'
  } else if (typeof obj.network === 'string' && obj.network.trim()) {
    net = obj.network.trim()
  }
  const transportType = str(transport.type)
  form.network = transportType || net || 'none'
  form.password = str(obj.password)
  form.congestion_control = str(obj.congestion_control) || 'bbr'
  form.udp_relay_mode = str(obj.udp_relay_mode) || 'native'
  form.zero_rtt_handshake = !!obj.zero_rtt_handshake
  form.tls = {
    enabled: !!tls.enabled,
    server_name: str(tls.server_name),
    utls: { enabled: !!utls.enabled, fingerprint: str(utls.fingerprint) || 'chrome' },
    reality: {
      enabled: !!reality.enabled,
      public_key: str(reality.public_key),
      short_id: str(reality.short_id)
    },
    alpn: Array.isArray(tls.alpn) ? tls.alpn.map(String) : ['h3']
  }
  if (form.type === 'selector' || form.type === 'urltest') {
    form.outbounds = Array.isArray(obj.outbounds) ? obj.outbounds.map(String) : []
    form.default_out = str(obj.default)
    form.url = str(obj.url)
    form.interval = str(obj.interval)
    form.tolerance = num(obj.tolerance)
    form.interrupt_exist_connections = !!obj.interrupt_exist_connections
  }
  if (Array.isArray(obj.attach_groups)) {
    form.attach_groups = obj.attach_groups.map(String)
  } else {
    form.attach_groups = []
  }
}

const loadTypesAndCandidates = async () => {
  try {
    const [tTypes, obs] = await Promise.all([
      api.types(),
      api.outbounds(),
      Object.keys(proxyMap.value).length === 0 ? fetchProxies(true).catch(() => {}) : Promise.resolve(),
    ])
    outboundTypes.value = tTypes.outbounds || []
    outbounds.value = obs
  } catch {
    // ignore
  }
}

watch(
  () => form.outbounds,
  (newOutbounds) => {
    if (form.default_out && !newOutbounds.includes(form.default_out)) {
      form.default_out = newOutbounds[0] || ''
    }
  },
  { deep: true },
)

const openCreate = () => {
  sourceJson.value = '{}'
  const def = statusStore.status?.defaults?.outbound_type
  resetForm(def && outboundTypes.value.includes(def) ? def : outboundTypes.value[0] || 'vless')
  form.attach_groups = getDefaultSelectedGroups(outbounds.value)
  validateMsg.value = ''
  activeTab.value = 'form'
}

const openEdit = async (targetTag: string) => {
  validateMsg.value = ''
  activeTab.value = 'form'
  try {
    const data = await api.getOutbound(targetTag)
    sourceJson.value = JSON.stringify(data, null, 2)
    fillForm(data)
  } catch (e) {
    showToast((e as Error).message || t('outboundsDialog.loadConfigFailed'), 'error')
    emit('update:modelValue', false)
  }
}

watch(
  () => props.modelValue,
  async (visible) => {
    if (!visible) return
    await loadTypesAndCandidates()
    if (props.isEdit && props.tag) {
      await openEdit(props.tag)
    } else {
      openCreate()
    }
  },
  { immediate: true }
)

// 类型切换时确保默认必要字段存在，但不抹除用户已填的 tag/server
watch(
  () => form.type,
  (newType) => {
    if (!newType) return
    if (!form.network) form.network = 'none'
    if (newType === 'tuic') {
      if (!form.congestion_control) form.congestion_control = 'bbr'
      if (!form.udp_relay_mode) form.udp_relay_mode = 'native'
      if (!form.tls.alpn.length) form.tls.alpn = ['h3']
    }
    if (newType === 'urltest') {
      if (!form.url) form.url = 'https://www.gstatic.com/generate_204'
      if (!form.interval) form.interval = '3m'
    }
  }
)

// 切换至「源码」Tab 时，若源码未被脏改，自动同步表单最新的配置数据
watch(activeTab, (tab) => {
  if (tab === 'source' && !srcTab.value?.isDirty()) {
    try {
      sourceJson.value = JSON.stringify(buildBody(), null, 2)
    } catch {
      // ignore
    }
  }
})

// 从 JSON / 节点链接导入弹窗
const nodeImportDialog = ref(false)
const nodeImportText = ref('')
const fillFromNode = () => {
  nodeImportText.value = ''
  nodeImportDialog.value = true
}
const doFillFromNode = async () => {
  try {
    const res = await api.parseNode(nodeImportText.value)
    fillForm(res.data as Outbound)
    if (!form.attach_groups.length) {
      form.attach_groups = getDefaultSelectedGroups(outbounds.value)
    }
    nodeImportDialog.value = false
    showToast(t('outboundsDialog.parseAndFillSuccess'), 'success')
  } catch (e) {
    showToast((e as Error).message || t('outboundsDialog.parseFailed'), 'error')
  }
}

function buildBody(): Outbound {
  const body: Outbound = { type: form.type, tag: form.tag.trim() }
  if (form.server.trim()) body.server = form.server.trim()
  if (typeof form.server_port === 'number') body.server_port = form.server_port

  if (!isSelector.value && !isUrlTest.value) {
    body.attach_groups = [...form.attach_groups]
  }

  if (form.type === 'vless') {
    if (form.uuid.trim()) body.uuid = form.uuid.trim()
    if (form.flow.trim() && form.flow !== 'none') body.flow = form.flow.trim()
    if (form.network.trim() && form.network !== 'none') body.network = form.network.trim()
    body.tls = {
      enabled: form.tls.enabled,
      server_name: form.tls.server_name.trim(),
      utls: { enabled: form.tls.utls.enabled, fingerprint: form.tls.utls.fingerprint },
      reality: {
        enabled: form.tls.reality.enabled,
        public_key: form.tls.reality.public_key.trim(),
        short_id: form.tls.reality.short_id.trim()
      }
    }
    return body
  }

  if (form.type === 'tuic') {
    if (form.uuid.trim()) body.uuid = form.uuid.trim()
    if (form.password) body.password = form.password
    body.congestion_control = form.congestion_control
    body.udp_relay_mode = form.udp_relay_mode
    body.zero_rtt_handshake = form.zero_rtt_handshake
    body.tls = {
      enabled: form.tls.enabled,
      server_name: form.tls.server_name.trim(),
      alpn: form.tls.alpn.length ? [...form.tls.alpn] : ['h3']
    }
    return body
  }

  if (form.type === 'selector' || form.type === 'urltest') {
    if (form.outbounds.length) body.outbounds = [...form.outbounds]
    if (form.type === 'selector') {
      if (form.default_out) body.default = form.default_out
    } else {
      if (form.url.trim()) body.url = form.url.trim()
      if (form.interval.trim()) body.interval = form.interval.trim()
      if (typeof form.tolerance === 'number') body.tolerance = form.tolerance
    }
    body.interrupt_exist_connections = form.interrupt_exist_connections
    return body
  }

  return body
}

const handleResult = (res: unknown) => {
  const r = (res ?? {}) as { reload_error?: string; message?: string }
  if (r.reload_error) {
    showToast(r.message || t('outboundsDialog.saveSuccessReloadFailed', { error: r.reload_error }), 'warning')
  } else {
    showToast(t('outboundsDialog.saveSuccess'), 'success')
  }
}

const save = async () => {
  validateMsg.value = validateForm()
  if (validateMsg.value) return
  saving.value = true
  statusStore.markReloading(4000)
  try {
    const body = srcTab.value?.isDirty() ? JSON.parse(srcTab.value.getText()) : buildBody()
    const res = props.isEdit ? await api.updateOutbound(props.tag || form.tag, body) : await api.createOutbound(body)
    handleResult(res)
    emit('update:modelValue', false)
    emit('saved', res)
    await statusStore.refresh()
  } catch (e) {
    statusStore.clearReloading()
    showToast((e as Error).message || t('outboundsDialog.operationFailed'), 'error')
  } finally {
    saving.value = false
  }
}

const handleDelete = async () => {
  const targetTag = props.tag || form.tag
  if (!targetTag) return

  const first = await showConfirmDialog({
    title: t('outboundsDialog.deleteConfirmTitle'),
    message: t('outboundsDialog.deleteConfirmMsg', { tag: targetTag }),
    confirmText: t('outboundsDialog.deleteBtn'),
    confirmButtonClass: 'btn-error'
  })
  if (!first.confirmed) return

  saving.value = true
  statusStore.markReloading(4000)
  try {
    await api.deleteOutbound(targetTag)
    showToast(t('outboundsDialog.deleteSuccess'), 'success')
    emit('update:modelValue', false)
    emit('saved', null)
    await statusStore.refresh()
  } catch (e) {
    const err = e as Error & { references?: string[] }
    if (err.references?.length) {
      const second = await showConfirmDialog({
        title: t('outboundsDialog.referencedTitle'),
        message: t('outboundsDialog.referencedMsg', { references: err.references.join('、') }),
        confirmText: t('outboundsDialog.forceDelete'),
        confirmButtonClass: 'btn-error'
      })
      if (!second.confirmed) return
      try {
        statusStore.markReloading(4000)
        await api.deleteOutbound(targetTag, true)
        showToast(t('outboundsDialog.deleteSuccessCleanedRef'), 'success')
        emit('update:modelValue', false)
        emit('saved', null)
        await statusStore.refresh()
      } catch (e2) {
        statusStore.clearReloading()
        showToast((e2 as Error).message || t('outboundsDialog.deleteFailed'), 'error')
      }
      return
    }
    statusStore.clearReloading()
    showToast(err.message || t('outboundsDialog.deleteFailed'), 'error')
  } finally {
    saving.value = false
  }
}

const genUuid = async () => {
  generating.value = true
  try {
    form.uuid = await api.genUuid()
  } catch (e) {
    showToast((e as Error).message || t('outboundsDialog.uuidGenFailed'), 'error')
  } finally {
    generating.value = false
  }
}

const genKeypair = async () => {
  generating.value = true
  try {
    const kp = await api.genRealityKeypair()
    form.tls.reality.public_key = kp.public_key
    await showConfirmDialog({
      title: t('outboundsDialog.realityKeypairGenTitle'),
      message: t('outboundsDialog.realityKeypairGenMsg', { privateKey: kp.private_key }),
      confirmText: t('outboundsDialog.savedKeypairAck')
    })
  } catch (e) {
    showToast((e as Error).message || t('outboundsDialog.keypairGenFailed'), 'error')
  } finally {
    generating.value = false
  }
}

const genPassword = async () => {
  generating.value = true
  try {
    form.password = await api.genPassword()
  } catch (e) {
    showToast((e as Error).message || t('outboundsDialog.passwordGenFailed'), 'error')
  } finally {
    generating.value = false
  }
}

onMounted(() => {
  void loadTypesAndCandidates()
})
</script>

<template>
  <div>
    <!-- 新建/编辑弹窗 -->
    <DialogWrapper
      :model-value="modelValue"
      :title="isEdit ? $t('outboundsDialog.editTitle') : $t('outboundsDialog.createTitle')"
      box-class="max-w-[720px]"
      :body-class="activeTab === 'source' ? 'overflow-hidden' : 'overflow-y-auto'"
      @update:model-value="(val) => emit('update:modelValue', !!val)"
    >
      <TabsRoot v-model="activeTab">
        <TabsList class="self-start shrink-0">
          <TabsTrigger value="form">{{ $t('form') }}</TabsTrigger>
          <TabsTrigger value="source">{{ $t('source') }}</TabsTrigger>
        </TabsList>
        <TabsContent value="form">
          <div ref="formContentRef">
            <div class="mb-4">
              <button class="btn btn-ghost btn-xs" @click="fillFromNode">
                <ClipboardPasteIcon class="h-3.5 w-3.5" />
                {{ $t('outboundsDialog.importFromNode') }}
              </button>
              <span class="ml-2 text-xs text-base-content/60">{{ $t('outboundsDialog.importTip') }}</span>
            </div>
            <div class="grid gap-x-4 gap-y-4" style="grid-template-columns: 170px minmax(0, 1fr)">
            <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.type') }}</label>
            <template v-if="outboundTypes.length">
              <!-- 类型允许随时修改 -->
              <SelectField v-model="form.type" :options="outboundTypes" />
            </template>
            <div v-else class="alert alert-warning py-2 text-xs">{{ $t('outboundsDialog.noBackendWarn') }}</div>

            <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.tag') }} <span class="text-destructive">*</span></label>
            <input v-model="form.tag" type="text" class="input input-bordered input-sm w-full" :placeholder="$t('outboundsDialog.tagPlaceholder')" />

            <!-- server / server_port 区域（带展开折叠动画） -->
            <div class="col-span-2 collapse-row" :class="{ 'is-open': !isSelector && !isUrlTest }">
              <div class="collapse-inner">
                <div class="grid gap-x-4 gap-y-4" style="grid-template-columns: 170px minmax(0, 1fr)">
                  <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.server') }} <span class="text-destructive">*</span></label>
                  <input v-model="form.server" type="text" class="input input-bordered input-sm w-full" :placeholder="$t('outboundsDialog.serverPlaceholder')" />
                  <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.serverPort') }} <span class="text-destructive">*</span></label>
                  <input v-model.number="form.server_port" type="number" min="1" max="65535" class="input input-bordered input-sm w-40" />
                </div>
              </div>
            </div>

            <!-- vless 专属字段区域（带展开折叠动画） -->
            <div class="col-span-2 collapse-row" :class="{ 'is-open': isVless }">
              <div class="collapse-inner">
                <div class="grid gap-x-4 gap-y-4" style="grid-template-columns: 170px minmax(0, 1fr)">
                  <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.uuid') }} <span class="text-destructive">*</span></label>
                  <div class="flex gap-2">
                    <input v-model="form.uuid" type="text" class="input input-bordered input-sm w-full font-mono" placeholder="UUID" />
                    <button class="btn btn-ghost btn-sm shrink-0" :disabled="generating" @click="genUuid">
                      <KeyRoundIcon class="h-4 w-4" />
                      {{ $t('outboundsDialog.generate') }}
                    </button>
                  </div>
                  <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.flow') }}</label>
                  <SelectField
                    v-model="form.flow"
                    :options="flowOptions"
                    :placeholder="$t('outboundsDialog.flowPlaceholder')"
                  />
                  <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.network') }}</label>
                  <SelectField v-model="form.network" :options="networkOptionsAll" />

                  <div class="flex items-center gap-2 text-[13px] font-semibold text-base-content" style="grid-column: 1 / -1">
                    <span class="h-px flex-1 bg-base-300" />{{ $t('outboundsDialog.tls') }}<span class="h-px flex-1 bg-base-300" />
                  </div>
                  <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.tlsEnabled') }}</label>
                  <Switch v-model="form.tls.enabled" class="self-center" />

                  <!-- TLS 展开内容（嵌套动画） -->
                  <div class="col-span-2 collapse-row" :class="{ 'is-open': form.tls.enabled }">
                    <div class="collapse-inner">
                      <div class="grid gap-x-4 gap-y-4 pt-2" style="grid-template-columns: 170px minmax(0, 1fr)">
                        <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.serverName') }} <span class="text-destructive">*</span></label>
                        <input v-model="form.tls.server_name" type="text" class="input input-bordered input-sm w-full" :placeholder="$t('outboundsDialog.serverNamePlaceholder')" />
                        <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.utlsEnabled') }}</label>
                        <Switch v-model="form.tls.utls.enabled" class="self-center" />

                        <!-- UTLS 展开内容 -->
                        <div class="col-span-2 collapse-row" :class="{ 'is-open': form.tls.utls.enabled }">
                          <div class="collapse-inner">
                            <div class="grid gap-x-4 gap-y-4" style="grid-template-columns: 170px minmax(0, 1fr)">
                              <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.utlsFingerprint') }}</label>
                              <SelectField v-model="form.tls.utls.fingerprint" :options="fingerprintOptions" />
                            </div>
                          </div>
                        </div>

                        <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.realityEnabled') }}</label>
                        <Switch v-model="form.tls.reality.enabled" class="self-center" />

                        <!-- Reality 展开内容 -->
                        <div class="col-span-2 collapse-row" :class="{ 'is-open': form.tls.reality.enabled }">
                          <div class="collapse-inner">
                            <div class="grid gap-x-4 gap-y-4" style="grid-template-columns: 170px minmax(0, 1fr)">
                              <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.realityPublicKey') }} <span class="text-destructive">*</span></label>
                              <div class="flex gap-2">
                                <input v-model="form.tls.reality.public_key" type="text" class="input input-bordered input-sm w-full font-mono" :placeholder="$t('outboundsDialog.realityPublicKeyPlaceholder')" />
                                <button class="btn btn-ghost btn-sm shrink-0" :disabled="generating" @click="genKeypair">
                                  <LoaderIcon class="h-4 w-4" />
                                  {{ $t('outboundsDialog.genKeypair') }}
                                </button>
                              </div>
                              <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.realityShortId') }}</label>
                              <input v-model="form.tls.reality.short_id" type="text" class="input input-bordered input-sm w-full" :placeholder="$t('outboundsDialog.realityShortIdPlaceholder')" />
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- tuic 专属字段区域（带展开折叠动画） -->
            <div class="col-span-2 collapse-row" :class="{ 'is-open': isTuic }">
              <div class="collapse-inner">
                <div class="grid gap-x-4 gap-y-4" style="grid-template-columns: 170px minmax(0, 1fr)">
                  <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.uuid') }} <span class="text-destructive">*</span></label>
                  <div class="flex gap-2">
                    <input v-model="form.uuid" type="text" class="input input-bordered input-sm w-full font-mono" placeholder="UUID" />
                    <button class="btn btn-ghost btn-sm shrink-0" :disabled="generating" @click="genUuid">
                      <KeyRoundIcon class="h-4 w-4" />
                      {{ $t('outboundsDialog.generate') }}
                    </button>
                  </div>
                  <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.password') }} <span class="text-destructive">*</span></label>
                  <div class="flex gap-2">
                    <input
                      v-model="form.password"
                      :type="showPassword ? 'text' : 'password'"
                      class="input input-bordered input-sm w-full font-mono"
                      :placeholder="$t('outboundsDialog.passwordPlaceholder')"
                    />
                    <button
                      type="button"
                      class="btn btn-ghost btn-sm shrink-0"
                      :title="showPassword ? $t('outboundsDialog.hidePassword') : $t('outboundsDialog.showPassword')"
                      @click="showPassword = !showPassword"
                    >
                      <EyeSlashIcon v-if="showPassword" class="h-4 w-4" />
                      <EyeIcon v-else class="h-4 w-4" />
                    </button>
                    <button
                      type="button"
                      class="btn btn-ghost btn-sm shrink-0"
                      :disabled="generating"
                      @click="genPassword"
                    >
                      <KeyRoundIcon class="h-4 w-4" />
                      {{ $t('outboundsDialog.generate') }}
                    </button>
                  </div>
                  <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.congestionControl') }}</label>
                  <SelectField v-model="form.congestion_control" :options="congestionOptions" />
                  <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.udpRelayMode') }}</label>
                  <SelectField v-model="form.udp_relay_mode" :options="udpRelayOptions" />
                  <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.zeroRttHandshake') }}</label>
                  <Switch v-model="form.zero_rtt_handshake" class="self-center" />
                  <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.tlsEnabled') }}</label>
                  <Switch v-model="form.tls.enabled" class="self-center" />

                  <!-- TUIC TLS 展开内容 -->
                  <div class="col-span-2 collapse-row" :class="{ 'is-open': form.tls.enabled }">
                    <div class="collapse-inner">
                      <div class="grid gap-x-4 gap-y-4 pt-2" style="grid-template-columns: 170px minmax(0, 1fr)">
                        <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.serverName') }} <span class="text-destructive">*</span></label>
                        <input v-model="form.tls.server_name" type="text" class="input input-bordered input-sm w-full" :placeholder="$t('outboundsDialog.serverNamePlaceholder')" />
                        <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.alpn') }}</label>
                        <ChipInput v-model="form.tls.alpn" :suggestions="alpnOptions" :placeholder="$t('outboundsDialog.alpnPlaceholder')" />
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- selector / urltest 组字段区域（带展开折叠动画） -->
            <div class="col-span-2 collapse-row" :class="{ 'is-open': isSelector || isUrlTest }">
              <div class="collapse-inner">
                <div class="grid gap-x-4 gap-y-4" style="grid-template-columns: 170px minmax(0, 1fr)">
                  <label class="pt-1 text-sm font-medium text-base-content/80 self-start">{{ $t('outboundsDialog.memberOutbounds') }} <span class="text-destructive">*</span></label>
                  <GroupMemberPicker
                    v-model="form.outbounds"
                    :exclude-tag="form.tag || props.tag"
                    :group-name="props.tag || form.tag"
                  />
                  <template v-if="isSelector">
                    <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.defaultOutbound') }}</label>
                    <SelectField v-model="form.default_out" :options="form.outbounds" :placeholder="$t('outboundsDialog.defaultOutboundPlaceholder')" />
                  </template>
                  <template v-if="isUrlTest">
                    <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.testUrl') }}</label>
                    <input v-model="form.url" type="text" class="input input-bordered input-sm w-full" :placeholder="$t('outboundsDialog.testUrlPlaceholder')" />
                    <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.interval') }}</label>
                    <input v-model="form.interval" type="text" class="input input-bordered input-sm w-full" :placeholder="$t('outboundsDialog.intervalPlaceholder')" />
                    <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.tolerance') }}</label>
                    <input v-model.number="form.tolerance" type="number" min="0" max="65535" class="input input-bordered input-sm w-40" />
                  </template>
                  <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.interruptExistConnections') }}</label>
                  <Switch v-model="form.interrupt_exist_connections" class="self-center" />
                </div>
              </div>
            </div>

            <!-- 其他类型兜底提示 -->
            <div class="col-span-2 collapse-row" :class="{ 'is-open': !isVless && !isTuic && !isSelector && !isUrlTest }">
              <div class="collapse-inner">
                <div class="grid gap-x-4 gap-y-4" style="grid-template-columns: 170px minmax(0, 1fr)">
                  <label class="pt-2 text-sm text-base-content/70">{{ $t('outboundsDialog.otherFields') }}</label>
                  <p class="text-xs leading-relaxed text-base-content/60">{{ $t('outboundsDialog.otherFieldsTip') }}</p>
                </div>
              </div>
            </div>

            <!-- 加入分组 (Proxy Groups) 多选区域 -->
            <div class="col-span-2 collapse-row" :class="{ 'is-open': !isSelector && !isUrlTest }">
              <div class="collapse-inner">
                <div class="grid gap-x-4 gap-y-2 pt-3 border-t border-base-200" style="grid-template-columns: 170px minmax(0, 1fr)">
                  <div class="flex flex-col gap-1">
                    <label class="pt-1 text-sm font-medium text-base-content/80">
                      {{ $t('outboundsDialog.attachGroups') }}
                    </label>
                    <span class="text-xs text-base-content/50 leading-tight">
                      {{ $t('outboundsDialog.attachGroupsTip') }}
                    </span>
                  </div>

                  <div class="pt-1">
                    <div v-if="availableProxyGroups.length" class="flex flex-wrap items-center gap-2">
                      <label
                        v-for="grp in availableProxyGroups"
                        :key="grp"
                        class="flex items-center gap-2 px-3 py-1.5 rounded-lg border text-xs cursor-pointer select-none transition-all duration-150"
                        :class="form.attach_groups.includes(grp)
                          ? 'border-primary/50 bg-primary/10 text-primary font-medium shadow-xs'
                          : 'border-base-300 hover:border-base-content/30 bg-base-100 text-base-content/80 hover:bg-base-200/50'"
                      >
                        <input
                          type="checkbox"
                          :value="grp"
                          v-model="form.attach_groups"
                          class="checkbox checkbox-primary checkbox-xs rounded"
                        />
                        <span class="font-mono tracking-tight">{{ grp }}</span>
                      </label>
                    </div>
                    <div v-else class="text-xs text-base-content/50 py-1.5">
                      {{ $t('outboundsDialog.noAvailableGroups') }}
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <p v-if="validateMsg" class="mt-3 text-[0.8rem] font-medium text-destructive">{{ validateMsg }}</p>
          </div>
        </div>
      </TabsContent>
        <TabsContent value="source">
          <ResourceSourceTab ref="srcTab" :initial="sourceJson" :height="sourceEditorHeight" />
        </TabsContent>
      </TabsRoot>
      <template #footer>
        <div class="flex items-center justify-between w-full">
          <div>
            <button
              v-if="isEdit"
              type="button"
              class="btn btn-error btn-outline btn-sm"
              :disabled="saving"
              @click="handleDelete"
            >
              {{ $t('outboundsDialog.deleteBtn') }}
            </button>
          </div>
          <div class="flex justify-end gap-2">
            <button class="btn btn-ghost btn-sm" @click="emit('update:modelValue', false)">{{ $t('cancel') }}</button>
            <button class="btn btn-primary btn-sm" :disabled="saving" @click="save">{{ $t('save') }}</button>
          </div>
        </div>
      </template>
    </DialogWrapper>

    <!-- 从 JSON / 节点链接导入弹窗 -->
    <DialogWrapper v-model="nodeImportDialog" :title="$t('outboundsDialog.importDialogTitle')" box-class="max-w-[560px]">
      <textarea
        v-model="nodeImportText"
        rows="10"
        class="textarea textarea-bordered w-full font-mono text-xs"
        :placeholder="$t('outboundsDialog.importDialogPlaceholder')"
      />
      <p class="mt-1 text-xs text-base-content/60">{{ $t('outboundsDialog.importDialogTip') }}</p>
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <button class="btn btn-ghost btn-sm" @click="nodeImportDialog = false">{{ $t('cancel') }}</button>
          <button class="btn btn-primary btn-sm" @click="doFillFromNode">{{ $t('outboundsDialog.parseAndFill') }}</button>
        </div>
      </template>
    </DialogWrapper>
  </div>
</template>

<style scoped>
.collapse-row {
  display: grid;
  grid-template-rows: 0fr;
  opacity: 0;
  transform: translateY(-2px);
  transition: grid-template-rows 0.28s cubic-bezier(0.4, 0, 0.2, 1),
              opacity 0.22s cubic-bezier(0.4, 0, 0.2, 1),
              transform 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}

.collapse-row.is-open {
  grid-template-rows: 1fr;
  opacity: 1;
  transform: translateY(0);
}

.collapse-inner {
  overflow: hidden;
}

@media (prefers-reduced-motion: reduce) {
  .collapse-row {
    transition: none;
  }
}
</style>
