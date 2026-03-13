<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, useAttrs, watch, type CSSProperties } from 'vue'

defineOptions({ inheritAttrs: false })

type DayCell = {
  iso: string
  label: number
  inMonth: boolean
  today: boolean
  selected: boolean
}

const props = withDefaults(
  defineProps<{
    modelValue?: string
    required?: boolean
    disabled?: boolean
    readonly?: boolean
    placeholder?: string
  }>(),
  {
    modelValue: '',
    required: false,
    disabled: false,
    readonly: false,
    placeholder: 'YYYY-MM-DD',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: string): void
  (e: 'blur'): void
}>()

const attrs = useAttrs()
const rootRef = ref<HTMLElement | null>(null)
const panelRef = ref<HTMLElement | null>(null)

const YEAR_START = 1970
const YEAR_END = 2100
const PANEL_MARGIN = 8
const PANEL_GAP = 8
const MIN_PANEL_WIDTH = 286
const MAX_PANEL_WIDTH = 360
const DEFAULT_PANEL_HEIGHT = 332
const WEEK_LABELS = ['一', '二', '三', '四', '五', '六', '日']

const draft = ref(props.modelValue ?? '')
const open = ref(false)
const viewYear = ref(new Date().getFullYear())
const viewMonth = ref(new Date().getMonth() + 1)
const panelTop = ref(0)
const panelLeft = ref(0)
const panelWidth = ref(MIN_PANEL_WIDTH)
const panelPlacement = ref<'top' | 'bottom'>('bottom')

const years = computed(() => {
  const list: number[] = []
  for (let year = YEAR_START; year <= YEAR_END; year += 1) list.push(year)
  return list
})

const months = computed(() => Array.from({ length: 12 }, (_, index) => index + 1))
const inputValue = computed(() => draft.value)

const forwardedAttrs = computed(() => {
  const {
    pattern: _pattern,
    title: _title,
    type: _type,
    value: _value,
    onInput: _onInput,
    onBlur: _onBlur,
    onClick: _onClick,
    onKeydown: _onKeydown,
    ...rest
  } = attrs as Record<string, unknown>
  return rest
})

function pad2(value: number) {
  return String(value).padStart(2, '0')
}

function formatISO(year: number, month: number, day: number) {
  return `${year}-${pad2(month)}-${pad2(day)}`
}

function isValidDateParts(year: number, month: number, day: number) {
  if (!Number.isInteger(year) || !Number.isInteger(month) || !Number.isInteger(day)) return false
  if (month < 1 || month > 12 || day < 1 || day > 31) return false
  const probe = new Date(year, month - 1, day)
  return probe.getFullYear() === year && probe.getMonth() + 1 === month && probe.getDate() === day
}

function normalizeFlexibleInput(value: string) {
  return value
    .trim()
    .replace(/[./年]/g, '-')
    .replace(/月/g, '-')
    .replace(/日/g, '')
    .replace(/\s+/g, '')
}

function parseISO(value: string) {
  const match = value.trim().match(/^(\d{4})-(\d{2})-(\d{2})$/)
  if (!match) return null

  const year = Number(match[1])
  const month = Number(match[2])
  const day = Number(match[3])
  if (!isValidDateParts(year, month, day)) return null

  return { year, month, day, iso: formatISO(year, month, day) }
}

function parseFlexible(value: string) {
  const normalized = normalizeFlexibleInput(value)
  const match = normalized.match(/^(\d{4})-(\d{1,2})-(\d{1,2})$/)
  if (!match) return null

  const year = Number(match[1])
  const month = Number(match[2])
  const day = Number(match[3])
  if (!isValidDateParts(year, month, day)) return null

  return { year, month, day, iso: formatISO(year, month, day) }
}

function syncViewFromISO(value: string) {
  const parsed = parseISO(value)
  if (!parsed) return
  viewYear.value = parsed.year
  viewMonth.value = parsed.month
}

