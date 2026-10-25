<template>
  <DialogWrapper
    v-model="isOpen"
    :title="$t('overviewCardSettings')"
  >
    <div class="flex flex-col text-sm">
      <Draggable
        v-model="orderedCards"
        :animation="150"
        ghost-class="ghost"
        handle=".drag-handle"
        item-key="card"
      >
        <template #item="{ element: cardItem }">
          <div class="setting-item mb-2">
            <Bars3Icon class="drag-handle text-base-content/50 h-5 w-5 shrink-0 cursor-move" />
            <div class="setting-item-label">
              {{ getCardLabel(cardItem) }}
            </div>
            <input
              type="checkbox"
              class="toggle"
              :checked="cardItem?.visible"
              @change="cardItem.visible = !cardItem.visible"
            />
          </div>
        </template>
      </Draggable>
    </div>
  </DialogWrapper>
</template>

<script setup lang="ts">
import DialogWrapper from '@/components/common/DialogWrapper.vue'
import { OVERVIEW_CARD } from '@/constant'
import { overviewCardOrder } from '@/store/settings'
import { Bars3Icon } from '@heroicons/vue/24/outline'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Draggable from 'vuedraggable'

const isOpen = defineModel<boolean>({ required: true })
const { t, te } = useI18n()

const cardKeyToLabelMap: Record<string, string> = {
  CoreControlCard: 'coreControlCard',
  ChartsCard: 'chartsCard',
  NetworkCard: 'networkCard',
  ProviderTrafficOverview: 'providerTrafficOverview',
  TopologyCharts: 'topologyCharts',
  EarthGlobeCard: 'earthGlobeCard',
  ConnectionHistory: 'connectionHistory',
  RuleHitCountCard: 'ruleHitCountCard',
}

const getCardLabel = (item: { card?: OVERVIEW_CARD } | string) => {
  const cardKey = typeof item === 'string' ? item : item?.card
  if (!cardKey) return ''
  const labelKey = cardKeyToLabelMap[cardKey] || cardKey
  return te(labelKey) ? t(labelKey) : labelKey
}

const orderedCards = computed({
  get: () => overviewCardOrder.value,
  set: (newOrder: { card: OVERVIEW_CARD; visible: boolean }[]) => {
    overviewCardOrder.value = newOrder
  },
})
</script>

<style scoped>
.ghost {
  opacity: 0.5;
}
</style>
