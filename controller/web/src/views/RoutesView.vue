<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { showToast } from '@/helper/toast'
import { showConfirmDialog } from '@/helper/confirmDialog'
import { PlusIcon, RefreshCw } from 'lucide-vue-next'
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { SelectField } from '@/components/ui/select'
import { api } from '../api'
import SourcePane from '../components/SourcePane.vue'
import RouteEditDialog from '@/components/routes/RouteEditDialog.vue'
import type { RouteInfo, RouteRule } from '../api'
import { useStatusStore } from '../store/status'
import { RULE_FIELDS, RULE_SUMMARY_ORDER } from './routeFields'

const statusStore = useStatusStore()

const loading = ref(false)
const outerTab = ref('form')
const savingFinal = ref(false)
const routes = ref<RouteInfo['routes']>([])
const finalTag = ref('')
const outboundTags = ref<string[]>([])

const routeDialogVisible = ref(false)
const isEdit = ref(false)
const editingId = ref('')

const actionOptions = [
  { value: 'route', label: '出站（route）' },
  { value: 'direct', label: '直连（direct）' },
  { value: 'bypass', label: '绕过（bypass）' },
  { value: 'reject', label: '拒绝（reject）' },
  { value: 'hijack-dns', label: 'DNS 劫持（hijack-dns）' },
  { value: 'sniff', label: '协议嗅探（sniff）' },
  { value: 'resolve', label: 'DNS 解析（resolve）' },
  { value: 'route-options', label: '路由选项（route-options）' },
] as const

const rows = computed<any[]>(() => routes.value.map((r) => ({ id: r.id, ...r.rule })))

function fmtList(v: unknown): string {
  if (Array.isArray(v)) return v.join(', ')
  return v == null ? '—' : String(v)
}

function ruleSummary(row: Record<string, unknown>): {
  items: Array<{ k: string; v: string }>
  otherCount: number
} {
  if (row.type === 'logical') {
    const count = Array.isArray(row.rules) ? row.rules.length : 0
    return {
      items: [{ k: 'logical', v: `${String(row.mode || 'and')} · ${count} 条子规则` }],
      otherCount: 0,
    }
  }
  const items = RULE_SUMMARY_ORDER.filter((k) => row[k] != null).map((k) => ({
    k,
    v: fmtList(row[k]),
  }))
  const otherCount = RULE_FIELDS.filter(
    (f) => !RULE_SUMMARY_ORDER.includes(f.key) && row[f.key] != null,
  ).length
  return { items, otherCount }
}

function actionText(row: Record<string, unknown>): string {
  const action = typeof row.action === 'string' && row.action ? row.action : 'route'
  if (action === 'route') {
    return typeof row.outbound === 'string' && row.outbound ? `出站 → ${row.outbound}` : '出站'
  }
  const label = actionOptions.find((a) => a.value === action)?.label
  return label ? label.split('（')[0] : action
}

const loadRoutes = async () => {
  loading.value = true
  try {
    const data = await api.routes()
    routes.value = data.routes
    finalTag.value = data.final || ''
  } catch (e) {
    showToast((e as Error).message || '加载路由失败', 'error')
  } finally {
    loading.value = false
  }
}

const loadTags = async () => {
  try {
    const obs = await api.outbounds()
    outboundTags.value = obs.map((o) => String(o.tag)).filter(Boolean)
  } catch (e) {
    showToast((e as Error).message || '加载标签列表失败', 'error')
  }
}

const handleResult = (res: unknown) => {
  const r = (res ?? {}) as { reload_error?: string; message?: string }
  if (r.reload_error) {
    showToast(r.message || `配置已保存，但实例重载失败：${r.reload_error}`, 'warning')
  } else {
    showToast('保存成功', 'success')
  }
}

const saveFinal = async () => {
  if (!finalTag.value) {
    showToast('请先选择 final outbound', 'warning')
    return
  }
  savingFinal.value = true
  try {
    const cfg = (await api.config()) as Record<string, any>
    if (typeof cfg.route !== 'object' || cfg.route === null || Array.isArray(cfg.route)) {
      cfg.route = {}
    }
    cfg.route.final = finalTag.value
    handleResult(await api.saveConfig(cfg))
    await loadRoutes()
    await statusStore.refresh()
  } catch (e) {
    showToast((e as Error).message || '保存 final 失败', 'error')
  } finally {
    savingFinal.value = false
  }
}

const openCreate = () => {
  isEdit.value = false
  editingId.value = ''
  routeDialogVisible.value = true
}

