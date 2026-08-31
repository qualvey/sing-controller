<template>
  <CtrlsBar>
    <div
      v-if="!isLargeCtrlsBar"
      class="flex flex-col gap-2 p-2"
    >
      <div class="flex items-center gap-2">
        <SegmentedControl
          v-model="dnsTabShow"
          :options="tabOptions"
        />
        <div class="flex-1"></div>
        <button
          v-if="dnsTabShow === 'servers'"
          class="btn btn-circle btn-sm btn-primary"
          title="新建 Server"
          @click="openCreateServerModal"
        >
          <PlusIcon class="h-4 w-4" />
        </button>
        <button
          v-else-if="dnsTabShow === 'rules'"
          class="btn btn-circle btn-sm btn-primary"
          title="新建规则"
          @click="openCreateRuleModal"
        >
          <PlusIcon class="h-4 w-4" />
        </button>
        <button
          class="btn btn-circle btn-sm"
          :disabled="loading"
          title="刷新"
          @click="fetchDns"
        >
          <ArrowPathIcon
            class="h-4 w-4"
            :class="{ 'animate-spin': loading }"
          />
        </button>
      </div>

      <div
        v-if="dnsTabShow === 'servers' || dnsTabShow === 'rules'"
        class="flex w-full items-center gap-2"
      >
        <TextInput
          v-model="dnsFilter"
          class="w-full flex-1"
          :placeholder="dnsTabShow === 'servers' ? '搜索 DNS Server / tag / 地址 | Regex' : '搜索 DNS 规则 / 域名 / tag | Regex'"
          :clearable="true"
        />
      </div>
    </div>

    <div
      v-else
      class="flex flex-wrap items-center gap-2 p-2"
    >
      <SegmentedControl
        v-model="dnsTabShow"
        :options="tabOptions"
      />
      <TextInput
        v-if="dnsTabShow === 'servers' || dnsTabShow === 'rules'"
        v-model="dnsFilter"
        class="w-80"
        :placeholder="dnsTabShow === 'servers' ? '搜索 DNS Server / tag / 地址 | Regex' : '搜索 DNS 规则 / 域名 / tag | Regex'"
        :clearable="true"
      />
      <button
        v-if="dnsTabShow === 'servers'"
        class="btn btn-primary btn-sm gap-1.5"
        @click="openCreateServerModal"
      >
        <PlusIcon class="h-4 w-4" />
        新建 Server
      </button>
      <button
        v-else-if="dnsTabShow === 'rules'"
        class="btn btn-primary btn-sm gap-1.5"
        @click="openCreateRuleModal"
      >
        <PlusIcon class="h-4 w-4" />
        新建规则
      </button>
      <button
        class="btn btn-ghost btn-sm gap-1.5"
        :disabled="loading"
        @click="fetchDns"
      >
        <ArrowPathIcon
          class="h-4 w-4"
          :class="{ 'animate-spin': loading }"
        />
        刷新
      </button>
      <div class="flex-1"></div>
      <div
        v-if="dnsTabShow === 'servers'"
        class="text-base-content/60 flex items-center gap-2 text-xs"
      >
        <span>{{ servers.length }} 个 Server</span>
      </div>
      <div
        v-else-if="dnsTabShow === 'rules'"
        class="text-base-content/60 flex items-center gap-2 text-xs"
      >
        <span>{{ rules.length }} 条规则</span>
      </div>
    </div>
  </CtrlsBar>
</template>

<script setup lang="ts">
import {
  dnsFilter,
  dnsTabShow,
  fetchDns,
  loading,
  rules,
  servers,
} from '@/assembly/dns'
import CtrlsBar from '@/components/common/CtrlsBar.vue'
import SegmentedControl, { type SegmentOption } from '@/components/common/SegmentedControl.vue'
import TextInput from '@/components/common/TextInput.vue'
import { useCtrlsBar } from '@/composables/useCtrlsBar'
import { openCreateRuleModal, openCreateServerModal } from '@/composables/dnsManagement'
import { ArrowPathIcon, PlusIcon } from '@heroicons/vue/24/outline'
import { computed } from 'vue'

const { isLargeCtrlsBar } = useCtrlsBar()

const tabOptions = computed<SegmentOption[]>(() => [
  { value: 'servers', label: 'Servers', count: servers.value.length },
  { value: 'rules', label: '规则', count: rules.value.length },
  { value: 'options', label: '选项' },
  { value: 'source', label: '源码' },
])
</script>
