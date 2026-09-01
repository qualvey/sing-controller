<template>
  <div class="w-full max-w-[780px] mx-auto font-[-apple-system,BlinkMacSystemFont,'PingFang_SC','Helvetica_Neue',sans-serif]">
    <!-- Header -->
    <div class="flex items-center gap-2.5 mb-1">
      <span class="text-base font-bold text-[#e8e8e8]">
        {{ $t('inbounds') || '入站' }}
      </span>
      <span
        class="text-[11px] font-semibold px-1.5 py-0.5 rounded-[4px] bg-[#1f1f22] text-[#6b6b75] border border-[#2e2e33] font-mono leading-none"
      >
        {{ inbounds.length }}
      </span>

      <div class="flex-1" />

      <!-- Search toggle button -->
      <button
        type="button"
        class="w-[30px] h-[30px] flex items-center justify-center rounded-[7px] cursor-pointer transition-all duration-150"
        :class="showSearch ? 'bg-[#1f1f22] border border-[#2e2e33] text-[#e8e8e8]' : 'bg-transparent border border-transparent text-[#6b6b75] hover:text-[#9898a4]'"
        :title="showSearch ? '关闭搜索' : '展开搜索'"
        @click="toggleSearch"
      >
        <SearchIcon />
      </button>

      <!-- 新建 Inbound 按钮 -->
      <button
        type="button"
        class="flex items-center gap-[5px] text-[12px] font-medium text-[#e8e8e8] bg-[#1f1f22] hover:bg-[#27272b] border border-[#2e2e33] hover:border-[#3e3e44] rounded-[7px] px-3 py-1.5 cursor-pointer transition-all duration-150 select-none shadow-xs"
        @click="openCreateInboundModal"
      >
        <PlusIcon />
        {{ $t('createInbound') || '新建 Inbound' }}
      </button>
    </div>

    <!-- Search input -->
    <div
      v-if="showSearch"
      class="mb-3 mt-2"
    >
      <input
        ref="searchInputRef"
        v-model="inboundsFilter"
        placeholder="搜索名称、协议、端口…"
        class="w-full px-3 py-[7px] text-[13px] bg-[#1f1f22] border border-[#2e2e33] focus:border-[#3e3e44] rounded-[7px] text-[#e8e8e8] outline-hidden placeholder:text-[#5a5a64] transition-colors"
      />
    </div>

    <!-- Table Container -->
    <div class="bg-[#1a1a1d] rounded-[10px] border border-[#2a2a2e] overflow-hidden mt-3 shadow-xs">
      <!-- Column headers -->
      <div class="grid grid-cols-[100px_1fr_160px_120px_100px] px-4 h-9 items-center border-b border-[#232326] bg-[#18181b] select-none">
        <span class="text-[11px] font-semibold text-[#5a5a64] tracking-[0.05em] uppercase">协议</span>
        <span class="text-[11px] font-semibold text-[#5a5a64] tracking-[0.05em] uppercase">名称</span>
        <span class="text-[11px] font-semibold text-[#5a5a64] tracking-[0.05em] uppercase">端口</span>
        <span class="text-[11px] font-semibold text-[#5a5a64] tracking-[0.05em] uppercase">绑定用户</span>
        <span class="text-[11px] font-semibold text-[#5a5a64] tracking-[0.05em] uppercase text-right"></span>
      </div>

      <!-- Rows -->
      <div
        v-if="inbounds.length === 0"
        class="py-10 text-center text-[13px] text-[#5a5a64]"
      >
        {{ $t('noInboundsTip') || '暂无匹配的入站配置' }}
      </div>

      <div
        v-else
        class="divide-y divide-[#232326]"
      >
        <div
          v-for="row in inbounds"
          :key="row.tag"
          class="group grid grid-cols-[100px_1fr_160px_120px_100px] px-4 h-12 items-center hover:bg-[#1f1f23] transition-colors duration-120 cursor-default"
          @click="openEditInboundModal(row.tag)"
        >
          <!-- Protocol -->
          <div>
            <ProtocolTag :protocol="row.type" />
          </div>

          <!-- Name -->
          <div class="text-[13px] font-semibold text-[#e8e8e8] overflow-hidden text-ellipsis whitespace-nowrap pr-2">
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
              class="text-[#4a4a54] text-xs font-mono"
            >—</span>
          </div>

          <!-- User -->
          <div class="text-[12px] overflow-hidden text-ellipsis whitespace-nowrap pr-2">
            <template v-if="getBoundUsers(row).length > 0">
              <span class="text-[#9898a4]">
                {{ getBoundUsers(row).map((u) => u.name).join(', ') }}
              </span>
            </template>
            <span
              v-else
              class="text-[#4a4a54]"
            >
              未绑定
            </span>
          </div>

          <!-- Actions (Hover 渐显) -->
          <div
            class="flex gap-1.5 justify-end opacity-100 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity duration-120"
            @click.stop
          >
            <button
              type="button"
              class="text-[11px] font-medium text-[#9898a4] hover:text-[#e8e8e8] bg-[#26262a] hover:bg-[#2e2e34] border border-[#323238] hover:border-[#3e3e46] rounded-[5px] px-2.5 py-[3px] leading-[1.4] cursor-pointer transition-all duration-120"
              @click.stop="openEditInboundModal(row.tag)"
            >
              edit
            </button>
            <button
              type="button"
              class="text-[11px] font-medium text-[#e53535] bg-[rgba(229,53,53,0.08)] hover:bg-[rgba(229,53,53,0.16)] border border-[rgba(229,53,53,0.2)] hover:border-[rgba(229,53,53,0.35)] rounded-[5px] px-2.5 py-[3px] leading-[1.4] cursor-pointer transition-all duration-120"
              @click.stop="handleRemove(row)"
            >
              删除
            </button>
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
import { openCreateInboundModal, openEditInboundModal } from '@/composables/inboundManagement'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{
  inbounds: Inbound[]
}>()

const showSearch = ref(false)
const searchInputRef = ref<HTMLInputElement | null>(null)

const toggleSearch = () => {
  showSearch.value = !showSearch.value
  if (showSearch.value) {
    nextTick(() => {
      searchInputRef.value?.focus()
    })
  } else {
    inboundsFilter.value = ''
  }
}

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

// PlusIcon svg 组件（与 Figma 一致）
const PlusIcon = {
  template: `
    <svg width="12" height="12" viewBox="0 0 12 12" fill="none" class="shrink-0">
      <path d="M6 1v10M1 6h10" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
    </svg>
  `,
}

// SearchIcon svg 组件（与 Figma 一致）
const SearchIcon = {
  template: `
    <svg width="13" height="13" viewBox="0 0 13 13" fill="none" class="shrink-0">
      <circle cx="5.5" cy="5.5" r="4" stroke="currentColor" stroke-width="1.4" />
      <path d="M8.5 8.5L11 11" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
    </svg>
  `,
}
</script>
