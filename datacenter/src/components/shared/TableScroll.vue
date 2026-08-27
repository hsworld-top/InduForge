<template>
  <div class="dc-table-layout">
    <div class="dc-table-scroll"><slot /></div>
    <div v-if="$slots.pagination" class="dc-table-layout__pagination">
      <slot name="pagination" />
    </div>
    <slot name="floating" />
  </div>
</template>

<style scoped>
.dc-table-layout {
  position: relative;
  min-width: 0;
  min-height: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.dc-table-scroll {
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  overflow: auto;
}

.dc-table-scroll :deep(.el-table) {
  min-width: 920px;
}

.dc-table-layout__pagination {
  position: relative;
  z-index: 2;
  flex: 0 0 auto;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}
</style>
