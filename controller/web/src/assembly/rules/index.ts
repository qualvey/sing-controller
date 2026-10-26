// 组装层 · rules 门面。持有 rules / ruleProviderList 统一状态与渲染派生,
// 按后端类型路由到 clash / singbox 实现。
import { toggleRuleDisabledAPI, toggleRuleDisabledSingBoxAPI } from '@/api/clash'
import { Channel, channel } from '@/assembly/backend'
import { RULE_TAB_TYPE } from '@/constant'
import { toSearchRegex } from '@/helper/search'
import type { Rule, RuleProvider } from '@/types'
import { computed, ref } from 'vue'

export const rulesFilter = ref('')
export const rulesTabShow = ref(RULE_TAB_TYPE.RULES)

export const rules = ref<Rule[]>([])
export const ruleProviderList = ref<RuleProvider[]>([])

export const renderRules = computed(() => {
  const searchRegex = toSearchRegex(rulesFilter.value)

  if (!searchRegex) {
    return rules.value
  }

  return rules.value.filter((rule) => {
    return searchRegex.testAny([rule.type, rule.payload, rule.proxy])
  })
})

export const renderRulesProvider = computed(() => {
  const searchRegex = toSearchRegex(rulesFilter.value)

  if (!searchRegex) {
    return ruleProviderList.value
  }

  return ruleProviderList.value.filter((ruleProvider) => {
    return searchRegex.testAny([ruleProvider.name, ruleProvider.behavior, ruleProvider.vehicleType])
  })
})

const load = () => (channel.value === Channel.Singbox ? import('./singbox') : import('./clash'))

export const fetchRules = async () => (await load()).fetchRules()

import { api } from '@/api'

// 规则启用切换：sing-box 模式走 controller 的 /api/routes/{uuid}/toggle，
// Clash 模式走 Clash /rules/{uuid} 或 /rules/disable 端点。
export const toggleRuleDisabled = (rule: Rule, disabled: boolean) => {
  if (channel.value === Channel.Singbox && rule.uuid) {
    return api.toggleRouteDisabled(rule.uuid, disabled)
  }
  return rule.uuid
    ? toggleRuleDisabledSingBoxAPI(rule.uuid)
    : toggleRuleDisabledAPI({ [rule.index]: disabled })
}

// 规则集更新动作(Clash 专属),经 rules 域门面暴露给 view。
export { updateRuleProviderAPI } from '@/api/clash'

export const reorderRules = async (newRules: Rule[]) => {
  if (channel.value === Channel.Singbox) {
    const { reorderRoutes } = await import('./singbox')
    return reorderRoutes(newRules)
  }
  throw new Error('当前后端不支持修改规则顺序')
}
