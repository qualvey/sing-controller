<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { api } from '../api'
import SourcePane from '../components/SourcePane.vue'
import ResourceSourceTab from '../components/ResourceSourceTab.vue'
import type { Inbound } from '../api'
import { useStatusStore } from '../stores/status'

const statusStore = useStatusStore()

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
const formRef = ref<FormInstance>()

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
})

// 用户池（tuic 用户区：只读 + 选择绑定，用户在 Users 页管理）
const poolUsers = ref<import('../api').UserMeta[]>([])
const selectedPoolUsers = ref<string[]>([])
// certificate provider 列表（证书来源以 provider 为主）
const certProviders = ref<string[]>([])

const isMixed = computed(() => form.type === 'mixed')
const isTuic = computed(() => form.type === 'tuic')

const rules = computed<FormRules>(() => {
  const base: FormRules = {
    tag: [{ required: true, message: 'tag 必填', trigger: 'blur' }]
  }
  if (isMixed.value) {
    base.listen = [{ required: true, message: 'listen 必填', trigger: 'blur' }]
    base.listen_port = [
      { required: true, message: 'listen_port 必填', trigger: 'blur' },
      { type: 'number', min: 1, max: 65535, message: '端口范围 1-65535', trigger: 'blur' }
    ]
  }
  return base
})

function str(v: unknown): string {
  return typeof v === 'string' ? v : ''
}

function num(v: unknown): number | undefined {
  return typeof v === 'number' ? v : undefined
}

function resetForm(type: string) {
  form.type = type
  form.tag = ''
  form.listen = ''
  form.listen_port = undefined
  form.users = [{ username: '', password: '' }]
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
    form.alpn = Array.isArray(tls.alpn) ? (tls.alpn as string[]).join(',') : ''
    form.minVersion = str(tls.min_version)
    form.maxVersion = str(tls.max_version)
    form.idleTimeout = str(t.idle_timeout)
    form.keepAlivePeriod = str(t.keep_alive_period)
    form.initialPacketSize = typeof t.initial_packet_size === 'number' ? t.initial_packet_size : 1452
    form.disablePathMTU = Boolean(t.disable_path_mtu_discovery)
  }
}

// 加载用户池与证书 provider（打开 tuic 表单时）
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
    certProviders.value = list.map((p: { id: string }) => p.id)
  } catch {
    certProviders.value = []
  }
}

// 从后端 settings 拉取默认值填充 listen（不依赖 statusStore 异步时序）
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
  // listen_port 默认自动分配（可手动修改），分配失败则留空
  form.listen_port = undefined
  void allocatePort(false)
  selectedPoolUsers.value = []
  void loadPoolUsers()
  void loadCertProviders()
  dialogVisible.value = true
  formRef.value?.clearValidate()
}

