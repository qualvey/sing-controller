<template>
  <div class="border-base-content/10 bg-base-100 overflow-hidden rounded-lg border shadow-xs">
    <table class="table table-sm w-full">
      <thead>
        <tr class="border-base-content/10">
          <th class="w-[140px]">tag</th>
          <th class="w-24">type</th>
          <th>地址</th>
          <th class="w-16">TLS</th>
          <th class="w-[120px]">detour</th>
          <th class="w-[130px] text-right">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="row in servers"
          :key="row.tag"
          class="hover:bg-base-200/40 border-base-content/5 transition-colors"
        >
          <td>
            <span class="font-medium text-xs">{{ row.tag }}</span>
          </td>
          <td>
            <span class="badge badge-sm badge-neutral badge-outline font-mono text-[11px]">
              {{ row.type }}
            </span>
          </td>
          <td class="text-xs">
            <span v-if="row.server" class="font-mono">
              {{ row.server }}<template v-if="row.server_port">:{{ row.server_port }}</template>
            </span>
            <span v-else-if="row.inet4_range || row.inet6_range" class="font-mono text-base-content/80">
              {{ row.inet4_range || '' }} {{ row.inet6_range || '' }}
            </span>
            <span v-else class="text-base-content/40">—</span>
          </td>
          <td class="text-xs">
            <span v-if="row.tls?.enabled" class="text-success font-semibold">✓</span>
            <span v-else class="text-base-content/40">—</span>
          </td>
          <td class="text-xs">
            <span v-if="row.detour" class="badge badge-sm badge-ghost text-[11px]">{{ row.detour }}</span>
            <span v-else class="text-base-content/40">—</span>
          </td>
          <td class="text-right">
            <button
              class="btn btn-ghost btn-xs text-primary"
              @click="openEditServerModal(row)"
            >
              编辑
            </button>
            <button
              class="btn btn-ghost btn-xs text-error"
              @click="deleteDnsServer(String(row.tag))"
            >
              删除
            </button>
          </td>
        </tr>
      </tbody>
    </table>

    <div
      v-if="!servers.length"
      class="text-base-content/60 py-10 text-center text-sm"
    >
      暂无 DNS server
    </div>
  </div>
</template>

<script setup lang="ts">
import { deleteDnsServer } from '@/assembly/dns'
import { openEditServerModal } from '@/composables/dnsManagement'

defineProps<{
  servers: Array<Record<string, any>>
}>()
</script>
