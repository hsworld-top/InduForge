<template>
  <div class="industrial-workbench">
    <header class="industrial-workbench__toolbar">
      <div class="industrial-workbench__title">
        <span class="industrial-workbench__title-icon"><IconTablerCpu /></span>
        <h1>工业采集</h1>
      </div>
      <div class="industrial-workbench__toolbar-actions">
        <div class="industrial-workbench__agent">
          <span
            ><i :class="{ 'is-online': selectedAgent?.status === 'online' }" />采集调试代理</span
          >
          <CollectorAgentSelector
            v-model="agentId"
            :project-id="projectId"
            @change="selectedAgent = $event"
          />
        </div>
        <el-button @click="connectionList?.reload(1)">刷新</el-button>
        <el-button type="primary" @click="wizardVisible = true">
          <IconTablerPlus />新建工业连接
        </el-button>
      </div>
    </header>

    <section class="industrial-workbench__surface">
      <CollectorConnectionList
        v-show="!connectionListCollapsed"
        ref="connectionList"
        :project-id="projectId"
        :selected-id="selectedId"
        @select="selectConnection"
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
                aria-label="展开连接列表"
                title="展开连接列表"
                @click="connectionListCollapsed = false"
              >
                <IconTablerLayoutSidebarLeftExpand />
              </button>
              <div class="industrial-workbench__identity">
                <span class="industrial-workbench__mark"
                  ><CollectorDriverIcon
                    :protocol-family="activeConnection.protocolFamily"
                    :driver-id="activeConnection.driverId"
                /></span>
                <div>
                  <div class="industrial-workbench__name-row">
                    <h2>{{ activeConnection.name }}</h2>
                  </div>
                  <p>
                    <span>{{
                      formatCollectorProtocolFamily(activeConnection.protocolFamily)
                    }}</span>
                    <span
                      >{{ activeConnection.driverId }}@{{ activeConnection.driverVersion }}</span
                    >
                    <span>{{
                      selectedAgent ? formatCollectorAgentName(selectedAgent) : '未选择调试代理'
                    }}</span>
                  </p>
                </div>
              </div>
            </div>
            <div class="industrial-workbench__connection-actions">
              <div
                class="industrial-workbench__connection-state"
                :class="`is-${connectionStatus.tone}`"
              >
                <strong><i />{{ connectionStatus.label }}</strong>
                <time v-if="connectionTestTime">
                  {{ connectionStatus.timeLabel }}：{{ connectionTestTime }}
                </time>
              </div>
              <el-button
                :disabled="!canTest || connectionTestLoading"
                :loading="connectionTestLoading"
                @click="runConnectionTest"
              >
                测试连接
              </el-button>
            </div>
          </header>

          <section class="industrial-workbench__content">
            <el-tabs v-model="activeTab" class="industrial-workbench__tabs">
              <el-tab-pane label="连接配置" name="config">
                <CollectorConnectionEditor
                  :project-id="projectId"
                  :connection="activeConnection"
                  @saved="onConnectionSaved"
                />
              </el-tab-pane>
              <el-tab-pane label="变量配置" name="points">
                <div class="industrial-workbench__points">
                  <CollectorPointGroupTree
                    :project-id="projectId"
                    :connection-id="activeConnection.id"
                    @select="groupId = $event"
                  />
                  <CollectorPointTable
                    ref="pointTable"
                    :project-id="projectId"
                    :connection-id="activeConnection.id"
                    :group-id="groupId"
                    @selection="selectedPointIds = $event"
                    @import="importVisible = true"
                  />
                </div>
              </el-tab-pane>
              <el-tab-pane label="设备浏览" name="discovery">
                <CollectorDiscoveryPanel
                  :project-id="projectId"
                  :connection-id="activeConnection.id"
                  :agent-id="agentId"
                  :enabled="canBrowse"
                  @points="createDiscoveredPoints"
                />
              </el-tab-pane>
              <el-tab-pane label="实时调试" name="debug">
                <CollectorLiveDebugPanel
                  :project-id="projectId"
                  :connection-id="activeConnection.id"
                  :agent-id="agentId"
                  :point-ids="selectedPointIds"
                  :enabled="canRead"
                />
              </el-tab-pane>
              <el-tab-pane label="通信诊断" name="diagnostics">
                <CollectorDiagnosticsPanel
                  :enabled="canTest"
                  :loading="connectionTestLoading"
                  :task="activeConnectionTestTask || null"
                  @run="runConnectionTest"
                />
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
              aria-label="展开连接列表"
              title="展开连接列表"
              @click="connectionListCollapsed = false"
            >
              <IconTablerLayoutSidebarLeftExpand />
            </button>
          </header>
          <section class="industrial-workbench__empty">
            <span><IconTablerTopologyStar3 /></span>
            <strong>建立第一条工业采集连接</strong>
            <p>无需调试代理也可以先完成协议配置和变量建模，连接测试与设备浏览可稍后执行。</p>
            <el-button type="primary" @click="wizardVisible = true">创建工业采集连接</el-button>
          </section>
        </template>
      </main>
    </section>

    <CollectorConnectionWizard
      v-model="wizardVisible"
      :project-id="projectId"
      @created="onCreated"
    />
    <CollectorImportDialog
      v-if="activeConnection"
      v-model="importVisible"
      :project-id="projectId"
      :connection-id="activeConnection.id"
      @committed="pointTable?.reload()"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import dayjs from 'dayjs'
