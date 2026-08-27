<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { alarms, computes, points } from '@induforge/runtime-sdk'

// 内置 Demo 工程会同步创建下列 IF 内置库数据点。
const POINT_PATHS = {
  temperature: 'db.IF关系库.demo_line_current.temperature',
  pressure: 'db.IF关系库.demo_line_current.pressure',
  // 查询只有一个完整结果输出时，数据点路径不再追加 output key。
  history: 'db.IF时序库.demo_temperature_history',
  setpoint: 'realtime.IF实时库.demo.line1.setpoint',
  lineEvents: 'mqtt.IF消息库.demo_line_events',
}
const COMPUTE_REF = 'temperatureConvert'

const temperaturePoint = points.byPath(POINT_PATHS.temperature)
const pressurePoint = points.byPath(POINT_PATHS.pressure)
const historyPoint = points.byPath(POINT_PATHS.history)
const setpointPoint = points.byPath(POINT_PATHS.setpoint)
const lineEventsPoint = points.byPath(POINT_PATHS.lineEvents)

const loading = ref(false)
const operationLoading = ref(false)
const message = ref('')
const temperature = ref(null)
const pressure = ref(null)
const historyRows = ref([])
const currentAlarms = ref([])
const targetValue = ref(80)
const computeResult = ref(null)
const latestLineEvent = ref(null)
let unsubscribeLineEvents = null
let unsubscribeAlarms = null

const qualityClass = computed(() => {
  const quality = String(temperature.value?.quality || '').toLowerCase()
  if (quality === 'good') return 'is-good'
  if (quality === 'bad') return 'is-bad'
  return 'is-unknown'
})

function resultItems(data) {
  if (Array.isArray(data)) return data
  if (Array.isArray(data?.items)) return data.items
  if (Array.isArray(data?.data)) return data.data
  if (Array.isArray(data?.rows)) return data.rows
  return []
}

function showFailure(result, fallback) {
  if (result?.code === 0) return false
  message.value = result?.msg || fallback
  return true
}

async function loadCurrentValues() {
  const [temperatureResult, pressureResult] = await Promise.all([
    temperaturePoint.read(),
    pressurePoint.read(),
  ])
  if (!showFailure(temperatureResult, '读取温度失败')) temperature.value = temperatureResult.data
  if (!showFailure(pressureResult, '读取压力失败')) pressure.value = pressureResult.data
}

async function loadHistory() {
  const result = await historyPoint.read()
  if (!showFailure(result, '读取历史失败')) {
    historyRows.value = resultItems(result.data?.value ?? result.data)
  }
}

async function loadCurrentAlarms() {
  const result = await alarms.current.list({ status: 'active', page: 1, pageSize: 20 })
  if (!showFailure(result, '读取当前报警失败')) currentAlarms.value = resultItems(result.data)
}

async function loadDashboard() {
  loading.value = true
  message.value = ''
  try {
    await Promise.all([loadCurrentValues(), loadHistory(), loadCurrentAlarms()])
  } finally {
    loading.value = false
  }
}

async function subscribeRuntimeChanges() {
  const pointResult = await lineEventsPoint.subscribe((sample) => {
    latestLineEvent.value = sample?.data ?? sample
  })
  if (pointResult.code === 0 && typeof pointResult.data === 'function') {
    unsubscribeLineEvents = pointResult.data
  }

  const alarmResult = await alarms.changes.subscribe(() => loadCurrentAlarms())
  if (alarmResult.code === 0 && typeof alarmResult.data === 'function') {
    unsubscribeAlarms = alarmResult.data
  }
}

async function publishLineEvent() {
  operationLoading.value = true
  message.value = ''
  try {
    const payload = {
      line: '一号线',
      event: 'operator_test',
      running: true,
      timestamp: new Date().toISOString(),
    }
    const result = await lineEventsPoint.publish(payload)
    message.value = result.code === 0 ? '测试消息已发布，等待订阅回传' : result.msg
  } finally {
    operationLoading.value = false
  }
}

async function writeSetpoint() {
  operationLoading.value = true
  message.value = ''
  try {
    const result = await setpointPoint.set(Number(targetValue.value))
    message.value = result.code === 0 ? '设定值已提交' : result.msg
  } finally {
    operationLoading.value = false
  }
}

async function runCompute() {
  operationLoading.value = true
  message.value = ''
  try {
    const result = await computes.byRef(COMPUTE_REF).run({
      temperature: temperature.value?.value,
      pressure: pressure.value?.value,
    })
    if (!showFailure(result, '运行计算失败')) computeResult.value = result.data
  } finally {
    operationLoading.value = false
  }
}

async function acknowledgeAlarm(alarm) {
  const id = alarm.id || alarm.alarmId || alarm.stateId
  if (!id) return
  const result = await alarms.actions.acknowledge(id, { comment: '页面确认' })
  message.value = result.code === 0 ? '报警已确认' : result.msg
  if (result.code === 0) await loadCurrentAlarms()
}

