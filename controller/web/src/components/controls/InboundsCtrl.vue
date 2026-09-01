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
          class="btn btn-circle btn-sm"
          :disabled="loading || usersLoading"
          :title="$t('refresh')"
          @click="handleRefresh"
        >
          <ArrowPathIcon
            class="h-4 w-4"
            :class="{ 'animate-spin': loading || usersLoading }"
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
          :placeholder="$t('searchInbounds')"
          :clearable="true"
        />
      </div>
      <div
        v-else-if="inboundsTabShow === 'users'"
        class="flex w-full items-center gap-2"
      >
        <TextInput
          v-model="usersFilter"
          class="w-full flex-1"
          :placeholder="$t('searchUsers') || '搜索用户名 / UUID / 绑定入站 | Regex'"
          :clearable="true"
        />
      </div>

      <div
        v-if="inboundsTabShow === 'form'"
        class="text-base-content/60 flex items-center justify-between px-1 text-xs"
      >
        <span>{{ inbounds.length }} {{ $t('inbounds') }}</span>
      </div>
      <div
        v-else-if="inboundsTabShow === 'users'"
        class="text-base-content/60 flex items-center justify-between px-1 text-xs"
      >
        <span>{{ users.length }} {{ $t('users') }}</span>
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
        :placeholder="$t('searchInbounds')"
        :clearable="true"
      />
      <TextInput
        v-else-if="inboundsTabShow === 'users'"
        v-model="usersFilter"
        class="w-80"
        :placeholder="$t('searchUsers') || '搜索用户名 / UUID / 绑定入站 | Regex'"
        :clearable="true"
      />
      <button
        class="btn btn-ghost btn-sm gap-1.5"
        :disabled="loading || usersLoading"
        @click="handleRefresh"
      >
        <ArrowPathIcon
          class="h-4 w-4"
          :class="{ 'animate-spin': loading || usersLoading }"
        />
        {{ $t('refresh') }}
      </button>
      <div class="flex-1"></div>
      <div
        v-if="inboundsTabShow === 'form'"
        class="text-base-content/60 flex items-center gap-2 text-xs"
      >
        <span>{{ inbounds.length }} {{ $t('inbounds') }}</span>
      </div>
      <div
        v-else-if="inboundsTabShow === 'users'"
        class="text-base-content/60 flex items-center gap-2 text-xs"
      >
        <span>{{ users.length }} {{ $t('users') }}</span>
      </div>
    </div>
  </CtrlsBar>
</template>

<script setup lang="ts">
import {
  fetchInbounds,
  fetchPoolUsers,
  inbounds,
  inboundsFilter,
  inboundsTabShow,
  loading,
  poolUsers,
} from '@/assembly/inbounds'
import { fetchUsers, loading as usersLoading, users, usersFilter } from '@/assembly/users'
import CtrlsBar from '@/components/common/CtrlsBar.vue'
import SegmentedControl, { type SegmentOption } from '@/components/common/SegmentedControl.vue'
import TextInput from '@/components/common/TextInput.vue'
import { useCtrlsBar } from '@/composables/useCtrlsBar'
import { ArrowPathIcon } from '@heroicons/vue/24/outline'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const { isLargeCtrlsBar } = useCtrlsBar()

const tabOptions = computed<SegmentOption[]>(() => [
  { value: 'form', label: t('inbounds'), count: inbounds.value.length },
  { value: 'users', label: t('users') || '用户', count: poolUsers.value.length || users.value.length },
  { value: 'cert', label: t('cert') || '证书' },
  { value: 'source', label: t('source') },
])

const handleRefresh = async () => {
  if (inboundsTabShow.value === 'users') {
    await Promise.all([fetchUsers(), fetchPoolUsers()])
  } else {
    await Promise.all([fetchInbounds(), fetchPoolUsers()])
  }
}
</script>


