<template>
  <DcDrawer
    v-model="visible"
    :width="uiPrefs.prefs.drawerWidth"
    title=""
    @resize="uiPrefs.setDrawerWidth"
  >
    <!-- 头部：名称 + 状态 + 关闭 -->
    <template #default>
      <div class="dpd">
        <!-- 自定义头 -->
        <div class="dpd__header">
          <div class="dpd__header-left">
            <span class="dpd__name">{{ datapoint?.name || '-' }}</span>
            <StatusBadge
              v-if="datapoint?.status"
              :tone="resolveStatusTone(datapoint.status)"
              :text="formatStatus(datapoint.status)"
            />
          </div>
          <div class="dpd__header-right">
            <!-- 关闭按钮 -->
            <el-tooltip :content="ui('关闭', 'Close')" placement="bottom">
              <button
                type="button"
                class="dpd__icon-btn"
                :aria-label="ui('关闭详情抽屉', 'Close details drawer')"
                @click="handleClose"
              >
                <Close />
              </button>
            </el-tooltip>
          </div>
        </div>

        <!-- 路径行 -->
        <div class="dpd__path-row">
          <span class="dpd__path">{{ datapoint?.path || '-' }}</span>
          <el-tooltip :content="ui('复制路径', 'Copy Path')" placement="bottom">
            <button
              type="button"
              class="dpd__icon-btn"
              :aria-label="ui('复制数据点路径', 'Copy data point path')"
              @click="copyPath"
            >
              <CopyDocument />
            </button>
          </el-tooltip>
        </div>

        <!-- 区块 1：基础属性 -->
        <section class="dpd__section">
          <h4 class="dpd__section-title">{{ ui('基础属性', 'Basic Properties') }}</h4>
          <dl class="dpd__grid">
            <div>
              <dt>{{ ui('来源类型', 'Source Type') }}</dt>
              <dd>{{ formatSourceType(datapoint?.sourceType) }}</dd>
            </div>
            <div>
              <dt>{{ ui('数据类型', 'Data Type') }}</dt>
              <dd class="dpd__mono">{{ datapoint?.dataType || '-' }}</dd>
            </div>
            <div>
              <dt>{{ ui('单位', 'Unit') }}</dt>
              <dd>{{ datapoint?.unit || '-' }}</dd>
            </div>
            <div>
              <dt>{{ ui('精度', 'Precision') }}</dt>
              <dd>{{ (datapoint as DataPointExtended)?.precision ?? '-' }}</dd>
            </div>
            <div class="dpd__grid-full">
              <dt>{{ ui('更新时间', 'Updated At') }}</dt>
              <dd class="dpd__mono">{{ formatTime(getUpdatedAt(datapoint)) }}</dd>
            </div>
          </dl>
        </section>

        <section class="dpd__section">
          <div class="dpd__section-header">
            <h4 class="dpd__section-title">{{ ui('开发态当前值', 'Development Current Value') }}</h4>
            <button
              type="button"
              class="dpd__edit-btn"
              :disabled="isInvalid"
              :title="isInvalid ? invalidUsageHint : ui('设置开发态默认值', 'Set Development Default Value')"
              :aria-label="ui('设置开发态默认值', 'Set Development Default Value')"
              @click="openDefaultValueDialog"
            >
              {{ ui('设置默认值', 'Set Default') }}
            </button>
          </div>
          <dl class="dpd__grid">
            <div class="dpd__grid-full">
              <dt>{{ ui('值', 'Value') }}</dt>
              <dd class="dpd__mono">{{ formatCurrentValue(datapoint?.currentValue?.value) }}</dd>
            </div>
            <div>
              <dt>{{ ui('质量', 'Quality') }}</dt>
              <dd>{{ datapoint?.currentValue?.quality || 'unknown' }}</dd>
            </div>
            <div>
              <dt>{{ ui('值来源', 'Value Origin') }}</dt>
              <dd>{{ formatOriginLabel(datapoint?.currentValue?.originLabel) }}</dd>
            </div>
            <div class="dpd__grid-full">
              <dt>{{ ui('默认值', 'Default Value') }}</dt>
              <dd class="dpd__mono">{{ formatStoredDefaultValue(datapoint?.defaultValue) }}</dd>
            </div>
            <div>
              <dt>{{ ui('观测时间', 'Observed At') }}</dt>
              <dd class="dpd__mono">
                {{
                  formatTime(
                    datapoint?.currentValue?.observedAt || datapoint?.currentValue?.timestamp,
                  )
                }}
              </dd>
            </div>
            <div>
              <dt>{{ ui('源时间', 'Source Time') }}</dt>
              <dd class="dpd__mono">{{ formatTime(datapoint?.currentValue?.sourceTimestamp) }}</dd>
            </div>
          </dl>
        </section>

        <!-- 区块 2：标签 -->
        <section class="dpd__section">
          <div class="dpd__section-header">
            <h4 class="dpd__section-title">{{ ui('标签', 'Tags') }}</h4>
            <button
              type="button"
              class="dpd__edit-btn"
              :disabled="isInvalid"
              :title="isInvalid ? invalidUsageHint : ui('编辑标签', 'Edit Tags')"
              :aria-label="ui('编辑标签', 'Edit Tags')"
              @click="openTagDialog"
            >
              {{ ui('编辑', 'Edit') }}
            </button>
          </div>
          <div class="dpd__tags">
            <el-tag
              v-for="tag in normalizeTags(datapoint?.tags)"
              :key="tag"
              size="small"
              effect="plain"
              class="dpd__tag"
            >
              {{ tag }}
            </el-tag>
            <span v-if="normalizeTags(datapoint?.tags).length === 0" class="dpd__muted">
              {{ ui('未设置', 'Not Set') }}
            </span>
          </div>
        </section>

        <!-- 区块 3：来源 -->
        <section class="dpd__section">
          <div class="dpd__section-header">
            <h4 class="dpd__section-title">{{ ui('来源', 'Source') }}</h4>
          </div>

          <!-- 来源 LinkChip -->
          <div v-if="sourceNavigation" class="dpd__source-chip">
            <LinkChip
              :module="sourceNavigation.module"
              :object-id="sourceNavigation.objectId"
              :label="sourceNavigation.label"
              @click="handleLinkChipClick"
            />
          </div>
          <p v-else-if="collectorSourceName" class="dpd__muted">
            {{ ui('工业采集', 'Industrial Collection') }} · {{ collectorSourceName }}
          </p>
          <p v-else class="dpd__muted">{{ ui('无来源信息', 'No source information') }}</p>

          <!-- 失效原因（仅 invalid 时显示；无 invalidReason 时给兜底文案） -->
          <div v-if="datapoint?.status === 'invalid'" class="dpd__invalid-reason">
            <span class="dpd__invalid-label">{{ ui('失效原因：', 'Invalid Reason: ') }}</span>
            {{
              (datapoint as DataPointExtended)?.invalidReason || ui('该数据点已失效，但后端未提供原因', 'This data point is invalid, but no reason was provided.')
            }}
          </div>

          <!-- 配置 JSON 折叠区 -->
          <div v-if="hasSourceConfig" class="dpd__config-collapse">
            <button
              type="button"
              class="dpd__collapse-trigger"
              @click="configExpanded = !configExpanded"
            >
              <span>{{ ui('来源配置', 'Source Configuration') }}</span>
              <ArrowDown :class="{ 'is-expanded': configExpanded }" class="dpd__collapse-icon" />
            </button>
            <pre v-if="configExpanded" class="dpd__code">{{
              formatSourceConfig((datapoint as DataPointExtended)?.sourceConfig)
            }}</pre>
          </div>
        </section>

        <section class="dpd__section">
          <div class="dpd__section-header">
            <h4 class="dpd__section-title">{{ ui('历史存储', 'History Storage') }}</h4>
            <button
              type="button"
              class="dpd__edit-btn"
              :disabled="isInvalid"
              :title="isInvalid ? invalidUsageHint : ui('编辑历史存储', 'Edit History Storage')"
              :aria-label="ui('编辑历史存储', 'Edit History Storage')"
              @click="openHistoryStorage"
            >
              {{ ui('编辑', 'Edit') }}
            </button>
          </div>
          <div class="dpd__perm-summary">
            <span class="dpd__perm-label">{{ ui('保存方式：', 'Storage Mode: ') }}</span>
            <span class="dpd__perm-value">{{ historyStorageSummaryText }}</span>
          </div>
          <p v-if="historyStorageDetail?.source" class="dpd__muted">
            {{ ui('来源：', 'Source: ') }}{{ historyStorageDetail.source.name || ui('所属来源', 'Owning Source') }}
          </p>
        </section>

        <section class="dpd__section">
          <div class="dpd__section-header">
            <h4 class="dpd__section-title">{{ ui('报警设置', 'Alarm Settings') }}</h4>
            <button
              type="button"
              class="dpd__edit-btn"
              :aria-label="ui('前往报警单元', 'Go to Alarm Units')"
              @click="openAlarmWorkspace"
            >
              {{ ui('前往报警单元', 'Go to Alarm Units') }}
            </button>
          </div>
          <div class="dpd__perm-summary">
            <span class="dpd__perm-label">{{ ui('有效报警：', 'Active Alarms: ') }}</span>
            <span class="dpd__perm-value">{{ ui(`${alarmSummary?.count || 0} 条`, `${alarmSummary?.count || 0}`) }}</span>
          </div>
          <div v-if="alarmSummary?.items.length" class="dpd__usage-chips">
            <LinkChip
              v-for="item in alarmSummary.items"
              :key="item.id"
              module="alarm"
              :object-id="item.id"
              :label="item.displayName"
              @click="handleLinkChipClick"
            />
          </div>
          <p v-else class="dpd__muted">{{ ui('当前数据点尚未配置报警', 'No alarms configured for this data point') }}</p>
        </section>

        <!-- 区块 4：引用 -->
        <section v-if="usages.length > 0" class="dpd__section">
          <h4 class="dpd__section-title">{{ ui('引用', 'References') }}</h4>
          <div v-for="group in usageGroups" :key="group.module" class="dpd__usage-group">
            <p class="dpd__usage-group-label">{{ group.label }}</p>
            <div class="dpd__usage-chips">
              <LinkChip
                v-for="item in group.items"
                :key="item.objectId"
                :module="group.module"
                :object-id="item.objectId"
                :label="item.label"
                @click="handleLinkChipClick"
              />
            </div>
          </div>
        </section>

        <!-- 区块 5：运行态权限 -->
        <section class="dpd__section">
          <div class="dpd__section-header">
            <h4 class="dpd__section-title">{{ ui('运行态权限', 'Runtime Permissions') }}</h4>
            <button
              type="button"
              class="dpd__edit-btn"
              :disabled="isInvalid"
              :title="isInvalid ? invalidUsageHint : ui('编辑运行态权限', 'Edit Runtime Permissions')"
              :aria-label="ui('编辑运行态权限', 'Edit Runtime Permissions')"
              @click="openPermissionDialog"
            >
              {{ ui('编辑', 'Edit') }}
            </button>
          </div>
          <div class="dpd__perm-summary">
            <span class="dpd__perm-label">{{ ui('写权限摘要：', 'Write Permission: ') }}</span>
            <span class="dpd__perm-value">{{ writeSummary }}</span>
          </div>
          <div class="dpd__perm-badges">
            <el-tooltip
              v-for="item in capabilityBadges"
              :key="item.key"
              :content="item.reason || item.text"
              placement="top"
            >
              <span
                ><StatusBadge :tone="item.enabled ? 'success' : 'muted'" :text="item.text"
              /></span>
            </el-tooltip>
          </div>
        </section>
      </div>
    </template>
  </DcDrawer>

  <!-- 标签 dialog -->
  <DataPointTagDialog
    :visible="tagDialogVisible"
    :datapoint="tagDialogDatapoint"
    :project-id="projectId"
    :tag-options="tagOptions"
    :saving="tagSaving"
    @submit="handleTagSubmit"
    @cancel="tagDialogVisible = false"
  />

  <!-- 写权限 dialog -->
  <RuntimePermissionDialog
    :visible="permDialogVisible"
    :datapoint="permDialogDatapoint"
    :project-id="projectId"
    :saving="permSaving"
    @submit="handlePermSubmit"
    @cancel="permDialogVisible = false"
  />

  <HistoryStorageConfigDrawer
    v-model="historyStorageDrawerVisible"
    :title="ui('数据点历史存储', 'Data Point History Storage')"
    allow-inherit
    :behavior="historyStorageDetail?.behavior || 'inherit'"
    :configuration="historyStorageDetail?.configuration"
    :targets="historyStorageTargets"
    :saving="historyStorageSaving"
    @save="saveHistoryStorage"
  />

  <el-dialog
    v-model="defaultValueDialogVisible"
    :title="ui('设置开发态默认值', 'Set Development Default Value')"
    width="520px"
    append-to-body
  >
    <p class="dpd__dialog-hint">{{ ui('未连接运行态数据时，计算调试和页面预览会使用该值。', 'Compute debugging and page previews use this value when runtime data is unavailable.') }}</p>
    <el-select
      v-if="defaultValueEditorKind === 'boolean'"
      v-model="defaultValueDraft"
      :placeholder="ui('请选择', 'Select')"
    >
      <el-option label="true" value="true" />
      <el-option label="false" value="false" />
    </el-select>
    <el-input
      v-else-if="defaultValueEditorKind === 'json'"
      v-model="defaultValueDraft"
      type="textarea"
      :rows="8"
      :placeholder="ui('请输入合法 JSON', 'Enter valid JSON')"
    />
    <el-input v-else v-model="defaultValueDraft" :placeholder="defaultValuePlaceholder" />
    <template #footer>
      <el-button :loading="defaultValueSaving" @click="clearDefaultValue">{{ ui('清除默认值', 'Clear Default') }}</el-button>
      <el-button @click="defaultValueDialogVisible = false">{{ ui('取消', 'Cancel') }}</el-button>
      <el-button type="primary" :loading="defaultValueSaving" @click="saveDefaultValue"
        >{{ ui('保存', 'Save') }}</el-button
      >
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowDown, Close, CopyDocument } from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import { TIME_FORMAT } from '@/constants'
import { updateDatapoint, updateDatapointRuntimeGrant } from '@/api/datapoint.api'
import {
  normalizeRuntimeGrantPayload,
  summarizeRuntimeGrant,
} from '@/utils/runtime-permission-grants'
import { useUiPrefsStore } from '@/stores/ui-prefs.store'
import DcDrawer from '@/components/shared/DcDrawer.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import LinkChip from '@/components/shared/LinkChip.vue'
import DataPointTagDialog from './DataPointTagDialog.vue'
import RuntimePermissionDialog from './RuntimePermissionDialog.vue'
import HistoryStorageConfigDrawer from '@/components/history-storage/HistoryStorageConfigDrawer.vue'
import {
  getDatapointHistoryStorage,
  listHistoryStorageTargets,
  saveDatapointHistoryStorage,
  type HistoryStorageSavePayload,
} from '@/api/history-storage.api'
import type {
  HistoryStorageDatapointDetail,
  HistoryStorageTargetOption,
} from '@/api/schemas/history-storage.schema'
import { historyStorageModeLabels } from '@/models/history-storage'
import { getDatapointAlarmSummary } from '@/api/alarm.api'
import type { AlarmDatapointSummary } from '@/api/schemas/alarm.schema'
import { resolveDatapointSourceNavigation } from '@/models/datapoint-source'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

