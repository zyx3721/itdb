import { ref } from 'vue'
import { defineStore } from 'pinia'

export type NoticeType = 'success' | 'error' | 'info'

type AppNotice = {
  id: number
  type: NoticeType
  text: string
}

const DEFAULT_DURATION = 2200

export const useNoticeStore = defineStore('notice', () => {
  const current = ref<AppNotice | null>(null)
  const visible = ref(false)

  let timer: ReturnType<typeof setTimeout> | null = null
  let sequence = 0

  function clearTimer() {
    if (!timer) return
    clearTimeout(timer)
    timer = null
  }

  function show(type: NoticeType, text: string, duration = DEFAULT_DURATION) {
    const message = String(text ?? '').trim()
    if (!message) return

    clearTimer()
    sequence += 1
    current.value = {
      id: sequence,
      type,
      text: message,
    }
    visible.value = true
    timer = setTimeout(() => {
      visible.value = false
      timer = null
    }, duration)
  }

  function success(text: string, duration?: number) {
    show('success', text, duration)
  }

  function error(text: string, duration?: number) {
    show('error', text, duration)
  }

  function info(text: string, duration?: number) {
    show('info', text, duration)
  }

  function hide() {
    clearTimer()
    visible.value = false
  }

  return {
    current,
    visible,
    show,
    success,
    error,
    info,
    hide,
  }
})
