<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import api from '../api/client';

type Summary = {
  counts: Record<string, number>;
};

type ModuleAction = {
  label: string;
  to: RouteLocationRaw;
  iconKey?: string;
};

type HomeModule = {
  key: string;
  title: string;
  description: string;
  iconKey: string;
  countKey?: string;
  entry: RouteLocationRaw;
  actions: ModuleAction[];
};

// 模块主图标 SVG（72×72，使用 currentColor）
const moduleIcons: Record<string, string> = {
  items: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="5" rx="1"/><rect x="2" y="10" width="20" height="5" rx="1"/><rect x="2" y="17" width="20" height="5" rx="1"/><circle cx="6" cy="5.5" r="0.8" fill="currentColor" stroke="none"/><circle cx="6" cy="12.5" r="0.8" fill="currentColor" stroke="none"/><circle cx="6" cy="19.5" r="0.8" fill="currentColor" stroke="none"/></svg>`,
  software: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="16" rx="2"/><path d="M8 21h8M12 19v2"/><path d="M7 8l3 3-3 3M13 14h4"/></svg>`,
  invoices: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="8" y1="13" x2="16" y2="13"/><line x1="8" y1="17" x2="12" y2="17"/><line x1="8" y1="9" x2="10" y2="9"/></svg>`,
  reports: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21.21 15.89A10 10 0 1 1 8 2.83"/><path d="M22 12A10 10 0 0 0 12 2v10z"/></svg>`,
  contracts: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><path d="M9 15l2 2 4-4"/></svg>`,
  agents: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 21V7a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v14"/><path d="M3 21h18"/><path d="M9 21V12h6v9"/><rect x="9" y="7" width="2" height="2"/><rect x="13" y="7" width="2" height="2"/></svg>`,
  browse: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 5h2"/><path d="M3 12h5"/><path d="M3 19h2"/><circle cx="7" cy="5" r="2"/><circle cx="10" cy="12" r="2"/><circle cx="7" cy="19" r="2"/><path d="M9 5h12"/><path d="M12 12h9"/><path d="M9 19h12"/></svg>`,
  files: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>`,
  racks: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="2" width="16" height="20" rx="1"/><rect x="7" y="5" width="10" height="3" rx="0.5"/><rect x="7" y="10" width="10" height="3" rx="0.5"/><rect x="7" y="15" width="10" height="3" rx="0.5"/><circle cx="17.5" cy="6.5" r="0.6" fill="currentColor" stroke="none"/><circle cx="17.5" cy="11.5" r="0.6" fill="currentColor" stroke="none"/><circle cx="17.5" cy="16.5" r="0.6" fill="currentColor" stroke="none"/></svg>`,
  locations: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7z"/><circle cx="12" cy="9" r="2.5"/></svg>`,
  labels: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M6 2H18a2 2 0 0 1 2 2v4a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2z"/><path d="M6 12h12"/><path d="M6 16h12"/><path d="M4 8v14"/><path d="M20 8v14"/></svg>`,
  settings: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`,
};

// 操作按钮小图标 SVG
const actionIcons: Record<string, string> = {
  search: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>`,
  plus: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="16"/><line x1="8" y1="12" x2="16" y2="12"/></svg>`,
  gear: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`,
  table: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18"/><path d="M3 15h18"/><path d="M9 3v18"/></svg>`,
  printer: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 6 2 18 2 18 9"/><path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"/><rect x="6" y="14" width="12" height="8"/></svg>`,
  users: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
  tag: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"/><line x1="7" y1="7" x2="7.01" y2="7"/></svg>`,
};

const summary = ref<Summary | null>(null);
const loading = ref(false);
const error = ref('');

function withCreateQuery(to: RouteLocationRaw): RouteLocationRaw {
  if (typeof to === 'string') {
    return { path: to, query: { create: '1' } };
  }
  const next = { ...(to as Record<string, unknown>) };
  next.query = { ...((next.query as Record<string, unknown>) ?? {}), create: '1' };
  return next as RouteLocationRaw;
}

