<template>
  <DcDrawer v-model="visible" :title="title" :width="520">
    <template #actions>
      <button
        type="button"
        class="history-config__close"
        :title="ui('关闭', 'Close')"
        :aria-label="ui('关闭历史存储设置', 'Close history storage settings')"
        @click="visible = false"
      >
        <IconTablerX />
      </button>
    </template>

    <div class="history-config">
      <section class="history-config__mode-section">
        <h3>{{ ui('保存方式', 'Storage Mode') }}</h3>
        <div class="history-config__modes" role="radiogroup" :aria-label="ui('历史保存方式', 'History storage mode')">
          <button
            v-if="allowInherit"
            type="button"
            :class="{ 'is-active': draft.behavior === 'inherit' }"
            @click="selectBehavior('inherit')"
          >
            <IconTablerArrowBackUp />
            <span><strong>{{ ui('沿用来源设置', 'Inherit Source Settings') }}</strong><small>{{ ui('跟随所属来源', 'Follow the parent source') }}</small></span>
          </button>
          <button
            type="button"
            :class="{ 'is-active': draft.behavior === 'off' }"
            @click="selectBehavior('off')"
          >
            <IconTablerHistoryOff />
            <span><strong>{{ ui('不保存历史', 'Do Not Store History') }}</strong><small>{{ ui('停止新增历史记录', 'Stop creating history records') }}</small></span>
          </button>
          <button
            v-for="mode in modeOptions"
            :key="mode.value"
            type="button"
            :class="{ 'is-active': draft.behavior === 'custom' && draft.writeMode === mode.value }"
            @click="selectMode(mode.value)"
          >
            <component :is="mode.icon" />
            <span>
              <strong>{{ mode.label }}</strong>
              <small>{{ mode.hint }}</small>
            </span>
          </button>
        </div>
      </section>

      <template v-if="draft.behavior === 'custom'">
        <section class="history-config__section">
          <h3>{{ ui('基本设置', 'Basic Settings') }}</h3>
          <label
            v-if="draft.writeMode === 'interval_latest' || draft.writeMode === 'periodic_snapshot'"
            class="history-config__field"
          >
            <span>{{ ui('保存间隔', 'Storage Interval') }}</span>
            <div class="history-config__number-unit">
              <el-input-number v-model="intervalMinutes" :min="1" :controls="false" />
              <span>{{ ui('分钟', 'minutes') }}</span>
            </div>
          </label>

          <div class="history-config__target-row">
            <label class="history-config__field">
              <span>{{ ui('主存储目标', 'Primary Target') }}</span>
              <el-select v-model="primaryTarget.connectionId" :placeholder="ui('请选择存储目标', 'Select a storage target')">
                <el-option
                  v-for="target in availableTargetsFor(0)"
                  :key="target.id"
                  :label="targetLabel(target)"
                  :value="target.id"
                />
              </el-select>
            </label>
            <RetentionField v-model="primaryTarget.retentionDays" />
          </div>
        </section>

        <section class="history-config__section">
          <button
            type="button"
            class="history-config__advanced-trigger"
            @click="advancedVisible = !advancedVisible"
          >
            <span>{{ ui('高级设置', 'Advanced Settings') }}</span>
            <IconTablerChevronDown :class="{ 'is-open': advancedVisible }" />
          </button>

          <div v-if="advancedVisible" class="history-config__advanced">
            <label v-if="draft.writeMode === 'on_change'" class="history-config__field">
              <span>{{ ui('绝对值死区', 'Absolute Deadband') }}</span>
              <el-input-number v-model="draft.deadband" :min="0" :controls="false" />
            </label>
            <label v-if="draft.writeMode === 'on_change'" class="history-config__field">
              <span>{{ ui('最长静默', 'Maximum Silence') }}</span>
              <div class="history-config__number-unit">
                <el-input-number
                  v-model="maxSilenceMinutes"
                  :min="1"
                  :controls="false"
                  :disabled="maxSilenceUnset"
                />
                <span>{{ ui('分钟', 'minutes') }}</span>
              </div>
              <el-checkbox v-model="maxSilenceUnset">{{ ui('不设置最长静默', 'No maximum silence') }}</el-checkbox>
            </label>
            <label v-if="draft.writeMode === 'periodic_snapshot'" class="history-config__field">
              <span>{{ ui('离线时', 'When Offline') }}</span>
              <el-select v-model="draft.offlineBehavior">
                <el-option :label="ui('继续保存并标记异常', 'Continue and mark as stale')" value="store_stale" />
                <el-option :label="ui('停止保存', 'Stop storing')" value="skip" />
              </el-select>
            </label>

            <div class="history-config__additional-head">
              <span>{{ ui('附加目标', 'Additional Targets') }}</span>
              <button type="button" @click="addTarget"><IconTablerPlus /> {{ ui('添加', 'Add') }}</button>
            </div>
            <div
              v-for="(target, index) in draft.targets.slice(1)"
              :key="index + 1"
              class="history-config__target-row is-additional"
            >
              <el-select v-model="target.connectionId" :placeholder="ui('请选择存储目标', 'Select a storage target')">
                <el-option
                  v-for="option in availableTargetsFor(index + 1)"
                  :key="option.id"
                  :label="targetLabel(option)"
                  :value="option.id"
                />
              </el-select>
              <RetentionField v-model="target.retentionDays" />
              <div class="history-config__target-actions">
                <button
                  type="button"
                  :disabled="index === 0"
                  :title="ui('上移', 'Move Up')"
                  @click="moveTarget(index + 1, -1)"
                >
                  <IconTablerArrowUp />
                </button>
                <button
                  type="button"
                  :disabled="index === draft.targets.length - 2"
                  :title="ui('下移', 'Move Down')"
                  @click="moveTarget(index + 1, 1)"
                >
                  <IconTablerArrowDown />
                </button>
                <button type="button" :title="ui('删除', 'Delete')" @click="removeTarget(index + 1)">
                  <IconTablerTrash />
                </button>
              </div>
            </div>
          </div>
        </section>

        <el-alert
          v-if="riskMessage"
          :title="riskMessage"
          type="warning"
          :closable="false"
          show-icon
        />
      </template>

      <p v-if="errorMessage" class="history-config__error">{{ errorMessage }}</p>
      <footer class="history-config__footer">
        <button type="button" class="history-config__cancel" @click="visible = false">{{ ui('取消', 'Cancel') }}</button>
        <button type="button" class="history-config__save" :disabled="saving" @click="submit">
          <IconTablerDeviceFloppy />
          {{ saving ? ui('保存中', 'Saving') : ui('保存', 'Save') }}
        </button>
      </footer>
    </div>
  </DcDrawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessageBox } from 'element-plus'
