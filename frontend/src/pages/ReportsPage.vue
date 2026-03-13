<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api/client'

type ReportMeta = {
  name: string
  description: string
  chartType?: string
}

type ReportResult = {
  meta: ReportMeta
  rows: Array<Record<string, unknown>>
  chart: Array<{ x: string; y: number }>
}

const reports = ref<ReportMeta[]>([])
const active = ref('')
const rows = ref<Array<Record<string, unknown>>>([])
const chart = ref<Array<{ x: string; y: number }>>([])
const loading = ref(false)
const error = ref('')
const keyword = ref('')
const router = useRouter()
const reportPickerOpen = ref(false)
const reportPickerRoot = ref<HTMLElement | null>(null)

const reportTitleMap: Record<string, string> = {
  itemperagent: '每个厂商的硬件数量(代理)',
  softwareperagent: '每个厂商已安装的软件数量(代理)',
  invoicesperagent: '每个供应商的单据数量(代理)',
  itemsperlocation: '每个地点的硬件数量',
  percsupitems: '在保的硬件数量',
  itemlistperlocation: '按地点显示硬件',
  itemsendwarranty: '今日前后授权过期的硬件',
  allips: '指定了 IPv4 的硬件清单',
  noinvoice: '无单据的硬件',
  nolocation: '无地点的硬件',
  depreciation3: '硬件折旧价值 3 年',
  depreciation5: '硬件折旧价值 5 年',
}

const reportColumnOrderMap: Record<string, string[]> = {
  itemperagent: ['totalcount', 'Agent', 'ID'],
  softwareperagent: ['totalcount', 'Agent', 'ID'],
  invoicesperagent: ['totalcount', 'Agent', 'ID'],
  itemsperlocation: ['totalcount', 'Location'],
  percsupitems: ['Type', 'Items'],
  itemlistperlocation: ['ID', 'type', 'manufacturer', 'model', 'dnsname', 'Location'],
  itemsendwarranty: ['ID', 'ipv4', 'type', 'manufacturer', 'model', 'dnsname', 'label', 'RemainingDays'],
  allips: ['ID', 'ipv4', 'ipv6', 'type', 'manufacturer', 'model', 'dnsname', 'label'],
  noinvoice: ['ID', 'type', 'manufacturer', 'model', 'PurchaseDate'],
  nolocation: ['ID', 'type', 'manufacturer', 'model'],
  depreciation3: ['ID', 'type', 'manufacturer', 'model', 'PurchaseDate', 'PurchasePrice', 'Months', 'CurrentValue'],
  depreciation5: ['ID', 'type', 'manufacturer', 'model', 'PurchaseDate', 'PurchasePrice', 'Months', 'CurrentValue'],
}

const reportColumnLabelMap: Record<string, string> = {
  ID: '编号',
  totalcount: '数量',
  Agent: '厂商',
  Location: '地点',
  Type: '类型',
  Items: '数量',
  type: '类型',
  manufacturer: '厂商',
  model: '型号',
  dnsname: '业务跳线',
  label: '标签',
  RemainingDays: '维保剩余天数',
  ipv4: 'IPv4',
  ipv6: 'IPv6',
  PurchaseDate: '采购日期',
  PurchasePrice: '采购价格',
  Months: '月数',
  CurrentValue: '当前价值',
}

const reportEditTargetMap: Record<string, 'items' | 'agents'> = {
  itemperagent: 'agents',
  softwareperagent: 'agents',
  invoicesperagent: 'agents',
  itemlistperlocation: 'items',
  itemsendwarranty: 'items',
  allips: 'items',
  noinvoice: 'items',
  nolocation: 'items',
  depreciation3: 'items',
  depreciation5: 'items',
}

