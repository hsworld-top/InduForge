<template>
  <div class="collector-point-table">
    <div class="collector-point-table__toolbar">
      <div class="collector-point-table__filters">
        <el-input
          v-model="search"
          clearable
          placeholder="搜索变量名称、编码或地址"
          class="collector-point-table__search"
          @keyup.enter="load(1)"
          @clear="load(1)"
        />
        <span>共 {{ total }} 个变量</span>
      </div>
      <div class="collector-point-table__actions">
        <el-tooltip
          v-if="supportsPointRead"
          :content="readCurrentPageDisabledReason"
          :disabled="canReadCurrentPage && items.length > 0"
          placement="top"
        >
          <span>
            <el-button
              :disabled="!canReadCurrentPage || items.length === 0"
              :loading="readCurrentPageLoading"
              @click="readCurrentPage"
            >
              <IconTablerDatabaseSearch />
              {{ readCurrentPageLoading ? `正在获取 ${items.length} 项` : '获取当前页数据' }}
            </el-button>
          </span>
        </el-tooltip>
        <el-button @click="emit('import')">批量导入</el-button>
        <el-button @click="exportVisible = true">导出变量</el-button>
        <el-button type="primary" @click="openCreate(groupId)">新建变量</el-button>
      </div>
    </div>

    <div class="collector-point-table__content">
      <el-table
        ref="tableRef"
        v-loading="loading"
        :data="items"
        height="100%"
        row-key="id"
        scrollbar-always-on
        class="collector-point-table__table"
        @selection-change="onSelectionChange"
      >
        <el-table-column type="selection" width="44" />
        <el-table-column prop="name" label="变量名称" min-width="160">
          <template #default="scope">
            <button
              type="button"
              class="collector-point-table__identity"
              @click="openDetail(scope.row)"
            >
              <strong :title="scope.row.name">{{ scope.row.name }}</strong>
            </button>
          </template>
        </el-table-column>
        <el-table-column prop="addressText" label="变量地址" min-width="180" show-overflow-tooltip>
          <template #default="scope"
            ><code>{{ scope.row.addressText }}</code></template
          >
        </el-table-column>
        <el-table-column prop="dataType" label="数据类型" width="100" />
        <el-table-column label="最近值" min-width="150">
          <template #default="scope">
            <div class="collector-point-table__debug-value">
              <el-tooltip
                :content="debugValue(scope.row)"
                :disabled="!scope.row.latestDebugSnapshot"
                placement="top"
              >
                <span>{{ debugValue(scope.row) }}</span>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="质量" width="92" align="center">
          <template #default="scope">
            <el-tag
              v-if="tableQuality(scope.row.latestDebugSnapshot?.quality)"
              :type="
                tableQuality(scope.row.latestDebugSnapshot?.quality) === 'Good'
                  ? 'success'
                  : 'danger'
              "
              size="small"
              effect="light"
            >
              {{ tableQuality(scope.row.latestDebugSnapshot?.quality) }}
            </el-tag>
            <span v-else class="collector-point-table__empty-value">—</span>
          </template>
        </el-table-column>
        <el-table-column label="数据时间" width="160">
          <template #default="scope">
            <span class="collector-point-table__debug-time">{{
              collectorDebugTime(scope.row.latestDebugSnapshot?.sourceTimestamp || null)
            }}</span>
          </template>
        </el-table-column>
        <el-table-column v-if="showElementCount" prop="elementCount" label="元素" width="74" />
        <el-table-column label="采集周期" width="104">
          <template #default="scope">{{ formatAcquisitionInterval(scope.row) }}</template>
        </el-table-column>
        <el-table-column label="启用" width="82" align="center">
          <template #default="scope">
            <el-switch
              :model-value="scope.row.enabled"
              :loading="savingIds.has(scope.row.id)"
              @change="toggleEnabled(scope.row, Boolean($event))"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="144" fixed="right">
          <template #default="scope">
            <div class="collector-point-table__row-actions">
              <el-button link type="primary" @click="openDetail(scope.row)">查看</el-button>
              <el-button link type="primary" @click="openEdit(scope.row)">编辑</el-button>
              <el-button link type="danger" @click="removeOne(scope.row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <DataCenterPagination
      v-if="total > 0"
      class="collector-point-table__pagination"
      :page="page"
      :page-size="pageSize"
      :total="total"
      :total-pages="totalPages"
      @change="handlePaginationChange"
    />

    <BulkActionBar :selected-count="selected.length" @clear="clearSelection">
      <el-button size="small" @click="batchSetEnabled(true)">批量启用</el-button>
      <el-button size="small" @click="batchSetEnabled(false)">批量停用</el-button>
      <el-button size="small" @click="moveVisible = true">移动分组</el-button>
      <el-button size="small" type="danger" plain @click="removeSelected">批量删除</el-button>
    </BulkActionBar>

    <el-dialog v-model="moveVisible" title="批量移动分组" width="460px">
      <el-form label-position="top">
        <el-form-item label="目标分组">
          <el-tree-select
            v-model="moveGroupId"
            :data="groups"
            node-key="id"
            :props="{ label: 'name', children: 'children' }"
            check-strictly
            clearable
            default-expand-all
            placeholder="未分组"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="moveVisible = false">取消</el-button>
        <el-button type="primary" :loading="batchLoading" @click="moveSelected">确认移动</el-button>
      </template>
    </el-dialog>

    <CollectorPointExportDialog
      v-model="exportVisible"
      :project-id="projectId"
      :connection-id="connectionId"
      :connection-name="connectionName"
      :group-id="groupId"
      :group-name="currentGroupName"
      :search="search"
      :page="page"
      :page-size="pageSize"
      :total="total"
      :current-page-count="items.length"
      :selected-ids="selected.map((item) => item.id)"
    />

    <CollectorPointDrawer
      v-model="drawerVisible"
      :initial-mode="drawerMode"
      :project-id="projectId"
      :connection-id="connectionId"
      :driver-id="driverId"
      :point="currentPoint"
      :groups="groups"
      :default-group-id="createGroupId"
      :create-defaults="createDefaults"
      :source-locked="createSourceLocked"
      @saved="onDrawerSaved"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  deleteCollectorPointsBatch,
  listCollectorPointGroups,
  listCollectorPoints,
  moveCollectorPointsBatch,
  updateCollectorPointsBatch,
} from '@/api/collector.api'
import type { CollectorPoint } from '@/api/schemas/collector.schema'
import BulkActionBar from '@/components/shared/BulkActionBar.vue'
import DataCenterPagination from '@/components/shared/DataCenterPagination.vue'
import type { CollectorPointCreateDefaults } from './collector-workbench-model'
import CollectorPointDrawer, { type CollectorPointGroupNode } from './CollectorPointDrawer.vue'
import CollectorPointExportDialog from './CollectorPointExportDialog.vue'
import IconTablerDatabaseSearch from '~icons/tabler/database-search'
import {
  collectorDebugTime,
  formatCollectorDebugValue,
  hasCollectorDebugSuccess,
} from './collector-debug-snapshot'

