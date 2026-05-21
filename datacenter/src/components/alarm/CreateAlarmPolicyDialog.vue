<template>
  <DcDialog
    v-model="visible"
    :title="t('alarm.createPolicy')"
    width="520px"
    body-max-height="calc(100vh - 220px)"
    @close="handleClose"
  >
    <form class="create-alarm-policy" @submit.prevent="submit">
      <label>
        <span>{{ t("alarm.name") }}</span>
        <input v-model="draft.name" type="text" :placeholder="t('alarm.namePlaceholder')" />
      </label>
      <label>
        <span>{{ t("alarm.group") }}</span>
        <select v-model="groupValue">
          <option value="">{{ t("alarm.root") }}</option>
          <option v-for="group in groups" :key="group.id" :value="group.id">
            {{ group.name }}
          </option>
        </select>
      </label>
      <section class="create-alarm-policy__mode is-wide">
        <span>{{ t("alarm.mode") }}</span>
        <div class="create-alarm-policy__mode-options">
          <button
            v-for="item in modeOptions"
            :key="item.value"
            type="button"
            :class="{ 'is-active': draft.mode === item.value }"
            :title="item.tooltip"
            @click="draft.mode = item.value"
          >
            <strong>{{ item.label }}</strong>
            <small>{{ item.description }}</small>
          </button>
        </div>
      </section>
      <label class="is-wide">
        <span>{{ t("alarm.description") }}</span>
        <textarea v-model="draft.description" rows="4" :placeholder="t('alarm.descriptionPlaceholder')" />
      </label>
    </form>

    <p v-if="localError || error" class="create-alarm-policy__error">
      {{ localError || error }}
    </p>

    <template #footer>
      <div class="create-alarm-policy__footer">
        <button type="button" class="is-ghost" @click="close">{{ t("actions.cancel") }}</button>
        <button type="button" :disabled="submitting" @click="submit">
          {{ submitting ? t("alarm.creating") : t("actions.create") }}
        </button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import type {
  AlarmPolicyGroup,
  AlarmPolicyMode,
} from "@/api/schemas/alarm.schema";
import DcDialog from "@/components/shared/DcDialog.vue";
import {
  createDefaultAlarmPolicyDraft,
  type AlarmPolicyDraft,
} from "@/components/alarm/alarmPolicyModel";
import { t } from "@/i18n/runtime";

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    groups: AlarmPolicyGroup[];
    submitting?: boolean;
    error?: string;
  }>(),
  {
    submitting: false,
    error: "",
  },
);

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  submit: [draft: AlarmPolicyDraft];
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit("update:modelValue", value),
});

const draft = reactive<AlarmPolicyDraft>(createDefaultAlarmPolicyDraft());
const localError = ref("");
const groupValue = computed({
  get: () => draft.groupId ?? "",
  set: (value: string) => {
    draft.groupId = value || null;
  },
});

const modeOptions: Array<{
  value: AlarmPolicyMode;
  label: string;
  description: string;
  tooltip: string;
}> = [
  {
    value: "per_target",
    label: "统一模板报警",
    description: "一套条件应用到目标点",
    tooltip: "一套报警条件会分别应用到每个目标点。多个目标点不会合并计算。",
  },
  {
    value: "derived",
    label: "计算结果报警",
    description: "先计算，再对结果报警",
    tooltip: "选择多个输入点并编写计算表达式，条件集判断的是计算结果。",
  },
];

const reset = () => {
  Object.assign(draft, createDefaultAlarmPolicyDraft());
  localError.value = "";
};

const close = () => {
  visible.value = false;
};

const handleClose = () => {
  reset();
};

const submit = () => {
  localError.value = "";
  if (!draft.name.trim()) {
    localError.value = t("alarm.nameRequired");
    return;
  }
  emit("submit", {
    ...draft,
    name: draft.name.trim(),
    isEnabled: false,
    dirty: true,
  });
};

watch(
  () => props.modelValue,
  (next) => {
    if (next) {
      reset();
    }
  },
);
</script>

<style scoped>
.create-alarm-policy {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.create-alarm-policy label {
  min-width: 0;
  display: grid;
  gap: 6px;
}

.create-alarm-policy label.is-wide {
  grid-column: 1 / -1;
}

.create-alarm-policy label > span,
.create-alarm-policy__mode > span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.create-alarm-policy__mode {
  min-width: 0;
  display: grid;
  gap: 6px;
}

.create-alarm-policy__mode.is-wide {
  grid-column: 1 / -1;
}

.create-alarm-policy__mode-options {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.create-alarm-policy__mode-options button {
  min-width: 0;
  display: grid;
  gap: 4px;
  padding: 10px 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
  cursor: pointer;
  font-family: inherit;
  text-align: left;
}

.create-alarm-policy__mode-options button:hover {
  border-color: rgba(37, 99, 235, 0.32);
  background: var(--dc-primary-soft);
}

.create-alarm-policy__mode-options button.is-active {
  border-color: var(--dc-primary);
  background: rgba(37, 99, 235, 0.08);
  box-shadow: inset 0 0 0 1px rgba(37, 99, 235, 0.08);
}

.create-alarm-policy__mode-options strong,
.create-alarm-policy__mode-options small {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.create-alarm-policy__mode-options strong {
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 800;
}

.create-alarm-policy__mode-options small {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.create-alarm-policy input,
.create-alarm-policy select,
.create-alarm-policy textarea {
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

.create-alarm-policy input,
.create-alarm-policy select {
  height: 34px;
  padding: 0 9px;
}

.create-alarm-policy textarea {
  padding: 9px;
  line-height: 1.5;
  resize: vertical;
}

.create-alarm-policy__error {
  margin: 12px 0 0;
  padding: 9px 10px;
  border: 1px solid rgba(220, 38, 38, 0.28);
  border-radius: var(--dc-radius-sm);
  background: rgba(220, 38, 38, 0.06);
  color: var(--dc-danger, #b91c1c);
  font-size: 13px;
}

.create-alarm-policy__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.create-alarm-policy__footer button {
  height: 32px;
  padding: 0 14px;
  border: 1px solid var(--dc-primary);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary);
  color: #fff;
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  font-weight: 700;
}

.create-alarm-policy__footer button.is-ghost {
  border-color: var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.create-alarm-policy__footer button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

@media (max-width: 620px) {
  .create-alarm-policy {
    grid-template-columns: 1fr;
  }

  .create-alarm-policy__mode-options {
    grid-template-columns: 1fr;
  }
}
</style>
