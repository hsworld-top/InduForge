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
  width: 76px;
  flex: 0 0 76px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 18px;
  padding: 12px 9px;
  background: #26322e;
  color: rgba(255, 255, 255, 0.76);
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
  border-radius: 7px;
  background: rgba(255, 255, 255, 0.08);
  color: #fffdf7;
  font-size: 13px;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.datacenter-nav-rail__caption {
  color: rgba(255, 253, 247, 0.54);
  font-size: 10px;
  letter-spacing: 0.22em;
  writing-mode: vertical-rl;
}

.datacenter-nav-rail__nav {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.datacenter-nav-rail__item {
  width: 56px;
  min-height: 50px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 7px;
  background: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.74);
  font-size: 11px;
  line-height: 1;
  transition:
    background-color 0.18s ease,
    border-color 0.18s ease,
    color 0.18s ease,
    transform 0.18s ease;
}

.datacenter-nav-rail__item:hover {
  background: rgba(255, 255, 255, 0.14);
  color: #fffdf7;
  transform: translateY(-1px);
}

.datacenter-nav-rail__item.is-active {
  border-color: #f3f0e7;
  background: #f3f0e7;
  color: #26322e;
  box-shadow: none;
  font-weight: 900;
}
</style>
