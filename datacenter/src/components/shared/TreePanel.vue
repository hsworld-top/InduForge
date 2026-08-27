<template>
  <aside class="dc-tree-panel">
    <header class="dc-tree-panel__header">
      <h2>{{ title }}</h2>
      <div class="dc-tree-panel__actions"><slot name="actions" /></div>
    </header>

    <div class="dc-tree-panel__search">
      <el-input
        :model-value="searchText"
        clearable
        size="small"
        :placeholder="searchPlaceholder"
        :prefix-icon="Search"
        @update:model-value="emit('update:searchText', String($event || ''))"
      />
    </div>

    <slot name="notice" />

    <button
      type="button"
      class="dc-tree-panel__all"
      :class="{ 'is-active': allActive }"
      @click="emit('selectAll')"
    >
      <IconTablerFolders />
      <span>{{ allLabel }}</span>
    </button>

    <div v-loading="loading" class="dc-tree-panel__body" :aria-busy="loading">
      <slot />
    </div>
  </aside>
</template>

<script setup lang="ts">
import { Search } from '@element-plus/icons-vue'
import IconTablerFolders from '~icons/tabler/folders'

withDefaults(
  defineProps<{
    title: string
    searchText?: string
    searchPlaceholder?: string
    allLabel: string
    allActive?: boolean
    loading?: boolean
  }>(),
  {
    searchText: '',
    searchPlaceholder: '搜索',
    allActive: false,
    loading: false,
  },
)

const emit = defineEmits<{
  'update:searchText': [value: string]
  selectAll: []
}>()
</script>

<style scoped>
.dc-tree-panel {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle, var(--dc-surface-raised));
}

.dc-tree-panel__header {
  min-height: 42px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 10px;
  border-bottom: 1px solid var(--dc-border);
}

.dc-tree-panel__header h2 {
  min-width: 0;
  margin: 0;
  color: var(--dc-text);
  font-size: 14px;
  font-weight: 700;
  line-height: 1.3;
}

.dc-tree-panel__actions {
  display: flex;
  align-items: center;
  gap: 5px;
}

.dc-tree-panel__actions :deep(button) {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  cursor: pointer;
}

.dc-tree-panel__actions :deep(button:hover) {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.dc-tree-panel__actions :deep(button.is-primary) {
  border-color: var(--dc-primary);
  background: var(--dc-primary);
  color: white;
}

.dc-tree-panel__actions :deep(svg),
.dc-tree-panel__all svg {
  width: 16px;
  height: 16px;
}

.dc-tree-panel__search {
  padding: 8px 8px 6px;
}

.dc-tree-panel__all {
  min-height: 32px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  align-items: center;
  gap: 7px;
  margin: 0 6px 4px;
  padding: 3px 7px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-size: 13px;
  text-align: left;
}

.dc-tree-panel__all:hover,
.dc-tree-panel__all.is-active {
  border-color: rgba(29, 78, 216, 0.24);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.dc-tree-panel__body {
  min-height: 0;
  flex: 1;
  display: grid;
  align-content: start;
  gap: 2px;
  padding: 2px 6px 8px;
  overflow: auto;
}
</style>