const activeMeta = computed(() => reports.value.find((r) => r.name === active.value))
const activeReportTitle = computed(() => {
  if (!active.value) return ''
  return reportTitleMap[active.value] ?? activeMeta.value?.description ?? active.value
})
const columns = computed(() => {
  const rowKeys = Object.keys(rows.value[0] ?? {})
  const preferred = reportColumnOrderMap[active.value] ?? rowKeys
  const ordered = preferred.filter((key) => rowKeys.includes(key))
  const remaining = rowKeys.filter((key) => !ordered.includes(key))
  return [...ordered, ...remaining]
})

const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return rows.value
  return rows.value.filter((row) =>
    columns.value.some((key) => String(row[key] ?? '').toLowerCase().includes(q)),
  )
})

const tableRows = computed(() => filteredRows.value.map((row, idx) => ({ index: idx + 1, row })))

function columnDisplayName(key: string) {
  return reportColumnLabelMap[key] ?? key
}

function isIDColumn(key: string) {
  return key.trim().toLowerCase() === 'id'
}

function parsePositiveID(value: unknown) {
  const id = Number.parseInt(String(value ?? '').trim(), 10)
  return Number.isFinite(id) && id > 0 ? id : 0
}

function canOpenRowEditor(key: string, row: Record<string, unknown>) {
  if (!isIDColumn(key)) return false
  if (!reportEditTargetMap[active.value]) return false
  return parsePositiveID(row[key]) > 0
}

function openRowEditorByID(value: unknown) {
  const target = reportEditTargetMap[active.value]
  const id = parsePositiveID(value)
  if (!target || !id) return
  void router.push({ path: `/resources/${target}`, query: { edit: String(id) } })
}

function reportDisplayName(report: ReportMeta) {
  return reportTitleMap[report.name] ?? report.description ?? report.name
}

function toggleReportPicker() {
  reportPickerOpen.value = !reportPickerOpen.value
}

function selectReport(name: string) {
  reportPickerOpen.value = false
  if (!name) return
  if (active.value === name) {
    keyword.value = ''
    void loadReport()
    return
  }
  active.value = name
}

function onReportPickerPointerDown(event: PointerEvent) {
  const root = reportPickerRoot.value
  const target = event.target as Node | null
  if (root && target && !root.contains(target)) {
    reportPickerOpen.value = false
  }
}

async function loadReports() {
  const { data } = await api.get<ReportMeta[]>('/reports')
  reports.value = data
  if (!active.value && reports.value.length > 0) {
    active.value = reports.value[0]?.name ?? ''
  }
}

async function loadReport() {
  if (!active.value) {
    rows.value = []
    chart.value = []
    return
  }
  loading.value = true
  error.value = ''
  try {
    const { data } = await api.get<ReportResult>(`/reports/${active.value}`, { params: { limit: 1000 } })
    rows.value = data.rows ?? []
    chart.value = data.chart ?? []
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '报表加载失败'
  } finally {
    loading.value = false
  }
}

watch(active, () => {
  keyword.value = ''
  void loadReport()
})

onMounted(async () => {
  document.addEventListener('pointerdown', onReportPickerPointerDown)
  try {
    await loadReports()
    await loadReport()
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '报表目录加载失败'
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onReportPickerPointerDown)
})
</script>

