<template>
  <DialogWrapper
    :model-value="modelValue"
    :title="isEdit ? $t('editCertProvider') : $t('createCertProvider')"
    box-class="max-w-[680px]"
    :body-class="activeTab === 'source' ? 'overflow-hidden' : 'overflow-y-auto'"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <TabsRoot v-model="activeTab">
      <TabsList class="self-start shrink-0">
        <TabsTrigger value="form">{{ $t('form') }}</TabsTrigger>
        <TabsTrigger value="source">{{ $t('source') }}</TabsTrigger>
      </TabsList>

      <TabsContent value="form">
        <div ref="formContentRef">
          <div class="mb-4">
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              @click="fillFromJson"
            >
              <ClipboardPasteIcon class="h-3.5 w-3.5" />
              从 JSON 填充（粘贴解析）
            </button>
            <span class="ml-2 text-xs text-base-content/50">
              粘贴完整 Certificate Provider JSON，自动解析并填充下方字段
            </span>
          </div>

          <div
            class="grid gap-x-4 gap-y-4 pt-2 text-sm"
            style="grid-template-columns: 140px minmax(0, 1fr)"
          >
            <label class="text-base-content/70 pt-2">type <span class="text-destructive">*</span></label>
            <SelectField
              v-model="providerForm.type"
              :options="['acme']"
            />

            <label class="text-base-content/70 pt-2">
              {{ $t('certProviderTag') }} <span class="text-destructive">*</span>
            </label>
            <input
              v-model="providerForm.tag"
              type="text"
              class="input input-bordered input-sm w-full"
              :placeholder="$t('certProviderTagPlaceholder')"
            />

            <label class="text-base-content/70 pt-2">
              {{ $t('certDomain') }} <span class="text-destructive">*</span>
            </label>
            <div class="flex flex-col gap-1.5">
              <div class="flex flex-wrap gap-1.5">
                <span
                  v-for="d in providerForm.domain"
                  :key="d"
                  class="badge badge-primary badge-outline gap-1"
                >
                  {{ d }}
                  <button
                    type="button"
                    class="cursor-pointer text-xs opacity-70 hover:opacity-100"
                    @click="removeDomain(d)"
                  >
                    ×
                  </button>
                </span>
              </div>
              <input
                v-model="domainInput"
                type="text"
                class="input input-bordered input-sm w-full"
                :placeholder="$t('certDomainPlaceholder')"
                @keydown.enter.prevent="addDomain"
                @blur="addDomain"
              />
            </div>

            <label class="text-base-content/70 pt-2">{{ $t('certEmail') }}</label>
            <input
              v-model="providerForm.email"
              type="email"
              class="input input-bordered input-sm w-full"
              :placeholder="$t('certEmailPlaceholder')"
            />

            <label class="text-base-content/70 pt-2">{{ $t('certDataDir') }}</label>
            <input
              v-model="providerForm.data_directory"
              type="text"
              class="input input-bordered input-sm w-full"
              :placeholder="$t('certDataDirPlaceholder')"
            />

            <label class="text-base-content/70 pt-2">{{ $t('certKeyType') }}</label>
            <SelectField
              v-model="providerForm.key_type"
              :options="['', 'ed25519', 'p256', 'p384']"
              :placeholder="$t('certKeyTypePlaceholder')"
            />

            <div
              class="flex items-center gap-2 text-[13px] font-semibold text-base-content/90"
              style="grid-column: 1 / -1"
            >
              <span class="border-base-content/10 h-px flex-1 border-t" />
              {{ $t('certChallengeTitle') }}
              <span class="border-base-content/10 h-px flex-1 border-t" />
            </div>

            <label class="text-base-content/70 pt-2">{{ $t('disableHttpChallenge') }}</label>
            <Switch
              v-model="providerForm.disable_http_challenge"
              class="self-center"
            />

            <label class="text-base-content/70 pt-2">{{ $t('disableTlsAlpnChallenge') }}</label>
            <Switch
              v-model="providerForm.disable_tls_alpn_challenge"
              class="self-center"
            />

            <div
              class="flex items-center gap-2 text-[13px] font-semibold text-base-content/90"
              style="grid-column: 1 / -1"
            >
              <span class="border-base-content/10 h-px flex-1 border-t" />
              {{ $t('certDns01Title') }}
              <span class="border-base-content/10 h-px flex-1 border-t" />
            </div>

            <label class="text-base-content/70 pt-2">{{ $t('dnsProvider') }}</label>
            <SelectField
              v-model="providerForm.provider"
              :options="providerOptions"
            />

            <template v-if="providerForm.provider === 'cloudflare'">
              <label class="text-base-content/70 pt-2">API Token <span class="text-destructive">*</span></label>
              <input
                v-model="providerForm.cf_api_token"
                type="password"
                class="input input-bordered input-sm w-full font-mono"
                :placeholder="$t('cfApiTokenPlaceholder')"
              />

              <label class="text-base-content/70 pt-2">Zone Token</label>
              <input
                v-model="providerForm.cf_zone_token"
                type="password"
                class="input input-bordered input-sm w-full font-mono"
                :placeholder="$t('cfZoneTokenPlaceholder')"
              />
            </template>

            <template v-else-if="providerForm.provider === 'alidns'">
              <label class="text-base-content/70 pt-2">AccessKey ID <span class="text-destructive">*</span></label>
              <input
                v-model="providerForm.ali_access_key_id"
                type="text"
                class="input input-bordered input-sm w-full font-mono"
                :placeholder="$t('aliAccessKeyIdPlaceholder')"
              />

              <label class="text-base-content/70 pt-2">AccessKey Secret <span class="text-destructive">*</span></label>
              <input
                v-model="providerForm.ali_access_key_secret"
                type="password"
                class="input input-bordered input-sm w-full font-mono"
                :placeholder="$t('aliAccessKeySecretPlaceholder')"
              />

              <label class="text-base-content/70 pt-2">Region ID</label>
              <input
                v-model="providerForm.ali_region_id"
                type="text"
                class="input input-bordered input-sm w-full font-mono"
                :placeholder="$t('aliRegionIdPlaceholder')"
              />

              <label class="text-base-content/70 pt-2">Security Token</label>
              <input
                v-model="providerForm.ali_security_token"
                type="password"
                class="input input-bordered input-sm w-full font-mono"
                :placeholder="$t('aliSecurityTokenPlaceholder')"
              />
            </template>

            <template v-else-if="providerForm.provider === 'acmedns'">
              <label class="text-base-content/70 pt-2">Username <span class="text-destructive">*</span></label>
              <input
                v-model="providerForm.acme_username"
                type="text"
                class="input input-bordered input-sm w-full font-mono"
                :placeholder="$t('acmeUsernamePlaceholder')"
              />

              <label class="text-base-content/70 pt-2">Password <span class="text-destructive">*</span></label>
              <input
                v-model="providerForm.acme_password"
                type="password"
                class="input input-bordered input-sm w-full font-mono"
                :placeholder="$t('acmePasswordPlaceholder')"
              />

              <label class="text-base-content/70 pt-2">Subdomain <span class="text-destructive">*</span></label>
              <input
                v-model="providerForm.acme_subdomain"
                type="text"
                class="input input-bordered input-sm w-full font-mono"
                :placeholder="$t('acmeSubdomainPlaceholder')"
              />

              <label class="text-base-content/70 pt-2">Server URL</label>
              <input
                v-model="providerForm.acme_server_url"
                type="text"
                class="input input-bordered input-sm w-full font-mono"
                :placeholder="$t('acmeServerUrlPlaceholder')"
              />
            </template>

            <template v-else-if="providerForm.provider === 'custom'">
              <label class="text-base-content/70 pt-2">{{ $t('credentialsJson') }} <span class="text-destructive">*</span></label>
              <div class="flex flex-col gap-1">
                <textarea
                  v-model="providerForm.dns01_challenge_json"
                  rows="4"
                  class="textarea textarea-bordered font-mono text-xs w-full"
                  :placeholder="$t('customDnsCredentialsPlaceholder')"
                />
                <p class="text-base-content/50 text-xs">
                  {{ $t('customDnsTip') }}
                </p>
              </div>
            </template>
          </div>

          <p
            v-if="validateMsg"
            class="mt-3 text-[0.8rem] font-medium text-destructive"
          >
            {{ validateMsg }}
          </p>
        </div>
      </TabsContent>

      <TabsContent value="source">
        <ResourceSourceTab
          ref="srcTab"
          :initial="sourceJson"
          :height="sourceEditorHeight"
        />
      </TabsContent>
    </TabsRoot>

    <template #footer>
      <div class="flex items-center justify-between w-full">
        <div>
          <button
            v-if="isEdit && providerData?.id"
            type="button"
            class="btn btn-ghost btn-sm text-error hover:bg-error/10"
            :disabled="saving"
            @click="handleDelete"
          >
            {{ $t('delete') || '删除' }}
          </button>
        </div>
        <div class="flex gap-2">
          <button
            type="button"
            class="btn btn-ghost btn-sm"
            @click="emit('update:modelValue', false)"
          >
            {{ $t('cancel') }}
          </button>
          <button
            type="button"
            class="btn btn-primary btn-sm"
            :disabled="saving"
            @click="save"
          >
            {{ $t('save') }}
          </button>
        </div>
      </div>
    </template>
  </DialogWrapper>

  <!-- 从 JSON 填充弹窗 -->
  <DialogWrapper
    v-model="jsonDialog"
    title="从 JSON 填充"
    box-class="max-w-[560px]"
  >
    <textarea
      v-model="jsonText"
      rows="10"
      class="textarea textarea-bordered w-full font-mono text-xs"
      placeholder='{"type":"acme","tag":"letsencrypt","domain":["example.com"]}'
    />
    <p class="mt-1 text-xs text-base-content/50">
      解析成功后填充表单字段（请检查解析结果）
    </p>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <button
          type="button"
          class="btn btn-ghost btn-sm"
          @click="jsonDialog = false"
        >
          取消
        </button>
        <button
          type="button"
          class="btn btn-primary btn-sm"
          @click="doFillFromJson"
        >
          解析并填充
        </button>
      </div>
    </template>
  </DialogWrapper>
