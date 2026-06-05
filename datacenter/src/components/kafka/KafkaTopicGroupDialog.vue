<template>
  <WorkbenchGroupDialog
    ref="dialogRef"
    v-model="visible"
    :mode="mode"
    :title="mode === 'create' ? '新建 Topic 分组' : '编辑 Topic 分组'"
    :group="group"
    :group-options="groupOptions"
    :initial-parent-id="initialParentId"
    :loading="loading"
    @submit="$emit('submit', $event)"
  />
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import WorkbenchGroupDialog from '@/components/workbench/WorkbenchGroupDialog.vue'
import type { KafkaTopicGroup } from './types'
import { collectKafkaTopicGroupIds, flattenKafkaTopicGroups } from './kafkaTopicTreeModel'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    mode: 'create' | 'edit'
    groups?: KafkaTopicGroup[]
    group?: KafkaTopicGroup | null
    initialParentId?: string | null
    loading?: boolean
  }>(),
  {
    groups: () => [],
    group: null,
    initialParentId: null,
    loading: false,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', value: { name: string; parentId: string | null }): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const dialogRef = ref<InstanceType<typeof WorkbenchGroupDialog> | null>(null)
const blockedIds = computed(() =>
  props.mode === 'edit' ? collectKafkaTopicGroupIds(props.group as any) : new Set<string>(),
)
const groupOptions = computed(() => flattenKafkaTopicGroups(props.groups, blockedIds.value))

function closeSilently() {
  dialogRef.value?.closeSilently()
}

defineExpose({ closeSilently })
</script>
