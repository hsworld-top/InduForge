<template>
  <div class="history-storage-workspace">
    <section class="history-storage-workspace__panel">
      <header class="history-storage-workspace__head">
        <FilterToolbar>
          <el-input
            v-model="search"
            class="history-storage-workspace__search"
            size="small"
            clearable
            :placeholder="ui('搜索来源名称', 'Search source names')"
            :prefix-icon="Search"
          />

          <el-popover trigger="click" placement="bottom-start" :width="150">
            <template #reference>
              <PillButton :active="scopeType !== ''">{{ scopeTypeLabel }}</PillButton>
            </template>
            <div class="history-storage-workspace__filter-menu">
              <button
                v-for="option in scopeTypeOptions"
                :key="option.value"
                type="button"
                :class="{ 'is-active': scopeType === option.value }"
                @click="scopeType = option.value"
              >
                {{ option.label }}
              </button>
            </div>
          </el-popover>

          <el-popover trigger="click" placement="bottom-start" :width="160">
            <template #reference>
              <PillButton :active="historyState !== ''">{{ historyStateLabel }}</PillButton>
            </template>
            <div class="history-storage-workspace__filter-menu">
              <button
                v-for="option in historyStateOptions"
                :key="option.value"
                type="button"
                :class="{ 'is-active': historyState === option.value }"
                @click="historyState = option.value"
              >
                {{ option.label }}
              </button>
            </div>
          </el-popover>

          <button
            type="button"
            class="history-storage-workspace__icon-button"
            :title="ui('刷新', 'Refresh')"
            :aria-label="ui('刷新历史存储来源', 'Refresh history storage sources')"
            @click="loadSources"
          >
            <IconTablerRefresh />
          </button>

          <span class="history-storage-workspace__total">{{ ui(`共 ${pagination.total}`, `${pagination.total} total`) }}</span>
        </FilterToolbar>
      </header>

      <section class="history-storage-workspace__content-panel">
        <TableScroll>
          <div v-loading="loading" class="history-storage-workspace__table-wrap">
            <el-table
              :data="sources"
              row-key="scope.id"
              height="100%"
              class="history-storage-workspace__table"
            >
              <el-table-column :label="ui('来源', 'Source')" min-width="260">
                <template #default="{ row }">
                  <div class="history-storage-workspace__identity">
                    <span
                      :class="
                        row.scope.type === 'collector_connection' ? 'is-collector' : 'is-source'
                      "
                    >
                      <IconTablerFunction v-if="row.scope.type === 'compute_unit'" />
                      <IconTablerCpu v-else-if="row.scope.type === 'collector_connection'" />
                      <IconTablerRouter v-else />
                    </span>
                    <div>
                      <strong>{{ row.scope.name }}</strong>
                      <small>{{ sourceTypeLabel(row) }}</small>
                    </div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column :label="ui('数据点', 'Data Points')" width="110" align="center">
                <template #default="{ row }">
                  <span class="history-storage-workspace__point-count">{{
                    row.datapointCount
                  }}</span>
                </template>
              </el-table-column>
              <el-table-column :label="ui('历史存储', 'History Storage')" min-width="300">
                <template #default="{ row }">
                  <div class="history-storage-workspace__summary">
                    <StatusBadge
                      :tone="row.historyState === 'enabled' ? 'success' : 'muted'"
                      :text="localizedHistoryStorageSummary(row)"
                    />
                    <small v-if="row.pointOverrideCount > 0">
                      {{ ui(`${row.pointOverrideCount} 个数据点单独设置`, `${row.pointOverrideCount} point override${row.pointOverrideCount === 1 ? '' : 's'}`) }}
                    </small>
                  </div>
                </template>
              </el-table-column>
              <el-table-column :label="ui('操作', 'Actions')" width="88" align="right" fixed="right">
                <template #default="{ row }">
                  <button
                    type="button"
                    class="history-storage-workspace__settings"
                    :title="ui('设置历史存储', 'Configure History Storage')"
                    :aria-label="ui(`设置 ${row.scope.name} 的历史存储`, `Configure history storage for ${row.scope.name}`)"
                    @click="openSettings(row)"
                  >
                    <IconTablerSettings />
                  </button>
                </template>
              </el-table-column>
              <template #empty>
                <el-empty :description="ui('暂无匹配的来源', 'No matching sources')" />
              </template>
            </el-table>
          </div>

          <template #pagination>
            <DataCenterPagination
              :page="pagination.page"
              :page-size="pagination.pageSize"
              :total="pagination.total"
              :total-pages="pagination.totalPages"
              @change="handlePageChange"
            />
          </template>
        </TableScroll>
      </section>
    </section>

    <HistoryStorageConfigDrawer
      v-model="drawerVisible"
      :title="drawerTitle"
      :behavior="editingDetail?.behavior || 'off'"
      :configuration="editingDetail?.configuration"
      :targets="targets"
      :saving="saving"
      @save="saveSettings"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import IconTablerCpu from '~icons/tabler/cpu'
