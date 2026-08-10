import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/outbounds' },
    { path: '/inbounds', name: 'inbounds', component: () => import('./views/InboundsView.vue') },
    { path: '/proxies', name: 'proxies', component: () => import('./views/ProxiesView.vue') },
    { path: '/connections', name: 'connections', component: () => import('./views/ConnectionsView.vue') },
    { path: '/logs', name: 'logs', component: () => import('./views/LogsView.vue') },
    { path: '/outbounds', name: 'outbounds', component: () => import('./views/OutboundsView.vue') },
    { path: '/routes', name: 'routes', component: () => import('./views/RoutesView.vue') },
    { path: '/rule-sets', name: 'rule-sets', component: () => import('./views/RuleSetsView.vue') },
    { path: '/dns', name: 'dns', component: () => import('./views/DnsView.vue') },
    { path: '/certificate', name: 'certificate', component: () => import('./views/CertificateView.vue') },
    { path: '/diagnostics', name: 'diagnostics', component: () => import('./views/DiagnosticsView.vue') },
    { path: '/config', name: 'config', component: () => import('./views/ConfigView.vue') },
    { path: '/settings', name: 'settings', component: () => import('./views/SettingsView.vue') }
  ]
})

export default router
