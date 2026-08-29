<template>
  <div class="access-source-workspace">
    <section class="access-source-workspace__panel">
      <!-- 头部：单行 toolbar 撑满 -->
      <header class="access-source-workspace__head">
        <FilterToolbar>
          <!-- 搜索框 -->
          <el-input
            v-model="searchInputValue"
            class="access-source-workspace__search"
            size="small"
            clearable
            :placeholder="t('accessSources.search')"
            :prefix-icon="SearchIcon"
            @input="handleSearchInput"
            @clear="handleSearchClear"
          />

          <!-- 类型筛选 pill -->
          <el-popover
            trigger="click"
            placement="bottom-start"
            :width="160"
            popper-class="access-source-workspace__popover"
          >
            <template #reference>
              <PillButton :active="filterType !== 'all'">
                {{ typeLabel }}
              </PillButton>
            </template>
            <div class="access-source-workspace__pop-list">
              <button
                v-for="item in typeOptions"
                :key="item.value"
                type="button"
                class="access-source-workspace__pop-item"
                :class="{ 'is-active': filterType === item.value }"
                @click="selectType(item.value)"
              >
                {{ item.label }}
              </button>
            </div>
          </el-popover>

          <div class="access-source-workspace__view-switcher">
            <el-tooltip :content="t('accessSources.cardView')" placement="top">
              <button
                type="button"
                :class="{ 'is-active': viewMode === 'card' }"
                :aria-label="t('accessSources.switchCard')"
                @click="viewMode = 'card'"
              >
                <IconTablerLayoutGrid />
              </button>
            </el-tooltip>
            <el-tooltip :content="t('accessSources.listView')" placement="top">
              <button
                type="button"
                :class="{ 'is-active': viewMode === 'list' }"
                :aria-label="t('accessSources.switchList')"
                @click="viewMode = 'list'"
              >
                <IconTablerListDetails />
              </button>
            </el-tooltip>
          </div>

          <!-- 刷新 -->
          <button
            type="button"
            class="access-source-workspace__icon-btn"
            :title="t('accessSources.refresh')"
            @click="$emit('refresh')"
          >
            <IconTablerRefresh class="access-source-workspace__icon-btn-icon" />
          </button>

          <!-- 概览数字：只展示接入源总数，连接态进入工作台后由用户手动触发 -->
          <div class="access-source-workspace__overview">
            <span class="access-source-workspace__overview-text">{{ t('accessSources.total', { count: overviewTotal }) }}</span>
          </div>

          <template #actions>
            <button type="button" class="access-source-workspace__primary" @click="$emit('create')">
              <IconTablerPlus class="access-source-workspace__action-icon" />
              <span>{{ t('accessSources.create') }}</span>
            </button>
          </template>
        </FilterToolbar>
      </header>

      <section class="access-source-workspace__content-panel">
        <TableScroll>
          <AccessSourceList
            v-if="viewMode === 'card'"
            v-loading="loading"
            :connections="filteredConnections"
            :selected-connection-id="activeConnectionId"
            :testing-connection-id="testingConnectionId"
            :draggable="canReorderConnections"
            @open="handleOpen"
            @edit="handleEdit"
            @test="handleTestConnection"
            @delete-connection="handleDeleteConnection"
            @reorder="handleReorderConnections"
            @create="$emit('create')"
          />

          <div v-else v-loading="loading" class="access-source-workspace__table-wrap">
            <div class="access-source-workspace__table-shell">
              <el-table
                :data="filteredConnections"
                row-key="id"
                class="access-source-workspace__table"
              >
                <el-table-column :label="t('accessSources.name')" min-width="220">
                  <template #default="{ row }">
                    <div class="access-source-workspace__source-identity">
                      <span
                        class="access-source-workspace__source-icon"
                        :class="`is-${resolveCategory(row)}`"
                      >
                        <component :is="resolveConnectionIcon(row)" />
                      </span>
                      <button
                        type="button"
                        class="access-source-workspace__name-button"
                        @click="handleOpen(row)"
                      >
                        {{ row.name || t('accessSources.unnamed') }}
                      </button>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column :label="t('accessSources.type')" width="150">
                  <template #default="{ row }">
                    <span
                      class="access-source-workspace__type-badge"
                      :class="`is-${resolveCategory(row)}`"
                    >
                      {{ resolveConnectionTypeLabel(row) }}
                    </span>
                  </template>
                </el-table-column>
                <el-table-column :label="t('accessSources.endpoint')" min-width="260" show-overflow-tooltip>
                  <template #default="{ row }">
                    <span
                      class="access-source-workspace__endpoint"
                      :class="{ 'is-pending': isConnectionEndpointPending(row) }"
                    >
                      {{ resolveConnectionEndpoint(row) }}
                    </span>
                  </template>
                </el-table-column>
                <el-table-column :label="t('accessSources.datapoints')" width="104" align="center">
                  <template #default="{ row }">
                    <span
                      class="access-source-workspace__point-count"
                      :class="{ 'is-empty': resolveDatapointCount(row) === 0 }"
                    >
                      {{ resolveDatapointCount(row) }}
                    </span>
                  </template>
                </el-table-column>
                <el-table-column :label="t('accessSources.status')" width="138">
                  <template #default="{ row }">
                    <span
                      class="access-source-workspace__state"
                      :class="`is-${resolveConnectionState(row).tone}`"
                      :title="resolveConnectionState(row).detail"
                    >
                      {{ resolveConnectionState(row).label }}
                    </span>
                  </template>
                </el-table-column>
                <el-table-column :label="t('accessSources.actions')" width="184" align="right" fixed="right">
                  <template #default="{ row }">
                    <div class="access-source-workspace__table-actions">
                      <button type="button" class="is-workbench" @click="handleOpen(row)">
                        <span>{{ t('accessSources.workbench') }}</span>
                        <IconTablerArrowRight />
                      </button>
                      <button
                        v-if="row.testCapability.status === 'supported'"
                        type="button"
                        class="is-icon"
                        :disabled="testingConnectionId === row.id"
                        :title="t('accessSources.testSaved')"
                        :aria-label="t('accessSources.testSource')"
                        @click="handleTestConnection(row)"
                      >
                        <IconTablerLoader2
                          v-if="testingConnectionId === row.id"
                          class="is-spinning"
                        />
                        <IconTablerPlugConnected v-else />
                      </button>
                      <span
                        v-else
                        class="access-source-workspace__action-placeholder"
                        aria-hidden="true"
                      ></span>
                      <button
                        type="button"
                        class="is-icon"
                        :title="t('accessSources.edit')"
                        :aria-label="t('accessSources.editSource')"
                        @click="handleEdit(row)"
                      >
                        <IconTablerSettings />
                      </button>
                      <button
                        type="button"
                        class="is-icon is-danger"
                        :title="t('accessSources.delete')"
                        :aria-label="t('accessSources.deleteSource')"
                        @click="handleDeleteConnection(row)"
                      >
                        <IconTablerTrash />
                      </button>
                    </div>
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </div>

          <template #pagination>
            <DataCenterPagination
              v-if="pagination.total > 0"
              :page="pagination.page"
              :page-size="pagination.pageSize"
              :total="pagination.total"
              :total-pages="pagination.totalPages"
              @change="handlePaginationChange"
            />
          </template>
        </TableScroll>
      </section>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerArrowRight from '~icons/tabler/arrow-right'
