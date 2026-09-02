<template>
  <DcDrawer v-model="visible" :title="t('quickAlarm.title')" :width="820" :max="920">
    <template #actions>
      <button
        type="button"
        class="quick-alarm__icon"
        :title="t('quickAlarm.close')"
        @click="visible = false"
      >
        <IconTablerX />
      </button>
    </template>

    <form class="quick-alarm" @submit.prevent="submit">
      <section class="quick-alarm__context">
        <div class="quick-alarm__point">
          <span>{{ t('quickAlarm.datapoint') }}</span>
          <strong>{{ pointTitle }}</strong>
          <small :title="pointPath">{{ pointPath }}</small>
        </div>
        <label class="quick-alarm__field">
          <span>{{ t('quickAlarm.directory') }}</span>
          <AlarmGroupSelect v-model="groupId" :project-id="projectId" />
        </label>
        <label class="quick-alarm__field is-switch">
          <span>{{ t('quickAlarm.enable') }}</span>
          <el-switch v-model="isEnabled" />
        </label>
      </section>

      <el-alert
        v-if="!isCompatible"
        :title="t('quickAlarm.incompatible')"
        type="warning"
        :closable="false"
        show-icon
      />

      <div class="quick-alarm__mode" role="tablist" :aria-label="t('quickAlarm.configMode')">
        <button
          type="button"
          :class="{ 'is-active': configurationMode === 'basic' }"
          role="tab"
          :aria-selected="configurationMode === 'basic'"
          @click="setConfigurationMode('basic')"
        >
          {{ t('quickAlarm.basic') }}
        </button>
        <button
          type="button"
          :class="{ 'is-active': configurationMode === 'advanced' }"
          role="tab"
          :aria-selected="configurationMode === 'advanced'"
          @click="setConfigurationMode('advanced')"
        >
          {{ t('quickAlarm.advanced') }}
        </button>
      </div>

      <section
        v-if="points.length === 1 && (category !== 'number' || activeNameItems.length)"
        class="quick-alarm__names"
      >
        <div>
          <strong>{{ t('quickAlarm.alarmName') }}</strong>
          <span>{{ t('quickAlarm.autoName') }}</span>
        </div>
        <div class="quick-alarm__name-fields">
          <label v-for="item in activeNameItems" :key="item.slot" class="quick-alarm__field">
            <span>{{ item.label }}</span>
            <el-input
              v-model="alarmNames[item.slot]"
              :placeholder="suggestedAlarmName(item.slot)"
              maxlength="100"
              clearable
            />
          </label>
        </div>
      </section>

      <section class="quick-alarm__config">
        <header v-if="category !== 'number'">
          <h3>{{ t('quickAlarm.config') }}</h3>
        </header>

        <div v-if="category === 'number'" class="quick-alarm__numeric">
          <div class="quick-alarm__numeric-group">
            <strong>{{ t('quickAlarm.limitAlarms') }}</strong>
          </div>
          <div
            class="quick-alarm__numeric-head"
            :class="{ 'is-advanced': configurationMode === 'advanced' }"
          >
            <span>{{ t('quickAlarm.enable') }}</span>
            <span>{{ t('quickAlarm.alarmItem') }}</span>
            <span>{{ t('quickAlarm.limitParams') }}</span>
            <span>{{ t('quickAlarm.alarmText') }}</span>
            <span>{{ t('quickAlarm.level') }}</span>
          </div>
          <template v-for="item in numericItems" :key="item.key">
            <div v-if="item.key === 'rate'" class="quick-alarm__numeric-group is-divider">
              <strong>{{ t('quickAlarm.dynamicAlarms') }}</strong>
            </div>
            <div
              class="quick-alarm__numeric-row"
              :class="{
                'is-enabled': numericEnabled[item.key],
                'is-advanced': configurationMode === 'advanced',
              }"
            >
              <el-checkbox
                v-model="numericEnabled[item.key]"
                :aria-label="`${t('quickAlarm.enable')} ${item.label}`"
              />
              <strong>{{ item.label }}</strong>

              <div v-if="item.kind === 'threshold'" class="quick-alarm__numeric-params">
                <el-input-number
                  v-model="numericThresholds[item.key as NumericThresholdKey]"
                  :disabled="!numericEnabled[item.key]"
                  controls-position="right"
                  :placeholder="item.label"
                />
              </div>
              <div
                v-else-if="item.kind === 'rate_of_change'"
                class="quick-alarm__numeric-params is-rate"
              >
                <el-select v-model="rateDirection" :disabled="!numericEnabled[item.key]">
                  <el-option :label="t('quickAlarm.bidirectional')" value="absolute" />
                  <el-option :label="t('quickAlarm.rise')" value="rise" />
                  <el-option :label="t('quickAlarm.fall')" value="fall" />
                </el-select>
                <el-input-number
                  v-model="rateLimit"
                  :disabled="!numericEnabled[item.key]"
                  :min="0"
                  controls-position="right"
                  :placeholder="t('quickAlarm.rate')"
                />
                <el-select v-model="rateWindowSeconds" :disabled="!numericEnabled[item.key]">
                  <el-option :label="t('quickAlarm.tenSeconds')" :value="10" />
                  <el-option :label="t('quickAlarm.oneMinute')" :value="60" />
                  <el-option :label="t('quickAlarm.fiveMinutes')" :value="300" />
                </el-select>
              </div>
              <div v-else class="quick-alarm__numeric-params is-deviation">
                <el-input-number
                  v-model="deviationBaseline"
                  :disabled="!numericEnabled[item.key]"
                  controls-position="right"
                  :placeholder="t('quickAlarm.baseline')"
                />
                <el-input-number
                  v-model="deviationLimit"
                  :disabled="!numericEnabled[item.key]"
                  :min="0"
                  controls-position="right"
                  :placeholder="t('quickAlarm.allowedDeviation')"
                />
              </div>

              <el-input
                v-model="numericMessages[item.key]"
                :disabled="!numericEnabled[item.key]"
                :placeholder="t('quickAlarm.alarmText')"
              />

              <el-select
                v-model="numericSeverities[item.key]"
                :disabled="!numericEnabled[item.key]"
              >
                <el-option
                  v-for="option in severityOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
            </div>
          </template>
          <div class="quick-alarm__numeric-common">
            <label :class="{ 'is-advanced': configurationMode === 'advanced' }">
              <el-checkbox v-model="numericEnabled.offline">{{
                t('quickAlarm.offlineAlarm')
              }}</el-checkbox>
              <el-select v-model="numericSeverities.offline" :disabled="!numericEnabled.offline">
                <el-option
                  v-for="option in severityOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
            </label>
            <label>
              <span>{{ t('quickAlarm.triggerDelaySeconds') }}</span>
              <el-input-number v-model="triggerDelaySeconds" :min="0" controls-position="right" />
            </label>
          </div>
        </div>

        <div
          v-else
          class="quick-alarm__types"
          role="radiogroup"
          :aria-label="t('quickAlarm.alarmType')"
        >
          <button
            v-for="option in typeOptions"
            :key="option.value"
            type="button"
            :class="{ 'is-active': alarmType === option.value }"
            role="radio"
            :aria-checked="alarmType === option.value"
            @click="alarmType = option.value"
          >
            {{ option.label }}
          </button>
        </div>

        <div v-if="category !== 'number'" class="quick-alarm__params">
          <template v-if="alarmType === 'state'">
            <label class="quick-alarm__field is-primary">
              <span>{{ t('quickAlarm.valueEquals') }}</span>
              <el-select v-if="category === 'boolean'" v-model="booleanExpected">
                <el-option label="true" :value="true" />
                <el-option label="false" :value="false" />
              </el-select>
              <el-input
                v-else
                v-model="textExpected"
                :placeholder="t('quickAlarm.expectedValue')"
              />
            </label>
          </template>

          <template v-else-if="alarmType === 'text'">
            <label class="quick-alarm__field is-primary">
              <span>{{ t('quickAlarm.containsText') }}</span>
              <el-input v-model="textExpected" :placeholder="t('quickAlarm.matchText')" />
            </label>
          </template>

          <template v-else-if="alarmType === 'stale'">
            <label class="quick-alarm__field is-primary">
              <span>{{ t('quickAlarm.staleMinutes') }}</span>
              <el-input-number v-model="staleMinutes" :min="1" controls-position="right" />
            </label>
          </template>

          <div v-else class="quick-alarm__no-param">
            {{ noParamHint }}
          </div>

          <label class="quick-alarm__field">
            <span>{{ t('quickAlarm.severity') }}</span>
            <el-select v-model="severity">
              <el-option
                v-for="option in severityOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </el-select>
          </label>
          <label class="quick-alarm__field">
            <span>{{ t('quickAlarm.triggerDelaySeconds') }}</span>
            <el-input-number v-model="triggerDelaySeconds" :min="0" controls-position="right" />
          </label>
        </div>
      </section>

      <section v-if="configurationMode === 'advanced'" class="quick-alarm__advanced">
        <template v-if="category === 'number'">
          <header class="quick-alarm__advanced-head">
            <h3>{{ t('quickAlarm.additionalLimits') }}</h3>
            <div>
              <button type="button" @click="addAdvancedLimit('high')">
                {{ t('quickAlarm.addHigh') }}
              </button>
              <button type="button" @click="addAdvancedLimit('low')">
                {{ t('quickAlarm.addLow') }}
              </button>
            </div>
          </header>
          <div v-if="advancedLimits.length" class="quick-alarm__limit-list">
            <div class="quick-alarm__limit-head" aria-hidden="true">
              <span>{{ t('quickAlarm.name') }}</span
              ><span>{{ t('quickAlarm.direction') }}</span
              ><span>{{ t('quickAlarm.limit') }}</span
              ><span>{{ t('quickAlarm.alarmText') }}</span
              ><span>{{ t('quickAlarm.level') }}</span
              ><span></span>
            </div>
            <div v-for="item in advancedLimits" :key="item.id" class="quick-alarm__limit-row">
              <el-input v-model="item.name" :placeholder="t('quickAlarm.customLimit')" />
              <el-select v-model="item.direction">
                <el-option :label="t('quickAlarm.above')" value="high" />
                <el-option :label="t('quickAlarm.below')" value="low" />
              </el-select>
              <el-input-number
                v-model="item.threshold"
                controls-position="right"
                :placeholder="t('quickAlarm.limit')"
              />
              <el-input v-model="item.message" :placeholder="t('quickAlarm.alarmText')" />
              <el-select v-model="item.severity">
                <el-option
                  v-for="option in severityOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
              <button
                type="button"
                class="quick-alarm__remove"
                :title="t('quickAlarm.removeLimit')"
                @click="removeAdvancedLimit(item.id)"
              >
                ×
              </button>
            </div>
          </div>
        </template>

        <header>
          <h3>{{ t('quickAlarm.triggerAndRecovery') }}</h3>
        </header>
        <div class="quick-alarm__advanced-grid">
          <label class="quick-alarm__field">
            <span>{{ t('quickAlarm.clearDelaySeconds') }}</span>
            <el-input-number v-model="clearDelaySeconds" :min="0" controls-position="right" />
          </label>
          <label v-if="category === 'number'" class="quick-alarm__field">
            <span>{{ t('quickAlarm.deadband') }}</span>
            <el-input-number v-model="deadband" :min="0" controls-position="right" />
          </label>
          <label class="quick-alarm__field">
            <span>{{ t('quickAlarm.notificationMode') }}</span>
            <el-select v-model="notificationMode">
              <el-option :label="t('quickAlarm.inheritNotifications')" value="inherit" />
              <el-option :label="t('quickAlarm.noNotifications')" value="off" />
              <el-option :label="t('quickAlarm.custom')" value="custom" />
            </el-select>
          </label>
        </div>
        <div v-if="notificationMode === 'custom'" class="quick-alarm__notification">
          <el-checkbox v-model="notifyOnRaise">{{ t('quickAlarm.notifyOnRaise') }}</el-checkbox>
          <el-checkbox v-model="notifyOnClear">{{ t('quickAlarm.notifyOnClear') }}</el-checkbox>
          <label class="quick-alarm__field">
            <span>{{ t('quickAlarm.repeatSeconds') }}</span>
            <el-input-number
              v-model="repeatIntervalSeconds"
              :min="1"
              controls-position="right"
              :placeholder="t('quickAlarm.noRepeat')"
            />
          </label>
          <label class="quick-alarm__field is-wide">
            <span>{{ t('quickAlarm.messageTemplate') }}</span>
            <el-input v-model="messageTemplate" :placeholder="t('quickAlarm.defaultMessage')" />
          </label>
        </div>
      </section>

      <footer class="quick-alarm__footer">
        <div>
          <button type="button" class="quick-alarm__cancel" @click="visible = false">
            {{ t('quickAlarm.cancel') }}
          </button>
          <button
            type="submit"
            class="quick-alarm__primary"
            :disabled="
              saving ||
              loadingPreset ||
              !isCompatible ||
              (category === 'number' && !numericDraftCount)
            "
          >
            <IconTablerDeviceFloppy />{{ submitText }}
          </button>
        </div>
      </footer>
    </form>
  </DcDrawer>
