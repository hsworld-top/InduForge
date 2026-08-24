<template>
  <DcDialog
    :model-value="visible"
    :title="`报警项：${datapointName || '-'}`"
    width="min(760px, calc(100vw - 32px))"
    body-max-height="520px"
    @update:model-value="handleVisibleChange"
  >
    <div v-loading="loading" class="alarm-point-summary">
      <el-table v-if="items.length" :data="items" size="small" row-key="id">
        <el-table-column label="报警名称" min-width="150">
          <template #default="{ row }">
            <div class="alarm-point-summary__name">
              <strong>{{ row.displayName }}</strong>
              <span>{{
                row.mode === 'derived' ? '组合报警' : alarmConditionLabels[row.alarmType]
              }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="条件" width="112">
          <template #default="{ row }">{{ alarmItemConditionSummary(row) }}</template>
        </el-table-column>
        <el-table-column label="最高等级" width="86">
          <template #default="{ row }">
            <StatusBadge
              :tone="severityTone(alarmItemHighestSeverity(row))"
              :text="alarmSeverityLabels[alarmItemHighestSeverity(row)]"
            />
          </template>
        </el-table-column>
        <el-table-column label="状态" width="72">
          <template #default="{ row }">
            <StatusBadge
              :tone="row.isEnabled ? 'success' : 'muted'"
              :text="row.isEnabled ? '已启用' : '已停用'"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="76" align="right">
          <template #default="{ row }">
            <button
              type="button"
              class="alarm-point-summary__open"
              :aria-label="`打开报警 ${row.displayName}`"
              @click="openPolicy(row.id)"
            >
              打开
              <IconTablerArrowRight />
            </button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-else-if="!loading" :image-size="56" description="暂无关联报警" />
    </div>

    <template #footer>
      <el-button @click="emit('cancel')">关闭</el-button>
      <el-button type="primary" @click="openAlarmWorkspace">前往报警单元</el-button>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import IconTablerArrowRight from '~icons/tabler/arrow-right'
import type { AlarmItem, AlarmSeverity } from '@/api/schemas/alarm.schema'
import DcDialog from '@/components/shared/DcDialog.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import {
  alarmConditionLabels,
  alarmItemConditionSummary,
  alarmItemHighestSeverity,
  alarmSeverityLabels,
} from '@/models/alarm-policy'

withDefaults(
  defineProps<{
    visible: boolean
    datapointName?: string
    items?: AlarmItem[]
    loading?: boolean
  }>(),
  { datapointName: '', items: () => [], loading: false },
)

const emit = defineEmits<{
  cancel: []
  navigate: [policyId?: string]
}>()

function severityTone(value: AlarmSeverity): 'muted' | 'warning' | 'danger' {
  return value === 'critical'
    ? 'danger'
    : value === 'major' || value === 'warning'
      ? 'warning'
      : 'muted'
}

function openPolicy(policyId: string) {
  emit('navigate', policyId)
}

function openAlarmWorkspace() {
  emit('navigate')
}

function handleVisibleChange(value: boolean) {
  if (!value) emit('cancel')
}
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