import IconTablerLayoutGrid from '~icons/tabler/layout-grid'
import IconTablerListDetails from '~icons/tabler/list-details'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerSettings from '~icons/tabler/settings'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerPlugConnected from '~icons/tabler/plug-connected'
import { deleteConnection, getConnectionDeleteImpact, updateConnectionOrder } from '@/api/data.api'
import { testSavedConnection } from '@/api/connection.api'
import { useConfirm } from '@/composables/useConfirm'
import { getApiErrorMessage } from '@/utils/request'
import AccessSourceList from './AccessSourceList.vue'
import PillButton from '@/components/shared/PillButton.vue'
import DataCenterPagination from '@/components/shared/DataCenterPagination.vue'
import FilterToolbar from '@/components/shared/FilterToolbar.vue'
import TableScroll from '@/components/shared/TableScroll.vue'
import { isBuiltinStoreType } from './workbench/builtin-store'
import { resolveAccessSourceVisual } from './access-source-visual'
import type { Connection as AccessSourceConnection } from '@/api/schemas/connection.schema'
import { t } from '@/i18n/runtime'

/* Search 图标赋值给变量，传给 el-input prefix-icon */
const SearchIcon = Search

const props = defineProps<{
  connections: AccessSourceConnection[]
  selectedConnectionId?: string | null
  projectId?: string | number | null
  loading?: boolean
  pagination: {
    page: number
    pageSize: number
    total: number
    totalPages: number
  }
}>()

