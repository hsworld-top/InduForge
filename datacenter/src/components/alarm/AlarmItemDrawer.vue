<template>
  <DcDrawer v-model="visible" :title="drawerTitle" :width="760" :max="920">
    <template #actions>
      <button
        type="button"
        class="alarm-editor__icon"
        :title="t('alarmEditor.close')"
        @click="visible = false"
      >
        <IconTablerX />
      </button>
    </template>

    <form class="alarm-editor" @submit.prevent="submit">
      <section v-if="item" class="alarm-editor__impact">
        {{ t('alarmEditor.editImpact') }}
        <strong>{{
          item.mode === 'point'
            ? item.datapointName || item.path
            : t('alarmEditor.currentComposite')
        }}</strong
        >。
      </section>

      <section class="alarm-editor__context-card">
        <div v-if="showIdentityFields" class="alarm-editor__grid">
          <label v-if="showIdentityFields" class="alarm-editor__field is-wide"
            ><span>{{
              draft.mode === 'derived'
                ? t('alarmEditor.compositeName')
                : t('alarmEditor.alarmNameOptional')
            }}</span
            ><el-input
              v-model="draft.displayName"
              maxlength="100"
              :placeholder="t('alarmEditor.autoName')"
          /></label>
        </div>

        <div class="alarm-editor__context-grid">
          <div class="alarm-editor__point-field">
            <span>{{
              draft.mode === 'derived'
                ? t('alarmEditor.compositeInputs')
                : t('alarmEditor.alarmPoint')
            }}</span>
            <div v-if="draft.selectedPoints.length" class="alarm-editor__point-compact">
              <div>
                <strong>{{ pointPreview }}</strong>
                <small v-if="selectedPointPath" :title="selectedPointPath">
                  {{ selectedPointPath }}
                </small>
              </div>
              <button v-if="canManagePoints" type="button" @click="pointPickerVisible = true">
                {{ t('alarmEditor.adjust') }}
              </button>
            </div>
            <button
              v-else
              type="button"
              class="alarm-editor__choose-point"
              @click="pointPickerVisible = true"
            >
              <IconTablerPlus />{{ t('alarmEditor.selectPoint') }}
            </button>
          </div>
          <label class="alarm-editor__field"
            ><span>{{ t('alarmEditor.directory') }}</span
            ><AlarmGroupSelect
              v-model="draft.groupId"
              :project-id="projectId"
              :initial-label="item?.groupName || ''"
            />
          </label>
          <label class="alarm-editor__field is-switch"
            ><span>{{ t('alarmEditor.enabled') }}</span
            ><el-switch v-model="draft.isEnabled"
          /></label>
        </div>
        <label v-if="showIdentityFields" class="alarm-editor__field"
          ><span>{{ t('alarmEditor.descriptionOptional') }}</span
          ><el-input v-model="draft.description" maxlength="500"
        /></label>
      </section>

      <section v-if="draft.mode === 'derived'" class="alarm-editor__section is-card">
        <div class="alarm-editor__section-head">
          <div>
            <h3>{{ t('alarmEditor.expression') }}</h3>
            <p>{{ t('alarmEditor.expressionHint') }}</p>
          </div>
        </div>
        <div class="alarm-editor__aliases">
          <label v-for="point in draft.selectedPoints" :key="point.datapointId"
            ><span
              >{{ point.name }}<small>{{ formatPointDataType(point.dataType) }}</small></span
            ><el-input v-model="point.inputKey" :placeholder="t('alarmEditor.expressionAlias')"
          /></label>
        </div>
        <el-input
          v-model="draft.derivedExpression"
          type="textarea"
          :rows="3"
          :placeholder="t('alarmEditor.expressionExample')"
        />
      </section>

      <section class="alarm-editor__section is-card">
        <div class="alarm-editor__section-head">
          <div>
            <h3>
              {{
                draft.evaluationMode === 'highest_matching'
                  ? t('alarmEditor.limitLevels')
                  : t('alarmEditor.alarmCondition')
              }}
            </h3>
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
              <IconTablerPlus />{{ t('alarmEditor.addHigh') }}
            </button>
            <button type="button" class="alarm-editor__secondary" @click="addLevel('low')">
              <IconTablerPlus />{{ t('alarmEditor.addLow') }}
            </button>
          </div>
          <div class="alarm-editor__levels">
            <div class="alarm-editor__level-head" aria-hidden="true">
              <span>{{ t('alarmEditor.levelName') }}</span
              ><span>{{ t('alarmEditor.direction') }}</span
              ><span>{{ t('alarmEditor.threshold') }}</span
              ><span>{{ t('alarmEditor.severity') }}</span
              ><span></span>
            </div>
            <article
              v-for="(condition, index) in draft.conditions"
              :key="condition.id"
              :class="['alarm-editor__level', `is-${condition.severity}`]"
            >
              <el-input
                v-model="condition.label"
                class="alarm-editor__level-label"
                :placeholder="t('alarmEditor.levelExample')"
              />
              <el-select
                v-model="condition.operator"
                class="alarm-editor__direction"
                @change="sortLevels"
                ><el-option :label="t('alarmEditor.highLimit')" value="gt" /><el-option
                  :label="t('alarmEditor.lowLimit')"
                  value="lt"
              /></el-select>
              <el-input-number
                :model-value="numberParam(condition, 'threshold')"
                controls-position="right"
                :placeholder="t('alarmEditor.threshold')"
                @update:model-value="setThreshold(condition, $event)"
              />
              <el-select
                v-model="condition.severity"
                class="alarm-editor__severity"
                @change="applyConditionSeverity(condition, $event)"
                ><el-option
                  v-for="option in severityOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
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
                  >{{ t('alarmEditor.deadband')
                  }}<el-input-number
                    v-model="condition.deadband"
                    :min="0"
                    controls-position="right"
                /></label>
                <label
                  >{{ t('alarmEditor.triggerDelayMs')
                  }}<el-input-number
                    v-model="condition.triggerDelayMs"
                    :min="0"
                    controls-position="right"
                /></label>
                <label
                  >{{ t('alarmEditor.clearDelayMs')
                  }}<el-input-number
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
              :label="conditionKindLabel(kind)"
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
              :placeholder="t('alarmEditor.threshold')"
              @update:model-value="setParam(singleCondition, 'threshold', $event)"
          /></template>
          <template v-else-if="singleCondition.kind === 'range'"
            ><el-input-number
              :model-value="numberParam(singleCondition, 'lower')"
              :placeholder="t('alarmEditor.lower')"
              @update:model-value="setParam(singleCondition, 'lower', $event)" /><el-input-number
              :model-value="numberParam(singleCondition, 'upper')"
              :placeholder="t('alarmEditor.upper')"
              @update:model-value="setParam(singleCondition, 'upper', $event)"
          /></template>
          <template v-else-if="singleCondition.kind === 'state'"
            ><el-select
              v-if="stateUsesBooleanOptions"
              :model-value="Boolean(singleCondition.params.expected)"
              :placeholder="t('alarmEditor.expectedState')"
              @update:model-value="setParam(singleCondition, 'expected', $event)"
              ><el-option label="true" :value="true" /><el-option
                label="false"
                :value="false" /></el-select
            ><el-input
              v-else
              :model-value="String(singleCondition.params.expected ?? '')"
              :placeholder="t('alarmEditor.expectedValue')"
              @update:model-value="setParam(singleCondition, 'expected', $event)"
          /></template>
          <template
            v-else-if="
              singleCondition.kind === 'transition' && singleCondition.operator === 'from_to'
            "
            ><el-select
              v-if="isBooleanPoint"
              :model-value="Boolean(singleCondition.params.from)"
              :placeholder="t('alarmEditor.fromValue')"
              @update:model-value="setParam(singleCondition, 'from', $event)"
              ><el-option label="false" :value="false" /><el-option
                label="true"
                :value="true" /></el-select
            ><el-input
              v-else
              :model-value="String(singleCondition.params.from ?? '')"
              :placeholder="t('alarmEditor.fromValue')"
              @update:model-value="setParam(singleCondition, 'from', $event)" /><el-select
              v-if="isBooleanPoint"
              :model-value="Boolean(singleCondition.params.to)"
              :placeholder="t('alarmEditor.toValue')"
              @update:model-value="setParam(singleCondition, 'to', $event)"
              ><el-option label="false" :value="false" /><el-option
                label="true"
                :value="true" /></el-select
            ><el-input
              v-else
              :model-value="String(singleCondition.params.to ?? '')"
              :placeholder="t('alarmEditor.toValue')"
              @update:model-value="setParam(singleCondition, 'to', $event)"
          /></template>
          <template v-else-if="singleCondition.kind === 'text_match'"
            ><el-input
              :model-value="String(singleCondition.params.expected || '')"
              :placeholder="t('alarmEditor.matchText')"
              @update:model-value="setParam(singleCondition, 'expected', $event)"
          /></template>
          <template v-else-if="singleCondition.kind === 'rate_of_change'"
            ><el-select
              :model-value="String(singleCondition.params.direction || 'absolute')"
              @update:model-value="setParam(singleCondition, 'direction', $event)"
              ><el-option :label="t('alarmEditor.riseRate')" value="rise" /><el-option
                :label="t('alarmEditor.fallRate')"
                value="fall" /><el-option
                :label="t('alarmEditor.absoluteRate')"
                value="absolute" /></el-select
            ><el-input-number
              :model-value="numberParam(singleCondition, 'limit')"
              :min="0"
              :placeholder="t('alarmEditor.rateLimit')"
              @update:model-value="setParam(singleCondition, 'limit', $event)" /><el-input-number
              :model-value="numberParam(singleCondition, 'windowMs')"
              :min="1"
              :placeholder="t('alarmEditor.windowMs')"
              @update:model-value="setParam(singleCondition, 'windowMs', $event)"
          /></template>
          <template v-else-if="singleCondition.kind === 'deviation'"
            ><el-input-number
              :model-value="numberParam(singleCondition, 'baseline')"
              :placeholder="t('alarmEditor.baseline')"
              @update:model-value="setParam(singleCondition, 'baseline', $event)" /><el-input-number
              :model-value="numberParam(singleCondition, 'limit')"
              :min="0"
              :placeholder="t('alarmEditor.deviation')"
              @update:model-value="setParam(singleCondition, 'limit', $event)"
          /></template>
          <template v-else-if="singleCondition.kind === 'quality'">
            <el-select
              :model-value="qualityParams(singleCondition)"
              multiple
              :placeholder="t('alarmEditor.selectBadQuality')"
              @update:model-value="setParam(singleCondition, 'qualities', $event)"
              ><el-option :label="t('alarmEditor.bad')" value="bad" /><el-option
                :label="t('alarmEditor.unknownUncertain')"
                value="unknown"
            /></el-select>
          </template>
          <template v-else-if="singleCondition.kind === 'stale'">
            <el-input-number
              :model-value="staleDisplayValue(singleCondition)"
              :min="1"
              :placeholder="t('alarmEditor.staleTime')"
              @update:model-value="setStaleDisplayValue(singleCondition, $event)"
            />
            <el-select v-model="staleUnit" @change="refreshStaleDisplay">
              <el-option :label="t('alarmEditor.seconds')" value="seconds" />
              <el-option :label="t('alarmEditor.minutes')" value="minutes" />
              <el-option :label="t('alarmEditor.hours')" value="hours" />
            </el-select>
          </template>
          <el-select
            v-model="singleCondition.severity"
            @change="applyConditionSeverity(singleCondition, $event)"
            ><el-option
              v-for="option in severityOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
          /></el-select>
          <div v-if="advancedVisible" class="alarm-editor__level-advanced">
            <label v-if="conditionSupportsDeadband(singleCondition.kind)"
              >{{ t('alarmEditor.deadband')
              }}<el-input-number v-model="singleCondition.deadband" :min="0" /></label
            ><label v-if="singleCondition.kind !== 'transition'"
              >{{ t('alarmEditor.triggerDelayMs')
              }}<el-input-number v-model="singleCondition.triggerDelayMs" :min="0" /></label
            ><label
              >{{ t('alarmEditor.clearDelayMs')
              }}<el-input-number v-model="singleCondition.clearDelayMs" :min="0"
            /></label>
          </div>
        </article>
      </section>

      <section class="alarm-editor__section is-card is-collapsible">
        <button
          type="button"
          class="alarm-editor__collapse"
          @click="advancedVisible = !advancedVisible"
        >
          <span>{{ t('alarmEditor.advanced') }}</span
          ><IconTablerChevronDown :class="{ 'is-open': advancedVisible }" />
        </button>
        <div v-if="advancedVisible" class="alarm-editor__advanced">
          <label class="alarm-editor__field"
            ><span>{{ t('alarmEditor.notification') }}</span
            ><el-select v-model="draft.notification.mode" @change="changeNotificationMode"
              ><el-option
                :label="t('alarmEditor.inheritNotifications')"
                value="inherit" /><el-option
                :label="t('alarmEditor.noNotifications')"
                value="off" /><el-option
                :label="t('alarmEditor.customNotifications')"
                value="custom" /></el-select
          ></label>
          <template v-if="draft.notification.mode === 'custom'"
            ><div class="alarm-editor__checks">
              <el-checkbox v-model="draft.notification.notifyOnRaise">{{
                t('alarmEditor.notifyRaise')
              }}</el-checkbox
              ><el-checkbox v-model="draft.notification.notifyOnClear">{{
                t('alarmEditor.notifyClear')
              }}</el-checkbox>
            </div>
            <label class="alarm-editor__field"
              ><span>{{ t('alarmEditor.channels') }}</span
              ><el-select v-model="draft.notification.channelIds" multiple
                ><el-option
                  v-for="channel in enabledChannels"
                  :key="channel.id"
                  :label="channel.name"
                  :value="channel.id" /></el-select></label
          ></template>
        </div>
      </section>

      <section class="alarm-editor__section is-card is-collapsible">
        <button type="button" class="alarm-editor__collapse" @click="debugVisible = !debugVisible">
          <span>{{ t('alarmEditor.debug') }}</span
          ><IconTablerChevronDown :class="{ 'is-open': debugVisible }" />
        </button>
        <div v-if="debugVisible" class="alarm-editor__debug">
          <div class="alarm-editor__trial-table">
            <div class="alarm-editor__trial-row is-head">
              <span>{{ t('alarmEditor.offsetMs') }}</span
              ><span>{{
                draft.mode === 'derived' ? t('alarmEditor.inputJson') : t('alarmEditor.value')
              }}</span
              ><span>{{ t('alarmEditor.quality') }}</span
              ><span>{{ t('alarmEditor.offline') }}</span
              ><span>{{ t('alarmEditor.sourceTime') }}</span
              ><span></span>
            </div>
            <div v-for="row in trialRows" :key="row.id" class="alarm-editor__trial-row">
              <el-input-number v-model="row.offsetMs" :min="0" controls-position="right" />
              <el-input
                v-if="draft.mode === 'derived'"
                v-model="row.inputsText"
                placeholder='{"temperature": 80, "running": true}'
              />
              <el-input v-else v-model="row.valueText" :placeholder="t('alarmEditor.mockValue')" />
              <el-select v-model="row.quality"
                ><el-option :label="t('alarmEditor.good')" value="good" /><el-option
                  :label="t('alarmEditor.bad')"
                  value="bad" /><el-option :label="t('alarmEditor.unknown')" value="unknown"
              /></el-select>
              <el-checkbox v-model="row.offline" />
              <div class="alarm-editor__source-time">
                <el-checkbox v-model="row.hasSourceTimestamp">{{
                  t('alarmEditor.yes')
                }}</el-checkbox>
                <el-input-number
                  v-model="row.sourceAgeMs"
                  :disabled="!row.hasSourceTimestamp"
                  :min="0"
                  controls-position="right"
                />
              </div>
              <button
                type="button"
                class="alarm-editor__icon is-danger"
                :disabled="trialRows.length === 1"
                @click="removeTrialRow(row.id)"
              >
                <IconTablerTrash />
              </button>
            </div>
          </div>
          <button type="button" class="alarm-editor__secondary" @click="addTrialRow">
            <IconTablerPlus />{{ t('alarmEditor.addSample') }}
          </button>
          <button
            type="button"
            class="alarm-editor__secondary"
            :disabled="trialLoading"
            @click="runTrial"
          >
            {{ t('alarmEditor.trial') }}</button
          ><button
            v-if="item"
            type="button"
            class="alarm-editor__secondary"
            :disabled="contractLoading"
            @click="loadContract"
          >
            {{ t('alarmEditor.contract') }}
          </button>
          <div v-if="trialResult" class="alarm-editor__trial-result">
            <div class="alarm-editor__trial-result-head">
              <strong>{{ t('alarmEditor.trialProcess') }}</strong>
              <span>{{
                trialResult.triggered ? t('alarmEditor.finalAlarm') : t('alarmEditor.finalNormal')
              }}</span>
            </div>
            <div class="alarm-editor__result-table">
              <div class="alarm-editor__result-row is-head">
                <span>{{ t('alarmEditor.time') }}</span
                ><span>{{ t('alarmEditor.evaluation') }}</span
                ><span>{{ t('alarmEditor.status') }}</span
                ><span>{{ t('alarmEditor.activeLevel') }}</span
                ><span>{{ t('alarmEditor.candidateLevel') }}</span
                ><span>{{ t('alarmEditor.remainingDelay') }}</span
                ><span>{{ t('alarmEditor.details') }}</span>
              </div>
              <div
                v-for="(step, index) in trialResult.steps"
                :key="`${step.observedAt}-${index}`"
                class="alarm-editor__result-row"
              >
                <span>{{ formatTrialTime(step.observedAt) }}</span>
                <span>{{ trialEvaluationLabel(step.evaluationState) }}</span>
                <span>{{ trialStateLabel(step.state) }}</span>
                <span>{{ step.activeCondition?.label || '—' }}</span>
                <span>{{ step.candidateCondition?.label || '—' }}</span>
                <span
                  >{{ step.remainingTriggerDelayMs }} / {{ step.remainingClearDelayMs }} ms</span
                >
                <span>{{ trialReasonLabel(step.reason, step.calculatedRate) }}</span>
              </div>
            </div>
          </div>
          <pre v-if="debugOutput">{{ debugOutput }}</pre>
        </div>
      </section>

      <footer class="alarm-editor__footer">
        <button type="button" class="alarm-editor__cancel" @click="visible = false">
          {{ t('alarmEditor.cancel') }}</button
        ><button type="submit" class="alarm-editor__primary" :disabled="saving">
          <IconTablerDeviceFloppy />{{ saveButtonText }}
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
import { secureRandomUUID } from '@/utils/secure-random-uuid'
import { computed, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import IconTablerChevronDown from '~icons/tabler/chevron-down'
import IconTablerDeviceFloppy from '~icons/tabler/device-floppy'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerX from '~icons/tabler/x'
import AlarmDatapointManagerDialog from './AlarmDatapointManagerDialog.vue'
import DcDrawer from '@/components/shared/DcDrawer.vue'
import AlarmGroupSelect from './AlarmGroupSelect.vue'
import type {
  AlarmCondition,
  AlarmConditionKind,
  AlarmItem,
  AlarmItemMode,
  AlarmItemSave,
  AlarmNotificationChannel,
  AlarmTrialResult,
  AlarmTrialSample,
} from '@/api/schemas/alarm.schema'
import { AlarmItemSaveSchema } from '@/api/schemas/alarm.schema'
import {
  getAlarmItemContract,
  testAlarmItem,
  validateBatchCreateAlarmItems,
  validateAlarmItem,
} from '@/api/alarm.api'
import {
  alarmPointCategory,
  buildAlarmItemPayload,
  compatibleAlarmPoints,
  conditionKindsFor,
  alarmItemToDraft,
  createAlarmCondition,
  createAlarmItemDraft,
  sortAlarmLevels,
  type AlarmItemDraft,
  type AlarmPointSelection,
} from '@/models/alarm-item'
import { useAlarmLevelDefinitions } from '@/composables/useAlarmLevelDefinitions'
import { getApiErrorMessage } from '@/utils/request'
import { datacenterLocale, t } from '@/i18n/runtime'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    projectId: string
    mode?: AlarmItemMode
    item?: AlarmItem | null
    channels?: AlarmNotificationChannel[]
    initialPoints?: AlarmPointSelection[]
    saving?: boolean
  }>(),
  {
    mode: 'point',
    item: null,
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
  set: (value) => {
    if (value) emit('update:modelValue', true)
    else void requestClose()
  },
})
const draft = ref<AlarmItemDraft>(createAlarmItemDraft(props.mode))
const initialDraftSnapshot = ref('')
const isDirty = computed(
  () => props.modelValue && JSON.stringify(draft.value) !== initialDraftSnapshot.value,
)