import IconTablerArrowBackUp from '~icons/tabler/arrow-back-up'
import IconTablerArrowDown from '~icons/tabler/arrow-down'
import IconTablerArrowUp from '~icons/tabler/arrow-up'
import IconTablerChevronDown from '~icons/tabler/chevron-down'
import IconTablerClock from '~icons/tabler/clock'
import IconTablerDeviceFloppy from '~icons/tabler/device-floppy'
import IconTablerHistoryOff from '~icons/tabler/history-off'
import IconTablerListDetails from '~icons/tabler/list-details'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefreshDot from '~icons/tabler/refresh-dot'
import IconTablerRepeat from '~icons/tabler/repeat'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerX from '~icons/tabler/x'
import DcDrawer from '@/components/shared/DcDrawer.vue'
import RetentionField from './HistoryStorageRetentionField.vue'
import {
  buildHistoryStoragePayload,
  createHistoryStorageDraft,
  validateHistoryStorageDraft,
  type HistoryStorageDraft,
} from '@/models/history-storage'
import type {
  HistoryStorageBehavior,
  HistoryStorageConfiguration,
  HistoryStorageTargetOption,
  HistoryStorageWriteMode,
} from '@/api/schemas/history-storage.schema'
import type { HistoryStorageSavePayload } from '@/api/history-storage.api'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title: string
    allowInherit?: boolean
    behavior: HistoryStorageBehavior
    configuration?: HistoryStorageConfiguration | null
    targets: HistoryStorageTargetOption[]
    saving?: boolean
  }>(),
  { allowInherit: false, configuration: null, saving: false },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'save', payload: HistoryStorageSavePayload): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => {
    if (value) emit('update:modelValue', true)
    else void requestClose()
  },
})
const draft = ref<HistoryStorageDraft>(
  createHistoryStorageDraft(props.behavior, props.configuration, props.targets),
)
const advancedVisible = ref(false)
const errorMessage = ref('')
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
    ui('历史存储设置有未保存修改，确认放弃这些修改？', 'History storage settings contain unsaved changes. Discard them?'),
    ui('关闭历史存储设置', 'Close History Storage Settings'),
    { confirmButtonText: ui('放弃修改', 'Discard'), cancelButtonText: ui('继续编辑', 'Keep Editing'), type: 'warning' },
  )
    .then(() => true)
    .catch(() => false)
  if (discard) emit('update:modelValue', false)
}

