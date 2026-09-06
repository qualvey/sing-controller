<template>
  <div class="w-full">
    <!-- 空状态 -->
    <div
      v-if="!users.length"
      class="py-16 text-center text-sm text-base-content/50"
    >
      {{ $t('noMatchingUsers') || '暂无匹配的用户' }}
    </div>

    <!-- 卡片网格列表 -->
    <div
      v-else
      class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4"
    >
      <div
        v-for="u in users"
        :key="u.name"
        class="group relative flex flex-col justify-between rounded-xl border border-base-content/10 bg-base-100 p-4 transition-all duration-150 hover:border-base-content/25 hover:shadow-md cursor-pointer select-none"
        @click="openEditUserModal(u)"
      >
        <div>
          <!-- 头部区域：用户头像 + 用户名 + 绑定数胶囊 -->
          <div class="flex items-start justify-between gap-2">
            <div class="flex items-center gap-2.5 min-w-0 flex-1">
              <div class="size-9 rounded-xl bg-primary/10 text-primary flex items-center justify-center shrink-0">
                <UserIcon class="size-4.5" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="text-base font-semibold text-base-content tracking-tight truncate">
                  <HighlightText
                    :text="u.name"
                    :filter="usersFilter"
                  />
                </div>
              </div>
            </div>

            <!-- 右侧入站数量徽章 -->
            <div class="shrink-0">
              <span
                v-if="(u.bound_inbounds || []).length"
                class="inline-flex items-center text-xs font-semibold px-2.5 py-0.5 rounded-full bg-primary/10 text-primary border border-primary/20 leading-tight"
              >
                {{ (u.bound_inbounds || []).length }} Inbounds
              </span>
              <span
                v-else
                class="inline-flex items-center text-xs font-medium px-2 py-0.5 rounded-full bg-base-200 text-base-content/50 leading-tight"
              >
                未绑定
              </span>
            </div>
          </div>

          <!-- 核心元数据（UUID / Password / Flow） -->
          <div class="mt-3 flex flex-col gap-2 rounded-lg bg-base-200/50 p-2.5 text-xs">
            <!-- UUID -->
            <div
              v-if="u.uuid"
              class="flex items-center justify-between gap-2 font-mono text-[11px]"
            >
              <div class="flex items-center gap-1.5 min-w-0 flex-1">
                <span class="text-base-content/50 font-medium select-none shrink-0">UUID:</span>
                <span
                  class="truncate text-base-content/85 select-all"
                  :title="u.uuid"
                >
                  <HighlightText
                    :text="u.uuid"
                    :filter="usersFilter"
                  />
                </span>
              </div>
              <button
                type="button"
                class="btn btn-ghost btn-circle btn-xs text-base-content/40 hover:text-base-content shrink-0"
                title="复制 UUID"
                @click.stop="copyText(u.uuid, 'UUID')"
              >
                <ClipboardDocumentIcon class="size-3.5" />
              </button>
            </div>

            <!-- Password -->
            <div
              v-if="u.password"
              class="flex items-center justify-between gap-2 font-mono text-[11px]"
            >
              <div class="flex items-center gap-1.5 min-w-0 flex-1">
                <span class="text-base-content/50 font-medium select-none shrink-0">PWD:</span>
                <span class="text-base-content/85 select-all font-mono">
                  {{ showPasswords[u.name] ? u.password : '••••••••' }}
                </span>
              </div>
              <div
                class="flex items-center gap-0.5 shrink-0"
                @click.stop
              >
                <button
                  type="button"
                  class="btn btn-ghost btn-circle btn-xs text-base-content/40 hover:text-base-content"
                  :title="showPasswords[u.name] ? '隐藏密码' : '显示明文'"
                  @click="togglePassword(u.name)"
                >
                  <EyeSlashIcon
                    v-if="showPasswords[u.name]"
                    class="size-3.5"
                  />
                  <EyeIcon
                    v-else
                    class="size-3.5"
                  />
                </button>
                <button
                  type="button"
                  class="btn btn-ghost btn-circle btn-xs text-base-content/40 hover:text-base-content"
                  title="复制密码"
                  @click="copyText(u.password, '密码')"
                >
                  <ClipboardDocumentIcon class="size-3.5" />
                </button>
              </div>
            </div>

            <!-- Flow -->
            <div
              v-if="u.flow"
              class="flex items-center justify-between gap-2 text-[11px]"
            >
              <span class="text-base-content/50 font-medium select-none">Flow:</span>
              <span class="inline-flex items-center text-[10px] font-mono px-2 py-0.5 rounded-md bg-base-300/60 text-base-content/80 border border-base-content/10">
                {{ u.flow }}
              </span>
            </div>
          </div>

          <!-- 绑定入站列表区域 -->
          <div class="mt-3 flex flex-col gap-1.5">
            <div class="flex items-center justify-between text-[11px] text-base-content/50 font-medium select-none">
              <span>绑定入站</span>
              <span>{{ (u.bound_inbounds || []).length }} 个</span>
            </div>
            <div class="flex flex-wrap items-center gap-1.5 min-h-6">
              <template v-if="(u.bound_inbounds || []).length">
                <span
                  v-for="t in u.bound_inbounds"
                  :key="t"
                  class="inline-flex items-center text-[11px] font-mono font-medium px-2 py-0.5 rounded-md bg-primary/10 text-primary border border-primary/20 leading-tight select-none"
                >
                  <HighlightText
                    :text="t"
                    :filter="usersFilter"
                  />
                </span>
              </template>
              <span
                v-else
                class="text-xs text-base-content/30 italic select-none"
              >
                未绑定到任何入站
              </span>
            </div>
          </div>
        </div>

        <!-- 底部独立操作栏 -->
        <div
          class="mt-4 flex items-center justify-end gap-1.5 border-t border-base-content/10 pt-3"
          @click.stop
        >
          <button
            type="button"
            class="btn btn-ghost btn-xs text-primary gap-1"
            @click="openEditUserModal(u)"
          >
            <PencilSquareIcon class="size-3.5" />
            {{ $t('edit') || '编辑' }}
          </button>
          <button
            type="button"
            class="btn btn-ghost btn-xs text-error hover:bg-error/10 gap-1"
            @click="handleRemove(u)"
          >
            <TrashIcon class="size-3.5" />
            {{ $t('delete') || '删除' }}
          </button>
        </div>
      </div>
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
  PencilSquareIcon,
  TrashIcon,
  UserIcon,
} from '@heroicons/vue/24/outline'
import { useClipboard } from '@vueuse/core'
import { reactive } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

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
    title: t('delete') || '删除用户',
    message: `删除用户「${u.name}」？将同时从其绑定的所有入站中移除。`,
    confirmText: t('delete') || '删除',
    confirmButtonClass: 'btn-error',
  })
  if (!confirmed) return
  await deleteUser(u.name)
}
</script>
