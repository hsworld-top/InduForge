<template>
  <header v-if="draft" class="alarm-editor-header">
    <div class="alarm-editor-header__summary">
      <div>
        <div class="alarm-editor-header__title">
          <strong>{{ draft.name || t("alarm.unnamedPolicy") }}</strong>
          <span
            class="alarm-editor-header__state"
            :class="{ 'is-off': !draft.isEnabled }"
          >
            {{ draft.isEnabled ? t("alarm.running") : t("alarm.disabledState") }}
          </span>
          <em v-if="draft.dirty">{{ t("alarm.unsaved") }}</em>
        </div>
        <code>{{ draft.description || summaryText }}</code>
      </div>
    </div>

    <div class="alarm-editor-header__fields">
      <label>
        <span>{{ t("alarm.status") }}</span>
        <input
          type="checkbox"
          :checked="draft.isEnabled"
          @change="emit('toggle')"
        />
      </label>
      <label>
        <span>{{ t("alarm.mode") }}</span>
        <select
          :value="draft.mode"
          @change="emit('update', { mode: inputValue($event) as AlarmPolicyMode, dirty: true })"
        >
          <option value="per_target">{{ t("alarm.modes.perTarget") }}</option>
          <option value="derived">{{ t("alarm.modes.derived") }}</option>
        </select>
      </label>
      <label>
        <span>{{ t("alarm.group") }}</span>
        <select :value="draft.groupId ?? ''" @change="updateGroup(inputValue($event))">
          <option value="">{{ t("alarm.root") }}</option>
          <option v-for="group in groups" :key="group.id" :value="group.id">
            {{ group.name }}
          </option>
        </select>
      </label>
    </div>

    <div class="alarm-editor-header__actions">
      <button type="button" @click="emit('checkCurrent')">
        {{ t("alarm.checkPolicy") }}
      </button>
      <button type="button" class="is-primary" :disabled="saving" @click="emit('save')">
        {{ saving ? t("alarm.saving") : t("actions.save") }}
      </button>
      <button
        type="button"
        class="is-danger"
        :disabled="deleting"
        @click="emit('delete')"
      >
        {{ t("actions.delete") }}
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type {
  AlarmPolicyGroup,
  AlarmPolicyMode,
} from "@/api/schemas/alarm.schema";
import type { AlarmPolicyDraft } from "@/components/alarm/alarmPolicyModel";
import { t } from "@/i18n/runtime";

const props = defineProps<{
  draft: AlarmPolicyDraft | null;
  groups: AlarmPolicyGroup[];
  saving: boolean;
  deleting: boolean;
}>();

const emit = defineEmits<{
  save: [];
  toggle: [];
  delete: [];
  checkCurrent: [];
  update: [patch: Partial<AlarmPolicyDraft>];
}>();

const summaryText = computed(() => {
  if (!props.draft) {
    return t("alarm.selectPolicyHint");
  }
  if (props.draft.mode === "derived") {
    return props.draft.derivedExpression || t("alarm.modes.derived");
  }
  return props.draft.targets.map((target) => target.path).join("、") || t("alarm.modes.perTarget");
});

const inputValue = (event: Event) =>
  (event.target as HTMLInputElement | HTMLSelectElement).value;

const updateGroup = (value: string) => {
  emit("update", { groupId: value || null, dirty: true });
};
</script>

<style scoped>
.alarm-editor-header {
  min-height: 120px;
  display: grid;
  grid-template-columns: minmax(260px, 1fr) minmax(320px, 440px) auto;
  align-items: start;
  justify-content: space-between;
  gap: 16px;
  padding: 14px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.alarm-editor-header__summary {
  min-width: 0;
}

.alarm-editor-header__title {
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
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
  font-size: 18px;
}

.alarm-editor-header__summary code {
  margin-top: 4px;
  color: var(--dc-text-secondary);
  font-family: var(--dc-font-mono);
  font-size: 12px;
}

.alarm-editor-header__summary em,
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

.alarm-editor-header__summary em {
  border: 1px solid rgba(217, 119, 6, 0.26);
  background: rgba(217, 119, 6, 0.08);
  color: #b45309;
}

.alarm-editor-header__fields {
  display: grid;
  grid-template-columns: auto minmax(120px, 1fr) minmax(120px, 1fr);
  gap: 10px;
}

.alarm-editor-header__fields label {
  min-width: 0;
  display: grid;
  gap: 6px;
}

.alarm-editor-header__fields label > span {
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.alarm-editor-header__fields input[type="checkbox"] {
  width: 34px;
  height: 34px;
  margin: 0;
  accent-color: var(--dc-primary);
}

.alarm-editor-header__fields select {
  width: 100%;
  height: 34px;
  min-width: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  font-family: inherit;
  font-size: 13px;
  outline: none;
  padding: 0 9px;
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

.alarm-editor-header__actions .is-primary {
  border-color: var(--dc-primary);
  background: var(--dc-primary);
  color: #fff;
}

.alarm-editor-header__actions .is-primary:hover:not(:disabled) {
  background: var(--dc-primary);
  color: #fff;
}

.alarm-editor-header__actions button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.alarm-editor-header__actions .is-danger {
  border-color: rgba(220, 38, 38, 0.28);
  color: var(--dc-danger, #b91c1c);
}

@media (max-width: 1120px) {
  .alarm-editor-header {
    grid-template-columns: 1fr;
  }

  .alarm-editor-header__fields {
    grid-template-columns: auto repeat(2, minmax(0, 1fr));
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