const emit = defineEmits<{
  (event: 'create'): void
  (event: 'refresh'): void
  (event: 'edit', connection: AccessSourceConnection): void
  (
    event: 'query-change',
    value: { page: number; pageSize: number; search: string; typeGroup: string },
  ): void
}>()

/* ── toolbar 概览计算属性 ── */
const overviewTotal = computed(() => props.pagination.total)

const route = useRoute()
const router = useRouter()
const { confirm } = useConfirm()
const testingConnectionId = ref('')

/* ── URL 同步：从 query 读取初始筛选值 ── */
const filterQ = ref(String(route.query.q || ''))
const filterType = ref(String(route.query.type || 'all'))
const viewMode = ref<'card' | 'list'>('card')
const connectionOrder = ref<string[]>([])
const hasLocalConnectionOrder = ref(false)

/* 搜索框双向绑定值（防抖前的输入缓存） */
const searchInputValue = ref(filterQ.value)

/* 300ms 防抖 timer */
let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null

const handleSearchInput = (val: string) => {
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
  searchDebounceTimer = setTimeout(() => {
    filterQ.value = val.trim()
    syncQuery()
    emitQueryChange(1, props.pagination.pageSize)
  }, 300)
}

const handleSearchClear = () => {
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
  filterQ.value = ''
  searchInputValue.value = ''
  syncQuery()
  emitQueryChange(1, props.pagination.pageSize)
}

/* 当前激活的连接 id：优先 URL params，其次 props，最后 null */
const activeConnectionId = computed<string | null>(() => {
  const paramId = String(route.params.objectId || '')
  if (paramId) return paramId
  return props.selectedConnectionId || null
})

/* 把当前筛选写入 URL query（仅写入非空字段）*/
function syncQuery() {
  const query: Record<string, string> = {}
  if (filterQ.value) query.q = filterQ.value
  if (filterType.value && filterType.value !== 'all') query.type = filterType.value
  void router.replace({ query })
}

/* ── 类型筛选选项 ── */
const typeOptions = computed(() => [
  { label: t('accessSources.all'), value: 'all' },
  { label: t('accessSources.builtin'), value: 'builtin' },
  { label: t('accessSources.database'), value: 'database' },
  { label: t('accessSources.stream'), value: 'stream' },
])

const typeLabel = computed(() => {
  const found = typeOptions.value.find((o) => o.value === filterType.value)
  return found ? (filterType.value === 'all' ? t('accessSources.type') : found.label) : t('accessSources.type')
})

const selectType = (val: string) => {
  filterType.value = val
  syncQuery()
  emitQueryChange(1, props.pagination.pageSize)
}

const canReorderConnections = computed(
  () =>
    filterType.value === 'all' &&
    !filterQ.value &&
    props.pagination.total <= props.pagination.pageSize,
)

const emitQueryChange = (page: number, pageSize: number) => {
  emit('query-change', {
    page,
    pageSize,
    search: filterQ.value,
    typeGroup: filterType.value,
  })
}

const handlePaginationChange = (value: { page: number; pageSize: number }) => {
  emitQueryChange(value.page, value.pageSize)
}

/* ── 分类判断（与旧 resolveCategory 逻辑一致）── */
const resolveCategory = (connection: AccessSourceConnection) => {
  const category = resolveAccessSourceVisual(connection.type).category
  return category === 'other' ? 'all' : category
}

