<template>
  <div class="relative size-full overflow-x-hidden">
    <DnsCtrl />

    <div
      class="p-3"
      :style="padding"
    >
      <div
        v-if="loading && !servers.length && !rules.length"
        class="text-base-content/50 py-16 text-center text-sm"
      >
        <span class="loading loading-spinner loading-md text-primary mb-2 block mx-auto"></span>
        加载 DNS 配置中…
      </div>

      <template v-else>
        <!-- Servers 视图 -->
        <template v-if="dnsTabShow === 'servers'">
          <div class="base-container">
            <DnsServerTable :servers="renderServers" />
          </div>
        </template>

        <!-- 规则视图 -->
        <template v-else-if="dnsTabShow === 'rules'">
          <div class="base-container">
            <DnsRuleTable :rules="renderRules" />
          </div>
        </template>

        <!-- 选项视图 -->
        <template v-else-if="dnsTabShow === 'options'">
          <div class="base-container">
            <DnsOptionsForm />
          </div>
        </template>

        <!-- 源码视图 -->
        <template v-else-if="dnsTabShow === 'source'">
          <div class="base-container">
            <SourcePane
              segment="dns"
              @saved="fetchDns"
            />
          </div>
        </template>
      </template>
    </div>

    <!-- 独立弹窗 -->
    <DnsServerDialog />
    <DnsRuleDialog />
  </div>
</template>

<script setup lang="ts">
import {
  dnsTabShow,
  fetchDns,
  fetchTags,
  loading,
  renderRules,
  renderServers,
  rules,
  servers,
} from '@/assembly/dns'
import DnsCtrl from '@/components/controls/DnsCtrl.vue'
import DnsOptionsForm from '@/components/dns/DnsOptionsForm.vue'
import DnsRuleDialog from '@/components/dns/DnsRuleDialog.vue'
import DnsRuleTable from '@/components/dns/DnsRuleTable.vue'
import DnsServerDialog from '@/components/dns/DnsServerDialog.vue'
import DnsServerTable from '@/components/dns/DnsServerTable.vue'
import SourcePane from '@/components/SourcePane.vue'
import { usePaddingForViews } from '@/composables/paddingViews'
import { onMounted } from 'vue'

const { padding } = usePaddingForViews({
  offsetTop: 0,
  offsetBottom: 8,
})

onMounted(async () => {
  await Promise.all([fetchDns(), fetchTags()])
})
</script>
