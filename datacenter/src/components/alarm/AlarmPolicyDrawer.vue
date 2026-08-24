<template>
  <DcDrawer v-model="visible" :title="drawerTitle" :width="700" :max="820">
    <template #actions>
      <button type="button" class="alarm-editor__icon" title="关闭" @click="visible = false">
        <IconTablerX />
      </button>
    </template>

    <form class="alarm-editor" @submit.prevent="submit">
      <section v-if="item" class="alarm-editor__impact">
        当前编辑的是一条独立报警项，修改只影响
        <strong>{{
          item.mode === 'point' ? item.datapointName || item.path : '当前组合报警'
        }}</strong
        >。
      </section>

      <section class="alarm-editor__section">
        <div class="alarm-editor__grid" :class="{ 'is-point-create': !showIdentityFields }">
          <label v-if="showIdentityFields" class="alarm-editor__field is-wide"
            ><span>{{ draft.mode === 'derived' ? '组合报警名称' : '显示名称（可选）' }}</span
            ><el-input
              v-model="draft.displayName"
              maxlength="100"
              placeholder="普通报警留空时按类型自动生成"
          /></label>
          <label class="alarm-editor__field"
            ><span>目录</span
            ><el-select v-model="draft.groupId" clearable placeholder="根目录"
              ><el-option label="根目录" :value="null" /><el-option
                v-for="group in groups"
                :key="group.id"
                :label="group.fullPath || group.name"
                :value="group.id" /></el-select
          ></label>
          <label class="alarm-editor__field is-switch"
            ><span>启用</span><el-switch v-model="draft.isEnabled"
          /></label>
        </div>
        <label v-if="showIdentityFields" class="alarm-editor__field"
          ><span>说明（可选）</span><el-input v-model="draft.description" maxlength="500"
        /></label>
      </section>

      <section class="alarm-editor__section">
        <div class="alarm-editor__section-head">
          <div>
            <h3>{{ draft.mode === 'derived' ? '组合输入点' : '报警数据点' }}</h3>
            <p>只应用到明确选择的数据点；后续新建点不会自动加入。</p>
          </div>
          <button
            v-if="canManagePoints"
            type="button"
            class="alarm-editor__secondary"
            @click="pointPickerVisible = true"
          >
            <IconTablerPlus />{{ selectedPointCount ? '管理数据点' : '选择数据点' }}
          </button>
        </div>
        <div v-if="draft.selectedPoints.length" class="alarm-editor__point-summary">
          <div>
            <strong>已选择 {{ selectedPointCount }} 个数据点</strong><span>{{ pointPreview }}</span
            ><small v-if="!item && draft.mode === 'point'"
              >保存后将创建 {{ selectedPointCount }} 条独立报警项，后续可分别维护。</small
            >
          </div>
          <button v-if="canManagePoints" type="button" @click="pointPickerVisible = true">
            查看与调整
          </button>
        </div>
        <div v-else class="alarm-editor__empty">请选择要配置报警的数据点</div>
      </section>

      <section v-if="draft.mode === 'derived'" class="alarm-editor__section">
        <div class="alarm-editor__section-head">
          <div>
            <h3>组合表达式</h3>
            <p>
              显示每个输入的数据类型；变量名仅用于表达式，不会修改数据点名称。保存时会校验表达式和结果条件的类型。
            </p>
          </div>
        </div>
        <div class="alarm-editor__aliases">
          <label v-for="point in draft.selectedPoints" :key="point.datapointId"
            ><span
              >{{ point.name }}<small>{{ formatPointDataType(point.dataType) }}</small></span
            ><el-input v-model="point.inputKey" placeholder="表达式变量名"
          /></label>
        </div>
        <el-input
          v-model="draft.derivedExpression"
          type="textarea"
          :rows="3"
          placeholder="例如：temperature > 80 && pressure > 1.2"
        />
      </section>

      <section class="alarm-editor__section">
        <div class="alarm-editor__section-head">
          <div>
            <h3>{{ draft.evaluationMode === 'highest_matching' ? '越限等级' : '报警条件' }}</h3>
            <p>{{ conditionHint }}</p>
          </div>
          <el-segmented
            v-if="isNumericPoint && draft.mode === 'point'"
            v-model="draft.evaluationMode"
            :options="evaluationOptions"
            @change="changeEvaluationMode"
          />
        </div>

        <template v-if="draft.evaluationMode === 'highest_matching'">
          <div class="alarm-editor__level-toolbar">
            <button type="button" class="alarm-editor__secondary" @click="addLevel('high')">
              <IconTablerPlus />增加高限
            </button>
            <button type="button" class="alarm-editor__secondary" @click="addLevel('low')">
              <IconTablerPlus />增加低限
            </button>
          </div>
          <div class="alarm-editor__levels">
            <article
              v-for="(condition, index) in draft.conditions"
              :key="condition.id"
              class="alarm-editor__level"
            >
              <el-input
                v-model="condition.label"
                class="alarm-editor__level-label"
                placeholder="高 / 高高 / 自定义"
              />
              <el-select
                v-model="condition.operator"
                class="alarm-editor__direction"
                @change="sortLevels"
                ><el-option label="高限" value="gt" /><el-option label="低限" value="lt"
              /></el-select>
              <el-input-number
                :model-value="numberParam(condition, 'threshold')"
                controls-position="right"
                placeholder="阈值"
                @update:model-value="setThreshold(condition, $event)"
              />
              <el-select v-model="condition.severity" class="alarm-editor__severity"
                ><el-option
                  v-for="severity in severityOptions"
                  :key="severity"
                  :label="alarmSeverityLabels[severity]"
                  :value="severity"
              /></el-select>
              <button
                type="button"
                class="alarm-editor__icon is-danger"
                :disabled="draft.conditions.length === 1"
                @click="removeLevel(index)"
              >
                <IconTablerTrash />
              </button>
              <div v-if="advancedVisible" class="alarm-editor__level-advanced">
                <label
                  >死区<el-input-number
                    v-model="condition.deadband"
                    :min="0"
                    controls-position="right"
                /></label>
                <label
                  >触发延时 ms<el-input-number
                    v-model="condition.triggerDelayMs"
                    :min="0"
                    controls-position="right"
                /></label>
                <label
                  >清除延时 ms<el-input-number
                    v-model="condition.clearDelayMs"
                    :min="0"
                    controls-position="right"
                /></label>
              </div>
            </article>
          </div>
        </template>

        <article v-else class="alarm-editor__single-condition">
          <el-select :model-value="singleCondition.kind" @update:model-value="changeSingleKind"
            ><el-option
              v-for="kind in availableKinds"
              :key="kind"
              :label="alarmConditionLabels[kind]"
              :value="kind"
          /></el-select>
          <el-select v-model="singleCondition.operator"
            ><el-option
              v-for="option in operatorOptions(singleCondition.kind)"
              :key="option.value"
              :label="option.label"
              :value="option.value"
          /></el-select>
          <template v-if="singleCondition.kind === 'threshold'"
            ><el-input-number
              :model-value="numberParam(singleCondition, 'threshold')"
              controls-position="right"
              placeholder="阈值"
              @update:model-value="setParam(singleCondition, 'threshold', $event)"
          /></template>
          <template v-else-if="singleCondition.kind === 'range'"
            ><el-input-number
              :model-value="numberParam(singleCondition, 'lower')"
              placeholder="下限"
              @update:model-value="setParam(singleCondition, 'lower', $event)" /><el-input-number
              :model-value="numberParam(singleCondition, 'upper')"
              placeholder="上限"
              @update:model-value="setParam(singleCondition, 'upper', $event)"
          /></template>
          <template v-else-if="singleCondition.kind === 'state'"
            ><el-select
              v-if="stateUsesBooleanOptions"
              :model-value="Boolean(singleCondition.params.expected)"
              placeholder="期望状态"
              @update:model-value="setParam(singleCondition, 'expected', $event)"
              ><el-option label="true" :value="true" /><el-option
                label="false"
                :value="false" /></el-select
            ><el-input
              v-else
              :model-value="String(singleCondition.params.expected ?? '')"
              placeholder="期望值"
              @update:model-value="setParam(singleCondition, 'expected', $event)"
          /></template>
          <template
            v-else-if="
              singleCondition.kind === 'transition' && singleCondition.operator === 'from_to'
            "
            ><el-select
              v-if="isBooleanPoint"
              :model-value="Boolean(singleCondition.params.from)"
              placeholder="起始值"
              @update:model-value="setParam(singleCondition, 'from', $event)"
              ><el-option label="false" :value="false" /><el-option
                label="true"
                :value="true" /></el-select
            ><el-input
              v-else
              :model-value="String(singleCondition.params.from ?? '')"
              placeholder="起始值"
              @update:model-value="setParam(singleCondition, 'from', $event)" /><el-select
              v-if="isBooleanPoint"
              :model-value="Boolean(singleCondition.params.to)"
              placeholder="目标值"
              @update:model-value="setParam(singleCondition, 'to', $event)"
              ><el-option label="false" :value="false" /><el-option
                label="true"
                :value="true" /></el-select
            ><el-input
              v-else
              :model-value="String(singleCondition.params.to ?? '')"
              placeholder="目标值"
              @update:model-value="setParam(singleCondition, 'to', $event)"
          /></template>
          <template v-else-if="singleCondition.kind === 'text_match'"
            ><el-input
              :model-value="String(singleCondition.params.expected || '')"
              placeholder="匹配文本"
              @update:model-value="setParam(singleCondition, 'expected', $event)"
          /></template>
          <template v-else-if="singleCondition.kind === 'rate_of_change'"
            ><el-input-number
              :model-value="numberParam(singleCondition, 'limit')"
              placeholder="变化率"
              @update:model-value="setParam(singleCondition, 'limit', $event)" /><el-input-number
              :model-value="numberParam(singleCondition, 'windowMs')"
              :min="1"
              placeholder="窗口 ms"
              @update:model-value="setParam(singleCondition, 'windowMs', $event)"
          /></template>
          <template v-else-if="singleCondition.kind === 'deviation'"
            ><el-input-number
              :model-value="numberParam(singleCondition, 'baseline')"
              placeholder="基准值"
              @update:model-value="setParam(singleCondition, 'baseline', $event)" /><el-input-number
              :model-value="numberParam(singleCondition, 'limit')"
              :min="0"
              placeholder="偏差"
              @update:model-value="setParam(singleCondition, 'limit', $event)"
          /></template>
          <el-select v-model="singleCondition.severity"
            ><el-option
              v-for="severity in severityOptions"
              :key="severity"
              :label="alarmSeverityLabels[severity]"
              :value="severity"
          /></el-select>
          <div v-if="advancedVisible" class="alarm-editor__level-advanced">
            <label>死区<el-input-number v-model="singleCondition.deadband" :min="0" /></label
            ><label
              >触发延时 ms<el-input-number
                v-model="singleCondition.triggerDelayMs"
                :min="0" /></label
            ><label
              >清除延时 ms<el-input-number v-model="singleCondition.clearDelayMs" :min="0"
            /></label>
          </div>
        </article>
      </section>

      <section class="alarm-editor__section is-collapsible">
        <button
          type="button"
          class="alarm-editor__collapse"
          @click="advancedVisible = !advancedVisible"
        >
          <span>高级设置</span><IconTablerChevronDown :class="{ 'is-open': advancedVisible }" />
        </button>
        <div v-if="advancedVisible" class="alarm-editor__advanced">
          <label class="alarm-editor__field"
            ><span>通知</span
            ><el-select v-model="draft.notification.mode" @change="changeNotificationMode"
              ><el-option label="沿用工程通知设置" value="inherit" /><el-option
                label="不发送通知"
                value="off" /><el-option label="单独设置" value="custom" /></el-select
          ></label>
          <template v-if="draft.notification.mode === 'custom'"
            ><div class="alarm-editor__checks">
              <el-checkbox v-model="draft.notification.notifyOnRaise">触发时通知</el-checkbox
              ><el-checkbox v-model="draft.notification.notifyOnClear">恢复时通知</el-checkbox>
            </div>
            <label class="alarm-editor__field"
              ><span>通知渠道</span
              ><el-select v-model="draft.notification.channelIds" multiple
                ><el-option
                  v-for="channel in enabledChannels"
                  :key="channel.id"
                  :label="channel.name"
                  :value="channel.id" /></el-select></label
          ></template>
        </div>
      </section>

      <section class="alarm-editor__section is-collapsible">
        <button type="button" class="alarm-editor__collapse" @click="debugVisible = !debugVisible">
          <span>开发调试</span><IconTablerChevronDown :class="{ 'is-open': debugVisible }" />
        </button>
        <div v-if="debugVisible" class="alarm-editor__debug">
          <div v-if="draft.mode === 'derived'" class="alarm-editor__debug-inputs">
            <label v-for="point in draft.selectedPoints" :key="point.datapointId"
              ><span>{{ point.inputKey || point.name }}</span
              ><el-input
                v-model="derivedTrialInputs[point.inputKey || point.datapointId]"
                placeholder="模拟值"
            /></label>
          </div>
          <el-input v-else v-model="trialValues" placeholder="输入模拟值，多个值用逗号分隔" />
          <label v-if="needsSampleInterval" class="alarm-editor__trial-interval"
            ><span>相邻模拟值间隔 ms</span
            ><el-input-number v-model="trialIntervalMs" :min="1" controls-position="right" /></label
          ><button
            type="button"
            class="alarm-editor__secondary"
            :disabled="trialLoading"
            @click="runTrial"
          >
            试算</button
          ><button
            v-if="item"
            type="button"
            class="alarm-editor__secondary"
            :disabled="contractLoading"
            @click="loadContract"
          >
            查看契约
          </button>
          <pre v-if="debugOutput">{{ debugOutput }}</pre>
        </div>
      </section>

      <footer class="alarm-editor__footer">
        <button type="button" class="alarm-editor__cancel" @click="visible = false">取消</button
        ><button type="submit" class="alarm-editor__primary" :disabled="saving">
          <IconTablerDeviceFloppy />{{ saving ? '保存中' : '保存' }}
        </button>
      </footer>
    </form>

    <AlarmDatapointManagerDialog
      v-model="pointPickerVisible"
      :project-id="projectId"
      :mode="draft.mode"
      :points="draft.selectedPoints"
      @apply="replacePoints"
    />
  </DcDrawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import IconTablerChevronDown from '~icons/tabler/chevron-down'
