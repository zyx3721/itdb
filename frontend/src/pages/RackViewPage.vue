<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '../api/client'
import { useBootstrapStore } from '../stores/bootstrap'

type RackViewDepth = 'F' | 'M' | 'B'

type RackItemRow = {
  id: number
  manufacturer: string
  model: string
  label: string
  statusText: string
  statusColor: string
  ipv4: string
  uSize: number
  rackID: number
  rackPosition: number
  rackPosDepth: number
}

type RackViewCell = {
  key: string
  kind: 'empty' | 'item'
  colspan: number
  rowspan: number
  itemID?: number
  title?: string
  subtitle?: string
  tip?: string
  statusClass?: string
  style?: Record<string, string>
  highlight?: boolean
}

type RackViewRow = {
  unit: number
  cells: RackViewCell[]
}

type RackViewBuildResult = {
  rows: RackViewRow[]
  warnings: string[]
  moreItems: RackItemRow[]
}

const route = useRoute()
const router = useRouter()
const bootstrap = useBootstrapStore()

const loading = ref(false)
const loadError = ref('')
const rack = ref<Record<string, unknown> | null>(null)
const rackItems = ref<RackItemRow[]>([])

const rackID = computed(() => {
  const value = Number.parseInt(String(route.params.id ?? '').trim(), 10)
  return Number.isFinite(value) && value > 0 ? value : 0
})

const highlightItemID = computed(() => {
  const raw = Array.isArray(route.query.highlight) ? route.query.highlight[0] : route.query.highlight
  const value = Number.parseInt(String(raw ?? '').trim(), 10)
  return Number.isFinite(value) && value > 0 ? value : 0
})

const rackTitle = computed(() => {
  const row = rack.value
  if (!row) return ''
  const label = String(row.label ?? '').trim() || `机架${rackID.value}`
  const units = Number.parseInt(String(row.usize ?? '').trim(), 10)
  return units > 0 ? `${label},${units}U 晟图` : `${label} 晟图`
})

const rackMetaText = computed(() => {
  const row = rack.value
  if (!row) return ''
  const model = String(row.model ?? '').trim()
  const locationRows = (bootstrap.lookups.locations ?? []) as Record<string, unknown>[]
  const areaRows = (bootstrap.lookups.locareas ?? []) as Record<string, unknown>[]
  const location = locationRows.find((entry) => Number(entry.id ?? 0) === Number(row.locationid ?? 0))
  const area = areaRows.find((entry) => Number(entry.id ?? 0) === Number(row.locareaid ?? 0))
  const parts = [
    `编号: ${rackID.value}`,
    model ? `型号: ${model}` : '',
    location ? `地点: ${String(location.name ?? '').trim()}` : '',
    area ? `区域: ${String(area.areaname ?? '').trim()}` : '',
  ].filter(Boolean)
  return parts.join(' / ')
})

function naturalCompare(a: unknown, b: unknown) {
  return String(a ?? '').localeCompare(String(b ?? ''), 'zh-CN', { numeric: true, sensitivity: 'base' })
}

function getRackDepthMask(raw: unknown) {
  const value = Number(raw ?? 0)
  return Number.isFinite(value) && value > 0 ? value : 0
}

function firstRackIPv4(ipv4: string) {
  return ipv4
    .split(',')
    .map((part) => part.trim())
    .find(Boolean) ?? ''
}

function getRackItemStatusClass(statusText: string) {
  const status = statusText.trim().toLowerCase()
  if (!status) return ''
  if (status.includes('库存') || status.includes('stored')) return 'rack-view-status-stored'
  if (status.includes('故障') || status.includes('defective')) return 'rack-view-status-defective'
  if (status.includes('报废') || status.includes('obsolete')) return 'rack-view-status-obsolete'
  return 'rack-view-status-active'
}