</template>

<script setup lang="ts">
import { api } from '@/api'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import ResourceSourceTab from '@/components/ResourceSourceTab.vue'
import { SelectField } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { TabsContent, TabsList, TabsRoot, TabsTrigger } from '@/components/ui/tabs'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { showToast } from '@/helper/toast'
import { useStatusStore } from '@/store/status'
import { ClipboardPasteIcon } from '@lucide/vue'
import { useResizeObserver } from '@vueuse/core'
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  modelValue: boolean
  isEdit: boolean
  providerData?: { id: string; provider: Record<string, any> } | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'saved'): void
}>()

const statusStore = useStatusStore()
const activeTab = ref('form')
const srcTab = ref<InstanceType<typeof ResourceSourceTab>>()
const sourceJson = ref('{}')
const saving = ref(false)
const domainInput = ref('')
const validateMsg = ref('')

const formContentRef = ref<HTMLElement>()
const sharedTabHeight = ref<number>(460)

const jsonDialog = ref(false)
const jsonText = ref('')

const providerOptions = ['cloudflare', 'alidns', 'acmedns', 'custom']

useResizeObserver(formContentRef, (entries) => {
  const entry = entries[0]
  if (entry && activeTab.value === 'form' && entry.contentRect.height > 60) {
    sharedTabHeight.value = Math.max(300, Math.round(entry.contentRect.height))
  }
})

