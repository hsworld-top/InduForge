<template>
  <div class="industrial-workbench">
    <header class="industrial-workbench__toolbar">
      <div class="industrial-workbench__title">
        <span class="industrial-workbench__title-icon"><IconTablerCpu /></span>
        <h1>{{ ui('工业采集', 'Industrial Collection') }}</h1>
      </div>
      <div class="industrial-workbench__toolbar-actions">
        <el-button @click="connectionList?.reload()">{{ ui('刷新', 'Refresh') }}</el-button>
        <el-button type="primary" @click="wizardVisible = true">
          <IconTablerPlus />{{ ui('新建工业连接', 'New Industrial Connection') }}
        </el-button>
      </div>
    </header>

    <section class="industrial-workbench__surface">
      <CollectorConnectionList
        v-show="!connectionListCollapsed"
        ref="connectionList"
        :project-id="projectId"
        :selected-id="selectedId"
        :session-states="connectionSessions"
        @select="selectConnection"
        @connect="connectConnection"
        @disconnect="disconnectConnection"
        @collapse="connectionListCollapsed = true"
        @loaded="onConnectionsLoaded"
        @deleted="onConnectionDeleted"
      />

      <main class="industrial-workbench__main">
        <template v-if="activeConnection">
          <header class="industrial-workbench__connection-head">
            <div class="industrial-workbench__connection-leading">
              <button
                v-if="connectionListCollapsed"
                type="button"
                class="industrial-workbench__sidebar-toggle"
                :aria-label="ui('展开连接列表', 'Expand Connection List')"
                :title="ui('展开连接列表', 'Expand Connection List')"
                @click="connectionListCollapsed = false"
              >
                <IconTablerLayoutSidebarLeftExpand />
              </button>
              <div class="industrial-workbench__identity">
                <span class="industrial-workbench__mark"
                  ><CollectorDriverIcon
                    :protocol-family="activeConnection.protocolFamily"
                    :driver-id="activeConnection.driverId"
                    :default-acquisition="activeConnection.defaultAcquisition"
                /></span>
                <div>
                  <div class="industrial-workbench__name-row">
                    <h2>{{ activeConnection.name }}</h2>
                  </div>
                  <p>
                    <span>{{
                      formatCollectorProtocolFamily(activeConnection.protocolFamily)
                    }}</span>
                    <span v-if="activeDriver">{{
                      formatCollectorDriverDisplayName(activeDriver)
                    }}</span>
                    <span>{{ ui('驱动版本', 'Driver Version') }} {{ activeConnection.driverVersion }}</span>
                  </p>
                </div>
              </div>
            </div>
          </header>

          <section class="industrial-workbench__content">
            <el-tabs v-model="activeTab" class="industrial-workbench__tabs">
              <el-tab-pane :label="ui('连接配置', 'Connection')" name="config">
                <CollectorConnectionEditor
                  :project-id="projectId"
                  :connection="activeConnection"
                  :agent="selectedAgent"
                  @saved="onConnectionSaved"
                  @refresh-agent="refreshAgentResources"
                />
              </el-tab-pane>
              <el-tab-pane :label="ui('变量建模', 'Point Modeling')" name="points">
                <div class="industrial-workbench__points">
                  <CollectorPointGroupTree
                    :key="activeConnection.id"
                    :project-id="projectId"
                    :connection-id="activeConnection.id"
                    @select="groupId = $event"
                    @create-point="pointTable?.openCreate($event)"
                    @changed="onPointGroupsChanged"
                  />
                  <CollectorPointTable
                    ref="pointTable"
                    :project-id="projectId"
                    :connection-id="activeConnection.id"
                    :connection-name="activeConnection.name"
                    :driver-id="activeConnection.driverId"
                    :group-id="groupId"
                    :show-element-count="supportsElementCount"
                    :supports-point-read="supportsPointRead"
                    :can-read-current-page="canRead"
                    :read-current-page-loading="pointReadLoading"
                    :read-current-page-disabled-reason="pointReadDisabledReason"
                    @read-current-page="readCurrentPage"
                    @import="importVisible = true"
                    @navigate-datapoint="
                      emit('navigate', { module: 'datapoint', objectId: $event })
                    "
                    @saved="discoveryPanel?.refreshExisting()"
                  />
                </div>
              </el-tab-pane>
              <el-tab-pane :label="ui('在线验证', 'Online Validation')" name="validation">
                <div class="industrial-workbench__validation">
                  <header class="industrial-workbench__validation-head">
                    <div class="industrial-workbench__agent">
                      <span>
                        <i :class="{ 'is-online': selectedAgent?.status === 'online' }" />{{ ui('调试代理', 'Debug Agent') }}
                      </span>
                      <CollectorAgentSelector
                        ref="agentSelector"
                        v-model="agentId"
                        :project-id="projectId"
                        @change="selectedAgent = $event"
                      />
                    </div>
                    <div class="industrial-workbench__connection-actions">
                      <div
                        class="industrial-workbench__connection-state"
                        :class="`is-${connectionStatus.tone}`"
                      >
                        <strong><i />{{ connectionStatus.label }}</strong>
                        <time v-if="connectionSessionTime">{{ connectionSessionTime }}</time>
                      </div>
                      <el-tooltip
                        :disabled="!connectionDisabledReason"
                        :content="connectionDisabledReason"
                        placement="bottom"
                      >
                        <span>
                          <el-button
                            :type="
                              activeSessionState.status === 'connected' ? 'default' : 'primary'
                            "
                            :disabled="!canManageConnection || connectionSessionBusy"
                            :loading="connectionSessionBusy"
                            @click="toggleActiveConnection"
                          >
                            {{ connectionActionLabel }}
                          </el-button>
                        </span>
                      </el-tooltip>
                    </div>
                  </header>
                  <el-tabs v-model="validationTab" class="industrial-workbench__validation-tabs">
                    <el-tab-pane v-if="supportsDeviceBrowse" :label="ui('设备浏览', 'Device Browse')" name="discovery">
                      <CollectorDiscoveryPanel
                        ref="discoveryPanel"
                        :project-id="projectId"
                        :connection-id="activeConnection.id"
                        :protocol-family="activeConnection.protocolFamily"
                        :agent-id="agentId"
                        :workspace-session-id="workspaceSessionId"
                        :enabled="canBrowse"
                        :batch-loading="discoveryBatchLoading"
                        @points="createDiscoveredPoints"
                        @create-point="openDiscoveredPoint"
                        @session-error="markActiveSessionError"
                      />
                    </el-tab-pane>
                    <el-tab-pane :label="ui('连接诊断', 'Diagnostics')" name="diagnostic">
                      <CollectorDiagnosticPanel
                        :project-id="projectId"
                        :connection-id="activeConnection.id"
                      />
                    </el-tab-pane>
                  </el-tabs>
                </div>
              </el-tab-pane>
            </el-tabs>
          </section>
        </template>

        <template v-else>
          <header
            v-if="connectionListCollapsed"
            class="industrial-workbench__connection-head is-empty"
          >
            <button
              type="button"
              class="industrial-workbench__sidebar-toggle"
              :aria-label="ui('展开连接列表', 'Expand Connection List')"
              :title="ui('展开连接列表', 'Expand Connection List')"
              @click="connectionListCollapsed = false"
            >
              <IconTablerLayoutSidebarLeftExpand />
            </button>
          </header>
          <section class="industrial-workbench__empty">
            <span><IconTablerTopologyStar3 /></span>
            <strong>{{ ui('建立第一条工业采集连接', 'Create Your First Industrial Connection') }}</strong>
            <p>{{ ui('无需调试代理也可以先完成协议配置和变量建模，连接测试与设备浏览可稍后执行。', 'Configure the protocol and model points without a debug agent. Connection tests and device browsing can be done later.') }}</p>
            <el-button type="primary" @click="wizardVisible = true">{{ ui('创建工业采集连接', 'Create Industrial Connection') }}</el-button>
          </section>
        </template>
      </main>
    </section>

    <CollectorConnectionWizard
      v-model="wizardVisible"
      :project-id="projectId"
      :agent="selectedAgent"
      @created="onCreated"
      @refresh-agent="refreshAgentResources"
    />
    <CollectorImportDialog
      v-if="activeConnection"
      v-model="importVisible"
      :project-id="projectId"
      :connection-id="activeConnection.id"
      :driver-id="activeConnection.driverId"
      @committed="pointTable?.reload()"
    />
    <CollectorBatchResultDialog
      v-model="batchResultVisible"
      :title="batchResultTitle"
      :success-count="batchSuccessCount"
      :failed="batchFailures"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import dayjs from 'dayjs'
