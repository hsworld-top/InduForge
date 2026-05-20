<template>
  <form class="alarm-policy-form" @submit.prevent>
    <section class="alarm-policy-form__section">
      <header>
        <strong>{{ draft.mode === "derived" ? "输入点" : "目标点" }}</strong>
        <span>{{ draft.mode === "derived" ? "用于表达式计算" : "同一条件集应用到这些点" }}</span>
      </header>
      <AlarmPointPicker
        :project-id="projectId"
        :items="draft.mode === 'derived' ? draft.inputs : draft.targets"
        :with-key="draft.mode === 'derived'"
        @update="updatePoints"
      />
    </section>

    <section v-if="draft.mode === 'derived'" class="alarm-policy-form__section">
      <header>
        <strong>计算表达式</strong>
        <span>支持输入点变量与 + - * /</span>
      </header>
      <textarea
        :value="draft.derivedExpression"
        class="alarm-policy-form__code"
        rows="4"
        spellcheck="false"
        placeholder="例如：(tempA + tempB) / 2"
        @input="updateField('derivedExpression', inputValue($event))"
      />
    </section>

    <AlarmConditionMatrix
      :conditions="draft.conditions"
      @update="updateField('conditions', $event)"
    />
  </form>
</template>

<script setup lang="ts">
import type {
  AlarmInputRef,
  AlarmTargetRef,
} from "@/api/schemas/alarm.schema";
import type { AlarmPolicyDraft } from "@/components/alarm/alarmPolicyModel";
import AlarmConditionMatrix from "./AlarmConditionMatrix.vue";
import AlarmPointPicker from "./AlarmPointPicker.vue";

const props = defineProps<{
  projectId: string;
  draft: AlarmPolicyDraft;
}>();

const emit = defineEmits<{
  update: [patch: Partial<AlarmPolicyDraft>];
}>();

const inputValue = (event: Event) =>
  (event.target as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement).value;

const updateField = <K extends keyof AlarmPolicyDraft>(
  field: K,
  value: AlarmPolicyDraft[K],
) => {
  emit("update", { [field]: value, dirty: true } as Partial<AlarmPolicyDraft>);
};

const updatePoints = (items: Array<AlarmInputRef | AlarmTargetRef>) => {
  if (props.draft.mode === "derived") {
    updateField("inputs", items as AlarmInputRef[]);
    return;
  }
  updateField("targets", items as AlarmTargetRef[]);
};
</script>

<style scoped>
.alarm-policy-form {
  display: grid;
  gap: 14px;
}

.alarm-policy-form__section {
  display: grid;
  gap: 10px;
  padding: 14px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.alarm-policy-form__section header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}

.alarm-policy-form__section strong {
  color: var(--dc-text);
  font-size: 13px;
}

.alarm-policy-form__section header span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.alarm-policy-form__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.alarm-policy-form label {
  min-width: 0;
  display: grid;
  gap: 5px;
}

.alarm-policy-form label.is-wide {
  grid-column: 1 / -1;
}

.alarm-policy-form label.is-switch {
  grid-template-columns: auto 1fr;
  align-items: center;
  align-content: end;
  gap: 8px;
  min-height: 34px;
}

.alarm-policy-form label > span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.alarm-policy-form input,
.alarm-policy-form select,
.alarm-policy-form textarea {
  width: 100%;
  min-width: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  font-family: inherit;
  font-size: 13px;
  outline: none;
}

.alarm-policy-form input:focus,
.alarm-policy-form select:focus,
.alarm-policy-form textarea:focus {
  border-color: var(--dc-primary);
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.1);
}

.alarm-policy-form input,
.alarm-policy-form select {
  height: 34px;
  padding: 0 9px;
}

.alarm-policy-form input[type="checkbox"] {
  width: 15px;
  height: 15px;
  padding: 0;
  accent-color: var(--dc-primary);
}

.alarm-policy-form textarea {
  padding: 9px;
  line-height: 1.5;
  resize: vertical;
}

.alarm-policy-form__code {
  font-family: var(--dc-font-mono);
  font-size: 12px;
}

@media (max-width: 720px) {
  .alarm-policy-form__grid {
    grid-template-columns: 1fr;
  }
}
</style>
