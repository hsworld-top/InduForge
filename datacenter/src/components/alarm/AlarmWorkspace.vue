<template>
  <div class="alarm-workspace">
    <section class="alarm-workspace__panel">
      <header class="alarm-workspace__head">
        <div class="alarm-workspace__tabs-row">
          <el-segmented v-model="activeTab" :options="tabOptions" size="small" />
        </div>
        <div v-if="activeTab === 'items'" class="alarm-workspace__toolbar-row">
          <div class="alarm-workspace__toolbar">
            <el-input
              v-model="search"
              class="alarm-workspace__search"
              size="small"
              clearable
              placeholder="搜索数据点或报警名称"
              :prefix-icon="Search"
              @keyup.enter="reload"
            />
            <PillButton :active="directoryVisible" @click="toggleDirectory"
              ><template #icon><IconTablerFolders /></template>目录</PillButton
            >
            <el-select
              v-model="modeFilter"
              size="small"
              class="alarm-workspace__filter"
              @change="reload"
              ><el-option label="全部模式" value="" /><el-option
                label="点位报警"
                value="point" /><el-option label="组合报警" value="derived"
            /></el-select>
            <el-select
              v-model="typeFilter"
              size="small"
              class="alarm-workspace__filter is-wide"
              @change="reload"
              ><el-option label="全部报警类型" value="" /><el-option
                v-for="kind in conditionKinds"
                :key="kind"
                :label="alarmConditionLabels[kind]"
                :value="kind"
            /></el-select>
            <el-select
              v-model="severityFilter"
              size="small"
              class="alarm-workspace__filter"
              @change="reload"
              ><el-option label="全部等级" value="" /><el-option
                v-for="severity in severityValues"
                :key="severity"
                :label="alarmSeverityLabels[severity]"
                :value="severity"
            /></el-select>
            <el-select
              v-model="enabledFilter"
              size="small"
              class="alarm-workspace__filter"
              @change="reload"
              ><el-option label="全部状态" value="" /><el-option
                label="已启用"
                value="true" /><el-option label="已停用" value="false"
            /></el-select>
            <button type="button" class="alarm-workspace__icon" title="刷新" @click="loadItems">
              <IconTablerRefresh />
            </button>
          </div>
          <div class="alarm-workspace__create-actions">
            <el-dropdown trigger="click" @command="handleExcelCommand"
              ><button type="button" class="alarm-workspace__secondary">
                <IconTablerFileSpreadsheet />Excel</button
              ><template #dropdown
                ><el-dropdown-menu
                  ><el-dropdown-item command="template">下载导入模板</el-dropdown-item
                  ><el-dropdown-item command="import">导入报警项</el-dropdown-item
                  ><el-dropdown-item command="export" :disabled="pagination.total === 0"
                    >导出当前范围</el-dropdown-item
                  ></el-dropdown-menu
                ></template
              ></el-dropdown
            >
            <button type="button" class="alarm-workspace__secondary" @click="openCreate('derived')">
              <IconTablerFunction />组合报警
            </button>
            <button type="button" class="alarm-workspace__primary" @click="openCreate('point')">
              <IconTablerPlus />新建报警
            </button>
          </div>
        </div>
      </header>

      <section class="alarm-workspace__content-panel">
        <template v-if="activeTab === 'items'">
          <AlarmDirectoryPanel
            v-if="directoryVisible"
            :project-id="projectId"
            :groups="groups"
            :selected-id="groupFilter"
            @select="selectGroup"
            @changed="loadGroups"
          />
          <div class="alarm-workspace__table-panel">
            <BulkActionBar
              :selected-count="batchAffectedCount"
              item-label="条"
              @clear="clearBatchSelection"
            >
              <button
                type="button"
                class="alarm-workspace__batch-action"
                :class="{ 'is-active': !batchAllResults }"
                @click="selectCurrentPage"
              >
                当前页
              </button>
              <button
                type="button"
                class="alarm-workspace__batch-action"
                :class="{ 'is-active': batchAllResults }"
                :disabled="pagination.total === 0"
                @click="selectAllResults"
              >
                全部结果
              </button>
              <button type="button" class="alarm-workspace__batch-action" @click="openBatchEdit">
                批量修改
              </button>
              <button
                type="button"
                class="alarm-workspace__batch-action"
                @click="batchSetEnabled(true)"
              >
                启用
              </button>
              <button
                type="button"
                class="alarm-workspace__batch-action"
                @click="batchSetEnabled(false)"
              >
                停用
              </button>
              <button
                type="button"
                class="alarm-workspace__batch-action is-danger"
                @click="batchRemove"
              >
                删除
              </button>
            </BulkActionBar>
            <div v-loading="loading" class="alarm-workspace__table-scroll">
              <el-table
                ref="tableRef"
                :data="items"
                row-key="id"
                height="100%"
                scrollbar-always-on
                class="alarm-workspace__table"
                @selection-change="handleSelectionChange"
                @row-dblclick="openEdit"
              >
                <el-table-column type="selection" width="46" fixed="left" />
                <el-table-column label="数据点" min-width="190"
                  ><template #default="{ row }"
                    ><button
                      type="button"
                      class="alarm-workspace__point-link"
                      @click="filterByDatapoint(row)"
                    >
                      {{ alarmItemPointSummary(row) }}
                    </button></template
                  ></el-table-column
                >
                <el-table-column label="报警类型" width="128"
                  ><template #default="{ row }"
                    ><span class="alarm-workspace__ellipsis">{{
                      row.mode === 'derived' ? '组合报警' : alarmConditionLabels[row.alarmType]
                    }}</span></template
                  ></el-table-column
                >
                <el-table-column label="条件摘要" min-width="190"
                  ><template #default="{ row }"
                    ><span
                      class="alarm-workspace__ellipsis"
                      :title="alarmItemConditionSummary(row)"
                      >{{ alarmItemConditionSummary(row) }}</span
                    ></template
                  ></el-table-column
                >
                <el-table-column label="显示名称" min-width="160"
                  ><template #default="{ row }"
                    ><strong class="alarm-workspace__name" :title="row.displayName">{{
                      row.displayName
                    }}</strong></template
                  ></el-table-column
                >
                <el-table-column label="最高等级" width="96"
                  ><template #default="{ row }"
                    ><StatusBadge
                      :tone="severityTone(alarmItemHighestSeverity(row))"
                      :text="alarmSeverityLabels[alarmItemHighestSeverity(row)]" /></template
                ></el-table-column>
                <el-table-column label="状态" width="82"
                  ><template #default="{ row }"
                    ><el-switch
                      :model-value="row.isEnabled"
                      :loading="togglingId === row.id"
                      @change="toggleItem(row, Boolean($event))" /></template
                ></el-table-column>
                <el-table-column label="目录" min-width="130"
                  ><template #default="{ row }"
                    ><span class="alarm-workspace__ellipsis">{{
                      row.groupName || '根目录'
                    }}</span></template
                  ></el-table-column
                >
                <el-table-column label="操作" width="104" align="right" fixed="right"
                  ><template #default="{ row }"
                    ><div class="alarm-workspace__row-actions">
                      <button type="button" title="编辑" @click="openEdit(row)">
                        <IconTablerEdit /></button
                      ><button
                        type="button"
                        class="is-danger"
                        title="删除"
                        @click="removeItem(row)"
                      >
                        <IconTablerTrash />
                      </button></div></template
                ></el-table-column>
                <template #empty><el-empty :image-size="54" description="暂无报警项" /></template>
              </el-table>
            </div>
            <DataCenterPagination
              :page="pagination.page"
              :page-size="pagination.pageSize"
              :total="pagination.total"
              :total-pages="pagination.totalPages"
              @change="changePage"
            />
          </div>
        </template>
        <AlarmNotificationSettings
          v-else-if="activeTab === 'notifications'"
          :project-id="projectId"
          @changed="channels = $event"
        />
        <AlarmHistorySettings v-else :project-id="projectId" />
      </section>
    </section>

    <AlarmPolicyDrawer
      v-model="drawerVisible"
      :project-id="projectId"
      :mode="creatingMode"
      :item="editingItem"
      :groups="groups"
      :channels="channels"
      :initial-points="initialPoints"
      :saving="saving"
      @save="saveItem"
      @open-conflict="openById"
    />
    <AlarmBatchEditDialog
      v-model="batchEditVisible"
      :selection="currentSelection"
      :affected-count="batchAffectedCount"
      :items="batchAllResults ? [] : selectedItems"
      :groups="groups"
      :saving="saving"
      @save="applyBatchEdit"
    />
    <AlarmExcelImportDialog v-model="importVisible" :project-id="projectId" @imported="loadItems" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, type TableInstance } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import IconTablerEdit from '~icons/tabler/edit'
