<template>
  <DialogWrapper
    v-model="isOpen"
    :title="isEdit ? t('profiles.editProvider', '编辑订阅源') : t('profiles.addProvider', '添加订阅源')"
    box-class="max-w-[600px]"
  >
    <div class="flex flex-col gap-3 py-2 text-xs">
      <div class="flex flex-col gap-1">
        <label class="font-medium text-base-content/80">
          {{ t('profiles.providerName', '订阅名称') }}
        </label>
        <TextInput
          v-model="form.name"
          placeholder="例如：主力机场 A"
        />
      </div>
      <div class="flex flex-col gap-1">
        <label class="font-medium text-base-content/80">
          {{ t('profiles.providerTag', '标识 (Tag)') }}
          <span class="text-error">*</span>
        </label>
        <TextInput
          v-model="form.tag"
          placeholder="例如：airport-a (用于生成 outbound-airport-a.json)"
          :disabled="isEdit"
        />
      </div>
      <div class="flex flex-col gap-1">
        <label class="font-medium text-base-content/80">
          {{ t('profiles.providerUrl', '订阅 URL') }}
          <span class="text-error">*</span>
        </label>
        <TextInput
          v-model="form.url"
          placeholder="https://example.com/api/v1/client/subscribe?token=..."
        />
      </div>
      <div class="grid grid-cols-2 gap-2">
        <div class="flex flex-col gap-1">
          <label class="font-medium text-base-content/80">
            {{ t('profiles.providerFormat', '订阅格式') }}
          </label>
          <SelectInput
            v-model="form.format"
            :options="[
              { label: '自动检测 (Auto)', value: 'auto' },
              { label: 'Clash YAML (proxies 数组)', value: 'clash' },
              { label: 'sing-box JSON (outbounds 格式)', value: 'sing-box' },
              { label: 'Base64 节点 URI', value: 'base64' },
            ]"
          />
        </div>
        <div class="flex flex-col gap-1">
          <label class="font-medium text-base-content/80">
            {{ t('profiles.updateInterval', '自动更新间隔 (分钟)') }}
          </label>
          <input
            v-model.number="form.update_interval_min"
            type="number"
            min="0"
            class="input input-sm input-bordered w-full"
            placeholder="0 表示不自动更新"
          />
        </div>
      </div>
      <div class="form-control rounded-md border border-base-300 px-3 py-2">
        <label class="label cursor-pointer justify-start gap-2 p-0">
          <input
            v-model="form.use_full_config"
            @change="form.use_full_config && (form.use_provider_rules = false)"
            type="checkbox"
            class="checkbox checkbox-sm checkbox-primary"
          />
          <span class="label-text">
            {{ t('profiles.useFullConfig', '使用订阅提供的完整 sing-box 配置') }}
          </span>
        </label>
        <span class="mt-1 text-base-content/60">
          {{ t('profiles.useFullConfigTip', '启用后，该方案将仅使用此订阅配置，不合并本地配置或其他订阅源') }}
        </span>
      </div>
      <div class="form-control rounded-md border border-base-300 px-3 py-2">
        <label class="label cursor-pointer justify-start gap-2 p-0">
          <input
            v-model="form.use_provider_rules"
            @change="form.use_provider_rules && (form.use_full_config = false)"
            type="checkbox"
            class="checkbox checkbox-sm checkbox-primary"
          />
          <span class="label-text">
            {{ t('profiles.useProviderRules', '使用该订阅配置中的规则') }}
          </span>
        </label>
        <span class="mt-1 text-base-content/60">
          {{ t('profiles.useProviderRulesTip', '保存后将重新拉取，并把订阅中的 route / dns 规则合并到当前方案') }}
        </span>
      </div>
      <div class="form-control">
        <label class="label cursor-pointer justify-start gap-2">
          <input
            type="checkbox"
            v-model="form.enabled"
            class="checkbox checkbox-sm checkbox-primary"
          />
          <span class="label-text">
            {{ t('profiles.enableProvider', '启用该订阅源') }}
          </span>
        </label>
      </div>
    </div>

    <template #footer>
      <div class="flex items-center justify-between w-full">
        <div>
          <button
            v-if="isEdit && form.id"
            type="button"
            class="btn btn-ghost btn-sm text-error hover:bg-error/10"
            :disabled="isSaving"
            @click="handleDelete"
          >
            {{ t('common.delete', '删除') }}
          </button>
        </div>
        <div class="flex gap-2">
          <button
            type="button"
            class="btn btn-ghost btn-sm"
            @click="isOpen = false"
          >
            {{ t('common.cancel', '取消') }}
          </button>
          <button
            type="button"
            class="btn btn-primary btn-sm"
            :disabled="!form.tag || !form.url || isSaving"
            @click="handleSave"
          >
            <span
              v-if="isSaving"
              class="loading loading-spinner loading-xs"
            />
            {{ t('profiles.saveAndPull', '保存并拉取') }}
          </button>
        </div>
      </div>
    </template>
  </DialogWrapper>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import TextInput from '@/components/common/TextInput.vue'
