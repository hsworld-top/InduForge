<template>
  <aside class="access-source-detail">
    <template v-if="connection">
      <div class="access-source-detail__hero">
        <div class="access-source-detail__hero-main">
          <span
            class="access-source-detail__status"
            :class="`is-${connection.status || 'unknown'}`"
          >
            {{ statusText }}
          </span>
          <h3>{{ connection.name || '-' }}</h3>
          <p>{{ typeLabel }}</p>
        </div>
        <el-button type="primary" size="small" @click="$emit('open', connection)">
          {{ openButtonLabel }}
        </el-button>
      </div>

      <div class="access-source-detail__metrics">
        <div v-for="metric in metrics" :key="metric.label">
          <span>{{ metric.label }}</span>
          <strong>{{ metric.value }}</strong>
        </div>
      </div>

      <div class="access-source-detail__tabs" role="tablist">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          type="button"
          :class="{ 'is-active': activeTab === tab.value }"
          @click="activeTab = tab.value"
        >
          {{ tab.label }}
        </button>
      </div>

      <div class="access-source-detail__body">
        <template v-if="activeTab === 'overview'">
          <section class="access-source-detail__section">
            <div class="access-source-detail__section-title">接入摘要</div>
            <dl class="access-source-detail__list">
              <div>
                <dt>地址</dt>
                <dd>{{ endpointText }}</dd>
              </div>
              <div>
                <dt>数据点来源</dt>
                <dd>{{ sourceModeText }}</dd>
              </div>
              <div>
                <dt>当前能力</dt>
                <dd>{{ capabilityText }}</dd>
              </div>
            </dl>
          </section>

          <section class="access-source-detail__section">
            <div class="access-source-detail__section-title">下一步操作</div>
            <button
              type="button"
              class="access-source-detail__primary-card"
              @click="$emit('open', connection)"
            >
              <strong>{{ openButtonLabel }}</strong>
              <span>{{ openHint }}</span>
            </button>
          </section>
        </template>

        <template v-else-if="activeTab === 'config'">
          <section class="access-source-detail__section">
            <div class="access-source-detail__section-title">连接配置</div>
            <dl class="access-source-detail__list">
              <div v-for="item in configRows" :key="item.label">
                <dt>{{ item.label }}</dt>
                <dd>{{ item.value }}</dd>
              </div>
            </dl>
          </section>
        </template>

        <template v-else-if="activeTab === 'mapping'">
          <section class="access-source-detail__section">
            <div class="access-source-detail__section-title">查询与数据点</div>
            <div v-if="detailLoading" class="access-source-detail__empty-line">加载映射中...</div>
            <template v-else-if="isSqlConnection">
              <div
                v-for="item in queryMappings"
                :key="item.query.id"
                class="access-source-detail__resource"
              >
                <div>
                  <strong>{{ item.query.name }}</strong>
                  <span>{{ item.point?.path || '保存后会自动生成 db.query 数据点' }}</span>
                </div>
                <small :class="{ 'is-muted': !item.point }">
                  {{ item.point ? resolvePointStatus(item.point) : '未同步' }}
                </small>
              </div>
              <div v-if="queryMappings.length === 0" class="access-source-detail__empty-line">
                暂无保存查询，进入查询工作台保存 SQL 后会生成数据点。
              </div>
            </template>
            <div v-else class="access-source-detail__empty-line">
              {{ mappingHint }}
            </div>
          </section>
        </template>

        <template v-else-if="activeTab === 'preview'">
          <section class="access-source-detail__section">
            <div class="access-source-detail__section-title">开发态预览</div>
            <div class="access-source-detail__preview-box">
              <strong>{{ previewTitle }}</strong>
              <span>{{ previewHint }}</span>
              <el-button size="small" @click="$emit('open', connection)"> 进入工作台 </el-button>
            </div>
          </section>
        </template>

        <template v-else-if="activeTab === 'datapoints'">
          <section class="access-source-detail__section">
            <div class="access-source-detail__section-title">自动数据点</div>
            <div v-if="detailLoading" class="access-source-detail__empty-line">加载数据点中...</div>
            <template v-else>
              <button
                v-for="point in detailDataPoints"
                :key="point.id"
                type="button"
                class="access-source-detail__resource"
                @click="copyPointPath(point.path)"
              >
                <div>
                  <strong>{{ point.name }}</strong>
                  <span>{{ point.path }}</span>
                </div>
                <small>{{ resolvePointStatus(point) }}</small>
              </button>
              <div v-if="detailDataPoints.length === 0" class="access-source-detail__empty-line">
                {{ datapointEmptyText }}
              </div>
            </template>
          </section>
        </template>

        <template v-else>
          <section class="access-source-detail__section">
            <div class="access-source-detail__section-title">最近记录</div>
            <dl class="access-source-detail__list">
              <div>
                <dt>连接状态</dt>
                <dd>{{ statusText }}</dd>
              </div>
              <div>
                <dt>最后错误</dt>
                <dd>{{ connection.lastErrorMessage || '-' }}</dd>
              </div>
              <div>
                <dt>更新时间</dt>
                <dd>{{ connection.updatedAt || '-' }}</dd>
              </div>
            </dl>
          </section>
        </template>
      </div>
    </template>

    <div v-else class="access-source-detail__empty">
      <div class="access-source-detail__empty-mark" />
      <strong>选择一个接入源</strong>
      <span>这里会展示配置摘要、查询映射、预览入口和自动数据点。</span>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'