import { ElMessage } from 'element-plus'
import {
  createCollectorPointsBatch,
  createCollectorTask,
  getCollectorConnection,
  getCollectorDriver,
  getCollectorTask,
} from '@/api/collector.api'
import type { CollectorAgent } from '@/api/schemas/collector-dev.schema'
import { CollectorPointReadResultSchema } from '@/api/schemas/collector.schema'
import type {
  CollectorConnection,
  CollectorDriverDetail,
  CollectorPoint,
  CollectorPointBatchFailure,
  CollectorTask,
} from '@/api/schemas/collector.schema'
import {
  agentSupportsOperation,
  collectorDriverFeatures,
  collectorDriverSupportsFeature,
  formatCollectorDriverDisplayName,
  formatCollectorProtocolFamily,
  type CollectorDebugConnectionState,
  type CollectorPointCreateDefaults,
} from './collector-workbench-model'
import IconTablerCpu from '~icons/tabler/cpu'
import IconTablerLayoutSidebarLeftExpand from '~icons/tabler/layout-sidebar-left-expand'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerTopologyStar3 from '~icons/tabler/topology-star-3'
import CollectorAgentSelector from './CollectorAgentSelector.vue'
import CollectorBatchResultDialog from './CollectorBatchResultDialog.vue'
import CollectorConnectionEditor from './CollectorConnectionEditor.vue'
import CollectorDiagnosticPanel from './CollectorDiagnosticPanel.vue'
import CollectorDriverIcon from './CollectorDriverIcon.vue'
import CollectorConnectionList from './CollectorConnectionList.vue'
import CollectorConnectionWizard from './CollectorConnectionWizard.vue'
import CollectorDiscoveryPanel from './CollectorDiscoveryPanel.vue'
import CollectorImportDialog from './CollectorImportDialog.vue'
import CollectorPointGroupTree from './CollectorPointGroupTree.vue'
import CollectorPointTable from './CollectorPointTable.vue'
import {
  currentPageCollectorPointIds,
  summarizeCollectorPointRead,
} from './collector-debug-snapshot'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = defineProps<{
  projectId: string
  connection?: { id: string }
}>()
const emit = defineEmits<{
  navigate: [payload: { module: 'datapoint'; objectId: string }]
}>()
const workspaceSessionId = crypto.randomUUID()
const selectedId = ref(props.connection?.id || '')
const activeConnection = ref<CollectorConnection | null>(null)
const activeDriver = ref<CollectorDriverDetail | null>(null)
const agentId = ref('')
const selectedAgent = ref<CollectorAgent>()
const activeTab = ref('config')
const validationTab = ref('diagnostic')
const wizardVisible = ref(false)
const importVisible = ref(false)
const batchResultVisible = ref(false)
const batchResultTitle = ref(ui('批量新增结果', 'Batch Create Results'))
const batchSuccessCount = ref(0)
const batchFailures = ref<CollectorPointBatchFailure[]>([])
const groupId = ref<string | null>(null)
const connectionListCollapsed = ref(false)
const connectionList = ref<InstanceType<typeof CollectorConnectionList>>()
const agentSelector = ref<InstanceType<typeof CollectorAgentSelector>>()
const pointTable = ref<InstanceType<typeof CollectorPointTable>>()
const discoveryPanel = ref<InstanceType<typeof CollectorDiscoveryPanel>>()
const discoveryBatchLoading = ref(false)
const pointReadLoading = ref(false)
const connectionSessions = ref<Record<string, CollectorDebugConnectionState>>({})
const activeSessionState = computed<CollectorDebugConnectionState>(() =>
  activeConnection.value
    ? connectionSessions.value[activeConnection.value.id] || { status: 'disconnected' }
    : { status: 'disconnected' },
)
const connectionSessionBusy = computed(() =>
  ['connecting', 'disconnecting'].includes(activeSessionState.value.status),
)
const connectionStatus = computed(() => {
  const status = activeSessionState.value.status
  if (status === 'connected') return { label: ui('已连接', 'Connected'), tone: 'success' as const }
  if (status === 'connecting') return { label: ui('连接中', 'Connecting'), tone: 'primary' as const }
  if (status === 'disconnecting') return { label: ui('断开中', 'Disconnecting'), tone: 'primary' as const }
  if (status === 'error') return { label: ui('连接异常', 'Connection Error'), tone: 'danger' as const }
  return { label: ui('未连接', 'Disconnected'), tone: 'warning' as const }
})
const connectionSessionTime = computed(() => activeSessionState.value.connectedAt || '')
const connectionActionLabel = computed(() => {
  if (activeSessionState.value.status === 'connected') return ui('断开', 'Disconnect')
  if (activeSessionState.value.status === 'error') return ui('重新连接', 'Reconnect')
  return ui('连接', 'Connect')
})