import SelectInput from '@/components/common/SelectInput.vue'
import {
  controllerApi,
  type NodeProvider,
} from '@/api/controller'
import { showNotification } from '@/helper/notification'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { deleteProvider } from '@/composables/profiles'

const props = defineProps<{
  provider?: NodeProvider | null
}>()

const emit = defineEmits<{
  (e: 'saved'): void
}>()

const isOpen = defineModel<boolean>({ default: false })
const { t } = useI18n()

const isSaving = ref(false)
const isEdit = computed(() => !!props.provider?.id)

const form = reactive<Partial<NodeProvider>>({
  id: '',
  tag: '',
  name: '',
  type: 'remote',
  url: '',
  format: 'auto',
  update_interval_min: 720,
  enabled: true,
  use_full_config: false,
  use_provider_rules: false,
})

const resetForm = () => {
  if (props.provider) {
    Object.assign(form, {
      id: props.provider.id,
      tag: props.provider.tag,
      name: props.provider.name,
      type: props.provider.type || 'remote',
      url: props.provider.url,
      format: props.provider.format || 'auto',
      update_interval_min: props.provider.update_interval_min ?? 720,
      enabled: props.provider.enabled ?? true,
      use_full_config: props.provider.use_full_config ?? false,
      use_provider_rules: props.provider.use_provider_rules ?? false,
    })
  } else {
    Object.assign(form, {
      id: '',
      tag: '',
      name: '',
      type: 'remote',
      url: '',
      format: 'auto',
      update_interval_min: 720,
      enabled: true,
      use_full_config: false,
      use_provider_rules: false,
    })
  }
}

watch(
  () => [isOpen.value, props.provider],
  ([open]) => {
    if (open) {
      resetForm()
    }
  },
  { immediate: true },
)

const handleSave = async () => {
  if (!form.tag || !form.url) return
  isSaving.value = true
  try {
    if (form.id) {
      await controllerApi.updateProvider(form.id, form)
      showNotification({ content: t('profiles.providerUpdated', '订阅源已更新'), type: 'alert-success' })
    } else {
      await controllerApi.createProvider(form)
      showNotification({ content: t('profiles.providerCreated', '订阅源已保存并拉取'), type: 'alert-success' })
    }
    isOpen.value = false
    emit('saved')
  } catch (err: unknown) {
    const msg = (err as any)?.response?.data?.message || (err as Error)?.message || '保存订阅源失败'
    showNotification({ content: msg, type: 'alert-error' })
  } finally {
    isSaving.value = false
  }
}

const handleDelete = async () => {
  if (!form.id) return
  const { confirmed } = await showConfirmDialog({
    title: t('common.delete', '删除订阅源'),
    message: t('profiles.deleteProviderConfirm', `确定删除订阅源 "${form.name || form.tag}" 吗？此操作不可逆。`),
    confirmText: t('common.delete', '删除'),
    confirmButtonClass: 'btn-error',
  })
  if (!confirmed) return
  isSaving.value = true
  try {
    await deleteProvider(form.id)
    isOpen.value = false
    emit('saved')
  } finally {
    isSaving.value = false
  }
}
</script>
