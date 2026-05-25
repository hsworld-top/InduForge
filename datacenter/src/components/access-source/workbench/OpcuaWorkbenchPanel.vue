<template>
  <section class="opcua-workbench">
    <header class="opcua-workbench__header">
      <WorkbenchSourceHeader
        :title="connection.name || '未命名 OPC UA 接入源'"
        fallback-title="未命名 OPC UA 接入源"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #actions>
          <el-button size="small" :icon="IconTablerPlugConnected" :loading="testing" @click="runConnectionTest">
            测试连接
          </el-button>
          <button
            type="button"
            class="workbench-source-header__icon-action"
            title="从 OPC UA 导入变量"
            aria-label="从 OPC UA 导入变量"
            @click="importVisible = true"
          >
            <IconTablerUpload />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action is-primary"
            title="新建变量"
            aria-label="新建变量"
            @click="openCreateNode"
          >
            <IconTablerPlus />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action"
            title="变量预览"
            aria-label="变量预览"
            @click="openPreview"
          >
            <IconTablerActivityHeartbeat />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action"
            title="建模校验"
            aria-label="建模校验"
            @click="runValidation"
          >
            <IconTablerChecklist />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action"
            title="刷新"
            aria-label="刷新"
            @click="reloadAll"
          >
            <IconTablerRefresh />
          </button>
        </template>
      </WorkbenchSourceHeader>
    </header>

    <div class="opcua-workbench__body">
      <OpcuaGroupTree
        :groups="groups"
        :nodes="nodes"
        :selected-group-id="selectedGroupId"
        @select="selectGroup"
        @create="openCreateGroup"
        @edit="openEditGroup"
        @delete="removeGroup"
      />

      <main class="opcua-workbench__main">
        <div class="opcua-workbench__bar">
          <div class="opcua-workbench__title">
            <div>
              <strong>{{ currentGroup?.name || '全部变量' }}</strong>
              <span>{{ filteredNodes.length }} 个变量，保存后自动同步数据点</span>
            </div>
            <div class="opcua-workbench__metrics">
              <span>{{ activeNodeCount }} active</span>
              <span>{{ generatedDatapointCount }} dp</span>
              <span>{{ validationIssues.length }} issues</span>
            </div>
          </div>
          <el-input v-model="nodeKeyword" class="opcua-workbench__search" size="small" placeholder="搜索变量" clearable />
        </div>
        <OpcuaNodeTable
          :nodes="filteredNodes"
          :loading="loading"
          :selected-node-id="selectedNodeId"
          @select="selectNode"
          @edit="openEditNode"
          @delete="removeNode"
        />
      </main>

      <OpcuaInspectorPanel
        :group="currentGroup"
        :node="selectedNode"
        :nodes="nodes"
        :issues="validationIssues"
      />
    </div>

    <OpcuaGroupDialog
      v-model="groupDialogVisible"
      :mode="groupDialogMode"
      :groups="groups"
      :group-value="editingGroup"
      :loading="saving"
      @submit="saveGroup"
    />
    <OpcuaNodeDialog
      v-model="nodeDialogVisible"
      :mode="nodeDialogMode"
      :groups="groups"
      :node="editingNode"
      :default-group-id="selectedGroupId"
      :loading="saving"
      @submit="saveNode"
    />
    <OpcuaImportDialog v-model="importVisible" :loading="saving" @submit="importNodes" />
    <OpcuaPreviewDialog v-model="previewVisible" :nodes="previewNodes" :diagnostics="previewDiagnostics" />
    <OpcuaValidationDrawer
      v-model="validationVisible"
      :issues="validationIssues"
      @locate="locateNode"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import OpcuaGroupDialog from '@/components/opcua/OpcuaGroupDialog.vue'
import OpcuaGroupTree from '@/components/opcua/OpcuaGroupTree.vue'
import OpcuaImportDialog from '@/components/opcua/OpcuaImportDialog.vue'
import OpcuaInspectorPanel from '@/components/opcua/OpcuaInspectorPanel.vue'
import OpcuaNodeDialog from '@/components/opcua/OpcuaNodeDialog.vue'
import OpcuaNodeTable from '@/components/opcua/OpcuaNodeTable.vue'
import OpcuaPreviewDialog from '@/components/opcua/OpcuaPreviewDialog.vue'
import OpcuaValidationDrawer from '@/components/opcua/OpcuaValidationDrawer.vue'
import type { OpcuaNode, OpcuaNodeGroup, OpcuaValidationIssue } from '@/components/opcua/types'
import IconTablerActivityHeartbeat from '~icons/tabler/activity-heartbeat'
import IconTablerChecklist from '~icons/tabler/checklist'
import IconTablerPlugConnected from '~icons/tabler/plug-connected'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerUpload from '~icons/tabler/upload'

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