watch(
  () => [props.modelValue, props.behavior, props.configuration, props.targets] as const,
  () => {
    if (!props.modelValue) return
    draft.value = createHistoryStorageDraft(props.behavior, props.configuration, props.targets)
    initialDraftSnapshot.value = JSON.stringify(draft.value)
    advancedVisible.value = false
    errorMessage.value = ''
  },
  { deep: true },
)

const modeOptions = computed<Array<{
  value: HistoryStorageWriteMode
  label: string
  hint: string
  icon: unknown
}>>(() => [
  {
    value: 'on_change',
    label: ui('变化时保存', 'On Change'),
    hint: ui('值或质量变化时记录', 'Record value or quality changes'),
    icon: IconTablerRefreshDot,
  },
  {
    value: 'interval_latest',
    label: ui('按间隔保存', 'At Intervals'),
    hint: ui('每个时间窗口记录最后值', 'Record the latest value in each interval'),
    icon: IconTablerClock,
  },
  {
    value: 'periodic_snapshot',
    label: ui('按周期保存', 'Periodic Snapshot'),
    hint: ui('定时记录当前最新值', 'Record the current latest value periodically'),
    icon: IconTablerRepeat,
  },
  {
    value: 'every_sample',
    label: ui('保存每次采样', 'Every Sample'),
    hint: ui('记录收到的每一个样本', 'Record every received sample'),
    icon: IconTablerListDetails,
  },
])

const primaryTarget = computed(() => draft.value.targets[0]!)
const intervalMinutes = computed({
  get: () => Math.max(1, Math.round(draft.value.intervalMs / 60_000)),
  set: (value: number) => (draft.value.intervalMs = Math.max(1, value || 1) * 60_000),
})
const maxSilenceMinutes = computed({
  get: () => Math.max(1, Math.round((draft.value.maxSilenceMs || 3_600_000) / 60_000)),
  set: (value: number) => (draft.value.maxSilenceMs = Math.max(1, value || 1) * 60_000),
})
const maxSilenceUnset = computed({
  get: () => draft.value.maxSilenceMs === null,
  set: (value: boolean) => (draft.value.maxSilenceMs = value ? null : 3_600_000),
})
const selectedTargetOptions = computed(
  () =>
    draft.value.targets
      .map((item) => props.targets.find((target) => target.id === item.connectionId))
      .filter(Boolean) as HistoryStorageTargetOption[],
)
const riskMessage = computed(() => {
  if (draft.value.writeMode === 'every_sample') return ui('保存每次采样可能产生较大的历史数据量。', 'Storing every sample may generate a large amount of history data.')
  if (draft.value.targets.some((target) => target.retentionDays === null))
    return ui('永久保留会持续占用存储空间；以后关闭或删除配置，也不会立即删除目标库已有表和数据。', 'Permanent retention continuously consumes storage. Disabling or deleting this configuration does not immediately remove existing target tables or data.')
  if (selectedTargetOptions.value.some((target) => target.lastTestStatus !== 'succeeded'))
    return ui('所选目标尚未通过最近连接测试；目标不可达时不会自动切换主备。', 'A selected target has not passed its latest connection test. Targets do not automatically fail over when unavailable.')
  return ''
})

