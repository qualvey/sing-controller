<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { VideoPlay, CaretRight, Lightning } from '@element-plus/icons-vue'
import { subscribeGroups, selectOutbound, uRLTest, setClashMode } from '../api/singbox'
import { fetchProxyLatency } from '../api/clash'
import type { Groups, Group } from '@/gen/daemon/started_service_pb'
import { useRuntimeStore } from '../stores/runtime'

const runtime = useRuntimeStore()
const DEFAULT_TEST_URL = 'http://www.gstatic.com/generate_204'

const groups = ref<Group[]>([])
const latencyMap = ref<Record<string, number>>({})
const expanded = ref<Set<string>>(new Set())
const testing = ref<Set<string>>(new Set())
const connected = ref(false)
const failed = ref(false)
const closed = ref(false)
const testUrl = ref(localStorage.getItem('proxy-test-url') || DEFAULT_TEST_URL)

const allNodes = computed(() => groups.value.flatMap((g) => g.items))
const totalNodes = computed(() => allNodes.value.length)

// 入场动画：滚动到视口时 bounceIn
const cardRefs = ref<HTMLElement[]>([])
const bounced = ref<Set<HTMLElement>>(new Set())
let observer: IntersectionObserver | undefined
const observeCards = () => {
  observer?.disconnect()
  observer = new IntersectionObserver(
    (entries) => {
      for (const e of entries) {
        if (e.isIntersecting && !bounced.value.has(e.target as HTMLElement)) {
          ;(e.target as HTMLElement).classList.add('card-bounce-in')
          bounced.value.add(e.target as HTMLElement)
        }
      }
    },
    { threshold: 0.15 }
  )
  cardRefs.value.forEach((el) => el && observer?.observe(el))
}

const syncLatencies = (gs: Group[]) => {
  const next: Record<string, number> = {}
  for (const g of gs) {
    for (const item of g.items || []) {
      if (item.urlTestDelay > 0) next[item.tag] = item.urlTestDelay
    }
  }
  latencyMap.value = next
}

const handleGroups = (gs: Groups) => {
  if (closed.value) return
  groups.value = gs.group || []
  syncLatencies(groups.value)
  connected.value = true
}

const run = async () => {
  let delay = 1000
  while (!closed.value) {
    try {
      for await (const g of subscribeGroups(abort.signal)) {
        handleGroups(g)
        delay = 1000
      }
    } catch {
      connected.value = false
      failed.value = true
    }
    if (closed.value) break
    await new Promise((r) => setTimeout(r, delay))
    delay = Math.min(delay * 2, 30000)
  }
}

const abort = new AbortController()

const toggleGroup = (tag: string) => {
  const s = new Set(expanded.value)
  if (s.has(tag)) s.delete(tag)
  else s.add(tag)
  expanded.value = s
}

const isExpanded = (tag: string) => expanded.value.has(tag)

const latencyColor = (d: number | undefined) => {
  if (d === undefined || d < 0) return ''
  if (d < 100) return 'text-green-600 dark:text-green-400'
  if (d < 300) return 'text-yellow-600 dark:text-yellow-400'
  return 'text-red-600 dark:text-red-400'
}

const latencyBg = (d: number | undefined) => {
  if (d === undefined || d < 0) return 'bg-gray-300 dark:bg-gray-600'
  if (d < 100) return 'bg-green-500'
  if (d < 300) return 'bg-yellow-500'
  return 'bg-red-500'
}

const formatLatency = (d: number | undefined) => (d === undefined || d < 0 ? '—' : `${d} ms`)

// 组内延迟分布（好/中/差/未连接）——预览条，返回百分比数组
const latencyBands = (g: Group): number[] => {
  const items = g.items || []
  const counts = [0, 0, 0, 0] // good / medium / bad / none
  for (const it of items) {
    const d = latencyMap.value[it.tag]
    if (d === undefined || d < 0) counts[3]++
    else if (d < 100) counts[0]++
    else if (d < 300) counts[1]++
    else counts[2]++
  }
  const total = Math.max(1, items.length)
  return counts.map((c) => (c / total) * 100)
}

const bandColors = ['bg-green-500', 'bg-yellow-500', 'bg-red-500', 'bg-gray-300 dark:bg-gray-600']

const pick = async (group: string, node: string) => {
  try {
    await selectOutbound(group, node)
  } catch (e) {
    ElMessage.error((e as Error).message || '切换失败')
  }
}

