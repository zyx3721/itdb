import { computed, ref, type Ref } from 'vue'

export type SortDirection = 'asc' | 'desc'

type UseTableSortOptions<T> = {
  initialKey?: string
  initialDirection?: SortDirection
  primaryKey?: string
  primaryDefaultDirection?: SortDirection
  getSortValue?: (row: T, key: string) => unknown
}

const collator = new Intl.Collator('zh-CN', {
  numeric: true,
  sensitivity: 'base',
})

function toNumber(value: unknown): number | null {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value !== 'string') return null
  const v = value.trim()
  if (!v) return null
  if (!/^[-+]?\d+(\.\d+)?$/.test(v)) return null
  const n = Number(v)
  return Number.isFinite(n) ? n : null
}

function compareUnknown(a: unknown, b: unknown) {
  const na = toNumber(a)
  const nb = toNumber(b)
  if (na !== null && nb !== null) return na - nb

  const sa = a === null || a === undefined ? '' : String(a)
  const sb = b === null || b === undefined ? '' : String(b)
  return collator.compare(sa, sb)
}

export function useTableSort<T>(rows: Ref<T[]>, options: UseTableSortOptions<T> = {}) {
  const sortKey = ref(options.initialKey ?? '')
  const sortDirection = ref<SortDirection>(options.initialDirection ?? 'desc')

  function setSort(key: string, direction: SortDirection) {
    sortKey.value = key
    sortDirection.value = direction
  }

  function toggleSort(key: string) {
    if (sortKey.value === key) {
      sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
      return
    }
    sortKey.value = key
    if (options.primaryKey && key === options.primaryKey) {
      sortDirection.value = options.primaryDefaultDirection ?? 'desc'
      return
    }
    sortDirection.value = 'asc'
  }

  function getSortIcon(key: string) {
    if (sortKey.value !== key) return '↕'
    return sortDirection.value === 'asc' ? '▲' : '▼'
  }

  const sortedRows = computed(() => {
    const list = [...rows.value]
    const key = sortKey.value
    if (!key) return list

    const factor = sortDirection.value === 'asc' ? 1 : -1
    const getSortValue = options.getSortValue

    list.sort((a, b) => {
      const av = getSortValue ? getSortValue(a, key) : (a as Record<string, unknown>)[key]
      const bv = getSortValue ? getSortValue(b, key) : (b as Record<string, unknown>)[key]
      return compareUnknown(av, bv) * factor
    })
    return list
  })

  return {
    sortKey,
    sortDirection,
    sortedRows,
    setSort,
    toggleSort,
    getSortIcon,
  }
}