import IconTablerFunction from '~icons/tabler/function'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerRouter from '~icons/tabler/router'
import IconTablerSettings from '~icons/tabler/settings'
import DataCenterPagination from '@/components/shared/DataCenterPagination.vue'
import FilterToolbar from '@/components/shared/FilterToolbar.vue'
import PillButton from '@/components/shared/PillButton.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import TableScroll from '@/components/shared/TableScroll.vue'
import HistoryStorageConfigDrawer from '@/components/history-storage/HistoryStorageConfigDrawer.vue'
import { formatCollectorProtocolFamily } from '@/components/collector-workbench/collector-workbench-model'
import {
  getHistoryStorageSource,
  listHistoryStorageSources,
  listHistoryStorageTargets,
  saveHistoryStorageSource,
  type HistoryStorageSavePayload,
} from '@/api/history-storage.api'
import type {
  HistoryStorageScopeType,
  HistoryStorageSourceDetail,
  HistoryStorageSourceItem,
  HistoryStorageTargetOption,
} from '@/api/schemas/history-storage.schema'
import { historyStorageSummary } from '@/models/history-storage'
import { getApiErrorMessage } from '@/utils/request'
import { useConfirm } from '@/composables/useConfirm'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = defineProps<{ projectId: string }>()
const sources = ref<HistoryStorageSourceItem[]>([])
const targets = ref<HistoryStorageTargetOption[]>([])
const loading = ref(false)
const saving = ref(false)
const search = ref('')
const scopeType = ref<HistoryStorageScopeType | ''>('')
const historyState = ref<'enabled' | 'disabled' | ''>('')
const pagination = reactive({ page: 1, pageSize: 20, total: 0, totalPages: 0 })
const drawerVisible = ref(false)
const drawerTitle = ref(ui('历史存储设置', 'History Storage Settings'))
const editingRow = ref<HistoryStorageSourceItem | null>(null)
const editingDetail = ref<HistoryStorageSourceDetail | null>(null)
const { confirm } = useConfirm()
let searchTimer: ReturnType<typeof setTimeout> | null = null
let sourceRequestSeq = 0
let settingsRequestSeq = 0

const scopeTypeOptions = computed<Array<{ label: string; value: HistoryStorageScopeType | '' }>>(() => [
  { label: ui('全部来源', 'All Sources'), value: '' },
  { label: ui('接入源', 'Access Sources'), value: 'access_source' },
  { label: ui('工业采集', 'Industrial Collection'), value: 'collector_connection' },
  { label: ui('计算单元', 'Compute Units'), value: 'compute_unit' },
])
const historyStateOptions = computed<Array<{ label: string; value: 'enabled' | 'disabled' | '' }>>(() => [
  { label: ui('全部状态', 'All States'), value: '' },
  { label: ui('已保存历史', 'History Enabled'), value: 'enabled' },
  { label: ui('未保存历史', 'History Disabled'), value: 'disabled' },
])
const scopeTypeLabel = computed(
  () => scopeTypeOptions.value.find((option) => option.value === scopeType.value)?.label || ui('来源类别', 'Source Type'),
)
const historyStateLabel = computed(
  () =>
    historyStateOptions.value.find((option) => option.value === historyState.value)?.label || ui('历史状态', 'History State'),
)

