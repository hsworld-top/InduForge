<!--
  布局容器直接子项分配配置。
  只处理 HorizontalLayout / VerticalLayout 的一键平均分配，复杂比例仍交给单个子项的“布局占位”面板。
-->
<script setup lang="ts">
import type { LayoutOccupancyAxis } from './layout-occupancy'

defineProps<{
  visible: boolean
  axis: LayoutOccupancyAxis
  childCount: number
}>()

const emit = defineEmits<{
  (event: 'distributeAverage'): void
}>()

function actionLabel(axis: LayoutOccupancyAxis): string {
  return axis === 'row' ? '水平平均分配宽度' : '垂直平均分配高度'
}

function description(axis: LayoutOccupancyAxis): string {
  return axis === 'row'
    ? '每个直接子项使用相同宽度份额，适合按钮组、工具栏和顶部区域。'
    : '每个直接子项使用相同高度份额，适合上下区域等高排布。'
}
</script>

<template>
  <div v-if="visible" class="layout-distribution-section">
    <div class="layout-distribution-header">
      <span class="layout-distribution-title">子项分配</span>
      <span class="layout-distribution-count">{{ childCount }} 个直接子项</span>
    </div>
    <div class="layout-distribution-body">
      <button
        type="button"
        class="layout-distribution-button"
        data-test="layout-distribute-average"
        :disabled="childCount === 0"
        @click="emit('distributeAverage')"
      >
        {{ actionLabel(axis) }}
      </button>
      <div class="layout-distribution-desc">{{ description(axis) }}</div>
    </div>
  </div>
</template>

<style scoped>
.layout-distribution-section {
  overflow: hidden;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-shell-surface);
}

.layout-distribution-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 30px;
  padding: 0 10px;
  border-bottom: 1px solid var(--designer-border-soft);
  background: var(--designer-group-surface);
}

.layout-distribution-title {
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-sm);
  font-weight: 600;
  letter-spacing: 0.02em;
}

.layout-distribution-count {
  color: var(--designer-text-muted);
  font-size: 12px;
}

.layout-distribution-body {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
  padding: 8px 10px;
}

.layout-distribution-button {
  width: 100%;
  height: 28px;
  border: 1px solid var(--designer-primary-border);
  border-radius: var(--designer-radius-sm);
  background: var(--designer-primary-soft);
  color: var(--designer-primary-text);
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
}

.layout-distribution-button:disabled {
  border-color: var(--designer-border-color);
  background: var(--designer-group-surface);
  color: var(--designer-text-muted);
  cursor: not-allowed;
}

.layout-distribution-desc {
  color: var(--designer-text-muted);
  font-size: 12px;
  line-height: 1.5;
}
</style>
