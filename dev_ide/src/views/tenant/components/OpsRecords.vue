<template>
  <div class="ops-records ck-workbench-page">
    <div class="records-toolbar ck-workbench-toolbar">
      <div class="records-primary">
        <div class="records-switcher"><slot name="switcher" /></div>
        <el-input
          v-model="filters.search"
          class="records-search ck-toolbar-search rounded-full-input"
          size="small"
          clearable
          placeholder="搜索对象或操作"
          aria-label="搜索对象或操作"
        />
        <div class="records-actions">
          <OpsRealtimeIndicator :status="connection" />
          <el-button size="small" :disabled="loading" @click="monitor.refresh()">刷新</el-button>
        </div>
      </div>
      <div class="records-filters" role="group" aria-label="运维记录筛选">
        <div class="records-filter">
          <span>类型</span>
          <el-select
            v-model="filters.recordType"
            clearable
            placeholder="全部类型"
            aria-label="记录类型"
            ><el-option label="操作记录" value="operation" /><el-option
              label="系统事件"
              value="event"
          /></el-select>
        </div>
        <div class="records-filter">
          <span>对象</span>
          <el-select
            v-model="filters.objectType"
            clearable
            placeholder="全部对象"
            aria-label="对象类型"
            @change="filters.objectId = ''"
            ><el-option
              v-for="(label, value) in objectLabels"
              :key="value"
              :label="label"
              :value="value"
          /></el-select>
        </div>
        <div class="records-filter">
          <span>状态</span>
          <el-select v-model="filters.status" clearable placeholder="全部状态" aria-label="记录状态"
            ><el-option
              v-for="(label, value) in statusLabels"
              :key="value"
              :label="label"
              :value="value"
          /></el-select>
        </div>
        <div class="records-filter">
          <span>环境</span>
          <el-tooltip
            content="环境筛选仅包含有明确历史环境关联的记录；未关联的节点/中心系统事件可在全部环境中查看。"
            ><el-select
              v-model="filters.environmentId"
              clearable
              placeholder="全部环境"
              aria-label="记录环境"
              ><el-option
                v-for="environment in environments"
                :key="environment.id"
                :label="environment.name"
                :value="environment.id" /></el-select
          ></el-tooltip>
        </div>
        <div class="records-filter records-filter--time">
          <span>时间</span>
          <el-date-picker
            v-model="range"
            type="datetimerange"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
            :clearable="true"
            aria-label="记录时间范围"
          />
        </div>
      </div>
      <div v-if="filters.objectId" class="records-context">
        <span>当前对象</span>
        <el-tag size="small" closable @close="filters.objectId = ''">{{
          objectName || '指定对象'
        }}</el-tag>
      </div>
    </div>
    <div class="records-content ck-content-area">
      <div
        v-if="connection === 'stale' && loaded && rows.length"
        class="records-notice"
        role="status"
      >
        <span>更新暂未完成，当前保留上次记录。</span
        ><button type="button" @click="monitor.refresh()">重试</button>
      </div>
      <div class="ck-content-scroll records-scroll">
        <p v-if="loading && !loaded" class="records-empty" role="status">正在加载运维记录…</p>
        <p v-else-if="!rows.length" class="records-empty">
          {{ connection === 'stale' ? '记录加载中断，请重试' : '没有符合筛选条件的运维记录' }}
        </p>
        <div v-if="rows.length" class="ck-table-shell records-table-shell">
          <table :aria-busy="loading" aria-label="运维记录">
            <thead>
              <tr>
                <th>操作 / 事件</th>
                <th>对象</th>
                <th>发起人</th>
                <th :aria-sort="sort === 'time_desc' ? 'descending' : 'ascending'">
                  <button
                    type="button"
                    aria-label="切换记录时间排序"
                    @click="sort = sort === 'time_desc' ? 'time_asc' : 'time_desc'"
                  >
                    时间 {{ sort === 'time_desc' ? '↓' : '↑' }}
                  </button>
                </th>
                <th>耗时</th>
                <th>结果</th>
              </tr>
            </thead>
            <tbody>
              <template v-for="row in rows" :key="row.id">
                <tr class="record-row" :class="{ 'is-expanded': expandedId === row.id }">
                  <td>
                    <button
                      type="button"
                      class="record-expand"
                      :aria-expanded="expandedId === row.id"
                      @click="expandedId = expandedId === row.id ? '' : row.id"
                    >
                      {{ expandedId === row.id ? '▾' : '▸' }}
                      {{ opsBusinessText(row.title) }}</button
                    ><small>{{ row.recordType === 'operation' ? '操作记录' : '系统事件' }}</small>
                  </td>
                  <td>
                    {{ row.objectName || objectLabels[row.objectType]
                    }}<small>{{ objectLabels[row.objectType] }}</small>
                  </td>
                  <td class="record-actor">{{ row.actorDisplayName || '未知用户' }}</td>
                  <td class="record-time">{{ formatDateTime(row.time) }}</td>
                  <td>
                    {{
                      row.completedAt
                        ? runDurationLabel({
                            id: row.id,
                            completedAt: row.completedAt || undefined,
                            startedAt: row.time,
                            durationMs: row.durationMs,
                          })
                        : '—'
                    }}
                  </td>
                  <td>
                    <span class="record-status" :class="'record-status-' + row.status">{{
                      statusLabels[row.status]
                    }}</span>
                  </td>
                </tr>
                <tr v-if="expandedId === row.id">
                  <td colspan="6" class="record-detail">
                    <div class="record-detail-heading">
                      <strong>{{ row.taskRef?.runId ? '执行明细' : '记录详情' }}</strong
                      ><span>{{ row.objectName || objectLabels[row.objectType] }}</span>
                    </div>
                    <DeploymentRunEvents
                      v-if="row.taskRef?.runId"
                      :key="row.taskRef.runId"
                      :run-id="row.taskRef.runId"
                      :active="active"
                      :revision="revision"
                      :live="row.status === 'running' || row.status === 'accepted'"
                    />
                    <p v-else-if="row.message">
                      <OpsMessage
                        :text="row.message"
                        :kind="
                          ['failed', 'warning'].includes(row.status)
                            ? 'error'
                            : row.status === 'running' || row.status === 'accepted'
                              ? 'progress'
                              : 'info'
                        "
                      />
                    </p>
                    <p v-else-if="!row.taskRef && !row.detailUnavailableReason">无补充信息</p>
                    <small v-if="row.detailUnavailableReason">{{
                      row.detailUnavailableReason
                    }}</small>
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
      </div>
      <div class="ck-pagination-bar">
        <span v-if="loading && loaded" class="records-updating" role="status">正在更新…</span
        ><WorkbenchPagination v-model:page="page" v-model:limit="limit" :total="total" />
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, computed, watch, onBeforeUnmount } from 'vue'
import { opsAPI, type OpsRecord, type OpsRecordQuery } from '@/api/ops.api'
import { createOpsRealtimeMonitor } from '@/utils/ops-realtime'
import { formatDateTime } from '@/utils/date'
import WorkbenchPagination from '@/components/WorkbenchPagination.vue'
import OpsRealtimeIndicator from './OpsRealtimeIndicator.vue'
import DeploymentRunEvents from './DeploymentRunEvents.vue'
import OpsMessage from './OpsMessage.vue'
import { opsBusinessText } from '../utils/ops-business'
import { runDurationLabel } from '../utils/deployment-details'
const props = defineProps<{
  active: boolean
  environmentId?: string
  objectId?: string
  objectType?: string
  objectName?: string
  environments: { id: string; name: string }[]
}>()
const objectLabels = {
  project: '工程',
  deployment: '工程部署',
  foundation: '基础服务',
  environment: '运行环境',
  node: '物理节点',
  cluster: '中心系统',
}
const statusLabels = {
  accepted: '已受理',
  running: '执行中',
  success: '成功',
  failed: '失败',
  warning: '异常',
  recovered: '已恢复',
  info: '记录',
}
const filters = reactive({
  search: '',
  recordType: '',
  objectType: props.objectType || '',
  objectId: props.objectId || '',
  status: '',
  environmentId: props.environmentId || '',
})
const range = ref<string[] | null>(null),
  sort = ref<'time_desc' | 'time_asc'>('time_desc')
