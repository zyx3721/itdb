<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '../api/client'
import { useTableSort } from '../composables/useTableSort'
import DateInput from '../components/DateInput.vue'
import { resourceMap, type FieldType, type ResourceConfig, type ResourceField } from '../resources'
import { useBootstrapStore } from '../stores/bootstrap'
import { useAuthStore } from '../stores/auth'
import { useNoticeStore } from '../stores/notice'

type GenericRow = Record<string, unknown>
type SortDirection = 'asc' | 'desc'
type ItemEditorTabKey = 'itemData' | 'itemLinks' | 'invoiceLinks' | 'logs' | 'softwareLinks' | 'contractLinks' | 'files'
type ItemOverviewTabKey = 'items' | 'software' | 'invoices' | 'contracts'
type SoftwareEditorTabKey = 'softwareData' | 'itemLinks' | 'invoiceLinks' | 'contractLinks' | 'files'
type InvoiceEditorTabKey = 'invoiceData' | 'itemLinks' | 'softwareLinks' | 'contractLinks' | 'files'
type ContractEditorTabKey = 'contractData' | 'events' | 'itemLinks' | 'softwareLinks' | 'invoiceLinks' | 'files'
type FileEditorTabKey = 'fileData' | 'itemLinks' | 'softwareLinks' | 'contractLinks'
type SoftwareOverviewTabKey = 'items' | 'invoices' | 'contracts'
type InvoiceOverviewTabKey = 'items' | 'software' | 'contracts'
type ContractOverviewTabKey = 'items' | 'software' | 'invoices'
type FileOverviewTabKey = 'items' | 'software' | 'invoices' | 'contracts'
type AgentOverviewTabKey = 'items' | 'software' | 'invoicesVendor' | 'invoicesBuyer'
type LocationOverviewTabKey = 'items' | 'racks'
type OverviewResourceKey = 'items' | 'software' | 'invoices' | 'contracts' | 'racks'
type OverviewRow = {
  id: number
  index: number
  text: string
  resourceKey: OverviewResourceKey
  tip: string
}
type ItemRelationRow = {
  id: number
  itemTypeID: number
  itemType: string
  manufacturer: string
  model: string
  label: string
  functionText: string
  dnsName: string
  username: string
  statusText: string
  statusColor: string
  ipv4: string
  principal: string
  sn: string
  userID: number
  locationID: number
  rackID: number
  manufacturerID: number
  uSize: number
  rackPosition: number
  rackPosDepth: number
}
type RackViewDepth = 'F' | 'M' | 'B'
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
}
type RackViewRow = {
  unit: number
  cells: RackViewCell[]
}
type RackViewBuildResult = {
  rows: RackViewRow[]
  warnings: string[]
  moreItems: ItemRelationRow[]
}
type InvoiceRelationRow = {
  id: number
  number: string
  vendor: string
  buyer: string
  description: string
  files: string
  date: string
  vendorID: number
  buyerID: number
}
type InvoiceFileEntry = SoftwareInstallEntry & {
  title: string
  fileName: string
  previewTip: string
}
type SoftwareRelationRow = {
  id: number
  title: string
  version: string
  manufacturer: string
  manufacturerID: number
}
type ContractRelationRow = {
  id: number
  number: string
  title: string
  type: string
  startDate: string
  currentEndDate: string
  contractor: string
  contractorID: number
}
type FileRelationRow = {
  id: number
  typeID: number
  typeDesc: string
  title: string
  fileName: string
  date: string
}
type ManagedLinkedFileRow = FileRelationRow & {
  fname: string
  directlyLinked: boolean
  viaInvoiceIDs: number[]
  viaInvoiceNumbers: string[]
}
type SoftwareInvoiceDisplayEntry = {
  id: number
  number: string
  files: InvoiceFileEntry[]
}
type AgentContactRow = { name: string; phones: string; email: string; role: string; comments: string }
type AgentTypeBadge = { key: number; label: string; className: string }
type AgentURLRow = { description: string; url: string }
type RelationCacheKey = 'items' | 'software' | 'invoices' | 'contracts' | 'files'
type ContractRenewalRow = {
  endDateBefore: string
  endDateAfter: string
  effectiveDate: string
  notes: string
  enteredDate: string
  enteredBy: string
}
type DeleteTarget =
  | { kind: 'row'; id: number }
  | { kind: 'rows'; ids: number[] }
  | { kind: 'contractEvent'; contractID: number; eventID: number }
  | { kind: 'locationArea'; locationID: number; areaID: number; areaName: string }
type SoftwareInstallEntry = { index: number; id: number | null; text: string }
type LinkedUploadScope = 'software' | 'items' | 'invoices' | 'contracts'
type EditorDataDirtyPayload = {
  sourceId: string
  timestamp: number
  relationKeys?: RelationCacheKey[]
  bootstrapKeys?: string[]
}

const route = useRoute()
const router = useRouter()
const bootstrap = useBootstrapStore()
const auth = useAuthStore()
const noticeStore = useNoticeStore()
const tagIDByName = computed(() => {
  const rows = (bootstrap.lookups.tags ?? []) as Record<string, unknown>[]
  const map = new Map<string, number>()
  for (const row of rows) {
    const name = String(row.name ?? '').trim()
    const id = Number(row.id ?? 0)
    if (!name || !Number.isFinite(id) || id <= 0) continue
    map.set(name.toLowerCase(), id)
  }
  return map
})

const resource = computed<ResourceConfig | null>(() => {
  const key = route.params.resource as string
  return resourceMap[key] ?? null
})

const rows = ref<GenericRow[]>([])
const recordDetail = ref<GenericRow | null>(null)
const selectedId = ref<number | null>(null)
const form = reactive<Record<string, any>>({})
const pendingCleanupFileLinks = ref<number[]>([])
const error = ref('')
const loading = ref(false)
const saving = ref(false)
const search = ref('')
const drawerOpen = ref(false)
const confirmOpen = ref(false)
const deleting = ref(false)
const deleteTarget = ref<DeleteTarget | null>(null)
const selectedRowIds = ref<number[]>([])
const pageSize = ref(25)
const page = ref(1)
const activeItemTab = ref<ItemEditorTabKey>('itemData')
const activeOverviewTab = ref<ItemOverviewTabKey>('items')
const activeSoftwareTab = ref<SoftwareEditorTabKey>('softwareData')
const activeSoftwareOverviewTab = ref<SoftwareOverviewTabKey>('items')
const activeInvoiceTab = ref<InvoiceEditorTabKey>('invoiceData')
const activeInvoiceOverviewTab = ref<InvoiceOverviewTabKey>('items')
const activeContractTab = ref<ContractEditorTabKey>('contractData')
const activeContractOverviewTab = ref<ContractOverviewTabKey>('items')
const activeFileTab = ref<FileEditorTabKey>('fileData')
const activeFileOverviewTab = ref<FileOverviewTabKey>('items')
const activeAgentOverviewTab = ref<AgentOverviewTabKey>('items')
const activeLocationOverviewTab = ref<LocationOverviewTabKey>('items')
const itemActions = ref<GenericRow[]>([])
const itemTags = ref<string[]>([])
const itemTagEditorOpen = ref(false)
const itemTagInput = ref('')
const itemTagSaving = ref(false)
const itemTagMessage = ref('')
const softwareTags = ref<string[]>([])
const softwareTagEditorOpen = ref(false)
const softwareTagInput = ref('')
const softwareTagSaving = ref(false)
const softwareTagMessage = ref('')
const locationAreas = ref<GenericRow[]>([])
const locationAreasLoading = ref(false)
const locationAreaSaving = ref(false)
const editingLocationAreaId = ref<number | null>(null)
const locationAreaName = ref('')
const nextPendingLocationAreaId = ref(1)
const locationFloorplanInput = ref<HTMLInputElement | null>(null)
const fileUploadInput = ref<HTMLInputElement | null>(null)
const softwareUploadInput = ref<HTMLInputElement | null>(null)
const itemUploadInput = ref<HTMLInputElement | null>(null)
const invoiceUploadInput = ref<HTMLInputElement | null>(null)
const contractUploadInput = ref<HTMLInputElement | null>(null)
const locationFloorplanPreviewURL = ref('')
const locationFloorplanLoading = ref(false)
const contractEvents = ref<GenericRow[]>([])
const contractEventsLoading = ref(false)
const contractEventSaving = ref(false)
const editingContractEventId = ref<number | null>(null)
const contractEventForm = reactive({
  siblingId: '',
  startDate: '',
  endDate: '',
  description: '',
})
const locationFloorplanImageExtensions = new Set(['.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp', '.svg', '.avif'])
const locationFloorplanAccept = Array.from(locationFloorplanImageExtensions).join(',')
const invoiceUploadAllowedExtensions = new Set([...locationFloorplanImageExtensions, '.pdf'])
const invoiceUploadAccept = Array.from(invoiceUploadAllowedExtensions).join(',')
const relationCache = reactive<Record<RelationCacheKey, GenericRow[]>>({
  items: [],
  software: [],
  invoices: [],
  contracts: [],
  files: [],
})
const relationCacheLoading = reactive<Record<RelationCacheKey, boolean>>({
  items: false,
  software: false,
  invoices: false,
  contracts: false,
  files: false,
})
const dirtyRelationCacheKeys = new Set<RelationCacheKey>()
const dirtyBootstrapLookupKeys = new Set<string>()
const relationCacheRefreshPromises: Partial<Record<RelationCacheKey, Promise<boolean>>> = {}
let bootstrapSnapshotPromise: Promise<Record<string, Record<string, unknown>[]> | null> | null = null
const resourceEditorSyncSourceId = `resource-page-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
const resourceEditorSyncChannelName = 'itdb-resource-editor-sync'
const resourceEditorSyncStorageKey = 'itdb-resource-editor-sync'
let resourceEditorSyncChannel: BroadcastChannel | null = null
const agentContacts = ref<AgentContactRow[]>([{ name: '', phones: '', email: '', role: '', comments: '' }])
const agentURLs = ref<AgentURLRow[]>([{ description: '', url: '' }])
const itemLinkFilter = ref('')
const invoiceLinkFilter = ref('')
const softwareLinkFilter = ref('')
const contractLinkFilter = ref('')
const fileLinkFilter = ref('')
const agentContactInfo = ref('')
const softwareRelationSortState = reactive({
  items: { key: 'id', direction: 'asc' as SortDirection },
  invoices: { key: 'id', direction: 'asc' as SortDirection },
  contracts: { key: 'id', direction: 'asc' as SortDirection },
  files: { key: 'id', direction: 'asc' as SortDirection },
})
type NonSoftwareRelationSortScope =
  | 'itemItems'
  | 'itemInvoices'
  | 'itemSoftware'
  | 'itemContracts'
  | 'invoiceItems'
  | 'invoiceSoftware'
  | 'invoiceContracts'
  | 'contractItems'
  | 'contractSoftware'
  | 'contractInvoices'
  | 'fileItems'
  | 'fileSoftware'
  | 'fileContracts'
const nonSoftwareRelationSortState = reactive<Record<NonSoftwareRelationSortScope, { key: string; direction: SortDirection }>>({
  itemItems: { key: 'id', direction: 'asc' },
  itemInvoices: { key: 'id', direction: 'asc' },
  itemSoftware: { key: 'id', direction: 'asc' },
  itemContracts: { key: 'id', direction: 'asc' },
  invoiceItems: { key: 'id', direction: 'asc' },
  invoiceSoftware: { key: 'id', direction: 'asc' },
  invoiceContracts: { key: 'id', direction: 'asc' },
  contractItems: { key: 'id', direction: 'asc' },
  contractSoftware: { key: 'id', direction: 'asc' },
  contractInvoices: { key: 'id', direction: 'asc' },
  fileItems: { key: 'id', direction: 'asc' },
  fileSoftware: { key: 'id', direction: 'asc' },
  fileContracts: { key: 'id', direction: 'asc' },
})
const softwareUploadForm = reactive({
  title: '',
  typeId: '',
  date: '',
  file: null as File | null,
})
const softwareUploading = ref(false)
const itemUploadForm = reactive({
  title: '',
  typeId: '',
  date: '',
  file: null as File | null,
})
const itemUploading = ref(false)
const invoiceUploadForm = reactive({
  title: '',
  date: '',
  file: null as File | null,
})
const invoiceUploading = ref(false)
const contractUploadForm = reactive({
  title: '',
  typeId: '',
  date: '',
  file: null as File | null,
})
const contractUploading = ref(false)
const contractRenewals = ref<ContractRenewalRow[]>([])
let loadRowsSeq = 0

const resourceEditorDependencies: Record<string, { relationKeys: RelationCacheKey[]; bootstrapKeys: string[] }> = {
  items: {
    relationKeys: ['items', 'software', 'invoices', 'contracts', 'files'],
    bootstrapKeys: ['itemtypes', 'dpttypes', 'statustypes', 'agents', 'users', 'locations', 'locareas', 'racks', 'tags', 'items_ref', 'software_ref', 'invoices_ref', 'contracts_ref', 'files_ref'],
  },
  software: {
    relationKeys: ['items', 'invoices', 'contracts', 'files'],
    bootstrapKeys: ['agents', 'filetypes', 'tags', 'items_ref', 'invoices_ref', 'contracts_ref', 'files_ref'],
  },
  invoices: {
    relationKeys: ['items', 'software', 'contracts', 'files'],
    bootstrapKeys: ['agents', 'items_ref', 'software_ref', 'contracts_ref', 'files_ref'],
  },
  contracts: {
    relationKeys: ['items', 'software', 'invoices', 'files'],
    bootstrapKeys: ['agents', 'contracttypes', 'contractsubtypes', 'users', 'filetypes', 'items_ref', 'software_ref', 'invoices_ref', 'contracts_ref', 'files_ref'],
  },
  files: {
    relationKeys: ['items', 'software', 'contracts'],
    bootstrapKeys: ['filetypes', 'items_ref', 'software_ref', 'contracts_ref', 'files_ref'],
  },
  agents: {
    relationKeys: ['items', 'software', 'invoices', 'contracts'],
    bootstrapKeys: ['agents'],
  },
  users: {
    relationKeys: ['items'],
    bootstrapKeys: ['users'],
  },
  locations: {
    relationKeys: ['items'],
    bootstrapKeys: ['locations', 'locareas', 'racks'],
  },
  racks: {
    relationKeys: ['items'],
    bootstrapKeys: ['locations', 'locareas', 'racks'],
  },
}

const resourceMutationImpact: Record<string, { relationKeys: RelationCacheKey[]; bootstrapKeys: string[] }> = {
  items: { relationKeys: ['items'], bootstrapKeys: ['items_ref'] },
  software: { relationKeys: ['software'], bootstrapKeys: ['software_ref'] },
  invoices: { relationKeys: ['invoices'], bootstrapKeys: ['invoices_ref'] },
  contracts: { relationKeys: ['contracts'], bootstrapKeys: ['contracts_ref'] },
  files: { relationKeys: ['files'], bootstrapKeys: ['files_ref'] },
  agents: { relationKeys: [], bootstrapKeys: ['agents'] },
  users: { relationKeys: [], bootstrapKeys: ['users'] },
  locations: { relationKeys: [], bootstrapKeys: ['locations'] },
  racks: { relationKeys: [], bootstrapKeys: ['racks'] },
}

const itemEditorTabs: { key: ItemEditorTabKey; label: string }[] = [
  { key: 'itemData', label: '硬件数据' },
  { key: 'itemLinks', label: '内部硬件关联' },
  { key: 'invoiceLinks', label: '关联单据' },
  { key: 'logs', label: '维护日志' },
  { key: 'softwareLinks', label: '软件关联' },
  { key: 'contractLinks', label: '关联合同' },
  { key: 'files', label: '上传文件' },
]

const softwareEditorTabs: { key: SoftwareEditorTabKey; label: string }[] = [
  { key: 'softwareData', label: '软件数据' },
  { key: 'itemLinks', label: '硬件关联' },
  { key: 'invoiceLinks', label: '单据关联' },
  { key: 'contractLinks', label: '合同关联' },
  { key: 'files', label: '上传文件' },
]

const invoiceEditorTabs: { key: InvoiceEditorTabKey; label: string }[] = [
  { key: 'invoiceData', label: '单据数据' },
  { key: 'itemLinks', label: '硬件关联' },
  { key: 'softwareLinks', label: '软件关联' },
  { key: 'contractLinks', label: '合同关联' },
  { key: 'files', label: '上传文件' },
]

const contractEditorTabs: { key: ContractEditorTabKey; label: string }[] = [
  { key: 'contractData', label: '合同数据' },
  { key: 'events', label: '事件历史' },
  { key: 'itemLinks', label: '硬件关联' },
  { key: 'softwareLinks', label: '软件关联' },
  { key: 'invoiceLinks', label: '单据关联' },
  { key: 'files', label: '上传文件' },
]

const fileEditorTabs: { key: FileEditorTabKey; label: string }[] = [
  { key: 'fileData', label: '文件数据' },
  { key: 'itemLinks', label: '硬件关联' },
  { key: 'softwareLinks', label: '软件关联' },
  { key: 'contractLinks', label: '合同关联' },
]

const pageableResourceKeys = new Set(['items', 'software', 'invoices'])
const ascPrimaryResourceKeys = new Set(['software', 'invoices', 'files', 'contracts', 'locations', 'users', 'racks', 'agents'])
const resourcePageConfig: Record<string, { defaultSize: number; sizeOptions: number[] }> = {
  items: { defaultSize: 18, sizeOptions: [10, 18, 25, 50, 100, -1] },
  software: { defaultSize: 6, sizeOptions: [6, 9, 25, 50, 100, -1] },
  agents: { defaultSize: 25, sizeOptions: [10, 25, 50, 100, -1] },
  files: { defaultSize: 25, sizeOptions: [10, 25, 50, 100, -1] },
  contracts: { defaultSize: 25, sizeOptions: [10, 25, 50, 100, -1] },
  locations: { defaultSize: 25, sizeOptions: [10, 25, 50, 100, -1] },
  users: { defaultSize: 25, sizeOptions: [10, 25, 50, 100, -1] },
  racks: { defaultSize: 25, sizeOptions: [10, 25, 50, 100, -1] },
}

function isInvoiceFileTypeRow(row: Record<string, unknown>) {
  const id = Number(row.id ?? 0)
  const text = String(row.typedesc ?? '').trim()
  return id === 3 || /^invoice$/i.test(text) || text === '发票'
}

function buildNonInvoiceUploadTypeOptions() {
  const rows = (bootstrap.lookups.filetypes ?? []) as Record<string, unknown>[]
  return rows
    .filter((row) => !isInvoiceFileTypeRow(row))
    .map((row) => ({
      value: Number(row.id ?? 0),
      label: decodeHtmlEntities(String(row.typedesc ?? row.id ?? '')),
    }))
    .filter((row) => row.value > 0 && row.label)
}

const softwareUploadTypeOptions = computed(() => buildNonInvoiceUploadTypeOptions())

const commonUploadTypeOptions = computed(() => buildNonInvoiceUploadTypeOptions())

const softwareLicenseQtyOptions = Array.from({ length: 400 }, (_, i) => i + 1)

const invoiceFileTypeID = computed(() => {
  const rows = (bootstrap.lookups.filetypes ?? []) as Record<string, unknown>[]
  const found = rows.find((row) => isInvoiceFileTypeRow(row))
  const id = Number(found?.id ?? 3)
  return Number.isFinite(id) && id > 0 ? id : 3
})

const invoiceFileTypeLabel = computed(() => {
  const rows = (bootstrap.lookups.filetypes ?? []) as Record<string, unknown>[]
  const found = rows.find((row) => Number(row.id ?? 0) === invoiceFileTypeID.value)
  const text = String(found?.typedesc ?? 'invoice').trim()
  return text || 'invoice'
})

const isInvoiceTypeFile = computed(() => Number(form.typeId ?? 0) === invoiceFileTypeID.value)
const isEditingInvoiceFile = computed(() => isFileResource.value && Number(selectedId.value ?? 0) > 0 && isInvoiceTypeFile.value)
const fileUploadAccept = computed(() => (isInvoiceTypeFile.value ? invoiceUploadAccept : ''))
const fileEditorTypeOptions = computed(() => {
  if (!isFileResource.value) return [] as { label: string; value: string | number }[]
  if (isEditingInvoiceFile.value) {
    return [
      {
        label: invoiceFileTypeLabel.value,
        value: String(invoiceFileTypeID.value),
      },
    ]
  }
  return getOptionsByFieldKey('typeId')
})
const showFileEditorTypePlaceholder = computed(() => !isEditingInvoiceFile.value)

const canWrite = computed(() => !auth.isReadOnly && !resource.value?.readonly)
const rowKey = (row: GenericRow) => Number(row.id ?? 0)
const readonlyResource = computed(() => !resource.value)
const primarySortKey = computed(() => resource.value?.columns[0]?.key ?? '')
const isItemResource = computed(() => resource.value?.key === 'items')
const isSoftwareResource = computed(() => resource.value?.key === 'software')
const isInvoiceResource = computed(() => resource.value?.key === 'invoices')
const isContractResource = computed(() => resource.value?.key === 'contracts')
const isFileResource = computed(() => resource.value?.key === 'files')
const isAgentResource = computed(() => resource.value?.key === 'agents')
const isLocationResource = computed(() => resource.value?.key === 'locations')
const isUserResource = computed(() => resource.value?.key === 'users')
const isRackResource = computed(() => resource.value?.key === 'racks')
const canEditNonInvoiceFileAssociations = computed(() => !isFileResource.value || !isInvoiceTypeFile.value)
const actionHeaderText = computed(() => resource.value?.actionHeader ?? '操作')
const deleteConfirmMessage = computed(() => {
  if (!deleteTarget.value) return ''
  if (deleteTarget.value.kind === 'rows') {
    return `确认批量删除已选择的 ${deleteTarget.value.ids.length} 条记录吗？`
  }
  if (deleteTarget.value.kind === 'contractEvent') {
    return `确认删除事件 编号=${deleteTarget.value.eventID} 吗？`
  }
  if (deleteTarget.value.kind === 'locationArea') {
    return `确认删除区域“${deleteTarget.value.areaName}”吗？`
  }
  return `确认删除 编号=${deleteTarget.value.id} 的记录吗？`
})

function canDeleteMainRow(row: GenericRow) {
  if (!canWrite.value) return false
  const id = rowKey(row)
  if (!id) return false
  if (resource.value?.key === 'users' && String(row.username ?? '').trim().toLowerCase() === 'admin') {
    return false
  }
  return true
}

function getSourceKey(field: ResourceField) {
  return field.readKey ?? field.key
}

function resetForm() {
  Object.keys(form).forEach((k) => delete form[k])
  pendingCleanupFileLinks.value = []
  if (!resource.value) return
  for (const field of resource.value.fields) {
    if (field.type === 'multiselect') {
      form[field.key] = []
      continue
    }
    if (field.type === 'file') {
      form[field.key] = null
      continue
    }
    form[field.key] = ''
  }
  form.fileLinks = []
}

function toOptionLabel(item: Record<string, unknown>) {
  if (item.name !== undefined && item.floor !== undefined) {
    const name = String(item.name ?? '').trim()
    const floor = String(item.floor ?? '').trim()
    return decodeHtmlEntities(floor ? `${name}, 楼层:${floor}` : name)
  }
  if (item.statusdesc !== undefined) {
    return decodeHtmlEntities(String(item.statusdesc ?? '').trim() || String(item.id ?? ''))
  }
  if (item.dptname !== undefined) {
    return decodeHtmlEntities(String(item.dptname ?? '').trim() || String(item.id ?? ''))
  }
  if (item.label || item.model) {
    const label = String(item.label ?? '').trim()
    const model = String(item.model ?? '').trim()
    const id = String(item.id ?? '').trim()
    return decodeHtmlEntities([id && `${id}:`, label, model].filter(Boolean).join(' '))
  }
  if (item.stitle || item.sversion) {
    return decodeHtmlEntities(`${String(item.stitle ?? '')} ${String(item.sversion ?? '')}`.trim())
  }
  if (item.number || item.title) {
    const prefix = String(item.number ?? '').trim()
    const suffix = String(item.title ?? '').trim()
    return decodeHtmlEntities([prefix, suffix].filter(Boolean).join(' / ') || String(item.id))
  }
  return decodeHtmlEntities(String(item.typedesc ?? item.name ?? item.username ?? item.title ?? item.areaname ?? item.label ?? item.id))
}

function formatRackOptionLabel(item: Record<string, unknown>) {
  const label = String(item.label ?? '').trim() || `机架${String(item.id ?? '').trim()}`
  const prefix = String(item.id ?? '').trim()
  const itemID = Number(item.id ?? 0)
  const fallbackRows = (bootstrap.lookups.racks ?? []) as Record<string, unknown>[]
  const fallback = fallbackRows.find((row) => Number(row.id ?? 0) === itemID) ?? null
  const units = Number.parseInt(String(item.usize ?? item.uSize ?? item.height ?? fallback?.usize ?? fallback?.uSize ?? '').trim(), 10)
  const content = units > 0 ? `${label},${units}U 晟图` : `${label} 晟图`
  return decodeHtmlEntities(prefix ? `${prefix}: ${content}` : content)
}

function toFieldOptionLabel(field: ResourceField, item: Record<string, unknown>) {
  if (resource.value?.key === 'contracts' && field.key === 'parentId') {
    const id = String(item.id ?? '').trim()
    const title = String(item.title ?? '').trim()
    return decodeHtmlEntities([id && `${id}:`, title].filter(Boolean).join(' ') || String(item.id ?? ''))
  }
  if (resource.value?.key === 'items' && field.key === 'rackId') {
    return formatRackOptionLabel(item)
  }
  return toOptionLabel(item)
}

async function recordRecentViewHistory(id: number) {
  if (!resource.value) return
  const rowID = Number(id)
  if (!Number.isFinite(rowID) || rowID <= 0) return

  const resolved = router.resolve({
    path: `/resources/${resource.value.key}`,
    query: { edit: String(rowID), vh: String(Date.now()) },
  })
  const description = `${resource.value.title}: ${rowID}`

  try {
    await api.post('/view-history', {
      url: resolved.fullPath,
      description,
    })
    window.dispatchEvent(new CustomEvent('itdb:view-history-updated'))
  } catch {
    // Ignore recent-history write failures so normal editing is not blocked.
  }
}

function toOptionValue(item: Record<string, unknown>) {
  return Number(item.id)
}

function getAgentTypeMask(item: Record<string, unknown>) {
  const mask = Number(item.type ?? 0)
  return Number.isFinite(mask) ? mask : 0
}

function getRackDepthMask(raw: unknown) {
  const value = Number(raw ?? 0)
  return Number.isFinite(value) && value > 0 ? value : 0
}

function parsePositiveInt(raw: unknown) {
  const value = Number.parseInt(String(raw ?? '').trim(), 10)
  return Number.isFinite(value) && value > 0 ? value : 0
}

function getRackUnitSpan(startUnit: number, units: number, reverse: boolean) {
  if (!Number.isFinite(startUnit) || startUnit <= 0 || !Number.isFinite(units) || units <= 0) return [] as number[]
  return reverse
    ? Array.from({ length: units }, (_, index) => startUnit + index)
    : Array.from({ length: units }, (_, index) => startUnit - index)
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

function getRackItemStatusStyle(statusText: string, statusColor = ''): Record<string, string> {
  const color = String(statusColor || getStatusColorByText(statusText)).trim()
  if (!color) return {}
  return {
    '--rack-cell-bg': color,
    '--rack-cell-fg': getReadableTextColor(color),
  }
}

function getRackItemTitle(row: ItemRelationRow) {
  return [`${row.manufacturer || '-'}`, `${row.model || '-'}`, row.id > 0 ? `[ID:${row.id}]` : '[新建]'].join(' ').trim()
}

function getRackItemSubtitle(row: ItemRelationRow) {
  const parts: string[] = []
  if (row.label.trim()) parts.push(row.label.trim())
  const firstIP = firstRackIPv4(row.ipv4)
  if (firstIP) parts.push(`[ip:${firstIP}]`)
  return parts.join(' ')
}

function getRackItemTip(row: ItemRelationRow) {
  const statusText = row.statusText.trim() ? `状态：${row.statusText.trim()}` : '状态：使用中'
  const sizeText = row.uSize > 0 ? `${row.uSize}U` : '-'
  const posText = row.rackPosition > 0 ? `${row.rackPosition}` : '-'
  const idText = row.id > 0 ? `编号：${row.id}` : '新建硬件'
  return `${idText} / ${statusText} / 位置：${posText}U / 高度：${sizeText}`
}

type BuildRackViewDataOptions = {
  totalUnits: number
  reverse: boolean
  itemRows: ItemRelationRow[]
}

function buildRackViewData(options: BuildRackViewDataOptions): RackViewBuildResult {
  const { totalUnits, reverse, itemRows } = options
  if (totalUnits <= 0) {
    return { rows: [], warnings: [], moreItems: [] }
  }

  const rackRows: Record<number, Partial<Record<RackViewDepth | `${RackViewDepth}T`, number>>> = {}
  const itemMap = new Map<number, ItemRelationRow>()
  const warnings: string[] = []
  const moreItems: ItemRelationRow[] = []
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

    const occupiedUnits = getRackUnitSpan(rackPosition, units, reverse)
    if (occupiedUnits.some((unit) => unit < 1 || unit > totalUnits)) {
      warnings.push(`硬件 ${item.id}（${item.manufacturer || '-'} ${item.model || '-'}）超出机架边界`)
      continue
    }

    for (const pos of occupiedUnits) {
      const rowState = (rackRows[pos] ??= {})
      const isTop = pos === rackPosition ? 1 : 0

      if ((depthMask & 4) === 4 && rowState.F && rowState.F !== item.id) {
        warnings.push(`第 ${pos}U 前侧位置冲突：硬件 ${item.id} 与 ${rowState.F}`)
      }
      if ((depthMask & 2) === 2 && rowState.M && rowState.M !== item.id) {
        warnings.push(`第 ${pos}U 中部位置冲突：硬件 ${item.id} 与 ${rowState.M}`)
      }
      if ((depthMask & 1) === 1 && rowState.B && rowState.B !== item.id) {
        warnings.push(`第 ${pos}U 后侧位置冲突：硬件 ${item.id} 与 ${rowState.B}`)
      }

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
          colspan = state.M === state.B ? 2 : 1
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
          })
        }
      } else if (!state.B) {
        cells.push({ key: `${unit}-B-empty`, kind: 'empty', colspan: 1, rowspan: 1 })
      }
    }

    rows.push({ unit, cells })
  }

  return {
    rows,
    warnings: Array.from(new Set(warnings)),
    moreItems,
  }
}

function splitTagNames(raw: unknown) {
  if (Array.isArray(raw)) {
    return raw
      .map((value) => String(value ?? '').trim())
      .filter(Boolean)
  }
  if (raw === null || raw === undefined) return [] as string[]
  return String(raw)
    .split(',')
    .map((value) => value.trim())
    .filter(Boolean)
}

function compareTagNamesByID(left: string, right: string) {
  const leftID = tagIDByName.value.get(left.toLowerCase())
  const rightID = tagIDByName.value.get(right.toLowerCase())
  const leftKnown = Number.isFinite(leftID)
  const rightKnown = Number.isFinite(rightID)

  if (leftKnown && rightKnown && leftID !== rightID) return Number(leftID) - Number(rightID)
  if (leftKnown !== rightKnown) return leftKnown ? -1 : 1
  return left.localeCompare(right, 'zh-CN', { numeric: true, sensitivity: 'base' })
}

function sortTagNamesByID(raw: unknown) {
  return splitTagNames(raw).sort(compareTagNamesByID)
}

function formatSortedTagText(raw: unknown) {
  const names = sortTagNamesByID(raw)
  return names.length > 0 ? names.join(', ') : ''
}

function assignSoftwareTags(raw: unknown) {
  softwareTags.value = sortTagNamesByID(raw)
}

function assignItemTags(raw: unknown) {
  itemTags.value = sortTagNamesByID(raw)
}

function withSelectedRow(rows: Record<string, unknown>[], filtered: Record<string, unknown>[], selectedID: number) {
  if (!Number.isFinite(selectedID) || selectedID <= 0) return filtered
  if (filtered.some((item) => Number(item.id ?? 0) === selectedID)) return filtered
  const selected = rows.find((item) => Number(item.id ?? 0) === selectedID)
  return selected ? [selected, ...filtered] : filtered
}

function filterLookupRows(field: ResourceField, rows: Record<string, unknown>[]) {
  const resourceKey = resource.value?.key ?? ''

  if (resourceKey === 'items' && field.key === 'manufacturerId') {
    const filtered = rows.filter((item) => (getAgentTypeMask(item) & 8) === 8)
    return withSelectedRow(rows, filtered, Number(form.manufacturerId ?? 0))
  }
  if (resourceKey === 'software' && field.key === 'manufacturerId') {
    const filtered = rows.filter((item) => (getAgentTypeMask(item) & 2) === 2)
    return withSelectedRow(rows, filtered, Number(form.manufacturerId ?? 0))
  }
  if (resourceKey === 'invoices' && field.key === 'vendorId') {
    const filtered = rows.filter((item) => (getAgentTypeMask(item) & 4) === 4)
    return withSelectedRow(rows, filtered, Number(form.vendorId ?? 0))
  }
  if (resourceKey === 'invoices' && field.key === 'buyerId') {
    const filtered = rows.filter((item) => (getAgentTypeMask(item) & 1) === 1)
    return withSelectedRow(rows, filtered, Number(form.buyerId ?? 0))
  }
  if (resourceKey === 'contracts' && field.key === 'contractorId') {
    const filtered = rows.filter((item) => (getAgentTypeMask(item) & 16) === 16)
    return withSelectedRow(rows, filtered, Number(form.contractorId ?? 0))
  }
  if (resourceKey === 'contracts' && field.key === 'parentId') {
    const currentID = Number(selectedId.value ?? 0)
    return rows.filter((item) => {
      const id = Number(item.id ?? 0)
      return id > 0 && id !== currentID
    })
  }
  if (resourceKey === 'contracts' && field.key === 'subTypeId') {
    const typeID = Number(form.typeId ?? 0)
    if (!Number.isFinite(typeID) || typeID <= 0) return []
    return rows.filter((item) => Number(item.contypeid ?? 0) === typeID)
  }
  if ((resourceKey === 'items' || resourceKey === 'racks') && field.key === 'locAreaId') {
    const locationID = Number(form.locationId ?? 0)
    if (!Number.isFinite(locationID) || locationID <= 0) return []
    return rows.filter((item) => Number(item.locationid ?? 0) === locationID)
  }
  if (resourceKey === 'items' && field.key === 'rackId') {
    const locationID = Number(form.locationId ?? 0)
    if (!Number.isFinite(locationID) || locationID <= 0) return []
    return rows.filter((item) => Number(item.locationid ?? 0) === locationID)
  }
  if (resourceKey === 'files' && field.key === 'typeId') {
    const selectedTypeID = Number(form.typeId ?? 0)
    return rows.filter((item) => {
      const isInvoiceType = isInvoiceFileTypeRow(item)
      if (selectedTypeID === invoiceFileTypeID.value) return isInvoiceType
      return !isInvoiceType
    })
  }

  return rows
}

function getFieldOptions(field: ResourceField) {
  if (field.options) return field.options
  if (field.optionsKey) {
    const list = (bootstrap.lookups[field.optionsKey] ?? []) as Record<string, unknown>[]
    const filtered = filterLookupRows(field, list)
    return filtered.map((item) => ({ label: toFieldOptionLabel(field, item), value: toOptionValue(item) }))
  }
  return []
}

const fieldMap = computed<Record<string, ResourceField>>(() => {
  const map: Record<string, ResourceField> = {}
  if (!resource.value) return map
  for (const field of resource.value.fields) map[field.key] = field
  return map
})

function getFieldConfig(key: string) {
  return fieldMap.value[key]
}

function getOptionsByFieldKey(key: string) {
  const field = getFieldConfig(key)
  if (!field) return []
  return getFieldOptions(field)
}

function keepSelectValueInOptions(fieldKey: string) {
  const current = String(form[fieldKey] ?? '').trim()
  if (!current) return
  const options = getOptionsByFieldKey(fieldKey)
  const valid = options.some((opt) => String(opt.value) === current)
  if (!valid) form[fieldKey] = ''
}

function syncDependentSelections() {
  const resourceKey = resource.value?.key ?? ''
  if (resourceKey === 'contracts') {
    keepSelectValueInOptions('subTypeId')
    return
  }
  if (resourceKey === 'items') {
    keepSelectValueInOptions('locAreaId')
    keepSelectValueInOptions('rackId')
    return
  }
  if (resourceKey === 'racks') {
    keepSelectValueInOptions('locAreaId')
    return
  }
  if (resourceKey === 'files') {
    keepSelectValueInOptions('typeId')
  }
}

function getMultiValues(key: string) {
  const value = form[key]
  return Array.isArray(value) ? value : []
}

function setSelectedFileLinks(fileIDs: number[]) {
  form.fileLinks = Array.from(new Set(fileIDs.filter((id) => Number.isFinite(id) && id > 0)))
    .sort((left, right) => naturalCompare(left, right))
    .map((id) => String(id))
}

function toLocalDateText(raw: unknown) {
  const ts = Number(raw ?? 0)
  if (!ts) return '-'
  const date = new Date(ts * 1000)
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

function toLocalDateTimeMinuteText(raw: unknown) {
  const ts = Number(raw ?? 0)
  if (!ts) return '-'
  const date = new Date(ts * 1000)
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  const hh = String(date.getHours()).padStart(2, '0')
  const mm = String(date.getMinutes()).padStart(2, '0')
  return `${y}-${m}-${d} ${hh}:${mm}`
}

function toLocalDateEightAMTimestamp(raw: unknown) {
  const normalized = normalizeDateInput(raw)
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(normalized)
  if (!match) return 0
  const year = Number(match[1])
  const month = Number(match[2])
  const day = Number(match[3])
  if (!year || !month || !day) return 0
  return Math.floor(new Date(year, month - 1, day, 8, 0, 0, 0).getTime() / 1000)
}

function resetContractEventForm() {
  editingContractEventId.value = null
  contractEventForm.siblingId = ''
  contractEventForm.startDate = ''
  contractEventForm.endDate = ''
  contractEventForm.description = ''
}

function editContractEvent(row: GenericRow) {
  editingContractEventId.value = Number(row.id ?? 0) || null
  contractEventForm.siblingId = Number(row.siblingid ?? 0) > 0 ? String(Number(row.siblingid)) : ''
  contractEventForm.startDate = normalizeDateInput(row.startdate)
  contractEventForm.endDate = normalizeDateInput(row.enddate)
  contractEventForm.description = String(row.description ?? '')
}

async function loadContractEvents() {
  if (!isContractResource.value) return
  const contractID = Number(selectedId.value ?? 0)
  if (!contractID) {
    contractEvents.value = []
    return
  }
  contractEventsLoading.value = true
  error.value = ''
  try {
    const { data } = await api.get(`/contracts/${contractID}/events`)
    contractEvents.value = Array.isArray(data) ? data : []
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '合同事件加载失败'
  } finally {
    contractEventsLoading.value = false
  }
}

async function saveContractEvent() {
  if (!canWrite.value || !isContractResource.value) return
  const contractID = Number(selectedId.value ?? 0)
  if (!contractID) return
  if (!contractEventForm.startDate || !contractEventForm.endDate) {
    const message = '请填写开始日期和结束日期'
    error.value = message
    noticeStore.error(message)
    return
  }
  if (isDateRangeInvalid(contractEventForm.startDate, contractEventForm.endDate)) {
    const message = '合同事件中，结束日期不能早于开始日期'
    error.value = message
    noticeStore.error(message)
    return
  }
  contractEventSaving.value = true
  error.value = ''
  try {
    const payload = {
      siblingId: contractEventForm.siblingId ? Number(contractEventForm.siblingId) : 0,
      startDate: contractEventForm.startDate,
      endDate: contractEventForm.endDate,
      description: contractEventForm.description,
    }
    if (editingContractEventId.value) {
      await api.put(`/contracts/${contractID}/events/${editingContractEventId.value}`, payload)
    } else {
      await api.post(`/contracts/${contractID}/events`, payload)
    }
    resetContractEventForm()
    await loadContractEvents()
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '合同事件保存失败'
  } finally {
    contractEventSaving.value = false
  }
}

async function removeContractEvent(row: GenericRow) {
  if (!canWrite.value || !isContractResource.value) return
  const contractID = Number(selectedId.value ?? 0)
  const eventID = Number(row.id ?? 0)
  if (!contractID || !eventID) return
  openDeleteConfirm({ kind: 'contractEvent', contractID, eventID })
}

function resetLocationAreaEditor() {
  editingLocationAreaId.value = null
  locationAreaName.value = ''
}

function isPendingLocationArea(row: GenericRow | null | undefined) {
  return Boolean(row?.__pending)
}

function findLocationAreaRowById(areaID: number) {
  return locationAreas.value.find((row) => Number(row.id ?? 0) === areaID) ?? null
}

function upsertPendingLocationArea(areaName: string) {
  const editingID = Number(editingLocationAreaId.value ?? 0)
  if (editingID > 0) {
    const index = locationAreas.value.findIndex((row) => Number(row.id ?? 0) === editingID && isPendingLocationArea(row))
    if (index >= 0) {
      locationAreas.value[index] = {
        ...locationAreas.value[index],
        areaname: areaName,
      }
      resetLocationAreaEditor()
      return
    }
  }
  locationAreas.value = [
    ...locationAreas.value,
    {
      id: nextPendingLocationAreaId.value++,
      areaname: areaName,
      __pending: true,
    },
  ]
  resetLocationAreaEditor()
}

async function syncPendingLocationAreas(locationID: number) {
  const pendingRows = locationAreas.value
    .filter((row) => isPendingLocationArea(row))
    .sort((a, b) => naturalCompare(a.id, b.id))

  if (pendingRows.length === 0) return

  for (const row of pendingRows) {
    const areaName = String(row.areaname ?? '').trim()
    if (!areaName) continue
    const { data } = await api.post(`/locations/${locationID}/areas`, { areaName })
    const tempID = Number(row.id ?? 0)
    const createdID = Number((data as GenericRow | null)?.id ?? 0)
    locationAreas.value = locationAreas.value.map((entry) => {
      if (Number(entry.id ?? 0) !== tempID) return entry
      return {
        id: createdID > 0 ? createdID : tempID,
        areaname: areaName,
        locationid: locationID,
      }
    })
  }

  nextPendingLocationAreaId.value = 1
  resetLocationAreaEditor()
  await loadLocationAreas()
}

function editLocationArea(row: GenericRow) {
  editingLocationAreaId.value = Number(row.id ?? 0) || null
  locationAreaName.value = String(row.areaname ?? '')
}

async function loadLocationAreas() {
  if (!isLocationResource.value) return
  const locationID = Number(selectedId.value ?? 0)
  if (!locationID) {
    locationAreas.value = []
    return
  }
  locationAreasLoading.value = true
  error.value = ''
  try {
    const { data } = await api.get(`/locations/${locationID}/areas`)
    locationAreas.value = Array.isArray(data) ? data : []
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '地点区域加载失败'
  } finally {
    locationAreasLoading.value = false
  }
}

function revokeLocationFloorplanPreviewURL() {
  if (!locationFloorplanPreviewURL.value) return
  window.URL.revokeObjectURL(locationFloorplanPreviewURL.value)
  locationFloorplanPreviewURL.value = ''
}

async function fetchLocationFloorplanBlobByID(locationID: number, fileName: string) {
  if (!locationID || !fileName) return null
  const response = await api.get(`/locations/${locationID}/floorplan`, {
    params: { v: fileName },
    responseType: 'blob',
  })
  const blob = response.data instanceof Blob ? response.data : new Blob([response.data])
  if (!blob || blob.size <= 0) return null
  return blob
}

async function fetchLocationFloorplanBlob() {
  const locationID = Number(selectedId.value ?? 0)
  const fileName = String(recordDetail.value?.floorplanfn ?? '').trim()
  return fetchLocationFloorplanBlobByID(locationID, fileName)
}

function openBlobInNewWindow(blob: Blob) {
  const blobURL = window.URL.createObjectURL(blob)
  const win = window.open(blobURL, '_blank', 'noopener')
  if (!win) {
    const link = document.createElement('a')
    link.href = blobURL
    link.target = '_blank'
    link.rel = 'noopener'
    document.body.appendChild(link)
    link.click()
    link.remove()
  }
  window.setTimeout(() => {
    window.URL.revokeObjectURL(blobURL)
  }, 60_000)
}

function openLocationFloorplanPicker() {
  if (!canWrite.value) return
  locationFloorplanInput.value?.click()
}

function getLocationFloorplanFileValidationError(file: File | null) {
  if (!(file instanceof File)) return ''
  const fileName = String(file.name ?? '').trim()
  const ext = fileName ? fileName.slice(Math.max(0, fileName.lastIndexOf('.'))).toLowerCase() : ''
  if (locationFloorplanImageExtensions.has(ext)) return ''
  return '建筑平面图仅支持图片文件扩展名：.jpg、.jpeg、.png、.gif、.bmp、.webp、.svg、.avif'
}

function getInvoiceUploadFileValidationError(file: File | null) {
  if (!(file instanceof File)) return ''
  const fileName = String(file.name ?? '').trim()
  const ext = fileName ? fileName.slice(Math.max(0, fileName.lastIndexOf('.'))).toLowerCase() : ''
  if (invoiceUploadAllowedExtensions.has(ext)) return ''
  return '单据上传仅支持图片或 PDF 扩展名：.jpg、.jpeg、.png、.gif、.bmp、.webp、.svg、.avif、.pdf'
}

function getFileUploadValidationError(file: File | null) {
  if (!isInvoiceTypeFile.value) return ''
  const message = getInvoiceUploadFileValidationError(file)
  if (!message) return ''
  return message.replace('单据上传', '发票类型文件上传')
}

function handleLocationFloorplanChange(event: Event) {
  const input = event.target as HTMLInputElement | null
  const file = input?.files?.[0] ?? null
  const message = getLocationFloorplanFileValidationError(file)
  if (message) {
    form.file = null
    if (input) input.value = ''
    error.value = message
    noticeStore.error(message)
    return
  }
  form.file = file
}

function openFileUploadPicker() {
  if (!canWrite.value) return
  fileUploadInput.value?.click()
}

function openSoftwareUploadPicker() {
  if (!canWrite.value) return
  softwareUploadInput.value?.click()
}

function openItemUploadPicker() {
  if (!canWrite.value) return
  itemUploadInput.value?.click()
}

function openInvoiceUploadPicker() {
  if (!canWrite.value) return
  invoiceUploadInput.value?.click()
}

function openContractUploadPicker() {
  if (!canWrite.value) return
  contractUploadInput.value?.click()
}

function clearLinkedUploadInput(scope: LinkedUploadScope) {
  if (scope === 'software' && softwareUploadInput.value) softwareUploadInput.value.value = ''
  if (scope === 'items' && itemUploadInput.value) itemUploadInput.value.value = ''
  if (scope === 'invoices' && invoiceUploadInput.value) invoiceUploadInput.value.value = ''
  if (scope === 'contracts' && contractUploadInput.value) contractUploadInput.value.value = ''
}

function handleFileUploadChange(event: Event) {
  const input = event.target as HTMLInputElement | null
  const file = input?.files?.[0] ?? null
  const message = getFileUploadValidationError(file)
  if (message) {
    form.file = null
    if (input) input.value = ''
    error.value = message
    noticeStore.error(message)
    return
  }
  form.file = file
}

function downloadSelectedFile() {
  if (!selectedId.value || !recordDetail.value?.fname) return
  void downloadFile({ id: selectedId.value, fname: String(recordDetail.value.fname ?? '') })
}

async function loadLocationFloorplanPreview() {
  revokeLocationFloorplanPreviewURL()
  if (!isLocationResource.value || !drawerOpen.value) return
  if (!selectedId.value || !locationFloorplanName.value) return
  locationFloorplanLoading.value = true
  try {
    const blob = await fetchLocationFloorplanBlob()
    if (!blob) return
    locationFloorplanPreviewURL.value = window.URL.createObjectURL(blob)
  } catch {
    locationFloorplanPreviewURL.value = ''
  } finally {
    locationFloorplanLoading.value = false
  }
}

async function openLocationFloorplanInNewWindow() {
  if (!selectedId.value || !locationFloorplanName.value) return
  try {
    const blob = await fetchLocationFloorplanBlob()
    if (!blob) {
      error.value = '平面图文件不存在或无法读取'
      return
    }
    openBlobInNewWindow(blob)
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '平面图打开失败'
  }
}

async function openLocationFloorplanByRow(row: GenericRow) {
  const locationID = Number(row.id ?? 0)
  const fileName = String(row.floorplanfn ?? '').trim()
  if (!locationID || !fileName) return
  try {
    const blob = await fetchLocationFloorplanBlobByID(locationID, fileName)
    if (!blob) {
      error.value = '平面图文件不存在或无法读取'
      return
    }
    openBlobInNewWindow(blob)
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '平面图打开失败'
  }
}

function getLocationFloorplanDisplayName(fileName: unknown) {
  const value = String(fileName ?? '').trim()
  return value || '查看图片'
}

function getFileDisplayName(fileName: unknown) {
  const value = String(fileName ?? '').trim()
  return value || '下载文件'
}

async function saveLocationArea() {
  if (!canWrite.value || !isLocationResource.value) return
  const locationID = Number(selectedId.value ?? 0)
  const areaName = locationAreaName.value.trim()
  if (!areaName) {
    const message = '请填写区域名称'
    error.value = message
    noticeStore.error(message)
    return
  }
  error.value = ''
  const editingRow = editingLocationAreaId.value ? findLocationAreaRowById(editingLocationAreaId.value) : null
  if (!locationID || isPendingLocationArea(editingRow)) {
    upsertPendingLocationArea(areaName)
    return
  }
  locationAreaSaving.value = true
  const shouldRecordRecentHistory = !editingLocationAreaId.value
  try {
    const payload = { areaName }
    if (editingLocationAreaId.value) {
      await api.put(`/locations/${locationID}/areas/${editingLocationAreaId.value}`, payload)
    } else {
      await api.post(`/locations/${locationID}/areas`, payload)
    }
    if (shouldRecordRecentHistory && locationID > 0) {
      void recordRecentViewHistory(locationID)
    }
    resetLocationAreaEditor()
    await loadLocationAreas()
    await loadRows()
    await refreshBootstrapIfNeeded()
    await ensureRelationCache(['items', 'software', 'invoices', 'contracts', 'files'])
    broadcastEditorDataDirty({ bootstrapKeys: ['locareas'] })
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '地点区域保存失败'
  } finally {
    locationAreaSaving.value = false
  }
}

async function removeLocationArea(row: GenericRow) {
  if (!canWrite.value || !isLocationResource.value) return
  if (!selectedId.value || isPendingLocationArea(row)) {
    const areaID = Number(row.id ?? 0)
    locationAreas.value = locationAreas.value.filter((entry) => Number(entry.id ?? 0) !== areaID)
    if (editingLocationAreaId.value === areaID) {
      resetLocationAreaEditor()
    }
    return
  }
  const locationID = Number(selectedId.value ?? 0)
  const areaID = Number(row.id ?? 0)
  if (!locationID || !areaID) return
  openDeleteConfirm({
    kind: 'locationArea',
    locationID,
    areaID,
    areaName: String(row.areaname ?? areaID),
  })
}

function parseAgentContacts(raw: unknown): AgentContactRow[] {
  const text = String(raw ?? '').trim()
  if (!text) return [{ name: '', phones: '', email: '', role: '', comments: '' }]
  const rows = text
    .split('|')
    .map((row) => row.trim())
    .filter(Boolean)
    .map((row) => {
      const [name = '', phones = '', email = '', role = '', comments = ''] = row.split('#')
      return { name, phones, email, role, comments }
    })
  return rows.length > 0 ? rows : [{ name: '', phones: '', email: '', role: '', comments: '' }]
}

function parseAgentContactsForTable(raw: unknown): AgentContactRow[] {
  const text = String(raw ?? '').trim()
  if (!text || text === '####') return []
  return text
    .split('|')
    .map((row) => row.trim())
    .filter(Boolean)
    .map((row) => {
      const [name = '', phones = '', email = '', role = '', comments = ''] = row.split('#')
      return { name, phones, email, role, comments }
    })
    .filter((row) => row.name || row.phones || row.email || row.role || row.comments)
}

function formatAgentContactSummaryLine(contact: AgentContactRow) {
  return [contact.name, contact.phones, contact.email, contact.role, contact.comments]
    .map((value) => String(value ?? '').trim())
    .filter(Boolean)
    .join(', ')
}

function getAgentContactSummaryLines(raw: unknown) {
  return parseAgentContactsForTable(raw)
    .map((row) => formatAgentContactSummaryLine(row))
    .filter(Boolean)
}

function parseAgentURLs(raw: unknown): AgentURLRow[] {
  const text = String(raw ?? '').trim()
  if (!text) return [{ description: '', url: '' }]
  const rows = text
    .split('|')
    .map((row) => row.trim())
    .filter(Boolean)
    .map((row) => {
      const [description = '', url = ''] = row.split('#')
      return { description, url }
    })
  return rows.length > 0 ? rows : [{ description: '', url: '' }]
}

function createDefaultRenewalRow(): ContractRenewalRow {
  return {
    endDateBefore: '',
    endDateAfter: '',
    effectiveDate: '',
    notes: '',
    enteredDate: '',
    enteredBy: '',
  }
}

function parseContractRenewals(raw: unknown): ContractRenewalRow[] {
  const text = String(raw ?? '').trim()
  if (!text) return [createDefaultRenewalRow()]
  const rows = text
    .split('|')
    .map((row) => row.trim())
    .filter(Boolean)
    .map((row) => {
      const [endDateBefore = '', endDateAfter = '', effectiveDate = '', notes = '', enteredDate = '', enteredBy = ''] = row.split('#')
      return {
        endDateBefore: endDateBefore.trim(),
        endDateAfter: endDateAfter.trim(),
        effectiveDate: effectiveDate.trim(),
        notes: notes.trim(),
        enteredDate: enteredDate.trim(),
        enteredBy: enteredBy.trim(),
      }
    })
    .filter((row) => row.endDateBefore || row.endDateAfter || row.effectiveDate || row.notes || row.enteredDate || row.enteredBy)
  return rows.length > 0 ? rows : [createDefaultRenewalRow()]
}

function serializeContractRenewals(rows: ContractRenewalRow[]) {
  const sanitizedRows = rows
    .map((row) => ({
      endDateBefore: sanitizePipeHashText(row.endDateBefore),
      endDateAfter: sanitizePipeHashText(row.endDateAfter),
      effectiveDate: sanitizePipeHashText(row.effectiveDate),
      notes: sanitizePipeHashText(row.notes),
      enteredDate: sanitizePipeHashText(row.enteredDate),
      enteredBy: sanitizePipeHashText(row.enteredBy),
    }))
    .filter(
      (row) => row.endDateBefore || row.endDateAfter || row.effectiveDate || row.notes || row.enteredDate || row.enteredBy,
    )
  if (sanitizedRows.length === 0) return ''
  return sanitizedRows
    .map((row) =>
      [row.endDateBefore, row.endDateAfter, row.effectiveDate, row.notes, row.enteredDate, row.enteredBy].join('#'),
    )
    .join('|')
}

function sanitizePipeHashText(raw: unknown) {
  return String(raw ?? '')
    .replace(/\|/g, ' ')
    .replace(/#/g, ' ')
    .trim()
}

function ensureAgentRows() {
  if (agentContacts.value.length === 0) agentContacts.value.push({ name: '', phones: '', email: '', role: '', comments: '' })
  if (agentURLs.value.length === 0) agentURLs.value.push({ description: '', url: '' })
}

function resetNonItemEditorState() {
  activeSoftwareTab.value = 'softwareData'
  activeSoftwareOverviewTab.value = 'items'
  activeInvoiceTab.value = 'invoiceData'
  activeInvoiceOverviewTab.value = 'items'
  activeContractTab.value = 'contractData'
  activeContractOverviewTab.value = 'items'
  activeFileTab.value = 'fileData'
  activeFileOverviewTab.value = 'items'
  activeAgentOverviewTab.value = 'items'
  activeLocationOverviewTab.value = 'items'
  softwareTags.value = []
  locationAreas.value = []
  locationAreasLoading.value = false
  locationAreaSaving.value = false
  editingLocationAreaId.value = null
  locationAreaName.value = ''
  nextPendingLocationAreaId.value = 1
  contractEvents.value = []
  contractEventsLoading.value = false
  contractEventSaving.value = false
  editingContractEventId.value = null
  contractEventForm.siblingId = ''
  contractEventForm.startDate = ''
  contractEventForm.endDate = ''
  contractEventForm.description = ''
  locationFloorplanLoading.value = false
  revokeLocationFloorplanPreviewURL()
  agentContacts.value = [{ name: '', phones: '', email: '', role: '', comments: '' }]
  agentURLs.value = [{ description: '', url: '' }]
  agentContactInfo.value = ''
  fileLinkFilter.value = ''
  softwareTagEditorOpen.value = false
  softwareTagInput.value = ''
  softwareTagSaving.value = false
  softwareTagMessage.value = ''
  softwareRelationSortState.items.key = 'id'
  softwareRelationSortState.items.direction = 'asc'
  softwareRelationSortState.invoices.key = 'id'
  softwareRelationSortState.invoices.direction = 'asc'
  softwareRelationSortState.contracts.key = 'id'
  softwareRelationSortState.contracts.direction = 'asc'
  softwareRelationSortState.files.key = 'id'
  softwareRelationSortState.files.direction = 'asc'
  nonSoftwareRelationSortState.itemItems.key = 'id'
  nonSoftwareRelationSortState.itemItems.direction = 'asc'
  nonSoftwareRelationSortState.itemInvoices.key = 'id'
  nonSoftwareRelationSortState.itemInvoices.direction = 'asc'
  nonSoftwareRelationSortState.itemSoftware.key = 'id'
  nonSoftwareRelationSortState.itemSoftware.direction = 'asc'
  nonSoftwareRelationSortState.itemContracts.key = 'id'
  nonSoftwareRelationSortState.itemContracts.direction = 'asc'
  nonSoftwareRelationSortState.invoiceItems.key = 'id'
  nonSoftwareRelationSortState.invoiceItems.direction = 'asc'
  nonSoftwareRelationSortState.invoiceSoftware.key = 'id'
  nonSoftwareRelationSortState.invoiceSoftware.direction = 'asc'
  nonSoftwareRelationSortState.invoiceContracts.key = 'id'
  nonSoftwareRelationSortState.invoiceContracts.direction = 'asc'
  nonSoftwareRelationSortState.contractItems.key = 'id'
  nonSoftwareRelationSortState.contractItems.direction = 'asc'
  nonSoftwareRelationSortState.contractSoftware.key = 'id'
  nonSoftwareRelationSortState.contractSoftware.direction = 'asc'
  nonSoftwareRelationSortState.contractInvoices.key = 'id'
  nonSoftwareRelationSortState.contractInvoices.direction = 'asc'
  nonSoftwareRelationSortState.fileItems.key = 'id'
  nonSoftwareRelationSortState.fileItems.direction = 'asc'
  nonSoftwareRelationSortState.fileSoftware.key = 'id'
  nonSoftwareRelationSortState.fileSoftware.direction = 'asc'
  nonSoftwareRelationSortState.fileContracts.key = 'id'
  nonSoftwareRelationSortState.fileContracts.direction = 'asc'
  softwareUploadForm.title = ''
  softwareUploadForm.typeId = ''
  softwareUploadForm.date = ''
  softwareUploadForm.file = null
  clearLinkedUploadInput('software')
  softwareUploading.value = false
  itemUploadForm.title = ''
  itemUploadForm.typeId = ''
  itemUploadForm.date = ''
  itemUploadForm.file = null
  clearLinkedUploadInput('items')
  itemUploading.value = false
  invoiceUploadForm.title = ''
  invoiceUploadForm.date = ''
  invoiceUploadForm.file = null
  clearLinkedUploadInput('invoices')
  invoiceUploading.value = false
  contractUploadForm.title = ''
  contractUploadForm.typeId = ''
  contractUploadForm.date = ''
  contractUploadForm.file = null
  clearLinkedUploadInput('contracts')
  contractUploading.value = false
  contractRenewals.value = [createDefaultRenewalRow()]
}

function normalizeRelationCacheKeys(raw: unknown): RelationCacheKey[] {
  const source = Array.isArray(raw) ? raw : []
  const valid = new Set<RelationCacheKey>(['items', 'software', 'invoices', 'contracts', 'files'])
  return source
    .map((value) => String(value ?? '').trim())
    .filter((value): value is RelationCacheKey => valid.has(value as RelationCacheKey))
}

function normalizeBootstrapLookupKeys(raw: unknown): string[] {
  const source = Array.isArray(raw) ? raw : []
  return source
    .map((value) => String(value ?? '').trim())
    .filter(Boolean)
}

function applyEditorDataDirtyPayload(payload: EditorDataDirtyPayload) {
  for (const key of normalizeRelationCacheKeys(payload.relationKeys)) dirtyRelationCacheKeys.add(key)
  for (const key of normalizeBootstrapLookupKeys(payload.bootstrapKeys)) dirtyBootstrapLookupKeys.add(key)
}

function handleEditorDataDirtyMessage(raw: unknown) {
  if (!raw || typeof raw !== 'object') return
  const payload = raw as Partial<EditorDataDirtyPayload>
  if (String(payload.sourceId ?? '') === resourceEditorSyncSourceId) return
  applyEditorDataDirtyPayload({
    sourceId: String(payload.sourceId ?? ''),
    timestamp: Number(payload.timestamp ?? Date.now()),
    relationKeys: normalizeRelationCacheKeys(payload.relationKeys),
    bootstrapKeys: normalizeBootstrapLookupKeys(payload.bootstrapKeys),
  })
}

function broadcastEditorDataDirty(payload: { relationKeys?: RelationCacheKey[]; bootstrapKeys?: string[] }) {
  const relationKeys = normalizeRelationCacheKeys(payload.relationKeys)
  const bootstrapKeys = normalizeBootstrapLookupKeys(payload.bootstrapKeys)
  if (relationKeys.length === 0 && bootstrapKeys.length === 0) return
  const message: EditorDataDirtyPayload = {
    sourceId: resourceEditorSyncSourceId,
    timestamp: Date.now(),
    relationKeys,
    bootstrapKeys,
  }
  try {
    resourceEditorSyncChannel?.postMessage(message)
  } catch {}
  try {
    window.localStorage.setItem(resourceEditorSyncStorageKey, JSON.stringify(message))
  } catch {}
}

function broadcastResourceMutation(resourceKey: string, extra?: { relationKeys?: RelationCacheKey[]; bootstrapKeys?: string[] }) {
  const base = resourceMutationImpact[resourceKey] ?? { relationKeys: [], bootstrapKeys: [] }
  const relationKeys = [...base.relationKeys, ...(extra?.relationKeys ?? [])]
  const bootstrapKeys = [...base.bootstrapKeys, ...(extra?.bootstrapKeys ?? [])]
  broadcastEditorDataDirty({ relationKeys, bootstrapKeys })
}

async function refreshRelationCacheEntry(key: RelationCacheKey) {
  const existingPromise = relationCacheRefreshPromises[key]
  if (existingPromise) return existingPromise
  const refreshPromise = (async () => {
    relationCacheLoading[key] = true
    try {
      const params: Record<string, unknown> = {}
      if (pageableResourceKeys.has(key)) {
        params.limit = -1
        params.offset = 0
      }
      const { data } = await api.get(`/${key}`, { params })
      relationCache[key] = Array.isArray(data) ? data : []
      return true
    } catch {
      return false
    } finally {
      relationCacheLoading[key] = false
      delete relationCacheRefreshPromises[key]
    }
  })()
  relationCacheRefreshPromises[key] = refreshPromise
  return refreshPromise
}

async function refreshRelationCache(keys: RelationCacheKey[]) {
  const uniqueKeys = Array.from(new Set(keys))
  const results = await Promise.all(uniqueKeys.map((key) => refreshRelationCacheEntry(key)))
  return uniqueKeys.filter((_, index) => results[index])
}

async function fetchBootstrapSnapshot() {
  if (bootstrapSnapshotPromise) return bootstrapSnapshotPromise
  bootstrapSnapshotPromise = api
    .get('/bootstrap')
    .then(({ data }) => ((data && typeof data === 'object' ? (data as Record<string, Record<string, unknown>[]>) : null)))
    .catch(() => null)
    .finally(() => {
      bootstrapSnapshotPromise = null
    })
  return bootstrapSnapshotPromise
}

async function refreshBootstrapLookups(keys: string[]) {
  const uniqueKeys = Array.from(new Set(keys.filter(Boolean)))
  if (uniqueKeys.length === 0) return false
  const snapshot = await fetchBootstrapSnapshot()
  if (!snapshot) return false
  if (!bootstrap.loaded) {
    bootstrap.lookups = snapshot
    bootstrap.loaded = true
    return true
  }
  for (const key of uniqueKeys) {
    if (!Object.prototype.hasOwnProperty.call(snapshot, key)) continue
    bootstrap.lookups[key] = snapshot[key] as never
  }
  return true
}

async function refreshDirtyEditorDependencies(resourceKey: string) {
  const config = resourceEditorDependencies[resourceKey]
  if (!config) return
  const relationKeys = config.relationKeys.filter((key) => dirtyRelationCacheKeys.has(key))
  const bootstrapKeys = config.bootstrapKeys.filter((key) => dirtyBootstrapLookupKeys.has(key))
  if (relationKeys.length === 0 && bootstrapKeys.length === 0) return

  const [refreshedRelationKeys, bootstrapUpdated] = await Promise.all([
    relationKeys.length > 0 ? refreshRelationCache(relationKeys) : Promise.resolve([] as RelationCacheKey[]),
    bootstrapKeys.length > 0 ? refreshBootstrapLookups(bootstrapKeys) : Promise.resolve(false),
  ])

  for (const key of refreshedRelationKeys) dirtyRelationCacheKeys.delete(key)
  if (bootstrapUpdated) {
    for (const key of bootstrapKeys) dirtyBootstrapLookupKeys.delete(key)
  }
}

function onEditorDataDirtyStorage(event: StorageEvent) {
  if (event.key !== resourceEditorSyncStorageKey || !event.newValue) return
  try {
    handleEditorDataDirtyMessage(JSON.parse(event.newValue))
  } catch {}
}

function setupEditorDataDirtySync() {
  if (typeof window === 'undefined') return
  if ('BroadcastChannel' in window) {
    resourceEditorSyncChannel = new BroadcastChannel(resourceEditorSyncChannelName)
    resourceEditorSyncChannel.addEventListener('message', (event) => {
      handleEditorDataDirtyMessage(event.data)
    })
  }
  window.addEventListener('storage', onEditorDataDirtyStorage)
}

function teardownEditorDataDirtySync() {
  if (typeof window === 'undefined') return
  window.removeEventListener('storage', onEditorDataDirtyStorage)
  resourceEditorSyncChannel?.close()
  resourceEditorSyncChannel = null
}

async function ensureRelationCache(keys: RelationCacheKey[]) {
  for (const key of keys) {
    if (relationCache[key].length > 0 || relationCacheLoading[key]) continue
    relationCacheLoading[key] = true
    try {
      const endpoint = `/${key}`
      const params: Record<string, unknown> = {}
      if (pageableResourceKeys.has(key)) {
        params.limit = -1
        params.offset = 0
      }
      const { data } = await api.get(endpoint, { params })
      relationCache[key] = Array.isArray(data) ? data : []
    } catch {
      relationCache[key] = []
    } finally {
      relationCacheLoading[key] = false
    }
  }
}

function getRelationLookupText(type: 'items' | 'software' | 'invoices' | 'contracts', id: number) {
  if (type === 'items') {
    const row = buildItemRelationSource().find((x) => x.id === id)
    return row ? formatOverviewItemText(row) : `- - [-, ID:${id}]`
  }
  if (type === 'software') {
    const row = buildSoftwareRelationSource().find((x) => x.id === id)
    return row ? formatOverviewSoftwareText(row) : `- - [ID:${id}]`
  }
  if (type === 'invoices') {
    const row = buildInvoiceRelationSource().find((x) => x.id === id)
    return row ? formatOverviewInvoiceText(row) : `(-) - - [ID:${id}]`
  }
  const row = buildContractRelationSource().find((x) => x.id === id)
  return row ? formatOverviewContractText(row) : `(- -) - -- [ID:${id}]`
}

function getOverviewJumpTip(resourceKey: OverviewResourceKey, id: number) {
  if (resourceKey === 'items') return `在新窗口编辑硬件 ${id}`
  if (resourceKey === 'software') return `在新窗口编辑软件 ${id}`
  if (resourceKey === 'invoices') return `在新窗口编辑单据 ${id}`
  if (resourceKey === 'contracts') return `在新窗口编辑合同 ${id}`
  return `在新窗口编辑机架 ${id}`
}

function toOverviewRows<T extends { id: number }>(
  source: T[],
  resourceKey: OverviewResourceKey,
  render: (row: T) => string,
): OverviewRow[] {
  return [...source]
    .filter((row) => Number(row.id) > 0)
    .sort((a, b) => naturalCompare(a.id, b.id))
    .map((row, index) => ({
      id: Number(row.id),
      index: index + 1,
      text: render(row),
      resourceKey,
      tip: getOverviewJumpTip(resourceKey, Number(row.id)),
    }))
}

function addAgentContactRow() {
  agentContacts.value.push({ name: '', phones: '', email: '', role: '', comments: '' })
}

function removeAgentContactRow(index: number) {
  if (agentContacts.value.length <= 1) return
  agentContacts.value.splice(index, 1)
}

function addAgentURLRow() {
  agentURLs.value.push({ description: '', url: '' })
}

function removeAgentURLRow(index: number) {
  if (agentURLs.value.length <= 1) return
  agentURLs.value.splice(index, 1)
}

function initializeItemDefaults() {
  if (!isItemResource.value) return
  if (form.status === '' || form.status === null || form.status === undefined) {
    const statusOptions = getOptionsByFieldKey('status')
    if (statusOptions.length > 0) form.status = String(statusOptions[0]?.value ?? '')
  }
  if (form.rackPosDepth === '' || form.rackPosDepth === null || form.rackPosDepth === undefined) {
    form.rackPosDepth = '4'
  }
  if (form.ports === '' || form.ports === null || form.ports === undefined) {
    form.ports = '0'
  }
}

function initializeSoftwareDefaults() {
  if (!isSoftwareResource.value) return
  if (form.licenseQty === '' || form.licenseQty === null || form.licenseQty === undefined) {
    form.licenseQty = '1'
  }
  if (form.licenseType === '' || form.licenseType === null || form.licenseType === undefined) {
    form.licenseType = '0'
  }
}

function initializeRackDefaults() {
  if (!isRackResource.value) return
  if (form.revNums === '' || form.revNums === null || form.revNums === undefined) {
    form.revNums = '0'
  }
}

function compactJoin(parts: unknown[], separator = ' ') {
  return parts
    .map((part) => String(part ?? '').trim())
    .filter(Boolean)
    .join(separator)
}

function formatOverviewItemText(row: ItemRelationRow) {
  const name = compactJoin([row.manufacturer || '-', row.model || '-'])
  return `${name} [${row.itemType || '-'}, ID:${row.id}]`
}

function formatOverviewSoftwareText(row: SoftwareRelationRow) {
  const name = compactJoin([row.manufacturer || '-', row.title || '-', row.version || ''])
  return `${name} [ID:${row.id}]`
}

function formatOverviewInvoiceText(row: InvoiceRelationRow) {
  return `(${row.number || '-'}) - ${row.date || '-'} [ID:${row.id}]`
}

function formatOverviewContractText(row: ContractRelationRow) {
  const contractLabel = compactJoin([row.title || '-', row.number || '-'])
  return `(${contractLabel}) - ${row.startDate || '-'}-${row.currentEndDate || '-'} [ID:${row.id}]`
}

function getStatusColorByText(statusText: unknown) {
  const target = String(statusText ?? '').trim()
  if (!target) return ''
  const rows = (bootstrap.lookups.statustypes ?? []) as Record<string, unknown>[]
  const match = rows.find((row) => String(row.statusdesc ?? '').trim() === target)
  const color = String(match?.color ?? '').trim()
  return /^#([\da-f]{3}|[\da-f]{6})$/i.test(color) ? color : ''
}

function getReadableTextColor(backgroundColor: string) {
  const raw = backgroundColor.trim().replace('#', '')
  if (!/^[\da-f]{3}$|^[\da-f]{6}$/i.test(raw)) return '#12324b'
  const hex = raw.length === 3 ? raw.split('').map((part) => `${part}${part}`).join('') : raw
  const red = Number.parseInt(hex.slice(0, 2), 16)
  const green = Number.parseInt(hex.slice(2, 4), 16)
  const blue = Number.parseInt(hex.slice(4, 6), 16)
  const luminance = (red * 299 + green * 587 + blue * 114) / 1000
  return luminance >= 160 ? '#173651' : '#ffffff'
}

function getItemIDBadgeStyle(row: GenericRow) {
  const color = getStatusColorByText(row.status)
  if (!color) return {}
  return {
    backgroundColor: color,
    borderColor: color,
    color: getReadableTextColor(color),
  }
}

function isInvoiceLinkedFile(row: Pick<FileRelationRow, 'typeID' | 'typeDesc'> | null | undefined) {
  if (!row) return false
  return invoiceFileTypeID.value > 0
    ? Number(row.typeID ?? 0) === invoiceFileTypeID.value
    : /invoice|发票/i.test(String(row.typeDesc ?? ''))
}

const managedLinkedFileRows = computed<ManagedLinkedFileRow[]>(() => {
  const fileByID = new Map<number, FileRelationRow>()
  for (const row of buildFileRelationSource()) fileByID.set(row.id, row)

  const merged = new Map<number, ManagedLinkedFileRow>()
  const upsertRow = (fileID: number, source: FileRelationRow, options?: { direct?: boolean; invoiceID?: number; invoiceNumber?: string }) => {
    const existing = merged.get(fileID)
    if (!existing) {
      merged.set(fileID, {
        ...source,
        fname: source.fileName,
        directlyLinked: Boolean(options?.direct),
        viaInvoiceIDs: options?.invoiceID ? [options.invoiceID] : [],
        viaInvoiceNumbers: options?.invoiceNumber ? [options.invoiceNumber] : [],
      })
      return
    }
    if (options?.direct) existing.directlyLinked = true
    if (options?.invoiceID && !existing.viaInvoiceIDs.includes(options.invoiceID)) {
      existing.viaInvoiceIDs.push(options.invoiceID)
    }
    if (options?.invoiceNumber && !existing.viaInvoiceNumbers.includes(options.invoiceNumber)) {
      existing.viaInvoiceNumbers.push(options.invoiceNumber)
    }
  }

  for (const rawID of getMultiValues('fileLinks')) {
    const fileID = Number(rawID)
    const source = fileByID.get(fileID)
    if (!source) continue
    upsertRow(fileID, source, { direct: true })
  }

  const invoiceByID = new Map<number, InvoiceRelationRow>()
  for (const row of buildInvoiceRelationSource()) invoiceByID.set(row.id, row)
  for (const rawInvoiceID of getMultiValues('invoiceLinks')) {
    const invoiceID = Number(rawInvoiceID)
    const invoice = invoiceByID.get(invoiceID)
    if (!invoice) continue
    const invoiceNumber = String(invoice.number ?? '').trim() || String(invoiceID)
    for (const entry of parseInvoiceFileEntriesValue(invoice.files)) {
      const fileID = Number(entry.id ?? 0)
      if (!fileID) continue
      const source = fileByID.get(fileID) ?? {
        id: fileID,
        typeID: invoiceFileTypeID.value,
        typeDesc: invoiceFileTypeLabel.value,
        title: entry.title || '',
        fileName: entry.fileName || entry.text || '',
        date: '',
      }
      upsertRow(fileID, source, { invoiceID, invoiceNumber })
    }
  }

  return Array.from(merged.values()).sort((a, b) => naturalCompare(a.id, b.id))
})

function canRemoveManagedLinkedFile(row: Pick<ManagedLinkedFileRow, 'typeID' | 'typeDesc' | 'directlyLinked'> | null | undefined) {
  if (!canWrite.value) return false
  if (!row?.directlyLinked) return false
  if (isInvoiceResource.value) return true
  return !isInvoiceLinkedFile(row)
}

const fileAssociationCount = computed(() => {
  if (!isFileResource.value) return 0
  const itemCount = getMultiValues('itemLinks').length
  const softwareCount = getMultiValues('softwareLinks').length
  const invoiceCount = getMultiValues('invoiceLinks').length
  const contractCount = getMultiValues('contractLinks').length
  return itemCount + softwareCount + invoiceCount + contractCount
})

const fileUploadedByText = computed(() => {
  if (!recordDetail.value) return '-'
  const by = String(recordDetail.value.uploader ?? '').trim()
  const at = toLocalDateTimeMinuteText(recordDetail.value.uploaddate)
  if (!by) return at === '-' ? '-' : at
  return at === '-' ? by : `${by} / ${at}`
})

const selectedFileUploadName = computed(() => {
  const file = form.file
  return file instanceof File ? file.name : ''
})

const selectedSoftwareUploadFileName = computed(() => {
  const file = softwareUploadForm.file
  return file instanceof File ? file.name : ''
})

const selectedItemUploadFileName = computed(() => {
  const file = itemUploadForm.file
  return file instanceof File ? file.name : ''
})

const selectedInvoiceUploadFileName = computed(() => {
  const file = invoiceUploadForm.file
  return file instanceof File ? file.name : ''
})

const selectedContractUploadFileName = computed(() => {
  const file = contractUploadForm.file
  return file instanceof File ? file.name : ''
})

function normalizeKeyword(raw: unknown) {
  return String(raw ?? '').trim().toLowerCase()
}

function decodeHtmlEntities(raw: string) {
  return raw
    .replace(/&amp;/gi, '&')
    .replace(/&lt;/gi, '<')
    .replace(/&gt;/gi, '>')
    .replace(/&quot;/gi, '"')
    .replace(/&#39;/gi, "'")
}

function containsKeyword(keyword: string, ...parts: unknown[]) {
  if (!keyword) return true
  const text = parts.map((x) => String(x ?? '').toLowerCase()).join(' ')
  return text.includes(keyword)
}

function isMultiSelected(fieldKey: string, id: number) {
  return getMultiValues(fieldKey).some((value) => Number(value) === id)
}

function buildItemRelationSource() {
  const byID = new Map<number, ItemRelationRow>()
  const users = new Map<number, string>()
  const userRows = (bootstrap.lookups.users ?? []) as Record<string, unknown>[]
  for (const row of userRows) {
    const id = Number(row.id ?? 0)
    if (id > 0) users.set(id, String(row.username ?? ''))
  }

  const fallback = (bootstrap.lookups.items_ref ?? []) as Record<string, unknown>[]
  for (const row of fallback) {
    const id = Number(row.id ?? 0)
    if (id <= 0) continue
    byID.set(id, {
      id,
      itemTypeID: Number(row.itemtypeid ?? row.itemTypeId ?? 0),
      itemType: '',
      manufacturer: '',
      model: String(row.model ?? ''),
      label: String(row.label ?? ''),
      functionText: '',
      dnsName: '',
      username: '',
      statusText: '',
      statusColor: '',
      ipv4: '',
      principal: '',
      sn: '',
      userID: 0,
      locationID: 0,
      rackID: 0,
      manufacturerID: 0,
      uSize: Number(row.usize ?? row.uSize ?? 0),
      rackPosition: Number(row.rackposition ?? row.rackPosition ?? 0),
      rackPosDepth: Number(row.rackposdepth ?? row.rackPosDepth ?? 0),
    })
  }

  const sourceRows =
    relationCache.items.length > 0
      ? relationCache.items
      : resource.value?.key === 'items'
        ? rows.value
        : fallback
  for (const row of sourceRows) {
    const id = Number(row.id ?? 0)
    if (id <= 0) continue
    const sn = [row.sn, row.sn2, row.sn3]
      .map((x) => String(x ?? '').trim())
      .filter(Boolean)
      .join(' ')
    byID.set(id, {
      id,
      itemTypeID: Number(row.itemtypeid ?? row.itemTypeId ?? 0),
      itemType: String(row.itemType ?? ''),
      manufacturer: String(row.manufacturer ?? ''),
      model: String(row.model ?? ''),
      label: String(row.label ?? ''),
      functionText: String(row.function ?? ''),
      dnsName: String(row.dnsname ?? row.dnsName ?? ''),
      username: String(row.username ?? users.get(Number(row.userid ?? row.userId ?? 0)) ?? ''),
      statusText: String(row.status ?? ''),
      statusColor: String(row.statuscolor ?? row.statusColor ?? ''),
      ipv4: String(row.ipv4 ?? ''),
      principal: String(row.principal ?? ''),
      sn,
      userID: Number(row.userid ?? row.userId ?? 0),
      locationID: Number(row.locationid ?? row.locationId ?? 0),
      rackID: Number(row.rackid ?? row.rackId ?? 0),
      manufacturerID: Number(row.manufacturerid ?? row.manufacturerId ?? 0),
      uSize: Number(row.usize ?? row.uSize ?? 0),
      rackPosition: Number(row.rackposition ?? row.rackPosition ?? 0),
      rackPosDepth: Number(row.rackposdepth ?? row.rackPosDepth ?? 0),
    })
  }

  return Array.from(byID.values())
}

function buildInvoiceRelationSource() {
  const fallback = (bootstrap.lookups.invoices_ref ?? []) as Record<string, unknown>[]
  const source = relationCache.invoices.length > 0 ? relationCache.invoices : fallback
  return source
    .map((row) => ({
      id: Number(row.id ?? 0),
      number: String(row.number ?? ''),
      vendor: String(row.vendor ?? ''),
      buyer: String(row.buyer ?? ''),
      description: String(row.description ?? ''),
      files: String(row.files ?? ''),
      date: normalizeDateInput(row.date),
      vendorID: Number(row.vendorid ?? row.vendorId ?? 0),
      buyerID: Number(row.buyerid ?? row.buyerId ?? 0),
    }))
    .filter((row) => row.id > 0)
}

function buildSoftwareRelationSource() {
  const fallback = (bootstrap.lookups.software_ref ?? []) as Record<string, unknown>[]
  const source = relationCache.software.length > 0 ? relationCache.software : fallback
  return source
    .map((row) => ({
      id: Number(row.id ?? 0),
      title: String(row.title ?? row.stitle ?? ''),
      version: String(row.version ?? row.sversion ?? ''),
      manufacturer: String(row.manufacturer ?? ''),
      manufacturerID: Number(row.manufacturerid ?? 0),
    }))
    .filter((row) => row.id > 0)
}

function buildContractRelationSource() {
  const contractors = new Map<number, string>()
  const agentRows = (bootstrap.lookups.agents ?? []) as Record<string, unknown>[]
  for (const row of agentRows) {
    const id = Number(row.id ?? 0)
    if (id > 0) contractors.set(id, String(row.title ?? ''))
  }
  const fallback = (bootstrap.lookups.contracts_ref ?? []) as Record<string, unknown>[]
  const source = relationCache.contracts.length > 0 ? relationCache.contracts : fallback
  return source
    .map((row) => {
      const contractorID = Number(row.contractorid ?? 0)
      return {
        id: Number(row.id ?? 0),
        number: String(row.number ?? ''),
        title: String(row.title ?? ''),
        type: String(row.type ?? ''),
        startDate: normalizeDateInput(row.startdate ?? row.startDate),
        currentEndDate: normalizeDateInput(row.currentenddate ?? row.currentEndDate),
        contractor: String(row.contractor ?? row.agtitle ?? contractors.get(contractorID) ?? ''),
        contractorID,
      }
    })
    .filter((row) => row.id > 0)
}

function buildFileRelationSource() {
  const fallback = (bootstrap.lookups.files_ref ?? []) as Record<string, unknown>[]
  const source = relationCache.files.length > 0 ? relationCache.files : fallback
  return source
    .map((row) => ({
      id: Number(row.id ?? 0),
      typeID: Number(row.type ?? row.typeid ?? 0),
      typeDesc: String(row.typedesc ?? row.typeDesc ?? ''),
      title: String(row.title ?? ''),
      fileName: String(row.fname ?? ''),
      date: normalizeDateInput(row.date),
    }))
    .filter((row) => row.id > 0)
}

function naturalCompare(a: unknown, b: unknown) {
  const numA = Number(a)
  const numB = Number(b)
  if (Number.isFinite(numA) && Number.isFinite(numB) && String(a).trim() !== '' && String(b).trim() !== '') {
    return numA - numB
  }
  const sa = String(a ?? '')
  const sb = String(b ?? '')
  return sa.localeCompare(sb, 'zh-CN', { numeric: true, sensitivity: 'base' })
}

function sortRelationRowsWithLinkedPriority<T>(
  rows: T[],
  selected: Set<number>,
  idOf: (row: T) => number,
  key: string,
  direction: SortDirection,
  pick: (row: T, sortKey: string) => unknown,
) {
  const factor = direction === 'asc' ? 1 : -1
  return [...rows].sort((a, b) => {
    const rankA = selected.has(idOf(a)) ? 0 : 1
    const rankB = selected.has(idOf(b)) ? 0 : 1
    if (rankA !== rankB) return rankA - rankB
    return naturalCompare(pick(a, key), pick(b, key)) * factor
  })
}

function toggleSoftwareRelationSort(scope: 'items' | 'invoices' | 'contracts' | 'files', key: string) {
  const state = softwareRelationSortState[scope]
  if (state.key === key) {
    state.direction = state.direction === 'asc' ? 'desc' : 'asc'
    return
  }
  state.key = key
  state.direction = 'asc'
}

function getSoftwareRelationSortIcon(scope: 'items' | 'invoices' | 'contracts' | 'files', key: string) {
  const state = softwareRelationSortState[scope]
  if (state.key !== key) return '↕'
  return state.direction === 'asc' ? '↑' : '↓'
}

function toggleNonSoftwareRelationSort(scope: NonSoftwareRelationSortScope, key: string) {
  const state = nonSoftwareRelationSortState[scope]
  if (state.key === key) {
    state.direction = state.direction === 'asc' ? 'desc' : 'asc'
    return
  }
  state.key = key
  state.direction = 'asc'
}

function getNonSoftwareRelationSortIcon(scope: NonSoftwareRelationSortScope, key: string) {
  const state = nonSoftwareRelationSortState[scope]
  if (state.key !== key) return '↕'
  return state.direction === 'asc' ? '↑' : '↓'
}

function pickItemRelationSortValue(row: ItemRelationRow, key: string) {
  if (key === 'id') return row.id
  if (key === 'itemType') return row.itemType
  if (key === 'manufacturer') return row.manufacturer
  if (key === 'model') return row.model
  if (key === 'label') return row.label
  if (key === 'functionText') return row.functionText
  if (key === 'dnsName') return row.dnsName
  if (key === 'username') return row.username
  if (key === 'principal') return row.principal
  if (key === 'sn') return row.sn
  return row.id
}

function pickSoftwareRelationSortValue(row: SoftwareRelationRow, key: string) {
  if (key === 'id') return row.id
  if (key === 'manufacturer') return row.manufacturer
  if (key === 'title') return row.title
  if (key === 'version') return row.version
  if (key === 'titleVersion') return `${row.title || ''} ${row.version || ''}`.trim()
  return row.id
}

function pickInvoiceRelationSortValue(row: InvoiceRelationRow, key: string) {
  if (key === 'id') return row.id
  if (key === 'vendor') return row.vendor
  if (key === 'number') return row.number
  if (key === 'description') return `${row.description || ''} ${row.files || ''}`.trim()
  if (key === 'date') return row.date
  return row.id
}

function pickContractRelationSortValue(row: ContractRelationRow, key: string) {
  if (key === 'id') return row.id
  if (key === 'contractor') return row.contractor
  if (key === 'title') return row.title
  return row.id
}

function sortItemRelationRows(rows: ItemRelationRow[], selected: Set<number>, state: { key: string; direction: SortDirection }) {
  return sortRelationRowsWithLinkedPriority(rows, selected, (row) => row.id, state.key, state.direction, pickItemRelationSortValue)
}

function sortSoftwareRelationRows(
  rows: SoftwareRelationRow[],
  selected: Set<number>,
  state: { key: string; direction: SortDirection },
) {
  return sortRelationRowsWithLinkedPriority(
    rows,
    selected,
    (row) => row.id,
    state.key,
    state.direction,
    pickSoftwareRelationSortValue,
  )
}

function sortInvoiceRelationRows(
  rows: InvoiceRelationRow[],
  selected: Set<number>,
  state: { key: string; direction: SortDirection },
) {
  return sortRelationRowsWithLinkedPriority(rows, selected, (row) => row.id, state.key, state.direction, pickInvoiceRelationSortValue)
}

function sortContractRelationRows(
  rows: ContractRelationRow[],
  selected: Set<number>,
  state: { key: string; direction: SortDirection },
) {
  return sortRelationRowsWithLinkedPriority(rows, selected, (row) => row.id, state.key, state.direction, pickContractRelationSortValue)
}

const itemTypeSoftwareSupportMap = computed(() => {
  const map = new Map<number, boolean>()
  const rows = (bootstrap.lookups.itemtypes ?? []) as Record<string, unknown>[]
  for (const row of rows) {
    const id = Number(row.id ?? 0)
    if (id <= 0) continue
    map.set(id, Number(row.hassoftware ?? 0) === 1)
  }
  return map
})

function isSoftwareSupportedItem(row: ItemRelationRow) {
  if (row.itemTypeID <= 0) return true
  return itemTypeSoftwareSupportMap.value.get(row.itemTypeID) ?? true
}


function sortWithLinkedFirst(idA: number, idB: number, selected: Set<number>) {
  const rankA = selected.has(idA) ? 0 : 1
  const rankB = selected.has(idB) ? 0 : 1
  if (rankA !== rankB) return rankA - rankB
  return idA - idB
}

const itemRelationRowsForItemTab = computed<ItemRelationRow[]>(() => {
  const selected = new Set(getMultiValues('itemLinks').map((x) => Number(x)))
  const currentID = Number(selectedId.value ?? 0)
  const keyword = normalizeKeyword(itemLinkFilter.value)
  return buildItemRelationSource()
    .filter((row) => row.id !== currentID)
    .filter((row) =>
      containsKeyword(keyword, row.id, row.itemType, row.manufacturer, row.model, row.label, row.functionText, row.dnsName, row.principal, row.sn),
    )
    .sort((a, b) => sortWithLinkedFirst(a.id, b.id, selected))
})

const itemRelationRowsForDnsTab = computed<ItemRelationRow[]>(() => {
  const selected = new Set(getMultiValues('itemLinks').map((x) => Number(x)))
  const keyword = normalizeKeyword(itemLinkFilter.value)
  return buildItemRelationSource()
    .filter((row) =>
      containsKeyword(keyword, row.id, row.itemType, row.manufacturer, row.model, row.label, row.dnsName, row.username, row.sn),
    )
    .sort((a, b) => sortWithLinkedFirst(a.id, b.id, selected))
})

const invoiceRelationRowsForItemTab = computed<InvoiceRelationRow[]>(() => {
  const selected = new Set(getMultiValues('invoiceLinks').map((x) => Number(x)))
  const keyword = normalizeKeyword(invoiceLinkFilter.value)
  return buildInvoiceRelationSource()
    .filter((row) => containsKeyword(keyword, row.id, row.vendor, row.number, row.description, row.files, row.date))
    .sort((a, b) => sortWithLinkedFirst(a.id, b.id, selected))
})

const invoiceRelationRowsForContractTab = computed<InvoiceRelationRow[]>(() => {
  const selected = new Set(getMultiValues('invoiceLinks').map((x) => Number(x)))
  const keyword = normalizeKeyword(invoiceLinkFilter.value)
  return buildInvoiceRelationSource()
    .filter((row) => containsKeyword(keyword, row.id, row.vendor, row.number, row.date))
    .sort((a, b) => sortWithLinkedFirst(a.id, b.id, selected))
})

const softwareRelationRowsDetailed = computed<SoftwareRelationRow[]>(() => {
  const selected = new Set(getMultiValues('softwareLinks').map((x) => Number(x)))
  const keyword = normalizeKeyword(softwareLinkFilter.value)
  return buildSoftwareRelationSource()
    .filter((row) => containsKeyword(keyword, row.id, row.manufacturer, row.title, row.version))
    .sort((a, b) => sortWithLinkedFirst(a.id, b.id, selected))
})

const contractRelationRows = computed<ContractRelationRow[]>(() => {
  const selected = new Set(getMultiValues('contractLinks').map((x) => Number(x)))
  const keyword = normalizeKeyword(contractLinkFilter.value)
  return buildContractRelationSource()
    .filter((row) => containsKeyword(keyword, row.id, row.contractor, row.title))
    .sort((a, b) => sortWithLinkedFirst(a.id, b.id, selected))
})

const softwareItemRelationRows = computed<ItemRelationRow[]>(() => {
  const selected = new Set(getMultiValues('itemLinks').map((x) => Number(x)))
  const keyword = normalizeKeyword(itemLinkFilter.value)
  const rows = buildItemRelationSource()
    .filter((row) => isSoftwareSupportedItem(row))
    .filter((row) =>
      containsKeyword(keyword, row.id, row.itemType, row.manufacturer, row.model, row.label, row.dnsName, row.username, row.sn),
    )
  const state = softwareRelationSortState.items
  return sortRelationRowsWithLinkedPriority(rows, selected, (row) => row.id, state.key, state.direction, (row, key) => {
    if (key === 'id') return row.id
    if (key === 'itemType') return row.itemType
    if (key === 'manufacturer') return row.manufacturer
    if (key === 'model') return row.model
    if (key === 'label') return row.label
    if (key === 'dnsName') return row.dnsName
    if (key === 'username') return row.username
    if (key === 'sn') return row.sn
    return row.id
  })
})

const softwareInvoiceRelationRows = computed<InvoiceRelationRow[]>(() => {
  const selected = new Set(getMultiValues('invoiceLinks').map((x) => Number(x)))
  const keyword = normalizeKeyword(invoiceLinkFilter.value)
  const rows = buildInvoiceRelationSource().filter((row) =>
    containsKeyword(keyword, row.id, row.vendor, row.number, row.description, row.files, row.date),
  )
  const state = softwareRelationSortState.invoices
  return sortRelationRowsWithLinkedPriority(rows, selected, (row) => row.id, state.key, state.direction, (row, key) => {
    if (key === 'id') return row.id
    if (key === 'vendor') return row.vendor
    if (key === 'number') return row.number
    if (key === 'description') return `${row.description || ''} ${row.files || ''}`.trim()
    if (key === 'date') return row.date
    return row.id
  })
})

const softwareContractRelationRows = computed<ContractRelationRow[]>(() => {
  const selected = new Set(getMultiValues('contractLinks').map((x) => Number(x)))
  const keyword = normalizeKeyword(contractLinkFilter.value)
  const rows = buildContractRelationSource().filter((row) => containsKeyword(keyword, row.id, row.contractor, row.title))
  const state = softwareRelationSortState.contracts
  return sortRelationRowsWithLinkedPriority(rows, selected, (row) => row.id, state.key, state.direction, (row, key) => {
    if (key === 'id') return row.id
    if (key === 'contractor') return row.contractor
    if (key === 'title') return row.title
    return row.id
  })
})

const softwareFileRelationRows = computed<FileRelationRow[]>(() => {
  const selected = new Set(getMultiValues('fileLinks').map((x) => Number(x)))
  const keyword = normalizeKeyword(fileLinkFilter.value)
  const rows = buildFileRelationSource().filter((row) => {
    if (isInvoiceLinkedFile(row)) return false
    return containsKeyword(keyword, row.id, row.typeDesc, row.title, row.fileName, row.date)
  })
  const state = softwareRelationSortState.files
  return sortRelationRowsWithLinkedPriority(rows, selected, (row) => row.id, state.key, state.direction, (row, key) => {
    if (key === 'id') return row.id
    if (key === 'typeDesc') return row.typeDesc
    if (key === 'title') return row.title
    if (key === 'fileName') return row.fileName
    if (key === 'date') return row.date
    return row.id
  })
})

const invoiceFileRelationRows = computed<FileRelationRow[]>(() => {
  const selected = new Set(getMultiValues('fileLinks').map((x) => Number(x)))
  const keyword = normalizeKeyword(fileLinkFilter.value)
  const rows = buildFileRelationSource().filter((row) => {
    if (!isInvoiceLinkedFile(row)) return false
    return containsKeyword(keyword, row.id, row.typeDesc, row.title, row.fileName, row.date)
  })
  const state = softwareRelationSortState.files
  return sortRelationRowsWithLinkedPriority(rows, selected, (row) => row.id, state.key, state.direction, (row, key) => {
    if (key === 'id') return row.id
    if (key === 'typeDesc') return row.typeDesc
    if (key === 'title') return row.title
    if (key === 'fileName') return row.fileName
    if (key === 'date') return row.date
    return row.id
  })
})

const itemItemRelationRows = computed<ItemRelationRow[]>(() => {
  const selected = new Set(getMultiValues('itemLinks').map((x) => Number(x)))
  return sortItemRelationRows(itemRelationRowsForItemTab.value, selected, nonSoftwareRelationSortState.itemItems)
})

const itemInvoiceRelationRows = computed<InvoiceRelationRow[]>(() => {
  const selected = new Set(getMultiValues('invoiceLinks').map((x) => Number(x)))
  return sortInvoiceRelationRows(invoiceRelationRowsForItemTab.value, selected, nonSoftwareRelationSortState.itemInvoices)
})

const itemSoftwareRelationRows = computed<SoftwareRelationRow[]>(() => {
  const selected = new Set(getMultiValues('softwareLinks').map((x) => Number(x)))
  return sortSoftwareRelationRows(softwareRelationRowsDetailed.value, selected, nonSoftwareRelationSortState.itemSoftware)
})

const itemContractRelationRows = computed<ContractRelationRow[]>(() => {
  const selected = new Set(getMultiValues('contractLinks').map((x) => Number(x)))
  return sortContractRelationRows(contractRelationRows.value, selected, nonSoftwareRelationSortState.itemContracts)
})

const invoiceItemRelationRows = computed<ItemRelationRow[]>(() => {
  const selected = new Set(getMultiValues('itemLinks').map((x) => Number(x)))
  return sortItemRelationRows(itemRelationRowsForDnsTab.value, selected, nonSoftwareRelationSortState.invoiceItems)
})

const invoiceSoftwareRelationRows = computed<SoftwareRelationRow[]>(() => {
  const selected = new Set(getMultiValues('softwareLinks').map((x) => Number(x)))
  return sortSoftwareRelationRows(softwareRelationRowsDetailed.value, selected, nonSoftwareRelationSortState.invoiceSoftware)
})

const invoiceContractRelationRows = computed<ContractRelationRow[]>(() => {
  const selected = new Set(getMultiValues('contractLinks').map((x) => Number(x)))
  return sortContractRelationRows(contractRelationRows.value, selected, nonSoftwareRelationSortState.invoiceContracts)
})

const contractItemRelationRows = computed<ItemRelationRow[]>(() => {
  const selected = new Set(getMultiValues('itemLinks').map((x) => Number(x)))
  return sortItemRelationRows(itemRelationRowsForDnsTab.value, selected, nonSoftwareRelationSortState.contractItems)
})

const contractSoftwareRelationRows = computed<SoftwareRelationRow[]>(() => {
  const selected = new Set(getMultiValues('softwareLinks').map((x) => Number(x)))
  return sortSoftwareRelationRows(softwareRelationRowsDetailed.value, selected, nonSoftwareRelationSortState.contractSoftware)
})

const contractInvoiceRelationRows = computed<InvoiceRelationRow[]>(() => {
  const selected = new Set(getMultiValues('invoiceLinks').map((x) => Number(x)))
  return sortInvoiceRelationRows(invoiceRelationRowsForContractTab.value, selected, nonSoftwareRelationSortState.contractInvoices)
})

const fileItemRelationRows = computed<ItemRelationRow[]>(() => {
  const selected = new Set(getMultiValues('itemLinks').map((x) => Number(x)))
  return sortItemRelationRows(itemRelationRowsForDnsTab.value, selected, nonSoftwareRelationSortState.fileItems)
})

const fileSoftwareRelationRows = computed<SoftwareRelationRow[]>(() => {
  const selected = new Set(getMultiValues('softwareLinks').map((x) => Number(x)))
  return sortSoftwareRelationRows(softwareRelationRowsDetailed.value, selected, nonSoftwareRelationSortState.fileSoftware)
})

const fileContractRelationRows = computed<ContractRelationRow[]>(() => {
  const selected = new Set(getMultiValues('contractLinks').map((x) => Number(x)))
  return sortContractRelationRows(contractRelationRows.value, selected, nonSoftwareRelationSortState.fileContracts)
})

const itemOverviewRows = computed(() => {
  const ids = (fieldKey: string) => getMultiValues(fieldKey).map((v) => Number(v)).filter((v) => Number.isFinite(v) && v > 0)

  if (activeOverviewTab.value === 'items') {
    const byID = new Map<number, ItemRelationRow>()
    for (const row of buildItemRelationSource()) byID.set(row.id, row)
    const rows = ids('itemLinks').map((id) => {
      const row = byID.get(id)
      return {
        id,
        text: row ? formatOverviewItemText(row) : `- - [-, ID:${id}]`,
      }
    })
    return toOverviewRows(rows, 'items', (row) => row.text)
  }

  if (activeOverviewTab.value === 'software') {
    const byID = new Map<number, SoftwareRelationRow>()
    for (const row of buildSoftwareRelationSource()) byID.set(row.id, row)
    const rows = ids('softwareLinks').map((id) => {
      const row = byID.get(id)
      return {
        id,
        text: row ? formatOverviewSoftwareText(row) : `- - [ID:${id}]`,
      }
    })
    return toOverviewRows(rows, 'software', (row) => row.text)
  }

  if (activeOverviewTab.value === 'invoices') {
    const byID = new Map<number, InvoiceRelationRow>()
    for (const row of buildInvoiceRelationSource()) byID.set(row.id, row)
    const rows = ids('invoiceLinks').map((id) => {
      const row = byID.get(id)
      return {
        id,
        text: row ? formatOverviewInvoiceText(row) : `(-) - - [ID:${id}]`,
      }
    })
    return toOverviewRows(rows, 'invoices', (row) => row.text)
  }

  const byID = new Map<number, ContractRelationRow>()
  for (const row of buildContractRelationSource()) byID.set(row.id, row)
  const rows = ids('contractLinks').map((id) => {
    const row = byID.get(id)
    return {
      id,
      text: row ? formatOverviewContractText(row) : `(- -) - -- [ID:${id}]`,
    }
  })
  return toOverviewRows(rows, 'contracts', (row) => row.text)
})

const softwareOverviewRows = computed(() => {
  const targetKey =
    activeSoftwareOverviewTab.value === 'items'
      ? 'itemLinks'
      : activeSoftwareOverviewTab.value === 'invoices'
        ? 'invoiceLinks'
        : 'contractLinks'
  const type =
    activeSoftwareOverviewTab.value === 'items' ? 'items' : activeSoftwareOverviewTab.value === 'invoices' ? 'invoices' : 'contracts'
  const rows = getMultiValues(targetKey).map((idRaw) => {
    const id = Number(idRaw)
    return {
      id,
      text: getRelationLookupText(type, id),
    }
  })
  return toOverviewRows(rows, type, (row) => row.text)
})

const invoiceOverviewRows = computed(() => {
  const targetKey =
    activeInvoiceOverviewTab.value === 'items'
      ? 'itemLinks'
      : activeInvoiceOverviewTab.value === 'software'
        ? 'softwareLinks'
        : 'contractLinks'
  const type =
    activeInvoiceOverviewTab.value === 'items' ? 'items' : activeInvoiceOverviewTab.value === 'software' ? 'software' : 'contracts'
  const rows = getMultiValues(targetKey).map((idRaw) => {
    const id = Number(idRaw)
    return {
      id,
      text: getRelationLookupText(type, id),
    }
  })
  return toOverviewRows(rows, type, (row) => row.text)
})

const contractOverviewRows = computed(() => {
  const targetKey =
    activeContractOverviewTab.value === 'items'
      ? 'itemLinks'
      : activeContractOverviewTab.value === 'software'
        ? 'softwareLinks'
        : 'invoiceLinks'
  const type =
    activeContractOverviewTab.value === 'items' ? 'items' : activeContractOverviewTab.value === 'software' ? 'software' : 'invoices'
  const rows = getMultiValues(targetKey).map((idRaw) => {
    const id = Number(idRaw)
    return {
      id,
      text: getRelationLookupText(type, id),
    }
  })
  return toOverviewRows(rows, type, (row) => row.text)
})

const fileOverviewRows = computed(() => {
  const targetKey =
    activeFileOverviewTab.value === 'items'
      ? 'itemLinks'
      : activeFileOverviewTab.value === 'software'
        ? 'softwareLinks'
        : activeFileOverviewTab.value === 'invoices'
          ? 'invoiceLinks'
          : 'contractLinks'
  const type =
    activeFileOverviewTab.value === 'items'
      ? 'items'
      : activeFileOverviewTab.value === 'software'
        ? 'software'
        : activeFileOverviewTab.value === 'invoices'
          ? 'invoices'
          : 'contracts'
  const rows = getMultiValues(targetKey).map((idRaw) => {
    const id = Number(idRaw)
    return {
      id,
      text: getRelationLookupText(type, id),
    }
  })
  return toOverviewRows(rows, type, (row) => row.text)
})

const agentRelatedItemRows = computed(() => {
  const id = Number(selectedId.value ?? 0)
  if (!id) return []
  return buildItemRelationSource().filter((row) => row.manufacturerID === id)
})

const agentRelatedSoftwareRows = computed(() => {
  const id = Number(selectedId.value ?? 0)
  if (!id) return []
  return buildSoftwareRelationSource().filter((row) => row.manufacturerID === id)
})

const agentRelatedVendorInvoiceRows = computed(() => {
  const id = Number(selectedId.value ?? 0)
  if (!id) return []
  return buildInvoiceRelationSource().filter((row) => row.vendorID === id)
})

const agentRelatedBuyerInvoiceRows = computed(() => {
  const id = Number(selectedId.value ?? 0)
  if (!id) return []
  return buildInvoiceRelationSource().filter((row) => row.buyerID === id)
})

const agentOverviewRows = computed(() => {
  if (activeAgentOverviewTab.value === 'items') {
    return toOverviewRows(agentRelatedItemRows.value, 'items', formatOverviewItemText)
  }
  if (activeAgentOverviewTab.value === 'software') {
    return toOverviewRows(agentRelatedSoftwareRows.value, 'software', formatOverviewSoftwareText)
  }
  if (activeAgentOverviewTab.value === 'invoicesVendor') {
    return toOverviewRows(agentRelatedVendorInvoiceRows.value, 'invoices', formatOverviewInvoiceText)
  }
  return toOverviewRows(agentRelatedBuyerInvoiceRows.value, 'invoices', formatOverviewInvoiceText)
})

const locationRelatedItemRows = computed(() => {
  const id = Number(selectedId.value ?? 0)
  if (!id) return []
  return buildItemRelationSource().filter((row) => row.locationID === id)
})

const locationRelatedRackRows = computed(() => {
  const locationID = Number(selectedId.value ?? 0)
  if (!locationID) return []
  const rows = (bootstrap.lookups.racks ?? []) as Record<string, unknown>[]
  return rows
    .filter((row) => Number(row.locationid ?? 0) === locationID)
    .map((row) => ({
      id: Number(row.id ?? 0),
      label: String(row.label ?? '-'),
    }))
    .filter((row) => row.id > 0)
})

const userRelatedItemRows = computed(() => {
  const id = Number(selectedId.value ?? 0)
  if (!id) return []
  return buildItemRelationSource().filter((row) => row.userID === id)
})

const rackRelatedItemRows = computed(() => {
  const id = Number(selectedId.value ?? 0)
  if (!id) return []
  return buildItemRelationSource().filter((row) => row.rackID === id)
})

const userOverviewRows = computed(() => {
  return toOverviewRows(userRelatedItemRows.value, 'items', formatOverviewItemText)
})

const locationOverviewRows = computed(() => {
  if (activeLocationOverviewTab.value === 'racks') {
    return toOverviewRows(locationRelatedRackRows.value, 'racks', (row) => row.label || '-')
  }
  return toOverviewRows(locationRelatedItemRows.value, 'items', formatOverviewItemText)
})

const sortedLocationAreas = computed(() => {
  return [...locationAreas.value].sort((a, b) => naturalCompare(a.id, b.id))
})

const locationAssociatedRackCount = computed(() => locationRelatedRackRows.value.length)
const locationAssociationSummary = computed(() => `${locationRelatedItemRows.value.length} / ${locationAssociatedRackCount.value}`)
const locationFloorplanName = computed(() => String(recordDetail.value?.floorplanfn ?? '').trim())
const selectedLocationFloorplanName = computed(() => {
  const file = form.file
  return file instanceof File ? file.name : ''
})

const userItemCount = computed(() => userRelatedItemRows.value.length)
const rackPopulation = computed(() => rackRelatedItemRows.value.length)
const rackOccupationUnits = computed(() => {
  return rackRelatedItemRows.value.reduce((sum, row) => {
    const used = Number(row.uSize ?? 0)
    return sum + (Number.isFinite(used) && used > 0 ? used : 0)
  }, 0)
})
const rackTotalUnits = computed(() => {
  const total = Number(form.uSize ?? 0)
  return Number.isFinite(total) && total > 0 ? total : 0
})
const rackOccupationPercent = computed(() => {
  if (rackTotalUnits.value <= 0) return 0
  const ratio = (rackOccupationUnits.value / rackTotalUnits.value) * 100
  return Math.max(0, Math.min(100, Number(ratio.toFixed(1))))
})
const rackViewData = computed(() =>
  buildRackViewData({
    totalUnits: rackTotalUnits.value,
    reverse: Number(form.revNums ?? 0) === 1,
    itemRows: rackRelatedItemRows.value,
  }),
)

const itemLocAreaOptions = computed(() => {
  if (!isItemResource.value) return [] as ReturnType<typeof getOptionsByFieldKey>
  return getOptionsByFieldKey('locAreaId')
})

const contractRenewalEnteredByOptions = computed(() => {
  const rows = (bootstrap.lookups.users ?? []) as Record<string, unknown>[]
  const options: { label: string; value: string }[] = []
  const seen = new Set<string>()
  for (const row of rows) {
    const username = String(row.username ?? '').trim()
    const key = username.toLowerCase()
    if (!username || seen.has(key)) continue
    seen.add(key)
    options.push({ label: username, value: username })
  }
  return options.sort((a, b) => naturalCompare(a.label, b.label))
})

function getContractRenewalEnteredByOptions(currentRaw: unknown) {
  const current = String(currentRaw ?? '').trim()
  if (!current) return contractRenewalEnteredByOptions.value
  const exists = contractRenewalEnteredByOptions.value.some((option) => option.value === current)
  if (exists) return contractRenewalEnteredByOptions.value
  return [{ label: current, value: current }, ...contractRenewalEnteredByOptions.value]
}

const itemRackOptions = computed(() => {
  if (!isItemResource.value) return [] as ReturnType<typeof getOptionsByFieldKey>
  return getOptionsByFieldKey('rackId')
})

const selectedItemRackLookupRow = computed(() => {
  if (!isItemResource.value) return null as Record<string, unknown> | null
  const rackID = Number(form.rackId ?? 0)
  if (!Number.isFinite(rackID) || rackID <= 0) return null
  const rows = (bootstrap.lookups.racks ?? []) as Record<string, unknown>[]
  return rows.find((row) => Number(row.id ?? 0) === rackID) ?? null
})

const selectedItemRackTotalUnits = computed(() => parsePositiveInt(selectedItemRackLookupRow.value?.usize ?? selectedItemRackLookupRow.value?.uSize))

const selectedItemRackReverse = computed(() => Number(selectedItemRackLookupRow.value?.revnums ?? 0) === 1)

const itemRackPositionOptions = computed(() => {
  if (!isItemResource.value) return [] as number[]
  const total = selectedItemRackTotalUnits.value
  if (total <= 0) return [] as number[]
  return Array.from({ length: total }, (_, index) => index + 1)
})

const itemLocAreaPlaceholderText = computed(() => {
  const locationID = Number(form.locationId ?? 0)
  if (!Number.isFinite(locationID) || locationID <= 0) return '请选择'
  return itemLocAreaOptions.value.length === 0 ? '未定义区域' : '请选择'
})

const itemRackPlaceholderText = computed(() => {
  const locationID = Number(form.locationId ?? 0)
  if (!Number.isFinite(locationID) || locationID <= 0) return '请选择'
  return itemRackOptions.value.length === 0 ? '在该地点无机架' : '请选择'
})

function resetItemEditorState() {
  activeItemTab.value = 'itemData'
  activeOverviewTab.value = 'items'
  itemActions.value = []
  itemTags.value = []
  itemTagEditorOpen.value = false
  itemTagInput.value = ''
  itemTagSaving.value = false
  itemTagMessage.value = ''
  itemLinkFilter.value = ''
  invoiceLinkFilter.value = ''
  softwareLinkFilter.value = ''
  contractLinkFilter.value = ''
  fileLinkFilter.value = ''
  softwareTagMessage.value = ''
}

function normalizeDateInput(raw: unknown) {
  if (raw === null || raw === undefined || raw === '') return ''
  const str = String(raw).trim()
  if (!str) return ''
  if (/^\d{4}-\d{2}-\d{2}$/.test(str)) return str
  const ts = Number(str)
  if (!Number.isNaN(ts) && ts > 0) {
    const date = new Date(ts * 1000)
    const y = date.getFullYear()
    const m = String(date.getMonth() + 1).padStart(2, '0')
    const d = String(date.getDate()).padStart(2, '0')
    return `${y}-${m}-${d}`
  }
  return str
}

type RequiredValidationRule = {
  key: string
  label: string
  type: FieldType
  optionsKey?: string
}

const requiredFieldLabelOverrides: Record<string, Record<string, string>> = {
  agents: {
    types: '类型',
  },
  items: {
    uSize: '大小(U)',
  },
  contracts: {
    number: '数量',
  },
  files: {
    date: '签署日期',
  },
  locations: {
    name: '建筑名称',
  },
  racks: {
    uSize: '大小(U)*',
  },
}

function getRequiredFieldLabel(resourceKey: string, field: ResourceField) {
  return requiredFieldLabelOverrides[resourceKey]?.[field.key] ?? field.label
}

function appendRequiredRule(
  rules: RequiredValidationRule[],
  fieldKey: string,
  fallbackLabel: string,
  resourceFields: ResourceField[],
  resourceKey: string,
) {
  if (rules.some((rule) => rule.key === fieldKey)) return
  const field = resourceFields.find((item) => item.key === fieldKey)
  if (!field) return
  rules.push({
    key: field.key,
    label: requiredFieldLabelOverrides[resourceKey]?.[field.key] ?? fallbackLabel,
    type: field.type,
    optionsKey: field.optionsKey,
  })
}

function buildRequiredValidationRules() {
  if (!resource.value) return [] as RequiredValidationRule[]
  const resourceKey = resource.value.key
  const resourceFields = resource.value.fields
  const rules: RequiredValidationRule[] = resourceFields
    .filter((field) => field.required)
    .map((field) => ({
      key: field.key,
      label: getRequiredFieldLabel(resourceKey, field),
      type: field.type,
      optionsKey: field.optionsKey,
    }))

  if (resourceKey === 'software') {
    appendRequiredRule(rules, 'purchaseDate', '采购日期', resourceFields, resourceKey)
  }
  if (resourceKey === 'items') {
    const rackID = Number(form.rackId ?? 0)
    const rackPosition = Number(form.rackPosition ?? 0)
    if (Number.isFinite(rackID) && rackID > 0) {
      appendRequiredRule(rules, 'rackPosition', '机架位置', resourceFields, resourceKey)
    }
    if (Number.isFinite(rackPosition) && rackPosition > 0) {
      appendRequiredRule(rules, 'uSize', '大小(U)', resourceFields, resourceKey)
    }
  }
  if (resourceKey === 'agents' && !selectedId.value) {
    appendRequiredRule(rules, 'types', '类型', resourceFields, resourceKey)
  }
  if (resourceKey === 'users') {
    appendRequiredRule(rules, 'userType', '类型', resourceFields, resourceKey)
    if (!selectedId.value) {
      appendRequiredRule(rules, 'password', '密码', resourceFields, resourceKey)
    }
  }
  return rules
}

function hasRequiredFieldValue(rule: RequiredValidationRule) {
  const value = form[rule.key]
  if (rule.type === 'select') {
    if (value === '' || value === null || value === undefined) return false
    const numeric = Number(value)
    if (rule.optionsKey) return Number.isFinite(numeric) && numeric > 0
    if (!Number.isNaN(numeric)) return true
    return String(value).trim() !== ''
  }
  if (rule.type === 'multiselect') {
    return Array.isArray(value) && value.length > 0
  }
  if (rule.type === 'number') {
    if (value === '' || value === null || value === undefined) return false
    const numeric = Number(value)
    return Number.isFinite(numeric)
  }
  if (rule.type === 'file') {
    return value instanceof File
  }
  if (rule.type === 'date') {
    return Boolean(normalizeDateInput(value))
  }
  return String(value ?? '').trim() !== ''
}

function getRequiredFieldValidationError() {
  const missingLabels = buildRequiredValidationRules()
    .filter((rule) => !hasRequiredFieldValue(rule))
    .map((rule) => rule.label)

  if (missingLabels.length === 0) return ''
  return `请完善必填项：${missingLabels.join('、')}`
}

function isDateRangeInvalid(startRaw: unknown, endRaw: unknown) {
  const start = normalizeDateInput(startRaw)
  const end = normalizeDateInput(endRaw)
  if (!start || !end) return false
  if (!/^\d{4}-\d{2}-\d{2}$/.test(start) || !/^\d{4}-\d{2}-\d{2}$/.test(end)) return false
  return end < start
}

function getContractDateValidationError() {
  if (isDateRangeInvalid(form.startDate, form.currentEndDate)) {
    return '合同属性中，结束日期不能早于开始日期'
  }
  const invalidRenewalIndex = contractRenewals.value.findIndex((row) => isDateRangeInvalid(row.endDateBefore, row.endDateAfter))
  if (invalidRenewalIndex >= 0) {
    return `备件第 ${invalidRenewalIndex + 1} 行: 到期后不能早于到期前`
  }
  const invalidRenewalEffectiveIndex = contractRenewals.value.findIndex((row) => isDateRangeInvalid(row.effectiveDate, row.endDateAfter))
  if (invalidRenewalEffectiveIndex >= 0) {
    return `备件第 ${invalidRenewalEffectiveIndex + 1} 行: 到期后不能早于生效日期`
  }
  return ''
}

function getRackViewValidationError() {
  if (!isRackResource.value) return ''
  return rackViewData.value.warnings[0] ?? ''
}

function getItemRackPlacementValidationError() {
  if (resource.value?.key !== 'items') return ''

  const rackID = Number(form.rackId ?? 0)
  if (!Number.isFinite(rackID) || rackID <= 0) return ''

  const rackPosition = Number(form.rackPosition ?? 0)
  if (!Number.isFinite(rackPosition) || rackPosition <= 0) return ''

  const units = Number(form.uSize ?? 0)
  if (!Number.isFinite(units) || units <= 0) return ''

  const totalUnits = selectedItemRackTotalUnits.value
  const occupiedUnits = getRackUnitSpan(rackPosition, units, selectedItemRackReverse.value)
  if (totalUnits > 0 && occupiedUnits.some((unit) => unit < 1 || unit > totalUnits)) {
    return '机架位置超出所选机架范围'
  }

  const currentID = Number(selectedId.value ?? 0)
  const usedByOthers = new Set<number>()
  for (const row of buildItemRelationSource()) {
    if (row.rackID !== rackID || row.id === currentID) continue
    const rowPosition = Number(row.rackPosition ?? 0)
    const rowUnits = Number(row.uSize ?? 0)
    if (!Number.isFinite(rowPosition) || rowPosition <= 0 || !Number.isFinite(rowUnits) || rowUnits <= 0) continue
    for (const unit of getRackUnitSpan(rowPosition, rowUnits, selectedItemRackReverse.value)) {
      usedByOthers.add(unit)
    }
  }

  const conflicts = occupiedUnits.filter((unit) => usedByOthers.has(unit))
  if (conflicts.length === 0) return ''

  const conflictText = Array.from(new Set(conflicts))
    .sort((a, b) => naturalCompare(a, b))
    .join('、')
  return `机架行 ${conflictText} 已被其他硬件占用`
}

function parseMultiSelectValue(field: ResourceField, raw: unknown) {
  if (Array.isArray(raw)) return raw.map((v) => String(Number(v)))

  if (typeof raw === 'number' && raw > 0) {
    const opts = getFieldOptions(field)
    const selected: string[] = []
    for (const opt of opts) {
      const v = Number(opt.value)
      if (v > 0 && (raw & v) === v) selected.push(String(v))
    }
    return selected
  }

  if (typeof raw === 'string' && raw.trim() !== '') {
    return raw
      .split(',')
      .map((x) => x.trim())
      .filter(Boolean)
  }
  return []
}

function convertReadValue(field: ResourceField, sourceValue: unknown) {
  if (field.type === 'multiselect') return parseMultiSelectValue(field, sourceValue)
  if (field.type === 'number') {
    if (sourceValue === null || sourceValue === undefined || sourceValue === '') return ''
    return Number(sourceValue)
  }
  if (field.type === 'select') {
    if (sourceValue === null || sourceValue === undefined || sourceValue === '') return ''
    if (field.optionsKey) {
      const numeric = Number(sourceValue)
      if (!Number.isFinite(numeric) || numeric <= 0) return ''
      return String(numeric)
    }
    return String(sourceValue)
  }
  if (field.type === 'date') return normalizeDateInput(sourceValue)
  if (field.type === 'file') return null
  if (field.type === 'text' || field.type === 'textarea') {
    if (sourceValue === null || sourceValue === undefined) return ''
    return String(sourceValue)
  }
  return sourceValue ?? ''
}

function toRequestPayload() {
  if (!resource.value) return {}
  const payload: Record<string, unknown> = {}
  for (const field of resource.value.fields) {
    const value = form[field.key]
    if (field.type === 'number') {
      payload[field.key] = value === '' || value === null ? 0 : Number(value)
      continue
    }
    if (field.type === 'multiselect') {
      payload[field.key] = Array.isArray(value) ? value.map((v) => Number(v)) : []
      continue
    }
    if (field.type === 'select') {
      payload[field.key] = value === '' ? 0 : Number(value)
      continue
    }
    if (field.type !== 'file') payload[field.key] = value
  }
  if (resource.value.key === 'agents') {
    payload.contacts = agentContacts.value
      .map((row) => ({
        name: sanitizePipeHashText(row.name),
        phones: sanitizePipeHashText(row.phones),
        email: sanitizePipeHashText(row.email),
        role: sanitizePipeHashText(row.role),
        comments: sanitizePipeHashText(row.comments),
      }))
      .filter((row) => row.name || row.phones || row.email || row.role || row.comments)
    payload.urls = agentURLs.value
      .map((row) => ({
        description: sanitizePipeHashText(row.description),
        url: sanitizePipeHashText(row.url),
      }))
      .filter((row) => row.description || row.url)
  }
  for (const relationKey of ['itemLinks', 'softwareLinks', 'invoiceLinks', 'contractLinks', 'fileLinks'] as const) {
    if (Array.isArray(form[relationKey])) {
      payload[relationKey] = getMultiValues(relationKey).map((value) => Number(value))
    }
  }
  if (resource.value.key === 'contracts') {
    payload.renewals = serializeContractRenewals(contractRenewals.value)
  }
  if (['items', 'software', 'invoices', 'contracts'].includes(resource.value.key)) {
    payload.cleanupFileLinks = Array.from(new Set(pendingCleanupFileLinks.value.filter((id) => Number.isFinite(id) && id > 0)))
      .sort((left, right) => naturalCompare(left, right))
  }
  return payload
}

function toMultipartPayload() {
  if (!resource.value) return new FormData()
  const data = new FormData()
  for (const field of resource.value.fields) {
    const value = form[field.key]
    if (field.type === 'file') {
      const file = value as File | null
      if (file) data.append(field.key, file)
      continue
    }
    if (field.type === 'multiselect') {
      data.append(field.key, Array.isArray(value) ? value.join(',') : '')
      continue
    }
    data.append(field.key, String(value ?? ''))
  }
  return data
}

function openCreate() {
  selectedId.value = null
  recordDetail.value = null
  resetForm()
  resetItemEditorState()
  resetNonItemEditorState()
  initializeItemDefaults()
  initializeSoftwareDefaults()
  initializeRackDefaults()
  if (resource.value?.key === 'contracts') {
    contractRenewals.value = [createDefaultRenewalRow()]
  }
  if (resource.value?.key === 'users') {
    form.userType = '0'
  }
  if (resource.value?.key === 'items') {
    itemUploadForm.date = ''
  }
  if (resource.value?.key === 'invoices') {
    invoiceUploadForm.date = ''
  }
  if (resource.value?.key === 'contracts') {
    contractUploadForm.date = ''
  }
  if (resource.value?.key === 'agents') {
    ensureAgentRows()
  }
  if (resource.value && ['items', 'software', 'invoices', 'contracts', 'files', 'agents', 'locations', 'users', 'racks'].includes(resource.value.key)) {
    void ensureRelationCache(['items', 'software', 'invoices', 'contracts', 'files'])
  }
  if (resource.value?.key === 'racks') {
    void refreshRelationCache(['items'])
  }
  drawerOpen.value = true
}

function isCreateQueryEnabled() {
  const raw = route.query.create
  const text = Array.isArray(raw) ? String(raw[0] ?? '') : String(raw ?? '')
  const normalized = text.trim().toLowerCase()
  return normalized === '1' || normalized === 'true' || normalized === 'yes' || normalized === 'y'
}

function getEditQueryID() {
  const raw = route.query.edit
  const text = Array.isArray(raw) ? String(raw[0] ?? '') : String(raw ?? '')
  const id = Number.parseInt(text.trim(), 10)
  return Number.isFinite(id) && id > 0 ? id : 0
}

function clearRouteQueryKey(removingKey: string) {
  const nextQuery: Record<string, string | string[]> = {}
  for (const [key, value] of Object.entries(route.query)) {
    if (key === removingKey) continue
    if (typeof value === 'string') {
      nextQuery[key] = value
      continue
    }
    if (Array.isArray(value)) {
      nextQuery[key] = value.filter((v): v is string => typeof v === 'string')
    }
  }
  void router.replace({ path: route.path, query: nextQuery })
}

function clearCreateQuery() {
  clearRouteQueryKey('create')
}

function clearEditQuery() {
  clearRouteQueryKey('edit')
  clearRouteQueryKey('vh')
}

function applyCreateQueryAction() {
  if (!resource.value || !canWrite.value) return
  if (!isCreateQueryEnabled()) return
  openCreate()
  clearCreateQuery()
}

async function applyEditQueryAction() {
  if (!resource.value || !canWrite.value) return false
  const editID = getEditQueryID()
  if (!editID) return false
  try {
    await openEdit({ id: editID })
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '编辑数据加载失败'
  } finally {
    clearEditQuery()
  }
  return true
}

async function applyRouteQueryActions() {
  const edited = await applyEditQueryAction()
  if (edited) return
  applyCreateQueryAction()
}

async function openEdit(row: GenericRow) {
  if (!resource.value) return
  selectedId.value = rowKey(row)
  recordDetail.value = null
  resetForm()
  resetItemEditorState()
  resetNonItemEditorState()
  await refreshDirtyEditorDependencies(resource.value.key)
  if (resource.value.key === 'racks') {
    await refreshRelationCache(['items'])
  }
  const { data } = await api.get(`${resource.value.endpoint}/${selectedId.value}`)
  recordDetail.value = data as GenericRow
  for (const field of resource.value.fields) {
    const sourceKey = getSourceKey(field)
    form[field.key] = convertReadValue(field, data[sourceKey])
  }
  form.fileLinks = Array.isArray(data.fileLinks) ? data.fileLinks.map((id: unknown) => String(Number(id))) : []
  if (resource.value.key === 'items') {
    itemActions.value = Array.isArray(data.actions) ? (data.actions as GenericRow[]) : []
    assignItemTags(data.tags)
    initializeItemDefaults()
  }
  if (resource.value.key === 'software') {
    assignSoftwareTags(data.tags)
    initializeSoftwareDefaults()
  }
  initializeRackDefaults()
  if (resource.value.key === 'locations') {
    locationAreas.value = Array.isArray(data.areas) ? (data.areas as GenericRow[]) : []
    await loadLocationAreas()
    await loadLocationFloorplanPreview()
  }
  if (resource.value.key === 'contracts') {
    contractRenewals.value = parseContractRenewals(data.renewals)
    contractUploadForm.date = ''
    await loadContractEvents()
  }
  if (resource.value.key === 'invoices') {
    invoiceUploadForm.date = ''
  }
  if (resource.value.key === 'items') {
    itemUploadForm.date = ''
  }
  if (resource.value.key === 'agents') {
    agentContacts.value = parseAgentContacts(data.contacts)
    agentURLs.value = parseAgentURLs(data.urls)
    agentContactInfo.value = String(data.contactinfo ?? '')
    if (form.contactInfo === '' || form.contactInfo === null || form.contactInfo === undefined) {
      form.contactInfo = agentContactInfo.value
    }
    ensureAgentRows()
  }
  drawerOpen.value = true
}

async function refreshBootstrapIfNeeded() {
  if (!resource.value) return
  const refKeys = new Set(['items', 'software', 'invoices', 'contracts', 'files', 'agents', 'users', 'locations', 'racks'])
  if (refKeys.has(resource.value.key)) {
    relationCache.items = []
    relationCache.software = []
    relationCache.invoices = []
    relationCache.contracts = []
    relationCache.files = []
    bootstrap.reset()
    await bootstrap.load()
  }
}

function getInputValue(event: Event) {
  const target = event.target as HTMLInputElement | null
  return target?.value ?? ''
}

function getInputChecked(event: Event) {
  const target = event.target as HTMLInputElement | null
  return Boolean(target?.checked)
}

function onSearchInput(event: Event) {
  search.value = getInputValue(event)
  page.value = 1
  void loadRows()
}

async function loadRows() {
  if (!resource.value) return
  const seq = ++loadRowsSeq
  const resourceKey = resource.value.key
  const endpoint = resource.value.endpoint
  loading.value = true
  error.value = ''
  try {
    const params: Record<string, unknown> = { search: search.value || undefined }
    if (pageableResourceKeys.has(resourceKey)) {
      params.limit = -1
      params.offset = 0
    }
    const { data } = await api.get(endpoint, {
      params,
    })
    if (seq !== loadRowsSeq || !resource.value || resource.value.key !== resourceKey) return
    rows.value = Array.isArray(data) ? data : []
    if (resourceKey in relationCache) {
      relationCache[resourceKey as RelationCacheKey] = rows.value
    }
  } catch (err: unknown) {
    if (seq !== loadRowsSeq) return
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '数据加载失败'
  } finally {
    if (seq !== loadRowsSeq) return
    loading.value = false
  }
}

async function save() {
  if (!resource.value) return
  const currentResourceKey = resource.value.key
  saving.value = true
  error.value = ''
  try {
    const requiredError = getRequiredFieldValidationError()
    if (requiredError) {
      error.value = requiredError
      noticeStore.error(requiredError)
      return
    }

    if (resource.value.key === 'items') {
      const itemRackPlacementError = getItemRackPlacementValidationError()
      if (itemRackPlacementError) {
        error.value = itemRackPlacementError
        noticeStore.error(itemRackPlacementError)
        return
      }
    }

    if (resource.value.key === 'contracts') {
      const contractDateError = getContractDateValidationError()
      if (contractDateError) {
        error.value = contractDateError
        noticeStore.error(contractDateError)
        return
      }
    }
    if (resource.value.key === 'racks') {
      await refreshRelationCache(['items'])
      const rackViewError = getRackViewValidationError()
      if (rackViewError) {
        error.value = rackViewError
        noticeStore.error(rackViewError)
        return
      }
    }
    if (resource.value.key === 'locations') {
      const floorplanFileError = getLocationFloorplanFileValidationError(form.file instanceof File ? form.file : null)
      if (floorplanFileError) {
        error.value = floorplanFileError
        noticeStore.error(floorplanFileError)
        return
      }
    }
    if (resource.value.key === 'files') {
      const fileUploadError = getFileUploadValidationError(form.file instanceof File ? form.file : null)
      if (fileUploadError) {
        error.value = fileUploadError
        noticeStore.error(fileUploadError)
        return
      }
    }
    let responseData: GenericRow | null = null
    if (resource.value.multipart) {
      const body = toMultipartPayload()
      if (selectedId.value) {
        const { data } = await api.put(`${resource.value.endpoint}/${selectedId.value}`, body)
        responseData = (data as GenericRow) ?? null
      } else {
        const { data } = await api.post(resource.value.endpoint, body)
        responseData = (data as GenericRow) ?? null
      }
    } else {
      const body = toRequestPayload()
      if (selectedId.value) {
        const { data } = await api.put(`${resource.value.endpoint}/${selectedId.value}`, body)
        responseData = (data as GenericRow) ?? null
      } else {
        const { data } = await api.post(resource.value.endpoint, body)
        responseData = (data as GenericRow) ?? null
      }
    }
    let recentHistoryID = Number(responseData?.id ?? selectedId.value ?? 0)
    if (resource.value.key === 'locations') {
      const savedID = Number(selectedId.value ?? responseData?.id ?? 0)
      if (savedID > 0) {
        selectedId.value = savedID
        recentHistoryID = savedID
        await syncPendingLocationAreas(savedID)
      }
    }
    if (recentHistoryID > 0) {
      void recordRecentViewHistory(recentHistoryID)
    }
    drawerOpen.value = false
    await refreshBootstrapIfNeeded()
    await loadRows()
    const dependencyConfig = resourceEditorDependencies[currentResourceKey]
    if (dependencyConfig?.relationKeys?.length) {
      await ensureRelationCache(dependencyConfig.relationKeys)
    }
    broadcastResourceMutation(currentResourceKey, currentResourceKey === 'locations' ? { bootstrapKeys: ['locareas'] } : undefined)
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '保存失败'
    if (error.value) noticeStore.error(error.value)
  } finally {
    saving.value = false
  }
}

function openDeleteConfirm(target: DeleteTarget) {
  deleteTarget.value = target
  confirmOpen.value = true
}

function closeDeleteConfirm() {
  if (deleting.value) return
  confirmOpen.value = false
  deleteTarget.value = null
}

async function remove(id: number) {
  if (!resource.value) return
  openDeleteConfirm({ kind: 'row', id })
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  deleting.value = true
  error.value = ''
  const target = deleteTarget.value
  try {
    if (target.kind === 'rows') {
      if (!resource.value) return
      const currentResourceKey = resource.value.key
      let successCount = 0
      let failedCount = 0
      let firstError = ''
      for (const id of target.ids) {
        try {
          await api.delete(`${resource.value.endpoint}/${id}`)
          successCount += 1
        } catch (err: unknown) {
          failedCount += 1
          if (!firstError) {
            firstError = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '删除失败'
          }
        }
      }
      if (successCount > 0) {
        await refreshBootstrapIfNeeded()
        await loadRows()
        selectedRowIds.value = []
        broadcastResourceMutation(currentResourceKey, currentResourceKey === 'locations' ? { bootstrapKeys: ['locareas'] } : undefined)
        noticeStore.success(`已删除 ${successCount} 条记录`)
      }
      if (failedCount > 0) {
        noticeStore.error(firstError || `有 ${failedCount} 条记录删除失败`)
      }
    } else if (target.kind === 'contractEvent') {
      await api.delete(`/contracts/${target.contractID}/events/${target.eventID}`)
      if (editingContractEventId.value === target.eventID) resetContractEventForm()
      await loadContractEvents()
    } else if (target.kind === 'locationArea') {
      await api.delete(`/locations/${target.locationID}/areas/${target.areaID}`)
      if (editingLocationAreaId.value === target.areaID) resetLocationAreaEditor()
      await loadLocationAreas()
      await loadRows()
      await refreshBootstrapIfNeeded()
      await ensureRelationCache(['items', 'software', 'invoices', 'contracts', 'files'])
      broadcastEditorDataDirty({ bootstrapKeys: ['locareas'] })
    } else {
      if (!resource.value) return
      const currentResourceKey = resource.value.key
      await api.delete(`${resource.value.endpoint}/${target.id}`)
      await refreshBootstrapIfNeeded()
      await loadRows()
      selectedRowIds.value = selectedRowIds.value.filter((id) => id !== target.id)
      broadcastResourceMutation(currentResourceKey, currentResourceKey === 'locations' ? { bootstrapKeys: ['locareas'] } : undefined)
    }
    confirmOpen.value = false
    deleteTarget.value = null
  } catch (err: unknown) {
    const fallback =
      target.kind === 'contractEvent'
        ? '合同事件删除失败'
        : target.kind === 'locationArea'
        ? '地点区域删除失败'
        : '删除失败'
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? fallback
    confirmOpen.value = false
    deleteTarget.value = null
  } finally {
    deleting.value = false
  }
}

function formatAgentType(value: unknown) {
  const mask = Number(value)
  if (!Number.isFinite(mask) || mask <= 0) return '-'
  const labels: string[] = []
  if ((mask & 4) === 4) labels.push('供应商')
  if ((mask & 2) === 2) labels.push('软件厂商')
  if ((mask & 8) === 8) labels.push('硬件厂商')
  if ((mask & 1) === 1) labels.push('采购方')
  if ((mask & 16) === 16) labels.push('承包方')
  return labels.join(' / ') || '-'
}

function parseAgentTypeBadges(value: unknown): AgentTypeBadge[] {
  const mask = Number(value)
  if (!Number.isFinite(mask) || mask <= 0) return [{ key: 0, label: '-', className: 'agent-type-empty' }]
  const badges: AgentTypeBadge[] = []
  if ((mask & 4) === 4) badges.push({ key: 4, label: '供应商', className: 'agent-type-vendor' })
  if ((mask & 2) === 2) badges.push({ key: 2, label: '软件厂商', className: 'agent-type-software' })
  if ((mask & 8) === 8) badges.push({ key: 8, label: '硬件厂商', className: 'agent-type-hardware' })
  if ((mask & 1) === 1) badges.push({ key: 1, label: '采购方', className: 'agent-type-buyer' })
  if ((mask & 16) === 16) badges.push({ key: 16, label: '承包方', className: 'agent-type-contractor' })
  return badges
}

function formatUserType(value: unknown) {
  const t = Number(value)
  if (t === 0) return '完全访问'
  if (t === 1 || t === 2) return '只读'
  return String(value ?? '-')
}

function formatWarrantyRemain(row: GenericRow) {
  const purchase = Number(row.purchasedate ?? 0)
  const warrantyMonths = Number(row.warrantymonths ?? 0)
  if (!purchase || !warrantyMonths) return '-'

  const start = new Date(purchase * 1000)
  const end = new Date(start.getTime())
  end.setMonth(end.getMonth() + warrantyMonths)

  const now = new Date()
  if (end.getTime() < now.getTime()) return '已过保'

  const diff = diffYMD(now, end)
  const years = diff.years
  const months = diff.months
  const days = diff.days
  if (years > 0) {
    return `${years} 年 ${months} 月, ${days} 天`
  }
  return `${months} 月, ${days} 天`
}

function formatSoftwareQty(row: GenericRow) {
  const used = Number(row.usedqty ?? 0)
  const total = Number(row.licqty ?? 0)
  if (total > 0) return `${used}/${total}`
  if (used > 0) return String(used)
  return '-'
}

function getRackOccupationInfo(row: GenericRow) {
  const occupiedRaw = Number(row.occupation ?? 0)
  const sizeRaw = Number(row.usize ?? 0)
  const occupied = Number.isFinite(occupiedRaw) && occupiedRaw > 0 ? occupiedRaw : 0
  const size = Number.isFinite(sizeRaw) && sizeRaw > 0 ? sizeRaw : 0
  const rawPercent = size > 0 ? (occupied / size) * 100 : 0
  const percent = Math.max(0, Math.min(100, rawPercent))
  return { occupied, size, percent }
}

function getRackOccupationTitle(row: GenericRow) {
  const { occupied, size } = getRackOccupationInfo(row)
  if (size > 0) return `${occupied}U 在用 / 总计 ${size}U`
  return `${occupied}U 在用`
}

function getSoftwareQtyColorClass(row: GenericRow) {
  const used = Number(row.usedqty ?? 0)
  const total = Number(row.licqty ?? 0)
  if (total <= 0) return ''
  if (used > total) return 'software-qty-over'
  if (used < total) return 'software-qty-under'
  return ''
}

function splitLinkedDisplayEntries(rawValue: unknown) {
  const raw = String(rawValue ?? '').trim()
  if (!raw || raw === '-') return [] as string[]
  if (raw.includes('|')) {
    return raw
      .split('|')
      .map((part) => part.trim())
      .filter((part) => part.length > 0)
  }

  const matches = [...raw.matchAll(/\(\d+\)\s*/g)]
  if (matches.length <= 1) return [raw]

  const chunks: string[] = []
  for (let idx = 0; idx < matches.length; idx += 1) {
    const start = matches[idx]?.index ?? 0
    const end = matches[idx + 1]?.index ?? raw.length
    const chunk = raw
      .slice(start, end)
      .trim()
      .replace(/[|,]\s*$/, '')
      .trim()
    if (chunk) chunks.push(chunk)
  }
  return chunks
}

function parseLinkedNumericIDs(rawValue: unknown) {
  return String(rawValue ?? '')
    .split(',')
    .map((part) => Number.parseInt(part.trim(), 10))
    .filter((id) => Number.isFinite(id) && id > 0)
}

function parseSoftwareInstalledEntries(row: GenericRow): SoftwareInstallEntry[] {
  const chunks = splitLinkedDisplayEntries(row.installedon)

  return chunks.map((part, idx) => {
    const match = part.match(/^\((\d+)\)\s*(.*)$/)
    if (!match) {
      return { index: idx + 1, id: null, text: part }
    }
    const id = Number(match[1])
    const text = decodeHtmlEntities(String(match[2] ?? '').trim())
    return {
      index: idx + 1,
      id: Number.isFinite(id) && id > 0 ? id : null,
      text: text || `编号=${id}`,
    }
  })
}

function parseInvoiceFileEntriesValue(rawValue: unknown): InvoiceFileEntry[] {
  return splitLinkedDisplayEntries(rawValue).map((part, idx) => {
    const match = part.match(/^\((\d+)\)\s*(.*)$/)
    if (!match) {
      const plainText = decodeHtmlEntities(part)
      return {
        index: idx + 1,
        id: null,
        text: plainText,
        title: '',
        fileName: plainText,
        previewTip: plainText ? `查看文件: ${plainText}` : '查看文件',
      }
    }
    const id = Number(match[1])
    const rawText = decodeHtmlEntities(String(match[2] ?? '').trim())
    const splitParts = rawText.includes(' / ') ? rawText.split(' / ', 2) : []
    const titleText = String(splitParts[0] ?? '').trim()
    const fileNameText = String(splitParts[1] ?? (splitParts.length > 0 ? '' : rawText)).trim()
    const text = fileNameText || rawText
    const tipText = [titleText, fileNameText || rawText].filter(Boolean).join(' ')
    return {
      index: idx + 1,
      id: Number.isFinite(id) && id > 0 ? id : null,
      text: text || `编号=${id}`,
      title: titleText,
      fileName: fileNameText || rawText,
      previewTip: tipText ? `查看文件: ${tipText}` : `查看文件: ${text || `编号=${id}`}`,
    }
  })
}

function parseInvoiceFileEntries(row: { files?: unknown }): InvoiceFileEntry[] {
  return parseInvoiceFileEntriesValue(row.files)
}

function getSoftwareInvoiceDisplayEntries(row: GenericRow) {
  const invoiceIDs = parseLinkedNumericIDs(row.invoice).sort((a, b) => naturalCompare(a, b))
  if (invoiceIDs.length === 0) return [] as SoftwareInvoiceDisplayEntry[]
  const byID = new Map<number, InvoiceRelationRow>()
  for (const entry of buildInvoiceRelationSource()) byID.set(entry.id, entry)
  return invoiceIDs.map((id) => {
    const invoice = byID.get(id)
    return {
      id,
      number: String(invoice?.number ?? '').trim() || String(id),
      files: invoice ? parseInvoiceFileEntriesValue(invoice.files) : [],
    }
  })
}

function getInstalledItemStatusText(id: number | null) {
  if (!id) return ''
  const item = relationCache.items.find((row) => Number(row.id ?? 0) === id)
  return String(item?.status ?? '').trim()
}

function getInstalledItemStatusClass(id: number | null) {
  const status = getInstalledItemStatusText(id).toLowerCase()
  if (!status) return ''
  if (status.includes('库存') || status.includes('stored')) return 'software-installed-status-stored'
  if (status.includes('故障') || status.includes('defective')) return 'software-installed-status-defective'
  if (status.includes('报废') || status.includes('obsolete')) return 'software-installed-status-obsolete'
  return ''
}

function getInstalledItemStatusTip(id: number | null) {
  const status = getInstalledItemStatusText(id).toLowerCase()
  if (status.includes('库存') || status.includes('stored')) return '状态：库存'
  if (status.includes('故障') || status.includes('defective')) return '状态：有故障'
  if (status.includes('报废') || status.includes('obsolete')) return '状态：报废'
  return '状态：使用中'
}

function openInstalledItem(id: number | null) {
  if (!id) return
  void router.push({ path: '/resources/items', query: { edit: String(id) } })
}

function openResourceEditInCurrentWindow(resourceKey: string, id: number | null) {
  if (!id || id <= 0) return
  void router.push({ path: `/resources/${resourceKey}`, query: { edit: String(id) } })
}

function openResourceEditInNewWindow(resourceKey: string, id: number | null) {
  if (!id || id <= 0) return
  const target = router.resolve({ path: `/resources/${resourceKey}`, query: { edit: String(id) } })
  window.open(target.href, '_blank', 'noopener')
}

function openRackViewInNewWindow(rackID: number | null, highlightID?: number | null) {
  if (!rackID || rackID <= 0) return
  const query: Record<string, string> = {}
  if (highlightID && highlightID > 0) query.highlight = String(highlightID)
  const target = router.resolve({ path: `/rack-view/${rackID}`, query })
  window.open(target.href, '_blank', 'noopener')
}

function openSelectedRackEditor() {
  const rackID = Number(form.rackId ?? 0)
  if (!rackID) return
  openResourceEditInNewWindow('racks', rackID)
}

function openSelectedRackViewInNewWindow() {
  const rackID = Number(form.rackId ?? 0)
  const highlightID = Number(selectedId.value ?? 0)
  openRackViewInNewWindow(rackID, highlightID > 0 ? highlightID : null)
}

function openOverviewEntry(entry: OverviewRow) {
  openResourceEditInNewWindow(entry.resourceKey, entry.id)
}

function openSoftwareRelatedItemInNewWindow(id: number | null) {
  openResourceEditInNewWindow('items', id)
}

function openSoftwareRelatedInvoiceInNewWindow(id: number | null) {
  openResourceEditInNewWindow('invoices', id)
}

function openSoftwareRelatedInvoiceInCurrentWindow(id: number | null) {
  openResourceEditInCurrentWindow('invoices', id)
}

function openSoftwareRelatedContractInNewWindow(id: number | null) {
  openResourceEditInNewWindow('contracts', id)
}

function openSoftwareRelatedFileInNewWindow(id: number | null) {
  openResourceEditInNewWindow('files', id)
}

async function openFilePreviewInNewWindow(id: number | null) {
  if (!id || id <= 0) return
  try {
    const response = await api.get(`/files/${id}/download`, { responseType: 'blob' })
    const blob = response.data instanceof Blob ? response.data : new Blob([response.data])
    if (!blob || blob.size <= 0) {
      noticeStore.error('文件预览失败')
      return
    }
    openBlobInNewWindow(blob)
  } catch (err: unknown) {
    const message = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '文件预览失败'
    error.value = message
    noticeStore.error(message)
  }
}

function downloadLinkedFileByID(id: unknown, fname?: unknown) {
  const fileID = Number(id ?? 0)
  if (!fileID) return
  void downloadFile({ id: fileID, fname: String(fname ?? '') })
}

function addLinkedFileSelection(id: unknown) {
  const fileID = Number(id ?? 0)
  if (!fileID) return
  const current = getMultiValues('fileLinks').map((value) => Number(value))
  current.push(fileID)
  setSelectedFileLinks(current)
  pendingCleanupFileLinks.value = pendingCleanupFileLinks.value.filter((existingID) => existingID !== fileID)
}

function unlinkLinkedFileSelection(id: unknown, scheduleCleanup: boolean) {
  const fileID = Number(id ?? 0)
  if (!fileID) return
  setSelectedFileLinks(getMultiValues('fileLinks').map((value) => Number(value)).filter((value) => value !== fileID))
  if (scheduleCleanup && !pendingCleanupFileLinks.value.includes(fileID)) {
    pendingCleanupFileLinks.value = [...pendingCleanupFileLinks.value, fileID].sort((left, right) => naturalCompare(left, right))
  }
}

function removeLinkedFileSelection(id: unknown) {
  unlinkLinkedFileSelection(id, true)
}

function toggleUploadFileSelection(id: unknown, checked: boolean) {
  if (checked) {
    addLinkedFileSelection(id)
    return
  }
  unlinkLinkedFileSelection(id, false)
}

function openAgentByField(fieldKey: 'vendorId' | 'buyerId' | 'contractorId') {
  const id = Number(form[fieldKey] ?? 0)
  if (!id) return
  openResourceEditInNewWindow('agents', id)
}

function openSoftwareManufacturerByField() {
  const id = Number(form.manufacturerId ?? 0)
  if (!id) return
  openResourceEditInNewWindow('agents', id)
}

function openParentContractEditor() {
  const id = Number(form.parentId ?? 0)
  if (!id) return
  openResourceEditInNewWindow('contracts', id)
}

function getSoftwareItemTypeTooltip(row: ItemRelationRow) {
  const status = String(row.statusText ?? '').trim()
  return status ? `状态：${status}` : '状态：使用中'
}

function getSoftwareRelationIDTip(type: 'item' | 'software' | 'invoice' | 'contract' | 'file', id: number) {
  if (type === 'item') return `在新窗口编辑硬件 ${id}`
  if (type === 'software') return `在新窗口编辑软件 ${id}`
  if (type === 'invoice') return `在新窗口编辑单据 ${id}`
  if (type === 'contract') return `在新窗口编辑合同 ${id}`
  return `在新窗口编辑文件 ${id}`
}

async function refreshSoftwareTags() {
  const id = Number(selectedId.value ?? 0)
  if (!id) {
    softwareTags.value = []
    return
  }
  const { data } = await api.get(`/software/${id}`)
  assignSoftwareTags(data.tags)
}

async function mutateSoftwareTag(name: string, action: 'add' | 'remove') {
  const id = Number(selectedId.value ?? 0)
  if (!id) {
    softwareTagMessage.value = '请先保存软件，再编辑标记'
    return
  }
  const tagName = name.trim()
  if (!tagName) return
  softwareTagSaving.value = true
  softwareTagMessage.value = ''
  try {
    await api.post(`/software/${id}/tags`, { name: tagName, action })
    await refreshBootstrapIfNeeded()
    await Promise.all([refreshSoftwareTags(), loadRows()])
    broadcastEditorDataDirty({ bootstrapKeys: ['tags'] })
    softwareTagMessage.value = action === 'add' ? '标记已添加' : '标记关联已移除'
  } catch (err: unknown) {
    softwareTagMessage.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '标记操作失败'
  } finally {
    softwareTagSaving.value = false
  }
}

async function addSoftwareTag() {
  const name = softwareTagInput.value.trim()
  if (!name) return
  await mutateSoftwareTag(name, 'add')
  if (!softwareTagMessage.value.includes('失败')) {
    softwareTagInput.value = ''
  }
}

async function removeSoftwareTag(name: string) {
  await mutateSoftwareTag(name, 'remove')
}

function toggleSoftwareTagEditor() {
  softwareTagEditorOpen.value = !softwareTagEditorOpen.value
  if (!softwareTagEditorOpen.value) {
    softwareTagInput.value = ''
    softwareTagMessage.value = ''
  }
}

async function refreshItemTags() {
  const id = Number(selectedId.value ?? 0)
  if (!id) {
    itemTags.value = []
    return
  }
  const { data } = await api.get(`/items/${id}`)
  assignItemTags(data.tags)
}

async function mutateItemTag(name: string, action: 'add' | 'remove') {
  const id = Number(selectedId.value ?? 0)
  if (!id) {
    itemTagMessage.value = '请先保存硬件，再编辑标记'
    return
  }
  const tagName = name.trim()
  if (!tagName) return
  itemTagSaving.value = true
  itemTagMessage.value = ''
  try {
    await api.post(`/items/${id}/tags`, { name: tagName, action })
    await refreshBootstrapIfNeeded()
    await Promise.all([refreshItemTags(), loadRows()])
    broadcastEditorDataDirty({ bootstrapKeys: ['tags'] })
    itemTagMessage.value = action === 'add' ? '标记已添加' : '标记关联已移除'
  } catch (err: unknown) {
    itemTagMessage.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '标记操作失败'
  } finally {
    itemTagSaving.value = false
  }
}

async function addItemTag() {
  const name = itemTagInput.value.trim()
  if (!name) return
  await mutateItemTag(name, 'add')
  if (!itemTagMessage.value.includes('失败')) {
    itemTagInput.value = ''
  }
}

async function removeItemTag(name: string) {
  await mutateItemTag(name, 'remove')
}

function toggleItemTagEditor() {
  itemTagEditorOpen.value = !itemTagEditorOpen.value
  if (!itemTagEditorOpen.value) {
    itemTagInput.value = ''
    itemTagMessage.value = ''
  }
}

function onSoftwareUploadFileChange(event: Event) {
  const input = event.target as HTMLInputElement | null
  softwareUploadForm.file = input?.files?.[0] ?? null
}

async function uploadSoftwareFile() {
  await uploadLinkedFile('software')
}

function addContractRenewalRow() {
  contractRenewals.value.push(createDefaultRenewalRow())
}

function removeContractRenewalRow(index: number) {
  if (contractRenewals.value.length <= 1) return
  contractRenewals.value.splice(index, 1)
}

function onItemUploadFileChange(event: Event) {
  const input = event.target as HTMLInputElement | null
  itemUploadForm.file = input?.files?.[0] ?? null
}

function onInvoiceUploadFileChange(event: Event) {
  const input = event.target as HTMLInputElement | null
  const file = input?.files?.[0] ?? null
  const message = getInvoiceUploadFileValidationError(file)
  if (message) {
    invoiceUploadForm.file = null
    if (input) input.value = ''
    error.value = message
    noticeStore.error(message)
    return
  }
  invoiceUploadForm.file = file
}

function onContractUploadFileChange(event: Event) {
  const input = event.target as HTMLInputElement | null
  contractUploadForm.file = input?.files?.[0] ?? null
}

function getLinkedUploadValidationError(scope: LinkedUploadScope) {
  const missing: string[] = []
  if (scope === 'software') {
    if (!softwareUploadForm.title.trim()) missing.push('标题')
    if (!String(softwareUploadForm.typeId ?? '').trim()) missing.push('文件类型')
    if (!softwareUploadForm.date) missing.push('签署日期')
    if (!softwareUploadForm.file) missing.push('选择文件')
  } else if (scope === 'items') {
    if (!itemUploadForm.title.trim()) missing.push('标题')
    if (!String(itemUploadForm.typeId ?? '').trim()) missing.push('文件类型')
    if (!itemUploadForm.date) missing.push('签署日期')
    if (!itemUploadForm.file) missing.push('选择文件')
  } else if (scope === 'invoices') {
    if (!invoiceUploadForm.title.trim()) missing.push('标题')
    if (!invoiceUploadForm.date) missing.push('签署日期')
    if (!invoiceUploadForm.file) missing.push('选择文件')
    const fileTypeError = getInvoiceUploadFileValidationError(invoiceUploadForm.file)
    if (fileTypeError) return fileTypeError
  } else {
    if (!contractUploadForm.title.trim()) missing.push('标题')
    if (!String(contractUploadForm.typeId ?? '').trim()) missing.push('文件类型')
    if (!contractUploadForm.date) missing.push('签署日期')
    if (!contractUploadForm.file) missing.push('选择文件')
  }
  return missing.length > 0 ? `请完善必填项：${missing.join('、')}` : ''
}

async function uploadLinkedFile(scope: LinkedUploadScope) {
  const resourceID = Number(selectedId.value ?? 0)
  if (!resourceID) {
    const message = '请先保存当前记录，再上传文件'
    error.value = message
    noticeStore.error(message)
    return
  }

  const validationError = getLinkedUploadValidationError(scope)
  if (validationError) {
    error.value = validationError
    noticeStore.error(validationError)
    return
  }

  const body = new FormData()
  if (scope === 'software') {
    const file = softwareUploadForm.file as File
    softwareUploading.value = true
    body.append('title', softwareUploadForm.title.trim())
    body.append('typeId', String(softwareUploadForm.typeId))
    body.append('date', softwareUploadForm.date)
    body.append('file', file)
    body.append('softwareLinks', String(resourceID))
  } else if (scope === 'items') {
    const file = itemUploadForm.file as File
    itemUploading.value = true
    body.append('title', itemUploadForm.title.trim())
    body.append('typeId', String(itemUploadForm.typeId))
    body.append('date', itemUploadForm.date)
    body.append('file', file)
    body.append('itemLinks', String(resourceID))
  } else if (scope === 'invoices') {
    const file = invoiceUploadForm.file as File
    invoiceUploading.value = true
    body.append('title', invoiceUploadForm.title.trim())
    body.append('typeId', String(invoiceFileTypeID.value))
    body.append('date', invoiceUploadForm.date)
    body.append('file', file)
    body.append('invoiceLinks', String(resourceID))
  } else {
    const file = contractUploadForm.file as File
    contractUploading.value = true
    body.append('title', contractUploadForm.title.trim())
    body.append('typeId', String(contractUploadForm.typeId))
    body.append('date', contractUploadForm.date)
    body.append('file', file)
    body.append('contractLinks', String(resourceID))
  }

  error.value = ''
  try {
    const { data: created } = await api.post('/files', body)

    if (scope === 'software') {
      softwareUploadForm.title = ''
      softwareUploadForm.typeId = ''
      softwareUploadForm.date = ''
      softwareUploadForm.file = null
      clearLinkedUploadInput('software')
    } else if (scope === 'items') {
      itemUploadForm.title = ''
      itemUploadForm.typeId = ''
      itemUploadForm.date = ''
      itemUploadForm.file = null
      clearLinkedUploadInput('items')
    } else if (scope === 'invoices') {
      invoiceUploadForm.title = ''
      invoiceUploadForm.date = ''
      invoiceUploadForm.file = null
      clearLinkedUploadInput('invoices')
    } else {
      contractUploadForm.title = ''
      contractUploadForm.typeId = ''
      contractUploadForm.date = ''
      contractUploadForm.file = null
      clearLinkedUploadInput('contracts')
    }

    await refreshBootstrapIfNeeded()
    await ensureRelationCache(['items', 'software', 'invoices', 'contracts', 'files'])
    const newFileID = Number((created as { id?: unknown })?.id ?? 0)
    if (newFileID > 0) {
      addLinkedFileSelection(newFileID)
    }
    if (resourceID > 0) {
      void recordRecentViewHistory(resourceID)
    }
    const endpoint =
      scope === 'software'
        ? '/software'
        : scope === 'items'
          ? '/items'
          : scope === 'invoices'
            ? '/invoices'
            : '/contracts'
    const { data } = await api.get(`${endpoint}/${resourceID}`)
    recordDetail.value = data as GenericRow
    form.fileLinks = Array.isArray(data.fileLinks) ? data.fileLinks.map((fid: unknown) => String(Number(fid))) : []
    fileLinkFilter.value = ''
    await loadRows()
    broadcastEditorDataDirty({ relationKeys: ['files'], bootstrapKeys: ['files_ref'] })
    const successMessage =
      scope === 'software'
        ? '文件已上传并关联到当前软件'
        : scope === 'items'
          ? '文件已上传并关联到当前硬件'
          : scope === 'invoices'
            ? '文件已上传并关联到当前单据'
            : '文件已上传并关联到当前合同'
    noticeStore.success(successMessage)
  } catch (err: unknown) {
    const message = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '上传文件失败'
    error.value = message
    noticeStore.error(message)
  } finally {
    softwareUploading.value = false
    itemUploading.value = false
    invoiceUploading.value = false
    contractUploading.value = false
  }
}

function formatTableValue(value: unknown, key: string, row: GenericRow) {
  if (key === 'warrantyRemain') return formatWarrantyRemain(row)
  if (key === 'qty') return formatSoftwareQty(row)
  if (resource.value?.key === 'software' && key === 'tags') {
    return formatSortedTagText(value) || '-'
  }

  if (resource.value?.key === 'agents' && key === 'type') return formatAgentType(value)
  if (resource.value?.key === 'users' && key === 'usertype') return formatUserType(value)

  if (resource.value?.key === 'racks' && key === 'usize') {
    if (value === null || value === undefined || value === '') return '-'
    return `${value}U`
  }

  if (resource.value?.key === 'racks' && key === 'depth') {
    if (value === null || value === undefined || value === '') return '-'
    return `${value}mm`
  }

  if (value === null || value === undefined || value === '') return '-'
  const lowerKey = key.toLowerCase()
  const likelyDate =
    lowerKey.includes('date') || lowerKey.includes('uploaddate') || lowerKey.includes('purch') || lowerKey.includes('maintend')
  if (likelyDate) {
    const n = Number(value)
    if (!Number.isNaN(n) && n > 0) return normalizeDateInput(n)
  }
  return decodeHtmlEntities(String(value))
}

function getCellTitle(key: string, row: GenericRow) {
  if (resource.value?.key === 'racks' && key === 'occupation') {
    return getRackOccupationTitle(row)
  }
  if (resource.value?.key === 'items' && (key === 'purchasedate' || key === 'purchaseDate')) {
    const timestamp = toLocalDateEightAMTimestamp(row.purchasedate ?? row.purchaseDate ?? '')
    return timestamp > 0 ? String(timestamp) : undefined
  }
  if (resource.value?.key === 'items' && key === 'warrantyRemain') {
    const days = getWarrantyRemainDays(row)
    return Number.isFinite(days) ? String(days) : undefined
  }
  return undefined
}

function diffYMD(from: Date, to: Date) {
  const start = new Date(from.getTime())
  const end = new Date(to.getTime())
  const years = end.getFullYear() - start.getFullYear()
  const shiftedByYears = new Date(start.getTime())
  shiftedByYears.setFullYear(shiftedByYears.getFullYear() + years)

  let adjYears = years
  let cursor = shiftedByYears
  if (cursor.getTime() > end.getTime()) {
    adjYears -= 1
    cursor = new Date(start.getTime())
    cursor.setFullYear(cursor.getFullYear() + adjYears)
  }

  let months = (end.getFullYear() - cursor.getFullYear()) * 12 + (end.getMonth() - cursor.getMonth())
  const shiftedByMonths = new Date(cursor.getTime())
  shiftedByMonths.setMonth(shiftedByMonths.getMonth() + months)
  if (shiftedByMonths.getTime() > end.getTime()) {
    months -= 1
  }

  cursor = new Date(cursor.getTime())
  cursor.setMonth(cursor.getMonth() + months)
  const oneDay = 24 * 60 * 60 * 1000
  const days = Math.max(0, Math.floor((end.getTime() - cursor.getTime()) / oneDay))

  return { years: Math.max(0, adjYears), months: Math.max(0, months), days }
}

function getCellClass(key: string, row: GenericRow) {
  const classes: string[] = []

  if (key === 'warrantyRemain') {
    const text = formatWarrantyRemain(row)
    if (text !== '-' && text !== '') {
      classes.push(text === '已过保' ? 'warranty-expired' : 'warranty-good')
    }
  }

  if (resource.value?.key === 'software' && key === 'qty') {
    const qtyClass = getSoftwareQtyColorClass(row)
    if (qtyClass) classes.push(qtyClass)
  }

  if (resource.value?.key === 'software' && key === 'installedon') {
    classes.push('software-installed-cell')
  }

  if (resource.value?.key === 'agents' && key === 'contactinfo') {
    classes.push('agent-contactinfo-cell')
  }

  if (resource.value?.key === 'files' && key === 'links' && Number(row.links ?? 0) === 0) {
    classes.push('file-links-zero-cell')
  }

  return classes.join(' ')
}

function getWarrantyRemainDays(row: GenericRow) {
  const purchase = Number(row.purchasedate ?? 0)
  const warrantyMonths = Number(row.warrantymonths ?? 0)
  if (!purchase || !warrantyMonths) return Number.NEGATIVE_INFINITY

  const start = new Date(purchase * 1000)
  const end = new Date(start.getTime())
  end.setMonth(end.getMonth() + warrantyMonths)
  const oneDay = 24 * 60 * 60 * 1000
  const diffDays = (end.getTime() - Date.now()) / oneDay
  return diffDays >= 0 ? Math.floor(diffDays) : Math.ceil(diffDays)
}

function getSortValue(row: GenericRow, key: string) {
  if (key === 'warrantyRemain') return getWarrantyRemainDays(row)
  if (key === 'qty') {
    const used = Number(row.usedqty ?? 0)
    const total = Number(row.licqty ?? 0)
    return total > 0 ? used / total : used
  }
  if (resource.value?.key === 'software' && key === 'tags') {
    return formatSortedTagText(row.tags)
  }

  const value = row[key]
  if (value === null || value === undefined) return ''

  const lowerKey = key.toLowerCase()
  if (
    lowerKey.includes('date') ||
    lowerKey.includes('uploaddate') ||
    lowerKey.includes('purch') ||
    lowerKey.includes('maintend')
  ) {
    const num = Number(value)
    if (!Number.isNaN(num) && num > 0) return num

    const text = String(value).trim()
    if (/^\d{4}-\d{2}-\d{2}$/.test(text)) {
      return new Date(text).getTime() / 1000
    }
  }

  return value
}

function getUploadFileUnlinkTip() {
  const resourceLabel =
    resource.value?.key === 'items'
      ? '硬件'
      : resource.value?.key === 'software'
        ? '软件'
        : resource.value?.key === 'invoices'
          ? '单据'
          : resource.value?.key === 'contracts'
            ? '合同'
            : '当前记录'
  return `解除关联，保存${resourceLabel}后生效。若文件是孤立的(没有其他内容与之相关联)，也不会被删除`
}

const uploadTabTip = reactive({
  visible: false,
  text: '',
  left: 0,
  top: 0,
  placement: 'below' as 'below' | 'above',
})

function showUploadTabTip(event: Event, text: string) {
  const target = event.currentTarget as HTMLElement | null
  if (!target || !text) return

  const rect = target.getBoundingClientRect()
  const viewportPadding = 12
  const estimatedLines = Math.max(1, Math.ceil(text.length / 22))
  const estimatedHeight = estimatedLines * 18 + 18
  const tooltipWidth = Math.min(260, Math.max(180, window.innerWidth - viewportPadding * 2))
  const halfWidth = tooltipWidth / 2
  const shouldShowAbove = rect.bottom + estimatedHeight + 16 > window.innerHeight && rect.top > estimatedHeight + 16

  let left = rect.left + rect.width / 2
  left = Math.max(viewportPadding + halfWidth, Math.min(window.innerWidth - viewportPadding - halfWidth, left))

  uploadTabTip.visible = true
  uploadTabTip.text = text
  uploadTabTip.left = left
  uploadTabTip.top = shouldShowAbove ? rect.top - estimatedHeight - 12 : rect.bottom + 10
  uploadTabTip.placement = shouldShowAbove ? 'above' : 'below'
}

function hideUploadTabTip() {
  uploadTabTip.visible = false
}

const { sortKey, sortedRows, setSort, toggleSort: toggleSortInner, getSortIcon } = useTableSort(rows, {
  primaryKey: primarySortKey.value,
  primaryDefaultDirection: 'desc',
  getSortValue: (row, key) => getSortValue(row as GenericRow, key),
})

function getPrimaryDefaultDirection(): 'asc' | 'desc' {
  const key = resource.value?.key ?? ''
  return ascPrimaryResourceKeys.has(key) ? 'asc' : 'desc'
}

function getDefaultSortState() {
  const first = resource.value?.columns[0]?.key ?? ''
  return { key: first, direction: getPrimaryDefaultDirection() }
}

function toggleSort(key: string) {
  if (sortKey.value !== key && key === primarySortKey.value) {
    setSort(key, getPrimaryDefaultDirection())
    return
  }
  toggleSortInner(key)
}

const pageOptions = computed(() => {
  const key = resource.value?.key ?? ''
  return resourcePageConfig[key]?.sizeOptions ?? [10, 25, 50, 100, -1]
})

const totalRows = computed(() => sortedRows.value.length)
const totalPages = computed(() => {
  if (pageSize.value === -1) return 1
  if (pageSize.value <= 0) return 1
  return Math.max(1, Math.ceil(totalRows.value / pageSize.value))
})

const pagedRows = computed(() => {
  if (pageSize.value === -1) return sortedRows.value
  const start = (page.value - 1) * pageSize.value
  return sortedRows.value.slice(start, start + pageSize.value)
})

const selectedRowIdSet = computed(() => new Set(selectedRowIds.value))
const selectableCurrentRows = computed(() => sortedRows.value.filter((row) => canDeleteMainRow(row as GenericRow)))
const selectablePageRows = computed(() => pagedRows.value.filter((row) => canDeleteMainRow(row as GenericRow)))
const selectedRowCount = computed(() => selectedRowIds.value.length)
const allPageRowsSelected = computed(() => {
  if (selectablePageRows.value.length === 0) return false
  return selectablePageRows.value.every((row) => selectedRowIdSet.value.has(rowKey(row as GenericRow)))
})
const allCurrentRowsSelected = computed(() => {
  if (selectableCurrentRows.value.length === 0) return false
  return selectableCurrentRows.value.every((row) => selectedRowIdSet.value.has(rowKey(row as GenericRow)))
})

const pageStart = computed(() => {
  if (totalRows.value === 0) return 0
  if (pageSize.value === -1) return 1
  return (page.value - 1) * pageSize.value + 1
})

const pageEnd = computed(() => {
  if (totalRows.value === 0) return 0
  if (pageSize.value === -1) return totalRows.value
  return Math.min(page.value * pageSize.value, totalRows.value)
})

const visiblePages = computed(() => {
  const pages: number[] = []
  const maxVisible = 5
  const total = totalPages.value
  const start = Math.max(1, Math.min(page.value - Math.floor(maxVisible / 2), total - maxVisible + 1))
  const end = Math.min(total, start + maxVisible - 1)
  for (let p = start; p <= end; p += 1) pages.push(p)
  return pages
})

function setPage(next: number) {
  if (totalPages.value <= 0) {
    page.value = 1
    return
  }
  page.value = Math.min(Math.max(1, next), totalPages.value)
}

function onPageSizeChange() {
  page.value = 1
}

function syncSelectedRowsWithVisibleRows() {
  const visibleIds = new Set(
    sortedRows.value
      .map((row) => rowKey(row as GenericRow))
      .filter((id) => Number.isFinite(id) && id > 0),
  )
  selectedRowIds.value = selectedRowIds.value.filter((id) => visibleIds.has(id))
}

function isRowSelected(row: GenericRow) {
  return selectedRowIdSet.value.has(rowKey(row))
}

function updateRowSelection(id: number, checked: boolean) {
  if (!id) return
  if (checked) {
    if (!selectedRowIdSet.value.has(id)) {
      selectedRowIds.value = [...selectedRowIds.value, id]
    }
    return
  }
  selectedRowIds.value = selectedRowIds.value.filter((entry) => entry !== id)
}

function toggleRowSelection(row: GenericRow, event: Event) {
  if (!canDeleteMainRow(row)) return
  const checked = (event.target as HTMLInputElement | null)?.checked ?? false
  updateRowSelection(rowKey(row), checked)
}

function toggleCurrentPageSelection(event: Event) {
  const checked = (event.target as HTMLInputElement | null)?.checked ?? false
  const pageIds = selectablePageRows.value.map((row) => rowKey(row as GenericRow))
  if (checked) {
    selectedRowIds.value = Array.from(new Set([...selectedRowIds.value, ...pageIds]))
    return
  }
  const pageIdSet = new Set(pageIds)
  selectedRowIds.value = selectedRowIds.value.filter((id) => !pageIdSet.has(id))
}

function setAllCurrentRowsSelected(checked: boolean) {
  const currentIds = selectableCurrentRows.value.map((row) => rowKey(row as GenericRow))
  if (checked) {
    selectedRowIds.value = Array.from(new Set([...selectedRowIds.value, ...currentIds]))
    return
  }
  const currentIdSet = new Set(currentIds)
  selectedRowIds.value = selectedRowIds.value.filter((id) => !currentIdSet.has(id))
}

function clearSelectedRows() {
  selectedRowIds.value = []
}

function requestRemoveSelectedRows() {
  if (!resource.value || !canWrite.value || selectedRowCount.value === 0) return
  openDeleteConfirm({ kind: 'rows', ids: [...selectedRowIds.value] })
}

async function downloadFile(row: GenericRow) {
  const id = Number(row.id ?? 0)
  if (!id) return
  try {
    const response = await api.get(`/files/${id}/download`, { responseType: 'blob' })
    const disposition = response.headers['content-disposition'] ?? ''
    const fileName = parseDownloadFilename(disposition) || String(row.fname ?? `file-${id}`)

    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    link.download = fileName
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(url)
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '文件下载失败'
  }
}

function parseDownloadFilename(contentDisposition: string) {
  const utf8Matched = /filename\*\s*=\s*UTF-8''([^;]+)/i.exec(contentDisposition)
  if (utf8Matched?.[1]) {
    try {
      return decodeURIComponent(utf8Matched[1].trim())
    } catch {
      // ignore malformed header and fallback below
    }
  }

  const quotedMatched = /filename\s*=\s*"([^"]+)"/i.exec(contentDisposition)
  if (quotedMatched?.[1]) return quotedMatched[1].trim()

  const plainMatched = /filename\s*=\s*([^;]+)/i.exec(contentDisposition)
  if (plainMatched?.[1]) return plainMatched[1].trim()

  return ''
}

watch(
  resource,
  async () => {
    closeDeleteConfirm()
    clearSelectedRows()
    resetForm()
    resetItemEditorState()
    resetNonItemEditorState()
    const sortState = getDefaultSortState()
    if (sortState.key) {
      setSort(sortState.key, sortState.direction)
    }
    const cfg = resourcePageConfig[resource.value?.key ?? '']
    pageSize.value = cfg?.defaultSize ?? 25
    page.value = 1
    await loadRows()
    if (resource.value && ['items', 'software', 'invoices', 'contracts', 'files', 'agents', 'locations', 'users', 'racks'].includes(resource.value.key)) {
      void ensureRelationCache(['items', 'software', 'invoices', 'contracts', 'files'])
    }
    initializeItemDefaults()
    await applyRouteQueryActions()
  },
  { immediate: true },
)

watch(
  () => [route.query.edit, route.query.create] as const,
  async () => {
    if (!resource.value) return
    const hasEdit = getEditQueryID() > 0
    const hasCreate = isCreateQueryEnabled()
    if (!hasEdit && !hasCreate) return
    await applyRouteQueryActions()
  },
)

watch(sortedRows, () => {
  syncSelectedRowsWithVisibleRows()
})

watch(
  () => [
    resource.value?.key ?? '',
    String(form.typeId ?? ''),
    String(form.locationId ?? ''),
    String(form.locAreaId ?? ''),
    String(form.vendorId ?? ''),
    String(form.buyerId ?? ''),
    String(form.contractorId ?? ''),
    String(form.manufacturerId ?? ''),
  ],
  () => {
    syncDependentSelections()
  },
)

watch(
  () => [resource.value?.key ?? '', String(form.rackId ?? '')] as const,
  ([resourceKey, rackID], [, previousRackID]) => {
    if (resourceKey !== 'items' || !drawerOpen.value) return
    if (rackID === previousRackID) return
    form.rackPosition = ''
  },
  { flush: 'sync' },
)

watch(
  () => [resource.value?.key ?? '', selectedItemRackTotalUnits.value] as const,
  ([resourceKey, totalUnits]) => {
    if (resourceKey !== 'items') return
    const rackPosition = Number(form.rackPosition ?? 0)
    if (!Number.isFinite(rackPosition) || rackPosition <= 0) return
    if (Number(totalUnits ?? 0) > 0 && rackPosition > Number(totalUnits)) {
      form.rackPosition = ''
    }
  },
)

watch(
  () => [resource.value?.key, selectedId.value, locationFloorplanName.value, drawerOpen.value] as const,
  () => {
    void loadLocationFloorplanPreview()
  },
)

watch([totalRows, totalPages], () => {
  if (page.value > totalPages.value) {
    page.value = totalPages.value
  }
  if (totalRows.value === 0) {
    page.value = 1
  }
})

onMounted(() => {
  resetForm()
  setupEditorDataDirtySync()
  window.addEventListener('scroll', hideUploadTabTip, true)
  window.addEventListener('resize', hideUploadTabTip)
})

onBeforeUnmount(() => {
  revokeLocationFloorplanPreviewURL()
  teardownEditorDataDirtySync()
  window.removeEventListener('scroll', hideUploadTabTip, true)
  window.removeEventListener('resize', hideUploadTabTip)
})
</script>

<template>
  <section class="page-shell" v-if="!readonlyResource && resource">
    <header class="page-header">
      <h2>{{ resource.title }}</h2>
      <div class="header-actions">
        <div class="search-inline">
          <span class="search-label">查询</span>
          <input
            :value="search"
            class="search-input"
            placeholder="输入关键字实时搜索"
            @input="onSearchInput"
            @compositionupdate="onSearchInput"
            @compositionend="onSearchInput"
          />
        </div>
        <button class="ghost-btn" @click="loadRows">刷新</button>
        <button
          v-if="canWrite"
          class="ghost-btn"
          :disabled="totalRows === 0 || allCurrentRowsSelected"
          @click="setAllCurrentRowsSelected(true)"
        >
          全选当前结果
        </button>
        <button v-if="canWrite" class="ghost-btn" :disabled="selectedRowCount === 0" @click="clearSelectedRows">清空选择</button>
        <button v-if="canWrite" class="danger" :disabled="selectedRowCount === 0" @click="requestRemoveSelectedRows">
          批量删除（{{ selectedRowCount }}）
        </button>
        <button v-if="canWrite" @click="openCreate">新增</button>
      </div>
    </header>

    <p v-if="loading">加载中...</p>

    <div class="table-toolbar" v-if="!loading">
      <div class="length-control">
        <span>每页</span>
        <select v-model.number="pageSize" @change="onPageSizeChange">
          <option v-for="size in pageOptions" :key="size" :value="size">
            {{ size === -1 ? '全部' : size }}
          </option>
        </select>
        <span>条</span>
      </div>
      <div class="table-meta">显示 {{ pageStart }} - {{ pageEnd }}，共 {{ totalRows }} 条</div>
      <div v-if="canWrite" class="table-selection-meta">已选 {{ selectedRowCount }} 条</div>
    </div>

    <div class="table-wrap" v-if="!loading">
      <table :class="['resource-table', `resource-table-${resource.key}`]">
        <thead>
          <tr>
            <th v-if="canWrite" class="selection-col">
              <input
                type="checkbox"
                :checked="allPageRowsSelected"
                :disabled="selectablePageRows.length === 0"
                @change="toggleCurrentPageSelection"
              />
            </th>
            <th v-if="canWrite">{{ actionHeaderText }}</th>
            <th
              v-for="col in resource.columns"
              :key="col.key"
              :class="['sortable-th', { 'is-active': sortKey === col.key }]"
            >
              <button
                type="button"
                class="th-sort-btn"
                :class="{ 'quick-tip': !!col.tooltip, 'quick-tip-below': !!col.tooltip }"
                :data-quick-tip="col.tooltip || undefined"
                :title="col.tooltip || undefined"
                @click="toggleSort(col.key)"
              >
                <span>{{ col.label }}</span>
                <span class="th-sort-icon">{{ getSortIcon(col.key) }}</span>
              </button>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in pagedRows" :key="rowKey(row)">
            <td v-if="canWrite" class="selection-col">
              <input
                type="checkbox"
                :checked="isRowSelected(row)"
                :disabled="!canDeleteMainRow(row)"
                @change="toggleRowSelection(row, $event)"
              />
            </td>
            <td v-if="canWrite" class="actions-cell">
              <div class="row-actions">
                <button v-if="canWrite" class="small-btn" @click="openEdit(row)">编辑</button>
                <button v-if="canWrite" class="small-btn danger" @click="remove(rowKey(row))">删除</button>
              </div>
            </td>
            <td
              v-for="col in resource.columns"
              :key="col.key"
              :class="[getCellClass(col.key, row), { 'quick-tip': Boolean(getCellTitle(col.key, row)) }]"
              :data-quick-tip="getCellTitle(col.key, row) || undefined"
            >
              <template v-if="resource.key === 'items' && col.key === 'id'">
                <span class="item-id-status-badge" :style="getItemIDBadgeStyle(row)">
                  {{ formatTableValue(row[col.key], col.key, row) }}
                </span>
              </template>
              <template v-else-if="resource.key === 'software' && col.key === 'invoice'">
                <div v-if="getSoftwareInvoiceDisplayEntries(row).length > 0" class="software-invoice-display-list">
                  <div
                    v-for="entry in getSoftwareInvoiceDisplayEntries(row)"
                    :key="`software-invoice-${rowKey(row)}-${entry.id}`"
                    class="software-invoice-display-entry"
                  >
                    <button
                      type="button"
                      class="relation-jump-btn quick-tip"
                      :data-quick-tip="`编辑单据 ${entry.id}`"
                      @click="openSoftwareRelatedInvoiceInCurrentWindow(entry.id)"
                    >
                      {{ entry.id }}
                    </button>
                    <div v-if="entry.files.length > 0" class="software-invoice-file-actions">
                      <button
                        v-for="file in entry.files"
                        :key="`software-invoice-file-${rowKey(row)}-${entry.id}-${file.index}-${file.id ?? 'na'}`"
                        type="button"
                        class="field-link-icon quick-tip software-invoice-file-btn"
                        :data-quick-tip="file.previewTip"
                        :disabled="!file.id"
                        @click="file.id ? openFilePreviewInNewWindow(file.id) : undefined"
                      >
                        <img class="field-link-icon-image" src="/images/down.png" :alt="file.previewTip" />
                      </button>
                    </div>
                  </div>
                </div>
                <span v-else>-</span>
              </template>
              <template v-else-if="resource.key === 'software' && col.key === 'installedon'">
                <div v-if="parseSoftwareInstalledEntries(row).length > 0" class="software-installed-list">
                  <div
                    v-for="entry in parseSoftwareInstalledEntries(row)"
                    :key="`installed-${rowKey(row)}-${entry.index}-${entry.id ?? 'na'}`"
                    class="software-installed-entry"
                  >
                    <a
                      v-if="entry.id"
                      href="#"
                      class="software-installed-link"
                      @click.prevent="openInstalledItem(entry.id)"
                    >
                      <span>{{ entry.index }}：</span>
                      <span
                        :class="['software-installed-id', getInstalledItemStatusClass(entry.id), 'quick-tip']"
                        :data-quick-tip="getInstalledItemStatusTip(entry.id)"
                      >
                        ({{ entry.id }})
                      </span>
                      <span>{{ entry.text }}</span>
                    </a>
                    <span v-else>{{ entry.index }}：{{ entry.text }}</span>
                  </div>
                </div>
                <span v-else>-</span>
              </template>
              <template v-else-if="resource.key === 'invoices' && col.key === 'files'">
                <div v-if="parseInvoiceFileEntries(row).length > 0" class="software-installed-list">
                  <div
                    v-for="entry in parseInvoiceFileEntries(row)"
                    :key="`invoice-file-${rowKey(row)}-${entry.index}-${entry.id ?? 'na'}`"
                    class="software-installed-entry"
                  >
                    <a
                      v-if="entry.id"
                      href="#"
                      class="software-installed-link quick-tip"
                      :data-quick-tip="entry.title || '-'"
                      @click.prevent="openFilePreviewInNewWindow(entry.id)"
                    >
                      <span>{{ entry.index }}：</span>
                      <span class="software-installed-id">({{ entry.id }})</span>
                      <span>{{ entry.text }}</span>
                    </a>
                    <span v-else>{{ entry.index }}：{{ entry.text }}</span>
                  </div>
                </div>
                <span v-else>-</span>
              </template>
              <template v-else-if="resource.key === 'agents' && col.key === 'type'">
                <div class="agent-type-list">
                  <span
                    v-for="badge in parseAgentTypeBadges(row.type)"
                    :key="`agent-type-${rowKey(row)}-${badge.key}`"
                    :class="['agent-type-badge', badge.className]"
                  >
                    {{ badge.label }}
                  </span>
                </div>
              </template>
              <template v-else-if="resource.key === 'agents' && col.key === 'contacts'">
                <div v-if="getAgentContactSummaryLines(row.contacts).length > 0" class="agent-contact-summary">
                  <div
                    v-for="(line, idx) in getAgentContactSummaryLines(row.contacts)"
                    :key="`agent-contact-${rowKey(row)}-${idx}`"
                    class="agent-contact-summary-line"
                  >
                    {{ line }}
                  </div>
                </div>
                <span v-else class="agent-contact-empty">-</span>
              </template>
              <template v-else-if="resource.key === 'racks' && col.key === 'occupation'">
                <div class="rack-occupation-cell">
                  <div class="rack-usage-track">
                    <div class="rack-usage-fill" :style="{ width: `${getRackOccupationInfo(row).percent}%` }" />
                  </div>
                  <span class="rack-usage-text">
                    {{ getRackOccupationInfo(row).occupied }}U / {{ getRackOccupationInfo(row).size || '-' }}U
                  </span>
                </div>
              </template>
              <template v-else-if="resource.key === 'locations' && col.key === 'floorplanfn'">
                <button
                  v-if="String(row.floorplanfn ?? '').trim()"
                  type="button"
                  class="location-floorplan-table-link quick-tip"
                  data-quick-tip="在新窗口查看建筑平面图"
                  :aria-label="`查看建筑平面图：${getLocationFloorplanDisplayName(row.floorplanfn)}`"
                  @click="openLocationFloorplanByRow(row)"
                >
                  <span class="location-floorplan-table-link-icon">↗</span>
                  <span class="location-floorplan-table-link-text">{{ getLocationFloorplanDisplayName(row.floorplanfn) }}</span>
                </button>
                <span v-else>-</span>
              </template>
              <template v-else-if="resource.key === 'files' && col.key === 'fname'">
                <button
                  v-if="String(row.fname ?? '').trim()"
                  type="button"
                  class="location-floorplan-table-link quick-tip"
                  data-quick-tip="点击下载文件"
                  :aria-label="`下载文件：${getFileDisplayName(row.fname)}`"
                  @click="downloadFile(row)"
                >
                  <span class="location-floorplan-table-link-icon">↓</span>
                  <span class="location-floorplan-table-link-text">{{ getFileDisplayName(row.fname) }}</span>
                </button>
                <span v-else>-</span>
              </template>
              <template v-else>
                {{ formatTableValue(row[col.key], col.key, row) }}
              </template>
            </td>
          </tr>
          <tr v-if="totalRows === 0">
            <td :colspan="resource.columns.length + (canWrite ? 2 : 0)">暂无数据</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="table-pagination" v-if="!loading && totalRows > 0 && pageSize !== -1">
      <button class="ghost-btn small-btn" :disabled="page <= 1" @click="setPage(1)">首页</button>
      <button class="ghost-btn small-btn" :disabled="page <= 1" @click="setPage(page - 1)">上一页</button>
      <button
        v-for="p in visiblePages"
        :key="`page-${p}`"
        class="small-btn"
        :class="p === page ? 'page-btn-active' : 'ghost-btn'"
        @click="setPage(p)"
      >
        {{ p }}
      </button>
      <button class="ghost-btn small-btn" :disabled="page >= totalPages" @click="setPage(page + 1)">下一页</button>
      <button class="ghost-btn small-btn" :disabled="page >= totalPages" @click="setPage(totalPages)">末页</button>
    </div>

    <div v-if="drawerOpen" class="drawer-mask">
      <aside
        class="drawer"
        :class="{
          'item-drawer': isItemResource,
          'wide-drawer': isSoftwareResource || isInvoiceResource || isContractResource || isFileResource || isAgentResource || isLocationResource || isUserResource || isRackResource,
        }"
        role="dialog"
        aria-modal="true"
      >
        <div class="drawer-header">
          <h3>{{ selectedId ? `编辑 编号=${selectedId}` : '新增记录' }}{{ isItemResource ? ' - 硬件' : '' }}</h3>
          <button
            class="drawer-close-btn quick-tip"
            type="button"
            aria-label="关闭"
            data-quick-tip="关闭"
            @click="drawerOpen = false"
          >
            ×
          </button>
        </div>
        <form
          class="drawer-form"
          :class="{
            'drawer-form-item': isItemResource,
            'drawer-form-complex': isSoftwareResource || isInvoiceResource || isContractResource || isFileResource || isAgentResource || isLocationResource || isUserResource || isRackResource,
            'drawer-form-software': isSoftwareResource,
            'drawer-form-tabbed': isItemResource || isInvoiceResource || isContractResource || isFileResource,
          }"
          @submit.prevent="save"
        >
        <template v-if="isItemResource">
          <div class="item-tabs">
            <button
              v-for="tab in itemEditorTabs"
              :key="tab.key"
              type="button"
              class="item-tab-btn"
              :class="{ active: activeItemTab === tab.key }"
              @mouseenter="tab.key === 'files' ? showUploadTabTip($event, getUploadFileUnlinkTip()) : undefined"
              @mouseleave="hideUploadTabTip"
              @focus="tab.key === 'files' ? showUploadTabTip($event, getUploadFileUnlinkTip()) : undefined"
              @blur="hideUploadTabTip"
              @click="activeItemTab = tab.key; hideUploadTabTip()"
            >
              {{ tab.label }}
            </button>
          </div>

          <section v-show="activeItemTab === 'itemData'" class="item-tab-pane">
            <div class="item-layout hardware-item-layout">
              <section class="item-block">
                <h4>内在特性</h4>
                <label>
                  <span>硬件类型 <sup class="req">*</sup></span>
                  <span class="quick-tip field-select-tip" data-quick-tip="根据硬件类型分组">
                    <select v-model="form.itemTypeId">
                      <option value="">请选择</option>
                      <option v-for="opt in getOptionsByFieldKey('itemTypeId')" :key="`itemType-${opt.value}`" :value="String(opt.value)">
                        {{ opt.label }}
                      </option>
                    </select>
                  </span>
                </label>

                <div class="item-radio-row">
                  <span>从属部件 <sup class="req">*</sup></span>
                  <label><input v-model.number="form.isPart" type="radio" :value="1" /> 是</label>
                  <label><input v-model.number="form.isPart" type="radio" :value="0" /> 否</label>
                </div>

                <div class="item-radio-row">
                  <span>机架式 <sup class="req">*</sup></span>
                  <label><input v-model.number="form.rackMountable" type="radio" :value="1" /> 是</label>
                  <label><input v-model.number="form.rackMountable" type="radio" :value="0" /> 否</label>
                </div>

                <label>
                  <span>厂商 <sup class="req">*</sup></span>
                  <span class="quick-tip field-select-tip" data-quick-tip="根据代理菜单中定义的硬件厂商分组">
                    <select v-model="form.manufacturerId">
                      <option value="">请选择</option>
                      <option v-for="opt in getOptionsByFieldKey('manufacturerId')" :key="`manufacturer-${opt.value}`" :value="String(opt.value)">
                        {{ opt.label }}
                      </option>
                    </select>
                  </span>
                </label>
                <label><span>型号 <sup class="req">*</sup></span><input v-model="form.model" type="text" /></label>
                <label>
                  <span>大小(U)</span>
                  <select v-model="form.uSize">
                    <option value="">请选择</option>
                    <option v-for="i in 44" :key="`usize-${i}`" :value="String(i)">{{ i }}</option>
                  </select>
                </label>
                <label><span>设备序列号</span><input v-model="form.sn" type="text" /></label>
                <label><span>序列号2</span><input v-model="form.sn2" type="text" /></label>
                <label><span>Service Tag</span><input v-model="form.sn3" type="text" /></label>
                <label><span>注释</span><textarea v-model="form.comments" /></label>
                <label>
                  <span>标签</span>
                  <span class="quick-tip field-select-tip" data-quick-tip="在可打印表格上也显示此文本">
                    <input v-model="form.label" type="text" />
                  </span>
                </label>
              </section>

              <section class="item-block">
                <h4>使用</h4>
                <label>
                  <span>状态 <sup class="req">*</sup></span>
                  <select v-model="form.status">
                    <option value="">请选择</option>
                    <option v-for="opt in getOptionsByFieldKey('status')" :key="`status-${opt.value}`" :value="String(opt.value)">
                      {{ opt.label }}
                    </option>
                  </select>
                </label>

                <label>
                  <span>所属部门</span>
                  <select v-model="form.dptId">
                    <option value="">请选择</option>
                    <option v-for="opt in getOptionsByFieldKey('dptId')" :key="`dpt-${opt.value}`" :value="String(opt.value)">
                      {{ opt.label }}
                    </option>
                  </select>
                </label>
                <label><span>负责人 <sup class="req">*</sup></span><input v-model="form.principal" type="text" /></label>
                <label>
                  <span>地点</span>
                  <select v-model="form.locationId">
                    <option value="">请选择</option>
                    <option v-for="opt in getOptionsByFieldKey('locationId')" :key="`location-${opt.value}`" :value="String(opt.value)">
                      {{ opt.label }}
                    </option>
                  </select>
                </label>
                <label>
                  <span>区域/房间</span>
                  <select v-model="form.locAreaId">
                    <option value="">{{ itemLocAreaPlaceholderText }}</option>
                    <option v-for="opt in itemLocAreaOptions" :key="`locArea-${opt.value}`" :value="String(opt.value)">
                      {{ opt.label }}
                    </option>
                  </select>
                </label>
                <div class="item-rack-field">
                  <div class="item-rack-select-label">
                    <span class="item-rack-select-actions">
                      <button
                        class="field-link-icon quick-tip"
                        type="button"
                        data-quick-tip="查看机架晟图"
                        :disabled="!Number(form.rackId ?? 0)"
                        @click="openSelectedRackViewInNewWindow"
                      >
                        <img class="field-link-icon-image" src="/images/eye.png" alt="查看机架晟图" />
                      </button>
                      <button
                        class="field-link-icon quick-tip"
                        type="button"
                        data-quick-tip="编辑机架"
                        :disabled="!Number(form.rackId ?? 0)"
                        @click="openSelectedRackEditor"
                      >
                        ✎
                      </button>
                    </span>
                    <span class="item-rack-text-label">机架</span>
                  </div>

                  <select id="item-rack-id" v-model="form.rackId">
                    <option value="">{{ itemRackPlaceholderText }}</option>
                    <option v-for="opt in itemRackOptions" :key="`rack-${opt.value}`" :value="String(opt.value)">
                      {{ opt.label }}
                    </option>
                  </select>
                </div>

                <div class="item-inline-2 item-rack-placement-row">
                  <label class="item-rack-position-field">
                    <span>机架位置</span>
                    <span class="quick-tip field-select-tip" data-quick-tip="机架行">
                      <select v-model.number="form.rackPosition">
                        <option value="">选择</option>
                        <option v-for="position in itemRackPositionOptions" :key="`rack-pos-${position}`" :value="position">
                          {{ position }}
                        </option>
                      </select>
                    </span>
                  </label>
                  <label class="item-rack-depth-field">
                    <span aria-hidden="true"></span>
                    <span class="quick-tip field-select-tip" data-quick-tip="占用机架深度。(F)前, (M)中, (B)后">
                      <select v-model="form.rackPosDepth">
                        <option value="4">F--</option>
                        <option value="6">FM-</option>
                        <option value="3">-MB</option>
                        <option value="2">-M-</option>
                        <option value="1">--B</option>
                        <option value="7">FMB</option>
                      </select>
                    </span>
                  </label>
                </div>

                <label><span>用途</span><input v-model="form.function" type="text" /></label>
                <label>
                  <span>使用人</span>
                  <select v-model="form.userId">
                    <option value="">请选择</option>
                    <option v-for="opt in getOptionsByFieldKey('userId')" :key="`user-${opt.value}`" :value="String(opt.value)">
                      {{ opt.label }}
                    </option>
                  </select>
                </label>
                <label><span>维护记录</span><textarea v-model="form.maintenanceInfo" /></label>
                <h4>账目</h4>
                <label>
                  <span>供应商</span>
                  <span class="quick-tip field-select-tip" data-quick-tip="诸如捐赠者、供应商之类信息最好在相关单据中录入">
                    <input v-model="form.origin" type="text" />
                  </span>
                </label>
                <label><span>采购价格(￥)</span><input v-model="form.purchPrice" type="text" /></label>
              </section>

              <section class="item-block">
                <h4>维保</h4>
                <label><span>采购日期</span><DateInput v-model="form.purchaseDate" /></label>
                <label><span>维保月数</span><input v-model.number="form.warrantyMonths" type="number" /></label>
                <label><span>维保信息</span><input v-model="form.warrInfo" type="text" /></label>
                <h4>硬件配置</h4>
                <label><span>硬盘</span><input v-model="form.hd" type="text" /></label>
                <label><span>内存</span><input v-model="form.ram" type="text" /></label>
                <label><span>CPU型号</span><input v-model="form.cpu" type="text" /></label>
                <label><span>Raid卡型号</span><input v-model="form.raid" type="text" /></label>
                <label><span>Raid配置</span><textarea v-model="form.raidConfig" /></label>
              </section>

              <section class="item-block">
                <h4>网络</h4>
                <label><span>MACs</span><input v-model="form.macs" type="text" /></label>
                <label><span>IPv4</span><input v-model="form.ipv4" type="text" /></label>
                <label><span>IPv6</span><input v-model="form.ipv6" type="text" /></label>
                <label><span>远程管理IP</span><input v-model="form.remAdmIp" type="text" /></label>
                <label><span>管理跳线</span><input v-model="form.dnsName" type="text" /></label>
                <label><span>Bond名称</span><input v-model="form.panelPort" type="text" /></label>
                <label><span>业务跳线</span><input v-model="form.switchPort" type="text" /></label>
                <label>
                  <span>交换机</span>
                  <select v-model="form.switchId">
                    <option value="">请选择</option>
                    <option v-for="opt in getOptionsByFieldKey('switchId')" :key="`switch-${opt.value}`" :value="String(opt.value)">
                      {{ opt.label }}
                    </option>
                  </select>
                </label>
                <label>
                  <span>网络端口</span>
                  <select v-model="form.ports">
                    <option value="0">0</option>
                    <option v-for="i in 60" :key="`port-${i}`" :value="String(i)">{{ i }}</option>
                  </select>
                </label>
              </section>
            </div>

            <div class="item-bottom-grid hardware-item-bottom-grid">
              <section class="item-bottom-block">
                <h4>关联概览</h4>
                <div class="item-overview-tabs">
                  <button type="button" :class="{ active: activeOverviewTab === 'items' }" @click="activeOverviewTab = 'items'">硬件</button>
                  <button type="button" :class="{ active: activeOverviewTab === 'software' }" @click="activeOverviewTab = 'software'">软件</button>
                  <button type="button" :class="{ active: activeOverviewTab === 'invoices' }" @click="activeOverviewTab = 'invoices'">单据</button>
                  <button type="button" :class="{ active: activeOverviewTab === 'contracts' }" @click="activeOverviewTab = 'contracts'">合同</button>
                </div>
                <div class="item-overview-list">
                  <div v-for="entry in itemOverviewRows" :key="`overview-${activeOverviewTab}-${entry.id}`" class="item-overview-row">
                    <span class="mono item-overview-index">{{ entry.index }}:</span>
                    <button class="item-overview-link quick-tip" type="button" :data-quick-tip="entry.tip" @click="openOverviewEntry(entry)">
                      <span>{{ entry.text }}</span>
                    </button>
                  </div>
                  <div v-if="itemOverviewRows.length === 0" class="muted-text">暂无关联记录</div>
                </div>
              </section>

              <section class="item-bottom-block">
                <h4>
                  Tags
                  <span class="tag-editor-title-tip quick-tip" data-quick-tip="更改将立即保存。删除标记仅删除关联，不会删除标记本身，可在“标记”菜单中维护。">
                    （
                    <button class="text-link-btn" type="button" @click="toggleItemTagEditor">
                      {{ itemTagEditorOpen ? '编辑完成' : '编辑标记' }}
                    </button>
                    ）
                  </span>
                </h4>
                <div class="item-tags software-tag-list">
                  <span v-for="tag in itemTags" :key="`item-tag-${tag}`" class="software-tag-chip">
                    <span>{{ tag }}</span>
                    <button
                      v-if="itemTagEditorOpen && selectedId && canWrite"
                      type="button"
                      class="software-tag-remove-btn"
                      @click="removeItemTag(tag)"
                    >
                      ×
                    </button>
                  </span>
                  <span v-if="itemTags.length === 0" class="muted-text">暂无标记</span>
                </div>
                <div v-if="itemTagEditorOpen && selectedId && canWrite" class="software-tag-editor item-tag-editor">
                  <input
                    v-model="itemTagInput"
                    class="software-tag-input"
                    type="text"
                    placeholder="输入标记后按回车或点击新增"
                    @keyup.enter.prevent="addItemTag"
                  />
                  <button :disabled="itemTagSaving" type="button" class="small-btn" @click="addItemTag">
                    {{ itemTagSaving ? '处理中...' : '新增' }}
                  </button>
                </div>
                <p v-else-if="itemTagEditorOpen && !selectedId" class="muted-text">请先保存硬件记录，再编辑标记。</p>
                <p v-if="itemTagMessage" class="muted-text">{{ itemTagMessage }}</p>
              </section>

              <section class="item-bottom-block">
                <h4>关联文件</h4>
                <div class="software-managed-file-list">
                  <div
                    v-for="file in managedLinkedFileRows"
                    :key="`item-file-${String(file.id ?? '')}`"
                    class="software-managed-file-card"
                  >
                    <div class="software-managed-file-card-head">
                      <div class="software-managed-file-card-name">#{{ file.id }} {{ file.fileName || '-' }}</div>
                      <div class="software-managed-file-card-actions">
                        <button
                          v-if="canRemoveManagedLinkedFile(file)"
                          type="button"
                          class="field-link-icon quick-tip quick-tip-bottom-left"
                          :data-quick-tip="'解除关联，保存硬件后生效。\n若文件是孤立的(没有其他内容与之相关联)，则会将其删除'"
                          @click="removeLinkedFileSelection(file.id)"
                        >
                          <img class="field-link-icon-image" src="/images/delete.png" alt="删除关联" />
                        </button>
                        <button
                          type="button"
                          class="field-link-icon quick-tip quick-tip-bottom-left"
                          :data-quick-tip="`在新窗口编辑文件 ${Number(file.id ?? 0) || '-'}`"
                          @click="openResourceEditInNewWindow('files', Number(file.id ?? 0))"
                        >
                          <img class="field-link-icon-image" src="/images/edit.png" alt="编辑文件" />
                        </button>
                        <button
                          type="button"
                          class="field-link-icon quick-tip quick-tip-bottom-left"
                          :data-quick-tip="`下载文件: ${file.fileName || '-'}`"
                          @click="downloadLinkedFileByID(file.id, file.fileName)"
                        >
                          <img class="field-link-icon-image" src="/images/down.png" alt="下载文件" />
                        </button>
                      </div>
                    </div>
                    <div class="software-managed-file-card-meta">
                      <div class="software-managed-file-card-field">
                        <span>类型</span>
                        <strong>{{ file.typeDesc || '-' }}</strong>
                      </div>
                      <div class="software-managed-file-card-field">
                        <span>日期</span>
                        <strong>{{ file.date || '-' }}</strong>
                      </div>
                      <div class="software-managed-file-card-field software-managed-file-card-title">
                        <span>标题</span>
                        <strong>{{ file.title || '-' }}</strong>
                      </div>
                    </div>
                  </div>
                  <div v-if="managedLinkedFileRows.length === 0" class="muted-text">暂无关联文件</div>
                </div>
                <p class="muted-text">上传文件请在“上传文件”页签或“文件”菜单中维护。</p>
              </section>
            </div>
          </section>

          <section v-show="activeItemTab === 'itemLinks'" class="item-tab-pane">
            <div class="item-rel-panel">
              <div class="item-rel-header">
                <h4>内部硬件关联</h4>
                <input v-model="itemLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ itemItemRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemItems', 'id')">
                          编号 <span>{{ getNonSoftwareRelationSortIcon('itemItems', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemItems', 'itemType')">
                          类型 <span>{{ getNonSoftwareRelationSortIcon('itemItems', 'itemType') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemItems', 'manufacturer')">
                          厂商 <span>{{ getNonSoftwareRelationSortIcon('itemItems', 'manufacturer') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemItems', 'model')">
                          型号 <span>{{ getNonSoftwareRelationSortIcon('itemItems', 'model') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemItems', 'label')">
                          标签 <span>{{ getNonSoftwareRelationSortIcon('itemItems', 'label') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemItems', 'dnsName')">
                          管理跳线 <span>{{ getNonSoftwareRelationSortIcon('itemItems', 'dnsName') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemItems', 'principal')">
                          负责人 <span>{{ getNonSoftwareRelationSortIcon('itemItems', 'principal') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemItems', 'sn')">
                          设备序列号 <span>{{ getNonSoftwareRelationSortIcon('itemItems', 'sn') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="row in itemItemRelationRows" :key="`item-rel-${row.id}`" :class="{ 'is-linked': isMultiSelected('itemLinks', row.id) }">
                      <td><input v-model="form.itemLinks" type="checkbox" :value="String(row.id)" /></td>
                      <td>
                        <button
                          type="button"
                          class="relation-jump-btn quick-tip"
                          :data-quick-tip="getSoftwareRelationIDTip('item', row.id)"
                          @click="openResourceEditInNewWindow('items', row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.itemType || '-' }}</td>
                      <td>{{ row.manufacturer || '-' }}</td>
                      <td>{{ row.model || '-' }}</td>
                      <td>{{ row.label || '-' }}</td>
                      <td>{{ row.dnsName || '-' }}</td>
                      <td>{{ row.principal || '-' }}</td>
                      <td>{{ row.sn || '-' }}</td>
                    </tr>
                    <tr v-if="itemItemRelationRows.length === 0">
                      <td colspan="9" class="item-rel-empty-cell">暂无可关联硬件</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>

          <section v-show="activeItemTab === 'invoiceLinks'" class="item-tab-pane">
            <div class="item-rel-panel">
              <div class="item-rel-header">
                <h4>关联单据</h4>
                <input v-model="invoiceLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ itemInvoiceRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemInvoices', 'id')">
                          编号 <span>{{ getNonSoftwareRelationSortIcon('itemInvoices', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemInvoices', 'vendor')">
                          供应商 <span>{{ getNonSoftwareRelationSortIcon('itemInvoices', 'vendor') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemInvoices', 'number')">
                          订单编号 <span>{{ getNonSoftwareRelationSortIcon('itemInvoices', 'number') }}</span>
                        </button>
                      </th>
                      <th class="item-invoice-desc-col">
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemInvoices', 'description')">
                          单据描述 / 文件 <span>{{ getNonSoftwareRelationSortIcon('itemInvoices', 'description') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemInvoices', 'date')">
                          日期 <span>{{ getNonSoftwareRelationSortIcon('itemInvoices', 'date') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="row in itemInvoiceRelationRows"
                      :key="`inv-rel-${row.id}`"
                      :class="{ 'is-linked': isMultiSelected('invoiceLinks', row.id) }"
                    >
                      <td><input v-model="form.invoiceLinks" type="checkbox" :value="String(row.id)" /></td>
                      <td>
                        <button
                          type="button"
                          class="relation-jump-btn quick-tip"
                          :data-quick-tip="getSoftwareRelationIDTip('invoice', row.id)"
                          @click="openResourceEditInNewWindow('invoices', row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.vendor || '-' }}</td>
                      <td>{{ row.number || '-' }}</td>
                      <td class="item-invoice-desc-col">
                        <div v-if="row.description || parseInvoiceFileEntriesValue(row.files).length > 0" class="item-invoice-desc-cell">
                          <div v-if="row.description" class="item-invoice-desc-text">{{ row.description }}</div>
                          <div v-if="parseInvoiceFileEntriesValue(row.files).length > 0" class="item-invoice-file-list">
                            <button
                              v-for="entry in parseInvoiceFileEntriesValue(row.files)"
                              :key="`item-invoice-file-${row.id}-${entry.index}-${entry.id ?? 'na'}`"
                              type="button"
                              class="location-floorplan-table-link quick-tip"
                              :data-quick-tip="entry.id ? `在新窗口预览文件 ${entry.id}` : entry.text"
                              :disabled="!entry.id"
                              @click="entry.id ? openFilePreviewInNewWindow(entry.id) : undefined"
                            >
                              <span class="location-floorplan-table-link-icon">{{ entry.id ? '↗' : '·' }}</span>
                              <span class="location-floorplan-table-link-text">{{ entry.text }}</span>
                            </button>
                          </div>
                        </div>
                        <span v-else>-</span>
                      </td>
                      <td>{{ row.date || '-' }}</td>
                    </tr>
                    <tr v-if="itemInvoiceRelationRows.length === 0">
                      <td colspan="6" class="item-rel-empty-cell">暂无可关联单据</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>

          <section v-show="activeItemTab === 'logs'" class="item-tab-pane">
            <p v-if="!selectedId" class="muted-text">新增记录后可查看维护日志。</p>
            <div v-else class="table-wrap item-log-wrap">
              <table class="item-log-table">
                <thead>
                  <tr>
                    <th>编号</th>
                    <th>更新日期</th>
                    <th>问题描述</th>
                    <th>处理情况</th>
                    <th>提交日期</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in itemActions" :key="`log-${String(row.id ?? '')}-${String(row.entrydate ?? '')}`">
                    <td>{{ row.id ?? '-' }}</td>
                    <td>{{ toLocalDateText(row.actiondate) }}</td>
                    <td>{{ row.description || '-' }}</td>
                    <td>{{ row.invoiceinfo || '-' }}</td>
                    <td>{{ toLocalDateText(row.entrydate) }}</td>
                  </tr>
                  <tr v-if="itemActions.length === 0">
                    <td colspan="5">暂无维护日志</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <section v-show="activeItemTab === 'softwareLinks'" class="item-tab-pane">
            <div class="item-rel-panel">
              <div class="item-rel-header">
                <h4>软件关联</h4>
                <input v-model="softwareLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ itemSoftwareRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemSoftware', 'id')">
                          编号 <span>{{ getNonSoftwareRelationSortIcon('itemSoftware', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemSoftware', 'manufacturer')">
                          厂商 <span>{{ getNonSoftwareRelationSortIcon('itemSoftware', 'manufacturer') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemSoftware', 'titleVersion')">
                          标题/版本 <span>{{ getNonSoftwareRelationSortIcon('itemSoftware', 'titleVersion') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="row in itemSoftwareRelationRows"
                      :key="`software-rel-${row.id}`"
                      :class="{ 'is-linked': isMultiSelected('softwareLinks', row.id) }"
                    >
                      <td><input v-model="form.softwareLinks" type="checkbox" :value="String(row.id)" /></td>
                      <td>
                        <button
                          type="button"
                          class="relation-jump-btn quick-tip"
                          :data-quick-tip="getSoftwareRelationIDTip('software', row.id)"
                          @click="openResourceEditInNewWindow('software', row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.manufacturer || '-' }}</td>
                      <td>{{ `${row.title || '-'} ${row.version || ''}`.trim() }}</td>
                    </tr>
                    <tr v-if="itemSoftwareRelationRows.length === 0">
                      <td colspan="4" class="item-rel-empty-cell">暂无可关联软件</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>

          <section v-show="activeItemTab === 'contractLinks'" class="item-tab-pane">
            <div class="item-rel-panel">
              <div class="item-rel-header">
                <h4>关联合同</h4>
                <input v-model="contractLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ itemContractRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemContracts', 'id')">
                          编号 <span>{{ getNonSoftwareRelationSortIcon('itemContracts', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemContracts', 'contractor')">
                          承包方 <span>{{ getNonSoftwareRelationSortIcon('itemContracts', 'contractor') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('itemContracts', 'title')">
                          标题 <span>{{ getNonSoftwareRelationSortIcon('itemContracts', 'title') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="row in itemContractRelationRows"
                      :key="`contract-rel-${row.id}`"
                      :class="{ 'is-linked': isMultiSelected('contractLinks', row.id) }"
                    >
                      <td><input v-model="form.contractLinks" type="checkbox" :value="String(row.id)" /></td>
                      <td>
                        <button
                          class="relation-jump-btn quick-tip"
                          type="button"
                          :data-quick-tip="getSoftwareRelationIDTip('contract', row.id)"
                          @click="openResourceEditInNewWindow('contracts', row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.contractor || '-' }}</td>
                      <td>{{ row.title || '-' }}</td>
                    </tr>
                    <tr v-if="itemContractRelationRows.length === 0">
                      <td colspan="4" class="item-rel-empty-cell">暂无可关联合同</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>

          <section v-show="activeItemTab === 'files'" class="item-tab-pane">
            <div class="item-rel-panel linked-file-panel">
              <div class="item-rel-header">
                <h4>关联文件</h4>
                <input v-model="fileLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ softwareFileRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'id')">
                          编号 <span>{{ getSoftwareRelationSortIcon('files', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'typeDesc')">
                          类型 <span>{{ getSoftwareRelationSortIcon('files', 'typeDesc') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'title')">
                          标题 <span>{{ getSoftwareRelationSortIcon('files', 'title') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'fileName')">
                          文件名 <span>{{ getSoftwareRelationSortIcon('files', 'fileName') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'date')">
                          签署日期 <span>{{ getSoftwareRelationSortIcon('files', 'date') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="row in softwareFileRelationRows" :key="`item-file-rel-${row.id}`" :class="{ 'is-linked': isMultiSelected('fileLinks', row.id) }">
                      <td><input :checked="isMultiSelected('fileLinks', row.id)" type="checkbox" @change="toggleUploadFileSelection(row.id, getInputChecked($event))" /></td>
                      <td>
                        <button
                          type="button"
                          class="relation-jump-btn quick-tip"
                          :data-quick-tip="getSoftwareRelationIDTip('file', row.id)"
                          @click="openSoftwareRelatedFileInNewWindow(row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.typeDesc || '-' }}</td>
                      <td>{{ row.title || '-' }}</td>
                      <td>{{ row.fileName || '-' }}</td>
                      <td>{{ row.date || '-' }}</td>
                    </tr>
                    <tr v-if="softwareFileRelationRows.length === 0">
                      <td colspan="6" class="item-rel-empty-cell">暂无可关联文件</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            <section v-if="canWrite" class="software-upload-panel linked-upload-panel">
              <h4>上传文件</h4>
              <div class="software-upload-grid">
                <label>
                  <span>标题 <sup class="req">*</sup></span>
                  <input v-model="itemUploadForm.title" type="text" />
                </label>
                <label>
                  <span>类型 <sup class="req">*</sup></span>
                  <select v-model="itemUploadForm.typeId">
                    <option value="">请选择</option>
                    <option v-for="opt in commonUploadTypeOptions" :key="`item-upload-type-${opt.value}`" :value="String(opt.value)">
                      {{ opt.label }}
                    </option>
                  </select>
                </label>
                <label>
                  <span>签署日期 <sup class="req">*</sup></span>
                  <DateInput v-model="itemUploadForm.date" />
                </label>
                <div class="asset-field-row linked-upload-picker-row">
                  <span>选择文件 <sup class="req">*</sup></span>
                  <div class="location-floorplan-picker linked-upload-picker">
                    <button class="ghost-btn small-btn location-floorplan-picker-btn linked-upload-picker-btn" type="button" @click="openItemUploadPicker">
                      选择文件
                    </button>
                    <span :class="selectedItemUploadFileName ? 'location-floorplan-picker-name' : 'location-floorplan-picker-name muted-text'">
                      {{ selectedItemUploadFileName || '未选择文件' }}
                    </span>
                    <input ref="itemUploadInput" class="location-floorplan-input" type="file" @change="onItemUploadFileChange" />
                  </div>
                </div>
              </div>
              <div class="software-upload-actions">
                <button type="button" :disabled="itemUploading" @click="uploadLinkedFile('items')">
                  {{ itemUploading ? '上传中...' : '上传文件' }}
                </button>
                <span class="muted-text">上传后会自动写入“文件”菜单并关联到当前硬件。</span>
              </div>
            </section>
          </section>

          <div class="item-form-actions">
            <button :disabled="saving" type="submit">{{ saving ? '提交中...' : selectedId ? '保存修改' : '创建' }}</button>
            <button class="ghost-btn" type="button" @click="drawerOpen = false">取消</button>
          </div>
        </template>

        <template v-else-if="isSoftwareResource">
          <div class="software-editor-shell">
          <div class="item-tabs">
            <button
              v-for="tab in softwareEditorTabs"
              :key="tab.key"
              type="button"
              class="item-tab-btn"
              :class="{ active: activeSoftwareTab === tab.key }"
              @mouseenter="tab.key === 'files' ? showUploadTabTip($event, getUploadFileUnlinkTip()) : undefined"
              @mouseleave="hideUploadTabTip"
              @focus="tab.key === 'files' ? showUploadTabTip($event, getUploadFileUnlinkTip()) : undefined"
              @blur="hideUploadTabTip"
              @click="activeSoftwareTab = tab.key; hideUploadTabTip()"
            >
              {{ tab.label }}
            </button>
          </div>

          <div class="software-editor-scroll">
          <section v-show="activeSoftwareTab === 'softwareData'" class="item-tab-pane">
            <div class="asset-grid-2 software-data-grid">
              <section class="asset-block software-attr-block">
                <h4>软件属性</h4>
                <label><span>编号</span><input :value="selectedId ?? '-'" type="text" disabled /></label>
                <div class="asset-field-row">
                  <span class="field-label-with-icon">
                    <span class="field-link-icon-wrap">
                      <button
                        type="button"
                        class="field-link-icon quick-tip quick-tip-top-right"
                        data-quick-tip="在新窗口编辑厂商（代理）"
                        :disabled="!Number(form.manufacturerId ?? 0)"
                        @click="openSoftwareManufacturerByField"
                      >
                        ✎
                      </button>
                    </span>
                    <span>厂商 <sup class="req">*</sup></span>
                  </span>
                  <span class="quick-tip field-select-tip" data-quick-tip="可在“代理”菜单中新增更多厂商">
                    <select v-model="form.manufacturerId">
                      <option value="">请选择</option>
                      <option v-for="opt in getOptionsByFieldKey('manufacturerId')" :key="`soft-man-${opt.value}`" :value="String(opt.value)">
                        {{ opt.label }}
                      </option>
                    </select>
                  </span>
                </div>
                <label><span>标题 <sup class="req">*</sup></span><input v-model="form.title" type="text" /></label>
                <label><span>版本 <sup class="req">*</sup></span><input v-model="form.version" type="text" /></label>
                <label><span>采购日期 <sup class="req">*</sup></span><DateInput v-model="form.purchaseDate" /></label>
                <label>
                  <span>授权数量</span>
                  <select v-model="form.licenseQty">
                    <option v-for="qty in softwareLicenseQtyOptions" :key="`lic-qty-${qty}`" :value="String(qty)">
                      {{ qty }}
                    </option>
                  </select>
                </label>
                <div class="asset-field-row">
                  <span>授权类型</span>
                  <div class="license-type-group">
                    <label><input v-model="form.licenseType" type="radio" value="0" />按设备</label>
                    <label><input v-model="form.licenseType" type="radio" value="1" />按 CPU</label>
                    <label><input v-model="form.licenseType" type="radio" value="2" />按核心</label>
                  </div>
                </div>
                <label><span>许可信息</span><textarea v-model="form.slicenseInfo" /></label>
                <label><span>其它信息</span><textarea v-model="form.info" /></label>
              </section>

              <section class="asset-block software-side-block">
                <h4>关联概览</h4>
                <div class="item-overview-tabs">
                  <button type="button" :class="{ active: activeSoftwareOverviewTab === 'items' }" @click="activeSoftwareOverviewTab = 'items'">硬件</button>
                  <button type="button" :class="{ active: activeSoftwareOverviewTab === 'invoices' }" @click="activeSoftwareOverviewTab = 'invoices'">单据</button>
                  <button type="button" :class="{ active: activeSoftwareOverviewTab === 'contracts' }" @click="activeSoftwareOverviewTab = 'contracts'">合同</button>
                </div>
                <div class="item-overview-list">
                  <div v-for="entry in softwareOverviewRows" :key="`soft-overview-${activeSoftwareOverviewTab}-${entry.id}`" class="item-overview-row">
                    <span class="mono item-overview-index">{{ entry.index }}:</span>
                    <button class="item-overview-link quick-tip" type="button" :data-quick-tip="entry.tip" @click="openOverviewEntry(entry)">
                      <span>{{ entry.text }}</span>
                    </button>
                  </div>
                  <div v-if="softwareOverviewRows.length === 0" class="muted-text">暂无关联记录</div>
                </div>

                <h4>
                  Tags
                  <span class="tag-editor-title-tip quick-tip" data-quick-tip="更改将立即保存。删除标记仅删除关联，不会删除标记本身，可在“标记”菜单中维护。">
                    （
                    <button class="text-link-btn" type="button" @click="toggleSoftwareTagEditor">
                      {{ softwareTagEditorOpen ? '编辑完成' : '编辑标记' }}
                    </button>
                    ）
                  </span>
                </h4>
                <div class="item-tags software-tag-list">
                  <span v-for="tag in softwareTags" :key="`soft-tag-${tag}`" class="software-tag-chip">
                    <span>{{ tag }}</span>
                    <button
                      v-if="softwareTagEditorOpen && selectedId && canWrite"
                      type="button"
                      class="software-tag-remove-btn"
                      @click="removeSoftwareTag(tag)"
                    >
                      ×
                    </button>
                  </span>
                  <span v-if="softwareTags.length === 0" class="muted-text">暂无标记</span>
                </div>
                <div v-if="softwareTagEditorOpen && selectedId && canWrite" class="software-tag-editor">
                  <input
                    v-model="softwareTagInput"
                    class="software-tag-input"
                    type="text"
                    placeholder="输入标记后按回车或点击新增"
                    @keyup.enter.prevent="addSoftwareTag"
                  />
                  <button :disabled="softwareTagSaving" type="button" class="small-btn" @click="addSoftwareTag">
                    {{ softwareTagSaving ? '处理中...' : '新增' }}
                  </button>
                </div>
                <p v-else-if="softwareTagEditorOpen && !selectedId" class="muted-text">请先保存软件记录，再编辑标记。</p>
                <p v-if="softwareTagMessage" class="muted-text">{{ softwareTagMessage }}</p>

                <h4>管理文件</h4>
                <div class="software-managed-file-list">
                  <div
                    v-for="file in managedLinkedFileRows"
                    :key="`soft-file-${String(file.id ?? '')}`"
                    class="software-managed-file-card"
                  >
                    <div class="software-managed-file-card-head">
                      <div class="software-managed-file-card-name">#{{ file.id }} {{ file.fileName || '-' }}</div>
                      <div class="software-managed-file-card-actions">
                        <button
                          v-if="canRemoveManagedLinkedFile(file)"
                          type="button"
                          class="field-link-icon quick-tip quick-tip-bottom-left"
                          :data-quick-tip="'解除关联，保存软件后生效。\n若文件是孤立的(没有其他内容与之相关联)，则会将其删除'"
                          @click="removeLinkedFileSelection(file.id)"
                        >
                          <img class="field-link-icon-image" src="/images/delete.png" alt="删除关联" />
                        </button>
                        <button
                          type="button"
                          class="field-link-icon quick-tip quick-tip-bottom-left"
                          :data-quick-tip="`在新窗口编辑文件 ${Number(file.id ?? 0) || '-'}`"
                          @click="openResourceEditInNewWindow('files', Number(file.id ?? 0))"
                        >
                          <img class="field-link-icon-image" src="/images/edit.png" alt="编辑文件" />
                        </button>
                        <button
                          type="button"
                          class="field-link-icon quick-tip quick-tip-bottom-left"
                          :data-quick-tip="`下载文件: ${file.fileName || '-'}`"
                          @click="downloadLinkedFileByID(file.id, file.fileName)"
                        >
                          <img class="field-link-icon-image" src="/images/down.png" alt="下载文件" />
                        </button>
                      </div>
                    </div>
                    <div class="software-managed-file-card-meta">
                      <div class="software-managed-file-card-field">
                        <span>类型</span>
                        <strong>{{ file.typeDesc || '-' }}</strong>
                      </div>
                      <div class="software-managed-file-card-field">
                        <span>日期</span>
                        <strong>{{ file.date || '-' }}</strong>
                      </div>
                      <div class="software-managed-file-card-field software-managed-file-card-title">
                        <span>标题</span>
                        <strong>{{ file.title || '-' }}</strong>
                      </div>
                    </div>
                  </div>
                  <div v-if="managedLinkedFileRows.length === 0" class="muted-text">暂无关联文件</div>
                </div>
              </section>
            </div>
          </section>

          <section v-show="activeSoftwareTab === 'itemLinks'" class="item-tab-pane">
            <div class="item-rel-panel">
              <div class="item-rel-header">
                <h4>硬件关联</h4>
                <input v-model="itemLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ softwareItemRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table software-item-rel-table">
                  <thead>
                    <tr>
                      <th class="software-item-col-contact">关联</th>
                      <th class="software-item-col-num">
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('items', 'id')">
                          编号 <span>{{ getSoftwareRelationSortIcon('items', 'id') }}</span>
                        </button>
                      </th>
                      <th class="software-item-col-type">
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('items', 'itemType')">
                          类型 <span>{{ getSoftwareRelationSortIcon('items', 'itemType') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('items', 'manufacturer')">
                          厂商 <span>{{ getSoftwareRelationSortIcon('items', 'manufacturer') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('items', 'model')">
                          型号 <span>{{ getSoftwareRelationSortIcon('items', 'model') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('items', 'label')">
                          标签 <span>{{ getSoftwareRelationSortIcon('items', 'label') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('items', 'dnsName')">
                          管理跳线 <span>{{ getSoftwareRelationSortIcon('items', 'dnsName') }}</span>
                        </button>
                      </th>
                      <th class="software-item-col-user">
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('items', 'principal')">
                          负责人 <span>{{ getSoftwareRelationSortIcon('items', 'principal') }}</span>
                        </button>
                      </th>
                      <th class="software-item-col-sn">
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('items', 'sn')">
                          设备序列号 <span>{{ getSoftwareRelationSortIcon('items', 'sn') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="row in softwareItemRelationRows"
                      :key="`soft-item-rel-${row.id}`"
                      :class="{ 'is-linked': isMultiSelected('itemLinks', row.id) }"
                    >
                      <td><input v-model="form.itemLinks" type="checkbox" :value="String(row.id)" /></td>
                      <td>
                        <button
                          type="button"
                          class="relation-jump-btn quick-tip"
                          :data-quick-tip="getSoftwareRelationIDTip('item', row.id)"
                          @click="openSoftwareRelatedItemInNewWindow(row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td class="quick-tip software-item-col-type" :data-quick-tip="getSoftwareItemTypeTooltip(row)">{{ row.itemType || '-' }}</td>
                      <td>{{ row.manufacturer || '-' }}</td>
                      <td>{{ row.model || '-' }}</td>
                      <td>{{ row.label || '-' }}</td>
                      <td>{{ row.dnsName || '-' }}</td>
                      <td class="software-item-col-user">{{ row.principal || '-' }}</td>
                      <td class="software-item-col-sn">{{ row.sn || '-' }}</td>
                    </tr>
                    <tr v-if="softwareItemRelationRows.length === 0">
                      <td colspan="9" class="item-rel-empty-cell">暂无可关联硬件</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <p class="muted-text">仅显示“硬件类型”中支持软件的硬件条目。</p>
            </div>
          </section>

          <section v-show="activeSoftwareTab === 'invoiceLinks'" class="item-tab-pane">
            <div class="item-rel-panel">
              <div class="item-rel-header">
                <h4>单据关联</h4>
                <input v-model="invoiceLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ softwareInvoiceRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('invoices', 'id')">
                          编号 <span>{{ getSoftwareRelationSortIcon('invoices', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('invoices', 'vendor')">
                          供应商 <span>{{ getSoftwareRelationSortIcon('invoices', 'vendor') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('invoices', 'number')">
                          订单编号 <span>{{ getSoftwareRelationSortIcon('invoices', 'number') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('invoices', 'description')">
                          描述 <span>{{ getSoftwareRelationSortIcon('invoices', 'description') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('invoices', 'date')">
                          日期 <span>{{ getSoftwareRelationSortIcon('invoices', 'date') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="row in softwareInvoiceRelationRows"
                      :key="`soft-inv-rel-${row.id}`"
                      :class="{ 'is-linked': isMultiSelected('invoiceLinks', row.id) }"
                    >
                      <td><input v-model="form.invoiceLinks" type="checkbox" :value="String(row.id)" /></td>
                      <td>
                        <button
                          type="button"
                          class="relation-jump-btn quick-tip"
                          :data-quick-tip="getSoftwareRelationIDTip('invoice', row.id)"
                          @click="openSoftwareRelatedInvoiceInNewWindow(row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.vendor || '-' }}</td>
                      <td>{{ row.number || '-' }}</td>
                      <td>{{ row.description || '-' }}</td>
                      <td>{{ row.date || '-' }}</td>
                    </tr>
                    <tr v-if="softwareInvoiceRelationRows.length === 0">
                      <td colspan="6" class="item-rel-empty-cell">暂无可关联单据</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>

          <section v-show="activeSoftwareTab === 'contractLinks'" class="item-tab-pane">
            <div class="item-rel-panel">
              <div class="item-rel-header">
                <h4>合同关联</h4>
                <input v-model="contractLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ softwareContractRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('contracts', 'id')">
                          编号 <span>{{ getSoftwareRelationSortIcon('contracts', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('contracts', 'contractor')">
                          承包方 <span>{{ getSoftwareRelationSortIcon('contracts', 'contractor') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('contracts', 'title')">
                          标题 <span>{{ getSoftwareRelationSortIcon('contracts', 'title') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="row in softwareContractRelationRows"
                      :key="`soft-contr-rel-${row.id}`"
                      :class="{ 'is-linked': isMultiSelected('contractLinks', row.id) }"
                    >
                      <td><input v-model="form.contractLinks" type="checkbox" :value="String(row.id)" /></td>
                      <td>
                        <button
                          type="button"
                          class="relation-jump-btn quick-tip"
                          :data-quick-tip="getSoftwareRelationIDTip('contract', row.id)"
                          @click="openSoftwareRelatedContractInNewWindow(row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.contractor || '-' }}</td>
                      <td>{{ row.title || '-' }}</td>
                    </tr>
                    <tr v-if="softwareContractRelationRows.length === 0">
                      <td colspan="4" class="item-rel-empty-cell">暂无可关联合同</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>

          <section v-show="activeSoftwareTab === 'files'" class="item-tab-pane">
            <div class="item-rel-panel">
              <div class="item-rel-header">
                <h4>文件</h4>
                <input v-model="fileLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ softwareFileRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'id')">
                          编号 <span>{{ getSoftwareRelationSortIcon('files', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'typeDesc')">
                          类型 <span>{{ getSoftwareRelationSortIcon('files', 'typeDesc') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'title')">
                          标题 <span>{{ getSoftwareRelationSortIcon('files', 'title') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'fileName')">
                          文件名 <span>{{ getSoftwareRelationSortIcon('files', 'fileName') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'date')">
                          签署日期 <span>{{ getSoftwareRelationSortIcon('files', 'date') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="row in softwareFileRelationRows" :key="`soft-file-rel-${row.id}`" :class="{ 'is-linked': isMultiSelected('fileLinks', row.id) }">
                      <td><input :checked="isMultiSelected('fileLinks', row.id)" type="checkbox" @change="toggleUploadFileSelection(row.id, getInputChecked($event))" /></td>
                      <td>
                        <button
                          type="button"
                          class="relation-jump-btn quick-tip"
                          :data-quick-tip="getSoftwareRelationIDTip('file', row.id)"
                          @click="openSoftwareRelatedFileInNewWindow(row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.typeDesc || '-' }}</td>
                      <td>{{ row.title || '-' }}</td>
                      <td>{{ row.fileName || '-' }}</td>
                      <td>{{ row.date || '-' }}</td>
                    </tr>
                    <tr v-if="softwareFileRelationRows.length === 0">
                      <td colspan="6" class="item-rel-empty-cell">暂无可关联文件</td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <section v-if="canWrite" class="software-upload-panel linked-upload-panel">
                <h4>上传文件</h4>
                <div class="software-upload-grid">
                  <label>
                    <span>标题 <sup class="req">*</sup></span>
                    <input v-model="softwareUploadForm.title" type="text" />
                  </label>
                  <label>
                    <span>文件类型 <sup class="req">*</sup></span>
                    <select v-model="softwareUploadForm.typeId">
                      <option value="">请选择</option>
                      <option v-for="opt in softwareUploadTypeOptions" :key="`soft-upload-type-${opt.value}`" :value="String(opt.value)">
                        {{ opt.label }}
                      </option>
                    </select>
                  </label>
                  <label>
                    <span>签署日期 <sup class="req">*</sup></span>
                    <DateInput v-model="softwareUploadForm.date" />
                  </label>
                  <div class="asset-field-row linked-upload-picker-row">
                    <span>选择文件 <sup class="req">*</sup></span>
                    <div class="location-floorplan-picker linked-upload-picker">
                      <button class="ghost-btn small-btn location-floorplan-picker-btn linked-upload-picker-btn" type="button" @click="openSoftwareUploadPicker">
                        选择文件
                      </button>
                      <span :class="selectedSoftwareUploadFileName ? 'location-floorplan-picker-name' : 'location-floorplan-picker-name muted-text'">
                        {{ selectedSoftwareUploadFileName || '未选择文件' }}
                      </span>
                      <input ref="softwareUploadInput" class="location-floorplan-input" type="file" @change="onSoftwareUploadFileChange" />
                    </div>
                  </div>
                </div>
                <div class="software-upload-actions">
                  <button type="button" :disabled="softwareUploading" @click="uploadSoftwareFile">
                    {{ softwareUploading ? '上传中...' : '上传文件' }}
                  </button>
                  <span class="muted-text">上传成功后会自动写入“文件”菜单并关联到当前软件。</span>
                </div>
              </section>
            </div>
          </section>
          </div>
          </div>

          <div class="item-form-actions software-item-form-actions">
            <button :disabled="saving" type="submit">{{ saving ? '提交中...' : selectedId ? '保存修改' : '创建' }}</button>
            <button class="ghost-btn" type="button" @click="drawerOpen = false">取消</button>
          </div>
        </template>

        <template v-else-if="isInvoiceResource">
          <div class="item-tabs">
            <button
              v-for="tab in invoiceEditorTabs"
              :key="tab.key"
              type="button"
              class="item-tab-btn"
              :class="{ active: activeInvoiceTab === tab.key }"
              @mouseenter="tab.key === 'files' ? showUploadTabTip($event, getUploadFileUnlinkTip()) : undefined"
              @mouseleave="hideUploadTabTip"
              @focus="tab.key === 'files' ? showUploadTabTip($event, getUploadFileUnlinkTip()) : undefined"
              @blur="hideUploadTabTip"
              @click="activeInvoiceTab = tab.key; hideUploadTabTip()"
            >
              {{ tab.label }}
            </button>
          </div>

          <section v-show="activeInvoiceTab === 'invoiceData'" class="item-tab-pane">
            <div class="asset-grid-2 software-data-grid">
              <section class="asset-block software-attr-block">
                <h4>单据属性</h4>
                <label><span>编号</span><input :value="selectedId ?? '-'" type="text" disabled /></label>
                <div class="asset-field-row">
                  <span class="field-label-with-icon">
                    <button
                      class="field-link-icon quick-tip quick-tip-top-right"
                      type="button"
                      data-quick-tip="在新窗口编辑供应商（代理）"
                      :disabled="Number(form.vendorId ?? 0) <= 0"
                      @click="openAgentByField('vendorId')"
                    >
                      ↗
                    </button>
                    供应商 <sup class="req">*</sup>
                  </span>
                  <select v-model="form.vendorId">
                    <option value="">请选择</option>
                    <option v-for="opt in getOptionsByFieldKey('vendorId')" :key="`inv-vendor-${opt.value}`" :value="String(opt.value)">
                      {{ opt.label }}
                    </option>
                  </select>
                </div>
                <div class="asset-field-row">
                  <span class="field-label-with-icon">
                    <button
                      class="field-link-icon quick-tip quick-tip-top-right"
                      type="button"
                      data-quick-tip="在新窗口编辑采购方（代理）"
                      :disabled="Number(form.buyerId ?? 0) <= 0"
                      @click="openAgentByField('buyerId')"
                    >
                      ↗
                    </button>
                    采购方 <sup class="req">*</sup>
                  </span>
                  <select v-model="form.buyerId">
                    <option value="">请选择</option>
                    <option v-for="opt in getOptionsByFieldKey('buyerId')" :key="`inv-buyer-${opt.value}`" :value="String(opt.value)">
                      {{ opt.label }}
                    </option>
                  </select>
                </div>
                <label><span>订单编号 <sup class="req">*</sup></span><input v-model="form.number" type="text" /></label>
                <label><span>日期 <sup class="req">*</sup></span><DateInput v-model="form.date" /></label>
                <label><span>描述</span><textarea v-model="form.description" /></label>
              </section>

              <section class="asset-block software-side-block">
                <h4>关联概览</h4>
                <div class="item-overview-tabs">
                  <button type="button" :class="{ active: activeInvoiceOverviewTab === 'items' }" @click="activeInvoiceOverviewTab = 'items'">硬件</button>
                  <button type="button" :class="{ active: activeInvoiceOverviewTab === 'software' }" @click="activeInvoiceOverviewTab = 'software'">软件</button>
                  <button type="button" :class="{ active: activeInvoiceOverviewTab === 'contracts' }" @click="activeInvoiceOverviewTab = 'contracts'">合同</button>
                </div>
                <div class="item-overview-list">
                  <div v-for="entry in invoiceOverviewRows" :key="`inv-overview-${activeInvoiceOverviewTab}-${entry.id}`" class="item-overview-row">
                    <span class="mono item-overview-index">{{ entry.index }}:</span>
                    <button class="item-overview-link quick-tip" type="button" :data-quick-tip="entry.tip" @click="openOverviewEntry(entry)">
                      <span>{{ entry.text }}</span>
                    </button>
                  </div>
                  <div v-if="invoiceOverviewRows.length === 0" class="muted-text">暂无关联记录</div>
                </div>

                <h4>管理文件</h4>
                <div class="software-managed-file-list">
                  <div
                    v-for="file in managedLinkedFileRows"
                    :key="`inv-file-${String(file.id ?? '')}`"
                    class="software-managed-file-card"
                  >
                    <div class="software-managed-file-card-head">
                      <div class="software-managed-file-card-name">#{{ file.id }} {{ file.fileName || '-' }}</div>
                      <div class="software-managed-file-card-actions">
                        <button
                          v-if="canRemoveManagedLinkedFile(file)"
                          type="button"
                          class="field-link-icon quick-tip quick-tip-bottom-left"
                          :data-quick-tip="'解除关联，保存单据后生效。\n若文件是孤立的(没有其他内容与之相关联)，则会将其删除'"
                          @click="removeLinkedFileSelection(file.id)"
                        >
                          <img class="field-link-icon-image" src="/images/delete.png" alt="删除关联" />
                        </button>
                        <button
                          type="button"
                          class="field-link-icon quick-tip quick-tip-bottom-left"
                          :data-quick-tip="`在新窗口编辑文件 ${Number(file.id ?? 0) || '-'}`"
                          @click="openResourceEditInNewWindow('files', Number(file.id ?? 0))"
                        >
                          <img class="field-link-icon-image" src="/images/edit.png" alt="编辑文件" />
                        </button>
                        <button
                          type="button"
                          class="field-link-icon quick-tip quick-tip-bottom-left"
                          :data-quick-tip="`下载文件: ${file.fileName || '-'}`"
                          @click="downloadLinkedFileByID(file.id, file.fileName)"
                        >
                          <img class="field-link-icon-image" src="/images/down.png" alt="下载文件" />
                        </button>
                      </div>
                    </div>
                    <div class="software-managed-file-card-meta">
                      <div class="software-managed-file-card-field">
                        <span>类型</span>
                        <strong>{{ file.typeDesc || '-' }}</strong>
                      </div>
                      <div class="software-managed-file-card-field">
                        <span>日期</span>
                        <strong>{{ file.date || '-' }}</strong>
                      </div>
                      <div class="software-managed-file-card-field software-managed-file-card-title">
                        <span>标题</span>
                        <strong>{{ file.title || '-' }}</strong>
                      </div>
                    </div>
                  </div>
                  <div v-if="managedLinkedFileRows.length === 0" class="muted-text">暂无关联文件</div>
                </div>
              </section>
            </div>
          </section>

          <section v-show="activeInvoiceTab === 'itemLinks'" class="item-tab-pane">
            <div class="item-rel-panel">
              <div class="item-rel-header">
                <h4>硬件关联</h4>
                <input v-model="itemLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ invoiceItemRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('invoiceItems', 'id')">
                          编号 <span>{{ getNonSoftwareRelationSortIcon('invoiceItems', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('invoiceItems', 'itemType')">
                          类型 <span>{{ getNonSoftwareRelationSortIcon('invoiceItems', 'itemType') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('invoiceItems', 'manufacturer')">
                          厂商 <span>{{ getNonSoftwareRelationSortIcon('invoiceItems', 'manufacturer') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('invoiceItems', 'model')">
                          型号 <span>{{ getNonSoftwareRelationSortIcon('invoiceItems', 'model') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('invoiceItems', 'label')">
                          标签 <span>{{ getNonSoftwareRelationSortIcon('invoiceItems', 'label') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('invoiceItems', 'dnsName')">
                          管理跳线 <span>{{ getNonSoftwareRelationSortIcon('invoiceItems', 'dnsName') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('invoiceItems', 'principal')">
                          负责人 <span>{{ getNonSoftwareRelationSortIcon('invoiceItems', 'principal') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('invoiceItems', 'sn')">
                          设备序列号 <span>{{ getNonSoftwareRelationSortIcon('invoiceItems', 'sn') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="row in invoiceItemRelationRows" :key="`inv-item-rel-${row.id}`" :class="{ 'is-linked': isMultiSelected('itemLinks', row.id) }">
                      <td><input v-model="form.itemLinks" type="checkbox" :value="String(row.id)" /></td>
                      <td>
                        <button
                          class="relation-jump-btn quick-tip"
                          type="button"
                          :data-quick-tip="getSoftwareRelationIDTip('item', row.id)"
                          @click="openResourceEditInNewWindow('items', row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.itemType || '-' }}</td>
                      <td>{{ row.manufacturer || '-' }}</td>
                      <td>{{ row.model || '-' }}</td>
                      <td>{{ row.label || '-' }}</td>
                      <td>{{ row.dnsName || '-' }}</td>
                      <td>{{ row.principal || '-' }}</td>
                      <td>{{ row.sn || '-' }}</td>
                    </tr>
                    <tr v-if="invoiceItemRelationRows.length === 0">
                      <td colspan="9" class="item-rel-empty-cell">暂无可关联硬件</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>

          <section v-show="activeInvoiceTab === 'softwareLinks'" class="item-tab-pane">
            <div class="item-rel-panel">
              <div class="item-rel-header">
                <h4>软件关联</h4>
                <input v-model="softwareLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ invoiceSoftwareRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('invoiceSoftware', 'id')">
                          编号 <span>{{ getNonSoftwareRelationSortIcon('invoiceSoftware', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button
                          type="button"
                          class="relation-sort-btn"
                          @click="toggleNonSoftwareRelationSort('invoiceSoftware', 'manufacturer')"
                        >
                          厂商 <span>{{ getNonSoftwareRelationSortIcon('invoiceSoftware', 'manufacturer') }}</span>
                        </button>
                      </th>
                      <th>
                        <button
                          type="button"
                          class="relation-sort-btn"
                          @click="toggleNonSoftwareRelationSort('invoiceSoftware', 'titleVersion')"
                        >
                          标题/版本 <span>{{ getNonSoftwareRelationSortIcon('invoiceSoftware', 'titleVersion') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="row in invoiceSoftwareRelationRows"
                      :key="`inv-soft-rel-${row.id}`"
                      :class="{ 'is-linked': isMultiSelected('softwareLinks', row.id) }"
                    >
                      <td><input v-model="form.softwareLinks" type="checkbox" :value="String(row.id)" /></td>
                      <td>
                        <button
                          class="relation-jump-btn quick-tip"
                          type="button"
                          :data-quick-tip="getSoftwareRelationIDTip('software', row.id)"
                          @click="openResourceEditInNewWindow('software', row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.manufacturer || '-' }}</td>
                      <td>{{ `${row.title || '-'} ${row.version || ''}`.trim() }}</td>
                    </tr>
                    <tr v-if="invoiceSoftwareRelationRows.length === 0">
                      <td colspan="4" class="item-rel-empty-cell">暂无可关联软件</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>

          <section v-show="activeInvoiceTab === 'contractLinks'" class="item-tab-pane">
            <div class="item-rel-panel">
              <div class="item-rel-header">
                <h4>合同关联</h4>
                <input v-model="contractLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ invoiceContractRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('invoiceContracts', 'id')">
                          编号 <span>{{ getNonSoftwareRelationSortIcon('invoiceContracts', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button
                          type="button"
                          class="relation-sort-btn"
                          @click="toggleNonSoftwareRelationSort('invoiceContracts', 'contractor')"
                        >
                          承包方 <span>{{ getNonSoftwareRelationSortIcon('invoiceContracts', 'contractor') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('invoiceContracts', 'title')">
                          标题 <span>{{ getNonSoftwareRelationSortIcon('invoiceContracts', 'title') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="row in invoiceContractRelationRows"
                      :key="`inv-contr-rel-${row.id}`"
                      :class="{ 'is-linked': isMultiSelected('contractLinks', row.id) }"
                    >
                      <td><input v-model="form.contractLinks" type="checkbox" :value="String(row.id)" /></td>
                      <td>
                        <button
                          class="relation-jump-btn quick-tip"
                          type="button"
                          :data-quick-tip="getSoftwareRelationIDTip('contract', row.id)"
                          @click="openResourceEditInNewWindow('contracts', row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.contractor || '-' }}</td>
                      <td>{{ row.title || '-' }}</td>
                    </tr>
                    <tr v-if="invoiceContractRelationRows.length === 0">
                      <td colspan="4" class="item-rel-empty-cell">暂无可关联合同</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>

          <section v-show="activeInvoiceTab === 'files'" class="item-tab-pane">
            <div class="item-rel-panel linked-file-panel">
              <div class="item-rel-header">
                <h4>关联文件</h4>
                <input v-model="fileLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ softwareFileRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'id')">
                          编号 <span>{{ getSoftwareRelationSortIcon('files', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'typeDesc')">
                          类型 <span>{{ getSoftwareRelationSortIcon('files', 'typeDesc') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'title')">
                          标题 <span>{{ getSoftwareRelationSortIcon('files', 'title') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'fileName')">
                          文件名 <span>{{ getSoftwareRelationSortIcon('files', 'fileName') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'date')">
                          签署日期 <span>{{ getSoftwareRelationSortIcon('files', 'date') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="row in invoiceFileRelationRows" :key="`inv-file-rel-${row.id}`" :class="{ 'is-linked': isMultiSelected('fileLinks', row.id) }">
                      <td><input :checked="isMultiSelected('fileLinks', row.id)" type="checkbox" @change="toggleUploadFileSelection(row.id, getInputChecked($event))" /></td>
                      <td>
                        <button
                          type="button"
                          class="relation-jump-btn quick-tip"
                          :data-quick-tip="getSoftwareRelationIDTip('file', row.id)"
                          @click="openSoftwareRelatedFileInNewWindow(row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.typeDesc || '-' }}</td>
                      <td>{{ row.title || '-' }}</td>
                      <td>{{ row.fileName || '-' }}</td>
                      <td>{{ row.date || '-' }}</td>
                    </tr>
                    <tr v-if="invoiceFileRelationRows.length === 0">
                      <td colspan="6" class="item-rel-empty-cell">暂无可关联发票文件</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            <section v-if="canWrite" class="software-upload-panel linked-upload-panel">
              <h4>上传文件</h4>
              <div class="software-upload-grid">
                <label>
                  <span>标题 <sup class="req">*</sup></span>
                  <input v-model="invoiceUploadForm.title" type="text" />
                </label>
                <label>
                  <span>文件类型</span>
                  <input :value="`固定：${invoiceFileTypeLabel}`" type="text" disabled />
                </label>
                <label>
                  <span>签署日期 <sup class="req">*</sup></span>
                  <DateInput v-model="invoiceUploadForm.date" />
                </label>
                <div class="asset-field-row linked-upload-picker-row">
                  <span>选择文件 <sup class="req">*</sup></span>
                  <div class="location-floorplan-picker linked-upload-picker">
                    <button class="ghost-btn small-btn location-floorplan-picker-btn linked-upload-picker-btn" type="button" @click="openInvoiceUploadPicker">
                      选择文件
                    </button>
                    <span :class="selectedInvoiceUploadFileName ? 'location-floorplan-picker-name' : 'location-floorplan-picker-name muted-text'">
                      {{ selectedInvoiceUploadFileName || '未选择文件' }}
                    </span>
                    <input ref="invoiceUploadInput" class="location-floorplan-input" type="file" :accept="invoiceUploadAccept" @change="onInvoiceUploadFileChange" />
                  </div>
                </div>
              </div>
              <div class="software-upload-actions">
                <button type="button" :disabled="invoiceUploading" @click="uploadLinkedFile('invoices')">
                  {{ invoiceUploading ? '上传中...' : '上传文件' }}
                </button>
                <span class="muted-text">上传成功后会自动写入“文件”菜单并关联到当前单据。</span>
              </div>
            </section>
          </section>

          <div class="item-form-actions">
            <button :disabled="saving" type="submit">{{ saving ? '提交中...' : selectedId ? '保存修改' : '创建' }}</button>
            <button class="ghost-btn" type="button" @click="drawerOpen = false">取消</button>
          </div>
        </template>

        <template v-else-if="isContractResource">
          <div class="item-tabs">
            <button
              v-for="tab in contractEditorTabs"
              :key="tab.key"
              type="button"
              class="item-tab-btn"
              :class="{ active: activeContractTab === tab.key }"
              @mouseenter="tab.key === 'files' ? showUploadTabTip($event, getUploadFileUnlinkTip()) : undefined"
              @mouseleave="hideUploadTabTip"
              @focus="tab.key === 'files' ? showUploadTabTip($event, getUploadFileUnlinkTip()) : undefined"
              @blur="hideUploadTabTip"
              @click="activeContractTab = tab.key; hideUploadTabTip()"
            >
              {{ tab.label }}
            </button>
          </div>

          <section v-show="activeContractTab === 'contractData'" class="item-tab-pane">
            <div class="asset-grid-2">
              <section class="asset-block">
                <h4>合同属性</h4>
                <label><span>编号</span><input :value="selectedId ?? '-'" type="text" disabled /></label>
                <label><span>标题 <sup class="req">*</sup></span><input v-model="form.title" type="text" /></label>
                <label><span>数量 <sup class="req">*</sup></span><input v-model="form.number" type="text" /></label>
                <label>
                  <span>合同类型 <sup class="req">*</sup></span>
                  <select v-model="form.typeId">
                    <option value="">请选择</option>
                    <option v-for="opt in getOptionsByFieldKey('typeId')" :key="`contr-type-${opt.value}`" :value="String(opt.value)">
                      {{ opt.label }}
                    </option>
                  </select>
                </label>
                <label>
                  <span>合同子类型</span>
                  <select v-model="form.subTypeId">
                    <option value="">请选择</option>
                    <option v-for="opt in getOptionsByFieldKey('subTypeId')" :key="`contr-sub-${opt.value}`" :value="String(opt.value)">
                      {{ opt.label }}
                    </option>
                  </select>
                </label>
                <div class="asset-field-row">
                  <span class="field-label-with-icon">
                    <button
                      class="field-link-icon quick-tip quick-tip-top-right"
                      type="button"
                      data-quick-tip="在新窗口编辑承包方（代理）"
                      :disabled="Number(form.contractorId ?? 0) <= 0"
                      @click="openAgentByField('contractorId')"
                    >
                      ↗
                    </button>
                    承包方 <sup class="req">*</sup>
                  </span>
                  <span
                    class="field-tip-wrap quick-tip"
                    data-quick-tip="承包方代理类型"
                  >
                    <select v-model="form.contractorId">
                      <option value="">请选择</option>
                      <option v-for="opt in getOptionsByFieldKey('contractorId')" :key="`contr-agent-${opt.value}`" :value="String(opt.value)">
                        {{ opt.label }}
                      </option>
                    </select>
                  </span>
                </div>
                <div class="asset-field-row">
                  <span class="field-label-with-icon">
                    <button
                      class="field-link-icon quick-tip quick-tip-top-right"
                      type="button"
                      data-quick-tip="在新窗口编辑上级合同"
                      :disabled="Number(form.parentId ?? 0) <= 0"
                      @click="openParentContractEditor"
                    >
                      ↗
                    </button>
                    上级合同
                  </span>
                  <select v-model="form.parentId">
                    <option value="">请选择</option>
                    <option v-for="opt in getOptionsByFieldKey('parentId')" :key="`contr-parent-${opt.value}`" :value="String(opt.value)">
                      {{ opt.label }}
                    </option>
                  </select>
                </div>
                <label><span>总成本 (¥)</span><input v-model="form.totalCost" type="text" /></label>
                <label><span>开始日期 <sup class="req">*</sup></span><DateInput v-model="form.startDate" /></label>
                <label><span>结束日期 <sup class="req">*</sup></span><DateInput v-model="form.currentEndDate" /></label>
                <div class="contract-textarea-grid">
                  <div class="contract-description-offset">
                    <label><span>合同描述</span><textarea v-model="form.description" /></label>
                  </div>
                  <div class="contract-comments-offset">
                    <label><span>注释</span><textarea v-model="form.comments" /></label>
                  </div>
                </div>
              </section>

              <section class="asset-block">
                <h4>关联概览</h4>
                <div class="item-overview-tabs">
                  <button type="button" :class="{ active: activeContractOverviewTab === 'items' }" @click="activeContractOverviewTab = 'items'">硬件</button>
                  <button type="button" :class="{ active: activeContractOverviewTab === 'software' }" @click="activeContractOverviewTab = 'software'">软件</button>
                  <button type="button" :class="{ active: activeContractOverviewTab === 'invoices' }" @click="activeContractOverviewTab = 'invoices'">单据</button>
                </div>
                <div class="item-overview-list">
                  <div v-for="entry in contractOverviewRows" :key="`contr-overview-${activeContractOverviewTab}-${entry.id}`" class="item-overview-row">
                    <span class="mono item-overview-index">{{ entry.index }}:</span>
                    <button class="item-overview-link quick-tip" type="button" :data-quick-tip="entry.tip" @click="openOverviewEntry(entry)">
                      <span>{{ entry.text }}</span>
                    </button>
                  </div>
                  <div v-if="contractOverviewRows.length === 0" class="muted-text">暂无关联记录</div>
                </div>

                <h4>备件</h4>
                <div class="asset-table-wrap">
                  <table :class="['asset-inner-table', 'contract-renewal-table', { 'has-actions': canWrite }]">
                    <thead>
                      <tr>
                        <th v-if="canWrite">操作</th>
                        <th>到期前</th>
                        <th>到期后</th>
                        <th>生效日期</th>
                        <th>备注</th>
                        <th>录入日期</th>
                        <th>录入人</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="(row, idx) in contractRenewals" :key="`contract-renewal-${idx}`">
                        <td v-if="canWrite"><button class="ghost-btn small-btn" type="button" @click="removeContractRenewalRow(idx)">删除</button></td>
                        <td><DateInput v-model="row.endDateBefore" /></td>
                        <td><DateInput v-model="row.endDateAfter" /></td>
                        <td><DateInput v-model="row.effectiveDate" /></td>
                        <td><input v-model="row.notes" type="text" /></td>
                        <td><DateInput v-model="row.enteredDate" /></td>
                        <td>
                          <select v-model="row.enteredBy">
                            <option value=""></option>
                            <option
                              v-for="opt in getContractRenewalEnteredByOptions(row.enteredBy)"
                              :key="`contract-renewal-entered-by-${idx}-${opt.value}`"
                              :value="opt.value"
                            >
                              {{ opt.label }}
                            </option>
                          </select>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <button v-if="canWrite" class="ghost-btn small-btn" type="button" @click="addContractRenewalRow">新增备件</button>

                <h4>关联文件</h4>
                <div class="software-managed-file-list">
                  <div
                    v-for="file in managedLinkedFileRows"
                    :key="`contr-file-${String(file.id ?? '')}`"
                    class="software-managed-file-card"
                  >
                    <div class="software-managed-file-card-head">
                      <div class="software-managed-file-card-name">#{{ file.id }} {{ file.fileName || '-' }}</div>
                      <div class="software-managed-file-card-actions">
                        <button
                          v-if="canRemoveManagedLinkedFile(file)"
                          type="button"
                          class="field-link-icon quick-tip quick-tip-bottom-left"
                          :data-quick-tip="'解除关联，保存合同后生效。\n若文件是孤立的(没有其他内容与之相关联)，则会将其删除'"
                          @click="removeLinkedFileSelection(file.id)"
                        >
                          <img class="field-link-icon-image" src="/images/delete.png" alt="删除关联" />
                        </button>
                        <button
                          type="button"
                          class="field-link-icon quick-tip quick-tip-bottom-left"
                          :data-quick-tip="`在新窗口编辑文件 ${Number(file.id ?? 0) || '-'}`"
                          @click="openResourceEditInNewWindow('files', Number(file.id ?? 0))"
                        >
                          <img class="field-link-icon-image" src="/images/edit.png" alt="编辑文件" />
                        </button>
                        <button
                          type="button"
                          class="field-link-icon quick-tip quick-tip-bottom-left"
                          :data-quick-tip="`下载文件: ${file.fileName || '-'}`"
                          @click="downloadLinkedFileByID(file.id, file.fileName)"
                        >
                          <img class="field-link-icon-image" src="/images/down.png" alt="下载文件" />
                        </button>
                      </div>
                    </div>
                    <div class="software-managed-file-card-meta">
                      <div class="software-managed-file-card-field">
                        <span>类型</span>
                        <strong>{{ file.typeDesc || '-' }}</strong>
                      </div>
                      <div class="software-managed-file-card-field">
                        <span>日期</span>
                        <strong>{{ file.date || '-' }}</strong>
                      </div>
                      <div class="software-managed-file-card-field software-managed-file-card-title">
                        <span>标题</span>
                        <strong>{{ file.title || '-' }}</strong>
                      </div>
                    </div>
                  </div>
                  <div v-if="managedLinkedFileRows.length === 0" class="muted-text">暂无关联文件</div>
                </div>
              </section>
            </div>
          </section>

          <section v-show="activeContractTab === 'events'" class="item-tab-pane">
            <p v-if="!selectedId" class="muted-text">新增合同并保存后，可维护事件历史。</p>
            <template v-else>
              <div class="table-wrap item-log-wrap">
                <table class="item-log-table">
                  <thead>
                    <tr>
                      <th>编号</th>
                      <th>同组编号</th>
                      <th>开始日期</th>
                      <th>结束日期</th>
                      <th>描述</th>
                      <th v-if="canWrite">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="row in contractEvents" :key="`contract-event-${String(row.id ?? '')}`">
                      <td>{{ row.id ?? '-' }}</td>
                      <td>{{ row.siblingid || '-' }}</td>
                      <td>{{ toLocalDateText(row.startdate) }}</td>
                      <td>{{ toLocalDateText(row.enddate) }}</td>
                      <td>{{ row.description || '-' }}</td>
                      <td v-if="canWrite" class="actions-cell">
                        <button class="small-btn" type="button" @click="editContractEvent(row)">编辑</button>
                        <button class="small-btn danger" type="button" @click="removeContractEvent(row)">删除</button>
                      </td>
                    </tr>
                    <tr v-if="contractEventsLoading">
                      <td :colspan="canWrite ? 6 : 5">加载中...</td>
                    </tr>
                    <tr v-else-if="contractEvents.length === 0">
                      <td :colspan="canWrite ? 6 : 5">暂无事件记录</td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <form v-if="canWrite" class="settings-grid section-gap" @submit.prevent="saveContractEvent">
                <label>
                  <span>同组编号</span>
                  <input v-model="contractEventForm.siblingId" type="number" min="0" />
                </label>
                <label>
                  <span>开始日期 <sup class="req">*</sup></span>
                  <DateInput v-model="contractEventForm.startDate" />
                </label>
                <label>
                  <span>结束日期 <sup class="req">*</sup></span>
                  <DateInput v-model="contractEventForm.endDate" />
                </label>
                <label class="full-span">
                  <span>描述</span>
                  <textarea v-model="contractEventForm.description" />
                </label>
                <div class="inline-actions full-span">
                  <button :disabled="contractEventSaving" type="submit">
                    {{ contractEventSaving ? '保存中...' : editingContractEventId ? '保存事件修改' : '新增事件' }}
                  </button>
                  <button class="ghost-btn" type="button" @click="resetContractEventForm">重置</button>
                </div>
              </form>
            </template>
          </section>

          <section v-show="activeContractTab === 'itemLinks'" class="item-tab-pane">
            <div class="item-rel-panel">
              <div class="item-rel-header">
                <h4>硬件关联</h4>
                <input v-model="itemLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ contractItemRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('contractItems', 'id')">
                          编号 <span>{{ getNonSoftwareRelationSortIcon('contractItems', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('contractItems', 'itemType')">
                          类型 <span>{{ getNonSoftwareRelationSortIcon('contractItems', 'itemType') }}</span>
                        </button>
                      </th>
                      <th>
                        <button
                          type="button"
                          class="relation-sort-btn"
                          @click="toggleNonSoftwareRelationSort('contractItems', 'manufacturer')"
                        >
                          厂商 <span>{{ getNonSoftwareRelationSortIcon('contractItems', 'manufacturer') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('contractItems', 'model')">
                          型号 <span>{{ getNonSoftwareRelationSortIcon('contractItems', 'model') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('contractItems', 'label')">
                          标签 <span>{{ getNonSoftwareRelationSortIcon('contractItems', 'label') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('contractItems', 'dnsName')">
                          管理条线 <span>{{ getNonSoftwareRelationSortIcon('contractItems', 'dnsName') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('contractItems', 'principal')">
                          负责人 <span>{{ getNonSoftwareRelationSortIcon('contractItems', 'principal') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('contractItems', 'sn')">
                          设备序列号 <span>{{ getNonSoftwareRelationSortIcon('contractItems', 'sn') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="row in contractItemRelationRows" :key="`contr-item-rel-${row.id}`" :class="{ 'is-linked': isMultiSelected('itemLinks', row.id) }">
                      <td><input v-model="form.itemLinks" type="checkbox" :value="String(row.id)" /></td>
                      <td>
                        <button
                          class="relation-jump-btn quick-tip"
                          type="button"
                          :data-quick-tip="getSoftwareRelationIDTip('item', row.id)"
                          @click="openResourceEditInNewWindow('items', row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.itemType || '-' }}</td>
                      <td>{{ row.manufacturer || '-' }}</td>
                      <td>{{ row.model || '-' }}</td>
                      <td>{{ row.label || '-' }}</td>
                      <td>{{ row.dnsName || '-' }}</td>
                      <td>{{ row.principal || '-' }}</td>
                      <td>{{ row.sn || '-' }}</td>
                    </tr>
                    <tr v-if="contractItemRelationRows.length === 0">
                      <td colspan="9" class="item-rel-empty-cell">暂无可关联硬件</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>

          <section v-show="activeContractTab === 'softwareLinks'" class="item-tab-pane">
            <div class="item-rel-panel">
              <div class="item-rel-header">
                <h4>软件关联</h4>
                <input v-model="softwareLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ contractSoftwareRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('contractSoftware', 'id')">
                          编号 <span>{{ getNonSoftwareRelationSortIcon('contractSoftware', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button
                          type="button"
                          class="relation-sort-btn"
                          @click="toggleNonSoftwareRelationSort('contractSoftware', 'manufacturer')"
                        >
                          厂商 <span>{{ getNonSoftwareRelationSortIcon('contractSoftware', 'manufacturer') }}</span>
                        </button>
                      </th>
                      <th>
                        <button
                          type="button"
                          class="relation-sort-btn"
                          @click="toggleNonSoftwareRelationSort('contractSoftware', 'titleVersion')"
                        >
                          标题/版本 <span>{{ getNonSoftwareRelationSortIcon('contractSoftware', 'titleVersion') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="row in contractSoftwareRelationRows"
                      :key="`contr-soft-rel-${row.id}`"
                      :class="{ 'is-linked': isMultiSelected('softwareLinks', row.id) }"
                    >
                      <td><input v-model="form.softwareLinks" type="checkbox" :value="String(row.id)" /></td>
                      <td>
                        <button
                          class="relation-jump-btn quick-tip"
                          type="button"
                          :data-quick-tip="getSoftwareRelationIDTip('software', row.id)"
                          @click="openResourceEditInNewWindow('software', row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.manufacturer || '-' }}</td>
                      <td>{{ `${row.title || '-'} ${row.version || ''}`.trim() }}</td>
                    </tr>
                    <tr v-if="contractSoftwareRelationRows.length === 0">
                      <td colspan="4" class="item-rel-empty-cell">暂无可关联软件</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>

          <section v-show="activeContractTab === 'invoiceLinks'" class="item-tab-pane">
            <div class="item-rel-panel">
              <div class="item-rel-header">
                <h4>单据关联</h4>
                <input v-model="invoiceLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ contractInvoiceRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('contractInvoices', 'id')">
                          编号 <span>{{ getNonSoftwareRelationSortIcon('contractInvoices', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button
                          type="button"
                          class="relation-sort-btn"
                          @click="toggleNonSoftwareRelationSort('contractInvoices', 'vendor')"
                        >
                          供应商 <span>{{ getNonSoftwareRelationSortIcon('contractInvoices', 'vendor') }}</span>
                        </button>
                      </th>
                      <th>
                        <button
                          type="button"
                          class="relation-sort-btn"
                          @click="toggleNonSoftwareRelationSort('contractInvoices', 'number')"
                        >
                          订单编号 <span>{{ getNonSoftwareRelationSortIcon('contractInvoices', 'number') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('contractInvoices', 'date')">
                          日期 <span>{{ getNonSoftwareRelationSortIcon('contractInvoices', 'date') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="row in contractInvoiceRelationRows"
                      :key="`contr-inv-rel-${row.id}`"
                      :class="{ 'is-linked': isMultiSelected('invoiceLinks', row.id) }"
                    >
                      <td><input v-model="form.invoiceLinks" type="checkbox" :value="String(row.id)" /></td>
                      <td>
                        <button
                          class="relation-jump-btn quick-tip"
                          type="button"
                          :data-quick-tip="getSoftwareRelationIDTip('invoice', row.id)"
                          @click="openResourceEditInNewWindow('invoices', row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.vendor || '-' }}</td>
                      <td>{{ row.number || '-' }}</td>
                      <td>{{ row.date || '-' }}</td>
                    </tr>
                    <tr v-if="contractInvoiceRelationRows.length === 0">
                      <td colspan="5" class="item-rel-empty-cell">暂无可关联单据</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>

          <section v-show="activeContractTab === 'files'" class="item-tab-pane">
            <div class="item-rel-panel linked-file-panel">
              <div class="item-rel-header">
                <h4>关联文件</h4>
                <input v-model="fileLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ softwareFileRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'id')">
                          编号 <span>{{ getSoftwareRelationSortIcon('files', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'typeDesc')">
                          类型 <span>{{ getSoftwareRelationSortIcon('files', 'typeDesc') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'title')">
                          标题 <span>{{ getSoftwareRelationSortIcon('files', 'title') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'fileName')">
                          文件名 <span>{{ getSoftwareRelationSortIcon('files', 'fileName') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleSoftwareRelationSort('files', 'date')">
                          签署日期 <span>{{ getSoftwareRelationSortIcon('files', 'date') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="row in softwareFileRelationRows" :key="`contr-file-rel-${row.id}`" :class="{ 'is-linked': isMultiSelected('fileLinks', row.id) }">
                      <td><input :checked="isMultiSelected('fileLinks', row.id)" type="checkbox" @change="toggleUploadFileSelection(row.id, getInputChecked($event))" /></td>
                      <td>
                        <button
                          type="button"
                          class="relation-jump-btn quick-tip"
                          :data-quick-tip="getSoftwareRelationIDTip('file', row.id)"
                          @click="openSoftwareRelatedFileInNewWindow(row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.typeDesc || '-' }}</td>
                      <td>{{ row.title || '-' }}</td>
                      <td>{{ row.fileName || '-' }}</td>
                      <td>{{ row.date || '-' }}</td>
                    </tr>
                    <tr v-if="softwareFileRelationRows.length === 0">
                      <td colspan="6" class="item-rel-empty-cell">暂无可关联文件</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            <section v-if="canWrite" class="software-upload-panel linked-upload-panel">
              <h4>上传文件</h4>
              <div class="software-upload-grid">
                <label>
                  <span>标题 <sup class="req">*</sup></span>
                  <input v-model="contractUploadForm.title" type="text" />
                </label>
                <label>
                  <span>类型 <sup class="req">*</sup></span>
                  <select v-model="contractUploadForm.typeId">
                    <option value="">请选择</option>
                    <option v-for="opt in commonUploadTypeOptions" :key="`contract-upload-type-${opt.value}`" :value="String(opt.value)">
                      {{ opt.label }}
                    </option>
                  </select>
                </label>
                <label>
                  <span>签署日期 <sup class="req">*</sup></span>
                  <DateInput v-model="contractUploadForm.date" />
                </label>
                <div class="asset-field-row linked-upload-picker-row">
                  <span>选择文件 <sup class="req">*</sup></span>
                  <div class="location-floorplan-picker linked-upload-picker">
                    <button class="ghost-btn small-btn location-floorplan-picker-btn linked-upload-picker-btn" type="button" @click="openContractUploadPicker">
                      选择文件
                    </button>
                    <span :class="selectedContractUploadFileName ? 'location-floorplan-picker-name' : 'location-floorplan-picker-name muted-text'">
                      {{ selectedContractUploadFileName || '未选择文件' }}
                    </span>
                    <input ref="contractUploadInput" class="location-floorplan-input" type="file" @change="onContractUploadFileChange" />
                  </div>
                </div>
              </div>
              <div class="software-upload-actions">
                <button type="button" :disabled="contractUploading" @click="uploadLinkedFile('contracts')">
                  {{ contractUploading ? '上传中...' : '上传文件' }}
                </button>
                <span class="muted-text">上传后会自动写入“文件”菜单并关联到当前合同。</span>
              </div>
            </section>
          </section>

          <div class="item-form-actions">
            <button :disabled="saving" type="submit">{{ saving ? '提交中...' : selectedId ? '保存修改' : '创建' }}</button>
            <button class="ghost-btn" type="button" @click="drawerOpen = false">取消</button>
          </div>
        </template>

        <template v-else-if="isFileResource">
          <div class="item-tabs">
            <button
              v-for="tab in fileEditorTabs"
              :key="tab.key"
              type="button"
              class="item-tab-btn"
              :class="{ active: activeFileTab === tab.key }"
              @click="activeFileTab = tab.key"
            >
              {{ tab.label }}
            </button>
          </div>

          <section v-show="activeFileTab === 'fileData'" class="item-tab-pane">
            <div class="asset-grid-2">
              <section class="asset-block">
                <h4>文件属性</h4>
                <label><span>编号</span><input :value="selectedId ?? '-'" type="text" disabled /></label>
                <label><span>标题 <sup class="req">*</sup></span><input v-model="form.title" type="text" /></label>
                <label>
                  <span>类型 <sup class="req">*</sup></span>
                  <select v-model="form.typeId">
                    <option v-if="showFileEditorTypePlaceholder" value="">请选择</option>
                    <option v-for="opt in fileEditorTypeOptions" :key="`file-type-${opt.value}`" :value="String(opt.value)">
                      {{ opt.label }}
                    </option>
                  </select>
                </label>
                <label><span>签署日期 <sup class="req">*</sup></span><DateInput v-model="form.date" /></label>
                <div class="asset-field-row">
                  <span>文件名称</span>
                  <div v-if="selectedId && recordDetail?.fname" class="location-floorplan-link-wrap">
                    <button
                      class="item-overview-link location-floorplan-link quick-tip"
                      type="button"
                      data-quick-tip="下载当前文件"
                      @click="downloadSelectedFile"
                    >
                      {{ recordDetail?.fname }}
                    </button>
                  </div>
                </div>
                <label>
                  <span class="quick-tip file-association-tip" data-quick-tip="引用此文件的硬件、软件、合同总数">关联数</span>
                  <input :value="fileAssociationCount" type="text" disabled />
                </label>
                <label><span>上传人</span><input :value="fileUploadedByText" type="text" disabled /></label>
                <div class="asset-field-row">
                  <span>上传文件</span>
                  <div class="location-floorplan-picker">
                    <button class="ghost-btn small-btn location-floorplan-picker-btn" type="button" @click="openFileUploadPicker">选择文件</button>
                    <span :class="selectedFileUploadName ? 'location-floorplan-picker-name' : 'location-floorplan-picker-name muted-text'">
                      {{ selectedFileUploadName || '未选择文件' }}
                    </span>
                    <input ref="fileUploadInput" class="location-floorplan-input" type="file" :accept="fileUploadAccept" @change="handleFileUploadChange" />
                  </div>
                </div>
                <p v-if="isInvoiceTypeFile" class="muted-text">
                  该文件类型为“{{ invoiceFileTypeLabel }}”，按原系统逻辑仅通过“单据”功能维护关联。
                </p>
              </section>

              <section class="asset-block">
                <h4>关联概览</h4>
                <div class="item-overview-tabs">
                  <button type="button" :class="{ active: activeFileOverviewTab === 'items' }" @click="activeFileOverviewTab = 'items'">硬件</button>
                  <button type="button" :class="{ active: activeFileOverviewTab === 'invoices' }" @click="activeFileOverviewTab = 'invoices'">单据</button>
                  <button type="button" :class="{ active: activeFileOverviewTab === 'contracts' }" @click="activeFileOverviewTab = 'contracts'">合同</button>
                  <button type="button" :class="{ active: activeFileOverviewTab === 'software' }" @click="activeFileOverviewTab = 'software'">软件</button>
                </div>
                <div class="item-overview-list">
                  <div v-for="entry in fileOverviewRows" :key="`file-overview-${activeFileOverviewTab}-${entry.id}`" class="item-overview-row">
                    <span class="mono item-overview-index">{{ entry.index }}:</span>
                    <button class="item-overview-link quick-tip" type="button" :data-quick-tip="entry.tip" @click="openOverviewEntry(entry)">
                      <span>{{ entry.text }}</span>
                    </button>
                  </div>
                  <div v-if="fileOverviewRows.length === 0" class="muted-text">暂无关联记录</div>
                </div>
              </section>
            </div>
          </section>

          <section v-show="activeFileTab === 'itemLinks'" class="item-tab-pane">
            <div v-if="canEditNonInvoiceFileAssociations" class="item-rel-panel">
              <div class="item-rel-header">
                <h4>硬件关联</h4>
                <input v-model="itemLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ fileItemRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('fileItems', 'id')">
                          编号 <span>{{ getNonSoftwareRelationSortIcon('fileItems', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('fileItems', 'itemType')">
                          类型 <span>{{ getNonSoftwareRelationSortIcon('fileItems', 'itemType') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('fileItems', 'manufacturer')">
                          厂商 <span>{{ getNonSoftwareRelationSortIcon('fileItems', 'manufacturer') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('fileItems', 'model')">
                          型号 <span>{{ getNonSoftwareRelationSortIcon('fileItems', 'model') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('fileItems', 'label')">
                          标签 <span>{{ getNonSoftwareRelationSortIcon('fileItems', 'label') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('fileItems', 'dnsName')">
                          管理跳线 <span>{{ getNonSoftwareRelationSortIcon('fileItems', 'dnsName') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('fileItems', 'principal')">
                          负责人 <span>{{ getNonSoftwareRelationSortIcon('fileItems', 'principal') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('fileItems', 'sn')">
                          设备序列号 <span>{{ getNonSoftwareRelationSortIcon('fileItems', 'sn') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="row in fileItemRelationRows" :key="`file-item-rel-${row.id}`" :class="{ 'is-linked': isMultiSelected('itemLinks', row.id) }">
                      <td><input v-model="form.itemLinks" type="checkbox" :value="String(row.id)" /></td>
                      <td>
                        <button
                          class="relation-jump-btn quick-tip"
                          type="button"
                          :data-quick-tip="getSoftwareRelationIDTip('item', row.id)"
                          @click="openResourceEditInNewWindow('items', row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.itemType || '-' }}</td>
                      <td>{{ row.manufacturer || '-' }}</td>
                      <td>{{ row.model || '-' }}</td>
                      <td>{{ row.label || '-' }}</td>
                      <td>{{ row.dnsName || '-' }}</td>
                      <td>{{ row.principal || '-' }}</td>
                      <td>{{ row.sn || '-' }}</td>
                    </tr>
                    <tr v-if="fileItemRelationRows.length === 0" class="item-rel-empty-row">
                      <td colspan="9" class="item-rel-empty-cell">暂无可关联硬件</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
            <p v-else class="muted-text">该文件类型为“{{ invoiceFileTypeLabel }}”，不能在此关联硬件。</p>
          </section>

          <section v-show="activeFileTab === 'softwareLinks'" class="item-tab-pane">
            <div v-if="canEditNonInvoiceFileAssociations" class="item-rel-panel">
              <div class="item-rel-header">
                <h4>软件关联</h4>
                <input v-model="softwareLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ fileSoftwareRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('fileSoftware', 'id')">
                          编号 <span>{{ getNonSoftwareRelationSortIcon('fileSoftware', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('fileSoftware', 'manufacturer')">
                          厂商 <span>{{ getNonSoftwareRelationSortIcon('fileSoftware', 'manufacturer') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('fileSoftware', 'titleVersion')">
                          标题/版本 <span>{{ getNonSoftwareRelationSortIcon('fileSoftware', 'titleVersion') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="row in fileSoftwareRelationRows"
                      :key="`file-soft-rel-${row.id}`"
                      :class="{ 'is-linked': isMultiSelected('softwareLinks', row.id) }"
                    >
                      <td><input v-model="form.softwareLinks" type="checkbox" :value="String(row.id)" /></td>
                      <td>
                        <button
                          class="relation-jump-btn quick-tip"
                          type="button"
                          :data-quick-tip="getSoftwareRelationIDTip('software', row.id)"
                          @click="openResourceEditInNewWindow('software', row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.manufacturer || '-' }}</td>
                      <td>{{ `${row.title || '-'} ${row.version || ''}`.trim() }}</td>
                    </tr>
                    <tr v-if="fileSoftwareRelationRows.length === 0">
                      <td colspan="4" class="item-rel-empty-cell">暂无可关联软件</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
            <p v-else class="muted-text">该文件类型为“{{ invoiceFileTypeLabel }}”，不能在此关联软件。</p>
          </section>

          <section v-show="activeFileTab === 'contractLinks'" class="item-tab-pane">
            <div v-if="canEditNonInvoiceFileAssociations" class="item-rel-panel">
              <div class="item-rel-header">
                <h4>合同关联</h4>
                <input v-model="contractLinkFilter" class="item-rel-filter" placeholder="输入关键字筛选" />
                <span>共 {{ fileContractRelationRows.length }} 条</span>
              </div>
              <div class="table-wrap item-rel-table-wrap">
                <table class="item-rel-table">
                  <thead>
                    <tr>
                      <th>关联</th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('fileContracts', 'id')">
                          编号 <span>{{ getNonSoftwareRelationSortIcon('fileContracts', 'id') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('fileContracts', 'contractor')">
                          承包方 <span>{{ getNonSoftwareRelationSortIcon('fileContracts', 'contractor') }}</span>
                        </button>
                      </th>
                      <th>
                        <button type="button" class="relation-sort-btn" @click="toggleNonSoftwareRelationSort('fileContracts', 'title')">
                          标题 <span>{{ getNonSoftwareRelationSortIcon('fileContracts', 'title') }}</span>
                        </button>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="row in fileContractRelationRows"
                      :key="`file-contr-rel-${row.id}`"
                      :class="{ 'is-linked': isMultiSelected('contractLinks', row.id) }"
                    >
                      <td><input v-model="form.contractLinks" type="checkbox" :value="String(row.id)" /></td>
                      <td>
                        <button
                          class="relation-jump-btn quick-tip"
                          type="button"
                          :data-quick-tip="getSoftwareRelationIDTip('contract', row.id)"
                          @click="openResourceEditInNewWindow('contracts', row.id)"
                        >
                          {{ row.id }}
                        </button>
                      </td>
                      <td>{{ row.contractor || '-' }}</td>
                      <td>{{ row.title || '-' }}</td>
                    </tr>
                    <tr v-if="fileContractRelationRows.length === 0">
                      <td colspan="4" class="item-rel-empty-cell">暂无可关联合同</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
            <p v-else class="muted-text">该文件类型为“{{ invoiceFileTypeLabel }}”，不能在此关联合同。</p>
          </section>

          <div class="item-form-actions">
            <button :disabled="saving" type="submit">{{ saving ? '提交中...' : selectedId ? '保存修改' : '创建' }}</button>
            <button class="ghost-btn" type="button" @click="drawerOpen = false">取消</button>
          </div>
        </template>

        <template v-else-if="isAgentResource">
          <div class="asset-grid-2">
            <section class="asset-block">
              <h4>代理属性</h4>
              <label><span>编号</span><input :value="selectedId ?? '-'" type="text" disabled /></label>
              <label><span>名称 <sup class="req">*</sup></span><input v-model="form.title" type="text" /></label>
              <label>
                <span>类型 <sup class="req">*</sup></span>
                <span
                  class="field-tip-wrap quick-tip"
                  data-quick-tip="按住 Ctrl 可多选；供应商/采购方用于单据与合同，软件厂商用于软件，硬件厂商用于硬件，承包方用于合同。"
                >
                  <select v-model="form.types" multiple>
                    <option v-for="opt in getOptionsByFieldKey('types')" :key="`agent-type-${opt.value}`" :value="String(opt.value)">
                      {{ opt.label }}
                    </option>
                  </select>
                </span>
              </label>
              <label>
                <span>合同信息</span>
                <span class="field-tip-wrap quick-tip" data-quick-tip="地址, 电话号码, 其余信息">
                  <textarea v-model="form.contactInfo" />
                </span>
              </label>
            </section>

            <section class="asset-block">
              <h4>关联概览</h4>
              <div class="item-overview-tabs">
                <button type="button" :class="{ active: activeAgentOverviewTab === 'items' }" @click="activeAgentOverviewTab = 'items'">硬件</button>
                <button type="button" :class="{ active: activeAgentOverviewTab === 'software' }" @click="activeAgentOverviewTab = 'software'">软件</button>
                <button type="button" :class="{ active: activeAgentOverviewTab === 'invoicesVendor' }" @click="activeAgentOverviewTab = 'invoicesVendor'">单据(供应方)</button>
                <button type="button" :class="{ active: activeAgentOverviewTab === 'invoicesBuyer' }" @click="activeAgentOverviewTab = 'invoicesBuyer'">单据(采购方)</button>
              </div>
              <div class="item-overview-list">
                <div v-for="entry in agentOverviewRows" :key="`agent-overview-${activeAgentOverviewTab}-${entry.id}`" class="item-overview-row">
                  <span class="mono item-overview-index">{{ entry.index }}:</span>
                  <button class="item-overview-link quick-tip" type="button" :data-quick-tip="entry.tip" @click="openOverviewEntry(entry)">
                    <span>{{ entry.text }}</span>
                  </button>
                </div>
                <div v-if="agentOverviewRows.length === 0" class="muted-text">暂无关联记录</div>
              </div>
            </section>
          </div>

          <div class="asset-grid-2">
            <section class="asset-block">
              <h4>合同</h4>
              <div class="asset-table-wrap">
                <table class="asset-inner-table">
                  <thead>
                    <tr>
                      <th class="agent-contact-delete-col">操作</th>
                      <th>姓名</th>
                      <th>电话</th>
                      <th>邮箱</th>
                      <th>角色</th>
                      <th>备注</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="(row, idx) in agentContacts" :key="`agent-contact-${idx}`">
                      <td class="agent-contact-delete-col"><button class="ghost-btn small-btn" type="button" @click="removeAgentContactRow(idx)">删除</button></td>
                      <td><input v-model="row.name" type="text" /></td>
                      <td><input v-model="row.phones" type="text" /></td>
                      <td><input v-model="row.email" type="text" /></td>
                      <td><input v-model="row.role" type="text" /></td>
                      <td><input v-model="row.comments" type="text" /></td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <button class="ghost-btn small-btn" type="button" @click="addAgentContactRow">新增联系人</button>
            </section>

            <section class="asset-block">
              <div class="agent-urls-head">
                <h4>URLs</h4>
                <span class="agent-urls-tip">提示：当描述填写为“service”时，硬件修改页面会显示此链接。</span>
              </div>
              <div class="asset-table-wrap">
                <table class="asset-inner-table">
                  <thead>
                    <tr>
                      <th>操作</th>
                      <th>描述</th>
                      <th>URL</th>
                      <th>跳转</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="(row, idx) in agentURLs" :key="`agent-url-${idx}`">
                      <td><button class="ghost-btn small-btn" type="button" @click="removeAgentURLRow(idx)">删除</button></td>
                      <td><input v-model="row.description" type="text" /></td>
                      <td><input v-model="row.url" type="text" /></td>
                      <td class="agent-url-jump-cell">
                        <a
                          v-if="String(row.url || '').trim()"
                          class="agent-url-link quick-tip"
                          :href="String(row.url)"
                          target="_blank"
                          rel="noopener"
                          data-quick-tip="在新窗口打开该链接"
                        >
                          GO
                        </a>
                        <span v-else class="muted-text">-</span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <button class="ghost-btn small-btn" type="button" @click="addAgentURLRow">新增URL</button>
            </section>
          </div>

          <div class="item-form-actions">
            <button :disabled="saving" type="submit">{{ saving ? '提交中...' : selectedId ? '保存修改' : '创建' }}</button>
            <button class="ghost-btn" type="button" @click="drawerOpen = false">取消</button>
          </div>
        </template>

        <template v-else-if="isLocationResource">
          <div class="asset-grid-3 location-asset-grid">
            <section class="asset-block location-main-block">
              <h4>地点属性</h4>
              <label><span>编号</span><input :value="selectedId ?? '-'" type="text" disabled /></label>
              <label><span>建筑名称 <sup class="req">*</sup></span><input v-model="form.name" type="text" /></label>
              <label><span>楼层 <sup class="req">*</sup></span><input v-model="form.floor" type="text" /></label>
              <div class="asset-field-row">
                <span>文件名称</span>
                <div v-if="selectedId && locationFloorplanName" class="location-floorplan-link-wrap">
                  <button
                    class="item-overview-link location-floorplan-link quick-tip"
                    type="button"
                    data-quick-tip="在新窗口预览平面图"
                    @click="openLocationFloorplanInNewWindow"
                  >
                    {{ locationFloorplanName }}
                  </button>
                </div>
              </div>
              <label><span>关联(硬件/机架)</span><input :value="locationAssociationSummary" type="text" disabled /></label>
              <div class="asset-field-row">
                <span>建筑平面图</span>
                <div class="location-floorplan-picker">
                  <button class="ghost-btn small-btn location-floorplan-picker-btn" type="button" @click="openLocationFloorplanPicker">选择文件</button>
                  <span :class="selectedLocationFloorplanName ? 'location-floorplan-picker-name' : 'location-floorplan-picker-name muted-text'">
                    {{ selectedLocationFloorplanName || '未选择文件' }}
                  </span>
                  <input ref="locationFloorplanInput" class="location-floorplan-input" type="file" :accept="locationFloorplanAccept" @change="handleLocationFloorplanChange" />
                </div>
              </div>
              <p class="muted-text">如果你选择了新的文件，它将替换当前文件，同时保留它的关联关系。</p>
            </section>

            <section class="asset-block location-areas-block">
              <h4>区域：房间，办公室</h4>
              <div class="table-wrap item-log-wrap">
                <table class="item-log-table">
                  <thead>
                    <tr>
                      <th>编号</th>
                      <th>区域名称</th>
                      <th v-if="canWrite">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="row in sortedLocationAreas" :key="`loc-area-${String(row.id ?? '')}`">
                      <td>{{ row.id ?? '-' }}</td>
                      <td>{{ row.areaname ?? '-' }}</td>
                      <td v-if="canWrite" class="actions-cell">
                        <button class="small-btn" type="button" @click="editLocationArea(row)">编辑</button>
                        <button class="small-btn danger" type="button" @click="removeLocationArea(row)">删除</button>
                      </td>
                    </tr>
                    <tr v-if="locationAreasLoading">
                      <td :colspan="canWrite ? 3 : 2">加载中...</td>
                    </tr>
                    <tr v-else-if="sortedLocationAreas.length === 0">
                      <td :colspan="canWrite ? 3 : 2">暂无区域数据</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <form v-if="canWrite" class="inline-form section-gap" @submit.prevent="saveLocationArea">
                <input v-model="locationAreaName" type="text" placeholder="请输入区域名称" />
                <button :disabled="locationAreaSaving" type="submit">
                  {{ locationAreaSaving ? '保存中...' : editingLocationAreaId ? '保存区域修改' : '新增区域' }}
                </button>
              </form>
            </section>

            <section class="asset-block location-overview-block">
              <h4>关联概览</h4>
              <div class="item-overview-tabs">
                <button type="button" :class="{ active: activeLocationOverviewTab === 'items' }" @click="activeLocationOverviewTab = 'items'">硬件</button>
                <button type="button" :class="{ active: activeLocationOverviewTab === 'racks' }" @click="activeLocationOverviewTab = 'racks'">机架</button>
              </div>
              <div class="item-overview-list">
                <div v-for="entry in locationOverviewRows" :key="`loc-overview-${activeLocationOverviewTab}-${entry.id}`" class="item-overview-row">
                  <span class="mono item-overview-index">{{ entry.index }}:</span>
                  <button class="item-overview-link quick-tip" type="button" :data-quick-tip="entry.tip" @click="openOverviewEntry(entry)">
                    <span>{{ entry.text }}</span>
                  </button>
                </div>
                <div v-if="locationOverviewRows.length === 0" class="muted-text">暂无关联记录</div>
              </div>
            </section>

            <section class="asset-block location-floorplan-block">
              <h4>平面图预览</h4>
              <div class="location-floorplan-view">
                <img v-if="selectedId && locationFloorplanPreviewURL" :src="locationFloorplanPreviewURL" alt="地点平面图预览" />
                <div v-else class="muted-text">{{ locationFloorplanLoading ? '平面图加载中...' : '暂无平面图' }}</div>
              </div>
            </section>
          </div>

          <div class="item-form-actions">
            <button :disabled="saving" type="submit">{{ saving ? '提交中...' : selectedId ? '保存修改' : '创建' }}</button>
            <button class="ghost-btn" type="button" @click="drawerOpen = false">取消</button>
          </div>
        </template>

        <template v-else-if="isUserResource">
          <div class="asset-grid-2 user-asset-grid">
            <section class="asset-block">
              <h4>用户属性</h4>
              <label><span>编号</span><input :value="selectedId ?? '-'" type="text" disabled /></label>
              <label><span>用户名 <sup class="req">*</sup></span><input v-model="form.username" type="text" /></label>
              <label>
                <span>类型 <sup class="req">*</sup></span>
                <select v-model="form.userType">
                  <option v-for="opt in getOptionsByFieldKey('userType')" :key="`user-type-${opt.value}`" :value="String(opt.value)">
                    {{ opt.label }}
                  </option>
                </select>
              </label>
              <label><span>用户描述</span><input v-model="form.userDesc" type="text" /></label>
              <label>
                <span>密码 <sup v-if="!selectedId" class="req">*</sup></span>
                <input v-model="form.password" type="password" autocomplete="new-password" />
              </label>
              <label><span>硬件数</span><input :value="userItemCount" type="text" disabled /></label>
              <ul class="asset-tip-list">
                <li><strong>能够网页登录、负责硬件的用户</strong></li>
                <li><sup>1</sup> 编辑用户时，密码留空表示不修改密码</li>
                <li><sup>2</sup> 添加用户时，密码不能为空</li>
              </ul>
            </section>

            <section class="asset-block">
              <h4>关联硬件</h4>
              <div class="item-overview-list">
                <div v-for="entry in userOverviewRows" :key="`user-overview-${entry.id}`" class="item-overview-row">
                  <span class="mono item-overview-index">{{ entry.index }}:</span>
                  <button class="item-overview-link quick-tip" type="button" :data-quick-tip="entry.tip" @click="openOverviewEntry(entry)">
                    <span>{{ entry.text }}</span>
                  </button>
                </div>
                <div v-if="userOverviewRows.length === 0" class="muted-text">暂无关联硬件</div>
              </div>
            </section>
          </div>

          <div class="item-form-actions">
            <button :disabled="saving" type="submit">{{ saving ? '提交中...' : selectedId ? '保存修改' : '创建' }}</button>
            <button class="ghost-btn" type="button" @click="drawerOpen = false">取消</button>
          </div>
        </template>

        <template v-else-if="isRackResource">
          <div class="asset-grid-2 rack-asset-grid">
            <section class="asset-block">
              <h4>机架属性</h4>
              <label><span>编号</span><input :value="selectedId ?? '-'" type="text" disabled /></label>
              <label><span>高度(U)* <sup class="req">*</sup></span><input v-model.number="form.uSize" type="number" min="1" /></label>
              <label>
                <span>编号方向</span>
                <select v-model="form.revNums">
                  <option value="0">1=Bottom</option>
                  <option value="1">1=Top</option>
                </select>
              </label>
              <label><span>标签 <sup class="req">*</sup></span><input v-model="form.label" type="text" /></label>
              <label><span>深度(mm) <sup class="req">*</sup></span><input v-model.number="form.depth" type="number" min="1" /></label>
              <label><span>型号</span><input v-model="form.model" type="text" /></label>
              <label>
                <span>地点 <sup class="req">*</sup></span>
                <select v-model="form.locationId">
                  <option value="">请选择</option>
                  <option v-for="opt in getOptionsByFieldKey('locationId')" :key="`rack-loc-${opt.value}`" :value="String(opt.value)">
                    {{ opt.label }}
                  </option>
                </select>
              </label>
              <label>
                <span>区域</span>
                <select v-model="form.locAreaId">
                  <option value="">请选择</option>
                  <option v-for="opt in getOptionsByFieldKey('locAreaId')" :key="`rack-area-${opt.value}`" :value="String(opt.value)">
                    {{ opt.label }}
                  </option>
                </select>
              </label>
              <label><span>硬件</span><input :value="rackPopulation" type="text" disabled /></label>
              <div class="rack-occupation-row">
                <span>在用</span>
                <div class="rack-occupation-value">
                  <div class="rack-occupation-tip quick-tip" :data-quick-tip="`${rackOccupationUnits}/${rackTotalUnits}U`">
                    <div class="rack-occupation-bar">
                      <div class="rack-occupation-fill" :style="{ width: `${rackOccupationPercent}%` }" />
                    </div>
                  </div>
                </div>
              </div>
              <label><span>注释</span><textarea v-model="form.comments" /></label>
            </section>

            <section class="asset-block rack-view-block">
              <h4>机架晟图</h4>
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
                        <th title="机架单元">RU</th>
                        <th>前侧</th>
                        <th>中部</th>
                        <th>后侧</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="row in rackViewData.rows" :key="`rack-view-${row.unit}`">
                        <td class="rack-view-ru">{{ row.unit }}</td>
                        <td
                          v-for="cell in row.cells"
                          :key="cell.key"
                          :colspan="cell.colspan"
                          :rowspan="cell.rowspan"
                          :class="['rack-view-cell', cell.kind === 'empty' ? 'rack-view-empty' : 'rack-view-occupied']"
                          :style="cell.style"
                        >
                          <template v-if="cell.kind === 'item' && cell.itemID">
                            <button
                              class="rack-view-link quick-tip"
                              type="button"
                              :data-quick-tip="cell.tip || undefined"
                              @click="openResourceEditInNewWindow('items', cell.itemID)"
                            >
                              <span class="rack-view-title">{{ cell.title }}</span>
                              <span v-if="cell.subtitle" class="rack-view-subtitle">{{ cell.subtitle }}</span>
                            </button>
                          </template>
                          <span v-else>&nbsp;</span>
                        </td>
                      </tr>
                    </tbody>
                    <tfoot>
                      <tr>
                        <td colspan="4" class="rack-view-base" />
                      </tr>
                      <tr>
                        <td class="rack-view-wheel-spacer" />
                        <td class="rack-view-wheel"><img src="/images/rackwheel.png" alt="机架底轮" /></td>
                        <td class="rack-view-wheel-spacer" />
                        <td class="rack-view-wheel"><img src="/images/rackwheel.png" alt="机架底轮" /></td>
                      </tr>
                    </tfoot>
                  </table>
                </div>

                <div v-if="rackViewData.moreItems.length > 0" class="rack-view-side-note">
                  <h5>已分配到该机架但未设置机架位置、深度位或高度的硬件</h5>
                  <ul>
                    <li v-for="row in rackViewData.moreItems" :key="`rack-more-${row.id}`">
                      <button class="rack-view-side-link" type="button" @click="openResourceEditInNewWindow('items', row.id)">
                        硬件 {{ row.id }}：{{ row.manufacturer || '-' }} {{ row.model || '-' }} {{ row.label || '' }}
                      </button>
                    </li>
                  </ul>
                </div>
                <div v-if="rackViewData.warnings.length > 0" class="rack-view-warning-list">
                  <p v-for="warning in rackViewData.warnings" :key="warning">{{ warning }}</p>
                </div>
              </div>
              <div v-else class="muted-text">
                {{ rackTotalUnits > 0 ? '当前机架暂无可显示的晟图内容' : '请先填写机架高度后查看晟图' }}
              </div>
            </section>
          </div>

          <div class="item-form-actions">
            <button :disabled="saving" type="submit">{{ saving ? '提交中...' : selectedId ? '保存修改' : '创建' }}</button>
            <button class="ghost-btn" type="button" @click="drawerOpen = false">取消</button>
          </div>
        </template>

        <template v-else>
          <label v-for="field in resource.fields" :key="field.key">
            <span>{{ field.label }}</span>

            <input
              v-if="['text', 'number', 'date', 'password'].includes(field.type)"
              :type="field.type === 'textarea' ? 'text' : field.type"
              v-model="form[field.key]"
            />

            <textarea v-else-if="field.type === 'textarea'" v-model="form[field.key]" />

            <select v-else-if="field.type === 'select'" v-model="form[field.key]">
              <option value="">请选择</option>
              <option v-for="opt in getFieldOptions(field)" :key="`${field.key}-${opt.value}`" :value="String(opt.value)">
                {{ opt.label }}
              </option>
            </select>

            <select v-else-if="field.type === 'multiselect'" multiple v-model="form[field.key]">
              <option v-for="opt in getFieldOptions(field)" :key="`${field.key}-${opt.value}`" :value="String(opt.value)">
                {{ opt.label }}
              </option>
            </select>

            <input
              v-else-if="field.type === 'file'"
              type="file"
              @change="form[field.key] = ($event.target as HTMLInputElement)?.files?.[0] ?? null"
            />
          </label>

          <button :disabled="saving" type="submit">{{ saving ? '提交中...' : selectedId ? '保存修改' : '创建' }}</button>
        </template>
        </form>
      </aside>
    </div>

    <div v-if="confirmOpen" class="dialog-mask">
      <section class="drawer modal-narrow" role="dialog" aria-modal="true">
        <div class="drawer-header">
          <h3>删除确认</h3>
          <button
            class="dialog-close-btn quick-tip"
            type="button"
            aria-label="关闭"
            data-quick-tip="关闭"
            @click="closeDeleteConfirm"
          >
            ×
          </button>
        </div>
        <div class="drawer-form">
          <p class="delete-confirm-text">{{ deleteConfirmMessage }}</p>
          <div class="inline-actions">
            <button class="danger" type="button" :disabled="deleting" @click="confirmDelete">
              {{ deleting ? '删除中...' : '确认删除' }}
            </button>
            <button class="ghost-btn" type="button" :disabled="deleting" @click="closeDeleteConfirm">取消</button>
          </div>
        </div>
      </section>
    </div>

    <Teleport to="body">
      <div
        v-if="uploadTabTip.visible"
        class="global-upload-tab-tip"
        :class="`is-${uploadTabTip.placement}`"
        :style="{ left: `${uploadTabTip.left}px`, top: `${uploadTabTip.top}px` }"
      >
        <div class="global-upload-tab-tip__body">{{ uploadTabTip.text }}</div>
      </div>
    </Teleport>
  </section>

  <section class="page-shell" v-else>
    <p class="error-text">资源不存在</p>
  </section>
</template>

<style scoped>
.warranty-good {
  color: #15803d;
  font-weight: 600;
}

.warranty-expired {
  color: #be123c;
  font-weight: 600;
}

.sortable-th {
  padding: 0;
}

.th-sort-btn {
  all: unset;
  box-sizing: border-box;
  width: 100%;
  padding: 10px 12px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  cursor: pointer;
  user-select: none;
}

.th-sort-btn.quick-tip {
  position: relative;
}

.th-sort-icon {
  color: #7b93a8;
  font-size: 12px;
  line-height: 1;
}

.sortable-th.is-active .th-sort-icon {
  color: #2f7fba;
}

.software-installed-id {
  display: inline-block;
  padding: 0 4px;
  border-radius: 3px;
  margin-right: 4px;
  font-weight: 700;
}

.software-installed-list {
  display: grid;
  gap: 4px;
  align-items: start;
  justify-items: start;
}

.software-installed-entry {
  min-width: 0;
}

.software-installed-link {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0 4px;
  min-width: 0;
  color: #1f5f95;
  text-decoration: underline;
}

.software-installed-link:hover {
  color: #0b4370;
}

.software-invoice-display-list {
  display: grid;
  gap: 6px;
  justify-items: start;
}

.software-invoice-display-entry {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.software-invoice-file-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.software-invoice-file-btn {
  width: 28px;
  height: 28px;
  padding: 0;
}

.software-installed-status-stored {
  background: #178f2f;
  color: #efefef;
}

.software-installed-status-defective {
  background: #cf1f1f;
  color: #fff;
}

.software-installed-status-obsolete {
  background: #cecece;
  color: #2f2f2f;
}

.resource-table td.actions-cell {
  display: table-cell;
  text-align: center;
  vertical-align: middle;
}

.resource-table .selection-col {
  width: 44px;
  min-width: 44px;
  text-align: center;
  vertical-align: middle;
}

.table-selection-meta {
  color: #315072;
  font-size: 0.95rem;
}

.resource-table-agents th:nth-child(4) {
  width: 320px;
  min-width: 320px;
}

.resource-table-agents .agent-contactinfo-cell {
  width: 320px;
  min-width: 320px;
  max-width: 320px;
  white-space: normal;
  word-break: break-word;
  overflow-wrap: anywhere;
  line-height: 1.55;
}

.resource-table-files .file-links-zero-cell {
  background: linear-gradient(180deg, #fff3f2 0%, #ffe4e1 100%);
  color: #b45309;
  font-weight: 700;
}

.row-actions {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  flex-wrap: wrap;
}

.agent-type-list {
  display: inline-flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 4px;
}

.agent-type-badge {
  display: inline-flex;
  align-items: center;
  padding: 1px 6px;
  border-radius: 999px;
  font-size: 12px;
  line-height: 1.4;
  border: 1px solid transparent;
}

.agent-type-buyer {
  color: #b00000;
  border-color: #f6c4c4;
  background: #fff3f3;
}

.agent-type-software {
  color: #7a2e9f;
  border-color: #e6cdf3;
  background: #faf5ff;
}

.agent-type-hardware {
  color: #6d3c00;
  border-color: #ecd8bf;
  background: #fff8ef;
}

.agent-type-vendor {
  color: #0f7e2f;
  border-color: #bfe8ca;
  background: #f1fdf5;
}

.agent-type-contractor {
  color: #2b4fcc;
  border-color: #c6d2ff;
  background: #f3f6ff;
}

.agent-type-empty {
  color: #6b7f92;
  border-color: #d5dee6;
  background: #f6f9fc;
}

.agent-contacts-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.agent-contacts-table td {
  border: 0;
  border-bottom: 1px dotted #b8c3cf;
  padding: 4px 6px;
  text-align: left;
  font-size: 0.86rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.agent-contact-delete-col {
  min-width: 84px;
  white-space: nowrap;
}

.agent-contact-empty {
  color: #6b7f92;
}

.agent-contact-summary {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: stretch;
}

.agent-contact-summary-line {
  padding: 4px 6px;
  border-bottom: 1px dotted #b8c3cf;
  line-height: 1.5;
  text-align: left;
  white-space: normal;
  word-break: break-word;
}

.agent-contact-summary-line:last-child {
  border-bottom: 0;
}

.rack-occupation-cell {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 150px;
}

.rack-usage-track {
  width: 70px;
  border: 1px solid #6f7f90;
  background: #f6f7f9;
  height: 12px;
  border-radius: 2px;
  overflow: hidden;
}

.rack-usage-fill {
  height: 100%;
  background: #8ece03;
}

.rack-usage-text {
  color: #35506a;
  font-size: 0.82rem;
}

.table-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
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
  min-width: 96px;
}

.table-meta {
  color: #4b647b;
  font-size: 0.92rem;
}

.table-pagination {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}

.page-btn-active {
  background: linear-gradient(180deg, #2f7fba, #1f6ca7);
  border-color: #1f6ca7;
}

.delete-confirm-text {
  margin: 0;
  text-align: center;
}

.drawer-mask {
  position: fixed;
  inset: 0;
  z-index: 60;
  background: rgba(15, 23, 42, 0.42);
  display: grid;
  place-items: center;
  padding: 18px;
}

.drawer-close-btn {
  width: 32px;
  height: 32px;
  border-radius: 999px;
  border: 1px solid #c8d2dd;
  background: #fff;
  color: #1f3a56;
  font-size: 22px;
  line-height: 1;
  font-weight: 500;
  padding: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  box-shadow: none;
}

.drawer-close-btn:hover {
  background: #f1f5f9;
  transform: none;
}

.item-drawer {
  width: min(1380px, calc(100vw - 36px));
}

.wide-drawer {
  width: min(1180px, calc(100vw - 36px));
}

.drawer-form-item {
  padding: 10px;
  gap: 10px;
  align-content: start;
}

.drawer-form-complex {
  padding: 10px;
  gap: 10px;
  align-content: start;
}

.drawer-form-software {
  overflow: hidden;
  display: grid;
  grid-template-rows: minmax(0, 1fr) auto;
}

.drawer-form-tabbed {
  overflow: visible;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  align-content: stretch;
}

.drawer-form-tabbed > .item-tabs {
  grid-row: 1;
}

.drawer-form-tabbed > .item-tab-pane {
  grid-row: 2;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  align-content: start;
}

.drawer-form-tabbed > .item-tab-pane::-webkit-scrollbar {
  width: 8px;
}

.drawer-form-tabbed > .item-tab-pane::-webkit-scrollbar-track {
  background: rgba(16, 42, 67, 0.08);
}

.drawer-form-tabbed > .item-tab-pane::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: rgba(16, 42, 67, 0.3);
}

.drawer-form-tabbed > .item-tab-pane::-webkit-scrollbar-thumb:hover {
  background: rgba(16, 42, 67, 0.44);
}

.software-editor-shell {
  min-height: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  gap: 0;
}

.software-editor-scroll {
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  display: grid;
  align-content: start;
  margin-top: 10px;
  padding-right: 4px;
}

.software-editor-scroll::-webkit-scrollbar {
  width: 8px;
}

.software-editor-scroll::-webkit-scrollbar-track {
  background: rgba(16, 42, 67, 0.08);
}

.software-editor-scroll::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: rgba(16, 42, 67, 0.3);
}

.software-editor-scroll::-webkit-scrollbar-thumb:hover {
  background: rgba(16, 42, 67, 0.44);
}

.asset-grid-2 {
  display: grid;
  grid-template-columns: repeat(2, minmax(320px, 1fr));
  gap: 10px;
}

.rack-asset-grid {
  grid-template-columns: minmax(320px, 360px) minmax(0, 1fr);
  align-items: start;
}

.user-asset-grid {
  grid-template-columns: minmax(280px, 0.94fr) minmax(320px, 1.06fr);
}

.software-data-grid {
  grid-template-columns: minmax(320px, 0.9fr) minmax(420px, 1.1fr);
}

.software-attr-block {
  max-width: 540px;
}

.software-side-block {
  min-width: 0;
}

.rack-view-block {
  min-width: 0;
}

.field-select-tip {
  display: block;
  width: 100%;
}

.asset-grid-3 {
  display: grid;
  grid-template-columns: minmax(320px, 1.05fr) minmax(360px, 1.25fr) minmax(300px, 0.9fr);
  gap: 10px;
}

.location-asset-grid {
  grid-template-columns: repeat(2, minmax(320px, 1fr));
  align-items: start;
}

.asset-block {
  border: 1px solid #c3c6cc;
  background: #f0f0f0;
  border-radius: 6px;
  padding: 10px;
  display: grid;
  gap: 8px;
  align-content: start;
}

.asset-block h4 {
  margin: 0;
  color: #0f4f80;
  border-bottom: 2px solid #4f96da;
  padding-bottom: 4px;
}

.agent-urls-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
}

.agent-urls-tip {
  color: #5f6f82;
  font-size: 0.85rem;
  line-height: 1.45;
  text-align: right;
  max-width: 420px;
}

.asset-block label {
  margin: 0;
  display: grid;
  grid-template-columns: 110px minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  font-weight: 500;
}

.asset-block label > span {
  text-align: right;
}

.asset-field-row {
  margin: 0;
  display: grid;
  grid-template-columns: 110px minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  font-weight: 500;
}

.asset-field-row > span {
  text-align: right;
}

.location-main-block label {
  grid-template-columns: 120px minmax(0, 1fr);
}

.location-main-block .asset-field-row {
  grid-template-columns: 120px minmax(0, 1fr);
}

.location-floorplan-picker {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.location-floorplan-picker-btn {
  flex: 0 0 auto;
  cursor: pointer;
}

.linked-upload-picker,
.invoice-upload-picker {
  min-height: 34px;
}

.linked-upload-picker-btn,
.invoice-upload-picker-btn {
  min-width: 92px;
  border: 1px solid #86acd0;
  background: linear-gradient(180deg, #ffffff 0%, #dbeeff 100%);
  color: #0f4f80;
  font-weight: 600;
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.9) inset;
}

.linked-upload-picker-btn:hover,
.invoice-upload-picker-btn:hover {
  background: linear-gradient(180deg, #ffffff 0%, #c8e3fb 100%);
  border-color: #5f93c6;
}

.location-floorplan-picker-name {
  min-width: 0;
  color: #2b4157;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-association-tip {
  display: inline-flex;
  justify-content: flex-end;
  align-items: center;
  width: 100%;
}

.file-association-tip[data-quick-tip]:hover::after {
  left: auto;
  right: -6px;
  bottom: calc(100% + 8px);
  transform: none;
  max-width: 220px;
  white-space: normal;
  text-align: left;
}

.file-association-tip[data-quick-tip]:hover::before {
  left: auto;
  right: 12px;
  bottom: calc(100% + 2px);
  transform: none;
}

.location-floorplan-input {
  display: none;
}

.license-type-group {
  display: flex;
  align-items: center;
  gap: 20px;
  flex-wrap: wrap;
  min-height: 36px;
}

.license-type-group > label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
}

.field-link-icon-wrap {
  display: inline-flex;
  width: 20px;
  height: 20px;
  flex: 0 0 20px;
  pointer-events: none;
}

.asset-tip-list {
  margin: 2px 0 0;
  padding-left: 18px;
  color: #35556f;
  display: grid;
  gap: 4px;
}

.field-label-with-icon {
  display: inline-flex;
  justify-content: flex-end;
  align-items: center;
  gap: 6px;
  width: 100%;
  pointer-events: none;
}

.field-link-icon {
  width: 20px;
  height: 20px;
  border: 1px solid #9db8d2;
  border-radius: 4px;
  background: #f5faff;
  color: #2d6ea4;
  font-size: 12px;
  line-height: 1;
  padding: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  flex: 0 0 20px;
  pointer-events: auto;
}

.field-link-icon:hover {
  background: #e5f0ff;
}

.field-link-icon-image {
  display: block;
  width: 12px;
  height: 12px;
}

.field-link-icon:disabled {
  opacity: 0.45;
  cursor: not-allowed;
  background: #eef3f8;
}

.field-tip-wrap {
  display: block;
  width: 100%;
}

.contract-textarea-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px 40px;
}

.contract-textarea-grid label {
  grid-template-columns: 64px minmax(0, 1fr);
  align-items: start;
}

.contract-description-offset {
  padding-left: 36px;
}

.contract-description-offset > label {
  width: 125%;
  gap: 0px 18px;
}

.contract-comments-offset {
  padding-left: 0px;
}

.contract-textarea-grid textarea {
  min-height: 140px;
  width: 100%;
}

.asset-table-wrap {
  border: 1px solid #d8e5f1;
  background: #fff;
  max-height: 260px;
  overflow: auto;
}

.asset-inner-table {
  min-width: 720px;
}

.asset-inner-table th,
.asset-inner-table td {
  padding: 6px;
}

.contract-renewal-table {
  min-width: 1180px;
}

.contract-renewal-table.has-actions th:nth-child(1),
.contract-renewal-table.has-actions td:nth-child(1) {
  min-width: 104px;
  white-space: nowrap;
}

.contract-renewal-table.has-actions th:nth-child(2),
.contract-renewal-table.has-actions td:nth-child(2),
.contract-renewal-table.has-actions th:nth-child(3),
.contract-renewal-table.has-actions td:nth-child(3),
.contract-renewal-table.has-actions th:nth-child(4),
.contract-renewal-table.has-actions td:nth-child(4),
.contract-renewal-table:not(.has-actions) th:nth-child(1),
.contract-renewal-table:not(.has-actions) td:nth-child(1),
.contract-renewal-table:not(.has-actions) th:nth-child(2),
.contract-renewal-table:not(.has-actions) td:nth-child(2),
.contract-renewal-table:not(.has-actions) th:nth-child(3),
.contract-renewal-table:not(.has-actions) td:nth-child(3) {
  min-width: 148px;
}

.contract-renewal-table.has-actions th:nth-child(5),
.contract-renewal-table.has-actions td:nth-child(5),
.contract-renewal-table:not(.has-actions) th:nth-child(4),
.contract-renewal-table:not(.has-actions) td:nth-child(4) {
  min-width: 280px;
}

.contract-renewal-table.has-actions th:nth-child(6),
.contract-renewal-table.has-actions td:nth-child(6),
.contract-renewal-table.has-actions th:nth-child(7),
.contract-renewal-table.has-actions td:nth-child(7),
.contract-renewal-table:not(.has-actions) th:nth-child(5),
.contract-renewal-table:not(.has-actions) td:nth-child(5),
.contract-renewal-table:not(.has-actions) th:nth-child(6),
.contract-renewal-table:not(.has-actions) td:nth-child(6) {
  min-width: 160px;
}

.agent-url-jump-cell {
  text-align: center;
  vertical-align: middle;
}

.agent-url-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 44px;
  padding: 2px 8px;
  border-radius: 6px;
  border: 1px solid #7fb0da;
  background: linear-gradient(180deg, #f3f9ff 0%, #deeeff 100%);
  color: #1c5f95;
  font-weight: 700;
  text-decoration: none;
}

.agent-url-link:hover {
  border-color: #5d9ad0;
  background: linear-gradient(180deg, #e9f4ff 0%, #d2e6fb 100%);
}

.item-tabs {
  display: flex;
  gap: 2px;
  overflow-x: auto;
  padding: 4px;
  border: 1px solid #7da7ca;
  border-radius: 8px 8px 0 0;
  background: linear-gradient(180deg, #3372bf 0%, #1f5ca4 100%);
}

.item-tab-btn {
  border-radius: 8px 8px 0 0;
  border: 1px solid #6f95bf;
  background: linear-gradient(180deg, #f7fbff 0%, #d9eafb 100%);
  color: #1d5f95;
  padding: 8px 16px;
  font-weight: 700;
  white-space: nowrap;
  margin-bottom: -1px;
  box-shadow: none;
}

.item-tab-btn.active {
  color: #ef6c00;
  background: linear-gradient(180deg, #ffffff 0%, #f5fbff 100%);
  border-color: #7ba8cc;
}


.global-upload-tab-tip {
  position: fixed;
  z-index: 14000;
  transform: translateX(-50%);
  max-width: min(260px, calc(100vw - 24px));
  pointer-events: none;
}

.global-upload-tab-tip::before {
  content: '';
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  border-style: solid;
}

.global-upload-tab-tip.is-below::before {
  top: -6px;
  border-width: 0 5px 6px 5px;
  border-color: transparent transparent rgba(12, 23, 42, 0.94) transparent;
}

.global-upload-tab-tip.is-above::before {
  bottom: -6px;
  border-width: 6px 5px 0 5px;
  border-color: rgba(12, 23, 42, 0.94) transparent transparent transparent;
}

.global-upload-tab-tip__body {
  background: rgba(12, 23, 42, 0.94);
  color: #fff;
  border-radius: 6px;
  padding: 6px 10px;
  font-size: 12px;
  line-height: 1.4;
  white-space: pre-line;
  text-align: left;
  box-shadow: 0 10px 24px rgba(12, 23, 42, 0.2);
}

.item-tab-pane {
  border: 1px solid #b7d0e6;
  border-top: 0;
  border-radius: 0 0 8px 8px;
  background: #f4f6f8;
  padding: 10px;
  min-height: 0;
  display: grid;
  gap: 10px;
}

.item-layout {
  display: grid;
  grid-template-columns: repeat(4, minmax(260px, 1fr));
  gap: 10px;
}

.hardware-item-layout {
  grid-template-columns: repeat(4, minmax(220px, 1fr));
}

.item-block {
  border: 1px solid #c3c6cc;
  background: #f0f0f0;
  border-radius: 6px;
  padding: 10px;
  display: grid;
  gap: 6px;
}

.item-block h4 {
  margin: 0;
  font-size: 2rem;
  color: #0f4d83;
  border-bottom: 2px solid #4c95d9;
  padding-bottom: 4px;
}

.item-block h5 {
  margin: 8px 0 0;
  font-size: 1.35rem;
  color: #0f4d83;
  border-bottom: 2px solid #4c95d9;
  padding-bottom: 3px;
}

.item-block label {
  margin: 0;
  display: grid;
  grid-template-columns: 108px minmax(0, 1fr);
  align-items: center;
  gap: 6px;
  font-weight: 500;
}


.item-block label > span {
  text-align: right;
}

.item-block input,
.item-block select,
.item-block textarea {
  border-radius: 0;
  border-color: #c5c8cc;
  background: #fff;
  padding: 4px 6px;
}

.item-block textarea {
  min-height: 68px;
  resize: vertical;
}

.item-radio-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 8px;
}

.item-radio-row > span {
  color: #2f5473;
  font-weight: 600;
  min-width: 96px;
  text-align: right;
}

.item-radio-row label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  grid-template-columns: none;
}

.item-inline-2 {
  display: grid;
  gap: 6px;
  grid-template-columns: 1fr 1fr;
}

.item-rack-placement-row {
  grid-template-columns: minmax(0, 1.35fr) minmax(0, 0.85fr);
  gap: 4px;
}

.item-rack-position-field {
  min-width: 0;
}

.item-rack-depth-field {
  grid-template-columns: 0 minmax(0, 1fr) !important;
  gap: 4px !important;
}

.item-rack-field {
  margin: 0;
  display: grid;
  grid-template-columns: 108px minmax(0, 1fr);
  align-items: center;
  gap: 6px;
  font-weight: 500;
}

.item-rack-select-label {
  display: inline-flex;
  justify-self: end;
  align-items: center;
  gap: 2px;
  width: auto;
  padding-right: 0;
  transform: none;
}

.item-rack-text-label {
  display: inline;
  margin: 0;
}

.item-rack-select-actions {
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

.item-rack-depth-field > span:first-child {
  width: 0;
  overflow: hidden;
}

.req {
  color: #dc2626;
}

.item-bottom-grid {
  display: grid;
  grid-template-columns: 1.15fr 0.85fr 1.2fr;
  gap: 10px;
}

.hardware-item-bottom-grid {
  grid-template-columns: repeat(3, minmax(240px, 1fr));
}

.item-rack-preview-block {
  grid-column: 1 / -1;
}

.item-rack-preview-meta {
  margin: 0;
  color: #234d72;
  font-weight: 600;
}

.item-rack-preview-wrap {
  margin-top: 0;
}

.item-bottom-block {
  border: 1px solid #bfd7ea;
  border-radius: 6px;
  padding: 10px;
  background: #fff;
  min-height: 190px;
  display: grid;
  align-content: start;
  gap: 8px;
}

.item-bottom-block h4 {
  margin: 0;
  color: #0f4f80;
  border-bottom: 2px solid #4f96da;
  padding-bottom: 4px;
}

.item-overview-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.item-overview-tabs button {
  background: #f7fbff;
  color: #1f5f8e;
  border: 1px solid #9fc0dd;
  border-radius: 6px;
  padding: 4px 10px;
  font-size: 0.9rem;
  box-shadow: none;
}

.item-overview-tabs button.active {
  background: #dfefff;
  border-color: #74a7d5;
  color: #124c79;
}

.item-overview-list,
.item-file-list {
  max-height: 186px;
  overflow: auto;
  border: 1px solid #d8e5f1;
  background: #f8fbff;
}

.item-overview-row {
  padding: 6px 8px;
  border-bottom: 1px solid #e1ebf4;
  display: flex;
  gap: 8px;
  align-items: center;
}

.item-file-row {
  justify-content: space-between;
  gap: 10px;
}

.item-file-main {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: 1 1 auto;
}

.item-file-main > span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-file-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: 0 0 auto;
}

.item-overview-index {
  min-width: 28px;
  text-align: right;
  color: #2a4f73;
}

.item-overview-link {
  all: unset;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: #1f5f95;
  cursor: pointer;
  text-decoration: underline;
}

.item-overview-link:hover {
  color: #0b4370;
}

.item-id-status-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 46px;
  padding: 4px 10px;
  border: 1px solid transparent;
  border-radius: 999px;
  background: #eef6ff;
  color: #173651;
  font-weight: 700;
  line-height: 1;
}

.item-invoice-desc-col {
  text-align: center;
}

.item-invoice-desc-col .relation-sort-btn {
  width: 100%;
  justify-content: center;
}

.item-invoice-desc-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  text-align: center;
}

.item-invoice-desc-text {
  white-space: pre-wrap;
  word-break: break-word;
}

.item-invoice-file-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
}

.item-invoice-file-list .location-floorplan-table-link {
  max-width: min(260px, 100%);
}

.location-floorplan-link {
  width: auto;
  max-width: 100%;
}

.location-floorplan-link-wrap {
  justify-self: start;
  width: max-content;
  max-width: 100%;
  display: flex;
}

.location-floorplan-table-link {
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  max-width: 100%;
  padding: 5px 12px;
  border: 1px solid #9fc1e4;
  border-radius: 999px;
  background: linear-gradient(180deg, #f8fcff 0%, #e3f1ff 100%);
  color: #18527f;
  font-weight: 700;
  line-height: 1;
  cursor: pointer;
  transition:
    border-color 0.18s ease,
    background 0.18s ease,
    transform 0.18s ease,
    box-shadow 0.18s ease;
}

.location-floorplan-table-link:hover {
  border-color: #6fa4d7;
  background: linear-gradient(180deg, #edf7ff 0%, #d6eaff 100%);
  box-shadow: 0 6px 14px rgba(56, 113, 168, 0.16);
  transform: translateY(-1px);
}

.location-floorplan-table-link:active {
  transform: translateY(0);
}

.location-floorplan-table-link:disabled {
  opacity: 0.66;
  cursor: default;
  box-shadow: none;
  transform: none;
}

.location-floorplan-table-link-icon {
  font-size: 0.92rem;
  flex: 0 0 auto;
}

.location-floorplan-table-link-text {
  display: block;
  max-width: min(240px, 100%);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-overview-row:nth-child(even) {
  background: #eef5ff;
}

.software-managed-file-list {
  max-height: 232px;
  overflow: auto;
  border: 1px solid #d8e5f1;
  background: #f8fbff;
  display: grid;
  gap: 10px;
  padding: 10px;
}

.software-managed-file-card {
  border: 1px solid #c7dced;
  border-radius: 8px;
  background: linear-gradient(180deg, #ffffff 0%, #eef6ff 100%);
  padding: 10px 12px;
  display: grid;
  gap: 10px;
  box-shadow: 0 6px 14px rgba(33, 91, 145, 0.08);
}

.software-managed-file-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.software-managed-file-card-name {
  min-width: 0;
  color: #1e4f78;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.software-managed-file-card-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: 0 0 auto;
}

.software-managed-file-card-meta {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.software-managed-file-card-field {
  display: grid;
  gap: 4px;
}

.software-managed-file-card-field > span {
  color: #5f7387;
  font-size: 0.78rem;
  letter-spacing: 0.02em;
}

.software-managed-file-card-field > strong {
  color: #173e60;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.software-managed-file-card-title {
  grid-column: span 1;
}

.location-floorplan-block {
  min-height: 420px;
}

.location-floorplan-view {
  border: 1px solid #d8e5f1;
  background: #f8fbff;
  min-height: 360px;
  border-radius: 6px;
  overflow: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 8px;
}

.location-floorplan-view img {
  max-width: 100%;
  max-height: 680px;
  object-fit: contain;
}

.rack-occupation-row {
  margin: 0;
  display: grid;
  grid-template-columns: 110px minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  font-weight: 500;
}

.rack-occupation-row > span {
  text-align: right;
}

.rack-occupation-value {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.rack-occupation-tip {
  display: flex;
  flex: 0 1 240px;
  width: 100%;
  max-width: 100%;
  min-width: 0;
}

.rack-occupation-bar {
  flex: 0 1 240px;
  width: 100%;
  max-width: 240px;
  height: 14px;
  border: 1px solid #8ea7bf;
  border-radius: 999px;
  background: #f1f5f9;
  overflow: hidden;
}

.rack-occupation-fill {
  height: 100%;
  min-width: 1px;
  background: linear-gradient(90deg, #6dc93c 0%, #8ece03 100%);
}

.rack-occupation-text {
  color: #234d72;
  font-size: 0.86rem;
  white-space: nowrap;
}

.rack-view-wrap {
  display: grid;
  gap: 14px;
  min-width: 0;
}

.rack-view-scroller {
  width: 100%;
  overflow-x: auto;
  overflow-y: hidden;
}

.rack-view-table {
  width: max-content;
  min-width: 100%;
  border-collapse: collapse;
  table-layout: auto;
  background: #f9fbfe;
  border: 1px solid #90a7bb;
}

.rack-view-ru-col {
  width: 46px;
}

.rack-view-main-col {
  width: auto;
}

.rack-view-table th,
.rack-view-table td {
  border: 1px solid #9fb3c7;
  height: 1px;
  font-size: 7px;
}

.rack-view-table th {
  background: linear-gradient(180deg, #eef5fb 0%, #dbe7f3 100%);
  color: #244766;
  text-align: center;
  padding: 6px 4px;
  font-weight: 700;
}

.rack-view-ru {
  width: 46px;
  min-width: 46px;
  background: #fff;
  color: #244766;
  text-align: center;
  font-weight: 700;
  padding: 0;
  white-space: nowrap;
}

.rack-view-cell {
  height: 34px;
  padding: 0;
  vertical-align: middle;
  text-align: center;
}

.rack-view-empty {
  background: #f5f9fd;
}

.rack-view-occupied {
  background: var(--rack-cell-bg, linear-gradient(180deg, #e9f3fb 0%, #d7e7f6 100%));
  color: var(--rack-cell-fg, #163756);
}

.rack-view-status-active {
  background: linear-gradient(180deg, #d9ecff 0%, #c8e0fb 100%);
}

.rack-view-status-stored {
  background: linear-gradient(180deg, #dff4de 0%, #c4e7bf 100%);
}

.rack-view-status-defective {
  background: linear-gradient(180deg, #ffe1e1 0%, #f5c1c1 100%);
}

.rack-view-status-obsolete {
  background: linear-gradient(180deg, #ededed 0%, #d8d8d8 100%);
}

.rack-view-link {
  width: 100%;
  min-width: max-content;
  min-height: 100%;
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 0;
  background: transparent;
  padding: 6px 8px;
  text-align: center;
  color: inherit;
  cursor: pointer;
}

.rack-view-link:hover {
  background: rgba(255, 255, 255, 0.28);
}

.rack-view-title,
.rack-view-subtitle {
  display: inline-block;
  white-space: nowrap;
}

.rack-view-title {
  font-weight: 700;
  line-height: 1.25;
}

.rack-view-subtitle {
  font-size: 0.8rem;
  color: inherit;
  opacity: 0.92;
  line-height: 1.2;
}

.rack-view-base {
  background: #666;
  border-color: #666 !important;
  height: 8px;
  padding: 0;
}

.rack-view-wheel-spacer {
  border: 0 !important;
  background: transparent;
}

.rack-view-wheel {
  border: 0 !important;
  background: transparent;
  text-align: center;
  padding: 2px 0 0;
}

.rack-view-wheel img {
  height: 30px;
}

.rack-view-side-note {
  border: 1px dashed #b6c8d8;
  background: #f8fbfe;
  border-radius: 8px;
  padding: 10px 12px;
}

.rack-view-side-note h5 {
  margin: 0 0 8px;
  color: #244766;
  font-size: 0.94rem;
}

.rack-view-side-note ul {
  margin: 0;
  padding-left: 18px;
}

.rack-view-side-link {
  border: 0;
  padding: 0;
  background: transparent;
  color: #245fa8;
  text-align: left;
  cursor: pointer;
}

.rack-view-side-link:hover {
  text-decoration: underline;
}

.rack-view-warning-list {
  display: grid;
  gap: 6px;
  border: 1px solid #f2d1d1;
  background: #fff5f5;
  color: #a63838;
  border-radius: 8px;
  padding: 10px 12px;
}

.rack-view-warning-list p {
  margin: 0;
}

.item-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.item-tags > span {
  border: 1px solid #9dc0dd;
  border-radius: 999px;
  background: #ecf6ff;
  color: #1e567f;
  padding: 3px 10px;
  font-weight: 600;
}

.tag-editor-title-tip {
  font-size: 0.82rem;
  font-weight: 500;
  color: #4a657e;
}

.text-link-btn {
  border: 0;
  background: transparent;
  color: #2a62b7;
  padding: 0;
  font: inherit;
  text-decoration: underline;
  cursor: pointer;
}

.software-tag-list {
  align-items: center;
}

.software-tag-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding-right: 4px;
}

.software-tag-remove-btn {
  width: 18px;
  height: 18px;
  border: 1px solid #9fb8d2;
  border-radius: 999px;
  background: #fff;
  color: #245785;
  padding: 0;
  line-height: 1;
  cursor: pointer;
}

.software-tag-remove-btn:hover {
  background: #f0f6ff;
}

.software-tag-editor {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}

.software-tag-input {
  width: min(360px, 100%);
}

.item-tag-editor .software-tag-input {
  width: min(320px, 100%);
}

.item-tag-editor .small-btn {
  white-space: nowrap;
}

.item-rel-panel {
  display: grid;
  gap: 10px;
}

.item-rel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.item-rel-header h4 {
  margin: 0;
  color: #0f4f80;
}

.item-rel-filter {
  width: min(260px, 100%);
}

.item-rel-table-wrap {
  max-height: 520px;
  overflow: auto;
}

.item-rel-table {
  min-width: 800px;
}

.software-item-rel-table {
  min-width: 1150px;
  table-layout: fixed;
}

.software-item-rel-table .software-item-col-contact {
  width: 80px;
  min-width: 80px;
}

.software-item-rel-table .software-item-col-num {
  width: 80px;
  min-width: 80px;
}

.software-item-rel-table .software-item-col-type {
  width: 100px;
  min-width: 100px;
}

.software-item-rel-table .software-item-col-user {
  width: 100px;
  min-width: 100px;
}

.software-item-rel-table .software-item-col-sn {
  width: 220px;
  min-width: 220px;
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.item-rel-table tr.is-linked {
  background: #e9f4ff;
}

.item-rel-empty-row td,
.item-rel-empty-cell {
  text-align: center;
  color: #6a7f95;
  background: #f8fbff;
  font-weight: 600;
}

.relation-sort-btn {
  all: unset;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  cursor: pointer;
  font-weight: 700;
}

.relation-jump-btn {
  min-width: 54px;
  border: 1px solid #8ab4d7;
  border-radius: 5px;
  background: linear-gradient(180deg, #f5fbff 0%, #dbeeff 100%);
  color: #1f5f95;
  padding: 2px 8px;
  font-weight: 700;
  cursor: pointer;
}

.relation-jump-btn:hover {
  background: linear-gradient(180deg, #ffffff 0%, #cfe6fd 100%);
}

.software-upload-panel {
  border: 1px solid #c6d9ea;
  border-radius: 8px;
  background: #f8fbff;
  padding: 10px;
  display: grid;
  gap: 10px;
}

.software-upload-panel h4 {
  margin: 0;
  color: #0f4f80;
  border-bottom: 2px solid #4f96da;
  padding-bottom: 4px;
}

.software-upload-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(240px, 1fr));
  gap: 8px 12px;
}

.software-upload-grid label {
  margin: 0;
  display: grid;
  grid-template-columns: 96px minmax(0, 1fr);
  gap: 8px;
  align-items: center;
}

.software-upload-grid label > span {
  text-align: right;
}

.software-upload-grid .linked-upload-picker-row {
  grid-template-columns: 96px minmax(0, 1fr);
  gap: 8px;
}

.software-upload-grid .linked-upload-picker-row > span {
  text-align: right;
}

.software-upload-actions {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.item-log-wrap {
  max-height: 520px;
  overflow: auto;
}

.item-log-table {
  min-width: 760px;
}

.location-areas-block .item-log-table {
  min-width: 500px;
}

.item-form-actions {
  position: sticky;
  bottom: 0;
  background: #f6fbff;
  border-top: 1px solid #c6ddee;
  padding-top: 8px;
  margin-top: 4px;
  display: flex;
  gap: 10px;
  justify-content: flex-start;
}

.drawer-form-tabbed > .item-form-actions {
  grid-row: 3;
  position: static;
  margin-top: 0;
  padding-top: 10px;
}

.software-item-form-actions {
  position: static;
  margin-top: 0;
  padding-top: 10px;
}

@media (max-width: 1680px) {
  .item-layout {
    grid-template-columns: repeat(2, minmax(320px, 1fr));
  }

  .hardware-item-layout {
    grid-template-columns: repeat(4, minmax(180px, 1fr));
  }

  .item-bottom-grid {
    grid-template-columns: repeat(2, minmax(280px, 1fr));
  }

  .hardware-item-bottom-grid {
    grid-template-columns: repeat(3, minmax(210px, 1fr));
  }

  .asset-grid-2 {
    grid-template-columns: repeat(2, minmax(300px, 1fr));
  }

  .user-asset-grid {
    grid-template-columns: minmax(260px, 0.92fr) minmax(300px, 1.08fr);
  }

  .software-data-grid {
    grid-template-columns: minmax(300px, 0.9fr) minmax(360px, 1.1fr);
  }

  .asset-grid-3 {
    grid-template-columns: repeat(2, minmax(280px, 1fr));
  }

  .location-asset-grid {
    grid-template-columns: repeat(2, minmax(280px, 1fr));
  }

  .software-upload-grid {
    grid-template-columns: repeat(2, minmax(220px, 1fr));
  }

  .contract-textarea-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 1280px) {
  .item-layout {
    grid-template-columns: repeat(2, minmax(260px, 1fr));
  }

  .hardware-item-layout {
    grid-template-columns: repeat(4, minmax(145px, 1fr));
  }

  .item-bottom-grid {
    grid-template-columns: 1fr;
  }

  .hardware-item-bottom-grid {
    grid-template-columns: repeat(3, minmax(180px, 1fr));
  }

  .asset-grid-2 {
    grid-template-columns: minmax(260px, 0.95fr) minmax(260px, 1.05fr);
  }

  .software-data-grid {
    grid-template-columns: minmax(260px, 0.9fr) minmax(280px, 1.1fr);
  }

  .asset-grid-3 {
    grid-template-columns: repeat(2, minmax(240px, 1fr));
  }

  .location-asset-grid {
    grid-template-columns: repeat(2, minmax(240px, 1fr));
  }

  .software-upload-grid {
    grid-template-columns: 1fr;
  }

  .contract-textarea-grid {
    grid-template-columns: 1fr;
  }

  .user-asset-grid {
    grid-template-columns: minmax(240px, 0.9fr) minmax(280px, 1.1fr);
  }
}

@media (max-width: 1080px) {
  .item-layout,
  .item-bottom-grid,
  .asset-grid-2,
  .software-data-grid,
  .asset-grid-3,
  .software-upload-grid,
  .contract-textarea-grid {
    grid-template-columns: 1fr;
  }

  .user-asset-grid {
    grid-template-columns: 1fr;
  }

  .hardware-item-layout {
    grid-template-columns: repeat(2, minmax(220px, 1fr));
  }

  .location-asset-grid {
    grid-template-columns: repeat(2, minmax(220px, 1fr));
  }
}

@media (max-width: 920px) {
  .location-asset-grid,
  .hardware-item-layout {
    grid-template-columns: 1fr;
  }

  .software-managed-file-card-meta {
    grid-template-columns: 1fr;
  }
}
</style>
