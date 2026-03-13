<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '../api/client'
import { useAuthStore } from '../stores/auth'
import { useBootstrapStore } from '../stores/bootstrap'
import { useNoticeStore } from '../stores/notice'

type VisibleDictionaryName = 'itemtypes' | 'contracttypes' | 'statustypes' | 'filetypes' | 'dpttypes' | 'tags'
type DictionaryRow = Record<string, unknown>
type TagRelatedRow = { id: number; txt: string }
type TagRelatedTarget = 'items' | 'software'
type FieldType = 'text' | 'number' | 'boolean'
type DictionaryField = { key: string; label: string; type: FieldType }
type EditorMode = 'dictionary' | 'contractSubtype'
type DeleteTarget =
  | { kind: 'dictionary'; id: number }
  | { kind: 'dictionaryBatch'; dictionary: VisibleDictionaryName; ids: number[] }
  | { kind: 'contractSubtype'; id: number }
  | { kind: 'contractSubtypeBatch'; ids: number[] }

const auth = useAuthStore()
const bootstrap = useBootstrapStore()
const noticeStore = useNoticeStore()
const route = useRoute()
const router = useRouter()

const dictionaryOrder: VisibleDictionaryName[] = ['itemtypes', 'contracttypes', 'statustypes', 'filetypes', 'dpttypes', 'tags']

const dictionaryConfig: Record<VisibleDictionaryName, { title: string; fields: DictionaryField[] }> = {
  itemtypes: {
    title: '硬件类型',
    fields: [
      { key: 'typedesc', label: '描述', type: 'text' },
      { key: 'hassoftware', label: '支持软件', type: 'boolean' },
    ],
  },
  contracttypes: {
    title: '合同类型',
    fields: [{ key: 'name', label: '类型名称', type: 'text' }],
  },
  statustypes: {
    title: '状态类型',
    fields: [{ key: 'statusdesc', label: '描述', type: 'text' }],
  },
  filetypes: {
    title: '文件类型',
    fields: [{ key: 'typedesc', label: '描述', type: 'text' }],
  },
  dpttypes: {
    title: '所属部门',
    fields: [{ key: 'dptname', label: '部门名称', type: 'text' }],
  },
  tags: {
    title: '标记',
    fields: [{ key: 'name', label: '名称', type: 'text' }],
  },
}
const dictionaryHeaderTitleMap: Record<VisibleDictionaryName, string> = {
  itemtypes: '\u4fee\u6539\u786c\u4ef6\u7c7b\u578b',
  contracttypes: '\u4fee\u6539\u5408\u540c\u7c7b\u578b',
  statustypes: '\u4fee\u6539\u72b6\u6001\u7c7b\u578b',
  filetypes: '\u4fee\u6539\u6587\u4ef6\u7c7b\u578b',
  dpttypes: '\u4fee\u6539\u6240\u5c5e\u90e8\u95e8',
  tags: '\u7f16\u8f91\u6807\u8bb0',
}

const fixedStatusTypeColorMap: Record<string, string> = {
  使用中: '#2f7fba',
  库存: '#16a34a',
  有故障: '#dc2626',
  报废: '#9ca3af',
}

const dictionaries = ref<Record<string, DictionaryRow[]>>({})
const active = ref<VisibleDictionaryName>('itemtypes')
const selectedContractTypeId = ref<number | null>(null)

const loading = ref(false)
const error = ref('')
const saving = ref(false)
const deleting = ref(false)

const editorOpen = ref(false)
const editorMode = ref<EditorMode>('dictionary')
const editingId = ref<number | null>(null)
const editorValues = reactive<Record<string, string | number>>({})
const confirmOpen = ref(false)
const deleteTarget = ref<DeleteTarget | null>(null)
const selectedDictionaryIds = ref<number[]>([])
const selectedContractTypeIds = ref<number[]>([])
const selectedContractSubtypeIds = ref<number[]>([])

const canWrite = computed(() => !auth.isReadOnly)
const pageTitle = computed(() => dictionaryHeaderTitleMap[active.value])
const createDictionaryButtonText = computed(() => `新增${dictionaryConfig[active.value].title}`)

const visibleDictionaryNames = computed<VisibleDictionaryName[]>(() =>
  dictionaryOrder.filter((name) => dictionaries.value[name] !== undefined),
)

const activeRows = computed(() => (dictionaries.value[active.value] ?? []) as DictionaryRow[])
const tagRows = computed(() => (dictionaries.value.tags ?? []) as DictionaryRow[])
const contractTypeRows = computed(() => (dictionaries.value.contracttypes ?? []) as DictionaryRow[])
const contractSubtypeRows = computed(() => {
  const typeID = Number(selectedContractTypeId.value ?? 0)
  return ((dictionaries.value.contractsubtypes ?? []) as DictionaryRow[]).filter((row) => Number(row.contypeid ?? 0) === typeID)
})
const selectedTagId = ref<number | null>(null)
const tagRelatedTarget = ref<TagRelatedTarget | null>(null)
const selectedTagRow = computed<DictionaryRow | null>(() => {
  const id = selectedTagId.value
  if (!id) return null
  return tagRows.value.find((row) => Number(row.id ?? 0) === id) ?? null
})
const relatedTagItems = ref<TagRelatedRow[]>([])
const relatedTagSoftware = ref<TagRelatedRow[]>([])
const tagRelatedLoading = ref(false)
const tagRelatedError = ref('')
let tagRelatedSeq = 0
const activeTagRelatedRows = computed<TagRelatedRow[]>(() => {
  if (tagRelatedTarget.value === 'items') return relatedTagItems.value
  if (tagRelatedTarget.value === 'software') return relatedTagSoftware.value
  return []
})
const tagRelatedPanelTitle = computed(() => {
  if (tagRelatedTarget.value === 'items') return '\u5173\u8054\u786c\u4ef6'
  if (tagRelatedTarget.value === 'software') return '\u5173\u8054\u8f6f\u4ef6'
  return ''
})

const editorFields = computed<DictionaryField[]>(() => {
  if (editorMode.value === 'contractSubtype') {
    return [{ key: 'name', label: '子类型名称', type: 'text' }]
  }
  return dictionaryConfig[active.value].fields
})

const editorTitle = computed(() => {
  if (editorMode.value === 'contractSubtype') {
    return editingId.value ? '编辑合同子类型' : '新增合同子类型'
  }
  return editingId.value ? `编辑${dictionaryConfig[active.value].title}` : `新增${dictionaryConfig[active.value].title}`
})