import IconTablerFileSpreadsheet from '~icons/tabler/file-spreadsheet'
import IconTablerFolders from '~icons/tabler/folders'
import IconTablerFunction from '~icons/tabler/function'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerTrash from '~icons/tabler/trash'
import AlarmBatchEditDialog from './AlarmBatchEditDialog.vue'
import AlarmDirectoryPanel from './AlarmDirectoryPanel.vue'
import AlarmExcelImportDialog from './AlarmExcelImportDialog.vue'
import AlarmHistorySettings from './AlarmHistorySettings.vue'
import AlarmNotificationSettings from './AlarmNotificationSettings.vue'
import AlarmPolicyDrawer from './AlarmPolicyDrawer.vue'
import BulkActionBar from '@/components/shared/BulkActionBar.vue'
import DataCenterPagination from '@/components/shared/DataCenterPagination.vue'
import PillButton from '@/components/shared/PillButton.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import {
  batchCreateAlarmItems,
  batchDeleteAlarmItems,
  batchUpdateAlarmItems,
  createAlarmItem,
  deleteAlarmItem,
  downloadAlarmImportTemplate,
  exportAlarmItems,
  getAlarmItem,
  listAlarmChannels,
  listAlarmGroupTree,
  listAlarmItems,
  setAlarmItemEnabled,
  updateAlarmItem,
} from '@/api/alarm.api'
import type {
  AlarmBatchUpdate,
  AlarmConditionKind,
  AlarmGroup,
  AlarmItem,
  AlarmItemMode,
  AlarmItemSave,
  AlarmItemSelection,
  AlarmNotificationChannel,
  AlarmSeverity,
} from '@/api/schemas/alarm.schema'
import {
  alarmConditionLabels,
  alarmItemConditionSummary,
  alarmItemHighestSeverity,
  alarmItemPointSummary,
  alarmSeverityLabels,
  ensureBuiltinAlarmNotificationChannel,
  type AlarmPointSelection,
} from '@/models/alarm-policy'
import { getApiErrorMessage } from '@/utils/request'
import { useConfirm } from '@/composables/useConfirm'

