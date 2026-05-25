<template>
  <el-dialog :model-value="modelValue" title="变量预览" width="820px" @close="$emit('update:modelValue', false)">
    <div class="opcua-preview__summary">
      <strong>{{ nodes.length }} 个变量</strong>
      <span>{{ diagnostics.join('；') || '真实采集由运行态执行，当前用于核对建模变量。' }}</span>
    </div>
    <el-table class="opcua-preview__table" :data="nodes" height="360" size="small">
      <el-table-column prop="name" label="变量名" min-width="150" />
      <el-table-column prop="nodeId" label="NodeId" min-width="240" show-overflow-tooltip />
      <el-table-column label="最近值" min-width="120">
        <template #default="{ row }">{{ row.lastValue ?? '-' }}</template>
      </el-table-column>
      <el-table-column label="质量" width="100">
        <template #default="{ row }">
          <el-tag size="small" type="info">{{ row.quality || 'unknown' }}</el-tag>
        </template>
      </el-table-column>
    </el-table>
  </el-dialog>
</template>

<script setup lang="ts">
import type { OpcuaNode } from './types'

defineProps<{
  modelValue: boolean
  nodes: OpcuaNode[]
  diagnostics: string[]
}>()

defineEmits<{
  (event: 'update:modelValue', value: boolean): void
}>()
</script>

<style scoped>
.opcua-preview__summary {
  display: grid;
  gap: 4px;
  margin-bottom: 10px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
}

.opcua-preview__summary strong {
  color: var(--dc-text);
  font-size: 13px;
}

.opcua-preview__summary span {
  color: var(--dc-text-muted);
  font-size: 13px;
}

.opcua-preview__table :deep(.el-table) {
  --el-table-header-bg-color: var(--dc-surface-subtle);
  --el-table-border-color: var(--dc-border);
}
</style>
