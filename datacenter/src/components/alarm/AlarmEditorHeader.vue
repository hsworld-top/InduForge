<template>
  <header class="alarm-editor-header">
    <div class="alarm-editor-header__summary">
      <span class="alarm-editor-header__state" :class="{ 'is-off': !draft?.isEnabled }">
        {{ draft?.isEnabled ? "运行中" : "已停用" }}
      </span>
      <div>
        <strong>{{ draft?.name || "未选择报警规则" }}</strong>
        <code>{{ draft?.targetPath || "从左侧选择规则" }}</code>
      </div>
      <em v-if="draft?.dirty">未保存</em>
    </div>

    <div class="alarm-editor-header__actions">
      <button type="button" :disabled="!draft || saving" @click="emit('save')">
        保存
      </button>
      <button type="button" :disabled="!draft || saving" @click="emit('toggle')">
        {{ draft?.isEnabled ? "停用" : "启用" }}
      </button>
      <button
        type="button"
        class="is-danger"
        :disabled="!draft || deleting"
        @click="emit('delete')"
      >
        删除
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import type { AlarmRuleDraft } from "@/components/alarm/alarmRuleModel";

defineProps<{
  draft: AlarmRuleDraft | null;
  saving: boolean;
  deleting: boolean;
}>();

const emit = defineEmits<{
  save: [];
  toggle: [];
  delete: [];
}>();
</script>

<style scoped>
.alarm-editor-header {
  min-height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.alarm-editor-header__summary {
  min-width: 0;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
}

.alarm-editor-header__summary strong,
.alarm-editor-header__summary code {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alarm-editor-header__summary strong {
  color: var(--dc-text);
  font-size: 14px;
}

.alarm-editor-header__summary code {
  margin-top: 4px;
  color: var(--dc-text-secondary);
  font-family: var(--dc-font-mono);
  font-size: 12px;
}

.alarm-editor-header__summary em {
  height: 22px;
  display: inline-flex;
  align-items: center;
  padding: 0 8px;
  border: 1px solid rgba(217, 119, 6, 0.26);
  border-radius: var(--dc-radius-xs);
  background: rgba(217, 119, 6, 0.08);
  color: #b45309;
  font-size: 12px;
  font-style: normal;
  font-weight: 700;
  white-space: nowrap;
}

.alarm-editor-header__state {
  width: 8px;
  height: 38px;
  overflow: hidden;
  border-radius: 999px;
  background: #16a34a;
  color: transparent;
}

.alarm-editor-header__state.is-off {
  background: var(--dc-text-muted);
}

.alarm-editor-header__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.alarm-editor-header__actions button {
  height: 32px;
  padding: 0 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  font-weight: 700;
}

.alarm-editor-header__actions button:hover:not(:disabled) {
  border-color: rgba(37, 99, 235, 0.26);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.alarm-editor-header__actions button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.alarm-editor-header__actions .is-danger {
  border-color: rgba(220, 38, 38, 0.28);
  color: var(--dc-danger, #b91c1c);
}

@media (max-width: 720px) {
  .alarm-editor-header {
    align-items: stretch;
    flex-direction: column;
  }

  .alarm-editor-header__actions {
    justify-content: flex-end;
  }
}
</style>