async function requestClose() {
  if (!isDirty.value) {
    emit('update:modelValue', false)
    return
  }
  const discard = await ElMessageBox.confirm(
    t('alarmEditor.discardConfirm'),
    t('alarmEditor.closeEditor'),
    {
      confirmButtonText: t('alarmEditor.discard'),
      cancelButtonText: t('alarmEditor.continueEdit'),
      type: 'warning',
    },
  )
    .then(() => true)
    .catch(() => false)
  if (discard) emit('update:modelValue', false)
}
const pointPickerVisible = ref(false)
const advancedVisible = ref(false)
const debugVisible = ref(false)
type TrialQuality = 'good' | 'bad' | 'unknown'
type TrialRow = {
  id: string
  offsetMs: number
  valueText: string
  inputsText: string
  quality: TrialQuality
  offline: boolean
  sourceAgeMs: number
  hasSourceTimestamp: boolean
}
const createTrialRow = (offsetMs = 0): TrialRow => ({
  id: secureRandomUUID(),
  offsetMs,
  valueText: '',
  inputsText: '{}',
  quality: 'good',
  offline: false,
  sourceAgeMs: 0,
  hasSourceTimestamp: true,
})
const trialRows = ref<TrialRow[]>([createTrialRow(0), createTrialRow(1000)])
const staleUnit = ref<'seconds' | 'minutes' | 'hours'>('minutes')
const debugOutput = ref('')
const trialResult = ref<AlarmTrialResult | null>(null)
const trialLoading = ref(false)
const contractLoading = ref(false)
const { definitions: severityDefinitions, loadDefinitions } = useAlarmLevelDefinitions(
  () => props.projectId,
)
const severityOptions = computed(() =>
  severityDefinitions.value.map((item) => ({
    value: item.key,
    label:
      datacenterLocale.value === 'en' && ['info', 'warning', 'major', 'critical'].includes(item.key)
        ? t(`alarm.severities.${item.key}`)
        : item.displayName,
  })),
)
const evaluationOptions = computed(() => [
  { label: t('alarmEditor.limitMode'), value: 'highest_matching' },
  { label: t('alarmEditor.otherMode'), value: 'single' },
])
const drawerTitle = computed(() =>
  props.item
    ? t('alarmEditor.editTitle', { name: props.item.displayName })
    : props.mode === 'derived'
      ? t('alarmEditor.newComposite')
      : t('alarmEditor.newAlarm'),
)
const saveButtonText = computed(() =>
  props.saving
    ? t('alarmEditor.saving')
    : props.item
      ? t('alarmEditor.saveChanges')
      : t('alarmEditor.createAlarm'),
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
const isFixedPointCreate = computed(
  () => !props.item && draft.value.mode === 'point' && props.initialPoints.length === 1,
)
const canManagePoints = computed(
  () => draft.value.mode === 'derived' || (!props.item && !isFixedPointCreate.value),
)
const selectedPointPath = computed(() => {
  if (draft.value.selectedPoints.length !== 1)
    return t('alarmEditor.pointCount', { count: selectedPointCount.value })
  return draft.value.selectedPoints[0]?.path || ''
})
const conditionHint = computed(() =>
  draft.value.evaluationMode === 'highest_matching'
    ? t('alarmEditor.limitHint')
    : t('alarmEditor.singleHint'),
)

watch(
  () => props.modelValue,
  (opened) => {
    if (!opened) return
    void loadDefinitions()
    const next = props.item ? alarmItemToDraft(props.item) : createAlarmItemDraft(props.mode)
    if (!props.item && props.initialPoints.length)
      next.selectedPoints = props.initialPoints.map((item) => ({ ...item }))
    if (
      !props.item &&
      next.mode === 'point' &&
      next.selectedPoints.length &&
      alarmPointCategory(next.selectedPoints[0]?.dataType) !== 'number'
    ) {
      next.evaluationMode = 'single'
      next.conditions = [
        createAlarmCondition(conditionKindsFor(next.selectedPoints, next.mode, 'single')[0]),
      ]
    }
    draft.value = next
    advancedVisible.value = false
    debugVisible.value = false
    debugOutput.value = ''
    trialResult.value = null
    trialRows.value = [createTrialRow(0), createTrialRow(1000)]
    const stale = next.conditions.find((condition) => condition.kind === 'stale')
    if (stale) staleUnit.value = inferStaleUnit(numberParam(stale, 'maxAgeMs') || 60000)
    initialDraftSnapshot.value = JSON.stringify(draft.value)
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
      ? createAlarmCondition('threshold', t('alarmEditor.high'))
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
        ? `${t('alarmEditor.high')}${t('alarmEditor.high').repeat(sameDirection)}`
        : t('alarmEditor.high')
      : sameDirection
        ? `${t('alarmEditor.low')}${t('alarmEditor.low').repeat(sameDirection)}`
        : t('alarmEditor.low'),
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
    createAlarmCondition(
      kind,
      draft.value.mode === 'derived' ? t('alarmEditor.resultCondition') : '',
    ),
  ]
}
function setParam(condition: AlarmCondition, key: string, value: unknown) {
  condition.params = { ...condition.params, [key]: value }
}
function applyConditionSeverity(condition: AlarmCondition, value: unknown) {
  condition.severity = String(value)
}
function numberParam(condition: AlarmCondition, key: string) {
  const value = condition.params[key]
  return typeof value === 'number' ? value : undefined
}
function formatPointDataType(dataType?: string) {
  return String(dataType || t('alarmEditor.unknownType')).toLowerCase()
}
function conditionKindLabel(kind: AlarmConditionKind) {
  return t(`alarm.conditionLabels.${kind}`)
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
      { label: t('alarmEditor.operators.gt'), value: 'gt' },
      { label: t('alarmEditor.operators.gte'), value: 'gte' },
      { label: t('alarmEditor.operators.lt'), value: 'lt' },
      { label: t('alarmEditor.operators.lte'), value: 'lte' },
    ],
    range: [
      { label: t('alarmEditor.operators.outside'), value: 'outside' },
      { label: t('alarmEditor.operators.between'), value: 'between' },
    ],
    state: [
      { label: t('alarmEditor.operators.eq'), value: 'eq' },
      { label: t('alarmEditor.operators.ne'), value: 'ne' },
    ],
    transition: [
      { label: t('alarmEditor.operators.changed'), value: 'changed' },
      { label: t('alarmEditor.operators.rising'), value: 'rising' },
      { label: t('alarmEditor.operators.falling'), value: 'falling' },
      { label: t('alarmEditor.operators.from_to'), value: 'from_to' },
    ],
    text_match: [
      { label: t('alarmEditor.operators.contains'), value: 'contains' },
      { label: t('alarmEditor.operators.eq'), value: 'eq' },
      { label: t('alarmEditor.operators.ne'), value: 'ne' },
      { label: t('alarmEditor.operators.regex'), value: 'regex' },
    ],
    rate_of_change: [
      { label: t('alarmEditor.operators.gt'), value: 'gt' },
      { label: t('alarmEditor.operators.gte'), value: 'gte' },
      { label: t('alarmEditor.operators.lt'), value: 'lt' },
      { label: t('alarmEditor.operators.lte'), value: 'lte' },
    ],
    deviation: [{ label: t('alarmEditor.operators.gt'), value: 'gt' }],
    offline: [{ label: t('alarmEditor.operators.is_offline'), value: 'is_offline' }],
    quality: [{ label: t('alarmEditor.operators.in'), value: 'in' }],
    stale: [{ label: t('alarmEditor.operators.age_gte'), value: 'age_gte' }],
  }
  return options[kind]
}

