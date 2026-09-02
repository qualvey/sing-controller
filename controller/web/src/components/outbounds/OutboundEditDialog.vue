<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { showToast } from '@/helper/toast'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { ClipboardPasteIcon, KeyRoundIcon, LoaderIcon } from '@lucide/vue'
import { EyeIcon, EyeSlashIcon } from '@heroicons/vue/24/outline'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import ChipInput from '@/components/common/ChipInput.vue'
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { SelectField } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { api } from '@/api'
import ResourceSourceTab from '@/components/ResourceSourceTab.vue'
import type { Outbound } from '@/api'
import { useStatusStore } from '@/store/status'

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
}

const form = reactive<OutboundForm>({
  type: '',
  tag: '',
  server: '',
  server_port: undefined,
  uuid: '',
  flow: '',
  network: 'tcp',
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
  interrupt_exist_connections: false
})

const networkOptions = ['tcp', 'udp', 'ws', 'grpc', 'h2', 'httpupgrade', 'xhttp']
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

const networkOptionsAll = computed(() =>
  form.network && !networkOptions.includes(form.network) ? [...networkOptions, form.network] : networkOptions
)

function str(v: unknown): string {
  return typeof v === 'string' ? v : ''
}

function num(v: unknown): number | undefined {
  return typeof v === 'number' ? v : undefined
}

function validateForm(): string {
  if (!form.tag.trim()) return 'tag 必填'
  if (!isSelector.value && !isUrlTest.value) {
    if (!form.server.trim()) return 'server 必填'
    if (typeof form.server_port !== 'number' || form.server_port < 1 || form.server_port > 65535) {
      return 'server_port 必填（1-65535）'
    }
  }
  if (isVless.value || isTuic.value) {
    if (!form.uuid.trim()) return 'uuid 必填'
    if (isTuic.value && !form.password) return 'password 必填'
  }
  if (isSelector.value || isUrlTest.value) {
    if (!form.outbounds.length) return '至少选择一个成员 outbound'
  }
  if (form.tls.enabled && !form.tls.server_name.trim()) return '启用 TLS 时 server_name 必填'
  if (form.tls.reality.enabled && !form.tls.reality.public_key.trim()) return '启用 Reality 时 public_key 必填'
  return ''
}

function resetForm(type: string) {
  form.type = type
  form.tag = ''
  form.server = ''
  form.server_port = undefined
  form.uuid = ''
  form.flow = ''
  form.network = 'tcp'
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
  form.network = str(transport.type) || str(obj.network) || 'tcp'
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
}

const loadTypesAndCandidates = async () => {
  try {
    const [t, obs] = await Promise.all([api.types(), api.outbounds()])
    outboundTypes.value = t.outbounds || []
    outbounds.value = obs
  } catch {
    // ignore
  }
}

const openCreate = () => {
  sourceJson.value = '{}'
  const def = statusStore.status?.defaults?.outbound_type
  resetForm(def && outboundTypes.value.includes(def) ? def : outboundTypes.value[0] || 'vless')
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
    showToast((e as Error).message || '加载节点配置失败', 'error')
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
    if (!form.network) form.network = 'tcp'
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
    nodeImportDialog.value = false
    showToast('已解析并填充，请检查表单', 'success')
  } catch (e) {
    showToast((e as Error).message || '解析失败', 'error')
  }
}

function buildBody(): Outbound {
  const body: Outbound = { type: form.type, tag: form.tag.trim() }
  if (form.server.trim()) body.server = form.server.trim()
  if (typeof form.server_port === 'number') body.server_port = form.server_port

  if (form.type === 'vless') {
    if (form.uuid.trim()) body.uuid = form.uuid.trim()
    if (form.flow.trim()) body.flow = form.flow.trim()
    if (form.network.trim()) body.network = form.network.trim()
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
    showToast(r.message || `配置已保存，但实例重载失败：${r.reload_error}`, 'warning')
  } else {
    showToast('保存成功', 'success')
  }
}

const save = async () => {
  validateMsg.value = validateForm()
  if (validateMsg.value) return
  saving.value = true
  try {
    const body = srcTab.value?.isDirty() ? JSON.parse(srcTab.value.getText()) : buildBody()
    const res = props.isEdit ? await api.updateOutbound(props.tag || form.tag, body) : await api.createOutbound(body)
    handleResult(res)
    emit('update:modelValue', false)
    emit('saved', res)
    await statusStore.refresh()
  } catch (e) {
    showToast((e as Error).message || '操作失败', 'error')
  } finally {
    saving.value = false
  }
}