const sourceEditorHeight = computed(() => {
  if (typeof window === 'undefined') return 440
  const maxAvailable = Math.max(300, Math.floor(window.innerHeight * 0.85 - 170))
  return Math.min(sharedTabHeight.value, maxAvailable)
})

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
  // Cloudflare
  cf_api_token: '',
  cf_zone_token: '',
  // AliDNS
  ali_access_key_id: '',
  ali_access_key_secret: '',
  ali_region_id: '',
  ali_security_token: '',
  // ACME-DNS
  acme_username: '',
  acme_password: '',
  acme_subdomain: '',
  acme_server_url: '',
  // Custom JSON
  dns01_challenge_json: '',
})

const resetForm = () => {
  validateMsg.value = ''
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
  providerForm.cf_api_token = ''
  providerForm.cf_zone_token = ''
  providerForm.ali_access_key_id = ''
  providerForm.ali_access_key_secret = ''
  providerForm.ali_region_id = ''
  providerForm.ali_security_token = ''
  providerForm.acme_username = ''
  providerForm.acme_password = ''
  providerForm.acme_subdomain = ''
  providerForm.acme_server_url = ''
  providerForm.dns01_challenge_json = ''
  domainInput.value = ''
}

const toList = (v: unknown): string[] => {
  if (v == null) return []
  return Array.isArray(v) ? v.map(String) : [String(v)]
}

