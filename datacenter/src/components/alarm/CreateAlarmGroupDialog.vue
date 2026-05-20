<template>
  <DcDialog
    v-model="visible"
    title="新建报警分组"
    width="460px"
    body-max-height="calc(100vh - 220px)"
    @close="reset"
  >
    <form class="create-alarm-group" @submit.prevent="submit">
      <label>
        <span>分组名</span>
        <input v-model.trim="name" type="text" />
      </label>
      <label>
        <span>描述</span>
        <textarea v-model.trim="description" rows="3" />
      </label>
      <label class="is-switch">
        <input v-model="isEnabled" type="checkbox" />
        <span>启用分组</span>
      </label>
      <p v-if="localError || error" class="create-alarm-group__error">
        {{ localError || error }}
      </p>
    </form>

    <template #footer>
      <div class="create-alarm-group__footer">
        <button type="button" class="is-ghost" @click="visible = false">取消</button>
        <button type="button" :disabled="submitting" @click="submit">
          {{ submitting ? "创建中" : "创建" }}
        </button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import type { AlarmPolicyGroupSave } from "@/api/schemas/alarm.schema";
import DcDialog from "@/components/shared/DcDialog.vue";

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
  submit: [payload: AlarmPolicyGroupSave];
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit("update:modelValue", value),
});

const name = ref("");
const description = ref("");
const isEnabled = ref(true);
const localError = ref("");

const reset = () => {
  name.value = "";
  description.value = "";
  isEnabled.value = true;
  localError.value = "";
};

const submit = () => {
  localError.value = "";
  if (!name.value) {
    localError.value = "请输入分组名";
    return;
  }
  emit("submit", {
    name: name.value,
    description: description.value || null,
    isEnabled: isEnabled.value,
  });
};
</script>

<style scoped>
.create-alarm-group {
  display: grid;
  gap: 12px;
}

.create-alarm-group label {
  display: grid;
  gap: 6px;
}

.create-alarm-group label > span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.create-alarm-group input,
.create-alarm-group textarea {
  width: 100%;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  font-family: inherit;
  font-size: 13px;
  outline: none;
}

.create-alarm-group input {
  height: 34px;
  padding: 0 10px;
}

.create-alarm-group textarea {
  padding: 9px 10px;
  resize: vertical;
}

.create-alarm-group .is-switch {
  grid-template-columns: auto 1fr;
  align-items: center;
  gap: 8px;
}

.create-alarm-group .is-switch input {
  width: 15px;
  height: 15px;
  padding: 0;
  accent-color: var(--dc-primary);
}

.create-alarm-group__error {
  margin: 0;
  padding: 9px 10px;
  border: 1px solid rgba(220, 38, 38, 0.28);
  border-radius: var(--dc-radius-sm);
  background: rgba(220, 38, 38, 0.06);
  color: var(--dc-danger, #b91c1c);
  font-size: 13px;
}

.create-alarm-group__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.create-alarm-group__footer button {
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

.create-alarm-group__footer button.is-ghost {
  border-color: var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}
</style>
