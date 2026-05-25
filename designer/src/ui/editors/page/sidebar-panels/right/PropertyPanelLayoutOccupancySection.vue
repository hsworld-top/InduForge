<!--
  属性面板：流式布局子项占位配置
  用用户可理解的固定、填满、比例模式映射底层 flex flowLayout。
-->
<script setup lang="ts">
import type {
  LayoutOccupancyMode,
  LayoutOccupancyState,
  LayoutOccupancyUnit,
} from './layout-occupancy'

const props = defineProps<{
  state: LayoutOccupancyState
}>()

const emit = defineEmits<{
  (event: 'modeChange', value: LayoutOccupancyMode): void
  (event: 'fixedValueChange', value: string): void
  (event: 'fixedUnitChange', value: LayoutOccupancyUnit): void
  (event: 'ratioChange', value: number): void
}>()

const modeOptions: Array<{ label: string; value: LayoutOccupancyMode }> = [
  { label: '固定', value: 'fixed' },
  { label: '填满剩余', value: 'fill' },
  { label: '比例', value: 'ratio' },
]

function modeDescription(mode: LayoutOccupancyMode): string {
  if (mode === 'fixed') {
    return props.state.axis === 'column' ? '当前子项使用固定高度。' : '当前子项使用固定宽度。'
  }
  if (mode === 'ratio') {
    return `按 ${props.state.ratio} 份分配父布局剩余空间，填满剩余相当于 1 份。`
  }
  return '按 1 份参与分配，自动填满父布局剩余空间。'
}

function handleRatioInput(event: Event) {
  const target = event.target as HTMLInputElement | null
  const value = Number(target?.value)
  emit('ratioChange', Number.isFinite(value) && value >= 2 ? value : 2)
}

function handleFixedValueInput(event: Event) {
  const target = event.target as HTMLInputElement | null
  emit('fixedValueChange', target?.value || '')
}

function handleFixedUnitChange(event: Event) {
  const target = event.target as HTMLSelectElement | null
  emit('fixedUnitChange', target?.value === '%' ? '%' : 'px')
}
</script>

<template>
  <div v-if="state.visible" class="layout-occupancy-section">
    <div class="layout-occupancy-header">
      <span class="layout-occupancy-title">布局占位</span>
      <span class="layout-occupancy-context">
        父布局方向：{{ state.axis === 'column' ? '垂直' : '水平' }}
      </span>
    </div>

    <div class="layout-occupancy-body">
      <div class="layout-occupancy-row">
        <span class="layout-occupancy-label">
          {{ state.axis === 'column' ? '高度模式' : '宽度模式' }}
        </span>
        <div class="layout-mode-group" role="group">
          <button
            v-for="option in modeOptions"
            :key="option.value"
            type="button"
            class="layout-mode-button"
            :class="{ 'is-active': state.mode === option.value }"
            :data-test="`occupancy-mode-${option.value}`"
            @click="emit('modeChange', option.value)"
          >
            {{ option.label }}
          </button>
        </div>
      </div>

      <div v-if="state.mode === 'fixed'" class="layout-occupancy-row">
        <span class="layout-occupancy-label">
          {{ state.axis === 'column' ? '固定高度' : '固定宽度' }}
        </span>
        <div class="layout-size-control">
          <input
            class="layout-size-input"
            type="number"
            min="1"
            :value="state.fixedValue"
            @input="handleFixedValueInput"
          />
          <select class="layout-size-unit" :value="state.fixedUnit" @change="handleFixedUnitChange">
            <option value="px">px</option>
            <option value="%">%</option>
          </select>
        </div>
      </div>

      <div v-if="state.mode === 'ratio'" class="layout-occupancy-row">
        <span class="layout-occupancy-label">分配份数</span>
        <input
          class="layout-ratio-input"
          type="number"
          min="2"
          step="1"
          :value="state.ratio"
          @input="handleRatioInput"
        />
      </div>

      <div class="layout-occupancy-desc">{{ modeDescription(state.mode) }}</div>
    </div>
  </div>
</template>

<style scoped>
.layout-occupancy-section {
  overflow: hidden;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-shell-surface);
}

.layout-occupancy-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 30px;
  padding: 0 10px;
  border-bottom: 1px solid var(--designer-border-soft);
  background: var(--designer-group-surface);
}

.layout-occupancy-title {
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-sm);
  font-weight: 600;
  letter-spacing: 0.02em;
}

.layout-occupancy-context {
  color: var(--designer-text-muted);
  font-size: 12px;
}

.layout-occupancy-body {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
  padding: 8px 10px;
}

.layout-occupancy-row {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  min-height: 28px;
}

.layout-occupancy-label {
  color: var(--designer-text-regular);
  font-size: var(--designer-font-sm);
}

.layout-mode-group {
  display: flex;
  min-width: 0;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-sm);
  overflow: hidden;
}

.layout-mode-button {
  flex: 1;
  min-width: 0;
  height: 26px;
  padding: 0 6px;
  border: 0;
  border-right: 1px solid var(--designer-border-soft);
  background: transparent;
  color: var(--designer-text-regular);
  cursor: pointer;
  font-size: 12px;
}

.layout-mode-button:last-child {
  border-right: 0;
}

.layout-mode-button.is-active {
  background: var(--designer-primary-soft);
  color: var(--designer-primary-text);
  font-weight: 600;
}

.layout-size-control {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 56px;
  gap: 6px;
}

.layout-size-input,
.layout-size-unit,
.layout-ratio-input {
  width: 100%;
  height: 26px;
  padding: 0 8px;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-sm);
  background: var(--designer-shell-surface);
  color: var(--designer-text-primary);
  font-size: 12px;
  box-sizing: border-box;
}

.layout-occupancy-desc {
  color: var(--designer-text-muted);
  font-size: 12px;
  line-height: 1.5;
}
</style>