function updateViewFromPartial(value: string) {
  const normalized = normalizeFlexibleInput(value)
  const match = normalized.match(/^(\d{1,4})?(?:-(\d{1,2})?)?(?:-(\d{1,2})?)?$/)
  if (!match) return

  const yearRaw = match[1]
  const monthRaw = match[2]

  if (yearRaw && yearRaw.length >= 3) {
    const year = Number(yearRaw)
    if (!Number.isNaN(year)) {
      viewYear.value = Math.min(Math.max(year, YEAR_START), YEAR_END)
    }
  }

  if (monthRaw) {
    const month = Number(monthRaw)
    if (!Number.isNaN(month) && month >= 1 && month <= 12) {
      viewMonth.value = month
    }
  }
}

function daysInMonth(year: number, month: number) {
  return new Date(year, month, 0).getDate()
}

const cells = computed<DayCell[]>(() => {
  const year = viewYear.value
  const month = viewMonth.value
  const firstDay = new Date(year, month - 1, 1)
  const jsWeekDay = firstDay.getDay()
  const weekStart = jsWeekDay === 0 ? 6 : jsWeekDay - 1
  const currentMonthDays = daysInMonth(year, month)
  const previousMonthDays = daysInMonth(year, month - 1 <= 0 ? 12 : month - 1)
  const selected = parseISO(props.modelValue ?? '')

  const today = new Date()
  const todayISO = formatISO(today.getFullYear(), today.getMonth() + 1, today.getDate())

  const list: DayCell[] = []

  for (let index = weekStart - 1; index >= 0; index -= 1) {
    const day = previousMonthDays - index
    const previous = month - 1 <= 0 ? { year: year - 1, month: 12 } : { year, month: month - 1 }
    const iso = formatISO(previous.year, previous.month, day)
    list.push({
      iso,
      label: day,
      inMonth: false,
      today: iso === todayISO,
      selected: selected?.iso === iso,
    })
  }

  for (let day = 1; day <= currentMonthDays; day += 1) {
    const iso = formatISO(year, month, day)
    list.push({
      iso,
      label: day,
      inMonth: true,
      today: iso === todayISO,
      selected: selected?.iso === iso,
    })
  }

  while (list.length % 7 !== 0) {
    const day = list.length - (weekStart + currentMonthDays) + 1
    const next = month + 1 > 12 ? { year: year + 1, month: 1 } : { year, month: month + 1 }
    const iso = formatISO(next.year, next.month, day)
    list.push({
      iso,
      label: day,
      inMonth: false,
      today: iso === todayISO,
      selected: selected?.iso === iso,
    })
  }

  while (list.length < 42) {
    const tail = list[list.length - 1]
    if (!tail) break

    const tailParsed = parseISO(tail.iso)
    if (!tailParsed) break

    const next = new Date(tailParsed.year, tailParsed.month - 1, tailParsed.day + 1)
    const iso = formatISO(next.getFullYear(), next.getMonth() + 1, next.getDate())
    list.push({
      iso,
      label: next.getDate(),
      inMonth: false,
      today: iso === todayISO,
      selected: selected?.iso === iso,
    })
  }

  return list
})

function clampPanelLeft(targetLeft: number, width: number) {
  const maxLeft = Math.max(PANEL_MARGIN, window.innerWidth - width - PANEL_MARGIN)
  return Math.min(Math.max(targetLeft, PANEL_MARGIN), maxLeft)
}

function measurePanelHeight() {
  const actualHeight = panelRef.value?.offsetHeight ?? DEFAULT_PANEL_HEIGHT
  return Math.min(actualHeight, window.innerHeight - PANEL_MARGIN * 2)
}

