<template>
  <div class="w-full">
    <!-- Table Container -->
    <div class="overflow-x-auto bg-base-100">
      <div class="min-w-[600px]">
        <!-- Column headers -->
        <div class="grid grid-cols-[100px_200px_110px_1fr_60px] px-4 sm:px-5 h-11 items-center border-b border-base-content/10 bg-base-200/50 select-none">
          <span class="text-xs font-semibold text-base-content/60 tracking-wider uppercase">{{ $t('protocol') || '协议' }}</span>
          <span class="text-xs font-semibold text-base-content/60 tracking-wider uppercase">{{ $t('name') || '名称' }}</span>
          <span class="text-xs font-semibold text-base-content/60 tracking-wider uppercase">{{ $t('port') || '端口' }}</span>
          <span class="text-xs font-semibold text-base-content/60 tracking-wider uppercase">{{ $t('boundUsers') || '绑定用户' }}</span>
          <span class="text-xs font-semibold text-base-content/60 tracking-wider uppercase text-right pr-2">{{ $t('actions') || '操作' }}</span>
        </div>

        <!-- Rows -->
        <div
          v-if="inbounds.length === 0"
          class="py-16 text-center text-sm text-base-content/50"
        >
          {{ $t('noInboundsTip') || '暂无匹配的入站配置' }}
        </div>

        <div
          v-else
          class="divide-y divide-base-content/5"
        >
          <div
            v-for="row in inbounds"
            :key="row.tag"
            class="group grid grid-cols-[100px_200px_110px_1fr_60px] px-4 sm:px-5 h-13 sm:h-14 items-center hover:bg-base-200/50 transition-colors duration-150 cursor-pointer"
            @click="openEditInboundModal(row.tag)"
          >
            <!-- Protocol -->
            <div>
              <ProtocolTag :protocol="row.type" />
            </div>

            <!-- Name -->
            <div class="text-sm font-semibold text-base-content overflow-hidden text-ellipsis whitespace-nowrap pr-2">
              <HighlightText
                :text="row.tag"
                :filter="inboundsFilter"
              />
            </div>

            <!-- Port -->
            <div>
              <PortBadge
                v-if="row.listen_port != null"
                :port="row.listen_port"
              />
              <span
                v-else
                class="text-base-content/30 text-xs font-mono"
              >—</span>
            </div>

            <!-- User -->
            <div class="text-sm overflow-hidden text-ellipsis whitespace-nowrap pr-2">
              <template v-if="getBoundUsers(row).length > 0">
                <span class="inline-flex items-center text-xs font-medium bg-primary/10 text-primary border border-primary/20 px-2 py-0.5 rounded-md">
                  {{ getBoundUsers(row).map((u) => u.name).join(', ') }}
                </span>
              </template>
              <span
                v-else
                class="text-base-content/30 font-mono text-sm select-none"
              >
                —
              </span>
            </div>

            <!-- Action (Delete with tooltip) -->
            <div
              class="flex items-center justify-end pr-1"
              @click.stop
            >
              <button
                type="button"
                class="btn btn-ghost btn-circle btn-sm text-base-content/30 hover:text-error hover:bg-error/10 transition-colors"
                :title="$t('delete') || '删除'"
                @mouseenter="showTip($event, $t('delete') || '删除')"
                @click.stop="handleRemove(row)"
              >
                <TrashIcon class="size-4.5" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Inbound, UserMeta } from '@/api'
import { deleteInbound, inboundsFilter, poolUsers } from '@/assembly/inbounds'
import HighlightText from '@/components/common/HighlightText.vue'
import PortBadge from '@/components/common/PortBadge.vue'
import ProtocolTag from '@/components/common/ProtocolTag.vue'
import { openEditInboundModal } from '@/composables/inboundManagement'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { useTooltip } from '@/helper/tooltip'
import { TrashIcon } from '@heroicons/vue/24/outline'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const { showTip } = useTooltip()

defineProps<{
  inbounds: Inbound[]
}>()

const getBoundUsers = (row: Inbound): UserMeta[] | null => {
  const bound = poolUsers.value.filter((u) => u.bound_inbounds?.includes(row.tag))
  if (bound.length > 0) return bound

  if (Array.isArray(row.users)) {
    return row.users
      .map((u) => {
        const item = (u ?? {}) as { username?: string; Username?: string; name?: string }
        const name = item.username || item.Username || item.name
        return name ? { name, bound_inbounds: [row.tag] } : null
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