const canManageConnection = computed(() =>
  activeConnection.value
    ? agentSupportsOperation(
        selectedAgent.value,
        activeConnection.value.driverId,
        activeConnection.value.driverVersion,
        activeConnection.value.schemaVersion,
        'connection.open',
      ) &&
      agentSupportsOperation(
        selectedAgent.value,
        activeConnection.value.driverId,
        activeConnection.value.driverVersion,
        activeConnection.value.schemaVersion,
        'connection.close',
      )
    : false,
)
const connectionDisabledReason = computed(() => {
  if (!selectedAgent.value) return ui('请先选择采集调试代理', 'Select a collector debug agent first')
  if (selectedAgent.value.status !== 'online') return ui('所选采集调试代理当前离线', 'The selected debug agent is offline')
  if (!canManageConnection.value) return ui('代理缺少当前驱动、版本或 Schema 能力', 'The agent does not support the current driver, version, or schema')
  return ''
})

const supportsDeviceBrowse = computed(
  () => activeDriver.value?.operations.includes('device.browse') ?? false,
)
const supportsPointRead = computed(
  () => activeDriver.value?.operations.includes('point.read') ?? false,
)
const supportsElementCount = computed(() =>
  collectorDriverSupportsFeature(activeDriver.value, collectorDriverFeatures.pointElementCount),
)
const canBrowse = computed(() =>
  activeConnection.value &&
  supportsDeviceBrowse.value &&
  activeSessionState.value.status === 'connected'
    ? agentSupportsOperation(
        selectedAgent.value,
        activeConnection.value.driverId,
        activeConnection.value.driverVersion,
        activeConnection.value.schemaVersion,
        'device.browse',
      )
    : false,
)
const canRead = computed(() =>
  activeConnection.value &&
  supportsPointRead.value &&
  activeSessionState.value.status === 'connected'
    ? agentSupportsOperation(
        selectedAgent.value,
        activeConnection.value.driverId,
        activeConnection.value.driverVersion,
        activeConnection.value.schemaVersion,
        'point.read',
      )
    : false,
)
const pointReadDisabledReason = computed(() => {
  if (!selectedAgent.value) return ui('请先选择采集调试代理', 'Select a collector debug agent first')
  if (selectedAgent.value.status !== 'online') return ui('所选采集调试代理当前离线', 'The selected debug agent is offline')
  if (activeSessionState.value.status !== 'connected') return ui('请先连接设备', 'Connect to the device first')
  if (!canRead.value) return ui('代理缺少当前驱动、版本、Schema 或变量读取能力', 'The agent cannot read points with the current driver, version, or schema')
  return ''
})