const confirmMessage = computed(() => {
  if (!deleteTarget.value) return ''
  if (deleteTarget.value.kind === 'dictionaryBatch') {
    return `确认批量删除已选择的 ${deleteTarget.value.ids.length} 条记录吗？`
  }
  if (deleteTarget.value.kind === 'contractSubtypeBatch') {
    return `确认批量删除已选择的 ${deleteTarget.value.ids.length} 条合同子类型吗？`
  }
  if (deleteTarget.value.kind === 'contractSubtype') {
    return `确认删除合同子类型 编号=${deleteTarget.value.id} 吗？`
  }
  return `确认删除 编号=${deleteTarget.value.id} 吗？`
})

const selectedDictionaryIdSet = computed(() => new Set(selectedDictionaryIds.value))
const selectedContractTypeIdSet = computed(() => new Set(selectedContractTypeIds.value))
const selectedContractSubtypeIdSet = computed(() => new Set(selectedContractSubtypeIds.value))
const selectableDictionaryRows = computed(() => activeRows.value.filter((row) => canDeleteDictionaryRow(row)))
const selectableContractTypeRows = computed(() => contractTypeRows.value.filter((row) => canDeleteDictionaryRow(row)))
const selectableContractSubtypeRows = computed(() => contractSubtypeRows.value.filter((row) => canDeleteDictionaryRow(row)))
const allDictionaryRowsSelected = computed(() => {
  if (selectableDictionaryRows.value.length === 0) return false
  return selectableDictionaryRows.value.every((row) => selectedDictionaryIdSet.value.has(Number(row.id ?? 0)))
})
const allContractTypeRowsSelected = computed(() => {
  if (selectableContractTypeRows.value.length === 0) return false
  return selectableContractTypeRows.value.every((row) => selectedContractTypeIdSet.value.has(Number(row.id ?? 0)))
})
const allContractSubtypeRowsSelected = computed(() => {
  if (selectableContractSubtypeRows.value.length === 0) return false
  return selectableContractSubtypeRows.value.every((row) => selectedContractSubtypeIdSet.value.has(Number(row.id ?? 0)))
})

function isVisibleDictionaryName(name: string): name is VisibleDictionaryName {
  return dictionaryOrder.includes(name as VisibleDictionaryName)
}

function normalizeRouteTab(raw: unknown): VisibleDictionaryName | '' {
  const tab = String(raw ?? '').trim()
  if (!tab) return ''
  if (tab === 'contractsubtypes') return 'contracttypes'
  return isVisibleDictionaryName(tab) ? tab : ''
}

function toEditorValue(field: DictionaryField, source: unknown): string | number {
  if (field.type === 'boolean') return Number(source ?? 0) === 1 ? 1 : 0
  if (field.type === 'number') return Number(source ?? 0)
  return decodeHtmlEntities(String(source ?? ''))
}

