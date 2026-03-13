<script setup lang="ts">
import { computed, nextTick, reactive, ref } from 'vue'
import QRCode from 'qrcode'
import api from '../api/client'
import { useAuthStore } from '../stores/auth'

type LabelItem = {
  id: number
  text: string
  status?: number
  statusdesc?: string
  statuscolor?: string
  itemtype?: string
  manufacturer?: string
  model?: string
  sn?: string
  label?: string
}

type LabelPreset = {
  id: number
  name: string
  rows: number
  cols: number
  lwidth: string
  lheight: string
  vpitch: string
  hpitch: string
  tmargin: string
  bmargin: string
  lmargin: string
  rmargin: string
  border: string
  padding: string
  fontsize: string
  headerfontsize: string
  barcodesize: string
  idfontsize: string
  wantbarcode: number
  wantheadertext: number
  wantheaderimage: number
  headertext: string
  image: string
  imagewidth: string
  imageheight: string
  papersize: string
  qrtext: string
  wantnotext: number
  wantraligntext: number
  labelskip?: string
}

type LabelPreview = {
  id: number
  text: string
  headerText: string
  qrText: string
  qrImage?: string
  itemType?: string
  manufacturer?: string
  model?: string
  sn?: string
  label?: string
  ipv4?: string
  ipv6?: string
  dnsName?: string
}

type PaperSizeInfo = {
  key: string
  widthMm: number
  heightMm: number
}

const paperNameDefaults = `
A0
A1
A2
A3
A4
A5
A6
A7
A8
A9
A10
A11
A12
B0
B1
B2
B3
B4
B5
B6
B7
B8
B9
B10
B11
B12
C0
C1
C2
C3
C4
C5
C6
C7
C8
C9
C10
C11
C12
C76
DL
E0
E1
E2
E3
E4
E5
E6
E7
E8
E9
E10
E11
E12
G0
G1
G2
G3
G4
G5
G6
G7
G8
G9
G10
G11
G12
RA0
RA1
RA2
RA3
RA4
SRA0
SRA1
SRA2
SRA3
SRA4
4A0
2A0
A2_EXTRA
A3+
A3_EXTRA
A3_SUPER
SUPER_A3
A4_EXTRA
A4_SUPER
SUPER_A4
A4_LONG
F4
SO_B5_EXTRA
A5_EXTRA
ANSI_E
ANSI_D
ANSI_C
ANSI_B
ANSI_A
USLEDGER
LEDGER
ORGANIZERK
BIBLE
USTABLOID
TABLOID
ORGANIZERM
USLETTER
LETTER
USLEGAL
LEGAL
GOVERNMENTLETTER
GLETTER
JUNIORLEGAL
JLEGAL
QUADDEMY
SUPER_B
QUARTO
GOVERNMENTLEGAL
FOLIO
MONARCH
EXECUTIVE
ORGANIZERL
STATEMENT
MEMO
FOOLSCAP
COMPACT
ORGANIZERJ
P1
P2
P3
P4
P5
P6
ARCH_E
ARCH_E1
ARCH_D
BROADSHEET
ARCH_C
ARCH_B
ARCH_A
ANNENV_A2
ANNENV_A6
ANNENV_A7
ANNENV_A8
ANNENV_A10
ANNENV_SLIM
COMMENV_N6_1/4
COMMENV_N6_3/4
COMMENV_N8
COMMENV_N9
COMMENV_N10
COMMENV_N11
COMMENV_N12
COMMENV_N14
CATENV_N1
CATENV_N1_3/4
CATENV_N2
CATENV_N3
CATENV_N6
CATENV_N7
CATENV_N8
CATENV_N9_1/2
CATENV_N9_3/4
CATENV_N10_1/2
CATENV_N12_1/2
CATENV_N13_1/2
CATENV_N14_1/4
CATENV_N14_1/2
JIS_B0
JIS_B1
JIS_B2
JIS_B3
JIS_B4
JIS_B5
JIS_B6
JIS_B7
JIS_B8
JIS_B9
JIS_B10
JIS_B11
JIS_B12
PA0
PA1
PA2
PA3
PA4
PA5
PA6
PA7
PA8
PA9
PA10
PASSPORT_PHOTO
E
L
3R
KG
4R
4D
2L
5R
8P
6R
6P
8R
6PW
S8R
4P
10R
4PW
S10R
11R
S11R
12R
S12R
NEWSPAPER_BROADSHEET
NEWSPAPER_BERLINER
NEWSPAPER_TABLOID
NEWSPAPER_COMPACT
CREDIT_CARD
BUSINESS_CARD
BUSINESS_CARD_ISO7810
BUSINESS_CARD_ISO216
BUSINESS_CARD_IT
BUSINESS_CARD_UK
BUSINESS_CARD_FR
BUSINESS_CARD_DE
BUSINESS_CARD_ES
BUSINESS_CARD_CA
BUSINESS_CARD_US
BUSINESS_CARD_JP
BUSINESS_CARD_HK
BUSINESS_CARD_AU
BUSINESS_CARD_DK
BUSINESS_CARD_SE
BUSINESS_CARD_RU
BUSINESS_CARD_CZ
BUSINESS_CARD_FI
BUSINESS_CARD_HU
BUSINESS_CARD_IL
4SHEET
6SHEET
12SHEET
16SHEET
32SHEET
48SHEET
64SHEET
96SHEET
EN_EMPEROR
EN_ANTIQUARIAN
EN_GRAND_EAGLE
EN_DOUBLE_ELEPHANT
EN_ATLAS
EN_COLOMBIER
EN_ELEPHANT
EN_DOUBLE_DEMY
EN_IMPERIAL
EN_PRINCESS
EN_CARTRIDGE
EN_DOUBLE_LARGE_POST
EN_ROYAL
EN_SHEET
EN_HALF_POST
EN_SUPER_ROYAL
EN_DOUBLE_POST
EN_MEDIUM
EN_DEMY
EN_LARGE_POST
EN_COPY_DRAUGHT
EN_POST
EN_CROWN
EN_PINCHED_POST
EN_BRIEF
EN_FOOLSCAP
EN_SMALL_FOOLSCAP
EN_POTT
BE_GRAND_AIGLE
BE_COLOMBIER
BE_DOUBLE_CARRE
BE_ELEPHANT
BE_PETIT_AIGLE
BE_GRAND_JESUS
BE_JESUS
BE_RAISIN
BE_GRAND_MEDIAN
BE_DOUBLE_POSTE
BE_COQUILLE
BE_PETIT_MEDIAN
BE_RUCHE
BE_PROPATRIA
BE_LYS
BE_POT
BE_ROSETTE
FR_UNIVERS
FR_DOUBLE_COLOMBIER
FR_GRANDE_MONDE
FR_DOUBLE_SOLEIL
FR_DOUBLE_JESUS
FR_GRAND_AIGLE
FR_PETIT_AIGLE
FR_DOUBLE_RAISIN
FR_JOURNAL
FR_COLOMBIER_AFFICHE
FR_DOUBLE_CAVALIER
FR_CLOCHE
FR_SOLEIL
FR_DOUBLE_CARRE
FR_DOUBLE_COQUILLE
FR_JESUS
FR_RAISIN
FR_CAVALIER
FR_DOUBLE_COURONNE
FR_CARRE
FR_COQUILLE
FR_DOUBLE_TELLIERE
FR_DOUBLE_CLOCHE
FR_DOUBLE_POT
FR_ECU
FR_COURONNE
FR_TELLIERE
FR_POT
`.trim().split('\n')

const isoASizeList: Array<[number, number]> = [
  [841, 1189],
  [594, 841],
  [420, 594],
  [297, 420],
  [210, 297],
  [148, 210],
  [105, 148],
  [74, 105],
  [52, 74],
  [37, 52],
  [26, 37],
  [18, 26],
  [13, 18],
]

const isoBSizeList: Array<[number, number]> = [
  [1000, 1414],
  [707, 1000],
  [500, 707],
  [353, 500],
  [250, 353],
  [176, 250],
  [125, 176],
  [88, 125],
  [62, 88],
  [44, 62],
  [31, 44],
  [22, 31],
  [15, 22],
]

const namedPaperSizes: Record<string, [number, number]> = {
  LETTER: [216, 279],
  USLETTER: [216, 279],
  LEGAL: [216, 356],
  USLEGAL: [216, 356],
  TABLOID: [279, 432],
  USTABLOID: [279, 432],
  LEDGER: [279, 432],
  USLEDGER: [279, 432],
  EXECUTIVE: [184, 267],
  FOLIO: [210, 330],
  F4: [210, 330],
  QUARTO: [215, 275],
  STATEMENT: [140, 216],
  MEMO: [140, 216],
  ANSI_A: [216, 279],
  ANSI_B: [279, 432],
  ANSI_C: [432, 559],
  ANSI_D: [559, 864],
  ANSI_E: [864, 1118],
  ARCH_A: [229, 305],
  ARCH_B: [305, 457],
  ARCH_C: [457, 610],
  ARCH_D: [610, 914],
  ARCH_E: [914, 1219],
  ARCH_E1: [762, 1067],
}

