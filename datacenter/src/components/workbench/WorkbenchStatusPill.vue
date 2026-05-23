<template>
  <span class="workbench-status-pill" :class="`is-${tone}`">
    <span class="workbench-status-pill__dot" />
    <span class="workbench-status-pill__label">{{ label }}</span>
  </span>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    label: string
    tone?: 'neutral' | 'success' | 'warning' | 'danger' | 'info'
  }>(),
  {
    tone: 'neutral',
  },
)
</script>

<style scoped>
.workbench-status-pill {
  --pill-color: var(--dc-text-muted, #64748b);
  --pill-bg: var(--dc-surface-subtle, #f8fafc);
  --pill-border: var(--dc-border, #e2e8f0);

  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  height: 22px;
  padding: 0 8px;
  border: 1px solid var(--pill-border);
  border-radius: 999px;
  background: var(--pill-bg);
  color: var(--pill-color);
  font-size: 11px;
  font-weight: 700;
  line-height: 1;
  white-space: nowrap;
}

.workbench-status-pill__dot {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: currentColor;
  box-shadow: 0 0 0 3px color-mix(in oklch, currentColor 16%, transparent);
  flex-shrink: 0;
}

.workbench-status-pill__label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.workbench-status-pill.is-success {
  --pill-color: #047857;
  --pill-bg: color-mix(in oklch, #10b981 11%, var(--dc-surface-raised, #fff));
  --pill-border: color-mix(in oklch, #10b981 26%, var(--dc-border, #e2e8f0));
}

.workbench-status-pill.is-warning {
  --pill-color: #a16207;
  --pill-bg: color-mix(in oklch, #f59e0b 13%, var(--dc-surface-raised, #fff));
  --pill-border: color-mix(in oklch, #f59e0b 30%, var(--dc-border, #e2e8f0));
}

.workbench-status-pill.is-danger {
  --pill-color: #b91c1c;
  --pill-bg: color-mix(in oklch, #ef4444 10%, var(--dc-surface-raised, #fff));
  --pill-border: color-mix(in oklch, #ef4444 26%, var(--dc-border, #e2e8f0));
}

.workbench-status-pill.is-info {
  --pill-color: var(--dc-primary, #2563eb);
  --pill-bg: var(--dc-primary-soft, #eff6ff);
  --pill-border: color-mix(in oklch, var(--dc-primary, #2563eb) 22%, var(--dc-border, #e2e8f0));
}
</style>