const groups = ref<OpcuaNodeGroup[]>([])
const nodes = ref<OpcuaNode[]>([])
const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const selectedGroupId = ref('')
const selectedNodeId = ref('')
const nodeKeyword = ref('')
const groupDialogVisible = ref(false)
const groupDialogMode = ref<'create' | 'edit'>('create')
const editingGroup = ref<OpcuaNodeGroup | null>(null)
const nodeDialogVisible = ref(false)
const nodeDialogMode = ref<'create' | 'edit'>('create')
const editingNode = ref<OpcuaNode | null>(null)
const importVisible = ref(false)
const previewVisible = ref(false)
const previewNodes = ref<OpcuaNode[]>([])
const previewDiagnostics = ref<string[]>([])
const validationVisible = ref(false)
const validationIssues = ref<OpcuaValidationIssue[]>([])

const config = computed(() => props.connection.config || {})
const endpointText = computed(() => String(config.value.endpoint || config.value.url || '未配置 endpoint'))
const sourceMetaRows = computed(() => [
  { label: '类型', value: 'OPC UA' },
  { label: 'Endpoint', value: endpointText.value },
])
const currentGroup = computed(() => groups.value.find((group) => group.id === selectedGroupId.value) || null)
const selectedNode = computed(() => nodes.value.find((node) => node.id === selectedNodeId.value) || null)
const activeNodeCount = computed(() => nodes.value.filter((node) => node.status === 'active').length)
const generatedDatapointCount = computed(() => nodes.value.filter((node) => node.datapointPath).length)
const filteredNodes = computed(() => {
  const text = nodeKeyword.value.trim().toLowerCase()
  return nodes.value.filter((node) => {
    const inGroup = !selectedGroupId.value || node.groupId === selectedGroupId.value
    const matched =
      !text ||
      [node.name, node.nodeId, node.code, node.datapointPath || ''].some((value) =>
        String(value || '').toLowerCase().includes(text),
      )
    return inGroup && matched
  })
})

const unwrapList = <T,>(response: any): T[] => response?.data?.list || response?.data?.data?.list || []

const reloadAll = async () => {
  loading.value = true
  try {
    const [groupResponse, nodeResponse] = await Promise.all([
      dataAPI.getOpcuaNodeGroups(props.projectId, props.connection.id),
      dataAPI.getOpcuaNodes(props.projectId, props.connection.id),
    ])
    groups.value = unwrapList<OpcuaNodeGroup>(groupResponse)
    nodes.value = unwrapList<OpcuaNode>(nodeResponse)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载 OPC UA 建模数据失败'))
  } finally {
    loading.value = false
  }
}

const selectGroup = (groupId: string) => {
  selectedGroupId.value = groupId
  selectedNodeId.value = ''
}

const selectNode = (node: OpcuaNode) => {
  selectedNodeId.value = node.id
}

const openCreateGroup = () => {
  editingGroup.value = null
  groupDialogMode.value = 'create'
  groupDialogVisible.value = true
}

const openEditGroup = (group: OpcuaNodeGroup) => {
  editingGroup.value = group
  groupDialogMode.value = 'edit'
  groupDialogVisible.value = true
}

const saveGroup = async (payload: Record<string, unknown>) => {
  saving.value = true
  try {
    if (groupDialogMode.value === 'edit' && editingGroup.value) {
      await dataAPI.updateOpcuaNodeGroup(props.projectId, props.connection.id, editingGroup.value.id, {
        ...payload,
        hasParentId: true,
      })
    } else {
      await dataAPI.createOpcuaNodeGroup(props.projectId, props.connection.id, payload)
    }
    groupDialogVisible.value = false
    await reloadAll()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存变量组失败'))
  } finally {
    saving.value = false
  }
}

const removeGroup = async (group: OpcuaNodeGroup) => {
  await ElMessageBox.confirm(`删除变量组“${group.name}”？组内变量会移动到未分组。`, '删除变量组')
  await dataAPI.deleteOpcuaNodeGroup(props.projectId, props.connection.id, group.id)
  if (selectedGroupId.value === group.id) selectedGroupId.value = ''
  await reloadAll()
}

