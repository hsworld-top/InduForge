<template>
  <section class="kafka-raw-output-panel">
    <section class="kafka-raw-output-panel__pullbar">
      <div class="kafka-raw-output-panel__pullbar-title">
        <span>{{ mapping.rawDataPointPath || '未生成数据点' }}</span>
      </div>
      <el-tooltip content="点击测试 Broker 网络连通性" placement="top">
        <WorkbenchStatusPill
          :label="networkStatusLabel"
          :tone="networkStatusTone"
          clickable
          :disabled="testingNetwork"
          @click="() => testNetwork(true)"
        />
      </el-tooltip>
      <div class="kafka-raw-output-panel__pullbar-actions">
        <el-input
          v-model="search"
          class="kafka-raw-output-panel__search"
          size="small"
          clearable
          placeholder="搜索 Topic、Key 或 Payload"
        >
          <template #prefix>
            <IconTablerSearch />
          </template>
        </el-input>
        <label class="kafka-raw-output-panel__limit">
          <span>拉取条数</span>
          <el-input-number
            v-model="pullLimit"
            :min="1"
            :max="1000"
            :step="10"
            size="small"
            controls-position="right"
          />
        </label>
        <el-button type="primary" size="small" :loading="loading" @click="pullSamples">
          <IconTablerDownload class="kafka-raw-output-panel__button-icon" />
          拉取样本
        </el-button>
        <el-tooltip :content="formatJson ? '关闭 JSON 格式化' : '开启 JSON 格式化'" placement="top">
          <button
            type="button"
            class="kafka-raw-output-panel__icon-btn"
            :class="{ 'is-active': formatJson }"
            @click="formatJson = !formatJson"
          >
            <IconTablerBraces />
          </button>
        </el-tooltip>
        <el-tooltip content="清空样本" placement="top">
          <button type="button" class="kafka-raw-output-panel__icon-btn" @click="clearSamples">
            <IconTablerTrash />
          </button>
        </el-tooltip>
      </div>
    </section>
    <KafkaSampleMessageList
      :messages="displayMessages"
      :loading="loading"
      :format-json="formatJson"
      empty-text="暂无 Kafka 样本"
      empty-hint="设置拉取条数后点击拉取样本"
      @copy="copyMessage"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import IconTablerBraces from '~icons/tabler/braces'
import IconTablerDownload from '~icons/tabler/download'
import IconTablerSearch from '~icons/tabler/search'
import IconTablerTrash from '~icons/tabler/trash'
import dataAPI from '@/api/data.api'
import WorkbenchStatusPill from '@/components/workbench/WorkbenchStatusPill.vue'
import { getApiErrorMessage } from '@/utils/request'
import KafkaSampleMessageList from './KafkaSampleMessageList.vue'
import type { KafkaPreview, KafkaPreviewSample, KafkaTopicMapping } from './types'

const props = withDefaults(
  defineProps<{
    projectId: string
    connectionId: string
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
const testingNetwork = ref(false)
const search = ref('')
const pullLimit = ref(1)
const formatJson = ref(true)
const samples = ref<KafkaPreviewSample[]>([])
const networkStatus = ref<'unknown' | 'testing' | 'available' | 'unavailable'>('unknown')

const networkStatusLabel = computed(() => {
  if (networkStatus.value === 'testing') return '测试中'
  if (networkStatus.value === 'available') return '可用'
  if (networkStatus.value === 'unavailable') return '不可用'
  return '未测试'
})
const networkStatusTone = computed(() => {
  if (networkStatus.value === 'available') return 'success'
  if (networkStatus.value === 'unavailable') return 'danger'
  if (networkStatus.value === 'testing') return 'info'
  return 'neutral'
})
const displayMessages = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  return samples.value
    .filter((sample) => {
      if (!keyword) return true
      return `${sample.topic || ''} ${sample.key || ''} ${sample.partition ?? ''} ${sample.offset ?? ''} ${formatPayload(sample.value)}`
        .toLowerCase()
        .includes(keyword)
    })
    .map((sample) => ({
      id: `${sample.topic || props.mapping.topic}-${sample.partition ?? '-'}-${sample.offset ?? '-'}`,
      topic: sample.topic || props.mapping.topic,
      partition: sample.partition,
      offset: sample.offset,
      key: sample.key,
      payload: sample.value,
      timestamp: sample.timestamp,
    }))
})