const props = defineProps<{
  projectId: string
  connectionId: string
  connectionName: string
  driverId: string
  groupId: string | null
  showElementCount: boolean
  supportsPointRead: boolean
  canReadCurrentPage: boolean
  readCurrentPageLoading: boolean
  readCurrentPageDisabledReason: string
}>()
const emit = defineEmits<{
  import: []
  saved: []
  readCurrentPage: [points: Array<Pick<CollectorPoint, 'id' | 'name'>>]
}>()
const tableRef = ref<{
  clearSelection: () => void
  toggleRowSelection: (row: CollectorPoint, selected: boolean) => void
}>()
const items = ref<CollectorPoint[]>([])
const selectedById = ref(new Map<string, CollectorPoint>())
const selected = computed(() => [...selectedById.value.values()])
const groups = ref<CollectorPointGroupNode[]>([])
const loading = ref(false)
const batchLoading = ref(false)
const savingIds = ref(new Set<string>())
const search = ref('')
const page = ref(1)
const pageSize = ref(50)
const total = ref(0)
const totalPages = computed(() => (total.value > 0 ? Math.ceil(total.value / pageSize.value) : 0))
const drawerVisible = ref(false)
const drawerMode = ref<'create' | 'edit' | 'detail'>('create')
const currentPoint = ref<CollectorPoint | null>(null)
const createGroupId = ref<string | null>(null)
const createDefaults = ref<CollectorPointCreateDefaults | null>(null)
const createSourceLocked = ref(false)
const moveVisible = ref(false)
const moveGroupId = ref<string | null>(null)
const exportVisible = ref(false)
let restoringSelection = false
const currentGroupName = computed(() => {
  if (!props.groupId) return '全部变量'
  return flattenGroups(groups.value).find((group) => group.id === props.groupId)?.name || '当前分组'
})

function readCurrentPage() {
  if (!props.canReadCurrentPage || props.readCurrentPageLoading || items.value.length === 0) return
  emit(
    'readCurrentPage',
    items.value.map(({ id, name }) => ({ id, name })),
  )
}

