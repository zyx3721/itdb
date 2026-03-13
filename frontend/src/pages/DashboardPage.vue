<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import api from '../api/client'

type Summary = {
  counts: Record<string, number>
}

type ModuleAction = {
  label: string
  to: RouteLocationRaw
  icon?: string
}

type HomeModule = {
  key: string
  title: string
  description: string
  image: string
  countKey?: string
  entry: RouteLocationRaw
  actions: ModuleAction[]
}

const summary = ref<Summary | null>(null)
const loading = ref(false)
const error = ref('')

function withCreateQuery(to: RouteLocationRaw): RouteLocationRaw {
  if (typeof to === 'string') {
    return { path: to, query: { create: '1' } }
  }
  const next = { ...(to as Record<string, unknown>) }
  next.query = { ...((next.query as Record<string, unknown>) ?? {}), create: '1' }
  return next as RouteLocationRaw
}

const modules: HomeModule[] = [
  {
    key: 'items',
    title: '硬件',
    description: '管理硬件资产，包含服务器、交换机、电话等设备。',
    image: '/images/big/hardware.png',
    countKey: 'items',
    entry: '/resources/items',
    actions: [
      { label: '查找', to: '/resources/items', icon: '/images/big/search24.png' },
      { label: '新增', to: { path: '/resources/items', query: { create: '1' } }, icon: '/images/big/plus.png' },
      { label: '硬件类型', to: '/dictionaries/itemtypes', icon: '/images/big/wheel24.png' },
      { label: '状态类型', to: '/dictionaries/statustypes', icon: '/images/big/wheel24.png' },
    ],
  },
  {
    key: 'software',
    title: '软件',
    description: '管理软件授权、版本和安装关系。',
    image: '/images/big/software.png',
    countKey: 'software',
    entry: '/resources/software',
    actions: [
      { label: '查找', to: '/resources/software', icon: '/images/big/search24.png' },
      { label: '新增', to: { path: '/resources/software', query: { create: '1' } }, icon: '/images/big/plus.png' },
    ],
  },
  {
    key: 'invoices',
    title: '单据',
    description: '管理采购单据及其关联信息。',
    image: '/images/big/document.png',
    countKey: 'invoices',
    entry: '/resources/invoices',
    actions: [
      { label: '查找', to: '/resources/invoices', icon: '/images/big/search24.png' },
      { label: '新增', to: { path: '/resources/invoices', query: { create: '1' } }, icon: '/images/big/plus.png' },
    ],
  },
  {
    key: 'reports',
    title: '报告',
    description: '查看统计报表与分析结果。',
    image: '/images/big/pie.png',
    entry: '/reports',
    actions: [{ label: '查看报告', to: '/reports', icon: '/images/big/spreadsheet24.png' }],
  },
  {
    key: 'contracts',
    title: '合同',
    description: '管理支持、授权、租赁等合同。',
    image: '/images/big/contract.png',
    countKey: 'contracts',
    entry: '/resources/contracts',
    actions: [
      { label: '查找', to: '/resources/contracts', icon: '/images/big/search24.png' },
      { label: '新增', to: { path: '/resources/contracts', query: { create: '1' } }, icon: '/images/big/plus.png' },
      { label: '合同类型', to: '/dictionaries/contracttypes', icon: '/images/big/wheel24.png' },
    ],
  },
  {
    key: 'agents',
    title: '代理',
    description: '管理厂商、供应商、采购方与承包方。',
    image: '/images/big/company.png',
    countKey: 'agents',
    entry: '/resources/agents',
    actions: [
      { label: '查找', to: '/resources/agents', icon: '/images/big/search24.png' },
      { label: '新增', to: { path: '/resources/agents', query: { create: '1' } }, icon: '/images/big/plus.png' },
    ],
  },
  {
    key: 'browse',
    title: '浏览数据',
    description: '按类型、用户、代理等维度进行树形浏览。',
    image: '/images/big/view_tree.png',
    entry: '/browse',
    actions: [{ label: '浏览', to: '/browse', icon: '/images/big/search24.png' }],
  },
  {
    key: 'files',
    title: '文件',
    description: '维护文件及其与资产的关联关系。',
    image: '/images/big/files128.png',
    countKey: 'files',
    entry: '/resources/files',
    actions: [
      { label: '查找', to: '/resources/files', icon: '/images/big/search24.png' },
      { label: '新增', to: { path: '/resources/files', query: { create: '1' } }, icon: '/images/big/plus.png' },
    ],
  },
  {
    key: 'racks',
    title: '机架',
    description: '新增与查看机架及占用状态。',
    image: '/images/big/rack1.png',
    countKey: 'racks',
    entry: '/resources/racks',
    actions: [
      { label: '查找', to: '/resources/racks', icon: '/images/big/search24.png' },
      { label: '新增', to: { path: '/resources/racks', query: { create: '1' } }, icon: '/images/big/plus.png' },
    ],
  },
  {
    key: 'locations',
    title: '地点',
    description: '管理地点、楼层及区域信息。',
    image: '/images/big/location.png',
    countKey: 'locations',
    entry: '/resources/locations',
    actions: [
      { label: '查找', to: '/resources/locations', icon: '/images/big/search24.png' },
      { label: '新增', to: { path: '/resources/locations', query: { create: '1' } }, icon: '/images/big/plus.png' },
    ],
  },
  {
    key: 'labels',
    title: '打印标签',
    description: '选择并打印资产标签。',
    image: '/images/big/labels.png',
    entry: '/labels',
    actions: [{ label: '进入打印', to: '/labels', icon: '/images/big/labels.png' }],
  },
  {
    key: 'settings',
    title: '设置',
    description: '维护系统参数、用户、标记等基础配置。',
    image: '/images/big/settings.png',
    entry: '/settings',
    actions: [
      { label: '系统设置', to: '/settings', icon: '/images/big/wheel24.png' },
      { label: '用户', to: '/resources/users', icon: '/images/big/users.png' },
      { label: '标记', to: '/dictionaries/tags', icon: '/images/big/tag.png' },
    ],
  },
]

