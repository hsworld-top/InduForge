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
        <div>
          <button type="button" @click="emit('back')">← 返回</button
          ><small>PROJECT INDUSTRIAL COLLECTOR</small>
          <h2>{{ connection?.name || '工业采集工作台' }}</h2>
        </div>
        <CollectorAgentSelector
          v-model="agentId"
          :project-id="projectId"
          @change="selectedAgent = $event"
        />
      </header>
      <template v-if="connection">
        <el-tabs v-model="activeTab" class="industrial-workbench__tabs">
          <el-tab-pane label="连接配置" name="config"
            ><CollectorConnectionEditor
              :project-id="projectId"
              :connection="connection"
              @saved="connection = $event"
          /></el-tab-pane>
          <el-tab-pane label="设备发现" name="discovery"
            ><CollectorDiscoveryPanel
              :project-id="projectId"
              :connection-id="connection.id"
              :agent-id="agentId"
              :enabled="canBrowse"
              @points="createDiscoveredPoints"
          /></el-tab-pane>
          <el-tab-pane label="点位管理" name="points"
            ><div class="industrial-workbench__points">
              <CollectorPointGroupTree
                :project-id="projectId"
                :connection-id="connection.id"
                @select="groupId = $event"
              /><CollectorPointTable
                ref="pointTable"
                :project-id="projectId"
                :connection-id="connection.id"
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
              :connection-id="connection.id"
              :agent-id="agentId"
              :point-ids="selectedPointIds"
              :enabled="canRead"
          /></el-tab-pane>
        </el-tabs>
      </template>
      <el-empty v-else description="请选择或创建工业连接" />
    </main>
    <CollectorConnectionWizard
      v-model="wizardVisible"
      :project-id="projectId"
      @created="onCreated"
    />
    <CollectorImportDialog
      v-if="connection"
      v-model="importVisible"
      :project-id="projectId"
      :connection-id="connection.id"
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
import CollectorAgentSelector from './CollectorAgentSelector.vue'
import CollectorConnectionEditor from './CollectorConnectionEditor.vue'
import CollectorConnectionList from './CollectorConnectionList.vue'
import CollectorConnectionWizard from './CollectorConnectionWizard.vue'
import CollectorDiscoveryPanel from './CollectorDiscoveryPanel.vue'
import CollectorImportDialog from './CollectorImportDialog.vue'
import CollectorLiveDebugPanel from './CollectorLiveDebugPanel.vue'
import CollectorPointGroupTree from './CollectorPointGroupTree.vue'
import CollectorPointTable from './CollectorPointTable.vue'
const props = defineProps<{ projectId: string; connection?: { id: string } }>()
const emit = defineEmits<{ back: [] }>()
const selectedId = ref(props.connection?.id || '')
const connection = ref<CollectorConnection | null>(null)
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
  connection.value
    ? agentSupportsOperation(
        selectedAgent.value,
        connection.value.driverId,
        connection.value.driverVersion,
        connection.value.schemaVersion,
        'device.browse',
      )
    : false,
)
const canRead = computed(() =>
  connection.value
    ? agentSupportsOperation(
        selectedAgent.value,
        connection.value.driverId,
        connection.value.driverVersion,
        connection.value.schemaVersion,
        'point.read',
      )
    : false,
)
async function selectConnection(id: string) {
  selectedId.value = id
  connection.value = await getCollectorConnection(props.projectId, id)
  activeTab.value = 'config'
  selectedPointIds.value = []
}
function onConnectionsLoaded(items: CollectorConnection[]) {
  if (!selectedId.value && items[0]) void selectConnection(items[0].id)
  else if (selectedId.value && !connection.value) void selectConnection(selectedId.value)
}
async function onCreated(id: string) {
  await connectionList.value?.reload(1)
  await selectConnection(id)
}
async function createDiscoveredPoints(points: Record<string, unknown>[]) {
  if (!connection.value) return
  await createCollectorPointsBatch(props.projectId, connection.value.id, points)
  activeTab.value = 'points'
  await pointTable.value?.reload(1)
}
</script>

<style scoped>
.industrial-workbench {
  display: flex;
  height: 100%;
  min-height: 0;
  background: #fff;
  color: #23313a;
}
.industrial-workbench__main {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  padding: 0 24px 20px;
}
.industrial-workbench__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px 0;
  border-bottom: 1px solid #dfe6ea;
}
.industrial-workbench__header div {
  display: grid;
  grid-template-columns: auto 1fr;
  column-gap: 14px;
  align-items: center;
}
.industrial-workbench__header button {
  grid-row: 1/3;
  border: 0;
  background: transparent;
  color: #39748d;
  cursor: pointer;
}
.industrial-workbench__header small {
  color: #8a969e;
  font-size: 10px;
  letter-spacing: 0.14em;
}
.industrial-workbench__header h2 {
  margin: 3px 0 0;
  font-size: 20px;
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
</style>
