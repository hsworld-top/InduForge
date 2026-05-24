<template>
  <section class="protocol-workbench">
    <div class="protocol-workbench__body">
      <aside class="protocol-workbench__side">
        <WorkbenchSourceHeader
          :title="connection.name || '未命名接入源'"
          fallback-title="未命名接入源"
          :status-label="statusLabel"
          :status-tone="statusTone"
          :meta="sourceMetaRows"
          @back="$emit('back')"
        >
          <template #actions>
            <el-input-number
              v-model="limit"
              :min="1"
              :max="100"
              :step="1"
              size="small"
              controls-position="right"
              class="protocol-workbench__limit"
            />
            <el-button size="small" :loading="testing" @click="runConnectionTest">
              测试连接
            </el-button>
            <el-button type="primary" size="small" :loading="previewing" @click="runPreview">
              获取样本
            </el-button>
          </template>
        </WorkbenchSourceHeader>

        <section class="protocol-workbench__panel">
          <div class="protocol-workbench__panel-title">连接配置</div>
          <dl class="protocol-workbench__meta">
            <template v-for="row in configRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd :title="row.value">{{ row.value }}</dd>
            </template>
          </dl>
        </section>

        <section class="protocol-workbench__panel">
          <div class="protocol-workbench__panel-title">诊断</div>
          <dl class="protocol-workbench__meta">
            <template v-for="row in diagnosticRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd :title="row.value">{{ row.value }}</dd>
            </template>
          </dl>
        </section>
      </aside>

      <main class="protocol-workbench__main">
        <WorkbenchStreamToolbar
          :title="`${protocolLabel} 样本`"
          :subtitle="endpointText"
          :icon="toolbarIcon"
          :status-label="previewStatusLabel"
          :status-tone="previewStatusTone"
          v-model:search="search"
          v-model:limit="displayLimit"
          v-model:format-json="formatJson"
          v-model:auto-scroll="autoScroll"
          v-model:show-timestamp="showTimestamp"
          :loading="previewing"
          @refresh="runPreview"
          @clear="clearSamples"
        />

        <WorkbenchStreamMessageList
          ref="messageListRef"
          :messages="filteredMessages"
          :loading="previewing"
          :show-timestamp="showTimestamp"
          :format-json="formatJson"
          empty-text="暂无样本"
          empty-hint="点击获取样本执行一次短时真实预览"
          @copy="copyMessage"
        />
      </main>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, markRaw, nextTick, ref } from 'vue'
import { ElMessage } from 'element-plus'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'
import WorkbenchStreamMessageList from '@/components/workbench/WorkbenchStreamMessageList.vue'
import WorkbenchStreamToolbar from '@/components/workbench/WorkbenchStreamToolbar.vue'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import IconTablerBraces from '~icons/tabler/braces'
import IconTablerDatabase from '~icons/tabler/database'
import IconTablerWebhook from '~icons/tabler/webhook'
import IconTablerWorldWww from '~icons/tabler/world-www'

type AccessSourceConnection = {
  id: string
  name?: string
  type?: string
  status?: string
  config?: Record<string, unknown>
}

type StreamMessage = {
  id: string
  topic: string
  payload: unknown
  qos: number
  timestamp: string
}

const props = defineProps<{
  connection: AccessSourceConnection
  projectId: string
}>()

defineEmits<{
  (event: 'back'): void
}>()

const previewing = ref(false)
const testing = ref(false)
const search = ref('')
const limit = ref(10)
const displayLimit = ref(100)
const formatJson = ref(true)
const autoScroll = ref(true)
const showTimestamp = ref(true)
const samples = ref<StreamMessage[]>([])
const diagnostics = ref<Record<string, unknown>>({})
const lastError = ref('')
const messageListRef = ref<InstanceType<typeof WorkbenchStreamMessageList> | null>(null)

const protocolLabelMap: Record<string, string> = {
  kafka: 'Kafka',
  http: 'HTTP',
  websocket: 'WebSocket',
  redis: 'Redis',
}

