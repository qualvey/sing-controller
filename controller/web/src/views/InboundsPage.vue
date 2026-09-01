<template>
  <div class="relative size-full overflow-x-hidden">
    <InboundsCtrl />

    <div
      class="p-3"
      :style="padding"
    >
      <!-- 入站表格视图 -->
      <template v-if="inboundsTabShow === 'form'">
        <div
          v-if="loading && !inbounds.length"
          class="text-base-content/50 py-16 text-center text-sm"
        >
          <span class="loading loading-spinner loading-md text-primary mb-2 block mx-auto"></span>
          {{ $t('loading') || '加载中…' }}
        </div>

        <template v-else>
          <div class="base-container">
            <InboundTable :inbounds="renderInbounds" />
          </div>
        </template>
      </template>

      <!-- 用户管理卡片网格视图 -->
      <template v-else-if="inboundsTabShow === 'users'">
        <div class="base-container">
          <UserGrid :users="renderUsers" />
        </div>
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

    <!-- 用户新建/编辑单条数据弹窗 -->
    <UserEditDialog />
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
import { fetchUsers, renderUsers } from '@/assembly/users'
import InboundsCtrl from '@/components/controls/InboundsCtrl.vue'
import CertManager from '@/components/inbounds/cert/CertManager.vue'
import InboundEditDialog from '@/components/inbounds/InboundEditDialog.vue'
import InboundTable from '@/components/inbounds/InboundTable.vue'
import SourcePane from '@/components/SourcePane.vue'
import UserEditDialog from '@/components/users/UserEditDialog.vue'
import UserGrid from '@/components/users/UserGrid.vue'
import { usePaddingForViews } from '@/composables/paddingViews'

import { onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'


const route = useRoute()

const { padding } = usePaddingForViews({
  offsetTop: 0,
  offsetBottom: 8,
})

watch(
  () => route.query.tab,
  (tab) => {
    if (tab && typeof tab === 'string') {
      inboundsTabShow.value = tab
    }
  },
)

onMounted(async () => {
  if (route.query.tab && typeof route.query.tab === 'string') {
    inboundsTabShow.value = route.query.tab
  }
  await Promise.all([
    fetchInbounds(),
    fetchInboundTypes(),
    fetchPoolUsers(),
    fetchUsers(),
    fetchCertProviders(),
  ])
})
</script>



