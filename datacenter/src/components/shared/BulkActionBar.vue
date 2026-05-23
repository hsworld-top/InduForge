<template>
  <div v-if="selectedCount > 0" class="dc-bulk-action-bar" role="status">
    <span class="dc-bulk-action-bar__count"> 已选 {{ selectedCount }} 项 </span>
    <div class="dc-bulk-action-bar__actions">
      <slot />
    </div>
    <button type="button" class="dc-bulk-action-bar__clear" @click="$emit('clear')">
      清空选择
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  selectedCount: number
}>()

defineEmits<{
  (event: 'clear'): void
}>()
</script>

<style scoped>
.dc-bulk-action-bar {
  position: absolute;
  left: 50%;
  bottom: 16px;
  z-index: 20;
  max-width: calc(100% - 32px);
  min-height: 44px;
  display: inline-flex;
  align-items: center;
  gap: 12px;
  padding: 8px 10px 8px 14px;
  border: 1px solid rgba(29, 78, 216, 0.2);
  border-radius: 12px;
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-popover);
  color: var(--dc-text);
  transform: translateX(-50%);
}

.dc-bulk-action-bar__count {
  flex: 0 0 auto;
  color: var(--dc-text-secondary);
  font-size: 13px;
  font-weight: 700;
  white-space: nowrap;
}

.dc-bulk-action-bar__actions {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.dc-bulk-action-bar__clear {
  height: 28px;
  flex: 0 0 auto;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: 8px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  font-weight: 600;
}

.dc-bulk-action-bar__clear:hover {
  border-color: rgba(29, 78, 216, 0.22);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
</style>
