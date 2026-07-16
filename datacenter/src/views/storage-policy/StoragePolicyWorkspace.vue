<template>
  <div class="storage-workspace">
    <header class="storage-workspace__header">
      <div>
        <h2>存储策略</h2>
        <p>历史归档面向统一数据点配置，实时当前值默认进入 IF 实时库。</p>
      </div>
      <div class="storage-workspace__actions">
        <el-button size="small" :icon="Refresh" :loading="loading" @click="reload">刷新</el-button>
        <el-button size="small" type="primary" :icon="Plus" @click="openCreate">新建策略</el-button>
      </div>
    </header>

    <section class="storage-workspace__metrics">
      <div>
        <span>启用策略</span>
        <strong>{{ summary.enabledCount }}</strong>
      </div>
      <div>
        <span>异常策略</span>
        <strong>{{ summary.errorCount }}</strong>
      </div>
      <div>
        <span>绑定点数</span>
        <strong>{{ summary.totalBindingCount }}</strong>
      </div>
      <div>
        <span>预计 rows/day</span>
        <strong>{{ formatNumber(summary.estimatedRowsPerDay) }}</strong>
      </div>
      <div>
        <span>实时当前值</span>
        <strong>IF 实时库</strong>
      </div>
    </section>

    <div class="storage-workspace__body">
      <aside class="storage-workspace__side">
        <div class="storage-workspace__filter">
          <el-input
            v-model="filters.search"
            size="small"
            clearable
            placeholder="搜索策略或目标"
            @keyup.enter="reloadFirstPage"
            @clear="reloadFirstPage"
          />
          <el-select
            v-model="filters.status"
            size="small"
            placeholder="状态"
            clearable
            @change="reloadFirstPage"
          >
            <el-option label="启用" value="enabled" />
            <el-option label="停用" value="disabled" />
            <el-option label="异常" value="error" />
          </el-select>
          <el-select
            v-model="filters.writeMode"
            size="small"
            placeholder="写入模式"
            clearable
            @change="reloadFirstPage"
          >
            <el-option label="每次采样" value="every_sample" />
            <el-option label="变化写入" value="on_change" />
            <el-option label="周期快照" value="periodic_snapshot" />
          </el-select>
          <el-button size="small" @click="reloadFirstPage">筛选</el-button>
        </div>

        <div v-loading="loading" class="storage-workspace__list">
          <button
            v-for="policy in policies"
            :key="policy.id"
            type="button"
            class="storage-policy-item"
            :class="{ 'is-active': selectedPolicy?.id === policy.id }"
            @click="selectPolicy(policy)"
          >
            <span class="storage-policy-item__top">
              <strong>{{ policy.name }}</strong>
              <em :class="`is-${policy.status}`">{{ statusText(policy.status) }}</em>
            </span>
            <span>{{ policy.target.name }} · {{ writeModeText(policy.writeMode) }}</span>
            <span>{{
              policy.bindingMode === 'static'
                ? `${policy.bindingCount} 点`
                : `${policy.estimate.matchedDataPointCount} 点动态命中`
            }}</span>
          </button>
          <div v-if="!loading && policies.length === 0" class="storage-workspace__empty">
            <strong>暂无存储策略</strong>
            <span>先选择历史目标，再把一批数据点纳入归档。</span>
          </div>
        </div>

        <el-pagination
          v-if="pagination.total > pagination.pageSize"
          small
          layout="prev, pager, next"
          :current-page="pagination.page"
          :page-size="pagination.pageSize"
          :total="pagination.total"
          @current-change="changePage"
        />
      </aside>

      <main class="storage-workspace__main">
        <section class="storage-targets">
          <div class="storage-section-title">
            <span>历史目标能力</span>
            <small>仅展示具备历史写入 capability 的接入源</small>
          </div>
          <div class="storage-targets__grid">
            <div v-for="target in targets" :key="target.id" class="storage-target">
              <strong>{{ target.name }}</strong>
              <span>{{ targetTypeText(target.type) }} · {{ target.status }}</span>
              <div>
                <em v-for="cap in target.capabilities" :key="cap">{{ capabilityText(cap) }}</em>
              </div>
            </div>
            <div v-if="targets.length === 0" class="storage-workspace__empty is-inline">
              <strong>暂无可用历史目标</strong>
              <span>可先创建 IF 时序库、TDengine 或关系库接入源。</span>
            </div>
          </div>
        </section>

        <section class="storage-detail">
          <div class="storage-section-title">
            <span>策略详情</span>
            <small>{{ selectedPolicy ? selectedPolicy.name : '选择左侧策略查看' }}</small>
          </div>
          <div v-if="selectedPolicy" class="storage-detail__grid">
            <dl>
              <div>
                <dt>目标</dt>
                <dd>{{ selectedPolicy.target.name }}</dd>
              </div>
              <div>
                <dt>能力</dt>
                <dd>{{ capabilityText(selectedPolicy.targetCapability) }}</dd>
              </div>
              <div>
                <dt>绑定</dt>
                <dd>{{ bindingModeText(selectedPolicy.bindingMode) }}</dd>
              </div>
              <div>
                <dt>模式</dt>
                <dd>{{ writeModeText(selectedPolicy.writeMode) }}</dd>
              </div>
              <div>
                <dt>保留</dt>
                <dd>{{ selectedPolicy.retentionDays }} 天</dd>
              </div>
              <div>
                <dt>质量</dt>
                <dd>{{ selectedPolicy.includeQualities.join(', ') }}</dd>
              </div>
            </dl>
            <div class="storage-estimate">
              <strong>写入预估</strong>
              <span
                >{{ selectedPolicy.estimate.matchedDataPointCount }} 点 ·
                {{ selectedPolicy.estimate.eventsPerSecond.toFixed(2) }} events/s</span
              >
              <span
                >{{ formatNumber(selectedPolicy.estimate.rowsPerDay) }} rows/day · 保留期
                {{ formatNumber(selectedPolicy.estimate.rowsByRetention) }} 行</span
              >
              <small>{{ selectedPolicy.estimate.assumption }}</small>
            </div>
            <div class="storage-diagnostics">
              <strong>诊断</strong>
              <p v-if="selectedPolicy.diagnostics.length === 0">当前没有诊断问题。</p>
              <p
                v-for="item in selectedPolicy.diagnostics"
                :key="`${item.type}-${item.message}`"
                :class="`is-${item.severity}`"
              >
                {{ diagnosticText(item) }}
              </p>
            </div>
            <div class="storage-detail__actions">
              <el-button size="small" :icon="Edit" @click="openEdit(selectedPolicy)"
                >编辑</el-button
              >
              <el-button
                size="small"
                type="danger"
                plain
                :icon="Delete"
                @click="removePolicy(selectedPolicy)"
                >删除</el-button
              >
            </div>
          </div>
          <div v-else class="storage-workspace__empty is-large">
            <strong>选择一个存储策略</strong>
            <span>这里会显示绑定范围、保留策略、目标能力和运行契约预估。</span>
          </div>
        </section>
      </main>
    </div>

    <el-drawer
      v-model="drawerVisible"
      :title="editingId ? '编辑存储策略' : '新建存储策略'"
      size="520px"
    >
      <div class="storage-form">
        <label>
          <span>策略名称</span>
          <el-input v-model="draft.name" placeholder="例如：关键设备秒级归档" />
        </label>
        <label>
          <span>说明</span>
          <el-input v-model="draft.description" type="textarea" :rows="2" placeholder="可选" />
        </label>
        <label>
          <span>历史目标</span>
          <el-select
            v-model="draft.targetConnectionId"
            placeholder="选择存储接入源"
            filterable
            @change="syncCapability"
          >
            <el-option
              v-for="target in targets"
              :key="target.id"
              :label="`${target.name} · ${targetTypeText(target.type)}`"
              :value="target.id"
            />
          </el-select>
        </label>
        <label>
          <span>目标能力</span>
          <el-select v-model="draft.targetCapability" placeholder="写入能力">
            <el-option
              v-for="cap in activeTargetCapabilities"
              :key="cap"
              :label="capabilityText(cap)"
              :value="cap"
            />
          </el-select>
        </label>

        <div class="storage-form__row">
          <label>
            <span>绑定模式</span>
            <el-segmented
              v-model="draft.bindingMode"
              :options="[
                { label: '静态绑定', value: 'static' },
                { label: '动态规则', value: 'dynamic' },
              ]"
            />
          </label>
          <label>
            <span>状态</span>
            <el-switch v-model="draft.enabled" active-text="启用" inactive-text="停用" />
          </label>
        </div>

        <section v-if="draft.bindingMode === 'static'" class="storage-form__block">
          <div class="storage-form__block-title">
            <span>静态数据点</span>
            <el-button size="small" @click="pickerVisible = true">添加数据点</el-button>
          </div>
          <div class="storage-form__chips">
            <el-tag
              v-for="point in selectedDatapoints"
              :key="point.datapointId"
              closable
              @close="removeDatapoint(point.datapointId)"
            >
              {{ point.datapointPath }}
            </el-tag>
            <span v-if="selectedDatapoints.length === 0">尚未选择数据点</span>
          </div>
        </section>

        <section v-else class="storage-form__block">
          <div class="storage-form__block-title">
            <span>动态规则</span>
            <small>新数据点满足条件时自动纳入</small>
          </div>
          <div class="storage-form__grid">
            <el-input v-model="draft.bindingFilter.search" placeholder="名称或 path 关键词" />
            <el-input
              v-model="draft.bindingFilter.type"
              placeholder="来源类型，如 collector.point"
            />
            <el-input v-model="draft.bindingFilter.accessSourceId" placeholder="接入源 ID" />
            <el-input
              v-model="tagInput"
              placeholder="标签，逗号分隔"
              @blur="syncTags"
              @keyup.enter="syncTags"
            />
          </div>
          <el-select v-model="draft.bindingFilter.status" placeholder="数据点状态" clearable>
            <el-option label="正常" value="active" />
            <el-option label="停用" value="inactive" />
            <el-option label="无效" value="invalid" />
          </el-select>
        </section>

        <div class="storage-form__row">
          <label>
            <span>写入模式</span>
            <el-select v-model="draft.writeMode">
              <el-option label="每次采样" value="every_sample" />
              <el-option label="变化写入" value="on_change" />
              <el-option label="周期快照" value="periodic_snapshot" />
            </el-select>
          </label>
          <label>
            <span>保留天数</span>
            <el-input-number
              v-model="draft.retentionDays"
              :min="1"
              :max="3650"
              controls-position="right"
            />
          </label>
        </div>

        <div class="storage-form__row">
          <label>
            <span>最小间隔 ms</span>
            <el-input-number v-model="draft.minIntervalMs" :min="0" controls-position="right" />
          </label>
          <label>
            <span>死区</span>
            <el-input-number v-model="draft.deadband" :min="0" controls-position="right" />
          </label>
        </div>

        <label v-if="draft.writeMode === 'periodic_snapshot'">
          <span>快照周期 ms</span>
          <el-input-number
            v-model="draft.snapshotIntervalMs"
            :min="1000"
            controls-position="right"
          />
        </label>

        <label>
          <span>入库质量</span>
          <el-checkbox-group v-model="draft.includeQualities">
            <el-checkbox label="Good" value="Good" />
            <el-checkbox label="Uncertain" value="Uncertain" />
            <el-checkbox label="Bad" value="Bad" />
          </el-checkbox-group>
        </label>

        <label>
          <span>目标表模式</span>
          <el-select v-model="draft.targetTableMode">
            <el-option label="自动创建" value="auto_create" />
            <el-option label="已有表映射" value="existing_mapping" />
          </el-select>
        </label>

        <div class="storage-form__estimate">
          <el-button size="small" :loading="estimating" @click="refreshEstimate"
            >刷新预估</el-button
          >
          <span v-if="draftEstimate"
            >{{ draftEstimate.matchedDataPointCount }} 点 ·
            {{ formatNumber(draftEstimate.rowsPerDay) }} rows/day</span
          >
        </div>
      </div>

      <template #footer>
        <div class="storage-drawer-footer">
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="saveDraft">保存</el-button>
        </div>
      </template>
    </el-drawer>

    <DatapointPickerDialog
      v-model="pickerVisible"
      :project-id="projectId"
      title="选择归档数据点"
      confirm-text="加入策略"
      row-action-text="加入"
      @select="addDatapoint"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Edit, Plus, Refresh } from '@element-plus/icons-vue'
