<script setup lang="ts">
const props = defineProps<{
  resolvedProps?: Record<string, unknown>
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: number): void
}>()

function normalizeNumber(value: unknown, fallback: number): number {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function clampValue(value: number): number {
  const min = normalizeNumber(props.resolvedProps?.min, Number.NEGATIVE_INFINITY)
  const max = normalizeNumber(props.resolvedProps?.max, Number.POSITIVE_INFINITY)
  return Math.min(max, Math.max(min, value))
}

function handleModelValueUpdate(value: unknown): void {
  emit('update:modelValue', clampValue(normalizeNumber(value, 0)))
}

function handleChange(delta: number): void {
  if (props.resolvedProps?.disabled === true) return
  const current = normalizeNumber(props.resolvedProps?.modelValue, 0)
  const step = normalizeNumber(props.resolvedProps?.step, 1)
  emit('update:modelValue', clampValue(current + delta * step))
}
</script>

<template>
  <el-input-number
    v-bind="resolvedProps"
    class="core-input-number"
    @update:model-value="handleModelValueUpdate"
  />
  <button
    class="core-input-number__hit core-input-number__hit--increase"
    type="button"
    aria-label="增加数值"
    @click.stop="handleChange(1)"
  />
  <button
    class="core-input-number__hit core-input-number__hit--decrease"
    type="button"
    aria-label="减少数值"
    @click.stop="handleChange(-1)"
  />
</template>

<style scoped>
.core-input-number {
  width: 100%;
  height: 100%;
}

.core-input-number__hit {
  position: absolute;
  right: 0;
  z-index: 2;
  width: 32px;
  height: 50%;
  border: 0;
  background: transparent;
  cursor: pointer;
}

.core-input-number__hit--increase {
  top: 0;
}

.core-input-number__hit--decrease {
  bottom: 0;
}
</style>
