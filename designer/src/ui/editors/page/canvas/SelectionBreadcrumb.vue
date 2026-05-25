<script setup lang="ts">
import type { SelectionBreadcrumbItem } from './selection-breadcrumb'

defineProps<{
  items: SelectionBreadcrumbItem[]
}>()

const emit = defineEmits<{
  (event: 'select', id: string): void
}>()

function handleSelect(item: SelectionBreadcrumbItem): void {
  if (item.current) return
  emit('select', item.id)
}
</script>

<template>
  <div v-if="items.length > 1" class="selection-breadcrumb" aria-label="选中层级路径">
    <template v-for="(item, index) in items" :key="item.id">
      <span v-if="index > 0" class="selection-breadcrumb__separator">/</span>
      <button
        class="selection-breadcrumb__item"
        :class="{ 'is-current': item.current }"
        :title="`${item.label} (${item.type})`"
        :disabled="item.current"
        :data-test="`selection-breadcrumb-item-${item.id}`"
        @click="handleSelect(item)"
      >
        {{ item.label }}
      </button>
    </template>
  </div>
</template>

<style scoped>
.selection-breadcrumb {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  max-width: 360px;
  height: 28px;
  margin-right: 6px;
  padding-right: 6px;
  border-right: 1px solid rgba(15, 23, 42, 0.1);
  color: #64748b;
}

.selection-breadcrumb__separator {
  flex: 0 0 auto;
  padding: 0 2px;
  font-size: 12px;
  color: #cbd5e1;
}

.selection-breadcrumb__item {
  display: inline-flex;
  align-items: center;
  max-width: 92px;
  height: 24px;
  padding: 0 6px;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: #475569;
  font-size: 12px;
  line-height: 24px;
  cursor: pointer;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.selection-breadcrumb__item:hover {
  background: rgba(59, 130, 246, 0.1);
  color: #2563eb;
}

.selection-breadcrumb__item.is-current {
  color: #1d4ed8;
  font-weight: 600;
  cursor: default;
}

.selection-breadcrumb__item:disabled {
  opacity: 1;
}

@media (prefers-color-scheme: dark) {
  .selection-breadcrumb {
    border-right-color: rgba(255, 255, 255, 0.1);
    color: #94a3b8;
  }

  .selection-breadcrumb__item {
    color: #cbd5e1;
  }

  .selection-breadcrumb__item:hover {
    background: rgba(59, 130, 246, 0.15);
    color: #60a5fa;
  }

  .selection-breadcrumb__item.is-current {
    color: #93c5fd;
  }
}
</style>
