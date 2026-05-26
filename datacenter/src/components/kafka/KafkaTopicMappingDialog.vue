<template>
  <DcDialog
    v-model="visible"
    :title="mode === 'create' ? '新建 Topic 映射' : '编辑 Topic 映射'"
    width="640px"
    :close-disabled="loading"
  >
    <el-form label-position="top" class="kafka-topic-dialog">
      <el-form-item label="显示名称" required>
        <el-input v-model="form.name" maxlength="100" show-word-limit placeholder="设备遥测" />
      </el-form-item>
      <el-form-item label="Topic" required>
        <el-input v-model="form.topic" maxlength="500" placeholder="device.telemetry" />
      </el-form-item>
      <el-form-item label="分组">
        <el-select v-model="form.groupId" clearable placeholder="根目录">
          <el-option label="根目录" :value="null" />
          <el-option
            v-for="group in groups"
            :key="String(group.id)"
            :label="group.name"
            :value="String(group.id)"
          />
        </el-select>
      </el-form-item>
      <div class="kafka-topic-dialog__grid">
        <el-form-item label="分区策略">
          <el-segmented v-model="form.partitionMode" :options="partitionModeOptions" />
        </el-form-item>
        <el-form-item v-if="form.partitionMode === 'single'" label="Partition" required>
          <el-input-number v-model="form.partition" :min="0" :step="1" />
        </el-form-item>
        <el-form-item label="起始位置">
          <el-select v-model="form.startPosition">
            <el-option label="Latest" value="latest" />
            <el-option label="Earliest" value="earliest" />
            <el-option label="指定 Offset" value="offset" />
          </el-select>
        </el-form-item>
        <el-form-item label="解码">
          <el-select v-model="form.decode">
            <el-option label="JSON" value="json" />
            <el-option label="String" value="string" />
            <el-option label="Binary" value="binary" />
          </el-select>
        </el-form-item>
        <el-form-item label="样本上限">
          <el-input-number v-model="form.sampleLimit" :min="1" :max="1000" :step="10" />
        </el-form-item>
        <el-form-item label="超时 ms">
          <el-input-number v-model="form.timeoutMs" :min="1000" :max="30000" :step="1000" />
        </el-form-item>
      </div>
      <el-form-item label="描述">
        <el-input
          v-model="form.description"
          type="textarea"
          :rows="3"
          maxlength="500"
          show-word-limit
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="kafka-topic-dialog__footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="loading" :disabled="!canSubmit" @click="submit">
          保存
        </el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import type { KafkaTopicGroup, KafkaTopicMapping } from './types'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    mode: 'create' | 'edit'
    groups?: KafkaTopicGroup[]
    mapping?: KafkaTopicMapping | null
    loading?: boolean
  }>(),
  {
    groups: () => [],
    mapping: null,
    loading: false,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', value: Record<string, unknown>): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const partitionModeOptions = [
  { label: '全部分区', value: 'all' },
  { label: '单分区', value: 'single' },
]

const form = reactive({
  name: '',
  topic: '',
  groupId: null as string | null,
  partitionMode: 'all',
  partition: 0,
  startPosition: 'latest',
  decode: 'json',
  sampleLimit: 100,
  timeoutMs: 5000,
  description: '',
})

const canSubmit = computed(() => {
  if (props.loading) return false
  if (!form.name.trim() || !form.topic.trim()) return false
  return form.partitionMode !== 'single' || form.partition >= 0
})

const resetForm = () => {
  form.name = props.mapping?.name || ''
  form.topic = props.mapping?.topic || ''
  form.groupId = props.mapping?.groupId ? String(props.mapping.groupId) : null
  form.partitionMode = props.mapping?.partitionMode || 'all'
  form.partition = props.mapping?.partition ?? 0
  form.startPosition = props.mapping?.startPosition || 'latest'
  form.decode = props.mapping?.decode || 'json'
  form.sampleLimit = props.mapping?.sampleLimit || 100
  form.timeoutMs = props.mapping?.timeoutMs || 5000
  form.description = props.mapping?.description || ''
}

const submit = () => {
  if (!canSubmit.value) return
  emit('submit', {
    name: form.name.trim(),
    topic: form.topic.trim(),
    groupId: form.groupId || null,
    partitionMode: form.partitionMode,
    partition: form.partitionMode === 'single' ? form.partition : null,
    startPosition: form.startPosition,
    decode: form.decode,
    sampleLimit: form.sampleLimit,
    timeoutMs: form.timeoutMs,
    description: form.description.trim(),
  })
}

watch(
  () => [props.modelValue, props.mapping?.id] as const,
  () => {
    if (props.modelValue) resetForm()
  },
  { immediate: true },
)
</script>

<style scoped>
.kafka-topic-dialog {
  display: grid;
  gap: 2px;
}

.kafka-topic-dialog :deep(.el-select),
.kafka-topic-dialog :deep(.el-input-number) {
  width: 100%;
}

.kafka-topic-dialog__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 12px;
}

.kafka-topic-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

@media (max-width: 640px) {
  .kafka-topic-dialog__grid {
    grid-template-columns: 1fr;
  }
}
</style>
