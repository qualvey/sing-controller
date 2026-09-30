<template>
  <section class="mt-4 rounded-lg border border-base-300 p-3">
    <h3 class="mb-1 font-medium">{{ $t('nodeHealthTitle') }}</h3>
    <p class="mb-3 text-xs text-base-content/60">{{ $t('nodeHealthDescription') }}</p>

    <div class="settings-grid">
      <div class="setting-item">
        <span class="setting-item-label">{{ $t('nodeHealthEnabled') }}</span>
        <input v-model="form.enabled" class="toggle" type="checkbox" />
      </div>
      <label class="setting-item">
        <span class="setting-item-label">{{ $t('nodeHealthInterval') }}</span>
        <input v-model.trim="form.interval" class="input input-sm w-28" placeholder="5m" />
      </label>
      <label class="setting-item">
        <span class="setting-item-label">{{ $t('nodeHealthTestUrl') }}</span>
        <input v-model.trim="form.test_url" class="input input-sm min-w-40 flex-1" type="url" />
      </label>
      <label class="setting-item">
        <span class="setting-item-label">{{ $t('nodeHealthTimeout') }}</span>
        <span class="flex items-center gap-2">
          <input v-model.number="form.timeout_ms" class="input input-sm w-24" type="number" min="500" max="60000" />
          ms
        </span>
      </label>
      <label class="setting-item">
        <span class="setting-item-label">{{ $t('nodeHealthFailureThreshold') }}</span>
        <input v-model.number="form.failure_threshold" class="input input-sm w-24" type="number" min="1" max="10" />
      </label>
      <label class="setting-item">
        <span class="setting-item-label">{{ $t('nodeHealthConcurrency') }}</span>
        <input v-model.number="form.concurrency" class="input input-sm w-24" type="number" min="1" max="32" />
      </label>
      <label class="setting-item">
        <span class="setting-item-label">{{ $t('nodeHealthWebhookUrl') }}</span>
        <input v-model.trim="form.webhook_url" class="input input-sm min-w-40 flex-1" type="url" />
      </label>
      <label class="setting-item">
        <span class="setting-item-label">{{ $t('nodeHealthWebhookToken') }}</span>
        <input
          v-model="form.webhook_token"
          class="input input-sm min-w-40 flex-1"
          type="password"
          :placeholder="tokenConfigured ? $t('nodeHealthTokenPlaceholder') : ''"
          autocomplete="new-password"
        />
      </label>
      <label v-if="tokenConfigured" class="setting-item">
        <span class="setting-item-label">{{ $t('nodeHealthClearToken') }}</span>
        <input v-model="clearToken" class="toggle" type="checkbox" />
      </label>
      <label class="setting-item">
        <span class="setting-item-label">{{ $t('nodeHealthFeishuSignSecret') }}</span>
        <input
          v-model="form.webhook_sign_secret"
          class="input input-sm min-w-40 flex-1"
          type="password"
          :placeholder="signSecretConfigured ? $t('nodeHealthTokenPlaceholder') : ''"
          autocomplete="new-password"
        />
      </label>
      <label v-if="signSecretConfigured" class="setting-item">
        <span class="setting-item-label">{{ $t('nodeHealthClearSignSecret') }}</span>
        <input v-model="clearSignSecret" class="toggle" type="checkbox" />
      </label>
    </div>

    <div class="mt-3 flex items-center gap-3">
      <button class="btn btn-primary btn-sm" :disabled="loading || saving" @click="save">
        {{ saving ? $t('nodeHealthSaving') : $t('save') }}
      </button>
      <button class="btn btn-outline btn-sm" :disabled="loading || saving || nodeHealthScanning" @click="scanNow">
        {{ nodeHealthScanning ? $t('nodeHealthScanning') : $t('nodeHealthScanNow') }}
      </button>
      <span v-if="loadError" class="text-xs text-error">{{ loadError }}</span>
    </div>
  </section>
</template>

<script setup lang="ts">
import { api, type NodeHealthConfig } from '@/api/controller'
import { showToast } from '@/helper/toast'
import { nodeHealthScanning, refreshNodeHealth } from '@/store/nodeHealth'
import { onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loading = ref(true)
const saving = ref(false)
const loadError = ref('')
const tokenConfigured = ref(false)
const signSecretConfigured = ref(false)
const clearToken = ref(false)
const clearSignSecret = ref(false)
let scanStateTimer: number | undefined
const form = reactive<NodeHealthConfig>({
  enabled: false,
  interval: '5m',
  test_url: 'https://www.gstatic.com/generate_204',
  timeout_ms: 5000,
  failure_threshold: 2,
  concurrency: 5,
  webhook_url: '',
  webhook_token: '',
  webhook_sign_secret: '',
})

onMounted(async () => {
  void refreshNodeHealth()
  try {
    const result = await api.nodeHealthConfig()
    Object.assign(form, result.config, {
      webhook_token: result.config.webhook_token || '',
      webhook_sign_secret: result.config.webhook_sign_secret || '',
    })
    tokenConfigured.value = result.webhook_token_configured
    signSecretConfigured.value = result.webhook_sign_secret_configured
  } catch (error) {
    loadError.value = (error as Error).message || t('nodeHealthLoadFailed')
  } finally {
    loading.value = false
  }
})

onUnmounted(() => {
  if (scanStateTimer !== undefined) window.clearInterval(scanStateTimer)
})

const scanNow = async () => {
  try {
    await api.triggerNodeHealthScan()
    nodeHealthScanning.value = true
    showToast(t('nodeHealthScanStarted'), 'info')
    void refreshNodeHealth()
    if (scanStateTimer !== undefined) window.clearInterval(scanStateTimer)
    scanStateTimer = window.setInterval(() => {
      void refreshNodeHealth().then(() => {
        if (!nodeHealthScanning.value && scanStateTimer !== undefined) {
          window.clearInterval(scanStateTimer)
          scanStateTimer = undefined
        }
      })
    }, 1500)
  } catch (error) {
    showToast((error as Error).message || t('nodeHealthScanFailed'), 'error')
  }
}

const save = async () => {
  saving.value = true
  try {
    await api.updateNodeHealthConfig({
      ...form,
      clear_webhook_token: clearToken.value,
      clear_webhook_sign_secret: clearSignSecret.value,
    })
    tokenConfigured.value = clearToken.value ? false : !!form.webhook_token || tokenConfigured.value
    signSecretConfigured.value = clearSignSecret.value
      ? false
      : !!form.webhook_sign_secret || signSecretConfigured.value
    form.webhook_token = ''
    form.webhook_sign_secret = ''
    clearToken.value = false
    clearSignSecret.value = false
    showToast(t('nodeHealthSaved'), 'success')
  } catch (error) {
    showToast((error as Error).message || t('nodeHealthSaveFailed'), 'error')
  } finally {
    saving.value = false
  }
}
</script>
