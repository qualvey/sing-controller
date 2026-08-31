<template>
  <CtrlsBar>
    <div
      v-if="!isLargeCtrlsBar"
      class="flex flex-col gap-2 p-2"
    >
      <div class="flex w-full items-center gap-2">
        <TextInput
          v-model="usersFilter"
          class="w-32 flex-1"
          placeholder="搜索用户 / UUID / 入站 | Regex"
          :clearable="true"
        />
        <button
          class="btn btn-circle btn-sm btn-primary"
          title="新建用户"
          @click="openCreateUserModal"
        >
          <PlusIcon class="h-4 w-4" />
        </button>
        <button
          class="btn btn-circle btn-sm"
          :disabled="loading"
          title="刷新"
          @click="fetchUsers"
        >
          <ArrowPathIcon
            class="h-4 w-4"
            :class="{ 'animate-spin': loading }"
          />
        </button>
      </div>
      <div class="text-base-content/60 flex items-center justify-between text-xs px-1">
        <span>{{ users.length }} 个用户</span>
        <span>可绑定入站：{{ bindableInbounds.length }} 个</span>
      </div>
    </div>
    <div
      v-else
      class="flex flex-wrap items-center gap-2 p-2"
    >
      <TextInput
        v-model="usersFilter"
        class="w-80"
        placeholder="搜索用户 / UUID / 入站 | Regex"
        :clearable="true"
      />
      <button
        class="btn btn-primary btn-sm gap-1.5"
        @click="openCreateUserModal"
      >
        <PlusIcon class="h-4 w-4" />
        新建用户
      </button>
      <button
        class="btn btn-ghost btn-sm gap-1.5"
        :disabled="loading"
        @click="fetchUsers"
      >
        <ArrowPathIcon
          class="h-4 w-4"
          :class="{ 'animate-spin': loading }"
        />
        刷新
      </button>
      <div class="flex-1"></div>
      <div class="text-base-content/60 flex items-center gap-2 text-xs">
        <span>{{ users.length }} 个用户</span>
        <span>·</span>
        <span>可绑定入站：{{ bindableInbounds.length }} 个 (vless/vmess/tuic/hysteria2 等)</span>
      </div>
    </div>
  </CtrlsBar>
</template>

<script setup lang="ts">
import {
  bindableInbounds,
  fetchUsers,
  loading,
  users,
  usersFilter,
} from '@/assembly/users'
import CtrlsBar from '@/components/common/CtrlsBar.vue'
import TextInput from '@/components/common/TextInput.vue'
import { useCtrlsBar } from '@/composables/useCtrlsBar'
import { openCreateUserModal } from '@/composables/userManagement'
import { ArrowPathIcon, PlusIcon } from '@heroicons/vue/24/outline'

const { isLargeCtrlsBar } = useCtrlsBar()
</script>
