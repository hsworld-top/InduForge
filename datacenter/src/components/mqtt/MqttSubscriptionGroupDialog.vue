<template>
  <WorkbenchGroupDialog
    ref="dialogRef"
    v-model="visible"
    :mode="mode"
    :title="mode === 'create' ? ui('新建分组', 'New Group') : ui('编辑分组', 'Edit Group')"
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
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)
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