const page = ref(1),
  limit = ref(10),
  total = ref(0),
  rows = ref<OpsRecord[]>([]),
  expandedId = ref(''),
  revision = ref(0)
const loading = ref(false),
  loaded = ref(false)
const connection = ref<'connecting' | 'online' | 'offline' | 'stale' | 'forbidden'>('connecting')
let controller: AbortController | undefined,
  disposed = false
const query = computed<OpsRecordQuery>(() => ({
  page: page.value,
  limit: limit.value,
  ...Object.fromEntries(Object.entries(filters).filter(([, value]) => value)),
  from: range.value?.[0],
  to: range.value?.[1],
  sort: sort.value,
}))
const queryKey = computed(() => JSON.stringify(query.value))
const monitor = createOpsRealtimeMonitor({
  status: (status) => {
    connection.value = status
  },
  async snapshot(_, signal) {
    if (!props.active) return
    const current = (controller = new AbortController()),
      key = queryKey.value
    const cancel = () => current.abort()
    signal.addEventListener('abort', cancel, { once: true })
    loading.value = true
    try {
      const result = await opsAPI.listRecords(query.value, current.signal)
      if (
        disposed ||
        current.signal.aborted ||
        signal.aborted ||
        key !== queryKey.value ||
        !props.active
      )
        return
      total.value = result.total
      const last = Math.max(1, Math.ceil(total.value / limit.value))
      if (page.value > last) {
        page.value = last
        return
      }
      rows.value = result.items
      loaded.value = true
      revision.value++
      if (!rows.value.some((row) => row.id === expandedId.value)) expandedId.value = ''
    } catch (error) {
      if (!current.signal.aborted) throw error
    } finally {
      signal.removeEventListener('abort', cancel)
      if (controller === current) loading.value = false
    }
  },
})
watch(
  () => props.environmentId,
  (value) => {
    filters.environmentId = value || ''
  },
)
watch(
  [filters, range, sort, limit],
  () => {
    page.value = 1
  },
  { deep: true },
)
watch(queryKey, () => {
  controller?.abort()
  expandedId.value = ''
  if (props.active) void monitor.refresh()
})
watch(
  () => [props.objectType, props.objectId],
  ([type, id]) => {
    filters.objectType = type || ''
    filters.objectId = id || ''
  },
)
watch(
  () => props.active,
  (active) => {
    if (!active) controller?.abort()
    monitor.setScope(active ? { topics: ['deployments', 'events'] } : null)
  },
  { immediate: true },
)
onBeforeUnmount(() => {
  disposed = true
  controller?.abort()
  monitor.dispose()
})
</script>
<style scoped>
.ops-records {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  min-width: 0;
  gap: 12px;
  font-size: 13px;
}
.records-toolbar {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  flex-shrink: 0;
  gap: 12px;
  padding: 12px 16px;
  background: var(--ck-bg-secondary, var(--el-bg-color));
  border: 1px solid var(--ck-border-light, var(--el-border-color-lighter));
  border-radius: var(--ck-radius-md, 10px);
}
.records-primary {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.records-search {
  width: 260px;
  max-width: 100%;
}
.records-switcher:empty {
  display: none;
}
.records-filters {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 12px;
  padding-top: 10px;
  border-top: 1px solid var(--ck-border-light, var(--el-border-color-lighter));
}
.records-filter {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.records-filter > span,
.records-context > span {
  flex-shrink: 0;
  color: var(--ck-text-secondary, var(--el-text-color-secondary));
  font-size: 12px;
}
.records-filter :deep(.el-select) {
  width: 112px;
}
.records-filter--time {
  flex: 0 1 342px;
}
.records-filter--time :deep(.el-date-editor) {
  width: 310px;
  min-width: 0;
  max-width: 100%;
}
.records-toolbar :deep(.el-input__inner),
.records-toolbar :deep(.el-select__placeholder),
.records-toolbar :deep(.el-range-input) {
  font-size: 12px;
}
.records-context {
  display: flex;
  align-items: center;
  gap: 8px;
}
.records-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 8px;
}
.records-content {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.records-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
}
.ck-pagination-bar {
  flex-shrink: 0;
}
.records-notice {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 10px 16px;
  font-size: 12px;
  color: var(--el-color-warning-dark-2);
  background: var(--el-color-warning-light-9);
}
.records-table-shell {
  overflow: visible;
}
table {
  width: 100%;
  min-width: 860px;
  border-collapse: collapse;
  table-layout: fixed;
  font-size: 13px;
}
th,
td {
  padding: 12px 14px;
  text-align: left;
  border-bottom: 1px solid var(--ck-border-light, var(--el-border-color-lighter));
  overflow-wrap: anywhere;
}
th {
  position: sticky;
  top: 0;
  z-index: 1;
  color: var(--ck-text-secondary, var(--el-text-color-secondary));
  font-weight: 600;
  background: var(--ck-bg-tertiary, var(--el-fill-color-light));
}
th:first-child {
  width: 24%;
}
th:nth-child(2) {
  width: 20%;
}
th:nth-child(3) {
  width: 14%;
}
th:nth-child(4) {
  width: 19%;
}
.record-row:hover,
.record-row.is-expanded {
  background: var(--ck-bg-tertiary, var(--el-fill-color-light));
}
.record-time {
  font-size: 12px;
  color: var(--ck-text-secondary, var(--el-text-color-secondary));
  font-variant-numeric: tabular-nums;
}
.record-expand {
  line-height: 1.6;
  font-weight: 500;
}
button:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: 3px;
  border-radius: 3px;
}
button {
  border: 0;
  padding: 0;
  background: none;
  color: var(--ck-primary, var(--el-color-primary));
  font: inherit;
  cursor: pointer;
  text-align: left;
}
small {
  display: block;
  margin-top: 4px;
  color: var(--ck-text-muted, var(--el-text-color-secondary));
  font-size: 12px;
}
.record-detail {
  padding: 16px 20px 20px;
  background: var(--ck-bg-tertiary, var(--el-fill-color-light));
}
.record-detail-heading {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}
.record-detail-heading strong {
  font-size: 13px;
  font-weight: 600;
}
.record-detail-heading span {
  font-size: 12px;
  color: var(--ck-text-secondary, var(--el-text-color-secondary));
}
.records-empty {
  padding: 32px;
  text-align: center;
  color: var(--ck-text-muted, var(--el-text-color-secondary));
}
.records-updating {
  font-size: 12px;
  color: var(--ck-text-muted, var(--el-text-color-secondary));
}
.record-status-failed,
.record-status-warning {
  color: var(--el-color-danger);
}
.record-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
  font-size: 12px;
}
.record-status::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}
.record-status-info {
  color: var(--ck-text-secondary, var(--el-text-color-secondary));
}
.record-status-success,
.record-status-recovered {
  color: var(--el-color-success);
}
.record-status-running,
.record-status-accepted {
  color: var(--el-color-primary);
}
@media (max-width: 640px) {
  .records-toolbar {
    padding: 10px 12px;
  }
  .records-search {
    flex: 1 1 180px;
    width: auto;
  }
  .records-filter--time {
    flex-basis: 100%;
  }
  .records-filter--time :deep(.el-date-editor) {
    flex: 1;
    width: auto;
  }
}
</style>
