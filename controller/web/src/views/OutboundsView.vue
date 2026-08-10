<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { api } from '../api'
import type { Outbound } from '../api'
import { useStatusStore } from '../stores/status'

const statusStore = useStatusStore()

const loading = ref(false)
const outbounds = ref<Outbound[]>([])
const outboundTypes = ref<string[]>([])

const dialogVisible = ref(false)
const isEdit = ref(false)
const editingTag = ref('')
const saving = ref(false)
const generating = ref(false)
const suppressTypeWatch = ref(false)
const formRef = ref<FormInstance>()

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
  rawJson: string
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
  interrupt_exist_connections: false,
  rawJson: ''
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

const rules = computed<FormRules>(() => {
  const base: FormRules = {
    tag: [{ required: true, message: 'tag 必填', trigger: 'blur' }]
  }
  // selector/urltest 无 server/server_port
  if (!isSelector.value && !isUrlTest.value) {
    base.server = [{ required: true, message: 'server 必填', trigger: 'blur' }]
    base.server_port = [
      { required: true, message: 'server_port 必填', trigger: 'blur' },
      { type: 'number', min: 1, max: 65535, message: '端口范围 1-65535', trigger: 'blur' }
    ]
  }
  if (isVless.value || isTuic.value) {
    base.uuid = [{ required: true, message: 'uuid 必填', trigger: 'blur' }]
    if (isTuic.value) {
      base.password = [{ required: true, message: 'password 必填', trigger: 'blur' }]
    }
  }
  if (isSelector.value || isUrlTest.value) {
    base.outbounds = [{ required: true, type: 'array', min: 1, message: '至少选择一个成员 outbound', trigger: 'change' }]
  }
  if (form.tls.enabled) {
    base['tls.server_name'] = [{ required: true, message: '启用 TLS 时 server_name 必填', trigger: 'blur' }]
  }
  if (form.tls.reality.enabled) {
    base['tls.reality.public_key'] = [{ required: true, message: '启用 Reality 时 public_key 必填', trigger: 'blur' }]
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
  form.rawJson = ''
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
  if (form.type === 'vless' || form.type === 'tuic') {
    form.rawJson = ''
  } else if (form.type === 'selector' || form.type === 'urltest') {
    form.outbounds = Array.isArray(obj.outbounds) ? obj.outbounds.map(String) : []
    form.default_out = str(obj.default)
    form.url = str(obj.url)
    form.interval = str(obj.interval)
    form.tolerance = num(obj.tolerance)
    form.interrupt_exist_connections = !!obj.interrupt_exist_connections
    form.rawJson = ''
  } else {
    const rest: Record<string, unknown> = { ...obj }
    delete rest.type
    delete rest.tag
    delete rest.server
    delete rest.server_port
    form.rawJson = Object.keys(rest).length ? JSON.stringify(rest, null, 2) : ''
  }
}

const openCreate = () => {
  isEdit.value = false
  editingTag.value = ''
  const def = statusStore.status?.defaults?.outbound_type
  resetForm(def && outboundTypes.value.includes(def) ? def : outboundTypes.value[0] || 'vless')
  dialogVisible.value = true
  formRef.value?.clearValidate()
}

const openEdit = async (row: Outbound) => {
  isEdit.value = true
  editingTag.value = row.tag
  dialogVisible.value = true
  formRef.value?.clearValidate()
  try {
    fillForm(await api.getOutbound(row.tag))
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
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

// 粘贴 JSON → 后端解析 → 填充表单字段
const fillFromJson = async () => {
  let text: string
  try {
    const { value } = await ElMessageBox.prompt(
      '粘贴 outbound JSON，解析成功后填充表单字段（type 变化会重置表单，请最后检查）',
      '从 JSON 填充',
      {
        confirmButtonText: '解析并填充',
        cancelButtonText: '取消',
        inputType: 'textarea',
        inputPlaceholder: '{"type":"vless","tag":"...","server":"...","server_port":443,"uuid":"...","tls":{...}}'
      }
    )
    text = value
  } catch {
    return
  }
  try {
    const res = await api.parseJson(text)
    suppressTypeWatch.value = true
    fillForm(res.data as Outbound)
    suppressTypeWatch.value = false
    ElMessage.success('已从 JSON 填充，请检查表单')
  } catch (e) {
    ElMessage.error((e as Error).message || '解析失败')
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

  // 其他类型：通用字段 + 原始 JSON 合并
  if (form.rawJson.trim()) {
    let extra: Record<string, unknown>
    try {
      extra = JSON.parse(form.rawJson.trim())
    } catch (e) {
      throw new Error(`原始 JSON 格式错误：${(e as Error).message}`)
    }
    if (typeof extra !== 'object' || extra === null || Array.isArray(extra)) {
      throw new Error('原始 JSON 必须为 JSON 对象')
    }
    Object.assign(body, extra)
    body.type = form.type
    if (!body.tag) body.tag = form.tag.trim()
  }
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
    const body = buildBody()
    const res = isEdit.value ? await api.updateOutbound(editingTag.value, body) : await api.createOutbound(body)
    handleResult(res)
    dialogVisible.value = false
    await Promise.all([loadOutbounds(), statusStore.refresh()])
  } catch (e) {
    ElMessage.error((e as Error).message || '操作失败')
  } finally {
    saving.value = false
  }
}

const remove = async (row: Outbound) => {
  try {
    await ElMessageBox.confirm(`确定删除 outbound「${row.tag}」？该操作不可恢复。`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  try {
    await api.deleteOutbound(row.tag)
    ElMessage.success('删除成功')
    await Promise.all([loadOutbounds(), statusStore.refresh()])
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}

const genUuid = async () => {
  generating.value = true
  try {
    form.uuid = await api.genUuid()
  } catch (e) {
    ElMessage.error((e as Error).message || 'UUID 生成失败')
  } finally {
    generating.value = false
  }
}

const genKeypair = async () => {
  generating.value = true
  try {
    const kp = await api.genRealityKeypair()
    form.tls.reality.public_key = kp.public_key
    ElMessage({
      type: 'success',
      dangerouslyUseHTMLString: true,
      duration: 8000,
      message: `已生成 Reality 密钥对<br/>private_key: <b>${kp.private_key}</b><br/><i>私钥仅展示一次，请妥善保存；public_key 已自动填入表单。</i>`
    })
  } catch (e) {
    ElMessage.error((e as Error).message || '密钥对生成失败')
  } finally {
    generating.value = false
  }
}

const loadOutbounds = async () => {
  loading.value = true
  try {
    outbounds.value = await api.outbounds()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载 outbounds 失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    const t = await api.types()
    outboundTypes.value = t.outbounds || []
  } catch (e) {
    ElMessage.error((e as Error).message || '加载类型列表失败')
  }
  await loadOutbounds()
})
</script>

<template>
  <div class="page">
    <div class="toolbar">
      <el-button type="primary" @click="openCreate">新建 Outbound</el-button>
      <el-button :loading="loading" @click="loadOutbounds">刷新</el-button>
    </div>

    <el-table :data="outbounds" v-loading="loading" border stripe>
      <el-table-column prop="type" label="类型" width="130" />
      <el-table-column prop="tag" label="tag" min-width="160" />
      <el-table-column label="server" min-width="160">
        <template #default="{ row }">{{ row.server ?? '—' }}</template>
      </el-table-column>
      <el-table-column label="server_port" width="120">
        <template #default="{ row }">{{ row.server_port ?? '—' }}</template>
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
      :title="isEdit ? '编辑 Outbound' : '新建 Outbound'"
      width="720px"
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="170px">
        <el-form-item>
          <el-button size="small" @click="fillFromJson">从 JSON 填充（粘贴解析）</el-button>
          <span class="fill-hint">粘贴完整 outbound JSON，自动解析并填充下方字段</span>
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-select v-model="form.type" :disabled="isEdit" style="width: 100%">
            <el-option v-for="t in outboundTypes" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>
        <el-form-item label="tag" prop="tag">
          <el-input v-model="form.tag" placeholder="唯一标识，如 vless-out" />
        </el-form-item>
        <template v-if="!isSelector && !isUrlTest">
          <el-form-item label="server" prop="server">
            <el-input v-model="form.server" placeholder="服务器地址 / IP" />
          </el-form-item>
          <el-form-item label="server_port" prop="server_port">
            <el-input-number v-model="form.server_port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
          </el-form-item>
        </template>

        <!-- vless 专属字段 -->
        <template v-if="isVless">
          <el-form-item label="uuid" prop="uuid">
            <div class="row">
              <el-input v-model="form.uuid" placeholder="UUID" />
              <el-button :loading="generating" @click="genUuid">生成</el-button>
            </div>
          </el-form-item>
          <el-form-item label="flow" prop="flow">
            <el-select v-model="form.flow" allow-create filterable clearable style="width: 100%" placeholder="留空则不启用">
              <el-option label="xtls-rprx-vision" value="xtls-rprx-vision" />
              <el-option label="xtls-rprx-vision-udp443" value="xtls-rprx-vision-udp443" />
            </el-select>
          </el-form-item>
          <el-form-item label="network" prop="network">
            <el-select v-model="form.network" style="width: 100%">
              <el-option v-for="n in networkOptionsAll" :key="n" :label="n" :value="n" />
            </el-select>
          </el-form-item>

          <el-form-item label="tls.enabled">
            <el-switch v-model="form.tls.enabled" />
          </el-form-item>
          <template v-if="form.tls.enabled">
            <el-form-item label="tls.server_name" prop="tls.server_name">
              <el-input v-model="form.tls.server_name" placeholder="SNI，如 www.example.com" />
            </el-form-item>
            <el-form-item label="tls.utls.enabled">
              <el-switch v-model="form.tls.utls.enabled" />
            </el-form-item>
            <el-form-item v-if="form.tls.utls.enabled" label="tls.utls.fingerprint">
              <el-select v-model="form.tls.utls.fingerprint" style="width: 100%">
                <el-option v-for="f in fingerprintOptions" :key="f" :label="f" :value="f" />
              </el-select>
            </el-form-item>
            <el-form-item label="tls.reality.enabled">
              <el-switch v-model="form.tls.reality.enabled" />
            </el-form-item>
            <template v-if="form.tls.reality.enabled">
              <el-form-item label="tls.reality.public_key" prop="tls.reality.public_key">
                <div class="row">
                  <el-input v-model="form.tls.reality.public_key" placeholder="Reality 公钥" />
                  <el-button :loading="generating" @click="genKeypair">生成密钥对</el-button>
                </div>
              </el-form-item>
              <el-form-item label="tls.reality.short_id">
                <el-input v-model="form.tls.reality.short_id" placeholder="short_id，可为空" />
              </el-form-item>
            </template>
          </template>
        </template>

        <!-- tuic 专属字段 -->
        <template v-else-if="isTuic">
          <el-form-item label="uuid" prop="uuid">
            <div class="row">
              <el-input v-model="form.uuid" placeholder="UUID" />
              <el-button :loading="generating" @click="genUuid">生成</el-button>
            </div>
          </el-form-item>
          <el-form-item label="password" prop="password">
            <el-input v-model="form.password" type="password" show-password placeholder="tuic 密码" />
          </el-form-item>
          <el-form-item label="congestion_control">
            <el-select v-model="form.congestion_control" style="width: 100%">
              <el-option v-for="c in congestionOptions" :key="c" :label="c" :value="c" />
            </el-select>
          </el-form-item>
          <el-form-item label="udp_relay_mode">
            <el-select v-model="form.udp_relay_mode" style="width: 100%">
              <el-option v-for="m in udpRelayOptions" :key="m" :label="m" :value="m" />
            </el-select>
          </el-form-item>
          <el-form-item label="zero_rtt_handshake">
            <el-switch v-model="form.zero_rtt_handshake" />
          </el-form-item>
          <el-form-item label="tls.enabled">
            <el-switch v-model="form.tls.enabled" />
          </el-form-item>
          <template v-if="form.tls.enabled">
            <el-form-item label="tls.server_name" prop="tls.server_name">
              <el-input v-model="form.tls.server_name" placeholder="SNI，如 www.example.com" />
            </el-form-item>
            <el-form-item label="tls.alpn">
              <el-select v-model="form.tls.alpn" multiple style="width: 100%">
                <el-option v-for="a in alpnOptions" :key="a" :label="a" :value="a" />
              </el-select>
            </el-form-item>
          </template>
        </template>

        <!-- selector / urltest 组字段 -->
        <template v-else-if="isSelector || isUrlTest">
          <el-form-item label="成员 outbounds" prop="outbounds">
            <el-select v-model="form.outbounds" multiple filterable style="width: 100%" placeholder="选择组成员（可多选）">
              <el-option v-for="t in groupCandidates" :key="t" :label="t" :value="t" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="isSelector" label="default">
            <el-select v-model="form.default_out" clearable filterable style="width: 100%" placeholder="可选，默认选中项">
              <el-option v-for="t in form.outbounds" :key="t" :label="t" :value="t" />
            </el-select>
          </el-form-item>
          <template v-if="isUrlTest">
            <el-form-item label="url">
              <el-input v-model="form.url" placeholder="测试地址，默认 https://www.gstatic.com/generate_204" />
            </el-form-item>
            <el-form-item label="interval">
              <el-input v-model="form.interval" placeholder="测试间隔，如 3m / 300s" />
            </el-form-item>
            <el-form-item label="tolerance">
              <el-input-number v-model="form.tolerance" :min="0" :max="65535" controls-position="right" style="width: 100%" />
            </el-form-item>
          </template>
          <el-form-item label="interrupt_exist_connections">
            <el-switch v-model="form.interrupt_exist_connections" />
          </el-form-item>
        </template>

        <!-- 其他类型：原始 JSON 兜底 -->
        <el-form-item v-else label="其他字段 (JSON)">
          <el-input
            v-model="form.rawJson"
            type="textarea"
            :rows="8"
            class="mono"
            placeholder='{"users": [{"username": "a", "password": "b"}], "network": "tcp"}'
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.toolbar {
  margin-bottom: 14px;
  display: flex;
  gap: 10px;
}
.row {
  display: flex;
  gap: 8px;
  width: 100%;
}
.row .el-input {
  flex: 1;
}
.fill-hint {
  font-size: 12px;
  color: #909399;
  margin-left: 10px;
}
</style>
