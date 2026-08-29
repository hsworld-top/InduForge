<template>
  <DcDrawer v-model="visible" :title="t('alarmBatch.title')" :width="560" :max="640">
    <div class="alarm-batch">
      <div class="alarm-batch__notice">
        {{ t('alarmBatch.notice', { count: affectedCount }) }}
      </div>
      <label class="alarm-batch__row"
        ><el-checkbox v-model="fields.isEnabled">{{ t('alarmBatch.enabled') }}</el-checkbox
        ><el-switch v-model="patch.isEnabled" :disabled="!fields.isEnabled"
      /></label>
      <label class="alarm-batch__row"
        ><el-checkbox v-model="fields.groupId">{{ t('alarmBatch.moveDirectory') }}</el-checkbox
        ><AlarmGroupSelect
          v-model="patch.groupId"
          :project-id="projectId"
          :disabled="!fields.groupId"
        />
        ></label
      >
      <label class="alarm-batch__row"
        ><el-checkbox v-model="fields.notification">{{ t('alarmBatch.notification') }}</el-checkbox
        ><el-select v-model="patch.notification.mode" :disabled="!fields.notification"
          ><el-option :label="t('alarmBatch.inherit')" value="inherit" /><el-option
            :label="t('alarmBatch.off')"
            value="off" /><el-option :label="t('alarmBatch.custom')" value="custom" /></el-select
      ></label>
      <section v-if="conditionsCompatible" class="alarm-batch__conditions">
        <el-checkbox v-model="fields.conditions">{{ t('alarmBatch.replaceConditions') }}</el-checkbox>
        <p>{{ t('alarmBatch.replaceHint') }}</p>
        <div v-if="fields.conditions" class="alarm-batch__condition-list">
          <article v-for="condition in patch.conditions" :key="condition.id">
            <el-input v-model="condition.label" :placeholder="t('alarmBatch.levelLabel')" />
            <el-select v-model="condition.severity"
              ><el-option
                v-for="item in severityDefinitions"
                :key="item.key"
                :label="item.displayName"
                :value="item.key"
            /></el-select>
            <el-input-number
              v-if="condition.kind === 'threshold'"
              :model-value="Number(condition.params.threshold)"
              @update:model-value="condition.params = { ...condition.params, threshold: $event }"
            />
            <div
              v-else-if="condition.kind === 'rate_of_change'"
              class="alarm-batch__condition-pair"
            >
              <el-select
                :model-value="String(condition.params.direction || 'absolute')"
                @update:model-value="condition.params = { ...condition.params, direction: $event }"
                ><el-option :label="t('alarmBatch.rise')" value="rise" /><el-option
                  :label="t('alarmBatch.fall')"
                  value="fall" /><el-option :label="t('alarmBatch.absolute')" value="absolute"
              /></el-select>
              <el-input-number
                :model-value="Number(condition.params.limit)"
                :min="0"
                @update:model-value="condition.params = { ...condition.params, limit: $event }"
              />
            </div>
            <el-select
              v-else-if="condition.kind === 'quality'"
              :model-value="condition.params.qualities"
              multiple
              @update:model-value="condition.params = { ...condition.params, qualities: $event }"
              ><el-option :label="t('alarmBatch.bad')" value="bad" /><el-option :label="t('alarmBatch.unknown')" value="unknown"
            /></el-select>
            <el-input-number
              v-else-if="condition.kind === 'stale'"
              :model-value="Number(condition.params.maxAgeMs)"
              :min="1"
              @update:model-value="condition.params = { ...condition.params, maxAgeMs: $event }"
            />
          </article>
        </div>
      </section>
      <div v-else class="alarm-batch__muted">{{ t('alarmBatch.incompatible') }}</div>
      <footer>
        <button type="button" class="dc-button" @click="visible = false">{{ t('alarmBatch.cancel') }}</button
        ><button
          type="button"
          class="dc-button dc-button--primary"
          :disabled="saving"
          @click="submit"
        >
          {{ t('alarmBatch.confirm') }}
        </button>
      </footer>
    </div>
  </DcDrawer>
</template>
<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { ElMessage } from 'element-plus'
import DcDrawer from '@/components/shared/DcDrawer.vue'
import AlarmGroupSelect from './AlarmGroupSelect.vue'
import type { AlarmBatchUpdate, AlarmItem, AlarmItemSelection } from '@/api/schemas/alarm.schema'
import { useAlarmLevelDefinitions } from '@/composables/useAlarmLevelDefinitions'
import { t } from '@/i18n/runtime'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    projectId: string
    selection: AlarmItemSelection
    affectedCount: number
    items?: AlarmItem[]
    saving?: boolean
  }>(),
  { items: () => [], saving: false },
)
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [payload: AlarmBatchUpdate]
}>()
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const fields = reactive({
  isEnabled: false,
  groupId: false,
  notification: false,
  conditions: false,
})
const patch = reactive({
  isEnabled: true,
  groupId: null as string | null,
  notification: { mode: 'inherit' as const, channelIds: [] as string[], messageTemplate: '' },
  conditions: [] as AlarmItem['conditions'],
})
const { definitions: severityDefinitions, loadDefinitions } = useAlarmLevelDefinitions(
  () => props.projectId,
)
const conditionsCompatible = computed(
  () =>
    props.items.length > 0 &&
    props.items.every(
      (item) =>
        item.mode === props.items[0]?.mode &&
        item.alarmType === props.items[0]?.alarmType &&
        item.evaluationMode === props.items[0]?.evaluationMode,
    ),
)
watch(
  () => props.modelValue,
  (opened) => {
    if (!opened) return
    void loadDefinitions()
    Object.assign(fields, {
      isEnabled: false,
      groupId: false,
      notification: false,
      conditions: false,
    })
    patch.conditions =
      props.items[0]?.conditions.map((condition) => ({
        ...condition,
        params: { ...condition.params },
      })) || []
  },
)
function submit() {
  const selectedFields = (Object.keys(fields) as Array<keyof typeof fields>).filter(
    (key) => fields[key],
  )
  if (!selectedFields.length) return ElMessage.warning(t('alarmBatch.selectField'))
  emit('save', {
    selection: props.selection,
    fields: selectedFields,
    patch: {
      isEnabled: patch.isEnabled,
      groupId: patch.groupId,
      notification: patch.notification,
      conditions: patch.conditions,
    },
    acknowledgedWarningKeys: [],
  })
}
</script>
<style scoped>
.alarm-batch {
  display: grid;
  gap: 14px;
}
.alarm-batch__notice {
  padding: 10px 12px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-text-secondary);
}
.alarm-batch__row {
  display: grid;
  grid-template-columns: 150px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
}
.alarm-batch__conditions {
  display: grid;
  gap: 8px;
  padding-top: 14px;
  border-top: 1px solid var(--dc-border);
}
.alarm-batch__conditions p,
.alarm-batch__muted {
  margin: 0;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.alarm-batch__condition-list {
  display: grid;
  gap: 8px;
}
.alarm-batch__condition-list article {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 110px 130px;
  gap: 8px;
}
.alarm-batch__condition-pair {
  display: grid;
  grid-template-columns: 90px minmax(0, 1fr);
  gap: 6px;
}
.alarm-batch footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 14px;
  border-top: 1px solid var(--dc-border);
}
</style>
