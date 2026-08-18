<template>
  <div class="history-storage-workspace">
    <section class="history-storage-workspace__panel">
      <header class="history-storage-workspace__head">
        <div class="history-storage-workspace__toolbar">
          <el-input
            v-model="search"
            class="history-storage-workspace__search"
            size="small"
            clearable
            placeholder="搜索来源名称"
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
            title="刷新"
            aria-label="刷新历史存储来源"
            @click="loadSources"
          >
            <IconTablerRefresh />
          </button>

          <span class="history-storage-workspace__total">共 {{ pagination.total }}</span>
        </div>
      </header>

      <section class="history-storage-workspace__content-panel">
        <div v-loading="loading" class="history-storage-workspace__table-wrap">
          <el-table
            :data="sources"
            row-key="scope.id"
            height="100%"
            class="history-storage-workspace__table"
          >
            <el-table-column label="来源" min-width="260">
              <template #default="{ row }">
                <div class="history-storage-workspace__identity">
                  <span
                    :class="
                      row.scope.type === 'collector_connection' ? 'is-collector' : 'is-source'
                    "
                  >
                    <IconTablerCpu v-if="row.scope.type === 'collector_connection'" />
                    <IconTablerRouter v-else />
                  </span>
                  <div>
                    <strong>{{ row.scope.name }}</strong>
                    <small>{{ sourceTypeLabel(row) }}</small>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="数据点" width="110" align="center">
              <template #default="{ row }">
                <span class="history-storage-workspace__point-count">{{ row.datapointCount }}</span>
              </template>
            </el-table-column>
            <el-table-column label="历史存储" min-width="300">
              <template #default="{ row }">
                <div class="history-storage-workspace__summary">
                  <StatusBadge
                    :tone="row.historyState === 'enabled' ? 'success' : 'muted'"
                    :text="historyStorageSummary(row)"
                  />
                  <small v-if="row.pointOverrideCount > 0">
                    {{ row.pointOverrideCount }} 个数据点单独设置
                  </small>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="88" align="right" fixed="right">
              <template #default="{ row }">
                <button
                  type="button"
                  class="history-storage-workspace__settings"
                  title="设置历史存储"
                  :aria-label="`设置 ${row.scope.name} 的历史存储`"
                  @click="openSettings(row)"
                >
                  <IconTablerSettings />
                </button>
              </template>
            </el-table-column>
            <template #empty>
              <el-empty description="暂无匹配的来源" />
            </template>
          </el-table>
        </div>

        <DataCenterPagination
          :page="pagination.page"
          :page-size="pagination.pageSize"
          :total="pagination.total"
          :total-pages="pagination.totalPages"
          @change="handlePageChange"
        />
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
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerRouter from '~icons/tabler/router'
import IconTablerSettings from '~icons/tabler/settings'
import DataCenterPagination from '@/components/shared/DataCenterPagination.vue'
import PillButton from '@/components/shared/PillButton.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
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
const drawerTitle = ref('历史存储设置')
const editingRow = ref<HistoryStorageSourceItem | null>(null)
const editingDetail = ref<HistoryStorageSourceDetail | null>(null)
const { confirm } = useConfirm()
let searchTimer: ReturnType<typeof setTimeout> | null = null

const scopeTypeOptions: Array<{ label: string; value: HistoryStorageScopeType | '' }> = [
  { label: '全部来源', value: '' },
  { label: '接入源', value: 'access_source' },
  { label: '工业采集', value: 'collector_connection' },
]
const historyStateOptions: Array<{ label: string; value: 'enabled' | 'disabled' | '' }> = [
  { label: '全部状态', value: '' },
  { label: '已保存历史', value: 'enabled' },
  { label: '未保存历史', value: 'disabled' },
]
const scopeTypeLabel = computed(
  () => scopeTypeOptions.find((option) => option.value === scopeType.value)?.label || '来源类别',
)
const historyStateLabel = computed(
  () =>
    historyStateOptions.find((option) => option.value === historyState.value)?.label || '历史状态',
)

async function loadSources() {
  loading.value = true
  try {
    const result = await listHistoryStorageSources(props.projectId, {
      page: pagination.page,
      pageSize: pagination.pageSize,
      search: search.value.trim(),
      scopeType: scopeType.value,
      historyState: historyState.value,
    })
    sources.value = result.list
    Object.assign(pagination, result.pagination)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载历史存储来源失败'))
  } finally {
    loading.value = false
  }
}

async function ensureTargets() {
  if (targets.value.length > 0) return
  targets.value = await listHistoryStorageTargets(props.projectId)
}

async function openSettings(row: HistoryStorageSourceItem) {
  try {
    const [detail] = await Promise.all([
      getHistoryStorageSource(props.projectId, row.scope.type, row.scope.id),
      ensureTargets(),
    ])
    editingRow.value = row
    editingDetail.value = detail
    drawerTitle.value = `${row.scope.name} · 历史存储`
    drawerVisible.value = true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载历史存储设置失败'))
  }
}

async function saveSettings(payload: HistoryStorageSavePayload) {
  const row = editingRow.value
  const detail = editingDetail.value
  if (!row || !detail) return
  if (payload.behavior === 'off' && detail.behavior === 'custom' && detail.pointOverrideCount > 0) {
    const accepted = await confirm(
      `关闭来源历史后，仍有 ${detail.pointOverrideCount} 个单点例外按各自设置生效。确认关闭？`,
      { title: '关闭来源历史', confirmText: '确认关闭', type: 'warning' },
    )
    if (!accepted) return
  }
  saving.value = true
  try {
    await saveHistoryStorageSource(props.projectId, row.scope.type, row.scope.id, payload)
    ElMessage.success('历史存储设置已保存')
    drawerVisible.value = false
    await loadSources()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存历史存储设置失败'))
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
  if (row.scope.type === 'collector_connection') {
    const protocol = row.scope.sourceType
      ? formatCollectorProtocolFamily(row.scope.sourceType)
      : '未知协议'
    return `工业采集 · ${protocol}`
  }
  const labels: Record<string, string> = {
    relational: '数据库',
    mqtt: 'MQTT',
    kafka: 'Kafka',
    http: 'HTTP',
    websocket: 'WebSocket',
    redis: 'Redis',
    tdengine: 'TDengine',
    'builtin.relation': 'IF关系库',
    'builtin.timeseries': 'IF时序库',
    'builtin.realtime': 'IF实时库',
    'builtin.message': 'IF消息库',
  }
  return labels[row.scope.sourceType || ''] || row.scope.sourceType || '接入源'
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
