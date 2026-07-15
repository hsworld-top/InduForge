<template>
  <div class="industrial-workbench">
    <CollectorConnectionList
      ref="connectionList"
      :project-id="projectId"
      :selected-id="selectedId"
      @select="selectConnection"
      @create="wizardVisible = true"
      @loaded="onConnectionsLoaded"
    />
    <main class="industrial-workbench__main">
      <header class="industrial-workbench__header">
        <div class="industrial-workbench__identity">
          <span class="industrial-workbench__mark"><IconTablerCpu /></span>
          <div>
            <div class="industrial-workbench__eyebrow">
              <span>工业采集工作台</span>
              <em v-if="activeConnection" :class="{ 'is-enabled': activeConnection.enabled }">
                {{ activeConnection.enabled ? '采集已启用' : '采集已停用' }}
              </em>
            </div>
            <h2>{{ activeConnection?.name || '工程采集连接' }}</h2>
            <p v-if="activeConnection">
              {{ activeConnection.protocolFamily.toUpperCase() }} ·
              {{ activeConnection.driverId }}@{{ activeConnection.driverVersion }}
            </p>
            <p v-else>统一管理工业协议连接、设备发现、点位与开发态调试</p>
          </div>
        </div>
        <div class="industrial-workbench__actions">
          <div class="industrial-workbench__agent">
            <span
              ><i :class="{ 'is-online': selectedAgent?.status === 'online' }" />开发调试代理</span
            >
            <CollectorAgentSelector
              v-model="agentId"
              :project-id="projectId"
              @change="selectedAgent = $event"
            />
          </div>
          <el-button type="primary" @click="wizardVisible = true">
            <IconTablerPlus />新增连接
          </el-button>
        </div>
      </header>
      <section v-if="activeConnection" class="industrial-workbench__content">
        <el-tabs v-model="activeTab" class="industrial-workbench__tabs">
          <el-tab-pane label="连接配置" name="config"
            ><CollectorConnectionEditor
              :project-id="projectId"
              :connection="activeConnection"
              @saved="activeConnection = $event"
          /></el-tab-pane>
          <el-tab-pane label="设备发现" name="discovery"
            ><CollectorDiscoveryPanel
              :project-id="projectId"
              :connection-id="activeConnection.id"
              :agent-id="agentId"
              :enabled="canBrowse"
              @points="createDiscoveredPoints"
          /></el-tab-pane>
          <el-tab-pane label="点位管理" name="points"
            ><div class="industrial-workbench__points">
              <CollectorPointGroupTree
                :project-id="projectId"
                :connection-id="activeConnection.id"
                @select="groupId = $event"
              /><CollectorPointTable
                ref="pointTable"
                :project-id="projectId"
                :connection-id="activeConnection.id"
                :group-id="groupId"
                @selection="selectedPointIds = $event"
              />
            </div>
            <div class="industrial-workbench__point-actions">
              <el-button @click="importVisible = true">批量导入</el-button>
            </div></el-tab-pane
          >
          <el-tab-pane label="实时调试" name="debug"
            ><CollectorLiveDebugPanel
              :project-id="projectId"
              :connection-id="activeConnection.id"
              :agent-id="agentId"
              :point-ids="selectedPointIds"
              :enabled="canRead"
          /></el-tab-pane>
        </el-tabs>
      </section>
      <section v-else class="industrial-workbench__empty">
        <span><IconTablerTopologyStar3 /></span>
        <strong>建立第一条工业采集连接</strong>
        <p>无需调试代理也可以先完成协议配置和点位建模，连接测试与设备发现稍后执行。</p>
        <el-button type="primary" @click="wizardVisible = true">创建工业采集连接</el-button>
      </section>
    </main>
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
import { createCollectorPointsBatch, getCollectorConnection } from '@/api/collector.api'
import type { CollectorAgent } from '@/api/schemas/collector-dev.schema'
import type { CollectorConnection } from '@/api/schemas/collector.schema'
import { agentSupportsOperation } from './collector-workbench-model'
import IconTablerCpu from '~icons/tabler/cpu'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerTopologyStar3 from '~icons/tabler/topology-star-3'
import CollectorAgentSelector from './CollectorAgentSelector.vue'
import CollectorConnectionEditor from './CollectorConnectionEditor.vue'
import CollectorConnectionList from './CollectorConnectionList.vue'
import CollectorConnectionWizard from './CollectorConnectionWizard.vue'
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
const connectionList = ref<InstanceType<typeof CollectorConnectionList>>()
const pointTable = ref<InstanceType<typeof CollectorPointTable>>()
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
async function selectConnection(id: string) {
  selectedId.value = id
  activeConnection.value = await getCollectorConnection(props.projectId, id)
  activeTab.value = 'config'
  selectedPointIds.value = []
}
function onConnectionsLoaded(items: CollectorConnection[]) {
  if (!selectedId.value && items[0]) void selectConnection(items[0].id)
  else if (selectedId.value && !activeConnection.value) void selectConnection(selectedId.value)
}
async function onCreated(id: string) {
  await connectionList.value?.reload(1)
  await selectConnection(id)
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
  background: #f2f5f6;
  color: #1d2a32;
}
.industrial-workbench__main {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}
.industrial-workbench__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  min-height: 92px;
  padding: 15px 24px;
  border-bottom: 1px solid #dce4e7;
  background: #fff;
}
.industrial-workbench__identity {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 12px;
}
.industrial-workbench__mark {
  display: inline-flex;
  width: 42px;
  height: 42px;
  align-items: center;
  justify-content: center;
  border-radius: 11px;
  background: #123f52;
  color: #fff;
  box-shadow: 0 8px 18px rgb(18 63 82 / 18%);
}
.industrial-workbench__mark svg {
  width: 22px;
  height: 22px;
}
.industrial-workbench__identity > div:last-child {
  min-width: 0;
}
.industrial-workbench__eyebrow {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #75848c;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
}
.industrial-workbench__eyebrow em {
  padding: 2px 7px;
  border-radius: 999px;
  background: #eef1f2;
  color: #75828a;
  font-size: 10px;
  font-style: normal;
  letter-spacing: 0;
}
.industrial-workbench__eyebrow em.is-enabled {
  background: #e8f6ef;
  color: #217653;
}
.industrial-workbench__header h2 {
  margin: 4px 0 1px;
  font-size: 20px;
  line-height: 1.15;
}
.industrial-workbench__header p {
  margin: 0;
  overflow: hidden;
  color: #7b8990;
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.industrial-workbench__actions {
  display: flex;
  align-items: flex-end;
  gap: 12px;
}
.industrial-workbench__actions :deep(.el-button svg) {
  width: 15px;
  margin-right: 6px;
}
.industrial-workbench__agent {
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.industrial-workbench__agent > span {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #77868e;
  font-size: 11px;
  font-weight: 600;
}
.industrial-workbench__agent i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #b0bbc0;
}
.industrial-workbench__agent i.is-online {
  background: #2b9c70;
  box-shadow: 0 0 0 3px rgb(43 156 112 / 12%);
}
.industrial-workbench__content {
  min-height: 0;
  flex: 1;
  margin: 16px 18px 18px;
  padding: 0 18px 16px;
  border: 1px solid #dce4e7;
  border-radius: 10px;
  background: #fff;
  box-shadow: 0 8px 24px rgb(34 61 73 / 5%);
}
.industrial-workbench__tabs {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
}
.industrial-workbench__tabs :deep(.el-tabs__content),
.industrial-workbench__tabs :deep(.el-tab-pane) {
  height: 100%;
  min-height: 0;
}
.industrial-workbench__points {
  display: flex;
  height: calc(100% - 48px);
  min-height: 0;
}
.industrial-workbench__point-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 10px;
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
  display: inline-flex;
  width: 68px;
  height: 68px;
  align-items: center;
  justify-content: center;
  margin-bottom: 18px;
  border: 1px solid #cfe0e7;
  border-radius: 20px;
  background: #eaf4f7;
  color: #246b86;
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
  color: #73828a;
  font-size: 13px;
  line-height: 1.7;
}
</style>
