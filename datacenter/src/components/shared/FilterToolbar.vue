<template>
  <div class="dc-filter-toolbar" :class="{ 'has-sections': Boolean($slots.tabs) }">
    <div v-if="$slots.tabs" class="dc-filter-toolbar__tabs"><slot name="tabs" /></div>
    <div v-if="$slots.default || $slots.actions" class="dc-filter-toolbar__row">
      <div class="dc-filter-toolbar__filters"><slot /></div>
      <div v-if="$slots.actions" class="dc-filter-toolbar__actions"><slot name="actions" /></div>
    </div>
  </div>
</template>

<style scoped>
.dc-filter-toolbar {
  min-width: 0;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.dc-filter-toolbar__tabs,
.dc-filter-toolbar__row {
  min-width: 0;
  display: flex;
  align-items: center;
}

.dc-filter-toolbar.has-sections .dc-filter-toolbar__tabs {
  min-height: 42px;
  padding: 0 14px;
  border-bottom: 1px solid var(--dc-border);
}

.dc-filter-toolbar__row {
  min-height: 48px;
  gap: 12px;
  padding: 8px 14px;
}

.dc-filter-toolbar__filters,
.dc-filter-toolbar__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
}

.dc-filter-toolbar__filters {
  min-width: 0;
  flex: 1 1 auto;
  overflow-x: auto;
  scrollbar-width: thin;
}

.dc-filter-toolbar__actions {
  flex: 0 0 auto;
}

@media (max-width: 760px) {
  .dc-filter-toolbar__row {
    align-items: stretch;
    flex-direction: column;
  }
  .dc-filter-toolbar__actions {
    justify-content: flex-end;
  }
}
</style>
