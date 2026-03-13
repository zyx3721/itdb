<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api/client'

type BrowseNode = {
  id: string
  label: string
  leaf: boolean
  resource?: string
  entityId?: number
}

type TreeRow = {
  node: BrowseNode
  prefix: string
  expanded: boolean
  loading: boolean
}

const router = useRouter()

const ROOT_ID = '0'

const nodesById = ref<Record<string, BrowseNode>>({})
const childrenByParent = ref<Record<string, string[]>>({})
const expandedById = ref<Record<string, boolean>>({})
const loadingByParent = ref<Record<string, boolean>>({})
const selectedId = ref('')
const error = ref('')

function setChildren(parentId: string, nodes: BrowseNode[]) {
  for (const node of nodes) {
    nodesById.value[node.id] = node
  }
  childrenByParent.value[parentId] = nodes.map((node) => node.id)
}

function isExpanded(id: string) {
  return !!expandedById.value[id]
}

function isLoading(parentId: string) {
  return !!loadingByParent.value[parentId]
}

function hasLoadedChildren(parentId: string) {
  return Object.prototype.hasOwnProperty.call(childrenByParent.value, parentId)
}

function getDirectChildCount(nodeId: string) {
  return (childrenByParent.value[nodeId] ?? []).length
}

function shouldShowBottomCount(node: BrowseNode) {
  if (node.leaf) return false
  if (!isExpanded(node.id)) return false
  if (!hasLoadedChildren(node.id)) return false
  const childIds = childrenByParent.value[node.id] ?? []
  if (childIds.length === 0) return true
  return childIds.every((childId) => nodesById.value[childId]?.leaf)
}

async function loadChildren(parentId: string) {
  if (isLoading(parentId)) return
  loadingByParent.value[parentId] = true
  error.value = ''
  try {
    const { data } = await api.get<BrowseNode[]>('/browse/tree', { params: { id: parentId } })
    setChildren(parentId, data)
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '浏览树加载失败'
  } finally {
    loadingByParent.value[parentId] = false
  }
}

async function toggleNode(node: BrowseNode) {
  if (node.leaf) return
  const nextExpanded = !isExpanded(node.id)
  expandedById.value[node.id] = nextExpanded
  if (nextExpanded && !hasLoadedChildren(node.id)) {
    await loadChildren(node.id)
  }
}

async function choose(node: BrowseNode) {
  selectedId.value = node.id
  if (node.leaf) {
    if (node.resource) {
      const query = node.entityId && node.entityId > 0 ? { edit: String(node.entityId) } : undefined
      await router.push({ path: `/resources/${node.resource}`, query })
    }
    return
  }
  await toggleNode(node)
}

const treeRows = computed<TreeRow[]>(() => {
  const rows: TreeRow[] = []

  function walk(parentId: string, ancestorLastFlags: boolean[]) {
    const childIds = childrenByParent.value[parentId] ?? []
    childIds.forEach((id, index) => {
      const node = nodesById.value[id]
      if (!node) return

      const isLast = index === childIds.length - 1
      let prefix = ''
      for (const flag of ancestorLastFlags) {
        prefix += flag ? '   ' : '│  '
      }
      prefix += isLast ? '└─ ' : '├─ '

      rows.push({
        node,
        prefix,
        expanded: isExpanded(id),
        loading: isLoading(id),
      })

      if (!node.leaf && isExpanded(id)) {
        walk(id, [...ancestorLastFlags, isLast])
      }
    })
  }

  walk(ROOT_ID, [])
  return rows
})

async function resetTree() {
  nodesById.value = {}
  childrenByParent.value = {}
  expandedById.value = {}
  loadingByParent.value = {}
  selectedId.value = ''
  await loadChildren(ROOT_ID)
}

onMounted(() => {
  void resetTree()
})
</script>

