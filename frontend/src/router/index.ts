import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useBootstrapStore } from '../stores/bootstrap'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: () => import('../pages/LoginPage.vue') },
    { path: '/rack-view/:id', name: 'rack-view', component: () => import('../pages/RackViewPage.vue') },
    {
      path: '/',
      component: () => import('../layouts/AppLayout.vue'),
      children: [
        { path: '', redirect: '/dashboard' },
        { path: 'dashboard', name: 'dashboard', component: () => import('../pages/DashboardPage.vue') },
        { path: 'resources/:resource', name: 'resource', component: () => import('../pages/ResourcePage.vue') },
        { path: 'history', name: 'history', component: () => import('../pages/HistoryPage.vue') },
        { path: 'labels', name: 'labels', component: () => import('../pages/LabelsPage.vue') },
        { path: 'reports', name: 'reports', component: () => import('../pages/ReportsPage.vue') },
        { path: 'browse', name: 'browse', component: () => import('../pages/BrowsePage.vue') },
        { path: 'dictionaries/:tab?', name: 'dictionaries', component: () => import('../pages/DictionariesPage.vue') },
        { path: 'settings', name: 'settings', component: () => import('../pages/SettingsPage.vue') },
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/dashboard' },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  const bootstrap = useBootstrapStore()

  if (to.path === '/login') {
    if (auth.isAuthenticated) return '/dashboard'
    return true
  }

  if (!auth.isAuthenticated) return '/login'

  if (!auth.user) {
    try {
      await auth.loadMe()
    } catch {
      auth.logout()
      return '/login'
    }
  }

  if (!bootstrap.loaded) {
    try {
      await bootstrap.load()
    } catch {
      // allow page to render and show fetch errors per view
    }
  }

  return true
})

export default router
