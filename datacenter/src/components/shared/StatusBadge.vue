<template>
  <span class="dc-status-badge" :class="`dc-status-badge--${resolvedTone}`">
    <span class="dc-status-badge__dot" aria-hidden="true"></span>
    <span class="dc-status-badge__text">{{ displayText }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type StatusBadgeTone = 'success' | 'warning' | 'danger' | 'info' | 'muted'

const props = withDefaults(
  defineProps<{
    tone?: StatusBadgeTone
    text: string
  }>(),
  {
    tone: 'muted',
  },
)

const resolvedTone = computed<StatusBadgeTone>(() => props.tone || 'muted')
const displayText = computed(() => props.text || '-')
</script>

<style scoped>
.dc-status-badge {
  height: 22px;
  max-width: 100%;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 0 8px;
  border: 1px solid transparent;
  border-radius: 6px;
  font-family: inherit;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0;
  line-height: 1;
  white-space: nowrap;
}

.dc-status-badge__dot {
  width: 6px;
  height: 6px;
  flex: 0 0 6px;
  border-radius: 999px;
  background: currentColor;
}

.dc-status-badge__text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.dc-status-badge--success {
  border-color: rgba(22, 163, 74, 0.2);
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.dc-status-badge--warning {
  border-color: rgba(217, 119, 6, 0.22);
  background: var(--dc-warning-soft);
  color: var(--dc-warning);
}

.dc-status-badge--danger {
  border-color: rgba(220, 38, 38, 0.2);
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.dc-status-badge--info {
  border-color: rgba(29, 78, 216, 0.2);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.dc-status-badge--muted {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
}
</style>
