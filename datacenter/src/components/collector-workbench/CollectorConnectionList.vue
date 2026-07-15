<template>
  <aside class="collector-connection-list">
    <div class="collector-connection-list__head">
      <div>
        <small>工业采集</small>
        <strong>已配置连接</strong>
      </div>
      <button type="button" aria-label="新增工业连接" @click="emit('create')">
        <IconTablerPlus />
      </button>
    </div>

    <el-input
      v-model="search"
      clearable
      :prefix-icon="IconTablerSearch"
      placeholder="搜索连接名称或协议"
      @keyup.enter="reload(1)"
      @clear="reload(1)"
    />

    <div v-loading="loading" class="collector-connection-list__items">
      <section
        v-for="group in groupedItems"
        :key="group.key"
        class="collector-connection-list__group"
      >
        <button
          type="button"
          class="collector-connection-list__group-head"
          @click="toggleGroup(group.key)"
        >
          <component
            :is="collapsedGroups.has(group.key) ? IconTablerChevronRight : IconTablerChevronDown"
          />
          <span>{{ group.label }}</span>
          <em>{{ group.items.length }}</em>
        </button>
        <div
          v-show="!collapsedGroups.has(group.key)"
          class="collector-connection-list__group-items"
        >
          <button
            v-for="item in group.items"
            :key="item.id"
            type="button"
            :class="['collector-connection-list__item', { 'is-active': item.id === selectedId }]"
            @click="emit('select', item.id)"
          >
            <span
              class="collector-connection-list__pulse"
              :class="{ 'is-enabled': item.enabled }"
            />
            <span class="collector-connection-list__identity">
              <strong>{{ item.name }}</strong>
              <small>{{ item.driverId }}</small>
            </span>
            <em>{{ item.enabled ? '已启用' : '已停用' }}</em>
          </button>
        </div>
      </section>
      <el-empty v-if="!loading && !items.length" description="暂无工业连接" :image-size="64" />
    </div>

    <div class="collector-connection-list__footer">
      <span>共 {{ total }} 个连接</span>
      <el-pagination
        v-if="total > pageSize"
        small
        layout="prev, pager, next"
        :pager-count="5"
        :current-page="page"
        :page-size="pageSize"
        :total="total"
        @current-change="reload"
      />
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { listCollectorConnections } from '@/api/collector.api'
import type { CollectorConnection } from '@/api/schemas/collector.schema'
import { groupCollectorConnections } from './collector-workbench-model'
import IconTablerChevronDown from '~icons/tabler/chevron-down'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerSearch from '~icons/tabler/search'

const props = defineProps<{ projectId: string; selectedId?: string }>()
const emit = defineEmits<{
  select: [id: string]
  create: []
  loaded: [items: CollectorConnection[]]
}>()
const items = ref<CollectorConnection[]>([])
const loading = ref(false)
const search = ref('')
const page = ref(1)
const pageSize = 20
const total = ref(0)
const collapsedGroups = ref(new Set<string>())
const groupedItems = computed(() => groupCollectorConnections(items.value))

function toggleGroup(key: string) {
  const next = new Set(collapsedGroups.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  collapsedGroups.value = next
}

async function reload(nextPage = page.value) {
  page.value = nextPage
  loading.value = true
  try {
    const result = await listCollectorConnections(props.projectId, {
      page: page.value,
      pageSize,
      search: search.value,
    })
    items.value = result.list
    total.value = result.pagination.total
    emit('loaded', result.list)
  } finally {
    loading.value = false
  }
}

defineExpose({ reload })
watch(
  () => props.projectId,
  () => reload(1),
)
onMounted(() => reload())
</script>

<style scoped>
.collector-connection-list {
  display: flex;
  width: 292px;
  min-width: 292px;
  min-height: 0;
  flex-direction: column;
  gap: 12px;
  padding: 16px 14px 12px;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}
.collector-connection-list__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 2px;
}
.collector-connection-list__head > div {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.collector-connection-list__head small {
  color: var(--dc-text-muted);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.1em;
}
.collector-connection-list__head strong {
  color: var(--dc-text);
  font-size: 16px;
}
.collector-connection-list__head > button {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  border: 1px solid var(--dc-primary);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary);
  color: #fff;
  cursor: pointer;
}
.collector-connection-list__head > button:hover {
  background: var(--dc-primary-hover);
}
.collector-connection-list__head svg {
  width: 17px;
  height: 17px;
}
.collector-connection-list__items {
  min-height: 0;
  flex: 1;
  overflow: auto;
  padding-right: 2px;
}
.collector-connection-list__group + .collector-connection-list__group {
  margin-top: 4px;
}
.collector-connection-list__group-head {
  display: grid;
  width: 100%;
  height: 34px;
  grid-template-columns: 16px minmax(0, 1fr) auto;
  align-items: center;
  gap: 5px;
  padding: 0 7px;
  border: none;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  cursor: pointer;
  text-align: left;
}
.collector-connection-list__group-head:hover {
  background: var(--dc-surface-subtle);
}
.collector-connection-list__group-head svg {
  width: 14px;
  height: 14px;
}
.collector-connection-list__group-head span {
  font-weight: 700;
}
.collector-connection-list__group-head em {
  min-width: 22px;
  padding: 1px 6px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 10px;
  font-style: normal;
  text-align: center;
}
.collector-connection-list__group-items {
  padding-left: 17px;
}
.collector-connection-list__item {
  display: grid;
  width: 100%;
  grid-template-columns: 9px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  margin-bottom: 3px;
  padding: 9px 8px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  cursor: pointer;
  text-align: left;
}
.collector-connection-list__item:hover {
  background: var(--dc-surface-subtle);
}
.collector-connection-list__item.is-active {
  border-color: color-mix(in srgb, var(--dc-primary) 20%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  box-shadow: inset 3px 0 0 var(--dc-primary);
}
.collector-connection-list__identity {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}
.collector-connection-list__identity strong,
.collector-connection-list__identity small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.collector-connection-list__identity strong {
  font-size: 13px;
}
.collector-connection-list__identity small {
  color: var(--dc-text-muted);
  font-size: 10px;
}
.collector-connection-list__item > em {
  color: var(--dc-text-muted);
  font-size: 10px;
  font-style: normal;
}
.collector-connection-list__item.is-active > em {
  color: var(--dc-primary);
}
.collector-connection-list__pulse {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--dc-border-strong);
}
.collector-connection-list__pulse.is-enabled {
  background: var(--dc-success);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--dc-success) 13%, transparent);
}
.collector-connection-list__footer {
  display: flex;
  min-height: 28px;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: var(--dc-text-muted);
  font-size: 11px;
}
.collector-connection-list__footer :deep(.el-pagination) {
  --el-pagination-button-width: 24px;
  --el-pagination-button-height: 24px;
}
</style>
