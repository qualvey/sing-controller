<template>
  <DialogWrapper
    v-model="isOpen"
    :title="isEdit ? t('profiles.editProfile', '编辑配置方案') : t('profiles.createProfile', '新建配置方案')"
    size="medium"
  >
    <div class="flex flex-col gap-3 py-2 text-xs">
      <div class="flex flex-col gap-1">
        <label class="font-medium text-base-content/80">
          {{ t('profiles.profileName', '方案名称') }}
          <span class="text-error">*</span>
        </label>
        <TextInput
          v-model="form.name"
          placeholder="例如：家庭主力分流 / 游戏低延迟模式"
        />
      </div>
      <div class="flex flex-col gap-1">
        <label class="font-medium text-base-content/80">
          {{ t('profiles.profileTag', '标识 (Tag)') }}
          <span class="text-error">*</span>
        </label>
        <TextInput
          v-model="form.tag"
          placeholder="例如：gaming / travel"
          :disabled="isEdit"
        />
      </div>
      <div class="flex flex-col gap-1">
        <label class="font-medium text-base-content/80">
          {{ t('profiles.profileDesc', '方案描述') }}
        </label>
        <TextInput
          v-model="form.description"
          placeholder="可选描述说明"
        />
      </div>

      <div class="flex flex-col gap-1.5">
        <label class="font-medium text-base-content/80">
          {{ t('profiles.mountedProviders', '挂载的订阅源 / 节点片段：') }}
        </label>
        <div class="flex flex-wrap gap-2 p-2 bg-base-300/30 rounded-lg border border-base-300">
          <label
            v-for="prov in providers"
            :key="prov.tag"
            class="cursor-pointer label gap-1.5 p-1 bg-base-100 rounded-md border border-base-300/60 hover:border-primary/50 text-xs"
          >
            <input
              type="checkbox"
              :value="prov.tag"
              v-model="form.provider_tags"
              class="checkbox checkbox-xs checkbox-primary"
            />
            <span>{{ prov.name }} ({{ prov.tag }})</span>
          </label>
          <span
            v-if="providers.length === 0"
            class="text-base-content/40 italic p-1"
          >
            {{ t('profiles.noAvailableProviders', '暂无可用的订阅源') }}
          </span>
        </div>
      </div>

      <div class="form-control">
        <label class="label cursor-pointer justify-start gap-2">
          <input
            type="checkbox"
            v-model="form.auto_reload"
            class="checkbox checkbox-sm checkbox-primary"
          />
          <span class="label-text">
            {{ t('profiles.autoReloadCore', '订阅更新后自动热重载内核') }}
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
          :disabled="!form.name || !form.tag || isSaving"
          @click="handleSave"
        >
          <span
            v-if="isSaving"
            class="loading loading-spinner loading-xs"
          />
          {{ t('common.save', '保存方案') }}
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
import {
  controllerApi,
  type ConfigProfile,
  type NodeProvider,
} from '@/api/controller'
import { showNotification } from '@/helper/notification'

const props = defineProps<{
  profile?: ConfigProfile | null
  providers: NodeProvider[]
}>()

const emit = defineEmits<{
  (e: 'saved'): void
}>()

const isOpen = defineModel<boolean>({ default: false })
const { t } = useI18n()

const isSaving = ref(false)
const isEdit = computed(() => !!props.profile?.id)

const form = reactive<Partial<ConfigProfile>>({
  id: '',
  tag: '',
  name: '',
  description: '',
  provider_tags: [],
  target_selectors: ['Proxy', 'Auto'],
  auto_reload: true,
})

const resetForm = () => {
  if (props.profile) {
    Object.assign(form, {
      id: props.profile.id,
      tag: props.profile.tag,
      name: props.profile.name,
      description: props.profile.description,
      provider_tags: [...(props.profile.provider_tags || [])],
      target_selectors: [...(props.profile.target_selectors || ['Proxy', 'Auto'])],
      auto_reload: props.profile.auto_reload ?? true,
    })
  } else {
    Object.assign(form, {
      id: '',
      tag: '',
      name: '',
      description: '',
      provider_tags: [],
      target_selectors: ['Proxy', 'Auto'],
      auto_reload: true,
    })
  }
}

watch(
  () => [isOpen.value, props.profile],
  ([open]) => {
    if (open) {
      resetForm()
    }
  },
  { immediate: true },
)

const handleSave = async () => {
  if (!form.name || !form.tag) return
  isSaving.value = true
  try {
    if (form.id) {
      await controllerApi.updateProfile(form.id, form)
      showNotification({ content: t('profiles.profileUpdated', '配置方案已更新'), type: 'alert-success' })
    } else {
      await controllerApi.createProfile(form)
      showNotification({ content: t('profiles.profileCreated', '配置方案已创建'), type: 'alert-success' })
    }
    isOpen.value = false
    emit('saved')
  } catch (err: unknown) {
    const msg = (err as any)?.response?.data?.message || (err as Error)?.message || '保存方案失败'
    showNotification({ content: msg, type: 'alert-error' })
  } finally {
    isSaving.value = false
  }
}
</script>
