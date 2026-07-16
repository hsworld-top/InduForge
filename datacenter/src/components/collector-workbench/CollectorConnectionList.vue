<template>
  <aside class="collector-connection-list">
    <div class="collector-connection-list__search-row">
      <el-input
        v-model="search"
        clearable
        :prefix-icon="IconTablerSearch"
        placeholder="搜索连接名称或协议"
        @keyup.enter="reload(1)"
        @clear="reload(1)"
      />
      <button
        type="button"
        aria-label="收起连接列表"
        title="收起连接列表"
        @click="emit('collapse')"
      >
        <IconTablerLayoutSidebarLeftCollapse />
      </button>
    </div>

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
          <CollectorDriverIcon :protocol-family="group.key" />
          <span :title="group.label">{{ group.label }}</span>
          <em>{{ group.items.length }}</em>
        </button>
        <div
          v-show="!collapsedGroups.has(group.key)"
          class="collector-connection-list__group-items"
        >
          <el-dropdown
            v-for="item in group.items"
            :key="item.id"
            class="collector-connection-list__context"
            trigger="contextmenu"
            @command="handleItemCommand($event, item)"
          >
            <div
              role="button"
              tabindex="0"
              :class="['collector-connection-list__item', { 'is-active': item.id === selectedId }]"
              @click="emit('select', item.id)"
              @keydown.enter="emit('select', item.id)"
            >
              <span
                class="collector-connection-list__pulse"
                :class="`is-${formatCollectorConnectionStatus(item.status).tone}`"
              />
              <span class="collector-connection-list__identity">
                <strong>{{ item.name }}</strong>
                <small>{{ formatCollectorConnectionSummary(item) }}</small>
              </span>
              <el-dropdown
                trigger="click"
                placement="bottom-end"
                @command="handleItemCommand($event, item)"
              >
                <button
                  type="button"
                  class="collector-connection-list__more"
                  :aria-label="`${item.name}更多操作`"
                  @click.stop
                >
                  <IconTablerDots />
                </button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="delete">
                      <IconTablerTrash />删除连接
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="delete"> <IconTablerTrash />删除连接 </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
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
import { deleteCollectorConnection, listCollectorConnections } from '@/api/collector.api'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { CollectorConnection } from '@/api/schemas/collector.schema'
import {
  formatCollectorConnectionStatus,
  formatCollectorConnectionSummary,
  groupCollectorConnections,
} from './collector-workbench-model'
import CollectorDriverIcon from './CollectorDriverIcon.vue'
import IconTablerChevronDown from '~icons/tabler/chevron-down'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerDots from '~icons/tabler/dots'
import IconTablerLayoutSidebarLeftCollapse from '~icons/tabler/layout-sidebar-left-collapse'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerSearch from '~icons/tabler/search'

const props = defineProps<{ projectId: string; selectedId?: string }>()
const emit = defineEmits<{
  select: [id: string]
  collapse: []
  loaded: [items: CollectorConnection[]]
  deleted: [id: string]
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

async function handleItemCommand(command: string, item: CollectorConnection) {
  if (command !== 'delete') return
  await removeConnection(item)
}

async function removeConnection(item: CollectorConnection) {
  // 删除连接会级联删除该连接下的变量分组和变量，必须在执行前明确提示影响范围。
  await ElMessageBox.confirm(
    `确认删除连接“${item.name}”？该连接下的变量分组和变量也会一并删除。`,
    '删除工业连接',
    {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      confirmButtonClass: 'el-button--danger',
    },
  )
  await deleteCollectorConnection(props.projectId, item.id)
  emit('deleted', item.id)
  ElMessage.success('工业连接已删除')
  await reload(items.value.length === 1 && page.value > 1 ? page.value - 1 : page.value)
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
  width: 320px;
  min-width: 320px;
  min-height: 0;
  flex-direction: column;
  gap: 12px;
  padding: 16px 14px 12px;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}
.collector-connection-list__search-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 32px;
  align-items: center;
  gap: 8px;
}
.collector-connection-list__search-row > button {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-muted);
  cursor: pointer;
}
.collector-connection-list__search-row > button:hover {
  border-color: var(--dc-primary);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.collector-connection-list__search-row svg {
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
  grid-template-columns: 16px 20px minmax(0, 1fr) auto;
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
  overflow: hidden;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
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
.collector-connection-list__context {
  display: block;
  width: 100%;
}
.collector-connection-list__item {
  display: grid;
  width: 100%;
  grid-template-columns: 9px minmax(0, 1fr) 30px;
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
.collector-connection-list__item:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--dc-primary) 45%, transparent);
  outline-offset: -2px;
}
.collector-connection-list__more {
  display: grid;
  width: 28px;
  height: 28px;
  place-items: center;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--dc-text-muted);
  cursor: pointer;
  opacity: 0.55;
}
.collector-connection-list__item:hover .collector-connection-list__more,
.collector-connection-list__item.is-active .collector-connection-list__more,
.collector-connection-list__more:focus-visible {
  opacity: 1;
}
.collector-connection-list__more:hover {
  background: color-mix(in srgb, var(--dc-primary) 9%, transparent);
  color: var(--dc-primary);
}
.collector-connection-list__more svg {
  width: 18px;
  height: 18px;
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
.collector-connection-list__pulse {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--dc-border-strong);
}
.collector-connection-list__pulse.is-primary {
  background: var(--dc-primary);
}
.collector-connection-list__pulse.is-success {
  background: var(--dc-success);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--dc-success) 13%, transparent);
}
.collector-connection-list__pulse.is-warning {
  background: #d99000;
}
.collector-connection-list__pulse.is-danger {
  background: var(--dc-danger);
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
