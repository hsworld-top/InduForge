<template>
  <aside class="collector-connection-list">
    <div class="collector-connection-list__search-row">
      <el-input
        v-model="search"
        clearable
        :prefix-icon="IconTablerSearch"
        :placeholder="ui('搜索连接名称或协议', 'Search connections or protocols')"
        @keyup.enter="applySearch"
        @clear="applySearch"
      />
      <button
        type="button"
        :aria-label="ui('收起连接列表', 'Collapse Connection List')"
        :title="ui('收起连接列表', 'Collapse Connection List')"
        @click="emit('collapse')"
      >
        <IconTablerLayoutSidebarLeftCollapse />
      </button>
    </div>

    <div
      ref="itemsContainer"
      v-loading="loading && !items.length"
      class="collector-connection-list__items"
      :aria-busy="loading"
      @scroll.passive="handleItemsScroll"
    >
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
                :class="`is-${connectionTone(item.id)}`"
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
                  :aria-label="ui(`${item.name}更多操作`, `More actions for ${item.name}`)"
                  @click.stop
                >
                  <IconTablerDots />
                </button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item
                      v-if="connectionStatus(item.id) === 'connected'"
                      command="disconnect"
                    >
                      <IconTablerPlugConnectedX />{{ ui('断开连接', 'Disconnect') }}
                    </el-dropdown-item>
                    <el-dropdown-item v-else command="connect" :disabled="connectionBusy(item.id)">
                      <IconTablerPlugConnected />{{
                        connectionStatus(item.id) === 'error' ? ui('重新连接', 'Reconnect') : ui('连接', 'Connect')
                      }}</el-dropdown-item
                    >
                    <el-dropdown-item divided command="delete">
                      <IconTablerTrash />{{ ui('删除连接', 'Delete Connection') }}
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item
                  v-if="connectionStatus(item.id) === 'connected'"
                  command="disconnect"
                >
                  <IconTablerPlugConnectedX />{{ ui('断开连接', 'Disconnect') }}
                </el-dropdown-item>
                <el-dropdown-item v-else command="connect" :disabled="connectionBusy(item.id)">
                  <IconTablerPlugConnected />{{
                    connectionStatus(item.id) === 'error' ? ui('重新连接', 'Reconnect') : ui('连接', 'Connect')
                  }}</el-dropdown-item
                >
                <el-dropdown-item divided command="delete">
                  <IconTablerTrash />{{ ui('删除连接', 'Delete Connection') }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </section>
      <el-empty v-if="!loading && !items.length" :description="ui('暂无工业连接', 'No Industrial Connections')" :image-size="64" />
      <div v-if="loading && items.length" class="collector-connection-list__load-state">
        <IconTablerLoader2 />{{ ui('正在加载', 'Loading') }}
      </div>
      <button
        v-else-if="hasMore"
        type="button"
        class="collector-connection-list__load-more"
        @click="loadMore"
      >
        {{ ui('加载更多', 'Load More') }}
      </button>
    </div>

    <div class="collector-connection-list__footer">
      <span>{{ ui(`已加载 ${items.length} / ${total} 个连接`, `Loaded ${items.length} / ${total} connections`) }}</span>
      <span v-if="hasMore">{{ ui('向下滚动继续加载', 'Scroll down to load more') }}</span>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import {
  deleteCollectorConnection,
  getCollectorConnectionDeleteImpact,
  listCollectorConnections,
} from '@/api/collector.api'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { CollectorConnection } from '@/api/schemas/collector.schema'
import {
  collectorDebugConnectionTone,
  formatCollectorConnectionSummary,
  groupCollectorConnections,
  type CollectorDebugConnectionState,
  type CollectorDebugConnectionStatus,
} from './collector-workbench-model'
import CollectorDriverIcon from './CollectorDriverIcon.vue'
import IconTablerChevronDown from '~icons/tabler/chevron-down'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerDots from '~icons/tabler/dots'
import IconTablerLayoutSidebarLeftCollapse from '~icons/tabler/layout-sidebar-left-collapse'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerPlugConnected from '~icons/tabler/plug-connected'
import IconTablerPlugConnectedX from '~icons/tabler/plug-connected-x'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerSearch from '~icons/tabler/search'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = defineProps<{
  projectId: string
  selectedId?: string
  sessionStates: Record<string, CollectorDebugConnectionState>
}>()
const emit = defineEmits<{
  select: [id: string]
  collapse: []
  loaded: [items: CollectorConnection[]]
  deleted: [id: string]
  connect: [connection: CollectorConnection]
  disconnect: [connection: CollectorConnection]
}>()
const items = ref<CollectorConnection[]>([])
const itemsContainer = ref<HTMLElement>()
const loading = ref(false)
const search = ref('')
const appliedSearch = ref('')
const page = ref(0)
const pageSize = 20
const total = ref(0)
const totalPages = ref(0)
const hasMore = computed(() => page.value < totalPages.value)
const collapsedGroups = ref(new Set<string>())
const groupedItems = computed(() => groupCollectorConnections(items.value))
let requestVersion = 0

function toggleGroup(key: string) {
  const next = new Set(collapsedGroups.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  collapsedGroups.value = next
}

function connectionStatus(connectionId: string): CollectorDebugConnectionStatus {
  return props.sessionStates[connectionId]?.status || 'disconnected'
}
function connectionTone(connectionId: string) {
  return collectorDebugConnectionTone(connectionStatus(connectionId))
}
function connectionBusy(connectionId: string) {
  return ['connecting', 'disconnecting'].includes(connectionStatus(connectionId))
}
async function handleItemCommand(command: string, item: CollectorConnection) {
  if (command === 'connect') {
    emit('connect', item)
    return
  }
  if (command === 'disconnect') {
    emit('disconnect', item)
    return
  }
  if (command === 'delete') await removeConnection(item)
}

async function removeConnection(item: CollectorConnection) {
  if (connectionStatus(item.id) !== 'disconnected') {
    ElMessage.warning(ui('请先断开调试长连接，再删除工业连接', 'Disconnect the debug session before deleting this connection'))
    return
  }
  const impact = await getCollectorConnectionDeleteImpact(props.projectId, item.id)
  if (!impact.canDelete) {
    const blockers = impact.blockingUsages
      .map(
        (usage) =>
          ui(
            `${usage.label || usage.type} ${usage.count} 项${usage.examples.length ? `（${usage.examples.map((example) => example.name).join('、')}）` : ''}`,
            `${usage.label || usage.type}: ${usage.count}${usage.examples.length ? ` (${usage.examples.map((example) => example.name).join(', ')})` : ''}`,
          ),
      )
      .join('\n')
    await ElMessageBox.alert(
      ui(`当前工业连接仍被以下配置引用，请先解除引用：\n${blockers}`, `This connection is still referenced. Remove these references first:\n${blockers}`),
      ui('无法删除工业连接', 'Cannot Delete Connection'),
      {
        type: 'warning',
        confirmButtonText: ui('知道了', 'OK'),
      },
    )
    return
  }
  const owned = impact.ownedResources.reduce((sum, resource) => sum + resource.count, 0)
  await ElMessageBox.confirm(
    ui(
      `确认删除连接“${item.name}”及 ${owned} 个来源内配置？${impact.generatedDatapoints.count} 个已生成数据点会保留并标记为无效。`,
      `Delete “${item.name}” and ${owned} owned configuration item${owned === 1 ? '' : 's'}? ${impact.generatedDatapoints.count} generated data point${impact.generatedDatapoints.count === 1 ? '' : 's'} will be retained and marked invalid.`,
    ),
    ui('删除工业连接', 'Delete Industrial Connection'),
    {
      type: 'warning',
      confirmButtonText: ui('删除', 'Delete'),
      cancelButtonText: ui('取消', 'Cancel'),
      confirmButtonClass: 'el-button--danger',
    },
  )
  await deleteCollectorConnection(props.projectId, item.id)
  emit('deleted', item.id)
  ElMessage.success(ui('工业连接已删除', 'Industrial connection deleted'))
  await reload()
}

function mergeConnections(current: CollectorConnection[], incoming: CollectorConnection[]) {
  const merged = new Map(current.map((item) => [item.id, item]))
  for (const item of incoming) merged.set(item.id, item)
  return Array.from(merged.values())
}

async function loadPage(nextPage: number, version: number) {
  loading.value = true
  try {
    const result = await listCollectorConnections(props.projectId, {
      page: nextPage,
      pageSize,
      search: appliedSearch.value,
    })
    if (version !== requestVersion) return
    items.value = nextPage === 1 ? result.list : mergeConnections(items.value, result.list)
    page.value = result.pagination.page
    total.value = result.pagination.total
    totalPages.value = result.pagination.totalPages
    emit('loaded', items.value)
  } finally {
    if (version === requestVersion) loading.value = false
  }
}

function applySearch() {
  appliedSearch.value = search.value.trim()
  void reload()
}

async function reload() {
  // 每次刷新都作废旧请求，避免快速搜索时较慢的响应覆盖当前树。
  const version = ++requestVersion
  page.value = 0
  total.value = 0
  totalPages.value = 0
  items.value = []
  if (itemsContainer.value) itemsContainer.value.scrollTop = 0
  await loadPage(1, version)
}

async function loadMore() {
  if (loading.value || !hasMore.value) return
  await loadPage(page.value + 1, requestVersion)
}

function handleItemsScroll(event: Event) {
  const target = event.currentTarget as HTMLElement
  if (target.scrollHeight - target.scrollTop - target.clientHeight <= 96) void loadMore()
}

async function fillVisibleArea() {
  // 宽屏下首批数据可能无法撑出滚动条，继续补载直到可滚动或全部加载完成。
  await nextTick()
  const container = itemsContainer.value
  if (!container || loading.value || !hasMore.value) return
  if (container.scrollHeight <= container.clientHeight + 1) await loadMore()
}

defineExpose({
  reload,
  getConnection: (connectionId: string) => items.value.find((item) => item.id === connectionId),
})
watch(
  () => props.projectId,
  () => reload(),
)
watch([items, hasMore, loading], () => void fillVisibleArea(), { flush: 'post' })
onMounted(() => void reload())
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
.collector-connection-list__load-state,
.collector-connection-list__load-more {
  display: flex;
  width: calc(100% - 17px);
  min-height: 32px;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin: 6px 0 0 17px;
  border: none;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-muted);
  font-size: 11px;
}
.collector-connection-list__load-more {
  cursor: pointer;
}
.collector-connection-list__load-more:hover,
.collector-connection-list__load-more:focus-visible {
  background: var(--dc-surface-subtle);
  color: var(--dc-primary);
}
.collector-connection-list__load-state svg {
  width: 14px;
  height: 14px;
  animation: collector-connection-spin 0.9s linear infinite;
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
  animation: collector-connection-pulse 1.1s ease-in-out infinite;
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
@keyframes collector-connection-pulse {
  50% {
    opacity: 0.35;
  }
}
@keyframes collector-connection-spin {
  to {
    transform: rotate(360deg);
  }
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
</style>