function updateConnectionSession(connectionId: string, state: CollectorDebugConnectionState) {
  connectionSessions.value = { ...connectionSessions.value, [connectionId]: state }
}

async function waitTaskCompletion(task: CollectorTask, timeoutMessage: string) {
  let current = task
  for (let index = 0; index < 30; index++) {
    current = await getCollectorTask(props.projectId, current.taskId)
    if (current.status === 'succeeded') return current
    if (['failed', 'cancelled', 'expired'].includes(current.status)) {
      throw new Error(current.errorMessage || timeoutMessage)
    }
    await new Promise((resolve) => setTimeout(resolve, 1000))
  }
  throw new Error(timeoutMessage)
}

async function connectConnection(connection: CollectorConnection) {
  const currentAgentId = agentId.value
  if (
    !currentAgentId ||
    ['connecting', 'connected'].includes(connectionSessions.value[connection.id]?.status || '')
  ) {
    return
  }
  if (
    !agentSupportsOperation(
      selectedAgent.value,
      connection.driverId,
      connection.driverVersion,
      connection.schemaVersion,
      'connection.open',
    )
  ) {
    ElMessage.warning(ui('当前调试代理不支持该驱动的长连接', 'The selected debug agent does not support persistent connections for this driver'))
    return
  }

  updateConnectionSession(connection.id, { status: 'connecting' })
  try {
    const task = await createCollectorTask(props.projectId, {
      agentId: currentAgentId,
      connectionId: connection.id,
      operation: 'connection.open',
      input: { workspaceSessionId },
      timeoutSeconds: 120,
    })
    const completed = await waitTaskCompletion(task, ui('建立调试长连接超时', 'Timed out while opening the debug connection'))
    const result = completed.result as {
      connected?: boolean
      serverName?: string
      connectedAt?: string
    } | null
    if (result?.connected !== true) throw new Error(ui('调试长连接未进入已连接状态', 'The debug connection did not enter the connected state'))
    updateConnectionSession(connection.id, {
      status: 'connected',
      connectedAt: result.connectedAt
        ? dayjs(result.connectedAt).format('YYYY-MM-DD HH:mm:ss')
        : dayjs().format('YYYY-MM-DD HH:mm:ss'),
      serverName: result.serverName || undefined,
    })
    ElMessage.success(ui(`“${connection.name}”已连接`, `“${connection.name}” connected`))
  } catch (error) {
    const message = error instanceof Error ? error.message : ui('建立调试长连接失败', 'Failed to open the debug connection')
    updateConnectionSession(connection.id, { status: 'error', message })
    ElMessage.error(message)
  }
}