</template>

<script setup lang="ts">
import { secureRandomUUID } from '@/utils/secure-random-uuid'
import { computed, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import IconTablerDeviceFloppy from '~icons/tabler/device-floppy'
import IconTablerX from '~icons/tabler/x'
import DcDrawer from '@/components/shared/DcDrawer.vue'
import AlarmGroupSelect from './AlarmGroupSelect.vue'
import { listAlarmItems, validatePresetAlarmConfiguration } from '@/api/alarm.api'
import { AlarmItemSaveSchema } from '@/api/schemas/alarm.schema'
import type {
  AlarmCondition,
  AlarmItem,
  AlarmItemSave,
  AlarmPresetSlot,
  AlarmSeverity,
} from '@/api/schemas/alarm.schema'
import {
  alarmPointCategory,
  compatibleAlarmPoints,
  createAlarmCondition,
  createAlarmItemDraft,
  type AlarmPointCategory,
  type AlarmPointSelection,
} from '@/models/alarm-item'
import { useAlarmLevelDefinitions } from '@/composables/useAlarmLevelDefinitions'
import { getApiErrorMessage } from '@/utils/request'
import { t } from '@/i18n/runtime'

type QuickAlarmType = 'state' | 'transition' | 'text' | 'quality' | 'stale' | 'offline'

type NumericThresholdKey = 'hh' | 'h' | 'l' | 'll'
type NumericAlarmKey = NumericThresholdKey | 'rate' | 'deviation' | 'offline'
interface AdvancedLimit {
  id: string
  direction: 'high' | 'low'
  name: string
  threshold?: number
  message: string
  severity: AlarmSeverity
}

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    projectId: string
    points?: AlarmPointSelection[]
    saving?: boolean
  }>(),
  { points: () => [], saving: false },
)
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [payload: AlarmItemSave | AlarmItemSave[], datapointIds: string[]]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const groupId = ref<string | null>(null)
const isEnabled = ref(true)
const alarmType = ref<QuickAlarmType>('state')
const severity = ref<AlarmSeverity>('warning')
const alarmNames = ref<Record<AlarmPresetSlot, string>>({
  limit: '',
  rate: '',
  deviation: '',
  state: '',
  transition: '',
  text: '',
  quality: '',
  stale: '',
  offline: '',
})
const numericEnabled = ref<Record<NumericAlarmKey, boolean>>({
  hh: false,
  h: false,
  l: false,
  ll: false,
  rate: false,
  deviation: false,
  offline: false,
})
const numericThresholds = ref<Record<NumericThresholdKey, number | undefined>>({
  hh: 100,
  h: 90,
  l: 10,
  ll: 0,
})
const numericSeverities = ref<Record<NumericAlarmKey, AlarmSeverity>>({
  hh: 'critical',
  h: 'warning',
  l: 'warning',
  ll: 'critical',
  rate: 'warning',
  deviation: 'warning',
  offline: 'warning',
})
const numericMessages = ref<Record<NumericAlarmKey, string>>({
  hh: t('quickAlarm.presets.limit'),
  h: t('quickAlarm.presets.limit'),
  l: t('quickAlarm.presets.limit'),
  ll: t('quickAlarm.presets.limit'),
  rate: t('quickAlarm.presets.rate'),
  deviation: t('quickAlarm.presets.deviation'),
  offline: t('quickAlarm.presets.offline'),
})
const rateLimit = ref<number>()
const rateDirection = ref<'absolute' | 'rise' | 'fall'>('absolute')
const rateWindowSeconds = ref(60)
const deviationBaseline = ref<number>()
const deviationLimit = ref<number>()
const booleanExpected = ref(true)
const textExpected = ref('')
const staleMinutes = ref(5)
const triggerDelaySeconds = ref(0)
const loadingPreset = ref(false)
const configurationMode = ref<'basic' | 'advanced'>('basic')
const clearDelaySeconds = ref(0)
const deadband = ref(0)
const notificationMode = ref<'inherit' | 'off' | 'custom'>('inherit')
const notifyOnRaise = ref(true)
const notifyOnClear = ref(true)
const repeatIntervalSeconds = ref<number>()
const messageTemplate = ref('')
const advancedLimits = ref<AdvancedLimit[]>([])
const { definitions: severityDefinitions, loadDefinitions } = useAlarmLevelDefinitions(
  () => props.projectId,
)