// 扩展类型：除 schema 核心字段外，允许后端额外字段（passthrough）
interface DataPointExtended {
  id: string
  path?: string
  name?: string
  status?: string
  sourceType?: string
  sourceId?: string | null
  dataType?: string
  defaultValue?: string | null
  unit?: string | null
  precision?: number | string | null
  tags?: unknown[]
  updatedAt?: string
  updated_at?: string
  createdAt?: string
  created_at?: string
  runtimePermissions?: Record<string, unknown>
  runtimePermissionGrants?: Record<string, unknown>
  writePermission?: Record<string, unknown>
  runtimeGrant?: Record<string, unknown>
  sourceConfig?: Record<string, unknown>
  invalidReason?: string | null
  capabilities?: Record<'get' | 'sub' | 'set', { enabled: boolean; reason?: string }>
  currentValue?: {
    value?: unknown
    quality?: string
    timestamp?: string
    observedAt?: string
    sourceTimestamp?: string
    valueOrigin?: string
    originLabel?: string
  } | null
  /** 报警策略与计算单元引用列表 */
  usages?: UsageItem[]
}

interface UsageItem {
  module: 'access-source' | 'industrial-collector' | 'compute' | 'alarm' | 'datapoint'
  objectId: string
  label: string
}

// LinkChip 的 module 类型
type LinkChipModule = 'datapoint' | 'access-source' | 'industrial-collector' | 'compute' | 'alarm'

