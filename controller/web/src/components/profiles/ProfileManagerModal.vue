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
          :class="{ 'tab-active !bg-primary !text-primary-content shadow-sm': activeTab === 'profiles' }"
          @click="activeTab = 'profiles'"
        >
          <RectangleGroupIcon class="size-4" />
          <span>{{ t('profiles.tabProfiles', '配置方案 (Profiles)') }}</span>
          <span class="badge badge-sm badge-neutral">{{ profilesList.length }}</span>
        </button>
        <button
          class="tab tab-md flex-1 gap-2 font-medium transition-all"
          :class="{ 'tab-active !bg-primary !text-primary-content shadow-sm': activeTab === 'providers' }"
          @click="activeTab = 'providers'"
        >
          <CloudArrowDownIcon class="size-4" />
          <span>{{ t('profiles.tabProviders', '节点订阅源 (Providers)') }}</span>
          <span class="badge badge-sm badge-neutral">{{ providersList.length }}</span>
        </button>
      </div>

      <!-- TAB 1: 配置组方案 (Profiles) -->
      <div
        v-if="activeTab === 'profiles'"
        class="flex flex-col gap-4"
      >
        <div class="flex items-center justify-between">
          <div class="text-xs text-base-content/60">
            {{ t('profiles.profilesDesc', '配置组将基础分流框架与选中的订阅节点片段动态合并，无感平滑切换。') }}
          </div>
          <button
            class="btn btn-primary btn-sm gap-1.5 shadow-sm"
            @click="openCreateProfileDialog"
          >
            <PlusIcon class="size-4" />
            {{ t('profiles.createProfile', '新建方案') }}
          </button>
        </div>

        <!-- 方案列表 -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3 max-h-[420px] overflow-y-auto pr-1">
          <div
            v-for="prof in profilesList"
            :key="prof.id"
            class="card bg-base-300/20 border transition-all duration-200 relative overflow-hidden flex flex-col justify-between p-4 rounded-xl"
            :class="prof.is_active ? 'border-primary shadow-md bg-primary/5 ring-1 ring-primary/30' : 'border-base-300/60 hover:border-base-content/20'"
          >
            <!-- 激活徽章 -->
            <div
              v-if="prof.is_active"
              class="absolute top-0 right-0 bg-primary text-primary-content px-3 py-0.5 text-[11px] font-semibold rounded-bl-lg flex items-center gap-1 shadow-sm"
            >
              <CheckCircleIcon class="size-3.5" />
              <span>运行中</span>
            </div>

            <div class="flex flex-col gap-2">
              <div class="flex items-center gap-2">
                <span class="font-bold text-sm text-base-content">{{ prof.name }}</span>
                <span class="badge badge-sm font-mono text-xs opacity-70">{{ prof.tag }}</span>
              </div>
              <p class="text-xs text-base-content/60 line-clamp-2 min-h-4">
                {{ prof.description || '无描述' }}
              </p>

              <!-- 关联的 Providers 标签 -->
              <div class="flex flex-wrap gap-1 mt-1">
                <span
                  v-for="pTag in prof.provider_tags"
                  :key="pTag"
                  class="badge badge-xs badge-outline badge-primary text-[10px] px-1.5 py-0.5"
                >
                  {{ pTag }}
                </span>
                <span
                  v-if="!prof.provider_tags || prof.provider_tags.length === 0"
                  class="text-[11px] text-base-content/40 italic"
                >
                  未挂载外部节点 (纯 Base)
                </span>
              </div>
            </div>

            <!-- 操作按钮栏 -->
            <div class="flex items-center justify-between mt-4 pt-3 border-t border-base-300/40">
              <div class="flex items-center gap-1.5 text-xs text-base-content/50">
                <span :class="prof.auto_reload ? 'text-success' : 'text-base-content/40'">●</span>
                <span>{{ prof.auto_reload ? '自动热重载' : '手动重启' }}</span>
              </div>

              <div class="flex items-center gap-1.5">
                <button
                  v-if="!prof.is_active"
                  class="btn btn-xs btn-primary gap-1"
                  :disabled="isApplying === prof.id"
                  @click="handleApplyProfile(prof.id)"
                >
                  <span
                    v-if="isApplying === prof.id"
                    class="loading loading-spinner loading-xs"
                  />
                  <CheckIcon
                    v-else
                    class="size-3.5"
                  />
                  <span>应用此方案</span>
                </button>
                <button
                  class="btn btn-xs btn-ghost btn-square"
                  title="编辑"
                  @click="openEditProfileDialog(prof)"
                >
                  <PencilSquareIcon class="size-3.5" />
                </button>
                <button
                  v-if="prof.id !== 'default'"
                  class="btn btn-xs btn-ghost btn-square text-error hover:bg-error/10"
                  title="删除"
                  @click="handleDeleteProfile(prof.id)"
                >
                  <TrashIcon class="size-3.5" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- TAB 2: 节点订阅源 (Providers) -->
      <div
        v-if="activeTab === 'providers'"
        class="flex flex-col gap-4"
      >
        <div class="flex items-center justify-between">
          <div class="text-xs text-base-content/60">
            {{ t('profiles.providersDesc', '支持 Clash YAML、sing-box JSON 与 Base64 订阅，自动提取并转换为独立节点片段。') }}
          </div>
          <button
            class="btn btn-primary btn-sm gap-1.5 shadow-sm"
            @click="openCreateProviderDialog"
          >
            <PlusIcon class="size-4" />
            {{ t('profiles.addProvider', '添加订阅源') }}
          </button>
        </div>

        <!-- 订阅源列表 -->
        <div class="flex flex-col gap-2.5 max-h-[420px] overflow-y-auto pr-1">
          <div
            v-if="providersList.length === 0"
            class="flex flex-col items-center justify-center py-12 text-base-content/40 gap-2 border border-dashed border-base-300 rounded-xl"
          >
            <CloudArrowDownIcon class="size-8 opacity-40" />
            <span class="text-xs">暂无订阅源，点击上方按钮添加</span>
          </div>

          <div
            v-for="prov in providersList"
            :key="prov.id"
            class="flex items-center justify-between p-3.5 bg-base-300/20 border border-base-300/50 hover:border-base-content/20 rounded-xl transition-all"
          >
            <div class="flex items-center gap-3 min-w-0">
              <div
                class="size-9 rounded-lg bg-base-300/60 flex items-center justify-center shrink-0"
                :class="prov.node_count > 0 ? 'text-primary' : 'text-base-content/40'"
              >
                <CloudArrowDownIcon class="size-5" />
              </div>
              <div class="flex flex-col min-w-0">
                <div class="flex items-center gap-2">
                  <span class="font-bold text-sm text-base-content truncate">{{ prov.name }}</span>
                  <span class="badge badge-xs font-mono text-[10px] opacity-70">{{ prov.tag }}</span>
                  <span class="badge badge-xs badge-info text-[10px] uppercase">{{ prov.format }}</span>
                </div>
                <div class="flex items-center gap-3 text-xs text-base-content/50 mt-0.5 truncate">
                  <span class="truncate max-w-[260px]">{{ prov.url || '本地文件' }}</span>
                  <span>•</span>
                  <span>{{ prov.node_count }} 节点</span>
                  <span v-if="prov.last_updated">• {{ formatTime(prov.last_updated) }}</span>
                </div>
                <div
                  v-if="prov.last_error"
                  class="text-[11px] text-error mt-0.5 truncate"
                >
                  错误: {{ prov.last_error }}
                </div>
              </div>
            </div>

            <!-- 操作按钮 -->
            <div class="flex items-center gap-1.5 shrink-0">
              <button
                class="btn btn-xs btn-outline gap-1"
                :disabled="isRefreshing === prov.id"
                title="刷新订阅"
                @click="handleRefreshProvider(prov.id)"
              >
                <ArrowPathIcon
                  class="size-3.5"
                  :class="{ 'animate-spin': isRefreshing === prov.id }"
                />
                <span>{{ isRefreshing === prov.id ? '更新中' : '更新' }}</span>
              </button>
              <button
                class="btn btn-xs btn-ghost btn-square"
                title="查看节点"
                @click="openInspectNodesDialog(prov.tag)"
              >
                <EyeIcon class="size-3.5" />
              </button>
              <button
                class="btn btn-xs btn-ghost btn-square"
                title="编辑"
                @click="openEditProviderDialog(prov)"
              >
                <PencilSquareIcon class="size-3.5" />
              </button>
              <button
                class="btn btn-xs btn-ghost btn-square text-error hover:bg-error/10"
                title="删除"
                @click="handleDeleteProvider(prov.id)"
              >
                <TrashIcon class="size-3.5" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 创建/编辑 Provider 弹窗 -->
    <DialogWrapper
      v-model="isProviderDialogVisible"
      :title="providerForm.id ? '编辑订阅源' : '添加订阅源'"
      size="medium"
    >
      <div class="flex flex-col gap-3 py-2 text-xs">
        <TextInput
          v-model="providerForm.name"
          label="订阅名称"
          placeholder="例如：主力机场 A"
        />
        <TextInput
          v-model="providerForm.tag"
          label="标识 (Tag)"
          placeholder="例如：airport-a (用于生成 outbound-airport-a.json)"
          :disabled="!!providerForm.id"
        />
        <TextInput
          v-model="providerForm.url"
          label="订阅 URL"
          placeholder="https://example.com/api/v1/client/subscribe?token=..."
        />
        <div class="grid grid-cols-2 gap-2">
          <SelectInput
            v-model="providerForm.format"
            label="订阅格式"
            :options="[
              { label: '自动检测 (Auto)', value: 'auto' },
              { label: 'Clash YAML (proxies 数组)', value: 'clash' },
              { label: 'sing-box JSON (outbounds 格式)', value: 'sing-box' },
              { label: 'Base64 节点 URI', value: 'base64' },
            ]"
          />
          <TextInput
            v-model.number="providerForm.update_interval_min"
            type="number"
            label="自动更新间隔 (分钟)"
            placeholder="0 表示不自动更新"
          />
        </div>
        <div class="form-control">
          <label class="label cursor-pointer justify-start gap-2">
            <input
              type="checkbox"
              v-model="providerForm.enabled"
              class="checkbox checkbox-sm checkbox-primary"
            />
            <span class="label-text">启用该订阅源</span>
          </label>
        </div>
        <div class="modal-action mt-2">
          <button
            class="btn btn-sm btn-ghost"
            @click="isProviderDialogVisible = false"
          >
            取消
          </button>
          <button
            class="btn btn-sm btn-primary"
            :disabled="!providerForm.tag || !providerForm.url"
            @click="handleSaveProvider"
          >
            保存并拉取
          </button>
        </div>
      </div>
    </DialogWrapper>

    <!-- 创建/编辑 Profile 弹窗 -->
    <DialogWrapper
      v-model="isProfileDialogVisible"
      :title="profileForm.id ? '编辑配置方案' : '新建配置方案'"
      size="medium"
    >
      <div class="flex flex-col gap-3 py-2 text-xs">
        <TextInput
          v-model="profileForm.name"
          label="方案名称"
          placeholder="例如：家庭主力分流 / 游戏低延迟模式"
        />
        <TextInput
          v-model="profileForm.tag"
          label="标识 (Tag)"
          placeholder="例如：gaming / travel"
          :disabled="!!profileForm.id"
        />
        <TextInput
          v-model="profileForm.description"
          label="方案描述"
          placeholder="可选描述说明"
        />

        <div class="flex flex-col gap-1.5">
          <label class="font-medium text-base-content/80">挂载的订阅源 / 节点片段：</label>
          <div class="flex flex-wrap gap-2 p-2 bg-base-300/30 rounded-lg border border-base-300">
            <label
              v-for="prov in providersList"
              :key="prov.tag"
              class="cursor-pointer label gap-1.5 p-1 bg-base-100 rounded-md border border-base-300/60 hover:border-primary/50 text-xs"
            >
              <input
                type="checkbox"
                :value="prov.tag"
                v-model="profileForm.provider_tags"
                class="checkbox checkbox-xs checkbox-primary"
              />
              <span>{{ prov.name }} ({{ prov.tag }})</span>
            </label>
            <span
              v-if="providersList.length === 0"
              class="text-base-content/40 italic p-1"
            >
              暂无可用的订阅源
            </span>
          </div>
        </div>

        <div class="form-control">
          <label class="label cursor-pointer justify-start gap-2">
            <input
              type="checkbox"
              v-model="profileForm.auto_reload"
              class="checkbox checkbox-sm checkbox-primary"
            />
            <span class="label-text">订阅更新后自动热重载内核</span>
          </label>
        </div>

        <div class="modal-action mt-2">
          <button
            class="btn btn-sm btn-ghost"
            @click="isProfileDialogVisible = false"
          >
            取消
          </button>
          <button
            class="btn btn-sm btn-primary"
            :disabled="!profileForm.name || !profileForm.tag"
            @click="handleSaveProfile"
          >
            保存方案
          </button>
        </div>
      </div>
    </DialogWrapper>

    <!-- 查看节点抽屉/弹窗 -->
    <DialogWrapper
      v-model="isInspectNodesVisible"
      :title="`订阅节点预览 (${inspectTargetTag})`"
      size="large"
    >
      <div class="flex flex-col gap-2 py-2 max-h-[460px] overflow-y-auto">
        <div class="text-xs text-base-content/60">
          共提取到 {{ inspectedNodes.length }} 个代理节点：
        </div>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-2">
          <div
            v-for="(node, i) in inspectedNodes"
            :key="i"
            class="p-2.5 bg-base-300/30 rounded-lg border border-base-300/60 flex items-center justify-between text-xs"
          >
            <div class="flex items-center gap-2 min-w-0">
              <span class="badge badge-xs font-mono uppercase">{{ node.type }}</span>
              <span class="font-medium truncate">{{ node.tag }}</span>
            </div>
            <span class="text-base-content/50 font-mono text-[11px] shrink-0">
              {{ node.server }}:{{ node.server_port }}
            </span>
          </div>
        </div>
      </div>
    </DialogWrapper>
  </DialogWrapper>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import TextInput from '@/components/common/TextInput.vue'