const category = computed<AlarmPointCategory>(() => alarmPointCategory(props.points[0]?.dataType))
const isCompatible = computed(() => compatibleAlarmPoints(props.points))
const pointTitle = computed(() =>
  props.points.length === 1
    ? props.points[0]?.name || t('quickAlarm.datapointFallback')
    : t('quickAlarm.selectedPoints', {
        count: props.points.length,
        category: categoryLabel.value,
      }),
)
const pointPath = computed(() =>
  props.points.length === 1
    ? props.points[0]?.path || ''
    : props.points
        .slice(0, 3)
        .map((point) => point.name || point.path)
        .join(', ') + (props.points.length > 3 ? '…' : ''),
)
const categoryLabel = computed(() => t(`quickAlarm.categories.${category.value}`))
const numericItems = computed<
  Array<{
    key: NumericAlarmKey
    label: string
    kind: 'threshold' | 'rate_of_change' | 'deviation' | 'offline'
  }>
>(() => [
  { key: 'hh', label: t('quickAlarm.levels.hh'), kind: 'threshold' },
  { key: 'h', label: t('quickAlarm.levels.h'), kind: 'threshold' },
  { key: 'l', label: t('quickAlarm.levels.l'), kind: 'threshold' },
  { key: 'll', label: t('quickAlarm.levels.ll'), kind: 'threshold' },
  { key: 'rate', label: t('quickAlarm.levels.rate'), kind: 'rate_of_change' },
  { key: 'deviation', label: t('quickAlarm.levels.deviation'), kind: 'deviation' },
])
const typeOptions = computed<Array<{ label: string; value: QuickAlarmType }>>(() => {
  if (category.value === 'boolean')
    return [
      { label: t('quickAlarm.types.state'), value: 'state' },
      { label: t('quickAlarm.types.transition'), value: 'transition' },
      { label: t('quickAlarm.types.offline'), value: 'offline' },
    ]
  if (category.value === 'structured')
    return [
      { label: t('quickAlarm.types.quality'), value: 'quality' },
      { label: t('quickAlarm.types.stale'), value: 'stale' },
      { label: t('quickAlarm.types.offline'), value: 'offline' },
    ]
  return [
    { label: t('quickAlarm.types.state'), value: 'state' },
    { label: t('quickAlarm.types.text'), value: 'text' },
    { label: t('quickAlarm.types.transition'), value: 'transition' },
    { label: t('quickAlarm.types.offline'), value: 'offline' },
  ]
})
const noParamHint = computed(() => {
  if (alarmType.value === 'offline') return t('quickAlarm.hints.offline')
  if (alarmType.value === 'quality') return t('quickAlarm.hints.quality')
  return t('quickAlarm.hints.transition')
})
const submitText = computed(() => {
  return props.saving ? t('quickAlarm.saving') : t('quickAlarm.save')
})
const numericDraftCount = computed(() => {
  const hasThreshold = (['hh', 'h', 'l', 'll'] as NumericThresholdKey[]).some(
    (key) => numericEnabled.value[key],
  )
  return (
    Number(hasThreshold) +
    Number(numericEnabled.value.rate) +
    Number(numericEnabled.value.deviation) +
    Number(numericEnabled.value.offline) +
    Number(advancedLimits.value.length > 0)
  )
})
const severityOptions = computed(() =>
  severityDefinitions.value.map((item) => ({
    value: item.key,
    label: ['info', 'warning', 'major', 'critical'].includes(item.key)
      ? t(`alarm.severities.${item.key}`)
      : item.displayName,
  })),
)
const presetNameLabel = (slot: AlarmPresetSlot) => t(`quickAlarm.presets.${slot}`)
const presetNameSuffix = (slot: AlarmPresetSlot) => t(`quickAlarm.suffixes.${slot}`)
const activeNameItems = computed(() => {
  const slots: AlarmPresetSlot[] = []
  if (category.value === 'number') {
    if (
      (['hh', 'h', 'l', 'll'] as NumericThresholdKey[]).some((key) => numericEnabled.value[key]) ||
      advancedLimits.value.length
    )
      slots.push('limit')
    if (numericEnabled.value.rate) slots.push('rate')
    if (numericEnabled.value.deviation) slots.push('deviation')
    if (numericEnabled.value.offline) slots.push('offline')
  } else {
    slots.push(nonNumericPresetSlot())
  }
  return slots.map((slot) => ({ slot, label: presetNameLabel(slot) }))
})

