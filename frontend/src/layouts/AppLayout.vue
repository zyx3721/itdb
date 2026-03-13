<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import api from '../api/client'
import { useAuthStore } from '../stores/auth'
import { useNoticeStore } from '../stores/notice'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const notice = useNoticeStore()

type ViewHistoryEntry = {
  id: number
  url: string
  description: string
}

const recentHistoryResourceTitleMap: Record<string, string> = {
  items: '硬件',
  software: '软件',
  invoices: '单据',
  agents: '代理',
  files: '文件',
  contracts: '合同',
  locations: '地点',
  users: '用户',
  racks: '机架',
}

const recentHistoryDictionaryTitleMap: Record<string, string> = {
  itemtypes: '硬件类型',
  contracttypes: '合同类型',
  statustypes: '状态类型',
  filetypes: '文件类型',
  dpttypes: '所属部门',
  tags: '标记',
}

const mainNavItems = [
  { to: '/dashboard', label: '首页' },
  { to: '/resources/items', label: '硬件', tooltip: '硬件清单' },
  { to: '/resources/software', label: '软件', tooltip: '软件清单' },
  { to: '/resources/invoices', label: '单据', tooltip: '单据清单' },
  { to: '/resources/agents', label: '代理', tooltip: '供应商/采购方/承包方/厂商' },
  { to: '/resources/files', label: '文件', tooltip: '文档, 手册, 认购书, 授权书, ...' },
  { to: '/resources/contracts', label: '合同', tooltip: '维保, 租赁, ...' },
  { to: '/resources/locations', label: '地点' },
  { to: '/resources/users', label: '用户' },
  { to: '/resources/racks', label: '机架' },
]

const dictionaryNavItems = [
  { to: '/dictionaries/itemtypes', label: '硬件类型' },
  { to: '/dictionaries/contracttypes', label: '合同类型' },
  { to: '/dictionaries/statustypes', label: '状态类型' },
  { to: '/dictionaries/filetypes', label: '文件类型' },
  { to: '/dictionaries/dpttypes', label: '所属部门' },
  { to: '/dictionaries/tags', label: '标记' },
]

const toolNavItems = [
  { to: '/labels', label: '打印标签' },
  { to: '/reports', label: '报告' },
  { to: '/browse', label: '浏览数据' },
  { to: '/settings', label: '设置' },
  { to: '/history', label: '操作日志' },
]

const dbFileInput = ref<HTMLInputElement | null>(null)
const importingDatabase = ref(false)

function triggerDatabaseImport() {
  dbFileInput.value?.click()
}

async function handleDatabaseFileSelected(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return

  if (!file.name.endsWith('.db')) {
    notice.error('请选择 .db 格式的数据库文件')
    return
  }

  importingDatabase.value = true
  notice.info('正在导入数据库，请稍候...')

  try {
    const fd = new FormData()
    fd.append('file', file)
    await api.post('/import/database', fd, { timeout: 0 })
    notice.success('数据库导入成功，即将跳转到登录页...')
    auth.logout()
    setTimeout(() => {
      window.location.href = '/login'
    }, 1500)
  } catch {
    // api client 拦截器已处理错误提示
  } finally {
    importingDatabase.value = false
  }
}

const viewRefreshTick = ref(0)
const downloadingDatabaseBackup = ref(false)
const downloadingFullBackup = ref(false)
const recentViewHistory = ref<ViewHistoryEntry[]>([])
const recentHistoryQuickTip = '记录新增、编辑操作记录'
// Use path-based key so same-path query changes (e.g. create=1 cleanup) do not remount the page.
const routeKey = computed(() => `${route.path}::${viewRefreshTick.value}`)

function onNavClick(target: string, event: MouseEvent) {
  const resolved = router.resolve(target)
  if (resolved.path !== route.path) return

  // 当前菜单重复点击时，强制重建右侧视图
  event.preventDefault()
  viewRefreshTick.value += 1
}

function logout() {
  auth.logout()
  router.replace('/login')
}

