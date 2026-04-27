<template>
  <div class="datacenter-shell">
    <DataCenterNavRail
      :modules="modules"
      :active-module="activeModule"
      @update:activeModule="$emit('update:activeModule', $event)"
    />

    <section class="datacenter-shell__surface">
      <header class="datacenter-shell__header">
        <div>
          <div class="datacenter-shell__eyebrow">数据中心工作台</div>
          <h1 class="datacenter-shell__title">{{ activeMeta?.label }}</h1>
          <p class="datacenter-shell__description">
            {{ activeMeta?.description }}
          </p>
        </div>
        <div class="datacenter-shell__actions">
          <slot name="actions" :active-module="activeModule" />
        </div>
      </header>

      <main class="datacenter-shell__body">
        <slot />
      </main>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import DataCenterNavRail from "@/components/layout/DataCenterNavRail.vue";
import type {
  DatacenterModuleId,
  DatacenterModuleMeta,
} from "@/config/datacenterModules";

const props = defineProps<{
  modules: DatacenterModuleMeta[];
  activeModule: DatacenterModuleId;
}>();

defineEmits<{
  (event: "update:activeModule", value: DatacenterModuleId): void;
}>();

const activeMeta = computed(() =>
  props.modules.find((item) => item.id === props.activeModule),
);
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
  padding: 12px;
  gap: 12px;
}

.datacenter-shell__header {
  min-height: 58px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 14px;
  border: 1px solid var(--dc-border);
  border-radius: 8px;
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.datacenter-shell__eyebrow {
  color: var(--dc-primary);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0;
}

.datacenter-shell__title {
  margin: 2px 0 1px;
  color: var(--dc-text);
  font-size: 18px;
  font-weight: 700;
  letter-spacing: 0;
}

.datacenter-shell__description {
  margin: 0;
  max-width: 74ch;
  color: var(--dc-text-secondary);
  font-size: 13px;
  line-height: 1.45;
}

.datacenter-shell__actions {
  flex: 0 0 auto;
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

  .datacenter-shell__header {
    align-items: flex-start;
    flex-direction: column;
    border-radius: 8px;
  }
}
</style>