const props = defineProps<{
  /** 是否显示 */
  modelValue: boolean
  /** 数据点详情 */
  datapoint: DataPointExtended | null
  /** 项目 ID */
  projectId: string
  /** 标签选项（由 DataPointList 传入） */
  tagOptions?: Array<{ value: string; name: string }>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  /** 抽屉关闭 */
  close: []
  /** 数据已更新，通知父组件刷新 */
  updated: []
  /** 跳转到另一个模块 */
  navigate: [payload: { module: LinkChipModule; objectId?: string; tab?: string }]
}>()

const uiPrefs = useUiPrefsStore()

// 来源配置折叠状态
const configExpanded = ref(false)

// 标签 dialog
const tagDialogVisible = ref(false)
const tagDialogDatapoint = ref<DataPointExtended | null>(null)
const tagSaving = ref(false)

// 权限 dialog
const permDialogVisible = ref(false)
const permDialogDatapoint = ref<DataPointExtended | null>(null)
const permSaving = ref(false)

const historyStorageDetail = ref<HistoryStorageDatapointDetail | null>(null)
const historyStorageTargets = ref<HistoryStorageTargetOption[]>([])
const historyStorageDrawerVisible = ref(false)
const historyStorageLoading = ref(false)
const historyStorageSaving = ref(false)
const alarmSummary = ref<AlarmDatapointSummary | null>(null)
const defaultValueDialogVisible = ref(false)
const defaultValueDraft = ref('')
const defaultValueSaving = ref(false)
let detailRequestSeq = 0