function suggestedAlarmName(slot: AlarmPresetSlot) {
  return `${props.points[0]?.name || t('quickAlarm.datapointFallback')}_${presetNameSuffix(slot)}`
}

watch(
  () => props.modelValue,
  (opened) => {
    if (!opened) return
    groupId.value = null
    isEnabled.value = true
    alarmType.value = typeOptions.value[0]?.value || 'state'
    severity.value = 'warning'
    Object.keys(alarmNames.value).forEach((key) => {
      alarmNames.value[key as AlarmPresetSlot] = ''
    })
    numericEnabled.value = {
      hh: false,
      h: false,
      l: false,
      ll: false,
      rate: false,
      deviation: false,
      offline: false,
    }
    numericThresholds.value = { hh: 100, h: 90, l: 10, ll: 0 }
    numericSeverities.value = {
      hh: 'critical',
      h: 'warning',
      l: 'warning',
      ll: 'critical',
      rate: 'warning',
      deviation: 'warning',
      offline: 'warning',
    }
    numericMessages.value = {
      hh: t('quickAlarm.presets.limit'),
      h: t('quickAlarm.presets.limit'),
      l: t('quickAlarm.presets.limit'),
      ll: t('quickAlarm.presets.limit'),
      rate: t('quickAlarm.presets.rate'),
      deviation: t('quickAlarm.presets.deviation'),
      offline: t('quickAlarm.presets.offline'),
    }
    rateLimit.value = undefined
    rateDirection.value = 'absolute'
    rateWindowSeconds.value = 60
    deviationBaseline.value = undefined
    deviationLimit.value = undefined
    booleanExpected.value = true
    textExpected.value = ''
    staleMinutes.value = 5
    triggerDelaySeconds.value = 0
    configurationMode.value = 'basic'
    resetAdvancedOptions()
    void loadDefinitions()
    if (props.points.length === 1) void loadPresetConfiguration()
  },
)

