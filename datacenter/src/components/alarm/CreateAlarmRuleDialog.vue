<template>
  <DcDialog
    v-model="visible"
    title="新建报警规则"
    width="760px"
    body-max-height="calc(100vh - 220px)"
    @close="handleClose"
  >
    <form class="create-alarm-rule" @submit.prevent="submit">
      <label class="create-alarm-rule__field">
        <span>规则名</span>
        <input v-model.trim="draft.name" type="text" placeholder="例如：炉温 HH" />
      </label>

      <label class="create-alarm-rule__field">
        <span>规则类型</span>
        <select v-model="draft.ruleType">
          <option
            v-for="item in alarmRuleTypeOptions"
            :key="item.value"
            :value="item.value"
          >
            {{ item.label }}
          </option>
        </select>
      </label>

      <div class="create-alarm-rule__field is-wide">
        <span>目标数据点</span>
        <DataPointPicker
          v-model="selectedDatapointId"
          :project-id="projectId"
          @select="handleDatapointSelect"
        />
      </div>

      <label class="create-alarm-rule__field">
        <span>严重度</span>
        <select v-model="draft.severity">
          <option
            v-for="item in alarmSeverityOptions"
            :key="item.value"
            :value="item.value"
          >
            {{ item.label }}
          </option>
        </select>
      </label>

      <label class="create-alarm-rule__field">
        <span>{{ conditionLabel }}</span>
        <input
          v-if="draft.ruleType !== 'cel'"
          v-model.trim="thresholdInput"
          type="number"
          step="any"
          placeholder="输入阈值"
        />
        <textarea
          v-else
          v-model.trim="expression"
          rows="3"
          placeholder="输入 CEL 条件"
        />
      </label>

      <label class="create-alarm-rule__switch">
        <input v-model="draft.isEnabled" type="checkbox" />
        <span>创建后启用</span>
      </label>

      <label class="create-alarm-rule__field is-wide">
        <span>描述</span>
        <textarea v-model.trim="draft.description" rows="3" placeholder="可选" />
      </label>

      <p v-if="localError || error" class="create-alarm-rule__error">
        {{ localError || error }}
      </p>
    </form>

    <template #footer>
      <div class="create-alarm-rule__footer">
        <button type="button" class="is-ghost" @click="close">取消</button>
        <button type="button" :disabled="submitting" @click="submit">
          {{ submitting ? "创建中" : "创建" }}
        </button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import type { Datapoint } from "@/api/schemas/datapoint.schema";
import type { AlarmRuleSave } from "@/api/schemas/alarm.schema";
import DcDialog from "@/components/shared/DcDialog.vue";
import {
  alarmRuleTypeOptions,
  alarmSeverityOptions,
  createAlarmDraft,
} from "@/components/alarm/alarmRuleModel";
import DataPointPicker from "./DataPointPicker.vue";

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    projectId: string;
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
  submit: [payload: AlarmRuleSave];
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit("update:modelValue", value),
});

const draft = reactive(createAlarmDraft());
const selectedDatapointId = ref("");
const selectedDatapoint = ref<Datapoint | null>(null);
const thresholdInput = ref("");
const expression = ref("");
const localError = ref("");

const conditionLabel = computed(() =>
  draft.ruleType === "cel" ? "初始条件" : "初始阈值",
);

const reset = () => {
  Object.assign(draft, createAlarmDraft());
  selectedDatapointId.value = "";
  selectedDatapoint.value = null;
  thresholdInput.value = "";
  expression.value = "";
  localError.value = "";
};

const close = () => {
  visible.value = false;
};

const handleClose = () => {
  reset();
};

const handleDatapointSelect = (datapoint: Datapoint) => {
  selectedDatapoint.value = datapoint;
  draft.targetDatapointId = datapoint.id;
  draft.targetPath = datapoint.path;
  draft.targetName = datapoint.name;
  draft.targetDataType = datapoint.dataType;
};

const buildCondition = () => {
  if (draft.ruleType === "cel") {
    return { expression: expression.value };
  }
  const condition: Record<string, unknown> = { limit: Number(thresholdInput.value) };
  if (draft.ruleType === "rate_of_change") {
    condition.windowMs = 60000;
  }
  return condition;
};

const submit = () => {
  localError.value = "";
  if (!draft.name) {
    localError.value = "请输入规则名";
    return;
  }
  if (!selectedDatapoint.value) {
    localError.value = "请选择目标数据点";
    return;
  }
  if (draft.ruleType === "cel" && !expression.value) {
    localError.value = "请输入 CEL 条件";
    return;
  }
  if (
    draft.ruleType !== "cel" &&
    (thresholdInput.value === "" || !Number.isFinite(Number(thresholdInput.value)))
  ) {
    localError.value = "请输入初始阈值";
    return;
  }

  emit("submit", {
    name: draft.name,
    description: draft.description ?? "",
    targetDatapointId: selectedDatapoint.value.id,
    targetPath: selectedDatapoint.value.path,
    targetName: selectedDatapoint.value.name,
    targetDataType: selectedDatapoint.value.dataType,
    ruleType: draft.ruleType,
    condition: buildCondition(),
    severity: draft.severity,
    isEnabled: draft.isEnabled,
    suppression: { enabled: false },
    messageTemplate: "",
  } as AlarmRuleSave);
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
.create-alarm-rule {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  padding: 2px 2px 8px;
}

.create-alarm-rule__field,
.create-alarm-rule__switch {
  display: grid;
  gap: 6px;
}

.create-alarm-rule__field.is-wide {
  grid-column: 1 / -1;
}

.create-alarm-rule__field > span,
.create-alarm-rule__switch span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.create-alarm-rule__field input,
.create-alarm-rule__field select,
.create-alarm-rule__field textarea {
  width: 100%;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  font-family: inherit;
  font-size: 13px;
  outline: none;
}

.create-alarm-rule__field input,
.create-alarm-rule__field select {
  height: 34px;
  padding: 0 10px;
}

.create-alarm-rule__field textarea {
  padding: 9px 10px;
  resize: vertical;
}

.create-alarm-rule__field input:focus,
.create-alarm-rule__field select:focus,
.create-alarm-rule__field textarea:focus {
  border-color: var(--dc-primary);
  box-shadow: 0 0 0 3px var(--dc-primary-soft);
}

.create-alarm-rule__switch {
  align-content: end;
  grid-template-columns: auto 1fr;
  align-items: center;
  gap: 8px;
  min-height: 34px;
}

.create-alarm-rule__switch input {
  width: 15px;
  height: 15px;
  margin: 0;
  accent-color: var(--dc-primary);
}

.create-alarm-rule__error {
  grid-column: 1 / -1;
  margin: 0;
  padding: 9px 10px;
  border: 1px solid rgba(220, 38, 38, 0.28);
  border-radius: var(--dc-radius-sm);
  background: rgba(220, 38, 38, 0.06);
  color: var(--dc-danger, #b91c1c);
  font-size: 13px;
}

.create-alarm-rule__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.create-alarm-rule__footer button {
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

.create-alarm-rule__footer button.is-ghost {
  border-color: var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.create-alarm-rule__footer button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

@media (max-width: 720px) {
  .create-alarm-rule {
    grid-template-columns: 1fr;
  }
}
</style>
