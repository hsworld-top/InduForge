<template>
  <el-drawer
    :model-value="modelValue"
    :title="drawerTitle"
    size="min(620px, 92vw)"
    append-to-body
    destroy-on-close
    @close="emit('update:modelValue', false)"
  >
    <div v-if="mode === 'detail' && point" class="collector-point-drawer__detail">
      <div class="collector-point-drawer__hero">
        <div>
          <h3>{{ point.name }}</h3>
          <p>{{ point.description || '暂无变量说明' }}</p>
        </div>
        <el-tag :type="point.enabled ? 'success' : 'info'">
          {{ point.enabled ? '已启用' : '已停用' }}
        </el-tag>
      </div>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="所属分组">{{
          groupLabel(point.groupId)
        }}</el-descriptions-item>
        <el-descriptions-item label="变量编码">
          <code>{{ point.code }}</code>
        </el-descriptions-item>
        <el-descriptions-item label="变量地址"
          ><code>{{ point.addressText }}</code></el-descriptions-item
        >
        <el-descriptions-item label="数据类型">{{ point.dataType }}</el-descriptions-item>
        <el-descriptions-item v-if="supportsElementCount" label="元素数量">{{
          point.elementCount
        }}</el-descriptions-item>
        <el-descriptions-item label="采集周期">{{ intervalLabel(point) }}</el-descriptions-item>
      </el-descriptions>
      <section class="collector-point-drawer__debug">
        <div class="collector-point-drawer__section-title">
          <span>最近调试结果</span>
        </div>
        <template v-if="point.latestDebugSnapshot">
          <el-alert
            v-if="point.latestDebugSnapshot.lastAttemptStatus === 'failed'"
            :title="`${collectorDebugFailureText(point.latestDebugSnapshot)} · ${point.latestDebugSnapshot.lastAttemptAt}`"
            type="warning"
            show-icon
            :closable="false"
          />
          <div class="collector-point-drawer__debug-value">
            <span>最近成功值</span>
            <pre>{{
              hasCollectorDebugSuccess(point.latestDebugSnapshot)
                ? formatCollectorDebugValue(
                    point.latestDebugSnapshot.value,
                    point.latestDebugSnapshot.valueText,
                  )
                : '—'
            }}</pre>
          </div>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="质量">
              <el-tag
                v-if="point.latestDebugSnapshot.quality"
                :type="collectorDebugQualityTone(point.latestDebugSnapshot.quality)"
                size="small"
              >
                {{ collectorDebugQualityLabel(point.latestDebugSnapshot.quality) }}
              </el-tag>
              <span v-else>—</span>
            </el-descriptions-item>
            <el-descriptions-item label="实际数据类型">{{
              point.latestDebugSnapshot.dataType || '—'
            }}</el-descriptions-item>
            <el-descriptions-item label="数据时间">{{
              collectorDebugTime(point.latestDebugSnapshot.sourceTimestamp)
            }}</el-descriptions-item>
            <el-descriptions-item label="服务端时间">{{
              collectorDebugTime(point.latestDebugSnapshot.serverTimestamp)
            }}</el-descriptions-item>
            <el-descriptions-item label="平台获取时间">{{
              collectorDebugTime(point.latestDebugSnapshot.readAt)
            }}</el-descriptions-item>
            <el-descriptions-item label="最近尝试">
              {{ point.latestDebugSnapshot.lastAttemptStatus === 'succeeded' ? '成功' : '失败' }}
              · {{ point.latestDebugSnapshot.lastAttemptAt }}
            </el-descriptions-item>
          </el-descriptions>
        </template>
        <el-empty v-else description="暂未获取调试数据" :image-size="72" />
      </section>
      <section class="collector-point-drawer__json">
        <span>协议地址参数</span>
        <pre>{{ formatJson(point.address) }}</pre>
      </section>
    </div>

    <el-form v-else label-position="top" class="collector-point-drawer__form">
      <section class="collector-point-drawer__section">
        <div class="collector-point-drawer__section-title">
          <span>基础信息</span>
        </div>
        <div class="collector-point-drawer__grid">
          <el-form-item label="变量名称" required>
            <el-input v-model="draft.name" maxlength="200" placeholder="例如 入口温度" />
          </el-form-item>
          <el-form-item label="所属分组">
            <el-tree-select
              v-model="draft.groupId"
              :data="groups"
              node-key="id"
              :props="{ label: 'name', children: 'children' }"
              check-strictly
              clearable
              default-expand-all
              placeholder="未分组"
            />
          </el-form-item>
          <el-form-item label="数据类型" required>
            <el-select v-model="draft.dataType" :disabled="sourceLocked" filterable>
              <el-option v-for="item in dataTypes" :key="item" :label="item" :value="item" />
            </el-select>
          </el-form-item>
          <el-form-item label="类型转换（暂未开放）">
            <el-select model-value="none" disabled>
              <el-option label="保持源类型" value="none" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="supportsElementCount" label="元素数量" required>
            <el-input-number v-model="draft.elementCount" :min="1" :max="65535" />
          </el-form-item>
        </div>
        <el-form-item label="变量说明">
          <el-input v-model="draft.description" type="textarea" :rows="3" maxlength="500" />
        </el-form-item>
      </section>

      <section class="collector-point-drawer__section">
        <div class="collector-point-drawer__section-title"><span>采集参数</span></div>
        <el-radio-group v-model="draft.acquisitionMode">
          <el-radio-button value="inherit">继承连接默认</el-radio-button>
          <el-radio-button value="override">单独覆盖</el-radio-button>
        </el-radio-group>
        <p class="collector-point-drawer__inherit-hint">
          {{
            draft.acquisitionMode === 'inherit'
              ? `当前继承：${acquisitionSummary(defaultAcquisition)}`
              : '只保存与连接默认不同的字段'
          }}
        </p>
        <div v-if="draft.acquisitionMode === 'override'" class="collector-point-drawer__grid">
          <el-form-item label="采集周期 (ms)"
            ><el-input-number v-model="draft.intervalMs" :min="1"
          /></el-form-item>
          <el-form-item label="读取超时 (ms)"
            ><el-input-number v-model="draft.timeoutMs" :min="1"
          /></el-form-item>
          <el-form-item label="失败重试"
            ><el-input-number v-model="draft.retryCount" :min="0"
          /></el-form-item>
          <el-form-item label="数值死区"
            ><el-input-number v-model="draft.deadband" :min="0"
          /></el-form-item>
          <el-form-item label="优先级"
            ><el-input-number v-model="draft.priority" :min="0"
          /></el-form-item>
          <el-form-item label="仅变化时采集"><el-switch v-model="draft.changeOnly" /></el-form-item>
        </div>
      </section>

      <section class="collector-point-drawer__section">
        <div class="collector-point-drawer__section-title">
          <span>协议地址</span>
          <el-tag v-if="addressHelperLabel" size="small" type="primary">{{
            addressHelperLabel
          }}</el-tag>
          <el-tag v-if="sourceLocked" size="small" type="info">设备识别 · 只读</el-tag>
        </div>
        <CollectorSchemaForm
          v-if="driver"
          v-model="draft.address"
          :schema="driver.addressSchema"
          :ui-schema="driver.uiSchema"
          section="address"
          :disabled="sourceLocked"
          :visible-field-names="pointFormState.visibleAddressFields"
          :required-field-names="pointFormState.requiredAddressFields"
        />
        <CollectorAddressHelper
          v-if="driver"
          v-model="draft.address"
          :helper="String(driver.uiSchema.addressHelper || '')"
          :schema="driver.addressSchema"
        />
        <el-skeleton v-else :rows="3" animated />
        <div class="collector-point-drawer__address-check">
          <el-button :loading="normalizing" @click="normalizeAddress">校验并规范化</el-button>
          <span v-if="normalizedAddressText"
            >规范地址：<code>{{ normalizedAddressText }}</code></span
          >
        </div>
      </section>

      <section class="collector-point-drawer__switch-row">
        <span>启用变量</span>
        <el-switch v-model="draft.enabled" />
      </section>
    </el-form>

    <template #footer>
      <template v-if="mode === 'detail'">
        <el-button @click="emit('update:modelValue', false)">关闭</el-button>
        <el-button type="primary" @click="mode = 'edit'">编辑变量</el-button>
      </template>
      <template v-else>
        <el-button @click="emit('update:modelValue', false)">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存变量</el-button>
      </template>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getApiErrorMessage } from '@/utils/request'
