<template>
  <div class="datacenter-shell">
    <DataCenterNavRail
      :modules="modules"
      :active-module="activeModule"
      @update:activeModule="$emit('update:activeModule', $event)"
    >
      <template #actions>
        <slot name="actions" :active-module="activeModule" />
      </template>
    </DataCenterNavRail>

    <section class="datacenter-shell__surface">
      <main class="datacenter-shell__body">
        <slot />
      </main>
    </section>
  </div>
</template>

<script setup lang="ts">
import DataCenterNavRail from "@/components/layout/DataCenterNavRail.vue";
import type {
  DatacenterModuleId,
  DatacenterModuleMeta,
} from "@/config/datacenterModules";

defineProps<{
  modules: DatacenterModuleMeta[];
  activeModule: DatacenterModuleId;
}>();

defineEmits<{
  (event: "update:activeModule", value: DatacenterModuleId): void;
}>();

</script>

<style scoped>
.datacenter-shell {
  height: 100%;
  min-height: 480px;
  display: flex;
  overflow: hidden;
  background: var(--dc-bg);
  color: var(--dc-text);
}

.datacenter-shell__surface {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 10px;
}

.datacenter-shell__body {
  min-height: 0;
  flex: 1;
  overflow: hidden;
}

.datacenter-shell :deep(.el-button--primary) {
  --el-button-bg-color: var(--dc-primary);
  --el-button-border-color: var(--dc-primary);
  --el-button-hover-bg-color: var(--dc-primary-hover);
  --el-button-hover-border-color: var(--dc-primary-hover);
  --el-button-active-bg-color: var(--dc-primary-hover);
  --el-button-active-border-color: var(--dc-primary-hover);
}

.datacenter-shell :deep(.el-button) {
  border-radius: var(--dc-radius-sm);
  font-weight: 600;
}

.datacenter-shell :deep(.el-input__wrapper),
.datacenter-shell :deep(.el-textarea__inner),
.datacenter-shell :deep(.el-select__wrapper) {
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  box-shadow: 0 0 0 1px var(--dc-border) inset;
}

.datacenter-shell :deep(.el-input__wrapper.is-focus),
.datacenter-shell :deep(.el-textarea__inner:focus),
.datacenter-shell :deep(.el-select__wrapper.is-focused) {
  box-shadow:
    0 0 0 1px var(--dc-primary) inset,
    0 0 0 3px rgba(29, 78, 216, 0.12);
}

.datacenter-shell :deep(.el-segmented) {
  --el-segmented-bg-color: var(--dc-surface-muted);
  --el-segmented-item-selected-color: var(--dc-primary);
  --el-segmented-item-selected-bg-color: var(--dc-surface-raised);
  border-radius: var(--dc-radius-sm);
}

.datacenter-shell :deep(.el-tag) {
  border-radius: var(--dc-radius-xs);
}

@media (max-width: 768px) {
  .datacenter-shell__surface {
    padding: 10px;
  }
}
</style>
