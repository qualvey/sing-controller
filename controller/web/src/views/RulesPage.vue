<template>
  <div class="relative size-full overflow-x-hidden">
    <div
      v-if="!isVirtualScroller"
      class="scrollbar-gutter-stable flex h-full w-full flex-col overflow-y-scroll"
    >
      <RulesCtrl />
      <div
        class="p-3"
        :style="padding"
      >
        <template v-if="rulesTabShow === RULE_TAB_TYPE.PROVIDER">
          <div class="base-container">
            <RuleProvider
              v-for="(ruleProvider, index) in renderRulesProvider"
              :key="ruleProvider.name"
              :ruleProvider="ruleProvider"
              :index="index + 1"
            />
          </div>
        </template>
        <template v-else>
          <Draggable
            v-if="canDrag"
            v-model="rules"
            item-key="uuid"
            :animation="150"
            :delay="1000"
            :delay-on-touch-only="true"
            :touch-start-threshold="5"
            ghost-class="opacity-30"
            drag-class="cursor-grabbing"
            filter="input, button, a, .collapse-content, .proxy-chain-node"
            :prevent-on-filter="false"
            class="base-container"
            @end="handleDragEnd"
          >
            <template #item="{ element: rule, index }">
              <RuleCard
                :key="rule.uuid || rule.payload + '-' + index"
                :rule="rule"
                :index="index + 1"
                :can-drag="true"
                :id="'rule-card-' + (index + 1)"
                :class="{
                  'rule-highlight-pulse': highlightedRuleIndex === index + 1,
                }"
              />
            </template>
          </Draggable>
          <div
            v-else
            class="base-container"
          >
            <RuleCard
              v-for="rule in renderRules"
              :key="rule.uuid || rule.payload + '-' + rules.indexOf(rule)"
              :rule="rule"
              :index="rules.indexOf(rule) + 1"
              :can-drag="false"
              :id="'rule-card-' + (rules.indexOf(rule) + 1)"
              :class="{
                'rule-highlight-pulse': highlightedRuleIndex === rules.indexOf(rule) + 1,
              }"
            />
          </div>
        </template>
      </div>
    </div>
    <VirtualScroller
      v-else
      :data="renderRules"
      :size="44"
    >
      <template v-slot:before>
        <RulesCtrl />
      </template>
      <template v-slot="{ item: rule }: { item: Rule }">
        <RuleCard
          :key="rule.uuid || rule.payload + '-' + rules.indexOf(rule)"
          :rule="rule"
          :index="rules.indexOf(rule) + 1"
          :can-drag="false"
          :id="'rule-card-' + (rules.indexOf(rule) + 1)"
          :class="{
            'rule-highlight-pulse': highlightedRuleIndex === rules.indexOf(rule) + 1,
          }"
        />
      </template>
    </VirtualScroller>

    <!-- 规则右键菜单与编辑弹窗 -->
    <RuleContextMenu />
    <RouteEditDialog
      v-model="isRouteEditModalOpen"
      :id="routeEditTargetId"
      :is-edit="isRouteEditMode"
      @saved="fetchRules"
    />

    <!-- 规则集 (RuleSet) 右键菜单与编辑弹窗 -->
    <RuleProviderContextMenu />
    <RuleSetEditDialog
      v-model="isRuleSetModalOpen"
      :is-edit="isRuleSetEditMode"
      :rule-set-data="editingRuleSetData"
      @saved="fetchRules"
    />
  </div>
</template>

