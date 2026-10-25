<template>
  <DialogWrapper
    v-model="isOpen"
    :title="t('profiles.inspectNodesTitle',  { tag })"
    size="large"
  >
    <div class="flex flex-col gap-2 py-2 max-h-[460px] overflow-y-auto">
      <div
        v-if="isLoading"
        class="flex items-center justify-center py-10"
      >
        <span class="loading loading-spinner loading-md text-primary" />
      </div>

      <template v-else>
        <div class="text-xs text-base-content/60">
          {{ t('profiles.extractedNodesCount',  { count: nodes.length }) }}
        </div>
        <div
          v-if="nodes.length === 0"
          class="flex flex-col items-center justify-center py-8 text-base-content/40 text-xs italic"
        >
          {{ t('profiles.noNodesExtracted', '暂无可用节点') }}
        </div>
        <div
          v-else
          class="grid grid-cols-1 md:grid-cols-2 gap-2"
        >
          <div
            v-for="(node, i) in nodes"
            :key="i"
            class="p-2.5 bg-base-300/30 rounded-lg border border-base-300/60 flex items-center justify-between text-xs"
          >
            <div class="flex items-center gap-2 min-w-0">
              <span class="badge badge-xs font-mono uppercase">{{ node.type }}</span>
              <span class="font-medium truncate">{{ node.tag }}</span>
            </div>
            <span
              v-if="node.server"
              class="text-base-content/50 font-mono text-[11px] shrink-0"
            >
              {{ node.server }}:{{ node.server_port }}
            </span>
          </div>
        </div>
      </template>
    </div>
  </DialogWrapper>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import { controllerApi } from '@/api/controller'
import { showNotification } from '@/helper/notification'

export interface PreviewNode {
  type?: string
  tag?: string
  server?: string
  server_port?: number
  [key: string]: unknown
}

const props = defineProps<{
  tag: string
}>()

const isOpen = defineModel<boolean>({ default: false })
const { t } = useI18n()

const isLoading = ref(false)
const nodes = ref<PreviewNode[]>([])

const fetchNodes = async () => {
  if (!props.tag) return
  isLoading.value = true
  try {
    const res = await controllerApi.getProviderNodes(props.tag)
    nodes.value = res?.outbounds || []
  } catch (err: unknown) {
    const msg = (err as any)?.response?.data?.message || (err as Error)?.message || '获取节点列表失败'
    showNotification({ content: msg, type: 'alert-error' })
  } finally {
    isLoading.value = false
  }
}

watch(
  () => [isOpen.value, props.tag],
  ([open]) => {
    if (open && props.tag) {
      fetchNodes()
    }
  },
  { immediate: true },
)
</script>
