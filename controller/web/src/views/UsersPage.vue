<template>
  <div class="relative size-full overflow-x-hidden">
    <UsersCtrl />

    <div
      class="p-3"
      :style="padding"
    >
      <div
        v-if="loading && !users.length"
        class="text-base-content/50 py-16 text-center text-sm"
      >
        <span class="loading loading-spinner loading-md text-primary mb-2 block mx-auto"></span>
        加载用户列表中…
      </div>

      <template v-else>
        <!-- 桌面端表格呈现 -->
        <div
          v-if="!isMiddleScreen"
          class="base-container"
        >
          <UserTable :users="renderUsers" />
        </div>

        <!-- 移动端卡片列表呈现 -->
        <div
          v-else
          class="flex flex-col gap-3"
        >
          <UserCard
            v-for="u in renderUsers"
            :key="u.name"
            :user="u"
          />
          <div
            v-if="!renderUsers.length"
            class="text-base-content/50 border-base-content/10 bg-base-100 rounded-lg border py-10 text-center text-sm"
          >
            暂无匹配的用户
          </div>
        </div>
      </template>
    </div>

    <!-- 用户创建/编辑弹窗 -->
    <UserEditDialog />
  </div>
</template>

<script setup lang="ts">
import { fetchUsers, loading, renderUsers, users } from '@/assembly/users'
import UsersCtrl from '@/components/controls/UsersCtrl.vue'
import UserCard from '@/components/users/UserCard.vue'
import UserEditDialog from '@/components/users/UserEditDialog.vue'
import UserTable from '@/components/users/UserTable.vue'
import { usePaddingForViews } from '@/composables/paddingViews'
import { isMiddleScreen } from '@/helper/utils'
import { onMounted } from 'vue'

const { padding } = usePaddingForViews({
  offsetTop: 0,
  offsetBottom: 8,
})

onMounted(() => {
  void fetchUsers()
})
</script>