import { ElMessage } from 'element-plus'
import {
  createCollectorPointsBatch,
  createCollectorTask,
  getCollectorConnection,
  getCollectorTask,
} from '@/api/collector.api'
import type { CollectorAgent } from '@/api/schemas/collector-dev.schema'
import type { CollectorConnection, CollectorTask } from '@/api/schemas/collector.schema'
import {
  agentSupportsOperation,
  formatCollectorAgentName,
  formatCollectorProtocolFamily,
} from './collector-workbench-model'
import IconTablerCpu from '~icons/tabler/cpu'
import IconTablerLayoutSidebarLeftExpand from '~icons/tabler/layout-sidebar-left-expand'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerTopologyStar3 from '~icons/tabler/topology-star-3'
import CollectorAgentSelector from './CollectorAgentSelector.vue'
import CollectorConnectionEditor from './CollectorConnectionEditor.vue'
import CollectorDriverIcon from './CollectorDriverIcon.vue'
import CollectorConnectionList from './CollectorConnectionList.vue'
import CollectorConnectionWizard from './CollectorConnectionWizard.vue'
import CollectorDiagnosticsPanel from './CollectorDiagnosticsPanel.vue'
import CollectorDiscoveryPanel from './CollectorDiscoveryPanel.vue'
import CollectorImportDialog from './CollectorImportDialog.vue'
import CollectorLiveDebugPanel from './CollectorLiveDebugPanel.vue'
import CollectorPointGroupTree from './CollectorPointGroupTree.vue'
import CollectorPointTable from './CollectorPointTable.vue'

const props = defineProps<{
  projectId: string
  connection?: { id: string }
}>()
const selectedId = ref(props.connection?.id || '')
const activeConnection = ref<CollectorConnection | null>(null)
const agentId = ref('')
const selectedAgent = ref<CollectorAgent>()
const activeTab = ref('config')
const wizardVisible = ref(false)
const importVisible = ref(false)
const groupId = ref<string | null>(null)
const selectedPointIds = ref<string[]>([])
const connectionListCollapsed = ref(false)
const connectionList = ref<InstanceType<typeof CollectorConnectionList>>()
const pointTable = ref<InstanceType<typeof CollectorPointTable>>()
const connectionTestTasks = ref<Record<string, CollectorTask>>({})
const testingConnectionIds = ref(new Set<string>())
const activeConnectionTestTask = computed(() =>
  activeConnection.value ? connectionTestTasks.value[activeConnection.value.id] : undefined,
)
const connectionTestLoading = computed(() =>
  activeConnection.value ? testingConnectionIds.value.has(activeConnection.value.id) : false,
)
const connectionStatus = computed(() => {
  const taskStatus = activeConnectionTestTask.value?.status
  const status = taskStatus || activeConnection.value?.lastTestStatus
  if (!status) return { label: '未检测', tone: 'warning' as const, timeLabel: '上次测试' }
  if (status === 'succeeded')
    return { label: '连接正常', tone: 'success' as const, timeLabel: '上次测试' }
  if (['failed', 'cancelled', 'expired'].includes(status))
    return { label: '连接失败', tone: 'danger' as const, timeLabel: '上次测试' }
  return { label: '测试中', tone: 'primary' as const, timeLabel: '开始时间' }
})
const connectionTestTime = computed(() => {
  const task = activeConnectionTestTask.value
  const time = task?.finishedAt || task?.createdAt || activeConnection.value?.lastTestedAt
  return time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : ''
})