function getStatusColorByText(statusText: string) {
  const rows = (bootstrap.lookups.statustypes ?? []) as Record<string, unknown>[]
  const match = rows.find((row) => String(row.statusdesc ?? '').trim() === statusText.trim())
  const color = String(match?.color ?? '').trim()
  return /^#([\da-f]{3}|[\da-f]{6})$/i.test(color) ? color : ''
}

function getReadableTextColor(backgroundColor: string) {
  const raw = backgroundColor.trim().replace('#', '')
  if (!/^[\da-f]{3}$|^[\da-f]{6}$/i.test(raw)) return '#143d5c'
  const hex = raw.length === 3 ? raw.split('').map((part) => `${part}${part}`).join('') : raw
  const red = Number.parseInt(hex.slice(0, 2), 16)
  const green = Number.parseInt(hex.slice(2, 4), 16)
  const blue = Number.parseInt(hex.slice(4, 6), 16)
  const luminance = (red * 299 + green * 587 + blue * 114) / 1000
  return luminance >= 160 ? '#173651' : '#ffffff'
}

function getRackItemStatusStyle(statusText: string, statusColor = ''): Record<string, string> {
  const color = String(statusColor || getStatusColorByText(statusText)).trim()
  if (!color) return {}
  return {
    '--rack-cell-bg': color,
    '--rack-cell-fg': getReadableTextColor(color),
  }
}

function getRackItemTitle(row: RackItemRow) {
  return [`${row.manufacturer || '-'}`, `${row.model || '-'}`, row.id > 0 ? `[ID:${row.id}]` : '[新建]'].join(' ').trim()
}

function getRackItemSubtitle(row: RackItemRow) {
  const parts: string[] = []
  if (row.label.trim()) parts.push(row.label.trim())
  const firstIP = firstRackIPv4(row.ipv4)
  if (firstIP) parts.push(`[ip:${firstIP}]`)
  return parts.join(' ')
}

function getRackItemTip(row: RackItemRow) {
  const statusText = row.statusText.trim() ? `状态：${row.statusText.trim()}` : '状态：使用中'
  const sizeText = row.uSize > 0 ? `${row.uSize}U` : '-'
  const posText = row.rackPosition > 0 ? `${row.rackPosition}` : '-'
  return `编号：${row.id} / ${statusText} / 位置：${posText}U / 高度：${sizeText}`
}

