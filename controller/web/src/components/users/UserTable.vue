<template>
  <div class="border-base-content/10 bg-base-100 overflow-hidden rounded-lg border shadow-xs">
    <!-- 头部工具栏：标题与新建按钮 -->
    <div class="flex items-center justify-between border-b border-base-content/10 px-4 py-3">
      <div class="flex items-center gap-2">
        <span class="text-sm font-semibold text-base-content/90">{{ $t('users') || '用户' }}</span>
        <span class="badge badge-sm badge-neutral badge-outline font-mono">{{ users.length }}</span>
      </div>
      <button
        class="btn btn-primary btn-sm gap-1.5"
        @click="openCreateUserModal"
      >
        <PlusIcon class="h-4 w-4" />
        {{ $t('createUser') || '新建用户' }}
      </button>
    </div>

    <table class="table table-sm w-full">

      <thead>
        <tr class="border-base-content/10">
          <th class="w-[160px]">name</th>
          <th class="w-[300px]">uuid</th>
          <th class="w-[160px]">password</th>
          <th class="w-[120px]">flow</th>
          <th>绑定入站</th>
          <th class="w-[140px] text-right">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="u in users"
          :key="u.name"
          class="hover:bg-base-200/40 border-base-content/5 transition-colors"
        >
          <td>
            <span class="flex items-center gap-1.5 font-medium">
              <UserIcon class="text-base-content/50 h-4 w-4 shrink-0" />
              <HighlightText
                :text="u.name"
                :filter="usersFilter"
              />
            </span>
          </td>
          <td>
            <div class="flex items-center gap-1.5">
              <span
                v-if="u.uuid"
                class="font-mono text-xs"
              >
                <HighlightText
                  :text="u.uuid"
                  :filter="usersFilter"
                />
              </span>
              <span
                v-else
                class="text-base-content/30 text-xs"
              >
                —
              </span>
              <button
                v-if="u.uuid"
                class="btn btn-ghost btn-xs text-base-content/50 hover:text-base-content"
                title="复制 UUID"
                @click="copyText(u.uuid, 'UUID')"
              >
                <ClipboardDocumentIcon class="h-3.5 w-3.5" />
              </button>
            </div>
          </td>
          <td>
            <div class="flex items-center gap-1.5">
              <span
                v-if="u.password"
                class="font-mono text-xs"
              >
                {{ showPasswords[u.name] ? u.password : '••••••' }}
              </span>
              <span
                v-else
                class="text-base-content/30 text-xs"
              >
                —
              </span>
              <template v-if="u.password">
                <button
                  class="btn btn-ghost btn-xs text-base-content/50 hover:text-base-content"
                  :title="showPasswords[u.name] ? '隐藏密码' : '显示密码'"
                  @click="togglePassword(u.name)"
                >
                  <EyeSlashIcon
                    v-if="showPasswords[u.name]"
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
                  @click="copyText(u.password, '密码')"
                >
                  <ClipboardDocumentIcon class="h-3.5 w-3.5" />
                </button>
              </template>
            </div>
          </td>
          <td>
            <span
              v-if="u.flow"
              class="badge badge-sm badge-neutral badge-outline font-mono text-[11px]"
            >
              <HighlightText
                :text="u.flow"
                :filter="usersFilter"
              />
            </span>
            <span
              v-else
              class="text-base-content/30 text-xs"
            >
              —
            </span>
          </td>
          <td>
            <div class="flex flex-wrap gap-1">
              <span
                v-for="t in u.bound_inbounds || []"
                :key="t"
                class="badge badge-sm badge-primary badge-outline"
              >
                <HighlightText
                  :text="t"
                  :filter="usersFilter"
                />
              </span>
              <span
                v-if="!(u.bound_inbounds || []).length"
                class="text-base-content/40 text-xs"
              >
                未绑定
              </span>
            </div>
          </td>
          <td class="text-right">
            <button
              class="btn btn-ghost btn-xs text-primary"
              @click="openEditUserModal(u)"
            >
              编辑
            </button>
            <button
              class="btn btn-ghost btn-xs text-error"
              @click="handleRemove(u)"
            >
              删除
            </button>
          </td>
        </tr>
      </tbody>
    </table>
    <div
      v-if="!users.length"
      class="text-base-content/60 py-10 text-center text-sm"
    >
      暂无匹配的用户。新建用户后可绑定到多个入站（vless/tuic 等），自动同步到各入站的 users[]
    </div>
  </div>
</template>

<script setup lang="ts">
import type { UserMeta } from '@/api'
import { deleteUser, usersFilter } from '@/assembly/users'
import HighlightText from '@/components/common/HighlightText.vue'
import { openCreateUserModal, openEditUserModal } from '@/composables/userManagement'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { showToast } from '@/helper/toast'
import {
  ClipboardDocumentIcon,
  EyeIcon,
  EyeSlashIcon,
  PlusIcon,
  UserIcon,
} from '@heroicons/vue/24/outline'

import { useClipboard } from '@vueuse/core'
import { reactive } from 'vue'

defineProps<{
  users: UserMeta[]
}>()

const showPasswords = reactive<Record<string, boolean>>({})
const { copy } = useClipboard()

const togglePassword = (name: string) => {
  showPasswords[name] = !showPasswords[name]
}

const copyText = async (text: string, label: string) => {
  await copy(text)
  showToast(`已复制 ${label}`, 'success')
}

const handleRemove = async (u: UserMeta) => {
  const { confirmed } = await showConfirmDialog({
    title: '删除用户',
    message: `删除用户 ${u.name}？将从其绑定的所有入站移除`,
    confirmButtonClass: 'btn-error',
  })
  if (!confirmed) return
  await deleteUser(u.name)
}
</script>
