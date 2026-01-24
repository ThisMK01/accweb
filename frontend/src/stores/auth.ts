import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { app } from '@/lib/accweb/types'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(window.localStorage.getItem('auth_token'))
  const admin = ref(window.localStorage.getItem('admin') === 'true')
  const mod = ref(window.localStorage.getItem('mod') === 'true')
  const read_only = ref(window.localStorage.getItem('read_only') === 'true')

  function login(data: app.AuthToken) {
    admin.value = data.admin!
    mod.value = data.mod!
    read_only.value = data.read_only!

    window.localStorage.setItem('auth_token', data.token!)
    window.localStorage.setItem('admin', String(admin.value))
    window.localStorage.setItem('mod', String(mod.value))
    window.localStorage.setItem('read_only', String(read_only.value))
  }

  function logout() {
    token.value = null
    admin.value = false
    mod.value = false
    read_only.value = false

    window.localStorage.removeItem('auth_token')
    window.localStorage.removeItem('admin')
    window.localStorage.removeItem('mod')
    window.localStorage.removeItem('read_only')
  }

  return { token, admin, mod, read_only, login, logout }
})
