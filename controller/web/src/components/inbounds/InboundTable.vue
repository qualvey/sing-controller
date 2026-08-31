<template>
  <div class="border-base-content/10 bg-base-100 overflow-hidden rounded-lg border shadow-xs">
    <!-- 头部工具栏：标题与新建按钮 -->
    <div class="flex items-center justify-between border-b border-base-content/10 px-4 py-3">
      <div class="flex items-center gap-2">
        <span class="text-sm font-semibold text-base-content/90">{{ $t('inbounds') }}</span>
        <span class="badge badge-sm badge-neutral badge-outline font-mono">{{ inbounds.length }}</span>
      </div>
      <button
        class="btn btn-primary btn-sm gap-1.5"
        @click="openCreateInboundModal"
      >
        <PlusIcon class="h-4 w-4" />
        {{ $t('createInbound') }}
      </button>
    </div>

    <table
      v-if="inbounds.length"
      class="table table-sm w-full"
    >
      <thead>
        <tr class="border-base-content/10 text-xs">
          <th class="w-[140px]">{{ $t('type') }}</th>
          <th class="w-[180px]">tag</th>
          <th>{{ $t('listen') }}</th>
          <th class="w-[120px]">{{ $t('listenPort') }}</th>
          <th class="w-[220px]">{{ $t('method') }}</th>
          <th class="w-[130px] text-right">{{ $t('actions') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="row in inbounds"
          :key="row.tag"
          class="hover:bg-base-200/40 border-base-content/5 transition-colors"
        >
          <td>
            <span class="badge badge-sm badge-neutral badge-outline font-mono text-[11px]">
              <HighlightText
                :text="row.type"
                :filter="inboundsFilter"
              />
            </span>
          </td>
          <td>
            <span class="font-medium text-xs">
              <HighlightText
                :text="row.tag"
                :filter="inboundsFilter"
              />
            </span>
          </td>
          <td>
            <span class="font-mono text-xs">
              <HighlightText
                :text="row.listen ?? '—'"
                :filter="inboundsFilter"
              />
            </span>
          </td>
          <td>
            <span
              v-if="row.listen_port != null"
              class="badge badge-sm badge-ghost font-mono text-xs"
            >
              <HighlightText
                :text="String(row.listen_port)"
                :filter="inboundsFilter"
              />
            </span>
            <span
              v-else
              class="text-base-content/30 text-xs"
            >
              —
            </span>
          </td>
          <td>
            <span
              v-if="row.method"
              class="font-mono text-xs text-base-content/80"
            >
              <HighlightText
                :text="row.method"
                :filter="inboundsFilter"
              />
            </span>
            <span
              v-else
              class="text-base-content/30 text-xs"
            >
              —
            </span>
          </td>
          <td class="text-right">
            <button
              class="btn btn-ghost btn-xs text-primary"
              @click="openEditInboundModal(row.tag)"
            >
              {{ $t('edit') }}
            </button>
            <button
              class="btn btn-ghost btn-xs text-error"
              @click="handleRemove(row)"
            >
              {{ $t('delete') }}
            </button>
          </td>
        </tr>
      </tbody>
    </table>

    <div
      v-else
      class="text-base-content/60 flex flex-col items-center justify-center py-12 text-center text-sm"
    >
      <p class="mb-3">{{ $t('noInboundsTip') }}</p>
      <button
        class="btn btn-primary btn-sm gap-1.5"
        @click="openCreateInboundModal"
      >
        <PlusIcon class="h-4 w-4" />
        {{ $t('createInbound') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Inbound } from '@/api'
import { deleteInbound, inboundsFilter } from '@/assembly/inbounds'
import HighlightText from '@/components/common/HighlightText.vue'
import { openCreateInboundModal, openEditInboundModal } from '@/composables/inboundManagement'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { PlusIcon } from '@heroicons/vue/24/outline'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{
  inbounds: Inbound[]
}>()

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
