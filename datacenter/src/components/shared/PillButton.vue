<template>
  <button
    type="button"
    class="dc-pill-button"
    :class="{ 'is-active': active }"
    :disabled="disabled"
    :aria-pressed="active"
  >
    <span v-if="$slots.icon" class="dc-pill-button__icon" aria-hidden="true">
      <slot name="icon" />
    </span>
    <span class="dc-pill-button__label">
      <slot />
    </span>
  </button>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    active?: boolean;
    disabled?: boolean;
  }>(),
  {
    active: false,
    disabled: false,
  },
);
</script>

<style scoped>
.dc-pill-button {
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  max-width: 100%;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: 12px;
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0;
  line-height: 1;
  transition:
    background-color 0.18s ease,
    border-color 0.18s ease,
    color 0.18s ease,
    box-shadow 0.18s ease;
}

.dc-pill-button:hover:not(:disabled) {
  border-color: rgba(29, 78, 216, 0.26);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.dc-pill-button:focus-visible {
  outline: none;
  border-color: var(--dc-primary);
  box-shadow: 0 0 0 3px rgba(29, 78, 216, 0.12);
}

.dc-pill-button.is-active {
  border-color: rgba(29, 78, 216, 0.26);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.dc-pill-button:disabled {
  cursor: not-allowed;
  opacity: 0.52;
}

.dc-pill-button__icon {
  width: 14px;
  height: 14px;
  flex: 0 0 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.dc-pill-button__icon :deep(svg) {
  width: 14px;
  height: 14px;
}

.dc-pill-button__label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
