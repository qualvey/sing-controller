<script setup lang="ts">
import { useCalculateMaxProxies } from '@/composables/proxiesScroll'
import { handlerProxySelect } from '@/assembly/proxies'
import { groupProxiesByProviderName } from '@/composables/renderProxies'
import { computed } from 'vue'
import ProxyNodeCard from './ProxyNodeCard.vue'
import ProxyNodeGrid from './ProxyNodeGrid.vue'

const props = defineProps<{
  name: string
  now: string
  renderProxies: string[]
}>()

const groupedProxies = computed(() => groupProxiesByProviderName(props.renderProxies))

const activeIndex = groupedProxies.value.reduce((acc, { proxies }) => {
  const index = proxies.indexOf(props.now)

  if (index !== -1) {
    return acc + index
  }
  return acc + proxies.length
}, 0)

const { maxProxies } = useCalculateMaxProxies(props.renderProxies.length, activeIndex)

const truncatedProxies = computed(() => {
  let displayCount = 0
  const truncatedProxies: { providerName: string; proxies: string[] }[] = []

  for (const { providerName, proxies } of groupedProxies.value) {
    if (displayCount + proxies.length > maxProxies.value) {
      truncatedProxies.push({
        providerName,
        proxies: proxies.slice(0, maxProxies.value - displayCount),
      })
      break
    } else {
      truncatedProxies.push({ providerName, proxies })
      displayCount += proxies.length
    }
  }
  return truncatedProxies
})
</script>

<template>
  <div class="flex flex-col gap-2">
    <div
      v-for="({ providerName, proxies }, index) in truncatedProxies"
      :key="index"
    >
      <div class="my-2 flex items-center gap-2">
        <span class="text-sm font-semibold text-primary">
          {{ providerName || '自建与基础出站 (Custom Outbounds)' }}
        </span>
        <span
          v-if="!providerName"
          class="rounded bg-primary/15 px-1.5 py-0.5 text-xs font-semibold text-primary"
        >
          Base
        </span>
        <span
          v-else
          class="rounded bg-base-300 px-1.5 py-0.5 text-xs font-medium text-base-content/70"
        >
          Provider
        </span>
      </div>
      <ProxyNodeGrid>
        <ProxyNodeCard
          v-for="node in proxies"
          :key="node"
          :name="node"
          :group-name="name"
          :active="node === now"
          @click.stop="handlerProxySelect(name, node)"
        />
      </ProxyNodeGrid>
    </div>
  </div>
</template>
