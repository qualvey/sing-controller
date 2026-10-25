// 组装层 · nodeStatus 门面。
// 节点在线状态与延迟由 controller 后台探测并落库,前端只做只读轮询 + 手动触发,
// 因此这里不参与 clash / sing-box 的后端路由:未配置数据库时整体降级为空。
import type { NodeProbeStatus, NodeProbeResult, NodeProbeState } from '@/api/controller'
import { api } from '@/api/controller'
import { useIntervalFn } from '@vueuse/core'
import { computed, ref } from 'vue'

/** 默认轮询周期,与后端 node_probe.interval(默认 120s)对齐。 */
const DEFAULT_POLL_INTERVAL = 120_000
/** 探测未启用 / 数据库未配置时不轮询。 */
const MIN_POLL_INTERVAL = 15_000

export const nodeStatusMap = ref<Record<string, NodeProbeState>>({})
export const nodeStatusAvailable = ref(false)
export const nodeStatusError = ref('')
export const nodeStatusRefreshing = ref(false)

/** 连续失败多少次后判定为「功能不可用」，避免网络抖动立刻抹掉状态点。 */
const MAX_CONSECUTIVE_FAILURES = 3
let consecutiveFailures = 0

const applyStates = (nodes: NodeProbeState[]) => {
  const next: Record<string, NodeProbeState> = {}
  for (const node of nodes) {
    next[node.node_tag] = node
  }
  nodeStatusMap.value = next
  nodeStatusAvailable.value = true
  nodeStatusError.value = ''
  consecutiveFailures = 0
}

/**
 * 刷新一次状态。
 * 数据库未配置 / 临时网络抖动时不弹错误提示，也不立刻清空已有状态：
 * 只有连续多次失败才判定为不可用，避免状态点在列表里闪烁。
 */
export const refreshNodeStatus = async () => {
  if (nodeStatusRefreshing.value) return

  nodeStatusRefreshing.value = true
  try {
    applyStates(await api.nodeProbes())
  } catch (error) {
    consecutiveFailures += 1
    if (consecutiveFailures >= MAX_CONSECUTIVE_FAILURES) {
      nodeStatusAvailable.value = false
    }
    nodeStatusError.value = error instanceof Error ? error.message : String(error)
  } finally {
    nodeStatusRefreshing.value = false
  }
}

/** 手动触发一次全量探测,随后刷新状态。失败时返回空数组并记录原因,不向外抛异常。 */
export const triggerNodeProbe = async (): Promise<NodeProbeResult[]> => {
  try {
    const response = await api.triggerNodeProbe()
    await refreshNodeStatus()

    return response.results
  } catch (error) {
    nodeStatusError.value = error instanceof Error ? error.message : String(error)
    return []
  }
}

/** 手动触发单个节点探测,随后刷新状态。失败时返回空数组并记录原因,不向外抛异常。 */
export const triggerSingleNodeProbe = async (tag: string): Promise<NodeProbeResult[]> => {
  try {
    const response = await api.triggerSingleNodeProbe(tag)
    await refreshNodeStatus()

    return response.results
  } catch (error) {
    nodeStatusError.value = error instanceof Error ? error.message : String(error)
    return []
  }
}

export const getNodeStatusByName = (name?: string): NodeProbeState | undefined =>
  name ? nodeStatusMap.value[name] : undefined

export const getNodeHealthByName = (name?: string): NodeProbeStatus => {
  const state = getNodeStatusByName(name)

  // 未探测过的节点不显示状态点,由调用方按 available 判断是否渲染。
  return state?.status ?? 'unknown'
}

const onlineCount = computed(
  () => Object.values(nodeStatusMap.value).filter((item) => item.status === 'online').length,
)
const offlineCount = computed(
  () => Object.values(nodeStatusMap.value).filter((item) => item.status === 'offline').length,
)
const unknownCount = computed(
  () => Object.values(nodeStatusMap.value).filter((item) => item.status === 'unknown').length,
)

export const nodeStatusSummary = computed(() => ({
  total: Object.keys(nodeStatusMap.value).length,
  online: onlineCount.value,
  offline: offlineCount.value,
  unknown: unknownCount.value,
}))

let pollHandle: ReturnType<typeof useIntervalFn> | null = null

/** 启动状态轮询。重复调用是安全的。 */
export const startNodeStatusPolling = (interval?: number) => {
  if (pollHandle) return

  const ms = Math.max(interval ?? DEFAULT_POLL_INTERVAL, MIN_POLL_INTERVAL)
  void refreshNodeStatus()
  pollHandle = useIntervalFn(refreshNodeStatus, ms, { immediate: false })
  pollHandle.resume()
}

export const stopNodeStatusPolling = () => {
  // useIntervalFn returns a Pausable with pause/resume controls; it does not
  // expose stop(). Calling the wrong method here throws during page unmount,
  // which can leave navigation in a broken state when the initial request is
  // rejected (for example, with 503 when database.dsn is not configured).
  pollHandle?.pause()
  pollHandle = null
}
