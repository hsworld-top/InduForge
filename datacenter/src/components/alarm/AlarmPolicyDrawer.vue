<template>
  <DcDrawer v-model="visible" :title="drawerTitle" :width="640" :max="760">
    <template #actions>
      <button
        type="button"
        class="alarm-editor__close"
        title="关闭"
        aria-label="关闭报警设置"
        @click="visible = false"
      >
        <IconTablerX />
      </button>
    </template>

    <form class="alarm-editor" @submit.prevent="submit">
      <section class="alarm-editor__section">
        <div class="alarm-editor__grid">
          <label class="alarm-editor__field is-wide">
            <span>名称</span>
            <el-input v-model="draft.name" maxlength="100" placeholder="例如：反应釜温度过高" />
          </label>
          <label class="alarm-editor__field">
            <span>目录</span>
            <el-select v-model="draft.groupId" clearable placeholder="根目录">
              <el-option label="根目录" :value="null" />
              <el-option
                v-for="group in groups"
                :key="group.id"
                :label="group.fullPath || group.name"
                :value="group.id"
              />
            </el-select>
          </label>
          <label class="alarm-editor__field alarm-editor__switch-field">
            <span>保存后启用</span>
            <el-switch v-model="draft.isEnabled" />
          </label>
        </div>
      </section>

      <section class="alarm-editor__section">
        <div class="alarm-editor__section-head">
          <div>
            <h3>{{ draft.mode === 'derived' ? '输入数据点' : '报警数据点' }}</h3>
            <p>只应用到当前明确选择的数据点，后续新增点不会自动加入。</p>
          </div>
          <button type="button" class="alarm-editor__secondary" @click="pointPickerVisible = true">
            <IconTablerPlus />
            {{ draft.bindings.length ? '管理数据点' : '选择数据点' }}
          </button>
        </div>
        <div v-if="draft.bindings.length" class="alarm-editor__points">
          <div v-if="bindingsCollapsed" class="alarm-editor__point-summary">
            <span
              ><strong>已选择 {{ draft.bindings.length }} 个</strong>，当前显示前 3 个</span
            >
            <button type="button" @click="pointPickerVisible = true">查看全部</button>
          </div>
          <div
            v-for="binding in visibleBindings"
            :key="binding.datapointId"
            class="alarm-editor__point"
          >
            <div class="alarm-editor__point-main">
              <strong>{{ binding.name || binding.path }}</strong>
              <code>{{ binding.path }}</code>
            </div>
            <input
              v-if="draft.mode === 'derived'"
              v-model="binding.inputKey"
              class="alarm-editor__key"
              aria-label="输入变量名"
              placeholder="变量名"
            />
            <span class="alarm-editor__type">{{ binding.dataType || '-' }}</span>
            <button
              type="button"
              class="alarm-editor__icon"
              title="移除数据点"
              @click="removeBinding(binding.datapointId)"
            >
              <IconTablerX />
            </button>
          </div>
        </div>
        <div v-else class="alarm-editor__empty">请选择要配置报警的数据点</div>
      </section>

      <section v-if="draft.mode === 'derived'" class="alarm-editor__section">
        <div class="alarm-editor__section-head">
          <div>
            <h3>组合表达式</h3>
            <p>使用上方变量名组合多个输入点。</p>
          </div>
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
            <h3>报警条件</h3>
            <p>每条条件独立判断，满足任意一条即可触发对应等级报警。</p>
          </div>
          <el-dropdown trigger="click" @command="addCondition">
            <button type="button" class="alarm-editor__secondary">
              <IconTablerPlus />
              添加条件
            </button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-for="kind in availableKinds" :key="kind" :command="kind">
                  {{ alarmConditionLabels[kind] }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>

        <div class="alarm-editor__conditions">
          <article
            v-for="(condition, index) in draft.conditions"
            :key="condition.id"
            class="alarm-editor__condition"
          >
            <div class="alarm-editor__condition-main">
              <el-select
                :model-value="condition.kind"
                class="alarm-editor__kind"
                @update:model-value="changeConditionKind(index, $event)"
              >
                <el-option
                  v-for="kind in availableKinds"
                  :key="kind"
                  :label="alarmConditionLabels[kind]"
                  :value="kind"
                />
              </el-select>
              <el-select v-model="condition.operator" class="alarm-editor__operator">
                <el-option
                  v-for="option in operatorOptions(condition.kind)"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>

              <template v-if="condition.kind === 'threshold'">
                <el-input-number
                  :model-value="numberParam(condition, 'threshold')"
                  controls-position="right"
                  placeholder="阈值"
                  @update:model-value="setParam(condition, 'threshold', $event)"
                />
              </template>
              <template v-else-if="condition.kind === 'range'">
                <el-input-number
                  :model-value="numberParam(condition, 'lower')"
                  controls-position="right"
                  placeholder="下限"
                  @update:model-value="setParam(condition, 'lower', $event)"
                />
                <el-input-number
                  :model-value="numberParam(condition, 'upper')"
                  controls-position="right"
                  placeholder="上限"
                  @update:model-value="setParam(condition, 'upper', $event)"
                />
              </template>
              <template v-else-if="condition.kind === 'state'">
                <el-select
                  :model-value="condition.params.expected"
                  placeholder="目标状态"
                  @update:model-value="setParam(condition, 'expected', $event)"
                >
                  <el-option label="true" :value="true" />
                  <el-option label="false" :value="false" />
                </el-select>
              </template>
              <template v-else-if="condition.kind === 'text_match'">
                <el-input
                  :model-value="String(condition.params.expected || '')"
                  placeholder="匹配文本"
                  @update:model-value="setParam(condition, 'expected', $event)"
                />
              </template>
              <template v-else-if="condition.kind === 'rate_of_change'">
                <el-input-number
                  :model-value="numberParam(condition, 'limit')"
                  controls-position="right"
                  placeholder="变化率"
                  @update:model-value="setParam(condition, 'limit', $event)"
                />
                <el-input-number
                  :model-value="numberParam(condition, 'windowMs')"
                  :min="1"
                  controls-position="right"
                  placeholder="窗口 ms"
                  @update:model-value="setParam(condition, 'windowMs', $event)"
                />
              </template>
              <template v-else-if="condition.kind === 'deviation'">
                <el-input-number
                  :model-value="numberParam(condition, 'baseline')"
                  controls-position="right"
                  placeholder="基准值"
                  @update:model-value="setParam(condition, 'baseline', $event)"
                />
                <el-input-number
                  :model-value="numberParam(condition, 'limit')"
                  :min="0"
                  controls-position="right"
                  placeholder="偏差值"
                  @update:model-value="setParam(condition, 'limit', $event)"
                />
              </template>
              <template v-else-if="condition.kind === 'expression'">
                <el-input
                  :model-value="String(condition.params.expression || '')"
                  placeholder="结果表达式"
                  @update:model-value="setParam(condition, 'expression', $event)"
                />
              </template>

              <el-select v-model="condition.severity" class="alarm-editor__severity">
                <el-option
                  v-for="severity in severityOptions"
                  :key="severity"
                  :label="alarmSeverityLabels[severity]"
                  :value="severity"
                />
              </el-select>
              <button
                type="button"
                class="alarm-editor__icon is-danger"
                title="删除条件"
                :disabled="draft.conditions.length === 1"
                @click="removeCondition(index)"
              >
                <IconTablerTrash />
              </button>
            </div>

            <div v-if="advancedVisible" class="alarm-editor__condition-advanced">
              <label
                >死区
                <el-input-number v-model="condition.deadband" :min="0" controls-position="right"
              /></label>
              <label
                >触发延时 ms
                <el-input-number
                  v-model="condition.triggerDelayMs"
                  :min="0"
                  controls-position="right"
              /></label>
              <label
                >清除延时 ms
                <el-input-number
                  v-model="condition.clearDelayMs"
                  :min="0"
                  controls-position="right"
              /></label>
            </div>
          </article>
        </div>
      </section>

      <section class="alarm-editor__section is-collapsible">
        <button
          type="button"
          class="alarm-editor__collapse"
          @click="advancedVisible = !advancedVisible"
        >
          <span>高级设置</span>
          <IconTablerChevronDown :class="{ 'is-open': advancedVisible }" />
        </button>
        <div v-if="advancedVisible" class="alarm-editor__advanced">
          <label class="alarm-editor__field">
            <span>通知</span>
            <el-select v-model="draft.notification.mode" @change="changeNotificationMode">
              <el-option label="沿用工程通知设置" value="inherit" />
              <el-option label="不发送通知" value="off" />
              <el-option label="单独设置" value="custom" />
            </el-select>
          </label>
          <template v-if="draft.notification.mode === 'custom'">
            <div class="alarm-editor__checks">
              <el-checkbox v-model="draft.notification.notifyOnRaise">触发时通知</el-checkbox>
              <el-checkbox v-model="draft.notification.notifyOnClear">恢复时通知</el-checkbox>
            </div>
            <label class="alarm-editor__field">
              <span>通知渠道</span>
              <el-select v-model="draft.notification.channelIds" multiple placeholder="选择渠道">
                <el-option
                  v-for="channel in enabledChannels"
                  :key="channel.id"
                  :label="channel.name"
                  :value="channel.id"
                />
              </el-select>
            </label>
            <label class="alarm-editor__field">
              <span>重复提醒（秒，留空关闭）</span>
              <el-input-number
                v-model="draft.notification.repeatIntervalSeconds"
                :min="1"
                controls-position="right"
              />
            </label>
            <label class="alarm-editor__field is-wide">
              <span>消息模板</span>
              <el-input
                v-model="draft.notification.messageTemplate"
                type="textarea"
                :rows="2"
                placeholder="留空使用工程默认模板"
              />
            </label>
          </template>
        </div>
      </section>

      <section v-if="policy?.id" class="alarm-editor__section is-collapsible">
        <button type="button" class="alarm-editor__collapse" @click="debugVisible = !debugVisible">
          <span>开发调试</span>
          <IconTablerChevronDown :class="{ 'is-open': debugVisible }" />
        </button>
        <div v-if="debugVisible" class="alarm-editor__debug">
          <el-input v-model="trialValue" placeholder="输入模拟值" />
          <button
            type="button"
            class="alarm-editor__secondary"
            :disabled="trialLoading"
            @click="runTrial"
          >
            试算
          </button>
          <button
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
        <button type="button" class="alarm-editor__cancel" @click="visible = false">取消</button>
        <button type="button" class="alarm-editor__primary" :disabled="saving" @click="submit">
          <IconTablerDeviceFloppy />
          {{ saving ? '保存中' : '保存' }}
        </button>
      </footer>
    </form>

    <AlarmDatapointManagerDialog
      v-model="pointPickerVisible"
      :project-id="projectId"
      :mode="draft.mode"
      :bindings="draft.bindings"
      @apply="replaceBindings"
    />
  </DcDrawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import IconTablerChevronDown from '~icons/tabler/chevron-down'
import IconTablerDeviceFloppy from '~icons/tabler/device-floppy'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerX from '~icons/tabler/x'
import AlarmDatapointManagerDialog from './AlarmDatapointManagerDialog.vue'
import DcDrawer from '@/components/shared/DcDrawer.vue'
import type {
  AlarmBinding,
  AlarmCondition,
  AlarmConditionKind,
  AlarmNotificationChannel,
  AlarmPolicy,
  AlarmPolicyGroup,
  AlarmPolicyMode,
  AlarmPolicySave,
  AlarmSeverity,
} from '@/api/schemas/alarm.schema'
import { AlarmPolicySaveSchema } from '@/api/schemas/alarm.schema'
import { getAlarmPolicyContract, testAlarmPolicy } from '@/api/alarm.api'
import {
  alarmBindingPreview,
  alarmConditionLabels,
  alarmSeverityLabels,
  compatibleAlarmBindings,
  conditionKindsFor,
  createAlarmCondition,
  createAlarmPolicyDraft,
  normalizeAlarmConditionsForBindings,
  policyToDraft,
} from '@/models/alarm-policy'
import { getApiErrorMessage } from '@/utils/request'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    projectId: string
    mode?: AlarmPolicyMode
    policy?: AlarmPolicy | null
    groups?: AlarmPolicyGroup[]
    channels?: AlarmNotificationChannel[]
    initialBindings?: AlarmPolicySave['bindings']
    saving?: boolean
  }>(),
  {
    mode: 'per_target',
    policy: null,
    groups: () => [],
    channels: () => [],
    initialBindings: () => [],
    saving: false,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [payload: AlarmPolicySave]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const draft = ref<AlarmPolicySave>(createAlarmPolicyDraft(props.mode))
const pointPickerVisible = ref(false)
const advancedVisible = ref(false)
const debugVisible = ref(false)
const trialValue = ref('')
const debugOutput = ref('')
const trialLoading = ref(false)
const contractLoading = ref(false)
const severityOptions: AlarmSeverity[] = ['info', 'warning', 'major', 'critical']

const drawerTitle = computed(() =>
  props.policy
    ? `编辑 · ${props.policy.name}`
    : props.mode === 'derived'
      ? '新建组合报警'
      : '新建报警',
)
const availableKinds = computed(() => conditionKindsFor(draft.value.bindings, draft.value.mode))
const enabledChannels = computed(() => props.channels.filter((channel) => channel.isEnabled))
const bindingPreview = computed(() => alarmBindingPreview(draft.value.bindings, draft.value.mode))
const bindingsCollapsed = computed(() => bindingPreview.value.collapsed)
const visibleBindings = computed(() => bindingPreview.value.visible)

watch(
  () => props.modelValue,
  (opened) => {
    if (!opened) return
    const next = props.policy ? policyToDraft(props.policy) : createAlarmPolicyDraft(props.mode)
    if (!props.policy && props.initialBindings.length)
      next.bindings = props.initialBindings.map((item) => ({ ...item }))
    next.conditions = normalizeAlarmConditionsForBindings(next.conditions, next.bindings, next.mode)
    draft.value = next
    advancedVisible.value = false
    debugVisible.value = false
    debugOutput.value = ''
  },
)

function replaceBindings(bindings: AlarmBinding[]) {
  draft.value.bindings = bindings
  draft.value.conditions = normalizeAlarmConditionsForBindings(
    draft.value.conditions,
    draft.value.bindings,
    draft.value.mode,
  )
}

function removeBinding(datapointId: string) {
  draft.value.bindings = draft.value.bindings.filter((item) => item.datapointId !== datapointId)
  draft.value.conditions = normalizeAlarmConditionsForBindings(
    draft.value.conditions,
    draft.value.bindings,
    draft.value.mode,
  )
}

function addCondition(kind: AlarmConditionKind) {
  draft.value.conditions.push(createAlarmCondition(kind))
}

function changeConditionKind(index: number, kind: AlarmConditionKind) {
  draft.value.conditions[index] = createAlarmCondition(kind)
}

function removeCondition(index: number) {
  if (draft.value.conditions.length > 1) draft.value.conditions.splice(index, 1)
}

function changeNotificationMode(mode: AlarmPolicySave['notification']['mode']) {
  if (mode !== 'custom') return
  draft.value.notification.notifyOnRaise ??= true
  draft.value.notification.notifyOnClear ??= true
  draft.value.notification.repeatIntervalSeconds ??= null
  if (!draft.value.notification.channelIds.length) {
    const runtimeChannel = enabledChannels.value.find((channel) => channel.id === 'runtime_inapp')
    draft.value.notification.channelIds = runtimeChannel ? [runtimeChannel.id] : []
  }
}

function setParam(condition: AlarmCondition, key: string, value: unknown) {
  condition.params = { ...condition.params, [key]: value }
}

function numberParam(condition: AlarmCondition, key: string): number | undefined {
  const value = condition.params[key]
  return typeof value === 'number' ? value : undefined
}

function operatorOptions(kind: AlarmConditionKind) {
  const options: Record<AlarmConditionKind, Array<{ label: string; value: string }>> = {
    threshold: [
      { label: '高于', value: 'gt' },
      { label: '不低于', value: 'gte' },
      { label: '低于', value: 'lt' },
      { label: '不高于', value: 'lte' },
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
    ],
    text_match: [
      { label: '包含', value: 'contains' },
      { label: '等于', value: 'eq' },
      { label: '不等于', value: 'ne' },
      { label: '正则匹配', value: 'regex' },
    ],
    rate_of_change: [{ label: '大于', value: 'gt' }],
    deviation: [{ label: '大于', value: 'gt' }],
    offline: [{ label: '离线', value: 'is_offline' }],
    expression: [{ label: '结果为真', value: 'is_true' }],
  }
  return options[kind]
}

function validateDraft(): boolean {
  if (!draft.value.name.trim()) return (ElMessage.warning('请填写报警名称'), false)
  if (!draft.value.bindings.length) return (ElMessage.warning('请至少选择一个数据点'), false)
  if (!compatibleAlarmBindings(draft.value.bindings) && draft.value.mode === 'per_target')
    return (ElMessage.warning('所选数据点类型不兼容'), false)
  if (draft.value.mode === 'derived' && !draft.value.derivedExpression.trim())
    return (ElMessage.warning('请填写组合表达式'), false)
  if (!draft.value.conditions.length) return (ElMessage.warning('请至少添加一个报警条件'), false)
  if (draft.value.notification.mode === 'custom' && !draft.value.notification.channelIds.length)
    return (ElMessage.warning('单独设置通知时请至少选择一个渠道'), false)
  return true
}

function submit() {
  if (!validateDraft()) return
  emit('save', AlarmPolicySaveSchema.parse(draft.value))
}

async function runTrial() {
  if (!props.policy?.id) return
  trialLoading.value = true
  try {
    const raw = trialValue.value.trim()
    const value = raw === '' ? null : Number.isNaN(Number(raw)) ? raw : Number(raw)
    debugOutput.value = JSON.stringify(
      await testAlarmPolicy(props.projectId, props.policy.id, value),
      null,
      2,
    )
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '报警试算失败'))
  } finally {
    trialLoading.value = false
  }
}