const visible = computed({
  get: () => props.modelValue,
  set: (val: boolean) => {
    emit('update:modelValue', val)
    if (!val) emit('close')
  },
})
const isInvalid = computed(() => props.datapoint?.status === 'invalid')
const invalidUsageHint = computed(() => ui('该数据点已失效，仅可查看或清理', 'This data point is invalid and can only be viewed or cleaned up'))

// 折叠重置（切换数据点时）
watch(
  () => [props.datapoint?.id, props.modelValue] as const,
  ([id, isVisible]) => {
    detailRequestSeq += 1
    configExpanded.value = false
    historyStorageDetail.value = null
    if (id && isVisible) {
      void loadHistoryStorage(String(id))
      void loadAlarmSummary(String(id))
    }
  },
)

const historyStorageSummaryText = computed(() => {
  if (historyStorageLoading.value) return ui('加载中', 'Loading')
  const detail = historyStorageDetail.value
  if (!detail) return ui('未保存', 'Not Stored')
  if (detail.behavior === 'off') return ui('不保存', 'Disabled')
  if (!detail.effectiveEnabled || !detail.configuration) return ui('沿用来源 · 未保存', 'Inherited · Not Stored')
  const mode = ui(historyStorageModeLabels[detail.configuration.writeMode], historyStorageModeEnglish(detail.configuration.writeMode))
  return detail.behavior === 'custom' ? ui(`单独设置 · ${mode}`, `Custom · ${mode}`) : ui(`沿用来源 · ${mode}`, `Inherited · ${mode}`)
})

