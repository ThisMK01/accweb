import { createRouter, createWebHistory } from 'vue-router'
import { token } from '@/lib/accweb/client'

import HomePage from '@/pages/HomePage.vue'
import SettingsPage from '@/pages/SettingsPage.vue'
import LoginPage from '@/pages/LoginPage.vue'
import InstancePage from '@/pages/InstancePage.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', component: HomePage },
    { path: '/login', component: LoginPage, meta: { title: 'Login' } },
    { path: '/settings', component: SettingsPage, meta: { title: 'Settings' } },
    { path: '/instance/:id', component: InstancePage, meta: { title: 'Instance' } },
  ],
})

router.beforeEach(async (to, from, next) => {
  document.title = 'AccWeb - ' + (to.meta.title || 'Home')

  if (to.path !== '/login') {
    try {
      await token()
    } catch (_) { }
  }

  next()
})


export default router
