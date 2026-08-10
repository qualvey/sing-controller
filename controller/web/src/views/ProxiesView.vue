<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, VideoPlay, CaretRight, CaretBottom } from '@element-plus/icons-vue'
import {
  fetchProxies,
  selectProxy,
  fetchGroupLatency,
  fetchProxyLatency,
  patchConfigs,
  type ProxyItem
} from '../api/clash'

const DEFAULT_TEST_URL = 'http://www.gstatic.com/generate_204'
const GROUP_TYPES = ['Selector', 'URLTest', 'Fallback', 'LoadBalance']

const loading = ref(false)
const proxies = ref<Record<string, ProxyItem>>({})
const expanded = ref<Set<string>>(new Set())
const latencyMap = ref<Record<string, number>>({})
const testingGroup = ref<string | null>(null)
const testingNode = ref<string | null>(null)
const testUrl = ref(localStorage.getItem('proxy-test-url') || DEFAULT_TEST_URL)

const groups = computed(() =>
  Object.values(proxies.value).filter((p) => GROUP_TYPES.includes(p.type))
)
const standaloneNodes = computed(() =>
  Object.values(proxies.value).filter((p) => !GROUP_TYPES.includes(p.type))
)

const proxiesMode = ref('rule')
const load = async () => {
  loading.value = true
  try {
    const { data } = await fetchProxies()
    proxies.value = data.proxies || {}
    try {
      const cfg = await fetchConfigsSafe()
      if (cfg.mode) proxiesMode.value = cfg.mode
    } catch {
      // 模式读取失败忽略
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '加载代理失败')
  } finally {
    loading.value = false
  }
}

const fetchConfigsSafe = async () => {
  const { fetchConfigs } = await import('../api/clash')
  const { data } = await fetchConfigs()
  return data
}

const toggleGroup = (name: string) => {
  const s = new Set(expanded.value)
  if (s.has(name)) s.delete(name)
  else s.add(name)
  expanded.value = s
}

const isExpanded = (name: string) => expanded.value.has(name)

const latencyColor = (d: number | undefined) => {
  if (d === undefined || d < 0) return ''
  if (d < 100) return 'text-green-600 dark:text-green-400'
  if (d < 300) return 'text-yellow-600 dark:text-yellow-400'
  return 'text-red-600 dark:text-red-400'
}

const formatLatency = (d: number | undefined) => (d === undefined || d < 0 ? '—' : `${d} ms`)

const pick = async (group: string, node: string) => {
  try {
    await selectProxy(group, node)
    const g = proxies.value[group]
    if (g) g.now = node
  } catch (e) {
    ElMessage.error((e as Error).message || '切换失败')
  }
}

const testGroup = async (name: string) => {
  if (testingGroup.value) return
  testingGroup.value = name
  try {
    const { data } = await fetchGroupLatency(name, testUrl.value || DEFAULT_TEST_URL, 5000)
    Object.assign(latencyMap.value, data)
  } catch (e) {
    ElMessage.error((e as Error).message || '测速失败')
  } finally {
    testingGroup.value = null
  }
}

const testAllGroups = async () => {
  if (testingGroup.value) return
  testingGroup.value = '__all__'
  try {
    const results = await Promise.all(
      groups.value.map((g) => fetchGroupLatency(g.name, testUrl.value || DEFAULT_TEST_URL, 5000))
    )
    results.forEach((r) => Object.assign(latencyMap.value, r.data))
  } catch (e) {
    ElMessage.error((e as Error).message || '测速失败')
  } finally {
    testingGroup.value = null
  }
}

const testNode = async (name: string) => {
  testingNode.value = name
  try {
    const { data } = await fetchProxyLatency(name, testUrl.value || DEFAULT_TEST_URL, 5000)
    latencyMap.value[name] = data.delay
  } catch {
    latencyMap.value[name] = -1
  } finally {
    testingNode.value = null
  }
}

const changeMode = async (m: string) => {
  try {
    await patchConfigs({ mode: m })
    proxiesMode.value = m
  } catch (e) {
    ElMessage.error((e as Error).message || '切换模式失败')
  }
}

const saveTestUrl = () => {
  localStorage.setItem('proxy-test-url', testUrl.value)
}

