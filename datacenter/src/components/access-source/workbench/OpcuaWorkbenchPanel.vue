<template>
  <section class="opcua-workbench">
    <aside class="opcua-workbench__side">
      <WorkbenchSourceHeader
        :title="localConnection.name || '未命名 OPC UA 接入源'"
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
            <span v-if="session.connected.value" class="opcua-workbench__connect-label is-hover"
              >断开</span
            >
          </button>
        </template>
      </WorkbenchSourceHeader>
      <OpcuaGroupTree
        :groups="groups"
        :selected-group-id="selectedGroupId"
        :total="nodePagination.total"
        @select="selectGroup"
        @create="openCreateGroup"
        @create-child="openCreateChildGroup"
        @edit="openEditGroup"
        @delete="removeGroup"
      />
    </aside>

    <main class="opcua-workbench__main">
      <div class="opcua-workbench__bar">
        <div class="opcua-workbench__actions">
          <button
            type="button"
            class="opcua-workbench__icon-action"
            :title="importActionTitle"
            :aria-label="importActionTitle"
            @click="openImportDialog"
          >
            <IconTablerUpload />
          </button>
          <el-dropdown
            trigger="click"
            :disabled="nodes.length === 0"
            @command="exportNodes"
          >
            <button
              type="button"
              class="opcua-workbench__icon-action"
              :disabled="nodes.length === 0"
              title="导出当前筛选结果"
              aria-label="导出当前筛选结果"
            >
              <IconTablerDownload />
            </button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="csv">导出 CSV</el-dropdown-item>
                <el-dropdown-item command="xlsx">导出 XLSX</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
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
            title="运行契约预览"
            aria-label="运行契约预览"
            @click="contractVisible = true"
          >
            <IconTablerFileDescription />
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
          <button
            type="button"
            class="opcua-workbench__icon-action"
            title="连接与会话配置"
            aria-label="连接与会话配置"
            @click="showEditDialog = true"
          >
            <IconTablerSettings />
          </button>
        </div>
        <el-input
          v-model="nodeKeyword"
          class="opcua-workbench__search"
          size="small"
          placeholder="搜索变量"
          clearable
        />
      </div>
      <OpcuaNodeTable
        :nodes="nodes"
        :loading="loading"
        :selected-node-id="selectedNodeId"
        :page="nodePagination.page"
        :page-size="nodePagination.pageSize"
        :total="nodePagination.total"
        @select="selectNode"
        @edit="openEditNode"
        @delete="removeNode"
        @row-contextmenu="openNodeMenu"
        @page-change="changeNodePage"
        @page-size-change="changeNodePageSize"
        @sort-change="changeNodeSort"
      />
      <div class="opcua-workbench__filters">
        <button
          v-for="item in quickFilters"
          :key="item.value"
          type="button"
          :class="{ 'is-active': quickFilter === item.value }"
          @click="quickFilter = item.value"
        >
          {{ item.label }}
        </button>
      </div>
    </main>

    <OpcuaInspectorPanel
      :group="currentGroup"
      :node="selectedNode"
      :nodes="nodes"
      :issues="scopedValidationIssues"
      :connection="localConnection"
      :project-id="projectId"
    />

    <OpcuaGroupDialog
      v-model="groupDialogVisible"
      :mode="groupDialogMode"
      :groups="groups"
      :group-value="editingGroup"
      :default-parent-id="defaultGroupParentId"
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
    <OpcuaImportDialog
      v-model="importVisible"
      :loading="saving"
      :connected="session.connected.value"
      :browse-loading="browsing"
      :browse-nodes="browseNodes"
      :browse-diagnostics="browseDiagnostics"
      @browse="loadBrowseNodes"
      @submit="importNodes"
    />
    <OpcuaPreviewDialog
      v-model="previewVisible"
      :nodes="previewNodes"
      :diagnostics="previewDiagnostics"
    />
    <OpcuaValidationDrawer
      v-model="validationVisible"
      :issues="scopedValidationIssues"
      :scope-label="validationScopeLabel"
      @locate="locateNode"
    />
    <ProtocolContractDrawer
      v-model="contractVisible"
      title="OPC UA 运行契约预览"
      :subtitle="contractSubtitle"
      :sections="contractSections"
      :issues="contractIssues"
    />
    <ConnectionDialog
      v-model="showEditDialog"
      mode="edit"
      :connection="localConnection"
      :project-id="projectId"
      @submit="handleEditConnectionSubmit"
    />
    <Teleport to="body">
      <div
        v-if="nodeMenu.visible"
        class="opcua-workbench__menu-mask"
        @click="closeNodeMenu"
        @contextmenu.prevent="closeNodeMenu"
      >
        <div
          class="opcua-workbench__context-menu"
          :style="{ left: `${nodeMenu.x}px`, top: `${nodeMenu.y}px` }"
          @click.stop
        >
          <button type="button" @click="runNodeMenuAction('edit')">
            <IconTablerPencil class="opcua-workbench__menu-icon" />
            <span>编辑</span>
          </button>
          <button type="button" @click="runNodeMenuAction('duplicate')">
            <IconTablerCopy class="opcua-workbench__menu-icon" />
            <span>复制为新变量</span>
          </button>
          <button type="button" @click="runNodeMenuAction('copy-info')">
            <IconTablerClipboard class="opcua-workbench__menu-icon" />
            <span>复制变量信息</span>
          </button>
          <button type="button" @click="runNodeMenuAction('copy-path')">
            <IconTablerRoute class="opcua-workbench__menu-icon" />
            <span>复制数据点路径</span>
          </button>
          <button type="button" @click="runNodeMenuAction('datapoint')">
            <IconTablerExternalLink class="opcua-workbench__menu-icon" />
            <span>查看数据点详情</span>
          </button>
          <button type="button" class="is-danger" @click="runNodeMenuAction('delete')">
            <IconTablerTrash class="opcua-workbench__menu-icon" />
            <span>删除变量</span>
          </button>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import dataAPI from '@/api/data.api'