import DatapointPickerDialog from '@/components/shared/DatapointPickerDialog.vue'
import type { Datapoint } from '@/api/schemas/datapoint.schema'
import {
  createStoragePolicy,
  deleteStoragePolicy,
  estimateStoragePolicy,
  getStoragePolicy,
  listStoragePolicies,
  listStorageTargets,
  updateStoragePolicy,
  type StorageDataPointFilter,
  type StoragePolicy,
  type StoragePolicyBinding,
  type StoragePolicyBindingMode,
  type StoragePolicyEstimate,
  type StoragePolicySavePayload,
  type StoragePolicySummary,
  type StoragePolicyWriteMode,
  type StorageTarget,
  type StorageTargetCapability,
} from '@/api/storage-policy.api'
import { getApiErrorMessage } from '@/utils/request'

const props = defineProps<{
  projectId: string
}>()

type DraftState = {
  name: string
  description: string
  targetConnectionId: string
  targetCapability: StorageTargetCapability | ''
  bindingMode: StoragePolicyBindingMode
  bindingFilter: StorageDataPointFilter
  writeMode: StoragePolicyWriteMode
  minIntervalMs: number | null
  deadband: number | null
  snapshotIntervalMs: number | null
  includeQualities: string[]
  retentionDays: number
  targetTableMode: 'auto_create' | 'existing_mapping'
  enabled: boolean
}