<script setup lang="ts">
import Draggable from 'vuedraggable'
import VirtualScroller from '@/components/common/VirtualScroller.vue'
import RulesCtrl from '@/components/controls/RulesCtrl'
import RuleCard from '@/components/rules/RuleCard.vue'
import RuleProvider from '@/components/rules/RuleProvider.vue'
import RuleContextMenu from '@/components/rules/RuleContextMenu.vue'
import RuleProviderContextMenu from '@/components/rules/RuleProviderContextMenu.vue'
import RouteEditDialog from '@/components/routes/RouteEditDialog.vue'
import RuleSetEditDialog from '@/components/rules/RuleSetEditDialog.vue'
import {
  isRouteEditModalOpen,
  isRouteEditMode,
  routeEditTargetId,
  openEditRouteModal,
  closeRouteEditModal,
  closeRuleContextMenu,
} from '@/composables/ruleContextMenu'
import {
  editingRuleSetData,
  isRuleSetEditMode,
  isRuleSetModalOpen,
} from '@/composables/ruleSetManagement'
import { usePaddingForViews } from '@/composables/paddingViews'
import { RULE_TAB_TYPE } from '@/constant'
import { Channel, channel } from '@/assembly/backend'
import {
  fetchRules,
  renderRules,
  renderRulesProvider,
  reorderRules,
  rules,
  rulesFilter,
  rulesTabShow,
} from '@/assembly/rules'
import { showToast } from '@/helper/toast'
import type { Rule } from '@/types'
import { computed, nextTick, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'

const { t } = useI18n()

fetchRules()

const route = useRoute()
const highlightedRuleIndex = ref<number | null>(null)

const canDrag = computed(() => {
  return channel.value === Channel.Singbox && !rulesFilter.value.trim() && rules.value.length > 1
})

const isReordering = ref(false)
const handleDragEnd = async () => {
  if (isReordering.value) return
  isReordering.value = true
  try {
    await reorderRules(rules.value)
    showToast(t('reorderRulesSuccess') || '规则顺序已更新', 'success')
  } catch (err: any) {
    showToast(err.message || '更新规则顺序失败', 'error')
    await fetchRules()
  } finally {
    isReordering.value = false
  }
}

const expandedRule = ref<string | null>(null)
provide('expandedRule', expandedRule)

const { padding } = usePaddingForViews({
  offsetTop: 12,
  offsetBottom: 8,
})
const isVirtualScroller = computed(() => {
  return rulesTabShow.value === RULE_TAB_TYPE.RULES && renderRules.value.length > 80
})

const handleRuleJump = async () => {
  const ruleParam = route.query.rule
  if (!ruleParam) return
  const ruleIndex = Number(ruleParam)
  if (!ruleIndex || isNaN(ruleIndex)) return

  // 确保切换到规则列表页
  rulesTabShow.value = RULE_TAB_TYPE.RULES

  if (rules.value.length === 0) {
    await fetchRules()
  }
  await nextTick()

  highlightedRuleIndex.value = ruleIndex

  // 平滑滚动至目标规则卡片
  setTimeout(() => {
    const el = document.getElementById(`rule-card-${ruleIndex}`)
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  }, 120)

  // 若带 edit 参数，自动唤起编辑模态框进行修改
  if (route.query.edit === '1' || route.query.edit === 'true') {
    openEditRouteModal(ruleIndex)
  }

  // 3.5秒后自动清除高亮
  setTimeout(() => {
    if (highlightedRuleIndex.value === ruleIndex) {
      highlightedRuleIndex.value = null
    }
  }, 3500)
}

onMounted(() => {
  handleRuleJump()
})

watch(
  () => route.query.rule,
  () => {
    handleRuleJump()
  }
)

onBeforeUnmount(() => {
  closeRouteEditModal()
  closeRuleContextMenu()
})
</script>

<style scoped>
:deep(.rule-highlight-pulse) {
  animation: rulePulse 2.8s cubic-bezier(0.2, 0.8, 0.2, 1);
  border-radius: 0.5rem;
}

@keyframes rulePulse {
  0% {
    box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.7);
    background-color: rgba(59, 130, 246, 0.16);
  }
  40% {
    box-shadow: 0 0 0 8px rgba(59, 130, 246, 0);
    background-color: rgba(59, 130, 246, 0.1);
  }
  100% {
    box-shadow: 0 0 0 0 rgba(59, 130, 246, 0);
    background-color: transparent;
  }
}
</style>
