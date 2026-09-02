<template>
  <DialogWrapper
    v-model="isOpen"
    :title="isEdit ? t('profiles.editProvider', '编辑订阅源') : t('profiles.addProvider', '添加订阅源')"
    size="medium"
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
      <div class="modal-action mt-2">
        <button
          class="btn btn-sm btn-ghost"
          @click="isOpen = false"
        >
          {{ t('common.cancel', '取消') }}
        </button>
        <button
          class="btn btn-sm btn-primary"
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
</script>
