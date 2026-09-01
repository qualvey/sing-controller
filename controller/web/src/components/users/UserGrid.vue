<template>
  <CardGridContainer
    :title="$t('users') || '用户池管理'"
    :count="users.length"
    :total-count="users.length"
    :empty-text="$t('noMatchingUsers') || '暂无用户数据'"
  >
    <template #actions>
      <button
        class="btn btn-primary btn-sm gap-1.5"
        @click="openCreateUserModal"
      >
        <PlusIcon class="h-4 w-4" />
        {{ $t('createUser') || '新建用户' }}
      </button>
    </template>

    <template #empty-actions>
      <button
        class="btn btn-primary btn-sm gap-1.5"
        @click="openCreateUserModal"
      >
        <PlusIcon class="h-4 w-4" />
        {{ $t('createUser') || '新建用户' }}
      </button>
    </template>

    <!-- 卡片列表项 -->
    <CardGridItem
      v-for="u in users"
      :key="u.name"
      :clickable="true"
      @click="openEditUserModal(u)"
    >
      <template #icon>
        <div class="h-7 w-7 rounded-lg bg-primary/10 text-primary flex items-center justify-center shrink-0 mt-0.5">
          <UserIcon class="h-4 w-4" />
        </div>
      </template>

      <template #title>
        <HighlightText
          :text="u.name"
          :filter="usersFilter"
          class="font-semibold text-sm text-base-content/95"
        />
      </template>

      <template #subtitle>
        <div class="flex flex-col gap-0.5 mt-0.5 text-xs">
          <!-- UUID -->
          <div
            v-if="u.uuid"
            class="flex items-center gap-1 font-mono text-[11px] text-base-content/60"
          >
            <span class="text-base-content/40">UUID:</span>
            <span class="truncate max-w-[140px]">{{ u.uuid }}</span>
            <button
              type="button"
              class="hover:text-base-content text-base-content/40 transition-colors p-0.5"
              title="复制 UUID"
              @click.stop="copyText(u.uuid, 'UUID')"
            >
              <ClipboardDocumentIcon class="h-3 w-3" />
            </button>
          </div>

          <!-- Password -->
          <div
            v-if="u.password"
            class="flex items-center gap-1 font-mono text-[11px] text-base-content/60"
          >
            <span class="text-base-content/40">PWD:</span>
            <span>{{ showPasswords[u.name] ? u.password : '••••••' }}</span>
            <button
              type="button"
              class="hover:text-base-content text-base-content/40 transition-colors p-0.5"
              :title="showPasswords[u.name] ? '隐藏密码' : '显示密码'"
              @click.stop="togglePassword(u.name)"
            >
              <EyeSlashIcon
                v-if="showPasswords[u.name]"
                class="h-3 w-3"
              />
              <EyeIcon
                v-else
                class="h-3 w-3"
              />
            </button>
            <button
              type="button"
              class="hover:text-base-content text-base-content/40 transition-colors p-0.5"
              title="复制密码"
              @click.stop="copyText(u.password, '密码')"
            >
              <ClipboardDocumentIcon class="h-3 w-3" />
            </button>
          </div>

          <!-- Flow -->
          <div
            v-if="u.flow"
            class="flex items-center gap-1 text-[11px] text-base-content/50"
          >
            <span class="text-base-content/40">Flow:</span>
            <span class="badge badge-xs badge-neutral badge-outline font-mono text-[10px]">{{ u.flow }}</span>
          </div>
        </div>
      </template>

      <template #badge>
        <span class="badge badge-sm badge-neutral font-mono font-medium text-[11px] px-2">
          {{ (u.bound_inbounds || []).length }} Inbounds
        </span>
      </template>

      <!-- 底部：绑定入站 Tags + 操作按钮 -->
      <template #tags>
        <div class="flex flex-wrap items-center gap-1 max-w-full">
          <span
            v-for="t in u.bound_inbounds || []"
            :key="t"
            class="badge badge-xs badge-primary badge-outline text-[10px] font-mono"
          >
            <HighlightText
              :text="t"
              :filter="usersFilter"
            />
          </span>
          <span
            v-if="!(u.bound_inbounds || []).length"
            class="text-base-content/30 text-[11px]"
          >
            未绑定入站
          </span>
        </div>
      </template>

      <template #actions>
        <button
          class="btn btn-ghost btn-xs text-primary h-7 px-2"
          @click.stop="openEditUserModal(u)"
        >
          {{ $t('edit') || '编辑' }}
        </button>
        <button
          class="btn btn-ghost btn-xs text-error h-7 px-2"
          @click.stop="handleRemove(u)"
        >
          {{ $t('delete') || '删除' }}
        </button>
      </template>
    </CardGridItem>
  </CardGridContainer>
</template>

<script setup lang="ts">
import type { UserMeta } from '@/api'
import { deleteUser, usersFilter } from '@/assembly/users'
import CardGridContainer from '@/components/common/card-grid/CardGridContainer.vue'
import CardGridItem from '@/components/common/card-grid/CardGridItem.vue'
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