const protocolLabel = computed(() => protocolLabelMap[props.connection.type || ''] || '协议')

const toolbarIcon = computed(() => {
  if (props.connection.type === 'http') return markRaw(IconTablerWorldWww)
  if (props.connection.type === 'websocket') return markRaw(IconTablerWebhook)
  if (props.connection.type === 'redis') return markRaw(IconTablerDatabase)
  return markRaw(IconTablerBraces)
})

const config = computed(() => props.connection.config || {})

const statusLabel = computed(() => {
  const labels: Record<string, string> = {
    connected: '在线',
    disconnected: '离线',
    error: '异常',
    unknown: '未知',
  }
  return labels[props.connection.status || 'unknown'] || '未知'
})

const statusTone = computed(() => {
  if (props.connection.status === 'connected') return 'success'
  if (props.connection.status === 'error') return 'danger'
  if (props.connection.status === 'disconnected') return 'warning'
  return 'neutral'
})

const previewStatusLabel = computed(() => {
  if (previewing.value) return '预览中'
  if (lastError.value) return '预览失败'
  if (samples.value.length > 0) return `样本 ${samples.value.length}`
  return '待预览'
})

const previewStatusTone = computed(() => {
  if (previewing.value) return 'info'
  if (lastError.value) return 'danger'
  if (samples.value.length > 0) return 'success'
  return 'neutral'
})

const endpointText = computed(() => {
  if (props.connection.type === 'http') {
    return [config.value.method || 'GET', config.value.baseUrl].filter(Boolean).join(' ')
  }
  if (props.connection.type === 'websocket') {
    return [config.value.url, config.value.topic].filter(Boolean).join(' / ')
  }
  if (props.connection.type === 'redis') {
    return [config.value.address, config.value.keyPattern || '*'].filter(Boolean).join(' / ')
  }
  if (props.connection.type === 'kafka') {
    return [config.value.brokers, config.value.topic].filter(Boolean).join(' / ')
  }
  return '等待配置'
})

const sourceMetaRows = computed(() => [
  { label: '类型', value: protocolLabel.value },
  { label: '地址', value: endpointText.value },
])

const configRows = computed(() => {
  const c = config.value
  if (props.connection.type === 'http') {
    return [
      { label: '方法', value: String(c.method || 'GET') },
      { label: 'URL', value: String(c.baseUrl || '未配置') },
      { label: '超时', value: `${c.timeoutMs || 5000}ms` },
      { label: 'Header', value: summarizeObject(c.headers) },
    ]
  }
  if (props.connection.type === 'websocket') {
    return [
      { label: 'URL', value: String(c.url || '未配置') },
      { label: 'Topic', value: String(c.topic || '可选') },
      { label: '心跳', value: `${c.heartbeatIntervalMs || 30000}ms` },
      { label: 'Header', value: summarizeObject(c.headers) },
    ]
  }
  if (props.connection.type === 'redis') {
    return [
      { label: '模式', value: String(c.mode || 'standalone') },
      { label: '地址', value: String(c.address || '未配置') },
      { label: 'DB', value: String(c.db ?? 0) },
      { label: 'Key', value: String(c.keyPattern || '*') },
    ]
  }
  return [
    { label: 'Broker', value: String(c.brokers || '未配置') },
    { label: 'Topic', value: String(c.topic || '未配置') },
    { label: 'Group', value: String(c.consumerGroup || '未配置') },
    { label: 'Offset', value: String(c.startPosition || 'latest') },
  ]
})

const diagnosticRows = computed(() => {
  const entries = Object.entries(diagnostics.value || {})
  if (lastError.value) {
    return [{ label: '错误', value: lastError.value }]
  }
  if (entries.length === 0) {
    return [{ label: '状态', value: '尚未执行预览' }]
  }
  return entries.map(([key, value]) => ({
    label: key,
    value: formatDiagnosticValue(value),
  }))
})

const filteredMessages = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  const list = samples.value.slice(0, displayLimit.value)
  if (!keyword) return list
  return list.filter((message) => {
    const haystack = `${message.topic} ${formatPayload(message.payload)}`.toLowerCase()
    return haystack.includes(keyword)
  })
})

