<template>
  <article class="access-source-card" :class="{ 'is-active': active }">
    <div class="access-source-card__top">
      <span
        class="access-source-card__icon"
        :class="`is-${resolveConnectionVisual(connection).category}`"
      >
        <component :is="resolveConnectionVisual(connection).icon" />
      </span>
      <div class="access-source-card__badges">
        <!-- Phase 2 仅配置角标 -->
        <StatusBadge v-if="isPhase2" tone="warning" text="仅配置" />
      </div>
    </div>

    <div class="access-source-card__body">
      <div class="access-source-card__name" :title="connection.name">
        {{ connection.name || '未命名连接' }}
      </div>
      <div class="access-source-card__type">
        <component
          :is="resolveConnectionVisual(connection).miniIcon"
          class="access-source-card__type-icon"
        />
        <span>{{ resolveConnectionType(connection) }}</span>
        <span class="access-source-card__divider">·</span>
        <span class="access-source-card__endpoint">
          {{ resolveConnectionEndpoint(connection) }}
        </span>
      </div>
    </div>

    <div class="access-source-card__divider-line"></div>

    <!-- 底部按钮区：左：打开工作台（outline），右：编辑+删除组合按钮 -->
    <div class="access-source-card__bottom">
      <button type="button" class="access-source-card__open" @click="$emit('open', connection)">
        <span>打开工作台</span>
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
import IconTablerArrowRight from '~icons/tabler/arrow-right'
import IconTablerBuildingFactory2 from '~icons/tabler/building-factory-2'
import IconTablerDatabase from '~icons/tabler/database'
import IconTablerMessages from '~icons/tabler/messages'
import IconTablerSettings from '~icons/tabler/settings'
import IconTablerTrash from '~icons/tabler/trash'
import StatusBadge from '@/components/shared/StatusBadge.vue'

type AccessSourceConnection = {
  id: string
  name?: string
  type?: string
  datapointCount?: number
  dataPointCount?: number
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
  /** 工业协议等 Phase 2 类型，显示「仅配置」角标 */
  isPhase2?: boolean
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
    return { category, icon: IconTablerMessages, miniIcon: IconTablerMessages }
  }
  if (category === 'industrial') {
    return {
      category,
      icon: IconTablerBuildingFactory2,
      miniIcon: IconTablerBuildingFactory2,
    }
  }
  return { category, icon: IconTablerDatabase, miniIcon: IconTablerDatabase }
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
    return [config['method'] || 'GET', config['baseUrl']].filter(Boolean).join(' ') || '未配置 URL'
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
</script>

<style scoped>
.access-source-card {
  min-height: 132px;
  display: flex;
  flex-direction: column;
  padding: 14px;
  border: 1px solid var(--dc-connection-card-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-connection-card-bg);
  box-shadow: var(--dc-connection-card-shadow);
  color: var(--dc-text);
  text-align: left;
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
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

/* 右上角角标区：竖向排列或横向排列均可，间距 6px */
.access-source-card__badges {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  justify-content: flex-end;
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
  gap: 7px;
  margin-top: 10px;
  color: var(--dc-text-secondary);
  font-size: 12px;
  line-height: 1.45;
}

.access-source-card__type-icon {
  width: 16px;
  height: 16px;
  flex: 0 0 auto;
  color: var(--dc-text-secondary);
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

.access-source-card__divider-line {
  height: 1px;
  margin: auto 0 12px;
  background: var(--dc-border);
  opacity: 0.6;
}

.access-source-card__bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

/* 「打开工作台」改为 outline 风格 */
.access-source-card__open {
  min-width: 0;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 0 12px;
  border: 1px solid var(--dc-primary);
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-primary);
  font-size: 13px;
  font-weight: 700;
  transition:
    background-color 0.18s ease,
    color 0.18s ease;
}

.access-source-card__open span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-card__open:hover {
  background: var(--dc-primary);
  color: var(--dc-surface-raised);
}

.access-source-card__open-icon {
  width: 17px;
  height: 17px;
  flex: 0 0 auto;
}

/* 编辑+删除操作组：边框容器 */
.access-source-card__actions {
  display: inline-flex;
  align-items: center;
  height: 32px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  overflow: hidden;
  background: var(--dc-surface-raised);
  flex-shrink: 0;
}

/* 中间竖向分隔线 */
.access-source-card__actions-divider {
  width: 1px;
  height: 100%;
  background: var(--dc-border);
  flex-shrink: 0;
}

/* 编辑按钮 */
.access-source-card__edit {
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  background: transparent;
  color: var(--dc-text-secondary);
  transition:
    background-color 0.18s ease,
    color 0.18s ease;
}

.access-source-card__edit svg {
  width: 15px;
  height: 15px;
}

.access-source-card__edit:hover {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

/* 删除按钮：默认比编辑视觉略轻，hover 强调危险色 */
.access-source-card__delete {
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  background: transparent;
  color: var(--dc-text-muted);
  transition:
    background-color 0.18s ease,
    color 0.18s ease;
}

.access-source-card__delete svg {
  width: 15px;
  height: 15px;
}

.access-source-card__delete:hover {
  background: rgba(220, 38, 38, 0.08);
  color: var(--dc-danger, #dc2626);
}

@media (max-width: 760px) {
  .access-source-card {
    min-height: 120px;
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
