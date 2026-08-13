<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { showToast } from '@/helper/toast'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { PlusIcon, RefreshCw, ClipboardPasteIcon, KeyRoundIcon, WandIcon, UserPlusIcon, TrashIcon } from 'lucide-vue-next'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import ChipInput from '@/components/common/ChipInput.vue'
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { SelectField, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { api } from '../api'
import SourcePane from '../components/SourcePane.vue'
import ResourceSourceTab from '../components/ResourceSourceTab.vue'
import type { Inbound, UserMeta } from '../api'
import { useStatusStore } from '../stores/status'

const statusStore = useStatusStore()

// shadowsocks 入站：method 枚举与默认值（与 sing-box option/shadowsocks.go 对齐）
const SS_METHODS = [
  'none',
  'aes-128-gcm',
  'aes-192-gcm',
  'aes-256-gcm',
  'chacha20-ietf-poly1305',
  'xchacha20-ietf-poly1305',
  '2022-blake3-aes-128-gcm',
  '2022-blake3-aes-256-gcm',
  '2022-blake3-chacha20-poly1305'
]
const SS_DEFAULT_METHOD = 'chacha20-ietf-poly1305'
const SS_DEFAULT_LISTEN = '::'
const SS_DEFAULT_PORT = 23010

const loading = ref(false)
const outerTab = ref('form')
const inbounds = ref<Inbound[]>([])
const inboundTypes = ref<string[]>([])

const dialogVisible = ref(false)
const srcTab = ref<InstanceType<typeof ResourceSourceTab>>()
const sourceJson = ref('{}')
const isEdit = ref(false)
const editingTag = ref('')
const saving = ref(false)
const suppressTypeWatch = ref(false)
const allocating = ref(false)
const generating = ref(false)
const validateMsg = ref('')

interface InboundForm {
  type: string
  tag: string
  listen: string
  listen_port?: number
  users: { username: string; password: string }[]
  // tuic
  tuicUsers: { name: string; uuid: string; password: string }[]
  congestionControl: string
  authTimeout: string
  heartbeat: string
  zeroRTT: boolean
  tlsEnabled: boolean
  certificateProvider: string
  serverName: string
  insecure: boolean
  certificatePath: string
  keyPath: string
  alpn: string
  minVersion: string
  maxVersion: string
  idleTimeout: string
  keepAlivePeriod: string
  initialPacketSize: number
  disablePathMTU: boolean
  // shadowsocks
  ssMethod: string
  ssPassword: string
}

const form = reactive<InboundForm>({
  type: '',
  tag: '',
  listen: '',
  listen_port: undefined,
  users: [{ username: '', password: '' }],
  tuicUsers: [{ name: '', uuid: '', password: '' }],
  congestionControl: 'bbr',
  authTimeout: '',
  heartbeat: '',
  zeroRTT: false,
  tlsEnabled: true,
  certificateProvider: '',
  serverName: '',
  insecure: false,
  certificatePath: '',
  keyPath: '',
  alpn: '',
  minVersion: '',
  maxVersion: '',
  idleTimeout: '',
  keepAlivePeriod: '',
  initialPacketSize: 1452,
  disablePathMTU: false,
  ssMethod: SS_DEFAULT_METHOD,
  ssPassword: ''
})

// 用户池（tuic/shadowsocks 用户区：只读 + 选择绑定，用户在 Users 页管理）
const poolUsers = ref<UserMeta[]>([])
const selectedPoolUsers = ref<string[]>([])
// certificate provider 列表（证书来源以 provider 为主）
const certProviders = ref<string[]>([])

const isMixed = computed(() => form.type === 'mixed')
const isTuic = computed(() => form.type === 'tuic')
const isShadowsocks = computed(() => form.type === 'shadowsocks')
const isSsNoEncryption = computed(() => form.ssMethod === 'none')

const TLS_VERSIONS = ['1.0', '1.1', '1.2', '1.3']

function str(v: unknown): string {
  return typeof v === 'string' ? v : ''
}

function num(v: unknown): number | undefined {
  return typeof v === 'number' ? v : undefined
}

// 手写校验（替代 el-form rules）
function validateForm(): string {
  if (!form.tag.trim()) return 'tag 必填'
  if (isMixed.value || isShadowsocks.value || isTuic.value) {
    if (!form.listen.trim()) return 'listen 必填'
    if (typeof form.listen_port !== 'number' || form.listen_port < 1 || form.listen_port > 65535) {
      return 'listen_port 必填（1-65535）'
    }
  }
  if (isShadowsocks.value && !isSsNoEncryption.value && !form.ssPassword.trim() && !selectedPoolUsers.value.length) {
    return 'password 必填（method 为 none 或绑定用户时除外）'
  }
  return ''
}

function resetForm(type: string) {
  form.type = type
  form.tag = ''
  form.listen = ''
  form.listen_port = 443
  form.users = [{ username: '', password: '' }]
  form.ssMethod = SS_DEFAULT_METHOD
  form.ssPassword = ''
}

function fillForm(obj: Inbound) {
  form.type = str(obj.type)
  form.tag = str(obj.tag)
  form.listen = str(obj.listen)
  form.listen_port = num(obj.listen_port)
  if (form.type === 'mixed') {
    form.users = Array.isArray(obj.users)
      ? obj.users.map((u) => {
          const rec = (u ?? {}) as Record<string, unknown>
          return { username: str(rec.username), password: str(rec.password) }
        })
      : [{ username: '', password: '' }]
  } else {
    form.users = []
  }
  if (form.type === 'shadowsocks') {
    const s = (obj ?? {}) as Record<string, unknown>
    form.ssMethod = str(s.method) || SS_DEFAULT_METHOD
    form.ssPassword = str(s.password)
  }
  if (form.type === 'tuic') {
    const t = (obj ?? {}) as Record<string, unknown>
    form.tuicUsers = Array.isArray(t.users)
      ? (t.users as Record<string, unknown>[]).map((u) => ({
          name: str(u.name),
          uuid: str(u.uuid),
          password: str(u.password)
        }))
      : [{ name: '', uuid: '', password: '' }]
    form.congestionControl = str(t.congestion_control) || 'bbr'
    form.authTimeout = str(t.auth_timeout)
    form.heartbeat = str(t.heartbeat)
    form.zeroRTT = Boolean(t.zero_rtt_handshake)
    const tls = (t.tls ?? {}) as Record<string, unknown>
    form.tlsEnabled = true // tuic 强制 TLS
    form.certificateProvider = str(tls.certificate_provider)
    form.serverName = str(tls.server_name)
    form.insecure = Boolean(tls.insecure)
    form.certificatePath = str(tls.certificate_path)
    form.keyPath = str(tls.key_path)
    form.alpn = 'h3'
    form.minVersion = str(tls.min_version)
    form.maxVersion = str(tls.max_version)
    form.idleTimeout = str(t.idle_timeout)
    form.keepAlivePeriod = str(t.keep_alive_period)
    form.initialPacketSize = typeof t.initial_packet_size === 'number' ? t.initial_packet_size : 1452
    form.disablePathMTU = Boolean(t.disable_path_mtu_discovery)
  }
}

// 生成 shadowsocks 随机密码：优先后端（16 字节 → 标准 base64），后端不可用时浏览器端兜底
const generateSsPassword = async (notify = true) => {
  generating.value = true
  try {
    form.ssPassword = await api.genPassword()
    if (notify) showToast('已生成随机密码', 'success')
  } catch {
    const bytes = new Uint8Array(16)
    crypto.getRandomValues(bytes)
    form.ssPassword = btoa(String.fromCharCode(...bytes))
    if (notify) showToast('后端生成失败，已用浏览器随机数生成', 'warning')
  } finally {
    generating.value = false
  }
}

// 加载用户池与证书 provider
const loadPoolUsers = async () => {
  try {
    poolUsers.value = await api.users()
  } catch {
    poolUsers.value = []
  }
}
const loadCertProviders = async () => {
  try {
    const cert = await api.certificate()
    const list = cert?.providers || []
    // 引用的是 provider 的 tag（如 letsencrypt），不是 meta id
    certProviders.value = list
      .map((p: { id: string; provider: Record<string, any> }) => p.provider?.tag)
      .filter((t: unknown): t is string => typeof t === 'string' && t.length > 0)
  } catch {
    certProviders.value = []
  }
}

// 从后端 settings 拉取默认值填充 listen
const fillDefaults = async () => {
  try {
    const s = await api.settings()
    if (s.defaults?.listen) form.listen = s.defaults.listen
  } catch {
    // 拉取失败保持现状，用户可手动填
  }
}

const openCreate = async () => {
  isEdit.value = false
  editingTag.value = ''
  sourceJson.value = '{}'
  const def = statusStore.status?.defaults
  const defType = def?.inbound_type
  resetForm(defType && inboundTypes.value.includes(defType) ? defType : inboundTypes.value[0] || 'mixed')
  // listen 默认值来自后端 settings.defaults.listen（实时拉取）
  form.listen = ''
  await fillDefaults()
  if (form.type === 'shadowsocks') {
    // shadowsocks 默认值：listen "::" / 端口 23010 / method chacha20-ietf-poly1305 / 密码自动生成
    form.listen = SS_DEFAULT_LISTEN
    form.listen_port = SS_DEFAULT_PORT
    void generateSsPassword(false)
  } else {
    // listen_port 默认 443（常见入站端口），可手动修改或自动分配
    form.listen_port = 443
  }
  selectedPoolUsers.value = []
  void loadPoolUsers()
  void loadCertProviders()
  validateMsg.value = ''
  dialogVisible.value = true
}

const openEdit = async (row: Inbound) => {
  isEdit.value = true
  editingTag.value = row.tag
  dialogVisible.value = true
  validateMsg.value = ''
  try {
    const data = await api.getInbound(row.tag)
    sourceJson.value = JSON.stringify(data, null, 2)
    fillForm(data)
    if (isTuic.value || isShadowsocks.value) {
      void loadPoolUsers().then(() => {
        selectedPoolUsers.value = poolUsers.value
          .filter((u) => u.bound_inbounds?.includes(data.tag))
          .map((u) => u.name)
      })
      if (isTuic.value) void loadCertProviders()
    }
  } catch (e) {
    showToast((e as Error).message || '加载失败', 'error')
    dialogVisible.value = false
  }
}

// 新建时切换类型 → 重置表单
watch(
  () => form.type,
  (t) => {
    if (!suppressTypeWatch.value && dialogVisible.value && !isEdit.value) {
      resetForm(t)
      if (t === 'shadowsocks') {
        // 切到 shadowsocks：应用默认 listen/端口 并自动生成密码
        form.listen = SS_DEFAULT_LISTEN
        form.listen_port = SS_DEFAULT_PORT
        void generateSsPassword(false)
      } else {
        void fillDefaults() // 切换类型后重新填充 listen 默认值
      }
    }
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
    fillForm(res.data as Inbound)
    suppressTypeWatch.value = false
    jsonDialog.value = false
    showToast('已从 JSON 填充，请检查表单', 'success')
  } catch (e) {
    showToast((e as Error).message || '解析失败', 'error')
  }
}

// 自动分配最小可用端口
const allocatePort = async (notify = true) => {
  allocating.value = true
  try {
    const res = await api.availablePort()
    form.listen_port = res.port
    if (notify) showToast(`已分配可用端口 ${res.port}`, 'success')
  } catch (e) {
    if (notify) showToast((e as Error).message || '端口分配失败', 'error')
  } finally {
    allocating.value = false
  }
}

function buildBody(): Inbound {
  const body: Inbound = { type: form.type, tag: form.tag.trim() }
  if (form.listen.trim()) body.listen = form.listen.trim()
  if (typeof form.listen_port === 'number') body.listen_port = form.listen_port

  if (form.type === 'mixed') {
    const users = form.users
      .map((u) => ({ username: u.username.trim(), password: u.password.trim() }))
      .filter((u) => u.username || u.password)
    if (users.some((u) => !u.username || !u.password)) {
      throw new Error('用户名和密码需同时填写')
    }
    if (users.length) body.users = users
  }
  if (form.type === 'tuic') {
    // 用户由用户池绑定投影（见 Users 页），此处不输出 users
    // 保留手动 users 兼容（编辑老配置时）
    const users = form.tuicUsers
      .map((u) => ({ name: u.name.trim(), uuid: u.uuid.trim(), password: u.password.trim() }))
      .filter((u) => u.uuid || u.password)
    if (users.some((u) => !u.uuid || !u.password)) {
      throw new Error('tuic 用户必须填写 uuid 和 password')
    }
    if (users.length) body.users = users
    if (form.congestionControl) body.congestion_control = form.congestionControl
    if (form.authTimeout.trim()) body.auth_timeout = form.authTimeout.trim()
    if (form.heartbeat.trim()) body.heartbeat = form.heartbeat.trim()
    if (form.zeroRTT) body.zero_rtt_handshake = true
    const tls: Record<string, unknown> = {}
    tls.enabled = true // tuic 强制 TLS
    if (form.certificateProvider.trim()) tls.certificate_provider = form.certificateProvider.trim()
    if (form.serverName.trim()) tls.server_name = form.serverName.trim()
    if (form.certificatePath.trim()) tls.certificate_path = form.certificatePath.trim()
    if (form.keyPath.trim()) tls.key_path = form.keyPath.trim()
    if (form.alpn.trim()) tls.alpn = form.alpn.split(',').map((s) => s.trim()).filter(Boolean)
    if (form.minVersion) tls.min_version = form.minVersion
    if (form.maxVersion) tls.max_version = form.maxVersion
    if (Object.keys(tls).length) body.tls = tls
    if (form.idleTimeout.trim()) body.idle_timeout = form.idleTimeout.trim()
    if (form.keepAlivePeriod.trim()) body.keep_alive_period = form.keepAlivePeriod.trim()
    if (form.initialPacketSize > 0) body.initial_packet_size = form.initialPacketSize
    if (form.disablePathMTU) body.disable_path_mtu_discovery = true
  }
  if (form.type === 'shadowsocks') {
    body.method = form.ssMethod || SS_DEFAULT_METHOD
    // method 为 none（无加密）时不发送密码；其余 method 密码必填（表单校验已拦）
    if (form.ssMethod !== 'none' && form.ssPassword.trim()) body.password = form.ssPassword.trim()
    // 用户由用户池绑定投影（见 Users 页），此处不输出 users
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

// 把用户池绑定同步到当前入站（tuic/shadowsocks：用户在 Users 页管理，此处只读选择）
const syncBoundUsers = async () => {
  const tag = isEdit.value ? editingTag.value : form.tag.trim()
  if (!tag) return
  const current = new Set(
    poolUsers.value.filter((u) => u.bound_inbounds?.includes(tag)).map((u) => u.name)
  )
  const target = new Set(selectedPoolUsers.value)
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

const save = async () => {
  validateMsg.value = validateForm()
  if (validateMsg.value) return
  saving.value = true
  try {
    // tuic/shadowsocks：先把用户池绑定同步到当前入站
    if (isTuic.value || isShadowsocks.value) {
      await syncBoundUsers()
    }
    const body = srcTab.value?.isDirty() ? JSON.parse(srcTab.value.getText()) : buildBody()
    const res = isEdit.value ? await api.updateInbound(editingTag.value, body) : await api.createInbound(body)
    handleResult(res)
    dialogVisible.value = false
    await Promise.all([loadInbounds(), statusStore.refresh()])
  } catch (e) {
    showToast((e as Error).message || '操作失败', 'error')
  } finally {
    saving.value = false
  }
}

const remove = async (row: Inbound) => {
  const { confirmed } = await showConfirmDialog({
    title: '删除确认',
    message: `确定删除 inbound「${row.tag}」？该操作不可恢复。`,
    confirmText: '删除',
    confirmButtonClass: 'btn-error'
  })
  if (!confirmed) return
  try {
    await api.deleteInbound(row.tag)
    showToast('删除成功', 'success')
    await Promise.all([loadInbounds(), statusStore.refresh()])
  } catch (e) {
    showToast((e as Error).message || '删除失败', 'error')
  }
}

const loadInbounds = async () => {
  loading.value = true
  try {
    inbounds.value = await api.inbounds()
  } catch (e) {
    showToast((e as Error).message || '加载 inbounds 失败', 'error')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    const t = await api.types()
    // 只保留已实现表单的类型（mixed/tuic/shadowsocks）；编辑旧配置时其他类型附加显示
    const implemented = ['mixed', 'tuic', 'shadowsocks']
    const all = t.inbounds || []
    inboundTypes.value = implemented.filter((x) => all.includes(x))
  } catch (e) {
    showToast((e as Error).message || '加载类型列表失败', 'error')
  }
  await loadInbounds()
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
            新建 Inbound
          </button>
          <button class="btn btn-ghost btn-sm" :disabled="loading" @click="loadInbounds">
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
        </div>

        <div class="overflow-hidden rounded-lg border border-[#e4e7ed] bg-white dark:border-[#303030] dark:bg-[#1d1e1f]">
          <div v-if="loading && !inbounds.length" class="p-10 text-center text-sm text-[#909399]">加载中…</div>
          <table v-else class="table table-sm w-full">
            <thead>
              <tr>
                <th class="w-[130px]">类型</th>
                <th class="w-[160px]">tag</th>
                <th>listen</th>
                <th class="w-[120px]">listen_port</th>
                <th class="w-[210px]">method</th>
                <th class="w-[130px] text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in inbounds" :key="row.tag">
                <td class="text-xs">{{ row.type }}</td>
                <td class="text-xs font-medium">{{ row.tag }}</td>
                <td class="text-xs">{{ row.listen ?? '—' }}</td>
                <td class="text-xs">{{ row.listen_port ?? '—' }}</td>
                <td class="text-xs">{{ row.method ?? '—' }}</td>
                <td class="text-right">
                  <button class="btn btn-ghost btn-xs text-primary" @click="openEdit(row)">编辑</button>
                  <button class="btn btn-ghost btn-xs text-error" @click="remove(row)">删除</button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!loading && !inbounds.length" class="py-8 text-center text-sm text-[#606266] dark:text-[#a6b0bf]">暂无 inbound</div>
        </div>
      </TabsContent>

      <TabsContent value="source">
        <SourcePane segment="inbounds" @saved="loadInbounds" />
      </TabsContent>
    </TabsRoot>

    <!-- 新建/编辑弹窗 -->
    <DialogWrapper v-model="dialogVisible" :title="isEdit ? '编辑 Inbound' : '新建 Inbound'" box-class="max-w-[680px]">
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
            <span class="ml-2 text-xs text-[#909399]">粘贴完整 inbound JSON，自动解析并填充下方字段</span>
          </div>
          <div class="grid gap-x-4 gap-y-5" style="grid-template-columns: 130px minmax(0, 1fr)">
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">类型</label>
            <SelectField v-model="form.type" :disabled="isEdit">
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="t in inboundTypes" :key="t" :value="t">{{ t }}</SelectItem>
              </SelectContent>
            </SelectField>
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tag <span class="text-destructive">*</span></label>
            <input v-model="form.tag" type="text" class="input input-bordered input-sm w-full" placeholder="唯一标识，如 mixed-in" />

            <!-- mixed -->
            <template v-if="isMixed">
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">listen <span class="text-destructive">*</span></label>
              <input v-model="form.listen" type="text" class="input input-bordered input-sm w-full" placeholder="监听地址，如 127.0.0.1" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">listen_port <span class="text-destructive">*</span></label>
              <div class="flex gap-2">
                <input v-model.number="form.listen_port" type="number" min="1" max="65535" class="input input-bordered input-sm w-full" />
                <button class="btn btn-ghost btn-sm shrink-0" :disabled="allocating" @click="() => allocatePort()">
                  <WandIcon class="h-4 w-4" />
                  自动分配
                </button>
              </div>

              <div class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]" style="grid-column: 1 / -1">
                <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />用户（可选）<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
              </div>
              <template v-for="(u, i) in form.users" :key="i">
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">用户 {{ i + 1 }}</label>
                <div class="flex items-center gap-2">
                  <input v-model="u.username" type="text" class="input input-bordered input-sm flex-1" placeholder="用户名" />
                  <input v-model="u.password" type="password" class="input input-bordered input-sm flex-1" placeholder="密码" />
                  <button class="btn btn-ghost btn-xs text-error shrink-0" title="删除用户" @click="form.users.splice(i, 1)">
                    <TrashIcon class="h-3.5 w-3.5" />
                  </button>
                </div>
              </template>
              <span style="grid-column: 2">
                <button class="btn btn-ghost btn-xs" @click="form.users.push({ username: '', password: '' })">
                  <UserPlusIcon class="h-3.5 w-3.5" />
                  添加用户
                </button>
              </span>
            </template>

            <!-- shadowsocks -->
            <template v-else-if="isShadowsocks">
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">listen <span class="text-destructive">*</span></label>
              <input v-model="form.listen" type="text" class="input input-bordered input-sm w-full" placeholder="监听地址，默认 ::" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">listen_port <span class="text-destructive">*</span></label>
              <div class="flex gap-2">
                <input v-model.number="form.listen_port" type="number" min="1" max="65535" class="input input-bordered input-sm w-full" />
                <button class="btn btn-ghost btn-sm shrink-0" :disabled="allocating" @click="() => allocatePort()">
                  <WandIcon class="h-4 w-4" />
                  自动分配
                </button>
              </div>

              <div class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]" style="grid-column: 1 / -1">
                <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />协议<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
              </div>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">method <span class="text-destructive">*</span></label>
              <div class="flex flex-col gap-1">
                <SelectField v-model="form.ssMethod">
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="m in SS_METHODS" :key="m" :value="m">{{ m }}</SelectItem>
                  </SelectContent>
                </SelectField>
                <p class="text-xs text-[#606266] dark:text-[#a6b0bf]">默认 chacha20-ietf-poly1305；none 为无加密（仅调试用），2022-blake3-* 为 SIP022 系</p>
              </div>
              <template v-if="!isSsNoEncryption">
                <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">password <span class="text-destructive">*</span></label>
                <div class="flex flex-col gap-1">
                  <div class="flex gap-2">
                    <input v-model="form.ssPassword" type="password" class="input input-bordered input-sm w-full" placeholder="新建时自动生成" />
                    <button class="btn btn-ghost btn-sm shrink-0" :disabled="generating" @click="generateSsPassword()">
                      <KeyRoundIcon class="h-4 w-4" />
                      重新生成
                    </button>
                  </div>
                  <p class="text-xs text-[#606266] dark:text-[#a6b0bf]">默认自动生成 16 字节随机密码（base64，如 8JCsPssfgS8tiRwiMlhARg==）</p>
                </div>
              </template>
              <p v-else class="text-xs text-[#606266] dark:text-[#a6b0bf]" style="grid-column: 2">method 为 none（无加密）时不需要密码</p>

              <div class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]" style="grid-column: 1 / -1">
                <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />用户（Users 页统一管理，可选）<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
              </div>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">绑定用户</label>
              <div class="flex flex-col gap-1">
                <ChipInput v-model="selectedPoolUsers" :suggestions="poolUsers.map((u) => u.name)" placeholder="从用户池选择（多选）" />
                <p class="text-xs text-[#606266] dark:text-[#a6b0bf]">绑定后注入 users[]（name+password）；非 2022 method 以用户密码为准，2022 method 以顶部 password 为主密钥</p>
              </div>
            </template>

            <!-- tuic -->
            <template v-else-if="isTuic">
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">listen <span class="text-destructive">*</span></label>
              <input v-model="form.listen" type="text" class="input input-bordered input-sm w-full" placeholder="监听地址，如 0.0.0.0" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">listen_port <span class="text-destructive">*</span></label>
              <div class="flex gap-2">
                <input v-model.number="form.listen_port" type="number" min="1" max="65535" class="input input-bordered input-sm w-full" />
                <button class="btn btn-ghost btn-sm shrink-0" :disabled="allocating" @click="() => allocatePort()">
                  <WandIcon class="h-4 w-4" />
                  自动分配
                </button>
              </div>

              <div class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]" style="grid-column: 1 / -1">
                <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />用户（Users 页统一管理）<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
              </div>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">绑定用户</label>
              <div class="flex flex-col gap-1">
                <ChipInput v-model="selectedPoolUsers" :suggestions="poolUsers.map((u) => u.name)" placeholder="从用户池选择（多选）" />
                <p class="text-xs text-[#606266] dark:text-[#a6b0bf]">用户统一在 Users 页创建/编辑；此处仅选择要绑定到本入站的用户，保存后自动注入 users[]（按类型取用字段）</p>
              </div>

              <div class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]" style="grid-column: 1 / -1">
                <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />协议<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
              </div>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">拥塞控制</label>
              <SelectField v-model="form.congestionControl">
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="cubic">cubic</SelectItem>
                  <SelectItem value="new_reno">new_reno</SelectItem>
                  <SelectItem value="bbr">bbr</SelectItem>
                </SelectContent>
              </SelectField>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">认证超时</label>
              <input v-model="form.authTimeout" type="text" class="input input-bordered input-sm w-full" placeholder="如 3s" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">心跳间隔</label>
              <input v-model="form.heartbeat" type="text" class="input input-bordered input-sm w-full" placeholder="如 10s" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">0-RTT 握手</label>
              <Switch v-model="form.zeroRTT" class="mt-1.5" />

              <div class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]" style="grid-column: 1 / -1">
                <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />TLS（tuic 强制启用）<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
              </div>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">证书来源</label>
              <div class="flex flex-col gap-1">
                <SelectField v-model="form.certificateProvider" :disabled="!!form.certificatePath.trim() || !!form.keyPath.trim()">
                  <SelectTrigger><SelectValue placeholder="选择 Certificate Provider（推荐）" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="id in certProviders" :key="id" :value="id">{{ id }}</SelectItem>
                  </SelectContent>
                </SelectField>
                <p class="text-xs text-[#606266] dark:text-[#a6b0bf]">证书 Provider 在「证书」页管理（引用其 tag）；与下方手动证书路径二选一</p>
              </div>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">server_name</label>
              <input v-model="form.serverName" type="text" class="input input-bordered input-sm w-full" placeholder="证书域名" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">证书路径</label>
              <input v-model="form.certificatePath" type="text" class="input input-bordered input-sm w-full" placeholder="/etc/sing-box/cert.pem" :disabled="!!form.certificateProvider" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">私钥路径</label>
              <input v-model="form.keyPath" type="text" class="input input-bordered input-sm w-full" placeholder="/etc/sing-box/key.pem" :disabled="!!form.certificateProvider" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">ALPN</label>
              <span class="badge badge-info mt-1 w-fit">h3</span>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">TLS 版本</label>
              <div class="flex gap-2">
                <SelectField v-model="form.minVersion" class="flex-1">
                  <SelectTrigger><SelectValue placeholder="最小" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="v in TLS_VERSIONS" :key="v" :value="v">{{ v }}</SelectItem>
                  </SelectContent>
                </SelectField>
                <SelectField v-model="form.maxVersion" class="flex-1">
                  <SelectTrigger><SelectValue placeholder="最大" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="v in TLS_VERSIONS" :key="v" :value="v">{{ v }}</SelectItem>
                  </SelectContent>
                </SelectField>
              </div>

              <div class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]" style="grid-column: 1 / -1">
                <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />QUIC<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
              </div>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">空闲超时</label>
              <input v-model="form.idleTimeout" type="text" class="input input-bordered input-sm w-full" placeholder="如 30s" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">保活周期</label>
              <input v-model="form.keepAlivePeriod" type="text" class="input input-bordered input-sm w-full" placeholder="如 15s" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">初始包大小</label>
              <input v-model.number="form.initialPacketSize" type="number" min="0" max="10000" class="input input-bordered input-sm w-40" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">禁用 MTU 发现</label>
              <Switch v-model="form.disablePathMTU" class="mt-1.5" />
            </template>

            <!-- 其他类型 -->
            <template v-else>
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">listen</label>
              <input v-model="form.listen" type="text" class="input input-bordered input-sm w-full" placeholder="监听地址（可选）" />
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">listen_port</label>
              <div class="flex gap-2">
                <input v-model.number="form.listen_port" type="number" min="1" max="65535" class="input input-bordered input-sm w-full" />
                <button class="btn btn-ghost btn-sm shrink-0" :disabled="allocating" @click="() => allocatePort()">
                  <WandIcon class="h-4 w-4" />
                  自动分配
                </button>
              </div>
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
        placeholder='{"type":"mixed","tag":"mixed-in","listen":"127.0.0.1","listen_port":2080}'
      />
      <p class="mt-1 text-xs text-[#909399]">解析成功后填充表单字段（type 变化会重置表单，请最后检查）</p>
      <div class="mt-4 flex justify-end gap-2">
        <button class="btn btn-ghost btn-sm" @click="jsonDialog = false">取消</button>
        <button class="btn btn-primary btn-sm" @click="doFillFromJson">解析并填充</button>
      </div>
    </DialogWrapper>
  </div>
</template>