function formatValue(sample, digits = 1) {
  const value = Number(sample?.value)
  return Number.isFinite(value) ? value.toFixed(digits) : '--'
}

function formatTime(value) {
  if (!value) return '--'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString()
}

onMounted(async () => {
  await loadDashboard()
  await subscribeRuntimeChanges()
})

onBeforeUnmount(() => {
  if (typeof unsubscribeLineEvents === 'function') unsubscribeLineEvents()
  if (typeof unsubscribeAlarms === 'function') unsubscribeAlarms()
})
</script>

<template>
  <main class="dashboard-shell">
    <header class="page-header">
      <div>
        <p class="eyebrow">一号线 / 工艺监控</p>
        <h1>生产运行概览</h1>
      </div>
      <button class="secondary-button" :disabled="loading" @click="loadDashboard">
        {{ loading ? '刷新中…' : '刷新数据' }}
      </button>
    </header>

    <p v-if="message" class="notice">{{ message }}</p>

    <section class="metric-grid" aria-label="实时指标">
      <article class="metric-card">
        <div class="metric-card__head">
          <span>设备温度</span>
          <i class="quality-dot" :class="qualityClass"></i>
        </div>
        <strong>{{ formatValue(temperature) }}<small>℃</small></strong>
        <p>{{ formatTime(temperature?.observedAt || temperature?.timestamp) }}</p>
      </article>
      <article class="metric-card">
        <div class="metric-card__head">
          <span>管线压力</span><i class="quality-dot is-good"></i>
        </div>
        <strong>{{ formatValue(pressure, 2) }}<small>MPa</small></strong>
        <p>{{ formatTime(pressure?.observedAt || pressure?.timestamp) }}</p>
      </article>
      <article class="metric-card metric-card--alarm">
        <div class="metric-card__head"><span>当前报警</span></div>
        <strong>{{ currentAlarms.length }}<small>条</small></strong>
        <p>{{ currentAlarms.length ? '请及时处理活动报警' : '当前运行正常' }}</p>
      </article>
    </section>

    <section class="content-grid">
      <article class="panel">
        <header class="panel__head">
          <div>
            <h2>温度历史</h2>
            <p>最近一小时样本</p>
          </div>
        </header>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>时间</th>
                <th>数值</th>
                <th>质量</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, index) in historyRows" :key="row.observed_at || index">
                <td>{{ formatTime(row.observed_at || row.observedAt || row.timestamp) }}</td>
                <td>{{ formatValue({ value: row.temperature }) }} ℃</td>
                <td>
                  <span class="status-tag">{{ row.quality || 'unknown' }}</span>
                </td>
              </tr>
              <tr v-if="!historyRows.length">
                <td colspan="3" class="empty-cell">暂无历史数据</td>
              </tr>
            </tbody>
          </table>
        </div>
      </article>

      <aside class="panel action-panel">
        <header class="panel__head">
          <div>
            <h2>运行操作</h2>
            <p>操作由节点侧再次鉴权</p>
          </div>
        </header>
        <label class="field-label" for="setpoint">温度设定值</label>
        <div class="input-action">
          <input id="setpoint" v-model.number="targetValue" type="number" step="0.1" />
          <button :disabled="operationLoading" @click="writeSetpoint">提交</button>
        </div>
        <button class="compute-button" :disabled="operationLoading" @click="runCompute">
          运行温度换算计算
        </button>
        <button class="compute-button" :disabled="operationLoading" @click="publishLineEvent">
          发布一号线测试消息
        </button>
        <pre v-if="computeResult" class="result-box">{{
          JSON.stringify(computeResult, null, 2)
        }}</pre>
        <div class="message-preview">
          <span>最新订阅消息</span>
          <pre class="result-box">{{
            latestLineEvent ? JSON.stringify(latestLineEvent, null, 2) : '等待消息…'
          }}</pre>
        </div>
      </aside>
    </section>

    <section class="panel alarm-panel">
      <header class="panel__head">
        <div>
          <h2>活动报警</h2>
          <p>当前状态来自节点报警存储</p>
        </div>
      </header>
      <ul v-if="currentAlarms.length" class="alarm-list">
        <li v-for="alarm in currentAlarms" :key="alarm.id || alarm.alarmId">
          <div>
            <strong>{{ alarm.name || alarm.alarmName || '未命名报警' }}</strong>
            <span>{{ alarm.datapointPath || alarm.path || '--' }}</span>
          </div>
          <div class="alarm-list__actions">
            <span class="level-tag">{{ alarm.levelName || alarm.level || '报警' }}</span>
            <button @click="acknowledgeAlarm(alarm)">确认</button>
          </div>
        </li>
      </ul>
      <p v-else class="empty-state">暂无活动报警</p>
    </section>
  </main>
</template>
