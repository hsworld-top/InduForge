<template>
  <DcDialog
    v-model="visible"
    title="批量设置条件"
    width="860px"
    body-max-height="calc(100vh - 220px)"
    @close="reset"
  >
    <AlarmConditionMatrix :conditions="conditions" @update="conditions = $event" />
    <p v-if="localError || error" class="alarm-bulk-condition__error">
      {{ localError || error }}
    </p>
    <template #footer>
      <div class="alarm-bulk-condition__footer">
        <button type="button" class="is-ghost" @click="visible = false">取消</button>
        <button type="button" :disabled="submitting" @click="submit">
          {{ submitting ? "应用中" : "应用到已选策略" }}
        </button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import type { AlarmCondition } from "@/api/schemas/alarm.schema";
import DcDialog from "@/components/shared/DcDialog.vue";
import AlarmConditionMatrix from "./AlarmConditionMatrix.vue";

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
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
  submit: [conditions: AlarmCondition[]];
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit("update:modelValue", value),
});

const conditions = ref<AlarmCondition[]>([]);
const localError = ref("");

const reset = () => {
  conditions.value = [];
  localError.value = "";
};

const submit = () => {
  localError.value = "";
  if (!conditions.value.length) {
    localError.value = "请至少配置一个条件";
    return;
  }
  emit("submit", conditions.value);
};
</script>

<style scoped>
.alarm-bulk-condition__error {
  margin: 12px 0 0;
  padding: 9px 10px;
  border: 1px solid rgba(220, 38, 38, 0.28);
  border-radius: var(--dc-radius-sm);
  background: rgba(220, 38, 38, 0.06);
  color: var(--dc-danger, #b91c1c);
  font-size: 13px;
}

.alarm-bulk-condition__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.alarm-bulk-condition__footer button {
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

.alarm-bulk-condition__footer button.is-ghost {
  border-color: var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.alarm-bulk-condition__footer button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}
</style>