const props = withDefaults(
  defineProps<{
    projectId: string
    selectedPolicyId?: string
    initialDatapoints?: AlarmPointSelection[]
  }>(),
  { selectedPolicyId: '', initialDatapoints: () => [] },
)
const emit = defineEmits<{ 'close-selected-policy': [] }>()
const { confirm } = useConfirm()
const activeTab = ref<'items' | 'notifications' | 'history'>('items')
const tabOptions = [
  { label: '报警配置', value: 'items' },
  { label: '通知设置', value: 'notifications' },
  { label: '报警历史', value: 'history' },
]
const items = ref<AlarmItem[]>([])
const groups = ref<AlarmGroup[]>([])
const channels = ref<AlarmNotificationChannel[]>([])
const loading = ref(false)
const saving = ref(false)
const togglingId = ref('')
const drawerVisible = ref(false)
const batchEditVisible = ref(false)
const importVisible = ref(false)
const creatingMode = ref<AlarmItemMode>('point')
const editingItem = ref<AlarmItem | null>(null)
const initialPoints = ref<AlarmPointSelection[]>([])
const selectedItems = ref<AlarmItem[]>([])
const batchAllResults = ref(false)
const isApplyingBatchScope = ref(false)
const tableRef = ref<TableInstance>()
const search = ref('')
const modeFilter = ref<AlarmItemMode | ''>('')
const typeFilter = ref<AlarmConditionKind | ''>('')
const severityFilter = ref<AlarmSeverity | ''>('')
const enabledFilter = ref('')
const groupFilter = ref('')
const datapointFilter = ref('')
const directoryVisible = ref(localStorage.getItem('datacenter.alarm.directory-visible') === 'true')
const severityValues: AlarmSeverity[] = ['info', 'warning', 'major', 'critical']
const conditionKinds: AlarmConditionKind[] = [
  'threshold',
  'range',
  'state',
  'transition',
  'text_match',
  'rate_of_change',
  'deviation',
  'offline',
  'expression',
]
const pagination = reactive({ page: 1, pageSize: 20, total: 0, totalPages: 1 })
const activeFilter = computed(() => ({
  search: search.value.trim() || undefined,
  mode: modeFilter.value || undefined,
  alarmType: typeFilter.value || undefined,
  severity: severityFilter.value || undefined,
  enabled: enabledFilter.value === '' ? undefined : enabledFilter.value === 'true',
  groupId: groupFilter.value || undefined,
  datapointId: datapointFilter.value || undefined,
}))
const currentSelection = computed<AlarmItemSelection>(() =>
  batchAllResults.value
    ? { ids: [], filter: activeFilter.value }
    : { ids: selectedItems.value.map((item) => item.id) },
)
const batchAffectedCount = computed(() =>
  batchAllResults.value ? pagination.total : selectedItems.value.length,
)

