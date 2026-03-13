import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import api from '../api/client'
import { useBootstrapStore } from './bootstrap'

export type AuthUser = {
  id: number
  username: string
  userType: number
  userDesc?: string
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem('itdb_token') ?? '')
  const user = ref<AuthUser | null>(null)
  const pending = ref(false)

  const isAuthenticated = computed(() => Boolean(token.value))
  const isReadOnly = computed(() => (user.value?.userType ?? 1) !== 0 && (user.value?.username ?? '') !== 'admin')

  function setSession(newToken: string, newUser: AuthUser) {
    token.value = newToken
    user.value = newUser
    localStorage.setItem('itdb_token', newToken)
  }

  async function login(username: string, password: string, mode: 'local' | 'ldap' = 'local') {
    pending.value = true
    try {
      const { data } = await api.post('/auth/login', { username, password, mode })
      setSession(data.token, data.user)
    } finally {
      pending.value = false
    }
  }

  async function loadMe() {
    if (!token.value) return
    const { data } = await api.get('/auth/me')
    user.value = data
  }

  function logout() {
    const bootstrap = useBootstrapStore()
    token.value = ''
    user.value = null
    bootstrap.reset()
    localStorage.removeItem('itdb_token')
  }

  return { token, user, pending, isAuthenticated, isReadOnly, login, loadMe, logout }
})