import {
  createCollectorPointsBatch,
  getCollectorDriver,
  normalizeCollectorPointAddress,
  updateCollectorPointsBatch,
} from '@/api/collector.api'
import type {
  CollectorDriverDetail,
  CollectorPoint,
  CollectorPointGroup,
} from '@/api/schemas/collector.schema'
import {
  collectorDriverFeatures,
  collectorDriverSupportsFeature,
  type CollectorPointCreateDefaults,
} from './collector-workbench-model'
import CollectorSchemaForm from './CollectorSchemaForm.vue'
import { resolveCollectorPointFormState } from './collector-point-form-rules'
import CollectorAddressHelper from './CollectorAddressHelper.vue'
import {
  collectorDebugFailureText,
  collectorDebugQualityLabel,
  collectorDebugQualityTone,
  collectorDebugTime,
  formatCollectorDebugValue,
  hasCollectorDebugSuccess,
} from './collector-debug-snapshot'

export type CollectorPointGroupNode = CollectorPointGroup & { children?: CollectorPointGroupNode[] }

type DrawerMode = 'create' | 'edit' | 'detail'
const props = defineProps<{
  modelValue: boolean
  initialMode: DrawerMode
  projectId: string
  connectionId: string
  driverId: string
  point?: CollectorPoint | null
  groups: CollectorPointGroupNode[]
  defaultGroupId?: string | null
  defaultAcquisition: Record<string, unknown>
  createDefaults?: CollectorPointCreateDefaults | null
  sourceLocked?: boolean
}>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: []
}>()

