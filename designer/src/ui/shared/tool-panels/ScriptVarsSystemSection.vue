<!--
  脚本与变量面板 — 系统脚本折叠块
-->
<script setup lang="ts">
import { useI18n } from "vue-i18n";
import IconEpEditPen from "~icons/ep/edit-pen";

defineProps<{
  selectedSystemKey: string;
}>();

const emit = defineEmits<{
  selectSystem: [key: string];
  openSystemEditor: [kind: "startup" | "shutdown"];
}>();
const { t } = useI18n();
</script>

<template>
  <el-collapse-item name="system">
    <template #title>
      <div class="section-title">
        <span>{{ t("scriptPanel.sections.system") }}</span>
        <span class="section-count">2</span>
      </div>
    </template>
    <div class="system-list">
      <button
        class="system-row"
        :class="{ 'is-selected': selectedSystemKey === 'startup' }"
        type="button"
        @click="emit('openSystemEditor', 'startup')"
        @dblclick="emit('openSystemEditor', 'startup')"
      >
        <el-icon class="node-icon icon-system"><IconEpEditPen /></el-icon>
        <span class="system-name">{{ t("scriptPanel.sections.startup") }}</span>
        <el-tooltip :content="t('scriptPanel.actions.edit')" placement="top">
          <el-button class="row-action" size="small" text circle @click.stop="emit('openSystemEditor', 'startup')">
            <IconEpEditPen />
          </el-button>
        </el-tooltip>
      </button>
      <button
        class="system-row"
        :class="{ 'is-selected': selectedSystemKey === 'shutdown' }"
        type="button"
        @click="emit('openSystemEditor', 'shutdown')"
        @dblclick="emit('openSystemEditor', 'shutdown')"
      >
        <el-icon class="node-icon icon-system"><IconEpEditPen /></el-icon>
        <span class="system-name">{{ t("scriptPanel.sections.shutdown") }}</span>
        <el-tooltip :content="t('scriptPanel.actions.edit')" placement="top">
          <el-button class="row-action" size="small" text circle @click.stop="emit('openSystemEditor', 'shutdown')">
            <IconEpEditPen />
          </el-button>
        </el-tooltip>
      </button>
    </div>
  </el-collapse-item>
</template>

<style scoped>
.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.section-count {
  min-width: 18px;
  height: 18px;
  padding: 0 6px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--designer-border-color) 55%, white);
  color: var(--designer-text-secondary);
  font-size: 12px;
  line-height: 18px;
  text-align: center;
  font-weight: 500;
}

.system-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 4px 4px 8px;
}

.system-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-height: 32px;
  padding: 0 8px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--designer-text-primary);
  cursor: pointer;
  text-align: left;
}

.system-row:hover,
.system-row.is-selected {
  background: var(--designer-primary-soft);
}

.node-icon {
  color: var(--designer-text-muted);
  flex-shrink: 0;
  font-size: 16px;
}

.system-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}

.row-action {
  opacity: 0;
  color: var(--designer-primary-text);
}

.system-row:hover .row-action,
.system-row.is-selected .row-action {
  opacity: 1;
}
</style>
