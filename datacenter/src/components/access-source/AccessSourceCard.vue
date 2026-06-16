<template>
  <article class="access-source-card" :class="{ 'is-active': active }">
    <div class="access-source-card__top">
      <span
        class="access-source-card__icon"
        :class="`is-${resolveConnectionVisual(connection).category}`"
      >
        <component :is="resolveConnectionVisual(connection).icon" />
      </span>
      <slot name="top-actions"></slot>
    </div>

    <div class="access-source-card__body">
      <div class="access-source-card__name" :title="connection.name">
        {{ connection.name || '未命名连接' }}
      </div>
      <div class="access-source-card__type">
        <span>{{ resolveConnectionType(connection) }}</span>
        <span class="access-source-card__divider">·</span>
        <span class="access-source-card__endpoint">
          {{ resolveConnectionEndpoint(connection) }}
        </span>
      </div>
      <div v-if="isIndustrialConnection" class="access-source-card__health">
        <span>{{ variableCountSummary }}</span>
        <span>{{ redundancySummary }}</span>
      </div>
    </div>

    <div class="access-source-card__bottom">
      <button type="button" class="access-source-card__open" @click="$emit('open', connection)">
        <span>工作台</span>
        <IconTablerArrowRight class="access-source-card__open-icon" />
      </button>
      <div class="access-source-card__actions">
        <button
          type="button"
          class="access-source-card__edit"
          :aria-label="`编辑连接 ${connection.name || ''}`"
          :title="`编辑连接 ${connection.name || ''}`"
          @click="$emit('edit', connection)"
        >
          <IconTablerSettings />
        </button>
        <span class="access-source-card__actions-divider" aria-hidden="true"></span>
        <button
          type="button"
          class="access-source-card__delete"
          :aria-label="`删除连接 ${connection.name || ''}`"
          :title="`删除连接 ${connection.name || ''}`"
          @click="$emit('delete-connection', connection)"
        >
          <IconTablerTrash />
        </button>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import IconTablerArrowRight from '~icons/tabler/arrow-right'
import IconTablerBuildingFactory2 from '~icons/tabler/building-factory-2'
import IconTablerDatabase from '~icons/tabler/database'
import IconTablerMessages from '~icons/tabler/messages'
import IconTablerSettings from '~icons/tabler/settings'
import IconTablerTrash from '~icons/tabler/trash'

type AccessSourceConnection = {
  id: string
  name?: string
  type?: string
  datapointCount?: number
  dataPointCount?: number
  variableCount?: number
  issueCount?: number
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
  config?: Record<string, unknown>
}

const props = defineProps<{
  connection: AccessSourceConnection
  active: boolean
}>()

defineEmits<{
  /** 单击「打开工作台」按钮 */
  (event: 'open', connection: AccessSourceConnection): void
  /** 单击编辑按钮 */
  (event: 'edit', connection: AccessSourceConnection): void
  /** 单击删除按钮 */
  (event: 'delete-connection', connection: AccessSourceConnection): void
}>()

const databaseTypes = new Set([
  'relational',
  'mysql',
  'postgresql',
  'sqlserver',
  'tdengine',
  'redis',
  'builtin.relation',
  'builtin.timeseries',
])
const streamTypes = new Set(['mqtt', 'kafka', 'websocket', 'http', 'builtin.message'])
const industrialTypes = new Set(['opcua', 'opcda', 's7', 'modbus'])
const builtinTypes = new Set([
  'builtin.relation',
  'builtin.timeseries',
  'builtin.realtime',
  'builtin.message',
])

const resolveConnectionCategory = (connection: AccessSourceConnection) => {
  const type = connection.type || ''
  if (type === 'builtin.realtime') return 'stream'
  if (databaseTypes.has(type)) return 'database'
  if (streamTypes.has(type)) return 'stream'
  if (industrialTypes.has(type)) return 'industrial'
  if (type.includes('opc') || type.includes('modbus')) return 'industrial'
  return 'database'
}

