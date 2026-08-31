<template>
  <div class="border-base-content/10 bg-base-100 flex flex-col gap-2.5 rounded-lg border p-3 shadow-xs">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2 font-medium">
        <div class="bg-primary/10 text-primary flex h-7 w-7 items-center justify-center rounded-full">
          <UserIcon class="h-4 w-4" />
        </div>
        <HighlightText
          :text="user.name"
          :filter="usersFilter"
          class="font-semibold text-sm"
        />
        <span
          v-if="user.flow"
          class="badge badge-sm badge-neutral badge-outline font-mono text-[10px]"
        >
          <HighlightText
            :text="user.flow"
            :filter="usersFilter"
          />
        </span>
      </div>
      <div class="flex items-center gap-1">
        <button
          class="btn btn-ghost btn-xs text-primary"
          @click="openEditUserModal(user)"
        >
          编辑
        </button>
        <button
          class="btn btn-ghost btn-xs text-error"
          @click="handleRemove"
        >
          删除
        </button>
      </div>
    </div>

    <div class="bg-base-200/40 flex flex-col gap-1.5 rounded-md p-2 text-xs">
      <div
        v-if="user.uuid"
        class="flex items-center justify-between"
      >
        <span class="text-base-content/50">UUID:</span>
        <div class="flex items-center gap-1">
          <span class="font-mono text-[11px] truncate max-w-[200px]">
            <HighlightText
              :text="user.uuid"
              :filter="usersFilter"
            />
          </span>
          <button
            class="btn btn-ghost btn-xs text-base-content/50 hover:text-base-content"
            title="复制 UUID"
            @click="copyText(user.uuid, 'UUID')"
          >
            <ClipboardDocumentIcon class="h-3.5 w-3.5" />
          </button>
        </div>
      </div>

      <div
        v-if="user.password"
        class="flex items-center justify-between"
      >
        <span class="text-base-content/50">Password:</span>
        <div class="flex items-center gap-1">
          <span class="font-mono text-[11px]">
            {{ showPassword ? user.password : '••••••' }}
          </span>
          <button
            class="btn btn-ghost btn-xs text-base-content/50 hover:text-base-content"
            :title="showPassword ? '隐藏密码' : '显示密码'"
            @click="showPassword = !showPassword"
          >
            <EyeSlashIcon
              v-if="showPassword"
              class="h-3.5 w-3.5"
            />
            <EyeIcon
              v-else
              class="h-3.5 w-3.5"
            />
          </button>
          <button
            class="btn btn-ghost btn-xs text-base-content/50 hover:text-base-content"
            title="复制密码"
            @click="copyText(user.password, '密码')"
          >
            <ClipboardDocumentIcon class="h-3.5 w-3.5" />
          </button>
        </div>
      </div>
    </div>

    <div class="flex flex-wrap items-center gap-1 pt-1">
      <span class="text-base-content/50 text-xs mr-1">绑定入站:</span>
      <span
        v-for="t in user.bound_inbounds || []"
        :key="t"
        class="badge badge-sm badge-primary badge-outline text-[11px]"
      >
        <HighlightText
          :text="t"
          :filter="usersFilter"
        />
      </span>
      <span
        v-if="!(user.bound_inbounds || []).length"
        class="text-base-content/40 text-xs"
      >
        未绑定
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { UserMeta } from '@/api'
import { deleteUser, usersFilter } from '@/assembly/users'
import HighlightText from '@/components/common/HighlightText.vue'
import { openEditUserModal } from '@/composables/userManagement'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { showToast } from '@/helper/toast'
import {
  ClipboardDocumentIcon,
  EyeIcon,
  EyeSlashIcon,
  UserIcon,
} from '@heroicons/vue/24/outline'
import { useClipboard } from '@vueuse/core'
import { ref } from 'vue'

const props = defineProps<{
  user: UserMeta
}>()

const showPassword = ref(false)
const { copy } = useClipboard()

const copyText = async (text: string, label: string) => {
  await copy(text)
  showToast(`已复制 ${label}`, 'success')
}

const handleRemove = async () => {
  const { confirmed } = await showConfirmDialog({
    title: '删除用户',
    message: `删除用户 ${props.user.name}？将从其绑定的所有入站移除`,
    confirmButtonClass: 'btn-error',
  })
  if (!confirmed) return
  await deleteUser(props.user.name)
}
</script>
