<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { showToast } from '@/helper/toast'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { PlusIcon } from 'lucide-vue-next'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { SelectField } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { api } from '../api'
import SourcePane from '../components/SourcePane.vue'
import ResourceSourceTab from '../components/ResourceSourceTab.vue'
import { useStatusStore } from '../stores/status'

const statusStore = useStatusStore()

const loading = ref(false)
const outerTab = ref('form')
const saving = ref(false)
const providers = ref<Array<{ id: string; provider: Record<string, any> }>>([])

const dialogVisible = ref(false)
const srcTab = ref<InstanceType<typeof ResourceSourceTab>>()
const sourceJson = ref('{}')
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
  dns01_challenge_json: ''
})

// chip 输入（替代 el-select multiple+allow-create）
const certChip = { certificate: '', certificate_path: '', certificate_directory_path: '' }
const addChip = (list: string[], input: { value: string }) => {
  const parts = input.value
    .split(/[,，\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  for (const p of parts) {
    if (!list.includes(p)) list.push(p)
  }
  input.value = ''
}
const removeChip = (list: string[], t: string) => {
  const i = list.indexOf(t)
  if (i >= 0) list.splice(i, 1)
}

const domainInput = ref('')
const addDomain = () => {
  const parts = domainInput.value
    .split(/[,，\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  for (const p of parts) {
    if (!providerForm.domain.includes(p)) providerForm.domain.push(p)
  }
  domainInput.value = ''
}

const resetProviderForm = () => {
  providerForm.type = 'acme'
  providerForm.tag = ''
  providerForm.domain = []
  providerForm.data_directory = ''
  providerForm.default_server_name = ''
  providerForm.email = ''
  providerForm.account_key = ''
  providerForm.disable_http_challenge = false
  providerForm.disable_tls_alpn_challenge = false
  providerForm.alternative_http_port = 0
  providerForm.alternative_tls_port = 0
  providerForm.key_type = ''
  providerForm.provider = 'cloudflare'
  providerForm.dns01_challenge_json = ''
  domainInput.value = ''
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
    showToast((e as Error).message || '加载证书配置失败', 'error')
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
    showToast('certificate 已保存', 'success')
    await statusStore.refresh()
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
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
  // dns01_challenge：字段级 JSON（云厂商凭证等）+ provider 下拉（表单值优先）
  let d01: Record<string, any> | undefined
  if (providerForm.dns01_challenge_json.trim()) {
    try {
      const parsed = JSON.parse(providerForm.dns01_challenge_json.trim())
      if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
        throw new Error('dns01_challenge 必须为 JSON 对象')
      }
      d01 = { ...parsed }
    } catch (e) {
      throw new Error(`dns01_challenge JSON 格式错误：${(e as Error).message}`)
    }
  }
  if (providerForm.provider) {
    d01 = d01 || {}
    d01.provider = providerForm.provider
  }
  if (d01) p.dns01_challenge = d01
  return p
}

function openCreateProvider() {
  isEdit.value = false
  editingId.value = ''
  sourceJson.value = '{}'
  resetProviderForm()
  dialogVisible.value = true
}

function openEditProvider(row: { id: string; provider: Record<string, any> }) {
  isEdit.value = true
  editingId.value = row.id
  sourceJson.value = JSON.stringify(row.provider, null, 2)
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
  // dns01_challenge.provider（枚举选择）；其余 dns01 字段（云厂商凭证等）保留在字段级 JSON
  const d01 = p.dns01_challenge
  if (d01 && typeof d01 === 'object') {
    providerForm.provider = typeof d01.provider === 'string' ? d01.provider : ''
    const rest: Record<string, any> = { ...d01 }
    delete rest.provider
    providerForm.dns01_challenge_json = Object.keys(rest).length ? JSON.stringify(rest, null, 2) : ''
  } else {
    providerForm.provider = ''
    providerForm.dns01_challenge_json = ''
  }
  dialogVisible.value = true
}

const saveProvider = async () => {
  saving.value = true
  try {
    const payload = srcTab.value?.isDirty() ? JSON.parse(srcTab.value.getText()) : buildProvider()
    if (isEdit.value) {
      await api.updateCertProvider(editingId.value, payload)
    } else {
      await api.createCertProvider(payload)
    }
    showToast('保存成功', 'success')
    dialogVisible.value = false
    await load()
    await statusStore.refresh()
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
  } finally {
    saving.value = false
  }
}

const removeProvider = async (row: { id: string; provider: Record<string, any> }) => {
  const first = await showConfirmDialog({
    title: '删除确认',
    message: `确定删除 certificate provider「${row.provider.tag || row.id}」？`,
    confirmText: '删除',
    confirmButtonClass: 'btn-error'
  })
  if (!first.confirmed) return
  try {
    await api.deleteCertProvider(row.id)
    showToast('删除成功', 'success')
    await load()
    await statusStore.refresh()
  } catch (e) {
    const err = e as Error & { references?: string[] }
    if (err.references?.length) {
      const second = await showConfirmDialog({
        title: '被引用确认',
        message: `该 provider 被引用：${err.references.join('、')}\n删除后将自动清除引用。确认删除？`,
        confirmText: '确认删除',
        confirmButtonClass: 'btn-error'
      })
      if (!second.confirmed) return
      try {
        await api.deleteCertProvider(row.id, true)
        showToast('删除成功（已清除引用）', 'success')
        await load()
        await statusStore.refresh()
      } catch (e2) {
        showToast((e2 as Error).message || '删除失败', 'error')
      }
      return
    }
    showToast(err.message || '删除失败', 'error')
  }
}

onMounted(load)
</script>

<template>
  <div>
    <TabsRoot v-model="outerTab">
      <TabsList>
        <TabsTrigger value="form">表单</TabsTrigger>
        <TabsTrigger value="source">源码</TabsTrigger>
      </TabsList>

      <TabsContent value="form">
        <div class="mb-5 rounded-lg border border-[#e4e7ed] bg-white p-4 dark:border-[#303030] dark:bg-[#1d1e1f]">
          <div class="mb-3.5 flex items-center font-semibold text-[#303133] dark:text-[#e5eaf3]">certificate（全局证书配置）</div>
          <div class="grid max-w-[720px] gap-x-4 gap-y-5" style="grid-template-columns: 180px minmax(0, 1fr)">
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">store</label>
            <div class="flex items-center gap-2">
              <SelectField v-model="certForm.store" class="w-full" :options="STORE_OPTIONS" />
              <span class="text-xs text-[#909399]">system 为默认（省略时）</span>
            </div>

            <template v-for="key in (['certificate', 'certificate_path', 'certificate_directory_path'] as const)" :key="key">
              <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">{{ key }}</label>
              <div class="flex flex-col gap-1.5">
                <div class="flex flex-wrap gap-1.5">
                  <span v-for="t in certForm[key]" :key="t" class="badge badge-primary badge-outline gap-1">
                    {{ t }}
                    <button type="button" class="cursor-pointer text-xs opacity-70 hover:opacity-100" @click="removeChip(certForm[key], t)">×</button>
                  </span>
                </div>
                <input
                  :value="certChip[key]"
                  type="text"
                  class="input input-bordered input-sm w-full"
                  :placeholder="`${key}（回车添加，可多值）`"
                  @input="certChip[key] = ($event.target as HTMLInputElement).value"
                  @keydown.enter.prevent="addChip(certForm[key], certChip[key] ? { value: certChip[key] } : { value: '' })"
                  @blur="certChip[key] && addChip(certForm[key], { value: certChip[key] })"
                />
              </div>
            </template>

            <span />
            <button class="btn btn-primary btn-sm w-fit" :disabled="saving" @click="saveCertificate">保存 certificate</button>
          </div>
        </div>

        <div class="rounded-lg border border-[#e4e7ed] bg-white p-4 dark:border-[#303030] dark:bg-[#1d1e1f]">
          <div class="mb-3.5 flex items-center font-semibold text-[#303133] dark:text-[#e5eaf3]">
            certificate_providers
            <button class="btn btn-primary btn-sm ml-3" @click="openCreateProvider">
              <PlusIcon class="h-4 w-4" />
              新建 Provider
            </button>
            <span class="ml-2 text-xs font-normal text-[#909399]">多态 provider（acme）；被 tls.certificate_provider 引用</span>
          </div>
          <div class="overflow-hidden rounded-lg border border-[#e4e7ed] dark:border-[#303030]">
            <div v-if="loading && !providers.length" class="p-10 text-center text-sm text-[#909399]">加载中…</div>
            <table v-else class="table table-sm w-full">
              <thead>
                <tr>
                  <th class="w-[120px]">tag</th>
                  <th class="w-20">type</th>
                  <th>域名</th>
                  <th class="w-[180px]">email</th>
                  <th class="w-[130px] text-right">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in providers" :key="row.id">
                  <td class="text-xs">{{ row.provider.tag || '—' }}</td>
                  <td class="text-xs">{{ row.provider.type }}</td>
                  <td class="text-xs">{{ Array.isArray(row.provider.domain) ? row.provider.domain.join(', ') : row.provider.domain || '—' }}</td>
                  <td class="text-xs">{{ row.provider.email || '—' }}</td>
                  <td class="text-right">
                    <button class="btn btn-ghost btn-xs text-primary" @click="openEditProvider(row)">编辑</button>
                    <button class="btn btn-ghost btn-xs text-error" @click="removeProvider(row)">删除</button>
                  </td>
                </tr>
              </tbody>
            </table>
            <div v-if="!loading && !providers.length" class="py-8 text-center text-sm text-[#606266] dark:text-[#a6b0bf]">暂无 provider</div>
          </div>
        </div>
      </TabsContent>

      <TabsContent value="source">
        <SourcePane segment="certificate" @saved="load" />
      </TabsContent>
    </TabsRoot>

    <!-- 新建/编辑 Provider 弹窗 -->
    <DialogWrapper v-model="dialogVisible" :title="isEdit ? '编辑 Certificate Provider' : '新建 Certificate Provider'" box-class="max-w-[640px]">
      <TabsRoot :model-value="'form'">
        <TabsList>
          <TabsTrigger value="form">表单</TabsTrigger>
          <TabsTrigger value="source">源码</TabsTrigger>
        </TabsList>
        <TabsContent value="form">
          <div class="grid gap-x-4 gap-y-5" style="grid-template-columns: 170px minmax(0, 1fr)">
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">type</label>
            <SelectField v-model="providerForm.type" :options="PROVIDER_TYPES"  />
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">tag</label>
            <input v-model="providerForm.tag" type="text" class="input input-bordered input-sm w-full" placeholder="provider tag（供 tls.certificate_provider 引用，可选）" />

            <div class="flex items-center gap-2 text-[13px] font-semibold text-[#303133] dark:text-[#e5eaf3]" style="grid-column: 1 / -1">
              <span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />ACME 选项<span class="h-px flex-1 bg-[#e4e7ed] dark:bg-[#303030]" />
            </div>

            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">domain</label>
            <div class="flex flex-col gap-1.5">
              <div class="flex flex-wrap gap-1.5">
                <span v-for="t in providerForm.domain" :key="t" class="badge badge-primary badge-outline gap-1">
                  {{ t }}
                  <button type="button" class="cursor-pointer text-xs opacity-70 hover:opacity-100" @click="removeChip(providerForm.domain, t)">×</button>
                </span>
              </div>
              <input v-model="domainInput" type="text" class="input input-bordered input-sm w-full" placeholder="证书域名（回车添加，可多值）" @keydown.enter.prevent="addDomain" @blur="addDomain" />
            </div>
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">email</label>
            <input v-model="providerForm.email" type="text" class="input input-bordered input-sm w-full" placeholder="ACME 账户邮箱" />
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">data_directory</label>
            <input v-model="providerForm.data_directory" type="text" class="input input-bordered input-sm w-full" placeholder="证书数据目录（可选）" />
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">default_server_name</label>
            <input v-model="providerForm.default_server_name" type="text" class="input input-bordered input-sm w-full" placeholder="默认服务器名（可选）" />
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">DNS-01 服务商</label>
            <div class="flex flex-col gap-1">
              <SelectField v-model="providerForm.provider" :options="DNS01_PROVIDERS" placeholder="不需要时清空（默认 cloudflare）"  />
              <p class="text-xs text-[#909399]">写入 dns01_challenge.provider（源码枚举 alidns/cloudflare/acmedns）；对应凭证（如 cloudflare.api_token）写在附加字段</p>
            </div>
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">account_key</label>
            <input v-model="providerForm.account_key" type="text" class="input input-bordered input-sm w-full" placeholder="账户密钥（可选）" />
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">key_type</label>
            <SelectField v-model="providerForm.key_type" :options="KEY_TYPE_OPTIONS" placeholder="默认"  />
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">禁用 HTTP 挑战</label>
            <Switch v-model="providerForm.disable_http_challenge" class="self-center" />
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">禁用 TLS-ALPN 挑战</label>
            <Switch v-model="providerForm.disable_tls_alpn_challenge" class="self-center" />
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">alternative_http_port</label>
            <input v-model.number="providerForm.alternative_http_port" type="number" min="0" max="65535" class="input input-bordered input-sm w-40" />
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">alternative_tls_port</label>
            <input v-model.number="providerForm.alternative_tls_port" type="number" min="0" max="65535" class="input input-bordered input-sm w-40" />
            <label class="pt-2 text-sm text-[#606266] dark:text-[#a6b0bf]">dns01_challenge (JSON)</label>
            <div class="flex flex-col gap-1">
              <textarea v-model="providerForm.dns01_challenge_json" rows="4" class="textarea textarea-bordered w-full font-mono text-xs" placeholder='{"cloudflare": {"api_token": "..."}, "ttl": "300s"}' />
              <p class="text-xs text-[#909399]">云厂商凭证等 dns01_challenge 子字段（provider 由上方下拉决定，此处不需写）；其余字段（external_account/http_client 等）用「源码」tab</p>
            </div>
          </div>
        </TabsContent>
        <TabsContent value="source">
          <ResourceSourceTab ref="srcTab" :initial="sourceJson" />
        </TabsContent>
      </TabsRoot>
      <div class="mt-5 flex justify-end gap-2">
        <button class="btn btn-ghost btn-sm" @click="dialogVisible = false">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="saving" @click="saveProvider">保存</button>
      </div>
    </DialogWrapper>
  </div>
</template>
