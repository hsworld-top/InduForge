<template>
  <DcDialog
    v-model="visible"
    title="新建报警分组"
    width="460px"
    body-max-height="320px"
    @close="resetForm"
  >
    <el-form label-position="top" class="alarm-group-dialog">
      <el-form-item label="名称" required>
        <el-input
          v-model="form.name"
          maxlength="40"
          show-word-limit
          placeholder="输入报警分组名称"
        />
      </el-form-item>

      <el-form-item label="上级分组">
        <el-select
          v-model="form.parentId"
          class="alarm-group-dialog__select"
          clearable
          placeholder="根目录"
        >
          <el-option label="根目录" :value="null" />
          <el-option
            v-for="group in groupOptions"
            :key="group.id"
            :label="group.label"
            :value="group.id"
          />
        </el-select>
      </el-form-item>

      <div v-if="error" class="alarm-group-dialog__error">{{ error }}</div>
    </el-form>

    <template #footer>
      <div class="alarm-group-dialog__footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="submitting"
          :disabled="!canSubmit"
          @click="submit"
        >
          创建
        </el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import type {
  AlarmPolicyGroup,
  AlarmPolicyGroupSave,
} from "@/api/schemas/alarm.schema";
import DcDialog from "@/components/shared/DcDialog.vue";

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
  submit: [payload: AlarmPolicyGroupSave];
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit("update:modelValue", value),
});

const form = reactive<AlarmPolicyGroupSave>({
  name: "",
  parentId: null,
});

const canSubmit = computed(
  () => form.name.trim().length > 0 && !props.submitting,
);

const flattenGroups = (
  groups: AlarmPolicyGroup[],
  parentId: string | null = null,
  depth = 0,
): Array<{ id: string; label: string }> =>
  groups
    .filter((group) => (group.parentId || null) === parentId)
    .flatMap((group) => [
      { id: group.id, label: `${"　".repeat(depth)}${group.name}` },
      ...flattenGroups(groups, group.id, depth + 1),
    ]);

const groupOptions = computed(() => flattenGroups(props.groups));

function resetForm() {
  form.name = "";
  form.parentId = null;
}

function submit() {
  if (!canSubmit.value) return;
  emit("submit", {
    name: form.name.trim(),
    parentId: form.parentId || null,
  });
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) resetForm();
  },
);
</script>

<style scoped>
.alarm-group-dialog {
  display: grid;
  gap: 2px;
}

.alarm-group-dialog__select {
  width: 100%;
}

.alarm-group-dialog__error {
  padding: 9px 10px;
  border: 1px solid rgba(220, 38, 38, 0.2);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
  font-size: 13px;
  line-height: 1.5;
}

.alarm-group-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
