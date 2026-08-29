<template>
  <DcDialog
    v-model="visible"
    :title="mode === 'create' ? ui('新建消费规则', 'New Consumer Rule') : ui('编辑消费规则', 'Edit Consumer Rule')"
    width="640px"
    :close-disabled="loading"
  >
    <el-form label-position="top" class="kafka-topic-dialog">
      <div class="kafka-topic-dialog__section">
        <div class="kafka-topic-dialog__section-title">{{ ui('基础信息', 'Basic Information') }}</div>
        <el-form-item>
          <template #label>
            <span class="kafka-topic-dialog__field-label">
              {{ ui('输出模式', 'Output Mode') }}
              <el-tooltip
                :content="ui('整包数据点会把最新消息写入一个数据点；字段数据点会把消息字段拆成多个数据点。', 'Raw-message mode writes the latest message to one data point; field mode splits message fields into multiple data points.')"
                placement="top"
              >
                <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
              </el-tooltip>
            </span>
          </template>
          <div class="kafka-topic-dialog__mode-options" :class="{ 'is-readonly': mode === 'edit' }">
            <button
              v-for="option in outputModeOptions"
              :key="option.value"
              type="button"
              class="kafka-topic-dialog__mode-option"
              :class="{ 'is-active': form.outputMode === option.value }"
              :disabled="mode === 'edit'"
              @click="form.outputMode = option.value"
            >
              <span class="kafka-topic-dialog__mode-radio">
                {{ form.outputMode === option.value ? '●' : '○' }}
              </span>
              <span>
                <strong>{{ option.label }}</strong>
                <em>{{ option.description }}</em>
              </span>
            </button>
          </div>
          <p v-if="mode === 'edit'" class="kafka-topic-dialog__hint">
            {{ ui('输出模式创建后不可修改。如需切换模式，请新建另一条消费规则。', 'Output mode cannot be changed after creation. Create another consumer rule to use a different mode.') }}
          </p>
        </el-form-item>
        <el-form-item required>
          <template #label>
            <span class="kafka-topic-dialog__field-label">
              {{ ui('名称', 'Name') }}
              <el-tooltip :content="ui('消费规则会出现在左侧树中，建议使用业务流名称。', 'The consumer rule appears in the tree. Use a business flow name.')" placement="top">
                <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
              </el-tooltip>
            </span>
          </template>
          <el-input v-model="form.name" maxlength="100" show-word-limit :placeholder="ui('设备遥测', 'Device Telemetry')" />
        </el-form-item>
        <el-form-item required>
          <template #label>
            <span class="kafka-topic-dialog__field-label">
              Topic
              <el-tooltip :content="ui('填写 Kafka 中真实存在或准备消费的 Topic 名称。', 'Enter the Kafka topic to consume.')" placement="top">
                <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
              </el-tooltip>
            </span>
          </template>
          <el-input v-model="form.topic" maxlength="500" placeholder="device.telemetry" />
        </el-form-item>
        <el-form-item>
          <template #label>
            <span class="kafka-topic-dialog__field-label">
              {{ ui('消费组名称', 'Consumer Group') }}
              <el-tooltip
                :content="ui('节点侧持久运行时用于记录消费进度；工作台预览不会使用该消费组，避免影响真实进度。', 'The node uses this group to track runtime progress. Workbench previews do not use it, so real progress is unaffected.')"
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
            :placeholder="ui('留空时由节点侧默认策略生成', 'Leave blank to generate on the node')"
          />
        </el-form-item>
      </div>

      <div v-if="form.outputMode === 'raw_message'" class="kafka-topic-dialog__section">
        <div class="kafka-topic-dialog__section-title">{{ ui('数据点输出', 'Data Point Output') }}</div>
        <el-form-item>
          <template #label>
            <span class="kafka-topic-dialog__field-label">
              {{ ui('输出内容', 'Output Content') }}
              <el-tooltip
                :content="ui('消息体只写入 payload/value；完整消息会包含 key、headers、partition、offset 和 payload。', 'Message body writes only payload/value; a complete message includes key, headers, partition, offset, and payload.')"
                placement="top"
              >
                <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
              </el-tooltip>
            </span>
          </template>
          <el-select v-model="form.rawOutputScope">
            <el-option :label="ui('消息体 payload', 'Message payload')" value="value" />
            <el-option :label="ui('完整消息', 'Complete message')" value="full_message" />
          </el-select>
        </el-form-item>
        <p class="kafka-topic-dialog__hint">
          {{ ui('整包数据点路径按规则名称自动生成，例如规则名 test1 对应 kafka.test1。', 'The raw-message data point path is generated from the rule name; for example, test1 becomes kafka.test1.') }}
        </p>
      </div>
      <div v-else class="kafka-topic-dialog__mode-note">
        {{ ui('保存后进入字段映射面板，可粘贴 JSON 样本或临时拉取样本生成数据点映射。', 'After saving, use the field mapping panel to paste or pull a JSON sample and generate data point mappings.') }}
      </div>

      <div class="kafka-topic-dialog__section">
        <el-collapse v-model="advancedSections" class="kafka-topic-dialog__advanced">
          <el-collapse-item :title="ui('高级配置：运行消费与样本参数', 'Advanced: Runtime Consumer and Sample Settings')" name="runtime">
            <div class="kafka-topic-dialog__grid">
              <el-form-item>
                <template #label>
                  <span class="kafka-topic-dialog__field-label">
                    {{ ui('分区策略', 'Partition Strategy') }}
                    <el-tooltip
                      :content="ui('全部分区适合常规预览；单分区可用于定位指定 Partition 的消息。', 'All partitions is suitable for normal previews; a single partition helps inspect a specific partition.')"
                      placement="top"
                    >
                      <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
                    </el-tooltip>
                  </span>
                </template>
                <el-segmented v-model="form.partitionMode" :options="partitionModeOptions" />
              </el-form-item>
              <el-form-item v-if="form.partitionMode === 'single'" :label="ui('分区编号', 'Partition Number')" required>
                <el-input-number v-model="form.partition" :min="0" :step="1" />
              </el-form-item>
              <el-form-item>
                <template #label>
                  <span class="kafka-topic-dialog__field-label">
                    {{ ui('起始位置', 'Start Position') }}
                    <el-tooltip
                      :content="ui('用于节点侧首次运行或开发态拉取样本；已有消费进度由消费组记录。', 'Used for the node’s first run or development sample pulls; existing progress is tracked by the consumer group.')"
                      placement="top"
                    >
                      <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
                    </el-tooltip>
                  </span>
                </template>
                <el-select v-model="form.startPosition">
                  <el-option :label="ui('最新位置', 'Latest')" value="latest" />
                  <el-option :label="ui('最早位置', 'Earliest')" value="earliest" />
                  <el-option :label="ui('指定 Offset', 'Specific Offset')" value="offset" />
                </el-select>
              </el-form-item>
              <el-form-item v-if="form.startPosition === 'offset'" :label="ui('起始 Offset', 'Start Offset')" required>
                <el-input-number v-model="form.startOffset" :min="0" :step="1" />
              </el-form-item>
              <el-form-item>
                <template #label>
                  <span class="kafka-topic-dialog__field-label">
                    {{ ui('消息解码', 'Message Decode') }}
                    <el-tooltip
                      :content="ui('JSON 会尝试解析消息体；String 保留文本；Binary 用于二进制载荷预览。', 'JSON parses the message body; String preserves text; Binary previews binary payloads.')"
                      placement="top"
                    >
                      <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
                    </el-tooltip>
                  </span>
                </template>
                <el-select v-model="form.decode">
                  <el-option label="JSON" value="json" />
                  <el-option :label="ui('文本', 'Text')" value="string" />
                  <el-option :label="ui('二进制', 'Binary')" value="binary" />
                </el-select>
              </el-form-item>
              <el-form-item>
                <template #label>
                  <span class="kafka-topic-dialog__field-label">
                    {{ ui('样本上限', 'Sample Limit') }}
                    <el-tooltip
                      :content="ui('单次预览最多读取的消息条数，数值越大等待时间可能越长。', 'Maximum messages read per preview. Larger values may take longer.')"
                      placement="top"
                    >
                      <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
                    </el-tooltip>
                  </span>
                </template>
                <el-input-number v-model="form.sampleLimit" :min="1" :max="1000" :step="10" />
              </el-form-item>
              <el-form-item>
                <template #label>
                  <span class="kafka-topic-dialog__field-label">
                    {{ ui('预览超时', 'Preview Timeout') }}
                    <el-tooltip
                      :content="ui('单次拉取样本等待消息的最长时间，单位毫秒。', 'Maximum time to wait for a sample, in milliseconds.')"
                      placement="top"
                    >
                      <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
                    </el-tooltip>
                  </span>
                </template>
                <el-input-number v-model="form.timeoutMs" :min="1000" :max="30000" :step="1000" />
              </el-form-item>
            </div>
          </el-collapse-item>
        </el-collapse>
      </div>

      <div class="kafka-topic-dialog__section">
        <div class="kafka-topic-dialog__section-title">{{ ui('备注', 'Notes') }}</div>
        <el-form-item :label="ui('描述', 'Description')">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="3"
            maxlength="500"
            show-word-limit
            :placeholder="ui('可填写 Topic 用途、消息来源或字段说明', 'Describe the topic purpose, message source, or fields')"
          />
        </el-form-item>
      </div>
    </el-form>

    <template #footer>
      <div class="kafka-topic-dialog__footer">
        <el-button @click="visible = false">{{ ui('取消', 'Cancel') }}</el-button>
        <el-button type="primary" :loading="loading" :disabled="!canSubmit" @click="submit">
          {{ ui('保存', 'Save') }}
        </el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import IconTablerHelpCircle from '~icons/tabler/help-circle'
