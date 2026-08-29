<template>
  <aside class="datacenter-nav-rail">
    <nav class="datacenter-nav-rail__nav" :aria-label="t('shell.moduleNavigation')">
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

    <div class="datacenter-nav-rail__actions" :aria-label="t('shell.quickActions')">
      <slot name="actions" />
    </div>
  </aside>
</template>

<script setup lang="ts">
import IconTablerBellRinging from '~icons/tabler/bell-ringing'
import IconTablerChartDots3 from '~icons/tabler/chart-dots-3'
import IconTablerDatabaseCog from '~icons/tabler/database-cog'
import IconTablerFunction from '~icons/tabler/function'
import IconTablerCpu from '~icons/tabler/cpu'
import IconTablerRouter from '~icons/tabler/router'
import type { DatacenterModuleId, DatacenterModuleMeta } from '@/config/datacenterModules'
import { t } from '@/i18n/runtime'

defineProps<{
  modules: DatacenterModuleMeta[]
  activeModule: DatacenterModuleId | null
}>()

defineEmits<{
  (event: 'update:activeModule', value: DatacenterModuleId): void
}>()

// 使用 v2 模块 ID 映射图标
const moduleIcons: Record<string, unknown> = {
  datapoint: IconTablerChartDots3,
  'access-source': IconTablerRouter,
  'industrial-collector': IconTablerCpu,
  'history-storage': IconTablerDatabaseCog,
  compute: IconTablerFunction,
  alarm: IconTablerBellRinging,
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

</style>