async function loadSources() {
  const seq = ++sourceRequestSeq
  loading.value = true
  try {
    const result = await listHistoryStorageSources(props.projectId, {
      page: pagination.page,
      pageSize: pagination.pageSize,
      search: search.value.trim(),
      scopeType: scopeType.value,
      historyState: historyState.value,
    })
    if (seq !== sourceRequestSeq) return
    sources.value = result.list
    Object.assign(pagination, result.pagination)
  } catch (error) {
    if (seq !== sourceRequestSeq) return
    ElMessage.error(getApiErrorMessage(error, ui('加载历史存储来源失败', 'Failed to load history storage sources')))
  } finally {
    if (seq === sourceRequestSeq) loading.value = false
  }
}

async function ensureTargets() {
  if (targets.value.length > 0) return
  targets.value = await listHistoryStorageTargets(props.projectId)
}

async function openSettings(row: HistoryStorageSourceItem) {
  const seq = ++settingsRequestSeq
  try {
    const [detail] = await Promise.all([
      getHistoryStorageSource(props.projectId, row.scope.type, row.scope.id),
      ensureTargets(),
    ])
    if (seq !== settingsRequestSeq) return
    editingRow.value = row
    editingDetail.value = detail
    drawerTitle.value = `${row.scope.name} · ${ui('历史存储', 'History Storage')}`
    drawerVisible.value = true
  } catch (error) {
    if (seq !== settingsRequestSeq) return
    ElMessage.error(getApiErrorMessage(error, ui('加载历史存储设置失败', 'Failed to load history storage settings')))
  }
}

async function saveSettings(payload: HistoryStorageSavePayload) {
  const row = editingRow.value
  const detail = editingDetail.value
  if (!row || !detail) return
  if (payload.behavior === 'off' && detail.behavior === 'custom' && detail.pointOverrideCount > 0) {
    const accepted = await confirm(
      ui(`关闭来源历史后，仍有 ${detail.pointOverrideCount} 个单点例外按各自设置生效。确认关闭？`, `Disabling source history leaves ${detail.pointOverrideCount} point override${detail.pointOverrideCount === 1 ? '' : 's'} active. Continue?`),
      { title: ui('关闭来源历史', 'Disable Source History'), confirmText: ui('确认关闭', 'Disable'), type: 'warning' },
    )
    if (!accepted) return
  }
  saving.value = true
  try {
    await saveHistoryStorageSource(props.projectId, row.scope.type, row.scope.id, payload)
    ElMessage.success(ui('历史存储设置已保存', 'History storage settings saved'))
    drawerVisible.value = false
    await loadSources()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('保存历史存储设置失败', 'Failed to save history storage settings')))
  } finally {
    saving.value = false
  }
}

function handlePageChange(value: { page: number; pageSize: number }) {
  pagination.page = value.page
  pagination.pageSize = value.pageSize
  void loadSources()
}

function sourceTypeLabel(row: HistoryStorageSourceItem) {
  if (row.scope.type === 'compute_unit') return ui('计算单元', 'Compute Unit')
  if (row.scope.type === 'collector_connection') {
    const protocol = row.scope.sourceType
      ? formatCollectorProtocolFamily(row.scope.sourceType)
      : ui('未知协议', 'Unknown Protocol')
    return `${ui('工业采集', 'Industrial Collection')} · ${protocol}`
  }
  const labels: Record<string, string> = {
    relational: ui('数据库', 'Database'),
    mqtt: 'MQTT',
    kafka: 'Kafka',
    http: 'HTTP',
    websocket: 'WebSocket',
    redis: 'Redis',
    tdengine: 'TDengine',
    'builtin.relation': ui('IF关系库', 'IF Relational Database'),
    'builtin.timeseries': ui('IF时序库', 'IF Time-series Database'),
    'builtin.realtime': ui('IF实时库', 'IF Realtime Database'),
    'builtin.message': ui('IF消息库', 'IF Message Database'),
  }
  return labels[row.scope.sourceType || ''] || row.scope.sourceType || ui('接入源', 'Access Source')
}