const loading = ref(false)
const saving = ref(false)
const estimating = ref(false)
const drawerVisible = ref(false)
const pickerVisible = ref(false)
const editingId = ref('')
const policies = ref<StoragePolicy[]>([])
const targets = ref<StorageTarget[]>([])
const selectedPolicy = ref<StoragePolicy | null>(null)
const selectedDatapoints = ref<StoragePolicyBinding[]>([])
const draftEstimate = ref<StoragePolicyEstimate | null>(null)
const tagInput = ref('')

const filters = reactive({
  search: '',
  status: '',
  writeMode: '',
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0,
})

const summary = reactive<StoragePolicySummary>({
  enabledCount: 0,
  errorCount: 0,
  totalBindingCount: 0,
  estimatedRowsPerDay: 0,
  estimatedEventsSecond: 0,
})

const draft = reactive<DraftState>(defaultDraft())

const activeTarget = computed(() =>
  targets.value.find((target) => target.id === draft.targetConnectionId),
)

const activeTargetCapabilities = computed<StorageTargetCapability[]>(() =>
  activeTarget.value?.capabilities?.length ? activeTarget.value.capabilities : [],
)

function defaultDraft(): DraftState {
  return {
    name: '',
    description: '',
    targetConnectionId: '',
    targetCapability: '',
    bindingMode: 'dynamic',
    bindingFilter: { status: 'active' },
    writeMode: 'every_sample',
    minIntervalMs: 0,
    deadband: null,
    snapshotIntervalMs: 60000,
    includeQualities: ['Good', 'Uncertain'],
    retentionDays: 90,
    targetTableMode: 'auto_create',
    enabled: true,
  }
}

