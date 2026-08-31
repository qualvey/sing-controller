<template>
  <DialogWrapper
    :model-value="modelValue"
    :title="isEdit ? $t('editCertProvider') : $t('createCertProvider')"
    box-class="max-w-[680px]"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <TabsRoot v-model="activeTab">
      <TabsList>
        <TabsTrigger value="form">{{ $t('form') }}</TabsTrigger>
        <TabsTrigger value="source">{{ $t('source') }}</TabsTrigger>
      </TabsList>

      <TabsContent value="form">
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

          <label class="text-base-content/70 pt-2">{{ $t('dns01Provider') }}</label>
          <div class="flex flex-col gap-1">
            <SelectField
              v-model="providerForm.provider"
              :options="['cloudflare', 'alidns', 'acmedns', 'custom']"
              :placeholder="$t('dns01ProviderPlaceholder')"
            />
            <p class="text-base-content/50 text-xs">
              {{ $t('dns01ProviderTip') }}
            </p>
          </div>

          <!-- Cloudflare 表单 -->
          <template v-if="providerForm.provider === 'cloudflare'">
            <label class="text-base-content/70 pt-2">
              {{ $t('cfApiToken') }} <span class="text-destructive">*</span>
            </label>
            <input
              v-model="providerForm.cf_api_token"
              type="password"
              class="input input-bordered input-sm w-full font-mono"
              :placeholder="$t('cfApiTokenPlaceholder')"
            />

            <label class="text-base-content/70 pt-2">{{ $t('cfZoneToken') }}</label>
            <input
              v-model="providerForm.cf_zone_token"
              type="password"
              class="input input-bordered input-sm w-full font-mono"
              :placeholder="$t('cfZoneTokenPlaceholder')"
            />
          </template>

          <!-- Alibaba Cloud DNS 表单 -->
          <template v-else-if="providerForm.provider === 'alidns'">
            <label class="text-base-content/70 pt-2">
              {{ $t('aliAccessKeyId') }} <span class="text-destructive">*</span>
            </label>
            <input
              v-model="providerForm.ali_access_key_id"
              type="text"
              class="input input-bordered input-sm w-full font-mono"
              placeholder="AccessKey ID"
            />

            <label class="text-base-content/70 pt-2">
              {{ $t('aliAccessKeySecret') }} <span class="text-destructive">*</span>
            </label>
            <input
              v-model="providerForm.ali_access_key_secret"
              type="password"
              class="input input-bordered input-sm w-full font-mono"
              placeholder="AccessKey Secret"
            />

            <label class="text-base-content/70 pt-2">{{ $t('aliRegionId') }}</label>
            <input
              v-model="providerForm.ali_region_id"
              type="text"
              class="input input-bordered input-sm w-full"
              :placeholder="$t('aliRegionIdPlaceholder')"
            />

            <label class="text-base-content/70 pt-2">{{ $t('aliSecurityToken') }}</label>
            <input
              v-model="providerForm.ali_security_token"
              type="password"
              class="input input-bordered input-sm w-full font-mono"
              :placeholder="$t('aliSecurityTokenPlaceholder')"
            />
          </template>

          <!-- ACME-DNS 表单 -->
          <template v-else-if="providerForm.provider === 'acmedns'">
            <label class="text-base-content/70 pt-2">
              {{ $t('acmeSubdomain') }} <span class="text-destructive">*</span>
            </label>
            <input
              v-model="providerForm.acme_subdomain"
              type="text"
              class="input input-bordered input-sm w-full font-mono"
              placeholder="Subdomain"
            />

            <label class="text-base-content/70 pt-2">
              {{ $t('acmeUsername') }} <span class="text-destructive">*</span>
            </label>
            <input
              v-model="providerForm.acme_username"
              type="text"
              class="input input-bordered input-sm w-full font-mono"
              placeholder="Username"
            />

            <label class="text-base-content/70 pt-2">
              {{ $t('acmePassword') }} <span class="text-destructive">*</span>
            </label>
            <input
              v-model="providerForm.acme_password"
              type="password"
              class="input input-bordered input-sm w-full font-mono"
              placeholder="Password"
            />

            <label class="text-base-content/70 pt-2">{{ $t('acmeServerUrl') }}</label>
            <input
              v-model="providerForm.acme_server_url"
              type="text"
              class="input input-bordered input-sm w-full"
              :placeholder="$t('acmeServerUrlPlaceholder')"
            />
          </template>

          <!-- 自定义 / JSON 兜底 -->
          <template v-else>
            <label class="text-base-content/70 pt-2">
              {{ $t('dns01Credentials') }} <span class="text-destructive">*</span>
            </label>
            <div class="flex flex-col gap-1">
              <textarea
                v-model="providerForm.dns01_challenge_json"
                rows="4"
                class="textarea textarea-bordered font-mono text-xs w-full"
                placeholder='{"provider": "cloudflare", "api_token": "your_token"}'
              />
              <p class="text-base-content/50 text-xs">
                {{ $t('dns01CredentialsTip') }}
              </p>
            </div>
          </template>
        </div>
      </TabsContent>

      <TabsContent value="source">
        <ResourceSourceTab
          ref="srcTab"
          :initial="sourceJson"
        />
      </TabsContent>
    </TabsRoot>

    <div class="mt-5 flex justify-end gap-2">
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
  </DialogWrapper>
</template>

<script setup lang="ts">
import { api } from '@/api'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import ResourceSourceTab from '@/components/ResourceSourceTab.vue'
import { SelectField } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { TabsContent, TabsList, TabsRoot, TabsTrigger } from '@/components/ui/tabs'
import { showToast } from '@/helper/toast'
import { useStatusStore } from '@/store/status'
import { reactive, ref, watch } from 'vue'
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

watch(
  () => [props.modelValue, props.isEdit, props.providerData],
  ([visible, isEdit, data]) => {
    if (!visible) return
    activeTab.value = 'form'
    resetForm()
    if (isEdit && data && typeof data === 'object' && 'provider' in data) {
      const p = (data as { id: string; provider: Record<string, any> }).provider
      sourceJson.value = JSON.stringify(p, null, 2)
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
    } else {
      sourceJson.value = '{}'
    }
  },
  { immediate: true },
)

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
  saving.value = true
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
    console.error('[CertProviderDialog] Save error:', e)
    showToast((e as Error).message || t('saveFailed'), 'error')
  } finally {
    saving.value = false
  }
}
</script>
