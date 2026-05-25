<template>
  <DcDialog v-model="visible" title="S7 读取计划估算" width="860px">
    <div class="s7-read-plan-dialog__metrics">
      <span><b>{{ estimate.variableCount }}</b>变量</span>
      <span><b>{{ estimate.blockCount }}</b>读取块</span>
      <span><b>{{ estimate.totalReadBytes }}</b>bytes</span>
      <span><b>{{ estimate.readsPerSecond.toFixed(2) }}</b>reads/s</span>
    </div>
    <el-table :data="estimate.plans" height="360" empty-text="暂无读取计划">
      <el-table-column prop="area" label="区域" width="76" />
      <el-table-column label="DB号" width="76"><template #default="{ row }">{{ row.dbNumber ?? '-' }}</template></el-table-column>
      <el-table-column label="读取范围" min-width="160" show-overflow-tooltip><template #default="{ row }">{{ row.displayRange || `${row.startByte}-${row.endByte}` }}</template></el-table-column>
      <el-table-column prop="readLength" label="字节" width="80" />
      <el-table-column prop="variableCount" label="变量数" width="90" />
      <el-table-column label="周期" width="96"><template #default="{ row }">{{ row.pollIntervalMs }}ms</template></el-table-column>
    </el-table>
    <div v-if="estimate.diagnostics.length" class="s7-read-plan-dialog__tips">
      <p v-for="item in estimate.diagnostics" :key="item">{{ item }}</p>
    </div>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import type { S7ReadPlanEstimate } from './types'
const props = defineProps<{ modelValue: boolean; estimate: S7ReadPlanEstimate }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()
const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
</script>

<style scoped>
.s7-read-plan-dialog__metrics {
  display: flex;
  gap: 8px;
  margin-bottom: 10px;
}
.s7-read-plan-dialog__metrics span {
  display: inline-flex;
  gap: 4px;
  align-items: center;
  padding: 5px 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  font-size: 12px;
}
.s7-read-plan-dialog__metrics b {
  color: var(--dc-primary);
}
.s7-read-plan-dialog__tips {
  margin-top: 10px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
  color: var(--dc-text-muted);
  font-size: 12px;
}
.s7-read-plan-dialog__tips p {
  margin: 4px 0 0;
}
</style>
