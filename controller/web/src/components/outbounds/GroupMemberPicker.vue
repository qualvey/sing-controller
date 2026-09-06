<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  proxyMap,
  fetchProxies,
  getLatencyByName,
  proxyProviederList,
} from '@/assembly/proxies'
import { api, type Outbound } from '@/api'
import { NOT_CONNECTED } from '@/constant'
import { lowLatency, mediumLatency } from '@/store/settings'
import {
  MagnifyingGlassIcon,
  XMarkIcon,
  CheckIcon,
  FunnelIcon,
  ArrowPathIcon,
  PlusIcon,
} from '@heroicons/vue/24/outline'

const props = withDefaults(
  defineProps<{
    /** 当前排除的 tag（不能把自己作为子成员） */
    excludeTag?: string
    /** 策略组名称（用于精确获取该组下的节点测速历史） */
    groupName?: string
  }>(),
  {
    excludeTag: '',
    groupName: '',
  },
)

const model = defineModel<string[]>({ default: () => [] })
const { t } = useI18n()

const searchKeyword = ref('')
const selectedProviderFilter = ref<string>('ALL') // 'ALL' | 'CUSTOM' | providerName
const onlyAliveFilter = ref(false)
const isLoadingNodes = ref(false)
const customOutbounds = ref<Outbound[]>([])

// 格式化协议类型缩写
const formatType = (rawType: string): string => {
  const type = (rawType || '').toLowerCase()
  if (type === 'shadowsocks') return 'SS'
  if (type === 'shadowsocksr') return 'SSR'
  if (type === 'hysteria' || type === 'hysteria2') return 'HY2'
  if (type === 'wireguard') return 'WG'
  if (type === 'selector') return 'Group'
  if (type === 'urltest') return 'Auto'
  return type ? type.toUpperCase() : 'NODE'
}

// 获取延迟背景颜色
const getLatencyBgClass = (latency: number): string => {
  if (latency === NOT_CONNECTED || latency <= 0) return 'bg-base-content/30'
  if (latency < lowLatency.value) return 'bg-low-latency'
  if (latency < mediumLatency.value) return 'bg-medium-latency'
  return 'bg-high-latency'
}

// 获取延迟文字颜色
const getLatencyTextClass = (latency: number): string => {
  if (latency === NOT_CONNECTED || latency <= 0) return 'text-base-content/40'
  if (latency < lowLatency.value) return 'text-low-latency'
  if (latency < mediumLatency.value) return 'text-medium-latency'
  return 'text-high-latency'
}

export interface CandidateNode {
  tag: string
  type: string
  latency: number
  provider: string
  isCustom: boolean
}

// 加载所有可用候选节点（聚合 proxyMap + api.outbounds）
const loadAllCandidates = async () => {
  isLoadingNodes.value = true
  try {
    const [obs] = await Promise.all([
      api.outbounds().catch(() => [] as Outbound[]),
      Object.keys(proxyMap.value).length === 0 ? fetchProxies(true).catch(() => {}) : Promise.resolve(),
    ])
    customOutbounds.value = obs
  } finally {
    isLoadingNodes.value = false
  }
}

onMounted(() => {
  void loadAllCandidates()
})

// 聚合并去重生成所有候选节点列表
const allCandidateNodes = computed<CandidateNode[]>(() => {
  const map = new Map<string, CandidateNode>()
  const exclude = (props.excludeTag || '').trim()

  // 1. 来自 runtime proxyMap (所有机场节点 + 运行时出站)
  for (const [name, p] of Object.entries(proxyMap.value)) {
    if (!name || name === exclude || name === 'GLOBAL') continue
    const latency = getLatencyByName(name, props.groupName)
    const providerName =
      (p as any)['provider-name'] ||
      proxyProviederList.value.find((g) => g.proxies.some((n) => n.name === name))?.name ||
      ''

    map.set(name, {
      tag: name,
      type: p.type || 'proxy',
      latency,
      provider: providerName,
      isCustom: !providerName,
    })
  }

  // 2. 来自主配置中的自定义出站 (补充可能未在 proxyMap 中的静态出站)
  for (const o of customOutbounds.value) {
    const tag = String(o.tag || '')
    if (!tag || tag === exclude) continue
    if (!map.has(tag)) {
      map.set(tag, {
        tag,
        type: String(o.type || 'custom'),
        latency: NOT_CONNECTED,
        provider: '',
        isCustom: true,
      })
    }
  }

  return Array.from(map.values())
})