const openEdit = (row: RouteRule) => {
  isEdit.value = true
  editingId.value = typeof row.id === 'string' ? row.id : ''
  routeDialogVisible.value = true
}

const remove = async (row: RouteRule) => {
  const { confirmed } = await showConfirmDialog({
    title: '删除确认',
    message: '确定删除这条路由规则？该操作不可恢复。',
    confirmText: '删除',
    confirmButtonClass: 'btn-error',
  })
  if (!confirmed) return
  const id = typeof row.id === 'string' ? row.id : ''
  if (!id) return
  try {
    await api.deleteRoute(id)
    showToast('删除成功', 'success')
    await Promise.all([loadRoutes(), statusStore.refresh()])
  } catch (e) {
    showToast((e as Error).message || '删除失败', 'error')
  }
}

onMounted(() => {
  loadTags()
  loadRoutes()
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
        <div
          class="mb-3.5 flex items-center gap-2.5 rounded-lg border border-[#e4e7ed] bg-white px-4 py-3 dark:border-[#303030] dark:bg-[#1d1e1f]"
        >
          <span class="font-semibold text-[#303133] dark:text-[#e5eaf3]">route.final</span>
          <SelectField
            v-model="finalTag"
            class="w-60"
            :disabled="!outboundTags.length"
            :options="outboundTags"
            placeholder="选择 final outbound"
          />
          <button
            class="btn btn-primary btn-sm"
            :disabled="savingFinal || !outboundTags.length"
            @click="saveFinal"
          >
            保存 final
          </button>
          <span class="text-xs text-[#909399]">未匹配任何规则时的默认出站（整体读取/回写配置）</span>
        </div>

        <div class="mb-3.5 flex flex-wrap items-center gap-2.5">
          <button class="btn btn-primary btn-sm" @click="openCreate">
            <PlusIcon class="h-4 w-4" />
            新建规则
          </button>
          <button class="btn btn-ghost btn-sm" :disabled="loading" @click="loadRoutes">
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
            刷新
          </button>
          <span class="text-xs text-[#909399]"
            >规则模型：rule(匹配条件) → action。默认 action 为出站（route）；reject 拒绝、bypass
            绕过、direct 直连</span
          >
        </div>

        <div
          class="overflow-hidden rounded-lg border border-[#e4e7ed] bg-white dark:border-[#303030] dark:bg-[#1d1e1f]"
        >
          <div v-if="loading && !rows.length" class="p-10 text-center text-sm text-[#909399]">
            加载中…
          </div>
          <table v-else class="table table-sm w-full">
            <thead>
              <tr>
                <th class="w-14">#</th>
                <th>rule</th>
                <th class="w-[150px]">action</th>
                <th class="w-[130px] text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, idx) in rows" :key="row.id">
                <td class="text-xs text-[#909399]">{{ idx + 1 }}</td>
                <td>
                  <div class="flex flex-col gap-0.5">
                    <template v-if="ruleSummary(row).items.length">
                      <span v-for="it in ruleSummary(row).items" :key="it.k" class="text-[13px]">
                        <b class="mr-1 font-medium text-[#909399]">{{ it.k }}</b> {{ it.v }}
                      </span>
                      <span v-if="ruleSummary(row).otherCount" class="text-[13px] text-[#c0c4cc]"
                        >+{{ ruleSummary(row).otherCount }} 项</span
                      >
                    </template>
                    <span v-else class="text-[13px] text-[#c0c4cc]">全部（无匹配条件）</span>
                    <span v-if="row.invert" class="text-[13px] font-semibold text-[#e6a23c]"
                      >[取反]</span
                    >
                  </div>
                </td>
                <td class="text-xs">{{ actionText(row) }}</td>
                <td class="text-right">
                  <button class="btn btn-ghost btn-xs text-primary" @click="openEdit(row)">
                    编辑
                  </button>
                  <button class="btn btn-ghost btn-xs text-error" @click="remove(row)">删除</button>
                </td>
              </tr>
            </tbody>
          </table>
          <div
            v-if="!loading && !rows.length"
            class="py-8 text-center text-sm text-[#606266] dark:text-[#a6b0bf]"
          >
            暂无规则
          </div>
        </div>
      </TabsContent>

      <TabsContent value="source">
        <SourcePane segment="route" @saved="loadRoutes" />
      </TabsContent>
    </TabsRoot>

    <!-- 独立的 RouteEditDialog 弹窗 -->
    <RouteEditDialog
      v-model="routeDialogVisible"
      :id="editingId"
      :is-edit="isEdit"
      @saved="loadRoutes"
    />
  </div>
</template>