function decodeHtmlEntities(raw: string) {
  return raw
    .replace(/&amp;/gi, '&')
    .replace(/&lt;/gi, '<')
    .replace(/&gt;/gi, '>')
    .replace(/&quot;/gi, '"')
    .replace(/&#39;/gi, "'")
}

function displayText(raw: unknown) {
  if (raw === undefined || raw === null || String(raw).trim() === '') return '-'
  return decodeHtmlEntities(String(raw))
}

function normalizeHexColor(raw: unknown): string {
  const color = String(raw ?? '').trim()
  if (!/^#[0-9a-fA-F]{6}$/.test(color)) return ''
  return color.toLowerCase()
}

function getStatusDescription(row: DictionaryRow): string {
  return String(row.statusdesc ?? '').trim()
}

function getStatusTypeColor(row: DictionaryRow): string {
  const fixedColor = fixedStatusTypeColorMap[getStatusDescription(row)]
  if (fixedColor) return fixedColor
  const customColor = normalizeHexColor(row.color)
  return customColor || '#64748b'
}

function canMutateDictionaryRow(_row: DictionaryRow): boolean {
  return true
}

function canDeleteDictionaryRow(row: DictionaryRow): boolean {
  if (!canWrite.value) return false
  if (!canMutateDictionaryRow(row)) return false
  return Number(row.id ?? 0) > 0
}

function readTagCount(row: DictionaryRow, key: 'itemCount' | 'softwareCount'): number {
  const n = Number(row[key] ?? 0)
  if (!Number.isFinite(n) || n < 0) return 0
  return n
}

function clearTagRelated() {
  relatedTagItems.value = []
  relatedTagSoftware.value = []
  tagRelatedTarget.value = null
  tagRelatedError.value = ''
}

function syncTagSelection() {
  if (!selectedTagId.value) return
  const found = tagRows.value.find((row) => Number(row.id ?? 0) === selectedTagId.value)
  if (found) return
  selectedTagId.value = null
  clearTagRelated()
}

async function showTagRelated(row: DictionaryRow, target: TagRelatedTarget) {
  const id = Number(row.id ?? 0)
  if (!id) return
  selectedTagId.value = id
  tagRelatedTarget.value = target
  tagRelatedError.value = ''
  const seq = ++tagRelatedSeq
  tagRelatedLoading.value = true
  if (target === 'items') {
    relatedTagItems.value = []
  } else {
    relatedTagSoftware.value = []
  }
  try {
    const endpoint = target === 'items' ? `/tags/${id}/items` : `/tags/${id}/software`
    const result = await api.get(endpoint)
    if (seq !== tagRelatedSeq) return
    if (target === 'items') {
      relatedTagItems.value = Array.isArray(result.data) ? (result.data as TagRelatedRow[]) : []
      return
    }
    relatedTagSoftware.value = Array.isArray(result.data) ? (result.data as TagRelatedRow[]) : []
  } catch (err: unknown) {
    if (seq !== tagRelatedSeq) return
    if (target === 'items') {
      relatedTagItems.value = []
    } else {
      relatedTagSoftware.value = []
    }
    tagRelatedError.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '关联信息加载失败'
  } finally {
    if (seq !== tagRelatedSeq) return
    tagRelatedLoading.value = false
  }
}

function openTagRelatedEntry(target: TagRelatedTarget | null, id: number) {
  if (!target) return
  const rowID = Number(id)
  if (!Number.isFinite(rowID) || rowID <= 0) return
  const resourceName = target === 'items' ? 'items' : 'software'
  void router.push({ name: 'resource', params: { resource: resourceName }, query: { edit: String(rowID) } })
}

function resetEditorValues(source?: DictionaryRow) {
  Object.keys(editorValues).forEach((key) => delete editorValues[key])
  for (const field of editorFields.value) {
    if (source) {
      editorValues[field.key] = toEditorValue(field, source[field.key])
      continue
    }
    if (field.type === 'boolean' || field.type === 'number') {
      editorValues[field.key] = 0
    } else {
      editorValues[field.key] = ''
    }
  }
  if (editorMode.value === 'dictionary' && active.value === 'statustypes') {
    editorValues.color = source ? getStatusTypeColor(source) : '#2f7fba'
  }
}

function syncContractTypeSelection() {
  if (active.value !== 'contracttypes') {
    selectedContractTypeId.value = null
    return
  }

  const ids = contractTypeRows.value.map((row) => Number(row.id ?? 0)).filter((id) => id > 0)
  if (ids.length === 0) {
    selectedContractTypeId.value = null
    return
  }
  if (!selectedContractTypeId.value) return
  if (!ids.includes(selectedContractTypeId.value)) {
    selectedContractTypeId.value = null
  }
}

function selectDictionary(name: VisibleDictionaryName) {
  if (active.value === name && route.params.tab === name) return
  active.value = name
  void router.replace({ name: 'dictionaries', params: { tab: name } })
}

function selectContractType(id: number) {
  selectedContractTypeId.value = id
}

function clearEditorRouteQuery(...keys: string[]) {
  const nextQuery = { ...route.query } as Record<string, string | string[] | null | undefined>
  for (const key of keys) delete nextQuery[key]
  const currentTab = normalizeRouteTab(route.params.tab) || active.value
  void router.replace({ name: 'dictionaries', params: { tab: currentTab }, query: nextQuery })
}

async function recordRecentViewHistory(id: number, mode: EditorMode) {
  const rowID = Number(id)
  if (!Number.isFinite(rowID) || rowID <= 0) return
  const tab = mode === 'contractSubtype' ? 'contracttypes' : active.value
  const query =
    mode === 'contractSubtype'
      ? { subtypeEdit: String(rowID), vh: String(Date.now()) }
      : { edit: String(rowID), vh: String(Date.now()) }
  const resolved = router.resolve({ name: 'dictionaries', params: { tab }, query })
  const title = mode === 'contractSubtype' ? '合同子类型' : dictionaryConfig[active.value].title

  try {
    await api.post('/view-history', {
      url: resolved.fullPath,
      description: `${title}: ${rowID}`,
    })
    window.dispatchEvent(new CustomEvent('itdb:view-history-updated'))
  } catch {
    // Ignore recent-history write failures so dictionary editing stays usable.
  }
}

function applyDictionaryEditById(id: number) {
  const row = activeRows.value.find((entry) => Number(entry.id ?? 0) === id)
  if (!row) return false
  openEditDictionaryRow(row)
  clearEditorRouteQuery('edit', 'vh')
  return true
}

function applyContractSubtypeEditById(id: number) {
  const row = ((dictionaries.value.contractsubtypes ?? []) as DictionaryRow[]).find((entry) => Number(entry.id ?? 0) === id)
  if (!row) return false
  const contractTypeID = Number(row.contypeid ?? 0)
  if (contractTypeID > 0) selectedContractTypeId.value = contractTypeID
  openEditSubtypeRow(row)
  clearEditorRouteQuery('subtypeEdit', 'vh')
  return true
}

function applyRouteQueryActions() {
  const subtypeRaw = String(route.query.subtypeEdit ?? '').trim()
  const subtypeID = Number(subtypeRaw)
  if (subtypeID > 0) {
    if (active.value !== 'contracttypes') {
      active.value = 'contracttypes'
      return
    }
    if (applyContractSubtypeEditById(subtypeID)) return
    clearEditorRouteQuery('subtypeEdit', 'vh')
    return
  }

  const editRaw = String(route.query.edit ?? '').trim()
  const editID = Number(editRaw)
  if (editID > 0) {
    if (applyDictionaryEditById(editID)) return
    clearEditorRouteQuery('edit', 'vh')
  }
}

function openCreateDictionaryRow() {
  if (!canWrite.value) return
  error.value = ''
  editorMode.value = 'dictionary'
  editingId.value = null
  resetEditorValues()
  editorOpen.value = true
}

function openEditDictionaryRow(row: DictionaryRow) {
  if (!canWrite.value) return
  if (!canMutateDictionaryRow(row)) return
  error.value = ''
  editorMode.value = 'dictionary'
  editingId.value = Number(row.id ?? 0)
  resetEditorValues(row)
  editorOpen.value = true
}

function openCreateSubtypeRow() {
  if (!canWrite.value) return
  if (!selectedContractTypeId.value) {
    error.value = '请先选择一个合同类型'
    noticeStore.error(error.value)
    return
  }
  error.value = ''
  editorMode.value = 'contractSubtype'
  editingId.value = null
  resetEditorValues()
  editorOpen.value = true
}

function openEditSubtypeRow(row: DictionaryRow) {
  if (!canWrite.value) return
  error.value = ''
  editorMode.value = 'contractSubtype'
  editingId.value = Number(row.id ?? 0)
  resetEditorValues(row)
  editorOpen.value = true
}

function closeEditor() {
  if (saving.value) return
  error.value = ''
  editorOpen.value = false
  editingId.value = null
}

function requestRemoveDictionaryRow(row: DictionaryRow) {
  if (!canWrite.value) return
  if (!canMutateDictionaryRow(row)) return
  const id = Number(row.id ?? 0)
  if (!id) return
  deleteTarget.value = { kind: 'dictionary', id }
  confirmOpen.value = true
}

function requestRemoveSubtypeRow(id: number) {
  if (!canWrite.value) return
  deleteTarget.value = { kind: 'contractSubtype', id }
  confirmOpen.value = true
}

function closeConfirm() {
  if (deleting.value) return
  confirmOpen.value = false
  deleteTarget.value = null
}

function clearDictionarySelections() {
  selectedDictionaryIds.value = []
  selectedContractTypeIds.value = []
  selectedContractSubtypeIds.value = []
}

function clearSelectedDictionaryRows() {
  selectedDictionaryIds.value = []
}

function clearSelectedContractTypeRows() {
  selectedContractTypeIds.value = []
}

function clearSelectedContractSubtypeRows() {
  selectedContractSubtypeIds.value = []
}

function syncSelection(list: typeof selectedDictionaryIds, rows: DictionaryRow[]) {
  const allowed = new Set(
    rows
      .map((row) => Number(row.id ?? 0))
      .filter((id) => Number.isFinite(id) && id > 0),
  )
  list.value = list.value.filter((id) => allowed.has(id))
}

function updateSelection(list: typeof selectedDictionaryIds, id: number, checked: boolean) {
  if (!id) return
  if (checked) {
    if (!list.value.includes(id)) {
      list.value = [...list.value, id]
    }
    return
  }
  list.value = list.value.filter((entry) => entry !== id)
}

function toggleDictionaryRowSelection(row: DictionaryRow, event: Event) {
  if (!canDeleteDictionaryRow(row)) return
  updateSelection(selectedDictionaryIds, Number(row.id ?? 0), (event.target as HTMLInputElement | null)?.checked ?? false)
}

function toggleContractTypeRowSelection(row: DictionaryRow, event: Event) {
  if (!canDeleteDictionaryRow(row)) return
  updateSelection(selectedContractTypeIds, Number(row.id ?? 0), (event.target as HTMLInputElement | null)?.checked ?? false)
}

function toggleContractSubtypeRowSelection(row: DictionaryRow, event: Event) {
  if (!canDeleteDictionaryRow(row)) return
  updateSelection(selectedContractSubtypeIds, Number(row.id ?? 0), (event.target as HTMLInputElement | null)?.checked ?? false)
}

function toggleSelectionByRows(list: typeof selectedDictionaryIds, rows: DictionaryRow[], event: Event) {
  const checked = (event.target as HTMLInputElement | null)?.checked ?? false
  const ids = rows.map((row) => Number(row.id ?? 0)).filter((id) => id > 0)
  if (checked) {
    list.value = Array.from(new Set([...list.value, ...ids]))
    return
  }
  const idSet = new Set(ids)
  list.value = list.value.filter((id) => !idSet.has(id))
}

function toggleAllDictionaryRowSelection(event: Event) {
  toggleSelectionByRows(selectedDictionaryIds, selectableDictionaryRows.value, event)
}

function toggleAllContractTypeRowSelection(event: Event) {
  toggleSelectionByRows(selectedContractTypeIds, selectableContractTypeRows.value, event)
}

function toggleAllContractSubtypeRowSelection(event: Event) {
  toggleSelectionByRows(selectedContractSubtypeIds, selectableContractSubtypeRows.value, event)
}

function requestRemoveSelectedDictionaryRows() {
  if (!canWrite.value || selectedDictionaryIds.value.length === 0) return
  deleteTarget.value = {
    kind: 'dictionaryBatch',
    dictionary: active.value,
    ids: [...selectedDictionaryIds.value],
  }
  confirmOpen.value = true
}

function requestRemoveSelectedContractTypeRows() {
  if (!canWrite.value || selectedContractTypeIds.value.length === 0) return
  deleteTarget.value = {
    kind: 'dictionaryBatch',
    dictionary: 'contracttypes',
    ids: [...selectedContractTypeIds.value],
  }
  confirmOpen.value = true
}

function requestRemoveSelectedContractSubtypeRows() {
  if (!canWrite.value || selectedContractSubtypeIds.value.length === 0) return
  deleteTarget.value = {
    kind: 'contractSubtypeBatch',
    ids: [...selectedContractSubtypeIds.value],
  }
  confirmOpen.value = true
}

function buildDictionaryPayload() {
  const payload: Record<string, unknown> = {}
  for (const field of dictionaryConfig[active.value].fields) {
    const raw = editorValues[field.key]
    if (field.type === 'boolean' || field.type === 'number') {
      payload[field.key] = Number(raw ?? 0)
      continue
    }
    payload[field.key] = String(raw ?? '').trim()
  }
  if (active.value === 'statustypes') {
    payload.color = normalizeHexColor(editorValues.color)
  }
  return payload
}

function hasRequiredText(payload: Record<string, unknown>) {
  const primaryTextField = editorFields.value.find((field) => field.type === 'text')
  if (!primaryTextField) return true
  return String(payload[primaryTextField.key] ?? '').trim().length > 0
}

function normalizeDictionaryConflictText(raw: unknown) {
  return decodeHtmlEntities(String(raw ?? '')).trim().toLocaleLowerCase()
}

function getDictionaryConflictMessage() {
  if (editorMode.value === 'contractSubtype') {
    const text = String(editorValues.name ?? '').trim()
    if (!text) return ''
    const normalized = normalizeDictionaryConflictText(text)
    const duplicated = contractSubtypeRows.value.some(
      (row) =>
        Number(row.id ?? 0) !== Number(editingId.value ?? 0) &&
        normalizeDictionaryConflictText(row.name) === normalized,
    )
    return duplicated ? `子类型名称“${text}”已存在` : ''
  }

  const textField = editorFields.value.find((field) => field.type === 'text')
  if (!textField) return ''
  const text = String(editorValues[textField.key] ?? '').trim()
  if (!text) return ''
  const normalized = normalizeDictionaryConflictText(text)
  const duplicated = activeRows.value.some(
    (row) =>
      Number(row.id ?? 0) !== Number(editingId.value ?? 0) &&
      normalizeDictionaryConflictText(row[textField.key]) === normalized,
  )
  return duplicated ? `${textField.label}“${text}”已存在` : ''
}

function showEditorError(message: string) {
  error.value = message
  noticeStore.error(message)
}

async function refreshBootstrapLookups() {
  try {
    bootstrap.reset()
    await bootstrap.load()
  } catch {
    // 字典页不阻塞主流程，忽略联动刷新失败
  }
}

async function saveEditor() {
  if (!canWrite.value) return
  saving.value = true
  error.value = ''
  try {
    let recentHistoryID = Number(editingId.value ?? 0)
    const recentHistoryMode: EditorMode = editorMode.value

    if (editorMode.value === 'contractSubtype') {
      const typeID = Number(selectedContractTypeId.value ?? 0)
      if (!typeID) throw new Error('请先选择一个合同类型后再操作子类型')
      const payload = {
        contypeid: typeID,
        name: String(editorValues.name ?? '').trim(),
      }
      if (!hasRequiredText(payload)) {
        showEditorError('请填写名称')
        return
      }
      const conflictMessage = getDictionaryConflictMessage()
      if (conflictMessage) {
        showEditorError(conflictMessage)
        return
      }
      if (editingId.value) {
        await api.put(`/dictionaries/contractsubtypes/${editingId.value}`, payload)
      } else {
        const { data } = await api.post('/dictionaries/contractsubtypes', payload)
        recentHistoryID = Number((data as { id?: number | string } | null | undefined)?.id ?? 0)
      }
    } else {
      const payload = buildDictionaryPayload()
      if (!hasRequiredText(payload)) {
        showEditorError('请填写必填文本字段')
        return
      }
      const conflictMessage = getDictionaryConflictMessage()
      if (conflictMessage) {
        showEditorError(conflictMessage)
        return
      }
      if (editingId.value) {
        await api.put(`/dictionaries/${active.value}/${editingId.value}`, payload)
      } else {
        const { data } = await api.post(`/dictionaries/${active.value}`, payload)
        recentHistoryID = Number((data as { id?: number | string } | null | undefined)?.id ?? 0)
      }
    }

    if (recentHistoryID > 0) {
      void recordRecentViewHistory(recentHistoryID, recentHistoryMode)
    }
    editorOpen.value = false
    editingId.value = null
    await load()
    await refreshBootstrapLookups()
  } catch (err: unknown) {
    const msg = (err as { response?: { data?: { error?: string } }; message?: string })?.response?.data?.error
    error.value = msg ?? (err as { message?: string })?.message ?? '保存失败'
    noticeStore.error(error.value)
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!deleteTarget.value || !canWrite.value) return
  deleting.value = true
  error.value = ''
  const target = deleteTarget.value
  try {
    if (target.kind === 'contractSubtypeBatch') {
      let successCount = 0
      let failedCount = 0
      let firstError = ''
      for (const id of target.ids) {
        try {
          await api.delete(`/dictionaries/contractsubtypes/${id}`)
          successCount += 1
        } catch (err: unknown) {
          failedCount += 1
          if (!firstError) {
            firstError = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '删除失败'
          }
        }
      }
      selectedContractSubtypeIds.value = []
      if (successCount > 0) noticeStore.success(`已删除 ${successCount} 条合同子类型`)
      if (failedCount > 0) noticeStore.error(firstError || `有 ${failedCount} 条合同子类型删除失败`)
    } else if (target.kind === 'dictionaryBatch') {
      let successCount = 0
      let failedCount = 0
      let firstError = ''
      for (const id of target.ids) {
        try {
          await api.delete(`/dictionaries/${target.dictionary}/${id}`)
          successCount += 1
        } catch (err: unknown) {
          failedCount += 1
          if (!firstError) {
            firstError = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '删除失败'
          }
        }
      }
      if (target.dictionary === 'contracttypes') {
        selectedContractTypeIds.value = []
      } else {
        selectedDictionaryIds.value = []
      }
      if (successCount > 0) noticeStore.success(`已删除 ${successCount} 条记录`)
      if (failedCount > 0) noticeStore.error(firstError || `有 ${failedCount} 条记录删除失败`)
    } else if (target.kind === 'contractSubtype') {
      await api.delete(`/dictionaries/contractsubtypes/${target.id}`)
      selectedContractSubtypeIds.value = selectedContractSubtypeIds.value.filter((id) => id !== target.id)
    } else {
      await api.delete(`/dictionaries/${active.value}/${target.id}`)
      if (active.value === 'contracttypes') {
        selectedContractTypeIds.value = selectedContractTypeIds.value.filter((id) => id !== target.id)
      } else {
        selectedDictionaryIds.value = selectedDictionaryIds.value.filter((id) => id !== target.id)
      }
    }
    confirmOpen.value = false
    deleteTarget.value = null
    await load()
    await refreshBootstrapLookups()
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '删除失败'
  } finally {
    deleting.value = false
  }
}

function displayValue(row: DictionaryRow, field: DictionaryField) {
  const raw = row[field.key]
  if (field.type === 'boolean') return Number(raw ?? 0) === 1 ? '是' : '否'
  return displayText(raw)
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await api.get('/dictionaries')
    dictionaries.value = data

    const fromRoute = normalizeRouteTab(route.params.tab)
    if (fromRoute && visibleDictionaryNames.value.includes(fromRoute)) {
      active.value = fromRoute
    } else if (!visibleDictionaryNames.value.includes(active.value)) {
      active.value = visibleDictionaryNames.value[0] ?? 'itemtypes'
    }

    syncContractTypeSelection()
    syncTagSelection()
    applyRouteQueryActions()
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '字典数据加载失败'
  } finally {
    loading.value = false
  }
}

watch(active, () => {
  closeEditor()
  closeConfirm()
  clearDictionarySelections()
  syncContractTypeSelection()
})

watch(activeRows, () => {
  syncSelection(selectedDictionaryIds, selectableDictionaryRows.value)
})

watch(contractTypeRows, () => {
  syncSelection(selectedContractTypeIds, selectableContractTypeRows.value)
})

watch(contractSubtypeRows, () => {
  syncSelection(selectedContractSubtypeIds, selectableContractSubtypeRows.value)
})

watch(
  () => [route.params.tab, route.query.edit, route.query.subtypeEdit],
  () => {
    const fromRoute = normalizeRouteTab(route.params.tab)
    if (fromRoute && fromRoute !== active.value) {
      active.value = fromRoute
      return
    }
    if (!fromRoute && route.params.tab) {
      void router.replace({ name: 'dictionaries', params: { tab: active.value } })
      return
    }
    if (!loading.value && (route.query.edit || route.query.subtypeEdit)) {
      applyRouteQueryActions()
    }
  },
)

void load()
</script>

<template>
  <section class="page-shell">
    <header class="page-header">
      <h2>{{ pageTitle }}</h2>
      <div class="header-actions">
        <button class="ghost-btn" @click="load">刷新</button>
        <button v-if="canWrite && active !== 'contracttypes'" class="dict-add-btn" @click="openCreateDictionaryRow">
          {{ createDictionaryButtonText }}
        </button>
      </div>
    </header>

    <p v-if="loading">加载中...</p>

    <div v-else class="dict-layout">
      <aside class="dict-nav-old">
        <button
          v-for="name in visibleDictionaryNames"
          :key="name"
          class="dict-tab"
          :class="{ active: active === name }"
          @click="selectDictionary(name)"
        >
          {{ dictionaryConfig[name].title }}
        </button>
      </aside>

      <div class="dict-content">
        <template v-if="active === 'contracttypes'">
          <div class="contract-types-layout" :class="{ 'with-subtypes': Boolean(selectedContractTypeId) }">
            <section class="dict-panel contract-type-main-panel">
              <div class="dict-panel-head">
                <h3>合同类型</h3>
                <div v-if="canWrite" class="dict-panel-actions">
                  <button class="small-btn ghost-btn" :disabled="selectedContractTypeIds.length === 0" @click="clearSelectedContractTypeRows">
                    清空选择
                  </button>
                  <button class="small-btn danger" :disabled="selectedContractTypeIds.length === 0" @click="requestRemoveSelectedContractTypeRows">
                    批量删除（{{ selectedContractTypeIds.length }}）
                  </button>
                  <button class="small-btn dict-add-btn" @click="openCreateDictionaryRow">新增合同类型</button>
                </div>
              </div>
              <div class="table-wrap">
                <table class="dict-edit-table">
                  <thead>
                    <tr>
                      <th v-if="canWrite" class="dict-selection-col">
                        <input
                          type="checkbox"
                          :checked="allContractTypeRowsSelected"
                          :disabled="selectableContractTypeRows.length === 0"
                          @change="toggleAllContractTypeRowSelection"
                        />
                      </th>
                      <th>编号</th>
                      <th>类型名称</th>
                      <th>操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="row in contractTypeRows"
                      :key="`ctype-${String(row.id ?? '')}`"
                      :class="{ 'is-selected': selectedContractTypeId === Number(row.id ?? 0) }"
                    >
                      <td v-if="canWrite" class="dict-selection-col">
                        <input
                          type="checkbox"
                          :checked="selectedContractTypeIdSet.has(Number(row.id ?? 0))"
                          :disabled="!canDeleteDictionaryRow(row)"
                          @change="toggleContractTypeRowSelection(row, $event)"
                        />
                      </td>
                      <td>{{ row.id }}</td>
                      <td>{{ displayText(row.name) }}</td>
                    <td class="actions-cell">
                      <button class="small-btn ghost-btn" @click="selectContractType(Number(row.id ?? 0))">查看/编辑 子类型</button>
                      <button v-if="canWrite" class="small-btn" @click="openEditDictionaryRow(row)">编辑</button>
                      <button v-if="canWrite" class="small-btn danger" @click="requestRemoveDictionaryRow(row)">删除</button>
                    </td>
                  </tr>
                    <tr v-if="contractTypeRows.length === 0">
                      <td :colspan="canWrite ? 4 : 3">暂无合同类型数据</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </section>

            <section v-if="selectedContractTypeId" class="dict-panel contract-subtype-panel">
              <div class="dict-panel-head">
                <h3>合同子类型（类型编号: {{ selectedContractTypeId }}）</h3>
                <div v-if="canWrite" class="dict-panel-actions">
                  <button
                    class="small-btn ghost-btn"
                    :disabled="selectedContractSubtypeIds.length === 0"
                    @click="clearSelectedContractSubtypeRows"
                  >
                    清空选择
                  </button>
                  <button
                    class="small-btn danger"
                    :disabled="selectedContractSubtypeIds.length === 0"
                    @click="requestRemoveSelectedContractSubtypeRows"
                  >
                    批量删除（{{ selectedContractSubtypeIds.length }}）
                  </button>
                  <button class="small-btn dict-add-btn" @click="openCreateSubtypeRow">新增子类型</button>
                </div>
              </div>
              <div class="table-wrap">
                <table class="dict-edit-table">
                  <thead>
                    <tr>
                      <th v-if="canWrite" class="dict-selection-col">
                        <input
                          type="checkbox"
                          :checked="allContractSubtypeRowsSelected"
                          :disabled="selectableContractSubtypeRows.length === 0"
                          @change="toggleAllContractSubtypeRowSelection"
                        />
                      </th>
                      <th>编号</th>
                      <th>子类型名称</th>
                      <th>操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="row in contractSubtypeRows" :key="`csub-${String(row.id ?? '')}`">
                      <td v-if="canWrite" class="dict-selection-col">
                        <input
                          type="checkbox"
                          :checked="selectedContractSubtypeIdSet.has(Number(row.id ?? 0))"
                          :disabled="!canDeleteDictionaryRow(row)"
                          @change="toggleContractSubtypeRowSelection(row, $event)"
                        />
                      </td>
                      <td>{{ row.id }}</td>
                      <td>{{ displayText(row.name) }}</td>
                    <td class="actions-cell">
                      <button v-if="canWrite" class="small-btn" @click="openEditSubtypeRow(row)">编辑</button>
                      <button v-if="canWrite" class="small-btn danger" @click="requestRemoveSubtypeRow(Number(row.id ?? 0))">删除</button>
                    </td>
                  </tr>
                    <tr v-if="contractSubtypeRows.length === 0">
                      <td :colspan="canWrite ? 4 : 3">暂无合同子类型数据</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </section>
          </div>
        </template>

        <template v-else>
          <div v-if="active === 'tags'" class="tags-dict-layout">
            <section class="dict-panel">
              <div class="dict-panel-head">
                <h3>标记</h3>
                <div v-if="canWrite" class="dict-panel-actions">
                  <button class="small-btn ghost-btn" :disabled="selectedDictionaryIds.length === 0" @click="clearSelectedDictionaryRows">
                    清空选择
                  </button>
                  <button class="small-btn danger" :disabled="selectedDictionaryIds.length === 0" @click="requestRemoveSelectedDictionaryRows">
                    批量删除（{{ selectedDictionaryIds.length }}）
                  </button>
                </div>
              </div>
              <div class="table-wrap">
                <table class="dict-edit-table">
                  <thead>
                    <tr>
                      <th v-if="canWrite" class="dict-selection-col">
                        <input
                          type="checkbox"
                          :checked="allDictionaryRowsSelected"
                          :disabled="selectableDictionaryRows.length === 0"
                          @change="toggleAllDictionaryRowSelection"
                        />
                      </th>
                      <th>编号</th>
                      <th>名称</th>
                      <th>关联硬件</th>
                      <th>关联软件</th>
                      <th v-if="canWrite">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="row in activeRows" :key="`${active}-${String(row.id ?? '')}`">
                      <td v-if="canWrite" class="dict-selection-col">
                        <input
                          type="checkbox"
                          :checked="selectedDictionaryIdSet.has(Number(row.id ?? 0))"
                          :disabled="!canDeleteDictionaryRow(row)"
                          @change="toggleDictionaryRowSelection(row, $event)"
                        />
                      </td>
                      <td>{{ row.id }}</td>
                      <td>{{ displayText(row.name) }}</td>
                      <td>
                        <a class="tag-count-link" href="#" @click.prevent="showTagRelated(row, 'items')">
                          {{ readTagCount(row, 'itemCount') }}
                        </a>
                      </td>
                      <td>
                        <a class="tag-count-link" href="#" @click.prevent="showTagRelated(row, 'software')">
                          {{ readTagCount(row, 'softwareCount') }}
                        </a>
                      </td>
                      <td v-if="canWrite" class="actions-cell">
                        <button v-if="canMutateDictionaryRow(row)" class="small-btn" @click="openEditDictionaryRow(row)">编辑</button>
                        <button v-if="canMutateDictionaryRow(row)" class="small-btn danger" @click="requestRemoveDictionaryRow(row)">删除</button>
                      </td>
                    </tr>
                    <tr v-if="activeRows.length === 0">
                      <td :colspan="canWrite ? 6 : 4">暂无数据</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </section>

            <section class="dict-panel tags-related-panel">
              <h3>关联信息</h3>
              <p v-if="!selectedTagRow" class="muted-text">点击“关联硬件/关联软件”数量查看详情</p>
              <template v-else>
                <p class="tag-selected-title">
                  <span class="tag-selected-label">当前标记</span>
                  <span class="tag-selected-value">{{ displayText(selectedTagRow.name) }}</span>
                </p>
                <p v-if="tagRelatedLoading">加载中...</p>
                <div v-else class="tag-columns">
                  <div v-if="tagRelatedTarget">
                    <h4 class="tag-related-section-title">{{ tagRelatedPanelTitle }}</h4>
                    <div class="tag-related-list">
                      <a
                        v-for="(entry, index) in activeTagRelatedRows"
                        :key="`tag-related-${tagRelatedTarget}-${entry.id}-${index}`"
                        class="tag-related-link"
                        href="#"
                        @click.prevent="openTagRelatedEntry(tagRelatedTarget, Number(entry.id ?? 0))"
                      >
                        <span class="tag-related-index">{{ index + 1 }}:</span>
                        <span class="tag-related-text">{{ displayText(entry.txt) }}</span>
                      </a>
                      <p v-if="activeTagRelatedRows.length === 0" class="muted-text">无</p>
                    </div>
                  </div>
                  <p v-else class="muted-text">点击“关联硬件/关联软件”数量查看详情</p>
                </div>
              </template>
            </section>
          </div>

          <section v-else class="dict-panel">
            <div class="dict-panel-head">
              <h3>{{ dictionaryConfig[active].title }}</h3>
              <div v-if="canWrite" class="dict-panel-actions">
                <button class="small-btn ghost-btn" :disabled="selectedDictionaryIds.length === 0" @click="clearSelectedDictionaryRows">
                  清空选择
                </button>
                <button class="small-btn danger" :disabled="selectedDictionaryIds.length === 0" @click="requestRemoveSelectedDictionaryRows">
                  批量删除（{{ selectedDictionaryIds.length }}）
                </button>
              </div>
            </div>
            <div class="table-wrap">
              <table class="dict-edit-table">
                <thead>
                  <tr>
                    <th v-if="canWrite" class="dict-selection-col">
                      <input
                        type="checkbox"
                        :checked="allDictionaryRowsSelected"
                        :disabled="selectableDictionaryRows.length === 0"
                        @change="toggleAllDictionaryRowSelection"
                      />
                    </th>
                    <th>编号</th>
                    <th v-for="field in dictionaryConfig[active].fields" :key="`head-${field.key}`">{{ field.label }}</th>
                    <th class="dict-main-actions-col">操作</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in activeRows" :key="`${active}-${String(row.id ?? '')}`">
                    <td v-if="canWrite" class="dict-selection-col">
                      <input
                        type="checkbox"
                        :checked="selectedDictionaryIdSet.has(Number(row.id ?? 0))"
                        :disabled="!canDeleteDictionaryRow(row)"
                        @change="toggleDictionaryRowSelection(row, $event)"
                      />
                    </td>
                    <td>{{ row.id }}</td>
                    <td v-for="field in dictionaryConfig[active].fields" :key="`${active}-${String(row.id ?? '')}-${field.key}`">
                      <span v-if="active === 'statustypes' && field.key === 'statusdesc'" class="status-type-cell">
                        <span class="status-type-color-dot" :style="{ backgroundColor: getStatusTypeColor(row) }"></span>
                        <span>{{ displayValue(row, field) }}</span>
                      </span>
                      <template v-else>
                        {{ displayValue(row, field) }}
                      </template>
                    </td>
                    <td class="actions-cell dict-main-actions-cell">
                      <button v-if="canWrite && canMutateDictionaryRow(row)" class="small-btn" @click="openEditDictionaryRow(row)">编辑</button>
                      <button
                        v-if="canWrite && canMutateDictionaryRow(row)"
                        class="small-btn danger"
                        @click="requestRemoveDictionaryRow(row)"
                      >
                        删除
                      </button>
                    </td>
                  </tr>
                  <tr v-if="activeRows.length === 0">
                    <td :colspan="dictionaryConfig[active].fields.length + (canWrite ? 3 : 2)">暂无数据</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>
        </template>
      </div>
    </div>

    <div v-if="canWrite && editorOpen" class="dialog-mask">
      <section class="drawer modal-narrow" role="dialog" aria-modal="true">
        <div class="drawer-header">
          <h3>{{ editorTitle }}</h3>
          <button class="dialog-close-btn quick-tip" type="button" aria-label="关闭" data-quick-tip="关闭" @click="closeEditor">×</button>
        </div>
        <form class="drawer-form" @submit.prevent="saveEditor">
          <label v-for="field in editorFields" :key="`editor-${field.key}`">
            <span>{{ field.label }}</span>
            <select v-if="field.type === 'boolean'" v-model.number="editorValues[field.key]">
              <option :value="0">否</option>
              <option :value="1">是</option>
            </select>
            <input v-else v-model="editorValues[field.key]" :type="field.type === 'number' ? 'number' : 'text'" />
          </label>
          <div v-if="editorMode === 'dictionary' && active === 'statustypes'" class="status-color-field">
            <span>颜色</span>
            <input v-model="editorValues.color" type="color" class="status-color-picker" />
          </div>

          <p v-if="editorMode === 'contractSubtype'" class="muted-text">当前合同类型编号：{{ selectedContractTypeId }}</p>

          <div class="inline-actions">
            <button :disabled="saving" type="submit">{{ saving ? '保存中...' : editingId ? '保存修改' : '新增' }}</button>
            <button class="ghost-btn" type="button" @click="closeEditor">取消</button>
          </div>
        </form>
      </section>
    </div>

    <div v-if="canWrite && confirmOpen" class="dialog-mask">
      <section class="drawer modal-narrow" role="dialog" aria-modal="true">
        <div class="drawer-header">
          <h3>删除确认</h3>
          <button class="dialog-close-btn quick-tip" type="button" aria-label="关闭" data-quick-tip="关闭" @click="closeConfirm">×</button>
        </div>
        <div class="drawer-form">
          <p>{{ confirmMessage }}</p>
          <div class="inline-actions">
            <button class="danger" type="button" :disabled="deleting" @click="confirmDelete">
              {{ deleting ? '删除中...' : '确认删除' }}
            </button>
            <button class="ghost-btn" type="button" :disabled="deleting" @click="closeConfirm">取消</button>
          </div>
        </div>
      </section>
    </div>
  </section>
</template>

<style scoped>
.dict-layout {
  display: grid;
  grid-template-columns: 220px 1fr;
  gap: 12px;
}

.dict-nav-old {
  border: 1px solid var(--line);
  border-radius: 12px;
  background: var(--surface-soft);
  padding: 10px;
  display: grid;
  gap: 8px;
  align-content: start;
}

.dict-nav-old .dict-tab {
  width: 100%;
  text-align: center;
}

.dict-content {
  min-width: 0;
}

.dict-panel {
  border: 1px solid var(--line);
  border-radius: 12px;
  background: var(--surface-soft);
  padding: 12px;
}

.dict-panel h3 {
  margin: 0 0 10px;
  font-size: 1.05rem;
}

.dict-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
}