function isValidDomain(domain: string): boolean {
  const d = domain.trim()
  if (!d) return false
  const toCheck = d.startsWith('*.') ? d.slice(2) : d
  if (
    !toCheck ||
    toCheck.includes('*') ||
    toCheck.includes('/') ||
    toCheck.includes(':') ||
    toCheck.includes(' ') ||
    toCheck.startsWith('.') ||
    toCheck.endsWith('.')
  ) {
    return false
  }
  const labels = toCheck.split('.')
  if (labels.length < 2) return false
  return labels.every((label) => /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$/.test(label))
}

function fillProviderForm(p: Record<string, any>) {
  providerForm.type = String(p.type || 'acme')
  providerForm.tag = typeof p.tag === 'string' ? p.tag : ''
  providerForm.domain = toList(p.domain)
  providerForm.data_directory = typeof p.data_directory === 'string' ? p.data_directory : ''
  providerForm.default_server_name =
    typeof p.default_server_name === 'string' ? p.default_server_name : ''
  providerForm.email = typeof p.email === 'string' ? p.email : ''
  providerForm.account_key = typeof p.account_key === 'string' ? p.account_key : ''
  providerForm.disable_http_challenge = p.disable_http_challenge === true
  providerForm.disable_tls_alpn_challenge = p.disable_tls_alpn_challenge === true
  providerForm.alternative_http_port = Number(p.alternative_http_port || 0)
  providerForm.alternative_tls_port = Number(p.alternative_tls_port || 0)
  providerForm.key_type = typeof p.key_type === 'string' ? p.key_type : ''

  const d01 = p.dns01_challenge
  if (d01 && typeof d01 === 'object') {
    const prov = typeof d01.provider === 'string' ? d01.provider : 'cloudflare'
    providerForm.provider = ['cloudflare', 'alidns', 'acmedns'].includes(prov) ? prov : 'custom'

    if (prov === 'cloudflare') {
      providerForm.cf_api_token = typeof d01.api_token === 'string' ? d01.api_token : ''
      providerForm.cf_zone_token = typeof d01.zone_token === 'string' ? d01.zone_token : ''
    } else if (prov === 'alidns') {
      providerForm.ali_access_key_id = typeof d01.access_key_id === 'string' ? d01.access_key_id : ''
      providerForm.ali_access_key_secret = typeof d01.access_key_secret === 'string' ? d01.access_key_secret : ''
      providerForm.ali_region_id = typeof d01.region_id === 'string' ? d01.region_id : ''
      providerForm.ali_security_token = typeof d01.security_token === 'string' ? d01.security_token : ''
    } else if (prov === 'acmedns') {
      providerForm.acme_username = typeof d01.username === 'string' ? d01.username : ''
      providerForm.acme_password = typeof d01.password === 'string' ? d01.password : ''
      providerForm.acme_subdomain = typeof d01.subdomain === 'string' ? d01.subdomain : ''
      providerForm.acme_server_url = typeof d01.server_url === 'string' ? d01.server_url : ''
    } else {
      providerForm.dns01_challenge_json = JSON.stringify(d01, null, 2)
    }
  } else {
    providerForm.provider = 'cloudflare'
  }
}

