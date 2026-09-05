<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import {
  isAssignModalOpen,
  assignTargetNodes,
  closeAssignModal,
  useUngroupedProxies,
} from '@/composables/ungroupedProxies'
import { proxyMap } from '@/assembly/proxies'
import { showToast } from '@/helper/toast'
import { FolderPlusIcon, CheckIcon, MagnifyingGlassIcon } from '@heroicons/vue/24/outline'

const { t } = useI18n()
const { availableProxyGroups, assignNodesToGroups } = useUngroupedProxies()

const isSaving = ref(false)
const selectedGroups = ref<string[]>([])
const searchKeyword = ref('')

// 当弹窗打开时，重置选择
watch(isAssignModalOpen, (visible) => {
  if (visible) {
    selectedGroups.value = []
    searchKeyword.value = ''
  }
})

const filteredGroups = computed(() => {
  const keyword = searchKeyword.value.trim().toLowerCase()
  if (!keyword) return availableProxyGroups.value
  return availableProxyGroups.value.filter((name) =>
    name.toLowerCase().includes(keyword)
  )
})

const isAllSelected = computed(() => {
  if (filteredGroups.value.length === 0) return false
  return filteredGroups.value.every((name) => selectedGroups.value.includes(name))
})

const toggleSelectAll = () => {
  if (isAllSelected.value) {
    selectedGroups.value = selectedGroups.value.filter(
      (name) => !filteredGroups.value.includes(name)
    )
  } else {
    const combined = new Set([...selectedGroups.value, ...filteredGroups.value])
    selectedGroups.value = Array.from(combined)
  }
}

const toggleGroup = (name: string) => {
  const idx = selectedGroups.value.indexOf(name)
  if (idx >= 0) {
    selectedGroups.value.splice(idx, 1)
  } else {
    selectedGroups.value.push(name)
  }
}

const handleConfirm = async () => {
  if (selectedGroups.value.length === 0) {
    showToast(t('ungroupedProxies.selectAtLeastOneGroup') || '请至少选择一个目标策略组', 'warning')
    return
  }

  isSaving.value = true
  try {
    await assignNodesToGroups(assignTargetNodes.value, selectedGroups.value)
    showToast(
      t('ungroupedProxies.assignSuccess') || `成功将节点分配至 ${selectedGroups.value.join(', ')}`,
      'success'
    )
    closeAssignModal()
  } catch (err: unknown) {
    showToast((err as Error).message || t('ungroupedProxies.assignFailed') || '分配失败', 'error')
  } finally {
    isSaving.value = false
  }
}
</script>

<template>
  <DialogWrapper
    v-model="isAssignModalOpen"
    :title="t('ungroupedProxies.assignTitle') || '分配节点至策略组'"
  >
    <div class="flex flex-col gap-3 py-1">
      <!-- 提示信息 -->
      <div class="bg-base-200/60 rounded-xl p-3 text-xs text-base-content/80 flex flex-col gap-1 border border-base-content/5">
        <div class="font-semibold text-base-content flex items-center gap-1.5">
          <FolderPlusIcon class="size-4 text-primary" />
          <span v-if="assignTargetNodes.length === 1">
            将节点「<span class="text-primary font-mono font-medium">{{ assignTargetNodes[0] }}</span>」加入策略组：
          </span>
          <span v-else>
            将已选定的 <span class="text-primary font-bold">{{ assignTargetNodes.length }}</span> 个未分组节点加入策略组：
          </span>
        </div>
        <div
          v-if="assignTargetNodes.length > 1"
          class="flex flex-wrap gap-1 max-h-20 overflow-y-auto mt-1 p-1 bg-base-100 rounded-lg"
        >
          <span
            v-for="tag in assignTargetNodes"
            :key="tag"
            class="badge badge-xs badge-ghost font-mono"
          >
            {{ tag }}
          </span>
        </div>
      </div>

      <!-- 搜索与快捷操作 -->
      <div class="flex items-center justify-between gap-2 mt-1">
        <div class="relative flex-1">
          <MagnifyingGlassIcon class="size-4 absolute left-2.5 top-1/2 -translate-y-1/2 text-base-content/40" />
          <input
            v-model="searchKeyword"
            type="text"
            placeholder="搜索策略组名称..."
            class="input input-sm input-bordered w-full pl-8 text-xs rounded-lg"
          />
        </div>
        <button
          class="btn btn-sm btn-ghost text-xs font-normal"
          @click="toggleSelectAll"
        >
          {{ isAllSelected ? (t('ungroupedProxies.deselectAll') || '取消全选') : (t('ungroupedProxies.selectAll') || '全选') }}
        </button>
      </div>

      <!-- 策略组选择列表 -->
      <div class="flex flex-col gap-1.5 max-h-60 overflow-y-auto pr-1">
        <div
          v-if="filteredGroups.length === 0"
          class="text-center py-6 text-xs text-base-content/40"
        >
          {{ t('ungroupedProxies.noGroupsAvailable') || '暂无可用的目标策略组' }}
        </div>

        <div
          v-for="groupName in filteredGroups"
          :key="groupName"
          class="flex items-center justify-between p-2.5 rounded-xl border cursor-pointer transition-all select-none"
          :class="[
            selectedGroups.includes(groupName)
              ? 'bg-primary/10 border-primary/40 text-primary shadow-xs'
              : 'bg-base-200/40 border-base-content/5 hover:bg-base-200/80 text-base-content/90',
          ]"
          @click="toggleGroup(groupName)"
        >
          <div class="flex items-center gap-2.5 min-w-0">
            <input
              type="checkbox"
              class="checkbox checkbox-sm checkbox-primary pointer-events-none rounded-md"
              :checked="selectedGroups.includes(groupName)"
            />
            <div class="min-w-0">
              <div class="font-medium text-xs sm:text-sm truncate">
                {{ groupName }}
              </div>
              <div class="text-[11px] opacity-60 flex items-center gap-2">
                <span>{{ proxyMap[groupName]?.type || 'selector' }}</span>
                <span>·</span>
                <span>{{ proxyMap[groupName]?.all?.length || 0 }} 个节点</span>
              </div>
            </div>
          </div>

          <div
            v-if="selectedGroups.includes(groupName)"
            class="size-5 rounded-full bg-primary text-primary-content flex items-center justify-center shrink-0"
          >
            <CheckIcon class="size-3.5 stroke-3" />
          </div>
        </div>
      </div>

      <!-- 底部按钮 -->
      <div class="flex items-center justify-end gap-2 pt-2 border-t border-base-content/10 mt-1">
        <button
          class="btn btn-sm btn-ghost rounded-lg font-normal"
          :disabled="isSaving"
          @click="closeAssignModal"
        >
          {{ t('cancel') || '取消' }}
        </button>
        <button
          class="btn btn-sm btn-primary rounded-lg gap-1.5 font-medium min-w-20"
          :disabled="isSaving || selectedGroups.length === 0"
          @click="handleConfirm"
        >
          <span
            v-if="isSaving"
            class="loading loading-spinner loading-xs"
          />
          <FolderPlusIcon
            v-else
            class="size-4"
          />
          <span>{{ isSaving ? '保存中...' : `加入已选分组 (${selectedGroups.length})` }}</span>
        </button>
      </div>
    </div>
  </DialogWrapper>
</template>