async function disconnectConnection(
  connection: CollectorConnection,
  options: { silent?: boolean; agentId?: string } = {},
) {
  const currentAgentId = options.agentId || agentId.value
  if (!currentAgentId) {
    updateConnectionSession(connection.id, { status: 'disconnected' })
    return
  }
  if (connectionSessions.value[connection.id]?.status === 'disconnecting') return

  updateConnectionSession(connection.id, { status: 'disconnecting' })
  try {
    const task = await createCollectorTask(props.projectId, {
      agentId: currentAgentId,
      connectionId: connection.id,
      operation: 'connection.close',
      input: { workspaceSessionId },
    })
    if (!options.silent) await waitTaskCompletion(task, ui('断开调试长连接超时', 'Timed out while closing the debug connection'))
    updateConnectionSession(connection.id, { status: 'disconnected' })
    if (!options.silent) ElMessage.success(ui(`“${connection.name}”已断开`, `“${connection.name}” disconnected`))
  } catch (error) {
    const message = error instanceof Error ? error.message : ui('断开调试长连接失败', 'Failed to close the debug connection')
    updateConnectionSession(
      connection.id,
      options.silent ? { status: 'disconnected' } : { status: 'error', message },
    )
    if (!options.silent) ElMessage.error(message)
  }
}

function markActiveSessionError(message: string) {
  if (!activeConnection.value) return
  updateConnectionSession(activeConnection.value.id, { status: 'error', message })
}

function toggleActiveConnection() {
  const connection = activeConnection.value
  if (!connection) return
  if (activeSessionState.value.status === 'connected') void disconnectConnection(connection)
  else void connectConnection(connection)
}

function closeAllPageSessions(currentAgentId = agentId.value) {
  if (!currentAgentId) return
  const connectedIds = Object.entries(connectionSessions.value)
    .filter(([, state]) => state.status !== 'disconnected')
    .map(([connectionId]) => connectionId)
  for (const connectionId of connectedIds) {
    void createCollectorTask(props.projectId, {
      agentId: currentAgentId,
      connectionId,
      operation: 'connection.close',
      input: { workspaceSessionId },
    }).catch(() => undefined)
  }
}