function buildRackViewData(totalUnits: number, reverse: boolean, itemRows: RackItemRow[], highlightID: number): RackViewBuildResult {
  if (totalUnits <= 0) return { rows: [], warnings: [], moreItems: [] }

  const rackRows: Record<number, Partial<Record<RackViewDepth | `${RackViewDepth}T`, number>>> = {}
  const itemMap = new Map<number, RackItemRow>()
  const warnings: string[] = []
  const moreItems: RackItemRow[] = []
  const sourceRows = [...itemRows].sort((a, b) => naturalCompare(a.id, b.id))

  for (const item of sourceRows) {
    itemMap.set(item.id, item)
    const rackPosition = Number(item.rackPosition ?? 0)
    const units = Number(item.uSize ?? 0)
    const depthMask = getRackDepthMask(item.rackPosDepth)
    if (!Number.isFinite(rackPosition) || rackPosition <= 0 || !Number.isFinite(units) || units <= 0 || depthMask <= 0) {
      moreItems.push(item)
      continue
    }

    if (reverse) {
      if (rackPosition + units - 1 > totalUnits) {
        warnings.push(`硬件 ${item.id}（${item.model || '-'}）超出机架边界`)
        continue
      }
      for (let pos = rackPosition; pos < rackPosition + units; pos += 1) {
        const rowState = (rackRows[pos] ??= {})
        const isTop = pos === rackPosition ? 1 : 0

        if ((depthMask & 4) === 4 && rowState.F && rowState.F !== item.id) warnings.push(`第 ${pos}U 前侧位置冲突：硬件 ${item.id} 与 ${rowState.F}`)
        if ((depthMask & 2) === 2 && rowState.M && rowState.M !== item.id) warnings.push(`第 ${pos}U 中部位置冲突：硬件 ${item.id} 与 ${rowState.M}`)
        if ((depthMask & 1) === 1 && rowState.B && rowState.B !== item.id) warnings.push(`第 ${pos}U 后侧位置冲突：硬件 ${item.id} 与 ${rowState.B}`)

        if ((depthMask & 4) === 4) {
          rowState.F = item.id
          rowState.FT = isTop
        }
        if ((depthMask & 2) === 2) {
          rowState.M = item.id
          rowState.MT = isTop
        }
        if ((depthMask & 1) === 1) {
          rowState.B = item.id
          rowState.BT = isTop
        }
      }
      continue
    }

    if (rackPosition - units + 1 < 1) {
      warnings.push(`硬件 ${item.id}（${item.model || '-'}）超出机架边界`)
      continue
    }

    for (let pos = rackPosition; pos > rackPosition - units; pos -= 1) {
      const rowState = (rackRows[pos] ??= {})
      const isTop = pos === rackPosition ? 1 : 0

      if ((depthMask & 4) === 4 && rowState.F && rowState.F !== item.id) warnings.push(`第 ${pos}U 前侧位置冲突：硬件 ${item.id} 与 ${rowState.F}`)
      if ((depthMask & 2) === 2 && rowState.M && rowState.M !== item.id) warnings.push(`第 ${pos}U 中部位置冲突：硬件 ${item.id} 与 ${rowState.M}`)
      if ((depthMask & 1) === 1 && rowState.B && rowState.B !== item.id) warnings.push(`第 ${pos}U 后侧位置冲突：硬件 ${item.id} 与 ${rowState.B}`)

      if ((depthMask & 4) === 4) {
        rowState.F = item.id
        rowState.FT = isTop
      }
      if ((depthMask & 2) === 2) {
        rowState.M = item.id
        rowState.MT = isTop
      }
      if ((depthMask & 1) === 1) {
        rowState.B = item.id
        rowState.BT = isTop
      }
    }
  }

  const unitSequence = reverse
    ? Array.from({ length: totalUnits }, (_, index) => index + 1)
    : Array.from({ length: totalUnits }, (_, index) => totalUnits - index)

  const rows: RackViewRow[] = []
  for (const unit of unitSequence) {
    const state = rackRows[unit] ?? {}
    const cells: RackViewCell[] = []
    let cell = 1
    let colspan = 1

    if (state.FT) {
      const itemID = Number(state.F ?? 0)
      const item = itemMap.get(itemID)
      if (item) {
        if (state.F !== state.M) colspan = 1
        else if (state.F === state.M && state.M !== state.B) colspan = 2
        else colspan = 3
        cells.push({
          key: `${unit}-F-${itemID}`,
          kind: 'item',
          colspan,
          rowspan: Math.max(1, item.uSize || 1),
          itemID,
          title: getRackItemTitle(item),
          subtitle: getRackItemSubtitle(item),
          tip: getRackItemTip(item),
          statusClass: getRackItemStatusClass(item.statusText),
          style: getRackItemStatusStyle(item.statusText, item.statusColor),
          highlight: itemID === highlightID,
        })
      }
    } else if (!state.F) {
      cells.push({ key: `${unit}-F-empty`, kind: 'empty', colspan: 1, rowspan: 1 })
      colspan = 1
    } else {
      colspan = 1
    }
    cell += colspan

    if (cell === 2) {
      if (state.MT) {
        const itemID = Number(state.M ?? 0)
        const item = itemMap.get(itemID)
        if (item) {
          if (state.M !== state.B) colspan = 1
          else colspan = 2
          cells.push({
            key: `${unit}-M-${itemID}`,
            kind: 'item',
            colspan,
            rowspan: Math.max(1, item.uSize || 1),
            itemID,
            title: getRackItemTitle(item),
            subtitle: getRackItemSubtitle(item),
            tip: getRackItemTip(item),
            statusClass: getRackItemStatusClass(item.statusText),
            style: getRackItemStatusStyle(item.statusText, item.statusColor),
            highlight: itemID === highlightID,
          })
        }
      } else if (!state.M) {
        cells.push({ key: `${unit}-M-empty`, kind: 'empty', colspan: 1, rowspan: 1 })
        colspan = 1
      } else {
        colspan = 1
      }
      cell += colspan
    }

    if (cell === 3) {
      if (state.BT) {
        const itemID = Number(state.B ?? 0)
        const item = itemMap.get(itemID)
        if (item) {
          cells.push({
            key: `${unit}-B-${itemID}`,
            kind: 'item',
            colspan: 1,
            rowspan: Math.max(1, item.uSize || 1),
            itemID,
            title: getRackItemTitle(item),
            subtitle: getRackItemSubtitle(item),
            tip: getRackItemTip(item),
            statusClass: getRackItemStatusClass(item.statusText),
            style: getRackItemStatusStyle(item.statusText, item.statusColor),
            highlight: itemID === highlightID,
          })
        }
      } else if (!state.B) {
        cells.push({ key: `${unit}-B-empty`, kind: 'empty', colspan: 1, rowspan: 1 })
      }
    }

    rows.push({ unit, cells })
  }

  return { rows, warnings: Array.from(new Set(warnings)), moreItems }
}