const runConnectionTest = async () => {
  testing.value = true
  try {
    const response = await dataAPI.testConnection(props.projectId, {
      type: props.connection.type,
      config: config.value,
    })
    const result = response?.data || response || {}
    ElMessage.success(result.message || '连接测试通过')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '连接测试失败'))
  } finally {
    testing.value = false
  }
}

const runPreview = async () => {
  previewing.value = true
  lastError.value = ''
  try {
    const response = await dataAPI.previewProtocol(props.projectId, props.connection.id, {
      limit: limit.value,
      timeoutMs: 5000,
      options: previewOptions.value,
    })
    const payload = response?.data || response || {}
    diagnostics.value = payload.diagnostics || {}
    samples.value = normalizeSamples(payload.samples || [], payload.rawPayload)
    if (autoScroll.value) {
      await nextTick()
      await messageListRef.value?.scrollToTop?.()
    }
    ElMessage.success(`已获取 ${samples.value.length} 条样本`)
  } catch (error) {
    lastError.value = getApiErrorMessage(error, '协议预览失败')
    diagnostics.value = { error: lastError.value }
    ElMessage.error(lastError.value)
  } finally {
    previewing.value = false
  }
}

const previewOptions = computed(() => {
  if (props.connection.type !== 'redis') return {}
  const options = config.value.options
  return options && typeof options === 'object' ? options : {}
})

const clearSamples = () => {
  samples.value = []
  diagnostics.value = {}
  lastError.value = ''
}

const copyMessage = async (message: StreamMessage) => {
  await navigator.clipboard.writeText(formatPayload(message.payload))
  ElMessage.success('Payload 已复制')
}

const normalizeSamples = (rawSamples: unknown[], rawPayload: unknown): StreamMessage[] => {
  const now = new Date().toISOString()
  const source = rawSamples.length > 0 ? rawSamples : rawPayload !== undefined ? [rawPayload] : []
  return source.map((sample, index) => ({
    id: `${props.connection.id}-${Date.now()}-${index}`,
    topic: resolveSampleTopic(sample, index),
    payload: sample,
    qos: 0,
    timestamp: now,
  }))
}

const resolveSampleTopic = (sample: unknown, index: number) => {
  if (sample && typeof sample === 'object') {
    const mapped = sample as Record<string, unknown>
    if (typeof mapped.topic === 'string') return mapped.topic
    if (typeof mapped.key === 'string') return mapped.key
  }
  return `${props.connection.type || 'sample'}#${index + 1}`
}

const summarizeObject = (value: unknown) => {
  if (!value || typeof value !== 'object') return '{}'
  const keys = Object.keys(value as Record<string, unknown>)
  return keys.length ? keys.join(', ') : '{}'
}

const formatDiagnosticValue = (value: unknown) => {
  if (value === null || value === undefined) return '-'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

const formatPayload = (value: unknown) => {
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}
</script>

<style scoped>
.protocol-workbench {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
}

.protocol-workbench__body {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
}

.protocol-workbench__side {
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow: auto;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
}

.protocol-workbench__panel {
  margin: 0 12px;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
}

.protocol-workbench__panel:last-child {
  margin-bottom: 12px;
}

.protocol-workbench__limit {
  width: 88px;
}

.protocol-workbench__panel-title {
  margin-bottom: 10px;
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.protocol-workbench__meta {
  display: grid;
  grid-template-columns: 58px minmax(0, 1fr);
  gap: 8px 10px;
  margin: 0;
}

.protocol-workbench__meta dt {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.protocol-workbench__meta dd {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  color: var(--dc-text-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.protocol-workbench__main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

@media (max-width: 920px) {
  .protocol-workbench__body {
    grid-template-columns: 1fr;
  }

  .protocol-workbench__side {
    max-height: 220px;
    border-right: 0;
    border-bottom: 1px solid var(--dc-border);
  }
}
</style>