// 单节点测速：gRPC URLTest 只接受组，单节点走 clash API（/proxies/{name}/delay）
const testOne = async (tag: string, name: string) => {
  testing.value = new Set(testing.value).add(tag + ':' + name)
  try {
    const { data } = await fetchProxyLatency(name, testUrl.value || DEFAULT_TEST_URL, 5000)
    latencyMap.value[name] = data.delay
  } catch {
    latencyMap.value[name] = -1
  } finally {
    const s = new Set(testing.value)
    s.delete(tag + ':' + name)
    testing.value = s
  }
}

// 组测速：URLTest RPC 一次传组名（服务端对组内节点批量测速，结果经 SubscribeGroups 流推送）
const testGroup = async (g: Group) => {
  if (isTesting(g.tag)) return
  testing.value = new Set(testing.value).add(g.tag)
  try {
    await uRLTest(g.tag)
  } catch (e) {
    ElMessage.error((e as Error).message || '测速失败')
  } finally {
    const s = new Set(testing.value)
    s.delete(g.tag)
    testing.value = s
  }
}

// 全部测速：对所有组逐个 URLTest
const testAll = async () => {
  const s = new Set(testing.value)
  groups.value.forEach((g) => s.add('__all__:' + g.tag))
  testing.value = s
  try {
    await Promise.all(groups.value.map((g) => uRLTest(g.tag)))
  } catch (e) {
    ElMessage.error((e as Error).message || '测速失败')
  } finally {
    const s2 = new Set(testing.value)
    groups.value.forEach((g) => s2.delete('__all__:' + g.tag))
    testing.value = s2
  }
}

const isTesting = (tag: string, name?: string) => {
  const key = name ? `${tag}:${name}` : tag
  return testing.value.has(key)
}

const changeMode = async (m: string) => {
  try {
    await setClashMode(m)
  } catch (e) {
    ElMessage.error((e as Error).message || '切换模式失败')
  }
}

const saveTestUrl = () => {
  localStorage.setItem('proxy-test-url', testUrl.value)
}

