<template>
  <DcDialog
    v-model="visible"
    :title="targetType === 'group' ? '移动分组' : '移动订阅'"
    width="460px"
    body-max-height="260px"
    :dirty="isDirty"
    :close-disabled="loading"
    @close="resetForm"
  >
    <el-form label-position="top" class="mqtt-subscription-move-dialog">
      <el-form-item label="目标分组">
        <el-select
          v-model="targetGroupId"
          class="mqtt-subscription-move-dialog__select"
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
    </el-form>

    <template #footer>
      <div class="mqtt-subscription-move-dialog__footer">
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
import DcDialog from "@/components/shared/DcDialog.vue";
import type {
  MqttSubscription,
  MqttSubscriptionGroup,
  MqttSubscriptionGroupNode,
} from "./mqttSubscriptionTreeModel";
import {
  collectMqttSubscriptionGroupIds,
  flattenMqttSubscriptionGroups,
} from "./mqttSubscriptionTreeModel";

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    targetType: "subscription" | "group";
    subscription?: MqttSubscription | null;
    group?: MqttSubscriptionGroupNode | null;
    groups: MqttSubscriptionGroup[];
    loading?: boolean;
  }>(),
  {
    subscription: null,
    group: null,
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

const targetGroupId = ref<string | null>(null);
const blockedIds = computed(() =>
  props.targetType === "group"
    ? collectMqttSubscriptionGroupIds(props.group)
    : new Set<string>(),
);
const currentGroupId = computed(() =>
  props.targetType === "group"
    ? props.group?.parentId || null
    : props.subscription?.groupId
      ? String(props.subscription.groupId)
      : null,
);
const groupOptions = computed(() =>
  flattenMqttSubscriptionGroups(props.groups, blockedIds.value),
);
const canSubmit = computed(
  () =>
    !props.loading &&
    Boolean(props.targetType === "group" ? props.group : props.subscription) &&
    targetGroupId.value !== currentGroupId.value,
);
const isDirty = computed(
  () =>
    visible.value &&
    Boolean(props.targetType === "group" ? props.group : props.subscription) &&
    targetGroupId.value !== currentGroupId.value,
);

function resetForm() {
  targetGroupId.value = currentGroupId.value;
}

function submit() {
  if (!canSubmit.value) return;
  emit("submit", targetGroupId.value || null);
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) resetForm();
  },
);

watch(
  () => [props.subscription?.id, props.group?.id, props.targetType],
  () => {
    if (props.modelValue) resetForm();
  },
);
</script>

<style scoped>
.mqtt-subscription-move-dialog {
  display: grid;
  gap: 2px;
}

.mqtt-subscription-move-dialog__select {
  width: 100%;
}

.mqtt-subscription-move-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