const resolveConnectionTypeLabel = (connection: AccessSourceConnection) => {
  const labels: Record<string, string> = {
    'builtin.relation': t('accessSources.builtinTypes.relation'),
    'builtin.timeseries': t('accessSources.builtinTypes.timeseries'),
    'builtin.realtime': t('accessSources.builtinTypes.realtime'),
    'builtin.message': t('accessSources.builtinTypes.message'),
    relational: String(connection.relationalConfig?.dbType || t('accessSources.database')),
    mqtt: 'MQTT',
    kafka: 'Kafka',
    websocket: 'WebSocket',
    http: 'HTTP',
    redis: 'Redis',
    tdengine: 'TDengine',
  }
  return labels[connection.type || ''] || connection.type || t('accessSources.unknownType')
}

const resolveConnectionIcon = (connection: AccessSourceConnection) =>
  resolveAccessSourceVisual(connection.type).icon

const resolveConnectionEndpoint = (connection: AccessSourceConnection) => {
  if (isBuiltinStoreType(connection.type || '')) return t('accessSources.builtinStore')
  if (connection.type === 'relational') {
    const config = connection.relationalConfig
    const address = [config?.host, config?.port].filter(Boolean).join(':')
    return [address, config?.database].filter(Boolean).join(' / ') || t('accessSources.notConfiguredDbAddress')
  }
  if (connection.type === 'mqtt') {
    const config = connection.mqttConfig
    return (
      [config?.brokerUrl || config?.host, config?.port].filter(Boolean).join(':') || t('accessSources.notConfiguredBroker')
    )
  }
  const config = connection.config || {}

  return String(
    config['endpoint'] ||
      config['url'] ||
      config['address'] ||
      config['brokers'] ||
      config['host'] ||
      config['serverProgId'] ||
      t('accessSources.pendingConfig'),
  )
}

const resolveDatapointCount = (connection: AccessSourceConnection) => connection.variableCount

const resolveConnectionState = (connection: AccessSourceConnection) => {
  if (!connection.enabled) return { label: t('accessSources.disabled'), tone: 'muted', detail: t('accessSources.disabledDetail') }
  if (connection.configurationState === 'incomplete') {
    return { label: t('accessSources.incomplete'), tone: 'warning', detail: t('accessSources.incompleteDetail') }
  }
  if (connection.testCapability.status === 'unsupported') {
    return {
      label: t('accessSources.normal'),
      tone: 'success',
      detail: connection.testCapability.reason || t('accessSources.workspaceTestHint'),
    }
  }
  if (connection.lastTest.status === 'succeeded') {
    return {
      label: t('accessSources.normal'),
      tone: 'success',
      detail: connection.lastTest.message || t('accessSources.testSucceeded'),
    }
  }
  if (connection.lastTest.status === 'failed') {
    return {
      label: t('accessSources.abnormal'),
      tone: 'danger',
      detail: connection.lastTest.message || t('accessSources.testFailed'),
    }
  }
  return { label: t('accessSources.notTested'), tone: 'muted', detail: t('accessSources.notTestedDetail') }
}

const isConnectionEndpointPending = (connection: AccessSourceConnection) => {
  const endpoint = resolveConnectionEndpoint(connection)
  return endpoint.startsWith('等待') || endpoint.startsWith('未配置')
}

watch(
  () => props.connections.map((connection) => connection.id),
  (ids) => {
    if (!hasLocalConnectionOrder.value) return
    const knownIds = new Set(ids)
    const preserved = connectionOrder.value.filter((id) => knownIds.has(id))
    const appended = ids.filter((id) => !preserved.includes(id))
    connectionOrder.value = [...preserved, ...appended]
  },
  { immediate: true },
)

/* ── 前端过滤（基于 props.connections）── */
const filteredConnections = computed(() => {
  const list = [...props.connections]
  if (hasLocalConnectionOrder.value && connectionOrder.value.length > 0) {
    const orderMap = new Map(connectionOrder.value.map((id, index) => [id, index]))
    list.sort((left, right) => {
      const leftOrder = orderMap.get(left.id) ?? Number.MAX_SAFE_INTEGER
      const rightOrder = orderMap.get(right.id) ?? Number.MAX_SAFE_INTEGER
      return leftOrder - rightOrder
    })
  }

  return list
})