async function loadPresetConfiguration() {
  const datapointId = props.points[0]?.datapointId
  if (!datapointId) return
  loadingPreset.value = true
  try {
    const result = await listAlarmItems(props.projectId, { datapointId, page: 1, pageSize: 100 })
    const items = result.list.filter((item) => item.presetSlot)
    if (!items.length) return
    groupId.value = items[0]?.groupId || null
    isEnabled.value = items.some((item) => item.isEnabled)
    if (category.value === 'number') restoreNumericItems(items)
    else restoreNonNumericItem(items[0]!)
    restoreAdvancedOptions(items)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('quickAlarm.loadExistingFailed')))
  } finally {
    loadingPreset.value = false
  }
}

function setConfigurationMode(mode: 'basic' | 'advanced') {
  if (mode === configurationMode.value) return
  if (mode === 'basic') ElMessage.info(t('quickAlarm.basicModeNotice'))
  configurationMode.value = mode
}

function resetAdvancedOptions() {
  clearDelaySeconds.value = 0
  deadband.value = 0
  notificationMode.value = 'inherit'
  notifyOnRaise.value = true
  notifyOnClear.value = true
  repeatIntervalSeconds.value = undefined
  messageTemplate.value = ''
  advancedLimits.value = []
}

function addAdvancedLimit(direction: 'high' | 'low') {
  const index = advancedLimits.value.filter((item) => item.direction === direction).length + 1
  advancedLimits.value.push({
    id: secureRandomUUID(),
    direction,
    name: `${direction === 'high' ? t('quickAlarm.additionalHigh') : t('quickAlarm.additionalLow')} ${index}`,
    threshold: undefined,
    message: '',
    severity: 'warning',
  })
}

function removeAdvancedLimit(id: string) {
  advancedLimits.value = advancedLimits.value.filter((item) => item.id !== id)
}

function restoreAdvancedOptions(items: AlarmItem[]) {
  const first = items[0]
  const firstCondition = first?.conditions[0]
  if (!first || !firstCondition) return
  clearDelaySeconds.value = firstCondition.clearDelayMs / 1000
  deadband.value = firstCondition.deadband
  notificationMode.value = first.notification.mode
  notifyOnRaise.value = first.notification.notifyOnRaise ?? true
  notifyOnClear.value = first.notification.notifyOnClear ?? true
  repeatIntervalSeconds.value = first.notification.repeatIntervalSeconds ?? undefined
  messageTemplate.value = first.notification.messageTemplate
  if (hasAdvancedConfiguration(items)) configurationMode.value = 'advanced'
}

function hasAdvancedConfiguration(items: AlarmItem[]) {
  return items.some((item) => {
    const notification = item.notification
    if (
      notification.mode !== 'inherit' ||
      notification.messageTemplate ||
      notification.repeatIntervalSeconds ||
      notification.notifyOnRaise != null ||
      notification.notifyOnClear != null
    )
      return true
    return item.conditions.some((condition) => {
      const presetLevel = String(condition.params.presetLevel || '')
      return (
        condition.params.configurationMode === 'advanced' ||
        presetLevel === 'custom' ||
        condition.clearDelayMs > 0 ||
        condition.deadband > 0
      )
    })
  })
}

function restoreNumericItems(items: AlarmItem[]) {
  const limit = items.find((item) => item.presetSlot === 'limit')
  if (limit) {
    alarmNames.value.limit = limit.displayName
    const custom = limit.conditions.filter((condition) => condition.params.presetLevel === 'custom')
    advancedLimits.value = custom.map((condition) => ({
      id: condition.id || secureRandomUUID(),
      direction: ['gt', 'gte'].includes(condition.operator) ? 'high' : 'low',
      name: String(condition.params.levelName || t('quickAlarm.additionalLimit')),
      threshold: Number(condition.params.threshold),
      message: condition.label,
      severity: condition.severity,
    }))
    if (custom.length) configurationMode.value = 'advanced'
    const standard = limit.conditions.filter(
      (condition) => condition.params.presetLevel !== 'custom',
    )
    const marked = standard.filter((condition) =>
      ['hh', 'h', 'l', 'll'].includes(String(condition.params.presetLevel || '')),
    )
    if (marked.length) {
      marked.forEach((condition) =>
        restoreThreshold(condition.params.presetLevel as NumericThresholdKey, condition),
      )
    } else {
      const high = standard
        .filter((condition) => ['gt', 'gte'].includes(condition.operator))
        .sort((a, b) => Number(b.params.threshold) - Number(a.params.threshold))
      const low = standard
        .filter((condition) => ['lt', 'lte'].includes(condition.operator))
        .sort((a, b) => Number(a.params.threshold) - Number(b.params.threshold))
      restoreThreshold('hh', high[0])
      restoreThreshold('h', high[1] || (high.length === 1 ? high[0] : undefined))
      if (high.length === 1) numericEnabled.value.hh = false
      restoreThreshold('ll', low[0])
      restoreThreshold('l', low[1] || (low.length === 1 ? low[0] : undefined))
      if (low.length === 1) numericEnabled.value.ll = false
    }
  }
  const rate = items.find((item) => item.presetSlot === 'rate')
  if (rate?.conditions[0]) {
    alarmNames.value.rate = rate.displayName
    numericEnabled.value.rate = true
    rateLimit.value = Number(rate.conditions[0].params.limit)
    rateDirection.value = String(rate.conditions[0].params.direction || 'absolute') as
      | 'absolute'
      | 'rise'
      | 'fall'
    rateWindowSeconds.value = Number(rate.conditions[0].params.windowMs || 60000) / 1000
    numericSeverities.value.rate = rate.conditions[0].severity
    numericMessages.value.rate = rate.conditions[0].label
  }
  const deviation = items.find((item) => item.presetSlot === 'deviation')
  if (deviation?.conditions[0]) {
    alarmNames.value.deviation = deviation.displayName
    numericEnabled.value.deviation = true
    deviationBaseline.value = Number(deviation.conditions[0].params.baseline)
    deviationLimit.value = Number(deviation.conditions[0].params.limit)
    numericSeverities.value.deviation = deviation.conditions[0].severity
    numericMessages.value.deviation = deviation.conditions[0].label
  }
  const offline = items.find((item) => item.presetSlot === 'offline')
  if (offline?.conditions[0]) {
    alarmNames.value.offline = offline.displayName
    numericEnabled.value.offline = true
    numericSeverities.value.offline = offline.conditions[0].severity
    numericMessages.value.offline = offline.conditions[0].label
  }
  const firstCondition = items.flatMap((item) => item.conditions)[0]
  triggerDelaySeconds.value = (firstCondition?.triggerDelayMs || 0) / 1000
}