import IconTablerDeviceFloppy from '~icons/tabler/device-floppy'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerX from '~icons/tabler/x'
import AlarmDatapointManagerDialog from './AlarmDatapointManagerDialog.vue'
import DcDrawer from '@/components/shared/DcDrawer.vue'
import type {
  AlarmCondition,
  AlarmConditionKind,
  AlarmItem,
  AlarmItemMode,
  AlarmItemSave,
  AlarmGroup,
  AlarmNotificationChannel,
  AlarmSeverity,
} from '@/api/schemas/alarm.schema'
import { AlarmItemSaveSchema } from '@/api/schemas/alarm.schema'
import {
  getAlarmItemContract,
  testAlarmItem,
  validateBatchCreateAlarmItems,
  validateAlarmItem,
} from '@/api/alarm.api'
import {
  alarmConditionLabels,
  alarmPointCategory,
  alarmSeverityLabels,
  buildAlarmItemPayload,
  compatibleAlarmPoints,
  conditionKindsFor,
  alarmItemToDraft,
  createAlarmCondition,
  createAlarmItemDraft,
  sortAlarmLevels,
  type AlarmItemDraft,
  type AlarmPointSelection,
} from '@/models/alarm-policy'
import { getApiErrorMessage } from '@/utils/request'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    projectId: string
    mode?: AlarmItemMode
    item?: AlarmItem | null
    groups?: AlarmGroup[]
    channels?: AlarmNotificationChannel[]
    initialPoints?: AlarmPointSelection[]
    saving?: boolean
  }>(),
  {
    mode: 'point',
    item: null,
    groups: () => [],
    channels: () => [],
    initialPoints: () => [],
    saving: false,
  },
)
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [payload: AlarmItemSave, datapointIds: string[]]
  'open-conflict': [id: string]
}>()
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const draft = ref<AlarmItemDraft>(createAlarmItemDraft(props.mode))
const pointPickerVisible = ref(false)
const advancedVisible = ref(false)
const debugVisible = ref(false)
const trialValues = ref('')
const derivedTrialInputs = ref<Record<string, string>>({})
const trialIntervalMs = ref(1000)
const debugOutput = ref('')
const trialLoading = ref(false)
const contractLoading = ref(false)
const severityOptions: AlarmSeverity[] = ['info', 'warning', 'major', 'critical']
const evaluationOptions = [
  { label: '越限报警', value: 'highest_matching' },
  { label: '其他报警类型', value: 'single' },
]
const drawerTitle = computed(() =>
  props.item
    ? `编辑 · ${props.item.displayName}`
    : props.mode === 'derived'
      ? '新建组合报警'
      : '新建报警',
)
const enabledChannels = computed(() => props.channels.filter((channel) => channel.isEnabled))
const isNumericPoint = computed(
  () =>
    !draft.value.selectedPoints.length ||
    alarmPointCategory(draft.value.selectedPoints[0]?.dataType) === 'number',
)
const isBooleanPoint = computed(
  () =>
    draft.value.mode === 'point' &&
    alarmPointCategory(draft.value.selectedPoints[0]?.dataType) === 'boolean',
)
const stateUsesBooleanOptions = computed(
  () => draft.value.mode === 'derived' || isBooleanPoint.value,
)
const needsSampleInterval = computed(() =>
  draft.value.conditions.some(
    (condition) => condition.kind === 'rate_of_change' || condition.kind === 'transition',
  ),
)
const showIdentityFields = computed(() => Boolean(props.item) || draft.value.mode === 'derived')
const availableKinds = computed(() =>
  conditionKindsFor(draft.value.selectedPoints, draft.value.mode, draft.value.evaluationMode),
)
const singleCondition = computed(() => draft.value.conditions[0] || createAlarmCondition())
const pointPreview = computed(
  () =>
    draft.value.selectedPoints
      .slice(0, 3)
      .map((point) => point.name || point.path)
      .join('、') + (draft.value.selectedPoints.length > 3 ? '…' : ''),
)
const selectedPointCount = computed(() => draft.value.selectedPoints.length)
const canManagePoints = computed(() => !props.item || draft.value.mode === 'derived')
const conditionHint = computed(() =>
  draft.value.evaluationMode === 'highest_matching'
    ? '当前配置处理一组越限等级；同一数据点还可以另建变化率、离线等报警。'
    : '当前配置只处理一种触发语义；同一数据点可以继续创建其他类型报警。',
)

