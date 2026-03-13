<script setup lang="ts">
import { computed, ref } from 'vue'
import api from '../api/client'

type HistoryResponse = {
  rows: Array<Record<string, unknown>>
  total: number
  limit: number
  offset: number
}

const text = {
  title: '操作日志',
  search: '查询',
  placeholder: '按用户/IP/SQL 实时检索',
  refresh: '刷新',
  exportExcel: '导出Excel',
  exporting: '导出中...',
  loading: '加载中...',
  empty: '暂无历史记录',
  historyLoadFailed: '历史记录加载失败',
  exportFailed: '导出 Excel 失败',
  display: '显示',
  total: '共',
  rowsUnit: '条',
  current: '当前',
  first: '首页',
  prev: '上一页',
  next: '下一页',
  last: '末页',
} as const

const rows = ref<Array<Record<string, unknown>>>([])
const loading = ref(false)
const error = ref('')
const search = ref('')
const total = ref(0)
const limit = ref(25)
const offset = ref(0)
const exporting = ref(false)
const pageSizeOptions = [10, 25, 50, 100]
let loadSeq = 0

function getInputValue(event: Event) {
  const target = event.target as HTMLInputElement | HTMLSelectElement | null
  return target?.value ?? ''
}

function onSearchInput(event: Event) {
  search.value = getInputValue(event)
  void load(0)
}

function onLimitChange(event: Event) {
  const next = Number(getInputValue(event))
  if (!Number.isFinite(next) || !pageSizeOptions.includes(next)) return
  limit.value = next
  void load(0, next)
}