function selectBehavior(behavior: 'inherit' | 'off') {
  draft.value.behavior = behavior
  errorMessage.value = ''
}

function selectMode(mode: HistoryStorageWriteMode) {
  draft.value.behavior = 'custom'
  draft.value.writeMode = mode
  errorMessage.value = ''
}

function targetLabel(target: HistoryStorageTargetOption) {
  const type = target.type === 'builtin.timeseries' ? ui('IF时序库', 'IF Time-series Database') : 'TDengine'
  const status =
    target.lastTestStatus === 'succeeded'
      ? ui(' · 测试成功', ' · Test Succeeded')
      : target.lastTestStatus === 'failed'
        ? ui(' · 测试失败', ' · Test Failed')
        : ui(' · 未测试', ' · Not Tested')
  return `${target.name} · ${type}${status}`
}

function availableTargetsFor(index: number) {
  const current = draft.value.targets[index]?.connectionId
  const used = new Set(draft.value.targets.map((target) => target.connectionId).filter(Boolean))
  return props.targets.filter((target) => target.id === current || !used.has(target.id))
}

function addTarget() {
  draft.value.targets.push({ connectionId: '', retentionDays: 30 })
  advancedVisible.value = true
}

function removeTarget(index: number) {
  draft.value.targets.splice(index, 1)
}

function moveTarget(index: number, direction: number) {
  const next = index + direction
  if (next < 1 || next >= draft.value.targets.length) return
  const [target] = draft.value.targets.splice(index, 1)
  draft.value.targets.splice(next, 0, target)
}

function submit() {
  const validation = validateHistoryStorageDraft(draft.value)
  if (validation) {
    const validationLabels: Record<string, string> = {
      请选择主存储目标: 'Select a primary storage target',
      请选择所有附加目标: 'Select every additional target',
      存储目标不能重复: 'Storage targets cannot be duplicated',
      保存间隔必须大于0: 'The storage interval must be greater than 0',
      死区不能小于0: 'The deadband cannot be less than 0',
      最长静默必须大于0: 'Maximum silence must be greater than 0',
      保留天数必须为正整数: 'Retention days must be a positive integer',
    }
    errorMessage.value = datacenterLocale.value === 'en' ? validationLabels[validation.replaceAll(' ', '')] || validation : validation
    return
  }
  emit('save', buildHistoryStoragePayload(draft.value))
}
</script>

