<template>
  <article class="access-source-card" :class="{ 'is-active': active }">
    <div class="access-source-card__top">
      <div class="access-source-card__heading">
        <span
          class="access-source-card__icon"
          :class="`is-${resolveConnectionVisual(connection).category}`"
        >
          <component :is="resolveConnectionVisual(connection).icon" />
        </span>
        <div class="access-source-card__name" :title="connection.name">
          {{ connection.name || t('accessSources.unnamed') }}
        </div>
      </div>
      <slot name="top-actions"></slot>
    </div>

    <div class="access-source-card__body">
      <div class="access-source-card__type">
        <span
          class="access-source-card__type-badge"
          :class="`is-${resolveConnectionVisual(connection).category}`"
        >
          {{ resolveConnectionType(connection) }}
        </span>
        <span class="access-source-card__divider">·</span>
        <span class="access-source-card__endpoint">
          {{ resolveConnectionEndpoint(connection) }}
        </span>
      </div>
      <div class="access-source-card__state-row">
        <span
          class="access-source-card__state"
          :class="`is-${connectionState.tone}`"
          :title="connectionState.detail"
        >
          {{ connectionState.label }}
        </span>
        <span>{{ pointCountLabel(connection.variableCount) }}</span>
      </div>
    </div>

    <div class="access-source-card__bottom">
      <button type="button" class="access-source-card__open" @click="$emit('open', connection)">
        <span>{{ t('accessSources.workbench') }}</span>
        <IconTablerArrowRight class="access-source-card__open-icon" />
      </button>
      <div class="access-source-card__actions">
        <button
          v-if="connection.testCapability.status === 'supported'"
          type="button"
          class="access-source-card__edit"
          :disabled="testing"
          :aria-label="`${t('accessSources.testSource')} ${connection.name || ''}`"
          :title="`${t('accessSources.testSaved')} ${connection.name || ''}`"
          @click="$emit('test', connection)"
        >
          <IconTablerLoader2 v-if="testing" class="is-spinning" />
          <IconTablerPlugConnected v-else />
        </button>
        <button
          type="button"
          class="access-source-card__edit"
          :aria-label="`${t('accessSources.editSource')} ${connection.name || ''}`"
          :title="`${t('accessSources.editSource')} ${connection.name || ''}`"
          @click="$emit('edit', connection)"
        >
          <IconTablerSettings />
        </button>
        <span class="access-source-card__actions-divider" aria-hidden="true"></span>
        <button
          type="button"
          class="access-source-card__delete"
          :aria-label="`${t('accessSources.deleteSource')} ${connection.name || ''}`"
          :title="`${t('accessSources.deleteSource')} ${connection.name || ''}`"
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
import IconTablerSettings from '~icons/tabler/settings'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerPlugConnected from '~icons/tabler/plug-connected'
import { resolveAccessSourceVisual } from './access-source-visual'
import type { Connection as AccessSourceConnection } from '@/api/schemas/connection.schema'
import { datacenterLocale, t } from '@/i18n/runtime'

const props = defineProps<{
  connection: AccessSourceConnection
  active: boolean
  testing?: boolean
}>()

const pointCountLabel = (count: number) =>
  datacenterLocale.value === 'en'
    ? `${count} data point${count === 1 ? '' : 's'}`
    : t('accessSources.pointCount', { count })

defineEmits<{
  /** 单击「打开工作台」按钮 */
  (event: 'open', connection: AccessSourceConnection): void
  /** 单击编辑按钮 */
  (event: 'edit', connection: AccessSourceConnection): void
  (event: 'test', connection: AccessSourceConnection): void
  /** 单击删除按钮 */
  (event: 'delete-connection', connection: AccessSourceConnection): void
}>()

const builtinTypes = new Set([
  'builtin.relation',
  'builtin.timeseries',
  'builtin.realtime',
  'builtin.message',
])

const resolveConnectionVisual = (connection: AccessSourceConnection) => {
  return resolveAccessSourceVisual(connection.type)
}

const resolveConnectionType = (connection: AccessSourceConnection) => {
  const builtinLabels: Record<string, string> = {
    'builtin.relation': t('accessSources.builtinTypes.relation'),
    'builtin.timeseries': t('accessSources.builtinTypes.timeseries'),
    'builtin.realtime': t('accessSources.builtinTypes.realtime'),
    'builtin.message': t('accessSources.builtinTypes.message'),
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
    return dbTypeLabels[dbType] || dbType || t('accessSources.databaseTimeseries')
  }
  if (connection.type === 'mqtt') return 'MQTT Broker'
  const protocolLabels: Record<string, string> = {
    kafka: 'Kafka Topic',
    http: 'HTTP Source',
    websocket: 'WebSocket',
    redis: 'Redis',
    tdengine: 'TDengine',
  }
  if (connection.type && protocolLabels[connection.type]) {
    return protocolLabels[connection.type]
  }
  return connection.type || t('accessSources.unknownType')
}