const canTest = computed(() =>
  activeConnection.value
    ? agentSupportsOperation(
        selectedAgent.value,
        activeConnection.value.driverId,
        activeConnection.value.driverVersion,
        activeConnection.value.schemaVersion,
        'connection.test',
      )
    : false,
)
const canBrowse = computed(() =>
  activeConnection.value
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
  activeConnection.value
    ? agentSupportsOperation(
        selectedAgent.value,
        activeConnection.value.driverId,
        activeConnection.value.driverVersion,
        activeConnection.value.schemaVersion,
        'point.read',
      )
    : false,
)

function updateConnectionTestTask(task: CollectorTask) {
  connectionTestTasks.value = { ...connectionTestTasks.value, [task.connectionId]: task }
}

function updateConnectionTesting(connectionId: string, testing: boolean) {
  const next = new Set(testingConnectionIds.value)
  if (testing) next.add(connectionId)
  else next.delete(connectionId)
  testingConnectionIds.value = next
}

async function runConnectionTest() {
  const connection = activeConnection.value
  const currentAgentId = agentId.value
  if (!connection || !currentAgentId || !canTest.value || connectionTestLoading.value) return
  updateConnectionTesting(connection.id, true)
  let task: CollectorTask | undefined
  try {
    task = await createCollectorTask(props.projectId, {
      agentId: currentAgentId,
      connectionId: connection.id,
      operation: 'connection.test',
      input: {},
    })
    updateConnectionTestTask(task)
    // 调试任务由 Agent 异步领取，页面只在有限时间内轮询，避免无限占用请求。
    for (let index = 0; index < 30; index++) {
      task = await getCollectorTask(props.projectId, task.taskId)
      updateConnectionTestTask(task)
      if (task.status === 'succeeded') {
        ElMessage.success('连接测试通过')
        return
      }
      if (['failed', 'cancelled', 'expired'].includes(task.status))
        throw new Error(task.errorMessage || '连接测试失败')
      await new Promise((resolve) => setTimeout(resolve, 1000))
    }
    throw new Error('连接测试超时')
  } catch (error) {
    const message = error instanceof Error ? error.message : '连接测试失败'
    if (task && !['succeeded', 'failed', 'cancelled', 'expired'].includes(task.status)) {
      updateConnectionTestTask({
        ...task,
        status: 'failed',
        errorMessage: message,
        finishedAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
      })
    }
    ElMessage.error(message)
  } finally {
    updateConnectionTesting(connection.id, false)
  }
}

async function selectConnection(id: string) {
  selectedId.value = id
  activeConnection.value = await getCollectorConnection(props.projectId, id)
  activeTab.value = 'config'
  groupId.value = null
  selectedPointIds.value = []
}
function onConnectionsLoaded(items: CollectorConnection[]) {
  if (!selectedId.value && items[0]) void selectConnection(items[0].id)
  else if (selectedId.value && !activeConnection.value) void selectConnection(selectedId.value)
}
function onConnectionDeleted(id: string) {
  if (selectedId.value !== id) return
  selectedId.value = ''
  activeConnection.value = null
  activeTab.value = 'config'
  groupId.value = null
  selectedPointIds.value = []
}

async function onCreated(id: string) {
  await connectionList.value?.reload(1)
  await selectConnection(id)
}
async function onConnectionSaved(connection: CollectorConnection) {
  activeConnection.value = connection
  await connectionList.value?.reload()
}
async function createDiscoveredPoints(points: Record<string, unknown>[]) {
  if (!activeConnection.value) return
  await createCollectorPointsBatch(props.projectId, activeConnection.value.id, points)
  activeTab.value = 'points'
  await pointTable.value?.reload(1)
}
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