watch(
  () => [props.modelValue, props.isEdit, props.providerData],
  ([visible, isEdit, data]) => {
    if (!visible) return
    activeTab.value = 'form'
    resetForm()
    if (isEdit && data && typeof data === 'object' && 'provider' in data) {
      const p = (data as { id: string; provider: Record<string, any> }).provider
      sourceJson.value = JSON.stringify(p, null, 2)
      fillProviderForm(p)
    } else {
      sourceJson.value = '{}'
    }
  },
  { immediate: true },
)

watch(activeTab, (tab) => {
  if (tab === 'source' && !srcTab.value?.isDirty()) {
    try {
      sourceJson.value = JSON.stringify(buildProvider(), null, 2)
    } catch {
      // ignore
    }
  }
})

const fillFromJson = () => {
  jsonText.value = ''
  jsonDialog.value = true
}

const doFillFromJson = () => {
  try {
    const raw = jsonText.value.trim()
    if (!raw) return
    let parsed = JSON.parse(raw)
    if (parsed && typeof parsed === 'object' && 'provider' in parsed) {
      parsed = parsed.provider
    }
    if (!parsed || typeof parsed !== 'object') {
      throw new Error('解析内容必须为 JSON 对象')
    }
    fillProviderForm(parsed)
    jsonDialog.value = false
    showToast('已从 JSON 填充，请检查表单', 'success')
  } catch (e) {
    showToast((e as Error).message || '解析失败', 'error')
  }
}