async function selectConnection(id: string) {
  selectedId.value = id
  const connection = await getCollectorConnection(props.projectId, id)
  const driver = await getCollectorDriver(connection.driverId)
  if (selectedId.value !== id) return
  activeConnection.value = connection
  activeDriver.value = driver
  activeTab.value = 'config'
  groupId.value = null
}
function onConnectionsLoaded(items: CollectorConnection[]) {
  if (!selectedId.value && items[0]) void selectConnection(items[0].id)
  else if (selectedId.value && !activeConnection.value) void selectConnection(selectedId.value)
}
function onConnectionDeleted(id: string) {
  const nextSessions = { ...connectionSessions.value }
  delete nextSessions[id]
  connectionSessions.value = nextSessions
  if (selectedId.value !== id) return
  selectedId.value = ''
  activeConnection.value = null
  activeDriver.value = null
  activeTab.value = 'config'
  groupId.value = null
}

async function onCreated(id: string) {
  await connectionList.value?.reload()
  await selectConnection(id)
}
async function onConnectionSaved(connection: CollectorConnection) {
  if (connectionSessions.value[connection.id]?.status === 'connected') {
    await disconnectConnection(connection)
  }
  activeConnection.value = connection
  activeDriver.value = await getCollectorDriver(connection.driverId)
  await connectionList.value?.reload()
}
async function reloadPointTableAfterRead(connectionId: string) {
  if (activeConnection.value?.id !== connectionId) return
  try {
    await pointTable.value?.reload()
  } catch {
    ElMessage.warning(ui('调试结果已保存，但刷新当前页失败', 'The debug result was saved, but the current page could not be refreshed'))
  }
}

async function readCurrentPage(points: Array<Pick<CollectorPoint, 'id' | 'name'>>) {
  const connection = activeConnection.value
  const currentAgentId = agentId.value
  if (
    !connection ||
    !currentAgentId ||
    !canRead.value ||
    pointReadLoading.value ||
    points.length === 0
  ) {
    return
  }

  pointReadLoading.value = true
  try {
    const task = await createCollectorTask(props.projectId, {
      agentId: currentAgentId,
      connectionId: connection.id,
      operation: 'point.read',
      input: { workspaceSessionId, pointIds: currentPageCollectorPointIds(points) },
    })
    const completed = await waitTaskCompletion(task, ui('获取当前页数据超时', 'Timed out while reading the current page'))
    const result = CollectorPointReadResultSchema.parse(completed.result)
    await reloadPointTableAfterRead(connection.id)

    const { successCount, failures } = summarizeCollectorPointRead(points, result)
    if (failures.length === 0) {
      ElMessage.success(ui(`已获取当前页 ${successCount} 个变量`, `Read ${successCount} point${successCount === 1 ? '' : 's'} on the current page`))
      return
    }
    batchResultTitle.value = ui('当前页数据获取结果', 'Current Page Read Results')
    batchSuccessCount.value = successCount
    batchFailures.value = failures
    batchResultVisible.value = true
  } catch (error) {
    const message = error instanceof Error ? error.message : ui('获取当前页数据失败', 'Failed to read the current page')
    await reloadPointTableAfterRead(connection.id)
    batchResultTitle.value = ui('当前页数据获取结果', 'Current Page Read Results')
    batchSuccessCount.value = 0
    batchFailures.value = points.map((point, index) => ({
      index,
      name: point.name,
      code: 'COLLECTOR_POINT_READ_FAILED',
      message,
    }))
    batchResultVisible.value = true
    if (
      activeConnection.value?.id === connection.id &&
      /会话.*(断开|建立)|连接.*断开/.test(message)
    ) {
      markActiveSessionError(message)
    }
    ElMessage.error(message)
  } finally {
    pointReadLoading.value = false
  }
}