.dict-panel-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.dict-selection-col {
  width: 44px;
  min-width: 44px;
  text-align: center;
  vertical-align: middle;
}

.dict-edit-table td input,
.dict-edit-table td select {
  min-width: 140px;
}

.dict-main-actions-col,
.dict-main-actions-cell {
  width: 176px;
  min-width: 176px;
}

.dict-main-actions-cell {
  display: table-cell !important;
  text-align: center;
  vertical-align: middle;
  white-space: nowrap;
  padding: 8px 10px;
}

.dict-main-actions-cell .small-btn {
  margin: 0 4px;
}

.dict-add-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.32rem 0.62rem;
  border-radius: 8px;
  line-height: 1.08;
  -webkit-tap-highlight-color: transparent;
}

.contract-types-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 12px;
}

.contract-types-layout.with-subtypes {
  grid-template-columns: minmax(0, 1fr) minmax(0, 0.95fr);
}

.contract-type-main-panel,
.contract-subtype-panel {
  min-width: 0;
}

.tags-dict-layout {
  display: grid;
  grid-template-columns: minmax(520px, 1.4fr) minmax(320px, 1fr);
  gap: 12px;
}

.tags-related-panel {
  min-width: 0;
}

.tags-related-panel h3 {
  margin: 0 0 18px;
  display: inline-flex;
  align-items: center;
  padding: 6px 12px;
  border-radius: 999px;
  background: linear-gradient(180deg, #f8fcff 0%, #e6f1ff 100%);
  border: 1px solid #cfe0f0;
  color: #123c64;
  box-shadow: 0 6px 14px rgba(56, 113, 168, 0.08);
}

.tag-selected-title {
  margin: 0 0 18px;
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.tag-selected-label {
  display: inline-flex;
  align-items: center;
  padding: 5px 10px;
  border-radius: 999px;
  background: #e0f2fe;
  border: 1px solid #bae6fd;
  color: #0f4b8a;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.tag-selected-value {
  display: inline-flex;
  align-items: center;
  padding: 6px 12px;
  border-radius: 10px;
  background: #f8fbff;
  border: 1px solid #d7e4f2;
  color: #173a5e;
  font-weight: 600;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.8);
}

.tag-columns h4 {
  margin: 0 0 18px;
}

.tag-related-section-title {
  display: inline-flex;
  align-items: center;
  padding: 6px 12px;
  border-left: 4px solid #5fa2dc;
  border-radius: 8px;
  background: linear-gradient(180deg, #fbfdff 0%, #edf5ff 100%);
  color: #1b446d;
  box-shadow: 0 4px 12px rgba(71, 122, 173, 0.08);
}

.tag-related-list {
  display: grid;
  gap: 8px;
}

.tag-related-link {
  display: inline-flex;
  align-items: flex-start;
  justify-self: start;
  width: fit-content;
  max-width: 100%;
  gap: 8px;
  padding: 7px 10px;
  border-radius: 10px;
  border: 1px solid transparent;
  background: #f7fbff;
  text-decoration: none;
  color: #0f4b8a;
  line-height: 1.5;
  cursor: pointer;
  transition:
    background 0.18s ease,
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    transform 0.18s ease;
}

.tag-related-link:hover {
  color: #0b3d73;
  text-decoration: underline;
  border-color: #c9ddf1;
  background: #eef6ff;
  box-shadow: 0 4px 10px rgba(56, 113, 168, 0.12);
  transform: translateY(-1px);
}

.tag-related-index {
  min-width: 30px;
  padding: 1px 8px;
  border-radius: 999px;
  background: #dbeafe;
  color: #1d4f7f;
  font-variant-numeric: tabular-nums;
  font-weight: 700;
  text-align: center;
}

.tag-related-text {
  flex: 0 1 auto;
  min-width: 0;
  color: #163a5d;
  word-break: break-word;
}

.tag-count-link {
  color: #0f4b8a;
  text-decoration: underline;
}

.tag-count-link:hover {
  color: #0b3d73;
}

.is-selected {
  background: #edf8f7;
}

.status-type-cell {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.status-type-color-dot {
  width: 12px;
  height: 12px;
  border-radius: 999px;
  border: 1px solid rgba(15, 23, 42, 0.18);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.22);
}

.status-color-field {
  display: grid;
  gap: 6px;
  font-weight: 500;
}

.status-color-picker {
  width: 46px;
  min-width: 46px;
  height: 30px;
  border-radius: 6px;
  padding: 0;
  cursor: pointer;
  border: 1px solid #cbd5e1;
  overflow: hidden;
}

.status-color-picker::-webkit-color-swatch-wrapper {
  padding: 0;
}

.status-color-picker::-webkit-color-swatch {
  border: 0;
}

.status-color-picker::-moz-color-swatch {
  border: 0;
}

@media (max-width: 1180px) {
  .dict-layout {
    grid-template-columns: 1fr;
  }

  .contract-types-layout {
    grid-template-columns: 1fr;
  }

  .contract-types-layout.with-subtypes {
    grid-template-columns: 1fr;
  }

  .tags-dict-layout {
    grid-template-columns: 1fr;
  }
}
</style>