const mode = ref<DrawerMode>('create')
const driver = ref<CollectorDriverDetail>()
const saving = ref(false)
const normalizing = ref(false)
const normalizedAddressText = ref('')
const draft = reactive({
  groupId: null as string | null,
  name: '',
  description: '',
  dataType: 'float32',
  elementCount: 1,
  intervalMs: 1000,
  timeoutMs: 3000,
  retryCount: 0,
  deadband: 0,
  changeOnly: false,
  priority: 0,
  acquisitionMode: 'inherit' as 'inherit' | 'override',
  enabled: true,
  address: {} as Record<string, unknown>,
})
const drawerTitle = computed(() =>
  mode.value === 'create' ? '新建变量' : mode.value === 'edit' ? '编辑变量' : '变量详情',
)
const driverDataTypes = computed(() => driver.value?.dataTypes || [])
const pointFormState = computed(() =>
  resolveCollectorPointFormState({
    driverId: props.driverId,
    addressHelper: String(driver.value?.uiSchema.addressHelper || ''),
    address: draft.address,
    dataType: draft.dataType,
    driverDataTypes: driverDataTypes.value,
  }),
)
const dataTypes = computed(() => pointFormState.value.allowedDataTypes)
const supportsElementCount = computed(() =>
  collectorDriverSupportsFeature(driver.value, collectorDriverFeatures.pointElementCount),
)
const defaultAcquisition = computed(() => props.defaultAcquisition || {})
const addressHelperLabel = computed(() => {
  const helper = String(driver.value?.uiSchema.addressHelper || '')
  return (
    (
      {
        siemens: '西门子地址助手',
        modbus: 'Modbus 地址助手',
        melsec: '三菱地址助手',
        omron: '欧姆龙地址助手',
        allen_bradley: '罗克韦尔标签助手',
      } as Record<string, string>
    )[helper] || ''
  )
})

watch(
  pointFormState,
  (state) => {
    if (draft.dataType !== state.dataType) draft.dataType = state.dataType
    if (!sameRecord(draft.address, state.address)) draft.address = state.address
  },
  { deep: true },
)

watch(
  () => props.modelValue,
  async (visible) => {
    if (!visible) return
    mode.value = props.initialMode
    if (!driver.value || driver.value.driverId !== props.driverId) {
      driver.value = await getCollectorDriver(props.driverId)
    }
    resetDraft()
  },
)