const rackTotalUnits = computed(() => {
  const total = Number(rack.value?.usize ?? 0)
  return Number.isFinite(total) && total > 0 ? total : 0
})

const rackReverse = computed(() => Number(rack.value?.revnums ?? 0) === 1)

const rackViewData = computed(() => buildRackViewData(rackTotalUnits.value, rackReverse.value, rackItems.value, highlightItemID.value))

async function loadRackView() {
  if (!rackID.value) {
    loadError.value = '无效的机架编号'
    return
  }

  if (!bootstrap.loaded) {
    await bootstrap.load()
  }

  loading.value = true
  loadError.value = ''
  try {
    const [rackResponse, itemsResponse] = await Promise.all([
      api.get(`/racks/${rackID.value}`),
      api.get('/items', { params: { limit: -1, offset: 0 } }),
    ])

    rack.value = rackResponse.data as Record<string, unknown>
    const itemRows = Array.isArray(itemsResponse.data) ? (itemsResponse.data as Record<string, unknown>[]) : []
    rackItems.value = itemRows
      .filter((row) => Number(row.rackid ?? 0) === rackID.value)
      .map((row) => ({
        id: Number(row.id ?? 0),
        manufacturer: String(row.manufacturer ?? ''),
        model: String(row.model ?? ''),
        label: String(row.label ?? ''),
        statusText: String(row.status ?? ''),
        statusColor: String(row.statuscolor ?? row.statusColor ?? ''),
        ipv4: String(row.ipv4 ?? ''),
        uSize: Number(row.usize ?? row.uSize ?? 0),
        rackID: Number(row.rackid ?? 0),
        rackPosition: Number(row.rackposition ?? row.rackPosition ?? 0),
        rackPosDepth: Number(row.rackposdepth ?? row.rackPosDepth ?? 0),
      }))
      .filter((row) => row.id > 0)
  } catch {
    rack.value = null
    rackItems.value = []
    loadError.value = '机架晟图加载失败'
  } finally {
    loading.value = false
  }
}

function openItemInNewWindow(id: number | undefined) {
  if (!id) return
  const target = router.resolve({ path: '/resources/items', query: { edit: String(id) } })
  window.open(target.href, '_blank', 'noopener')
}

function closeWindow() {
  window.close()
}

watch(() => [rackID.value, highlightItemID.value] as const, () => {
  void loadRackView()
})

onMounted(() => {
  void loadRackView()
})
</script>

