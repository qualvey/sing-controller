<template>
  <DialogWrapper
    v-model="isOpen"
    :title="t('profiles.managerTitle', '配置组与订阅源管理 (Profiles & Providers)')"
    size="large"
  >
    <div class="flex flex-col gap-5 py-1">
      <!-- 标签页切换：配置组方案 vs 订阅源管理 -->
      <div class="tabs tabs-boxed bg-base-300/40 p-1 rounded-xl">
        <button
          class="tab tab-md flex-1 gap-2 font-medium transition-all"
          :class="{ 'tab-active !bg-primary !text-primary-content shadow-sm': activeProfilesTab === 'profiles' }"
          @click="activeProfilesTab = 'profiles'"
        >
          <RectangleGroupIcon class="size-4" />
          <span>{{ t('profiles.tabProfiles', '配置方案 (Profiles)') }}</span>
          <span class="badge badge-sm badge-neutral">{{ profilesList.length }}</span>
        </button>
        <button
          class="tab tab-md flex-1 gap-2 font-medium transition-all"
          :class="{ 'tab-active !bg-primary !text-primary-content shadow-sm': activeProfilesTab === 'providers' }"
          @click="activeProfilesTab = 'providers'"
        >
          <CloudArrowDownIcon class="size-4" />
          <span>{{ t('profiles.tabProviders', '节点订阅源 (Providers)') }}</span>
          <span class="badge badge-sm badge-neutral">{{ providersList.length }}</span>
        </button>
      </div>

      <!-- TAB 1: 配置组方案 (Profiles) -->
      <ProfileCardList
        v-if="activeProfilesTab === 'profiles'"
        :profiles="profilesList"
        :is-applying="isApplyingProfileId"
        @create="handleOpenCreateProfile"
        @edit="handleOpenEditProfile"
        @delete="handleDeleteProfileConfirm"
        @apply="applyProfile"
      />

      <!-- TAB 2: 节点订阅源 (Providers) -->
      <ProviderTableList
        v-if="activeProfilesTab === 'providers'"
        :providers="providersList"
        :is-refreshing="isRefreshingProviderId"
        @create="handleOpenCreateProvider"
        @edit="handleOpenEditProvider"
        @delete="handleDeleteProviderConfirm"
        @refresh="refreshProvider"
        @inspect="handleOpenInspectNodes"
      />
    </div>

    <!-- 方案编辑/创建子弹窗 -->
    <ProfileEditDialog
      v-model="isProfileDialogVisible"
      :profile="selectedProfile"
      :providers="providersList"
      @saved="loadProfilesData"
    />

    <!-- 订阅源编辑/创建子弹窗 -->
    <ProviderEditDialog
      v-model="isProviderDialogVisible"
      :provider="selectedProvider"
      @saved="loadProfilesData"
    />

    <!-- 节点预览子弹窗 -->
    <ProviderNodesPreview
      v-model="isInspectNodesVisible"
      :tag="inspectTargetTag"
    />
  </DialogWrapper>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import {
  RectangleGroupIcon,
  CloudArrowDownIcon,
} from '@heroicons/vue/24/outline'
import type { ConfigProfile, NodeProvider } from '@/api/controller'
import {
  activeProfilesTab,
  profilesList,
  providersList,
  isApplyingProfileId,
  isRefreshingProviderId,
  loadProfilesData,
  applyProfile,
  deleteProfile,
  refreshProvider,
  deleteProvider,
} from '@/composables/profiles'
import ProfileCardList from './ProfileCardList.vue'
import ProviderTableList from './ProviderTableList.vue'
import ProfileEditDialog from './ProfileEditDialog.vue'
import ProviderEditDialog from './ProviderEditDialog.vue'
import ProviderNodesPreview from './ProviderNodesPreview.vue'

const isOpen = defineModel<boolean>({ default: false })
const { t } = useI18n()

// 编辑态状态
const isProfileDialogVisible = ref(false)
const selectedProfile = ref<ConfigProfile | null>(null)

const isProviderDialogVisible = ref(false)
const selectedProvider = ref<NodeProvider | null>(null)

const isInspectNodesVisible = ref(false)
const inspectTargetTag = ref('')

watch(
  () => isOpen.value,
  (open) => {
    if (open) {
      loadProfilesData()
    }
  },
)

onMounted(() => {
  loadProfilesData()
})

// Profile 弹窗打开逻辑
const handleOpenCreateProfile = () => {
  selectedProfile.value = null
  isProfileDialogVisible.value = true
}

const handleOpenEditProfile = (profile: ConfigProfile) => {
  selectedProfile.value = profile
  isProfileDialogVisible.value = true
}

const handleDeleteProfileConfirm = async (id: string) => {
  await deleteProfile(id)
}

// Provider 弹窗打开逻辑
const handleOpenCreateProvider = () => {
  selectedProvider.value = null
  isProviderDialogVisible.value = true
}

const handleOpenEditProvider = (provider: NodeProvider) => {
  selectedProvider.value = provider
  isProviderDialogVisible.value = true
}

const handleDeleteProviderConfirm = async (id: string) => {
  await deleteProvider(id)
}

const handleOpenInspectNodes = (tag: string) => {
  inspectTargetTag.value = tag
  isInspectNodesVisible.value = true
}
</script>
