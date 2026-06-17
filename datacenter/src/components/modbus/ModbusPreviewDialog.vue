<template>
  <DcDialog
    v-model="visible"
    title="变量预览"
    width="920px"
    body-max-height="calc(100vh - 180px)"
    class="modbus-preview-dialog"
  >
    <div class="modbus-preview">
      <div class="modbus-preview__summary">
        <span>{{ scopeLabel || '全部变量' }}</span>
        <strong>{{ registers.length }} 个变量</strong>
      </div>

      <div v-if="diagnostics.length" class="modbus-preview__diagnostics">
        <span v-for="item in diagnostics" :key="item">{{ item }}</span>
      </div>

      <el-table
        class="modbus-preview__table"
        :data="pagedRegisters"
        height="420"
        size="small"
        empty-text="当前范围暂无预览值"
      >
        <el-table-column prop="slaveId" label="从站" width="76" />
        <el-table-column label="区域" min-width="130" show-overflow-tooltip>
          <template #default="{ row }">{{ formatArea(row.area) }}</template>
        </el-table-column>
        <el-table-column prop="address" label="变量地址" width="110" />
        <el-table-column label="当前值" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">{{ formatValue(row.value) }}</template>
        </el-table-column>
        <el-table-column label="原始值" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">{{ formatRawValue(row.rawValue) }}</template>
        </el-table-column>
        <el-table-column prop="dataType" label="类型" width="110" />
        <el-table-column label="质量" width="92">
          <template #default="{ row }">
            <el-tag size="small" :type="qualityTagType(row.quality)">
              {{ formatQuality(row.quality) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">{{ formatTime(row.timestamp) }}</template>
        </el-table-column>
        <el-table-column label="诊断" min-width="130" show-overflow-tooltip>
          <template #default="{ row }">{{ row.error || '-' }}</template>
        </el-table-column>
      </el-table>

      <div class="modbus-preview__pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="registers.length"
          background
          layout="total, sizes, prev, pager, next"
          size="small"
        />
      </div>
    </div>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import dayjs from 'dayjs'
import { TIME_FORMAT } from '@/constants'
import DcDialog from '@/components/shared/DcDialog.vue'
import type { ModbusReadValue } from './types'

const props = defineProps<{
  modelValue: boolean
  registers: ModbusReadValue[]
  diagnostics: string[]
  scopeLabel?: string
}>()

const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()

const page = ref(1)
const pageSize = ref(10)

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const pagedRegisters = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return props.registers.slice(start, start + pageSize.value)
})

watch(
  () => [props.modelValue, props.registers.length],
  () => {
    page.value = 1
  },
)

const formatArea = (area?: string) => {
  const map: Record<string, string> = {
    coil: 'Coil',
    discrete_input: 'Discrete Input',
    input_register: 'Input Register',
    holding_register: 'Holding Register',
  }
  return map[String(area || '')] || area || '-'
}

const formatValue = (value: unknown) => {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

const formatRawValue = (value: unknown) => {
  if (!Array.isArray(value)) return formatValue(value)
  return value.length > 0 ? value.join(', ') : '-'
}

const formatQuality = (quality?: string) => {
  const normalized = String(quality || '').toLowerCase()
  if (normalized === 'good') return 'Good'
  if (normalized === 'bad') return 'Bad'
  return 'Unknown'
}

const qualityTagType = (quality?: string) => {
  const normalized = String(quality || '').toLowerCase()
  if (normalized === 'good') return 'success'
  if (normalized === 'bad') return 'danger'
  return 'info'
}

const formatTime = (value?: string) => {
  if (!value) return '-'
  const parsed = dayjs(value)
  return parsed.isValid() ? parsed.format(TIME_FORMAT) : value
}
</script>

<style scoped>
.modbus-preview {
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.modbus-preview__summary {
  min-height: 34px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
}

.modbus-preview__summary span,
.modbus-preview__summary strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modbus-preview__summary span {
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.modbus-preview__summary strong {
  color: var(--dc-text);
  font-size: 13px;
}

.modbus-preview__diagnostics {
  display: grid;
  gap: 4px;
  padding: 8px 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 12px;
}

.modbus-preview__table {
  width: 100%;
}

.modbus-preview__table :deep(.el-table__header th) {
  background: var(--dc-surface-subtle);
  color: var(--dc-text-secondary);
  font-weight: 700;
}

.modbus-preview__pagination {
  display: flex;
  justify-content: flex-end;
  padding-top: 2px;
}

:global(.modbus-preview-dialog .el-dialog__body) {
  overflow: hidden;
}
</style>