import type { KafkaTopicGroup, KafkaTopicMapping } from './types'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

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

const partitionModeOptions = computed(() => [
  { label: ui('全部分区', 'All Partitions'), value: 'all' },
  { label: ui('单分区', 'Single Partition'), value: 'single' },
])
const outputModeOptions = computed(() => [
  {
    label: ui('整包数据点', 'Raw Message Data Point'),
    value: 'raw_message',
    description: ui('把一条 Kafka 消息作为一个数据点，数据点名称跟随消费规则。', 'Use one Kafka message as one data point named after the consumer rule.'),
  },
  {
    label: ui('字段映射', 'Field Mappings'),
    value: 'field_mapping',
    description: ui('从 JSON 消息字段生成多个映射数据点，适合结构化业务数据。', 'Generate multiple mapped data points from JSON fields for structured business data.'),
  },
])
const advancedSections = ref<string[]>([])

const form = reactive({
  name: '',
  topic: '',
  consumerGroup: '',
  outputMode: 'raw_message',
  rawOutputScope: 'value',
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
  form.consumerGroup = props.mapping?.consumerGroup || ''
  form.outputMode = props.mapping?.outputMode || 'raw_message'
  form.rawOutputScope = props.mapping?.rawOutputScope || 'value'
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
    consumerGroup: form.consumerGroup.trim(),
    outputMode: form.outputMode,
    rawOutputScope: form.outputMode === 'raw_message' ? form.rawOutputScope : 'value',
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

.kafka-topic-dialog__mode-options {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.kafka-topic-dialog__mode-option {
  min-height: 86px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  text-align: left;
}

.kafka-topic-dialog__mode-option.is-active {
  border-color: color-mix(in oklch, var(--dc-primary) 44%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.kafka-topic-dialog__mode-option:disabled {
  cursor: not-allowed;
  opacity: 0.78;
}

.kafka-topic-dialog__mode-radio {
  line-height: 20px;
}

.kafka-topic-dialog__mode-option strong,
.kafka-topic-dialog__mode-option em {
  display: block;
}

.kafka-topic-dialog__mode-option em {
  margin-top: 4px;
  color: var(--dc-text-muted);
  font-size: 12px;
  font-style: normal;
  line-height: 1.5;
}

.kafka-topic-dialog__hint {
  margin: 6px 0 0;
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 1.5;
}

.kafka-topic-dialog__advanced {
  border-top: 0;
  border-bottom: 0;
}

.kafka-topic-dialog__advanced :deep(.el-collapse-item__header) {
  height: 36px;
  border-bottom-color: var(--dc-border);
  color: var(--dc-text);
  font-weight: 650;
}

.kafka-topic-dialog__advanced :deep(.el-collapse-item__wrap) {
  border-bottom: 0;
}

.kafka-topic-dialog__advanced :deep(.el-collapse-item__content) {
  padding: 10px 0 0;
}

.kafka-topic-dialog__mode-note {
  padding: 10px 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-size: 12px;
  line-height: 1.6;
}

.kafka-topic-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

@media (max-width: 640px) {
  .kafka-topic-dialog__grid,
  .kafka-topic-dialog__mode-options {
    grid-template-columns: 1fr;
  }
}
</style>