watch(
  () => props.modelValue,
  (opened) => {
    if (!opened) return
    const next = props.item ? alarmItemToDraft(props.item) : createAlarmItemDraft(props.mode)
    if (!props.item && props.initialPoints.length)
      next.selectedPoints = props.initialPoints.map((item) => ({ ...item }))
    draft.value = next
    advancedVisible.value = false
    debugVisible.value = false
    debugOutput.value = ''
    derivedTrialInputs.value = Object.fromEntries(
      next.selectedPoints.map((point) => [point.inputKey || point.datapointId, '']),
    )
  },
)

function replacePoints(points: AlarmPointSelection[]) {
  draft.value.selectedPoints = points
  if (
    draft.value.mode === 'point' &&
    !isNumericPoint.value &&
    draft.value.evaluationMode === 'highest_matching'
  )
    changeEvaluationMode('single')
}
function changeEvaluationMode(value: string | number | boolean) {
  const mode = String(value) as 'single' | 'highest_matching'
  draft.value.evaluationMode = mode
  draft.value.conditions = [
    mode === 'highest_matching'
      ? createAlarmCondition('threshold', '高')
      : createAlarmCondition(availableKinds.value[0]),
  ]
}
function addLevel(direction: 'high' | 'low') {
  const sameDirection = draft.value.conditions.filter((item) =>
    direction === 'high'
      ? ['gt', 'gte'].includes(item.operator)
      : ['lt', 'lte'].includes(item.operator),
  ).length
  const condition = createAlarmCondition(
    'threshold',
    direction === 'high'
      ? sameDirection
        ? `高${'高'.repeat(sameDirection)}`
        : '高'
      : sameDirection
        ? `低${'低'.repeat(sameDirection)}`
        : '低',
  )
  condition.operator = direction === 'high' ? 'gt' : 'lt'
  draft.value.conditions.push(condition)
  sortLevels()
}
function removeLevel(index: number) {
  if (draft.value.conditions.length > 1) draft.value.conditions.splice(index, 1)
}
function sortLevels() {
  draft.value.conditions = sortAlarmLevels(draft.value.conditions)
}
function setThreshold(condition: AlarmCondition, value: number | undefined) {
  setParam(condition, 'threshold', value)
  sortLevels()
}
function changeSingleKind(kind: AlarmConditionKind) {
  draft.value.conditions = [
    createAlarmCondition(kind, draft.value.mode === 'derived' ? '结果条件' : ''),
  ]
}
function setParam(condition: AlarmCondition, key: string, value: unknown) {
  condition.params = { ...condition.params, [key]: value }
}
function numberParam(condition: AlarmCondition, key: string) {
  const value = condition.params[key]
  return typeof value === 'number' ? value : undefined
}
function formatPointDataType(dataType?: string) {
  return String(dataType || '未知类型').toLowerCase()
}
function changeNotificationMode(mode: string | number | boolean) {
  if (mode !== 'custom') return
  draft.value.notification.notifyOnRaise ??= true
  draft.value.notification.notifyOnClear ??= true
  if (!draft.value.notification.channelIds.length)
    draft.value.notification.channelIds = enabledChannels.value.some(
      (channel) => channel.id === 'runtime_inapp',
    )
      ? ['runtime_inapp']
      : []
}
function operatorOptions(kind: AlarmConditionKind) {
  const options: Record<AlarmConditionKind, Array<{ label: string; value: string }>> = {
    threshold: [
      { label: '高于', value: 'gt' },
      { label: '大于等于', value: 'gte' },
      { label: '低于', value: 'lt' },
      { label: '小于等于', value: 'lte' },
    ],
    range: [
      { label: '超出区间', value: 'outside' },
      { label: '进入区间', value: 'between' },
    ],
    state: [
      { label: '等于', value: 'eq' },
      { label: '不等于', value: 'ne' },
    ],
    transition: [
      { label: '发生变化', value: 'changed' },
      { label: '上升沿', value: 'rising' },
      { label: '下降沿', value: 'falling' },
      { label: '指定状态变化', value: 'from_to' },
    ],
    text_match: [
      { label: '包含', value: 'contains' },
      { label: '等于', value: 'eq' },
      { label: '不等于', value: 'ne' },
      { label: '正则匹配', value: 'regex' },
    ],
    rate_of_change: [
      { label: '大于', value: 'gt' },
      { label: '大于等于', value: 'gte' },
      { label: '小于', value: 'lt' },
      { label: '小于等于', value: 'lte' },
    ],
    deviation: [{ label: '大于', value: 'gt' }],
    offline: [{ label: '离线', value: 'is_offline' }],
    expression: [{ label: '结果为真', value: 'is_true' }],
  }
  return options[kind]
}

