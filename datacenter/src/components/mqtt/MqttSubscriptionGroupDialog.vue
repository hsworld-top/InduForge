<template>
  <WorkbenchGroupDialog
    ref="dialogRef"
    v-model="visible"
    :mode="mode"
    :title="mode === 'create' ? '新建分组' : '编辑分组'"
    :group="group"
    :group-options="groupOptions"
    :initial-parent-id="initialParentId"
    :loading="loading"
    :max-name-length="60"
    @submit="$emit('submit', $event)"
  />
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import WorkbenchGroupDialog from '@/components/workbench/WorkbenchGroupDialog.vue'
import type { MqttSubscriptionGroup, MqttSubscriptionGroupNode } from './mqttSubscriptionTreeModel'
import {
  collectMqttSubscriptionGroupIds,
  flattenMqttSubscriptionGroups,
} from './mqttSubscriptionTreeModel'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    mode: 'create' | 'rename'
    groups: MqttSubscriptionGroup[]
    group?: MqttSubscriptionGroupNode | null
    initialParentId?: string | null
    loading?: boolean
  }>(),
  {
    group: null,
    initialParentId: null,
    loading: false,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', data: { name: string; parentId: string | null }): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const dialogRef = ref<InstanceType<typeof WorkbenchGroupDialog> | null>(null)
const blockedIds = computed(() =>
  props.mode === 'rename' ? collectMqttSubscriptionGroupIds(props.group) : new Set<string>(),
)
const groupOptions = computed(() => flattenMqttSubscriptionGroups(props.groups, blockedIds.value))

function closeSilently() {
  dialogRef.value?.closeSilently()
}

defineExpose({ closeSilently })
</script>
