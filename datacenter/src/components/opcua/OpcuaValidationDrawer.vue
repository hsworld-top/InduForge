<template>
  <el-drawer :model-value="modelValue" title="建模校验" size="420px" @close="$emit('update:modelValue', false)">
    <div class="opcua-validation__summary">
      <strong>{{ issues.length === 0 ? '建模通过' : `${issues.length} 个校验问题` }}</strong>
      <span>点击问题可回到对应变量</span>
    </div>
    <div v-if="issues.length === 0" class="opcua-validation__empty">当前建模无校验问题</div>
    <div v-for="issue in issues" :key="`${issue.code}-${issue.nodeId || issue.groupId}`" class="opcua-validation__item">
      <el-tag size="small" :type="issue.severity === 'error' ? 'danger' : 'warning'">
        {{ issue.severity }}
      </el-tag>
      <div>
        <strong>{{ issue.nodeName || issue.code }}</strong>
        <span>{{ issue.message }}</span>
      </div>
      <el-button v-if="issue.nodeId" text size="small" @click="$emit('locate', issue.nodeId)">
        定位变量
      </el-button>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import type { OpcuaValidationIssue } from './types'

defineProps<{
  modelValue: boolean
  issues: OpcuaValidationIssue[]
}>()

defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'locate', nodeId: string): void
}>()
</script>

<style scoped>
.opcua-validation__empty {
  padding: 16px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
  color: var(--dc-success);
}

.opcua-validation__summary {
  display: grid;
  gap: 4px;
  margin-bottom: 12px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
}

.opcua-validation__summary strong {
  color: var(--dc-text);
  font-size: 14px;
}

.opcua-validation__summary span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.opcua-validation__item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 10px;
  padding: 12px 0;
  border-bottom: 1px solid var(--dc-border);
}

.opcua-validation__item div {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.opcua-validation__item strong,
.opcua-validation__item span {
  overflow-wrap: anywhere;
}

.opcua-validation__item span {
  color: var(--dc-text-muted);
  font-size: 13px;
}
</style>