function formatBackupDate() {
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${year}${month}${day}`
}

function resolveDownloadFileName(disposition: unknown) {
  if (typeof disposition !== 'string') return ''
  const utf8 = disposition.match(/filename\*=UTF-8''([^;]+)/i)
  if (utf8?.[1]) {
    try {
      return decodeURIComponent(utf8[1])
    } catch {
      return utf8[1]
    }
  }
  const plain = disposition.match(/filename="?([^";]+)"?/i)
  return plain?.[1] ?? ''
}

async function triggerDownload(url: string, fallbackFileName: string, pendingRef: typeof downloadingDatabaseBackup, failedText: string) {
  if (pendingRef.value) return

  pendingRef.value = true
  try {
    const response = await api.get(url, {
      responseType: 'blob',
      timeout: 0,
    })
    const fileName = resolveDownloadFileName(response.headers['content-disposition']) || fallbackFileName
    const contentType = response.headers['content-type'] || 'application/octet-stream'
    const blob = response.data instanceof Blob ? response.data : new Blob([response.data], { type: contentType })
    const objectURL = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = objectURL
    link.download = fileName
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(objectURL)
  } catch {
    notice.error(failedText)
  } finally {
    pendingRef.value = false
  }
}

async function downloadDatabaseBackup() {
  await triggerDownload('/backups/database', `itdb-${formatBackupDate()}.db`, downloadingDatabaseBackup, '数据库备份下载失败')
}

async function downloadFullBackup() {
  await triggerDownload('/backups/full', `itdb-${formatBackupDate()}.tar.gz`, downloadingFullBackup, '完全备份下载失败')
}

async function loadRecentViewHistory() {
  try {
    const { data } = await api.get<ViewHistoryEntry[]>('/view-history')
    recentViewHistory.value = Array.isArray(data) ? data : []
  } catch {
    recentViewHistory.value = []
  }
}

function handleViewHistoryUpdated() {
  void loadRecentViewHistory()
}

function formatRecentViewHistory(entry: ViewHistoryEntry) {
  try {
    const parsed = new URL(entry.url, window.location.origin)
    const pathname = parsed.pathname.replace(/\/+$/, '')
    const editID = parsed.searchParams.get('edit')
    const subtypeEditID = parsed.searchParams.get('subtypeEdit')

    if (pathname.startsWith('/resources/')) {
      const resourceKey = pathname.split('/').filter(Boolean).pop() ?? ''
      const title = recentHistoryResourceTitleMap[resourceKey]
      if (title && editID) return `${title}: ${editID}`
    }

    if (pathname.startsWith('/dictionaries/')) {
      if (subtypeEditID) return `合同子类型: ${subtypeEditID}`
      const dictionaryKey = pathname.split('/').filter(Boolean).pop() ?? ''
      const title = recentHistoryDictionaryTitleMap[dictionaryKey]
      if (title && editID) return `${title}: ${editID}`
    }
  } catch {
    // Fallback to stored description below.
  }
  return entry.description
}

// 空闲 1 小时自动跳转登录页
const IDLE_TIMEOUT = 60 * 60 * 1000
let idleTimer: ReturnType<typeof setTimeout> | null = null

function resetIdleTimer() {
  if (idleTimer) clearTimeout(idleTimer)
  idleTimer = setTimeout(() => {
    auth.logout()
    router.replace('/login')
  }, IDLE_TIMEOUT)
}

const idleEvents = ['mousemove', 'keydown', 'click', 'scroll'] as const

onMounted(() => {
  void loadRecentViewHistory()
  window.addEventListener('itdb:view-history-updated', handleViewHistoryUpdated)
  resetIdleTimer()
  idleEvents.forEach(e => window.addEventListener(e, resetIdleTimer))
})

onBeforeUnmount(() => {
  window.removeEventListener('itdb:view-history-updated', handleViewHistoryUpdated)
  if (idleTimer) clearTimeout(idleTimer)
  idleEvents.forEach(e => window.removeEventListener(e, resetIdleTimer))
})
</script>

<template>
  <div class="app-shell">
    <aside class="app-sidebar">
      <div class="brand-block">
        <h1>ITDB<br>资产管理系统</h1>
      </div>

      <nav class="nav-list">
        <RouterLink
          v-for="item in mainNavItems"
          :key="item.to"
          :to="item.to"
          class="nav-link"
          :class="{ 'quick-tip': !!item.tooltip }"
          :data-quick-tip="item.tooltip || null"
          @click="onNavClick(item.to, $event)"
        >
          {{ item.label }}
        </RouterLink>

        <hr class="nav-sep" />

        <RouterLink
          v-for="item in dictionaryNavItems"
          :key="item.to"
          :to="item.to"
          class="nav-link"
          @click="onNavClick(item.to, $event)"
        >
          {{ item.label }}
        </RouterLink>

        <hr class="nav-sep" />

        <RouterLink
          v-for="item in toolNavItems"
          :key="item.to"
          :to="item.to"
          class="nav-link"
          @click="onNavClick(item.to, $event)"
        >
          {{ item.label }}
        </RouterLink>
        <button
          type="button"
          class="nav-link nav-action-btn quick-tip"
          data-quick-tip="请先确保当前数据已备份"
          :disabled="importingDatabase"
          @click="triggerDatabaseImport"
        >
          {{ importingDatabase ? '导入中...' : '导入' }}
        </button>
        <input
          ref="dbFileInput"
          type="file"
          accept=".db"
          style="display: none"
          @change="handleDatabaseFileSelected"
        />
        <hr class="nav-sep" />
        <button
          type="button"
          class="nav-link nav-action-btn quick-tip"
          data-quick-tip="下载数据库文件。包含除上传到文件（文档）外的所有数据"
          :disabled="downloadingDatabaseBackup"
          @click="downloadDatabaseBackup"
        >
          <span class="nav-action-content">
            <svg class="nav-action-icon-svg" viewBox="0 0 24 24" aria-hidden="true">
              <ellipse cx="12" cy="5" rx="7" ry="2.5" fill="none" stroke="currentColor" stroke-width="1.8" />
              <path d="M5 5v10c0 1.38 3.13 2.5 7 2.5s7-1.12 7-2.5V5" fill="none" stroke="currentColor" stroke-width="1.8" />
              <path d="M5 10c0 1.38 3.13 2.5 7 2.5s7-1.12 7-2.5" fill="none" stroke="currentColor" stroke-width="1.8" />
              <path d="M5 15c0 1.38 3.13 2.5 7 2.5s7-1.12 7-2.5" fill="none" stroke="currentColor" stroke-width="1.8" />
            </svg>
            <span>{{ downloadingDatabaseBackup ? '数据库备份中...' : '数据库备份' }}</span>
          </span>
        </button>
        <button
          type="button"
          class="nav-link nav-action-btn quick-tip"
          data-quick-tip="下载整个项目备份。排除 node_modules 与 dist 目录"
          :disabled="downloadingFullBackup"
          @click="downloadFullBackup"
        >
          <span class="nav-action-content">
            <svg class="nav-action-icon-svg" viewBox="0 0 24 24" aria-hidden="true">
              <path d="M4 8.5 12 4l8 4.5-8 4.5L4 8.5Z" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round" />
              <path d="M4 8.5V16l8 4 8-4V8.5" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round" />
              <path d="M12 13v7" fill="none" stroke="currentColor" stroke-width="1.8" />
            </svg>
            <span>{{ downloadingFullBackup ? '完全备份中...' : '完全备份' }}</span>
          </span>
        </button>

        <section v-if="recentViewHistory.length > 0" class="sidebar-recent-history">
          <div
            class="sidebar-recent-history-title sidebar-recent-history-title-tip"
            :data-quick-tip="recentHistoryQuickTip"
          >最近历史记录</div>
          <div class="sidebar-recent-history-list">
            <RouterLink
              v-for="entry in recentViewHistory"
              :key="`recent-history-${entry.id}`"
              :to="entry.url"
              class="sidebar-recent-history-link"
            >
              {{ formatRecentViewHistory(entry) }}
            </RouterLink>
          </div>
        </section>
      </nav>

      <div class="sidebar-footer">
        <div class="user-pill">
          <span class="mono">{{ auth.user?.username }}</span>
          <small>{{ auth.isReadOnly ? '只读权限' : '完全权限' }}</small>
        </div>
        <button class="ghost-btn" @click="logout">退出登录</button>
      </div>
    </aside>

    <main class="app-main">
      <RouterView :key="routeKey" />
    </main>
  </div>
</template>