// ── 来源配置相关 ──────────────────────────────────────────────────────────

const hasSourceConfig = computed(() => {
  const cfg = (props.datapoint as DataPointExtended)?.sourceConfig
  return !!cfg && Object.keys(cfg).length > 0
})

const sourceNavigation = computed(() => resolveDatapointSourceNavigation(props.datapoint))
const collectorSourceName = computed(() => {
  const source = historyStorageDetail.value?.source
  return source?.type === 'collector_connection' ? source.name || ui('所属采集连接', 'Owning Collector Connection') : ''
})

function formatSourceConfig(value?: Record<string, unknown>): string {
  if (!value || Object.keys(value).length === 0) return '{}'
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return '{}'
  }
}

// ── 引用区块 ────────────────────────────────────────────────────────────

const usages = computed<UsageItem[]>(() => (props.datapoint as DataPointExtended)?.usages ?? [])

// 按模块分组
const moduleLabel = (module: string) => ({
  'access-source': ui('接入源', 'Access Source'),
  'industrial-collector': ui('工业采集连接', 'Industrial Collector'),
  compute: ui('计算单元', 'Compute Unit'),
  alarm: ui('报警项', 'Alarm Item'),
  datapoint: ui('数据点', 'Data Point'),
}[module] || module)

const usageGroups = computed(() => {
  const groupMap = new Map<string, { module: LinkChipModule; label: string; items: UsageItem[] }>()
  for (const item of usages.value) {
    if (!groupMap.has(item.module)) {
      groupMap.set(item.module, {
        module: item.module,
        label: moduleLabel(item.module),
        items: [],
      })
    }
    groupMap.get(item.module)!.items.push(item)
  }
  return Array.from(groupMap.values())
})

// ── 权限摘要 ────────────────────────────────────────────────────────────

const writeSummary = computed(() => {
  const dp = props.datapoint
  if (!dp) return '-'
  const rp = dp.runtimePermissions as { write?: unknown } | undefined
  const rpg = dp.runtimePermissionGrants as { write?: unknown } | undefined
  const grant = normalizeRuntimeGrantPayload(
    rp?.write || rpg?.write || dp.writePermission || dp.runtimeGrant || {},
  )
  return summarizeRuntimeGrant(grant, datacenterLocale.value)
})

const capabilityBadges = computed(() => {
  const capabilities = props.datapoint?.capabilities
  return [
    {
      key: 'get',
      text: ui('可读', 'Readable'),
      enabled: capabilities?.get?.enabled === true,
      reason: capabilities?.get?.reason,
    },
    {
      key: 'sub',
      text: ui('可订阅', 'Subscribable'),
      enabled: capabilities?.sub?.enabled === true,
      reason: capabilities?.sub?.reason,
    },
    {
      key: 'set',
      text: ui('可写', 'Writable'),
      enabled: capabilities?.set?.enabled === true,
      reason: capabilities?.set?.reason,
    },
  ]
})

function formatCurrentValue(value: unknown): string {
  if (value === undefined || value === null) return '-'
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value)
    } catch {
      return String(value)
    }
  }
  return String(value)
}

function formatOriginLabel(label?: string) {
  if (!label) return ui('不可用', 'Unavailable')
  if (datacenterLocale.value !== 'en') return label
  return {
    当前值不可用: 'Current value unavailable',
    开发态默认值: 'Development default value',
    运行态当前值: 'Runtime current value',
    来源默认值: 'Source default value',
  }[label] || label
}

function formatStoredDefaultValue(value?: string | null) {
  if (value === undefined || value === null) return ui('未设置', 'Not Set')
  return value === '' ? ui('空字符串', 'Empty String') : value
}