function updatePanelPosition() {
  const element = rootRef.value
  if (!element) return

  const rect = element.getBoundingClientRect()
  const targetWidth = Math.min(Math.max(rect.width, MIN_PANEL_WIDTH), MAX_PANEL_WIDTH)
  const panelHeight = measurePanelHeight()
  const availableAbove = rect.top - PANEL_MARGIN - PANEL_GAP
  const availableBelow = window.innerHeight - rect.bottom - PANEL_MARGIN - PANEL_GAP
  const showAbove = panelHeight > availableBelow && availableAbove >= availableBelow
  const preferredTop = showAbove ? rect.top - panelHeight - PANEL_GAP : rect.bottom + PANEL_GAP
  const maxTop = Math.max(PANEL_MARGIN, window.innerHeight - panelHeight - PANEL_MARGIN)

  panelWidth.value = targetWidth
  panelLeft.value = clampPanelLeft(rect.left, targetWidth)
  panelTop.value = Math.min(Math.max(preferredTop, PANEL_MARGIN), maxTop)
  panelPlacement.value = showAbove ? 'top' : 'bottom'
}

function schedulePanelPosition() {
  nextTick(() => {
    updatePanelPosition()
    window.requestAnimationFrame(() => {
      if (open.value) updatePanelPosition()
    })
  })
}

function closePanel() {
  open.value = false
}

function openPanel() {
  if (props.disabled || props.readonly) return

  open.value = true
  const parsed = parseFlexible(draft.value)
  if (parsed) {
    viewYear.value = parsed.year
    viewMonth.value = parsed.month
  } else if (props.modelValue) {
    syncViewFromISO(props.modelValue)
  }

  schedulePanelPosition()
}

function togglePanel() {
  if (open.value) closePanel()
  else openPanel()
}

function emitValue(value: string) {
  emit('update:modelValue', value)
  emit('change', value)
}

function selectISO(iso: string) {
  draft.value = iso
  emitValue(iso)
  syncViewFromISO(iso)
  closePanel()
}

function pickToday() {
  const now = new Date()
  selectISO(formatISO(now.getFullYear(), now.getMonth() + 1, now.getDate()))
}

function clearValue() {
  draft.value = ''
  emitValue('')
  closePanel()
}

function prevMonth() {
  if (viewMonth.value === 1) {
    viewMonth.value = 12
    viewYear.value -= 1
  } else {
    viewMonth.value -= 1
  }
}

function nextMonth() {
  if (viewMonth.value === 12) {
    viewMonth.value = 1
    viewYear.value += 1
  } else {
    viewMonth.value += 1
  }
}

function onInput(event: Event) {
  const target = event.target as HTMLInputElement
  draft.value = target.value
  updateViewFromPartial(draft.value)

  const parsed = parseFlexible(draft.value)
  if (parsed) {
    emitValue(parsed.iso)
    viewYear.value = parsed.year
    viewMonth.value = parsed.month

    const normalized = normalizeFlexibleInput(draft.value)
    if (open.value && /^\d{4}-\d{1,2}-\d{1,2}$/.test(normalized)) {
      draft.value = parsed.iso
      closePanel()
    }
    return
  }

  if (!draft.value.trim() && props.modelValue) {
    emitValue('')
  }
}

function onBlur() {
  const parsed = parseFlexible(draft.value)

  if (!draft.value.trim()) {
    draft.value = ''
    emitValue('')
  } else if (parsed) {
    draft.value = parsed.iso
    if (parsed.iso !== props.modelValue) {
      emitValue(parsed.iso)
    }
  } else {
    draft.value = props.modelValue ?? ''
  }

  emit('blur')
}

function onInputClick() {
  openPanel()
}

function onInputKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    closePanel()
    return
  }

  if (event.key === 'ArrowDown' && !open.value) {
    event.preventDefault()
    openPanel()
  }
}

function onDocumentPointerDown(event: Event) {
  if (!open.value) return

  const target = event.target as Node | null
  if (!target) return
  if (rootRef.value?.contains(target)) return
  if (panelRef.value?.contains(target)) return
  closePanel()
}