const handleDelete = async () => {
  const targetTag = props.tag || form.tag
  if (!targetTag) return

  const first = await showConfirmDialog({
    title: '删除确认',
    message: `确定删除节点「${targetTag}」？该操作不可恢复。`,
    confirmText: '删除',
    confirmButtonClass: 'btn-error'
  })
  if (!first.confirmed) return

  saving.value = true
  try {
    await api.deleteOutbound(targetTag)
    showToast('删除成功', 'success')
    emit('update:modelValue', false)
    emit('saved', null)
    await statusStore.refresh()
  } catch (e) {
    const err = e as Error & { references?: string[] }
    if (err.references?.length) {
      const second = await showConfirmDialog({
        title: '被引用确认',
        message: `该节点被以下对象引用：${err.references.join('、')}\n强制删除将自动从相关配置中移除引用。确认删除？`,
        confirmText: '强制删除',
        confirmButtonClass: 'btn-error'
      })
      if (!second.confirmed) return
      try {
        await api.deleteOutbound(targetTag, true)
        showToast('删除成功（已移除引用）', 'success')
        emit('update:modelValue', false)
        emit('saved', null)
        await statusStore.refresh()
      } catch (e2) {
        showToast((e2 as Error).message || '删除失败', 'error')
      }
      return
    }
    showToast(err.message || '删除失败', 'error')
  } finally {
    saving.value = false
  }
}

const genUuid = async () => {
  generating.value = true
  try {
    form.uuid = await api.genUuid()
  } catch (e) {
    showToast((e as Error).message || 'UUID 生成失败', 'error')
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
      title: 'Reality 密钥对已生成',
      message: `private_key（仅展示一次，请妥善保存）：\n\n${kp.private_key}\n\npublic_key 已自动填入表单。`,
      confirmText: '我已保存'
    })
  } catch (e) {
    showToast((e as Error).message || '密钥对生成失败', 'error')
  } finally {
    generating.value = false
  }
}

