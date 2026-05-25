<template>
  <DcDrawer v-model="visible" title="S7 建模校验" :width="520">
    <template #actions>
      <el-tooltip content="关闭" placement="bottom">
        <button type="button" class="s7-validation-drawer__close" aria-label="关闭校验抽屉" @click="visible = false">
          <IconTablerX />
        </button>
      </el-tooltip>
    </template>
    <div class="s7-validation-drawer__scope">{{ scopeLabel }}</div>
    <el-table :data="issues" height="100%" empty-text="暂无校验问题">
      <el-table-column label="级别" width="76"><template #default="{ row }"><el-tag size="small" :type="row.severity === 'error' ? 'danger' : 'warning'">{{ row.severity }}</el-tag></template></el-table-column>
      <el-table-column prop="message" label="问题" min-width="220" show-overflow-tooltip />
      <el-table-column label="定位" width="76"><template #default="{ row }"><el-button v-if="row.variableId" text type="primary" @click="$emit('locate', row.variableId)">定位</el-button></template></el-table-column>
    </el-table>
  </DcDrawer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import DcDrawer from '@/components/shared/DcDrawer.vue'
import type { S7ValidationIssue } from './types'
import IconTablerX from '~icons/tabler/x'
const props = defineProps<{ modelValue: boolean; issues: S7ValidationIssue[]; scopeLabel: string }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void; (event: 'locate', variableId: string): void }>()
const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
</script>

<style scoped>
.s7-validation-drawer__scope {
  margin-bottom: 10px;
  color: var(--dc-text-muted);
  font-size: 12px;
}
.s7-validation-drawer__close {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}
.s7-validation-drawer__close:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
  color: var(--dc-primary);
}
.s7-validation-drawer__close svg {
  width: 15px;
  height: 15px;
}
</style>
