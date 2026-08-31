<template>
  <DialogWrapper
    :model-value="modelValue"
    :title="isEdit ? '编辑 Certificate Provider' : '新建 Certificate Provider'"
    box-class="max-w-[680px]"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <TabsRoot v-model="activeTab">
      <TabsList>
        <TabsTrigger value="form">表单</TabsTrigger>
        <TabsTrigger value="source">源码</TabsTrigger>
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

          <label class="text-base-content/70 pt-2">tag <span class="text-destructive">*</span></label>
          <input
            v-model="providerForm.tag"
            type="text"
            class="input input-bordered input-sm w-full"
            placeholder="provider 唯一标识（如 letsencrypt）"
          />

          <label class="text-base-content/70 pt-2">域名 (domain)</label>
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
              placeholder="输入域名后回车或失焦添加，可多值"
              @keydown.enter.prevent="addDomain"
              @blur="addDomain"
            />
          </div>

          <label class="text-base-content/70 pt-2">email</label>
          <input
            v-model="providerForm.email"
            type="email"
            class="input input-bordered input-sm w-full"
            placeholder="ACME 注册邮箱"
          />

          <label class="text-base-content/70 pt-2">data_directory</label>
          <input
            v-model="providerForm.data_directory"
            type="text"
            class="input input-bordered input-sm w-full"
            placeholder="证书存储目录（可选）"
          />

          <label class="text-base-content/70 pt-2">key_type</label>
          <SelectField
            v-model="providerForm.key_type"
            :options="['', 'ed25519', 'p256', 'p384']"
            placeholder="私钥算法（默认 p256）"
          />

          <div
            class="flex items-center gap-2 text-[13px] font-semibold text-base-content/90"
            style="grid-column: 1 / -1"
          >
            <span class="border-base-content/10 h-px flex-1 border-t" />
            验证方式（Challenge）
            <span class="border-base-content/10 h-px flex-1 border-t" />
          </div>

          <label class="text-base-content/70 pt-2">禁用 HTTP 验证</label>
          <Switch
            v-model="providerForm.disable_http_challenge"
            class="self-center"
          />

          <label class="text-base-content/70 pt-2">禁用 TLS-ALPN 验证</label>
          <Switch
            v-model="providerForm.disable_tls_alpn_challenge"
            class="self-center"
          />

          <label class="text-base-content/70 pt-2">DNS01 Provider</label>
          <div class="flex flex-col gap-1">
            <SelectField
              v-model="providerForm.provider"
              :options="['cloudflare', 'alidns', 'acmedns']"
              placeholder="选择 DNS01 服务商（默认 cloudflare）"
            />
            <p class="text-base-content/50 text-xs">
              配置 DNS01 验证以支持通配符证书申请
            </p>
          </div>

          <label class="text-base-content/70 pt-2">DNS01 凭证 (JSON)</label>
          <div class="flex flex-col gap-1">
            <textarea
              v-model="providerForm.dns01_challenge_json"
              rows="4"
              class="textarea textarea-bordered font-mono text-xs w-full"
              placeholder='{"api_token": "your_cloudflare_api_token"}'
            />
            <p class="text-base-content/50 text-xs">
              填入 DNS01 接口对应 API Token / 凭证参数（JSON 格式）
            </p>
          </div>
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
        取消
      </button>
      <button
        type="button"
        class="btn btn-primary btn-sm"
        :disabled="saving"
        @click="save"
      >
        保存
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
  providerForm.dns01_challenge_json = ''
  domainInput.value = ''
}

const toList = (v: unknown): string[] => {
  if (v == null) return []
  return Array.isArray(v) ? v.map(String) : [String(v)]
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
        providerForm.provider = typeof d01.provider === 'string' ? d01.provider : ''
        const rest: Record<string, any> = { ...d01 }
        delete rest.provider
        providerForm.dns01_challenge_json = Object.keys(rest).length
          ? JSON.stringify(rest, null, 2)
          : ''
      } else {
        providerForm.provider = 'cloudflare'
        providerForm.dns01_challenge_json = ''
      }
    } else {
      sourceJson.value = '{}'
    }
  },
  { immediate: true },
)

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

const removeDomain = (d: string) => {
  const idx = providerForm.domain.indexOf(d)
  if (idx >= 0) providerForm.domain.splice(idx, 1)
}

function buildProvider(): Record<string, any> {
  if (!providerForm.tag.trim()) {
    throw new Error('tag 必填')
  }
  const p: Record<string, any> = { type: providerForm.type, tag: providerForm.tag.trim() }
  if (providerForm.domain.length) p.domain = [...providerForm.domain]
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

const save = async () => {
  saving.value = true
  try {
    const payload = srcTab.value?.isDirty()
      ? JSON.parse(srcTab.value.getText())
      : buildProvider()

    if (props.isEdit && props.providerData?.id) {
      await api.updateCertProvider(props.providerData.id, payload)
    } else {
      await api.createCertProvider(payload)
    }
    showToast('保存成功', 'success')
    emit('update:modelValue', false)
    emit('saved')
    await statusStore.refresh()
  } catch (e) {
    showToast((e as Error).message || '保存失败', 'error')
  } finally {
    saving.value = false
  }
}
</script>