<template>
  <section class="page-shell">
    <header class="page-header">
      <h2>浏览数据</h2>
      <button class="ghost-btn" @click="resetTree">重置</button>
    </header>
    <p v-if="isLoading(ROOT_ID)">加载中...</p>

    <div class="tree-wrap">
      <ul class="tree-list">
        <li v-for="row in treeRows" :key="row.node.id" class="tree-row">
          <button class="tree-node" :class="{ active: selectedId === row.node.id, leaf: row.node.leaf }" @click="choose(row.node)">
            <span class="tree-prefix mono">{{ row.prefix }}</span>
            <span class="tree-toggle">{{ row.node.leaf ? '↗' : row.expanded ? '▾' : '▸' }}</span>
            <span class="tree-label">{{ row.node.label }}</span>
            <span class="tree-spacer" />
            <span
              v-if="shouldShowBottomCount(row.node)"
              class="tree-count"
              :class="{ empty: getDirectChildCount(row.node.id) === 0 }"
            >
              共 {{ getDirectChildCount(row.node.id) }} 条
            </span>
            <span v-if="row.loading" class="tree-loading">加载中...</span>
          </button>
        </li>
        <li v-if="!isLoading(ROOT_ID) && treeRows.length === 0" class="muted-text">空</li>
      </ul>
    </div>
  </section>
</template>

<style scoped>
.tree-wrap {
  border: 1px solid var(--line);
  border-radius: 12px;
  background: var(--surface-soft);
  padding: 10px 12px;
}

.tree-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 2px;
}

.tree-row {
  margin: 0;
  padding: 0;
}

.tree-node {
  width: 100%;
  min-height: 30px;
  border: 1px solid transparent;
  background: transparent;
  text-align: left;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 6px;
  color: var(--text);
  border-radius: 8px;
  transition: background-color 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
}

.tree-node:hover {
  background: rgba(13, 148, 136, 0.08);
}

.tree-node.active {
  border-color: rgba(13, 148, 136, 0.45);
  background: rgba(13, 148, 136, 0.12);
}

.tree-prefix {
  white-space: pre;
  color: #6d8096;
  flex: 0 0 auto;
}

.tree-toggle {
  width: 16px;
  text-align: center;
  color: #2f4763;
  flex: 0 0 auto;
}

.tree-label {
  display: block;
  min-width: 0;
  flex: 0 1 auto;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tree-spacer {
  flex: 1 1 auto;
  min-width: 8px;
}

.tree-count {
  flex: 0 0 auto;
  margin-left: 8px;
  padding: 2px 10px;
  border-radius: 999px;
  border: 1px solid #c7d9f6;
  background: linear-gradient(180deg, #edf4ff 0%, #dce9ff 100%);
  color: #214f9f;
  font-size: 12px;
  line-height: 1.35;
  font-weight: 600;
  white-space: nowrap;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.9);
}

.tree-count.empty {
  border-color: #d9dee8;
  background: linear-gradient(180deg, #f4f6f9 0%, #e8edf4 100%);
  color: #6c7789;
}

.tree-node.active .tree-count {
  border-color: #9fbee9;
  background: linear-gradient(180deg, #dfeeff 0%, #cddff8 100%);
}

.tree-node:focus-visible {
  outline: 2px solid rgba(37, 99, 235, 0.35);
  outline-offset: 1px;
}

.tree-node.leaf {
  border-color: transparent;
}

.tree-node.leaf .tree-toggle {
  color: #2563eb;
}

.tree-node.leaf .tree-label {
  color: #1e40af;
  text-decoration: none;
  background-image: linear-gradient(currentColor, currentColor);
  background-repeat: no-repeat;
  background-size: 100% 1px;
  background-position: 0 100%;
  font-weight: 500;
  transition: color 0.18s ease, background-size 0.18s ease;
}

.tree-node.leaf:hover {
  background: linear-gradient(90deg, rgba(37, 99, 235, 0.1), rgba(37, 99, 235, 0.03));
  border-color: rgba(37, 99, 235, 0.24);
  box-shadow: inset 0 0 0 1px rgba(37, 99, 235, 0.08);
}

.tree-node.leaf:hover .tree-label {
  color: #1d4ed8;
  background-size: 100% 2px;
}

.tree-node.leaf.active {
  border-color: rgba(37, 99, 235, 0.35);
  background: linear-gradient(90deg, rgba(37, 99, 235, 0.16), rgba(37, 99, 235, 0.05));
}

.tree-loading {
  margin-left: 8px;
  color: var(--text-muted);
  font-size: 12px;
  white-space: nowrap;
}
</style>
