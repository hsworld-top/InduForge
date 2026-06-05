<template>
  <DcDialog v-model="visible" :title="title" width="420px" :close-disabled="loading">
    <el-form label-position="top">
      <el-form-item label="目标分组">
        <el-select v-model="targetGroupId" class="kafka-topic-move-dialog__select" clearable placeholder="根目录">
          <el-option label="根目录" :value="null" />
          <el-option v-for="group in groupOptions" :key="group.id" :label="group.label" :value="group.id" />
        </el-select>
      </el-form-item>
    </el-form>
    <template #footer>
      <div class="kafka-topic-move-dialog__footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="loading" @click="submit">移动</el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import type { KafkaTopicGroup, KafkaTopicGroupNode, KafkaTopicMapping } from './types'
import { flattenKafkaTopicGroups } from './kafkaTopicTreeModel'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    targetType: 'mapping' | 'group'
    groups: KafkaTopicGroup[]
    mapping?: KafkaTopicMapping | null
    group?: KafkaTopicGroupNode | null
    loading?: boolean
  }>(),
  {
    mapping: null,
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
  set: (value) => emit('update:modelValue', value),
})
const targetGroupId = ref<string | null>(null)
const blockedIds = computed(() =>
  props.targetType === 'group' && props.group ? collectGroupIds(props.group) : new Set<string>(),
)
const groupOptions = computed(() => flattenKafkaTopicGroups(props.groups, blockedIds.value))
const title = computed(() => (props.targetType === 'group' ? '移动分组' : '移动 Topic 订阅'))

function submit() {
  emit('submit', targetGroupId.value || null)
}

function collectGroupIds(group: KafkaTopicGroupNode) {
  const result = new Set<string>([String(group.id)])
  group.children.forEach((child) => {
    collectGroupIds(child).forEach((id) => result.add(id))
  })
  return result
}

watch(
  () => [props.modelValue, props.mapping?.id, props.group?.id, props.targetType] as const,
  () => {
    if (!props.modelValue) return
    targetGroupId.value =
      props.targetType === 'group'
        ? props.group?.parentId
          ? String(props.group.parentId)
          : null
        : props.mapping?.groupId
          ? String(props.mapping.groupId)
          : null
  },
)
</script>

<style scoped>
.kafka-topic-move-dialog__select {
  width: 100%;
}

.kafka-topic-move-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
