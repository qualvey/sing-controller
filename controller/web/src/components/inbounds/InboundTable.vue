<template>
  <div class="border-base-content/10 bg-base-100 overflow-hidden rounded-lg border shadow-xs">
    <table class="table table-sm w-full">
      <thead>
        <tr class="border-base-content/10">
          <th class="w-[140px]">类型</th>
          <th class="w-[180px]">tag</th>
          <th>listen</th>
          <th class="w-[120px]">listen_port</th>
          <th class="w-[220px]">method</th>
          <th class="w-[130px] text-right">操作</th>
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
              编辑
            </button>
            <button
              class="btn btn-ghost btn-xs text-error"
              @click="handleRemove(row)"
            >
              删除
            </button>
          </td>
        </tr>
      </tbody>
    </table>
    <div
      v-if="!inbounds.length"
      class="text-base-content/60 py-10 text-center text-sm"
    >
      暂无 Inbound。点击右上角「新建 Inbound」进行添加。
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Inbound } from '@/api'
import { deleteInbound, inboundsFilter } from '@/assembly/inbounds'
import HighlightText from '@/components/common/HighlightText.vue'
import { openEditInboundModal } from '@/composables/inboundManagement'
import { showConfirmDialog } from '@/helper/confirmDialog'

defineProps<{
  inbounds: Inbound[]
}>()

const handleRemove = async (row: Inbound) => {
  const { confirmed } = await showConfirmDialog({
    title: '删除确认',
    message: `确定删除 inbound「${row.tag}」？该操作不可恢复。`,
    confirmText: '删除',
    confirmButtonClass: 'btn-error',
  })
  if (!confirmed) return
  await deleteInbound(row.tag)
}
</script>