async function loadContract() {
  if (!props.policy?.id) return
  contractLoading.value = true
  try {
    debugOutput.value = JSON.stringify(
      await getAlarmPolicyContract(props.projectId, props.policy.id),
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
.alarm-editor__close {
  width: 30px;
  height: 30px;
  display: inline-grid;
  place-items: center;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
}
.alarm-editor__close:hover {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}
.alarm-editor__close svg {
  width: 17px;
  height: 17px;
}
.alarm-editor__section {
  display: grid;
  gap: 12px;
  padding: 0;
}
.alarm-editor__section + .alarm-editor__section {
  padding-top: 16px;
  border-top: 1px solid var(--dc-border);
}
.alarm-editor__section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 0;
}
.alarm-editor__section-head h3 {
  margin: 0;
  font-size: 14px;
  letter-spacing: 0;
}
.alarm-editor__section-head p {
  margin: 4px 0 0;
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 18px;
}
.alarm-editor__grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 190px 110px;
  gap: 12px;
  align-items: end;
}
.alarm-editor__field {
  display: grid;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.alarm-editor__switch-field {
  justify-items: start;
}
.alarm-editor__points,
.alarm-editor__conditions {
  display: grid;
  gap: 8px;
}
.alarm-editor__point-summary {
  min-height: 38px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 10px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 20%, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.alarm-editor__point-summary strong {
  color: var(--dc-primary);
  letter-spacing: 0;
}
.alarm-editor__point-summary button {
  flex: 0 0 auto;
  padding: 4px 0;
  border: 0;
  background: transparent;
  color: var(--dc-primary);
  font-family: inherit;
  font-size: 12px;
  font-weight: 700;
}
.alarm-editor__point-summary button:hover {
  text-decoration: underline;
}
.alarm-editor__point {
  min-height: 48px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 116px auto 30px;
  align-items: center;
  gap: 8px;
  padding: 7px 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}
.alarm-editor__point-main {
  min-width: 0;
}
.alarm-editor__point strong,
.alarm-editor__point code {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.alarm-editor__point strong {
  font-size: 13px;
}
.alarm-editor__point code {
  margin-top: 2px;
  color: var(--dc-text-muted);
  font-size: 11px;
}
.alarm-editor__key {
  width: 116px;
  height: 30px;
  padding: 0 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}
.alarm-editor__type {
  color: var(--dc-text-muted);
  font-size: 12px;
}
.alarm-editor__empty {
  padding: 18px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 13px;
  text-align: center;
}
.alarm-editor__condition {
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}
.alarm-editor__condition-main {
  display: grid;
  grid-template-columns: 112px 112px minmax(110px, 1fr) 96px 30px;
  gap: 8px;
  align-items: center;
}
.alarm-editor__condition-main:has(.el-input-number + .el-input-number) {
  grid-template-columns: 100px 100px minmax(90px, 1fr) minmax(90px, 1fr) 92px 30px;
}
.alarm-editor__condition-advanced {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--dc-border);
}
.alarm-editor__condition-advanced label {
  display: grid;
  gap: 4px;
  color: var(--dc-text-muted);
  font-size: 11px;
}
.alarm-editor__secondary,
.alarm-editor__primary,
.alarm-editor__cancel,
.alarm-editor__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border-radius: var(--dc-radius-sm);
  font: inherit;
  cursor: pointer;
}
.alarm-editor__secondary {
  min-height: 32px;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.alarm-editor__secondary svg {
  width: 15px;
  height: 15px;
}
.alarm-editor__secondary:hover,
.alarm-editor__icon:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.alarm-editor__icon {
  width: 30px;
  height: 30px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--dc-text-secondary);
}
.alarm-editor__icon.is-danger {
  color: var(--dc-text-muted);
}
.alarm-editor__icon.is-danger:not(:disabled):hover {
  border-color: color-mix(in oklch, var(--el-color-danger) 28%, var(--dc-border));
  background: color-mix(in srgb, var(--el-color-danger) 8%, white);
  color: var(--el-color-danger);
}
.alarm-editor__icon:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.alarm-editor__collapse {
  width: 100%;
  height: 32px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border: 0;
  background: transparent;
  color: var(--dc-text);
  font-weight: 700;
  cursor: pointer;
}
.alarm-editor__collapse svg {
  transition: transform 0.18s ease;
}
.alarm-editor__collapse svg.is-open {
  transform: rotate(180deg);
}
.alarm-editor__advanced {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-top: 14px;
}
.alarm-editor__checks {
  display: flex;
  align-items: end;
  gap: 16px;
}
.alarm-editor__debug {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  gap: 8px;
  margin-top: 12px;
}
.alarm-editor__debug pre {
  grid-column: 1 / -1;
  max-height: 220px;
  overflow: auto;
  margin: 0;
  padding: 10px;
  border-radius: var(--dc-radius-sm);
  background: #111827;
  color: #e5e7eb;
  font-size: 11px;
}
.alarm-editor__footer {
  margin-top: auto;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 16px;
  border-top: 1px solid var(--dc-border);
}
.alarm-editor__primary,
.alarm-editor__cancel {
  height: 34px;
  padding: 0 16px;
}
.alarm-editor__primary {
  border: 1px solid var(--dc-primary);
  background: var(--dc-primary);
  color: white;
}
.alarm-editor__primary:not(:disabled):hover {
  background: color-mix(in srgb, var(--dc-primary) 88%, black);
}
.alarm-editor__primary:disabled {
  opacity: 0.6;
}
.alarm-editor__primary svg {
  width: 16px;
  height: 16px;
}
.alarm-editor__cancel {
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}
.alarm-editor__cancel:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  color: var(--dc-text);
}
.alarm-editor__close:focus-visible,
.alarm-editor__secondary:focus-visible,
.alarm-editor__icon:focus-visible,
.alarm-editor__point-summary button:focus-visible,
.alarm-editor__collapse:focus-visible,
.alarm-editor__primary:focus-visible,
.alarm-editor__cancel:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px rgba(29, 78, 216, 0.12);
}
@media (max-width: 720px) {
  .alarm-editor__grid,
  .alarm-editor__advanced {
    grid-template-columns: 1fr;
  }
  .alarm-editor__point,
  .alarm-editor__condition-main,
  .alarm-editor__condition-main:has(.el-input-number + .el-input-number),
  .alarm-editor__condition-advanced {
    grid-template-columns: 1fr;
  }
}
</style>
