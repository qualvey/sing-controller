<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useStorage } from '@vueuse/core'
import {
  useUngroupedProxies,
  openAssignModal,
  isBatchSpeedtesting,
} from '@/composables/ungroupedProxies'
import { proxyMap } from '@/assembly/proxies'
import { openProxyContextMenu } from '@/composables/proxyContextMenu'
import LatencyTag from '@/components/proxies/LatencyTag.vue'
import ProxyIcon from '@/components/proxies/ProxyIcon.vue'
import ProxyNodeGrid from '@/components/proxies/ProxyNodeGrid.vue'
import { nodeHealthEnabled, nodeHealthStatuses } from '@/store/nodeHealth'
import {
  BoltIcon,
  ChevronDownIcon,
  ChevronUpIcon,
  FolderPlusIcon,
  QueueListIcon,
  CheckIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'

const props = withDefaults(
  defineProps<{
    closeable?: boolean
  }>(),
  {
    closeable: false,
  },
)

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()
const { ungroupedNodeNames, batchSpeedtest } = useUngroupedProxies()
const nodeHealthStatus = (name: string) => {
  if (!nodeHealthEnabled.value) return undefined
  const status = nodeHealthStatuses.value[name]
  return status?.state === 'available' || status?.state === 'unavailable' ? status : undefined
}

const isCollapsed = useStorage('config/ungrouped-proxies-collapsed', false)
const isSelectionMode = ref(false)
const selectedNodeNames = ref<Set<string>>(new Set())

const isAllSelected = computed(() => {
  if (ungroupedNodeNames.value.length === 0) return false
  return ungroupedNodeNames.value.every((name) => selectedNodeNames.value.has(name))
})

const toggleSelectAll = () => {
  if (isAllSelected.value) {
    selectedNodeNames.value.clear()
  } else {
    selectedNodeNames.value = new Set(ungroupedNodeNames.value)
  }
}

const toggleSelectNode = (name: string) => {
  const set = new Set(selectedNodeNames.value)
  if (set.has(name)) {
    set.delete(name)
  } else {
    set.add(name)
  }
  selectedNodeNames.value = set
}

const handleSpeedtestAll = () => {
  batchSpeedtest()
}

const handleAssignSingle = (name: string) => {
  openAssignModal(name)
}

const handleAssignSelected = () => {
  if (selectedNodeNames.value.size === 0) {
    openAssignModal(ungroupedNodeNames.value)
  } else {
    openAssignModal(Array.from(selectedNodeNames.value))
  }
}

const handleContextMenu = (e: MouseEvent, name: string) => {
  openProxyContextMenu(e, {
    name,
    isGroup: false,
  })
}
</script>

<template>
  <div
    v-if="ungroupedNodeNames.length > 0"
    class="ungrouped-proxies-card base-container w-full rounded-2xl p-3 sm:p-4 border border-amber-500/20 dark:border-amber-500/15 shadow-sm transition-all"
  >
    <!-- 卡片头部 -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 pb-2 border-b border-base-200/60 dark:border-base-content/5">
      <!-- 标题与徽标 -->
      <div
        class="flex items-center gap-2.5 cursor-pointer select-none"
        @click="isCollapsed = !isCollapsed"
      >
        <div class="size-8 rounded-xl bg-amber-500/15 text-amber-500 flex items-center justify-center shrink-0">
          <QueueListIcon class="size-4.5 stroke-2" />
        </div>
        <div class="min-w-0">
          <div class="flex items-center gap-2">
            <span class="font-bold text-sm text-base-content">
              {{ t('ungroupedProxies.title') || '未分组节点' }}
            </span>
            <span class="badge badge-sm badge-warning font-semibold text-xs px-2">
              {{ ungroupedNodeNames.length }}
            </span>
          </div>
          <div class="text-[11px] text-base-content/50 truncate max-w-[280px] sm:max-w-md mt-0.5">
            {{ t('ungroupedProxies.desc') || '以下独立出站未加入任何策略组，可批量测速或再分配' }}
          </div>
        </div>
      </div>

      <!-- 右侧操作栏 -->
      <div class="flex items-center gap-1.5 self-end sm:self-auto shrink-0 flex-wrap">
        <!-- 批量测速按钮 -->
        <button
          class="btn btn-xs sm:btn-sm btn-ghost gap-1 text-base-content/80 hover:text-base-content rounded-lg"
          :disabled="isBatchSpeedtesting"
          :title="t('ungroupedProxies.speedtestAll') || '批量测速'"
          @click="handleSpeedtestAll"
        >
          <span
            v-if="isBatchSpeedtesting"
            class="loading loading-spinner loading-xs text-warning"
          />
          <BoltIcon
            v-else
            class="size-3.5 sm:size-4 text-warning"
          />
          <span class="text-xs">{{ isBatchSpeedtesting ? (t('ungroupedProxies.testing') || '测速中') : (t('ungroupedProxies.speedtestAll') || '批量测速') }}</span>
        </button>

        <!-- 多选模式切换 / 批量分配 -->
        <template v-if="isSelectionMode">
          <button
            class="btn btn-xs sm:btn-sm btn-ghost text-xs font-normal rounded-lg"
            @click="toggleSelectAll"
          >
            {{ isAllSelected ? (t('ungroupedProxies.deselectAll') || '取消全选') : (t('ungroupedProxies.selectAll') || '全选') }}
          </button>
          <button
            class="btn btn-xs sm:btn-sm btn-primary gap-1 text-xs rounded-lg font-medium shadow-xs"
            :disabled="selectedNodeNames.size === 0"
            @click="handleAssignSelected"
          >
            <FolderPlusIcon class="size-3.5" />
            <span>{{ t('ungroupedProxies.batchAssign') || '分配' }} ({{ selectedNodeNames.size }})</span>
          </button>
          <button
            class="btn btn-xs sm:btn-sm btn-ghost text-xs rounded-lg"
            @click="isSelectionMode = false; selectedNodeNames.clear()"
          >
            {{ t('cancel') || '取消' }}
          </button>
        </template>
        <template v-else>
          <button
            class="btn btn-xs sm:btn-sm btn-outline btn-warning gap-1 text-xs rounded-lg font-medium"
            @click="openAssignModal(ungroupedNodeNames)"
          >
            <FolderPlusIcon class="size-3.5 sm:size-4" />
            <span>{{ t('ungroupedProxies.batchAssign') || '全部加入策略组' }}</span>
          </button>
          <button
            class="btn btn-xs sm:btn-sm btn-ghost text-xs rounded-lg"
            @click="isSelectionMode = true"
          >
            多选
          </button>
        </template>

        <!-- 折叠/展开按钮 -->
        <button
          class="btn btn-xs sm:btn-sm btn-ghost btn-square rounded-lg"
          @click="isCollapsed = !isCollapsed"
        >
          <ChevronUpIcon
            v-if="!isCollapsed"
            class="size-4"
          />
          <ChevronDownIcon
            v-else
            class="size-4"
          />
        </button>

        <!-- 关闭按钮 (Dropdown 模式下有效) -->
        <button
          v-if="closeable"
          type="button"
          class="btn btn-xs sm:btn-sm btn-ghost btn-circle rounded-lg text-base-content/50 hover:text-base-content"
          title="关闭"
          @click="emit('close')"
        >
          <XMarkIcon class="size-4" />
        </button>
      </div>
    </div>

    <!-- 折叠预览 (Collapsed Preview) -->
    <div
      v-if="isCollapsed"
      class="flex flex-wrap gap-1.5 pt-2 text-xs cursor-pointer select-none"
      @click="isCollapsed = false"
    >
      <span
        v-for="name in ungroupedNodeNames.slice(0, 8)"
        :key="name"
        class="badge badge-sm badge-ghost text-[11px] font-mono py-1 px-2 text-base-content/70"
      >
        {{ name }}
      </span>
      <span
        v-if="ungroupedNodeNames.length > 8"
        class="badge badge-sm badge-ghost text-[11px] py-1 px-2 text-warning font-medium"
      >
        +{{ ungroupedNodeNames.length - 8 }} 更多...
      </span>
    </div>

    <!-- 展开节点卡片网格 (Expanded Grid) -->
    <div
      v-else
      class="pt-3"
    >
      <ProxyNodeGrid>
        <div
          v-for="name in ungroupedNodeNames"
          :key="name"
          class="group relative flex flex-col justify-between p-2 rounded-xl border transition-all duration-150 select-none bg-base-200/50 hover:bg-base-200 border-base-content/5 hover:border-amber-500/30 hover:shadow-sm"
          :class="[selectedNodeNames.has(name) && 'border-primary/50 bg-primary/5!']"
          @contextmenu.prevent.stop="handleContextMenu($event, name)"
        >
          <!-- 节点信息栏 -->
          <div class="flex items-start justify-between gap-1.5 min-w-0">
            <div class="flex items-center gap-1.5 min-w-0 flex-1">
              <!-- 多选模式下的 Checkbox -->
              <input
                v-if="isSelectionMode"
                type="checkbox"
                class="checkbox checkbox-xs checkbox-primary rounded shrink-0 cursor-pointer"
                :checked="selectedNodeNames.has(name)"
                @change="toggleSelectNode(name)"
                @click.stop
              />

              <!-- 节点图标与名称 -->
              <ProxyIcon
                v-if="proxyMap[name]?.icon"
                class="size-4 shrink-0 -mt-0.5"
                :icon="proxyMap[name].icon"
              />
              <span
                class="font-medium text-xs sm:text-sm text-base-content truncate cursor-pointer"
                :title="name"
                @click="isSelectionMode ? toggleSelectNode(name) : handleAssignSingle(name)"
              >
                {{ name }}
              </span>
            </div>

            <!-- 单节点快捷分配按钮 -->
            <button
              class="btn btn-ghost btn-xs btn-square size-6 opacity-0 group-hover:opacity-100 transition-opacity rounded-md hover:bg-primary/15 hover:text-primary shrink-0"
              :title="t('ungroupedProxies.quickAssign') || '加入策略组'"
              @click.stop="handleAssignSingle(name)"
            >
              <FolderPlusIcon class="size-3.5 stroke-2" />
            </button>
          </div>

          <!-- 底部协议与延迟 -->
          <div class="flex items-center justify-between gap-2 mt-2 pt-1 border-t border-base-content/5">
            <span class="badge badge-xs badge-ghost font-mono uppercase text-[10px] text-base-content/60 tracking-wider">
              {{ proxyMap[name]?.type || 'node' }}
            </span>
            <span
              v-if="nodeHealthStatus(name)"
              class="size-2 rounded-full"
              :class="nodeHealthStatus(name)?.state === 'available' ? 'bg-success' : 'bg-error'"
              :title="nodeHealthStatus(name)?.state === 'available' ? t('nodeHealthAvailable') : t('nodeHealthUnavailable')"
            />
            <LatencyTag
              class="shrink-0"
              :name="name"
              @click.stop
            />
          </div>
        </div>
      </ProxyNodeGrid>
    </div>
  </div>

  <!-- 独立节点为空时的空态展示 (仅在下拉模式下展示，非下拉模式自隐) -->
  <div
    v-else-if="closeable"
    class="ungrouped-proxies-card base-container w-full rounded-2xl p-4 border border-base-content/10 shadow-sm select-none"
  >
    <div class="flex items-center justify-between pb-2 mb-2 border-b border-base-content/5">
      <div class="flex items-center gap-2">
        <div class="size-7 rounded-lg bg-amber-500/15 text-amber-500 flex items-center justify-center shrink-0">
          <QueueListIcon class="size-4 stroke-2" />
        </div>
        <span class="font-bold text-sm text-base-content">
          {{ t('ungroupedProxies.title') || '未分组节点' }}
        </span>
      </div>
      <button
        type="button"
        class="btn btn-ghost btn-circle btn-xs text-base-content/50 hover:text-base-content"
        title="关闭"
        @click="emit('close')"
      >
        <XMarkIcon class="size-4" />
      </button>
    </div>
    <div class="py-6 flex flex-col items-center justify-center text-center">
      <div class="size-10 rounded-full bg-success/15 text-success flex items-center justify-center mb-2">
        <CheckIcon class="size-5 stroke-2" />
      </div>
      <div class="text-sm font-medium text-base-content">暂无独立出站节点</div>
      <div class="text-xs text-base-content/50 mt-1">所有代理节点均已分配至策略组中</div>
    </div>
  </div>
</template>

<style scoped>
.ungrouped-proxies-card {
  user-select: none;
}
</style>