const genPassword = async () => {
  generating.value = true
  try {
    form.password = await api.genPassword()
  } catch (e) {
    showToast((e as Error).message || '密码生成失败', 'error')
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
      :title="isEdit ? '编辑 Outbound' : '新建 Outbound'"
      box-class="max-w-[720px]"
      @update:model-value="(val) => emit('update:modelValue', !!val)"
    >
      <TabsRoot v-model="activeTab">
        <TabsList>
          <TabsTrigger value="form">表单</TabsTrigger>
          <TabsTrigger value="source">源码</TabsTrigger>
        </TabsList>
        <TabsContent value="form">
          <div class="mb-4">
            <button class="btn btn-ghost btn-xs" @click="fillFromNode">
              <ClipboardPasteIcon class="h-3.5 w-3.5" />
              从 JSON / 节点链接导入
            </button>
            <span class="ml-2 text-xs text-[#909399]">支持粘贴 vless://、vmess://、ss:// 等链接或 JSON 节点配置</span>
          </div>
          <div class="grid gap-x-4 gap-y-4" style="grid-template-columns: 170px minmax(0, 1fr)">
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">类型</label>
            <template v-if="outboundTypes.length">
              <!-- 类型允许随时修改 -->
              <SelectField v-model="form.type" :options="outboundTypes" />
            </template>
            <div v-else class="alert alert-warning py-2 text-xs">当前没有后端连接，无法使用该功能</div>

            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tag <span class="text-destructive">*</span></label>
            <input v-model="form.tag" type="text" class="input input-bordered input-sm w-full" placeholder="唯一标识，如 vless-out" />

            <!-- server / server_port 区域（带展开折叠动画） -->
            <div class="col-span-2 collapse-row" :class="{ 'is-open': !isSelector && !isUrlTest }">
              <div class="collapse-inner">
                <div class="grid gap-x-4 gap-y-4" style="grid-template-columns: 170px minmax(0, 1fr)">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">server <span class="text-destructive">*</span></label>
                  <input v-model="form.server" type="text" class="input input-bordered input-sm w-full" placeholder="服务器地址 / IP" />
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">server_port <span class="text-destructive">*</span></label>
                  <input v-model.number="form.server_port" type="number" min="1" max="65535" class="input input-bordered input-sm w-40" />
                </div>
              </div>
            </div>

            <!-- vless 专属字段区域（带展开折叠动画） -->
            <div class="col-span-2 collapse-row" :class="{ 'is-open': isVless }">
              <div class="collapse-inner">
                <div class="grid gap-x-4 gap-y-4" style="grid-template-columns: 170px minmax(0, 1fr)">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">uuid <span class="text-destructive">*</span></label>
                  <div class="flex gap-2">
                    <input v-model="form.uuid" type="text" class="input input-bordered input-sm w-full font-mono" placeholder="UUID" />
                    <button class="btn btn-ghost btn-sm shrink-0" :disabled="generating" @click="genUuid">
                      <KeyRoundIcon class="h-4 w-4" />
                      生成
                    </button>
                  </div>
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">flow</label>
                  <SelectField
                    v-model="form.flow"
                    :options="[
                      { value: 'none', label: '（无）' },
                      { value: 'xtls-rprx-vision', label: 'xtls-rprx-vision' },
                      { value: 'xtls-rprx-vision-udp443', label: 'xtls-rprx-vision-udp443' }
                    ]"
                    placeholder="留空则不启用"
                  />
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">network</label>
                  <SelectField v-model="form.network" :options="networkOptionsAll" />

                  <div class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]" style="grid-column: 1 / -1">
                    <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />TLS<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
                  </div>
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.enabled</label>
                  <Switch v-model="form.tls.enabled" class="self-center" />

                  <!-- TLS 展开内容（嵌套动画） -->
                  <div class="col-span-2 collapse-row" :class="{ 'is-open': form.tls.enabled }">
                    <div class="collapse-inner">
                      <div class="grid gap-x-4 gap-y-4 pt-2" style="grid-template-columns: 170px minmax(0, 1fr)">
                        <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.server_name <span class="text-destructive">*</span></label>
                        <input v-model="form.tls.server_name" type="text" class="input input-bordered input-sm w-full" placeholder="SNI，如 www.example.com" />
                        <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.utls.enabled</label>
                        <Switch v-model="form.tls.utls.enabled" class="self-center" />

                        <!-- UTLS 展开内容 -->
                        <div class="col-span-2 collapse-row" :class="{ 'is-open': form.tls.utls.enabled }">
                          <div class="collapse-inner">
                            <div class="grid gap-x-4 gap-y-4" style="grid-template-columns: 170px minmax(0, 1fr)">
                              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.utls.fingerprint</label>
                              <SelectField v-model="form.tls.utls.fingerprint" :options="fingerprintOptions" />
                            </div>
                          </div>
                        </div>

                        <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.reality.enabled</label>
                        <Switch v-model="form.tls.reality.enabled" class="self-center" />

                        <!-- Reality 展开内容 -->
                        <div class="col-span-2 collapse-row" :class="{ 'is-open': form.tls.reality.enabled }">
                          <div class="collapse-inner">
                            <div class="grid gap-x-4 gap-y-4" style="grid-template-columns: 170px minmax(0, 1fr)">
                              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.reality.public_key <span class="text-destructive">*</span></label>
                              <div class="flex gap-2">
                                <input v-model="form.tls.reality.public_key" type="text" class="input input-bordered input-sm w-full font-mono" placeholder="Reality 公钥" />
                                <button class="btn btn-ghost btn-sm shrink-0" :disabled="generating" @click="genKeypair">
                                  <LoaderIcon class="h-4 w-4" />
                                  生成密钥对
                                </button>
                              </div>
                              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.reality.short_id</label>
                              <input v-model="form.tls.reality.short_id" type="text" class="input input-bordered input-sm w-full" placeholder="short_id，可为空" />
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
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">uuid <span class="text-destructive">*</span></label>
                  <div class="flex gap-2">
                    <input v-model="form.uuid" type="text" class="input input-bordered input-sm w-full font-mono" placeholder="UUID" />
                    <button class="btn btn-ghost btn-sm shrink-0" :disabled="generating" @click="genUuid">
                      <KeyRoundIcon class="h-4 w-4" />
                      生成
                    </button>
                  </div>
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">password <span class="text-destructive">*</span></label>
                  <div class="flex gap-2">
                    <input
                      v-model="form.password"
                      :type="showPassword ? 'text' : 'password'"
                      class="input input-bordered input-sm w-full font-mono"
                      placeholder="tuic 密码"
                    />
                    <button
                      type="button"
                      class="btn btn-ghost btn-sm shrink-0"
                      :title="showPassword ? '隐藏密码' : '显示明文'"
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
                      生成
                    </button>
                  </div>
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">congestion_control</label>
                  <SelectField v-model="form.congestion_control" :options="congestionOptions" />
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">udp_relay_mode</label>
                  <SelectField v-model="form.udp_relay_mode" :options="udpRelayOptions" />
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">zero_rtt_handshake</label>
                  <Switch v-model="form.zero_rtt_handshake" class="self-center" />
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.enabled</label>
                  <Switch v-model="form.tls.enabled" class="self-center" />

                  <!-- TUIC TLS 展开内容 -->
                  <div class="col-span-2 collapse-row" :class="{ 'is-open': form.tls.enabled }">
                    <div class="collapse-inner">
                      <div class="grid gap-x-4 gap-y-4 pt-2" style="grid-template-columns: 170px minmax(0, 1fr)">
                        <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.server_name <span class="text-destructive">*</span></label>
                        <input v-model="form.tls.server_name" type="text" class="input input-bordered input-sm w-full" placeholder="SNI，如 www.example.com" />
                        <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.alpn</label>
                        <ChipInput v-model="form.tls.alpn" :suggestions="alpnOptions" placeholder="ALPN（回车添加）" />
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
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">成员 outbounds <span class="text-destructive">*</span></label>
                  <ChipInput v-model="form.outbounds" :suggestions="groupCandidates" placeholder="选择组成员（回车添加，可多选）" />
                  <template v-if="isSelector">
                    <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">default</label>
                    <SelectField v-model="form.default_out" :options="form.outbounds" placeholder="可选，默认选中项" />
                  </template>
                  <template v-if="isUrlTest">
                    <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">url</label>
                    <input v-model="form.url" type="text" class="input input-bordered input-sm w-full" placeholder="测试地址，默认 https://www.gstatic.com/generate_204" />
                    <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">interval</label>
                    <input v-model="form.interval" type="text" class="input input-bordered input-sm w-full" placeholder="测试间隔，如 3m / 300s" />
                    <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tolerance</label>
                    <input v-model.number="form.tolerance" type="number" min="0" max="65535" class="input input-bordered input-sm w-40" />
                  </template>
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">interrupt_exist_connections</label>
                  <Switch v-model="form.interrupt_exist_connections" class="self-center" />
                </div>
              </div>
            </div>

            <!-- 其他类型兜底提示 -->
            <div class="col-span-2 collapse-row" :class="{ 'is-open': !isVless && !isTuic && !isSelector && !isUrlTest }">
              <div class="collapse-inner">
                <div class="grid gap-x-4 gap-y-4" style="grid-template-columns: 170px minmax(0, 1fr)">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">其他字段</label>
                  <p class="text-xs leading-relaxed text-[#909399]">该类型无专属表单，请在「源码」tab 直接编辑完整 JSON（如 users/network 等字段）。</p>
                </div>
              </div>
            </div>
          </div>
          <p v-if="validateMsg" class="mt-3 text-[0.8rem] font-medium text-destructive">{{ validateMsg }}</p>
        </TabsContent>
        <TabsContent value="source">
          <ResourceSourceTab ref="srcTab" :initial="sourceJson" />
        </TabsContent>
      </TabsRoot>
      <div class="mt-5 flex items-center justify-between">
        <div>
          <button
            v-if="isEdit"
            type="button"
            class="btn btn-error btn-outline btn-sm"
            :disabled="saving"
            @click="handleDelete"
          >
            删除
          </button>
        </div>
        <div class="flex justify-end gap-2">
          <button class="btn btn-ghost btn-sm" @click="emit('update:modelValue', false)">取消</button>
          <button class="btn btn-primary btn-sm" :disabled="saving" @click="save">保存</button>
        </div>
      </div>
    </DialogWrapper>

    <!-- 从 JSON / 节点链接导入弹窗 -->
    <DialogWrapper v-model="nodeImportDialog" title="从 JSON / 节点链接导入" box-class="max-w-[560px]">
      <textarea
        v-model="nodeImportText"
        rows="10"
        class="textarea textarea-bordered w-full font-mono text-xs"
        placeholder="支持粘贴 vless://...、vmess://...、ss://... 或完整 JSON 节点配置"
      />
      <p class="mt-1 text-xs text-[#909399]">解析成功后将自动填充表单字段（若节点类型变化会重置表单，请最后检查确认）</p>
      <div class="mt-4 flex justify-end gap-2">
        <button class="btn btn-ghost btn-sm" @click="nodeImportDialog = false">取消</button>
        <button class="btn btn-primary btn-sm" @click="doFillFromNode">解析并填充</button>
      </div>
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