function debugValue(point: CollectorPoint) {
  const snapshot = point.latestDebugSnapshot
  return snapshot && hasCollectorDebugSuccess(snapshot)
    ? formatCollectorDebugValue(snapshot.value, snapshot.valueText)
    : '—'
}

function tableQuality(quality: string | null | undefined): 'Good' | 'Bad' | null {
  const normalized = quality?.trim().toLowerCase()
  if (!normalized) return null
  return normalized.startsWith('good') ? 'Good' : 'Bad'
}

function formatAcquisitionInterval(point: CollectorPoint) {
  const intervalMs = point.acquisition.intervalMs
  return typeof intervalMs === 'number' ? `${intervalMs} ms` : '-'
}
async function load(nextPage = page.value) {
  page.value = nextPage
  loading.value = true
  try {
    const result = await listCollectorPoints(props.projectId, props.connectionId, {
      page: page.value,
      pageSize: pageSize.value,
      search: search.value,
      groupId: props.groupId || undefined,
    })
    items.value = result.list
    total.value = result.pagination.total
    if (currentPoint.value) {
      currentPoint.value = items.value.find((item) => item.id === currentPoint.value?.id) || null
    }
    await nextTick()
    restoreCurrentPageSelection()
  } finally {
    loading.value = false
  }
}
function handlePaginationChange(value: { page: number; pageSize: number }) {
  pageSize.value = value.pageSize
  void load(value.page)
}

async function loadGroupChildren(parentId: string | null): Promise<CollectorPointGroupNode[]> {
  const children = await listCollectorPointGroups(props.projectId, props.connectionId, parentId)
  return Promise.all(
    children.map(async (group) => ({
      ...group,
      children: await loadGroupChildren(group.id),
    })),
  )
}
async function reloadGroups() {
  groups.value = await loadGroupChildren(null)
}
async function onDrawerSaved() {
  await load()
  emit('saved')
}

function openCreate(
  groupId: string | null = props.groupId,
  defaults: CollectorPointCreateDefaults | null = null,
  sourceLocked = false,
) {
  drawerMode.value = 'create'
  currentPoint.value = null
  createGroupId.value = groupId
  createDefaults.value = defaults
  createSourceLocked.value = sourceLocked
  drawerVisible.value = true
}
function openEdit(point: CollectorPoint) {
  drawerMode.value = 'edit'
  currentPoint.value = point
  createDefaults.value = null
  createSourceLocked.value = false
  drawerVisible.value = true
}
function openDetail(point: CollectorPoint) {
  drawerMode.value = 'detail'
  currentPoint.value = point
  createDefaults.value = null
  createSourceLocked.value = false
  drawerVisible.value = true
}
function pointPayload(point: CollectorPoint, overrides: Partial<CollectorPoint> = {}) {
  const next = { ...point, ...overrides }
  return {
    id: next.id,
    groupId: next.groupId,
    name: next.name,
    description: next.description,
    address: next.address,
    dataType: next.dataType,
    elementCount: next.elementCount,
    readOptions: next.readOptions,
    acquisition: next.acquisition,
    enabled: next.enabled,
    sortOrder: next.sortOrder,
    metadata: next.metadata,
  }
}
async function toggleEnabled(point: CollectorPoint, enabled: boolean) {
  savingIds.value = new Set(savingIds.value).add(point.id)
  try {
    const [updated] = await updateCollectorPointsBatch(props.projectId, props.connectionId, [
      pointPayload(point, { enabled }),
    ])
    const index = items.value.findIndex((item) => item.id === point.id)
    if (updated && index >= 0) items.value[index] = updated
    ElMessage.success(enabled ? '变量已启用' : '变量已停用')
  } finally {
    const next = new Set(savingIds.value)
    next.delete(point.id)
    savingIds.value = next
  }
}
async function batchSetEnabled(enabled: boolean) {
  if (!selected.value.length) return
  batchLoading.value = true
  try {
    await updateCollectorPointsBatch(
      props.projectId,
      props.connectionId,
      selected.value.map((point) => pointPayload(point, { enabled })),
    )
    ElMessage.success(`已${enabled ? '启用' : '停用'} ${selected.value.length} 个变量`)
    await load()
  } finally {
    batchLoading.value = false
  }
}
async function removeOne(point: CollectorPoint) {
  await ElMessageBox.confirm(`确认删除变量“${point.name}”？`, '删除变量', { type: 'warning' })
  await deleteCollectorPointsBatch(props.projectId, props.connectionId, [point.id])
  ElMessage.success('变量已删除')
  await loadAfterDelete(1)
}
async function removeSelected() {
  if (!selected.value.length) return
  await ElMessageBox.confirm(`确认删除选中的 ${selected.value.length} 个变量？`, '批量删除变量', {
    type: 'warning',
  })
  await deleteCollectorPointsBatch(
    props.projectId,
    props.connectionId,
    selected.value.map((item) => item.id),
  )
  ElMessage.success('变量已批量删除')
  await loadAfterDelete(selected.value.length)
}
async function loadAfterDelete(deletedCount: number) {
  const nextPage =
    items.value.length <= deletedCount && page.value > 1 ? page.value - 1 : page.value
  await load(nextPage)
}
async function moveSelected() {
  if (!selected.value.length) return
  batchLoading.value = true
  try {
    await moveCollectorPointsBatch(
      props.projectId,
      props.connectionId,
      selected.value.map((point) => point.id),
      moveGroupId.value,
    )
    moveVisible.value = false
    ElMessage.success('变量已移动到目标分组')
    await load()
  } finally {
    batchLoading.value = false
  }
}
function onSelectionChange(rows: CollectorPoint[]) {
  if (restoringSelection) return
  const next = new Map(selectedById.value)
  for (const item of items.value) next.delete(item.id)
  for (const row of rows) next.set(row.id, row)
  selectedById.value = next
}
function restoreCurrentPageSelection() {
  restoringSelection = true
  tableRef.value?.clearSelection()
  for (const item of items.value) {
    if (selectedById.value.has(item.id)) tableRef.value?.toggleRowSelection(item, true)
  }
  nextTick(() => {
    restoringSelection = false
  })
}
function clearSelection() {
  selectedById.value = new Map()
  restoringSelection = true
  tableRef.value?.clearSelection()
  nextTick(() => {
    restoringSelection = false
  })
}
function flattenGroups(nodes: CollectorPointGroupNode[]): CollectorPointGroupNode[] {
  return nodes.flatMap((node) => [node, ...flattenGroups(node.children || [])])
}
watch(
  () => props.groupId,
  () => {
    clearSelection()
    load(1)
  },
)
watch(
  () => props.connectionId,
  async () => {
    clearSelection()
    await Promise.all([reloadGroups(), load(1)])
  },
)
onMounted(() => Promise.all([reloadGroups(), load(1)]))
defineExpose({ reload: load, reloadGroups, openCreate })
</script>

