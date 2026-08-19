<template>
  <div v-loading="loading" class="alarm-history">
    <section class="alarm-history__section">
      <header class="alarm-history__head">
        <div>
          <h3>报警历史</h3>
          <p>配置发布到节点后，决定报警事件和通知投递结果的保存方式。</p>
        </div>
        <button
          type="button"
          class="alarm-history__primary"
          :disabled="loading || saving"
          @click="save"
        >
          <IconTablerDeviceFloppy />
          保存设置
        </button>
      </header>

      <div class="alarm-history__form">
        <div class="alarm-history__row">
          <div class="alarm-history__label">
            <strong>保存报警历史</strong>
            <span>保存触发、等级变化、确认和恢复等状态变化记录</span>
          </div>
          <el-switch v-model="draft.isEnabled" />
        </div>

        <div class="alarm-history__row" :class="{ 'is-disabled': !draft.isEnabled }">
          <div class="alarm-history__label">
            <strong>保留时间</strong>
            <span>节点按此期限定期清理已保存的报警历史</span>
          </div>
          <div class="alarm-history__retention">
            <el-segmented
              v-model="draft.retentionMode"
              :options="retentionOptions"
              size="small"
              :disabled="!draft.isEnabled"
            />
            <div v-if="draft.retentionMode === 'days'" class="alarm-history__days">
              <el-input-number
                v-model="draft.retentionDays"
                :min="1"
                :controls="false"
                :disabled="!draft.isEnabled"
              />
              <span>天</span>
            </div>
          </div>
        </div>

        <div class="alarm-history__row" :class="{ 'is-disabled': !draft.isEnabled }">
          <div class="alarm-history__label">
            <strong>保存通知投递记录</strong>
            <span>记录通知渠道的投递结果，便于后续追溯</span>
          </div>
          <el-switch v-model="draft.storeNotificationDeliveries" :disabled="!draft.isEnabled" />
        </div>
      </div>

      <div class="alarm-history__notice">
        <IconTablerInfoCircle />
        <span>此设置只影响历史记录。当前活动报警始终由节点维护，不受历史开关影响。</span>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import IconTablerDeviceFloppy from '~icons/tabler/device-floppy'
import IconTablerInfoCircle from '~icons/tabler/info-circle'
import { getAlarmHistorySettings, saveAlarmHistorySettings } from '@/api/alarm.api'
import {
  buildAlarmHistoryPayload,
  createAlarmHistoryDraft,
  validateAlarmHistoryDraft,
  type AlarmHistoryRetentionMode,
} from '@/models/alarm-history'
import { getApiErrorMessage } from '@/utils/request'
import { useConfirm } from '@/composables/useConfirm'

const props = defineProps<{ projectId: string }>()
const { confirm } = useConfirm()
const loading = ref(false)
const saving = ref(false)
const draft = ref(createAlarmHistoryDraft())
const retentionOptions: Array<{ label: string; value: AlarmHistoryRetentionMode }> = [
  { label: '按天', value: 'days' },
  { label: '永久', value: 'forever' },
]

onMounted(() => void load())
watch(
  () => props.projectId,
  () => void load(),
)

async function load() {
  loading.value = true
  try {
    draft.value = createAlarmHistoryDraft(await getAlarmHistorySettings(props.projectId))
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载报警历史设置失败'))
  } finally {
    loading.value = false
  }
}

async function save() {
  const validationMessage = validateAlarmHistoryDraft(draft.value)
  if (validationMessage) {
    ElMessage.warning(validationMessage)
    return
  }
  if (
    !draft.value.isEnabled &&
    !(await confirm('关闭后仅停止新增历史，不影响当前报警，已有历史不会立即删除。', {
      title: '关闭报警历史',
      confirmText: '确认保存',
      type: 'warning',
    }))
  ) {
    return
  }
  saving.value = true
  try {
    const saved = await saveAlarmHistorySettings(
      props.projectId,
      buildAlarmHistoryPayload(draft.value),
    )
    draft.value = createAlarmHistoryDraft(saved)
    ElMessage.success('报警历史设置已保存')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存报警历史设置失败'))
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.alarm-history {
  width: 100%;
  height: 100%;
  overflow: auto;
  background: var(--dc-surface-raised);
}
.alarm-history__section {
  width: min(720px, 100%);
  padding: 20px;
}
.alarm-history__head {
  min-height: 42px;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  padding-bottom: 18px;
  border-bottom: 1px solid var(--dc-border);
}
.alarm-history__head h3 {
  margin: 0;
  font-size: 15px;
  letter-spacing: 0;
}
.alarm-history__head p {
  margin: 6px 0 0;
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 1.6;
}
.alarm-history__form {
  display: grid;
}
.alarm-history__row {
  min-height: 76px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 24px;
  padding: 14px 0;
  border-bottom: 1px solid var(--dc-border);
}
.alarm-history__row.is-disabled .alarm-history__label {
  opacity: 0.58;
}
.alarm-history__label {
  min-width: 0;
  display: grid;
  gap: 5px;
}
.alarm-history__label strong {
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 650;
  letter-spacing: 0;
}
.alarm-history__label span {
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 1.5;
}
.alarm-history__retention {
  display: flex;
  align-items: center;
  gap: 10px;
}
.alarm-history__days {
  display: grid;
  grid-template-columns: 82px auto;
  align-items: center;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.alarm-history__days :deep(.el-input-number) {
  width: 82px;
}
.alarm-history__notice {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: 16px;
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 1.6;
}
.alarm-history__notice svg {
  width: 16px;
  flex: 0 0 16px;
  margin-top: 1px;
  color: var(--dc-primary);
}
.alarm-history__primary {
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 0 12px;
  border: 1px solid var(--dc-primary);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary);
  color: white;
  font-family: inherit;
  cursor: pointer;
}
.alarm-history__primary svg {
  width: 15px;
}
.alarm-history__primary:hover:not(:disabled) {
  background: color-mix(in srgb, var(--dc-primary) 88%, black);
}
.alarm-history__primary:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
.alarm-history__primary:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px rgba(29, 78, 216, 0.12);
}
@media (max-width: 620px) {
  .alarm-history__section {
    padding: 16px;
  }
  .alarm-history__head {
    align-items: stretch;
    flex-direction: column;
  }
  .alarm-history__primary {
    align-self: flex-start;
  }
  .alarm-history__row {
    grid-template-columns: minmax(0, 1fr);
    gap: 10px;
  }
  .alarm-history__retention {
    flex-wrap: wrap;
  }
}
</style>
