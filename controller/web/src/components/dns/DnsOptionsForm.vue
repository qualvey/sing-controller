<template>
  <div class="border-base-content/10 bg-base-100 rounded-lg border p-5 shadow-xs">
    <div class="text-base-content/90 mb-4 font-semibold text-sm">
      DNS 全局选项（options）
    </div>

    <div
      class="grid max-w-[620px] gap-x-4 gap-y-5 text-sm"
      style="grid-template-columns: 180px minmax(0, 1fr)"
    >
      <label class="text-base-content/70 pt-2">final（默认 server）</label>
      <SelectField
        v-model="optsForm.final"
        :options="dnsTags"
        placeholder="选择默认 DNS server（留空则不指定）"
      />

      <label class="text-base-content/70 pt-2">strategy（解析策略）</label>
      <SelectField
        v-model="optsForm.strategy"
        :options="STRATEGY_OPTIONS"
        placeholder="不指定"
      />

      <label class="text-base-content/70 pt-2">timeout</label>
      <input
        v-model="optsForm.timeout"
        type="text"
        class="input input-bordered input-sm w-full"
        placeholder="如 5s（留空用默认）"
      />

      <label class="text-base-content/70 pt-2">disable_cache</label>
      <div class="flex items-center gap-2">
        <Switch v-model="optsForm.disable_cache" />
        <span class="text-base-content/50 text-xs">禁用 DNS 缓存</span>
      </div>

      <label class="text-base-content/70 pt-2">independent_cache</label>
      <div class="flex items-center gap-2">
        <Switch v-model="optsForm.independent_cache" class="self-center" />
        <span class="text-base-content/50 text-xs">为每个 DNS server 独立缓存</span>
      </div>

      <label class="text-base-content/70 pt-2">reverse_mapping</label>
      <div class="flex items-center gap-2">
        <Switch v-model="optsForm.reverse_mapping" />
        <span class="text-base-content/50 text-xs">IP 反查域名</span>
      </div>

      <span />
      <button
        type="button"
        class="btn btn-primary btn-sm w-fit"
        :disabled="saving"
        @click="saveDnsOptions"
      >
        保存 DNS 选项
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  dnsTags,
  optsForm,
  saveDnsOptions,
  saving,
  STRATEGY_OPTIONS,
} from '@/assembly/dns'
import { SelectField } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
</script>