/* ── 事件处理 ── */
const handleOpen = (connection: AccessSourceConnection) => {
  /* 直接 router.push 到 v2 workbench 路由，保留筛选 query */
  const isDebug = route.path.startsWith('/debug/')
  const base = isDebug ? '/debug' : ''
  void router.push({
    path: `${base}/access-source/${connection.id}/workbench`,
    query: route.query,
  })
}

const handleEdit = (connection: AccessSourceConnection) => {
  emit('edit', connection)
}

const handleTestConnection = async (connection: AccessSourceConnection) => {
  if (!props.projectId || testingConnectionId.value) return
  testingConnectionId.value = connection.id
  try {
    const result = await testSavedConnection(String(props.projectId), connection.id)
    if (result.connected) ElMessage.success(result.message || t('accessSources.testSuccess'))
    else ElMessage.warning(result.message || t('accessSources.testRejected'))
    emit('refresh')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('accessSources.testError')))
  } finally {
    testingConnectionId.value = ''
  }
}

const handleReorderConnections = async (connectionIds: string[]) => {
  if (!props.projectId || !canReorderConnections.value) return
  const previousOrder = [...connectionOrder.value]
  const previousHasLocalOrder = hasLocalConnectionOrder.value
  connectionOrder.value = connectionIds
  hasLocalConnectionOrder.value = true
  try {
    await updateConnectionOrder(String(props.projectId), connectionIds)
    ElMessage.success(t('accessSources.orderSaved'))
  } catch (err) {
    connectionOrder.value = previousOrder
    hasLocalConnectionOrder.value = previousHasLocalOrder
    ElMessage.error(getApiErrorMessage(err, t('accessSources.orderSaveFailed')))
  }
}

/* 删除接入源：二次确认后调用 API，成功后通知父刷新 */
const handleDeleteConnection = async (connection: AccessSourceConnection) => {
  if (!props.projectId) return
  let impact
  try {
    impact = await getConnectionDeleteImpact(String(props.projectId), connection.id)
  } catch (err) {
    ElMessage.error(getApiErrorMessage(err, t('accessSources.impactFailed')))
    return
  }
  if (!impact.canDelete) {
    const blockers = impact.blockingUsages
      .map(
        (item) =>
          t('accessSources.impactItem', { label: item.label || item.type, count: item.count, examples: item.examples.length ? ` (${item.examples.map((example) => example.name).join(', ')})` : '' }),
      )
      .join('\n')
    await ElMessageBox.alert(
      t('accessSources.cannotDelete', { blockers }),
      t('accessSources.cannotDeleteTitle'),
      {
        type: 'warning',
        confirmButtonText: t('accessSources.understood'),
      },
    )
    return
  }
  const owned = impact.ownedResources.reduce((sum, item) => sum + item.count, 0)
  const ok = await confirm(
    t('accessSources.deleteConfirm', { name: connection.name || connection.id, owned, points: impact.generatedDatapoints.count }),
    { title: t('accessSources.deleteTitle'), confirmText: t('accessSources.delete'), type: 'error' },
  )
  if (!ok) return
  try {
    await deleteConnection(String(props.projectId), connection.id)
    ElMessage.success(t('accessSources.deleted'))
    emit('refresh')
  } catch (err) {
    ElMessage.error(getApiErrorMessage(err, t('accessSources.deleteFailed')))
  }
}
</script>

<style scoped>
.access-source-workspace {
  height: 100%;
  min-height: 0;
  display: flex;
}

.access-source-workspace__panel {
  min-width: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 14px;
  overflow: visible;
}

/* 头部：单行 toolbar，固定 56px */
.access-source-workspace__head {
  flex: 0 0 56px;
  display: flex;
  align-items: center;
  padding: 0 20px;
  border: 1px solid var(--dc-border);
  border-radius: 14px;
  background: var(--dc-surface-raised);
  box-shadow:
    0 8px 24px rgba(15, 23, 42, 0.06),
    0 1px 2px rgba(15, 23, 42, 0.04);
}