function restoreThreshold(key: NumericThresholdKey, condition?: AlarmCondition) {
  if (!condition) return
  numericEnabled.value[key] = true
  numericThresholds.value[key] = Number(condition.params.threshold)
  numericSeverities.value[key] = condition.severity
  numericMessages.value[key] = condition.label
}

function restoreNonNumericItem(item: AlarmItem) {
  if (!item.presetSlot || !item.conditions[0]) return
  alarmType.value = item.presetSlot === 'text' ? 'text' : item.presetSlot
  alarmNames.value[item.presetSlot] = item.displayName
  severity.value = item.conditions[0].severity
  triggerDelaySeconds.value = item.conditions[0].triggerDelayMs / 1000
  if (item.presetSlot === 'state') {
    const expected = item.conditions[0].params.expected
    if (category.value === 'boolean') booleanExpected.value = Boolean(expected)
    else textExpected.value = String(expected ?? '')
  }
  if (item.presetSlot === 'text')
    textExpected.value = String(item.conditions[0].params.expected ?? '')
  if (item.presetSlot === 'stale')
    staleMinutes.value = Number(item.conditions[0].params.maxAgeMs || 300000) / 60000
}

function applyCommonConditionSettings(
  condition: AlarmCondition,
  conditionSeverity = severity.value,
) {
  if (configurationMode.value === 'advanced') condition.params.configurationMode = 'advanced'
  else delete condition.params.configurationMode
  condition.severity = conditionSeverity
  condition.triggerDelayMs = triggerDelaySeconds.value * 1000
  condition.clearDelayMs =
    configurationMode.value === 'advanced' ? clearDelaySeconds.value * 1000 : 0
  condition.deadband =
    configurationMode.value === 'advanced' &&
    ['threshold', 'range', 'rate_of_change', 'deviation'].includes(condition.kind)
      ? deadband.value
      : 0
  return condition
}

function buildCondition(): AlarmCondition {
  let condition: AlarmCondition
  switch (alarmType.value) {
    case 'state':
      condition = createAlarmCondition('state')
      condition.params = {
        expected: category.value === 'boolean' ? booleanExpected.value : textExpected.value,
      }
      break
    case 'transition':
      condition = createAlarmCondition('transition')
      condition.operator = 'changed'
      break
    case 'text':
      condition = createAlarmCondition('text_match')
      condition.params = { expected: textExpected.value }
      break
    case 'quality':
      condition = createAlarmCondition('quality')
      condition.params = { qualities: ['bad', 'unknown'] }
      break
    case 'stale':
      condition = createAlarmCondition('stale')
      condition.params = { maxAgeMs: staleMinutes.value * 60000 }
      break
    default:
      condition = createAlarmCondition('offline')
  }
  return applyCommonConditionSettings(condition)
}

function buildNumericPayloads(): AlarmItemSave[] {
  const drafts: AlarmItemSave[] = []
  const thresholdConditions = (['hh', 'h', 'l', 'll'] as NumericThresholdKey[])
    .filter((key) => numericEnabled.value[key])
    .map((key) => {
      const condition = createAlarmCondition(
        'threshold',
        numericItems.value.find((item) => item.key === key)?.label,
      )
      condition.operator = key === 'hh' || key === 'h' ? 'gt' : 'lt'
      condition.params = { threshold: numericThresholds.value[key], presetLevel: key }
      condition.label = numericMessages.value[key].trim() || condition.label
      return applyCommonConditionSettings(condition, numericSeverities.value[key])
    })
  const customThresholdConditions = (
    configurationMode.value === 'advanced' ? advancedLimits.value : []
  ).map((item) => {
    const condition = createAlarmCondition('threshold', item.message.trim() || item.name.trim())
    condition.operator = item.direction === 'high' ? 'gt' : 'lt'
    condition.params = {
      threshold: item.threshold,
      presetLevel: 'custom',
      levelName: item.name.trim(),
    }
    return applyCommonConditionSettings(condition, item.severity)
  })
  thresholdConditions.push(...customThresholdConditions)
  if (thresholdConditions.length)
    drafts.push(buildPayload(thresholdConditions, 'highest_matching', 'limit'))

  if (numericEnabled.value.rate) {
    const condition = createAlarmCondition('rate_of_change')
    condition.label = numericMessages.value.rate.trim() || condition.label
    condition.params = {
      direction: rateDirection.value,
      limit: rateLimit.value,
      windowMs: rateWindowSeconds.value * 1000,
    }
    drafts.push(
      buildPayload(
        [applyCommonConditionSettings(condition, numericSeverities.value.rate)],
        'single',
        'rate',
      ),
    )
  }
  if (numericEnabled.value.deviation) {
    const condition = createAlarmCondition('deviation')
    condition.label = numericMessages.value.deviation.trim() || condition.label
    condition.params = {
      baseline: deviationBaseline.value,
      limit: deviationLimit.value,
    }
    drafts.push(
      buildPayload(
        [applyCommonConditionSettings(condition, numericSeverities.value.deviation)],
        'single',
        'deviation',
      ),
    )
  }
  if (numericEnabled.value.offline) {
    const condition = createAlarmCondition('offline')
    condition.label = numericMessages.value.offline.trim() || condition.label
    drafts.push(
      buildPayload(
        [applyCommonConditionSettings(condition, numericSeverities.value.offline)],
        'single',
        'offline',
      ),
    )
  }
  return drafts
}