const openCreateNode = () => {
  editingNode.value = null
  nodeDialogMode.value = 'create'
  nodeDialogVisible.value = true
}

const openEditNode = (node: OpcuaNode) => {
  editingNode.value = node
  nodeDialogMode.value = 'edit'
  nodeDialogVisible.value = true
}

const saveNode = async (payload: Record<string, unknown>) => {
  saving.value = true
  try {
    const saved =
      nodeDialogMode.value === 'edit' && editingNode.value
        ? await dataAPI.updateOpcuaNode(props.projectId, props.connection.id, editingNode.value.id, payload)
        : await dataAPI.createOpcuaNode(props.projectId, props.connection.id, payload)
    nodeDialogVisible.value = false
    await reloadAll()
    selectedNodeId.value = saved?.data?.id || saved?.data?.data?.id || ''
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存变量失败'))
  } finally {
    saving.value = false
  }
}

const removeNode = async (node: OpcuaNode) => {
  await ElMessageBox.confirm(`删除变量“${node.name}”？关联数据点会标记为失效。`, '删除变量')
  await dataAPI.deleteOpcuaNode(props.projectId, props.connection.id, node.id)
  if (selectedNodeId.value === node.id) selectedNodeId.value = ''
  await reloadAll()
}

const importNodes = async (rows: Array<Record<string, unknown>>) => {
  saving.value = true
  try {
    await dataAPI.batchImportOpcuaNodes(props.projectId, props.connection.id, {
      groupId: selectedGroupId.value || null,
      nodes: rows,
    })
    importVisible.value = false
    await reloadAll()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '导入变量失败'))
  } finally {
    saving.value = false
  }
}

const runConnectionTest = async () => {
  testing.value = true
  try {
    await dataAPI.testConnection(props.projectId, {
      type: 'opcua',
      name: props.connection.name,
      config: props.connection.config,
    })
    ElMessage.success('测试请求已完成')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '测试连接失败'))
  } finally {
    testing.value = false
  }
}

const openPreview = async () => {
  try {
    const response = await dataAPI.previewOpcuaNodes(props.projectId, props.connection.id, {
      groupId: selectedGroupId.value || null,
    })
    const data = response?.data?.data || response?.data || {}
    previewNodes.value = data.values || []
    previewDiagnostics.value = data.diagnostics || []
    previewVisible.value = true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '变量预览失败'))
  }
}

const runValidation = async () => {
  try {
    const response = await dataAPI.validateOpcuaModel(props.projectId, props.connection.id)
    const data = response?.data?.data || response?.data || {}
    validationIssues.value = data.issues || []
    validationVisible.value = true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '建模校验失败'))
  }
}

const locateNode = (nodeId: string) => {
  selectedNodeId.value = nodeId
  const node = nodes.value.find((item) => item.id === nodeId)
  selectedGroupId.value = node?.groupId || ''
  validationVisible.value = false
}

onMounted(reloadAll)
</script>

<style scoped>
.opcua-workbench {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
  overflow: hidden;
}

.opcua-workbench__header {
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.opcua-workbench__body {
  min-height: 0;
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr) 286px;
}

.opcua-workbench__main {
  min-width: 0;
  min-height: 0;
  background: var(--dc-surface-raised);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.opcua-workbench__bar {
  min-height: 46px;
  padding: 7px 10px 7px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
  display: grid;
  grid-template-columns: minmax(0, 1fr) 230px;
  align-items: center;
  gap: 12px;
}

.opcua-workbench__title {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 12px;
}

.opcua-workbench__title > div:first-child {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.opcua-workbench__bar strong {
  color: var(--dc-text);
  font-size: 14px;
  line-height: 18px;
}

.opcua-workbench__bar span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.opcua-workbench__metrics {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.opcua-workbench__metrics span {
  height: 22px;
  display: inline-flex;
  align-items: center;
  padding: 0 7px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font-size: 11px;
  font-weight: 700;
}

.opcua-workbench__search {
  min-width: 0;
}

@media (max-width: 1100px) {
  .opcua-workbench__body {
    grid-template-columns: 230px minmax(0, 1fr);
  }

  .opcua-workbench__body :deep(.opcua-inspector) {
    display: none;
  }
}
</style>