function resetDraft() {
  Object.assign(draft, defaultDraft())
  selectedDatapoints.value = []
  draftEstimate.value = null
  tagInput.value = ''
  syncCapability()
}

async function reload() {
  if (!props.projectId) return
  loading.value = true
  try {
    const [targetList, result] = await Promise.all([
      listStorageTargets(props.projectId),
      listStoragePolicies(props.projectId, {
        page: pagination.page,
        pageSize: pagination.pageSize,
        search: filters.search,
        status: filters.status,
        writeMode: filters.writeMode,
      }),
    ])
    targets.value = targetList
    policies.value = result.list
    pagination.total = Number(result.pagination?.total ?? result.list.length)
    Object.assign(summary, {
      enabledCount:
        result.summary?.enabledCount ??
        result.list.filter((item) => item.status === 'enabled').length,
      errorCount:
        result.summary?.errorCount ?? result.list.filter((item) => item.status === 'error').length,
      totalBindingCount:
        result.summary?.totalBindingCount ??
        result.list.reduce((total, item) => total + item.bindingCount, 0),
      estimatedRowsPerDay:
        result.summary?.estimatedRowsPerDay ??
        result.list.reduce((total, item) => total + item.estimate.rowsPerDay, 0),
      estimatedEventsSecond:
        result.summary?.estimatedEventsSecond ??
        result.list.reduce((total, item) => total + item.estimate.eventsPerSecond, 0),
    })
    if (selectedPolicy.value) {
      selectedPolicy.value =
        result.list.find((item) => item.id === selectedPolicy.value?.id) || null
    }
    if (!selectedPolicy.value && result.list.length > 0) {
      selectedPolicy.value = result.list[0]
    }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '存储策略加载失败'))
  } finally {
    loading.value = false
  }
}

