<template>
  <header v-if="draft" class="alarm-editor-header">
    <div class="alarm-editor-header__title-wrap">
      <div class="alarm-editor-header__meta">
        <span
          class="alarm-editor-header__state"
          :class="{ 'is-off': !draft.isEnabled }"
        >
          {{ draft.isEnabled ? t("common.enabled") : t("alarm.stopped") }}
        </span>
        <em v-if="draft.dirty">{{ t("alarm.unsaved") }}</em>
      </div>
      <strong>{{ draft.name || t("alarm.unnamedPolicy") }}</strong>
      <div class="alarm-editor-header__subline">
        <span class="alarm-editor-header__mode">{{ modeText }}</span>
        <code>{{ draft.description || summaryText }}</code>
      </div>
    </div>

    <div class="alarm-editor-header__actions">
      <button
        v-if="draft.mode === 'per_target'"
        type="button"
        class="alarm-editor-header__icon-action"
        title="数据点变量"
        aria-label="数据点变量"
        :disabled="saving"
        @click="emit('selectTarget')"
      >
        <IconTablerDatabaseImport class="alarm-editor-header__action-icon" />
      </button>
      <button
        type="button"
        class="alarm-editor-header__icon-action"
        :title="draft.isEnabled ? t('alarm.stopped') : t('common.enabled')"
        :aria-label="draft.isEnabled ? t('alarm.stopped') : t('common.enabled')"
        :disabled="saving"
        @click="emit('toggle')"
      >
        <IconTablerPlayerPause
          v-if="draft.isEnabled"
          class="alarm-editor-header__action-icon"
        />
        <IconTablerPlayerPlay v-else class="alarm-editor-header__action-icon" />
      </button>
      <button
        type="button"
        class="alarm-editor-header__icon-action"
        :title="t('alarm.checkPolicy')"
        :aria-label="t('alarm.checkPolicy')"
        @click="emit('checkCurrent')"
      >
        <IconTablerShieldCheck class="alarm-editor-header__action-icon" />
      </button>
      <button
        type="button"
        class="alarm-editor-header__icon-action is-danger"
        :title="t('actions.delete')"
        :aria-label="t('actions.delete')"
        :disabled="deleting"
        @click="emit('delete')"
      >
        <IconTablerTrash class="alarm-editor-header__action-icon" />
      </button>
      <button
        type="button"
        class="alarm-editor-header__save"
        :title="t('actions.save')"
        :aria-label="t('actions.save')"
        :disabled="saving || !draft.dirty"
        @click="emit('save')"
      >
        <IconTablerDeviceFloppy class="alarm-editor-header__action-icon" />
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from "vue";
import IconTablerDatabaseImport from "~icons/tabler/database-import";
import IconTablerDeviceFloppy from "~icons/tabler/device-floppy";
import IconTablerPlayerPause from "~icons/tabler/player-pause";
import IconTablerPlayerPlay from "~icons/tabler/player-play";
import IconTablerShieldCheck from "~icons/tabler/shield-check";
import IconTablerTrash from "~icons/tabler/trash";
import type { AlarmPolicyDraft } from "@/components/alarm/alarmPolicyModel";
import { t } from "@/i18n/runtime";

const props = defineProps<{
  draft: AlarmPolicyDraft | null;
  saving: boolean;
  deleting: boolean;
}>();

const emit = defineEmits<{
  save: [];
  toggle: [];
  delete: [];
  checkCurrent: [];
  selectTarget: [];
  update: [patch: Partial<AlarmPolicyDraft>];
}>();

const modeText = computed(() => {
  if (!props.draft) {
    return "";
  }
  return props.draft.mode === "derived"
    ? t("alarm.modes.derived")
    : t("alarm.modes.perTarget");
});

const summaryText = computed(() => {
  if (!props.draft) {
    return t("alarm.selectPolicyHint");
  }
  if (props.draft.mode === "derived") {
    return props.draft.derivedExpression
      ? `计算：${props.draft.derivedExpression}`
      : "等待配置计算表达式";
  }
  return props.draft.targets.length
    ? `${props.draft.targets.length} 个目标点共用条件`
    : "等待选择目标点";
});
</script>

<style scoped>
.alarm-editor-header {
  min-height: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface);
}

.alarm-editor-header__title-wrap {
  min-width: 0;
  display: grid;
  grid-template-columns: auto minmax(150px, 340px) minmax(0, 1fr);
  align-items: center;
  gap: 8px;
}

.alarm-editor-header__meta,
.alarm-editor-header__subline {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.alarm-editor-header__title-wrap strong,
.alarm-editor-header__subline code,
.alarm-editor-header__subline span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alarm-editor-header__title-wrap strong {
  color: var(--dc-text);
  font-size: 14px;
  font-weight: 800;
  line-height: 1.3;
}

.alarm-editor-header__subline code,
.alarm-editor-header__subline span {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.alarm-editor-header__subline code {
  font-family: var(--dc-font-mono);
}

.alarm-editor-header__mode {
  padding: 2px 6px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  font-weight: 800;
}

.alarm-editor-header__meta em,
.alarm-editor-header__state {
  height: 22px;
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  font-size: 12px;
  font-style: normal;
  font-weight: 700;
  padding: 0 8px;
  white-space: nowrap;
}

.alarm-editor-header__state {
  border: 1px solid rgba(22, 163, 74, 0.24);
  background: rgba(22, 163, 74, 0.1);
  color: #15803d;
}

.alarm-editor-header__state.is-off {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
}

.alarm-editor-header__meta em {
  border: 1px solid rgba(217, 119, 6, 0.26);
  background: rgba(217, 119, 6, 0.08);
  color: #b45309;
}

.alarm-editor-header__actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.alarm-editor-header__icon-action,
.alarm-editor-header__save {
  width: 30px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--dc-radius-sm);
  cursor: pointer;
  font-weight: 700;
}

.alarm-editor-header__icon-action {
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.alarm-editor-header__icon-action:hover:not(:disabled) {
  border-color: rgba(37, 99, 235, 0.26);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.alarm-editor-header__icon-action.is-danger {
  color: var(--dc-danger, #b91c1c);
}

.alarm-editor-header__save {
  border-color: var(--dc-primary);
  background: var(--dc-primary);
  color: #fff;
}

.alarm-editor-header__actions button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.alarm-editor-header__action-icon {
  width: 16px;
  height: 16px;
}

@media (max-width: 1120px) {
  .alarm-editor-header {
    align-items: flex-start;
    flex-direction: column;
  }
}

@media (max-width: 720px) {
  .alarm-editor-header {
    align-items: stretch;
  }

  .alarm-editor-header__actions {
    justify-content: flex-end;
  }

  .alarm-editor-header__fields {
    grid-template-columns: 1fr;
  }
}
</style>