function localizedHistoryStorageSummary(item: HistoryStorageSourceItem) {
  if (datacenterLocale.value !== 'en') return historyStorageSummary(item)
  if (item.historyState !== 'enabled' || !item.writeMode) return 'Not Stored'
  const modes: Record<string, string> = {
    on_change: 'On Change',
    interval_latest: 'At Intervals',
    periodic_snapshot: 'Periodic Snapshot',
    every_sample: 'Every Sample',
  }
  const parts = [modes[item.writeMode] || item.writeMode]
  if (item.primaryTargetName) parts.push(item.primaryTargetName)
  parts.push(item.retentionDays === null ? 'Forever' : `${item.retentionDays || 30} days`)
  if (item.targetCount > 1) parts.push(`${item.targetCount} targets`)
  return parts.join(' · ')
}

watch([scopeType, historyState], () => {
  pagination.page = 1
  void loadSources()
})
watch(search, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    pagination.page = 1
    void loadSources()
  }, 300)
})
onMounted(() => {
  void loadSources()
})
</script>

<style scoped>
.history-storage-workspace {
  height: 100%;
  min-height: 0;
  display: flex;
  color: var(--dc-text);
}

.history-storage-workspace__panel {
  min-width: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.history-storage-workspace__head {
  min-height: 56px;
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

.history-storage-workspace__toolbar {
  width: 100%;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.history-storage-workspace__search {
  width: 240px;
}

.history-storage-workspace__icon-button {
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

.history-storage-workspace__icon-button:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  color: var(--dc-primary);
}

.history-storage-workspace__icon-button:focus-visible,
.history-storage-workspace__settings:focus-visible {
  outline: none;
  border-color: var(--dc-primary);
  box-shadow: 0 0 0 3px rgba(29, 78, 216, 0.12);
}

.history-storage-workspace__icon-button svg {
  width: 16px;
  height: 16px;
}

.history-storage-workspace__total {
  margin-left: auto;
  color: var(--dc-text-muted);
  font-size: 12px;
  white-space: nowrap;
}

.history-storage-workspace__filter-menu {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: -6px -8px;
}

.history-storage-workspace__filter-menu button {
  width: 100%;
  padding: 7px 12px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 13px;
  text-align: left;
}

.history-storage-workspace__filter-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.history-storage-workspace__filter-menu button.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-weight: 700;
}

.history-storage-workspace__content-panel {
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

.history-storage-workspace__table-wrap {
  min-height: 0;
  flex: 1;
  overflow: hidden;
  padding: 12px 24px 0;
}

.history-storage-workspace__table {
  width: 100%;
  height: 100%;
}

.history-storage-workspace__table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.history-storage-workspace__table :deep(.el-table__header th) {
  height: 50px;
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0;
}

.history-storage-workspace__table :deep(.el-table__cell) {
  padding: 8px 0;
  color: var(--dc-text-secondary);
  font-size: 13px;
}

.history-storage-workspace__table :deep(.el-table__row) {
  height: 60px;
}

.history-storage-workspace__table :deep(.el-table__row:hover > td.el-table__cell) {
  background: var(--dc-surface-muted);
}

.history-storage-workspace__identity {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.history-storage-workspace__identity > span {
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  flex: 0 0 auto;
  border-radius: 8px;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.history-storage-workspace__identity > span.is-collector {
  background: rgba(5, 150, 105, 0.1);
  color: #047857;
}
.history-storage-workspace__identity svg {
  width: 17px;
}
.history-storage-workspace__identity div,
.history-storage-workspace__summary {
  min-width: 0;
  display: grid;
  justify-items: start;
  gap: 5px;
}

.history-storage-workspace__identity strong {
  overflow: hidden;
  font-size: 13px;
  letter-spacing: 0;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.history-storage-workspace__identity small,
.history-storage-workspace__summary small {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.history-storage-workspace__point-count {
  color: var(--dc-text-secondary);
  font-variant-numeric: tabular-nums;
}

.history-storage-workspace__settings {
  width: 30px;
  height: 30px;
  display: inline-grid;
  place-items: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.history-storage-workspace__settings:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.history-storage-workspace__settings svg {
  width: 16px;
  height: 16px;
}

@media (max-width: 760px) {
  .history-storage-workspace__head {
    min-height: 104px;
    padding: 10px 14px;
  }

  .history-storage-workspace__search {
    width: 100%;
  }

  .history-storage-workspace__total {
    display: none;
  }

  .history-storage-workspace__table-wrap {
    padding-inline: 12px;
  }
}
</style>
