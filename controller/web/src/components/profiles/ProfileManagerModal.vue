<template>
  <DialogWrapper
    v-model="isOpen"
    :title="t('profiles.managerTitle', '配置组与订阅源管理 (Profiles & Providers)')"
    size="large"
    :closable="!isLocked"
  >
    <div class="flex flex-col gap-4 py-1">
      <!-- 初次启动未就绪引导提示横幅 -->
      <div
        v-if="isLocked"
        class="alert alert-info bg-info/10 border-info/20 text-info-content text-xs rounded-xl flex items-start gap-3 p-3.5 shadow-sm"
      >
        <SparklesIcon class="size-5 text-info shrink-0 mt-0.5" />
        <div class="flex-1 min-w-0">
          <div class="font-bold text-sm text-base-content mb-1">
            欢迎使用 sing-controller！检测到首次启动待初始化
          </div>
          <div class="text-base-content/80 leading-relaxed">
            系统尚未生成 sing-box 主配置且内核未运行。请按以下步骤完成初始化：
            <ol class="list-decimal list-inside mt-1.5 space-y-1 font-medium">
              <li>在下方 <span class="badge badge-sm badge-neutral">节点订阅源 (Providers)</span> 点击“添加订阅源”填入您的订阅 URL 并拉取节点</li>
              <li>切换至 <span class="badge badge-sm badge-neutral">配置方案 (Profiles)</span> 点击“应用”生成完整配置并启动内核</li>
            </ol>
            <span class="text-warning text-[11px] block mt-1.5">
              ⚠️ 提示：在内核成功启动并联通 gRPC 之前，界面将保持锁定引导状态。
            </span>
          </div>
        </div>
      </div>

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
        :allow-reapply="statusStore.status?.config_exists === false || statusStore.status?.core_alive === false"
        :initializing="statusStore.status?.config_exists === false"
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
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import {
  RectangleGroupIcon,
  CloudArrowDownIcon,
  SparklesIcon,
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
  providerEditRequest,
} from '@/composables/profiles'
import { useStatusStore } from '@/store/status'
import { startBackendSession } from '@/assembly/session'
import { showNotification } from '@/helper/notification'
import ProfileCardList from './ProfileCardList.vue'
import ProviderTableList from './ProviderTableList.vue'
import ProfileEditDialog from './ProfileEditDialog.vue'
import ProviderEditDialog from './ProviderEditDialog.vue'
import ProviderNodesPreview from './ProviderNodesPreview.vue'

const isOpen = defineModel<boolean>({ default: false })
const { t } = useI18n()
const statusStore = useStatusStore()

// 判断是否处于冷启动未就绪强制锁定状态（无配置文件，或无节点且内核离线）
const isLocked = computed(() => {
  return (
    statusStore.status?.config_exists === false ||
    (statusStore.status?.core_alive === false && (statusStore.status?.outbounds ?? 0) === 0)
  )
})

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

watch(providerEditRequest, (provider) => {
  if (!provider) return
  activeProfilesTab.value = 'providers'
  selectedProvider.value = provider
  isProviderDialogVisible.value = true
  providerEditRequest.value = null
})

let pollTimer: ReturnType<typeof setInterval> | null = null

const startPolling = () => {
  if (pollTimer) return
  pollTimer = setInterval(async () => {
    await statusStore.refresh()
    if (statusStore.status?.core_alive) {
      await loadProfilesData()
      startBackendSession()
    }
  }, 1200)
}

const stopPolling = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// 当进入锁定状态时，自动切换到 providers 订阅源 tab 并开启自愈轮询
watch(
  isLocked,
  (locked, oldLocked) => {
    if (locked) {
      activeProfilesTab.value = 'providers'
      isOpen.value = true
      startPolling()
    } else {
      stopPolling()
      if (oldLocked) {
        showNotification({
          content: '🎉 sing-box 内核已成功启动并联通！引导锁定已解除。',
          type: 'alert-success',
        })
        startBackendSession()
      }
    }
  },
  { immediate: true },
)

onMounted(() => {
  loadProfilesData()
  void statusStore.refresh()
})

onUnmounted(() => {
  stopPolling()
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
