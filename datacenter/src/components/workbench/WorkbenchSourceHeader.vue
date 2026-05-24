<template>
  <div class="workbench-source-header">
    <button type="button" class="workbench-source-header__back" @click="$emit('back')">
      <IconTablerArrowLeft />
      <span>返回接入源</span>
    </button>

    <div class="workbench-source-header__head">
      <el-tooltip :content="title || fallbackTitle" placement="top" :show-after="400">
        <strong class="workbench-source-header__title">{{ title || fallbackTitle }}</strong>
      </el-tooltip>
      <slot name="status">
        <WorkbenchStatusPill v-if="statusLabel" :label="statusLabel" :tone="statusTone" />
      </slot>
    </div>

    <dl class="workbench-source-header__meta">
      <div v-for="row in meta" :key="row.label">
        <dt>{{ row.label }}</dt>
        <dd :title="row.value">{{ row.value }}</dd>
      </div>
    </dl>

    <div v-if="$slots.actions" class="workbench-source-header__actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
import IconTablerArrowLeft from '~icons/tabler/arrow-left'
import WorkbenchStatusPill from './WorkbenchStatusPill.vue'

type SourceMeta = {
  label: string
  value: string
}

withDefaults(
  defineProps<{
    title?: string
    fallbackTitle?: string
    statusLabel?: string
    statusTone?: 'success' | 'danger' | 'warning' | 'info' | 'neutral'
    meta?: SourceMeta[]
  }>(),
  {
    title: '',
    fallbackTitle: '未命名接入源',
    statusLabel: '',
    statusTone: 'neutral',
    meta: () => [],
  },
)

defineEmits<{
  (event: 'back'): void
}>()
</script>

<style scoped>
.workbench-source-header {
  padding: 12px;
  border-bottom: 1px solid var(--dc-border);
  background: linear-gradient(
    180deg,
    color-mix(in oklch, var(--dc-surface-raised) 92%, var(--dc-primary) 8%),
    var(--dc-surface-muted)
  );
}

.workbench-source-header__back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
  padding: 6px 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.workbench-source-header__back svg {
  width: 14px;
  height: 14px;
}

.workbench-source-header__back:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
  color: var(--dc-primary);
}

.workbench-source-header__head {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: start;
  gap: 8px;
}

.workbench-source-header__title {
  min-width: 0;
  display: block;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 15px;
  font-weight: 700;
  line-height: 26px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workbench-source-header__meta {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 6px;
  margin: 10px 0 0;
}

.workbench-source-header__meta div {
  min-width: 0;
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  min-height: 28px;
  padding: 5px 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: color-mix(in oklch, var(--dc-surface-raised) 78%, transparent);
}

.workbench-source-header__meta dt,
.workbench-source-header__meta dd {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workbench-source-header__meta dt {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.workbench-source-header__meta dd {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.workbench-source-header__actions {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
}

.workbench-source-header__actions :deep(.el-input) {
  min-width: 0;
  flex: 1;
}

.workbench-source-header__actions :deep(button) {
  flex: 0 0 auto;
}

.workbench-source-header__actions :deep(.workbench-source-header__icon-action) {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.workbench-source-header__actions :deep(.workbench-source-header__icon-action.is-primary) {
  border-color: var(--dc-primary);
  background: var(--dc-primary);
  color: var(--dc-surface-raised);
}

.workbench-source-header__actions :deep(.workbench-source-header__icon-action svg) {
  width: 15px;
  height: 15px;
}

.workbench-source-header__actions :deep(.workbench-source-header__icon-action:hover) {
  color: var(--dc-primary);
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
}

.workbench-source-header__actions :deep(.workbench-source-header__icon-action.is-primary:hover) {
  color: var(--dc-surface-raised);
  transform: translateY(-1px);
}
</style>