const addDomain = () => {
  if (!domainInput.value.trim()) return
  const parts = domainInput.value
    .split(/[,，\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  for (const p of parts) {
    if (!isValidDomain(p)) {
      showToast(t('invalidDomain', { domain: p }), 'error')
      continue
    }
    if (!providerForm.domain.includes(p)) {
      providerForm.domain.push(p)
    }
  }
  domainInput.value = ''
}

const removeDomain = (d: string) => {
  const idx = providerForm.domain.indexOf(d)
  if (idx >= 0) providerForm.domain.splice(idx, 1)
}

function buildProvider(): Record<string, any> {
  if (domainInput.value.trim()) {
    addDomain()
  }

  const tag = providerForm.tag.trim()
  if (!tag) {
    throw new Error(t('tagRequired'))
  }

  if (!providerForm.domain.length) {
    throw new Error(t('domainRequired'))
  }

  for (const d of providerForm.domain) {
    if (!isValidDomain(d)) {
      throw new Error(t('invalidDomain', { domain: d }))
    }
  }

  let d01: Record<string, any> | undefined

  if (providerForm.provider === 'cloudflare') {
    const apiToken = providerForm.cf_api_token.trim()
    if (!apiToken) {
      throw new Error(t('cfApiTokenRequired'))
    }
    d01 = {
      provider: 'cloudflare',
      api_token: apiToken,
    }
    if (providerForm.cf_zone_token.trim()) {
      d01.zone_token = providerForm.cf_zone_token.trim()
    }
  } else if (providerForm.provider === 'alidns') {
    const keyId = providerForm.ali_access_key_id.trim()
    const keySecret = providerForm.ali_access_key_secret.trim()
    if (!keyId || !keySecret) {
      throw new Error(t('aliKeyRequired'))
    }
    d01 = {
      provider: 'alidns',
      access_key_id: keyId,
      access_key_secret: keySecret,
    }
    if (providerForm.ali_region_id.trim()) d01.region_id = providerForm.ali_region_id.trim()
    if (providerForm.ali_security_token.trim()) d01.security_token = providerForm.ali_security_token.trim()
  } else if (providerForm.provider === 'acmedns') {
    const u = providerForm.acme_username.trim()
    const p = providerForm.acme_password.trim()
    const sub = providerForm.acme_subdomain.trim()
    if (!u || !p || !sub) {
      throw new Error(t('acmeFieldsRequired'))
    }
    d01 = {
      provider: 'acmedns',
      username: u,
      password: p,
      subdomain: sub,
    }
    if (providerForm.acme_server_url.trim()) d01.server_url = providerForm.acme_server_url.trim()
  } else if (providerForm.provider === 'custom') {
    const rawJson = providerForm.dns01_challenge_json.trim()
    if (!rawJson) {
      throw new Error(t('dns01CredentialsRequired'))
    }
    try {
      const parsed = JSON.parse(rawJson)
      if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed) || Object.keys(parsed).length === 0) {
        throw new Error(t('dns01CredentialsRequired'))
      }
      d01 = { ...parsed }
    } catch (e) {
      if ((e as Error).message === t('dns01CredentialsRequired')) {
        throw e
      }
      throw new Error(t('dns01CredentialsInvalidJson', { error: (e as Error).message }))
    }
  }

  const p: Record<string, any> = {
    type: providerForm.type || 'acme',
    tag,
    domain: [...providerForm.domain],
  }

  if (d01) {
    p.dns01_challenge = d01
  }

  if (providerForm.data_directory.trim()) p.data_directory = providerForm.data_directory.trim()
  if (providerForm.default_server_name.trim())
    p.default_server_name = providerForm.default_server_name.trim()
  if (providerForm.email.trim()) p.email = providerForm.email.trim()
  if (providerForm.account_key.trim()) p.account_key = providerForm.account_key.trim()
  if (providerForm.disable_http_challenge) p.disable_http_challenge = true
  if (providerForm.disable_tls_alpn_challenge) p.disable_tls_alpn_challenge = true
  if (providerForm.alternative_http_port)
    p.alternative_http_port = Number(providerForm.alternative_http_port)
  if (providerForm.alternative_tls_port)
    p.alternative_tls_port = Number(providerForm.alternative_tls_port)
  if (providerForm.key_type) p.key_type = providerForm.key_type

  return p
}

const save = async () => {
  validateMsg.value = ''
  saving.value = true
  statusStore.markReloading(4000)
  try {
    const payload =
      activeTab.value === 'source' && srcTab.value?.isDirty()
        ? JSON.parse(srcTab.value.getText())
        : buildProvider()

    console.log('[CertProviderDialog] Sending payload:', payload)

    if (props.isEdit && props.providerData?.id) {
      await api.updateCertProvider(props.providerData.id, payload)
    } else {
      await api.createCertProvider(payload)
    }
    showToast(t('saveSuccess'), 'success')
    emit('update:modelValue', false)
    emit('saved')
    await statusStore.refresh()
  } catch (e) {
    statusStore.clearReloading()
    console.error('[CertProviderDialog] Save error:', e)
    validateMsg.value = (e as Error).message || t('saveFailed')
    showToast((e as Error).message || t('saveFailed'), 'error')
  } finally {
    saving.value = false
  }
}

const handleDelete = async () => {
  if (!props.providerData?.id) return
  const { confirmed } = await showConfirmDialog({
    title: t('delete') || '删除证书提供者',
    message: '确定删除此证书提供者？该操作不可恢复。',
    confirmText: t('delete') || '删除',
    confirmButtonClass: 'btn-error',
  })
  if (!confirmed) return
  saving.value = true
  statusStore.markReloading(4000)
  try {
    await api.deleteCertProvider(props.providerData.id)
    showToast(t('deleteSuccess') || '删除成功', 'success')
    emit('update:modelValue', false)
    emit('saved')
    await statusStore.refresh()
  } catch (e) {
    statusStore.clearReloading()
    showToast((e as Error).message || t('deleteFailed') || '删除失败', 'error')
  } finally {
    saving.value = false
  }
}
</script>
