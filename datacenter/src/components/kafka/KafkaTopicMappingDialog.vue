<template>
  <DcDialog
    v-model="visible"
    :title="mode === 'create' ? '新建 Topic 订阅' : '编辑 Topic 订阅'"
    width="640px"
    :close-disabled="loading"
  >
    <el-form label-position="top" class="kafka-topic-dialog">
      <div class="kafka-topic-dialog__section">
        <div class="kafka-topic-dialog__section-title">基础信息</div>
        <el-form-item required>
          <template #label>
            <span class="kafka-topic-dialog__field-label">
              名称
              <el-tooltip content="用于在工作台左侧展示，建议填写业务含义清晰的名称。" placement="top">
                <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
              </el-tooltip>
            </span>
          </template>
          <el-input v-model="form.name" maxlength="100" show-word-limit placeholder="设备遥测" />
        </el-form-item>
        <el-form-item required>
          <template #label>
            <span class="kafka-topic-dialog__field-label">
              Topic
              <el-tooltip content="填写 Kafka 中真实存在或准备消费的 Topic 名称。" placement="top">
                <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
              </el-tooltip>
            </span>
          </template>
          <el-input v-model="form.topic" maxlength="500" placeholder="device.telemetry" />
        </el-form-item>
        <el-form-item>
          <template #label>
            <span class="kafka-topic-dialog__field-label">
              消费组名称
              <el-tooltip
                content="节点侧持久运行时用于记录消费进度；工作台预览不会使用该消费组，避免影响真实进度。"
                placement="top"
              >
                <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
              </el-tooltip>
            </span>
          </template>
          <el-input
            v-model="form.consumerGroup"
            clearable
            maxlength="200"
            placeholder="留空时由节点侧默认策略生成"
          />
        </el-form-item>
        <el-form-item v-if="mode === 'edit'">
          <template #label>
            <span class="kafka-topic-dialog__field-label">
              所属分组
              <el-tooltip content="为空时显示在根目录；需要调整层级时可选择已有分组。" placement="top">
                <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
              </el-tooltip>
            </span>
          </template>
          <el-select v-model="form.groupId" clearable placeholder="根目录">
            <el-option label="根目录" value="" />
            <el-option
              v-for="group in groupOptions"
              :key="group.id"
              :label="group.label"
              :value="group.id"
            />
          </el-select>
        </el-form-item>
      </div>

      <div class="kafka-topic-dialog__section">
        <div class="kafka-topic-dialog__section-title">读取参数</div>
        <div class="kafka-topic-dialog__grid">
          <el-form-item>
            <template #label>
              <span class="kafka-topic-dialog__field-label">
                分区策略
                <el-tooltip content="全部分区适合常规预览；单分区可用于定位指定 Partition 的消息。" placement="top">
                  <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
                </el-tooltip>
              </span>
            </template>
            <el-segmented v-model="form.partitionMode" :options="partitionModeOptions" />
          </el-form-item>
          <el-form-item v-if="form.partitionMode === 'single'" label="分区编号" required>
            <el-input-number v-model="form.partition" :min="0" :step="1" />
          </el-form-item>
          <el-form-item>
            <template #label>
              <span class="kafka-topic-dialog__field-label">
                起始位置
                <el-tooltip content="仅影响预览或首次读取样本的位置；运行态消费进度由订阅/消费组记录。" placement="top">
                  <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
                </el-tooltip>
              </span>
            </template>
            <el-select v-model="form.startPosition">
              <el-option label="最新位置" value="latest" />
              <el-option label="最早位置" value="earliest" />
              <el-option label="指定 Offset" value="offset" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="form.startPosition === 'offset'" label="起始 Offset" required>
            <el-input-number v-model="form.startOffset" :min="0" :step="1" />
          </el-form-item>
          <el-form-item>
            <template #label>
              <span class="kafka-topic-dialog__field-label">
                消息解码
                <el-tooltip content="JSON 会尝试解析消息体；String 保留文本；Binary 用于二进制载荷预览。" placement="top">
                  <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
                </el-tooltip>
              </span>
            </template>
            <el-select v-model="form.decode">
              <el-option label="JSON" value="json" />
              <el-option label="文本" value="string" />
              <el-option label="二进制" value="binary" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <template #label>
              <span class="kafka-topic-dialog__field-label">
                样本上限
                <el-tooltip content="单次预览最多读取的消息条数，数值越大等待时间可能越长。" placement="top">
                  <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
                </el-tooltip>
              </span>
            </template>
            <el-input-number v-model="form.sampleLimit" :min="1" :max="1000" :step="10" />
          </el-form-item>
          <el-form-item>
            <template #label>
              <span class="kafka-topic-dialog__field-label">
                预览超时
                <el-tooltip content="单次预览等待消息的最长时间，单位毫秒。" placement="top">
                  <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
                </el-tooltip>
              </span>
            </template>
            <el-input-number v-model="form.timeoutMs" :min="1000" :max="30000" :step="1000" />
          </el-form-item>
        </div>
      </div>

      <div class="kafka-topic-dialog__section">
        <div class="kafka-topic-dialog__section-title">备注</div>
        <el-form-item label="描述">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="3"
            maxlength="500"
            show-word-limit
            placeholder="可填写 Topic 用途、消息来源或字段说明"
          />
        </el-form-item>
      </div>
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
import IconTablerHelpCircle from '~icons/tabler/help-circle'
import type { KafkaTopicGroup, KafkaTopicMapping } from './types'
import { flattenKafkaTopicGroups } from './kafkaTopicTreeModel'

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
const groupOptions = computed(() => flattenKafkaTopicGroups(props.groups))

