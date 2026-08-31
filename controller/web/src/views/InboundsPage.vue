<template>
  <div class="relative size-full overflow-x-hidden">
    <InboundsCtrl />

    <div
      class="p-3"
      :style="padding"
    >
      <!-- 入站表单视图 -->
      <template v-if="inboundsTabShow === 'form'">
        <div
          v-if="loading && !inbounds.length"
          class="text-base-content/50 py-16 text-center text-sm"
        >
          <span class="loading loading-spinner loading-md text-primary mb-2 block mx-auto"></span>
          加载 Inbound 列表中…
        </div>

        <template v-else>
          <!-- 桌面端表格呈现 -->
          <div
            v-if="!isMiddleScreen"
            class="base-container"
          >
            <InboundTable :inbounds="renderInbounds" />
          </div>

          <!-- 移动端卡片列表呈现 -->
          <div
            v-else
            class="flex flex-col gap-3"
          >
            <InboundCard
              v-for="row in renderInbounds"
              :key="row.tag"
              :inbound="row"
            />
            <div
              v-if="!renderInbounds.length"
              class="text-base-content/50 border-base-content/10 bg-base-100 rounded-lg border py-10 text-center text-sm"
            >
              暂无匹配的 Inbound
            </div>
          </div>
        </template>
      </template>

      <!-- 证书管理视图 -->
      <template v-else-if="inboundsTabShow === 'cert'">
        <div class="base-container">
          <CertManager />
        </div>
      </template>

      <!-- 源码视图 -->
      <template v-else-if="inboundsTabShow === 'source'">
        <div class="base-container">
          <SourcePane
            segment="inbounds"
            @saved="fetchInbounds"
          />
        </div>
      </template>
    </div>

    <!-- 新建/编辑 Inbound 弹窗 -->
    <InboundEditDialog />
  </div>
</template>

<script setup lang="ts">
import {
  fetchCertProviders,
  fetchInbounds,
  fetchInboundTypes,
  fetchPoolUsers,
  inbounds,
  inboundsTabShow,
  loading,
  renderInbounds,
} from '@/assembly/inbounds'
import InboundsCtrl from '@/components/controls/InboundsCtrl.vue'
import CertManager from '@/components/inbounds/cert/CertManager.vue'
import InboundCard from '@/components/inbounds/InboundCard.vue'
import InboundEditDialog from '@/components/inbounds/InboundEditDialog.vue'
import InboundTable from '@/components/inbounds/InboundTable.vue'
import SourcePane from '@/components/SourcePane.vue'
import { usePaddingForViews } from '@/composables/paddingViews'
import { isMiddleScreen } from '@/helper/utils'
import { onMounted } from 'vue'

const { padding } = usePaddingForViews({
  offsetTop: 0,
  offsetBottom: 8,
})

onMounted(async () => {
  await Promise.all([
    fetchInbounds(),
    fetchInboundTypes(),
    fetchPoolUsers(),
    fetchCertProviders(),
  ])
})
</script>
