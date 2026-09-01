<template>
  <DialogWrapper
    v-model="isUserDrawerOpen"
    :title="$t('manageUsers') || '用户池管理'"
    box-class="max-w-[980px] w-full max-h-[85vh] flex flex-col"
    @update:model-value="(val) => (!val ? closeUserDrawer() : null)"
  >
    <div class="flex flex-col gap-4">
      <!-- 头部操作与搜索栏 -->
      <div class="flex flex-wrap items-center justify-between gap-2 border-b border-base-content/10 pb-3">
        <div class="flex flex-1 items-center gap-2 min-w-[240px]">
          <TextInput
            v-model="usersFilter"
            class="w-full max-w-sm"
            :placeholder="$t('searchUsers') || '搜索用户名 / UUID / 绑定入站...'"
            :clearable="true"
          />
          <button
            class="btn btn-ghost btn-sm btn-circle"
            :disabled="loading"
            :title="$t('refresh') || '刷新'"
            @click="fetchUsers"
          >
            <ArrowPathIcon
              class="h-4 w-4"
              :class="{ 'animate-spin': loading }"
            />
          </button>
        </div>

        <div class="flex items-center gap-2">
          <span class="text-xs text-base-content/60 font-mono">
            {{ renderUsers.length }} / {{ users.length }} {{ $t('users') || '用户' }}
          </span>
          <button
            class="btn btn-primary btn-sm gap-1.5"
            @click="openCreateUserModal"
          >
            <PlusIcon class="h-4 w-4" />
            {{ $t('createUser') || '新建用户' }}
          </button>
        </div>
      </div>

      <!-- 加载中 -->
      <div
        v-if="loading && !users.length"
        class="text-base-content/50 py-12 text-center text-sm"
      >
        <span class="loading loading-spinner loading-md text-primary mb-2 block mx-auto"></span>
        {{ $t('loading') || '加载中…' }}
      </div>

      <!-- 用户列表呈现 -->
      <template v-else>
        <!-- 桌面端表格 -->
        <div
          v-if="!isMiddleScreen"
          class="overflow-x-auto"
        >
          <UserTable :users="renderUsers" />
        </div>

        <!-- 移动端卡片 -->
        <div
          v-else
          class="flex flex-col gap-2.5 max-h-[60vh] overflow-y-auto"
        >
          <UserCard
            v-for="u in renderUsers"
            :key="u.name"
            :user="u"
          />
          <div
            v-if="!renderUsers.length"
            class="text-base-content/50 border-base-content/10 bg-base-100 rounded-lg border py-8 text-center text-sm"
          >
            {{ $t('noMatchingUsers') || '暂无匹配的用户' }}
          </div>
        </div>
      </template>
    </div>
  </DialogWrapper>
</template>

<script setup lang="ts">
import { fetchUsers, loading, renderUsers, users, usersFilter } from '@/assembly/users'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import TextInput from '@/components/common/TextInput.vue'
import UserCard from '@/components/users/UserCard.vue'
import UserTable from '@/components/users/UserTable.vue'
import {
  closeUserDrawer,
  isUserDrawerOpen,
  openCreateUserModal,
} from '@/composables/userManagement'
import { isMiddleScreen } from '@/helper/utils'
import { ArrowPathIcon, PlusIcon } from '@heroicons/vue/24/outline'
import { watch } from 'vue'

watch(isUserDrawerOpen, (isOpen) => {
  if (isOpen) {
    void fetchUsers()
  }
})
</script>
