<template>
  <el-dialog :model-value="modelValue" title="变量预览" width="760px" @close="$emit('update:modelValue', false)">
    <el-table :data="registers" height="280px">
      <el-table-column prop="name" label="变量名" />
      <el-table-column prop="unitId" label="从站" width="70" />
      <el-table-column prop="area" label="区域" width="130" />
      <el-table-column prop="address" label="地址" width="90" />
      <el-table-column prop="lastValue" label="最近值" width="90" />
      <el-table-column prop="quality" label="质量" width="90" />
    </el-table>
    <div class="modbus-preview-dialog__logs">
      <p v-for="item in diagnostics" :key="item">{{ item }}</p>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import type { ModbusRegister } from './types'

defineProps<{
  modelValue: boolean
  registers: ModbusRegister[]
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