const openEdit = async (row: Inbound) => {
  isEdit.value = true
  editingTag.value = row.tag
  dialogVisible.value = true
  formRef.value?.clearValidate()
  try {
    const data = await api.getInbound(row.tag)
    sourceJson.value = JSON.stringify(data, null, 2)
    fillForm(data)
    if (isTuic.value) {
      void loadPoolUsers().then(() => {
        selectedPoolUsers.value = poolUsers.value
          .filter((u) => u.bound_inbounds?.includes(data.tag))
          .map((u) => u.name)
      })
      void loadCertProviders()
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
    dialogVisible.value = false
  }
}

// 新建时切换类型 → 重置表单（从 JSON 填充时抑制）
watch(
  () => form.type,
  (t) => {
    if (!suppressTypeWatch.value && dialogVisible.value && !isEdit.value) {
      resetForm(t)
      void fillDefaults() // 切换类型后重新填充 listen 默认值
    }
  }
)

// 粘贴 JSON → 后端解析 → 填充表单字段
const fillFromJson = async () => {
  let text: string
  try {
    const { value } = await ElMessageBox.prompt(
      '粘贴 inbound JSON，解析成功后填充表单字段（type 变化会重置表单，请最后检查）',
      '从 JSON 填充',
      {
        confirmButtonText: '解析并填充',
        cancelButtonText: '取消',
        inputType: 'textarea',
        inputPlaceholder: '{"type":"mixed","tag":"mixed-in","listen":"127.0.0.1","listen_port":2080}'
      }
    )
    text = value
  } catch {
    return
  }
  try {
    const res = await api.parseJson(text)
    suppressTypeWatch.value = true
    fillForm(res.data as Inbound)
    suppressTypeWatch.value = false
    ElMessage.success('已从 JSON 填充，请检查表单')
  } catch (e) {
    ElMessage.error((e as Error).message || '解析失败')
  }
}

// 自动分配最小可用端口（起点=controller 配置的 min_port）；新建时默认触发，也可手动点击
const allocatePort = async (notify = true) => {
  allocating.value = true
  try {
    const res = await api.availablePort()
    form.listen_port = res.port
    if (notify) ElMessage.success(`已分配可用端口 ${res.port}`)
  } catch (e) {
    if (notify) ElMessage.error((e as Error).message || '端口分配失败')
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
    // tuic 强制 TLS
    tls.enabled = true
    if (form.certificateProvider.trim()) tls.certificate_provider = form.certificateProvider.trim()
    if (form.serverName.trim()) tls.server_name = form.serverName.trim()
    if (form.insecure) tls.insecure = true
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
  body.type = form.type
  if (!body.tag) body.tag = form.tag.trim()
  return body
}

const handleResult = (res: unknown) => {
  const r = (res ?? {}) as { reload_error?: string; message?: string }
  if (r.reload_error) {
    ElMessage.warning(r.message || `配置已保存，但实例重载失败：${r.reload_error}`)
  } else {
    ElMessage.success('保存成功')
  }
}

const save = async () => {
  const valid = await formRef.value?.validate().then(() => true).catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    // 源码 tab 被修改时用源码内容作为提交体（覆盖表单）
    // tuic：先把用户池绑定同步到当前入站（用户在 Users 页管理，这里只读选择）
    if (isTuic.value) {
      const tag = isEdit.value ? editingTag.value : form.tag.trim()
      if (tag) {
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
    }
    const body = srcTab.value?.isDirty() ? JSON.parse(srcTab.value.getText()) : buildBody()
    const res = isEdit.value ? await api.updateInbound(editingTag.value, body) : await api.createInbound(body)
    handleResult(res)
    dialogVisible.value = false
    await Promise.all([loadInbounds(), statusStore.refresh()])
  } catch (e) {
    ElMessage.error((e as Error).message || '操作失败')
  } finally {
    saving.value = false
  }
}

const remove = async (row: Inbound) => {
  try {
    await ElMessageBox.confirm(`确定删除 inbound「${row.tag}」？该操作不可恢复。`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  try {
    await api.deleteInbound(row.tag)
    ElMessage.success('删除成功')
    await Promise.all([loadInbounds(), statusStore.refresh()])
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}

const loadInbounds = async () => {
  loading.value = true
  try {
    inbounds.value = await api.inbounds()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载 inbounds 失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    const t = await api.types()
    inboundTypes.value = t.inbounds || []
  } catch (e) {
    ElMessage.error((e as Error).message || '加载类型列表失败')
  }
  await loadInbounds()
})
</script>

<template>
  <div class="page">
    <el-tabs v-model="outerTab">
      <el-tab-pane label="表单" name="form">
    <div class="toolbar">
      <el-button type="primary" @click="openCreate">新建 Inbound</el-button>
      <el-button :loading="loading" @click="loadInbounds">刷新</el-button>
    </div>

    <el-table :data="inbounds" v-loading="loading" border stripe>
      <el-table-column prop="type" label="类型" width="130" />
      <el-table-column prop="tag" label="tag" min-width="160" />
      <el-table-column label="listen" min-width="160">
        <template #default="{ row }">{{ row.listen ?? '—' }}</template>
      </el-table-column>
      <el-table-column label="listen_port" width="120">
        <template #default="{ row }">{{ row.listen_port ?? '—' }}</template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" link @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" link @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑 Inbound' : '新建 Inbound'"
      width="680px"
      :close-on-click-modal="false"
    >
      <el-tabs>
        <el-tab-pane label="表单">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="130px">
        <el-form-item>
          <el-button size="small" @click="fillFromJson">从 JSON 填充（粘贴解析）</el-button>
          <span class="fill-hint">粘贴完整 inbound JSON，自动解析并填充下方字段</span>
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-select v-model="form.type" :disabled="isEdit" style="width: 100%">
            <el-option v-for="t in inboundTypes" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>
        <el-form-item label="tag" prop="tag">
          <el-input v-model="form.tag" placeholder="唯一标识，如 mixed-in" />
        </el-form-item>

        <!-- mixed：渲染 users 动态行 -->
        <template v-if="isMixed">
          <el-form-item label="listen" prop="listen">
            <el-input v-model="form.listen" placeholder="监听地址，如 127.0.0.1" />
          </el-form-item>
          <el-form-item label="listen_port" prop="listen_port">
            <div class="row">
              <el-input-number v-model="form.listen_port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
              <el-button :loading="allocating" @click="allocatePort">自动分配</el-button>
            </div>
          </el-form-item>
          <el-divider content-position="left">用户（可选）</el-divider>
          <el-form-item v-for="(u, i) in form.users" :key="i" :label="`用户 ${i + 1}`">
            <div class="user-row">
              <el-input v-model="u.username" placeholder="用户名" />
              <el-input v-model="u.password" type="password" show-password placeholder="密码" />
              <el-button type="danger" link @click="form.users.splice(i, 1)">删除</el-button>
            </div>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" plain @click="form.users.push({ username: '', password: '' })">添加用户</el-button>
          </el-form-item>
        </template>

        <!-- tuic：完整字段表单 -->
        <template v-else-if="isTuic">
          <el-form-item label="listen" prop="listen">
            <el-input v-model="form.listen" placeholder="监听地址，如 0.0.0.0" />
          </el-form-item>
          <el-form-item label="listen_port" prop="listen_port">
            <div class="row">
              <el-input-number v-model="form.listen_port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
              <el-button :loading="allocating" @click="allocatePort">自动分配</el-button>
            </div>
          </el-form-item>
          <el-divider content-position="left">用户（Users 页统一管理）</el-divider>
          <el-form-item label="绑定用户">
            <el-select v-model="selectedPoolUsers" multiple filterable style="width: 100%" placeholder="从用户池选择（多选）">
              <el-option v-for="u in poolUsers" :key="u.name" :label="u.name" :value="u.name" />
            </el-select>
            <div class="mt-1 w-full text-xs text-[var(--el-text-color-secondary)]">
              用户统一在 Users 页创建/编辑；此处仅选择要绑定到本入站的用户，保存后自动注入 users[]（按类型取用字段）
            </div>
          </el-form-item>
          <el-form-item label="当前绑定" v-if="selectedPoolUsers.length">
            <div class="flex w-full flex-wrap gap-1">
              <el-tag v-for="name in selectedPoolUsers" :key="name" size="small" type="primary" effect="plain">
                {{ name }}
              </el-tag>
            </div>
          </el-form-item>
          <el-divider content-position="left">协议</el-divider>
          <el-form-item label="拥塞控制">
            <el-select v-model="form.congestionControl" style="width: 100%">
              <el-option label="cubic" value="cubic" />
              <el-option label="new_reno" value="new_reno" />
              <el-option label="bbr" value="bbr" />
            </el-select>
          </el-form-item>
          <el-form-item label="认证超时">
            <el-input v-model="form.authTimeout" placeholder="如 3s" />
          </el-form-item>
          <el-form-item label="心跳间隔">
            <el-input v-model="form.heartbeat" placeholder="如 10s" />
          </el-form-item>
          <el-form-item label="0-RTT 握手">
            <el-switch v-model="form.zeroRTT" />
          </el-form-item>
          <el-divider content-position="left">TLS（tuic 强制启用）</el-divider>
            <el-form-item label="证书来源">
              <el-select v-model="form.certificateProvider" clearable style="width: 100%" placeholder="选择 Certificate Provider（推荐）">
                <el-option v-for="id in certProviders" :key="id" :label="id" :value="id" />
              </el-select>
              <div class="mt-1 w-full text-xs text-[var(--el-text-color-secondary)]">
                证书 Provider 在「证书」页管理；也可改用下方证书路径（文件方式）
              </div>
            </el-form-item>
            <el-form-item label="server_name">
              <el-input v-model="form.serverName" placeholder="证书域名" />
            </el-form-item>
            <el-form-item label="忽略校验">
              <el-switch v-model="form.insecure" />
            </el-form-item>
            <el-form-item label="证书路径">
              <el-input v-model="form.certificatePath" placeholder="/etc/sing-box/cert.pem（可选，替代 Provider）" />
            </el-form-item>
            <el-form-item label="私钥路径">
              <el-input v-model="form.keyPath" placeholder="/etc/sing-box/key.pem" />
            </el-form-item>
            <el-form-item label="ALPN">
              <el-input v-model="form.alpn" placeholder="h3，逗号分隔多个" />
            </el-form-item>
            <el-form-item label="TLS 版本">
              <div class="row">
                <el-select v-model="form.minVersion" placeholder="最小" style="width: 50%">
                  <el-option v-for="v in ['1.0', '1.1', '1.2', '1.3']" :key="v" :label="v" :value="v" />
                </el-select>
                <el-select v-model="form.maxVersion" placeholder="最大" style="width: 50%">
                  <el-option v-for="v in ['1.0', '1.1', '1.2', '1.3']" :key="v" :label="v" :value="v" />
                </el-select>
              </div>
            </el-form-item>
          <el-divider content-position="left">QUIC</el-divider>
          <el-form-item label="空闲超时">
            <el-input v-model="form.idleTimeout" placeholder="如 30s" />
          </el-form-item>
          <el-form-item label="保活周期">
            <el-input v-model="form.keepAlivePeriod" placeholder="如 15s" />
          </el-form-item>
          <el-form-item label="初始包大小">
            <el-input-number v-model="form.initialPacketSize" :min="0" :max="10000" controls-position="right" style="width: 100%" />
          </el-form-item>
          <el-form-item label="禁用 MTU 发现">
            <el-switch v-model="form.disablePathMTU" />
          </el-form-item>
        </template>

        <!-- 其他类型：原始 JSON 兜底 -->
        <template v-else>
          <el-form-item label="listen" prop="listen">
            <el-input v-model="form.listen" placeholder="监听地址（可选）" />
          </el-form-item>
          <el-form-item label="listen_port" prop="listen_port">
            <div class="row">
              <el-input-number v-model="form.listen_port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
              <el-button :loading="allocating" @click="allocatePort">自动分配</el-button>
            </div>
          </el-form-item>
        </template>
      </el-form>
      </el-tab-pane>
      <el-tab-pane label="源码">
        <ResourceSourceTab ref="srcTab" :initial="sourceJson" />
      </el-tab-pane>
    </el-tabs>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
      </el-tab-pane>
      <el-tab-pane label="源码" name="source">
        <SourcePane segment="inbounds" @saved="loadInbounds" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.toolbar {
  margin-bottom: 14px;
  display: flex;
  gap: 10px;
}
.user-row {
  display: flex;
  gap: 8px;
  width: 100%;
}
.user-row .el-input {
  flex: 1;
}
.row {
  display: flex;
  gap: 8px;
  width: 100%;
}
.row .el-input-number {
  flex: 1;
}
.fill-hint {
  font-size: 12px;
  color: #909399;
  margin-left: 10px;
}
</style>
