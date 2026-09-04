<template>
  <div class="node-resources" :class="{ 'is-historical': !online }">
    <div v-for="metric in metrics" :key="metric.key" class="node-resource">
      <span>{{ metric.label }}</span>
      <span class="node-resource__track"><i :style="{ width: `${metric.value ?? 0}%` }" /></span>
      <span class="node-resource__value">{{
        metric.value == null ? '—' : `${metric.value}%`
      }}</span>
    </div>
    <small>{{ online ? '最近上报' : '未连接 · 最近上报' }}</small>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { OpsNode } from '@/api/ops.api'
import { nodeResourcePercent } from '@/utils/node-resources'

const props = defineProps<{ node: OpsNode }>()
const online = computed(() => props.node.observedStatus === 'online')
const metrics = computed(() =>
  (
    [
      ['cpu', 'CPU'],
      ['memory', '内存'],
      ['disk', '磁盘'],
    ] as const
  ).map(([key, label]) => ({
    key,
    label,
    value: nodeResourcePercent(props.node, key),
  })),
)
</script>

<style scoped>
.node-resources {
  min-width: 170px;
  color: var(--ck-text-secondary, var(--el-text-color-secondary));
}
.node-resource {
  display: grid;
  grid-template-columns: 28px minmax(40px, 1fr) 58px;
  gap: 8px;
  align-items: center;
  font-size: 12px;
  line-height: 20px;
}
.node-resource__track {
  height: 3px;
  background: var(--el-fill-color);
  border-radius: 2px;
  overflow: hidden;
}
.node-resource__track i {
  display: block;
  height: 100%;
  background: var(--el-color-primary);
}
.node-resource__value {
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.node-resources small {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}
.is-historical .node-resource__track i {
  background: var(--el-text-color-placeholder);
}
</style>