onMounted(() => {
  void load()
})
onBeforeUnmount(() => {
  saveTestUrl()
})
</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- 工具栏 -->
    <div class="flex flex-wrap items-center gap-3 rounded-lg border border-[var(--el-border-color)] bg-[var(--el-bg-color)] px-4 py-3">
      <el-select v-model="proxiesMode" size="small" style="width: 130px" @change="changeMode">
        <el-option label="Rule" value="rule" />
        <el-option label="Global" value="global" />
        <el-option label="Direct" value="direct" />
      </el-select>
      <el-input
        v-model="testUrl"
        size="small"
        placeholder="测速 URL"
        style="width: 260px"
        @change="saveTestUrl"
      />
      <el-button
        size="small"
        :icon="VideoPlay"
        :loading="testingGroup === '__all__'"
        @click="testAllGroups"
      >
        全部测速
      </el-button>
      <el-button size="small" :icon="Refresh" :loading="loading" @click="load">刷新</el-button>
      <span class="ml-auto text-xs text-[var(--el-text-color-secondary)]">
        {{ groups.length }} 组 · {{ standaloneNodes.length }} 节点
      </span>
    </div>

    <!-- 代理组卡片 -->
    <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
      <div
        v-for="g in groups"
        :key="g.name"
        class="rounded-lg border border-[var(--el-border-color)] bg-[var(--el-bg-color)] shadow-sm transition-shadow hover:shadow-md"
      >
        <div
          class="flex cursor-pointer items-center gap-2 px-4 py-3"
          @click="toggleGroup(g.name)"
        >
          <el-icon class="text-[var(--el-color-primary)]"><CaretRight v-if="!isExpanded(g.name)" /><CaretBottom v-else /></el-icon>
          <span class="flex-1 truncate text-sm font-semibold" :title="g.name">{{ g.name }}</span>
          <span
            class="rounded px-1.5 py-0.5 text-xs text-[var(--el-color-primary)]"
            style="background: color-mix(in srgb, var(--el-color-primary) 12%, transparent)"
          >{{ g.type }}</span>
          <span class="w-16 text-right text-xs" :class="latencyColor(latencyMap[g.name])">
            {{ formatLatency(latencyMap[g.name]) }}
          </span>
          <el-button
            size="small"
            text
            :loading="testingGroup === g.name"
            @click.stop="testGroup(g.name)"
          >测速</el-button>
        </div>

        <!-- 展开：节点列表 -->
        <div v-if="isExpanded(g.name)" class="border-t border-[var(--el-border-color)] p-3">
          <div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
            <div
              v-for="node in g.all || []"
              :key="node"
              class="flex cursor-pointer items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs transition-colors"
              :class="
                node === g.now
                  ? 'border-[var(--el-color-primary)] bg-[color-mix(in_srgb,var(--el-color-primary)_10%,transparent)]'
                  : 'border-[var(--el-border-color)] hover:border-[var(--el-color-primary)]'
              "
              @click="pick(g.name, node)"
            >
              <span class="flex-1 truncate" :title="node">{{ node }}</span>
              <span :class="latencyColor(latencyMap[node])">{{ formatLatency(latencyMap[node]) }}</span>
              <el-button
                size="small"
                text
                :loading="testingNode === node"
                @click.stop="testNode(node)"
              >↻</el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 未分组节点 -->
    <div v-if="standaloneNodes.length" class="rounded-lg border border-[var(--el-border-color)] bg-[var(--el-bg-color)] px-4 py-3">
      <div class="mb-2 text-sm font-semibold">节点</div>
      <div class="flex flex-wrap gap-2">
        <span
          v-for="n in standaloneNodes"
          :key="n.name"
          class="inline-flex items-center gap-1.5 rounded-md border border-[var(--el-border-color)] px-2.5 py-1 text-xs"
        >
          {{ n.name }}
          <span :class="latencyColor(latencyMap[n.name])">{{ formatLatency(latencyMap[n.name]) }}</span>
        </span>
      </div>
    </div>

    <el-empty v-if="!loading && !groups.length && !standaloneNodes.length" description="暂无代理（clash API 未配置或核心未启动）" />
  </div>
</template>
