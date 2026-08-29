<template>
  <DcDialog
    v-model="visible"
    :title="targetType === 'group' ? ui('移动分组', 'Move Group') : ui('移动订阅', 'Move Subscription')"
    width="460px"
    body-max-height="260px"
    :close-disabled="loading"
    @close="resetForm"
  >
    <el-form label-position="top" class="mqtt-subscription-move-dialog">
      <el-form-item :label="ui('目标分组', 'Target Group')">
        <el-select
          v-model="targetGroupId"
          class="mqtt-subscription-move-dialog__select"
          clearable
          :placeholder="ui('根目录', 'Root')"
        >
          <el-option :label="ui('根目录', 'Root')" :value="null" />
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
        <el-button @click="visible = false">{{ ui('取消', 'Cancel') }}</el-button>
        <el-button type="primary" :loading="loading" :disabled="!canSubmit" @click="submit">
          {{ ui('移动', 'Move') }}
        </el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)
import type {
  MqttSubscription,
  MqttSubscriptionGroup,
  MqttSubscriptionGroupNode,
} from './mqttSubscriptionTreeModel'
import {
  collectMqttSubscriptionGroupIds,
  flattenMqttSubscriptionGroups,
} from './mqttSubscriptionTreeModel'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    targetType: 'subscription' | 'group'
    subscription?: MqttSubscription | null
    group?: MqttSubscriptionGroupNode | null
    groups: MqttSubscriptionGroup[]
    loading?: boolean
  }>(),
  {
    subscription: null,
    group: null,
    loading: false,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', groupId: string | null): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const targetGroupId = ref<string | null>(null)
const blockedIds = computed(() =>
  props.targetType === 'group' ? collectMqttSubscriptionGroupIds(props.group) : new Set<string>(),
)
const currentGroupId = computed(() =>
  props.targetType === 'group'
    ? props.group?.parentId || null
    : props.subscription?.groupId
      ? String(props.subscription.groupId)
      : null,
)
const groupOptions = computed(() => flattenMqttSubscriptionGroups(props.groups, blockedIds.value))
const canSubmit = computed(
  () =>
    !props.loading &&
    Boolean(props.targetType === 'group' ? props.group : props.subscription) &&
    targetGroupId.value !== currentGroupId.value,
)
function resetForm() {
  targetGroupId.value = currentGroupId.value
}

function submit() {
  if (!canSubmit.value) return
  emit('submit', targetGroupId.value || null)
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) resetForm()
  },
)

watch(
  () => [props.subscription?.id, props.group?.id, props.targetType],
  () => {
    if (props.modelValue) resetForm()
  },
)
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
