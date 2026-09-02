<template>
  <div class="flex flex-col gap-4">
    <div class="flex items-center justify-between">
      <div class="text-xs text-base-content/60">
        {{ t('profiles.providersDesc', '支持 Clash YAML、sing-box JSON 与 Base64 订阅，自动提取并转换为独立节点片段。') }}
      </div>
      <button
        class="btn btn-primary btn-sm gap-1.5 shadow-sm"
        @click="emit('create')"
      >
        <PlusIcon class="size-4" />
        {{ t('profiles.addProvider', '添加订阅源') }}
      </button>
    </div>

    <!-- 订阅源列表 -->
    <div class="flex flex-col gap-2.5 max-h-[420px] overflow-y-auto pr-1">
      <div
        v-if="providers.length === 0"
        class="flex flex-col items-center justify-center py-12 text-base-content/40 gap-2 border border-dashed border-base-300 rounded-xl"
      >
        <CloudArrowDownIcon class="size-8 opacity-40" />
        <span class="text-xs">{{ t('profiles.noProvidersTip', '暂无订阅源，点击上方按钮添加') }}</span>
      </div>

      <div
        v-for="prov in providers"
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
              <span class="truncate max-w-[260px]">{{ prov.url || t('profiles.localFile', '本地文件') }}</span>
              <span>•</span>
              <span>{{ t('profiles.nodeCountUnit', '{count} 节点', { count: prov.node_count }) }}</span>
              <span v-if="prov.last_updated">• {{ formatTime(prov.last_updated) }}</span>
            </div>
            <div
              v-if="prov.last_error"
              class="text-[11px] text-error mt-0.5 truncate"
            >
              {{ t('profiles.errorPrefix', '错误: {error}', { error: prov.last_error }) }}
            </div>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="flex items-center gap-1.5 shrink-0">
          <button
            class="btn btn-xs btn-outline gap-1"
            :disabled="isRefreshing === prov.id"
            :title="t('profiles.refreshProvider', '刷新订阅')"
            @click="emit('refresh', prov.id)"
          >
            <ArrowPathIcon
              class="size-3.5"
              :class="{ 'animate-spin': isRefreshing === prov.id }"
            />
            <span>{{ isRefreshing === prov.id ? t('profiles.updating', '更新中') : t('profiles.update', '更新') }}</span>
          </button>
          <button
            class="btn btn-xs btn-ghost btn-square"
            :title="t('profiles.inspectNodes', '查看节点')"
            @click="emit('inspect', prov.tag)"
          >
            <EyeIcon class="size-3.5" />
          </button>
          <button
            class="btn btn-xs btn-ghost btn-square"
            :title="t('common.edit', '编辑')"
            @click="emit('edit', prov)"
          >
            <PencilSquareIcon class="size-3.5" />
          </button>
          <button
            class="btn btn-xs btn-ghost btn-square text-error hover:bg-error/10"
            :title="t('common.delete', '删除')"
            @click="emit('delete', prov.id)"
          >
            <TrashIcon class="size-3.5" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { NodeProvider } from '@/api/controller'
import {
  CloudArrowDownIcon,
  PlusIcon,
  ArrowPathIcon,
  EyeIcon,
  PencilSquareIcon,
  TrashIcon,
} from '@heroicons/vue/24/outline'

defineProps<{
  providers: NodeProvider[]
  isRefreshing?: string | null
}>()

const emit = defineEmits<{
  (e: 'create'): void
  (e: 'edit', provider: NodeProvider): void
  (e: 'delete', id: string): void
  (e: 'refresh', id: string): void
  (e: 'inspect', tag: string): void
}>()

const { t } = useI18n()

const formatTime = (iso?: string) => {
  if (!iso) return ''
  const d = new Date(iso)
  return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
}
</script>
