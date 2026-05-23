<template>
  <section class="industrial-workbench">
    <header class="industrial-workbench__head">
      <button type="button" class="industrial-workbench__back" @click="$emit('back')">
        <IconTablerArrowLeft />
      </button>
      <div class="industrial-workbench__title">
        <span>{{ protocolLabel }}</span>
        <strong>{{ connection.name || '未命名工业接入源' }}</strong>
      </div>
      <WorkbenchStatusPill :label="statusLabel" :tone="statusTone" />
      <div class="industrial-workbench__spacer"></div>
      <el-button size="small" :loading="testing" @click="runConnectionTest"> 测试连接 </el-button>
    </header>

    <div class="industrial-workbench__body">
      <aside class="industrial-workbench__side">
        <section class="industrial-workbench__panel">
          <div class="industrial-workbench__panel-title">连接参数</div>
          <dl class="industrial-workbench__meta">
            <template v-for="row in configRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd :title="row.value">{{ row.value }}</dd>
            </template>
          </dl>
        </section>

        <section class="industrial-workbench__panel">
          <div class="industrial-workbench__panel-title">测试结果</div>
          <div class="industrial-workbench__test" :class="`is-${testState}`">
            <IconTablerActivityHeartbeat />
            <strong>{{ testTitle }}</strong>
            <span>{{ testDetail }}</span>
          </div>
        </section>
      </aside>

      <main class="industrial-workbench__main">
        <section class="industrial-workbench__matrix">
          <div class="industrial-workbench__section-head">
            <div>
              <strong>{{ protocolLabel }} 工作台</strong>
              <span>{{ endpointText }}</span>
            </div>
          </div>

          <template v-if="connection.type === 'opcua'">
            <div class="industrial-workbench__opc-grid">
              <article
                v-for="node in opcNodes"
                :key="node.nodeId"
                class="industrial-workbench__node"
              >
                <span>{{ node.name }}</span>
                <strong>{{ node.nodeId }}</strong>
                <em>{{ node.dataType }}</em>
              </article>
            </div>
          </template>

          <template v-else>
            <div class="industrial-workbench__registers">
              <div class="industrial-workbench__register-head">
                <span>地址</span>
                <span>功能</span>
                <span>数量</span>
                <span>说明</span>
              </div>
              <div
                v-for="row in modbusRows"
                :key="row.address"
                class="industrial-workbench__register-row"
              >
                <span>{{ row.address }}</span>
                <span>{{ row.functionCode }}</span>
                <span>{{ row.quantity }}</span>
                <span>{{ row.note }}</span>
              </div>
            </div>
          </template>
        </section>

        <section class="industrial-workbench__log">
          <div class="industrial-workbench__section-head">
            <div>
              <strong>诊断日志</strong>
              <span>开发态短时验证</span>
            </div>
            <el-button size="small" @click="logs = []">清空</el-button>
          </div>
          <pre>{{ logText }}</pre>
        </section>
      </main>
    </div>
  </section>
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'
import WorkbenchStatusPill from '@/components/workbench/WorkbenchStatusPill.vue'
import IconTablerActivityHeartbeat from '~icons/tabler/activity-heartbeat'
import IconTablerArrowLeft from '~icons/tabler/arrow-left'

type AccessSourceConnection = {
  id: string
  name?: string
  type?: string
  status?: string
  config?: Record<string, any>
}

const props = defineProps<{
  connection: AccessSourceConnection
  projectId: string
}>()

defineEmits<{
  (event: 'back'): void
}>()

const testing = ref(false)
const testState = ref<'idle' | 'success' | 'error'>('idle')
const testTitle = ref('尚未测试')
const testDetail = ref('点击测试连接执行开发态短时验证')
const logs = ref<string[]>([])

const config = computed(() => props.connection.config || {})
const protocolLabel = computed(() => (props.connection.type === 'opcua' ? 'OPC UA' : 'Modbus'))

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

const endpointText = computed(() => {
  if (props.connection.type === 'opcua') {
    return String(config.value.endpoint || '未配置 endpoint')
  }
  if (String(config.value.mode || 'tcp') === 'rtu') {
    return `RTU ${formatObject(config.value.serialConfig)}`
  }
  return [config.value.host, config.value.port].filter(Boolean).join(':') || '未配置 host'
})

const configRows = computed(() => {
  const c = config.value
  if (props.connection.type === 'opcua') {
    return [
      { label: 'Endpoint', value: String(c.endpoint || '未配置') },
      { label: '安全策略', value: String(c.securityPolicy || 'None') },
      { label: '安全模式', value: String(c.securityMode || 'None') },
      { label: '认证', value: String(c.authType || 'anonymous') },
      { label: '采样', value: `${c.samplingMs || 1000}ms` },
    ]
  }
  return [
    { label: '模式', value: String(c.mode || 'tcp').toUpperCase() },
    { label: '地址', value: endpointText.value },
    { label: '站号', value: String(c.slaveId ?? 1) },
    { label: '起始', value: String(c.startAddress ?? 0) },
    { label: '数量', value: String(c.quantity ?? 1) },
    { label: '周期', value: `${c.pollIntervalMs || 1000}ms` },
  ]
})

