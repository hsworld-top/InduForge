<template>
  <el-dialog
    :model-value="modelValue"
    title="变量预览"
    width="820px"
    @close="$emit('update:modelValue', false)"
  >
    <div class="opcua-preview__summary">
      <strong>{{ nodes.length }} 个变量</strong>
      <span>{{ diagnostics.join('；') || '真实采集由运行态执行，当前用于核对建模变量。' }}</span>
    </div>
    <el-table class="opcua-preview__table" :data="nodes" height="360" size="small">
      <el-table-column prop="nodeId" label="NodeId" min-width="240" show-overflow-tooltip />
      <el-table-column label="当前值" min-width="120">
        <template #default="{ row }">{{ row.value ?? '-' }}</template>
      </el-table-column>
      <el-table-column prop="dataType" label="类型" width="110" />
      <el-table-column label="质量" width="100">
        <template #default="{ row }">
          <el-tag size="small" :type="row.quality === 'Good' ? 'success' : 'info'">
            {{ row.quality || 'unknown' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="sourceTimestamp" label="SourceTime" min-width="150" />
      <el-table-column prop="serverTimestamp" label="ServerTime" min-width="150" />
      <el-table-column label="错误" min-width="120">
        <template #default="{ row }">{{ row.error || '-' }}</template>
      </el-table-column>
    </el-table>
  </el-dialog>
</template>

<script setup lang="ts">
import type { OpcuaReadValue } from './types'

defineProps<{
  modelValue: boolean
  nodes: OpcuaReadValue[]
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
