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
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Inbound } from '@/api'
import { deleteInbound, inboundsFilter } from '@/assembly/inbounds'
import HighlightText from '@/components/common/HighlightText.vue'
import { openEditInboundModal } from '@/composables/inboundManagement'
import { showConfirmDialog } from '@/helper/confirmDialog'

const props = defineProps<{
  inbound: Inbound
}>()

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
