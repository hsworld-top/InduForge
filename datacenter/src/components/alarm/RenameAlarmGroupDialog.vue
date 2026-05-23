<template>
  <DcDialog
    v-model="visible"
    title="重命名报警分组"
    width="420px"
    body-max-height="220px"
    :dirty="isDirty"
    :close-disabled="loading"
    @close="resetForm"
  >
    <el-form label-position="top" class="alarm-group-rename-dialog">
      <el-form-item label="名称" required>
        <el-input
          v-model="name"
          maxlength="60"
          show-word-limit
          placeholder="输入报警分组名称"
          @keyup.enter="submit"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="alarm-group-rename-dialog__footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="loading"
          :disabled="!canSubmit"
          @click="submit"
        >
          保存
        </el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import type { AlarmPolicyGroup } from "@/api/schemas/alarm.schema";
import DcDialog from "@/components/shared/DcDialog.vue";

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    group: AlarmPolicyGroup | null;
    loading?: boolean;
  }>(),
  {
    loading: false,
  },
);

const emit = defineEmits<{
  (event: "update:modelValue", value: boolean): void;
  (event: "submit", name: string): void;
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit("update:modelValue", value),
});

const name = ref("");
const canSubmit = computed(
  () =>
    Boolean(props.group) &&
    name.value.trim().length > 0 &&
    name.value.trim() !== props.group?.name &&
    !props.loading,
);
const isDirty = computed(
  () =>
    visible.value &&
    Boolean(props.group) &&
    name.value.trim() !== (props.group?.name || ""),
);

function resetForm() {
  name.value = props.group?.name || "";
}

function submit() {
  if (!canSubmit.value) return;
  emit("submit", name.value.trim());
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) resetForm();
  },
);

watch(
  () => props.group?.id,
  () => {
    if (props.modelValue) resetForm();
  },
);
</script>

<style scoped>
.alarm-group-rename-dialog {
  display: grid;
  gap: 2px;
}

.alarm-group-rename-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