<template>
  <main class="rack-view-page">
    <header class="rack-view-page-head">
      <div>
        <h1>{{ rackTitle || '机架晟图' }}</h1>
        <p v-if="rackMetaText" class="rack-view-page-meta">{{ rackMetaText }}</p>
      </div>
      <button class="rack-view-page-close" type="button" @click="closeWindow">关闭</button>
    </header>

    <section class="rack-view-page-panel">
      <p v-if="loading" class="rack-view-page-empty">加载中...</p>
      <p v-else-if="loadError" class="rack-view-page-error">{{ loadError }}</p>
      <template v-else>
        <div v-if="rackViewData.rows.length > 0" class="rack-view-wrap">
          <div class="rack-view-scroller">
            <table class="rack-view-table">
              <colgroup>
                <col class="rack-view-ru-col" />
                <col class="rack-view-main-col" />
                <col class="rack-view-main-col" />
                <col class="rack-view-main-col" />
              </colgroup>
              <thead>
                <tr>
                  <th>RU</th>
                  <th>前</th>
                  <th>中</th>
                  <th>后</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in rackViewData.rows" :key="`rack-view-${row.unit}`">
                  <td class="rack-view-ru">{{ row.unit }}</td>
                  <template v-for="cell in row.cells" :key="cell.key">
                    <td
                      v-if="cell.rowspan > 0"
                      :rowspan="cell.rowspan"
                      :colspan="cell.colspan"
                      :class="[
                        'rack-view-cell',
                        cell.kind === 'empty' ? 'rack-view-empty' : 'rack-view-occupied',
                        cell.highlight ? 'rack-view-highlight' : '',
                      ]"
                      :style="cell.style"
                    >
                      <template v-if="cell.kind === 'item'">
                        <button class="rack-view-link" type="button" :title="cell.tip" @click="openItemInNewWindow(cell.itemID)">
                          <span class="rack-view-title">{{ cell.title }}</span>
                          <span v-if="cell.subtitle" class="rack-view-subtitle">{{ cell.subtitle }}</span>
                        </button>
                      </template>
                    </td>
                  </template>
                </tr>
                <tr>
                  <td colspan="4" class="rack-view-base" />
                </tr>
                <tr>
                  <td class="rack-view-wheel-spacer" />
                  <td class="rack-view-wheel"><img src="/images/rackwheel.png" alt="机架底轮" /></td>
                  <td class="rack-view-wheel-spacer" />
                  <td class="rack-view-wheel"><img src="/images/rackwheel.png" alt="机架底轮" /></td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        <p v-else class="rack-view-page-empty">{{ rackTotalUnits > 0 ? '当前机架暂无可显示的晟图内容' : '该机架未设置高度(U)' }}</p>

        <div v-if="rackViewData.moreItems.length > 0" class="rack-view-side-note">
          <h5>已分配到该机架但未设置机架位置、深度位或高度的硬件</h5>
          <ul>
            <li v-for="row in rackViewData.moreItems" :key="`rack-view-more-${row.id}`">
              <button class="rack-view-side-link" type="button" @click="openItemInNewWindow(row.id)">#{{ row.id }} {{ row.manufacturer || '-' }} {{ row.model || '-' }}</button>
            </li>
          </ul>
        </div>

        <div v-if="rackViewData.warnings.length > 0" class="rack-view-warning-list">
          <p v-for="warning in rackViewData.warnings" :key="warning">{{ warning }}</p>
        </div>
      </template>
    </section>
  </main>
</template>

<style scoped>
.rack-view-page {
  min-height: 100vh;
  padding: 24px;
  background: linear-gradient(180deg, #edf4fb 0%, #f8fbff 100%);
  color: #183b5b;
}

.rack-view-page-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
}

.rack-view-page-head h1 {
  margin: 0;
  font-size: 1.4rem;
  color: #0f4d83;
}

.rack-view-page-meta {
  margin: 6px 0 0;
  color: #59708a;
}

