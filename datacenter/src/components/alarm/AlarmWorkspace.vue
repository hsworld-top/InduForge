<template>
  <div class="alarm-workspace">
    <section class="alarm-workspace__panel">
      <FilterToolbar>
        <template #tabs>
          <el-segmented v-model="activeTab" :options="tabOptions" size="small" />
        </template>
        <template v-if="activeTab === 'items'">
          <el-input
            v-model="search"
            class="alarm-workspace__search"
            size="small"
            clearable
            :placeholder="t('alarm.searchItems')"
            :prefix-icon="Search"
            @keyup.enter="reload"
          />
          <PillButton :active="directoryVisible" @click="toggleDirectory"
            ><template #icon><IconTablerFolders /></template>{{ t('alarm.directory') }}</PillButton
          >
          <el-select
            v-model="modeFilter"
            size="small"
            class="alarm-workspace__filter"
            @change="reload"
            ><el-option :label="t('alarm.allModes')" value="" /><el-option
              :label="t('alarm.pointAlarm')"
              value="point" /><el-option :label="t('alarm.derivedAlarm')" value="derived"
          /></el-select>
          <el-select
            v-model="typeFilter"
            size="small"
            class="alarm-workspace__filter is-wide"
            @change="reload"
            ><el-option :label="t('alarm.allAlarmTypes')" value="" /><el-option
              v-for="kind in conditionKinds"
              :key="kind"
              :label="conditionLabel(kind)"
              :value="kind"
          /></el-select>
          <el-select
            v-model="severityFilter"
            size="small"
            class="alarm-workspace__filter"
            @change="reload"
            ><el-option :label="t('alarm.allLevels')" value="" /><el-option
              v-for="severity in severityValues"
              :key="severity"
              :label="severityLabel(severity)"
              :value="severity"
          /></el-select>
          <el-select
            v-model="enabledFilter"
            size="small"
            class="alarm-workspace__filter"
            @change="reload"
            ><el-option :label="t('alarm.allStatuses')" value="" /><el-option
              :label="t('alarm.enabled')"
              value="true" /><el-option :label="t('alarm.disabled')" value="false"
          /></el-select>
          <button type="button" class="alarm-workspace__icon" :title="t('alarm.refresh')" @click="loadItems">
            <IconTablerRefresh />
          </button>
        </template>
        <template v-if="activeTab === 'items'" #actions>
          <el-dropdown trigger="click" @command="handleExcelCommand"
            ><button type="button" class="alarm-workspace__secondary">
              <IconTablerFileSpreadsheet />Excel</button
            ><template #dropdown
              ><el-dropdown-menu
                ><el-dropdown-item command="template">{{ t('alarm.downloadTemplate') }}</el-dropdown-item
                ><el-dropdown-item command="import">{{ t('alarm.importItems') }}</el-dropdown-item
                ><el-dropdown-item command="export" :disabled="pagination.total === 0"
                  >{{ t('alarm.exportCurrentScope') }}</el-dropdown-item
                ></el-dropdown-menu
              ></template
            ></el-dropdown
          >
          <button type="button" class="alarm-workspace__secondary" @click="openCreate('derived')">
            <IconTablerFunction />{{ t('alarm.createDerived') }}
          </button>
          <button type="button" class="alarm-workspace__primary" @click="openCreate('point')">
            <IconTablerPlus />{{ t('alarm.createAlarm') }}
          </button>
        </template>
      </FilterToolbar>

      <section class="alarm-workspace__content-panel">
        <template v-if="activeTab === 'items'">
          <AlarmDirectoryPanel
            v-if="directoryVisible"
            :project-id="projectId"
            :selected-id="groupFilter"
            @select="selectGroup"
          />
          <div class="alarm-workspace__table-panel">
            <TableScroll>
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
                  <el-table-column :label="t('alarm.alarmName')" min-width="170" fixed="left"
                    ><template #default="{ row }"
                      ><strong class="alarm-workspace__name" :title="row.displayName">{{
                        row.displayName
                      }}</strong></template
                    ></el-table-column
                  >
                  <el-table-column :label="t('alarm.datapoint')" min-width="140"
                    ><template #default="{ row }"
                      ><button
                        type="button"
                        class="alarm-workspace__point-link"
                        :title="pointSummary(row)"
                        @click="filterByDatapoint(row)"
                      >
                        {{ pointSummary(row) }}
                      </button></template
                    ></el-table-column
                  >
                  <el-table-column :label="t('alarm.datapointPath')" min-width="230" show-overflow-tooltip
                    ><template #default="{ row }"
                      ><span class="alarm-workspace__path" :title="row.path || '—'">{{
                        row.path || '—'
                      }}</span></template
                    ></el-table-column
                  >
                  <el-table-column :label="t('alarm.highestLevel')" width="96"
                    ><template #default="{ row }"
                      ><span
                        class="alarm-workspace__severity"
                        :style="severityBadgeStyle(highestSeverity(row))"
                        >{{ severityLabel(highestSeverity(row)) }}</span
                      ></template
                    ></el-table-column
                  >
                  <el-table-column :label="t('alarm.status')" width="82"
                    ><template #default="{ row }"
                      ><el-switch
                        :model-value="row.isEnabled"
                        :loading="togglingId === row.id"
                        @change="toggleItem(row, Boolean($event))" /></template
                  ></el-table-column>
                  <el-table-column :label="t('alarm.alarmType')" width="112"
                    ><template #default="{ row }"
                      ><span class="alarm-workspace__ellipsis">{{
                        row.mode === 'derived' ? t('alarm.derivedAlarm') : conditionLabel(row.alarmType)
                      }}</span></template
                    ></el-table-column
                  >
                  <el-table-column :label="t('alarm.conditionSummary')" min-width="180"
                    ><template #default="{ row }"
                      ><span
                        class="alarm-workspace__ellipsis"
                        :title="conditionSummary(row)"
                        >{{ conditionSummary(row) }}</span
                      ></template
                    ></el-table-column
                  >
                  <el-table-column :label="t('alarm.directoryColumn')" min-width="130"
                    ><template #default="{ row }"
                      ><span class="alarm-workspace__ellipsis">{{
                        row.groupName || t('alarm.rootDirectory')
                      }}</span></template
                    ></el-table-column
                  >
                  <el-table-column :label="t('alarm.actions')" width="104" align="right" fixed="right"
                    ><template #default="{ row }"
                      ><div class="alarm-workspace__row-actions">
                        <button type="button" :title="t('alarm.edit')" @click="openEdit(row)">
                          <IconTablerEdit /></button
                        ><button
                          type="button"
                          class="is-danger"
                          :title="t('alarm.delete')"
                          @click="removeItem(row)"
                        >
                          <IconTablerTrash />
                        </button></div></template
                  ></el-table-column>
                  <template #empty><el-empty :image-size="54" :description="t('alarm.emptyItems')" /></template>
                </el-table>
              </div>
              <template #pagination>
                <DataCenterPagination
                  :page="pagination.page"
                  :page-size="pagination.pageSize"
                  :total="pagination.total"
                  :total-pages="pagination.totalPages"
                  @change="changePage"
                />
              </template>
              <template #floating>
                <BulkActionBar
                  :selected-count="batchAffectedCount"
                  :item-label="t('alarm.itemUnit')"
                  @clear="clearBatchSelection"
                >
                  <button
                    type="button"
                    class="alarm-workspace__batch-action"
                    :class="{ 'is-active': !batchAllResults }"
                    @click="selectCurrentPage"
                  >
                    {{ t('alarm.currentPage') }}
                  </button>
                  <button
                    type="button"
                    class="alarm-workspace__batch-action"
                    :class="{ 'is-active': batchAllResults }"
                    :disabled="pagination.total === 0"
                    @click="selectAllResults"
                  >
                    {{ t('alarm.allResults') }}
                  </button>
                  <button
                    type="button"
                    class="alarm-workspace__batch-action"
                    @click="openBatchEdit"
                  >
                    {{ t('alarm.batchEdit') }}
                  </button>
                  <button
                    type="button"
                    class="alarm-workspace__batch-action"
                    @click="batchSetEnabled(true)"
                  >
                    {{ t('alarm.enable') }}
                  </button>
                  <button
                    type="button"
                    class="alarm-workspace__batch-action"
                    @click="batchSetEnabled(false)"
                  >
                    {{ t('alarm.disable') }}
                  </button>
                  <button
                    type="button"
                    class="alarm-workspace__batch-action is-danger"
                    @click="batchRemove"
                  >
                    {{ t('alarm.delete') }}
                  </button>
                </BulkActionBar>
              </template>
            </TableScroll>
          </div>
        </template>
        <AlarmNotificationSettings
          v-else-if="activeTab === 'notifications'"
          :project-id="projectId"
          @changed="channels = $event"
        />
        <AlarmLevelSettings
          v-else-if="activeTab === 'levels'"
          :project-id="projectId"
          @changed="levelSettings = $event"
        />
        <AlarmHistorySettings v-else :project-id="projectId" />
      </section>
    </section>

    <AlarmItemDrawer
      v-model="drawerVisible"
      :project-id="projectId"
      :mode="creatingMode"
      :item="editingItem"
      :channels="channels"
      :initial-points="initialPoints"
      :saving="saving"
      @save="saveItem"
      @open-conflict="openById"
    />
    <AlarmDatapointManagerDialog
      v-model="pointPickerVisible"
      :project-id="projectId"
      mode="point"
      :points="initialPoints"
      @apply="applyQuickAlarmPoints"
    />
    <QuickAlarmDrawer
      v-model="quickDrawerVisible"
      :project-id="projectId"
      :points="initialPoints"
      :saving="saving"
      @save="savePresetItem"
    />
    <AlarmBatchEditDialog
      v-model="batchEditVisible"
      :project-id="projectId"
      :selection="currentSelection"
      :affected-count="batchAffectedCount"
      :items="batchAllResults ? [] : selectedItems"
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
import AlarmLevelSettings from './AlarmLevelSettings.vue'
import AlarmNotificationSettings from './AlarmNotificationSettings.vue'
import AlarmItemDrawer from './AlarmItemDrawer.vue'
import AlarmDatapointManagerDialog from './AlarmDatapointManagerDialog.vue'
import QuickAlarmDrawer from './QuickAlarmDrawer.vue'
import BulkActionBar from '@/components/shared/BulkActionBar.vue'
import DataCenterPagination from '@/components/shared/DataCenterPagination.vue'
import FilterToolbar from '@/components/shared/FilterToolbar.vue'
import PillButton from '@/components/shared/PillButton.vue'
import TableScroll from '@/components/shared/TableScroll.vue'
import {
  batchCreateAlarmItems,
  batchDeleteAlarmItems,
  batchUpdateAlarmItems,
  createAlarmItem,
  deleteAlarmItem,
  downloadAlarmImportTemplate,
  exportAlarmItems,
  getAlarmItem,
  getAlarmLevelSettings,
  listAlarmChannels,
  listAlarmItems,
  setAlarmItemEnabled,
  savePresetAlarmConfiguration,
  updateAlarmItem,
} from '@/api/alarm.api'
import type {
  AlarmBatchUpdate,
  AlarmConditionKind,
  AlarmItem,
  AlarmItemMode,
  AlarmItemSave,
  AlarmItemSelection,
  AlarmLevelSettings as AlarmLevelSettingsModel,
  AlarmNotificationChannel,
  AlarmSeverity,
} from '@/api/schemas/alarm.schema'
import {
  alarmConditionLabels,
  alarmItemConditionSummary,
  alarmItemHighestSeverity,
  alarmItemPointSummary,
  alarmSeverityLabel,
  defaultAlarmSeverityDefinitions,
  ensureBuiltinAlarmNotificationChannel,
  type AlarmPointSelection,
} from '@/models/alarm-item'
import { getApiErrorMessage } from '@/utils/request'
import { useConfirm } from '@/composables/useConfirm'
import { datacenterLocale, t } from '@/i18n/runtime'

