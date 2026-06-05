<template>
  <el-dialog
    :model-value="modelValue"
    title="运行态读取预估"
    width="760px"
    @close="$emit('update:modelValue', false)"
  >
    <div class="modbus-read-plan-dialog__summary">
      <span
        ><strong>{{ estimate.registerCount }}</strong
        >变量</span
      >
      <span
        ><strong>{{ estimate.readCount }}</strong
        >次读取</span
      >
      <span
        ><strong>{{ estimate.readsPerSecond.toFixed(2) }}</strong
        >reads/s</span
      >
    </div>
    <el-table :data="estimate.plans" height="300px">
      <el-table-column prop="unitId" label="从站地址" width="88" />
      <el-table-column prop="displayArea" label="寄存器区" width="150" />
      <el-table-column prop="displayRange" label="读取范围" />
      <el-table-column prop="displayCycle" label="周期" width="90" />
      <el-table-column prop="registerCount" label="本次读取变量数" width="132" />
    </el-table>
    <div class="modbus-read-plan-dialog__tips">
      <p v-for="item in estimate.diagnostics" :key="item">{{ item }}</p>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import type { ModbusReadPlanEstimate } from './types'

defineProps<{ modelValue: boolean; estimate: ModbusReadPlanEstimate }>()
defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()
</script>

<style scoped>
.modbus-read-plan-dialog__summary,
.modbus-read-plan-dialog__tips {
  margin-bottom: 10px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.modbus-read-plan-dialog__summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.modbus-read-plan-dialog__summary span {
  min-height: 46px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  font-weight: 700;
}

.modbus-read-plan-dialog__summary strong {
  color: var(--dc-primary);
  font-size: 18px;
}

.modbus-read-plan-dialog__tips {
  margin: 10px 0 0;
}

.modbus-read-plan-dialog__tips p {
  margin: 0 0 4px;
}
</style>