function localValidate() {
  if (draft.value.mode === 'derived' && !draft.value.displayName.trim()) return '请填写组合报警名称'
  if (!selectedPointCount.value) return '请至少选择一个数据点'
  if (draft.value.mode === 'point' && !compatibleAlarmPoints(draft.value.selectedPoints))
    return '所选数据点类型不兼容'
  if (draft.value.mode === 'derived' && draft.value.selectedPoints.length < 2)
    return '组合报警至少需要两个输入点'
  if (draft.value.mode === 'derived' && !draft.value.derivedExpression.trim())
    return '请填写组合表达式'
  if (
    draft.value.mode === 'derived' &&
    draft.value.selectedPoints.some(
      (point) => !/^[A-Za-z_][A-Za-z0-9_]*$/.test(point.inputKey?.trim() || ''),
    )
  )
    return '表达式变量名必须以字母或下划线开头，且只能包含字母、数字和下划线'
  if (draft.value.notification.mode === 'custom' && !draft.value.notification.channelIds.length)
    return '单独设置通知时请至少选择一个渠道'
  return ''
}
async function submit() {
  const message = localValidate()
  if (message) return ElMessage.warning(message)
  let payload = AlarmItemSaveSchema.parse(buildAlarmItemPayload(draft.value))
  // 多点创建时名称和说明不共享。由服务端按各数据点和报警类型生成默认显示名称，后续可单项编辑。
  if (!props.item && payload.mode === 'point') {
    payload = { ...payload, displayName: '', description: null }
  }
  try {
    const datapointIds = draft.value.selectedPoints.map((point) => point.datapointId)
    const validation =
      !props.item && payload.mode === 'point'
        ? await validateBatchCreateAlarmItems(props.projectId, {
            datapointIds,
            draft: { ...payload, datapointId: undefined },
          })
        : await validateAlarmItem(props.projectId, payload)
    if (validation.errors.length) {
      const conflict = validation.errors[0]
      const conflictPoints = Array.from(
        new Set(validation.errors.map((item) => item.datapointName).filter(Boolean)),
      )
      const conflictPreview = conflictPoints.slice(0, 5).join('、')
      const conflictSuffix =
        conflictPoints.length > 5 ? ` 等 ${conflictPoints.length} 个数据点` : conflictPreview
      await ElMessageBox.alert(
        `${conflict.message}${conflictPreview ? `：${conflictSuffix}` : ''}`,
        '无法保存',
        { confirmButtonText: conflict.conflictAlarmItemId ? '打开已有报警' : '知道了' },
      )
      if (conflict.conflictAlarmItemId) emit('open-conflict', conflict.conflictAlarmItemId)
      return
    }
    if (validation.warnings.length) {
      const warningPoints = Array.from(
        new Set(validation.warnings.map((item) => item.datapointName).filter(Boolean)),
      )
      const total = warningPoints.length || validation.warnings.length
      const preview = warningPoints.slice(0, 5).join('、')
      await ElMessageBox.confirm(
        `该条件与已有报警重叠，影响 ${total} 个数据点${preview ? `（${preview}${total > 5 ? ' 等' : ''}）` : ''}。确认后两个报警未来可同时活动，是否继续？`,
        '重叠条件确认',
        { confirmButtonText: '确认保存', cancelButtonText: '返回修改', type: 'warning' },
      )
      payload = {
        ...payload,
        acknowledgedWarningKeys: validation.warnings.map((item) => item.ackKey).filter(Boolean),
      }
    }
    emit('save', payload, datapointIds)
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(getApiErrorMessage(error, '校验报警配置失败'))
  }
}
async function runTrial() {
  const message = localValidate()
  if (message) return ElMessage.warning(message)
  trialLoading.value = true
  try {
    const values =
      draft.value.mode === 'derived'
        ? [
            Object.fromEntries(
              draft.value.selectedPoints.map((point) => [
                point.inputKey || point.datapointId,
                parseTrialValue(
                  derivedTrialInputs.value[point.inputKey || point.datapointId] || '',
                ),
              ]),
            ),
          ]
        : trialValues.value
            .split(',')
            .map((item) => item.trim())
            .filter(Boolean)
            .map(parseTrialValue)
    debugOutput.value = JSON.stringify(
      await testAlarmItem(
        props.projectId,
        {
          ...buildAlarmItemPayload(draft.value),
        },
        values,
        needsSampleInterval.value ? { sampleIntervalMs: trialIntervalMs.value } : {},
      ),
      null,
      2,
    )
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '报警试算失败'))
  } finally {
    trialLoading.value = false
  }
}
function parseTrialValue(value: string): string | number | boolean {
  if (value === 'true') return true
  if (value === 'false') return false
  return Number.isNaN(Number(value)) || value.trim() === '' ? value : Number(value)
}
async function loadContract() {
  if (!props.item) return
  contractLoading.value = true
  try {
    debugOutput.value = JSON.stringify(
      await getAlarmItemContract(props.projectId, props.item.id),
      null,
      2,
    )
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载报警契约失败'))
  } finally {
    contractLoading.value = false
  }
}
</script>

