<template>
  <section class="opcua-workbench">
    <aside class="opcua-workbench__side">
      <WorkbenchSourceHeader
        :title="connection.name || '未命名 OPC UA 接入源'"
        fallback-title="未命名 OPC UA 接入源"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #status>
          <button
            type="button"
            class="opcua-workbench__connect-action"
            :class="{ 'is-connected': session.connected.value }"
            :title="session.connected.value ? '断开 OPC UA 开发态会话' : '连接 OPC UA 开发态会话'"
            @click="toggleSession"
          >
            <span class="opcua-workbench__connect-label is-default">
              {{ session.connected.value ? '已连接' : '连接' }}
            </span>
            <span v-if="session.connected.value" class="opcua-workbench__connect-label is-hover">断开</span>
          </button>
        </template>
      </WorkbenchSourceHeader>
      <OpcuaGroupTree
        :groups="groups"
        :nodes="nodes"
        :selected-group-id="selectedGroupId"
        @select="selectGroup"
        @create="openCreateGroup"
        @edit="openEditGroup"
        @delete="removeGroup"
      />
    </aside>

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
          <div class="opcua-workbench__actions">
            <button
              type="button"
              class="opcua-workbench__icon-action"
              :title="importActionTitle"
              :aria-label="importActionTitle"
              @click="importVisible = true"
            >
              <IconTablerUpload />
            </button>
            <button
              type="button"
              class="opcua-workbench__icon-action is-primary"
              title="新建变量"
              aria-label="新建变量"
              @click="openCreateNode"
            >
              <IconTablerPlus />
            </button>
            <button
              type="button"
              class="opcua-workbench__icon-action"
              :disabled="!session.connected.value"
              :title="previewActionTitle"
              :aria-label="previewActionTitle"
              @click="openPreview"
            >
              <IconTablerActivityHeartbeat />
            </button>
            <button
              type="button"
              class="opcua-workbench__icon-action"
              :title="validationActionTitle"
              :aria-label="validationActionTitle"
              @click="runValidation"
            >
              <IconTablerChecklist />
            </button>
            <button
              type="button"
              class="opcua-workbench__icon-action"
              title="刷新"
              aria-label="刷新"
              @click="reloadAll"
            >
              <IconTablerRefresh />
            </button>
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
        :issues="scopedValidationIssues"
      />

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
      :issues="scopedValidationIssues"
      :scope-label="validationScopeLabel"
      @locate="locateNode"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'
import { useProtocolDevSession } from './useProtocolDevSession'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import OpcuaGroupDialog from '@/components/opcua/OpcuaGroupDialog.vue'
import OpcuaGroupTree from '@/components/opcua/OpcuaGroupTree.vue'
import OpcuaImportDialog from '@/components/opcua/OpcuaImportDialog.vue'
import OpcuaInspectorPanel from '@/components/opcua/OpcuaInspectorPanel.vue'
import OpcuaNodeDialog from '@/components/opcua/OpcuaNodeDialog.vue'
import OpcuaNodeTable from '@/components/opcua/OpcuaNodeTable.vue'
import OpcuaPreviewDialog from '@/components/opcua/OpcuaPreviewDialog.vue'
import OpcuaValidationDrawer from '@/components/opcua/OpcuaValidationDrawer.vue'
import type { OpcuaNode, OpcuaNodeGroup, OpcuaReadValue, OpcuaValidationIssue } from '@/components/opcua/types'
import IconTablerActivityHeartbeat from '~icons/tabler/activity-heartbeat'
import IconTablerChecklist from '~icons/tabler/checklist'
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
const previewNodes = ref<OpcuaReadValue[]>([])
const previewDiagnostics = ref<string[]>([])
const validationVisible = ref(false)
const validationIssues = ref<OpcuaValidationIssue[]>([])

