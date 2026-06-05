<template>
  <section class="kafka-preview-panel">
    <WorkbenchStreamToolbar
      :title="`${mapping.name || mapping.topic} 消息预览`"
      :subtitle="mapping.topic"
      :loading="loading"
      :status-label="statusLabel"
      :status-tone="statusTone"
      :search="search"
      :limit="displayLimit"
      :format-json="formatJson"
      :auto-scroll="autoScroll"
      :show-timestamp="showTimestamp"
      @update:search="search = $event"
      @update:limit="displayLimit = $event"
      @update:format-json="formatJson = $event"
      @update:auto-scroll="autoScroll = $event"
      @update:show-timestamp="showTimestamp = $event"
      @refresh="runPreview"
      @clear="clearSamples"
    />
    <WorkbenchStreamMessageList
      :messages="displayMessages"
      :loading="loading"
      :format-json="formatJson"
      :show-timestamp="showTimestamp"
      empty-text="暂无 Kafka 样本"
      empty-hint="点击刷新执行一次短时预览"
      @copy="copyMessage"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import dataAPI from '@/api/data.api'
import WorkbenchStreamMessageList from '@/components/workbench/WorkbenchStreamMessageList.vue'
import WorkbenchStreamToolbar from '@/components/workbench/WorkbenchStreamToolbar.vue'
import { getApiErrorMessage } from '@/utils/request'
import type { KafkaPreview, KafkaPreviewSample, KafkaTopicMapping } from './types'

const props = defineProps<{
  projectId: string
  mapping: KafkaTopicMapping
  connected: boolean
}>()

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
const autoScroll = ref(false)
const showTimestamp = ref(true)
const samples = ref<KafkaPreviewSample[]>([])
const lastPreview = ref<KafkaPreview | null>(null)
const lastError = ref('')

const statusLabel = computed(() => {
  if (loading.value) return '预览中'
  if (lastError.value) return '预览失败'
  if (samples.value.length > 0) return `样本 ${samples.value.length}`
  return '待预览'
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
      topic: `${sample.topic || props.mapping.topic} / p${sample.partition ?? '-'}`,
      payload: sample.value,
      timestamp: sample.timestamp,
      qos: 0,
      raw: sample,
    }))
})

const runPreview = async () => {
  if (!props.connected) {
    ElMessage.warning('请先连接 Kafka 后再预览消息')
    return
  }
  loading.value = true
  lastError.value = ''
  try {
    const preview = await dataAPI.previewKafkaTopicMapping(props.projectId, props.mapping.id, {
      limit: displayLimit.value,
      timeoutMs: props.mapping.timeoutMs,
      decode: props.mapping.decode,
    })
    lastPreview.value = preview
    samples.value = preview.samples || []
    emit('samples', { mappingId: String(props.mapping.id), samples: samples.value, preview })
  } catch (error) {
    lastError.value = getApiErrorMessage(error, 'Kafka 预览失败')
    samples.value = []
    ElMessage.error(lastError.value)
  } finally {
    loading.value = false
  }
}

const clearSamples = () => {
  samples.value = []
  lastPreview.value = null
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

onMounted(() => {
  if (props.connected) void runPreview()
})

watch(
  () => props.mapping.id,
  () => {
    clearSamples()
    if (props.connected) void runPreview()
  },
)

watch(
  () => props.connected,
  (connected) => {
    if (!connected) {
      clearSamples()
    }
  },
)
</script>

<style scoped>
.kafka-preview-panel {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}
</style>