const modules: HomeModule[] = [
  {
    key: 'items',
    title: '硬件',
    description: '管理硬件资产，包含服务器、交换机、电话等设备。',
    iconKey: 'items',
    countKey: 'items',
    entry: '/resources/items',
    actions: [
      { label: '查找', to: '/resources/items', iconKey: 'search' },
      { label: '新增', to: { path: '/resources/items', query: { create: '1' } }, iconKey: 'plus' },
      { label: '硬件类型', to: '/dictionaries/itemtypes', iconKey: 'gear' },
      { label: '状态类型', to: '/dictionaries/statustypes', iconKey: 'gear' },
    ],
  },
  {
    key: 'software',
    title: '软件',
    description: '管理软件授权、版本和安装关系。',
    iconKey: 'software',
    countKey: 'software',
    entry: '/resources/software',
    actions: [
      { label: '查找', to: '/resources/software', iconKey: 'search' },
      {
        label: '新增',
        to: { path: '/resources/software', query: { create: '1' } },
        iconKey: 'plus',
      },
    ],
  },
  {
    key: 'invoices',
    title: '单据',
    description: '管理采购单据及其关联信息。',
    iconKey: 'invoices',
    countKey: 'invoices',
    entry: '/resources/invoices',
    actions: [
      { label: '查找', to: '/resources/invoices', iconKey: 'search' },
      {
        label: '新增',
        to: { path: '/resources/invoices', query: { create: '1' } },
        iconKey: 'plus',
      },
    ],
  },
  {
    key: 'reports',
    title: '报告',
    description: '查看统计报表与分析结果。',
    iconKey: 'reports',
    entry: '/reports',
    actions: [{ label: '查看报告', to: '/reports', iconKey: 'table' }],
  },
  {
    key: 'contracts',
    title: '合同',
    description: '管理支持、授权、租赁等合同。',
    iconKey: 'contracts',
    countKey: 'contracts',
    entry: '/resources/contracts',
    actions: [
      { label: '查找', to: '/resources/contracts', iconKey: 'search' },
      {
        label: '新增',
        to: { path: '/resources/contracts', query: { create: '1' } },
        iconKey: 'plus',
      },
      { label: '合同类型', to: '/dictionaries/contracttypes', iconKey: 'gear' },
    ],
  },
  {
    key: 'agents',
    title: '代理',
    description: '管理厂商、供应商、采购方与承包方。',
    iconKey: 'agents',
    countKey: 'agents',
    entry: '/resources/agents',
    actions: [
      { label: '查找', to: '/resources/agents', iconKey: 'search' },
      { label: '新增', to: { path: '/resources/agents', query: { create: '1' } }, iconKey: 'plus' },
    ],
  },
  {
    key: 'browse',
    title: '浏览数据',
    description: '按类型、用户、代理等维度进行树形浏览。',
    iconKey: 'browse',
    entry: '/browse',
    actions: [{ label: '浏览', to: '/browse', iconKey: 'search' }],
  },
  {
    key: 'files',
    title: '文件',
    description: '维护文件及其与资产的关联关系。',
    iconKey: 'files',
    countKey: 'files',
    entry: '/resources/files',
    actions: [
      { label: '查找', to: '/resources/files', iconKey: 'search' },
      { label: '新增', to: { path: '/resources/files', query: { create: '1' } }, iconKey: 'plus' },
    ],
  },
  {
    key: 'racks',
    title: '机架',
    description: '新增与查看机架及占用状态。',
    iconKey: 'racks',
    countKey: 'racks',
    entry: '/resources/racks',
    actions: [
      { label: '查找', to: '/resources/racks', iconKey: 'search' },
      { label: '新增', to: { path: '/resources/racks', query: { create: '1' } }, iconKey: 'plus' },
    ],
  },
  {
    key: 'locations',
    title: '地点',
    description: '管理地点、楼层及区域信息。',
    iconKey: 'locations',
    countKey: 'locations',
    entry: '/resources/locations',
    actions: [
      { label: '查找', to: '/resources/locations', iconKey: 'search' },
      {
        label: '新增',
        to: { path: '/resources/locations', query: { create: '1' } },
        iconKey: 'plus',
      },
    ],
  },
  {
    key: 'labels',
    title: '打印标签',
    description: '选择并打印资产标签。',
    iconKey: 'labels',
    entry: '/labels',
    actions: [{ label: '进入打印', to: '/labels', iconKey: 'printer' }],
  },
  {
    key: 'settings',
    title: '设置',
    description: '维护系统参数、用户、标记等基础配置。',
    iconKey: 'settings',
    entry: '/settings',
    actions: [
      { label: '系统设置', to: '/settings', iconKey: 'gear' },
      { label: '用户', to: '/resources/users', iconKey: 'users' },
      { label: '标记', to: '/dictionaries/tags', iconKey: 'tag' },
    ],
  },
];

const cards = computed(() => {
  const counts = summary.value?.counts ?? {};
  return modules.map(module => ({
    ...module,
    actions: module.actions.map(action =>
      action.label === '新增' ? { ...action, to: withCreateQuery(action.to) } : action
    ),
    count: module.countKey ? Number(counts[module.countKey] ?? 0) : null,
  }));
});

async function load() {
  loading.value = true;
  error.value = '';
  try {
    const { data } = await api.get<Summary>('/dashboard/summary');
    summary.value = data;
  } catch (err: unknown) {
    error.value =
      (err as { response?: { data?: { error?: string } } })?.response?.data?.error ??
      '首页数据加载失败';
  } finally {
    loading.value = false;
  }
}

onMounted(load);
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
          <span class="legacy-home-media" v-html="moduleIcons[card.iconKey]" />
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
            <RouterLink
              v-for="action in card.actions"
              :key="`${card.key}-${action.label}`"
              :to="action.to"
              class="legacy-action-link"
            >
              <span
                v-if="action.iconKey"
                class="legacy-action-icon"
                v-html="actionIcons[action.iconKey]"
              />
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
  color: #2a7cb7;
  transition:
    background 0.15s,
    color 0.15s;
}

.legacy-home-media-link:hover {
  background: #e8f4ff;
  color: #0f6b62;
}

.legacy-home-media {
  width: 52px;
  height: 52px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.legacy-home-media :deep(svg) {
  width: 52px;
  height: 52px;
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
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.legacy-action-icon :deep(svg) {
  width: 16px;
  height: 16px;
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