function onWindowChange() {
  if (!open.value) return
  updatePanelPosition()
}

const panelStyle = computed<CSSProperties>(() => ({
  position: 'fixed',
  top: `${panelTop.value}px`,
  left: `${panelLeft.value}px`,
  width: `${panelWidth.value}px`,
}))

watch(
  () => props.modelValue,
  (value) => {
    const safe = value ?? ''
    if (safe === draft.value) return
    draft.value = safe
    syncViewFromISO(safe)
  },
)

watch([viewYear, viewMonth], () => {
  if (!open.value) return
  schedulePanelPosition()
})

watch(open, (value) => {
  if (value) {
    window.addEventListener('resize', onWindowChange)
    window.addEventListener('scroll', onWindowChange, true)
    document.addEventListener('pointerdown', onDocumentPointerDown)
  } else {
    window.removeEventListener('resize', onWindowChange)
    window.removeEventListener('scroll', onWindowChange, true)
    document.removeEventListener('pointerdown', onDocumentPointerDown)
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onWindowChange)
  window.removeEventListener('scroll', onWindowChange, true)
  document.removeEventListener('pointerdown', onDocumentPointerDown)
})
</script>

<template>
  <div ref="rootRef" class="app-date-input" :class="{ 'is-open': open, 'is-disabled': disabled }">
    <input
      class="app-date-input__field"
      type="text"
      inputmode="numeric"
      :value="inputValue"
      :placeholder="placeholder"
      maxlength="16"
      :required="required"
      :disabled="disabled"
      :readonly="readonly"
      autocomplete="off"
      spellcheck="false"
      v-bind="forwardedAttrs"
      @input="onInput"
      @click="onInputClick"
      @keydown="onInputKeydown"
      @blur="onBlur"
    />
    <button
      class="app-date-input__trigger"
      type="button"
      :disabled="disabled || readonly"
      aria-label="打开日期选择"
      @click="togglePanel"
    >
      📅
    </button>

    <teleport to="body">
      <transition name="app-date-panel-pop">
        <section
          v-if="open"
          ref="panelRef"
          :class="['app-date-panel', `is-${panelPlacement}`]"
          :style="panelStyle"
        >
          <header class="app-date-panel__header">
            <button type="button" class="app-date-panel__arrow" @click="prevMonth">‹</button>
            <div class="app-date-panel__pickers">
              <select v-model.number="viewYear">
                <option v-for="year in years" :key="`year-${year}`" :value="year">{{ year }} 年</option>
              </select>
              <select v-model.number="viewMonth">
                <option v-for="month in months" :key="`month-${month}`" :value="month">{{ month }} 月</option>
              </select>
            </div>
            <button type="button" class="app-date-panel__arrow" @click="nextMonth">›</button>
          </header>

          <div class="app-date-panel__week">
            <span v-for="weekLabel in WEEK_LABELS" :key="`wk-${weekLabel}`">{{ weekLabel }}</span>
          </div>

          <div class="app-date-panel__days">
            <button
              v-for="cell in cells"
              :key="cell.iso"
              type="button"
              :class="[
                'app-date-panel__day',
                { 'is-out': !cell.inMonth, 'is-selected': cell.selected, 'is-today': cell.today },
              ]"
              @click="selectISO(cell.iso)"
            >
              {{ cell.label }}
            </button>
          </div>

          <footer class="app-date-panel__footer">
            <button type="button" class="ghost-btn small-btn" @click="clearValue">清空</button>
            <button type="button" class="ghost-btn small-btn" @click="pickToday">今天</button>
          </footer>
        </section>
      </transition>
    </teleport>
  </div>
</template>

<style scoped>
.app-date-input {
  width: 100%;
  min-width: 0;
  display: inline-flex;
  align-items: stretch;
  border: 1px solid #c7d4e5;
  border-radius: 10px;
  background: #fff;
  overflow: hidden;
}

