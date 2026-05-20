<template>
  <form class="alarm-rule-form" @submit.prevent>
    <section class="alarm-rule-form__section">
      <header>
        <strong>基础信息</strong>
        <span>规则身份与目标点位</span>
      </header>
      <div class="alarm-rule-form__grid">
        <label>
          <span>规则名</span>
          <input
            :value="draft.name"
            type="text"
            @input="updateField('name', inputValue($event))"
          />
        </label>
        <label>
          <span>目标路径</span>
          <input
            :value="draft.targetPath"
            type="text"
            @input="updateField('targetPath', inputValue($event))"
          />
        </label>
        <label>
          <span>规则类型</span>
          <select
            :value="draft.ruleType"
            @change="changeRuleType(inputValue($event) as AlarmRuleType)"
          >
            <option
              v-for="item in alarmRuleTypeOptions"
              :key="item.value"
              :value="item.value"
            >
              {{ item.label }}
            </option>
          </select>
        </label>
        <label>
          <span>严重度</span>
          <select
            :value="draft.severity"
            @change="updateField('severity', inputValue($event) as AlarmSeverity)"
          >
            <option
              v-for="item in alarmSeverityOptions"
              :key="item.value"
              :value="item.value"
            >
              {{ item.label }}
            </option>
          </select>
        </label>
        <label class="is-wide">
          <span>描述</span>
          <textarea
            :value="draft.description ?? ''"
            rows="3"
            @input="updateField('description', inputValue($event))"
          />
        </label>
      </div>
    </section>

    <section class="alarm-rule-form__section">
      <header>
        <strong>触发条件</strong>
        <span>{{ conditionHint }}</span>
      </header>

      <div v-if="draft.ruleType === 'cel'" class="alarm-rule-form__grid">
        <label class="is-wide">
          <span>CEL 表达式</span>
          <textarea
            :value="stringCondition('expression')"
            class="alarm-rule-form__code"
            rows="7"
            spellcheck="false"
            @input="updateCondition('expression', inputValue($event))"
          />
        </label>
      </div>

      <div v-else class="alarm-rule-form__grid">
        <label>
          <span>limit</span>
          <input
            :value="numberText('limit')"
            type="number"
            step="any"
            @input="updateNumberCondition('limit', inputValue($event))"
          />
        </label>

        <template v-if="draft.ruleType === 'rate_of_change'">
          <label>
            <span>windowMs</span>
            <input
              :value="numberText('windowMs')"
              type="number"
              min="0"
              step="1"
              @input="updateNumberCondition('windowMs', inputValue($event))"
            />
          </label>
          <label>
            <span>direction</span>
            <select
              :value="stringCondition('direction') || 'up'"
              @change="updateCondition('direction', inputValue($event))"
            >
              <option value="up">up</option>
              <option value="down">down</option>
            </select>
          </label>
        </template>

        <template v-else>
          <label>
            <span>hysteresis</span>
            <input
              :value="numberText('hysteresis')"
              type="number"
              step="any"
              placeholder="可选"
              @input="updateOptionalNumberCondition('hysteresis', inputValue($event))"
            />
          </label>
          <label>
            <span>durationMs</span>
            <input
              :value="numberText('durationMs')"
              type="number"
              min="0"
              step="1"
              placeholder="可选"
              @input="updateOptionalNumberCondition('durationMs', inputValue($event))"
            />
          </label>
        </template>
      </div>
    </section>
  </form>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type {
  AlarmRuleType,
  AlarmSeverity,
} from "@/api/schemas/alarm.schema";
import {
  alarmRuleTypeOptions,
  alarmSeverityOptions,
  type AlarmCondition,
  type AlarmRuleDraft,
} from "@/components/alarm/alarmRuleModel";

const props = defineProps<{
  draft: AlarmRuleDraft;
}>();

const emit = defineEmits<{
  update: [patch: Partial<AlarmRuleDraft>];
}>();

const conditionHint = computed(() => {
  if (props.draft.ruleType === "cel") {
    return "使用 condition.expression";
  }
  if (props.draft.ruleType === "rate_of_change") {
    return "使用 condition.limit / windowMs / direction";
  }
  return "使用 condition.limit，可选 hysteresis / durationMs";
});

const inputValue = (event: Event) => (event.target as HTMLInputElement).value;

const updateField = <K extends keyof AlarmRuleDraft>(
  field: K,
  value: AlarmRuleDraft[K],
) => {
  emit("update", { [field]: value, dirty: true } as Partial<AlarmRuleDraft>);
};

const cleanConditionForType = (ruleType: AlarmRuleType): AlarmCondition => {
  if (ruleType === "cel") {
    return { expression: "" };
  }
  if (ruleType === "rate_of_change") {
    return { limit: undefined, windowMs: 60000, direction: "up" };
  }
  return { limit: undefined };
};

const changeRuleType = (ruleType: AlarmRuleType) => {
  emit("update", {
    ruleType,
    condition: cleanConditionForType(ruleType),
    dirty: true,
  });
};

const updateCondition = (field: string, value: unknown) => {
  emit("update", {
    condition: { ...props.draft.condition, [field]: value },
    dirty: true,
  });
};

const updateNumberCondition = (field: string, value: string) => {
  updateCondition(field, value === "" ? undefined : Number(value));
};

const updateOptionalNumberCondition = (field: string, value: string) => {
  const nextCondition = { ...props.draft.condition };
  if (value === "") {
    delete nextCondition[field];
  } else {
    nextCondition[field] = Number(value);
  }
  emit("update", { condition: nextCondition, dirty: true });
};

const numberText = (field: string) => {
  const value = props.draft.condition[field];
  return typeof value === "number" && Number.isFinite(value) ? String(value) : "";
};

const stringCondition = (field: string) => {
  const value = props.draft.condition[field];
  return typeof value === "string" ? value : "";
};
</script>

<style scoped>
.alarm-rule-form {
  display: grid;
  gap: 12px;
}

.alarm-rule-form__section {
  display: grid;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.alarm-rule-form__section header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}

.alarm-rule-form__section strong {
  color: var(--dc-text);
  font-size: 13px;
}

.alarm-rule-form__section header span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.alarm-rule-form__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.alarm-rule-form label {
  min-width: 0;
  display: grid;
  gap: 5px;
}

.alarm-rule-form label.is-wide {
  grid-column: 1 / -1;
}

.alarm-rule-form label > span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.alarm-rule-form input,
.alarm-rule-form select,
.alarm-rule-form textarea {
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

.alarm-rule-form input,
.alarm-rule-form select {
  height: 34px;
  padding: 0 9px;
}

.alarm-rule-form textarea {
  padding: 9px;
  line-height: 1.5;
  resize: vertical;
}

.alarm-rule-form input:focus,
.alarm-rule-form select:focus,
.alarm-rule-form textarea:focus {
  border-color: var(--dc-primary);
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.1);
}

.alarm-rule-form__code {
  font-family: var(--dc-font-mono);
  font-size: 12px;
}

@media (max-width: 720px) {
  .alarm-rule-form__grid {
    grid-template-columns: 1fr;
  }
}
</style>
