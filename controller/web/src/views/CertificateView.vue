<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import SourcePane from '../components/SourcePane.vue'
import { useStatusStore } from '../stores/status'

const statusStore = useStatusStore()

const loading = ref(false)
const outerTab = ref('form')
const saving = ref(false)
const providers = ref<Array<{ id: string; provider: Record<string, any> }>>([])

const dialogVisible = ref(false)
const isEdit = ref(false)
const editingId = ref('')

// 顶层 certificate 段（option/certificate.go）
const certForm = reactive({
  store: 'system',
  certificate: [] as string[],
  certificate_path: [] as string[],
  certificate_directory_path: [] as string[]
})
const STORE_OPTIONS = ['system', 'mozilla', 'chrome', 'none']

// certificate_providers（option/certificate_provider.go）：acme
const PROVIDER_TYPES = ['acme']
const KEY_TYPE_OPTIONS = ['ed25519', 'p256', 'p384']
// dns01_challenge.provider 枚举（option/acme.go enum:"alidns,cloudflare,acmedns"）——只读选项，默认 cloudflare
const DNS01_PROVIDERS = ['alidns', 'cloudflare', 'acmedns'] as const

const providerForm = reactive({
  type: 'acme',
  tag: '',
  domain: [] as string[],
  data_directory: '',
  default_server_name: '',
  email: '',
  account_key: '',
  disable_http_challenge: false,
  disable_tls_alpn_challenge: false,
  alternative_http_port: 0,
  alternative_tls_port: 0,
  key_type: '',
  provider: 'cloudflare',
  extraJson: ''
})

const resetProviderForm = () => {
  providerForm.type = 'acme'
  providerForm.tag = ''
  providerForm.domain = []
  providerForm.data_directory = ''
  providerForm.default_server_name = ''
  providerForm.email = ''
  providerForm.provider = ''
  providerForm.account_key = ''
  providerForm.disable_http_challenge = false
  providerForm.disable_tls_alpn_challenge = false
  providerForm.alternative_http_port = 0
  providerForm.alternative_tls_port = 0
  providerForm.key_type = ''
  providerForm.provider = 'cloudflare'
  providerForm.extraJson = ''
}

const load = async () => {
  loading.value = true
  try {
    const data = await api.certificate()
    const cert = data.certificate as Record<string, any> | null
    if (cert && typeof cert === 'object') {
      certForm.store = String(cert.store || 'system')
      certForm.certificate = toList(cert.certificate)
      certForm.certificate_path = toList(cert.certificate_path)
      certForm.certificate_directory_path = toList(cert.certificate_directory_path)
    } else {
      certForm.store = 'system'
      certForm.certificate = []
      certForm.certificate_path = []
      certForm.certificate_directory_path = []
    }
    providers.value = data.providers || []
  } catch (e) {
    ElMessage.error((e as Error).message || '加载证书配置失败')
  } finally {
    loading.value = false
  }
}

const toList = (v: unknown): string[] => {
  if (v == null) return []
  return Array.isArray(v) ? v.map(String) : [String(v)]
}