const opcNodes = computed(() => {
  const ns = config.value.options?.namespace || 'ns=2'
  return [
    { name: 'Root', nodeId: 'i=84', dataType: 'Object' },
    { name: 'Objects', nodeId: 'i=85', dataType: 'Folder' },
    { name: 'SampleValue', nodeId: `${ns};s=SampleValue`, dataType: 'Variant' },
    { name: 'DeviceStatus', nodeId: `${ns};s=DeviceStatus`, dataType: 'Boolean' },
  ]
})

const modbusRows = computed(() => {
  const start = Number(config.value.startAddress ?? 0)
  const quantity = Number(config.value.quantity ?? 1)
  const functionCode = config.value.options?.functionCode || 3
  return Array.from({ length: Math.min(quantity, 16) }, (_, index) => ({
    address: start + index,
    functionCode: `FC${functionCode}`,
    quantity: 1,
    note: index === 0 ? '配置起始寄存器' : '连续寄存器',
  }))
})

const logText = computed(() =>
  logs.value.length ? logs.value.join('\n') : '等待测试连接或节点侧采集事件',
)

const formatLogTime = () => dayjs().format('YYYY-MM-DD HH:mm:ss')

const runConnectionTest = async () => {
  testing.value = true
  const startedAt = performance.now()
  try {
    const response = await dataAPI.testConnection(props.projectId, {
      type: props.connection.type,
      config: config.value,
    })
    const result = response?.data || response || {}
    const duration = Math.round(performance.now() - startedAt)
    testState.value = 'success'
    testTitle.value = result.message || '测试通过'
    testDetail.value = result.detail || `耗时 ${duration}ms`
    logs.value.unshift(`[OK] ${formatLogTime()} ${testTitle.value} ${testDetail.value}`)
    ElMessage.success(testTitle.value)
  } catch (error) {
    const message = getApiErrorMessage(error, '连接测试失败')
    testState.value = 'error'
    testTitle.value = '测试失败'
    testDetail.value = message
    logs.value.unshift(`[ERR] ${formatLogTime()} ${message}`)
    ElMessage.error(message)
  } finally {
    testing.value = false
  }
}

const formatObject = (value: unknown) => {
  if (!value || typeof value !== 'object') return '未配置'
  return Object.entries(value as Record<string, unknown>)
    .map(([key, val]) => `${key}=${val}`)
    .join(', ')
}
</script>

<style scoped>
.industrial-workbench {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
}

.industrial-workbench__head {
  min-height: 56px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.industrial-workbench__back {
  width: 30px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.industrial-workbench__back svg,
.industrial-workbench__test svg {
  width: 16px;
  height: 16px;
}

.industrial-workbench__title {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.industrial-workbench__title span {
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.industrial-workbench__title strong {
  overflow: hidden;
  color: var(--dc-text);
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.industrial-workbench__spacer {
  flex: 1;
}

.industrial-workbench__body {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
}

.industrial-workbench__side {
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px;
  overflow: auto;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
}

.industrial-workbench__panel {
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
}

.industrial-workbench__panel-title {
  margin-bottom: 10px;
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.industrial-workbench__meta {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 8px 10px;
  margin: 0;
}

.industrial-workbench__meta dt {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.industrial-workbench__meta dd {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  color: var(--dc-text-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.industrial-workbench__test {
  display: grid;
  grid-template-columns: 20px minmax(0, 1fr);
  gap: 6px 8px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.industrial-workbench__test strong {
  color: var(--dc-text);
}

.industrial-workbench__test span {
  grid-column: 2;
  line-height: 1.45;
}

.industrial-workbench__test.is-success svg {
  color: var(--dc-success);
}

.industrial-workbench__test.is-error svg {
  color: var(--dc-danger);
}

.industrial-workbench__main {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: minmax(0, 1fr) 220px;
}

.industrial-workbench__matrix,
.industrial-workbench__log {
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.industrial-workbench__matrix {
  border-bottom: 1px solid var(--dc-border);
}

.industrial-workbench__section-head {
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.industrial-workbench__section-head strong,
.industrial-workbench__section-head span {
  display: block;
}

.industrial-workbench__section-head strong {
  color: var(--dc-text);
  font-size: 13px;
}

.industrial-workbench__section-head span {
  margin-top: 2px;
  color: var(--dc-text-muted);
  font-size: 11px;
}

.industrial-workbench__opc-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 12px;
  overflow: auto;
  padding: 14px;
}

.industrial-workbench__node {
  min-height: 92px;
  display: grid;
  align-content: center;
  gap: 7px;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: #ffffff;
}

.industrial-workbench__node span {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.industrial-workbench__node strong {
  overflow: hidden;
  color: var(--dc-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.industrial-workbench__node em {
  color: var(--dc-primary);
  font-style: normal;
  font-size: 12px;
}

.industrial-workbench__registers {
  overflow: auto;
}

.industrial-workbench__register-head,
.industrial-workbench__register-row {
  display: grid;
  grid-template-columns: 96px 96px 96px minmax(0, 1fr);
  align-items: center;
  min-height: 34px;
  padding: 0 14px;
  border-bottom: 1px solid var(--dc-border);
  font-size: 12px;
}

.industrial-workbench__register-head {
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-weight: 700;
}

.industrial-workbench__register-row {
  color: var(--dc-text-secondary);
}

.industrial-workbench__log pre {
  flex: 1;
  min-height: 0;
  margin: 0;
  padding: 12px;
  overflow: auto;
  background: #0f172a;
  color: #dbeafe;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.55;
  white-space: pre-wrap;
}
</style>