type AccessSourceConnection = {
  id: string
  name?: string
  type?: string
  status?: string
  datapointCount?: number
  dataPointCount?: number
  lastErrorMessage?: string
  updatedAt?: string
  relationalConfig?: {
    dbType?: string
    host?: string
    port?: number | string
    database?: string
  }
  mqttConfig?: {
    protocol?: string
    brokerUrl?: string
    host?: string
    port?: number | string
    topic?: string
    defaultTopic?: string
  }
  config?: Record<string, any>
}

const props = defineProps<{
  connection?: AccessSourceConnection | null
  projectId?: string | number | null
}>()

defineEmits<{
  (event: 'open', connection: AccessSourceConnection): void
}>()

const activeTab = ref('overview')
const detailLoading = ref(false)
const detailQueries = ref<any[]>([])
const detailDataPoints = ref<any[]>([])

const tabs = [
  { label: '概览', value: 'overview' },
  { label: '配置', value: 'config' },
  { label: '映射', value: 'mapping' },
  { label: '预览', value: 'preview' },
  { label: '数据点', value: 'datapoints' },
  { label: '记录', value: 'logs' },
]

const dbTypeLabels: Record<string, string> = {
  mysql: 'MySQL',
  postgresql: 'PostgreSQL',
  sqlserver: 'SQL Server',
}

const statusLabels: Record<string, string> = {
  connected: '在线',
  disconnected: '离线',
  error: '异常',
  unknown: '未知',
}

const isSqlConnection = computed(() => {
  const type = props.connection?.type || ''
  return type === 'relational' || ['mysql', 'postgresql', 'sqlserver', 'tdengine'].includes(type)
})

const typeLabel = computed(() => {
  const connection = props.connection
  if (!connection) return '-'

  if (connection.type === 'relational') {
    const dbType = connection.relationalConfig?.dbType || ''
    return dbTypeLabels[dbType] || dbType || '数据库'
  }

  if (connection.type === 'mqtt') return 'MQTT Broker'

  const protocolLabels: Record<string, string> = {
    kafka: 'Kafka Topic',
    http: 'HTTP Source',
    websocket: 'WebSocket',
    redis: 'Redis',
    opcua: 'OPC UA',
    s7: 'Siemens S7',
    modbus: 'Modbus',
    tdengine: 'TDengine',
  }
  if (connection.type && protocolLabels[connection.type]) {
    return protocolLabels[connection.type]
  }

  return connection.type || '未知类型'
})

const endpointText = computed(() => {
  const connection = props.connection
  if (!connection) return '-'

  if (connection.type === 'relational') {
    const config = connection.relationalConfig
    if (!config) return '未配置'
    const host = [config.host, config.port].filter(Boolean).join(':')
    return [host, config.database].filter(Boolean).join(' / ') || '未配置'
  }

  if (connection.type === 'mqtt') {
    const config = connection.mqttConfig
    if (!config) return '未配置'
    const endpoint = [config.brokerUrl || config.host, config.port].filter(Boolean).join(':')
    return [endpoint, config.topic || config.defaultTopic].filter(Boolean).join(' / ') || '未配置'
  }

  const config = connection.config || {}
  if (connection.type === 'kafka') {
    return [config.brokers, config.topic].filter(Boolean).join(' / ') || '未配置'
  }
  if (connection.type === 'http') {
    return '请求在工作台配置'
  }
  if (connection.type === 'websocket') {
    return [config.url, config.topic].filter(Boolean).join(' / ') || '未配置'
  }
  if (connection.type === 'redis') {
    return [config.address, config.keyPattern || '*'].filter(Boolean).join(' / ') || '未配置'
  }
  if (connection.type === 'opcua') {
    return [config.endpoint, config.securityMode || 'none'].filter(Boolean).join(' / ') || '未配置'
  }
  if (connection.type === 's7') {
    return [config.host, config.port].filter(Boolean).join(':') || '未配置'
  }
  if (connection.type === 'modbus') {
    if (config.mode === 'rtu') return 'RTU / 串口配置'
    return [config.host, config.port].filter(Boolean).join(':') || '未配置'
  }
  if (connection.type === 'tdengine') {
    return [config.dsn, config.database].filter(Boolean).join(' / ') || '未配置'
  }

  return '未配置'
})

