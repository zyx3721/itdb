import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '../api/client'

type LookupRecord = Record<string, unknown>
type LookupMap = Record<string, LookupRecord[]>

export const useBootstrapStore = defineStore('bootstrap', () => {
  const lookups = ref<LookupMap>({})
  const loading = ref(false)
  const loaded = ref(false)

  async function load() {
    if (loading.value || loaded.value) return
    loading.value = true
    try {
      const { data } = await api.get('/bootstrap')
      lookups.value = data
      loaded.value = true
    } finally {
      loading.value = false
    }
  }

  function reset() {
    lookups.value = {}
    loaded.value = false
  }

  return { lookups, loading, loaded, load, reset }
})