function buildPayload(
  conditions: AlarmCondition[],
  evaluationMode: 'single' | 'highest_matching',
  presetSlot: AlarmPresetSlot,
): AlarmItemSave {
  const draft = createAlarmItemDraft('point')
  return AlarmItemSaveSchema.parse({
    ...draft,
    datapointId: props.points[0]?.datapointId || '',
    displayName: props.points.length === 1 ? alarmNames.value[presetSlot].trim() : '',
    presetSlot,
    groupId: groupId.value,
    description: null,
    evaluationMode,
    conditions,
    notification:
      configurationMode.value === 'advanced'
        ? {
            mode: notificationMode.value,
            notifyOnRaise: notificationMode.value === 'custom' ? notifyOnRaise.value : null,
            notifyOnClear: notificationMode.value === 'custom' ? notifyOnClear.value : null,
            repeatIntervalSeconds:
              notificationMode.value === 'custom' ? repeatIntervalSeconds.value || null : null,
            channelIds: notificationMode.value === 'custom' ? ['runtime_inapp'] : [],
            messageTemplate: notificationMode.value === 'custom' ? messageTemplate.value : '',
          }
        : {
            mode: 'inherit',
            notifyOnRaise: null,
            notifyOnClear: null,
            repeatIntervalSeconds: null,
            channelIds: [],
            messageTemplate: '',
          },
    isEnabled: isEnabled.value,
  })
}

function localValidationMessage() {
  if (!props.points.length) return t('quickAlarm.validation.selectPoint')
  if (!isCompatible.value) return t('quickAlarm.validation.incompatible')
  if (category.value === 'number') {
    if (!numericDraftCount.value) return t('quickAlarm.validation.enableOne')
    const enabledThresholds = (['hh', 'h', 'l', 'll'] as NumericThresholdKey[]).filter(
      (key) => numericEnabled.value[key],
    )
    if (enabledThresholds.some((key) => typeof numericThresholds.value[key] !== 'number'))
      return t('quickAlarm.validation.thresholdRequired')
    if (
      advancedLimits.value.some((item) => !item.name.trim() || typeof item.threshold !== 'number')
    )
      return t('quickAlarm.validation.additionalRequired')
    const value = (key: NumericThresholdKey) => numericThresholds.value[key] as number
    if (numericEnabled.value.hh && numericEnabled.value.h && value('hh') <= value('h'))
      return t('quickAlarm.validation.hhGreaterH')
    if (numericEnabled.value.ll && numericEnabled.value.l && value('ll') >= value('l'))
      return t('quickAlarm.validation.llLessL')
    if (numericEnabled.value.h && numericEnabled.value.l && value('l') >= value('h'))
      return t('quickAlarm.validation.normalRange')
    if (numericEnabled.value.rate && (!rateLimit.value || rateLimit.value <= 0))
      return t('quickAlarm.validation.ratePositive')
    if (
      numericEnabled.value.deviation &&
      (typeof deviationBaseline.value !== 'number' ||
        typeof deviationLimit.value !== 'number' ||
        deviationLimit.value < 0)
    )
      return t('quickAlarm.validation.deviationRequired')
  }
  if (
    category.value !== 'number' &&
    ['state', 'text'].includes(alarmType.value) &&
    category.value !== 'boolean' &&
    !textExpected.value
  )
    return t('quickAlarm.validation.matchRequired')
  return ''
}

async function submit() {
  const message = localValidationMessage()
  if (message) return ElMessage.warning(message)
  const payloads =
    category.value === 'number'
      ? buildNumericPayloads()
      : [buildPayload([buildCondition()], 'single', nonNumericPresetSlot())]
  const datapointIds = props.points.map((point) => point.datapointId)
  try {
    const validation = await validatePresetAlarmConfiguration(props.projectId, {
      datapointIds,
      drafts: payloads.map(({ datapointId: _datapointId, ...draft }) => draft),
    })
    const errors = validation.errors
    if (errors.length) {
      await ElMessageBox.alert(
        errors[0]?.message || t('quickAlarm.validation.failed'),
        t('quickAlarm.validation.cannotCreate'),
        {
          confirmButtonText: t('quickAlarm.validation.understood'),
        },
      )
      return
    }
    const warnings = validation.warnings
    if (warnings.length) {
      await ElMessageBox.confirm(
        t('quickAlarm.validation.overlap', { count: warnings.length }),
        t('quickAlarm.validation.overlapTitle'),
        {
          confirmButtonText: t('quickAlarm.validation.continueCreate'),
          cancelButtonText: t('quickAlarm.validation.returnEdit'),
          type: 'warning',
        },
      )
      const acknowledgementKeys = warnings.map((item) => item.ackKey).filter(Boolean)
      payloads.forEach((payload) => (payload.acknowledgedWarningKeys = acknowledgementKeys))
    }
    emit('save', payloads.length === 1 ? payloads[0]! : payloads, datapointIds)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('quickAlarm.validation.failed')))
  }
}

function nonNumericPresetSlot(): AlarmPresetSlot {
  return alarmType.value === 'text' ? 'text' : alarmType.value
}
</script>

<style scoped>
.quick-alarm {
  min-height: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
  color: var(--dc-text);
}
.quick-alarm__context {
  display: grid;
  grid-template-columns: minmax(190px, 1fr) 210px 58px;
  gap: 8px;
  align-items: start;
  padding: 9px 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}
.quick-alarm__mode {
  width: fit-content;
  display: inline-grid;
  grid-template-columns: 104px 104px;
  gap: 4px;
  align-items: center;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}
.quick-alarm__mode button {
  height: 30px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--dc-text-secondary);
  font: inherit;
  font-size: 13px;
}
.quick-alarm__mode button.is-active {
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
  font-weight: 700;
  box-shadow: 0 1px 3px rgb(15 23 42 / 8%);
}
.quick-alarm__names {
  display: grid;
  grid-template-columns: 148px minmax(0, 1fr);
  gap: 10px;
  align-items: start;
  padding: 9px 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}
