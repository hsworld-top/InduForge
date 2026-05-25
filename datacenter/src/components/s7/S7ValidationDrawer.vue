<template>
  <el-drawer :model-value="modelValue" title="S7 建模校验" size="520px" @close="$emit('update:modelValue', false)">
    <div class="s7-validation-drawer__scope">{{ scopeLabel }}</div>
    <el-table :data="issues" height="100%" empty-text="暂无校验问题">
      <el-table-column label="级别" width="76"><template #default="{ row }"><el-tag size="small" :type="row.severity === 'error' ? 'danger' : 'warning'">{{ row.severity }}</el-tag></template></el-table-column>
      <el-table-column prop="message" label="问题" min-width="220" />
      <el-table-column label="定位" width="76"><template #default="{ row }"><el-button v-if="row.variableId" text type="primary" @click="$emit('locate', row.variableId)">定位</el-button></template></el-table-column>
    </el-table>
  </el-drawer>
</template>

<script setup lang="ts">
import type { S7ValidationIssue } from './types'
defineProps<{ modelValue: boolean; issues: S7ValidationIssue[]; scopeLabel: string }>()
defineEmits<{ (event: 'update:modelValue', value: boolean): void; (event: 'locate', variableId: string): void }>()
</script>

<style scoped>
.s7-validation-drawer__scope {
  margin-bottom: 10px;
  color: var(--dc-text-muted);
  font-size: 12px;
}
</style>