function reloadFirstPage() {
  pagination.page = 1
  void reload()
}

function changePage(page: number) {
  pagination.page = page
  void reload()
}

function selectPolicy(policy: StoragePolicy) {
  selectedPolicy.value = policy
}

function openCreate() {
  editingId.value = ''
  resetDraft()
  if (targets.value[0]) {
    draft.targetConnectionId = targets.value[0].id
    syncCapability()
  }
  drawerVisible.value = true
}

async function openEdit(policy: StoragePolicy) {
  editingId.value = policy.id
  let detail = policy
  try {
    detail = await getStoragePolicy(props.projectId, policy.id)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '存储策略详情加载失败'))
    return
  }
  selectedPolicy.value = detail
  Object.assign(draft, {
    name: detail.name,
    description: detail.description || '',
    targetConnectionId: detail.target.id,
    targetCapability: detail.targetCapability,
    bindingMode: detail.bindingMode,
    bindingFilter: { ...detail.bindingFilter },
    writeMode: detail.writeMode,
    minIntervalMs: detail.minIntervalMs ?? 0,
    deadband: detail.deadband ?? null,
    snapshotIntervalMs: detail.snapshotIntervalMs ?? 60000,
    includeQualities: [...detail.includeQualities],
    retentionDays: detail.retentionDays,
    targetTableMode: detail.targetTableMode,
    enabled: detail.status !== 'disabled',
  })
  selectedDatapoints.value = 'bindings' in detail ? detail.bindings : []
  tagInput.value = (detail.bindingFilter.tags || []).join(',')
  draftEstimate.value = detail.estimate
  drawerVisible.value = true
}

function syncCapability() {
  const caps = activeTargetCapabilities.value
  if (!caps.length) {
    draft.targetCapability = ''
    return
  }
  if (!draft.targetCapability || !caps.includes(draft.targetCapability)) {
    draft.targetCapability = caps[0]
  }
}

function syncTags() {
  draft.bindingFilter.tags = tagInput.value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

function addDatapoint(point: Datapoint) {
  const id = String(point.id)
  if (selectedDatapoints.value.some((item) => item.datapointId === id)) {
    ElMessage.warning('该数据点已加入策略')
    return
  }
  selectedDatapoints.value = [
    ...selectedDatapoints.value,
    {
      id,
      datapointId: id,
      datapointPath: point.path,
      datapointName: point.name,
      dataType: point.dataType || '',
      status: point.status || 'unknown',
    },
  ]
}

function removeDatapoint(id: string) {
  selectedDatapoints.value = selectedDatapoints.value.filter((item) => item.datapointId !== id)
}

async function refreshEstimate() {
  if (!validateDraft(false)) return
  estimating.value = true
  try {
    draftEstimate.value = await estimateStoragePolicy(props.projectId, buildPayload())
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '存储策略预估失败'))
  } finally {
    estimating.value = false
  }
}

async function saveDraft() {
  if (!validateDraft(true)) return
  saving.value = true
  try {
    const payload = buildPayload()
    const saved = editingId.value
      ? await updateStoragePolicy(props.projectId, editingId.value, payload)
      : await createStoragePolicy(props.projectId, payload)
    selectedPolicy.value = saved
    drawerVisible.value = false
    await reload()
    ElMessage.success('存储策略已保存')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存存储策略失败'))
  } finally {
    saving.value = false
  }
}