<style scoped>
.collector-point-table {
  container: collector-point-table / inline-size;
  position: relative;
  width: 0;
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding-left: 16px;
}
.collector-point-table__toolbar {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}
.collector-point-table__toolbar {
  min-height: 48px;
  padding-bottom: 10px;
}
.collector-point-table__filters,
.collector-point-table__actions,
.collector-point-table__row-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.collector-point-table__filters {
  min-width: 0;
  flex: 1 1 auto;
}
.collector-point-table__actions {
  flex: 0 0 auto;
}
.collector-point-table__filters span {
  color: var(--dc-text-muted);
  font-size: 11px;
}
.collector-point-table__search {
  width: min(290px, 34vw);
}
.collector-point-table__content {
  width: 100%;
  min-height: 0;
  flex: 1;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
}
.collector-point-table__table {
  width: 100%;
  height: 100%;
}
.collector-point-table__table :deep(.el-table__cell .cell) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.collector-point-table__debug-value {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
}
.collector-point-table__debug-value > span {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  color: var(--dc-text);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.collector-point-table__debug-value :deep(.el-tag) {
  flex: 0 0 auto;
}
.collector-point-table__debug-time,
.collector-point-table__empty-value {
  color: var(--dc-text-muted);
  font-size: 11px;
}
.collector-point-table__identity {
  display: block;
  width: 100%;
  overflow: hidden;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
  text-align: left;
}
.collector-point-table__identity strong {
  display: block;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.collector-point-table__identity:hover strong {
  color: var(--dc-primary);
}
.collector-point-table code {
  color: var(--dc-text-muted);
  font-size: 10px;
}
.collector-point-table__row-actions {
  flex-wrap: nowrap;
  padding-right: 4px;
  white-space: nowrap;
}
.collector-point-table__row-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}
.collector-point-table__pagination {
  margin-top: 8px;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
}
@container collector-point-table (max-width: 760px) {
  .collector-point-table__toolbar {
    align-items: stretch;
    flex-direction: column;
  }
  .collector-point-table__search {
    width: auto;
    min-width: 0;
    flex: 1;
  }
  .collector-point-table__actions {
    width: 100%;
    flex-wrap: wrap;
    justify-content: flex-end;
  }
}
</style>
