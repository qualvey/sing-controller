import type { ComputedRef, InjectionKey } from 'vue'

export interface TabsContext {
  modelValue: ComputedRef<string | number | undefined>
  changeTab: (value: string | number) => void
}

export const tabsContextKey: InjectionKey<TabsContext> = Symbol('TabsContext')
