<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { showToast } from '@/helper/toast'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { PlusIcon, RefreshCw } from 'lucide-vue-next'
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { api } from '../api'
import SourcePane from '../components/SourcePane.vue'
import OutboundEditDialog from '@/components/outbounds/OutboundEditDialog.vue'
import type { Outbound } from '../api'
import { useStatusStore } from '../stores/status'

const statusStore = useStatusStore()

const loading = ref(false)
const outerTab = ref('form')
const outbounds = ref<Outbound[]>([])

const dialogVisible = ref(false)
const isEdit = ref(false)
const editingTag = ref('')

const openCreate = () => {
  isEdit.value = false
  editingTag.value = ''
  dialogVisible.value = true
}

const openEdit = (row: Outbound) => {
  isEdit.value = true
  editingTag.value = row.tag
  dialogVisible.value = true
}

const remove = async (row: Outbound) => {
  const first = await showConfirmDialog({
    title: '删除确认',
    message: `确定删除 outbound「${row.tag}」？该操作不可恢复。`,
    confirmText: '删除',
    confirmButtonClass: 'btn-error'
  })
  if (!first.confirmed) return
  try {
    await api.deleteOutbound(row.tag)
    showToast('删除成功', 'success')
    await Promise.all([loadOutbounds(), statusStore.refresh()])
  } catch (e) {
    const err = e as Error & { references?: string[] }
    // 被 selector/urltest 组引用：确认后强制删除（后端自动从引用组拔除 tag）
    if (err.references?.length) {
      const second = await showConfirmDialog({
        title: '节点被组引用',
        message: `节点「${row.tag}」被以下组引用：${err.references.join('、')}\n删除后将自动从这些组中移除该节点。确认删除？`,
        confirmText: '确认删除',
        confirmButtonClass: 'btn-error'
      })
      if (!second.confirmed) return
      try {
        await api.deleteOutbound(row.tag, true)
        showToast(`删除成功，并已从 ${err.references.join('、')} 中移除`, 'success')
        await Promise.all([loadOutbounds(), statusStore.refresh()])
      } catch (e2) {
        showToast((e2 as Error).message || '删除失败', 'error')
      }
      return
    }
    showToast(err.message || '删除失败', 'error')
  }
}

const loadOutbounds = async () => {
  loading.value = true
  try {
    outbounds.value = await api.outbounds()
  } catch (e) {
    showToast((e as Error).message || '加载 outbounds 失败', 'error')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadOutbounds()
})
</script>

<template>
  <div>
    <TabsRoot v-model="outerTab">
      <TabsList>
        <TabsTrigger value="form">表单</TabsTrigger>
        <TabsTrigger value="source">源码</TabsTrigger>
      </TabsList>

      <TabsContent value="form">
        <div class="mb-3.5 flex items-center gap-2.5">
          <button class="btn btn-primary btn-sm" @click="openCreate">
            <PlusIcon class="h-4 w-4" />
            新建 Outbound
          </button>
          <button class="btn btn-ghost btn-sm" :disabled="loading" @click="loadOutbounds">
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
        </div>

        <div class="overflow-hidden rounded-lg border border-[#e4e7ed] bg-white dark:border-[#303030] dark:bg-[#1d1e1f]">
          <div v-if="loading && !outbounds.length" class="p-10 text-center text-sm text-[#909399]">加载中…</div>
          <table v-else class="table table-sm w-full">
            <thead>
              <tr>
                <th class="w-[130px]">类型</th>
                <th class="w-[160px]">tag</th>
                <th>server</th>
                <th class="w-[120px]">server_port</th>
                <th class="w-[130px] text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in outbounds" :key="row.tag">
                <td class="text-xs">{{ row.type }}</td>
                <td class="text-xs font-medium">{{ row.tag }}</td>
                <td class="text-xs">{{ row.server ?? '—' }}</td>
                <td class="text-xs">{{ row.server_port ?? '—' }}</td>
                <td class="text-right">
                  <button class="btn btn-ghost btn-xs text-primary" @click="openEdit(row)">编辑</button>
                  <button class="btn btn-ghost btn-xs text-error" @click="remove(row)">删除</button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!loading && !outbounds.length" class="py-8 text-center text-sm text-[#606266] dark:text-[#a6b0bf]">暂无 outbound</div>
        </div>
      </TabsContent>

      <TabsContent value="source">
        <SourcePane segment="outbounds" @saved="loadOutbounds" />
      </TabsContent>
    </TabsRoot>

    <!-- 复用 Outbound 新建/编辑弹窗 -->
    <OutboundEditDialog
      v-model="dialogVisible"
      :tag="editingTag"
      :is-edit="isEdit"
      @saved="loadOutbounds"
    />
  </div>
</template>
