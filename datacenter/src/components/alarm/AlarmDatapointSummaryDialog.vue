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
            <span
              class="alarm-point-summary__severity"
              :style="severityBadgeStyle(highestSeverity(row))"
              >{{ severityLabel(highestSeverity(row)) }}</span
            >
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
              @click="openItem(row.id)"
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
import { watch } from 'vue'
import type { AlarmItem, AlarmSeverity } from '@/api/schemas/alarm.schema'
import DcDialog from '@/components/shared/DcDialog.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import {
  alarmConditionLabels,
  alarmItemConditionSummary,
  alarmItemHighestSeverity,
  alarmSeverityLabel,
} from '@/models/alarm-item'
import { useAlarmLevelDefinitions } from '@/composables/useAlarmLevelDefinitions'

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
function severityLabel(value: AlarmSeverity) {
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