import { listStoragePolicies, type StoragePolicySummary } from '@/api/storage-policy.api'
import { getApiErrorMessage } from '@/utils/request'
import { downloadCsv, downloadXlsx } from '@/utils/tabular-file'
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
import ProtocolContractDrawer from './ProtocolContractDrawer.vue'
import type {
  OpcuaBrowseNode,
  OpcuaNode,
  OpcuaNodeGroup,
  OpcuaReadValue,
  OpcuaValidationIssue,
} from '@/components/opcua/types'
import IconTablerActivityHeartbeat from '~icons/tabler/activity-heartbeat'
import IconTablerChecklist from '~icons/tabler/checklist'
import IconTablerClipboard from '~icons/tabler/clipboard'
import IconTablerCopy from '~icons/tabler/copy'
import IconTablerDownload from '~icons/tabler/download'
import IconTablerExternalLink from '~icons/tabler/external-link'
import IconTablerFileDescription from '~icons/tabler/file-description'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerRoute from '~icons/tabler/route'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerUpload from '~icons/tabler/upload'
import IconTablerSettings from '~icons/tabler/settings'
import ConnectionDialog from '@/components/dialogs/ConnectionDialog.vue'

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

const emit = defineEmits<{
  (event: 'back'): void
  (event: 'update-connection'): void
}>()

const localConnection = ref<AccessSourceConnection>({ ...props.connection })

watch(
  () => props.connection,
  (newVal) => {
    if (newVal) {
      localConnection.value = { ...newVal }
    }
  },
  { deep: true },
)

const showEditDialog = ref(false)
const isSavingConnection = ref(false)