const statusText = computed(
  () => statusLabels[props.connection?.status || 'unknown'] || props.connection?.status || '未知',
)

const fallbackDatapointCount = computed(() => {
  const connection = props.connection
  return connection?.datapointCount ?? connection?.dataPointCount
})

const datapointCountText = computed(() => {
  const loaded = detailDataPoints.value.length
  if (loaded > 0) return `${loaded}`
  return typeof fallbackDatapointCount.value === 'number' ? `${fallbackDatapointCount.value}` : '0'
})

const metrics = computed(() => [
  { label: '状态', value: statusText.value },
  {
    label: '保存查询',
    value: isSqlConnection.value ? `${detailQueries.value.length}` : '-',
  },
  { label: '数据点', value: datapointCountText.value },
  { label: '类型', value: typeLabel.value },
])

const sourceModeText = computed(() => {
  if (isSqlConnection.value) return '保存 SQL 查询后自动生成 db.query 数据点'
  if (props.connection?.type === 'mqtt') return '订阅 Tag 自动生成 mqtt.tag 数据点'
  return '配置、预览与 artifact 契约驱动'
})

const capabilityText = computed(() => {
  if (isSqlConnection.value) return '表结构、SQL 执行、保存查询、数据点'
  if (props.connection?.type === 'mqtt') return '订阅、消息预览、Tag、数据点'
  if (['kafka', 'http', 'websocket', 'redis'].includes(props.connection?.type || '')) {
    return '短时真实抓样、配置契约'
  }
  return '配置契约'
})

const openButtonLabel = computed(() => {
  const connection = props.connection
  if (!connection) return '打开工作台'
  if (isSqlConnection.value) return '打开查询工作台'
  if (connection.type === 'mqtt') return '打开 MQTT 工作台'
  return '打开工作台'
})

const openHint = computed(() => {
  if (isSqlConnection.value) return '查看表、执行 SQL、保存查询并自动形成数据点。'
  if (props.connection?.type === 'mqtt') return '维护订阅、查看消息、生成和监控 Tag。'
  return '进入协议配置、预览和契约确认流程。'
})

const mappingHint = computed(() => {
  if (props.connection?.type === 'mqtt') {
    return 'MQTT 订阅中的 Tag 会作为 mqtt.tag 数据点进入统一数据点列表。'
  }
  return '该协议当前主要提供连接配置、短时预览与 artifact 契约。'
})

const previewTitle = computed(() =>
  ['kafka', 'http', 'websocket', 'redis'].includes(props.connection?.type || '')
    ? '支持统一短时真实预览'
    : '通过工作台查看开发态样本',
)

const previewHint = computed(() => {
  if (isSqlConnection.value) return 'SQL 工作台可直接执行只读查询并查看结果集。'
  if (props.connection?.type === 'mqtt')
    return 'MQTT 工作台会使用 preview session 建立临时消息查看通道。'
  return '抓样只在开发态短时执行，不持久运行节点侧采集任务。'
})

const datapointEmptyText = computed(() => {
  if (isSqlConnection.value) return '保存 SQL 查询后，后端会自动同步 db.query 数据点。'
  if (props.connection?.type === 'mqtt') return '创建 MQTT Tag 后会在数据点列表中出现。'
  return '当前接入源暂未生成数据点。'
})

const queryDataPointBySourceId = computed(() =>
  detailDataPoints.value.reduce(
    (records, point) => {
      if (point.sourceId) records[point.sourceId] = point
      return records
    },
    {} as Record<string, any>,
  ),
)

const queryMappings = computed(() =>
  detailQueries.value.map((query) => ({
    query,
    point: queryDataPointBySourceId.value[query.id] || null,
  })),
)

