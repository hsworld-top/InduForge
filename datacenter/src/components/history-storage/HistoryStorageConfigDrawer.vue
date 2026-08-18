<template>
  <DcDrawer v-model="visible" :title="title" :width="520">
    <template #actions>
      <button
        type="button"
        class="history-config__close"
        title="关闭"
        aria-label="关闭历史存储设置"
        @click="visible = false"
      >
        <IconTablerX />
      </button>
    </template>

    <div class="history-config">
      <section class="history-config__mode-section">
        <h3>保存方式</h3>
        <div class="history-config__modes" role="radiogroup" aria-label="历史保存方式">
          <button
            v-if="allowInherit"
            type="button"
            :class="{ 'is-active': draft.behavior === 'inherit' }"
            @click="selectBehavior('inherit')"
          >
            <IconTablerArrowBackUp />
            <span><strong>沿用来源设置</strong><small>跟随所属来源</small></span>
          </button>
          <button
            type="button"
            :class="{ 'is-active': draft.behavior === 'off' }"
            @click="selectBehavior('off')"
          >
            <IconTablerHistoryOff />
            <span><strong>不保存历史</strong><small>只保留当前值</small></span>
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
          <h3>基本设置</h3>
          <label
            v-if="draft.writeMode === 'interval_latest' || draft.writeMode === 'periodic_snapshot'"
            class="history-config__field"
          >
            <span>保存间隔</span>
            <div class="history-config__number-unit">
              <el-input-number v-model="intervalMinutes" :min="1" :controls="false" />
              <span>分钟</span>
            </div>
          </label>

          <div class="history-config__target-row">
            <label class="history-config__field">
              <span>主存储目标</span>
              <el-select v-model="primaryTarget.connectionId" placeholder="请选择存储目标">
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
            <span>高级设置</span>
            <IconTablerChevronDown :class="{ 'is-open': advancedVisible }" />
          </button>

          <div v-if="advancedVisible" class="history-config__advanced">
            <label v-if="draft.writeMode === 'on_change'" class="history-config__field">
              <span>绝对值死区</span>
              <el-input-number v-model="draft.deadband" :min="0" :controls="false" />
            </label>
            <label v-if="draft.writeMode === 'on_change'" class="history-config__field">
              <span>最长静默</span>
              <div class="history-config__number-unit">
                <el-input-number
                  v-model="maxSilenceMinutes"
                  :min="1"
                  :controls="false"
                  :disabled="maxSilenceUnset"
                />
                <span>分钟</span>
              </div>
              <el-checkbox v-model="maxSilenceUnset">不设置最长静默</el-checkbox>
            </label>
            <label v-if="draft.writeMode === 'periodic_snapshot'" class="history-config__field">
              <span>离线时</span>
              <el-select v-model="draft.offlineBehavior">
                <el-option label="继续保存并标记异常" value="store_stale" />
                <el-option label="停止保存" value="skip" />
              </el-select>
            </label>

            <div class="history-config__additional-head">
              <span>附加目标</span>
              <button type="button" @click="addTarget"><IconTablerPlus /> 添加</button>
            </div>
            <div
              v-for="(target, index) in draft.targets.slice(1)"
              :key="index + 1"
              class="history-config__target-row is-additional"
            >
              <el-select v-model="target.connectionId" placeholder="请选择存储目标">
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
                  title="上移"
                  @click="moveTarget(index + 1, -1)"
                >
                  <IconTablerArrowUp />
                </button>
                <button
                  type="button"
                  :disabled="index === draft.targets.length - 2"
                  title="下移"
                  @click="moveTarget(index + 1, 1)"
                >
                  <IconTablerArrowDown />
                </button>
                <button type="button" title="删除" @click="removeTarget(index + 1)">
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
        <button type="button" class="history-config__cancel" @click="visible = false">取消</button>
        <button type="button" class="history-config__save" :disabled="saving" @click="submit">
          <IconTablerDeviceFloppy />
          {{ saving ? '保存中' : '保存' }}
        </button>
      </footer>
    </div>
  </DcDrawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
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
  set: (value: boolean) => emit('update:modelValue', value),
})
const draft = ref<HistoryStorageDraft>(
  createHistoryStorageDraft(props.behavior, props.configuration, props.targets),
)
const advancedVisible = ref(false)
const errorMessage = ref('')

watch(
  () => [props.modelValue, props.behavior, props.configuration, props.targets] as const,
  () => {
    if (!props.modelValue) return
    draft.value = createHistoryStorageDraft(props.behavior, props.configuration, props.targets)
    advancedVisible.value = false
    errorMessage.value = ''
  },
  { deep: true },
)

const modeOptions: Array<{
  value: HistoryStorageWriteMode
  label: string
  hint: string
  icon: unknown
}> = [
  {
    value: 'on_change',
    label: '变化时保存',
    hint: '值或质量变化时记录',
    icon: IconTablerRefreshDot,
  },
  {
    value: 'interval_latest',
    label: '按间隔保存',
    hint: '每个时间窗口记录最后值',
    icon: IconTablerClock,
  },
  {
    value: 'periodic_snapshot',
    label: '按周期保存',
    hint: '定时记录当前最新值',
    icon: IconTablerRepeat,
  },
  {
    value: 'every_sample',
    label: '保存每次采样',
    hint: '记录收到的每一个样本',
    icon: IconTablerListDetails,
  },
]

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
  if (draft.value.writeMode === 'every_sample') return '保存每次采样可能产生较大的历史数据量。'
  if (draft.value.targets.some((target) => target.retentionDays === null))
    return '永久保留会持续占用存储空间。'
  if (selectedTargetOptions.value.some((target) => target.status !== 'connected'))
    return '所选目标当前未连接，保存后请在接入源中检查连接。'
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
  const type = target.type === 'builtin.timeseries' ? 'IF时序库' : 'TDengine'
  const status = target.status === 'connected' ? '' : ' · 未连接'
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
    errorMessage.value = validation
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