const handleEditConnectionSubmit = async (data: any) => {
  isSavingConnection.value = true
  try {
    await dataAPI.updateConnection(props.projectId, props.connection.id, {
      name: data.name,
      type: data.type,
      config: data.config,
    })
    ElMessage.success('连接配置更新成功')
    showEditDialog.value = false

    // 原地更新本地状态，实现 UI 的即时零延迟刷新
    const buildOpcuaEndpoint = (ip: any, port: any) => {
      const safeIp = String(ip || '').trim()
      const safePort = Number(port) || 4840
      return `opc.tcp://${safeIp}:${safePort}`
    }
    localConnection.value = {
      ...localConnection.value,
      name: data.name,
      config: {
        ...localConnection.value.config,
        ...data.config,
        endpoint: buildOpcuaEndpoint(data.config.ip, data.config.port),
      },
    }

    emit('update-connection')

    if (session.connected.value) {
      ElMessage.info('检测到当前开发态会话已连接，正在自动重连...')
      try {
        await session.disconnect()
      } catch (err) {
        console.error('断开旧会话失败:', err)
      }
      setTimeout(async () => {
        try {
          await session.connect()
          ElMessage.success('自动重连成功')
        } catch (err) {
          ElMessage.error(getApiErrorMessage(err, '自动重连失败，请手动重新连接'))
        }
      }, 500)
    }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '更新连接配置失败'))
  } finally {
    isSavingConnection.value = false
  }
}

const groups = ref<OpcuaNodeGroup[]>([])
const nodes = ref<OpcuaNode[]>([])
const nodePagination = ref({ page: 1, pageSize: 20, total: 0, totalPages: 0 })
const loading = ref(false)
const saving = ref(false)
const selectedGroupId = ref('')
const selectedNodeId = ref('')
const nodeKeyword = ref('')
const groupDialogVisible = ref(false)
const groupDialogMode = ref<'create' | 'edit'>('create')
const editingGroup = ref<OpcuaNodeGroup | null>(null)
const defaultGroupParentId = ref('')
const nodeDialogVisible = ref(false)
const nodeDialogMode = ref<'create' | 'edit'>('create')
const editingNode = ref<OpcuaNode | null>(null)
const importVisible = ref(false)
const browsing = ref(false)
const browseNodes = ref<OpcuaBrowseNode[]>([])
const browseDiagnostics = ref<string[]>([])
const previewVisible = ref(false)
const previewNodes = ref<OpcuaReadValue[]>([])
const previewDiagnostics = ref<string[]>([])
const validationVisible = ref(false)
const validationIssues = ref<OpcuaValidationIssue[]>([])
const contractVisible = ref(false)
const quickFilter = ref('all')
const storageSummary = ref<StoragePolicySummary | null>(null)
const nodeSort = ref<{ sortBy?: string; sortOrder?: string }>({})
let searchTimer: ReturnType<typeof window.setTimeout> | undefined
const nodeMenu = ref({
  visible: false,
  x: 0,
  y: 0,
  node: null as OpcuaNode | null,
})
const quickFilters = [
  { label: '全部', value: 'all' },
  { label: '异常', value: 'issue' },
  { label: '未同步', value: 'datapoint' },
  { label: '可写', value: 'writable' },
  { label: '已停用', value: 'disabled' },
]

const session = useProtocolDevSession({
  create: () => dataAPI.createOpcuaDevSession(props.projectId, props.connection.id),
  close: (sessionId: string) =>
    dataAPI.closeOpcuaDevSession(props.projectId, props.connection.id, sessionId),
})