const resolveConnectionVisual = (connection: AccessSourceConnection) => {
  const category = resolveConnectionCategory(connection)
  if (category === 'stream') {
    return { category, icon: IconTablerMessages }
  }
  if (category === 'industrial') {
    return {
      category,
      icon: IconTablerBuildingFactory2,
    }
  }
  return { category, icon: IconTablerDatabase }
}

const resolveConnectionType = (connection: AccessSourceConnection) => {
  const builtinLabels: Record<string, string> = {
    'builtin.relation': 'IF关系库',
    'builtin.timeseries': 'IF时序库',
    'builtin.realtime': 'IF实时库',
    'builtin.message': 'IF消息库',
  }
  if (connection.type && builtinLabels[connection.type]) {
    return builtinLabels[connection.type]
  }
  if (connection.type === 'relational') {
    const dbType = connection.relationalConfig?.dbType || ''
    const dbTypeLabels: Record<string, string> = {
      mysql: 'MySQL',
      postgresql: 'PostgreSQL',
      sqlserver: 'SQL Server',
    }
    return dbTypeLabels[dbType] || dbType || '数据库/时序库'
  }
  if (connection.type === 'mqtt') return 'MQTT Broker'
  const protocolLabels: Record<string, string> = {
    kafka: 'Kafka Topic',
    http: 'HTTP Source',
    websocket: 'WebSocket',
    redis: 'Redis',
    opcua: 'OPC UA',
    opcda: 'OPC DA',
    s7: 'Siemens S7',
    modbus: 'Modbus',
    tdengine: 'TDengine',
  }
  if (connection.type && protocolLabels[connection.type]) {
    return protocolLabels[connection.type]
  }
  return connection.type || '未知类型'
}

const resolveConnectionEndpoint = (connection: AccessSourceConnection) => {
  if (connection.type && builtinTypes.has(connection.type)) {
    const runtimeKey = String(connection.config?.['runtimeKey'] || '').trim()
    if (runtimeKey) return runtimeKey
    return '工程内置运行库'
  }
  if (connection.type === 'relational') {
    const config = connection.relationalConfig
    if (!config) return '未配置数据库地址'
    const host = [config.host, config.port].filter(Boolean).join(':')
    return [host, config.database].filter(Boolean).join(' / ') || '未配置数据库'
  }
  if (connection.type === 'mqtt') {
    const config = connection.mqttConfig
    if (!config) return '未配置 Broker'
    const host = config.brokerUrl || config.host
    const endpoint = [host, config.port].filter(Boolean).join(':')
    const topic = config.topic || config.defaultTopic
    return [endpoint, topic].filter(Boolean).join(' / ') || '未配置 Topic'
  }
  const config = connection.config || {}
  if (connection.type === 'kafka') {
    return [config['brokers'], config['topic']].filter(Boolean).join(' / ') || '未配置 Topic'
  }
  if (connection.type === 'http') {
    return '请求在工作台配置'
  }
  if (connection.type === 'websocket') {
    return [config['url'], config['topic']].filter(Boolean).join(' / ') || '未配置 WebSocket'
  }
  if (connection.type === 'redis') {
    return (
      [config['address'], config['keyPattern'] || '*'].filter(Boolean).join(' / ') || '未配置 Redis'
    )
  }
  if (connection.type === 'opcua') {
    return (
      [config['endpoint'], config['securityMode'] || 'none'].filter(Boolean).join(' / ') ||
      '未配置 OPC UA'
    )
  }
  if (connection.type === 'opcda') {
    return String(config['serverProgId'] || config['host'] || '未配置 OPC DA')
  }
  if (connection.type === 's7') {
    return [config['host'], config['port']].filter(Boolean).join(':') || '未配置 S7'
  }
  if (connection.type === 'modbus') {
    if (config['mode'] === 'rtu') return 'RTU / 串口配置'
    return [config['host'], config['port']].filter(Boolean).join(':') || '未配置 Modbus'
  }
  if (connection.type === 'tdengine') {
    return [config['dsn'], config['database']].filter(Boolean).join(' / ') || '未配置 TDengine'
  }
  return '等待接入配置'
}

const isIndustrialConnection = computed(
  () => resolveConnectionCategory(props.connection) === 'industrial',
)

