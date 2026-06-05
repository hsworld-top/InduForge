<template>
  <el-dialog
    :model-value="modelValue"
    title="变量预览"
    width="860px"
    @close="$emit('update:modelValue', false)"
  >
    <el-table :data="registers" height="320px">
      <el-table-column prop="registerId" label="变量 ID" min-width="130" show-overflow-tooltip />
      <el-table-column prop="slaveId" label="从站" width="70" />
      <el-table-column prop="area" label="区域" width="130" />
      <el-table-column prop="address" label="地址" width="90" />
      <el-table-column label="原始值" min-width="110">
        <template #default="{ row }">{{
          Array.isArray(row.rawValue) ? row.rawValue.join(', ') : '-'
        }}</template>
      </el-table-column>
      <el-table-column label="解析值" min-width="110">
        <template #default="{ row }">{{ row.value ?? '-' }}</template>
      </el-table-column>
      <el-table-column prop="dataType" label="类型" width="100" />
      <el-table-column prop="timestamp" label="时间" min-width="150" />
      <el-table-column label="错误" min-width="120">
        <template #default="{ row }">{{ row.error || '-' }}</template>
      </el-table-column>
    </el-table>
    <div class="modbus-preview-dialog__logs">
      <p v-for="item in diagnostics" :key="item">{{ item }}</p>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import type { ModbusReadValue } from './types'

defineProps<{
  modelValue: boolean
  registers: ModbusReadValue[]
  diagnostics: string[]
}>()
defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()
</script>

<style scoped>
.modbus-preview-dialog__logs {
  margin-top: 10px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
}

.modbus-preview-dialog__logs p {
  margin: 0 0 4px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
</style>