<template>
  <section class="page-shell">
    <header class="page-header">
      <h2>报告</h2>
      <div class="header-actions">
        <div class="search-inline">
          <span class="search-label">选择报告</span>
          <div ref="reportPickerRoot" class="report-picker">
            <button type="button" class="report-picker-btn" :aria-expanded="reportPickerOpen ? 'true' : 'false'" @click="toggleReportPicker">
              <span>{{ activeReportTitle || '请选择报告' }}</span>
              <span class="report-picker-caret">{{ reportPickerOpen ? '▲' : '▼' }}</span>
            </button>
            <div v-if="reportPickerOpen" class="report-picker-menu">
              <button
                v-for="report in reports"
                :key="report.name"
                type="button"
                class="report-picker-option"
                :class="{ active: active === report.name }"
                @click="selectReport(report.name)"
              >
                {{ reportDisplayName(report) }}
              </button>
              <div v-if="reports.length === 0" class="report-picker-empty">暂无报告</div>
            </div>
          </div>
        </div>
        <div class="search-inline">
          <span class="search-label">查询</span>
          <input v-model.trim="keyword" class="search-input" type="text" placeholder="输入关键词过滤当前报告" />
        </div>
        <button class="ghost-btn" @click="loadReport">刷新</button>
      </div>
    </header>

    <p class="muted-text">{{ activeReportTitle }}</p>
    <p v-if="loading">加载中...</p>

    <div v-else class="report-grid">
      <div v-if="chart.length > 0" class="metric-grid">
        <article v-for="item in chart" :key="item.x" class="metric-card">
          <h3>{{ item.x }}</h3>
          <p>{{ item.y }}</p>
        </article>
      </div>

      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>#</th>
              <th v-for="key in columns" :key="key">{{ columnDisplayName(key) }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in tableRows" :key="item.index">
              <td>{{ item.index }}</td>
              <td v-for="key in columns" :key="key">
                <button
                  v-if="canOpenRowEditor(key, item.row)"
                  type="button"
                  class="report-id-link-btn"
                  @click="openRowEditorByID(item.row[key])"
                >
                  {{ item.row[key] }}
                </button>
                <span v-else>{{ item.row[key] }}</span>
              </td>
            </tr>
            <tr v-if="tableRows.length === 0">
              <td :colspan="(columns.length || 0) + 1">暂无报表数据</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>
</template>

<style scoped>
.report-picker {
  position: relative;
  min-width: 260px;
}

.report-picker-btn {
  width: 100%;
  display: inline-flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 0.5rem 0.65rem;
  border: 1px solid var(--line);
  border-radius: 10px;
  background: #fff;
  color: var(--text);
  font: inherit;
  font-weight: 400;
  box-shadow: none;
  transform: none;
}

.report-picker-btn:hover,
.report-picker-btn:focus-visible {
  transform: none;
  border-color: #a8b9ca;
  outline: none;
}

.report-picker-caret {
  flex: 0 0 auto;
  color: var(--text-muted);
  font-size: 0.72rem;
}

.report-picker-menu {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  min-width: 100%;
  max-height: 320px;
  overflow: auto;
  padding: 6px;
  display: grid;
  gap: 4px;
  background: #fff;
  border: 1px solid var(--line);
  border-radius: 12px;
  box-shadow: 0 14px 30px rgba(15, 31, 48, 0.16);
  z-index: 30;
}

.report-picker-option {
  width: 100%;
  text-align: left;
  justify-content: flex-start;
  border-radius: 8px;
  padding: 0.45rem 0.6rem;
  border: 1px solid transparent;
  background: #fff;
  color: var(--text);
  font: inherit;
  font-weight: 400;
  box-shadow: none;
  transform: none;
}

.report-picker-option:hover,
.report-picker-option:focus-visible {
  transform: none;
  background: #f7fafc;
  border-color: #d9e2ec;
  outline: none;
}

.report-picker-option.active {
  background: #e8f3ff;
  border-color: #97bcdd;
  color: #1d5f95;
}

.report-picker-empty {
  padding: 8px 10px;
  color: var(--muted);
  font-size: 0.94rem;
}

.report-id-link-btn {
  min-width: 46px;
  padding: 0.24rem 0.58rem;
  border: 1px solid #8fb6d8;
  border-radius: 999px;
  background: linear-gradient(180deg, #f8fcff 0%, #e5f1fb 100%);
  color: #15507d;
  font: inherit;
  font-weight: 700;
  line-height: 1.2;
  box-shadow: none;
  transform: none;
}

.report-id-link-btn:hover,
.report-id-link-btn:focus-visible {
  transform: none;
  border-color: #5d95c5;
  background: linear-gradient(180deg, #ffffff 0%, #d8ebfb 100%);
  color: #0f4269;
  outline: none;
}
</style>