const saveCertificate = async () => {
  saving.value = true
  try {
    const cert: Record<string, any> = {}
    if (certForm.store && certForm.store !== 'system') cert.store = certForm.store
    if (certForm.certificate.length) cert.certificate = [...certForm.certificate]
    if (certForm.certificate_path.length) cert.certificate_path = [...certForm.certificate_path]
    if (certForm.certificate_directory_path.length) cert.certificate_directory_path = [...certForm.certificate_directory_path]
    await api.saveCertificate(Object.keys(cert).length ? cert : null)
    ElMessage.success('certificate 已保存')
    await statusStore.refresh()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

function buildProvider(): Record<string, any> {
  const p: Record<string, any> = { type: providerForm.type }
  if (providerForm.tag.trim()) p.tag = providerForm.tag.trim()
  if (providerForm.domain.length) p.domain = [...providerForm.domain]
  if (providerForm.data_directory.trim()) p.data_directory = providerForm.data_directory.trim()
  if (providerForm.default_server_name.trim()) p.default_server_name = providerForm.default_server_name.trim()
  if (providerForm.email.trim()) p.email = providerForm.email.trim()
  if (providerForm.account_key.trim()) p.account_key = providerForm.account_key.trim()
  if (providerForm.disable_http_challenge) p.disable_http_challenge = true
  if (providerForm.disable_tls_alpn_challenge) p.disable_tls_alpn_challenge = true
  if (providerForm.alternative_http_port) p.alternative_http_port = Number(providerForm.alternative_http_port)
  if (providerForm.alternative_tls_port) p.alternative_tls_port = Number(providerForm.alternative_tls_port)
  if (providerForm.key_type) p.key_type = providerForm.key_type
  // extraJson 兜底（dns01_challenge 特殊合并：provider 走表单枚举）
  if (providerForm.extraJson.trim()) {
    let extra: Record<string, any>
    try {
      extra = JSON.parse(providerForm.extraJson.trim())
    } catch (e) {
      throw new Error(`附加字段 JSON 格式错误：${(e as Error).message}`)
    }
    const d01 = extra.dns01_challenge
    if (d01 && typeof d01 === 'object') {
      const merged = { ...d01 }
      if (providerForm.provider) merged.provider = providerForm.provider
      p.dns01_challenge = merged
      delete extra.dns01_challenge
    } else if (providerForm.provider) {
      p.dns01_challenge = { provider: providerForm.provider }
    }
    for (const k of Object.keys(extra)) {
      if (!(k in p) && k !== 'type') p[k] = extra[k]
    }
  } else if (providerForm.provider) {
    p.dns01_challenge = { provider: providerForm.provider }
  }
  return p
}

function openCreateProvider() {
  isEdit.value = false
  editingId.value = ''
  resetProviderForm()
  dialogVisible.value = true
}

function openEditProvider(row: { id: string; provider: Record<string, any> }) {
  isEdit.value = true
  editingId.value = row.id
  resetProviderForm()
  const p = row.provider
  providerForm.type = String(p.type || 'acme')
  providerForm.tag = typeof p.tag === 'string' ? p.tag : ''
  providerForm.domain = toList(p.domain)
  providerForm.data_directory = typeof p.data_directory === 'string' ? p.data_directory : ''
  providerForm.default_server_name = typeof p.default_server_name === 'string' ? p.default_server_name : ''
  providerForm.email = typeof p.email === 'string' ? p.email : ''
  providerForm.account_key = typeof p.account_key === 'string' ? p.account_key : ''
  providerForm.disable_http_challenge = p.disable_http_challenge === true
  providerForm.disable_tls_alpn_challenge = p.disable_tls_alpn_challenge === true
  providerForm.alternative_http_port = Number(p.alternative_http_port || 0)
  providerForm.alternative_tls_port = Number(p.alternative_tls_port || 0)
  providerForm.key_type = typeof p.key_type === 'string' ? p.key_type : ''
  const extra: Record<string, any> = {}
  // dns01_challenge.provider（枚举选择）；其余 dns01 字段（云厂商凭证等）保留在 extraJson
  const d01 = p.dns01_challenge
  if (d01 && typeof d01 === 'object') {
    providerForm.provider = typeof d01.provider === 'string' ? d01.provider : ''
    const rest: Record<string, any> = { ...d01 }
    delete rest.provider
    if (Object.keys(rest).length) extra.dns01_challenge = rest
  } else {
    providerForm.provider = ''
  }
  const known = new Set([
    'type', 'tag', 'domain', 'data_directory', 'default_server_name', 'email', 'provider',
    'account_key', 'disable_http_challenge', 'disable_tls_alpn_challenge',
    'alternative_http_port', 'alternative_tls_port', 'key_type', 'dns01_challenge'
  ])
  for (const k of Object.keys(p)) {
    if (!known.has(k)) extra[k] = p[k]
  }
  providerForm.extraJson = Object.keys(extra).length ? JSON.stringify(extra, null, 2) : ''
  dialogVisible.value = true
}

const saveProvider = async () => {
  saving.value = true
  try {
    const payload = buildProvider()
    if (isEdit.value) {
      await api.updateCertProvider(editingId.value, payload)
    } else {
      await api.createCertProvider(payload)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    await load()
    await statusStore.refresh()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

const removeProvider = async (row: { id: string; provider: Record<string, any> }) => {
  try {
    await ElMessageBox.confirm(`确定删除 certificate provider「${row.provider.tag || row.id}」？`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  try {
    await api.deleteCertProvider(row.id)
    ElMessage.success('删除成功')
    await load()
    await statusStore.refresh()
  } catch (e) {
    const err = e as Error & { references?: string[] }
    if (err.references?.length) {
      try {
        await ElMessageBox.confirm(
          `该 provider 被引用：${err.references.join('、')}\\n删除后将自动清除引用。确认删除？`,
          '被引用确认',
          { type: 'warning', confirmButtonText: '确认删除', cancelButtonText: '取消' }
        )
      } catch {
        return
      }
      try {
        await api.deleteCertProvider(row.id, true)
        ElMessage.success('删除成功（已清除引用）')
        await load()
        await statusStore.refresh()
      } catch (e2) {
        ElMessage.error((e2 as Error).message || '删除失败')
      }
      return
    }
    ElMessage.error(err.message || '删除失败')
  }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <el-tabs v-model="outerTab">
      <el-tab-pane label="表单" name="form">
    <div class="section">
      <div class="section-title">certificate（全局证书配置）</div>
      <el-form label-width="180px" style="max-width: 720px">
        <el-form-item label="store">
          <el-select v-model="certForm.store" style="width: 100%">
            <el-option v-for="s in STORE_OPTIONS" :key="s" :label="s" :value="s" />
          </el-select>
          <span class="hint" style="margin-left: 8px">system 为默认（省略时）</span>
        </el-form-item>
        <el-form-item label="certificate">
          <el-select v-model="certForm.certificate" multiple filterable allow-create default-first-option style="width: 100%" placeholder="PEM 证书内容（可多值）" />
        </el-form-item>
        <el-form-item label="certificate_path">
          <el-select v-model="certForm.certificate_path" multiple filterable allow-create default-first-option style="width: 100%" placeholder="证书文件路径（可多值）" />
        </el-form-item>
        <el-form-item label="certificate_directory_path">
          <el-select v-model="certForm.certificate_directory_path" multiple filterable allow-create default-first-option style="width: 100%" placeholder="证书目录路径（可多值）" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="saveCertificate">保存 certificate</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="section">
      <div class="section-title">
        certificate_providers
        <el-button size="small" type="primary" style="margin-left: 12px" @click="openCreateProvider">新建 Provider</el-button>
        <span class="hint" style="margin-left: 8px">多态 provider（acme）；被 tls.certificate_provider 引用</span>
      </div>
      <el-table :data="providers" v-loading="loading" border stripe>
        <el-table-column label="tag" min-width="120">
          <template #default="{ row }">{{ row.provider.tag || '—' }}</template>
        </el-table-column>
        <el-table-column prop="type" label="type" width="90">
          <template #default="{ row }">{{ row.provider.type }}</template>
        </el-table-column>
        <el-table-column label="域名" min-width="200">
          <template #default="{ row }">{{ Array.isArray(row.provider.domain) ? row.provider.domain.join(', ') : row.provider.domain || '—' }}</template>
        </el-table-column>
        <el-table-column label="email" min-width="180">
          <template #default="{ row }">{{ row.provider.email || '—' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" link @click="openEditProvider(row)">编辑</el-button>
            <el-button size="small" type="danger" link @click="removeProvider(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑 Certificate Provider' : '新建 Certificate Provider'" width="640px" :close-on-click-modal="false">
      <el-form label-width="170px">
        <el-form-item label="type">
          <el-select v-model="providerForm.type" style="width: 100%">
            <el-option v-for="t in PROVIDER_TYPES" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>
        <el-form-item label="tag">
          <el-input v-model="providerForm.tag" placeholder="provider tag（供 tls.certificate_provider 引用，可选）" />
        </el-form-item>
        <el-divider content-position="left">ACME 选项</el-divider>
        <el-form-item label="domain">
          <el-select v-model="providerForm.domain" multiple filterable allow-create default-first-option style="width: 100%" placeholder="证书域名（可多值）" />
        </el-form-item>
        <el-form-item label="email">
          <el-input v-model="providerForm.email" placeholder="ACME 账户邮箱" />
        </el-form-item>
        <el-form-item label="data_directory">
          <el-input v-model="providerForm.data_directory" placeholder="证书数据目录（可选）" />
        </el-form-item>
        <el-form-item label="default_server_name">
          <el-input v-model="providerForm.default_server_name" placeholder="默认服务器名（可选）" />
        </el-form-item>
        <el-form-item label="DNS-01 服务商">
          <el-select v-model="providerForm.provider" clearable style="width: 100%" placeholder="不需要时清空（默认 cloudflare）">
            <el-option v-for="d in DNS01_PROVIDERS" :key="d" :label="d" :value="d" />
          </el-select>
          <div class="field-hint">写入 dns01_challenge.provider（源码枚举 alidns/cloudflare/acmedns）；对应凭证（如 cloudflare.api_token）写在附加字段</div>
        </el-form-item>
        <el-form-item label="account_key">
          <el-input v-model="providerForm.account_key" placeholder="账户密钥（可选）" />
        </el-form-item>
        <el-form-item label="key_type">
          <el-select v-model="providerForm.key_type" style="width: 100%" clearable placeholder="默认">
            <el-option v-for="k in KEY_TYPE_OPTIONS" :key="k" :label="k" :value="k" />
          </el-select>
        </el-form-item>
        <el-form-item label="禁用 HTTP 挑战">
          <el-switch v-model="providerForm.disable_http_challenge" />
        </el-form-item>
        <el-form-item label="禁用 TLS-ALPN 挑战">
          <el-switch v-model="providerForm.disable_tls_alpn_challenge" />
        </el-form-item>
        <el-form-item label="alternative_http_port">
          <el-input-number v-model="providerForm.alternative_http_port" :min="0" :max="65535" />
        </el-form-item>
        <el-form-item label="alternative_tls_port">
          <el-input-number v-model="providerForm.alternative_tls_port" :min="0" :max="65535" />
        </el-form-item>
        <el-form-item label="附加字段 (JSON)">
          <el-input v-model="providerForm.extraJson" type="textarea" :rows="4" class="mono" placeholder='{"dns01_challenge": {"provider": "cloudflare", "cloudflare": {"api_token": "..."}}}' />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveProvider">保存</el-button>
      </template>
    </el-dialog>
      </el-tab-pane>
      <el-tab-pane label="源码" name="source">
        <SourcePane segment="certificate" @saved="load" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.section {
  margin-bottom: 20px;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  padding: 16px;
}
.section-title {
  font-weight: 600;
  color: #303133;
  margin-bottom: 14px;
  display: flex;
  align-items: center;
}
.hint {
  font-size: 12px;
  color: #909399;
  font-weight: 400;
}
</style>
