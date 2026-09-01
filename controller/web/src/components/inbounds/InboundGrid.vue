<template>
  <CardGridContainer
    :title="$t('inbounds') || '入站配置'"
    :count="inbounds.length"
    :total-count="inbounds.length"
    :empty-text="$t('noInboundsTip') || '暂无入站配置'"
  >
    <template #actions>
      <button
        class="btn btn-primary btn-sm gap-1.5"
        @click="openCreateInboundModal"
      >
        <PlusIcon class="h-4 w-4" />
        {{ $t('createInbound') || '新建 Inbound' }}
      </button>
    </template>

    <template #empty-actions>
      <button
        class="btn btn-primary btn-sm gap-1.5"
        @click="openCreateInboundModal"
      >
        <PlusIcon class="h-4 w-4" />
        {{ $t('createInbound') || '新建 Inbound' }}
      </button>
    </template>

    <!-- 卡片列表项 -->
    <CardGridItem
      v-for="row in inbounds"
      :key="row.tag"
      :clickable="true"
      @click="openEditInboundModal(row.tag)"
    >
      <template #icon>
        <span class="badge badge-sm badge-neutral badge-outline font-mono text-[10px] uppercase tracking-wider shrink-0 mt-0.5">
          <HighlightText
            :text="row.type"
            :filter="inboundsFilter"
          />
        </span>
      </template>

      <template #title>
        <HighlightText
          :text="row.tag"
          :filter="inboundsFilter"
          class="font-semibold text-sm text-base-content/95"
        />
      </template>

      <template #subtitle>
        <div class="flex items-center gap-1.5 font-mono text-[11px] text-base-content/50">
          <span>{{ row.listen || '0.0.0.0' }}</span>
          <span
            v-if="row.method"
            class="text-base-content/40"
          >· {{ row.method }}</span>
        </div>
      </template>

      <template #badge>
        <span
          v-if="row.listen_port != null"
          class="badge badge-sm badge-neutral font-mono font-semibold text-xs px-2"
        >
          <HighlightText
            :text="String(row.listen_port)"
            :filter="inboundsFilter"
          />
        </span>
        <span
          v-else
          class="text-base-content/30 text-xs font-mono"
        >
          —
        </span>
      </template>

      <!-- 底部：已授权用户 Chips + 快捷操作 -->
      <template #tags>
        <div class="flex flex-wrap items-center gap-1 max-w-full">
          <template v-if="getBoundUsers(row).length > 0">
            <button
              v-for="u in getBoundUsers(row)"
              :key="u.name"
              type="button"
              class="badge badge-xs badge-primary badge-outline hover:badge-primary cursor-pointer gap-1 transition-colors text-[10px]"
              :title="`编辑用户 ${u.name}`"
              @click.stop="openEditUserModal(u)"
            >
              <UserIcon class="h-2.5 w-2.5" />
              <HighlightText
                :text="u.name"
                :filter="inboundsFilter"
              />
            </button>
          </template>
          <span
            v-else-if="isUserSupportedType(row.type)"
            class="text-base-content/30 text-[11px]"
          >
            未绑定用户
          </span>
        </div>
      </template>

      <template #actions>
        <button
          class="btn btn-ghost btn-xs text-primary h-7 px-2"
          @click.stop="openEditInboundModal(row.tag)"
        >
          {{ $t('edit') || '编辑' }}
        </button>
        <button
          class="btn btn-ghost btn-xs text-error h-7 px-2"
          @click.stop="handleRemove(row)"
        >
          {{ $t('delete') || '删除' }}
        </button>
      </template>
    </CardGridItem>
  </CardGridContainer>
</template>

<script setup lang="ts">
import type { Inbound, UserMeta } from '@/api'
import { deleteInbound, inboundsFilter, poolUsers } from '@/assembly/inbounds'
import { USER_TYPES } from '@/assembly/users'
import CardGridContainer from '@/components/common/card-grid/CardGridContainer.vue'
import CardGridItem from '@/components/common/card-grid/CardGridItem.vue'
import HighlightText from '@/components/common/HighlightText.vue'
import { openCreateInboundModal, openEditInboundModal } from '@/composables/inboundManagement'
import { openEditUserModal } from '@/composables/userManagement'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { PlusIcon, UserIcon } from '@heroicons/vue/24/outline'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{
  inbounds: Inbound[]
}>()

const isUserSupportedType = (type: string) => USER_TYPES.includes(type) || type === 'mixed'

const getBoundUsers = (row: Inbound): UserMeta[] => {
  const bound = poolUsers.value.filter((u) => u.bound_inbounds?.includes(row.tag))
  if (bound.length > 0) return bound

  if (row.type === 'mixed' && Array.isArray(row.users)) {
    return row.users
      .map((u) => {
        const item = (u ?? {}) as { username?: string }
        return item.username ? { name: item.username, bound_inbounds: [row.tag] } : null
      })
      .filter((u): u is UserMeta => !!u)
  }
  return []
}

const handleRemove = async (row: Inbound) => {
  const { confirmed } = await showConfirmDialog({
    title: t('confirmDelete') || '删除确认',
    message: `确定删除 inbound「${row.tag}」？该操作不可恢复。`,
    confirmText: t('delete') || '删除',
    confirmButtonClass: 'btn-error',
  })
  if (!confirmed) return
  await deleteInbound(row.tag)
}
</script>
