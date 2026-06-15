<template>
  <aside class="datacenter-nav-rail">
    <nav class="datacenter-nav-rail__nav" aria-label="数据中心模块导航">
      <button
        v-for="item in modules"
        :key="item.id"
        type="button"
        class="datacenter-nav-rail__item"
        :class="{ 'is-active': item.id === activeModule }"
        :aria-label="item.label"
        :title="item.label"
        @click="$emit('update:activeModule', item.id)"
      >
        <component :is="moduleIcons[item.id]" class="h-5 w-5" />
        <span class="datacenter-nav-rail__label">{{ item.label }}</span>
      </button>
    </nav>

    <div class="datacenter-nav-rail__actions" aria-label="数据中心快捷操作">
      <slot name="actions" />
    </div>
  </aside>
</template>

<script setup lang="ts">
import IconTablerBell from '~icons/tabler/bell'
import IconTablerCalculator from '~icons/tabler/calculator'
import IconTablerDatabase from '~icons/tabler/database'
import IconTablerPlugConnected from '~icons/tabler/plug-connected'
import type { DatacenterModuleId, DatacenterModuleMeta } from '@/config/datacenterModules'

const props = defineProps<{
  modules: DatacenterModuleMeta[]
  activeModule: DatacenterModuleId
}>()

defineEmits<{
  (event: 'update:activeModule', value: DatacenterModuleId): void
}>()

// 使用 v2 模块 ID 映射图标
const moduleIcons: Record<string, unknown> = {
  datapoint: IconTablerDatabase,
  'access-source': IconTablerPlugConnected,
  'storage-policy': IconTablerDatabase,
  compute: IconTablerCalculator,
  alarm: IconTablerBell,
}
</script>

<style scoped>
.datacenter-nav-rail {
  position: relative;
  z-index: 50;
  width: 48px;
  flex: 0 0 48px;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px 6px;
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  box-shadow: inset -1px 0 0 var(--dc-border);
}

.datacenter-nav-rail__nav {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.datacenter-nav-rail__actions {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: auto;
  padding-top: 8px;
  border-top: 1px solid var(--dc-border);
}

.datacenter-nav-rail__item {
  position: relative;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 12px;
  line-height: 1;
  transition:
    background-color 0.18s ease,
    border-color 0.18s ease,
    color 0.18s ease,
    transform 0.18s ease;
}

.datacenter-nav-rail__item:hover {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
  transform: translateY(-1px);
}

.datacenter-nav-rail__item.is-active {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  box-shadow: none;
  font-weight: 700;
}

.datacenter-nav-rail__label {
  position: absolute;
  z-index: 10;
  left: calc(100% + 8px);
  top: 50%;
  max-width: 120px;
  padding: 6px 9px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-popover);
  color: var(--dc-text);
  font-size: 12px;
  font-weight: 600;
  line-height: 1;
  opacity: 0;
  pointer-events: none;
  transform: translate(4px, -50%);
  transition:
    opacity 0.16s ease,
    transform 0.16s ease;
  white-space: nowrap;
}

.datacenter-nav-rail__item:hover .datacenter-nav-rail__label,
.datacenter-nav-rail__item:focus-visible .datacenter-nav-rail__label {
  opacity: 1;
  transform: translate(0, -50%);
}

.dark .datacenter-nav-rail {
  background: #162033;
  color: rgba(248, 250, 252, 0.76);
  box-shadow: inset -1px 0 0 rgba(255, 255, 255, 0.08);
}

.dark .datacenter-nav-rail__item {
  color: rgba(226, 232, 240, 0.76);
}

.dark .datacenter-nav-rail__actions {
  border-top-color: rgba(255, 255, 255, 0.08);
}

.dark .datacenter-nav-rail__item:hover {
  border-color: rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.12);
  color: #f8fafc;
}

.dark .datacenter-nav-rail__item.is-active {
  border-color: rgba(125, 168, 255, 0.9);
  background: #e7efff;
  color: var(--dc-primary);
}

.dark .datacenter-nav-rail__label {
  border-color: rgba(255, 255, 255, 0.12);
  background: #1e293b;
  color: #f8fafc;
}
</style>