const configRows = computed(() => {
  const connection = props.connection
  if (!connection) return []

  if (connection.type === 'relational') {
    const config = connection.relationalConfig || {}
    return [
      { label: '数据库类型', value: typeLabel.value },
      { label: 'IP / Host', value: config.host || '-' },
      { label: '端口', value: config.port || '-' },
      { label: '库', value: config.database || '-' },
    ]
  }

  if (connection.type === 'mqtt') {
    const config = connection.mqttConfig || {}
    return [
      { label: '协议', value: config.protocol || 'mqtt' },
      { label: 'Broker', value: config.brokerUrl || config.host || '-' },
      { label: '端口', value: config.port || 1883 },
      { label: 'Topic', value: config.topic || config.defaultTopic || '-' },
    ]
  }

  const config = connection.config || {}
  if (connection.type === 'kafka') {
    return [
      { label: 'Broker', value: config.brokers || '-' },
      { label: 'Topic', value: config.topic || '-' },
      { label: 'Consumer Group', value: config.consumerGroup || '-' },
      { label: 'Start Position', value: config.startPosition || 'latest' },
    ]
  }

  if (connection.type === 'http') {
    return [
      { label: '配置方式', value: '请求在工作台维护' },
      { label: 'URL', value: '每个请求独立配置完整地址' },
    ]
  }

  if (connection.type === 'websocket') {
    return [
      { label: 'URL', value: config.url || '-' },
      { label: 'Topic', value: config.topic || '-' },
      {
        label: 'Heartbeat',
        value: config.heartbeatIntervalMs ? `${config.heartbeatIntervalMs}ms` : '-',
      },
    ]
  }

  if (connection.type === 'redis') {
    return [
      { label: 'Mode', value: config.mode || 'standalone' },
      { label: 'Address', value: config.address || '-' },
      { label: 'DB', value: config.db ?? 0 },
      { label: 'Key Pattern', value: config.keyPattern || '*' },
    ]
  }

  if (connection.type === 'opcua') {
    return [
      { label: 'Endpoint', value: config.endpoint || '-' },
      { label: 'Security Policy', value: config.securityPolicy || 'None' },
      { label: 'Security Mode', value: config.securityMode || 'none' },
      { label: 'Auth Type', value: config.authType || 'anonymous' },
      {
        label: 'Sampling',
        value: config.samplingMs ? `${config.samplingMs}ms` : '-',
      },
    ]
  }

  if (connection.type === 's7') {
    return [
      { label: 'IP', value: config.host || '-' },
      { label: '端口', value: config.port || 102 },
      { label: 'Rack', value: config.rack ?? 0 },
      { label: 'Slot', value: config.slot ?? 1 },
      {
        label: '轮询周期',
        value: config.pollIntervalMs ? `${config.pollIntervalMs}ms` : '-',
      },
    ]
  }

  if (connection.type === 'modbus') {
    return [
      { label: 'Mode', value: config.mode || 'tcp' },
      {
        label: 'IP',
        value: config.mode === 'rtu' ? 'RTU 串口' : config.host || '-',
      },
      {
        label: '端口',
        value: config.mode === 'rtu' ? '-' : config.port || 502,
      },
      { label: 'Slave ID', value: config.slaveId ?? 1 },
      { label: 'Address', value: config.startAddress ?? 0 },
      { label: 'Quantity', value: config.quantity ?? 1 },
    ]
  }

  if (connection.type === 'tdengine') {
    return [
      { label: 'DSN', value: config.dsn || '-' },
      { label: 'Database', value: config.database || '-' },
      { label: 'Timezone', value: config.timezone || '-' },
    ]
  }

  return [{ label: '类型', value: connection.type || '-' }]
})

const loadDetailResources = async () => {
  detailQueries.value = []
  detailDataPoints.value = []
  if (!props.projectId || !props.connection?.id) return

  detailLoading.value = true
  try {
    if (isSqlConnection.value) {
      const queryResponse = await dataAPI.getQueries(props.projectId, {
        connectionId: props.connection.id,
        queryType: 'sql',
        page: 1,
        pageSize: 100,
      })
      detailQueries.value = queryResponse.data?.queries || queryResponse.data || []

      const sourceIds = detailQueries.value.map((query) => query.id).filter(Boolean)
      if (sourceIds.length > 0) {
        const pointResponse = await dataAPI.getDataPoints(props.projectId, {
          type: 'db.query',
          sourceIds: sourceIds.join(','),
          page: 1,
          pageSize: 200,
        })
        detailDataPoints.value = pointResponse.data?.datapoints || []
      }
      return
    }

    const pointResponse = await dataAPI.getDataPoints(props.projectId, {
      sourceId: props.connection.id,
      page: 1,
      pageSize: 100,
    })
    detailDataPoints.value = pointResponse.data?.datapoints || []
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载接入源详情失败'))
  } finally {
    detailLoading.value = false
  }
}

