import { computed, ref } from 'vue'
import { proxyGroupList, proxyMap, proxyLatencyTest, speedtestUrlWithDefault, fetchProxies } from '@/assembly/proxies'
import { speedtestTimeout } from '@/store/settings'
import { api } from '@/api'
import { showToast } from '@/helper/toast'
import { showNotification } from '@/helper/notification'
import { useI18n } from 'vue-i18n'

const SYSTEM_TYPES = new Set([
  'direct',
  'reject',
  'dns',
  'block',
  'compatible',
  'selector',
  'urltest',
])

const SYSTEM_NAMES = new Set([
  'GLOBAL',
  'DIRECT',
  'REJECT',
  'dns-out',
])

// 弹窗状态共享
export const isAssignModalOpen = ref(false)
export const assignTargetNodes = ref<string[]>([])

export const openAssignModal = (nodes: string[] | string) => {
  assignTargetNodes.value = Array.isArray(nodes) ? [...nodes] : [nodes]
  isAssignModalOpen.value = true
}

export const closeAssignModal = () => {
  isAssignModalOpen.value = false
  assignTargetNodes.value = []
}

export const isBatchSpeedtesting = ref(false)

export const useUngroupedProxies = () => {
  // 1. 收集当前所有策略组中包含的节点名
  const allGroupedMembers = computed(() => {
    const set = new Set<string>()
    for (const groupName of proxyGroupList.value) {
      const group = proxyMap.value[groupName]
      if (group?.all && Array.isArray(group.all)) {
        for (const tag of group.all) {
          set.add(tag)
        }
      }
    }
    return set
  })

  // 2. 筛选未属于任何分组的独立出站节点
  const ungroupedNodeNames = computed<string[]>(() => {
    const grouped = allGroupedMembers.value
    const result: string[] = []

    for (const [name, proxy] of Object.entries(proxyMap.value)) {
      if (SYSTEM_NAMES.has(name)) continue
      if (proxy.all && proxy.all.length > 0) continue // 排除策略组自身
      const typeLower = (proxy.type || '').toLowerCase()
      if (SYSTEM_TYPES.has(typeLower)) continue // 排除直连/拦截/DNS等系统出站

      if (!grouped.has(name)) {
        result.push(name)
      }
    }
    return result
  })

  // 3. 当前可分配的目标策略组列表 (selector / urltest)
  const availableProxyGroups = computed<string[]>(() => {
    return proxyGroupList.value.filter((name) => {
      const group = proxyMap.value[name]
      if (!group) return true
      const type = (group.type || '').toLowerCase()
      // selector 与 urltest 支持包含子节点
      return type === 'selector' || type === 'urltest' || !type
    })
  })

  // 4. 批量测速未分组节点
  const batchSpeedtest = async (targetNodes?: string[]) => {
    const list = targetNodes && targetNodes.length > 0 ? targetNodes : ungroupedNodeNames.value
    if (list.length === 0 || isBatchSpeedtesting.value) return

    isBatchSpeedtesting.value = true
    try {
      const testUrl = speedtestUrlWithDefault.value
      const timeout = speedtestTimeout.value || 5000

      // 并发测速，限制单批次并发度为 6
      const concurrency = 6
      for (let i = 0; i < list.length; i += concurrency) {
        const chunk = list.slice(i, i + concurrency)
        await Promise.allSettled(
          chunk.map((name) => proxyLatencyTest(name, testUrl, timeout).catch(() => {}))
        )
      }
    } finally {
      isBatchSpeedtesting.value = false
    }
  }

  // 5. 将节点分配至指定策略组
  const assignNodesToGroups = async (nodeTags: string[], groupTags: string[]) => {
    if (nodeTags.length === 0 || groupTags.length === 0) return

    // 获取当前配置中的所有出站
    const allOutbounds = await api.outbounds()
    let updatedCount = 0

    for (const groupTag of groupTags) {
      const groupOb = allOutbounds.find((o) => o.tag === groupTag)
      if (!groupOb) continue

      const rawMembers = (groupOb as any).outbounds
      const currentMembers: string[] = Array.isArray(rawMembers) ? [...rawMembers] : []
      let hasAdded = false

      for (const nodeTag of nodeTags) {
        if (!currentMembers.includes(nodeTag)) {
          currentMembers.push(nodeTag)
          hasAdded = true
        }
      }

      if (hasAdded) {
        ;(groupOb as any).outbounds = currentMembers
        await api.updateOutbound(groupTag, groupOb as any, false)
        updatedCount++
      }
    }

    if (updatedCount > 0) {
      // 热重载核心并拉取最新代理数据
      await api.reload()
      await fetchProxies(true)
    }
  }

  return {
    ungroupedNodeNames,
    availableProxyGroups,
    batchSpeedtest,
    assignNodesToGroups,
    isBatchSpeedtesting,
  }
}