async function loadItems() {
  if (!props.projectId) return
  loading.value = true
  try {
    const result = await listAlarmItems(props.projectId, {
      page: pagination.page,
      pageSize: pagination.pageSize,
      ...activeFilter.value,
    })
    items.value = result.list
    pagination.total = result.pagination.total
    pagination.totalPages = Math.max(1, Math.ceil(pagination.total / pagination.pageSize))
    await nextTick()
    restoreCurrentPageSelection()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载报警项失败'))
  } finally {
    loading.value = false
  }
}
async function loadGroups() {
  try {
    groups.value = await listAlarmGroupTree(props.projectId)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载报警目录失败'))
  }
}
async function loadChannels() {
  try {
    channels.value = ensureBuiltinAlarmNotificationChannel(
      props.projectId,
      await listAlarmChannels(props.projectId),
    )
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载通知渠道失败'))
  }
}
async function reload() {
  pagination.page = 1
  clearBatchSelection()
  await loadItems()
}
function toggleDirectory() {
  directoryVisible.value = !directoryVisible.value
  localStorage.setItem('datacenter.alarm.directory-visible', String(directoryVisible.value))
}
function selectGroup(id: string) {
  groupFilter.value = id
  void reload()
}
function filterByDatapoint(item: AlarmItem) {
  if (item.mode !== 'point') return
  datapointFilter.value = item.datapointId
  void reload()
}
function changePage(next: { page: number; pageSize: number }) {
  pagination.page = next.page
  pagination.pageSize = next.pageSize
  void loadItems()
}
function openCreate(mode: AlarmItemMode) {
  creatingMode.value = mode
  editingItem.value = null
  initialPoints.value = props.initialDatapoints.map((point) => ({ ...point }))
  drawerVisible.value = true
}
function openEdit(item: AlarmItem) {
  void openById(item.id)
}
async function openById(id: string) {
  try {
    const item = await getAlarmItem(props.projectId, id)
    creatingMode.value = item.mode
    editingItem.value = item
    initialPoints.value = []
    drawerVisible.value = true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载报警项失败'))
  }
}
async function saveItem(payload: AlarmItemSave, datapointIds: string[]) {
  saving.value = true
  try {
    if (editingItem.value) await updateAlarmItem(props.projectId, editingItem.value.id, payload)
    else if (payload.mode === 'point') {
      const { datapointId: _datapointId, ...draft } = payload
      await batchCreateAlarmItems(props.projectId, { datapointIds, draft })
    } else await createAlarmItem(props.projectId, payload)
    drawerVisible.value = false
    ElMessage.success(
      payload.mode === 'point' && datapointIds.length > 1
        ? `已创建 ${datapointIds.length} 条独立报警项`
        : '报警项已保存',
    )
    clearBatchSelection()
    await loadItems()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存报警项失败'))
  } finally {
    saving.value = false
  }
}
async function toggleItem(item: AlarmItem, enabled: boolean) {
  togglingId.value = item.id
  try {
    const saved = await setAlarmItemEnabled(props.projectId, item.id, enabled)
    const index = items.value.findIndex((row) => row.id === item.id)
    if (index >= 0) items.value[index] = saved
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '更新报警状态失败'))
  } finally {
    togglingId.value = ''
  }
}
async function removeItem(item: AlarmItem) {
  if (
    !(await confirm(`确认删除报警「${item.displayName}」？节点历史数据不会因此删除。`, {
      title: '删除报警',
      confirmText: '删除',
      type: 'warning',
    }))
  )
    return
  try {
    await deleteAlarmItem(props.projectId, item.id)
    ElMessage.success('报警项已删除')
    clearBatchSelection()
    await loadItems()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '删除报警项失败'))
  }
}
function openBatchEdit() {
  if (!batchAffectedCount.value) return ElMessage.warning('请选择报警项')
  batchEditVisible.value = true
}
function handleSelectionChange(rows: AlarmItem[]) {
  if (isApplyingBatchScope.value) return

  // Element Plus 每页只回传当前表格行；保留其他页已经选择的报警项。
  const currentPageIds = new Set(items.value.map((item) => item.id))
  const retained = selectedItems.value.filter((item) => !currentPageIds.has(item.id))
  selectedItems.value = [...retained, ...rows]
  batchAllResults.value = false
}
function selectTableRows(rows: AlarmItem[]) {
  isApplyingBatchScope.value = true
  selectedItems.value = [...rows]
  tableRef.value?.clearSelection()
  rows.forEach((item) => tableRef.value?.toggleRowSelection(item, true))
  isApplyingBatchScope.value = false
}
function restoreCurrentPageSelection() {
  const selectedIds = new Set(selectedItems.value.map((item) => item.id))
  isApplyingBatchScope.value = true
  tableRef.value?.clearSelection()
  items.value.forEach((item) => {
    if (selectedIds.has(item.id)) tableRef.value?.toggleRowSelection(item, true)
  })
  isApplyingBatchScope.value = false
}
function selectCurrentPage() {
  batchAllResults.value = false
  selectTableRows([...items.value])
}
function selectAllResults() {
  if (!pagination.total) return
  batchAllResults.value = true
  selectTableRows([...items.value])
}
function clearBatchSelection() {
  batchAllResults.value = false
  tableRef.value?.clearSelection()
  selectedItems.value = []
}
async function applyBatchEdit(payload: AlarmBatchUpdate) {
  saving.value = true
  try {
    const result = await batchUpdateAlarmItems(props.projectId, payload)
    batchEditVisible.value = false
    ElMessage.success(`已修改 ${result.affectedCount} 条报警项`)
    clearBatchSelection()
    await loadItems()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '批量修改失败'))
  } finally {
    saving.value = false
  }
}
async function batchSetEnabled(isEnabled: boolean) {
  if (!batchAffectedCount.value) return
  await applyBatchEdit({
    selection: currentSelection.value,
    fields: ['isEnabled'],
    patch: { isEnabled },
    acknowledgedWarningKeys: [],
  })
}
async function batchRemove() {
  if (
    !batchAffectedCount.value ||
    !(await confirm(`确认删除 ${batchAffectedCount.value} 条报警项？该操作不可撤销。`, {
      title: '批量删除',
      confirmText: '删除',
      type: 'warning',
    }))
  )
    return
  try {
    const result = await batchDeleteAlarmItems(props.projectId, currentSelection.value)
    ElMessage.success(`已删除 ${result.affectedCount} 条报警项`)
    clearBatchSelection()
    await loadItems()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '批量删除失败'))
  }
}
function saveBlob(blob: Blob, name: string) {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = name
  anchor.click()
  URL.revokeObjectURL(url)
}
async function handleExcelCommand(command: string) {
  try {
    if (command === 'import') return void (importVisible.value = true)
    if (command === 'template')
      return saveBlob(await downloadAlarmImportTemplate(props.projectId), '报警项导入模板.xlsx')
    const selection =
      selectedItems.value.length || batchAllResults.value
        ? currentSelection.value
        : { ids: [], filter: activeFilter.value }
    saveBlob(await exportAlarmItems(props.projectId, selection), '报警项.xlsx')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '处理 Excel 失败'))
  }
}
function severityTone(severity: AlarmSeverity) {
  return ({ info: 'info', warning: 'warning', major: 'warning', critical: 'danger' } as const)[
    severity
  ]
}
watch(search, () => {
  window.clearTimeout((loadItems as unknown as { timer?: number }).timer)
  ;(loadItems as unknown as { timer?: number }).timer = window.setTimeout(reload, 250)
})
watch(
  () => props.selectedPolicyId,
  (id) => {
    if (id) void openById(id)
  },
  { immediate: true },
)
watch(drawerVisible, (opened) => {
  if (!opened && props.selectedPolicyId) emit('close-selected-policy')
})
watch(
  () => props.projectId,
  async () => {
    pagination.page = 1
    await Promise.all([loadItems(), loadGroups(), loadChannels()])
  },
)
onMounted(() => Promise.all([loadItems(), loadGroups(), loadChannels()]))
</script>