async function removePolicy(policy: StoragePolicy) {
  const ok = await ElMessageBox.confirm(
    `确认删除存储策略「${policy.name}」？不会删除目标库中的历史数据或目标表。`,
    '删除存储策略',
    {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    },
  )
    .then(() => true)
    .catch(() => false)
  if (!ok) return
  await deleteStoragePolicy(props.projectId, policy.id)
  selectedPolicy.value = null
  await reload()
  ElMessage.success('存储策略已删除')
}

function validateDraft(requireBinding: boolean) {
  if (!draft.name.trim()) {
    ElMessage.warning('请输入策略名称')
    return false
  }
  if (!draft.targetConnectionId) {
    ElMessage.warning('请选择历史目标')
    return false
  }
  if (!draft.targetCapability) {
    ElMessage.warning('请选择目标写入能力')
    return false
  }
  if (draft.bindingMode === 'static' && requireBinding && selectedDatapoints.value.length === 0) {
    ElMessage.warning('静态绑定至少需要一个数据点')
    return false
  }
  if (draft.retentionDays < 1) {
    ElMessage.warning('保留天数必须大于 0')
    return false
  }
  if (draft.writeMode === 'periodic_snapshot' && !draft.snapshotIntervalMs) {
    ElMessage.warning('周期快照需要快照周期')
    return false
  }
  return true
}

function buildPayload(): StoragePolicySavePayload {
  syncTags()
  return {
    name: draft.name.trim(),
    description: draft.description.trim() || null,
    targetConnectionId: draft.targetConnectionId,
    targetCapability: draft.targetCapability,
    bindingMode: draft.bindingMode,
    bindingFilter: draft.bindingMode === 'dynamic' ? { ...draft.bindingFilter } : {},
    datapointIds:
      draft.bindingMode === 'static'
        ? selectedDatapoints.value.map((item) => item.datapointId)
        : [],
    writeMode: draft.writeMode,
    minIntervalMs: draft.minIntervalMs ?? null,
    deadband: draft.deadband ?? null,
    snapshotIntervalMs: draft.writeMode === 'periodic_snapshot' ? draft.snapshotIntervalMs : null,
    includeQualities: draft.includeQualities,
    retentionDays: draft.retentionDays,
    targetTableMode: draft.targetTableMode,
    targetTableConfig: {},
    status: draft.enabled ? 'enabled' : 'disabled',
  }
}

const statusText = (status: string) =>
  ({ enabled: '启用', disabled: '停用', error: '异常' })[status] || status

const writeModeText = (mode: string) =>
  ({ every_sample: '每次采样', on_change: '变化写入', periodic_snapshot: '周期快照' })[mode] || mode

const bindingModeText = (mode: string) =>
  ({ static: '静态绑定', dynamic: '动态规则' })[mode] || mode

const capabilityText = (capability: string) =>
  ({ timeseriesAppend: '时序追加', relationalAppend: '关系库追加' })[capability] || capability

const targetTypeText = (type: string) =>
  ({
    'builtin.timeseries': 'IF时序库',
    tdengine: 'TDengine',
    relational: '关系库',
    'builtin.relation': 'IF关系库',
  })[type] || type

const diagnosticText = (item: { message: string; suggest?: string }) =>
  item.suggest ? `${item.message}：${item.suggest}` : item.message

const formatNumber = (value: number) =>
  new Intl.NumberFormat('zh-CN').format(Math.round(value || 0))

watch(
  () => props.projectId,
  () => {
    selectedPolicy.value = null
    void reload()
  },
)

onMounted(reload)
</script>

<style scoped>
.storage-workspace {
  height: calc(100vh - 32px);
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  color: var(--dc-text);
}

.storage-workspace__header,
.storage-workspace__metrics,
.storage-workspace__body,
.storage-targets,
.storage-detail {
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.storage-workspace__header {
  min-height: 60px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 14px;
}

.storage-workspace__header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 900;
}