function localValidate() {
  if (draft.value.mode === 'derived' && !draft.value.displayName.trim())
    return t('alarmEditor.validations.compositeName')
  if (!selectedPointCount.value) return t('alarmEditor.validations.selectPoint')
  if (draft.value.mode === 'point' && !compatibleAlarmPoints(draft.value.selectedPoints))
    return t('alarmEditor.validations.incompatible')
  if (draft.value.mode === 'derived' && draft.value.selectedPoints.length < 2)
    return t('alarmEditor.validations.compositeInputs')
  if (draft.value.mode === 'derived' && !draft.value.derivedExpression.trim())
    return t('alarmEditor.validations.expression')
  if (
    draft.value.mode === 'derived' &&
    draft.value.selectedPoints.some(
      (point) => !/^[A-Za-z_][A-Za-z0-9_]*$/.test(point.inputKey?.trim() || ''),
    )
  )
    return t('alarmEditor.validations.alias')
  if (draft.value.notification.mode === 'custom' && !draft.value.notification.channelIds.length)
    return t('alarmEditor.validations.channel')
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
        conflictPoints.length > 5
          ? `${conflictPreview}… (${t('alarmEditor.pointCount', { count: conflictPoints.length })})`
          : conflictPreview
      await ElMessageBox.alert(
        `${conflict.message}${conflictPreview ? `：${conflictSuffix}` : ''}`,
        t('alarmEditor.cannotSave'),
        {
          confirmButtonText: conflict.conflictAlarmItemId
            ? t('alarmEditor.openExisting')
            : t('alarmEditor.understood'),
        },
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
        t('alarmEditor.conflictConfirm', {
          count: total,
          preview: preview ? ` (${preview}${total > 5 ? '…' : ''})` : '',
        }),
        t('alarmEditor.conflictTitle'),
        {
          confirmButtonText: t('alarmEditor.confirmSave'),
          cancelButtonText: t('alarmEditor.returnEdit'),
          type: 'warning',
        },
      )
      payload = {
        ...payload,
        acknowledgedWarningKeys: validation.warnings.map((item) => item.ackKey).filter(Boolean),
      }
    }
    emit('save', payload, datapointIds)
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(getApiErrorMessage(error, t('alarmEditor.validations.validateFailed')))
  }
}
async function runTrial() {
  const message = localValidate()
  if (message) return ElMessage.warning(message)
  trialLoading.value = true
  try {
    const baseTime = Date.now()
    const samples: AlarmTrialSample[] = [...trialRows.value]
      .sort((left, right) => left.offsetMs - right.offsetMs)
      .map((row) => {
        const observedAt = new Date(baseTime + row.offsetMs)
        const sample: AlarmTrialSample = {
          observedAt: observedAt.toISOString(),
          quality: row.quality,
          offline: row.offline,
        }
        if (row.hasSourceTimestamp)
          sample.sourceTimestamp = new Date(observedAt.getTime() - row.sourceAgeMs).toISOString()
        if (draft.value.mode === 'derived') {
          const inputs = JSON.parse(row.inputsText || '{}') as Record<string, unknown>
          sample.inputs = inputs
        } else {
          sample.value = parseTrialValue(row.valueText)
        }
        return sample
      })
    trialResult.value = await testAlarmItem(
      props.projectId,
      {
        ...buildAlarmItemPayload(draft.value),
      },
      samples,
    )
    debugOutput.value = ''
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarmEditor.validations.trialFailed')))
  } finally {
    trialLoading.value = false
  }
}
function addTrialRow() {
  const lastOffset = trialRows.value.at(-1)?.offsetMs || 0
  trialRows.value.push(createTrialRow(lastOffset + 1000))
}
function removeTrialRow(id: string) {
  if (trialRows.value.length > 1) trialRows.value = trialRows.value.filter((row) => row.id !== id)
}
function conditionSupportsDeadband(kind: AlarmConditionKind) {
  return ['threshold', 'range', 'rate_of_change', 'deviation'].includes(kind)
}
function qualityParams(condition: AlarmCondition) {
  return Array.isArray(condition.params.qualities)
    ? condition.params.qualities.filter((value): value is string => typeof value === 'string')
    : []
}
function staleUnitMultiplier() {
  return staleUnit.value === 'hours' ? 3600000 : staleUnit.value === 'minutes' ? 60000 : 1000
}
function staleDisplayValue(condition: AlarmCondition) {
  return (numberParam(condition, 'maxAgeMs') || 0) / staleUnitMultiplier()
}
function setStaleDisplayValue(condition: AlarmCondition, value: number | undefined) {
  setParam(condition, 'maxAgeMs', typeof value === 'number' ? value * staleUnitMultiplier() : value)
}
function refreshStaleDisplay() {
  // maxAgeMs 是唯一持久化值，切换单位只改变展示换算。
}
function inferStaleUnit(value: number): 'seconds' | 'minutes' | 'hours' {
  if (value % 3600000 === 0) return 'hours'
  if (value % 60000 === 0) return 'minutes'
  return 'seconds'
}
function parseTrialValue(value: string): string | number | boolean {
  if (value === 'true') return true
  if (value === 'false') return false
  return Number.isNaN(Number(value)) || value.trim() === '' ? value : Number(value)
}
async function loadContract() {
  if (!props.item) return
  const itemId = props.item.id
  contractLoading.value = true
  try {
    trialResult.value = null
    const result = await getAlarmItemContract(props.projectId, itemId)
    if (props.item?.id === itemId) debugOutput.value = JSON.stringify(result, null, 2)
  } catch (error) {
    if (props.item?.id !== itemId) return
    ElMessage.error(getApiErrorMessage(error, t('alarmEditor.validations.contractFailed')))
  } finally {
    if (props.item?.id === itemId) contractLoading.value = false
  }
}
function formatTrialTime(value: string) {
  return new Intl.DateTimeFormat(datacenterLocale.value, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    fractionalSecondDigits: 3,
    hour12: false,
  }).format(new Date(value))
}
function trialEvaluationLabel(value: AlarmTrialResult['steps'][number]['evaluationState']) {
  return t(`alarmEditor.evaluations.${value}`)
}
function trialStateLabel(value: AlarmTrialResult['steps'][number]['state']) {
  return t(`alarmEditor.states.${value}`)
}
function trialReasonLabel(reason: string, rate?: number | null) {
  const base = t(`alarmEditor.reasons.${reason}`)
  return typeof rate === 'number'
    ? t('alarmEditor.rateDetail', { reason: base, rate: rate.toFixed(4) })
    : base
}
</script>

