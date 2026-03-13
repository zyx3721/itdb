import axios from 'axios'
import { useNoticeStore } from '../stores/notice'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('itdb_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    const noticeStore = useNoticeStore()
    const responseMessage = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
    const status = (error as { response?: { status?: number } })?.response?.status
    const code = (error as { code?: string })?.code

    let message = (responseMessage ?? '').trim()
    if (!message) {
      if (code === 'ECONNABORTED') {
        message = '请求超时，请稍后重试'
      } else if (typeof navigator !== 'undefined' && navigator && navigator.onLine === false) {
        message = '网络不可用，请检查网络连接'
      } else if (status === 401) {
        message = '登录已失效，请重新登录'
        localStorage.removeItem('itdb_token')
        setTimeout(() => { window.location.href = '/login' }, 800)
      } else if (status === 403) {
        message = '当前账号无权限执行该操作'
      } else if (status === 404) {
        message = '请求的接口不存在'
      } else if (typeof status === 'number' && status >= 500) {
        message = '服务端异常，请稍后重试'
      } else {
        message = '操作失败，请稍后重试'
      }
    }

    noticeStore.error(message)
    return Promise.reject(error)
  },
)

export default api