.storage-workspace__header p {
  margin: 4px 0 0;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.storage-workspace__actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.storage-workspace__metrics {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  overflow: hidden;
}

.storage-workspace__metrics div {
  min-width: 0;
  display: grid;
  gap: 2px;
  padding: 10px 12px;
  border-right: 1px solid var(--dc-border);
}

.storage-workspace__metrics div:last-child {
  border-right: 0;
}

.storage-workspace__metrics span,
.storage-section-title small,
.storage-policy-item span,
.storage-target span,
.storage-estimate small,
.storage-workspace__empty span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.storage-workspace__metrics strong {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 17px;
  font-weight: 900;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.storage-workspace__body {
  min-height: 0;
  flex: 1 1 auto;
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  overflow: hidden;
}

.storage-workspace__side {
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 10px;
  border-right: 1px solid var(--dc-border);
}

.storage-workspace__filter {
  display: grid;
  gap: 8px;
}

.storage-workspace__list {
  min-height: 0;
  flex: 1 1 auto;
  display: grid;
  align-content: start;
  gap: 8px;
  overflow: auto;
}

.storage-policy-item {
  width: 100%;
  display: grid;
  gap: 5px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  cursor: pointer;
  text-align: left;
}

.storage-policy-item:hover,
.storage-policy-item.is-active {
  border-color: rgba(29, 78, 216, 0.36);
  background: var(--dc-primary-soft);
}

.storage-policy-item__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.storage-policy-item strong {
  min-width: 0;
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.storage-policy-item em {
  flex: 0 0 auto;
  padding: 2px 7px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 11px;
  font-style: normal;
  font-weight: 900;
}

.storage-policy-item em.is-enabled {
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.storage-policy-item em.is-error {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.storage-workspace__main {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  gap: 10px;
  padding: 10px;
  overflow: auto;
}

.storage-targets,
.storage-detail {
  padding: 12px;
}

.storage-section-title {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.storage-section-title span {
  font-size: 14px;
  font-weight: 900;
}

.storage-targets__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 8px;
}

.storage-target {
  display: grid;
  gap: 5px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
}

.storage-target strong {
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.storage-target div {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.storage-target em {
  padding: 2px 7px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-size: 11px;
  font-style: normal;
  font-weight: 900;
}

.storage-detail__grid {
  display: grid;
  grid-template-columns: minmax(260px, 0.9fr) minmax(260px, 1fr);
  gap: 12px;
}

.storage-detail dl {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.storage-detail dl div,
.storage-estimate,
.storage-diagnostics {
  min-width: 0;
  display: grid;
  gap: 4px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
}

.storage-detail dt {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.storage-detail dd {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  font-size: 13px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.storage-estimate strong,
.storage-diagnostics strong {
  font-size: 13px;
}

.storage-estimate span,
.storage-diagnostics p {
  margin: 0;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.storage-diagnostics p.is-warning {
  color: var(--dc-warning);
}

.storage-diagnostics p.is-error {
  color: var(--dc-danger);
}

.storage-detail__actions {
  grid-column: 1 / -1;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.storage-workspace__empty {
  display: grid;
  gap: 5px;
  padding: 12px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-sm);
  color: var(--dc-text-secondary);
}

.storage-workspace__empty.is-inline,
.storage-workspace__empty.is-large {
  align-content: center;
  min-height: 96px;
}

.storage-form {
  display: grid;
  gap: 14px;
}

.storage-form label {
  min-width: 0;
  display: grid;
  gap: 6px;
}

.storage-form label > span,
.storage-form__block-title span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 900;
}

.storage-form__row,
.storage-form__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.storage-form__block {
  display: grid;
  gap: 10px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
}

.storage-form__block-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.storage-form__block-title small {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.storage-form__chips {
  min-height: 32px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.storage-form__estimate {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.storage-drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

@media (max-width: 980px) {
  .storage-workspace__metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .storage-workspace__body {
    grid-template-columns: 1fr;
  }

  .storage-workspace__side {
    border-right: 0;
    border-bottom: 1px solid var(--dc-border);
  }

  .storage-detail__grid,
  .storage-form__row,
  .storage-form__grid {
    grid-template-columns: 1fr;
  }
}
</style>
