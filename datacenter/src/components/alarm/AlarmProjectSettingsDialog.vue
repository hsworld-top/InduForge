<template>
  <DcDialog
    v-model="visible"
    title="报警设置"
    width="460px"
    destroy-on-close
    :dirty="isDirty"
    class="alarm-settings-dialog"
    @close="reset"
  >
    <div class="alarm-settings-dialog__body">
      <label class="alarm-settings-dialog__field">
        <span class="alarm-settings-dialog__label">
          报警升级间隔
          <span
            class="alarm-settings-dialog__tip"
            title="例如 300s，用于提示 → 警告 → 重要 → 紧急的阶段升级。"
          >
            ?
          </span>
        </span>
        <span class="alarm-settings-dialog__control">
          <input
            v-model.number="form.escalationIntervalSeconds"
            type="number"
            min="30"
            step="1"
            :disabled="loading || submitting"
          />
          <span>秒</span>
        </span>
      </label>

      <label class="alarm-settings-dialog__field">
        <span class="alarm-settings-dialog__label">
          重复推送间隔
          <span
            class="alarm-settings-dialog__tip"
            title="例如 10s，用于报警未恢复期间按固定间隔提醒。"
          >
            ?
          </span>
        </span>
        <span class="alarm-settings-dialog__control">
          <input
            v-model.number="form.repeatNotificationIntervalSeconds"
            type="number"
            min="10"
            step="1"
            :disabled="loading || submitting"
          />
          <span>秒</span>
        </span>
      </label>

      <div class="alarm-settings-dialog__hint">
        报警升级间隔用于等级阶段升级；重复推送间隔用于报警未恢复期间按固定间隔提醒。升级时优先级跳到对应等级的默认值，不会从
        1 逐级递增，也不会为每一次数值变化推送事件。
      </div>

      <div class="alarm-settings-dialog__steps" aria-label="报警升级等级">
        <span>提示 100</span>
        <span>警告 300</span>
        <span>重要 600</span>
        <span>紧急 900</span>
      </div>

      <p v-if="localError || error" class="alarm-settings-dialog__error">
        {{ localError || error }}
      </p>
    </div>

    <template #footer>
      <div class="alarm-settings-dialog__footer">
        <button type="button" class="is-ghost" @click="visible = false">取消</button>
        <button type="button" :disabled="loading || submitting" @click="submit">
          {{ submitting ? '保存中' : '保存' }}
        </button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import type { AlarmProjectSettings, AlarmProjectSettingsSave } from '@/api/schemas/alarm.schema'
import DcDialog from '@/components/shared/DcDialog.vue'

const props = defineProps<{
  modelValue: boolean
  settings: AlarmProjectSettings | null
  loading?: boolean
  submitting?: boolean
  error?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  submit: [payload: AlarmProjectSettingsSave]
}>()

const form = reactive({
  escalationIntervalSeconds: 300,
  repeatNotificationIntervalSeconds: 60,
})
const localError = ref('')
const initialSnapshot = ref('')

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const formSnapshot = computed(() =>
  JSON.stringify({
    escalationIntervalSeconds: form.escalationIntervalSeconds,
    repeatNotificationIntervalSeconds: form.repeatNotificationIntervalSeconds,
  }),
)
const isDirty = computed(() => visible.value && formSnapshot.value !== initialSnapshot.value)

const syncForm = () => {
  form.escalationIntervalSeconds = props.settings?.escalationIntervalSeconds ?? 300
  form.repeatNotificationIntervalSeconds = props.settings?.repeatNotificationIntervalSeconds ?? 60
  localError.value = ''
  initialSnapshot.value = formSnapshot.value
}

const reset = () => {
  syncForm()
}

const submit = () => {
  localError.value = ''
  if (!Number.isInteger(form.escalationIntervalSeconds)) {
    localError.value = '报警升级间隔必须是整数秒'
    return
  }
  if (form.escalationIntervalSeconds < 30) {
    localError.value = '报警升级间隔不能小于 30 秒'
    return
  }
  if (!Number.isInteger(form.repeatNotificationIntervalSeconds)) {
    localError.value = '重复推送间隔必须是整数秒'
    return
  }
  if (form.repeatNotificationIntervalSeconds < 10) {
    localError.value = '重复推送间隔不能小于 10 秒'
    return
  }
  emit('submit', {
    escalationIntervalSeconds: form.escalationIntervalSeconds,
    repeatNotificationIntervalSeconds: form.repeatNotificationIntervalSeconds,
  })
  initialSnapshot.value = formSnapshot.value
}

watch(
  () =>
    [
      props.modelValue,
      props.settings?.escalationIntervalSeconds,
      props.settings?.repeatNotificationIntervalSeconds,
    ] as const,
  () => {
    if (props.modelValue) {
      syncForm()
    }
  },
  { immediate: true },
)
</script>

<style scoped>
.alarm-settings-dialog__body {
  display: grid;
  gap: 14px;
}

.alarm-settings-dialog__field {
  display: grid;
  gap: 7px;
}

.alarm-settings-dialog__label {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.alarm-settings-dialog__tip {
  width: 15px;
  height: 15px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: 999px;
  color: var(--dc-text-muted);
  cursor: help;
  font-size: 10px;
  line-height: 1;
}

.alarm-settings-dialog__control {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.alarm-settings-dialog__control input {
  width: 100%;
  height: 32px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  font: inherit;
  padding: 0 10px;
}

.alarm-settings-dialog__hint {
  padding: 10px 12px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-size: 12px;
  line-height: 1.6;
}

.alarm-settings-dialog__steps {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 6px;
}

.alarm-settings-dialog__steps span {
  min-width: 0;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.alarm-settings-dialog__error {
  margin: 0;
  color: var(--dc-danger);
  font-size: 12px;
}

.alarm-settings-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.alarm-settings-dialog__footer button {
  height: 30px;
  padding: 0 14px;
  border: 1px solid var(--dc-primary);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary);
  color: #fff;
  cursor: pointer;
  font-size: 13px;
  font-weight: 700;
}

.alarm-settings-dialog__footer button.is-ghost {
  border-color: var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.alarm-settings-dialog__footer button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
</style>
