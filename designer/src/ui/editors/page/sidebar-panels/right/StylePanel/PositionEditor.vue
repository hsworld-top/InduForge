<!--
  PositionEditor - 定位编辑器
  编辑 position、left/top/right/bottom
-->
<script setup lang="ts">
import { computed } from 'vue'

interface PositionStyleModel {
  position?: string
  left?: number | string | null
  top?: number | string | null
  zIndex?: number | string | null
  overflow?: string
}

const props = withDefaults(defineProps<{ modelValue?: PositionStyleModel }>(), {
  modelValue: () => ({}),
})

function parseValue(value: unknown): number | undefined {
  if (value === undefined || value === null || value === '') return undefined
  const num = Number.parseInt(String(value), 10)
  return Number.isNaN(num) ? undefined : num
}

function formatDisplayValue(value: unknown): string {
  if (value === undefined || value === null || value === '') {
    return '-'
  }
  return String(value)
}

const position = computed(() => props.modelValue.position || 'relative')
const showCoordinates = computed(() => ['absolute', 'fixed'].includes(position.value))
const left = computed(() => parseValue(props.modelValue.left))
const top = computed(() => parseValue(props.modelValue.top))
const zIndex = computed(() => parseValue(props.modelValue.zIndex) || 0)
const overflow = computed(() => props.modelValue.overflow || 'visible')
</script>

<template>
  <div class="position-editor">
    <div class="editor-group-title">定位</div>
    <div class="position-row">
      <div class="position-label">模式</div>
      <div class="position-control">
        <el-input :model-value="position" size="small" readonly />
      </div>
    </div>

    <div v-if="showCoordinates" class="position-row">
      <div class="position-label">位置</div>
      <div class="position-control">
        <div class="axis-inline-group">
          <div class="axis-inline-item">
            <span class="axis-inline-tag">X</span>
            <el-input :model-value="formatDisplayValue(left)" size="small" readonly />
          </div>
          <div class="axis-inline-item">
            <span class="axis-inline-tag">Y</span>
            <el-input :model-value="formatDisplayValue(top)" size="small" readonly />
          </div>
        </div>
      </div>
    </div>

    <div class="position-row">
      <div class="position-label">层级</div>
      <div class="position-control">
        <el-input :model-value="formatDisplayValue(zIndex)" size="small" readonly />
      </div>
    </div>

    <div class="position-row">
      <div class="position-label">溢出</div>
      <div class="position-control">
        <el-input :model-value="overflow" size="small" readonly />
      </div>
    </div>
  </div>
</template>

<style scoped>
.position-editor {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
}

.editor-group-title {
  display: flex;
  align-items: center;
  min-height: 30px;
  padding: 0 10px;
  margin: -8px -10px 0;
  background: var(--designer-group-surface);
  border-bottom: 1px solid var(--designer-border-soft);
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-sm);
  font-weight: 600;
  letter-spacing: 0.02em;
}

.position-row {
  display: flex;
  align-items: center;
  gap: var(--designer-gap-sm);
  min-height: 28px;
  padding: 2px 4px;
  border-radius: var(--designer-radius-sm);
  transition: background-color 0.15s ease;
}

.position-row:hover {
  background: var(--designer-hover-surface);
}

.position-label {
  width: 88px;
  min-width: 72px;
  max-width: 88px;
  font-size: var(--designer-font-sm);
  color: var(--designer-text-regular);
}

.position-control {
  flex: 1;
  min-width: 0;
}

.position-control :deep(.el-input) {
  width: 100%;
}

.axis-inline-group {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--designer-gap-xs);
}

.axis-inline-item {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.axis-inline-item :deep(.el-input) {
  width: 100%;
}

.axis-inline-tag {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  min-width: 18px;
  height: 18px;
  border-radius: 999px;
  background: var(--designer-group-surface);
  color: var(--designer-text-secondary);
  font-size: 11px;
  font-weight: 600;
}
</style>