const config = computed(() => localConnection.value.config || {})
const endpointText = computed(() =>
  String(config.value.endpoint || config.value.url || '未配置 endpoint'),
)
const sourceMetaRows = computed(() => [
  { label: '类型', value: 'OPC UA' },
  { label: 'Endpoint', value: endpointText.value },
])
const currentGroup = computed(
  () => groups.value.find((group) => group.id === selectedGroupId.value) || null,
)
const selectedNode = computed(
  () => nodes.value.find((node) => node.id === selectedNodeId.value) || null,
)
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
    const ids = new Set(
      nodes.value.filter((node) => node.groupId === selectedGroupId.value).map((node) => node.id),
    )
    return validationIssues.value.filter((item) => item.nodeId && ids.has(item.nodeId))
  }
  return validationIssues.value
})
const nodeExportHeaders = [
  '变量名',
  'Code',
  '分组',
  '协议地址 / NodeId',
  '数据类型',
  '采集周期',
  '发布能力',
  '数据点 path',
  '状态',
  '最近开发态读取值',
  '质量',
  '诊断问题摘要',
]
const issueExportHeaders = [
  '严重级别',
  '问题类型',
  '变量名',
  'Code',
  '分组',
  '定位字段',
  '问题说明',
  '建议处理',
]
const buildNodeExportRows = (items: OpcuaNode[]) =>
  items.map((node) => ({
    变量名: node.name,
    Code: node.code,
    分组: groupPathOf(node.groupId),
    '协议地址 / NodeId': node.nodeId,
    数据类型: node.dataType,
    采集周期: `${node.samplingMs}ms`,
    发布能力: node.accessLevel || 'read',
    '数据点 path': node.datapointPath || '',
    状态: node.status || '',
    最近开发态读取值: formatExportValue(node.lastValue),
    质量: node.quality || '',
    诊断问题摘要: issueSummaryForNode(node.id),
  }))
const nodeExportRows = computed(() => buildNodeExportRows(nodes.value))
const issueExportRows = computed(() =>
  buildIssueExportRows(nodes.value, scopedValidationIssues.value),
)
const buildIssueExportRows = (items: OpcuaNode[], issues: OpcuaValidationIssue[]) => {
  const nodeByID = new Map(items.map((item) => [item.id, item]))
  return issues
    .filter((issue) => !issue.nodeId || nodeByID.has(issue.nodeId))
    .map((issue) => {
      const node = issue.nodeId ? nodeByID.get(issue.nodeId) : null
      return {
        严重级别: issue.severity,
        问题类型: issue.code,
        变量名: issue.nodeName || node?.name || '',
        Code: node?.code || '',
        分组: groupPathOf(issue.groupId || node?.groupId),
        定位字段: issue.nodeId ? '变量' : issue.groupId ? '分组' : '连接',
        问题说明: issue.message,
        建议处理: '按诊断提示修正建模后重新校验',
      }
    })
}
const contractSubtitle = computed(() =>
  currentGroup.value
    ? `当前范围：${currentGroup.value.name}`
    : selectedNode.value
      ? `当前变量：${selectedNode.value.name}`
      : '当前范围：全部变量',
)
const contractIssues = computed(() =>
  scopedValidationIssues.value.map((issue) => ({
    severity: issue.severity,
    message: issue.message,
  })),
)
const contractSections = computed(() => [
  {
    title: '协议建模',
    rows: [
      { label: 'Endpoint', value: endpointText.value },
      { label: '变量数', value: nodePagination.value.total },
      { label: '当前页启用', value: nodes.value.filter((node) => node.status === 'active').length },
      { label: '发布能力', value: '按服务器能力收窄，不提供工作台写值' },
    ],
  },
  {
    title: '订阅策略',
    rows: [
      { label: '默认采样', value: samplingSummary.value },
      { label: '开发态位置', value: '平台侧 data_service' },
      { label: '运行态位置', value: '发布后的采集节点 / 数据引擎' },
      { label: '预览语义', value: '短时读取快照，不写历史、不触发报警计算' },
    ],
  },
  {
    title: '冗余与存储',
    rows: [
      { label: '当前值', value: 'IF 实时库', tone: 'ok' as const },
      { label: '历史归档', value: storageSummaryText.value },
      { label: '异常策略', value: storageSummary.value?.errorCount ?? 0 },
      { label: '设备冗余', value: deviceRedundancyText.value },
      { label: '采集冗余', value: '运行部署策略统一配置' },
    ],
    notes: ['同一 connection 同一时刻只有一个采集 Owner；设备 endpoint 故障时由该 Owner 按主备策略切换。'],
  },
])

