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
        <el-button @click="emit('import')">批量导入</el-button>
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
        @selection-change="selected = $event"
      >
        <el-table-column type="selection" width="44" />
        <el-table-column prop="name" label="变量名称" min-width="180">
          <template #default="scope">
            <button
              type="button"
              class="collector-point-table__identity"
              @click="openDetail(scope.row)"
            >
              <strong>{{ scope.row.name }}</strong>
              <small>{{ scope.row.code }}</small>
            </button>
          </template>
        </el-table-column>
        <el-table-column prop="addressText" label="变量地址" min-width="190" show-overflow-tooltip>
          <template #default="scope"
            ><code>{{ scope.row.addressText }}</code></template
          >
        </el-table-column>
        <el-table-column prop="dataType" label="数据类型" width="112" />
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
        <el-table-column label="操作" width="146" fixed="right">
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

    <div class="collector-point-table__footer">
      <span>第 {{ page }} 页 · 每页 {{ pageSize }} 条</span>
      <el-pagination
        layout="prev, pager, next"
        :current-page="page"
        :page-size="pageSize"
        :total="total"
        @current-change="load"
      />
    </div>

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
import { onMounted, ref, watch } from 'vue'
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
import type { CollectorPointCreateDefaults } from './collector-workbench-model'
import CollectorPointDrawer, { type CollectorPointGroupNode } from './CollectorPointDrawer.vue'

const props = defineProps<{
  projectId: string
  connectionId: string
  driverId: string
  groupId: string | null
  showElementCount: boolean
}>()
const emit = defineEmits<{
  selection: [pointIds: string[]]
  import: []
  saved: []
}>()
const tableRef = ref<{ clearSelection: () => void }>()
const items = ref<CollectorPoint[]>([])
const selected = ref<CollectorPoint[]>([])
const groups = ref<CollectorPointGroupNode[]>([])
const loading = ref(false)
const batchLoading = ref(false)
const savingIds = ref(new Set<string>())
const search = ref('')
const page = ref(1)
const pageSize = 50
const total = ref(0)
const drawerVisible = ref(false)
const drawerMode = ref<'create' | 'edit' | 'detail'>('create')
const currentPoint = ref<CollectorPoint | null>(null)
const createGroupId = ref<string | null>(null)
const createDefaults = ref<CollectorPointCreateDefaults | null>(null)
const createSourceLocked = ref(false)
const moveVisible = ref(false)
const moveGroupId = ref<string | null>(null)

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
      pageSize,
      search: search.value,
      groupId: props.groupId || undefined,
    })
    items.value = result.list
    total.value = result.pagination.total
    clearSelection()
  } finally {
    loading.value = false
  }
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
function clearSelection() {
  selected.value = []
  tableRef.value?.clearSelection()
}
watch(selected, (value) =>
  emit(
    'selection',
    value.map((item) => item.id),
  ),
)
watch(
  () => props.groupId,
  () => load(1),
)
watch(
  () => props.connectionId,
  async () => {
    await Promise.all([reloadGroups(), load(1)])
  },
)
onMounted(() => Promise.all([reloadGroups(), load(1)]))
defineExpose({ reload: load, reloadGroups, openCreate })
</script>

<style scoped>
.collector-point-table {
  position: relative;
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  padding-left: 16px;
}
.collector-point-table__toolbar,
.collector-point-table__footer {
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
.collector-point-table__filters span,
.collector-point-table__footer {
  color: var(--dc-text-muted);
  font-size: 11px;
}
.collector-point-table__search {
  width: min(290px, 34vw);
}
.collector-point-table__content {
  min-height: 0;
  flex: 1;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
}
.collector-point-table__identity {
  display: flex;
  width: 100%;
  flex-direction: column;
  gap: 2px;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
  text-align: left;
}
.collector-point-table__identity strong {
  color: var(--dc-text);
  font-size: 12px;
}
.collector-point-table__identity:hover strong {
  color: var(--dc-primary);
}
.collector-point-table__identity small,
.collector-point-table code {
  color: var(--dc-text-muted);
  font-size: 10px;
}
.collector-point-table__footer {
  min-height: 48px;
}
@media (max-width: 900px) {
  .collector-point-table__toolbar {
    align-items: stretch;
    flex-direction: column;
  }
  .collector-point-table__search {
    width: 100%;
  }
}
</style>
