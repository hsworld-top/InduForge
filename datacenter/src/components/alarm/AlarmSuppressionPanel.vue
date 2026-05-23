<template>
  <section class="alarm-suppression-panel">
    <header>
      <strong>抑制策略</strong>
      <span>{{ enabled ? '已启用' : '未启用' }}</span>
    </header>

    <div class="alarm-suppression-panel__grid">
      <label class="alarm-suppression-panel__switch">
        <input
          type="checkbox"
          :checked="enabled"
          @change="update('enabled', checkboxValue($event))"
        />
        <span>suppression.enabled</span>
      </label>
      <label>
        <span>durationMs</span>
        <input
          :value="numberText('durationMs')"
          type="number"
          min="0"
          step="1"
          :disabled="!enabled"
          @input="updateOptionalNumber('durationMs', inputValue($event))"
        />
      </label>
      <label class="is-wide">
        <span>reason</span>
        <input
          :value="stringValue('reason')"
          type="text"
          :disabled="!enabled"
          @input="update('reason', inputValue($event))"
        />
      </label>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AlarmPolicyDraft } from '@/components/alarm/alarmPolicyModel'

type AlarmSuppression = Record<string, unknown>

const props = defineProps<{
  draft: AlarmPolicyDraft
}>()

const emit = defineEmits<{
  update: [patch: Partial<AlarmPolicyDraft>]
}>()

const enabled = computed(() => props.draft.suppression.enabled === true)

const inputValue = (event: Event) => (event.target as HTMLInputElement).value
const checkboxValue = (event: Event) => (event.target as HTMLInputElement).checked

const updateSuppression = (suppression: AlarmSuppression) => {
  emit('update', { suppression, dirty: true })
}

const update = (field: string, value: unknown) => {
  updateSuppression({ ...props.draft.suppression, [field]: value })
}

const updateOptionalNumber = (field: string, value: string) => {
  const next = { ...props.draft.suppression }
  if (value === '') {
    delete next[field]
  } else {
    next[field] = Number(value)
  }
  updateSuppression(next)
}

const numberText = (field: string) => {
  const value = props.draft.suppression[field]
  return typeof value === 'number' && Number.isFinite(value) ? String(value) : ''
}

const stringValue = (field: string) => {
  const value = props.draft.suppression[field]
  return typeof value === 'string' ? value : ''
}
</script>

<style scoped>
.alarm-suppression-panel {
  display: grid;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.alarm-suppression-panel header {
  display: flex;
  justify-content: space-between;
  gap: 10px;
}

.alarm-suppression-panel strong {
  color: var(--dc-text);
  font-size: 13px;
}

.alarm-suppression-panel header span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.alarm-suppression-panel__grid {
  display: grid;
  grid-template-columns: 180px minmax(120px, 220px) minmax(0, 1fr);
  gap: 10px;
}

.alarm-suppression-panel label {
  min-width: 0;
  display: grid;
  gap: 5px;
}

.alarm-suppression-panel label > span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.alarm-suppression-panel__switch {
  height: 34px;
  display: flex;
  align-items: center;
  align-self: end;
  gap: 8px;
}

.alarm-suppression-panel__switch input {
  width: 14px;
  height: 14px;
}

.alarm-suppression-panel input[type='text'],
.alarm-suppression-panel input[type='number'] {
  width: 100%;
  height: 34px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  font-family: inherit;
  font-size: 13px;
  outline: none;
  padding: 0 9px;
}

.alarm-suppression-panel input:disabled {
  cursor: not-allowed;
  opacity: 0.56;
}

@media (max-width: 820px) {
  .alarm-suppression-panel__grid {
    grid-template-columns: 1fr;
  }
}
</style>
