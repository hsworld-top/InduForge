<template>
  <nav class="workbench-tabbar" aria-label="工作台标签页">
    <div
      v-for="tab in tabs"
      :key="tab.id"
      class="workbench-tabbar__item"
      :class="{ 'is-active': tab.id === activeId }"
    >
      <button type="button" class="workbench-tabbar__tab" @click="$emit('update:activeId', tab.id)">
        <component :is="tab.icon" v-if="tab.icon" />
        <span>{{ tab.title }}</span>
      </button>
      <button
        v-if="closable"
        type="button"
        class="workbench-tabbar__close"
        :aria-label="`关闭 ${tab.title}`"
        @click="$emit('close', tab.id)"
      >
        <IconTablerX />
      </button>
    </div>
  </nav>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import IconTablerX from '~icons/tabler/x'

export type WorkbenchTabBarItem = {
  id: string
  title: string
  icon?: Component
}

withDefaults(
  defineProps<{
    tabs: WorkbenchTabBarItem[]
    activeId: string
    closable?: boolean
  }>(),
  {
    closable: true,
  },
)

defineEmits<{
  (event: 'update:activeId', value: string): void
  (event: 'close', value: string): void
}>()
</script>

<style scoped>
.workbench-tabbar {
  min-height: 38px;
  display: flex;
  overflow-x: auto;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.workbench-tabbar:empty {
  display: none;
}

.workbench-tabbar__item {
  min-width: 132px;
  max-width: 240px;
  height: 38px;
  display: inline-flex;
  align-items: center;
  border-right: 1px solid var(--dc-border);
  color: var(--dc-text-secondary);
}

.workbench-tabbar__item.is-active {
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
  font-weight: 700;
}

.workbench-tabbar__tab {
  min-width: 0;
  height: 100%;
  flex: 1;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 0 8px;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
}

.workbench-tabbar__tab span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workbench-tabbar__tab svg,
.workbench-tabbar__close svg {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}

.workbench-tabbar__close {
  width: 24px;
  height: 24px;
  margin-right: 4px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-muted);
}

.workbench-tabbar__close:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-danger);
}
</style>
