<template>
  <div class="border-base-content/10 bg-base-100 overflow-hidden rounded-lg border shadow-xs">
    <table class="table table-sm w-full">
      <thead>
        <tr class="border-base-content/10">
          <th class="w-14">#</th>
          <th>规则</th>
          <th class="w-[180px]">动作</th>
          <th class="w-[130px] text-right">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(row, idx) in rules"
          :key="row.id"
          class="hover:bg-base-200/40 border-base-content/5 transition-colors"
        >
          <td class="text-base-content/50 text-xs tabular-nums">{{ idx + 1 }}</td>
          <td>
            <div class="flex flex-col gap-0.5">
              <template v-if="ruleSummary(row.rule).items.length">
                <span
                  v-for="it in ruleSummary(row.rule).items"
                  :key="it.k"
                  class="text-[13px]"
                >
                  <b class="text-base-content/50 mr-1 font-medium">{{ it.k }}</b>
                  <span class="text-base-content/90">{{ it.v }}</span>
                </span>
                <span
                  v-if="ruleSummary(row.rule).otherCount"
                  class="text-base-content/40 text-[13px]"
                >
                  +{{ ruleSummary(row.rule).otherCount }} 项
                </span>
              </template>
              <span
                v-else
                class="text-base-content/40 text-[13px]"
              >
                全部（无匹配条件）
              </span>
              <span
                v-if="row.rule.invert"
                class="badge badge-warning badge-outline w-fit text-xs font-semibold"
              >
                取反
              </span>
            </div>
          </td>
          <td class="text-xs">
            <span class="badge badge-sm badge-ghost text-[11px]">
              {{ ruleActionText(row.rule) }}
            </span>
          </td>
          <td class="text-right">
            <button
              class="btn btn-ghost btn-xs text-primary"
              @click="openEditRuleModal(row)"
            >
              编辑
            </button>
            <button
              class="btn btn-ghost btn-xs text-error"
              @click="deleteDnsRule(row.id)"
            >
              删除
            </button>
          </td>
        </tr>
      </tbody>
    </table>

    <div
      v-if="!rules.length"
      class="text-base-content/60 py-10 text-center text-sm"
    >
      暂无 DNS 规则
    </div>
  </div>
</template>

<script setup lang="ts">
import { deleteDnsRule } from '@/assembly/dns'
import { openEditRuleModal } from '@/composables/dnsManagement'
import { DNS_RULE_FIELDS, DNS_RULE_SUMMARY_ORDER } from '@/views/dnsRuleFields'

defineProps<{
  rules: Array<{ id: string; rule: Record<string, any> }>
}>()

function fmtList(v: unknown): string {
  if (Array.isArray(v)) return v.join(', ')
  return v == null ? '—' : String(v)
}

function ruleSummary(r: Record<string, any>): {
  items: Array<{ k: string; v: string }>
  otherCount: number
} {
  if (r.type === 'logical') {
    const count = Array.isArray(r.rules) ? r.rules.length : 0
    return {
      items: [{ k: 'logical', v: `${r.mode || 'and'} · ${count} 条子规则` }],
      otherCount: 0,
    }
  }
  const items = DNS_RULE_SUMMARY_ORDER.filter((k) => r[k] != null).map((k) => ({
    k,
    v: fmtList(r[k]),
  }))
  const otherCount = DNS_RULE_FIELDS.filter(
    (f) => !DNS_RULE_SUMMARY_ORDER.includes(f.key) && r[f.key] != null,
  ).length
  return { items, otherCount }
}

function ruleActionText(r: Record<string, any>): string {
  const action = typeof r.action === 'string' && r.action ? r.action : 'route'
  if (action === 'route') {
    return r.server
      ? `→ ${String(r.server)}`
      : r.outbound
        ? `→ ${fmtList(r.outbound)}`
        : '→ (未指定 server)'
  }
  return action
}
</script>