const defaultValueEditorKind = computed<'boolean' | 'number' | 'json' | 'string'>(() => {
  const type = String(props.datapoint?.dataType || '').toLowerCase()
  if (['bool', 'boolean'].includes(type)) return 'boolean'
  if (/(int|uint|float|double|decimal|number)/.test(type)) return 'number'
  if (/(object|json|array|list|map)/.test(type)) return 'json'
  return 'string'
})
const defaultValuePlaceholder = computed(() =>
  defaultValueEditorKind.value === 'number' ? ui('请输入数字', 'Enter a number') : ui('请输入默认值', 'Enter a default value'),
)

function historyStorageModeEnglish(mode: string) {
  return {
    on_change: 'On Change',
    interval_latest: 'Latest per Interval',
    periodic_snapshot: 'Periodic Snapshot',
    every_sample: 'Every Sample',
  }[mode] || mode
}

// ── 格式化工具 ────────────────────────────────────────────────────────────

function resolveStatusTone(status?: string): 'success' | 'danger' | 'warning' | 'muted' {
  switch (status) {
    case 'active':
      return 'success'
    case 'invalid':
      return 'danger'
    case 'error':
      return 'warning'
    default:
      return 'muted'
  }
}

function formatStatus(status?: string): string {
  switch (status) {
    case 'active':
      return ui('活跃', 'Active')
    case 'invalid':
      return ui('失效', 'Invalid')
    case 'error':
      return ui('错误', 'Error')
    case 'inactive':
      return ui('停用', 'Inactive')
    default:
      return ui('未知', 'Unknown')
  }
}

const sourceTypeLabel = (sourceType: string) => ({
  'db.query': ui('数据库查询', 'Database Query'),
  'mqtt.tag': ui('MQTT 变量', 'MQTT Variable'),
  'mqtt.subscription': ui('MQTT 订阅', 'MQTT Subscription'),
  'http.request': ui('HTTP 请求', 'HTTP Request'),
  'websocket.session': ui('WebSocket 会话', 'WebSocket Session'),
  'realtime.key': ui('实时库 Key', 'Realtime Key'),
  'kafka.field': ui('Kafka 字段', 'Kafka Field'),
  'kafka.raw': ui('Kafka 整包', 'Kafka Raw Message'),
  'calc.output': ui('计算输出', 'Compute Output'),
  'static.var': ui('静态变量', 'Static Variable'),
  'collector.point': ui('工业采集点', 'Industrial Collection Point'),
}[sourceType] || sourceType)

function formatSourceType(sourceType?: string): string {
  if (!sourceType) return '-'
  return sourceTypeLabel(sourceType)
}

function formatTime(value?: string | null): string {
  if (!value) return '-'
  return dayjs(value).format(TIME_FORMAT)
}

function getUpdatedAt(dp: DataPointExtended | null): string | undefined {
  if (!dp) return undefined
  return dp.updatedAt || dp.updated_at || dp.createdAt || dp.created_at || undefined
}

function normalizeTags(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  const result: string[] = []
  for (const item of value) {
    let label = ''
    if (typeof item === 'string') {
      label = item
    } else if (item && typeof item === 'object') {
      const r = item as Record<string, unknown>
      label = String(r.label || r.name || r.value || '')
    }
    const n = label.trim()
    if (n && !result.includes(n)) result.push(n)
  }
  return result
}

// ── 交互 ──────────────────────────────────────────────────────────────────

function handleClose() {
  visible.value = false
}

async function copyPath() {
  const path = props.datapoint?.path
  if (!path) return
  try {
    await navigator.clipboard.writeText(path)
    ElMessage.success(ui('已复制数据点路径', 'Data point path copied'))
  } catch {
    ElMessage.error(ui('复制失败，请手动复制', 'Copy failed; copy the path manually'))
  }
}

function openDefaultValueDialog() {
  if (isInvalid.value) return ElMessage.warning(invalidUsageHint.value)
  defaultValueDraft.value = props.datapoint?.defaultValue ?? ''
  defaultValueDialogVisible.value = true
}

function normalizeDefaultValueDraft(): string | null {
  const raw = defaultValueDraft.value
  if (defaultValueEditorKind.value === 'boolean') {
    return raw === 'true' || raw === 'false' ? raw : null
  }
  if (defaultValueEditorKind.value === 'number') {
    const value = Number(raw)
    return raw.trim() && Number.isFinite(value) ? raw.trim() : null
  }
  if (defaultValueEditorKind.value === 'json') {
    try {
      const value = JSON.parse(raw)
      if (value === null || typeof value !== 'object') return null
      return JSON.stringify(value)
    } catch {
      return null
    }
  }
  return raw
}