const cards = computed(() => {
  const counts = summary.value?.counts ?? {}
  return modules.map((module) => ({
    ...module,
    actions: module.actions.map((action) =>
      action.label === '新增'
        ? {
            ...action,
            to: withCreateQuery(action.to),
          }
        : action,
    ),
    count: module.countKey ? Number(counts[module.countKey] ?? 0) : null,
  }))
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await api.get<Summary>('/dashboard/summary')
    summary.value = data
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '首页数据加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="page-shell">
    <header class="page-header">
      <h2>首页</h2>
      <button class="ghost-btn" @click="load">刷新</button>
    </header>

    <p v-if="loading">加载中...</p>

    <div v-else class="legacy-home-grid">
      <article v-for="card in cards" :key="card.key" class="legacy-home-card">
        <RouterLink
          :to="card.entry"
          class="legacy-home-media-link quick-tip"
          :data-quick-tip="`进入${card.title}`"
        >
          <img :src="card.image" :alt="card.title" class="legacy-home-media" loading="lazy" />
        </RouterLink>

        <div class="legacy-home-body">
          <div class="legacy-home-head">
            <RouterLink :to="card.entry" class="legacy-home-title">{{ card.title }}</RouterLink>
            <span class="legacy-home-count" :class="{ 'is-muted': card.count === null }">
              {{ card.count === null ? '功能入口' : `总数：${card.count}` }}
            </span>
          </div>

          <p class="legacy-home-desc">{{ card.description }}</p>

          <div class="legacy-home-actions">
            <RouterLink v-for="action in card.actions" :key="`${card.key}-${action.label}`" :to="action.to" class="legacy-action-link">
              <img v-if="action.icon" :src="action.icon" :alt="action.label" class="legacy-action-icon" loading="lazy" />
              <span>{{ action.label }}</span>
            </RouterLink>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>

<style scoped>
.legacy-home-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(460px, 1fr));
  gap: 14px;
}

.legacy-home-card {
  display: grid;
  grid-template-columns: 92px 1fr;
  gap: 14px;
  align-items: start;
  border: 1px solid var(--line);
  border-radius: 14px;
  background: linear-gradient(180deg, #ffffff 0%, #f9fcff 100%);
  box-shadow: 0 8px 20px rgba(12, 42, 66, 0.06);
  padding: 12px;
}

.legacy-home-media-link {
  width: 92px;
  height: 92px;
  border-radius: 10px;
  border: 1px solid #d8e4ef;
  background: #f6fbff;
  display: grid;
  place-items: center;
  text-decoration: none;
}

.legacy-home-media {
  width: 72px;
  height: 72px;
  object-fit: contain;
}

.legacy-home-body {
  display: grid;
  gap: 10px;
  min-width: 0;
}

.legacy-home-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}

.legacy-home-title {
  color: #14456c;
  font-weight: 700;
  font-size: 1.08rem;
  text-decoration: none;
}

.legacy-home-title:hover {
  color: #0f6b62;
}

.legacy-home-count {
  color: #2a7cb7;
  font-size: 0.9rem;
  font-weight: 600;
  white-space: nowrap;
}

.legacy-home-count.is-muted {
  color: #6a8097;
}

.legacy-home-desc {
  margin: 0;
  color: #4d657c;
  line-height: 1.35;
  min-height: 2.7em;
}

.legacy-home-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.legacy-action-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 1px solid #c8d8e7;
  background: #f6fbff;
  border-radius: 8px;
  padding: 4px 10px;
  color: #28587d;
  text-decoration: none;
  font-size: 0.9rem;
}

.legacy-action-link:hover {
  background: #ebf5ff;
  border-color: #9ec0dc;
}

.legacy-action-icon {
  width: 16px;
  height: 16px;
  object-fit: contain;
}

@media (max-width: 1100px) {
  .legacy-home-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .legacy-home-card {
    grid-template-columns: 1fr;
  }

  .legacy-home-media-link {
    width: 100%;
    height: 96px;
  }
}
</style>
