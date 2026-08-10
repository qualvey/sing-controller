<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { VideoPlay, CaretRight, CaretBottom } from '@element-plus/icons-vue'
import { subscribeGroups, selectOutbound, uRLTest, setClashMode } from '../api/singbox'
import type { Groups, Group } from '@/gen/daemon/started_service_pb'

const DEFAULT_TEST_URL = 'http://www.gstatic.com/generate_204'

const groups = ref<Group[]>([])
const latencyMap = ref<Record<string, number>>({})
const expanded = ref<Set<string>>(new Set())
const testing = ref<Set<string>>(new Set())
const connected = ref(false)
const failed = ref(false)
const closed = ref(false)
const abort = new AbortController()
const testUrl = ref(localStorage.getItem('proxy-test-url') || DEFAULT_TEST_URL)

const allNodes = computed(() => groups.value.flatMap((g) => g.items))
const totalNodes = computed(() => allNodes.value.length)

const syncLatencies = (gs: Group[]) => {
  const next: Record<string, number> = {}
  for (const g of gs) {
    if (g.items) {
      for (const item of g.items) {
        if (item.urlTestDelay > 0) next[item.tag] = item.urlTestDelay
      }
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
  try {
    for await (const g of subscribeGroups(abort.signal)) {
      handleGroups(g)
    }
  } catch {
    connected.value = false
    failed.value = true
  }
}

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

const formatLatency = (d: number | undefined) => (d === undefined || d < 0 ? '—' : `${d} ms`)

const pick = async (group: string, node: string) => {
  try {
    await selectOutbound(group, node)
  } catch (e) {
    ElMessage.error((e as Error).message || '切换失败')
  }
}

const testOne = async (tag: string, name: string) => {
  testing.value = new Set(testing.value).add(tag + ':' + name)
  try {
    await uRLTest(name)
  } catch {
    latencyMap.value[name] = -1
  } finally {
    const s = new Set(testing.value)
    s.delete(tag + ':' + name)
    testing.value = s
  }
}

const testGroup = async (g: Group) => {
  const marks = new Set<string>()
  const s = new Set(testing.value)
  ;(g.items || []).forEach((item) => {
    s.add(g.tag + ':' + item.tag)
    marks.add(g.tag + ':' + item.tag)
  })
  testing.value = s
  try {
    await Promise.all(
      [...marks].map((m) => uRLTest(m.split(':').slice(1).join(':')))
    )
  } catch (e) {
    ElMessage.error((e as Error).message || '测速失败')
  } finally {
    const s2 = new Set(testing.value)
    marks.forEach((m) => s2.delete(m))
    testing.value = s2
  }
}

const testAll = async () => {
  const marks = new Set<string>()
  const s = new Set(testing.value)
  allNodes.value.forEach((item) => {
    s.add('__all__:' + item.tag)
    marks.add(item.tag)
  })
  testing.value = s
  try {
    await Promise.all([...marks].map((name) => uRLTest(name)))
  } catch (e) {
    ElMessage.error((e as Error).message || '测速失败')
  } finally {
    const s2 = new Set(testing.value)
    marks.forEach((m) => s2.delete('__all__:' + m))
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

onMounted(() => {
  void run()
})
onBeforeUnmount(() => {
  closed.value = true
  abort.abort()
  saveTestUrl()
})
</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- 工具栏 -->
    <div class="flex flex-wrap items-center gap-3 rounded-lg border border-[var(--el-border-color)] bg-[var(--el-bg-color)] px-4 py-3">
      <el-select size="small" style="width: 130px" placeholder="模式" @change="changeMode">
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
      <el-button size="small" :icon="VideoPlay" :loading="testing.size > 0" @click="testAll">
        全部测速
      </el-button>
      <span class="ml-auto flex items-center gap-2 text-xs text-[var(--el-text-color-secondary)]">
        <span class="inline-block h-2 w-2 rounded-full" :class="connected ? 'bg-green-500' : 'bg-red-500'"></span>
        {{ connected ? `${groups.length} 组 · ${totalNodes} 节点` : 'service API 不可用' }}
      </span>
    </div>

    <!-- 代理组卡片 -->
    <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
      <div
        v-for="g in groups"
        :key="g.tag"
        class="rounded-lg border border-[var(--el-border-color)] bg-[var(--el-bg-color)] shadow-sm transition-shadow hover:shadow-md"
      >
        <div class="flex cursor-pointer items-center gap-2 px-4 py-3" @click="toggleGroup(g.tag)">
          <el-icon class="text-[var(--el-color-primary)]">
            <CaretRight v-if="!isExpanded(g.tag)" /><CaretBottom v-else />
          </el-icon>
          <span class="flex-1 truncate text-sm font-semibold" :title="g.tag">{{ g.tag }}</span>
          <span
            class="rounded px-1.5 py-0.5 text-xs text-[var(--el-color-primary)]"
            style="background: color-mix(in srgb, var(--el-color-primary) 12%, transparent)"
          >{{ g.type }}</span>
          <span class="w-16 text-right text-xs" :class="latencyColor(latencyMap[g.selected])">
            {{ formatLatency(latencyMap[g.selected]) }}
          </span>
          <el-button size="small" text :loading="isTesting(g.tag)" @click.stop="testGroup(g)">
            测速
          </el-button>
        </div>

        <!-- 展开：节点列表 -->
        <div v-if="isExpanded(g.tag)" class="border-t border-[var(--el-border-color)] p-3">
          <div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
            <div
              v-for="item in g.items"
              :key="item.tag"
              class="flex cursor-pointer items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs transition-colors"
              :class="
                item.tag === g.selected
                  ? 'border-[var(--el-color-primary)] bg-[color-mix(in_srgb,var(--el-color-primary)_10%,transparent)]'
                  : 'border-[var(--el-border-color)] hover:border-[var(--el-color-primary)]'
              "
              @click="pick(g.tag, item.tag)"
            >
              <span class="flex-1 truncate" :title="item.tag">{{ item.tag }}</span>
              <span :class="latencyColor(latencyMap[item.tag])">{{ formatLatency(latencyMap[item.tag]) }}</span>
              <el-button
                size="small"
                text
                :loading="isTesting(g.tag, item.tag)"
                @click.stop="testOne(g.tag, item.tag)"
              >↻</el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <el-empty v-if="connected && !groups.length" description="无代理组（配置里没有 selector/urltest 等组）" />
    <el-empty v-else-if="!connected" :description="failed ? 'service API 不可用（检查 sing-box services[type=api] 配置）' : '连接中…'" />
  </div>
</template>
