import {
  fetchRules,
  ruleProviderList,
  rules,
  rulesFilter,
  rulesTabShow,
  updateRuleProviderAPI,
} from '@/assembly/rules'
import { useCtrlsBar } from '@/composables/useCtrlsBar'
import { openCreateRouteModal } from '@/composables/ruleContextMenu'
import { openCreateRuleSetModal } from '@/composables/ruleSetManagement'
import { RULE_TAB_TYPE } from '@/constant'
import { showNotification } from '@/helper/notification'
import { showToast } from '@/helper/toast'
import { api } from '@/api'
import { useStatusStore } from '@/store/status'
import {
  disconnectOnRuleDisable,
  displayLatencyInRule,
  displayNowNodeInRule,
} from '@/store/settings'
import { ArrowPathIcon, PlusIcon, WrenchScrewdriverIcon } from '@heroicons/vue/24/outline'
import { computed, defineComponent, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import CtrlsBar from '../common/CtrlsBar.vue'
import DialogWrapper from '../common/DialogWrapper.vue'
import SegmentedControl from '@/components/common/SegmentedControl.vue'
import SelectInput from '../common/SelectInput.vue'
import TextInput from '../common/TextInput.vue'

export default defineComponent({
  name: 'RulesCtrl',
  setup() {
    const { t } = useI18n()
    const statusStore = useStatusStore()
    const settingsModel = ref(false)
    const isUpgrading = ref(false)
    const { isLargeCtrlsBar } = useCtrlsBar()

    const finalTag = ref('')
    const outboundTags = ref<string[]>([])
    const savingFinal = ref(false)

    const loadRouteMeta = async () => {
      try {
        const [routesData, outboundsData] = await Promise.all([api.routes(), api.outbounds()])
        finalTag.value = routesData.final || ''
        outboundTags.value = outboundsData.map((o) => String(o.tag)).filter(Boolean)
      } catch {
        // silently ignore if unauthenticated
      }
    }

    const saveFinal = async (newFinal: string) => {
      if (!newFinal) return
      finalTag.value = newFinal
      savingFinal.value = true
      try {
        const cfg = (await api.config()) as Record<string, any>
        if (typeof cfg.route !== 'object' || cfg.route === null || Array.isArray(cfg.route)) {
          cfg.route = {}
        }
        cfg.route.final = newFinal
        const res = await api.saveConfig(cfg)
        const r = (res ?? {}) as { reload_error?: string; message?: string }
        if (r.reload_error) {
          showToast(`配置已保存，但实例重载失败：${r.reload_error}`, 'warning')
        } else {
          showToast(t('saveSuccess') || '保存成功', 'success')
        }
        await Promise.all([fetchRules(), statusStore.refresh()])
      } catch (e) {
        showToast((e as Error).message || '保存 final 失败', 'error')
      } finally {
        savingFinal.value = false
      }
    }

    watch(settingsModel, (open) => {
      if (open) {
        void loadRouteMeta()
      }
    })

    onMounted(() => {
      void loadRouteMeta()
    })

    const hasProviders = computed(() => {
      return ruleProviderList.value.length > 0
    })

    const handlerClickUpgradeAllProviders = async () => {
      if (isUpgrading.value) return
      isUpgrading.value = true
      try {
        let updateCount = 0

        await Promise.all(
          ruleProviderList.value.map((provider) =>
            updateRuleProviderAPI(provider.name).then(() => {
              updateCount++

              const isFinished = updateCount === ruleProviderList.value.length

              showNotification({
                key: 'updateFinishedTip',
                content: 'updateFinishedTip',
                params: {
                  number: `${updateCount}/${ruleProviderList.value.length}`,
                },
                type: isFinished ? 'alert-success' : 'alert-info',
                timeout: isFinished ? 2000 : 0,
              })
            }),
          ),
        )
        await fetchRules()
        isUpgrading.value = false
      } catch {
        await fetchRules()
        isUpgrading.value = false
      }
    }

    const isRefreshingRules = ref(false)
    const handleRefreshRules = async () => {
      if (isRefreshingRules.value) return
      isRefreshingRules.value = true
      try {
        await Promise.all([fetchRules(), loadRouteMeta(), statusStore.refresh()])
        showToast(t('refreshSuccess') || '规则已刷新', 'success')
      } catch (e) {
        showToast((e as Error).message || '刷新失败', 'error')
      } finally {
        isRefreshingRules.value = false
      }
    }

    const tabsWithNumbers = computed(() => {
      return Object.values(RULE_TAB_TYPE).map((type) => {
        return {
          type,
          count: type === RULE_TAB_TYPE.RULES ? rules.value.length : ruleProviderList.value.length,
        }
      })
    })

    return () => {
      const tabs = (
        <SegmentedControl
          modelValue={rulesTabShow.value}
          onUpdate:modelValue={(value) => (rulesTabShow.value = value as RULE_TAB_TYPE)}
          options={tabsWithNumbers.value.map(({ type, count }) => ({
            value: type,
            label: t(type),
            count,
          }))}
        />
      )

      const isProviderTab = rulesTabShow.value === RULE_TAB_TYPE.PROVIDER
      const isRefreshing = isProviderTab ? isUpgrading.value : isRefreshingRules.value
      const refreshButton = (
        <button
          class="btn btn-circle btn-sm"
          title={isProviderTab ? '全部更新' : (t('refresh') || '刷新规则')}
          onClick={isProviderTab ? handlerClickUpgradeAllProviders : handleRefreshRules}
        >
          <ArrowPathIcon class={['h-4 w-4', isRefreshing && 'animate-spin']} />
        </button>
      )

      const createButton = (
        <button
          class="btn btn-circle btn-sm btn-primary"
          title={isProviderTab ? '新建规则集' : (t('createRule') || '新建规则')}
          onClick={isProviderTab ? openCreateRuleSetModal : openCreateRouteModal}
        >
          <PlusIcon class="h-4 w-4" />
        </button>
      )

      const searchInput = (
        <TextInput
          class={isLargeCtrlsBar.value ? 'w-80' : 'w-32 flex-1'}
          v-model={rulesFilter.value}
          placeholder={`${t('search')} | Regex`}
          clearable={true}
        />
      )

      const settingsModal = (
        <>
          <button
            class={'btn btn-circle btn-sm'}
            onClick={() => (settingsModel.value = true)}
          >
            <WrenchScrewdriverIcon class="h-4 w-4" />
          </button>
          <DialogWrapper
            v-model={settingsModel.value}
            title={t('ruleSettings')}
          >
            <div class="flex flex-col gap-3 text-sm">
              <div class="settings-grid">
                {outboundTags.value.length > 0 && (
                  <div class="setting-item">
                    <div class="setting-item-label">route.final</div>
                    <SelectInput
                      class="select select-sm min-w-32 flex-1"
                      modelValue={finalTag.value}
                      onUpdate:modelValue={(val) => saveFinal(val as string)}
                      options={outboundTags.value.map((tag) => ({
                        value: tag,
                        label: tag,
                      }))}
                    />
                  </div>
                )}
                <div class="setting-item">
                  <div class="setting-item-label">{t('displaySelectedNode')}</div>
                  <input
                    class="toggle"
                    type="checkbox"
                    v-model={displayNowNodeInRule.value}
                  />
                </div>
                <div class="setting-item">
                  <div class="setting-item-label">{t('displayLatencyNumber')}</div>
                  <input
                    class="toggle"
                    type="checkbox"
                    v-model={displayLatencyInRule.value}
                  />
                </div>
                <div class="setting-item">
                  <div class="setting-item-label">{t('disconnectOnRuleDisable')}</div>
                  <input
                    class="toggle"
                    type="checkbox"
                    v-model={disconnectOnRuleDisable.value}
                  />
                </div>
              </div>
            </div>
          </DialogWrapper>
        </>
      )

      const content = !isLargeCtrlsBar.value ? (
        <div class="flex flex-col gap-2 p-2">
          {hasProviders.value && (
            <div class="flex items-center gap-2">
              {tabs}
            </div>
          )}
          <div class="flex w-full items-center gap-2">
            {searchInput}
            {createButton}
            {refreshButton}
            {settingsModal}
          </div>
        </div>
      ) : (
        <div class="flex flex-wrap items-center gap-2 p-2">
          {hasProviders.value && tabs}
          {searchInput}
          <div class="flex-1"></div>
          {createButton}
          {refreshButton}
          {settingsModal}
        </div>
      )

      return <CtrlsBar>{content}</CtrlsBar>
    }
  },
})