<style scoped>
.alarm-editor {
  min-height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
  color: var(--dc-text);
}
.alarm-editor__section {
  display: grid;
  gap: 12px;
}
.alarm-editor__section + .alarm-editor__section {
  padding-top: 16px;
  border-top: 1px solid var(--dc-border);
}
.alarm-editor__impact {
  padding: 10px 12px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 22%, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  font-size: 12px;
  color: var(--dc-text-secondary);
}
.alarm-editor__grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 190px 72px;
  gap: 12px;
  align-items: end;
}
.alarm-editor__grid.is-point-create {
  grid-template-columns: minmax(0, 1fr) 72px;
}
.alarm-editor__field {
  display: grid;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.alarm-editor__field.is-switch {
  justify-items: start;
}
.alarm-editor__section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.alarm-editor__section-head h3 {
  margin: 0;
  font-size: 14px;
}
.alarm-editor__section-head p {
  margin: 4px 0 0;
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 18px;
}
.alarm-editor__secondary,
.alarm-editor__primary,
.alarm-editor__cancel {
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 0 12px;
  border-radius: var(--dc-radius-sm);
  font: inherit;
  font-size: 12px;
  font-weight: 700;
}
.alarm-editor__secondary,
.alarm-editor__cancel {
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}
.alarm-editor__primary {
  border: 1px solid var(--dc-primary);
  background: var(--dc-primary);
  color: white;
}
.alarm-editor__secondary svg,
.alarm-editor__primary svg {
  width: 15px;
}
.alarm-editor__point-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}
.alarm-editor__point-summary div {
  min-width: 0;
  display: grid;
  gap: 3px;
}
.alarm-editor__point-summary span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--dc-text-muted);
  font-size: 12px;
}
.alarm-editor__point-summary small {
  color: var(--dc-text-secondary);
  font-size: 11px;
  line-height: 16px;
}
.alarm-editor__point-summary button {
  border: 0;
  background: none;
  color: var(--dc-primary);
  font: inherit;
  font-size: 12px;
  font-weight: 700;
}
.alarm-editor__empty {
  padding: 20px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-sm);
  text-align: center;
  color: var(--dc-text-muted);
  font-size: 12px;
}
.alarm-editor__aliases {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}
.alarm-editor__aliases label {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 140px;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  background: var(--dc-surface-muted);
  border-radius: var(--dc-radius-sm);
  font-size: 12px;
}
.alarm-editor__aliases span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.alarm-editor__aliases small {
  margin-left: 5px;
  color: var(--dc-text-muted);
  font-size: 11px;
}
.alarm-editor__level-toolbar {
  display: flex;
  gap: 8px;
}
.alarm-editor__levels {
  display: grid;
  gap: 8px;
}
.alarm-editor__level {
  display: grid;
  grid-template-columns: 100px 90px minmax(120px, 1fr) 100px 30px;
  gap: 8px;
  align-items: center;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}
