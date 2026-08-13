<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { showToast } from '@/helper/toast'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { PlusIcon, RefreshCw, ClipboardPasteIcon, KeyRoundIcon, LoaderIcon } from 'lucide-vue-next'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import ChipInput from '@/components/common/ChipInput.vue'
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { SelectField } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { api } from '../api'
import SourcePane from '../components/SourcePane.vue'
import ResourceSourceTab from '../components/ResourceSourceTab.vue'
import type { Outbound } from '../api'
import { useStatusStore } from '../stores/status'

const statusStore = useStatusStore()

const loading = ref(false)
const outerTab = ref('form')
const outbounds = ref<Outbound[]>([])
const outboundTypes = ref<string[]>([])

const dialogVisible = ref(false)
const srcTab = ref<InstanceType<typeof ResourceSourceTab>>()
const sourceJson = ref('{}')
const isEdit = ref(false)
const editingTag = ref('')
const saving = ref(false)
const generating = ref(false)
const suppressTypeWatch = ref(false)
const validateMsg = ref('')

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
  // selector / urltest 组
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

// 组（selector/urltest）候选成员：现有 outbound tags，排除自身
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

// 手写校验（替代 el-form rules）
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
  form.type = str(obj.type)
  form.tag = str(obj.tag)
  form.server = str(obj.server)
  form.server_port = num(obj.server_port)
  form.uuid = str(obj.uuid)
  form.flow = str(obj.flow)
  form.network = str(obj.network) || 'tcp'
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

const openCreate = () => {
  isEdit.value = false
  editingTag.value = ''
  sourceJson.value = '{}'
  const def = statusStore.status?.defaults?.outbound_type
  resetForm(def && outboundTypes.value.includes(def) ? def : outboundTypes.value[0] || 'vless')
  dialogVisible.value = true
  validateMsg.value = ''
}

const openEdit = async (row: Outbound) => {
  isEdit.value = true
  editingTag.value = row.tag
  dialogVisible.value = true
  validateMsg.value = ''
  try {
    const data = await api.getOutbound(row.tag)
    sourceJson.value = JSON.stringify(data, null, 2)
    fillForm(data)
  } catch (e) {
    showToast((e as Error).message || '加载失败', 'error')
    dialogVisible.value = false
  }
}

// 新建时切换类型 → 重置表单（从 JSON 填充时抑制）
watch(
  () => form.type,
  (t) => {
    if (!suppressTypeWatch.value && dialogVisible.value && !isEdit.value) resetForm(t)
  }
)

// 从 JSON 填充弹窗
const jsonDialog = ref(false)
const jsonText = ref('')
const fillFromJson = () => {
  jsonText.value = ''
  jsonDialog.value = true
}
const doFillFromJson = async () => {
  try {
    const res = await api.parseJson(jsonText.value)
    suppressTypeWatch.value = true
    fillForm(res.data as Outbound)
    suppressTypeWatch.value = false
    jsonDialog.value = false
    showToast('已从 JSON 填充，请检查表单', 'success')
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

  // 其他类型：表单覆盖常用字段（未覆盖字段用「源码」tab 编辑）
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
    const res = isEdit.value ? await api.updateOutbound(editingTag.value, body) : await api.createOutbound(body)
    handleResult(res)
    dialogVisible.value = false
    await Promise.all([loadOutbounds(), statusStore.refresh()])
  } catch (e) {
    showToast((e as Error).message || '操作失败', 'error')
  } finally {
    saving.value = false
  }
}