async function createDiscoveredPoints(points: Record<string, unknown>[]) {
  if (!activeConnection.value || points.length === 0) return
  discoveryBatchLoading.value = true
  try {
    const result = await createCollectorPointsBatch(
      props.projectId,
      activeConnection.value.id,
      points,
    )
    const failedIndexes = new Set(result.failed.map((item) => item.index))
    const createdIndexes = points
      .map((_, index) => index)
      .filter((index) => !failedIndexes.has(index))
    discoveryPanel.value?.markSelectedCreated(createdIndexes)
    if (result.list.length > 0) await pointTable.value?.reload(1)
    if (result.failed.length === 0) {
      ElMessage.success(ui(`已新增 ${result.list.length} 个变量`, `Created ${result.list.length} point${result.list.length === 1 ? '' : 's'}`))
    } else {
      batchResultTitle.value = ui('批量新增结果', 'Batch Create Results')
      batchSuccessCount.value = result.list.length
      batchFailures.value = result.failed
      batchResultVisible.value = true
    }
  } finally {
    discoveryBatchLoading.value = false
  }
}
function openDiscoveredPoint(defaults: CollectorPointCreateDefaults) {
  pointTable.value?.openCreate(groupId.value, defaults, true)
}
function refreshAgentResources() {
  void agentSelector.value?.reload()
}
async function onPointGroupsChanged() {
  await Promise.all([pointTable.value?.reloadGroups(), pointTable.value?.reload(1)])
}
watch(agentId, (nextAgentId, previousAgentId) => {
  if (previousAgentId && previousAgentId !== nextAgentId) closeAllPageSessions(previousAgentId)
  connectionSessions.value = {}
})
onBeforeUnmount(() => closeAllPageSessions())
</script>

<style scoped>
.industrial-workbench {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  gap: 14px;
  color: var(--dc-text);
}
.industrial-workbench__toolbar {
  display: flex;
  min-height: 64px;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 0 20px;
  border: 1px solid var(--dc-border);
  border-radius: 14px;
  background: var(--dc-surface-raised);
  box-shadow:
    0 8px 24px rgba(15, 23, 42, 0.06),
    0 1px 2px rgba(15, 23, 42, 0.04);
}
.industrial-workbench__title,
.industrial-workbench__identity,
.industrial-workbench__name-row,
.industrial-workbench__toolbar-actions {
  display: flex;
  align-items: center;
}
.industrial-workbench__title {
  min-width: 0;
  gap: 11px;
}
.industrial-workbench__title-icon {
  display: grid;
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  place-items: center;
  border-radius: var(--dc-radius-md);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.industrial-workbench__title-icon svg {
  width: 20px;
  height: 20px;
}
.industrial-workbench__title h1 {
  margin: 0;
  font-size: 17px;
  line-height: 1.25;
}

.industrial-workbench__toolbar-actions {
  gap: 9px;
}
.industrial-workbench__toolbar-actions :deep(.el-button svg) {
  width: 15px;
  margin-right: 6px;
}
.industrial-workbench__agent {
  display: flex;
  align-items: center;
  gap: 9px;
  margin-right: 4px;
}
.industrial-workbench__agent > span {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}
.industrial-workbench__agent i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--dc-border-strong);
}
.industrial-workbench__agent i.is-online {
  background: var(--dc-success);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--dc-success) 13%, transparent);
}
.industrial-workbench__surface {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: 14px;
  background: var(--dc-surface-raised);
  box-shadow:
    0 8px 24px rgba(15, 23, 42, 0.055),
    0 1px 2px rgba(15, 23, 42, 0.04);
}
.industrial-workbench__main {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  background: var(--dc-surface-subtle);
}
.industrial-workbench__connection-head {
  display: flex;
  min-height: 76px;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 13px 20px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}