.alarm-editor__level-advanced {
  grid-column: 1/-1;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--dc-border);
}
.alarm-editor__level-advanced label {
  display: grid;
  gap: 5px;
  color: var(--dc-text-muted);
  font-size: 11px;
}
.alarm-editor__single-condition {
  display: grid;
  grid-template-columns: 120px 110px repeat(2, minmax(110px, 1fr)) 100px;
  gap: 8px;
  align-items: center;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}
.alarm-editor__single-condition .alarm-editor__level-advanced {
  grid-column: 1/-1;
}
.alarm-editor__icon {
  width: 30px;
  height: 30px;
  display: inline-grid;
  place-items: center;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
}
.alarm-editor__icon svg {
  width: 16px;
}
.alarm-editor__icon.is-danger {
  color: var(--dc-danger);
}
.alarm-editor__collapse {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0;
  border: 0;
  background: none;
  color: var(--dc-text);
  font: inherit;
  font-size: 13px;
  font-weight: 700;
}
.alarm-editor__collapse svg {
  width: 16px;
  transition: transform 0.15s;
}
.alarm-editor__collapse svg.is-open {
  transform: rotate(180deg);
}
.alarm-editor__advanced {
  display: grid;
  gap: 12px;
  padding-top: 4px;
}
.alarm-editor__checks {
  display: flex;
  gap: 20px;
}
.alarm-editor__debug {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  gap: 8px;
}
.alarm-editor__debug-inputs {
  grid-column: 1/-1;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}
.alarm-editor__debug-inputs label,
.alarm-editor__trial-interval {
  display: grid;
  gap: 5px;
  color: var(--dc-text-muted);
  font-size: 11px;
}
.alarm-editor__trial-interval {
  min-width: 140px;
}
.alarm-editor__debug pre {
  grid-column: 1/-1;
  max-height: 240px;
  overflow: auto;
  margin: 0;
  padding: 10px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  font-size: 11px;
}
.alarm-editor__footer {
  position: sticky;
  bottom: 0;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: auto;
  padding: 14px 0 2px;
  background: var(--dc-surface-raised);
  border-top: 1px solid var(--dc-border);
}
@media (max-width: 760px) {
  .alarm-editor__grid,
  .alarm-editor__aliases {
    grid-template-columns: 1fr;
  }
  .alarm-editor__level,
  .alarm-editor__single-condition {
    grid-template-columns: 1fr 1fr;
  }
  .alarm-editor__level-advanced {
    grid-template-columns: 1fr;
  }
  .alarm-editor__debug {
    grid-template-columns: 1fr;
  }
  .alarm-editor__debug > * {
    grid-column: 1;
  }
  .alarm-editor__debug-inputs {
    grid-template-columns: 1fr;
  }
}
</style>
