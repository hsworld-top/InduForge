<template>
  <DatapointPickerDialog
    v-model="visible"
    :project-id="projectId"
    :title="title"
    :initial-selection="initialSelection"
    :validate-selection="compatible"
    @apply="apply"
  >
    <template #warning="{ points }">
      <span v-if="mode === 'point' && !compatible(points)" class="alarm-point-picker__warning">
        所选数据点类型不兼容
      </span>
    </template>
  </DatapointPickerDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import DatapointPickerDialog, {
  type DatapointPickerSelection,
} from '@/components/shared/DatapointPickerDialog.vue'
import type { AlarmItemMode } from '@/api/schemas/alarm.schema'
import { compatibleAlarmPoints, type AlarmPointSelection } from '@/models/alarm-item'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    projectId: string
    mode?: AlarmItemMode
    points?: AlarmPointSelection[]
  }>(),
  { mode: 'point', points: () => [] },
)
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  apply: [points: AlarmPointSelection[]]
}>()
const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
const title = computed(() => (props.mode === 'derived' ? '管理组合输入点' : '选择报警数据点'))
const initialSelection = computed<DatapointPickerSelection[]>(() =>
  props.points.map((point) => ({
    id: point.datapointId,
    path: point.path,
    name: point.name,
    dataType: point.dataType,
  })),
)
const toAlarmPoints = (points: DatapointPickerSelection[]): AlarmPointSelection[] =>
  points.map((point, index) => ({
    datapointId: String(point.id),
    path: point.path,
    name: point.name || point.path,
    dataType: point.dataType || 'string',
    inputKey: props.mode === 'derived' ? `input${index + 1}` : undefined,
  }))
const compatible = (points: DatapointPickerSelection[]) =>
  props.mode === 'derived' || compatibleAlarmPoints(toAlarmPoints(points))
function apply(points: DatapointPickerSelection[]) {
  const normalized = toAlarmPoints(points)
  if (props.mode === 'point' && !compatibleAlarmPoints(normalized)) return
  emit('apply', normalized)
}
</script>

<style scoped>
.alarm-point-picker__warning {
  color: var(--el-color-warning);
}
</style>