<style scoped>
.history-config {
  display: flex;
  min-height: 100%;
  flex-direction: column;
  gap: 16px;
}
.history-config__close {
  width: 30px;
  height: 30px;
  display: inline-grid;
  place-items: center;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
}
.history-config__close:hover {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}
.history-config__close:focus-visible,
.history-config__modes button:focus-visible,
.history-config__advanced-trigger:focus-visible,
.history-config__additional-head button:focus-visible,
.history-config__target-actions button:focus-visible,
.history-config__footer button:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px rgba(29, 78, 216, 0.12);
}
.history-config__close svg {
  width: 17px;
  height: 17px;
}
.history-config__mode-section {
  display: grid;
  gap: 8px;
}
.history-config__mode-section h3 {
  margin: 0;
  color: var(--dc-text);
  font-size: 13px;
  letter-spacing: 0;
}
.history-config__modes {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}
.history-config__modes button {
  min-width: 0;
  min-height: 58px;
  display: grid;
  grid-template-columns: 20px minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  text-align: left;
  transition:
    border-color 0.18s ease,
    background 0.18s ease,
    color 0.18s ease;
}
.history-config__modes button:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 30%, var(--dc-border));
  background: var(--dc-surface-muted);
}
.history-config__modes button.is-active {
  border-color: var(--dc-primary);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.history-config__modes svg {
  width: 18px;
  height: 18px;
}
.history-config__modes span {
  min-width: 0;
  display: grid;
  gap: 2px;
}
.history-config__modes strong {
  color: var(--dc-text);
  font-size: 13px;
  letter-spacing: 0;
}
.history-config__modes small {
  overflow: hidden;
  overflow-wrap: anywhere;
  color: var(--dc-text-muted);
  font-size: 11px;
  line-height: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.history-config__modes button.is-active small {
  color: var(--dc-primary);
}
.history-config__section {
  display: grid;
  gap: 12px;
  padding-top: 14px;
  border-top: 1px solid var(--dc-border);
}
.history-config__section h3 {
  margin: 0;
  font-size: 13px;
  letter-spacing: 0;
}
.history-config__field {
  min-width: 0;
  display: grid;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.history-config__field :deep(.el-select),
.history-config__field :deep(.el-input-number) {
  width: 100%;
}
.history-config__number-unit {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 48px;
  align-items: center;
  gap: 8px;
}
.history-config__target-row {
  display: grid;
  grid-template-columns: minmax(0, 1.3fr) minmax(180px, 0.7fr);
  align-items: end;
  gap: 10px;
}
.history-config__target-row.is-additional {
  grid-template-columns: minmax(0, 1fr) minmax(180px, 0.7fr) auto;
}
.history-config__advanced-trigger {
  width: 100%;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--dc-text);
  font-weight: 700;
}
.history-config__advanced-trigger svg {
  width: 18px;
  transition: transform 0.15s ease;
}
.history-config__advanced-trigger svg.is-open {
  transform: rotate(180deg);
}
.history-config__advanced {
  display: grid;
  gap: 12px;
}
.history-config__additional-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.history-config__additional-head button {
  height: 28px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border: 0;
  background: transparent;
  color: var(--dc-primary);
}
.history-config__additional-head button:hover {
  color: color-mix(in srgb, var(--dc-primary) 78%, black);
}
.history-config__target-actions {
  display: flex;
  gap: 4px;
}
.history-config__target-actions button {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border: 1px solid var(--dc-border);
  border-radius: 4px;
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}
.history-config__target-actions button:disabled {
  opacity: 0.35;
}
.history-config__target-actions button:not(:disabled):hover {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.history-config__target-actions svg {
  width: 15px;
}
.history-config__error {
  margin: 0;
  color: var(--el-color-danger);
  font-size: 12px;
}
.history-config__footer {
  margin-top: auto;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 16px;
  border-top: 1px solid var(--dc-border);
}
.history-config__footer button {
  height: 34px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 14px;
  border-radius: 5px;
  font-weight: 600;
}
.history-config__cancel {
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}
.history-config__cancel:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  color: var(--dc-text);
}
.history-config__save {
  border: 1px solid var(--dc-primary);
  background: var(--dc-primary);
  color: white;
}
.history-config__save:not(:disabled):hover {
  background: color-mix(in srgb, var(--dc-primary) 88%, black);
}
.history-config__save:disabled {
  opacity: 0.6;
}
.history-config__save svg {
  width: 16px;
}
@media (max-width: 640px) {
  .history-config__modes,
  .history-config__target-row,
  .history-config__target-row.is-additional {
    grid-template-columns: 1fr;
  }
}
</style>