const formatSpeed = (n: number) => {
  if (!n) return ''
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(v >= 100 || i === 0 ? 0 : 1)} ${units[i]}/s`
}

onMounted(() => {
  void run()
  observeCards()
})
onBeforeUnmount(() => {
  closed.value = true
  abort.abort()
  observer?.disconnect()
  saveTestUrl()
})
</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- 工具栏 -->
    <div class="flex flex-wrap items-center gap-3 rounded-lg border border-[var(--el-border-color)] bg-[var(--el-bg-color)] px-4 py-3">
      <el-select size="small" style="width: 120px" placeholder="模式" @change="changeMode">
        <el-option label="Rule" value="rule" />
        <el-option label="Global" value="global" />
        <el-option label="Direct" value="direct" />
      </el-select>
      <el-input v-model="testUrl" size="small" placeholder="测速 URL" style="width: 240px" @change="saveTestUrl" />
      <el-button size="small" :icon="VideoPlay" :loading="testing.size > 0" @click="testAll">全部测速</el-button>
      <span class="ml-auto flex items-center gap-2 text-xs text-[var(--el-text-color-secondary)]">
        <span class="inline-block h-2 w-2 rounded-full" :class="connected ? 'bg-green-500' : (failed ? 'bg-red-500' : 'bg-yellow-500')"></span>
        {{ connected ? `${groups.length} 组 · ${totalNodes} 节点` : (failed ? 'service API 不可用' : '连接中…') }}
      </span>
    </div>

    <!-- 代理组卡片：CSS 多列瀑布流（展开只影响本列，不拉扯水平卡片） -->
    <div class="columns-1 gap-4 md:columns-2">
      <div
        v-for="g in groups"
        :key="g.tag"
        :ref="(el) => { if (el) cardRefs.push(el as HTMLElement) }"
        class="group-card mb-4 break-inside-avoid overflow-hidden rounded-xl border border-[var(--el-border-color)] bg-[var(--el-bg-color)] shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:shadow-lg"
      >
        <!-- 头部：点击折叠 -->
        <div class="flex cursor-pointer items-center gap-2.5 px-4 py-3" @click="toggleGroup(g.tag)">
          <el-icon class="shrink-0 text-[var(--el-color-primary)] transition-transform duration-300" :class="{ 'rotate-90': isExpanded(g.tag) }">
            <CaretRight />
          </el-icon>
          <span class="flex-1 truncate text-sm font-semibold" :title="g.tag">{{ g.tag }}</span>
          <span class="shrink-0 rounded px-1.5 py-0.5 text-[11px] text-[var(--el-color-primary)]" style="background: color-mix(in srgb, var(--el-color-primary) 12%, transparent)">
            {{ g.type }} · {{ g.items?.length || 0 }}
          </span>
          <!-- 组实时下载速率 -->
          <span v-if="runtime.groupDownloadSpeed(g.tag) > 0" class="shrink-0 text-[11px] text-blue-500 tabular-nums">
            ↓{{ formatSpeed(runtime.groupDownloadSpeed(g.tag)) }}
          </span>
          <!-- 延迟标签：点击测速 -->
          <span
            class="latency-tag shrink-0 flex h-6 w-11 items-center justify-center rounded-full text-[11px] tabular-nums transition-all"
            :class="[latencyBg(latencyMap[g.selected]), 'text-white cursor-pointer hover:scale-110 active:scale-95']"
            :title="`点击测速 ${g.tag}`"
            @click.stop="testGroup(g)"
          >
            <span v-if="isTesting(g.tag)" class="inline-flex gap-0.5">
              <i class="h-1 w-1 animate-bounce rounded-full bg-white" style="animation-delay: 0ms" />
              <i class="h-1 w-1 animate-bounce rounded-full bg-white" style="animation-delay: 120ms" />
              <i class="h-1 w-1 animate-bounce rounded-full bg-white" style="animation-delay: 240ms" />
            </span>
            <span v-else>{{ formatLatency(latencyMap[g.selected]) }}</span>
          </span>
        </div>

        <!-- 折叠内容（grid-rows 动画） -->
        <div class="collapse-body" :class="{ open: isExpanded(g.tag) }">
          <div class="overflow-hidden">
            <!-- 未展开时显示：延迟分布预览条 -->
            <div v-if="!isExpanded(g.tag)" class="flex h-1.5 overflow-hidden rounded-full mx-4 mb-3 opacity-80">
              <div v-for="(w, i) in latencyBands(g)" :key="i" :class="bandColors[i]" :style="{ width: w + '%' }" class="transition-all duration-500" />
            </div>
            <!-- 展开：节点网格 -->
            <div v-else class="border-t border-[var(--el-border-color)] p-3">
              <div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
                <div
                  v-for="item in g.items"
                  :key="item.tag"
                  class="flex cursor-pointer items-center gap-1.5 rounded-lg border px-2.5 py-2 text-xs transition-all duration-150"
                  :class="
                    item.tag === g.selected
                      ? 'border-[var(--el-color-primary)] bg-[color-mix(in_srgb,var(--el-color-primary)_10%,transparent)] shadow-sm'
                      : 'border-[var(--el-border-color)] hover:-translate-y-px hover:border-[var(--el-color-primary)] hover:shadow-sm'
                  "
                  @click="pick(g.tag, item.tag)"
                >
                  <span class="flex-1 truncate" :title="item.tag">{{ item.tag }}</span>
                  <span :class="latencyColor(latencyMap[item.tag])" class="tabular-nums">{{ formatLatency(latencyMap[item.tag]) }}</span>
                  <button
                    class="rounded p-0.5 opacity-0 transition-opacity hover:bg-[var(--el-fill-color)] group-hover:opacity-100"
                    :class="{ 'opacity-100': isTesting(g.tag, item.tag) }"
                    :disabled="isTesting(g.tag, item.tag)"
                    @click.stop="testOne(g.tag, item.tag)"
                  >
                    <el-icon :size="12" class="text-[var(--el-text-color-secondary)]"><Lightning /></el-icon>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <el-empty v-if="connected && !groups.length" description="无代理组（配置里没有 selector/urltest 等组）" />
    <el-empty v-else-if="!connected" :description="failed ? 'service API 不可用（检查 sing-box services[type=api] 配置）' : '连接中…'" />
  </div>
</template>

<style scoped>
/* 折叠动画：grid-template-rows 0fr → 1fr */
.collapse-body {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows 0.35s cubic-bezier(0.32, 0.72, 0, 1);
}
.collapse-body.open {
  grid-template-rows: 1fr;
}

/* 入场动画：滚动到视口 bounceIn */
.card-bounce-in {
  animation: cardBounce 0.4s cubic-bezier(0.32, 0.72, 0, 1);
}
@keyframes cardBounce {
  0% {
    opacity: 0;
    transform: scale(0.95) translateY(8px);
  }
  100% {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}
@media (prefers-reduced-motion: reduce) {
  .collapse-body {
    transition: none;
  }
  .card-bounce-in {
    animation: none;
  }
}
</style>
