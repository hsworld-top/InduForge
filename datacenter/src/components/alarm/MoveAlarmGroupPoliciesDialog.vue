<template>
  <DcDialog
    v-model="visible"
    title="移动组内策略"
    width="460px"
    body-max-height="260px"
    @close="resetForm"
  >
    <el-form label-position="top" class="alarm-group-move-dialog">
      <el-form-item label="目标分组">
        <el-select
          v-model="groupId"
          class="alarm-group-move-dialog__select"
          clearable
          placeholder="根目录"
        >
          <el-option label="根目录" :value="null" />
          <el-option
            v-for="group in targetGroups"
            :key="group.id"
            :label="group.name"
            :value="group.id"
          />
        </el-select>
      </el-form-item>
      <p class="alarm-group-move-dialog__hint">
        将当前分组下 {{ policyCount }} 条策略移动到目标分组。
      </p>
    </el-form>

    <template #footer>
      <div class="alarm-group-move-dialog__footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="loading"
          :disabled="!canSubmit"
          @click="submit"
        >
          移动
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
    groups: AlarmPolicyGroup[];
    policyCount: number;
    loading?: boolean;
  }>(),
  {
    loading: false,
  },
);

const emit = defineEmits<{
  (event: "update:modelValue", value: boolean): void;
  (event: "submit", groupId: string | null): void;
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit("update:modelValue", value),
});

const groupId = ref<string | null>(null);
const currentGroupId = computed(() => props.group?.id || null);
const targetGroups = computed(() =>
  props.groups.filter((group) => group.id !== currentGroupId.value),
);
const canSubmit = computed(
  () =>
    Boolean(props.group) &&
    props.policyCount > 0 &&
    groupId.value !== currentGroupId.value &&
    !props.loading,
);

function resetForm() {
  groupId.value = currentGroupId.value;
}

function submit() {
  if (!canSubmit.value) return;
  emit("submit", groupId.value || null);
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
.alarm-group-move-dialog {
  display: grid;
  gap: 2px;
}

.alarm-group-move-dialog__select {
  width: 100%;
}

.alarm-group-move-dialog__hint {
  margin: -2px 0 0;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.alarm-group-move-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
