<template>
  <el-drawer :model-value="modelValue" title="建模校验" size="360px" @close="$emit('update:modelValue', false)">
    <div class="modbus-validation-drawer">
      <p>结果：{{ issues.length }} 个问题</p>
      <article v-for="issue in issues" :key="`${issue.code}-${issue.registerId}`">
        <el-tag size="small" :type="issue.severity === 'error' ? 'danger' : 'warning'">{{ issue.severity }}</el-tag>
        <strong>{{ issue.registerName || '寄存器组' }}</strong>
        <span>{{ issue.message }}</span>
        <el-button v-if="issue.registerId" size="small" link @click="$emit('locate', issue.registerId)">定位变量</el-button>
      </article>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import type { ModbusValidationIssue } from './types'

defineProps<{ modelValue: boolean; issues: ModbusValidationIssue[] }>()
defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'locate', registerId: string): void
}>()
</script>

<style scoped>
.modbus-validation-drawer {
  display: grid;
  gap: 10px;
}

.modbus-validation-drawer p,
.modbus-validation-drawer span {
  margin: 0;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.modbus-validation-drawer article {
  display: grid;
  gap: 6px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
}
</style>