const auth = useAuthStore()

const search = ref('')
const orderBy = ref<'type' | 'id' | 'id_desc' | 'model'>('type')
const items = ref<LabelItem[]>([])
const presets = ref<LabelPreset[]>([])
const selectedIds = ref<number[]>([])
const previewRows = ref<LabelPreview[]>([])
const previewSectionRef = ref<HTMLElement | null>(null)

const loadingItems = ref(false)
const loadingPresets = ref(false)
const loadingPreview = ref(false)
const printingPreview = ref(false)
const savingPreset = ref(false)
const deletingPreset = ref(false)
const confirmOpen = ref(false)
const pendingDeletePresetId = ref<number | null>(null)
const error = ref('')
const success = ref('')
let loadItemsSeq = 0

const form = reactive({
  name: '',
  rows: 8,
  cols: 3,
  lwidth: '66',
  lheight: '35',
  vpitch: '35',
  hpitch: '70',
  tmargin: '12',
  bmargin: '12',
  lmargin: '6',
  rmargin: '6',
  border: '200',
  padding: '1',
  fontsize: '6',
  headerfontsize: '6',
  barcodesize: '20',
  idfontsize: '7',
  wantbarcode: true,
  wantheadertext: true,
  wantheaderimage: false,
  headertext: 'IT资产标签',
  image: 'images/itdb.png',
  imagewidth: '5',
  imageheight: '5',
  papersize: 'A4',
  qrtext: '',
  wantnotext: false,
  wantraligntext: false,
  labelskip: '0',
})

const selectedCount = computed(() => selectedIds.value.length)
const allChecked = computed(() => {
  if (items.value.length === 0) return false
  const selected = new Set(selectedIds.value)
  return items.value.every((item) => selected.has(item.id))
})
const canWrite = computed(() => !auth.isReadOnly)
const confirmMessage = computed(() =>
  pendingDeletePresetId.value ? `确认删除预设 编号=${pendingDeletePresetId.value} 吗？` : '',
)
const orderLinks = computed(() => [
  { key: 'type' as const, text: '[类型]', tip: '订单: 状态, 硬件类型, 厂商, 编号' },
  { key: 'id' as const, text: '[编号]', tip: '订单: 状态, 编号, 硬件类型, 厂商' },
  { key: 'id_desc' as const, text: '[编号降序]', tip: '订单: 状态, 编号(逆序), 硬件类型, 厂商' },
  { key: 'model' as const, text: '[型号]', tip: '订单: 状态, 型号, 硬件类型, 厂商' },
])
const defaultStatusColorMap: Record<string, string> = {
  使用中: '#2f7fba',
  库存: '#16a34a',
  有故障: '#dc2626',
  报废: '#9ca3af',
}

const groupedItems = computed(() => {
  const groups: Array<{ key: string; title: string; color: string; rows: LabelItem[]; showHeader: boolean }> = []
  let current: { key: string; title: string; color: string; rows: LabelItem[]; showHeader: boolean } | null = null

  for (const item of items.value) {
    const statusID = Number(item.status ?? 0)
    const statusTitle = String(item.statusdesc ?? '').trim()
    const groupKey = `${statusID}-${statusTitle}`
    if (!current || current.key !== groupKey) {
      current = {
        key: groupKey,
        title: statusTitle,
        color: getStatusBlockColor(item),
        rows: [],
        showHeader: statusTitle.length > 0,
      }
      groups.push(current)
    }
    current.rows.push(item)
  }
  return groups
})
const paperOptions = computed(() => {
  const out: string[] = []
  const seen = new Set<string>()
  const push = (raw: unknown) => {
    const v = String(raw ?? '').trim()
    if (!v || seen.has(v)) return
    seen.add(v)
    out.push(v)
  }
  paperNameDefaults.forEach(push)
  presets.value.forEach((preset) => push(preset.papersize))
  push(form.papersize)
  return out
})
const rowOptions = Array.from({ length: 39 }, (_, i) => i + 1)
const colOptions = Array.from({ length: 9 }, (_, i) => i + 1)

type PreviewCell = {
  key: string
  kind: 'label' | 'skip' | 'blank'
  row?: LabelPreview
}

function getInputValue(event: Event) {
  const target = event.target as HTMLInputElement | null
  return target?.value ?? ''
}

function onSearchInput(event: Event) {
  search.value = getInputValue(event)
  void loadItems()
}

function toNumeric(input: string | number) {
  const value = Number(input)
  return Number.isFinite(value) ? value : 0
}

function clampNumber(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}

function getPaperSizeInfo(raw: unknown): PaperSizeInfo {
  const key = String(raw ?? 'A4').trim().toUpperCase() || 'A4'
  const named = namedPaperSizes[key]
  if (named) {
    return { key, widthMm: named[0], heightMm: named[1] }
  }

  const aMatch = key.match(/^A(\d{1,2})$/)
  if (aMatch) {
    const index = Number(aMatch[1])
    const size = isoASizeList[index]
    if (size) return { key, widthMm: size[0], heightMm: size[1] }
  }

  const bMatch = key.match(/^(?:B|JIS_B)(\d{1,2})$/)
  if (bMatch) {
    const index = Number(bMatch[1])
    const size = isoBSizeList[index]
    if (size) return { key, widthMm: size[0], heightMm: size[1] }
  }

  return { key: 'A4', widthMm: 210, heightMm: 297 }
}

function normalizeHexColor(raw: unknown) {
  const value = String(raw ?? '').trim()
  return /^#[0-9a-fA-F]{6}$/.test(value) ? value.toLowerCase() : ''
}

function getStatusBlockColor(item: LabelItem) {
  const custom = normalizeHexColor(item.statuscolor)
  if (custom) return custom
  const title = String(item.statusdesc ?? '').trim()
  return defaultStatusColorMap[title] ?? '#5b7aa4'
}

function formatItemIDType(item: LabelItem) {
  const id = Number(item.id ?? 0)
  const idText = Number.isFinite(id) && id > 0 ? String(Math.trunc(id)).padStart(4, '0') : '0000'
  const typeText = String(item.itemtype ?? '').trim() || '-'
  return `${idText}-${typeText}`
}

function formatItemText(item: LabelItem) {
  const manufacturer = String(item.manufacturer ?? '').trim()
  const model = String(item.model ?? '').trim()
  const sn = String(item.sn ?? '').trim()
  const label = String(item.label ?? '').trim()
  const core = [manufacturer || '-', model || '-', sn || '-'].join('-')
  return label ? `${core}-${label}` : core
}

function formatPreviewID(id: number) {
  return String(Math.max(0, Math.trunc(Number(id) || 0))).padStart(4, '0')
}

function normalizePreviewHeaderText(raw: string) {
  return raw.split('_NL_').join('\n').trim()
}

