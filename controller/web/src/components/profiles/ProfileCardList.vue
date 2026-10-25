<template>
  <div class="flex flex-col gap-4">
    <div class="flex items-center justify-between">
      <div class="text-xs text-base-content/60">
        {{ t('profiles.profilesDesc', '配置组将基础分流框架与选中的订阅节点片段动态合并，无感平滑切换。') }}
      </div>
      <button
        class="btn btn-primary btn-sm gap-1.5 shadow-sm"
        @click="emit('create')"
      >
        <PlusIcon class="size-4" />
        {{ t('profiles.createProfile', '新建方案') }}
      </button>
    </div>

    <!-- 方案列表 -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-3 max-h-[420px] overflow-y-auto pr-1">
      <div
        v-if="profiles.length === 0"
        class="col-span-full flex flex-col items-center justify-center py-12 text-base-content/40 gap-2 border border-dashed border-base-300 rounded-xl"
      >
        <RectangleGroupIcon class="size-8 opacity-40" />
        <span class="text-xs">{{ t('profiles.noProfilesTip', '暂无配置方案，点击右上角新建方案') }}</span>
      </div>

      <div
        v-for="prof in profiles"
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
          <span>{{ t('profiles.activeRunning', '当前方案') }}</span>
        </div>

        <div class="flex flex-col gap-2">
          <div class="flex items-center gap-2">
            <span class="font-bold text-sm text-base-content">{{ prof.name }}</span>
            <span class="badge badge-sm font-mono text-xs opacity-70">{{ prof.tag }}</span>
          </div>
          <p class="text-xs text-base-content/60 line-clamp-2 min-h-4">
            {{ prof.description || t('common.noDescription', '无描述') }}
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
              {{ t('profiles.pureBaseMode', '未挂载外部节点 (纯 Base)') }}
            </span>
          </div>
        </div>

        <!-- 操作按钮栏 -->
        <div class="flex items-center justify-between mt-4 pt-3 border-t border-base-300/40">
          <div class="flex items-center gap-1.5 text-xs text-base-content/50">
            <span :class="prof.auto_reload ? 'text-success' : 'text-base-content/40'">●</span>
            <span>{{ prof.auto_reload ? t('profiles.autoHotReload', '自动热重载') : t('profiles.manualRestart', '手动重启') }}</span>
          </div>

          <div class="flex items-center gap-1.5">
            <button
              v-if="!prof.is_active || allowReapply"
              class="btn btn-xs btn-primary gap-1"
              :disabled="isApplying === prof.id"
              @click="emit('apply', prof.id)"
            >
              <span
                v-if="isApplying === prof.id"
                class="loading loading-spinner loading-xs"
              />
              <CheckIcon
                v-else
                class="size-3.5"
              />
              <span>{{ allowReapply ? t(initializing ? 'profiles.initializeAndStart' : 'profiles.applyAndStart', initializing ? '初始化并启动' : '应用并启动') : t('profiles.applyThisProfile', '应用此方案') }}</span>
            </button>
            <button
              class="btn btn-xs btn-ghost btn-square"
              :title="t('common.edit', '编辑')"
              @click="emit('edit', prof)"
            >
              <PencilSquareIcon class="size-3.5" />
            </button>
            <button
              v-if="prof.id !== 'default'"
              class="btn btn-xs btn-ghost btn-square text-error hover:bg-error/10"
              :title="t('common.delete', '删除')"
              @click="emit('delete', prof.id)"
            >
              <TrashIcon class="size-3.5" />
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ConfigProfile } from '@/api/controller'
import {
  RectangleGroupIcon,
  PlusIcon,
  CheckCircleIcon,
  CheckIcon,
  PencilSquareIcon,
  TrashIcon,
} from '@heroicons/vue/24/outline'

defineProps<{
  profiles: ConfigProfile[]
  isApplying?: string | null
  allowReapply?: boolean
  initializing?: boolean
}>()

const emit = defineEmits<{
  (e: 'create'): void
  (e: 'edit', profile: ConfigProfile): void
  (e: 'delete', id: string): void
  (e: 'apply', id: string): void
}>()

const { t } = useI18n()
</script>