// 可用的 Provider 来源列表（用于筛选 Chip）
const providerList = computed(() => {
  const set = new Set<string>()
  for (const node of allCandidateNodes.value) {
    if (node.provider) set.add(node.provider)
  }
  return Array.from(set)
})

// 根据搜索词和筛选器过滤候选节点
const filteredCandidates = computed<CandidateNode[]>(() => {
  const kw = searchKeyword.value.trim().toLowerCase()
  const pFilter = selectedProviderFilter.value
  const onlyAlive = onlyAliveFilter.value

  return allCandidateNodes.value.filter((node) => {
    // 来源筛选
    if (pFilter === 'CUSTOM' && !node.isCustom) return false
    if (pFilter !== 'ALL' && pFilter !== 'CUSTOM' && node.provider !== pFilter) return false

    // 仅看已测速
    if (onlyAlive && (node.latency === NOT_CONNECTED || node.latency <= 0)) return false

    // 搜索词
    if (kw) {
      const matchTag = node.tag.toLowerCase().includes(kw)
      const matchType = node.type.toLowerCase().includes(kw)
      const matchProvider = node.provider.toLowerCase().includes(kw)
      if (!matchTag && !matchType && !matchProvider) return false
    }

    return true
  })
})

// 获取单个已选节点的详细信息（带缓存兜底）
const getNodeInfo = (tag: string): CandidateNode => {
  const found = allCandidateNodes.value.find((n) => n.tag === tag)
  if (found) return found
  return {
    tag,
    type: 'NODE',
    latency: NOT_CONNECTED,
    provider: '',
    isCustom: true,
  }
}

// 切换选择
const toggleNode = (tag: string) => {
  const list = [...model.value]
  const idx = list.indexOf(tag)
  if (idx >= 0) {
    list.splice(idx, 1)
  } else {
    list.push(tag)
  }
  model.value = list
}

// 移除单个
const removeNode = (tag: string) => {
  model.value = model.value.filter((t) => t !== tag)
}

// 清空所有已选
const clearAll = () => {
  model.value = []
}

// 全选当前过滤结果
const selectAllFiltered = () => {
  const toAdd = filteredCandidates.value.map((n) => n.tag)
  const set = new Set([...model.value, ...toAdd])
  model.value = Array.from(set)
}

// 取消选中当前过滤结果
const deselectAllFiltered = () => {
  const toRemove = new Set(filteredCandidates.value.map((n) => n.tag))
  model.value = model.value.filter((tag) => !toRemove.has(tag))
}

