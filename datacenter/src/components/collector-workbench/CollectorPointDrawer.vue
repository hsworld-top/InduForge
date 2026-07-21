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
          <el-form-item label="采集周期">
            <div class="collector-point-drawer__number-field">
              <el-input-number v-model="draft.intervalMs" :min="100" :step="100" />
              <span class="collector-point-drawer__unit">ms</span>
            </div>
          </el-form-item>
        </div>
        <el-form-item label="变量说明">
          <el-input v-model="draft.description" type="textarea" :rows="3" maxlength="500" />
        </el-form-item>
      </section>

      <section class="collector-point-drawer__section">
        <div class="collector-point-drawer__section-title">
          <span>协议地址</span>
          <el-tag v-if="sourceLocked" size="small" type="info">设备识别 · 只读</el-tag>
        </div>
        <CollectorSchemaForm
          v-if="driver"
          v-model="draft.address"
          :schema="driver.addressSchema"
          :ui-schema="{}"
          :disabled="sourceLocked"
        />
        <el-skeleton v-else :rows="3" animated />
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
const draft = reactive({
  groupId: null as string | null,
  name: '',
  description: '',
  dataType: 'float32',
  elementCount: 1,
  intervalMs: 1000,
  enabled: true,
  address: {} as Record<string, unknown>,
})
const drawerTitle = computed(() =>
  mode.value === 'create' ? '新建变量' : mode.value === 'edit' ? '编辑变量' : '变量详情',
)
const dataTypes = computed(() => driver.value?.dataTypes || [])
const supportsElementCount = computed(() =>
  collectorDriverSupportsFeature(driver.value, collectorDriverFeatures.pointElementCount),
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
  draft.dataType = point?.dataType || defaults?.dataType || driver.value?.dataTypes[0] || 'float32'
  draft.elementCount = supportsElementCount.value
    ? point?.elementCount || defaults?.elementCount || 1
    : 1
  draft.intervalMs = Number(point?.acquisition.intervalMs) || 1000
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
  return {
    groupId: draft.groupId,
    name: draft.name.trim(),
    description: draft.description.trim() || null,
    address: draft.address,
    dataType: draft.dataType,
    elementCount: supportsElementCount.value ? draft.elementCount : 1,
    readOptions: props.point?.readOptions || {},
    acquisition: { ...(props.point?.acquisition || {}), intervalMs: draft.intervalMs },
    enabled: draft.enabled,
    sortOrder: props.point?.sortOrder || 0,
    metadata: props.point?.metadata || {},
  }
}
async function save() {
  if (!draft.name.trim()) {
    ElMessage.warning('请填写变量名称')
    return
  }
  saving.value = true
  try {
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