import SelectInput from '@/components/common/SelectInput.vue'
import {
  controllerApi,
  type ConfigProfile,
  type NodeProvider,
} from '@/api/controller'
import {
  RectangleGroupIcon,
  CloudArrowDownIcon,
  PlusIcon,
  ArrowPathIcon,
  CheckCircleIcon,
  CheckIcon,
  PencilSquareIcon,
  TrashIcon,
  EyeIcon,
} from '@heroicons/vue/24/outline'

const isOpen = defineModel<boolean>()
const { t } = useI18n()

const activeTab = ref<'profiles' | 'providers'>('profiles')
const profilesList = ref<ConfigProfile[]>([])
const providersList = ref<NodeProvider[]>([])
const isApplying = ref<string | null>(null)
const isRefreshing = ref<string | null>(null)

// 弹窗状态
const isProviderDialogVisible = ref(false)
const isProfileDialogVisible = ref(false)
const isInspectNodesVisible = ref(false)
const inspectTargetTag = ref('')
const inspectedNodes = ref<any[]>([])

const providerForm = reactive<Partial<NodeProvider>>({
  id: '',
  tag: '',
  name: '',
  type: 'remote',
  url: '',
  format: 'auto',
  update_interval_min: 720,
  enabled: true,
})

const profileForm = reactive<Partial<ConfigProfile>>({
  id: '',
  tag: '',
  name: '',
  description: '',
  provider_tags: [],
  target_selectors: ['Proxy', 'Auto'],
  auto_reload: true,
})