.access-source-workspace__content-panel {
  min-width: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: 14px;
  background: var(--dc-surface-raised);
  box-shadow:
    0 8px 24px rgba(15, 23, 42, 0.055),
    0 1px 2px rgba(15, 23, 42, 0.04);
}

/* toolbar：搜索 + pill + 刷新 + 新增 */
.access-source-workspace__toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  width: 100%;
}

.access-source-workspace__search {
  width: 240px;
}

.access-source-workspace__view-switcher {
  height: 32px;
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}

.access-source-workspace__view-switcher button {
  width: 26px;
  height: 26px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: 5px;
  background: transparent;
  color: var(--dc-text-muted);
}

.access-source-workspace__view-switcher button.is-active {
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.1);
}

.access-source-workspace__view-switcher svg {
  width: 15px;
  height: 15px;
}

/* 刷新图标按钮 */
.access-source-workspace__icon-btn {
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  transition:
    border-color 0.18s ease,
    color 0.18s ease;
}

.access-source-workspace__icon-btn:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  color: var(--dc-primary);
}

.access-source-workspace__icon-btn-icon {
  width: 16px;
  height: 16px;
}

.access-source-workspace__primary {
  min-height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  padding: 0 12px;
  border: 1px solid var(--dc-primary);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary);
  color: var(--dc-on-primary);
  font-size: 13px;
  font-weight: 700;
  transition:
    background-color 0.18s ease,
    border-color 0.18s ease,
    transform 0.18s ease;
}

.access-source-workspace__primary:hover {
  transform: translateY(-1px);
  background: var(--dc-primary-hover);
  border-color: var(--dc-primary-hover);
}

.access-source-workspace__action-icon {
  width: 16px;
  height: 16px;
}

.access-source-workspace__table-wrap {
  min-height: 0;
  flex: 1;
  overflow: hidden;
  padding: 12px 24px 0;
  background: var(--dc-surface-raised);
}

.access-source-workspace__table-shell {
  width: 100%;
  min-width: 0;
  height: 100%;
  overflow: hidden;
  border: 0;
  border-radius: 0;
  background: var(--dc-surface-raised);
  box-shadow: none;
}

.access-source-workspace__table {
  width: 100%;
  height: 100%;
}

.access-source-workspace__table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.access-source-workspace__table :deep(.el-table__header th) {
  height: 50px;
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0;
}

.access-source-workspace__table :deep(.el-table__cell) {
  padding: 8px 0;
  color: var(--dc-text-secondary);
  font-size: 13px;
}

.access-source-workspace__table :deep(.el-table__row) {
  height: 60px;
}

.access-source-workspace__table :deep(.el-table__row td) {
  border-bottom-color: color-mix(in oklch, var(--dc-border) 66%, transparent);
  transition: background-color 0.16s ease;
}

.access-source-workspace__table :deep(.el-table__row:hover > td.el-table__cell) {
  background: color-mix(in oklch, var(--dc-primary-soft) 28%, var(--dc-surface-raised));
}

.access-source-workspace__table :deep(.el-table-fixed-column--right) {
  background: var(--dc-surface-raised);
}

.access-source-workspace__table :deep(th.el-table-fixed-column--right) {
  background: var(--dc-surface-raised);
}

.access-source-workspace__table :deep(.el-table__row:hover > td.el-table-fixed-column--right) {
  background: color-mix(in oklch, var(--dc-primary-soft) 28%, var(--dc-surface-raised));
}