.industrial-workbench__connection-head.is-empty {
  min-height: 58px;
  justify-content: flex-start;
}
.industrial-workbench__connection-leading {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 12px;
}
.industrial-workbench__sidebar-toggle {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  place-items: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-muted);
  cursor: pointer;
}
.industrial-workbench__sidebar-toggle:hover {
  border-color: var(--dc-primary);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.industrial-workbench__sidebar-toggle svg {
  width: 17px;
  height: 17px;
}
.industrial-workbench__identity {
  min-width: 0;
  gap: 12px;
}
.industrial-workbench__mark {
  display: grid;
  width: 40px;
  height: 40px;
  flex: 0 0 40px;
  place-items: center;
  border-radius: 10px;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.industrial-workbench__mark svg {
  width: 21px;
  height: 21px;
}
.industrial-workbench__identity > div {
  min-width: 0;
}
.industrial-workbench__name-row {
  gap: 9px;
}
.industrial-workbench__name-row h2 {
  margin: 0;
  overflow: hidden;
  font-size: 17px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.industrial-workbench__identity p {
  display: flex;
  gap: 9px;
  margin: 5px 0 0;
  color: var(--dc-text-muted);
  font-size: 11px;
}
.industrial-workbench__identity p span + span::before {
  margin-right: 9px;
  color: var(--dc-border-strong);
  content: '·';
}
.industrial-workbench__connection-actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 14px;
}
.industrial-workbench__connection-state {
  display: flex;
  min-width: 168px;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  color: var(--dc-text-muted);
  font-size: 10px;
}
.industrial-workbench__connection-state strong {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}
.industrial-workbench__connection-state time {
  color: var(--dc-text-muted);
  font-size: 10px;
  white-space: nowrap;
}
.industrial-workbench__connection-state i {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--dc-border-strong);
}
.industrial-workbench__connection-state.is-primary i {
  background: var(--dc-primary);
}
.industrial-workbench__connection-state.is-success i {
  background: var(--dc-success);
}
.industrial-workbench__connection-state.is-warning i {
  background: #d99000;
}
.industrial-workbench__connection-state.is-danger i {
  background: var(--dc-danger);
}
.industrial-workbench__content {
  display: flex;
  min-height: 0;
  flex: 1;
}
.industrial-workbench__tabs {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1;
  flex-direction: column;
}
.industrial-workbench__tabs :deep(.el-tabs__header) {
  flex: 0 0 46px;
  margin: 0;
  padding: 0 20px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}
.industrial-workbench__tabs :deep(.el-tabs__nav-wrap::after) {
  display: none;
}
.industrial-workbench__tabs :deep(.el-tabs__item) {
  height: 46px;
  color: var(--dc-text-secondary);
  font-size: 13px;
}
.industrial-workbench__tabs :deep(.el-tabs__item.is-active) {
  color: var(--dc-primary);
  font-weight: 700;
}
.industrial-workbench__tabs :deep(.el-tabs__active-bar) {
  background: var(--dc-primary);
}
.industrial-workbench__tabs :deep(.el-tabs__content) {
  min-height: 0;
  flex: 1;
  padding: 14px 16px 16px;
  overflow: hidden;
}
.industrial-workbench__tabs :deep(.el-tab-pane) {
  height: 100%;
  min-height: 0;
  overflow: auto;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  padding: 16px;
}
.industrial-workbench__validation {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
}
.industrial-workbench__validation-head {
  display: flex;
  min-height: 54px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 0 0 14px;
  border-bottom: 1px solid var(--dc-border);
}
.industrial-workbench__validation-tabs {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
}
.industrial-workbench__validation-tabs :deep(.el-tabs__header) {
  flex: 0 0 42px;
  padding: 0;
  border-bottom: 1px solid var(--dc-border);
}
.industrial-workbench__validation-tabs :deep(.el-tabs__content) {
  min-height: 0;
  flex: 1;
  padding: 12px 0 0;
}
.industrial-workbench__validation-tabs :deep(.el-tab-pane) {
  height: 100%;
  min-height: 0;
  overflow: auto;
  padding: 0;
  border: 0;
}

.industrial-workbench__points {
  display: flex;
  height: 100%;
  min-height: 0;
}
.industrial-workbench__empty {
  display: flex;
  max-width: 520px;
  align-self: center;
  flex: 1;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  padding: 40px;
  text-align: center;
}
.industrial-workbench__empty > span {
  display: grid;
  width: 68px;
  height: 68px;
  place-items: center;
  margin-bottom: 18px;
  border: 1px solid color-mix(in srgb, var(--dc-primary) 18%, var(--dc-border));
  border-radius: 20px;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.industrial-workbench__empty svg {
  width: 32px;
  height: 32px;
}
.industrial-workbench__empty strong {
  font-size: 18px;
}
.industrial-workbench__empty p {
  max-width: 430px;
  margin: 10px 0 20px;
  color: var(--dc-text-muted);
  font-size: 13px;
  line-height: 1.7;
}
@media (max-width: 1100px) {
  .industrial-workbench__title p,
  .industrial-workbench__connection-state {
    display: none;
  }
}
</style>