const unwrapList = <T,>(response: any): T[] =>
  response?.data?.list || response?.data?.data?.list || []
const unwrapData = (response: any) => response?.data?.data || response?.data || {}
const unwrapPagination = (response: any) =>
  response?.data?.pagination || response?.data?.data?.pagination

const reloadAll = async () => {
  loading.value = true
  try {
    const [groupResponse, nodeResponse] = await Promise.all([
      dataAPI.getOpcuaNodeGroups(props.projectId, props.connection.id),
      dataAPI.getOpcuaNodes(props.projectId, props.connection.id, {
        groupId: selectedGroupId.value || undefined,
        q: nodeKeyword.value.trim() || undefined,
        filter: quickFilter.value === 'all' ? undefined : quickFilter.value,
        sortBy: nodeSort.value.sortBy,
        sortOrder: nodeSort.value.sortOrder,
        page: nodePagination.value.page,
        pageSize: nodePagination.value.pageSize,
      }),
    ])
    groups.value = unwrapList<OpcuaNodeGroup>(groupResponse)
    nodes.value = unwrapList<OpcuaNode>(nodeResponse)
    nodePagination.value = { ...nodePagination.value, ...unwrapPagination(nodeResponse) }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载 OPC UA 建模数据失败'))
  } finally {
    loading.value = false
  }
}

const reloadStorageSummary = async () => {
  try {
    const result = await listStoragePolicies(props.projectId, { page: 1, pageSize: 1 })
    storageSummary.value = result.summary || null
  } catch {
    storageSummary.value = null
  }
}

const loadExportNodes = async () => {
  const pageSize = 100
  const firstResponse = await dataAPI.getOpcuaNodes(props.projectId, props.connection.id, {
    groupId: selectedGroupId.value || undefined,
    q: nodeKeyword.value.trim() || undefined,
    filter: quickFilter.value === 'all' ? undefined : quickFilter.value,
    sortBy: nodeSort.value.sortBy,
    sortOrder: nodeSort.value.sortOrder,
    page: 1,
    pageSize,
  })
  const firstPage = unwrapList<OpcuaNode>(firstResponse)
  const pagination = unwrapPagination(firstResponse) || {}
  const totalPages = Number(pagination.totalPages || Math.ceil(Number(pagination.total || firstPage.length) / pageSize) || 1)
  const result = [...firstPage]
  for (let page = 2; page <= totalPages; page += 1) {
    const response = await dataAPI.getOpcuaNodes(props.projectId, props.connection.id, {
      groupId: selectedGroupId.value || undefined,
      q: nodeKeyword.value.trim() || undefined,
      filter: quickFilter.value === 'all' ? undefined : quickFilter.value,
      sortBy: nodeSort.value.sortBy,
      sortOrder: nodeSort.value.sortOrder,
      page,
      pageSize,
    })
    result.push(...unwrapList<OpcuaNode>(response))
  }
  return result
}

const openImportDialog = async () => {
  importVisible.value = true
  if (session.connected.value && browseNodes.value.length === 0) {
    await loadBrowseNodes()
  }
}

const loadBrowseNodes = async () => {
  if (!session.connected.value || !session.sessionId.value) {
    ElMessage.warning('请先连接 OPC UA 开发态会话')
    return
  }
  browsing.value = true
  try {
    const response = await dataAPI.browseOpcuaDevSession(
      props.projectId,
      props.connection.id,
      session.sessionId.value,
    )
    const data = unwrapData(response)
    browseNodes.value = data.nodes || []
    browseDiagnostics.value = data.diagnostics || []
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '浏览 OPC UA 节点失败'))
  } finally {
    browsing.value = false
  }
}

const selectGroup = async (groupId: string) => {
  selectedGroupId.value = groupId
  selectedNodeId.value = ''
  nodePagination.value.page = 1
  await reloadAll()
}

const changeNodePage = async (page: number) => {
  nodePagination.value.page = page
  await reloadAll()
}

