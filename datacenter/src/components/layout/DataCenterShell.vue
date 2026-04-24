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
          <div class="datacenter-shell__eyebrow">InduForge Data Center</div>
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
  --dc-bg: #f4f1ea;
  --dc-paper: #fffdf7;
  --dc-ink: #20231f;
  --dc-muted: #687066;
  --dc-line: #d6cebf;
  --dc-deep: #26322e;
  --dc-accent: #197278;
  --dc-soft: #e8f0ed;
  --dc-shadow: 0 14px 34px rgba(48, 42, 32, 0.11);

  height: 100%;
  min-height: 480px;
  display: flex;
  overflow: hidden;
  background:
    linear-gradient(125deg, rgba(25, 114, 120, 0.1), transparent 34%),
    linear-gradient(315deg, rgba(184, 121, 36, 0.09), transparent 42%),
    var(--dc-bg);
}

.datacenter-shell__surface {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 14px;
  gap: 12px;
}

.datacenter-shell__header {
  min-height: 76px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 13px 16px;
  border: 1px solid var(--dc-line);
  border-radius: 8px;
  background: rgba(255, 253, 247, 0.94);
  box-shadow: var(--dc-shadow);
}

.datacenter-shell__eyebrow {
  color: var(--dc-accent);
  font-size: 11px;
  font-weight: 900;
  letter-spacing: 0.16em;
  text-transform: uppercase;
}

.datacenter-shell__title {
  margin: 4px 0 2px;
  color: var(--dc-ink);
  font-size: 23px;
  font-weight: 900;
  letter-spacing: -0.02em;
}

.datacenter-shell__description {
  margin: 0;
  max-width: 74ch;
  color: var(--dc-muted);
  font-size: 13px;
  line-height: 1.55;
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
  --el-button-bg-color: var(--dc-accent);
  --el-button-border-color: var(--dc-accent);
  --el-button-hover-bg-color: #255b5f;
  --el-button-hover-border-color: #255b5f;
  --el-button-active-bg-color: #174f54;
  --el-button-active-border-color: #174f54;
}

.datacenter-shell :deep(.el-button) {
  border-radius: 5px;
  font-weight: 800;
}

.datacenter-shell :deep(.el-input__wrapper),
.datacenter-shell :deep(.el-textarea__inner),
.datacenter-shell :deep(.el-select__wrapper) {
  border-radius: 5px;
  background: #faf8f0;
  box-shadow: 0 0 0 1px #d8cfbd inset;
}

.datacenter-shell :deep(.el-input__wrapper.is-focus),
.datacenter-shell :deep(.el-textarea__inner:focus),
.datacenter-shell :deep(.el-select__wrapper.is-focused) {
  box-shadow: 0 0 0 1px var(--dc-accent) inset;
}

.datacenter-shell :deep(.el-segmented) {
  --el-segmented-bg-color: #f1eee5;
  --el-segmented-item-selected-color: #fffdf7;
  --el-segmented-item-selected-bg-color: var(--dc-accent);
  border-radius: 5px;
}

.datacenter-shell :deep(.el-tag) {
  border-radius: 4px;
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
