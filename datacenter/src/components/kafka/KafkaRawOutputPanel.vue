<template>
  <section class="kafka-raw-output-panel">
    <header class="kafka-raw-output-panel__summary">
      <div class="kafka-raw-output-panel__title">
        <strong>{{ mapping.name || mapping.topic }}</strong>
        <span>{{ mapping.topic }}</span>
      </div>
      <div class="kafka-raw-output-panel__chips">
        <WorkbenchStatusPill label="整包数据点" tone="info" />
        <WorkbenchStatusPill
          :label="mapping.rawOutputScope === 'full_message' ? '完整消息' : '消息体'"
          tone="info"
        />
        <WorkbenchStatusPill :label="partitionLabel" tone="info" />
        <WorkbenchStatusPill
          :label="mapping.rawDataPointPath || '未生成数据点'"
          :tone="mapping.rawDataPointPath ? 'success' : 'neutral'"
        />
      </div>
    </header>

    <WorkbenchStreamToolbar
      title="整包样本测试"
      :subtitle="toolbarSubtitle"
      :loading="loading"
      :status-label="statusLabel"
      :status-tone="statusTone"
      :search="search"
      :limit="displayLimit"
      :format-json="formatJson"
      :auto-scroll="false"
      :show-timestamp="showTimestamp"
      @update:search="search = $event"
      @update:limit="displayLimit = $event"
      @update:format-json="formatJson = $event"
      @update:show-timestamp="showTimestamp = $event"
      @refresh="pullSamples"
      @clear="clearSamples"
    />
    <WorkbenchStreamMessageList
      :messages="displayMessages"
      :loading="loading"
      :format-json="formatJson"
      :show-timestamp="showTimestamp"
      empty-text="暂无 Kafka 样本"
      empty-hint="点击拉取样本执行一次临时读取"
      @copy="copyMessage"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import dataAPI from '@/api/data.api'
import WorkbenchStreamMessageList from '@/components/workbench/WorkbenchStreamMessageList.vue'
import WorkbenchStreamToolbar from '@/components/workbench/WorkbenchStreamToolbar.vue'
import WorkbenchStatusPill from '@/components/workbench/WorkbenchStatusPill.vue'
import { getApiErrorMessage } from '@/utils/request'
import type { KafkaPreview, KafkaPreviewSample, KafkaTopicMapping } from './types'

const props = withDefaults(
  defineProps<{
    projectId: string
    mapping: KafkaTopicMapping
    pullRequestId?: number
  }>(),
  {
    pullRequestId: 0,
  },
)

const emit = defineEmits<{
  (
    event: 'samples',
    payload: { mappingId: string; samples: KafkaPreviewSample[]; preview: KafkaPreview },
  ): void
}>()

const loading = ref(false)
const search = ref('')
const displayLimit = ref(100)
const formatJson = ref(true)
const showTimestamp = ref(true)
const samples = ref<KafkaPreviewSample[]>([])
const lastError = ref('')

const toolbarSubtitle = computed(() => {
  const group = props.mapping.consumerGroup || '默认消费组'
  return `${props.mapping.topic} / ${group}`
})
const partitionLabel = computed(() =>
  props.mapping.partitionMode === 'single'
    ? `partition ${props.mapping.partition ?? 0}`
    : '全部分区',
)
const statusLabel = computed(() => {
  if (loading.value) return '拉取中'
  if (lastError.value) return '拉取失败'
  if (samples.value.length > 0) return `样本 ${samples.value.length}`
  return '待拉取'
})
const statusTone = computed(() => {
  if (lastError.value) return 'danger'
  if (samples.value.length > 0) return 'success'
  return 'info'
})
const displayMessages = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  return samples.value
    .filter((sample) => {
      if (!keyword) return true
      return `${sample.topic || ''} ${sample.key || ''} ${formatPayload(sample.value)}`
        .toLowerCase()
        .includes(keyword)
    })
    .slice(0, displayLimit.value)
    .map((sample) => ({
      id: `${sample.topic || props.mapping.topic}-${sample.partition ?? '-'}-${sample.offset ?? '-'}`,
      topic: `${sample.topic || props.mapping.topic} / p${sample.partition ?? '-'} / o${sample.offset ?? '-'}`,
      payload: sample.value,
      timestamp: sample.timestamp,
      qos: 0,
    }))
})

const pullSamples = async () => {
  loading.value = true
  lastError.value = ''
  try {
    const preview = await dataAPI.previewKafkaTopicMapping(props.projectId, props.mapping.id, {
      limit: displayLimit.value,
      timeoutMs: props.mapping.timeoutMs,
      decode: props.mapping.decode,
    })
    samples.value = preview.samples || []
    emit('samples', { mappingId: String(props.mapping.id), samples: samples.value, preview })
    if (samples.value.length === 0) {
      ElMessage.warning('本次未拉取到样本，可调整起始位置、样本上限或超时后重试')
      return
    }
    ElMessage.success(`已拉取 ${samples.value.length} 条样本`)
  } catch (error) {
    lastError.value = getApiErrorMessage(error, 'Kafka 样本拉取失败')
    ElMessage.error(lastError.value)
  } finally {
    loading.value = false
  }
}

const clearSamples = () => {
  samples.value = []
  lastError.value = ''
}

const copyMessage = async (message: { payload?: unknown }) => {
  await navigator.clipboard.writeText(formatPayload(message.payload))
  ElMessage.success('Payload 已复制')
}

const formatPayload = (value: unknown) => {
  if (typeof value === 'string') return value
  try {
    return formatJson.value ? JSON.stringify(value, null, 2) : JSON.stringify(value)
  } catch {
    return String(value)
  }
}

watch(
  () => props.mapping.id,
  () => {
    clearSamples()
  },
)

watch(
  () => props.pullRequestId,
  (value, oldValue) => {
    if (value !== oldValue && value > 0) void pullSamples()
  },
)
</script>

<style scoped>
.kafka-raw-output-panel {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}

.kafka-raw-output-panel__summary {
  min-height: 52px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 9px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.kafka-raw-output-panel__title {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.kafka-raw-output-panel__title strong,
.kafka-raw-output-panel__title span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kafka-raw-output-panel__title strong {
  color: var(--dc-text);
  font-size: 13px;
}

.kafka-raw-output-panel__title span {
  color: var(--dc-text-muted);
  font-family: var(--dc-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  font-size: 11px;
}

.kafka-raw-output-panel__chips {
  min-width: 0;
  display: inline-flex;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}
</style>