const resolvePointStatus = (point: any) => (point.status === 'invalid' ? '失效' : '活跃')

const copyPointPath = async (path: string) => {
  if (!path) return
  try {
    await navigator.clipboard.writeText(path)
    ElMessage.success('数据点路径已复制')
  } catch {
    ElMessage.warning('复制失败，请手动复制路径')
  }
}

watch(
  () => [props.connection?.id, props.projectId],
  () => {
    activeTab.value = 'overview'
    void loadDetailResources()
  },
  { immediate: true },
)
</script>

<style scoped>
.access-source-detail {
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.access-source-detail__hero {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 14px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.access-source-detail__hero-main {
  min-width: 0;
}

.access-source-detail__hero h3 {
  margin: 8px 0 4px;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 16px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-detail__hero p {
  margin: 0;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.access-source-detail__status {
  display: inline-flex;
  padding: 3px 7px;
  border-radius: var(--dc-radius-xs);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-size: 11px;
  font-weight: 700;
}

.access-source-detail__status.is-connected {
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.access-source-detail__status.is-error {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.access-source-detail__metrics {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  padding: 12px;
  border-bottom: 1px solid var(--dc-border);
}

.access-source-detail__metrics div {
  min-width: 0;
  padding: 9px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}

.access-source-detail__metrics span,
.access-source-detail__resource span,
.access-source-detail__empty-line,
.access-source-detail__preview-box span {
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.access-source-detail__metrics strong {
  display: block;
  margin-top: 4px;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-detail__tabs {
  display: flex;
  gap: 4px;
  padding: 10px 12px 0;
  overflow-x: auto;
}

.access-source-detail__tabs button {
  flex: 0 0 auto;
  padding: 6px 8px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.access-source-detail__tabs button.is-active {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.access-source-detail__body {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.access-source-detail__section + .access-source-detail__section {
  margin-top: 14px;
}

.access-source-detail__section-title {
  margin-bottom: 8px;
  color: var(--dc-text);
  font-size: 12px;
  font-weight: 700;
}

.access-source-detail__list {
  margin: 0;
}

.access-source-detail__list div {
  display: flex;
  justify-content: space-between;
  gap: 14px;
  padding: 10px 0;
  border-bottom: 1px solid var(--dc-border);
}

.access-source-detail__list dt {
  flex: 0 0 auto;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.access-source-detail__list dd {
  min-width: 0;
  margin: 0;
  color: var(--dc-text);
  font-size: 12px;
  font-weight: 700;
  text-align: right;
  word-break: break-all;
}

.access-source-detail__primary-card,
.access-source-detail__preview-box,
.access-source-detail__resource {
  width: 100%;
  display: grid;
  gap: 5px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
  text-align: left;
}

.access-source-detail__primary-card strong,
.access-source-detail__preview-box strong,
.access-source-detail__resource strong {
  overflow: hidden;
  color: var(--dc-text);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-detail__primary-card:hover,
.access-source-detail__resource:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  background: var(--dc-primary-soft);
}

.access-source-detail__resource {
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  margin-bottom: 8px;
}

.access-source-detail__resource div {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.access-source-detail__resource span {
  overflow: hidden;
  font-family: var(--dc-font-mono);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-detail__resource small {
  color: var(--dc-success);
  font-size: 11px;
  font-weight: 700;
}

.access-source-detail__resource small.is-muted {
  color: var(--dc-text-muted);
}

.access-source-detail__preview-box {
  text-align: left;
}

.access-source-detail__preview-box .el-button {
  justify-self: flex-start;
  margin-top: 4px;
}

.access-source-detail__empty-line {
  padding: 12px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  line-height: 1.6;
}

.access-source-detail__empty {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--dc-text-secondary);
  text-align: center;
}

.access-source-detail__empty strong {
  color: var(--dc-text);
}

.access-source-detail__empty span {
  max-width: 230px;
  font-size: 12px;
  line-height: 1.7;
}

.access-source-detail__empty-mark {
  width: 58px;
  height: 58px;
  border: 1px solid rgba(29, 78, 216, 0.2);
  border-radius: var(--dc-radius-md);
  background: var(--dc-primary-soft);
}
</style>
