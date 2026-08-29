<template>
  <DcDialog
    :model-value="visible"
    :title="t('alarmSummary.title', { name: datapointName || '-' })"
    width="min(760px, calc(100vw - 32px))"
    body-max-height="520px"
    @update:model-value="handleVisibleChange"
  >
    <div v-loading="loading" class="alarm-point-summary">
      <el-table v-if="items.length" :data="items" size="small" row-key="id">
        <el-table-column :label="t('alarmSummary.alarmName')" min-width="150">
          <template #default="{ row }">
            <div class="alarm-point-summary__name">
              <strong>{{ row.displayName }}</strong>
              <span>{{
                row.mode === 'derived' ? t('alarmSummary.composite') : conditionTypeLabel(row.alarmType)
              }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="t('alarmSummary.condition')" width="112">
          <template #default="{ row }">{{ conditionSummary(row) }}</template>
        </el-table-column>
        <el-table-column :label="t('alarmSummary.highestLevel')" width="104">
          <template #default="{ row }">
            <span
              class="alarm-point-summary__severity"
              :style="severityBadgeStyle(highestSeverity(row))"
              >{{ severityLabel(highestSeverity(row)) }}</span
            >
          </template>
        </el-table-column>
        <el-table-column :label="t('alarmSummary.status')" width="82">
          <template #default="{ row }">
            <StatusBadge
              :tone="row.isEnabled ? 'success' : 'muted'"
              :text="row.isEnabled ? t('alarmSummary.enabled') : t('alarmSummary.disabled')"
            />
          </template>
        </el-table-column>
        <el-table-column :label="t('alarmSummary.actions')" width="82" align="right">
          <template #default="{ row }">
            <button
              type="button"
              class="alarm-point-summary__open"
              :aria-label="t('alarmSummary.openAria', { name: row.displayName })"
              @click="openItem(row.id)"
            >
              {{ t('alarmSummary.open') }}
              <IconTablerArrowRight />
            </button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-else-if="!loading" :image-size="56" :description="t('alarmSummary.empty')" />
    </div>

    <template #footer>
      <el-button @click="emit('cancel')">{{ t('alarmSummary.close') }}</el-button>
      <el-button type="primary" @click="openAlarmWorkspace">{{ t('alarmSummary.gotoWorkspace') }}</el-button>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import IconTablerArrowRight from '~icons/tabler/arrow-right'
import { watch } from 'vue'
import type { AlarmItem, AlarmSeverity } from '@/api/schemas/alarm.schema'
import DcDialog from '@/components/shared/DcDialog.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import {
  alarmItemHighestSeverity,
  alarmSeverityLabel,
} from '@/models/alarm-item'
import { useAlarmLevelDefinitions } from '@/composables/useAlarmLevelDefinitions'
import { datacenterLocale, t } from '@/i18n/runtime'

const props = withDefaults(
  defineProps<{
    visible: boolean
    projectId: string
    datapointName?: string
    items?: AlarmItem[]
    loading?: boolean
  }>(),
  { datapointName: '', items: () => [], loading: false },
)
const { definitions: severityDefinitions, loadDefinitions } = useAlarmLevelDefinitions(
  () => props.projectId,
)

const emit = defineEmits<{
  cancel: []
  navigate: [itemId?: string]
}>()

function highestSeverity(item: AlarmItem) {
  return alarmItemHighestSeverity(item, severityDefinitions.value)
}
function conditionTypeLabel(kind: string) {
  return t(`alarm.conditionLabels.${kind}`)
}
function localizedConditionLabel(label: string, kind: string) {
  if (datacenterLocale.value !== 'en') return label || conditionTypeLabel(kind)
  const builtinLabels: Record<string, string> = {
    '高': t('quickAlarm.levels.h'), '高限': t('quickAlarm.levels.h'), '高限报警': t('quickAlarm.levels.h'),
    '高高': t('quickAlarm.levels.hh'), '高高限': t('quickAlarm.levels.hh'), '高高限报警': t('quickAlarm.levels.hh'),
    '低': t('quickAlarm.levels.l'), '低限': t('quickAlarm.levels.l'), '低限报警': t('quickAlarm.levels.l'),
    '低低': t('quickAlarm.levels.ll'), '低低限': t('quickAlarm.levels.ll'), '低低限报警': t('quickAlarm.levels.ll'),
    '变化率': t('quickAlarm.levels.rate'), '变化率报警': t('quickAlarm.levels.rate'),
    '偏差': t('quickAlarm.levels.deviation'), '偏差报警': t('quickAlarm.levels.deviation'),
    '结果条件': conditionTypeLabel(kind),
  }
  return builtinLabels[label] || label || conditionTypeLabel(kind)
}
function conditionSummary(item: AlarmItem) {
  return item.conditions
    .map((condition) => localizedConditionLabel(condition.label, condition.kind))
    .join(' / ')
}
function severityLabel(value: AlarmSeverity) {
  if (datacenterLocale.value === 'en' && ['info', 'warning', 'major', 'critical'].includes(value)) {
    return t(`alarm.severities.${value}`)
  }
  return alarmSeverityLabel(value, severityDefinitions.value)
}
function severityBadgeStyle(value: AlarmSeverity) {
  const color = severityDefinitions.value.find((item) => item.key === value)?.color ?? '#64748b'
  return {
    color,
    borderColor: `${color}55`,
    backgroundColor: `${color}14`,
  }
}

function openItem(itemId: string) {
  emit('navigate', itemId)
}

function openAlarmWorkspace() {
  emit('navigate')
}

function handleVisibleChange(value: boolean) {
  if (!value) emit('cancel')
}
watch(
  () => props.visible,
  (visible) => {
    if (visible) void loadDefinitions()
  },
)
</script>

<style scoped>
.alarm-point-summary {
  min-height: 128px;
}

.alarm-point-summary__name {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.alarm-point-summary__name strong,
.alarm-point-summary__name span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alarm-point-summary__name strong {
  color: var(--dc-text);
  font-size: 13px;
}

.alarm-point-summary__name span {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.alarm-point-summary__severity {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 1px 7px;
  border: 1px solid;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

.alarm-point-summary__open {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  border: 0;
  background: transparent;
  color: var(--dc-primary);
  cursor: pointer;
  font-size: 12px;
}

.alarm-point-summary__open svg {
  width: 14px;
  height: 14px;
}
</style>