const pullSamples = async () => {
  loading.value = true
  try {
    const preview = await dataAPI.previewKafkaTopicMapping(props.projectId, props.mapping.id, {
      limit: pullLimit.value,
      timeoutMs: props.mapping.timeoutMs,
      decode: props.mapping.decode,
    })
    const nextSamples = preview.samples || []
    const seen = new Set(
      samples.value.map(
        (sample) =>
          `${sample.topic || props.mapping.topic}-${sample.partition ?? '-'}-${sample.offset ?? '-'}`,
      ),
    )
    const uniqueNextSamples = nextSamples.filter((sample) => {
      const key = `${sample.topic || props.mapping.topic}-${sample.partition ?? '-'}-${sample.offset ?? '-'}`
      if (seen.has(key)) return false
      seen.add(key)
      return true
    })
    samples.value = [
      ...uniqueNextSamples,
      ...samples.value,
    ]
    emit('samples', { mappingId: String(props.mapping.id), samples: samples.value, preview })
    if (nextSamples.length === 0) {
      ElMessage.warning('本次未拉取到样本，可调整起始位置、样本上限或超时后重试')
      return
    }
    ElMessage.success(
      `本次拉取 ${nextSamples.length} 条样本，新增 ${uniqueNextSamples.length} 条，当前共 ${samples.value.length} 条`,
    )
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, 'Kafka 样本拉取失败'))
  } finally {
    loading.value = false
  }
}

const testNetwork = async (notify = true) => {
  if (testingNetwork.value) return
  testingNetwork.value = true
  networkStatus.value = 'testing'
  try {
    await dataAPI.previewKafkaConnection(props.projectId, props.connectionId, {
      limit: 1,
      timeoutMs: 1000,
      probe: true,
    })
    networkStatus.value = 'available'
    if (notify) ElMessage.success('Kafka Broker 网络可用')
  } catch (error) {
    networkStatus.value = 'unavailable'
    if (notify) ElMessage.error(getApiErrorMessage(error, 'Kafka Broker 网络不可用'))
  } finally {
    testingNetwork.value = false
  }
}

const clearSamples = () => {
  samples.value = []
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
  void testNetwork(false)
})

watch(
  () => [props.mapping.id, props.connectionId] as const,
  () => {
    pullLimit.value = 1
    networkStatus.value = 'unknown'
    clearSamples()
    void testNetwork(false)
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

.kafka-raw-output-panel__pullbar {
  min-height: 54px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.kafka-raw-output-panel__pullbar-title {
  min-width: 180px;
}

.kafka-raw-output-panel__pullbar-title span {
  min-width: 0;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kafka-raw-output-panel__pullbar-title span {
  color: var(--dc-text-secondary);
  font-family: var(--dc-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  font-size: 12px;
  font-weight: 700;
}

.kafka-raw-output-panel__pullbar-actions {
  min-width: 0;
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 7px;
  flex-wrap: wrap;
}

.kafka-raw-output-panel__search {
  width: 240px;
}

.kafka-raw-output-panel__limit {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
  white-space: nowrap;
}

.kafka-raw-output-panel__limit :deep(.el-input-number) {
  width: 108px;
}

.kafka-raw-output-panel__button-icon {
  width: 14px;
  height: 14px;
  margin-right: 4px;
}

.kafka-raw-output-panel__icon-btn {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.kafka-raw-output-panel__icon-btn:hover,
.kafka-raw-output-panel__icon-btn.is-active {
  border-color: color-mix(in oklch, var(--dc-primary) 30%, var(--dc-border));
  color: var(--dc-primary);
}

.kafka-raw-output-panel__icon-btn svg,
.kafka-raw-output-panel :deep(.el-input__prefix svg) {
  width: 14px;
  height: 14px;
}

@media (max-width: 980px) {
  .kafka-raw-output-panel__pullbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .kafka-raw-output-panel__pullbar-actions {
    justify-content: flex-start;
  }

  .kafka-raw-output-panel__pullbar-title,
  .kafka-raw-output-panel__search {
    width: 100%;
  }
}
</style>