<style scoped>
.alarm-editor {
  min-height: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
  color: var(--dc-text);
}
.alarm-editor__section {
  display: grid;
  gap: 12px;
}
.alarm-editor__section.is-card {
  padding: 14px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}
.alarm-editor__context-card {
  display: grid;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
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
  grid-template-columns: minmax(0, 1fr);
  gap: 12px;
  align-items: end;
}
.alarm-editor__context-grid {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) 240px 64px;
  gap: 10px;
  align-items: start;
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
  margin: 2px 0 0;
  font-size: 15px;
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
  height: 34px;
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
.alarm-editor__point-field {
  min-width: 0;
  display: grid;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.alarm-editor__point-compact {
  min-height: 32px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 5px 9px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}
.alarm-editor__point-compact div {
  min-width: 0;
  display: grid;
  gap: 1px;
}
.alarm-editor__point-compact strong,
.alarm-editor__point-compact small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.alarm-editor__point-compact strong {
  color: var(--dc-text);
  font-size: 12px;
}
.alarm-editor__point-compact small {
  color: var(--dc-text-muted);
  font-size: 10px;
}
.alarm-editor__point-compact button {
  flex: 0 0 auto;
  border: 0;
  background: none;
  color: var(--dc-primary);
  font: inherit;
  font-size: 11px;
  font-weight: 700;
}
.alarm-editor__choose-point {
  min-height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
  font: inherit;
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
  gap: 6px;
}
.alarm-editor__level-head {
  display: grid;
  grid-template-columns: 100px 90px minmax(120px, 1fr) 100px 30px;
  gap: 8px;
  padding: 0 10px;
  color: var(--dc-text-muted);
  font-size: 10px;
  font-weight: 700;
}
.alarm-editor__level {
  display: grid;
  grid-template-columns: 100px 90px minmax(120px, 1fr) 100px 30px;
  gap: 8px;
  align-items: center;
  padding: 10px 10px 10px 8px;
  border: 1px solid var(--dc-border);
  border-left: 3px solid var(--dc-primary);
  border-radius: 8px;
  background: var(--dc-surface-raised);
}
.alarm-editor__level.is-warning {
  border-left-color: var(--dc-warning);
}
.alarm-editor__level.is-major,
.alarm-editor__level.is-critical {
  border-left-color: var(--dc-danger);
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
.alarm-editor__collapse span {
  display: inline-flex;
  align-items: center;
  gap: 8px;
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
  grid-template-columns: auto auto minmax(0, 1fr);
  gap: 8px;
}
.alarm-editor__trial-table {
  grid-column: 1/-1;
  overflow-x: auto;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
}
.alarm-editor__trial-row {
  min-width: 760px;
  display: grid;
  grid-template-columns: 120px minmax(180px, 1fr) 120px 56px 130px 34px;
  gap: 8px;
  align-items: center;
  padding: 7px 9px;
  border-top: 1px solid var(--dc-border);
}
.alarm-editor__trial-row:first-child {
  border-top: 0;
}
.alarm-editor__trial-row.is-head {
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}
.alarm-editor__source-time {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 6px;
  align-items: center;
}
.alarm-editor__trial-result {
  grid-column: 1/-1;
  display: grid;
  gap: 8px;
}
.alarm-editor__trial-result-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.alarm-editor__trial-result-head span {
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.alarm-editor__result-table {
  overflow-x: auto;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
}
.alarm-editor__result-row {
  min-width: 960px;
  display: grid;
  grid-template-columns: 105px 72px 86px 100px 100px 150px minmax(180px, 1fr);
  gap: 8px;
  align-items: center;
  padding: 8px 10px;
  border-top: 1px solid var(--dc-border);
  font-size: 12px;
}
.alarm-editor__result-row:first-child {
  border-top: 0;
}
.alarm-editor__result-row.is-head {
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
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
  margin: 4px -16px -16px;
  padding: 12px 16px;
  background: var(--dc-surface-raised);
  border-top: 1px solid var(--dc-border);
}
@media (max-width: 760px) {
  .alarm-editor__grid,
  .alarm-editor__context-grid,
  .alarm-editor__aliases {
    grid-template-columns: 1fr;
  }
  .alarm-editor__level,
  .alarm-editor__single-condition {
    grid-template-columns: 1fr 1fr;
  }
  .alarm-editor__level-head {
    display: none;
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
}
</style>
