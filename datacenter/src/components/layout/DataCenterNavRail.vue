<template>
  <aside class="datacenter-nav-rail">
    <div class="datacenter-nav-rail__brand">
      <div class="datacenter-nav-rail__mark">IF</div>
      <div class="datacenter-nav-rail__caption">DATA</div>
    </div>

    <nav class="datacenter-nav-rail__nav" aria-label="数据中心模块导航">
      <button
        v-for="item in modules"
        :key="item.id"
        type="button"
        class="datacenter-nav-rail__item"
        :class="{ 'is-active': item.id === activeModule }"
        :title="item.label"
        @click="$emit('update:activeModule', item.id)"
      >
        <component :is="moduleIcons[item.id]" class="h-5 w-5" />
        <span>{{ item.label }}</span>
      </button>
    </nav>
  </aside>
</template>

<script setup lang="ts">
import IconTablerBell from "~icons/tabler/bell";
import IconTablerCalculator from "~icons/tabler/calculator";
import IconTablerDatabase from "~icons/tabler/database";
import IconTablerPlugConnected from "~icons/tabler/plug-connected";
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

const moduleIcons = {
  datapoints: IconTablerDatabase,
  "access-sources": IconTablerPlugConnected,
  "compute-units": IconTablerCalculator,
  "alarm-units": IconTablerBell,
};
</script>

<style scoped>
.datacenter-nav-rail {
  width: 68px;
  flex: 0 0 68px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  padding: 10px 8px;
  background: #162033;
  color: rgba(248, 250, 252, 0.76);
  box-shadow: inset -1px 0 0 rgba(255, 255, 255, 0.08);
}

.datacenter-nav-rail__brand {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.datacenter-nav-rail__mark {
  width: 38px;
  height: 38px;
  display: grid;
  place-items: center;
  border: 1px solid rgba(255, 255, 255, 0.18);
  border-radius: var(--dc-radius-sm);
  background: rgba(29, 78, 216, 0.26);
  color: #f8fafc;
  font-size: 13px;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.datacenter-nav-rail__caption {
  color: rgba(226, 232, 240, 0.58);
  font-size: 10px;
  letter-spacing: 0.22em;
  writing-mode: vertical-rl;
}

.datacenter-nav-rail__nav {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.datacenter-nav-rail__item {
  width: 52px;
  min-height: 48px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: var(--dc-radius-sm);
  background: rgba(255, 255, 255, 0.06);
  color: rgba(226, 232, 240, 0.76);
  font-size: 11px;
  line-height: 1;
  transition:
    background-color 0.18s ease,
    border-color 0.18s ease,
    color 0.18s ease,
    transform 0.18s ease;
}

.datacenter-nav-rail__item:hover {
  background: rgba(255, 255, 255, 0.12);
  color: #f8fafc;
  transform: translateY(-1px);
}

.datacenter-nav-rail__item.is-active {
  border-color: rgba(125, 168, 255, 0.9);
  background: #e7efff;
  color: var(--dc-primary);
  box-shadow: none;
  font-weight: 700;
}
</style>
