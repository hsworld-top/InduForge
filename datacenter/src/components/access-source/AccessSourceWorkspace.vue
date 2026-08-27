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
            placeholder="搜索名称"
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
            <el-tooltip content="卡片视图" placement="top">
              <button
                type="button"
                :class="{ 'is-active': viewMode === 'card' }"
                aria-label="切换到卡片视图"
                @click="viewMode = 'card'"
              >
                <IconTablerLayoutGrid />
              </button>
            </el-tooltip>
            <el-tooltip content="列表视图" placement="top">
              <button
                type="button"
                :class="{ 'is-active': viewMode === 'list' }"
                aria-label="切换到列表视图"
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
            title="刷新"
            @click="$emit('refresh')"
          >
            <IconTablerRefresh class="access-source-workspace__icon-btn-icon" />
          </button>

          <!-- 概览数字：只展示接入源总数，连接态进入工作台后由用户手动触发 -->
          <div class="access-source-workspace__overview">
            <span class="access-source-workspace__overview-text">共 {{ overviewTotal }}</span>
          </div>

          <template #actions>
            <button type="button" class="access-source-workspace__primary" @click="$emit('create')">
              <IconTablerPlus class="access-source-workspace__action-icon" />
              <span>新增连接</span>
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
                <el-table-column label="名称" min-width="220">
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
                        {{ row.name || '未命名连接' }}
                      </button>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column label="类型" width="150">
                  <template #default="{ row }">
                    <span
                      class="access-source-workspace__type-badge"
                      :class="`is-${resolveCategory(row)}`"
                    >
                      {{ resolveConnectionTypeLabel(row) }}
                    </span>
                  </template>
                </el-table-column>
                <el-table-column label="连接地址" min-width="260" show-overflow-tooltip>
                  <template #default="{ row }">
                    <span
                      class="access-source-workspace__endpoint"
                      :class="{ 'is-pending': isConnectionEndpointPending(row) }"
                    >
                      {{ resolveConnectionEndpoint(row) }}
                    </span>
                  </template>
                </el-table-column>
                <el-table-column label="数据点" width="92" align="center">
                  <template #default="{ row }">
                    <span
                      class="access-source-workspace__point-count"
                      :class="{ 'is-empty': resolveDatapointCount(row) === 0 }"
                    >
                      {{ resolveDatapointCount(row) }}
                    </span>
                  </template>
                </el-table-column>
                <el-table-column label="状态" width="138">
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
                <el-table-column label="操作" width="184" align="right" fixed="right">
                  <template #default="{ row }">
                    <div class="access-source-workspace__table-actions">
                      <button type="button" class="is-workbench" @click="handleOpen(row)">
                        <span>工作台</span>
                        <IconTablerArrowRight />
                      </button>
                      <button
                        v-if="row.testCapability.status === 'supported'"
                        type="button"
                        class="is-icon"
                        :disabled="testingConnectionId === row.id"
                        title="测试已保存连接"
                        aria-label="测试接入源"
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
                        title="编辑"
                        aria-label="编辑接入源"
                        @click="handleEdit(row)"
                      >
                        <IconTablerSettings />
                      </button>
                      <button
                        type="button"
                        class="is-icon is-danger"
                        title="删除"
                        aria-label="删除接入源"
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
const typeOptions = [
  { label: '全部', value: 'all' },
  { label: '内置运行库', value: 'builtin' },
  { label: '数据库', value: 'database' },
  { label: '消息/流', value: 'stream' },
]

const typeLabel = computed(() => {
  const found = typeOptions.find((o) => o.value === filterType.value)
  return found ? (filterType.value === 'all' ? '类型' : found.label) : '类型'
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
    'builtin.relation': 'IF关系库',
    'builtin.timeseries': 'IF时序库',
    'builtin.realtime': 'IF实时库',
    'builtin.message': 'IF消息库',
    relational: String(connection.relationalConfig?.dbType || '数据库'),
    mqtt: 'MQTT',
    kafka: 'Kafka',
    websocket: 'WebSocket',
    http: 'HTTP',
    redis: 'Redis',
    tdengine: 'TDengine',
  }
  return labels[connection.type || ''] || connection.type || '未知类型'
}

const resolveConnectionIcon = (connection: AccessSourceConnection) =>
  resolveAccessSourceVisual(connection.type).icon