const loadData = async () => {
  try {
    const [profs, provs] = await Promise.all([
      controllerApi.profiles(),
      controllerApi.providers(),
    ])
    profilesList.value = profs
    providersList.value = provs
  } catch (err) {
    console.error('加载 Profiles & Providers 失败:', err)
  }
}

onMounted(() => {
  loadData()
})

const formatTime = (iso?: string) => {
  if (!iso) return ''
  const d = new Date(iso)
  return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
}

// 方案操作
const openCreateProfileDialog = () => {
  Object.assign(profileForm, {
    id: '',
    tag: '',
    name: '',
    description: '',
    provider_tags: [],
    target_selectors: ['Proxy', 'Auto'],
    auto_reload: true,
  })
  isProfileDialogVisible.value = true
}

const openEditProfileDialog = (prof: ConfigProfile) => {
  Object.assign(profileForm, {
    id: prof.id,
    tag: prof.tag,
    name: prof.name,
    description: prof.description,
    provider_tags: [...(prof.provider_tags || [])],
    target_selectors: [...(prof.target_selectors || ['Proxy', 'Auto'])],
    auto_reload: prof.auto_reload,
  })
  isProfileDialogVisible.value = true
}

const handleSaveProfile = async () => {
  try {
    if (profileForm.id) {
      await controllerApi.updateProfile(profileForm.id, profileForm)
    } else {
      await controllerApi.createProfile(profileForm)
    }
    isProfileDialogVisible.value = false
    await loadData()
  } catch (err: any) {
    alert(err?.response?.data?.message || err?.message || '保存方案失败')
  }
}