async function saveDefaultValue() {
  if (!props.datapoint) return
  const value = normalizeDefaultValueDraft()
  if (value === null) {
    ElMessage.warning(
      defaultValueEditorKind.value === 'json' ? ui('请输入合法的对象或数组 JSON', 'Enter valid object or array JSON') : ui('默认值格式无效', 'Invalid default value'),
    )
    return
  }
  defaultValueSaving.value = true
  try {
    await updateDatapoint(props.projectId, String(props.datapoint.id), { defaultValue: value })
    defaultValueDialogVisible.value = false
    ElMessage.success(ui('开发态默认值已保存', 'Development default value saved'))
    emit('updated')
  } catch {
    ElMessage.error(ui('保存开发态默认值失败', 'Failed to save development default value'))
  } finally {
    defaultValueSaving.value = false
  }
}

async function clearDefaultValue() {
  if (!props.datapoint) return
  defaultValueSaving.value = true
  try {
    await updateDatapoint(props.projectId, String(props.datapoint.id), { defaultValue: null })
    defaultValueDialogVisible.value = false
    ElMessage.success(ui('开发态默认值已清除', 'Development default value cleared'))
    emit('updated')
  } catch {
    ElMessage.error(ui('清除开发态默认值失败', 'Failed to clear development default value'))
  } finally {
    defaultValueSaving.value = false
  }
}

async function loadHistoryStorage(datapointId: string) {
  const seq = detailRequestSeq
  historyStorageLoading.value = true
  try {
    const result = await getDatapointHistoryStorage(props.projectId, datapointId)
    if (seq === detailRequestSeq) historyStorageDetail.value = result
  } catch {
    if (seq === detailRequestSeq) historyStorageDetail.value = null
  } finally {
    if (seq === detailRequestSeq) historyStorageLoading.value = false
  }
}

async function openHistoryStorage() {
  if (isInvalid.value) return ElMessage.warning(invalidUsageHint.value)
  const datapointId = props.datapoint?.id
  if (!datapointId) return
  try {
    const [detail, targets] = await Promise.all([
      getDatapointHistoryStorage(props.projectId, String(datapointId)),
      historyStorageTargets.value.length > 0
        ? Promise.resolve(historyStorageTargets.value)
        : listHistoryStorageTargets(props.projectId),
    ])
    historyStorageDetail.value = detail
    historyStorageTargets.value = targets
    historyStorageDrawerVisible.value = true
  } catch {
    ElMessage.error(ui('加载历史存储设置失败', 'Failed to load history storage settings'))
  }
}

async function saveHistoryStorage(payload: HistoryStorageSavePayload) {
  const datapointId = props.datapoint?.id
  if (!datapointId) return
  historyStorageSaving.value = true
  try {
    historyStorageDetail.value = await saveDatapointHistoryStorage(
      props.projectId,
      String(datapointId),
      payload,
    )
    historyStorageDrawerVisible.value = false
    ElMessage.success(ui('历史存储设置已保存', 'History storage settings saved'))
    emit('updated')
  } catch {
    ElMessage.error(ui('保存历史存储设置失败', 'Failed to save history storage settings'))
  } finally {
    historyStorageSaving.value = false
  }
}

async function loadAlarmSummary(datapointId: string) {
  const seq = detailRequestSeq
  try {
    const result = await getDatapointAlarmSummary(props.projectId, datapointId)
    if (seq === detailRequestSeq) alarmSummary.value = result
  } catch {
    if (seq === detailRequestSeq) alarmSummary.value = null
  }
}

function openAlarmWorkspace() {
  emit('navigate', { module: 'alarm' })
}

// ── 标签 dialog ────────────────────────────────────────────────────────────

function openTagDialog() {
  if (isInvalid.value) return ElMessage.warning(invalidUsageHint.value)
  if (!props.datapoint) return
  tagDialogDatapoint.value = props.datapoint
  tagDialogVisible.value = true
}

async function handleTagSubmit(tags: string[]) {
  if (!props.datapoint) return
  tagSaving.value = true
  try {
    await updateDatapoint(props.projectId, String(props.datapoint.id), { tags } as never)
    ElMessage.success(ui('标签已保存', 'Tags saved'))
    tagDialogVisible.value = false
    emit('updated')
  } catch {
    ElMessage.error(ui('保存标签失败', 'Failed to save tags'))
  } finally {
    tagSaving.value = false
  }
}

// ── 权限 dialog ────────────────────────────────────────────────────────────

function openPermissionDialog() {
  if (isInvalid.value) return ElMessage.warning(invalidUsageHint.value)
  if (!props.datapoint) return
  permDialogDatapoint.value = props.datapoint
  permDialogVisible.value = true
}

async function handlePermSubmit(grant: Record<string, unknown>) {
  if (!props.datapoint) return
  permSaving.value = true
  try {
    await updateDatapointRuntimeGrant(props.projectId, String(props.datapoint.id), { write: grant })
    ElMessage.success(ui('写权限已保存', 'Write permission saved'))
    permDialogVisible.value = false
    emit('updated')
  } catch {
    ElMessage.error(ui('保存运行态权限失败', 'Failed to save runtime permissions'))
  } finally {
    permSaving.value = false
  }
}

