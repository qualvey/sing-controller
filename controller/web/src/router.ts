import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/outbounds' },
    { path: '/inbounds', name: 'inbounds', component: () => import('./views/InboundsView.vue') },
    { path: '/outbounds', name: 'outbounds', component: () => import('./views/OutboundsView.vue') },
    { path: '/routes', name: 'routes', component: () => import('./views/RoutesView.vue') },
    { path: '/config', name: 'config', component: () => import('./views/ConfigView.vue') },
    { path: '/settings', name: 'settings', component: () => import('./views/SettingsView.vue') }
  ]
})

export default router