const resolveConnectionEndpoint = (connection: AccessSourceConnection) => {
  if (connection.type && builtinTypes.has(connection.type)) {
    const runtimeKey = String(connection.config?.['runtimeKey'] || '').trim()
    if (runtimeKey) return runtimeKey
    return t('accessSources.builtinStore')
  }
  if (connection.type === 'relational') {
    const config = connection.relationalConfig
    if (!config) return t('accessSources.notConfiguredDbAddress')
    const host = [config.host, config.port].filter(Boolean).join(':')
    return [host, config.database].filter(Boolean).join(' / ') || t('accessSources.notConfiguredDb')
  }
  if (connection.type === 'mqtt') {
    const config = connection.mqttConfig
    if (!config) return t('accessSources.notConfiguredBroker')
    const host = config.brokerUrl || config.host
    const endpoint = [host, config.port].filter(Boolean).join(':')
    const topic = config.topic || config.defaultTopic
    return [endpoint, topic].filter(Boolean).join(' / ') || t('accessSources.notConfiguredTopic')
  }
  const config = connection.config || {}
  if (connection.type === 'kafka') {
    return [config['brokers'], config['topic']].filter(Boolean).join(' / ') || t('accessSources.notConfiguredTopic')
  }
  if (connection.type === 'http') {
    return t('accessSources.requestInWorkbench')
  }
  if (connection.type === 'websocket') {
    return [config['url'], config['topic']].filter(Boolean).join(' / ') || t('accessSources.notConfiguredWebSocket')
  }
  if (connection.type === 'redis') {
    return (
      [config['address'], config['keyPattern'] || '*'].filter(Boolean).join(' / ') || t('accessSources.notConfiguredRedis')
    )
  }
  if (connection.type === 'tdengine') {
    return [config['host'], config['databaseName']].filter(Boolean).join(' / ') || t('accessSources.notConfiguredTdengine')
  }
  return t('accessSources.pendingConfig')
}

const connectionState = computed(() => {
  const connection = props.connection
  if (!connection.enabled) return { label: t('accessSources.disabled'), tone: 'muted', detail: t('accessSources.disabledDetail') }
  if (connection.configurationState === 'incomplete') {
    return { label: t('accessSources.incomplete'), tone: 'warning', detail: t('accessSources.incompleteDetail') }
  }
  if (connection.testCapability.status === 'unsupported') {
    return {
      label: t('accessSources.normal'),
      tone: 'success',
      detail: datacenterLocale.value === 'en' ? t('accessSources.workspaceTestHint') : connection.testCapability.reason || t('accessSources.workspaceTestHint'),
    }
  }
  if (connection.lastTest.status === 'succeeded') {
    return {
      label: t('accessSources.normal'),
      tone: 'success',
      detail: datacenterLocale.value === 'en' ? t('accessSources.testSucceeded') : connection.lastTest.message || t('accessSources.testSucceeded'),
    }
  }
  if (connection.lastTest.status === 'failed') {
    return {
      label: t('accessSources.abnormal'),
      tone: 'danger',
      detail:
        datacenterLocale.value === 'en' && /[\u4e00-\u9fff]/.test(connection.lastTest.message || '')
          ? t('accessSources.testFailed')
          : connection.lastTest.message || t('accessSources.testFailed'),
    }
  }
  return { label: t('accessSources.notTested'), tone: 'muted', detail: t('accessSources.notTestedDetail') }
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

.access-source-card__heading {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.access-source-card__icon {
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 14%, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.access-source-card__icon.is-database {
  border-color: color-mix(in srgb, var(--dc-success) 28%, var(--dc-border));
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.access-source-card__icon.is-stream {
  border-color: color-mix(in srgb, var(--dc-warning) 28%, var(--dc-border));
  background: var(--dc-warning-soft);
  color: var(--dc-warning);
}

.access-source-card__icon svg {
  width: 18px;
  height: 18px;
}

.access-source-card__body {
  min-width: 0;
  flex: 1 1 auto;
  margin-top: 14px;
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

.access-source-card__type-badge {
  min-height: 22px;
  display: inline-flex;
  align-items: center;
  padding: 0 8px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 18%, var(--dc-border));
  border-radius: 999px;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-size: 11px;
  font-weight: 650;
}

.access-source-card__type-badge.is-database {
  border-color: color-mix(in srgb, var(--dc-success) 28%, var(--dc-border));
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.access-source-card__type-badge.is-stream {
  border-color: color-mix(in srgb, var(--dc-warning) 28%, var(--dc-border));
  background: var(--dc-warning-soft);
  color: var(--dc-warning);
}

.access-source-card__type
  span:not(.access-source-card__divider):not(.access-source-card__endpoint) {
  min-width: 0;
  flex: 0 1 auto;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-card__state-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.access-source-card__state {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  white-space: nowrap;
}

.access-source-card__state.is-success {
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.access-source-card__state.is-warning {
  background: var(--dc-warning-soft);
  color: var(--dc-warning);
}

.access-source-card__state.is-danger {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
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
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  gap: 5px;
  padding: 0 9px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 20%, var(--dc-border));
  border-radius: 6px;
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
  font-size: 13px;
  font-weight: 700;
  transition:
    background-color 0.18s ease,
    border-color 0.18s ease,
    color 0.18s ease,
    transform 0.18s ease;
}

.access-source-card__open span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-card__open:hover {
  border-color: var(--dc-primary);
  background: var(--dc-primary);
  color: var(--dc-on-primary);
  transform: translateY(-1px);
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
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
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
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.access-source-card__delete {
  color: var(--dc-text-muted);
}

.access-source-card__delete:hover {
  border-color: #fecaca;
  background: rgba(220, 38, 38, 0.08);
  color: var(--dc-danger, #dc2626);
}

.is-spinning {
  animation: access-source-spin 0.8s linear infinite;
}

@keyframes access-source-spin {
  to {
    transform: rotate(360deg);
  }
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