.quick-alarm__names > div:first-child {
  display: grid;
  gap: 3px;
}
.quick-alarm__names strong {
  font-size: 13px;
}
.quick-alarm__names > div:first-child span {
  color: var(--dc-text-muted);
  font-size: 11px;
  line-height: 1.45;
}
.quick-alarm__name-fields {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 7px;
}
.quick-alarm__point,
.quick-alarm__field {
  min-width: 0;
  display: grid;
  gap: 4px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.quick-alarm__point strong,
.quick-alarm__point small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.quick-alarm__point strong {
  color: var(--dc-text);
  font-size: 13px;
}
.quick-alarm__point small {
  color: var(--dc-text-muted);
  font-size: 11px;
}
.quick-alarm__field.is-switch {
  justify-items: start;
}
.quick-alarm__config {
  display: grid;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}
.quick-alarm__config h3 {
  margin: 0;
  font-size: 14px;
}
.quick-alarm__advanced {
  display: grid;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}
.quick-alarm__advanced header h3 {
  margin: 0;
}
.quick-alarm__advanced-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.quick-alarm__advanced-head > div:last-child {
  display: flex;
  gap: 6px;
}
.quick-alarm__advanced-head button {
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: 6px;
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
  font: inherit;
  font-size: 12px;
}
.quick-alarm__limit-list {
  display: grid;
  gap: 5px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--dc-border);
}
.quick-alarm__limit-head,
.quick-alarm__limit-row {
  display: grid;
  grid-template-columns: 100px 72px 100px minmax(120px, 1fr) 78px 30px;
  gap: 7px;
  align-items: center;
}
.quick-alarm__limit-head {
  padding: 0 6px;
  color: var(--dc-text-muted);
  font-size: 11px;
}
.quick-alarm__limit-row {
  padding: 6px;
  border: 1px solid var(--dc-border);
  border-radius: 7px;
  background: var(--dc-surface-muted);
}
.quick-alarm__limit-row :deep(.el-input-number),
.quick-alarm__limit-row :deep(.el-select) {
  width: 100%;
}
.quick-alarm__remove {
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--el-color-danger);
  font-size: 20px;
  line-height: 1;
}
.quick-alarm__remove:hover {
  background: var(--el-color-danger-light-9);
}
.quick-alarm__advanced header h3 {
  font-size: 14px;
}
.quick-alarm__advanced-grid,
.quick-alarm__notification {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  align-items: end;
}
.quick-alarm__notification {
  padding-top: 10px;
  border-top: 1px solid var(--dc-border);
}
.quick-alarm__notification .is-wide {
  grid-column: span 2;
}
.quick-alarm__types {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.quick-alarm__types button {
  height: 30px;
  padding: 0 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font: inherit;
  font-size: 12px;
}
.quick-alarm__types button.is-active {
  border-color: var(--dc-primary);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-weight: 700;
}
.quick-alarm__numeric {
  display: grid;
  gap: 4px;
}
.quick-alarm__numeric-group {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 1px 2px 3px;
}
.quick-alarm__numeric-group.is-divider {
  margin-top: 5px;
  padding-top: 8px;
  border-top: 1px solid var(--dc-border);
}
.quick-alarm__numeric-group strong {
  color: var(--dc-text);
  font-size: 13px;
}
.quick-alarm__numeric-head,
.quick-alarm__numeric-row {
  display: grid;
  grid-template-columns: 32px 58px minmax(0, 1fr) 112px 82px;
  gap: 8px;
  align-items: center;
}
.quick-alarm__numeric-head {
  padding: 0 8px 2px;
  color: var(--dc-text-muted);
  font-size: 12px;
}
.quick-alarm__numeric-row {
  min-height: 38px;
  padding: 4px 7px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}
.quick-alarm__numeric-row.is-enabled {
  border-color: color-mix(in srgb, var(--dc-primary) 42%, var(--dc-border));
  background: var(--dc-primary-soft);
}
.quick-alarm__numeric-row strong {
  color: var(--dc-text-secondary);
  font-size: 13px;
}
.quick-alarm__numeric-params {
  min-width: 0;
  display: grid;
  gap: 6px;
}
.quick-alarm__numeric-params.is-rate {
  grid-template-columns: 92px minmax(0, 1fr) 88px;
}
.quick-alarm__numeric-params.is-deviation {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  color: var(--dc-text-muted);
  font-size: 12px;
}
.quick-alarm__numeric :deep(.el-input-number),
.quick-alarm__numeric :deep(.el-select) {
  width: 100%;
}
.quick-alarm__numeric-common {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: 5px;
  padding-top: 8px;
  border-top: 1px solid var(--dc-border);
}
.quick-alarm__numeric-common label {
  min-width: 0;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 8px;
  align-items: center;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.quick-alarm__params {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  align-items: end;
  padding-top: 9px;
  border-top: 1px solid var(--dc-border);
}
.quick-alarm__field.is-primary {
  grid-column: span 1;
}
.quick-alarm__no-param {
  min-height: 32px;
  display: flex;
  align-items: center;
  grid-column: 1;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.quick-alarm__params :deep(.el-input-number),
.quick-alarm__params :deep(.el-select) {
  width: 100%;
}
.quick-alarm__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  margin: 4px -16px -16px;
  padding: 9px 16px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}
.quick-alarm__footer > div {
  display: flex;
  gap: 8px;
}
.quick-alarm__cancel,
.quick-alarm__primary {
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 0 12px;
  border-radius: var(--dc-radius-sm);
  font: inherit;
  font-size: 13px;
  font-weight: 700;
}
.quick-alarm__cancel {
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}
.quick-alarm__primary {
  border: 1px solid var(--dc-primary);
  background: var(--dc-primary);
  color: white;
}
.quick-alarm__primary:disabled {
  opacity: 0.55;
}
.quick-alarm__icon {
  width: 30px;
  height: 30px;
  display: inline-grid;
  place-items: center;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
}
.quick-alarm__icon svg,
.quick-alarm__primary svg {
  width: 16px;
}
@media (max-width: 620px) {
  .quick-alarm__context,
  .quick-alarm__params {
    grid-template-columns: 1fr;
  }
}
</style>