const changeNodePageSize = async (pageSize: number) => {
  nodePagination.value.page = 1
  nodePagination.value.pageSize = pageSize
  await reloadAll()
}

const changeNodeSort = async (payload: { prop?: string; order?: string | null }) => {
  nodeSort.value = {
    sortBy: payload.prop || undefined,
    sortOrder: payload.order === 'descending' ? 'desc' : payload.order === 'ascending' ? 'asc' : undefined,
  }
  nodePagination.value.page = 1
  await reloadAll()
}

const selectNode = (node: OpcuaNode) => {
  selectedNodeId.value = node.id
}

const openCreateGroup = () => {
  editingGroup.value = null
  defaultGroupParentId.value = ''
  groupDialogMode.value = 'create'
  groupDialogVisible.value = true
}

const openCreateChildGroup = (group: OpcuaNodeGroup | null) => {
  editingGroup.value = null
  defaultGroupParentId.value = group?.id || ''
  groupDialogMode.value = 'create'
  groupDialogVisible.value = true
}

const openEditGroup = (group: OpcuaNodeGroup) => {
  editingGroup.value = group
  defaultGroupParentId.value = ''
  groupDialogMode.value = 'edit'
  groupDialogVisible.value = true
}

const saveGroup = async (payload: Record<string, unknown>) => {
  saving.value = true
  try {
    if (groupDialogMode.value === 'edit' && editingGroup.value) {
      await dataAPI.updateOpcuaNodeGroup(
        props.projectId,
        props.connection.id,
        editingGroup.value.id,
        {
          ...payload,
          hasParentId: true,
        },
      )
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

const openDuplicateNode = (node: OpcuaNode) => {
  editingNode.value = {
    ...node,
    id: '',
    name: `${node.name} 副本`,
    code: `${node.code}_copy`,
    datapointId: null,
    datapointPath: null,
    datapointStatus: null,
    lastValue: undefined,
    quality: undefined,
    lastUpdatedAt: null,
  }
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
        ? await dataAPI.updateOpcuaNode(
            props.projectId,
            props.connection.id,
            editingNode.value.id,
            payload,
          )
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

const exportNodes = async (format: string | number | object) => {
  loading.value = true
  try {
    const exportItems = await loadExportNodes()
    const rows = buildNodeExportRows(exportItems)
    if (rows.length === 0) {
      ElMessage.warning('当前筛选结果没有可导出的变量')
      return
    }
    const suffix = format === 'xlsx' ? 'xlsx' : 'csv'
    const filename = `opcua-variables-filtered.${suffix}`
    if (suffix === 'xlsx') {
      downloadXlsx(filename, [
        { name: '变量清单', headers: nodeExportHeaders, rows },
        { name: '问题清单', headers: issueExportHeaders, rows: buildIssueExportRows(exportItems, validationIssues.value) },
      ])
      return
    }
    downloadCsv(filename, nodeExportHeaders, rows)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '导出变量失败'))
  } finally {
    loading.value = false
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
    const response = await dataAPI.subscribeOpcuaDevSession(
      props.projectId,
      props.connection.id,
      session.sessionId.value,
      {
        groupId: selectedGroupId.value || null,
      },
    )
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

const openNodeMenu = (event: MouseEvent, node: OpcuaNode) => {
  event.preventDefault()
  selectedNodeId.value = node.id
  nodeMenu.value = {
    visible: true,
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 210),
    node,
  }
}

const closeNodeMenu = () => {
  nodeMenu.value.visible = false
}

const runNodeMenuAction = async (
  action: 'edit' | 'duplicate' | 'copy-info' | 'copy-path' | 'datapoint' | 'delete',
) => {
  const node = nodeMenu.value.node
  closeNodeMenu()
  if (!node) return
  if (action === 'edit') {
    openEditNode(node)
    return
  }
  if (action === 'duplicate') {
    openDuplicateNode(node)
    return
  }
  if (action === 'copy-info') {
    await copyText(
      [
        `name=${node.name}`,
        `code=${node.code}`,
        `nodeId=${node.nodeId}`,
        `browseName=${node.browseName || ''}`,
        `type=${node.dataType}`,
        `datapoint=${node.datapointPath || ''}`,
      ].join('\n'),
      '变量信息已复制',
    )
    return
  }
  if (action === 'copy-path') {
    if (!node.datapointPath) {
      ElMessage.warning('当前变量还没有数据点 path')
      return
    }
    await copyText(node.datapointPath, '数据点 path 已复制')
    return
  }
  if (action === 'datapoint') {
    ElMessage.info(node.datapointPath ? `数据点：${node.datapointPath}` : '当前变量还没有数据点')
    return
  }
  await removeNode(node)
}

function groupPathOf(groupId?: string | null) {
  if (!groupId) return '未分组'
  const byId = new Map(groups.value.map((group) => [group.id, group]))
  const segments: string[] = []
  const visited = new Set<string>()
  let currentId = groupId
  while (currentId && !visited.has(currentId)) {
    visited.add(currentId)
    const group = byId.get(currentId)
    if (!group) break
    segments.unshift(group.name)
    currentId = group.parentId || ''
  }
  return segments.join('/') || '未分组'
}

function issueSummaryForNode(nodeId: string) {
  return validationIssues.value
    .filter((issue) => issue.nodeId === nodeId)
    .map((issue) => `${issue.severity}:${issue.message}`)
    .join('; ')
}

const samplingSummary = computed(() => {
  const values = nodes.value.map((node) => Number(node.samplingMs || 0)).filter(Boolean)
  if (values.length === 0) return '-'
  const min = Math.min(...values)
  const max = Math.max(...values)
  return min === max ? `${min}ms` : `${min}-${max}ms`
})

const storageSummaryText = computed(() => {
  if (!storageSummary.value) return '未加载'
  if (storageSummary.value.enabledCount === 0) return '项目未配置历史归档'
  return `项目 ${storageSummary.value.enabledCount} 条启用策略，约 ${storageSummary.value.estimatedRowsPerDay} rows/day`
})

const deviceRedundancyText = computed(() => {
  const redundancy = config.value.redundancy
  if (!redundancy || redundancy.enabled === false) return '未配置'
  const count = Array.isArray(redundancy.endpoints) ? redundancy.endpoints.length : 0
  return count > 1 ? `主备优先级 · ${count} endpoint` : '待补备用路径'
})

watch(nodeKeyword, () => {
  if (searchTimer) window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    nodePagination.value.page = 1
    void reloadAll()
  }, 250)
})

watch(quickFilter, () => {
  nodePagination.value.page = 1
  void reloadAll()
})

onBeforeUnmount(() => {
  if (searchTimer) window.clearTimeout(searchTimer)
})

async function copyText(text: string, successMessage: string) {
  await navigator.clipboard.writeText(text)
  ElMessage.success(successMessage)
}

function formatExportValue(value: unknown) {
  if (value === null || value === undefined || value === '') return ''
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

onMounted(() => {
  void reloadAll()
  void reloadStorageSummary()
})
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
  grid-template-columns: auto minmax(180px, 260px);
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.opcua-workbench__actions {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
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

.opcua-workbench__search {
  min-width: 0;
}
.opcua-workbench__filters {
  min-height: 36px;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}
.opcua-workbench__filters button {
  height: 24px;
  padding: 0 9px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.opcua-workbench__filters button.is-active,
.opcua-workbench__filters button:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 32%, var(--dc-border));
  color: var(--dc-primary);
}
.opcua-workbench__menu-mask {
  position: fixed;
  inset: 0;
  z-index: 2100;
}
.opcua-workbench__context-menu {
  position: fixed;
  min-width: 154px;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}
.opcua-workbench__context-menu button {
  width: 100%;
  height: 30px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 13px;
  text-align: left;
}
.opcua-workbench__context-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}
.opcua-workbench__context-menu button.is-danger:hover {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}
.opcua-workbench__menu-icon {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
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