function escapeHtml(raw: unknown) {
  return String(raw ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

function resolvePreviewImageSrc(raw: string) {
  const value = String(raw ?? '').trim()
  if (!value) return ''
  if (value.startsWith('/') || value.startsWith('data:') || /^https?:\/\//i.test(value)) return value
  return `/${value.replace(/^\.?\//, '')}`
}

async function attachPreviewQrImages(rows: LabelPreview[]) {
  const cache = new Map<string, string>()
  return Promise.all(
    rows.map(async (row) => {
      const qrText = String(row.qrText ?? '').trim()
      if (!qrText) return { ...row, qrImage: '' }
      try {
        if (!cache.has(qrText)) {
          const svg = await QRCode.toString(qrText, {
            type: 'svg',
            margin: 0,
            errorCorrectionLevel: 'M',
            color: {
              dark: '#102a43',
              light: '#ffffff',
            },
          })
          cache.set(qrText, `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`)
        }
        return { ...row, qrImage: cache.get(qrText) ?? '' }
      } catch {
        return { ...row, qrImage: '' }
      }
    }),
  )
}

function buildPreviewDescription(row: LabelPreview) {
  const manufacturer = String(row.manufacturer ?? '').trim()
  const model = String(row.model ?? '').trim()
  const text = [manufacturer, model].filter(Boolean).join('/')
  return text.length > 37 ? `${text.slice(0, 37)}` : text
}

function getPreviewIdLine(row: LabelPreview) {
  if (form.wantnotext) return ''
  return `ID:${formatPreviewID(row.id)}`
}

function getPreviewBodyLines(row: LabelPreview) {
  if (form.wantnotext) return [] as string[]

  const lines: string[] = []
  const label = String(row.label ?? '').trim()
  const sn = String(row.sn ?? '').trim()
  const ipv4 = String(row.ipv4 ?? '').trim()
  const ipv6 = String(row.ipv6 ?? '').trim()
  const dnsName = String(row.dnsName ?? '').trim()
  const desc = buildPreviewDescription(row)

  if (label) lines.push(`LBL:${label}`)
  if (sn) lines.push(`SN:${sn}`)
  if (desc) lines.push(desc)
  if (ipv4) lines.push(`IPv4:${ipv4.slice(0, 15)}`)
  if (ipv6) lines.push(`IPv6:${ipv6}`)
  if (dnsName) lines.push(`HName:${dnsName}`)
  return lines
}

const previewColumns = computed(() => clampNumber(Math.trunc(toNumeric(form.cols) || 1), 1, 9))
const previewRowsPerPage = computed(() => clampNumber(Math.trunc(toNumeric(form.rows) || 1), 1, 60))
const previewSkipCount = computed(() => Math.max(0, Math.trunc(toNumeric(form.labelskip) || 0)))
const previewPageCapacity = computed(() => Math.max(1, previewColumns.value * previewRowsPerPage.value))
const previewLabelWidthPx = computed(() => clampNumber(toNumeric(form.lwidth) * 3.2, 150, 360))
const previewLabelHeightPx = computed(() => clampNumber(toNumeric(form.lheight) * 3.2, 96, 240))
const previewPaperInfo = computed(() => getPaperSizeInfo(form.papersize))
const previewGapX = computed(() =>
  clampNumber(Math.max(0, (toNumeric(form.hpitch) || 0) - (toNumeric(form.lwidth) || 0)) * 3.2, 0, 96),
)
const previewGapY = computed(() =>
  clampNumber(Math.max(0, (toNumeric(form.vpitch) || 0) - (toNumeric(form.lheight) || 0)) * 3.2, 0, 96),
)
const previewPaperScale = computed(() => {
  const widthMm = previewPaperInfo.value.widthMm
  const heightMm = previewPaperInfo.value.heightMm
  let scale = 2.55
  scale = Math.min(scale, 860 / widthMm, 1200 / heightMm)
  scale = Math.max(scale, 260 / widthMm)
  return scale
})
const previewSheetPaddingPx = computed(() => ({
  top: clampNumber((toNumeric(form.tmargin) || 0) * previewPaperScale.value, 0, 120),
  right: clampNumber((toNumeric(form.rmargin) || 0) * previewPaperScale.value, 0, 120),
  bottom: clampNumber((toNumeric(form.bmargin) || 0) * previewPaperScale.value, 0, 120),
  left: clampNumber((toNumeric(form.lmargin) || 0) * previewPaperScale.value, 0, 120),
}))
const previewPaperWidthPx = computed(() => previewPaperInfo.value.widthMm * previewPaperScale.value)
const previewPaperHeightPx = computed(() => previewPaperInfo.value.heightMm * previewPaperScale.value)
const previewGridWidthPx = computed(() => {
  if (previewColumns.value <= 0) return 0
  return previewColumns.value * previewLabelWidthPx.value + Math.max(0, previewColumns.value - 1) * previewGapX.value
})
const previewGridHeightPx = computed(() => {
  if (previewRowsPerPage.value <= 0) return 0
  return previewRowsPerPage.value * previewLabelHeightPx.value + Math.max(0, previewRowsPerPage.value - 1) * previewGapY.value
})
const previewSheetCanvasStyle = computed(() => ({
  width: `${Math.max(previewPaperWidthPx.value, previewGridWidthPx.value + previewSheetPaddingPx.value.left + previewSheetPaddingPx.value.right)}px`,
  minHeight: `${Math.max(previewPaperHeightPx.value, previewGridHeightPx.value + previewSheetPaddingPx.value.top + previewSheetPaddingPx.value.bottom)}px`,
  paddingTop: `${previewSheetPaddingPx.value.top}px`,
  paddingRight: `${previewSheetPaddingPx.value.right}px`,
  paddingBottom: `${previewSheetPaddingPx.value.bottom}px`,
  paddingLeft: `${previewSheetPaddingPx.value.left}px`,
}))
const previewHeaderLines = computed(() =>
  form.wantheadertext
    ? normalizePreviewHeaderText(String(form.headertext ?? ''))
        .split('\n')
        .filter((line: string) => line.trim().length > 0)
    : [],
)
const previewImageSrc = computed(() => (form.wantheaderimage ? resolvePreviewImageSrc(String(form.image ?? '')) : ''))
const previewGridStyle = computed(() => ({
  gridTemplateColumns: `repeat(${previewColumns.value}, ${previewLabelWidthPx.value}px)`,
  columnGap: `${previewGapX.value}px`,
  rowGap: `${previewGapY.value}px`,
}))
const previewCardVars = computed(() => {
  const borderGray = clampNumber(Math.trunc(toNumeric(form.border) || 0), 0, 255)
  return {
    '--label-preview-width': `${previewLabelWidthPx.value}px`,
    '--label-preview-height': `${previewLabelHeightPx.value}px`,
    '--label-preview-padding': `${clampNumber(toNumeric(form.padding) * 3.2, 2, 24)}px`,
    '--label-preview-border': `rgb(${borderGray}, ${borderGray}, ${borderGray})`,
    '--label-preview-font-size': `${clampNumber(toNumeric(form.fontsize) * 1.45, 10, 18)}px`,
    '--label-preview-id-size': `${clampNumber(toNumeric(form.idfontsize) * 1.45, 11, 20)}px`,
    '--label-preview-header-size': `${clampNumber(toNumeric(form.headerfontsize) * 1.45, 10, 18)}px`,
    '--label-preview-barcode-size': `${clampNumber(toNumeric(form.barcodesize) * 3.2, 36, 160)}px`,
    '--label-preview-image-width': `${clampNumber(toNumeric(form.imagewidth) * 3.2, 18, 96)}px`,
    '--label-preview-image-height': `${clampNumber(toNumeric(form.imageheight) * 3.2, 18, 96)}px`,
  } as Record<string, string>
})
const previewPages = computed<PreviewCell[][]>(() => {
  const cells: PreviewCell[] = []
  for (let index = 0; index < previewSkipCount.value; index += 1) {
    cells.push({ key: `skip-${index}`, kind: 'skip' })
  }
  for (const row of previewRows.value) {
    cells.push({ key: `label-${row.id}`, kind: 'label', row })
  }

  if (cells.length === 0) return []

  const pages: PreviewCell[][] = []
  const capacity = previewPageCapacity.value
  for (let offset = 0; offset < cells.length; offset += capacity) {
    const page = cells.slice(offset, offset + capacity)
    while (page.length < capacity) {
      page.push({ key: `blank-${pages.length}-${page.length}`, kind: 'blank' })
    }
    pages.push(page)
  }
  return pages
})

function isChecked(id: number) {
  return selectedIds.value.includes(id)
}

function toggleItem(id: number) {
  if (isChecked(id)) {
    selectedIds.value = selectedIds.value.filter((v) => v !== id)
    return
  }
  selectedIds.value = [...selectedIds.value, id]
}

function toggleAll() {
  if (allChecked.value) {
    selectedIds.value = []
    return
  }
  selectedIds.value = items.value.map((item) => item.id)
}

function setOrderBy(next: 'type' | 'id' | 'id_desc' | 'model') {
  if (orderBy.value === next) return
  orderBy.value = next
  void loadItems()
}

function fillFormFromPreset(preset: LabelPreset) {
  form.name = preset.name ?? ''
  form.rows = toNumeric(preset.rows) || 8
  form.cols = toNumeric(preset.cols) || 3
  form.lwidth = String(preset.lwidth ?? '66')
  form.lheight = String(preset.lheight ?? '35')
  form.vpitch = String(preset.vpitch ?? '35')
  form.hpitch = String(preset.hpitch ?? '70')
  form.tmargin = String(preset.tmargin ?? '12')
  form.bmargin = String(preset.bmargin ?? '12')
  form.lmargin = String(preset.lmargin ?? '6')
  form.rmargin = String(preset.rmargin ?? '6')
  form.border = String(preset.border ?? '200')
  form.padding = String(preset.padding ?? '1')
  form.fontsize = String(preset.fontsize ?? '6')
  form.headerfontsize = String(preset.headerfontsize ?? '6')
  form.barcodesize = String(preset.barcodesize ?? '20')
  form.idfontsize = String(preset.idfontsize ?? '7')
  form.wantbarcode = Number(preset.wantbarcode) === 1
  form.wantheadertext = Number(preset.wantheadertext) === 1
  form.wantheaderimage = Number(preset.wantheaderimage) === 1
  form.headertext = String(preset.headertext ?? '')
  form.image = String(preset.image ?? 'images/itdb.png')
  form.imagewidth = String(preset.imagewidth ?? '5')
  form.imageheight = String(preset.imageheight ?? '5')
  form.papersize = String(preset.papersize ?? 'A4')
  form.qrtext = String(preset.qrtext ?? '')
  form.wantnotext = Number(preset.wantnotext) === 1
  form.wantraligntext = Number(preset.wantraligntext) === 1
  form.labelskip = String(preset.labelskip ?? '0')
}

function buildPresetPayload() {
  return {
    name: form.name.trim(),
    rows: toNumeric(form.rows),
    cols: toNumeric(form.cols),
    lwidth: String(form.lwidth ?? ''),
    lheight: String(form.lheight ?? ''),
    vpitch: String(form.vpitch ?? ''),
    hpitch: String(form.hpitch ?? ''),
    tmargin: String(form.tmargin ?? ''),
    bmargin: String(form.bmargin ?? ''),
    lmargin: String(form.lmargin ?? ''),
    rmargin: String(form.rmargin ?? ''),
    border: String(form.border ?? ''),
    padding: String(form.padding ?? ''),
    fontsize: String(form.fontsize ?? ''),
    headerfontsize: String(form.headerfontsize ?? ''),
    barcodesize: String(form.barcodesize ?? ''),
    idfontsize: String(form.idfontsize ?? ''),
    wantbarcode: form.wantbarcode ? 1 : 0,
    wantheadertext: form.wantheadertext ? 1 : 0,
    wantheaderimage: form.wantheaderimage ? 1 : 0,
    headertext: String(form.headertext ?? ''),
    image: String(form.image ?? ''),
    imagewidth: String(form.imagewidth ?? ''),
    imageheight: String(form.imageheight ?? ''),
    papersize: String(form.papersize ?? 'A4'),
    qrtext: String(form.qrtext ?? ''),
    wantnotext: form.wantnotext ? 1 : 0,
    wantraligntext: form.wantraligntext ? 1 : 0,
  }
}

async function loadItems() {
  const seq = ++loadItemsSeq
  loadingItems.value = true
  error.value = ''
  success.value = ''
  const selectedSet = new Set(selectedIds.value)

  try {
    const { data } = await api.get<LabelItem[]>('/labels/items', {
      params: {
        search: search.value || undefined,
        orderBy: orderBy.value === 'type' ? undefined : orderBy.value,
        limit: 1500,
      },
    })
    if (seq !== loadItemsSeq) return
    items.value = data ?? []
    selectedIds.value = items.value.map((item) => item.id).filter((id) => selectedSet.has(id))
  } catch (err: unknown) {
    if (seq !== loadItemsSeq) return
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '标签硬件加载失败'
  } finally {
    if (seq !== loadItemsSeq) return
    loadingItems.value = false
  }
}

async function loadPresets() {
  loadingPresets.value = true
  error.value = ''
  try {
    const { data } = await api.get<LabelPreset[]>('/labels/presets')
    presets.value = data ?? []
    if (!form.name && presets.value.length > 0) {
      fillFormFromPreset(presets.value[0] as LabelPreset)
    }
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '标签预设加载失败'
  } finally {
    loadingPresets.value = false
  }
}

async function savePreset() {
  if (!canWrite.value) return
  const payload = buildPresetPayload()
  if (!payload.name) {
    error.value = '请先输入预设名称'
    return
  }

  savingPreset.value = true
  error.value = ''
  success.value = ''
  try {
    await api.post('/labels/presets', payload)
    success.value = '预设保存成功'
    await loadPresets()
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '预设保存失败'
  } finally {
    savingPreset.value = false
  }
}

function requestDeletePreset(id: number) {
  if (!canWrite.value) return
  pendingDeletePresetId.value = id
  confirmOpen.value = true
}

function closeConfirm() {
  if (deletingPreset.value) return
  confirmOpen.value = false
  pendingDeletePresetId.value = null
}

async function confirmDeletePreset() {
  if (!canWrite.value || !pendingDeletePresetId.value) return
  deletingPreset.value = true
  const id = pendingDeletePresetId.value
  error.value = ''
  success.value = ''
  try {
    await api.delete(`/labels/presets/${id}`)
    confirmOpen.value = false
    pendingDeletePresetId.value = null
    success.value = '预设已删除'
    await loadPresets()
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '预设删除失败'
  } finally {
    deletingPreset.value = false
  }
}

async function loadPreview() {
  if (selectedIds.value.length === 0) {
    previewRows.value = []
    return
  }
  loadingPreview.value = true
  error.value = ''
  success.value = ''
  try {
    const { data } = await api.post<LabelPreview[]>('/labels/preview', {
      itemIds: selectedIds.value,
      qrPrefix: form.qrtext || '',
      headerText: form.headertext || '',
    })
    previewRows.value = await attachPreviewQrImages(data ?? [])
    success.value = previewRows.value.length > 0 ? `已生成 ${previewRows.value.length} 条标签预览` : '当前选择没有可预览的标签数据'
    await nextTick()
    previewSectionRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '标签预览加载失败'
  } finally {
    loadingPreview.value = false
  }
}

function buildPrintLabelHtml(row: LabelPreview) {
  const headerHtml =
    previewImageSrc.value || previewHeaderLines.value.length > 0
      ? `
      <div class="label-header">
        ${
          previewImageSrc.value
            ? `<img src="${escapeHtml(previewImageSrc.value)}" alt="标签页头图片">`
            : ''
        }
        ${
          previewHeaderLines.value.length > 0
            ? `<div class="label-header-text">${previewHeaderLines.value
                .map((line) => `<span>${escapeHtml(line)}</span>`)
                .join('')}</div>`
            : ''
        }
      </div>`
      : ''

  const qrHtml = form.wantbarcode
    ? `
      <div class="label-barcode">
        <div class="label-barcode-code">
          ${
            row.qrImage
              ? `<img src="${escapeHtml(row.qrImage)}" alt="${escapeHtml(row.qrText || '二维码')}">`
              : '<strong>QR</strong>'
          }
        </div>
      </div>`
    : ''

  const bodyHtml = form.wantnotext
    ? ''
    : `
      <div class="label-text">
        <div class="label-id">${escapeHtml(getPreviewIdLine(row))}</div>
        ${getPreviewBodyLines(row)
          .map((line) => `<div class="label-line">${escapeHtml(line)}</div>`)
          .join('')}
      </div>`

  return `
    <article class="label-card${form.wantbarcode && form.wantraligntext ? ' is-right-text' : ''}${form.wantnotext ? ' is-no-text' : ''}">
      ${headerHtml}
      <div class="label-body">
        ${qrHtml}
        ${bodyHtml}
      </div>
    </article>
  `
}

function buildPrintDocumentHtml() {
  const paper = previewPaperInfo.value
  const borderGray = clampNumber(Math.trunc(toNumeric(form.border) || 0), 0, 255)
  const gapX = Math.max(0, (toNumeric(form.hpitch) || 0) - (toNumeric(form.lwidth) || 0))
  const gapY = Math.max(0, (toNumeric(form.vpitch) || 0) - (toNumeric(form.lheight) || 0))
  const previewHtml = previewPages.value
    .map(
      (page) => `
        <section class="sheet">
          <div class="grid">
            ${page
              .map((cell) => {
                if (cell.kind === 'skip') return '<div class="skip-cell">跳过</div>'
                if (cell.kind === 'blank') return '<div class="blank-cell" aria-hidden="true"></div>'
                if (cell.kind === 'label' && cell.row) return buildPrintLabelHtml(cell.row)
                return '<div class="blank-cell" aria-hidden="true"></div>'
              })
              .join('')}
          </div>
        </section>
      `,
    )
    .join('')

  return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>${escapeHtml(form.name.trim() || '打印标签')}</title>
  <style>
    @page {
      size: ${paper.widthMm}mm ${paper.heightMm}mm;
      margin: 0;
    }
    :root {
      --paper-width: ${paper.widthMm}mm;
      --paper-height: ${paper.heightMm}mm;
      --label-width: ${Math.max(toNumeric(form.lwidth) || 0, 10)}mm;
      --label-height: ${Math.max(toNumeric(form.lheight) || 0, 8)}mm;
      --gap-x: ${gapX}mm;
      --gap-y: ${gapY}mm;
      --padding-top: ${Math.max(toNumeric(form.tmargin) || 0, 0)}mm;
      --padding-right: ${Math.max(toNumeric(form.rmargin) || 0, 0)}mm;
      --padding-bottom: ${Math.max(toNumeric(form.bmargin) || 0, 0)}mm;
      --padding-left: ${Math.max(toNumeric(form.lmargin) || 0, 0)}mm;
      --label-padding: ${clampNumber(toNumeric(form.padding) || 0, 0, 8)}mm;
      --label-border: rgb(${borderGray}, ${borderGray}, ${borderGray});
      --label-font-size: ${clampNumber(toNumeric(form.fontsize) || 6, 4, 18)}pt;
      --label-id-size: ${clampNumber(toNumeric(form.idfontsize) || 7, 4, 20)}pt;
      --label-header-size: ${clampNumber(toNumeric(form.headerfontsize) || 6, 4, 18)}pt;
      --label-barcode-size: ${clampNumber(toNumeric(form.barcodesize) || 20, 8, 40)}mm;
      --label-image-width: ${clampNumber(toNumeric(form.imagewidth) || 0, 0, 40)}mm;
      --label-image-height: ${clampNumber(toNumeric(form.imageheight) || 0, 0, 40)}mm;
    }
    * { box-sizing: border-box; }
    html, body {
      margin: 0;
      padding: 0;
      background: #fff;
      color: #000;
      font-family: "Noto Sans SC", "Segoe UI", sans-serif;
      -webkit-print-color-adjust: exact;
      print-color-adjust: exact;
    }
    .sheet {
      width: var(--paper-width);
      min-height: var(--paper-height);
      padding: var(--padding-top) var(--padding-right) var(--padding-bottom) var(--padding-left);
      display: flex;
      align-items: flex-start;
      justify-content: flex-start;
      overflow: hidden;
      break-after: page;
      page-break-after: always;
    }
    .sheet:last-child {
      break-after: auto;
      page-break-after: auto;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(${previewColumns.value}, var(--label-width));
      column-gap: var(--gap-x);
      row-gap: var(--gap-y);
      align-items: start;
      justify-content: start;
    }
    .label-card,
    .skip-cell,
    .blank-cell {
      width: var(--label-width);
      height: var(--label-height);
      min-height: var(--label-height);
    }
    .label-card {
      border: 0.2mm solid var(--label-border);
      padding: var(--label-padding);
      background: #fff;
      display: flex;
      flex-direction: column;
      gap: 1.6mm;
      overflow: hidden;
    }
    .label-card.is-no-text .label-body {
      align-items: center;
      justify-content: center;
    }
    .label-card.is-right-text .label-body {
      display: grid;
      grid-template-columns: auto minmax(0, 1fr);
      gap: 2mm;
      align-items: start;
    }
    .label-header {
      display: flex;
      align-items: flex-start;
      gap: 1.8mm;
      min-height: 0;
    }
    .label-header img {
      width: var(--label-image-width);
      height: var(--label-image-height);
      object-fit: contain;
      flex: 0 0 auto;
    }
    .label-header-text {
      min-width: 0;
      display: flex;
      flex-direction: column;
      gap: 0.3mm;
      color: #004664;
      font-size: var(--label-header-size);
      font-weight: 700;
      line-height: 1.08;
    }
    .label-header-text > span {
      white-space: pre-line;
    }
    .label-body {
      flex: 1 1 auto;
      min-height: 0;
      display: flex;
      flex-direction: column;
      gap: 1.2mm;
    }
    .label-barcode {
      width: var(--label-barcode-size);
      flex: 0 0 auto;
    }
    .label-barcode-code {
      width: var(--label-barcode-size);
      height: var(--label-barcode-size);
      border: 0.2mm solid #274763;
      padding: 1mm;
      display: grid;
      place-items: center;
      background: #fff;
    }
    .label-barcode-code img {
      width: 100%;
      height: 100%;
      display: block;
      object-fit: contain;
    }
    .label-barcode-code strong {
      font-size: 10pt;
      letter-spacing: 0.08em;
      color: #173b5e;
    }
    .label-text {
      min-width: 0;
      display: flex;
      flex-direction: column;
      gap: 0.35mm;
      color: #000;
      font-size: var(--label-font-size);
      line-height: 1.1;
    }
    .label-id {
      font-size: var(--label-id-size);
      font-weight: 800;
      line-height: 1.02;
    }
    .label-line {
      min-width: 0;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
    .skip-cell,
    .blank-cell {
      border: 0.2mm dashed #c6d4e2;
      display: flex;
      align-items: center;
      justify-content: center;
      color: #6b7280;
      font-size: 9pt;
      letter-spacing: 0.08em;
    }
    .blank-cell {
      border-color: rgba(156, 173, 191, 0.45);
      color: transparent;
    }
    @media screen {
      body {
        background: #e9eff5;
        padding: 16px;
      }
      .sheet {
        margin: 0 auto 16px;
        background: #fff;
        box-shadow: 0 12px 28px rgba(16, 42, 67, 0.14);
      }
    }
  </style>
</head>
<body>
  ${previewHtml}
  <script>
    const waitForImages = async () => {
      const images = Array.from(document.images)
      await Promise.all(
        images.map((img) => {
          if (img.complete) return Promise.resolve()
          return new Promise((resolve) => {
            img.addEventListener('load', resolve, { once: true })
            img.addEventListener('error', resolve, { once: true })
          })
        }),
      )
    }
    window.addEventListener('load', async () => {
      await waitForImages()
      setTimeout(() => window.print(), 120)
    })
  <\/script>
</body>
</html>`
}

async function printPreview() {
  if (previewPages.value.length === 0 || printingPreview.value) return
  const popup = window.open('', '_blank')
  if (!popup) {
    error.value = '浏览器拦截了打印窗口，请允许弹出窗口后重试'
    return
  }

  popup.document.open()
  popup.document.write(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><title>生成中</title></head><body style="font-family:'Noto Sans SC',sans-serif;padding:24px;color:#1f2937;">正在生成打印页，请稍候...</body></html>`)
  popup.document.close()

  printingPreview.value = true
  error.value = ''
  try {
    if (form.wantbarcode && previewRows.value.some((row) => row.qrText && !row.qrImage)) {
      previewRows.value = await attachPreviewQrImages(previewRows.value)
    }
    const html = buildPrintDocumentHtml()
    popup.document.open()
    popup.document.write(html)
    popup.document.close()
  } catch {
    popup.close()
    error.value = '打印页生成失败'
  } finally {
    printingPreview.value = false
  }
}

loadItems()
loadPresets()
</script>

<template>
  <section class="page-shell labels-page">
    <header class="page-header">
      <h2>打印标签</h2>
      <div class="header-actions">
        <button class="ghost-btn" @click="loadItems">刷新硬件</button>
        <button class="ghost-btn" @click="loadPresets">刷新预设</button>
      </div>
    </header>
    <p v-if="success" class="success-text section-gap">{{ success }}</p>

    <div class="labels-layout section-gap">
      <section class="labels-main-panel">
        <div class="labels-main-top">
          <div class="labels-order-row">
            <b>订购人：</b>
            <a
              v-for="link in orderLinks"
              :key="link.key"
              href="#"
              :class="{ active: orderBy === link.key }"
              :title="link.tip"
              @click.prevent="setOrderBy(link.key)"
            >
              {{ link.text }}
            </a>
          </div>

          <div class="labels-filter-row">
            <b>过滤：</b>
            <input
              :value="search"
              class="search-input labels-filter-input"
              placeholder="输入关键字过滤硬件"
              @input="onSearchInput"
              @compositionupdate="onSearchInput"
              @compositionend="onSearchInput"
            />
          </div>
        </div>

        <p v-if="loadingItems" class="muted-text">硬件加载中...</p>

        <div v-else class="table-wrap labels-list-wrap">
          <table class="labels-items-table">
            <thead>
              <tr>
                <th style="width: 56px">
                  <input type="checkbox" :checked="allChecked" @change="toggleAll" />
                </th>
                <th>编号-类型</th>
                <th>标签文本</th>
              </tr>
            </thead>
            <tbody>
              <template v-for="group in groupedItems" :key="`group-${group.key}`">
                <tr v-if="group.showHeader" class="labels-status-row">
                  <td :colspan="3" :style="{ backgroundColor: group.color }">{{ group.title }}</td>
                </tr>
                <tr v-for="item in group.rows" :key="item.id" @click="toggleItem(item.id)">
                  <td>
                    <input type="checkbox" :checked="isChecked(item.id)" @click.stop @change="toggleItem(item.id)" />
                  </td>
                  <td class="mono">{{ formatItemIDType(item) }}</td>
                  <td class="mono label-text-cell">{{ formatItemText(item) }}</td>
                </tr>
              </template>
              <tr v-if="items.length === 0">
                <td colspan="3">暂无可选硬件</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="labels-main-actions">
          <button type="button" class="make-label-btn" :disabled="selectedCount === 0 || loadingPreview" @click="loadPreview">
            {{ loadingPreview ? '生成中...' : '生成标签预览' }}
          </button>
          <button
            type="button"
            class="print-label-btn"
            :disabled="previewPages.length === 0 || loadingPreview || printingPreview"
            @click="printPreview"
          >
            {{ printingPreview ? '生成打印页...' : '打印 / 导出 PDF' }}
          </button>
        </div>

        <ol class="labels-help-list">
          <li>从上面选择需要打印标签的硬件</li>
          <li>在右侧设置标签属性（手工或套用预设）</li>
          <li>点击“生成标签预览”确认数据</li>
          <li>后续导出 PDF 时，打印设置建议关闭自动缩放</li>
        </ol>
      </section>

      <aside class="labels-props-panel">
        <table class="label-props-table">
          <colgroup>
            <col class="label-props-col-name" />
            <col class="label-props-col-value" />
            <col class="label-props-col-presets" />
          </colgroup>
          <caption>标签属性：</caption>
          <thead>
            <tr>
              <th>属性</th>
              <th>值</th>
              <th>预设</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td colspan="2" class="props-inner-cell">
                <table class="props-inner-table">
                  <tbody>
                    <tr>
                      <td class="prop-key">预设名称：</td>
                      <td><input v-model="form.name" type="text" /></td>
                    </tr>
                    <tr>
                      <td class="prop-key">纸张大小：</td>
                      <td>
                        <select v-model="form.papersize">
                          <option v-for="paper in paperOptions" :key="paper" :value="paper">{{ paper }}</option>
                        </select>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">行：</td>
                      <td>
                        <select v-model.number="form.rows">
                          <option v-for="value in rowOptions" :key="`row-${value}`" :value="value">{{ value }}</option>
                        </select>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">列：</td>
                      <td>
                        <select v-model.number="form.cols">
                          <option v-for="value in colOptions" :key="`col-${value}`" :value="value">{{ value }}</option>
                        </select>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">宽度：</td>
                      <td>
                        <div class="prop-input-unit-row">
                          <input v-model="form.lwidth" type="text" />
                          <span class="prop-unit">mm</span>
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">高度：</td>
                      <td>
                        <div class="prop-input-unit-row">
                          <input v-model="form.lheight" type="text" />
                          <span class="prop-unit">mm</span>
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">垂直间距：</td>
                      <td>
                        <div class="prop-input-unit-row">
                          <input v-model="form.vpitch" type="text" />
                          <span class="prop-unit">mm</span>
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">水平间距：</td>
                      <td>
                        <div class="prop-input-unit-row">
                          <input v-model="form.hpitch" type="text" />
                          <span class="prop-unit">mm</span>
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">上边距：</td>
                      <td>
                        <div class="prop-input-unit-row">
                          <input v-model="form.tmargin" type="text" />
                          <span class="prop-unit">mm</span>
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">下边距：</td>
                      <td>
                        <div class="prop-input-unit-row">
                          <input v-model="form.bmargin" type="text" />
                          <span class="prop-unit">mm</span>
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">左边距：</td>
                      <td>
                        <div class="prop-input-unit-row">
                          <input v-model="form.lmargin" type="text" />
                          <span class="prop-unit">mm</span>
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">右边距：</td>
                      <td>
                        <div class="prop-input-unit-row">
                          <input v-model="form.rmargin" type="text" />
                          <span class="prop-unit">mm</span>
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">边框颜色(0-255)：</td>
                      <td title="0=黑色，255=白色">
                        <div class="prop-input-unit-row">
                          <input v-model="form.border" type="text" />
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">文本填充：</td>
                      <td>
                        <div class="prop-input-unit-row">
                          <input v-model="form.padding" type="text" />
                          <span class="prop-unit">mm</span>
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">字体大小：</td>
                      <td>
                        <div class="prop-input-unit-row">
                          <input v-model="form.fontsize" type="text" />
                          <span class="prop-unit">pt</span>
                          <small>(1pt=0.3527 mm)</small>
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">编号字体大小：</td>
                      <td>
                        <div class="prop-input-unit-row">
                          <input v-model="form.idfontsize" type="text" />
                          <span class="prop-unit">pt</span>
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">标题字体大小：</td>
                      <td>
                        <div class="prop-input-unit-row">
                          <input v-model="form.headerfontsize" type="text" />
                          <span class="prop-unit">mm</span>
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">条码大小：</td>
                      <td>
                        <div class="prop-input-unit-row">
                          <input v-model="form.barcodesize" type="text" />
                          <span class="prop-unit">mm</span>
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">页头图片：</td>
                      <td>
                        <div class="prop-image-row">
                          <input v-model="form.image" type="text" class="prop-image-input" />
                          <img class="prop-image-preview" :src="form.image" alt="预览" />
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">图片大小(WxH)：</td>
                      <td>
                        <div class="prop-size-row">
                          <input v-model="form.imagewidth" type="text" class="prop-size-input" />
                          <span class="prop-size-sep">X</span>
                          <input v-model="form.imageheight" type="text" class="prop-size-input" />
                          <span class="prop-unit">mm</span>
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">页头(_NL_换行)：</td>
                      <td>
                        <div class="prop-textarea-row">
                          <textarea v-model="form.headertext" rows="2" />
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">QR 条码：</td>
                      <td class="prop-qr-cell">
                        <div class="prop-inline-check-row">
                          <input v-model="form.wantbarcode" type="checkbox" />
                          <span
                            class="quick-tip quick-tip-below prop-qr-tip"
                            data-quick-tip="在二维码编号前追加文本，例如：http://服务器地址/itdb/?action=edititem&id="
                          >
                            <input v-model="form.qrtext" type="text" class="prop-qr-input" />
                          </span>
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">页头文字：</td>
                      <td class="prop-check-td">
                        <div class="prop-check-cell">
                          <input v-model="form.wantheadertext" type="checkbox" />
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">页头图片开关：</td>
                      <td class="prop-check-td">
                        <div class="prop-check-cell">
                          <input v-model="form.wantheaderimage" type="checkbox" />
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">无文本：</td>
                      <td class="prop-check-td">
                        <div class="prop-check-cell">
                          <span
                            class="quick-tip quick-tip-below prop-check-tip"
                            data-quick-tip="仅打印条码，不显示文字"
                          >
                            <input v-model="form.wantnotext" type="checkbox" />
                          </span>
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">条码右侧文字：</td>
                      <td class="prop-check-td">
                        <div class="prop-check-cell">
                          <input v-model="form.wantraligntext" type="checkbox" />
                        </div>
                      </td>
                    </tr>
                    <tr>
                      <td class="prop-key">忽略：</td>
                      <td>
                        <div class="prop-input-unit-row">
                          <span
                            class="quick-tip prop-ignore-tip"
                            data-quick-tip="当顶部标签已经被打印过时使用"
                          >
                            <input v-model="form.labelskip" type="text" />
                          </span>
                          <span class="prop-unit">标签</span>
                        </div>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </td>

              <td class="presets-cell">
                <div class="presets-wrap">
                  <div class="presets-list-wrap">
                    <p v-if="loadingPresets" class="muted-text">预设加载中...</p>
                    <template v-else>
                      <div v-for="preset in presets" :key="preset.id" class="preset-row">
                        <a href="#" class="preset-link" @click.prevent="fillFormFromPreset(preset)">{{ preset.name }}</a>
                        <button
                          v-if="canWrite"
                          class="preset-delete-btn"
                          type="button"
                          title="删除预设"
                          @click="requestDeletePreset(preset.id)"
                        >
                          <img src="/images/delete.png" alt="删" />
                        </button>
                      </div>
                      <p v-if="presets.length === 0" class="muted-text">暂无预设</p>
                    </template>
                  </div>

                  <div class="presets-tools-wrap">
                    <button v-if="canWrite" class="save-preset-btn" type="button" :disabled="savingPreset" @click="savePreset">
                      {{ savingPreset ? '保存中...' : '保存为新预设' }}
                    </button>

                    <img class="label-guide-image" src="/images/labelinfo.jpg" alt="标签参数示意图" />
                  </div>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </aside>

      <section ref="previewSectionRef" class="labels-card labels-preview-panel">
        <div class="labels-preview-head">
          <h3>预览结果（{{ previewRows.length }}）</h3>
          <div class="labels-preview-meta">
            <span>{{ previewPaperInfo.key }} {{ previewPaperInfo.widthMm }}×{{ previewPaperInfo.heightMm }} mm</span>
            <span>{{ previewColumns }} 列 × {{ previewRowsPerPage }} 行</span>
            <span>跳过 {{ previewSkipCount }} 个</span>
            <span>水平间距 {{ previewGapX.toFixed(0) }} px / 垂直间距 {{ previewGapY.toFixed(0) }} px</span>
            <span>{{ form.name.trim() || '未命名预设' }}</span>
          </div>
        </div>

        <div v-if="previewPages.length === 0" class="labels-preview-empty muted-text">请选择硬件后生成预览</div>
        <div v-else class="labels-preview-pages">
          <section v-for="(page, pageIndex) in previewPages" :key="`preview-page-${pageIndex}`" class="labels-preview-sheet">
            <header>第 {{ pageIndex + 1 }} 页</header>
            <div class="labels-preview-sheet-scroller">
              <div class="labels-preview-sheet-canvas" :style="previewSheetCanvasStyle">
                <div class="labels-preview-grid" :style="previewGridStyle">
                <template v-for="cell in page" :key="cell.key">
                  <div v-if="cell.kind === 'skip'" class="labels-preview-skip" :style="previewCardVars">跳过</div>
                  <div v-else-if="cell.kind === 'blank'" class="labels-preview-blank" :style="previewCardVars" aria-hidden="true" />
                  <article
                    v-else-if="cell.kind === 'label' && cell.row"
                    class="labels-preview-card"
                    :class="{ 'is-right-text': form.wantbarcode && form.wantraligntext, 'is-no-text': form.wantnotext }"
                    :style="previewCardVars"
                  >
                    <div v-if="previewImageSrc || previewHeaderLines.length > 0" class="labels-preview-header">
                      <img v-if="previewImageSrc" :src="previewImageSrc" alt="标签页头图片" />
                      <div v-if="previewHeaderLines.length > 0" class="labels-preview-header-text">
                        <span v-for="(line, lineIndex) in previewHeaderLines" :key="`${cell.row.id}-header-${lineIndex}`">{{ line }}</span>
                      </div>
                    </div>

                    <div class="labels-preview-body">
                      <div v-if="form.wantbarcode" class="labels-preview-barcode">
                        <div class="labels-preview-barcode-code">
                          <img v-if="cell.row.qrImage" :src="cell.row.qrImage" :alt="cell.row.qrText || '二维码'" />
                          <strong v-else>QR</strong>
                        </div>
                        <small v-if="!form.wantnotext">{{ cell.row.qrText || '-' }}</small>
                      </div>
                      <div v-if="!form.wantnotext" class="labels-preview-text">
                        <div class="labels-preview-id">{{ getPreviewIdLine(cell.row) }}</div>
                        <div
                          v-for="(line, lineIndex) in getPreviewBodyLines(cell.row)"
                          :key="`${cell.row.id}-line-${lineIndex}`"
                          class="labels-preview-line"
                        >
                          {{ line }}
                        </div>
                      </div>
                    </div>
                  </article>
                </template>
                </div>
              </div>
            </div>
          </section>
        </div>
      </section>
    </div>

    <div v-if="canWrite && confirmOpen" class="dialog-mask">
      <section class="drawer modal-narrow" role="dialog" aria-modal="true">
        <div class="drawer-header">
          <h3>删除确认</h3>
          <button class="dialog-close-btn quick-tip" type="button" aria-label="关闭" data-quick-tip="关闭" @click="closeConfirm">×</button>
        </div>
        <div class="drawer-form">
          <p style="text-align: center">{{ confirmMessage }}</p>
          <div class="inline-actions">
            <button class="danger" type="button" :disabled="deletingPreset" @click="confirmDeletePreset">
              {{ deletingPreset ? '删除中...' : '确认删除' }}
            </button>
            <button class="ghost-btn" type="button" :disabled="deletingPreset" @click="closeConfirm">取消</button>
          </div>
        </div>
      </section>
    </div>
  </section>
</template>

<style scoped>
.labels-page {
  overflow: visible;
}

.labels-layout {
  display: grid;
  grid-template-columns: 1fr;
  gap: 16px;
  align-items: start;
}

.labels-main-panel {
  border: 1px solid #9dbde2;
  border-radius: 10px;
  background: #f8fbff;
  box-shadow: inset 0 0 0 1px rgba(177, 206, 236, 0.26);
  padding: 10px;
  min-width: 0;
}

.labels-main-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
  flex-wrap: wrap;
}

.labels-order-row {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 0.95rem;
}

.labels-order-row a {
  color: #2349cc;
  text-decoration: none;
  font-weight: 700;
}

.labels-order-row a.active {
  color: #0f2f90;
  text-decoration: underline;
}

.labels-filter-row {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.labels-filter-input {
  min-width: 220px;
  width: 220px;
}

.labels-list-wrap {
  max-height: 470px;
}

.labels-items-table {
  min-width: 720px;
}

.labels-items-table td {
  padding-top: 6px;
  padding-bottom: 6px;
}

.labels-items-table tbody tr:not(.labels-status-row) td {
  text-align: center;
  vertical-align: middle;
}

.labels-status-row td {
  color: #fff;
  font-weight: 700;
  text-align: left;
  letter-spacing: 0.01em;
  padding: 6px 10px;
  text-shadow: 0 1px 1px rgba(0, 0, 0, 0.22);
}

.label-text-cell {
  text-align: center;
}

.make-label-btn {
  margin-top: 10px;
  border-radius: 9px;
  border: 1px solid #ff5555;
  background: #cc0000;
  color: #fff;
  padding: 6px 12px;
  font-size: 1.25rem;
  font-weight: 700;
  line-height: 1.1;
}

.make-label-btn:hover {
  background: #cc3333;
}

.labels-main-actions {
  margin-top: 10px;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}

.print-label-btn {
  margin-top: 10px;
  border-radius: 9px;
  border: 1px solid #2563eb;
  background: linear-gradient(180deg, #eff6ff 0%, #dbeafe 100%);
  color: #123b7a;
  padding: 9px 16px;
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.1;
}

.print-label-btn:hover {
  background: linear-gradient(180deg, #dbeafe 0%, #bfdbfe 100%);
}

.print-label-btn:disabled,
.make-label-btn:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.labels-help-list {
  margin: 14px 0 0;
  padding-left: 22px;
  line-height: 1.35;
}

.labels-preview-panel {
  display: grid;
  gap: 12px;
}

.labels-preview-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}

.labels-preview-head h3 {
  margin: 0;
}

.labels-preview-meta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.labels-preview-meta span {
  display: inline-flex;
  align-items: center;
  border: 1px solid #d9e4ef;
  border-radius: 999px;
  background: #fff;
  padding: 3px 9px;
  color: #4b637b;
  font-size: 0.84rem;
}

.labels-preview-empty {
  min-height: 120px;
  display: grid;
  place-items: center;
  border: 1px dashed #c8d6e4;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.72);
}

.labels-preview-pages {
  display: grid;
  gap: 16px;
}

.labels-preview-sheet {
  border: 1px solid #d8e1ea;
  border-radius: 12px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fbff 100%);
  padding: 12px;
  box-shadow: 0 8px 18px rgba(16, 42, 67, 0.06);
}

.labels-preview-sheet > header {
  margin-bottom: 10px;
  color: #1f4d79;
  font-weight: 700;
}

.labels-preview-sheet-scroller {
  overflow: auto;
}

.labels-preview-sheet-canvas {
  box-sizing: border-box;
  border: 1px solid rgba(188, 203, 219, 0.88);
  border-radius: 8px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(245, 249, 252, 0.98)),
    repeating-linear-gradient(
      0deg,
      rgba(184, 198, 214, 0.06) 0,
      rgba(184, 198, 214, 0.06) 1px,
      transparent 1px,
      transparent 18px
    );
  box-shadow:
    0 10px 22px rgba(16, 42, 67, 0.08),
    inset 0 0 0 1px rgba(255, 255, 255, 0.55);
}

.labels-preview-grid {
  display: grid;
  justify-content: start;
  align-items: start;
}

.labels-preview-card,
.labels-preview-skip,
.labels-preview-blank {
  width: var(--label-preview-width);
  min-height: var(--label-preview-height);
  height: var(--label-preview-height);
  box-sizing: border-box;
}

.labels-preview-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  overflow: hidden;
  border: 1px solid var(--label-preview-border);
  border-radius: 4px;
  background: #fff;
  padding: var(--label-preview-padding);
}

.labels-preview-header {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-height: 0;
}

.labels-preview-header img {
  width: var(--label-preview-image-width);
  height: var(--label-preview-image-height);
  object-fit: contain;
  flex: 0 0 auto;
}

.labels-preview-header-text {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  color: #0f4d83;
  font-size: var(--label-preview-header-size);
  font-weight: 700;
  line-height: 1.12;
}

.labels-preview-header-text > span {
  white-space: pre-line;
}

.labels-preview-body {
  min-height: 0;
  flex: 1 1 auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.labels-preview-card.is-right-text .labels-preview-body {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: start;
  gap: 10px;
}

.labels-preview-card.is-no-text .labels-preview-body {
  align-items: center;
  justify-content: center;
}

.labels-preview-barcode {
  width: var(--label-preview-barcode-size);
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 0 0 auto;
  color: #173b5e;
  text-align: center;
}

.labels-preview-barcode-code {
  width: var(--label-preview-barcode-size);
  height: var(--label-preview-barcode-size);
  border: 1px dashed #274763;
  border-radius: 6px;
  background: #fff;
  display: grid;
  place-items: center;
  padding: 6px;
}

.labels-preview-barcode-code img {
  width: 100%;
  height: 100%;
  display: block;
  object-fit: contain;
}

.labels-preview-barcode-code strong {
  font-size: 0.8rem;
  line-height: 1;
  letter-spacing: 0.08em;
}

.labels-preview-barcode small {
  max-width: 100%;
  font-size: 0.62rem;
  line-height: 1.2;
  overflow-wrap: anywhere;
}

.labels-preview-text {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
  color: #111827;
  font-size: var(--label-preview-font-size);
  line-height: 1.15;
}

.labels-preview-id {
  font-size: var(--label-preview-id-size);
  font-weight: 800;
  line-height: 1.05;
}

.labels-preview-line {
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.labels-preview-skip,
.labels-preview-blank {
  border-radius: 4px;
}

.labels-preview-skip {
  display: grid;
  place-items: center;
  border: 1px dashed #c6d4e2;
  background:
    linear-gradient(135deg, rgba(231, 239, 248, 0.9), rgba(246, 249, 252, 0.95)),
    repeating-linear-gradient(135deg, rgba(142, 164, 188, 0.18) 0, rgba(142, 164, 188, 0.18) 8px, transparent 8px, transparent 16px);
  color: #66809a;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.labels-preview-blank {
  border: 1px dashed rgba(156, 173, 191, 0.55);
  background: rgba(244, 247, 250, 0.92);
}

.labels-props-panel {
  border-radius: 10px;
  background: #d4eafd;
  box-shadow: inset 0 0 12px rgba(16, 42, 67, 0.3);
  padding: 10px 10px 12px;
  min-width: 0;
  max-width: 100%;
  overflow: auto;
}

.label-props-table {
  width: 100%;
  min-width: 720px;
  border-collapse: collapse;
  table-layout: fixed;
  background: #fff;
  font-size: 0.86rem;
}

.label-props-col-name {
  width: 28%;
}

.label-props-col-value {
  width: 46%;
}

.label-props-col-presets {
  width: 26%;
}

.label-props-table caption {
  text-align: left;
  margin: 0;
  padding: 10px 12px 12px;
  color: #404b57;
  font-weight: 700;
  font-size: clamp(1.5rem, 1.1rem + 1.2vw, 2.05rem);
  line-height: 1.08;
  font-size: 25px;
}

.label-props-table th,
.label-props-table td {
  border: 1px solid #dde5ef;
  padding: 0;
  vertical-align: top;
}

.label-props-table thead th {
  background: linear-gradient(180deg, #2f5ea0 0%, #5f90cf 100%);
  color: #fff;
  font-weight: 500;
  text-align: center;
  padding: 7px 10px;
}

.props-inner-cell {
  padding: 0 !important;
  vertical-align: top;
  overflow: visible;
}

.props-inner-table {
  width: 100%;
  max-width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.props-inner-table td {
  border: 1px solid #dde5ef;
  padding: 3px 6px;
  text-align: left;
  min-height: 31px;
  height: auto;
  min-width: 0;
  overflow: visible;
  box-sizing: border-box;
}

.props-inner-table td.prop-key {
  width: 40%;
  text-align: right;
  background: #f1f4f8;
  color: #1f2f43;
  font-weight: 500;
  white-space: nowrap;
}

.props-inner-table input:not([type='checkbox']),
.props-inner-table select,
.props-inner-table textarea {
  width: min(100%, 170px);
  min-width: 0;
  max-width: 100%;
  border-radius: 3px;
  padding: 3px 6px;
  box-sizing: border-box;
}

.props-inner-table textarea {
  width: 100%;
  min-width: 0;
  resize: vertical;
}

.prop-input-unit-row {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  min-width: 0;
  flex-wrap: wrap;
}

.prop-input-unit-row input:not([type='checkbox']),
.prop-input-unit-row select {
  flex: 1 1 132px;
  width: auto !important;
  min-width: 0 !important;
  max-width: 170px;
}

.prop-input-unit-row small {
  color: #5f6f84;
  white-space: nowrap;
}

.prop-unit {
  white-space: nowrap;
}

.prop-image-row {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  min-width: 0;
  flex-wrap: nowrap;
}

.prop-image-input {
  width: 0 !important;
  min-width: 0 !important;
  max-width: none !important;
  flex: 1 1 auto;
}

.prop-image-preview {
  width: 24px;
  height: 24px;
  border-radius: 3px;
  border: 1px solid #cad6e5;
  object-fit: contain;
  background: #fff;
}

.prop-size-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.prop-size-input {
  width: 100% !important;
  min-width: 0 !important;
  max-width: none !important;
}

.prop-size-sep {
  white-space: nowrap;
}

.prop-textarea-row {
  width: 100%;
  min-width: 0;
}

.prop-textarea-row textarea {
  width: 100% !important;
  min-width: 0 !important;
  max-width: none !important;
}

.prop-inline-check-row {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  column-gap: 8px;
  width: 100%;
  min-width: 0;
}

.prop-inline-check-row input[type='checkbox'] {
  margin: 0;
  flex: 0 0 auto;
}

.prop-qr-input {
  width: 100% !important;
  min-width: 120px !important;
  max-width: none !important;
}

.prop-qr-tip {
  display: block;
  width: 100%;
  min-width: 0;
}

.prop-qr-cell,
.prop-check-td {
  text-align: left;
}

.prop-check-cell {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  min-height: 24px;
  width: 100%;
}

.prop-check-cell input[type='checkbox'] {
  margin: 0;
}

.prop-check-tip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 100%;
}

.prop-ignore-tip {
  display: inline-flex;
  flex: 1 1 132px;
  min-width: 0;
  max-width: 170px;
}

.prop-ignore-tip input {
  width: 100% !important;
  min-width: 0 !important;
  max-width: none !important;
}

.presets-cell {
  width: 28%;
  min-width: 0;
  vertical-align: top;
  background: #f8fbff;
}

.presets-wrap {
  padding: 8px 8px 10px;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
  max-height: 760px;
  overflow: auto;
}

.presets-list-wrap {
  display: grid;
  align-content: start;
  gap: 2px;
}

.presets-tools-wrap {
  display: grid;
  align-content: start;
  justify-items: center;
  gap: 8px;
  padding-top: 0;
}

.preset-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  min-height: 22px;
}

.preset-link {
  color: #2349cc;
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preset-link:hover {
  text-decoration: underline;
}

.preset-delete-btn {
  border: 0;
  background: transparent;
  padding: 0;
  width: 22px;
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.preset-delete-btn img {
  width: 18px;
  height: 18px;
}

.save-preset-btn {
  margin-top: 0;
  justify-self: center;
  border-radius: 4px;
  padding: 5px 12px;
  font-size: 0.88rem;
}

.label-guide-image {
  width: min(228px, 100%);
  justify-self: center;
  border: 1px solid #ccd8e8;
  border-radius: 4px;
  background: #fff;
  margin-top: 0;
}

@media (max-width: 1760px) {
  .label-props-col-name {
    width: 30%;
  }

  .label-props-col-value {
    width: 45%;
  }

  .label-props-col-presets {
    width: 25%;
  }

  .props-inner-table td.prop-key {
    width: 42%;
  }
}

@media (max-width: 1720px) {
  .labels-layout {
    grid-template-columns: 1fr;
  }

  .labels-items-table {
    min-width: 680px;
  }

  .label-props-col-name {
    width: 32%;
  }

  .label-props-col-value {
    width: 46%;
  }

  .label-props-col-presets {
    width: 22%;
  }

  .presets-wrap {
    max-height: 520px;
  }
}

@media (max-width: 980px) {
  .label-props-table {
    font-size: 0.82rem;
  }

  .props-inner-table td.prop-key {
    width: 44%;
    white-space: normal;
    word-break: break-word;
  }

  .props-inner-table input:not([type='checkbox']),
  .props-inner-table select,
  .props-inner-table textarea {
    width: 100%;
    min-width: 0;
  }
}
</style>