.app-date-input.is-open {
  border-color: #4f87cf;
  box-shadow: 0 0 0 3px rgba(79, 135, 207, 0.18);
}

.app-date-input.is-disabled {
  opacity: 0.72;
  background: #f3f6fa;
}

.app-date-input__field {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: none;
  background: transparent;
  padding: 7px 10px;
  line-height: 1.35;
  font: inherit;
  color: #14314f;
}

.app-date-input__field::placeholder {
  color: #8ea4ba;
}

.app-date-input__trigger {
  width: 46px;
  border: 0;
  border-left: 1px solid #d8e1ed;
  background: #eef5ff;
  color: #2d5f9a;
  cursor: pointer;
  font-size: 14px;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.app-date-input__trigger:hover {
  background: #e4f0ff;
}

.app-date-input__trigger:disabled {
  cursor: not-allowed;
  color: #8ea4ba;
  background: #f3f6fa;
}

.app-date-panel {
  z-index: 14000;
  max-height: calc(100vh - 16px);
  border: 1px solid #c6d9ed;
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 20px 38px rgba(14, 42, 71, 0.18);
  overflow: auto;
}

.app-date-panel.is-top {
  transform-origin: bottom center;
}

.app-date-panel.is-bottom {
  transform-origin: top center;
}

.app-date-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px 14px 8px;
  background: linear-gradient(180deg, #f7fbff 0%, #edf5ff 100%);
  border-bottom: 1px solid #d9e5f2;
}

.app-date-panel__pickers {
  flex: 1;
  display: flex;
  gap: 8px;
}

.app-date-panel__pickers select {
  flex: 1;
  min-width: 0;
  border: 1px solid #c7d4e5;
  border-radius: 8px;
  padding: 6px 8px;
  font: inherit;
  color: #163451;
  background: #fff;
}

.app-date-panel__arrow {
  width: 32px;
  height: 32px;
  border: 1px solid #c7d4e5;
  border-radius: 8px;
  background: #fff;
  color: #234f7f;
  cursor: pointer;
  font-size: 18px;
  line-height: 1;
}

.app-date-panel__arrow:hover {
  background: #edf5ff;
}

.app-date-panel__week,
.app-date-panel__days {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
}

.app-date-panel__week {
  padding: 8px 12px 4px;
  color: #6c86a1;
  font-size: 12px;
  text-align: center;
}

.app-date-panel__days {
  padding: 0 12px 12px;
  gap: 4px;
}

.app-date-panel__day {
  height: 34px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: #17314b;
  cursor: pointer;
  font: inherit;
}

.app-date-panel__day:hover {
  background: #eef5ff;
  border-color: #d4e4f6;
}

.app-date-panel__day.is-out {
  color: #9db0c4;
}

.app-date-panel__day.is-selected {
  background: linear-gradient(180deg, #3d8ad8 0%, #2d6fbc 100%);
  border-color: #2d6fbc;
  color: #fff;
}

.app-date-panel__day.is-today:not(.is-selected) {
  border-color: #79a8da;
  color: #225b98;
  font-weight: 700;
}

.app-date-panel__footer {
  display: flex;
  justify-content: space-between;
  padding: 10px 12px 12px;
  border-top: 1px solid #dbe6f2;
  background: #fbfdff;
}

.app-date-panel-pop-enter-active,
.app-date-panel-pop-leave-active {
  transition:
    opacity 0.18s ease,
    transform 0.18s ease;
}

.app-date-panel.is-bottom.app-date-panel-pop-enter-from,
.app-date-panel.is-bottom.app-date-panel-pop-leave-to {
  opacity: 0;
  transform: translateY(-6px) scale(0.98);
}

.app-date-panel.is-top.app-date-panel-pop-enter-from,
.app-date-panel.is-top.app-date-panel-pop-leave-to {
  opacity: 0;
  transform: translateY(6px) scale(0.98);
}
</style>