// ── LinkChip 跳转 ──────────────────────────────────────────────────────────

function handleLinkChipClick(payload: { module: LinkChipModule; objectId: string }) {
  emit('navigate', { module: payload.module, objectId: payload.objectId, tab: 'workbench' })
}
</script>

<style scoped>
/* ── 整体容器 ── */
.dpd {
  display: flex;
  flex-direction: column;
  gap: 0;
  height: 100%;
  overflow-y: auto;
}

.dpd__dialog-hint {
  margin: 0 0 12px;
  color: var(--dc-text-muted);
  font-size: 13px;
}

/* ── 头部 ── */
.dpd__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 16px 10px;
  border-bottom: 1px solid var(--dc-border);
}

.dpd__header-left {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.dpd__header-right {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.dpd__name {
  font-size: 16px;
  font-weight: 700;
  color: var(--dc-text);
  overflow-wrap: anywhere;
}

/* ── 路径行 ── */
.dpd__path-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px 16px 12px;
  border-bottom: 1px solid var(--dc-border);
}

.dpd__path {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--dc-text-secondary);
  font-family: var(--dc-font-mono);
  font-size: 12px;
  line-height: 1.6;
}

/* ── icon 按钮 ── */
.dpd__icon-btn {
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--dc-text-muted);
  cursor: pointer;
  transition:
    background 0.18s,
    color 0.18s,
    border-color 0.18s;
}

.dpd__icon-btn svg {
  width: 15px;
  height: 15px;
}

.dpd__icon-btn:hover {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.dpd__icon-btn.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

/* ── 区块通用 ── */
.dpd__section {
  padding: 16px;
  border-bottom: 1px solid var(--dc-border);
}

.dpd__section:last-child {
  border-bottom: 0;
}

.dpd__section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.dpd__section-title {
  margin: 0 0 10px;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.dpd__section-header .dpd__section-title {
  margin: 0;
}

/* ── 编辑按钮 ── */
.dpd__edit-btn {
  height: 24px;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: 8px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  font-weight: 600;
  transition:
    background 0.18s,
    border-color 0.18s,
    color 0.18s;
}

.dpd__edit-btn:hover {
  border-color: rgba(29, 78, 216, 0.24);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

/* ── 基础属性 grid ── */
.dpd__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin: 0;
}

.dpd__grid > div {
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface);
}

.dpd__grid-full {
  grid-column: 1 / -1;
}

.dpd__grid dt {
  margin: 0 0 4px;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.dpd__grid dd {
  margin: 0;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 600;
  overflow-wrap: anywhere;
}

.dpd__mono {
  font-family: var(--dc-font-mono);
}

/* ── 标签 ── */
.dpd__tags {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  min-height: 28px;
}

.dpd__tag {
  --el-tag-bg-color: var(--dc-primary-soft);
  --el-tag-border-color: transparent;
  --el-tag-text-color: var(--dc-primary);
  border-radius: 999px;
  font-weight: 700;
}

.dpd__muted {
  color: var(--dc-text-muted);
  font-size: 12px;
}

/* ── 来源区块 ── */
.dpd__source-chip {
  margin-bottom: 8px;
}

.dpd__invalid-reason {
  padding: 8px 10px;
  border-radius: var(--dc-radius-md);
  background: rgba(220, 38, 38, 0.06);
  color: var(--dc-danger, #dc2626);
  font-size: 12px;
  line-height: 1.6;
  margin-bottom: 8px;
}

.dpd__invalid-label {
  font-weight: 600;
}

/* ── 来源配置折叠 ── */
.dpd__config-collapse {
  margin-top: 8px;
}

.dpd__collapse-trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  font-weight: 600;
}

.dpd__collapse-icon {
  width: 14px;
  height: 14px;
  transition: transform 0.2s;
}

.dpd__collapse-icon.is-expanded {
  transform: rotate(180deg);
}

.dpd__code {
  max-height: 200px;
  overflow: auto;
  margin: 8px 0 0;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-family: var(--dc-font-mono);
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

/* ── 引用区块 ── */
.dpd__usage-group {
  margin-bottom: 10px;
}

.dpd__usage-group-label {
  margin: 0 0 6px;
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 600;
}

.dpd__usage-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

/* ── 权限区块 ── */
.dpd__perm-summary {
  margin-bottom: 8px;
  font-size: 13px;
  color: var(--dc-text-secondary);
}

.dpd__perm-label {
  font-weight: 600;
  color: var(--dc-text-muted);
}

.dpd__perm-value {
  color: var(--dc-text);
}

.dpd__perm-badges {
  display: flex;
  gap: 6px;
}
</style>