<style scoped>
.alarm-workspace,
.alarm-workspace__panel {
  height: 100%;
  min-width: 0;
}
.alarm-workspace__panel {
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}
.alarm-workspace__head {
  flex: 0 0 auto;
  min-width: 0;
  border-bottom: 1px solid var(--dc-border);
}
.alarm-workspace__tabs-row {
  min-height: 44px;
  display: flex;
  align-items: center;
  padding: 8px 14px;
}
.alarm-workspace__toolbar-row {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 14px 10px;
  border-top: 1px solid var(--dc-border);
}
.alarm-workspace__toolbar {
  min-width: 0;
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  overflow-x: auto;
  padding-bottom: 1px;
}
.alarm-workspace__search {
  flex: 0 1 230px;
  min-width: 160px;
}
.alarm-workspace__filter {
  width: 112px;
  flex: 0 0 auto;
}
.alarm-workspace__filter.is-wide {
  width: 132px;
}
.alarm-workspace__create-actions {
  display: flex;
  gap: 8px;
  flex: 0 0 auto;
}
.alarm-workspace__primary,
.alarm-workspace__secondary,
.alarm-workspace__icon {
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border-radius: var(--dc-radius-sm);
  font: inherit;
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}
.alarm-workspace__primary {
  padding: 0 13px;
  border: 1px solid var(--dc-primary);
  background: var(--dc-primary);
  color: white;
}
.alarm-workspace__secondary {
  padding: 0 12px;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}
