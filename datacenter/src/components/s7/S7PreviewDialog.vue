<template>
  <DcDialog v-model="visible" title="S7 变量预览" width="860px">
    <el-table :data="values" height="360" empty-text="暂无预览值">
      <el-table-column prop="variableId" label="变量" min-width="160" show-overflow-tooltip />
      <el-table-column prop="address" label="地址" min-width="140" show-overflow-tooltip />
      <el-table-column label="当前值" min-width="120" show-overflow-tooltip
        ><template #default="{ row }">{{ formatValue(row.value) }}</template></el-table-column
      >
      <el-table-column prop="quality" label="质量" width="90" />
      <el-table-column prop="timestamp" label="更新时间" min-width="150" />
    </el-table>
    <div v-if="diagnostics.length" class="s7-preview-dialog__tips">
      <p v-for="item in diagnostics" :key="item">{{ item }}</p>
    </div>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import type { S7ReadValue } from './types'
const props = defineProps<{ modelValue: boolean; values: S7ReadValue[]; diagnostics: string[] }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()
const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
const formatValue = (value: unknown) =>
  typeof value === 'object' ? JSON.stringify(value) : String(value ?? '-')
</script>

<style scoped>
.s7-preview-dialog__tips {
  margin-top: 10px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
  color: var(--dc-text-muted);
  font-size: 12px;
}
.s7-preview-dialog__tips p {
  margin: 4px 0 0;
}
</style>