const handleDeleteProfile = async (id: string) => {
  if (!confirm('确定要删除该配置方案吗？')) return
  try {
    await controllerApi.deleteProfile(id)
    await loadData()
  } catch (err: any) {
    alert(err?.response?.data?.message || err?.message || '删除方案失败')
  }
}

const handleApplyProfile = async (id: string) => {
  isApplying.value = id
  try {
    await controllerApi.applyProfile(id)
    await loadData()
  } catch (err: any) {
    alert(err?.response?.data?.message || err?.message || '应用方案失败')
  } finally {
    isApplying.value = null
  }
}

// 订阅源操作
const openCreateProviderDialog = () => {
  Object.assign(providerForm, {
    id: '',
    tag: '',
    name: '',
    type: 'remote',
    url: '',
    format: 'auto',
    update_interval_min: 720,
    enabled: true,
  })
  isProviderDialogVisible.value = true
}

const openEditProviderDialog = (prov: NodeProvider) => {
  Object.assign(providerForm, {
    id: prov.id,
    tag: prov.tag,
    name: prov.name,
    type: prov.type,
    url: prov.url,
    format: prov.format,
    update_interval_min: prov.update_interval_min,
    enabled: prov.enabled,
  })
  isProviderDialogVisible.value = true
}

const handleSaveProvider = async () => {
  try {
    if (providerForm.id) {
      await controllerApi.updateProvider(providerForm.id, providerForm)
    } else {
      await controllerApi.createProvider(providerForm)
    }
    isProviderDialogVisible.value = false
    await loadData()
  } catch (err: any) {
    alert(err?.response?.data?.message || err?.message || '保存订阅源失败')
  }
}

const handleDeleteProvider = async (id: string) => {
  if (!confirm('确定要删除该订阅源及关联的节点片段吗？')) return
  try {
    await controllerApi.deleteProvider(id)
    await loadData()
  } catch (err: any) {
    alert(err?.response?.data?.message || err?.message || '删除订阅源失败')
  }
}

const handleRefreshProvider = async (id: string) => {
  isRefreshing.value = id
  try {
    await controllerApi.refreshProvider(id)
    await loadData()
  } catch (err: any) {
    alert(err?.response?.data?.message || err?.message || '刷新订阅失败')
  } finally {
    isRefreshing.value = null
  }
}

const openInspectNodesDialog = async (tag: string) => {
  inspectTargetTag.value = tag
  try {
    const res = await controllerApi.getProviderNodes(tag)
    inspectedNodes.value = res.outbounds || []
    isInspectNodesVisible.value = true
  } catch (err: any) {
    alert(err?.response?.data?.message || err?.message || '获取节点列表失败')
  }
}
</script>