.alarm-workspace__icon {
  width: 32px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--dc-text-muted);
}
.alarm-workspace__content-panel {
  min-width: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  overflow: hidden;
}
.alarm-workspace__content-panel > :not(.alarm-directories) {
  min-width: 0;
  min-height: 0;
  flex: 1;
}
.alarm-workspace__table-panel {
  position: relative;
  min-width: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.alarm-workspace__batch-action {
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: 8px;
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  cursor: pointer;
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
}
.alarm-workspace__batch-action:hover {
  border-color: rgba(29, 78, 216, 0.26);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.alarm-workspace__batch-action.is-active {
  border-color: rgba(29, 78, 216, 0.3);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.alarm-workspace__batch-action.is-danger:hover {
  border-color: rgba(220, 38, 38, 0.26);
  background: rgba(220, 38, 38, 0.08);
  color: var(--dc-danger);
}
.alarm-workspace__batch-action:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}
.alarm-workspace__table-scroll {
  min-width: 0;
  min-height: 0;
  flex: 1;
  overflow-x: auto;
}
.alarm-workspace__table-panel > .dc-pagination {
  position: relative;
  z-index: 1;
  flex: 0 0 auto;
}
.alarm-workspace__table {
  min-width: 1180px;
}
.alarm-workspace__name,
.alarm-workspace__ellipsis {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.alarm-workspace__point-link {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 0;
  border: 0;
  background: none;
  color: var(--dc-primary);
  font: inherit;
  text-align: left;
}
.alarm-workspace__row-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
  white-space: nowrap;
}
.alarm-workspace__row-actions button {
  width: 30px;
  height: 30px;
  display: inline-grid;
  place-items: center;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
}
.alarm-workspace__row-actions button.is-danger {
  color: var(--dc-danger);
}
@media (max-width: 900px) {
  .alarm-workspace__toolbar-row {
    align-items: flex-start;
    flex-direction: column;
    gap: 8px;
  }
  .alarm-workspace__toolbar {
    width: 100%;
  }
  .alarm-workspace__create-actions {
    width: 100%;
  }
}
</style>