const resolveConnectionEndpoint = (connection: AccessSourceConnection) => {
  if (isBuiltinStoreType(connection.type || '')) return '工程内置运行库'
  if (connection.type === 'relational') {
    const config = connection.relationalConfig
    const address = [config?.host, config?.port].filter(Boolean).join(':')
    return [address, config?.database].filter(Boolean).join(' / ') || '未配置数据库地址'
  }
  if (connection.type === 'mqtt') {
    const config = connection.mqttConfig
    return (
      [config?.brokerUrl || config?.host, config?.port].filter(Boolean).join(':') || '未配置 Broker'
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
      '等待接入配置',
  )
}

const resolveDatapointCount = (connection: AccessSourceConnection) => connection.variableCount

const resolveConnectionState = (connection: AccessSourceConnection) => {
  if (!connection.enabled) return { label: '已停用', tone: 'muted', detail: '配置已停用' }
  if (connection.configurationState === 'incomplete') {
    return { label: '配置不完整', tone: 'warning', detail: '请补齐接入源必填配置' }
  }
  if (connection.testCapability.status === 'unsupported') {
    return {
      label: '正常',
      tone: 'success',
      detail: connection.testCapability.reason || '请在所属工作台测试具体请求或会话',
    }
  }
  if (connection.lastTest.status === 'succeeded') {
    return {
      label: '正常',
      tone: 'success',
      detail: connection.lastTest.message || '最近测试成功',
    }
  }
  if (connection.lastTest.status === 'failed') {
    return {
      label: '异常',
      tone: 'danger',
      detail: connection.lastTest.message || '最近测试失败',
    }
  }
  return { label: '未测试', tone: 'muted', detail: '尚未测试已保存配置' }
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
    if (result.connected) ElMessage.success(result.message || '连接测试成功')
    else ElMessage.warning(result.message || '连接测试未通过')
    emit('refresh')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '连接测试失败'))
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
    ElMessage.success('接入源顺序已保存')
  } catch (err) {
    connectionOrder.value = previousOrder
    hasLocalConnectionOrder.value = previousHasLocalOrder
    ElMessage.error(getApiErrorMessage(err, '保存接入源顺序失败'))
  }
}

/* 删除接入源：二次确认后调用 API，成功后通知父刷新 */
const handleDeleteConnection = async (connection: AccessSourceConnection) => {
  if (!props.projectId) return
  let impact
  try {
    impact = await getConnectionDeleteImpact(String(props.projectId), connection.id)
  } catch (err) {
    ElMessage.error(getApiErrorMessage(err, '读取删除影响失败'))
    return
  }
  if (!impact.canDelete) {
    const blockers = impact.blockingUsages
      .map(
        (item) =>
          `${item.label || item.type} ${item.count} 项${item.examples.length ? `（${item.examples.map((example) => example.name).join('、')}）` : ''}`,
      )
      .join('\n')
    await ElMessageBox.alert(
      `当前接入源仍被以下配置引用，请先解除引用：\n${blockers}`,
      '无法删除接入源',
      {
        type: 'warning',
        confirmButtonText: '知道了',
      },
    )
    return
  }
  const owned = impact.ownedResources.reduce((sum, item) => sum + item.count, 0)
  const ok = await confirm(
    `将删除接入源「${connection.name || connection.id}」及 ${owned} 个来源内配置；${impact.generatedDatapoints.count} 个已生成数据点会保留并标记为无效。`,
    { title: '删除接入源', confirmText: '删除', type: 'error' },
  )
  if (!ok) return
  try {
    await deleteConnection(String(props.projectId), connection.id)
    ElMessage.success('接入源已删除')
    emit('refresh')
  } catch (err) {
    ElMessage.error(getApiErrorMessage(err, '删除失败'))
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
  color: var(--dc-surface-raised);
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
  border-color: #d1fae5;
  background: #ecfdf5;
  color: #047857;
}

.access-source-workspace__source-icon.is-stream {
  border-color: #ffedd5;
  background: #fff7ed;
  color: #c2410c;
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
  border-color: #d1fae5;
  background: #ecfdf5;
  color: #047857;
}

.access-source-workspace__type-badge.is-stream {
  border-color: #ffedd5;
  background: #fff7ed;
  color: #c2410c;
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
  background: #fff7ed;
  color: #9a5b13;
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
  background: #ecfdf5;
  color: #047857;
}

.access-source-workspace__state.is-warning {
  background: #fff7ed;
  color: #c2410c;
}

.access-source-workspace__state.is-danger {
  background: #fef2f2;
  color: #b91c1c;
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
  background: #fef2f2;
  color: #dc2626;
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
