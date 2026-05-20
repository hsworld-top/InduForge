<template>
  <DcDialog
    v-model="visible"
    title="重命名报警策略"
    width="420px"
    body-max-height="220px"
    @close="resetForm"
  >
    <el-form label-position="top" class="alarm-rename-dialog">
      <el-form-item label="名称" required>
        <el-input
          v-model="name"
          maxlength="80"
          show-word-limit
          placeholder="输入报警策略名称"
          @keyup.enter="submit"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="alarm-rename-dialog__footer">
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
import type { AlarmPolicy } from "@/api/schemas/alarm.schema";
import DcDialog from "@/components/shared/DcDialog.vue";

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    policy: AlarmPolicy | null;
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
    Boolean(props.policy) &&
    name.value.trim().length > 0 &&
    name.value.trim() !== props.policy?.name &&
    !props.loading,
);

function resetForm() {
  name.value = props.policy?.name || "";
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
  () => props.policy?.id,
  () => {
    if (props.modelValue) resetForm();
  },
);
</script>

<style scoped>
.alarm-rename-dialog {
  display: grid;
  gap: 2px;
}

.alarm-rename-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