const form = reactive({
  name: '',
  topic: '',
  groupId: '',
  consumerGroup: '',
  partitionMode: 'all',
  partition: 0,
  startPosition: 'latest',
  startOffset: null as number | null,
  decode: 'json',
  sampleLimit: 100,
  timeoutMs: 5000,
  description: '',
})

const canSubmit = computed(() => {
  if (props.loading) return false
  if (!form.name.trim() || !form.topic.trim()) return false
  if (
    form.startPosition === 'offset' &&
    (form.partitionMode !== 'single' || form.startOffset === null || form.startOffset < 0)
  ) {
    return false
  }
  return form.partitionMode !== 'single' || form.partition >= 0
})

const resetForm = () => {
  form.name = props.mapping?.name || ''
  form.topic = props.mapping?.topic || ''
  form.groupId = props.mapping?.groupId ? String(props.mapping.groupId) : ''
  form.consumerGroup = props.mapping?.consumerGroup || ''
  form.partitionMode = props.mapping?.partitionMode || 'all'
  form.partition = props.mapping?.partition ?? 0
  form.startPosition = props.mapping?.startPosition || 'latest'
  form.startOffset = props.mapping?.startOffset ?? null
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
    consumerGroup: form.consumerGroup.trim(),
    partitionMode: form.partitionMode,
    partition: form.partitionMode === 'single' ? form.partition : null,
    startPosition: form.startPosition,
    startOffset: form.startPosition === 'offset' ? form.startOffset : null,
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

watch(
  () => form.startPosition,
  (value) => {
    if (value === 'offset') {
      form.partitionMode = 'single'
      if (form.startOffset === null) form.startOffset = 0
    } else {
      form.startOffset = null
    }
  },
)
</script>

<style scoped>
.kafka-topic-dialog {
  display: grid;
  gap: 14px;
}

.kafka-topic-dialog :deep(.el-select),
.kafka-topic-dialog :deep(.el-input-number) {
  width: 100%;
}

.kafka-topic-dialog__section {
  display: grid;
  gap: 2px;
}

.kafka-topic-dialog__section-title {
  margin-bottom: 6px;
  padding-left: 8px;
  border-left: 3px solid var(--dc-primary);
  color: var(--dc-text);
  font-size: 14px;
  font-weight: 650;
}

.kafka-topic-dialog__field-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.kafka-topic-dialog__field-help {
  width: 14px;
  height: 14px;
  color: var(--dc-text-muted);
  cursor: help;
}

.kafka-topic-dialog__field-help:hover {
  color: var(--dc-primary);
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
