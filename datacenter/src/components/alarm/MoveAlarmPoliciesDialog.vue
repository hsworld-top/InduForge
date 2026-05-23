<template>
  <DcDialog
    v-model="visible"
    title="移动已选策略"
    width="460px"
    body-max-height="260px"
    :dirty="isDirty"
    :close-disabled="loading"
    @close="resetForm"
  >
    <el-form label-position="top" class="alarm-bulk-move-dialog">
      <el-form-item label="目标分组">
        <el-select
          v-model="groupId"
          class="alarm-bulk-move-dialog__select"
          clearable
          placeholder="根目录"
        >
          <el-option label="根目录" :value="null" />
          <el-option
            v-for="group in groups"
            :key="group.id"
            :label="group.name"
            :value="group.id"
          />
        </el-select>
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="alarm-bulk-move-dialog__footer">
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
    groups: AlarmPolicyGroup[];
    selectedCount: number;
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
const canSubmit = computed(() => props.selectedCount > 0 && !props.loading);
const isDirty = computed(() => visible.value && Boolean(groupId.value));

function resetForm() {
  groupId.value = null;
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
</script>

<style scoped>
.alarm-bulk-move-dialog {
  display: grid;
  gap: 2px;
}

.alarm-bulk-move-dialog__select {
  width: 100%;
}

.alarm-bulk-move-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
