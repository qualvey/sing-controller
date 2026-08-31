import { computed, ref } from 'vue'

// DNS Server 弹窗状态
export const isDnsServerModalOpen = ref(false)
export const editingServer = ref<Record<string, any> | null>(null)
export const isServerEditMode = computed(() => !!editingServer.value)

export const openCreateServerModal = () => {
  editingServer.value = null
  isDnsServerModalOpen.value = true
}

export const openEditServerModal = (server: Record<string, any>) => {
  editingServer.value = { ...server }
  isDnsServerModalOpen.value = true
}

export const closeDnsServerModal = () => {
  isDnsServerModalOpen.value = false
  editingServer.value = null
}

// DNS Rule 弹窗状态
export const isDnsRuleModalOpen = ref(false)
export const editingRule = ref<{ id: string; rule: Record<string, any> } | null>(null)
export const isRuleEditMode = computed(() => !!editingRule.value)

export const openCreateRuleModal = () => {
  editingRule.value = null
  isDnsRuleModalOpen.value = true
}

export const openEditRuleModal = (ruleRow: { id: string; rule: Record<string, any> }) => {
  editingRule.value = { id: ruleRow.id, rule: { ...ruleRow.rule } }
  isDnsRuleModalOpen.value = true
}

export const closeDnsRuleModal = () => {
  isDnsRuleModalOpen.value = false
  editingRule.value = null
}