const session = useProtocolDevSession({
  create: () => dataAPI.createOpcuaDevSession(props.projectId, props.connection.id),
  close: (sessionId: string) => dataAPI.closeOpcuaDevSession(props.projectId, props.connection.id, sessionId),
})

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
const importActionTitle = computed(() =>
  currentGroup.value ? `导入到「${currentGroup.value.name}」` : '导入到未分组',
)
const previewActionTitle = computed(() =>
  session.connected.value
    ? currentGroup.value
      ? `预览「${currentGroup.value.name}」变量`
      : '预览全部变量'
    : '连接后可预览当前分组变量',
)
const validationActionTitle = computed(() =>
  selectedNode.value
    ? `校验当前变量：${selectedNode.value.name}`
    : currentGroup.value
      ? `校验当前分组：${currentGroup.value.name}`
      : '校验全部变量',
)
const validationScopeLabel = computed(() => {
  if (selectedNode.value) return `当前变量：${selectedNode.value.name}`
  if (currentGroup.value) return `当前分组：${currentGroup.value.name}`
  return '全部变量'
})
const scopedValidationIssues = computed(() => {
  if (selectedNodeId.value) {
    return validationIssues.value.filter((item) => item.nodeId === selectedNodeId.value)
  }
  if (selectedGroupId.value) {
    const ids = new Set(nodes.value.filter((node) => node.groupId === selectedGroupId.value).map((node) => node.id))
    return validationIssues.value.filter((item) => item.nodeId && ids.has(item.nodeId))
  }
  return validationIssues.value
})
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
const unwrapData = (response: any) => response?.data?.data || response?.data || {}

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

const toggleSession = () => {
  if (session.connected.value) void session.disconnect()
  else void session.connect()
}

const openPreview = async () => {
  if (!session.connected.value || !session.sessionId.value) {
    ElMessage.warning('请先连接 OPC UA 开发态会话')
    return
  }
  try {
    const response = await dataAPI.subscribeOpcuaDevSession(props.projectId, props.connection.id, session.sessionId.value, {
      groupId: selectedGroupId.value || null,
    })
    const data = unwrapData(response)
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
    const data = unwrapData(response)
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
  grid-template-columns: 260px minmax(0, 1fr) 286px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
  overflow: hidden;
}

.opcua-workbench__side {
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--dc-border);
  overflow: hidden;
  background: var(--dc-surface-muted);
}

.opcua-workbench__side :deep(.opcua-groups) {
  flex: 1;
  min-height: 0;
}

.opcua-workbench__connect-action {
  min-width: 58px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 0 9px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 32%, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-size: 11px;
  font-weight: 700;
  white-space: nowrap;
}

.opcua-workbench__connect-label.is-hover {
  display: none;
}

.opcua-workbench__connect-action.is-connected:hover .opcua-workbench__connect-label.is-default {
  display: none;
}

.opcua-workbench__connect-action.is-connected:hover .opcua-workbench__connect-label.is-hover {
  display: inline;
}

.opcua-workbench__connect-action::before {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: currentColor;
  content: '';
}

.opcua-workbench__connect-action.is-connected {
  border-color: color-mix(in oklch, var(--dc-success) 32%, var(--dc-border));
  background: color-mix(in oklch, var(--dc-success) 12%, var(--dc-surface-raised));
  color: var(--dc-success);
}

.opcua-workbench__connect-action.is-connected:hover {
  border-color: rgba(220, 38, 38, 0.22);
  background: rgba(220, 38, 38, 0.08);
  color: #b91c1c;
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
  grid-template-columns: minmax(0, 1fr) auto 230px;
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

.opcua-workbench__actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.opcua-workbench__icon-action {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.opcua-workbench__icon-action:hover:not(:disabled) {
  color: var(--dc-primary);
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
}

.opcua-workbench__icon-action:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.opcua-workbench__icon-action.is-primary {
  border-color: var(--dc-primary);
  background: var(--dc-primary);
  color: var(--dc-surface-raised);
}

.opcua-workbench__icon-action svg {
  width: 15px;
  height: 15px;
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
  .opcua-workbench {
    grid-template-columns: 230px minmax(0, 1fr);
  }

  .opcua-workbench :deep(.opcua-inspector) {
    display: none;
  }
}
</style>