const variableCountSummary = computed(() => {
  const variableCount = Number(props.connection.variableCount ?? 0)
  const issueCount = Number(props.connection.issueCount ?? 0)
  const issueText = issueCount > 0 ? ` / 问题${issueCount}` : ''
  return `变量：${variableCount}${issueText}`
})

const redundancySummary = computed(() => {
  const redundancy = props.connection.config?.['redundancy'] as Record<string, any> | undefined
  if (!redundancy?.enabled) return '设备冗余：无'
  const endpoints = Array.isArray(redundancy.endpoints) ? redundancy.endpoints : []
  const enabledCount = endpoints.filter((endpoint) => endpoint?.enabled !== false).length
  return enabledCount > 1 ? '设备冗余：主备' : '设备冗余：待补'
})
</script>

<style scoped>
.access-source-card {
  box-sizing: border-box;
  min-width: 0;
  min-height: 148px;
  display: flex;
  flex-direction: column;
  padding: 14px;
  border: 1px solid var(--dc-connection-card-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-connection-card-bg);
  box-shadow: var(--dc-connection-card-shadow);
  color: var(--dc-text);
  text-align: left;
  overflow: hidden;
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease;
}

.access-source-card:hover,
.access-source-card.is-active {
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
  box-shadow: var(--dc-connection-card-hover-shadow);
}

.access-source-card.is-active {
  outline: 3px solid rgba(29, 78, 216, 0.1);
}

.access-source-card__top {
  flex: 0 0 auto;
  min-width: 0;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.access-source-card__icon {
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--dc-radius-sm);
  /* 去掉三色背景，统一中性色 */
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
}

.access-source-card__icon svg {
  width: 18px;
  height: 18px;
}

.access-source-card__body {
  min-width: 0;
  flex: 1 1 auto;
  margin-top: 12px;
}

.access-source-card__name {
  overflow: hidden;
  color: var(--dc-text);
  font-size: 15px;
  font-weight: 700;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-card__type {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 9px;
  color: var(--dc-text-secondary);
  font-size: 12px;
  line-height: 1.45;
}

.access-source-card__type
  span:not(.access-source-card__divider):not(.access-source-card__endpoint) {
  min-width: 0;
  flex: 0 1 auto;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-card__divider,
.access-source-card__endpoint {
  color: var(--dc-text-secondary);
}

.access-source-card__endpoint {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-card__health {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 10px;
}

.access-source-card__health span {
  max-width: 100%;
  min-height: 20px;
  display: inline-flex;
  align-items: center;
  padding: 0 6px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
  color: var(--dc-text-muted);
  font-size: 11px;
  white-space: nowrap;
}

.access-source-card__bottom {
  min-width: 0;
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: 14px;
}

.access-source-card__open {
  flex: 0 1 auto;
  min-width: 0;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  gap: 5px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--dc-primary);
  font-size: 13px;
  font-weight: 700;
  transition:
    color 0.18s ease,
    transform 0.18s ease;
}

.access-source-card__open span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-card__open:hover {
  color: color-mix(in oklch, var(--dc-primary) 82%, var(--dc-text));
  transform: translateX(1px);
}

.access-source-card__open-icon {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}

.access-source-card__actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex: 0 0 auto;
}

.access-source-card__actions-divider {
  width: 1px;
  height: 14px;
  background: var(--dc-border);
  flex-shrink: 0;
}

.access-source-card__edit,
.access-source-card__delete {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  transition:
    background-color 0.18s ease,
    color 0.18s ease;
}

.access-source-card__edit svg,
.access-source-card__delete svg {
  width: 15px;
  height: 15px;
}

.access-source-card__edit:hover {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.access-source-card__delete {
  color: var(--dc-text-muted);
}

.access-source-card__delete:hover {
  background: rgba(220, 38, 38, 0.08);
  color: var(--dc-danger, #dc2626);
}

@media (max-width: 760px) {
  .access-source-card {
    min-height: 144px;
    padding: 12px;
  }

  .access-source-card__name {
    font-size: 14px;
  }

  .access-source-card__type {
    font-size: 12px;
  }
}
</style>