const props = withDefaults(
  defineProps<{
    projectId: string
    selectedItemId?: string
    initialDatapoints?: AlarmPointSelection[]
  }>(),
  { selectedItemId: '', initialDatapoints: () => [] },
)
const emit = defineEmits<{ 'close-selected-item': [] }>()
const { confirm } = useConfirm()
const activeTab = ref<'items' | 'levels' | 'notifications' | 'history'>('items')
const tabOptions = computed(() => [
  { label: t('alarm.tabs.items'), value: 'items' },
  { label: t('alarm.tabs.levels'), value: 'levels' },
  { label: t('alarm.tabs.notifications'), value: 'notifications' },
  { label: t('alarm.tabs.history'), value: 'history' },
])
const items = ref<AlarmItem[]>([])
const channels = ref<AlarmNotificationChannel[]>([])
const levelSettings = ref<AlarmLevelSettingsModel>({
  projectId: props.projectId,
  severityDefinitions: defaultAlarmSeverityDefinitions.map((item) => ({ ...item })),
  escalationRules: [],
})
const loading = ref(false)
const saving = ref(false)
const togglingId = ref('')
const drawerVisible = ref(false)
const pointPickerVisible = ref(false)
const quickDrawerVisible = ref(false)
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
const severityValues = computed(() =>
  levelSettings.value.severityDefinitions.map((item) => item.key),
)
const conditionKinds: AlarmConditionKind[] = [
  'threshold',
  'range',
  'state',
  'transition',
  'text_match',
  'rate_of_change',
  'deviation',
  'quality',
  'stale',
  'offline',
]
const pagination = reactive({ page: 1, pageSize: 20, total: 0, totalPages: 0 })
let itemRequestSeq = 0
let channelRequestSeq = 0
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
  const seq = ++itemRequestSeq
  loading.value = true
  try {
    const result = await listAlarmItems(props.projectId, {
      page: pagination.page,
      pageSize: pagination.pageSize,
      ...activeFilter.value,
    })
    if (seq !== itemRequestSeq) return
    items.value = result.list
    pagination.total = result.pagination.total
    pagination.totalPages =
      pagination.total === 0 ? 0 : Math.ceil(pagination.total / pagination.pageSize)
    await nextTick()
    restoreCurrentPageSelection()
  } catch (error) {
    if (seq !== itemRequestSeq) return
    ElMessage.error(getApiErrorMessage(error, t('alarm.loadItemsFailed')))
  } finally {
    if (seq === itemRequestSeq) loading.value = false
  }
}
async function loadChannels() {
  const seq = ++channelRequestSeq
  try {
    const result = ensureBuiltinAlarmNotificationChannel(
      props.projectId,
      await listAlarmChannels(props.projectId),
    )
    if (seq === channelRequestSeq) channels.value = result
  } catch (error) {
    if (seq !== channelRequestSeq) return
    ElMessage.error(getApiErrorMessage(error, t('alarm.loadChannelsFailed')))
  }
}
async function loadLevelSettings() {
  try {
    levelSettings.value = await getAlarmLevelSettings(props.projectId)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarm.loadLevelsFailed')))
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
  if (mode === 'derived') {
    drawerVisible.value = true
    return
  }
  if (initialPoints.value.length) quickDrawerVisible.value = true
  else pointPickerVisible.value = true
}
function applyQuickAlarmPoints(points: AlarmPointSelection[]) {
  initialPoints.value = points
  pointPickerVisible.value = false
  quickDrawerVisible.value = true
}
async function savePresetItem(payload: AlarmItemSave | AlarmItemSave[], datapointIds: string[]) {
  const payloads = Array.isArray(payload) ? payload : [payload]
  saving.value = true
  try {
    await savePresetAlarmConfiguration(props.projectId, {
      datapointIds,
      drafts: payloads.map(({ datapointId: _datapointId, ...draft }) => draft),
    })
    quickDrawerVisible.value = false
    ElMessage.success(t('alarm.configSaved'))
    clearBatchSelection()
    await loadItems()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarm.saveConfigFailed')))
  } finally {
    saving.value = false
  }
}
function openEdit(item: AlarmItem) {
  void openById(item.id)
}
async function openById(id: string) {
  try {
    const item = await getAlarmItem(props.projectId, id)
    if (item.mode === 'point' && item.presetSlot) {
      creatingMode.value = 'point'
      editingItem.value = null
      initialPoints.value = [
        {
          datapointId: item.datapointId,
          path: item.path,
          name: item.datapointName || item.path,
          dataType: item.dataType,
        },
      ]
      quickDrawerVisible.value = true
      return
    }
    creatingMode.value = item.mode
    editingItem.value = item
    initialPoints.value = []
    drawerVisible.value = true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarm.loadItemFailed')))
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
        ? t('alarm.itemsCreated', { count: datapointIds.length })
        : t('alarm.itemSaved'),
    )
    clearBatchSelection()
    await loadItems()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarm.saveItemFailed')))
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
    ElMessage.error(getApiErrorMessage(error, t('alarm.updateStatusFailed')))
  } finally {
    togglingId.value = ''
  }
}
async function removeItem(item: AlarmItem) {
  if (
    !(await confirm(t('alarm.deleteConfirm', { name: item.displayName }), {
      title: t('alarm.deleteTitle'),
      confirmText: t('alarm.delete'),
      type: 'warning',
    }))
  )
    return
  try {
    await deleteAlarmItem(props.projectId, item.id)
    ElMessage.success(t('alarm.itemDeleted'))
    clearBatchSelection()
    await loadItems()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarm.deleteItemFailed')))
  }
}
function openBatchEdit() {
  if (!batchAffectedCount.value) return ElMessage.warning(t('alarm.selectItems'))
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
    ElMessage.success(t('alarm.itemsModified', { count: result.affectedCount }))
    clearBatchSelection()
    await loadItems()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarm.batchEditFailed')))
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
    !(await confirm(t('alarm.batchDeleteConfirm', { count: batchAffectedCount.value }), {
      title: t('alarm.batchDeleteTitle'),
      confirmText: t('alarm.delete'),
      type: 'warning',
    }))
  )
    return
  try {
    const result = await batchDeleteAlarmItems(props.projectId, currentSelection.value)
    ElMessage.success(t('alarm.itemsDeleted', { count: result.affectedCount }))
    clearBatchSelection()
    await loadItems()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarm.batchDeleteFailed')))
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
      return saveBlob(
        await downloadAlarmImportTemplate(props.projectId),
        t('alarm.importTemplateFile'),
      )
    const selection =
      selectedItems.value.length || batchAllResults.value
        ? currentSelection.value
        : { ids: [], filter: activeFilter.value }
    saveBlob(await exportAlarmItems(props.projectId, selection), t('alarm.exportFile'))
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarm.excelFailed')))
  }
}
function severityLabel(severity: AlarmSeverity) {
  if (['info', 'warning', 'major', 'critical'].includes(severity)) {
    return t(`alarm.severities.${severity}`)
  }
  return alarmSeverityLabel(severity, levelSettings.value.severityDefinitions)
}
function conditionLabel(kind: AlarmConditionKind) {
  return t(`alarm.conditionLabels.${kind}`)
}
function localizedConditionDetailLabel(label: string, kind: AlarmConditionKind) {
  if (datacenterLocale.value !== 'en') return label || conditionLabel(kind)
  const builtinLabels: Record<string, string> = {
    '高': t('quickAlarm.levels.h'),
    '高限': t('quickAlarm.levels.h'),
    '高限报警': t('quickAlarm.levels.h'),
    '高高': t('quickAlarm.levels.hh'),
    '高高限': t('quickAlarm.levels.hh'),
    '高高限报警': t('quickAlarm.levels.hh'),
    '低': t('quickAlarm.levels.l'),
    '低限': t('quickAlarm.levels.l'),
    '低限报警': t('quickAlarm.levels.l'),
    '低低': t('quickAlarm.levels.ll'),
    '低低限': t('quickAlarm.levels.ll'),
    '低低限报警': t('quickAlarm.levels.ll'),
    '变化率': t('quickAlarm.levels.rate'),
    '变化率报警': t('quickAlarm.levels.rate'),
    '偏差': t('quickAlarm.levels.deviation'),
    '偏差报警': t('quickAlarm.levels.deviation'),
    '结果条件': conditionLabel(kind),
  }
  return builtinLabels[label] || label || conditionLabel(kind)
}
function pointSummary(item: AlarmItem) {
  if (datacenterLocale.value === 'zh') return alarmItemPointSummary(item)
  if (item.mode === 'point') return item.datapointName || item.path || t('alarm.datapoint')
  const first = item.inputs[0]
  if (!first) return t('alarm.derivedAlarm')
  return item.inputs.length === 1
    ? first.name || first.path
    : `${t('alarm.derivedAlarm')} · ${first.name || first.path} +${item.inputs.length - 1}`
}
function conditionSummary(item: AlarmItem) {
  if (datacenterLocale.value === 'zh') return alarmItemConditionSummary(item)
  if (item.evaluationMode === 'highest_matching') {
    return `${conditionLabel('threshold')} · ${item.conditions
      .map((condition) => localizedConditionDetailLabel(condition.label, condition.kind))
      .join('/')}`
  }
  return item.conditions[0] ? conditionLabel(item.conditions[0].kind) : 'Not configured'
}
function severityBadgeStyle(severity: AlarmSeverity) {
  const color =
    levelSettings.value.severityDefinitions.find((item) => item.key === severity)?.color ??
    '#64748b'
  return {
    color,
    borderColor: `${color}55`,
    backgroundColor: `${color}14`,
  }
}
function highestSeverity(item: AlarmItem) {
  return alarmItemHighestSeverity(item, levelSettings.value.severityDefinitions)
}
watch(search, () => {
  window.clearTimeout((loadItems as unknown as { timer?: number }).timer)
  ;(loadItems as unknown as { timer?: number }).timer = window.setTimeout(reload, 250)
})
watch(
  () => props.selectedItemId,
  (id) => {
    if (id) void openById(id)
  },
  { immediate: true },
)
watch(drawerVisible, (opened) => {
  if (!opened && props.selectedItemId) emit('close-selected-item')
})
watch(
  () => props.projectId,
  async () => {
    pagination.page = 1
    await Promise.all([loadItems(), loadChannels(), loadLevelSettings()])
  },
)
onMounted(() => Promise.all([loadItems(), loadChannels(), loadLevelSettings()]))
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
  height: 100%;
  min-width: 0;
  min-height: 0;
  flex: 1;
  overflow-x: auto;
}
.alarm-workspace__table {
  min-width: 1290px;
}
.alarm-workspace__name,
.alarm-workspace__path,
.alarm-workspace__ellipsis {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.alarm-workspace__path {
  color: var(--dc-text-muted);
  font-family: var(--dc-font-mono, ui-monospace, SFMono-Regular, Menlo, monospace);
  font-size: 12px;
}
.alarm-workspace__severity {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 2px 8px;
  border: 1px solid;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
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
</style>
