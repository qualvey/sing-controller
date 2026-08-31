<template>
  <CtrlsBar>
    <div
      v-if="!isLargeCtrlsBar"
      class="flex flex-col gap-2 p-2"
    >
      <div class="flex items-center gap-2">
        <SegmentedControl
          v-model="inboundsTabShow"
          :options="tabOptions"
        />
        <div class="flex-1"></div>
        <button
          v-if="inboundsTabShow === 'form'"
          class="btn btn-circle btn-sm btn-primary"
          title="新建 Inbound"
          @click="openCreateInboundModal"
        >
          <PlusIcon class="h-4 w-4" />
        </button>
        <button
          class="btn btn-circle btn-sm"
          :disabled="loading"
          title="刷新"
          @click="fetchInbounds"
        >
          <ArrowPathIcon
            class="h-4 w-4"
            :class="{ 'animate-spin': loading }"
          />
        </button>
      </div>

      <div
        v-if="inboundsTabShow === 'form'"
        class="flex w-full items-center gap-2"
      >
        <TextInput
          v-model="inboundsFilter"
          class="w-full flex-1"
          placeholder="搜索 Inbound / tag / 端口 | Regex"
          :clearable="true"
        />
      </div>
      <div
        v-if="inboundsTabShow === 'form'"
        class="text-base-content/60 flex items-center justify-between px-1 text-xs"
      >
        <span>{{ inbounds.length }} 个入站</span>
      </div>
    </div>

    <div
      v-else
      class="flex flex-wrap items-center gap-2 p-2"
    >
      <SegmentedControl
        v-model="inboundsTabShow"
        :options="tabOptions"
      />
      <TextInput
        v-if="inboundsTabShow === 'form'"
        v-model="inboundsFilter"
        class="w-80"
        placeholder="搜索 Inbound / tag / 端口 | Regex"
        :clearable="true"
      />
      <button
        v-if="inboundsTabShow === 'form'"
        class="btn btn-primary btn-sm gap-1.5"
        @click="openCreateInboundModal"
      >
        <PlusIcon class="h-4 w-4" />
        新建 Inbound
      </button>
      <button
        class="btn btn-ghost btn-sm gap-1.5"
        :disabled="loading"
        @click="fetchInbounds"
      >
        <ArrowPathIcon
          class="h-4 w-4"
          :class="{ 'animate-spin': loading }"
        />
        刷新
      </button>
      <div class="flex-1"></div>
      <div
        v-if="inboundsTabShow === 'form'"
        class="text-base-content/60 flex items-center gap-2 text-xs"
      >
        <span>{{ inbounds.length }} 个入站</span>
      </div>
    </div>
  </CtrlsBar>
</template>

<script setup lang="ts">
import {
  fetchInbounds,
  inbounds,
  inboundsFilter,
  inboundsTabShow,
  loading,
} from '@/assembly/inbounds'
import CtrlsBar from '@/components/common/CtrlsBar.vue'
import SegmentedControl, { type SegmentOption } from '@/components/common/SegmentedControl.vue'
import TextInput from '@/components/common/TextInput.vue'
import { useCtrlsBar } from '@/composables/useCtrlsBar'
import { openCreateInboundModal } from '@/composables/inboundManagement'
import { ArrowPathIcon, PlusIcon } from '@heroicons/vue/24/outline'
import { computed } from 'vue'

const { isLargeCtrlsBar } = useCtrlsBar()

const tabOptions = computed<SegmentOption[]>(() => [
  { value: 'form', label: '表单', count: inbounds.value.length },
  { value: 'source', label: '源码' },
])
</script>
