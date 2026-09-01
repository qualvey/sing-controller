<template>
  <div class="border-base-content/10 bg-base-100 flex flex-col gap-2.5 rounded-lg border p-3 shadow-xs">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2 font-medium">
        <span class="badge badge-sm badge-neutral badge-outline font-mono text-[11px]">
          <HighlightText
            :text="inbound.type"
            :filter="inboundsFilter"
          />
        </span>
        <HighlightText
          :text="inbound.tag"
          :filter="inboundsFilter"
          class="font-semibold text-sm"
        />
      </div>
      <div class="flex items-center gap-1">
        <button
          class="btn btn-ghost btn-xs text-primary"
          @click="openEditInboundModal(inbound.tag)"
        >
          编辑
        </button>
        <button
          class="btn btn-ghost btn-xs text-error"
          @click="handleRemove"
        >
          删除
        </button>
      </div>
    </div>

    <div class="bg-base-200/40 flex flex-col gap-1.5 rounded-md p-2 text-xs">
      <div class="flex items-center justify-between">
        <span class="text-base-content/50">Listen:</span>
        <span class="font-mono text-[11px]">
          <HighlightText
            :text="inbound.listen ?? '—'"
            :filter="inboundsFilter"
          />
        </span>
      </div>

      <div class="flex items-center justify-between">
        <span class="text-base-content/50">Port:</span>
        <span class="badge badge-sm badge-ghost font-mono text-xs">
          <HighlightText
            :text="inbound.listen_port != null ? String(inbound.listen_port) : '—'"
            :filter="inboundsFilter"
          />
        </span>
      </div>

      <div
        v-if="inbound.method"
        class="flex items-center justify-between"
      >
        <span class="text-base-content/50">Method:</span>
        <span class="font-mono text-[11px] text-base-content/80">
          <HighlightText
            :text="inbound.method"
            :filter="inboundsFilter"
          />
        </span>
      </div>

      <!-- 用户列表 -->
      <div
        v-if="boundUsers.length > 0 || isUserSupportedType(inbound.type)"
        class="flex flex-col gap-1 pt-1 border-t border-base-content/5"
      >
        <span class="text-base-content/50 text-[11px]">{{ $t('users') || '授权用户' }}:</span>
        <div class="flex flex-wrap items-center gap-1.5">
          <template v-if="boundUsers.length > 0">
            <button
              v-for="u in boundUsers"
              :key="u.name"
              type="button"
              class="badge badge-sm badge-primary badge-outline hover:badge-primary cursor-pointer gap-1 transition-colors text-[11px]"
              :title="`编辑用户 ${u.name}`"
              @click.stop="openEditUserModal(u)"
            >
              <UserIcon class="h-3 w-3" />
              <HighlightText
                :text="u.name"
                :filter="inboundsFilter"
              />
            </button>
          </template>
          <span
            v-else
            class="text-base-content/30 text-[11px]"
          >
            暂无绑定用户
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Inbound, UserMeta } from '@/api'
import { deleteInbound, inboundsFilter, poolUsers } from '@/assembly/inbounds'
import { USER_TYPES } from '@/assembly/users'
import HighlightText from '@/components/common/HighlightText.vue'
import { openEditInboundModal } from '@/composables/inboundManagement'
import { openEditUserModal } from '@/composables/userManagement'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { UserIcon } from '@heroicons/vue/24/outline'
import { computed } from 'vue'

const props = defineProps<{
  inbound: Inbound
}>()

const isUserSupportedType = (type: string) => USER_TYPES.includes(type) || type === 'mixed'

const boundUsers = computed<UserMeta[]>(() => {
  const bound = poolUsers.value.filter((u) => u.bound_inbounds?.includes(props.inbound.tag))
  if (bound.length > 0) return bound

  if (props.inbound.type === 'mixed' && Array.isArray(props.inbound.users)) {
    return props.inbound.users
      .map((u) => {
        const item = (u ?? {}) as { username?: string }
        return item.username ? { name: item.username, bound_inbounds: [props.inbound.tag] } : null
      })
      .filter((u): u is UserMeta => !!u)
  }
  return []
})

const handleRemove = async () => {
  const { confirmed } = await showConfirmDialog({
    title: '删除确认',
    message: `确定删除 inbound「${props.inbound.tag}」？该操作不可恢复。`,
    confirmText: '删除',
    confirmButtonClass: 'btn-error',
  })
  if (!confirmed) return
  await deleteInbound(props.inbound.tag)
}
</script>