const remove = async (row: Outbound) => {
  const first = await showConfirmDialog({
    title: '删除确认',
    message: `确定删除 outbound「${row.tag}」？该操作不可恢复。`,
    confirmText: '删除',
    confirmButtonClass: 'btn-error'
  })
  if (!first.confirmed) return
  try {
    await api.deleteOutbound(row.tag)
    showToast('删除成功', 'success')
    await Promise.all([loadOutbounds(), statusStore.refresh()])
  } catch (e) {
    const err = e as Error & { references?: string[] }
    // 被 selector/urltest 组引用：确认后强制删除（后端自动从引用组拔除 tag）
    if (err.references?.length) {
      const second = await showConfirmDialog({
        title: '节点被组引用',
        message: `节点「${row.tag}」被以下组引用：${err.references.join('、')}\n删除后将自动从这些组中移除该节点。确认删除？`,
        confirmText: '确认删除',
        confirmButtonClass: 'btn-error'
      })
      if (!second.confirmed) return
      try {
        await api.deleteOutbound(row.tag, true)
        showToast(`删除成功，并已从 ${err.references.join('、')} 中移除`, 'success')
        await Promise.all([loadOutbounds(), statusStore.refresh()])
      } catch (e2) {
        showToast((e2 as Error).message || '删除失败', 'error')
      }
      return
    }
    showToast(err.message || '删除失败', 'error')
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
    // 私钥仅展示一次：用确认弹窗展示（复制后确认）
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

const loadOutbounds = async () => {
  loading.value = true
  try {
    outbounds.value = await api.outbounds()
  } catch (e) {
    showToast((e as Error).message || '加载 outbounds 失败', 'error')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    const t = await api.types()
    outboundTypes.value = t.outbounds || []
  } catch {
    // 类型依赖后端枚举（跟随 sing-box 版本），后端不可用时保持为空，仅提示
    showToast('当前没有后端连接，无法使用该功能', 'warning')
  }
  await loadOutbounds()
})
</script>

<template>
  <div>
    <TabsRoot v-model="outerTab">
      <TabsList>
        <TabsTrigger value="form">表单</TabsTrigger>
        <TabsTrigger value="source">源码</TabsTrigger>
      </TabsList>

      <TabsContent value="form">
        <div class="mb-3.5 flex items-center gap-2.5">
          <button class="btn btn-primary btn-sm" @click="openCreate">
            <PlusIcon class="h-4 w-4" />
            新建 Outbound
          </button>
          <button class="btn btn-ghost btn-sm" :disabled="loading" @click="loadOutbounds">
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
        </div>

        <div class="overflow-hidden rounded-lg border border-[#e4e7ed] bg-white dark:border-[#303030] dark:bg-[#1d1e1f]">
          <div v-if="loading && !outbounds.length" class="p-10 text-center text-sm text-[#909399]">加载中…</div>
          <table v-else class="table table-sm w-full">
            <thead>
              <tr>
                <th class="w-[130px]">类型</th>
                <th class="w-[160px]">tag</th>
                <th>server</th>
                <th class="w-[120px]">server_port</th>
                <th class="w-[130px] text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in outbounds" :key="row.tag">
                <td class="text-xs">{{ row.type }}</td>
                <td class="text-xs font-medium">{{ row.tag }}</td>
                <td class="text-xs">{{ row.server ?? '—' }}</td>
                <td class="text-xs">{{ row.server_port ?? '—' }}</td>
                <td class="text-right">
                  <button class="btn btn-ghost btn-xs text-primary" @click="openEdit(row)">编辑</button>
                  <button class="btn btn-ghost btn-xs text-error" @click="remove(row)">删除</button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!loading && !outbounds.length" class="py-8 text-center text-sm text-[#606266] dark:text-[#a6b0bf]">暂无 outbound</div>
        </div>
      </TabsContent>

      <TabsContent value="source">
        <SourcePane segment="outbounds" @saved="loadOutbounds" />
      </TabsContent>
    </TabsRoot>

    <!-- 新建/编辑弹窗 -->
    <DialogWrapper v-model="dialogVisible" :title="isEdit ? '编辑 Outbound' : '新建 Outbound'" box-class="max-w-[720px]">
      <TabsRoot :model-value="'form'">
        <TabsList>
          <TabsTrigger value="form">表单</TabsTrigger>
          <TabsTrigger value="source">源码</TabsTrigger>
        </TabsList>
        <TabsContent value="form">
          <div class="mb-4">
            <button class="btn btn-ghost btn-xs" @click="fillFromJson">
              <ClipboardPasteIcon class="h-3.5 w-3.5" />
              从 JSON 填充（粘贴解析）
            </button>
            <span class="ml-2 text-xs text-[#909399]">粘贴完整 outbound JSON，自动解析并填充下方字段</span>
          </div>
          <div class="grid gap-x-4 gap-y-5" style="grid-template-columns: 170px minmax(0, 1fr)">
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">类型</label>
            <template v-if="outboundTypes.length">
              <SelectField v-model="form.type" :disabled="isEdit" :options="outboundTypes"  />
            </template>
            <div v-else class="alert alert-warning py-2 text-xs">当前没有后端连接，无法使用该功能</div>
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tag <span class="text-destructive">*</span></label>
            <input v-model="form.tag" type="text" class="input input-bordered input-sm w-full" placeholder="唯一标识，如 vless-out" />

            <template v-if="!isSelector && !isUrlTest">
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">server <span class="text-destructive">*</span></label>
              <input v-model="form.server" type="text" class="input input-bordered input-sm w-full" placeholder="服务器地址 / IP" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">server_port <span class="text-destructive">*</span></label>
              <input v-model.number="form.server_port" type="number" min="1" max="65535" class="input input-bordered input-sm w-40" />
            </template>

            <!-- vless 专属字段 -->
            <template v-if="isVless">
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">uuid <span class="text-destructive">*</span></label>
              <div class="flex gap-2">
                <input v-model="form.uuid" type="text" class="input input-bordered input-sm w-full font-mono" placeholder="UUID" />
                <button class="btn btn-ghost btn-sm shrink-0" :disabled="generating" @click="genUuid">
                  <KeyRoundIcon class="h-4 w-4" />
                  生成
                </button>
              </div>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">flow</label>
              <SelectField v-model="form.flow" :options="[{ value: 'none', label: '（无）' }, { value: 'xtls-rprx-vision', label: 'xtls-rprx-vision' }, { value: 'xtls-rprx-vision-udp443', label: 'xtls-rprx-vision-udp443' }]" placeholder="留空则不启用" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">network</label>
              <SelectField v-model="form.network" :options="networkOptionsAll"  />

              <div class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]" style="grid-column: 1 / -1">
                <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />TLS<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
              </div>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.enabled</label>
              <Switch v-model="form.tls.enabled" class="self-center" />
              <template v-if="form.tls.enabled">
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.server_name <span class="text-destructive">*</span></label>
                <input v-model="form.tls.server_name" type="text" class="input input-bordered input-sm w-full" placeholder="SNI，如 www.example.com" />
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.utls.enabled</label>
                <Switch v-model="form.tls.utls.enabled" class="self-center" />
                <template v-if="form.tls.utls.enabled">
                  <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.utls.fingerprint</label>
                  <SelectField v-model="form.tls.utls.fingerprint" :options="fingerprintOptions"  />
                </template>
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.reality.enabled</label>
                <Switch v-model="form.tls.reality.enabled" class="self-center" />
                <template v-if="form.tls.reality.enabled">
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
                </template>
              </template>
            </template>

            <!-- tuic 专属字段 -->
            <template v-else-if="isTuic">
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">uuid <span class="text-destructive">*</span></label>
              <div class="flex gap-2">
                <input v-model="form.uuid" type="text" class="input input-bordered input-sm w-full font-mono" placeholder="UUID" />
                <button class="btn btn-ghost btn-sm shrink-0" :disabled="generating" @click="genUuid">
                  <KeyRoundIcon class="h-4 w-4" />
                  生成
                </button>
              </div>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">password <span class="text-destructive">*</span></label>
              <input v-model="form.password" type="password" class="input input-bordered input-sm w-full" placeholder="tuic 密码" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">congestion_control</label>
              <SelectField v-model="form.congestion_control" :options="congestionOptions"  />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">udp_relay_mode</label>
              <SelectField v-model="form.udp_relay_mode" :options="udpRelayOptions"  />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">zero_rtt_handshake</label>
              <Switch v-model="form.zero_rtt_handshake" class="self-center" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.enabled</label>
              <Switch v-model="form.tls.enabled" class="self-center" />
              <template v-if="form.tls.enabled">
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.server_name <span class="text-destructive">*</span></label>
                <input v-model="form.tls.server_name" type="text" class="input input-bordered input-sm w-full" placeholder="SNI，如 www.example.com" />
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tls.alpn</label>
                <ChipInput v-model="form.tls.alpn" :suggestions="alpnOptions" placeholder="ALPN（回车添加）" />
              </template>
            </template>

            <!-- selector / urltest 组字段 -->
            <template v-else-if="isSelector || isUrlTest">
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">成员 outbounds <span class="text-destructive">*</span></label>
              <ChipInput v-model="form.outbounds" :suggestions="groupCandidates" placeholder="选择组成员（回车添加，可多选）" />
              <template v-if="isSelector">
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">default</label>
                <SelectField v-model="form.default_out" :options="form.outbounds" placeholder="可选，默认选中项"  />
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
            </template>

            <!-- 其他类型：原始 JSON 兜底 -->
            <template v-else>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">其他字段</label>
              <p class="text-xs leading-relaxed text-[#909399]">该类型无专属表单，请在「源码」tab 直接编辑完整 JSON（如 users/network 等字段）。</p>
            </template>
          </div>
          <p v-if="validateMsg" class="mt-3 text-[0.8rem] font-medium text-destructive">{{ validateMsg }}</p>
        </TabsContent>
        <TabsContent value="source">
          <ResourceSourceTab ref="srcTab" :initial="sourceJson" />
        </TabsContent>
      </TabsRoot>
      <div class="mt-5 flex justify-end gap-2">
        <button class="btn btn-ghost btn-sm" @click="dialogVisible = false">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="saving" @click="save">保存</button>
      </div>
    </DialogWrapper>

    <!-- 从 JSON 填充弹窗 -->
    <DialogWrapper v-model="jsonDialog" title="从 JSON 填充" box-class="max-w-[560px]">
      <textarea
        v-model="jsonText"
        rows="10"
        class="textarea textarea-bordered w-full font-mono text-xs"
        placeholder='{"type":"vless","tag":"...","server":"...","server_port":443,"uuid":"...","tls":{...}}'
      />
      <p class="mt-1 text-xs text-[#909399]">解析成功后填充表单字段（type 变化会重置表单，请最后检查）</p>
      <div class="mt-4 flex justify-end gap-2">
        <button class="btn btn-ghost btn-sm" @click="jsonDialog = false">取消</button>
        <button class="btn btn-primary btn-sm" @click="doFillFromJson">解析并填充</button>
      </div>
    </DialogWrapper>
  </div>
</template>
