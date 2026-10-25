import { can, type Cap } from '@/assembly/backend'
import { ROUTE_NAME } from '@/constant'
import { renderRoutes } from '@/helper'
import { i18n } from '@/i18n'
import { language } from '@/store/settings'


import { activeBackend } from '@/store/setup'
// 移除顶层静态 import xxxPage
const ProxiesPage = () => import('@/views/ProxiesPage.vue')
const OverviewPage = () => import('@/views/OverviewPage.vue')
const ConnectionsPage = () => import('@/views/ConnectionsPage.vue')
const LogsPage = () => import('@/views/LogsPage.vue')
const InboundsPage = () => import('@/views/InboundsPage.vue')
const DnsPage = () => import('@/views/DnsPage.vue')
const SettingsPage = () => import('@/views/SettingsPage.vue')
const HomePage = () => import('@/views/HomePage.vue')

import { useTitle } from '@vueuse/core'
import { watch } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'

const childrenRouter = [
  {
    path: 'proxies',
    name: ROUTE_NAME.proxies,
    component: ProxiesPage,
  },
  {
    path: 'overview',
    name: ROUTE_NAME.overview,
    component: OverviewPage,
  },
  {
    path: 'connections',
    name: ROUTE_NAME.connections,
    component: ConnectionsPage,
  },
  {
    path: 'logs',
    name: ROUTE_NAME.logs,
    component: LogsPage,
  },
  //{
  //  path: 'tools',
  //  name: ROUTE_NAME.tools,
  //  component: () => import('@/views/ToolsPage.vue'),
  //},
  {
    path: 'inbounds',
    name: ROUTE_NAME.inbounds,
    component: InboundsPage
  },
  {
    path: 'users',
    redirect: { name: ROUTE_NAME.inbounds, query: { tab: 'users' } }
  },


  {
    path: 'rules',
    name: ROUTE_NAME.rules,
    component: () => import('@/views/RulesPage.vue')
  },
  {
    // Keep the previous route name/path working for notification links and bookmarks.
    path: 'routes',
    name: ROUTE_NAME.routes,
    redirect: { name: ROUTE_NAME.rules },
  },
  {
    path: 'dns',
    name: ROUTE_NAME.dns,
    component: DnsPage
  },
  {
    path: 'settings',
    name: ROUTE_NAME.settings,
    component: SettingsPage,
  }
]

// Routes that require a specific capability to be visitable.
const ROUTE_CAPABILITY: Partial<Record<string, Cap>> = {
  [ROUTE_NAME.rules]: 'rules',
  // [ROUTE_NAME.tools]: 'tools',
}

const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: ROUTE_NAME.proxies,
      component: HomePage,
      children: childrenRouter,
    },
    // {
    //   path: '/setup',
    //   name: ROUTE_NAME.setup,
    //   component: SetupPage,
    // },
    {
      path: '/:catchAll(.*)',
      redirect: ROUTE_NAME.proxies,
    },
  ],
})

const title = useTitle('zashboard')
const setTitleByName = (name: string | symbol | undefined) => {
  if (typeof name === 'string' && activeBackend.value) {
    const backend = activeBackend.value
    title.value = `${i18n.global.t(name)}`
  } else {
    title.value = 'zashboard'
  }
}

router.beforeEach((to, from) => {
  const toIndex = renderRoutes.value.findIndex((item) => item === to.name)
  const fromIndex = renderRoutes.value.findIndex((item) => item === from.name)

  if (toIndex === 0 && fromIndex === renderRoutes.value.length - 1) {
    to.meta.transition = 'slide-left'
  } else if (toIndex === renderRoutes.value.length - 1 && fromIndex === 0) {
    to.meta.transition = 'slide-right'
  } else if (toIndex !== fromIndex) {
    to.meta.transition = toIndex < fromIndex ? 'slide-right' : 'slide-left'
  }

  // if (!activeBackend.value && to.name !== ROUTE_NAME.setup) {
  //   router.push({ name: ROUTE_NAME.setup })
  //   return
  // }

  // Block navigation to a page the active backend's channels can't serve.
  const requiredCap = typeof to.name === 'string' ? ROUTE_CAPABILITY[to.name] : undefined
  if (requiredCap && !can(requiredCap)) {
    router.push({ name: ROUTE_NAME.proxies })
  }
})

router.afterEach((to) => {
  setTitleByName(to.name)
})

watch([language, activeBackend], () => {
  setTimeout(() => {
    setTitleByName(router.currentRoute.value.name)
  })
})

// 能力变化(切后端 / 内核探测出结果)后,把停留在已失效页面的用户送回代理页。
watch(renderRoutes, () => {
  const routeName = router.currentRoute.value.name
  const requiredCap = typeof routeName === 'string' ? ROUTE_CAPABILITY[routeName] : undefined
  if (requiredCap && !can(requiredCap)) {
    router.push({ name: ROUTE_NAME.proxies })
  }
})

export default router