function resetDraft() {
  const point = props.point
  const defaults = point ? null : props.createDefaults
  draft.groupId = point?.groupId ?? props.defaultGroupId ?? null
  draft.name = point?.name || defaults?.name || ''
  draft.description = point?.description || defaults?.description || ''
  draft.dataType = point?.dataType || defaults?.dataType || driverDataTypes.value[0] || 'float32'
  draft.elementCount = supportsElementCount.value
    ? point?.elementCount || defaults?.elementCount || 1
    : 1
  draft.intervalMs = Number(point?.acquisition.intervalMs) || 1000
  draft.timeoutMs =
    Number(point?.acquisition.timeoutMs) || Number(defaultAcquisition.value.timeoutMs) || 3000
  draft.retryCount = Number(point?.acquisition.retryCount) || 0
  draft.deadband = Number(point?.acquisition.deadband) || 0
  draft.changeOnly = Boolean(point?.acquisition.changeOnly)
  draft.priority = Number(point?.acquisition.priority) || 0
  draft.acquisitionMode = point?.acquisitionMode || 'inherit'
  normalizedAddressText.value = point?.addressText || ''
  draft.enabled = point?.enabled ?? defaults?.enabled ?? true
  draft.address = point
    ? { ...point.address }
    : {
        ...defaultSchemaValues(driver.value?.addressSchema || {}),
        ...(defaults?.address || {}),
      }
}

function defaultSchemaValues(schema: Record<string, unknown>) {
  const properties = (schema.properties || {}) as Record<string, Record<string, unknown>>
  return Object.fromEntries(
    Object.entries(properties)
      .filter(([, property]) => property.default !== undefined)
      .map(([name, property]) => [name, property.default]),
  )
}
function buildPayload() {
  const candidate = {
    intervalMs: draft.intervalMs,
    timeoutMs: draft.timeoutMs,
    retryCount: draft.retryCount,
    deadband: draft.deadband,
    changeOnly: draft.changeOnly,
    priority: draft.priority,
  }
  const overrides = Object.fromEntries(
    Object.entries(candidate).filter(([key, value]) => value !== defaultAcquisition.value[key]),
  )
  return {
    groupId: draft.groupId,
    name: draft.name.trim(),
    description: draft.description.trim() || null,
    address: pointFormState.value.address,
    dataType: pointFormState.value.dataType,
    elementCount: supportsElementCount.value ? draft.elementCount : 1,
    readOptions: props.point?.readOptions || {},
    acquisitionMode: draft.acquisitionMode,
    acquisitionOverrides: draft.acquisitionMode === 'override' ? overrides : {},
    enabled: draft.enabled,
    sortOrder: props.point?.sortOrder || 0,
    metadata: props.point?.metadata || {},
  }
}
function acquisitionSummary(value: Record<string, unknown>) {
  return `周期 ${Number(value.intervalMs) || 1000} ms · 超时 ${Number(value.timeoutMs) || 3000} ms · 重试 ${Number(value.retryCount) || 0} 次`
}
async function normalizeAddress() {
  normalizing.value = true
  try {
    const result = await normalizeCollectorPointAddress(props.projectId, props.connectionId, {
      address: pointFormState.value.address,
      dataType: pointFormState.value.dataType,
      elementCount: supportsElementCount.value ? draft.elementCount : 1,
    })
    if (result.errors.length) {
      ElMessage.warning(result.errors.map((item) => item.message).join('；'))
      return false
    }
    draft.address = { ...result.address }
    normalizedAddressText.value = result.addressText
    return true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '地址校验失败'))
    return false
  } finally {
    normalizing.value = false
  }
}
async function save() {
  if (!draft.name.trim()) {
    ElMessage.warning('请填写变量名称')
    return
  }
  if (pointFormState.value.error) {
    ElMessage.warning(pointFormState.value.error)
    return
  }
  saving.value = true
  try {
    if (!(await normalizeAddress())) return
    const payload = buildPayload()
    if (mode.value === 'edit' && props.point) {
      await updateCollectorPointsBatch(props.projectId, props.connectionId, [
        { id: props.point.id, ...payload },
      ])
    } else {
      const result = await createCollectorPointsBatch(props.projectId, props.connectionId, [
        payload,
      ])
      if (result.failed[0]) {
        ElMessage.error(result.failed[0].message)
        return
      }
    }
    ElMessage.success(mode.value === 'edit' ? '变量已更新' : '变量已创建')
    emit('update:modelValue', false)
    emit('saved')
  } catch (error) {
    ElMessage.error(
      getApiErrorMessage(error, mode.value === 'edit' ? '变量更新失败' : '变量创建失败'),
    )
  } finally {
    saving.value = false
  }
}

