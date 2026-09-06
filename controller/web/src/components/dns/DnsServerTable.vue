<template>
  <div class="w-full">
    <!-- Table Container -->
    <div class="overflow-x-auto bg-base-100">
      <div class="min-w-[700px]">
        <!-- Column headers -->
        <div class="grid grid-cols-[100px_180px_1fr_140px_100px_60px] px-4 sm:px-5 h-11 items-center border-b border-base-content/10 bg-base-200/50 select-none">
          <span class="text-xs font-semibold text-base-content/60 tracking-wider uppercase">{{ $t('protocol') || '协议' }}</span>
          <span class="text-xs font-semibold text-base-content/60 tracking-wider uppercase">{{ $t('name') || '名称' }}</span>
          <span class="text-xs font-semibold text-base-content/60 tracking-wider uppercase">{{ $t('address') || '地址' }}</span>
          <span class="text-xs font-semibold text-base-content/60 tracking-wider uppercase">{{ $t('detour') || '出站代理' }}</span>
          <span class="text-xs font-semibold text-base-content/60 tracking-wider uppercase">{{ $t('features') || '特性' }}</span>
          <span class="text-xs font-semibold text-base-content/60 tracking-wider uppercase text-right pr-2">{{ $t('actions') || '操作' }}</span>
        </div>

        <!-- Rows -->
        <div
          v-if="servers.length === 0"
          class="py-16 text-center text-sm text-base-content/50"
        >
          {{ $t('noDnsServersTip') || '暂无匹配的 DNS 服务器' }}
        </div>

        <div
          v-else
          class="divide-y divide-base-content/5"
        >
          <div
            v-for="row in servers"
            :key="row.tag"
            class="group grid grid-cols-[100px_180px_1fr_140px_100px_60px] px-4 sm:px-5 h-13 sm:h-14 items-center hover:bg-base-200/50 transition-colors duration-150 cursor-pointer"
            @click="openEditServerModal(row)"
          >
            <!-- Protocol / Type -->
            <div>
              <ProtocolTag :protocol="row.type" />
            </div>

            <!-- Tag / Name -->
            <div class="text-sm font-semibold text-base-content overflow-hidden text-ellipsis whitespace-nowrap pr-2">
              <HighlightText
                :text="String(row.tag || '')"
                :filter="dnsFilter"
              />
            </div>

            <!-- Address / Details -->
            <div class="text-xs overflow-hidden text-ellipsis whitespace-nowrap pr-2 flex items-center gap-1.5">
              <template v-if="row.server">
                <span class="font-mono text-base-content/90">
                  <HighlightText
                    :text="String(row.server)"
                    :filter="dnsFilter"
                  />
                </span>
                <PortBadge
                  v-if="row.server_port"
                  :port="row.server_port"
                />
              </template>
              <template v-else-if="row.inet4_range || row.inet6_range">
                <span class="font-mono text-base-content/80">
                  <HighlightText
                    :text="[row.inet4_range, row.inet6_range].filter(Boolean).join(' ')"
                    :filter="dnsFilter"
                  />
                </span>
              </template>
              <template v-else-if="row.interface">
                <span class="font-mono text-base-content/80">
                  <HighlightText
                    :text="String(row.interface)"
                    :filter="dnsFilter"
                  />
                </span>
              </template>
              <template v-else-if="row.path">
                <span class="font-mono text-base-content/80">
                  <HighlightText
                    :text="String(row.path)"
                    :filter="dnsFilter"
                  />
                </span>
              </template>
              <span
                v-else
                class="text-base-content/30 font-mono select-none"
              >—</span>
            </div>

            <!-- Detour -->
            <div class="text-xs overflow-hidden text-ellipsis whitespace-nowrap pr-2">
              <template v-if="row.detour">
                <span class="inline-flex items-center text-xs font-medium bg-primary/10 text-primary border border-primary/20 px-2 py-0.5 rounded-md">
                  <HighlightText
                    :text="String(row.detour)"
                    :filter="dnsFilter"
                  />
                </span>
              </template>
              <span
                v-else
                class="text-base-content/30 font-mono select-none"
              >—</span>
            </div>

            <!-- Features / TLS -->
            <div class="flex items-center gap-1 overflow-hidden pr-2">
              <span
                v-if="row.tls?.enabled"
                class="inline-flex items-center text-[11px] font-semibold tracking-wider uppercase px-2 py-0.5 rounded-md border text-success bg-success/10 border-success/25 select-none leading-none"
              >
                TLS
              </span>
              <span
                v-if="row.domain_resolver?.client_subnet"
                class="inline-flex items-center text-[10px] font-mono px-1.5 py-0.5 rounded bg-base-200 text-base-content/70 select-none leading-none"
                title="Client Subnet (EDNS)"
              >
                ECS
              </span>
              <span
                v-if="!row.tls?.enabled && !row.domain_resolver?.client_subnet"
                class="text-base-content/30 text-xs font-mono select-none"
              >—</span>
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
import { deleteDnsServer, dnsFilter } from '@/assembly/dns'
import HighlightText from '@/components/common/HighlightText.vue'
import PortBadge from '@/components/common/PortBadge.vue'
import ProtocolTag from '@/components/common/ProtocolTag.vue'
import { openEditServerModal } from '@/composables/dnsManagement'
import { useTooltip } from '@/helper/tooltip'
import { TrashIcon } from '@heroicons/vue/24/outline'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const { showTip } = useTooltip()

defineProps<{
  servers: Array<Record<string, any>>
}>()

const handleRemove = async (row: Record<string, any>) => {
  await deleteDnsServer(String(row.tag))
}
</script>