function asDate(value: unknown) {
  const n = Number(value)
  if (!n) return '-'
  const d = new Date(n * 1000)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(
    d.getHours(),
  ).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`
}

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / Math.max(limit.value, 1))))
const page = computed(() => Math.floor(offset.value / Math.max(limit.value, 1)) + 1)
const rangeStart = computed(() => (total.value > 0 ? offset.value + 1 : 0))
const rangeEnd = computed(() => (total.value > 0 ? Math.min(offset.value + limit.value, total.value) : 0))

const visiblePages = computed(() => {
  const pages: number[] = []
  const maxButtons = 5
  const totalPageCount = totalPages.value
  const current = page.value
  let start = Math.max(1, current - Math.floor(maxButtons / 2))
  let end = Math.min(totalPageCount, start + maxButtons - 1)
  if (end - start + 1 < maxButtons) {
    start = Math.max(1, end - maxButtons + 1)
  }
  for (let i = start; i <= end; i += 1) {
    pages.push(i)
  }
  return pages
})

async function load(newOffset = 0, newLimit?: number) {
  const seq = ++loadSeq
  loading.value = true
  error.value = ''
  const reqLimit = newLimit ?? limit.value
  try {
    const { data } = await api.get<HistoryResponse>('/history', {
      params: {
        search: search.value || undefined,
        limit: reqLimit,
        offset: newOffset,
      },
    })
    if (seq !== loadSeq) return
    rows.value = data.rows
    total.value = data.total
    offset.value = data.offset
    limit.value = Number(data.limit) > 0 ? Number(data.limit) : reqLimit
  } catch (err: unknown) {
    if (seq !== loadSeq) return
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? text.historyLoadFailed
  } finally {
    if (seq !== loadSeq) return
    loading.value = false
  }
}

function setPage(target: number) {
  const nextPage = Math.min(Math.max(1, target), totalPages.value)
  const nextOffset = (nextPage - 1) * limit.value
  return load(nextOffset)
}

function nextPage() {
  if (page.value >= totalPages.value) return
  void setPage(page.value + 1)
}

function prevPage() {
  if (page.value <= 1) return
  void setPage(page.value - 1)
}

function resolveFileName(disposition: unknown) {
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

async function exportExcel() {
  if (exporting.value) return
  exporting.value = true
  error.value = ''
  try {
    const response = await api.get('/history/export', {
      params: { search: search.value || undefined },
      responseType: 'blob',
    })
    const fileName = resolveFileName(response.headers['content-disposition']) || `history_${Date.now()}.xlsx`
    const blob = new Blob([response.data], {
      type: response.headers['content-type'] || 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = fileName
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(url)
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? text.exportFailed
  } finally {
    exporting.value = false
  }
}

load(0)
</script>

<template>
  <section class="page-shell">
    <header class="page-header">
      <h2>{{ text.title }}</h2>
      <div class="header-actions">
        <div class="search-inline">
          <span class="search-label">{{ text.search }}</span>
          <input
            :value="search"
            class="search-input"
            :placeholder="text.placeholder"
            @input="onSearchInput"
            @compositionupdate="onSearchInput"
            @compositionend="onSearchInput"
          />
        </div>
        <button class="ghost-btn" :disabled="loading || exporting" @click="load(0)">{{ text.refresh }}</button>
        <button class="ghost-btn" :disabled="loading || exporting" @click="exportExcel">
          {{ exporting ? text.exporting : text.exportExcel }}
        </button>
      </div>
    </header>

    <p v-if="loading">{{ text.loading }}</p>

    <div v-if="!loading" class="history-toolbar">
      <label class="length-control">
        {{ text.display }}
        <select :value="String(limit)" @change="onLimitChange">
          <option v-for="size in pageSizeOptions" :key="`history-page-size-${size}`" :value="String(size)">
            {{ size }}
          </option>
        </select>
        {{ text.rowsUnit }}
      </label>
      <span class="table-meta">{{ text.total }} {{ total }} {{ text.rowsUnit }}，{{ text.current }} {{ rangeStart }} - {{ rangeEnd }} {{ text.rowsUnit }}</span>
    </div>

    <div v-if="!loading" class="table-wrap">
      <table class="history-table">
        <thead>
          <tr>
            <th>编号</th>
            <th class="history-time-col">时间</th>
            <th>用户</th>
            <th>IP</th>
            <th>SQL</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="String(row.id)">
            <td>{{ row.id }}</td>
            <td class="history-time-col">{{ asDate(row.date) }}</td>
            <td>{{ row.authuser || '-' }}</td>
            <td>{{ row.ip || '-' }}</td>
            <td class="mono">{{ row.sql }}</td>
          </tr>
          <tr v-if="rows.length === 0">
            <td colspan="5">{{ text.empty }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="table-pagination history-pagination section-gap" v-if="!loading && total > 0 && totalPages > 1">
      <button class="ghost-btn small-btn" :disabled="page <= 1" @click="setPage(1)">{{ text.first }}</button>
      <button class="ghost-btn small-btn" :disabled="page <= 1" @click="prevPage">{{ text.prev }}</button>
      <button
        v-for="p in visiblePages"
        :key="`history-page-${p}`"
        class="small-btn"
        :class="p === page ? 'page-btn-active' : 'ghost-btn'"
        @click="setPage(p)"
      >
        {{ p }}
      </button>
      <button class="ghost-btn small-btn" :disabled="page >= totalPages" @click="nextPage">{{ text.next }}</button>
      <button class="ghost-btn small-btn" :disabled="page >= totalPages" @click="setPage(totalPages)">{{ text.last }}</button>
    </div>
  </section>
</template>

<style scoped>
.history-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 10px;
}

.length-control {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: #4b647b;
}

.length-control select {
  width: auto;
  min-width: 92px;
}

.table-meta {
  color: #4b647b;
  font-size: 0.92rem;
}

.history-pagination {
  justify-content: flex-end;
  gap: 12px;
}

.history-pagination .small-btn {
  min-width: 46px;
}

.history-table {
  min-width: 980px;
}

.history-time-col {
  min-width: 176px;
  width: 176px;
  white-space: nowrap;
}

.page-btn-active {
  background: linear-gradient(180deg, #2f7fba, #1f6ca7);
  border-color: #1f6ca7;
}

@media (max-width: 900px) {
  .history-toolbar {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