// 回车手动添加输入框内容（支持逗号分隔多值）
const handleManualAdd = () => {
  const parts = searchKeyword.value
    .split(/[,，\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  if (!parts.length) return

  const list = [...model.value]
  for (const p of parts) {
    if (!list.includes(p)) list.push(p)
  }
  model.value = list
  searchKeyword.value = ''
}
</script>

<template>
  <div class="flex flex-col gap-3 w-full">
    <!-- 1. 已选成员区 (Selected Members) -->
    <div class="rounded-xl border border-base-content/10 bg-base-200/40 p-3">
      <div class="flex items-center justify-between mb-2">
        <div class="flex items-center gap-2">
          <span class="text-xs font-semibold text-base-content">
            已选成员
          </span>
          <span class="badge badge-sm badge-neutral font-mono text-[11px] font-medium">
            {{ model.length }}
          </span>
        </div>
        <div class="flex items-center gap-2">
          <button
            v-if="model.length > 0"
            type="button"
            class="text-xs text-error/80 hover:text-error transition-colors cursor-pointer"
            @click="clearAll"
          >
            清空已选
          </button>
        </div>
      </div>

      <!-- 已选节点 Chip 列表（带协议 Badge + 延迟指示圆点） -->
      <div
        v-if="model.length"
        class="flex flex-wrap gap-1.5 max-h-36 overflow-y-auto p-1 scrollbar-thin"
      >
        <div
          v-for="tag in model"
          :key="tag"
          class="group inline-flex items-center gap-1.5 rounded-lg border border-primary/30 bg-primary/10 px-2 py-1 text-xs text-primary transition-all hover:border-primary/50 select-none shadow-2xs"
        >
          <!-- 延迟指示圆点 Indicator -->
          <span
            class="size-2 rounded-full shrink-0"
            :class="getLatencyBgClass(getNodeInfo(tag).latency)"
            :title="getNodeInfo(tag).latency > 0 ? `${getNodeInfo(tag).latency}ms` : '未测速/未连接'"
          />
          <!-- 协议 Badge Indicator -->
          <span class="rounded bg-primary/20 px-1 py-0.2 text-[10px] font-mono font-semibold tracking-tight text-primary">
            {{ formatType(getNodeInfo(tag).type) }}
          </span>
          <!-- 节点名称 -->
          <span class="font-mono text-xs text-base-content max-w-[180px] truncate" :title="tag">
            {{ tag }}
          </span>
          <!-- 延迟数值显示 (若有) -->
          <span
            v-if="getNodeInfo(tag).latency > 0"
            class="text-[10px] font-mono tabular-nums opacity-75"
          >
            {{ getNodeInfo(tag).latency }}ms
          </span>
          <!-- 删除按钮 -->
          <button
            type="button"
            class="ml-0.5 rounded p-0.5 text-base-content/40 hover:bg-error/15 hover:text-error transition-colors cursor-pointer"
            title="移除该节点"
            @click.stop="removeNode(tag)"
          >
            <XMarkIcon class="size-3.5" />
          </button>
        </div>
      </div>
      <div
        v-else
        class="rounded-lg border border-dashed border-base-content/15 p-3 text-center text-xs text-base-content/40 select-none"
      >
        暂无已选组成员，请在下方候选池中点击勾选节点添加
      </div>
    </div>

    <!-- 2. 候选池操作栏 (Search + Filters) -->
    <div class="flex flex-col gap-2">
      <div class="flex items-center gap-2">
        <!-- 搜索输入框 -->
        <div class="relative flex-1">
          <MagnifyingGlassIcon class="absolute left-2.5 top-1/2 -translate-y-1/2 size-4 text-base-content/40 pointer-events-none" />
          <input
            v-model="searchKeyword"
            type="text"
            class="input input-sm input-bordered w-full pl-8 pr-16 text-xs rounded-lg font-mono placeholder:font-sans"
            placeholder="搜索或手动输入节点回车添加..."
            @keydown.enter.prevent="handleManualAdd"
          />
          <button
            v-if="searchKeyword.trim()"
            type="button"
            class="btn btn-ghost btn-xs absolute right-1.5 top-1/2 -translate-y-1/2 h-6 min-h-0 text-primary gap-0.5 px-1.5"
            @click="handleManualAdd"
            title="按回车添加输入内容"
          >
            <PlusIcon class="size-3" />
            添加
          </button>
        </div>

        <!-- 刷新候选池按钮 -->
        <button
          type="button"
          class="btn btn-ghost btn-sm btn-square rounded-lg text-base-content/60 hover:text-base-content"
          :class="{ 'animate-spin': isLoadingNodes }"
          title="刷新节点列表"
          @click="loadAllCandidates"
        >
          <ArrowPathIcon class="size-4" />
        </button>
      </div>

      <!-- 来源与快捷筛选栏 -->
      <div class="flex flex-wrap items-center justify-between gap-1.5 text-xs">
        <div class="flex flex-wrap items-center gap-1">
          <button
            type="button"
            class="btn btn-xs rounded-md transition-all font-normal"
            :class="selectedProviderFilter === 'ALL' ? 'btn-primary' : 'btn-ghost text-base-content/70 hover:bg-base-200'"
            @click="selectedProviderFilter = 'ALL'"
          >
            全部 ({{ allCandidateNodes.length }})
          </button>
          <button
            type="button"
            class="btn btn-xs rounded-md transition-all font-normal"
            :class="selectedProviderFilter === 'CUSTOM' ? 'btn-primary' : 'btn-ghost text-base-content/70 hover:bg-base-200'"
            @click="selectedProviderFilter = 'CUSTOM'"
          >
            自建出站
          </button>
          <button
            v-for="p in providerList"
            :key="p"
            type="button"
            class="btn btn-xs rounded-md transition-all font-normal"
            :class="selectedProviderFilter === p ? 'btn-primary' : 'btn-ghost text-base-content/70 hover:bg-base-200'"
            @click="selectedProviderFilter = p"
          >
            {{ p }}
          </button>
        </div>

        <div class="flex items-center gap-1 ml-auto shrink-0">
          <label class="flex items-center gap-1 cursor-pointer select-none text-[11px] text-base-content/60 hover:text-base-content mr-1">
            <input
              v-model="onlyAliveFilter"
              type="checkbox"
              class="checkbox checkbox-xs checkbox-primary rounded"
            />
            仅测速可用
          </label>
          <button
            type="button"
            class="btn btn-ghost btn-xs text-[11px] text-primary"
            @click="selectAllFiltered"
          >
            全选当前
          </button>
          <button
            type="button"
            class="btn btn-ghost btn-xs text-[11px] text-base-content/50 hover:text-base-content"
            @click="deselectAllFiltered"
          >
            取消当前
          </button>
        </div>
      </div>
    </div>

    <!-- 3. 候选节点池列表 (Candidate Node Grid / Cards) -->
    <div
      class="rounded-xl border border-base-content/10 bg-base-100 p-2 max-h-64 overflow-y-auto scrollbar-thin"
    >
      <div
        v-if="isLoadingNodes"
        class="flex items-center justify-center py-8 text-xs text-base-content/50 gap-2"
      >
        <span class="loading loading-spinner loading-xs text-primary" />
        正在加载可用候选节点...
      </div>
      <div
        v-else-if="filteredCandidates.length === 0"
        class="py-8 text-center text-xs text-base-content/40 select-none"
      >
        {{ allCandidateNodes.length === 0 ? '暂无可用候选节点，请确保已有出站配置或订阅源' : '没有匹配的节点' }}
      </div>
      <div
        v-else
        class="grid grid-cols-1 sm:grid-cols-2 gap-1.5"
      >
        <div
          v-for="node in filteredCandidates"
          :key="node.tag"
          class="flex items-center justify-between gap-2 rounded-lg border px-2.5 py-1.5 transition-all cursor-pointer select-none text-xs"
          :class="[
            model.includes(node.tag)
              ? 'border-primary/50 bg-primary/10 text-primary shadow-2xs font-medium'
              : 'border-base-content/8 bg-base-200/30 hover:border-base-content/25 hover:bg-base-200/70 text-base-content/85',
          ]"
          @click="toggleNode(node.tag)"
        >
          <!-- 左侧：勾选状态 + 延迟圆点 + 协议 Badge + 节点名 -->
          <div class="flex items-center gap-2 min-w-0 flex-1">
            <!-- 勾选指示器 Checkbox -->
            <div
              class="size-4 rounded flex items-center justify-center shrink-0 border transition-all"
              :class="model.includes(node.tag) ? 'bg-primary border-primary text-primary-content' : 'border-base-content/25 bg-base-100'"
            >
              <CheckIcon v-if="model.includes(node.tag)" class="size-3 stroke-[3]" />
            </div>

            <!-- 延迟指示圆点 Indicator -->
            <span
              class="size-2 rounded-full shrink-0"
              :class="getLatencyBgClass(node.latency)"
              :title="node.latency > 0 ? `${node.latency}ms` : '未测速/未连接'"
            />

            <!-- 协议类型 Badge -->
            <span
              class="shrink-0 rounded px-1 py-0.2 text-[10px] font-mono font-semibold tracking-tight"
              :class="model.includes(node.tag) ? 'bg-primary/20 text-primary' : 'bg-base-300 text-base-content/70'"
            >
              {{ formatType(node.type) }}
            </span>

            <!-- 节点名与来源标签 -->
            <div class="min-w-0 flex-1 flex flex-col">
              <span class="truncate font-mono text-[11px] leading-tight" :title="node.tag">
                {{ node.tag }}
              </span>
              <span
                v-if="node.provider"
                class="text-[9px] text-base-content/40 truncate leading-none mt-0.5"
              >
                {{ node.provider }}
              </span>
            </div>
          </div>

          <!-- 右侧：测速延迟值 Indicator -->
          <div
            class="text-[10px] font-mono tabular-nums shrink-0 text-right"
            :class="getLatencyTextClass(node.latency)"
          >
            {{ node.latency > 0 ? `${node.latency}ms` : '-' }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