.rack-view-page-close {
  border: 1px solid #8db1d4;
  background: linear-gradient(180deg, #ffffff 0%, #e7f1fb 100%);
  color: #1d5f95;
}

.rack-view-page-panel {
  border: 1px solid #c7d9ea;
  border-radius: 10px;
  background: #fff;
  padding: 16px;
  box-shadow: 0 10px 24px rgba(16, 42, 67, 0.08);
}

.rack-view-page-empty,
.rack-view-page-error {
  margin: 0;
  color: #5f6f82;
}

.rack-view-page-error {
  color: #b42318;
}

.rack-view-wrap {
  overflow-x: auto;
}

.rack-view-scroller {
  display: table;
  min-width: max-content;
  margin-inline: auto;
}

.rack-view-table {
  width: max-content;
  min-width: 100%;
  border-collapse: collapse;
  table-layout: auto;
}

.rack-view-ru-col {
  width: 46px;
}

.rack-view-main-col {
  width: auto;
}

.rack-view-table th,
.rack-view-table td {
  border: 1px solid #aebfd1;
  height: 1px;
  font-size: 9px;
}

.rack-view-table th {
  background: #eff6fd;
  color: #164a77;
  padding: 6px 8px;
  font-weight: 700;
}

.rack-view-ru {
  background: #ffffff;
  color: #29506f;
  text-align: center;
  font-weight: 700;
  padding: 6px 4px;
}

.rack-view-cell {
  min-width: 156px;
  padding: 0;
  vertical-align: middle;
  text-align: center;
}

.rack-view-empty {
  background: rgba(201, 214, 226, 0.5);
}

.rack-view-occupied {
  background: var(--rack-cell-bg, linear-gradient(180deg, #e7f7df 0%, #d3efc7 100%));
  color: var(--rack-cell-fg, #143d5c);
}

.rack-view-status-active {
  background: linear-gradient(180deg, #e7f7df 0%, #d3efc7 100%);
}

.rack-view-status-stored {
  background: linear-gradient(180deg, #eef5ff 0%, #dce9ff 100%);
}

.rack-view-status-defective {
  background: linear-gradient(180deg, #fff1e4 0%, #ffd8bf 100%);
}

.rack-view-status-obsolete {
  background: linear-gradient(180deg, #f3f4f6 0%, #e5e7eb 100%);
}

.rack-view-highlight {
  box-shadow: inset 0 0 0 3px #ef6c00;
}

.rack-view-link {
  width: max-content;
  min-width: 100%;
  min-height: 100%;
  border: 0;
  background: transparent;
  color: inherit;
  padding: 8px 10px;
  text-align: center;
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.rack-view-link:hover {
  background: rgba(255, 255, 255, 0.24);
}

.rack-view-title,
.rack-view-subtitle {
  display: inline-block;
  white-space: nowrap;
  line-height: 1.35;
}

.rack-view-title {
  font-weight: 700;
}

.rack-view-subtitle {
  font-size: 0.85rem;
  color: inherit;
  opacity: 0.92;
}

.rack-view-base {
  height: 12px;
  background: linear-gradient(180deg, #d6dde4 0%, #b9c4cf 100%);
}

.rack-view-wheel-spacer {
  border: 0 !important;
  background: transparent;
  height: 20px;
}

.rack-view-wheel {
  border: 0 !important;
  background: transparent;
  text-align: center;
  padding-top: 4px;
}

.rack-view-wheel img {
  width: 18px;
  height: 18px;
}

.rack-view-side-note {
  margin-top: 12px;
  border-top: 1px dashed #c3d5e7;
  padding-top: 12px;
}

.rack-view-side-note h5 {
  margin: 0 0 8px;
  color: #17527f;
}

.rack-view-side-note ul {
  margin: 0;
  padding-left: 18px;
}

.rack-view-side-link {
  border: 0;
  background: transparent;
  color: #185f94;
  padding: 0;
  text-decoration: underline;
}

.rack-view-warning-list {
  margin-top: 12px;
  border-top: 1px dashed #efb4ac;
  padding-top: 12px;
  color: #b42318;
}

.rack-view-warning-list p {
  margin: 0 0 4px;
}
</style>