.access-source-workspace__source-identity {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.access-source-workspace__source-icon {
  width: 30px;
  height: 30px;
  flex: 0 0 30px;
  display: grid;
  place-items: center;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 14%, var(--dc-border));
  border-radius: 7px;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.access-source-workspace__source-icon.is-database {
  border-color: color-mix(in srgb, var(--dc-success) 28%, var(--dc-border));
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.access-source-workspace__source-icon.is-stream {
  border-color: color-mix(in srgb, var(--dc-warning) 28%, var(--dc-border));
  background: var(--dc-warning-soft);
  color: var(--dc-warning);
}

.access-source-workspace__source-icon svg {
  width: 16px;
  height: 16px;
}

.access-source-workspace__name-button {
  max-width: 100%;
  overflow: hidden;
  border: none;
  background: transparent;
  color: var(--dc-primary);
  padding: 4px 0;
  font-size: 13px;
  font-weight: 700;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-workspace__name-button:hover {
  color: var(--dc-primary-hover);
}

.access-source-workspace__type-badge {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 8px;
  border: 1px solid var(--dc-border);
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.access-source-workspace__type-badge.is-builtin {
  border-color: color-mix(in oklch, var(--dc-primary) 18%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.access-source-workspace__type-badge.is-database {
  border-color: color-mix(in srgb, var(--dc-success) 28%, var(--dc-border));
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.access-source-workspace__type-badge.is-stream {
  border-color: color-mix(in srgb, var(--dc-warning) 28%, var(--dc-border));
  background: var(--dc-warning-soft);
  color: var(--dc-warning);
}

.access-source-workspace__endpoint {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  color: var(--dc-text-secondary);
  font-family: 'Cascadia Code', 'SFMono-Regular', Consolas, monospace;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-workspace__endpoint.is-pending {
  padding: 3px 8px;
  border-radius: 5px;
  background: var(--dc-warning-soft);
  color: var(--dc-warning);
  font-family: inherit;
}

.access-source-workspace__point-count {
  min-width: 32px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 8px;
  border-radius: 999px;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-size: 12px;
  font-weight: 750;
  font-variant-numeric: tabular-nums;
}

.access-source-workspace__point-count.is-empty {
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
}

.access-source-workspace__state {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 0 8px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-size: 12px;
  white-space: nowrap;
}

.access-source-workspace__state.is-success {
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.access-source-workspace__state.is-warning {
  background: var(--dc-warning-soft);
  color: var(--dc-warning);
}

.access-source-workspace__state.is-danger {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.access-source-workspace__table-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
}

.access-source-workspace__table-actions button {
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 0 7px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
  transition:
    border-color 0.16s ease,
    background-color 0.16s ease,
    color 0.16s ease;
}

.access-source-workspace__table-actions button:hover {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.access-source-workspace__table-actions button.is-workbench {
  color: var(--dc-primary);
}

.access-source-workspace__table-actions button.is-workbench:hover {
  background: var(--dc-primary-soft);
  color: var(--dc-primary-hover);
}

.access-source-workspace__table-actions button.is-icon {
  width: 28px;
  min-width: 28px;
  padding: 0;
}

.access-source-workspace__action-placeholder {
  width: 28px;
  height: 28px;
  flex: none;
}

.access-source-workspace__table-actions button.is-danger:hover {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.access-source-workspace__table-actions svg {
  width: 14px;
  height: 14px;
}

/* toolbar 右侧概览：占据剩余空间并右对齐文字 */
.access-source-workspace__overview {
  flex: 1;
  min-width: 0;
  display: flex;
  justify-content: flex-end;
  padding: 0 8px;
}

.access-source-workspace__overview-text {
  color: var(--dc-text-muted);
  font-size: 12px;
  white-space: nowrap;
}

/* popover 内选项列表 */
.access-source-workspace__pop-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 4px 0;
}

.access-source-workspace__pop-item {
  width: 100%;
  padding: 7px 12px;
  border: none;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 13px;
  text-align: left;
  cursor: pointer;
  transition:
    background-color 0.14s ease,
    color 0.14s ease;
}

.access-source-workspace__pop-item:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.access-source-workspace__pop-item.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-weight: 700;
}

.is-spinning {
  animation: access-source-spin 0.8s linear infinite;
}

@keyframes access-source-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 760px) {
  .access-source-workspace__head {
    height: auto;
    padding: 10px 14px;
  }

  .access-source-workspace__toolbar {
    width: 100%;
  }

  .access-source-workspace__search {
    width: 100%;
  }

  .access-source-workspace__primary {
    flex: 1;
  }
}
</style>