function sameRecord(left: Record<string, unknown>, right: Record<string, unknown>) {
  const leftKeys = Object.keys(left)
  const rightKeys = Object.keys(right)
  return (
    leftKeys.length === rightKeys.length &&
    leftKeys.every(
      (key) => Object.prototype.hasOwnProperty.call(right, key) && left[key] === right[key],
    )
  )
}
function flattenGroups(groups: CollectorPointGroupNode[]): CollectorPointGroupNode[] {
  return groups.flatMap((group) => [group, ...flattenGroups(group.children || [])])
}
function groupLabel(groupId: string | null) {
  return flattenGroups(props.groups).find((group) => group.id === groupId)?.name || '未分组'
}
function intervalLabel(point: CollectorPoint) {
  const value = point.acquisition.intervalMs
  return typeof value === 'number' ? `${value} ms` : '-'
}
function formatJson(value: Record<string, unknown>) {
  return JSON.stringify(value, null, 2)
}
</script>

<style scoped>
.collector-point-drawer__hero {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 20px;
  padding: 18px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-lg);
  background: linear-gradient(135deg, var(--dc-primary-soft), var(--dc-surface-raised) 65%);
}
.collector-point-drawer__hero h3 {
  margin: 4px 0;
  font-size: 20px;
}
.collector-point-drawer__hero p {
  margin: 0;
  color: var(--dc-text-secondary);
}
.collector-point-drawer__debug {
  margin-top: 18px;
  padding: 18px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-lg);
  background: var(--dc-surface-raised);
}
.collector-point-drawer__debug :deep(.el-alert) {
  margin-bottom: 14px;
}
.collector-point-drawer__debug-value {
  margin-bottom: 14px;
}
.collector-point-drawer__debug-value > span {
  font-size: 12px;
  font-weight: 700;
}
.collector-point-drawer__debug-value pre {
  max-height: 180px;
  overflow: auto;
  margin: 8px 0 0;
  padding: 12px;
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-subtle);
  color: var(--dc-text);
  font:
    12px/1.6 Consolas,
    monospace;
  white-space: pre-wrap;
  word-break: break-all;
}
.collector-point-drawer__json {
  margin-top: 18px;
  padding: 18px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-lg);
  background: var(--dc-surface-raised);
}
.collector-point-drawer__json span,
.collector-point-drawer__section-title span {
  font-weight: 700;
}
.collector-point-drawer__json pre {
  overflow: auto;
  margin: 12px 0 0;
  padding: 14px;
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-subtle);
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.collector-point-drawer__section {
  margin-top: 22px;
  padding-top: 20px;
  border-top: 1px solid var(--dc-border);
}
.collector-point-drawer__section:first-child {
  margin-top: 0;
  padding-top: 0;
  border-top: 0;
}
.collector-point-drawer__section-title {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}
.collector-point-drawer__section-title span {
  font-size: 14px;
}
.collector-point-drawer__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 16px;
}
.collector-point-drawer__grid :deep(.el-select),
.collector-point-drawer__grid :deep(.el-tree-select),
.collector-point-drawer__grid :deep(.el-input-number) {
  width: 100%;
}
.collector-point-drawer__number-field {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.collector-point-drawer__unit {
  color: var(--dc-text-muted);
}
.collector-point-drawer__switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--dc-border);
}
.collector-point-drawer__inherit-hint {
  margin: 10px 0 14px;
  color: var(--dc-text-muted);
  font-size: 12px;
}
.collector-point-drawer__address-check {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 12px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.collector-point-drawer__switch-row span {
  font-size: 14px;
  font-weight: 600;
}
@media (max-width: 640px) {
  .collector-point-drawer__grid {
    grid-template-columns: 1fr;
  }
}
</style>
