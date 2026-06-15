<template>
  <section class="modbus-workbench">
    <aside class="modbus-workbench__side">
      <WorkbenchSourceHeader
        :title="connection.name || '未命名 Modbus 接入源'"
        fallback-title="未命名 Modbus 接入源"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #status>
          <button
            type="button"
            class="modbus-workbench__connect-action"
            :class="{ 'is-connected': session.connected.value }"
            :title="session.connected.value ? '断开 Modbus 开发态会话' : '连接 Modbus 开发态会话'"
            @click="toggleSession"
          >
            <span class="modbus-workbench__connect-label is-default">
              {{ session.connected.value ? '已连接' : '连接' }}
            </span>
            <span v-if="session.connected.value" class="modbus-workbench__connect-label is-hover"
              >断开</span
            >
          </button>
        </template>
      </WorkbenchSourceHeader>
      <ModbusGroupTree
        :groups="groups"
        :selected-group-id="selectedGroupId"
        :total="registerPagination.total"
        @select="selectGroup"
        @create="openCreateGroup"
        @create-child="openCreateChildGroup"
        @edit="openEditGroup"
        @delete="removeGroup"
      />
    </aside>

    <main class="modbus-workbench__main">
      <div class="modbus-workbench__bar">
        <div class="modbus-workbench__actions">
          <button
            type="button"
            class="modbus-workbench__icon-action"
            :title="importActionTitle"
            :aria-label="importActionTitle"
            @click="importVisible = true"
          >
            <IconTablerUpload />
          </button>
          <el-dropdown
            trigger="click"
            :disabled="registers.length === 0"
            @command="exportRegisters"
          >
            <button
              type="button"
              class="modbus-workbench__icon-action"
              :disabled="registers.length === 0"
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
            class="modbus-workbench__icon-action is-primary"
            title="新建变量"
            aria-label="新建变量"
            @click="openCreateRegister"
          >
            <IconTablerPlus />
          </button>
          <button
            type="button"
            class="modbus-workbench__icon-action"
            :disabled="!session.connected.value"
            :title="previewActionTitle"
            :aria-label="previewActionTitle"
            @click="openPreview"
          >
            <IconTablerActivityHeartbeat />
          </button>
          <button
            type="button"
            class="modbus-workbench__icon-action"
            :title="validationActionTitle"
            :aria-label="validationActionTitle"
            @click="runValidation"
          >
            <IconTablerChecklist />
          </button>
          <button
            type="button"
            class="modbus-workbench__icon-action"
            :title="readPlanActionTitle"
            :aria-label="readPlanActionTitle"
            @click="openReadPlan"
          >
            <IconTablerRoute />
          </button>
          <button
            type="button"
            class="modbus-workbench__icon-action"
            title="运行契约预览"
            aria-label="运行契约预览"
            @click="contractVisible = true"
          >
            <IconTablerFileDescription />
          </button>
          <button
            type="button"
            class="modbus-workbench__icon-action"
            title="刷新"
            aria-label="刷新"
            @click="reloadAll"
          >
            <IconTablerRefresh />
          </button>
        </div>
        <el-input
          v-model="registerKeyword"
          class="modbus-workbench__search"
          size="small"
          placeholder="搜索变量"
          clearable
        />
      </div>
      <ModbusRegisterTable
        :registers="registers"
        :loading="loading"
        :selected-register-id="selectedRegisterId"
        :page="registerPagination.page"
        :page-size="registerPagination.pageSize"
        :total="registerPagination.total"
        @select="selectRegister"
        @edit="openEditRegister"
        @delete="removeRegister"
        @row-contextmenu="openRegisterMenu"
        @page-change="changeRegisterPage"
        @page-size-change="changeRegisterPageSize"
        @sort-change="changeRegisterSort"
      />
      <div class="modbus-workbench__filters">
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

    <ModbusInspectorPanel
      :group="currentGroup"
      :register="selectedRegister"
      :registers="registers"
      :issues="scopedValidationIssues"
      :estimate="readPlanEstimate"
      :connection="connection"
      :project-id="projectId"
    />

    <ModbusGroupDialog
      v-model="groupDialogVisible"
      :mode="groupDialogMode"
      :groups="groups"
      :group-value="editingGroup"
      :default-parent-id="defaultGroupParentId"
      :loading="saving"
      @submit="saveGroup"
    />
    <ModbusRegisterDialog
      v-model="registerDialogVisible"
      :mode="registerDialogMode"
      :groups="groups"
      :register="editingRegister"
      :default-group-id="selectedGroupId"
      :loading="saving"
      @submit="saveRegister"
    />
    <ModbusImportDialog v-model="importVisible" :loading="saving" @submit="importRegisters" />
    <ModbusPreviewDialog
      v-model="previewVisible"
      :registers="previewRegisters"
      :diagnostics="previewDiagnostics"
    />
    <ModbusValidationDrawer
      v-model="validationVisible"
      :issues="scopedValidationIssues"
      :scope-label="validationScopeLabel"
      @locate="locateRegister"
    />
    <ModbusReadPlanDialog v-model="readPlanVisible" :estimate="readPlanEstimate" />
    <ProtocolContractDrawer
      v-model="contractVisible"
      title="Modbus 运行契约预览"
      :subtitle="contractSubtitle"
      :sections="contractSections"
      :issues="contractIssues"
    />
    <Teleport to="body">
      <div
        v-if="registerMenu.visible"
        class="modbus-workbench__menu-mask"
        @click="closeRegisterMenu"
        @contextmenu.prevent="closeRegisterMenu"
      >
        <div
          class="modbus-workbench__context-menu"
          :style="{ left: `${registerMenu.x}px`, top: `${registerMenu.y}px` }"
          @click.stop
        >
          <button type="button" @click="runRegisterMenuAction('edit')">
            <IconTablerPencil class="modbus-workbench__menu-icon" />
            <span>编辑</span>
          </button>
          <button type="button" @click="runRegisterMenuAction('duplicate')">
            <IconTablerCopy class="modbus-workbench__menu-icon" />
            <span>复制为新变量</span>
          </button>
          <button type="button" @click="runRegisterMenuAction('copy-info')">
            <IconTablerClipboard class="modbus-workbench__menu-icon" />
            <span>复制变量信息</span>
          </button>
          <button type="button" @click="runRegisterMenuAction('copy-path')">
            <IconTablerRoute class="modbus-workbench__menu-icon" />
            <span>复制数据点路径</span>
          </button>
          <button type="button" @click="runRegisterMenuAction('datapoint')">
            <IconTablerExternalLink class="modbus-workbench__menu-icon" />
            <span>查看数据点详情</span>
          </button>
          <button type="button" class="is-danger" @click="runRegisterMenuAction('delete')">
            <IconTablerTrash class="modbus-workbench__menu-icon" />
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
import ModbusGroupDialog from '@/components/modbus/ModbusGroupDialog.vue'
import ModbusGroupTree from '@/components/modbus/ModbusGroupTree.vue'
import ModbusImportDialog from '@/components/modbus/ModbusImportDialog.vue'
import ModbusInspectorPanel from '@/components/modbus/ModbusInspectorPanel.vue'
import ModbusPreviewDialog from '@/components/modbus/ModbusPreviewDialog.vue'
import ModbusReadPlanDialog from '@/components/modbus/ModbusReadPlanDialog.vue'
import ModbusRegisterDialog from '@/components/modbus/ModbusRegisterDialog.vue'
import ModbusRegisterTable from '@/components/modbus/ModbusRegisterTable.vue'
import ModbusValidationDrawer from '@/components/modbus/ModbusValidationDrawer.vue'
import ProtocolContractDrawer from './ProtocolContractDrawer.vue'
import type {
  ModbusReadPlanEstimate,
  ModbusReadValue,
  ModbusRegister,
  ModbusRegisterGroup,
  ModbusValidationIssue,
} from '@/components/modbus/types'
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

defineEmits<{ (event: 'back'): void }>()

const emptyEstimate = (): ModbusReadPlanEstimate => ({
  registerCount: 0,
  unitCount: 0,
  readCount: 0,
  readsPerSecond: 0,
  plans: [],
  diagnostics: [],
})

const groups = ref<ModbusRegisterGroup[]>([])
const registers = ref<ModbusRegister[]>([])
const readPlanEstimate = ref<ModbusReadPlanEstimate>(emptyEstimate())
const registerPagination = ref({ page: 1, pageSize: 20, total: 0, totalPages: 0 })
const loading = ref(false)
const saving = ref(false)
const selectedGroupId = ref('')
const selectedRegisterId = ref('')
const registerKeyword = ref('')
const groupDialogVisible = ref(false)
const groupDialogMode = ref<'create' | 'edit'>('create')
const editingGroup = ref<ModbusRegisterGroup | null>(null)
const defaultGroupParentId = ref('')
const registerDialogVisible = ref(false)
const registerDialogMode = ref<'create' | 'edit'>('create')
const editingRegister = ref<ModbusRegister | null>(null)
const importVisible = ref(false)
const previewVisible = ref(false)
const previewRegisters = ref<ModbusReadValue[]>([])
const previewDiagnostics = ref<string[]>([])
const validationVisible = ref(false)
const validationIssues = ref<ModbusValidationIssue[]>([])
const readPlanVisible = ref(false)
const contractVisible = ref(false)
const quickFilter = ref('all')
const storageSummary = ref<StoragePolicySummary | null>(null)
const registerSort = ref<{ sortBy?: string; sortOrder?: string }>({})
let searchTimer: ReturnType<typeof window.setTimeout> | undefined
const registerMenu = ref({
  visible: false,
  x: 0,
  y: 0,
  register: null as ModbusRegister | null,
})
const quickFilters = [
  { label: '全部', value: 'all' },
  { label: '异常', value: 'issue' },
  { label: '未同步', value: 'datapoint' },
  { label: '可写', value: 'writable' },
  { label: '已停用', value: 'disabled' },
]

const session = useProtocolDevSession({
  create: () => dataAPI.createModbusDevSession(props.projectId, props.connection.id),
  close: (sessionId: string) =>
    dataAPI.closeModbusDevSession(props.projectId, props.connection.id, sessionId),
})

const config = computed(() => props.connection.config || {})
const endpointText = computed(() => {
  if (config.value.mode === 'rtu')
    return String(config.value.serialConfig?.port || 'RTU 串口未配置')
  return `${config.value.host || '未配置 host'}:${config.value.port || 502}`
})
const sourceMetaRows = computed(() => [
  { label: '类型', value: 'Modbus' },
  { label: '端点', value: endpointText.value },
  { label: '默认从站', value: String(config.value.slaveId ?? 1) },
])
const currentGroup = computed(
  () => groups.value.find((group) => group.id === selectedGroupId.value) || null,
)
const selectedRegister = computed(
  () => registers.value.find((item) => item.id === selectedRegisterId.value) || null,
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
  selectedRegister.value
    ? `校验当前变量：${selectedRegister.value.name}`
    : currentGroup.value
      ? `校验当前分组：${currentGroup.value.name}`
      : '校验全部变量',
)
const readPlanActionTitle = computed(() =>
  currentGroup.value ? `当前分组读取计划：${currentGroup.value.name}` : '全部变量读取计划',
)
const validationScopeLabel = computed(() => {
  if (selectedRegister.value) return `当前变量：${selectedRegister.value.name}`
  if (currentGroup.value) return `当前分组：${currentGroup.value.name}`
  return '全部变量'
})
const scopedValidationIssues = computed(() => {
  if (selectedRegisterId.value) {
    return validationIssues.value.filter((item) => item.registerId === selectedRegisterId.value)
  }
  if (selectedGroupId.value) {
    const ids = new Set(
      registers.value
        .filter((item) => item.groupId === selectedGroupId.value)
        .map((item) => item.id),
    )
    return validationIssues.value.filter((item) => item.registerId && ids.has(item.registerId))
  }
  return validationIssues.value
})
const registerExportHeaders = [
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
const buildRegisterExportRows = (items: ModbusRegister[]) =>
  items.map((register) => ({
    变量名: register.name,
    Code: register.code,
    分组: groupPathOf(register.groupId),
    '协议地址 / NodeId': formatModbusAddress(register),
    数据类型: register.dataType,
    采集周期: `${register.pollIntervalMs}ms`,
    发布能力: register.accessLevel || 'read',
    '数据点 path': register.datapointPath || '',
    状态: register.status || '',
    最近开发态读取值: formatExportValue(register.lastValue),
    质量: register.quality || '',
    诊断问题摘要: issueSummaryForRegister(register.id),
  }))
const registerExportRows = computed(() => buildRegisterExportRows(registers.value))
const issueExportRows = computed(() =>
  buildIssueExportRows(registers.value, scopedValidationIssues.value),
)
const buildIssueExportRows = (items: ModbusRegister[], issues: ModbusValidationIssue[]) => {
  const registerByID = new Map(items.map((item) => [item.id, item]))
  return issues
    .filter((issue) => !issue.registerId || registerByID.has(issue.registerId))
    .map((issue) => {
      const register = issue.registerId ? registerByID.get(issue.registerId) : null
      return {
        严重级别: issue.severity,
        问题类型: issue.code,
        变量名: issue.registerName || register?.name || '',
        Code: register?.code || '',
        分组: groupPathOf(issue.groupId || register?.groupId),
        定位字段: issue.registerId ? '变量' : issue.groupId ? '分组' : '连接',
        问题说明: issue.message,
        建议处理: '按诊断提示修正建模后重新校验',
      }
    })
}
const contractSubtitle = computed(() =>
  selectedRegister.value
    ? `当前变量：${selectedRegister.value.name}`
    : currentGroup.value
      ? `当前范围：${currentGroup.value.name}`
      : '当前范围：全部变量',
)
const contractIssues = computed(() => [
  ...scopedValidationIssues.value.map((issue) => ({
    severity: issue.severity,
    message: issue.message,
  })),
  ...readPlanEstimate.value.diagnostics.map((message) => ({ severity: 'warning', message })),
])
const contractSections = computed(() => [
  {
    title: '协议建模',
    rows: [
      { label: '端点', value: endpointText.value },
      { label: '变量数', value: registerPagination.value.total },
      { label: '从站数', value: unitCount.value },
      { label: '发布能力', value: '寄存器区约束，只配置读写权限，不提供写值' },
    ],
  },
  {
    title: '读取计划',
    rows: [
      { label: '读取次数', value: readPlanEstimate.value.readCount },
      { label: '预计 reads/s', value: readPlanEstimate.value.readsPerSecond.toFixed(2) },
      { label: '开发态位置', value: '平台侧 data_service' },
      { label: 'RTU 策略', value: config.value.mode === 'rtu' ? 'Linux 环境优先级较低，建议节点侧验证' : 'TCP 优先' },
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
    notes: ['读取计划由系统按从站、寄存器区、地址和周期估算，工作台不提供手工编辑 readPlan。'],
  },
])

const unwrapList = <T,>(response: any): T[] =>
  response?.data?.list || response?.data?.data?.list || []
const unwrapData = (response: any) => response?.data?.data || response?.data || {}
const unwrapPagination = (response: any) =>
  response?.data?.pagination || response?.data?.data?.pagination

const reloadEstimate = async () => {
  const response = await dataAPI.getModbusReadPlanEstimate(props.projectId, props.connection.id, {
    groupId: selectedGroupId.value || undefined,
  })
  readPlanEstimate.value = { ...emptyEstimate(), ...unwrapData(response) }
}

const reloadAll = async () => {
  loading.value = true
  try {
    const [groupResponse, registerResponse] = await Promise.all([
      dataAPI.getModbusRegisterGroups(props.projectId, props.connection.id),
      dataAPI.getModbusRegisters(props.projectId, props.connection.id, {
        groupId: selectedGroupId.value || undefined,
        q: registerKeyword.value.trim() || undefined,
        filter: quickFilter.value === 'all' ? undefined : quickFilter.value,
        sortBy: registerSort.value.sortBy,
        sortOrder: registerSort.value.sortOrder,
        page: registerPagination.value.page,
        pageSize: registerPagination.value.pageSize,
      }),
    ])
    groups.value = unwrapList<ModbusRegisterGroup>(groupResponse)
    registers.value = unwrapList<ModbusRegister>(registerResponse)
    registerPagination.value = {
      ...registerPagination.value,
      ...unwrapPagination(registerResponse),
    }
    await reloadEstimate()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载 Modbus 建模数据失败'))
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

const loadExportRegisters = async () => {
  const pageSize = 100
  const firstResponse = await dataAPI.getModbusRegisters(props.projectId, props.connection.id, {
    groupId: selectedGroupId.value || undefined,
    q: registerKeyword.value.trim() || undefined,
    filter: quickFilter.value === 'all' ? undefined : quickFilter.value,
    sortBy: registerSort.value.sortBy,
    sortOrder: registerSort.value.sortOrder,
    page: 1,
    pageSize,
  })
  const firstPage = unwrapList<ModbusRegister>(firstResponse)
  const pagination = unwrapPagination(firstResponse) || {}
  const totalPages = Number(
    pagination.totalPages || Math.ceil(Number(pagination.total || firstPage.length) / pageSize) || 1,
  )
  const result = [...firstPage]
  for (let page = 2; page <= totalPages; page += 1) {
    const response = await dataAPI.getModbusRegisters(props.projectId, props.connection.id, {
      groupId: selectedGroupId.value || undefined,
      q: registerKeyword.value.trim() || undefined,
      filter: quickFilter.value === 'all' ? undefined : quickFilter.value,
      sortBy: registerSort.value.sortBy,
      sortOrder: registerSort.value.sortOrder,
      page,
      pageSize,
    })
    result.push(...unwrapList<ModbusRegister>(response))
  }
  return result
}

const selectGroup = async (groupId: string) => {
  selectedGroupId.value = groupId
  selectedRegisterId.value = ''
  registerPagination.value.page = 1
  await reloadAll()
}

const changeRegisterPage = async (page: number) => {
  registerPagination.value.page = page
  await reloadAll()
}

const changeRegisterPageSize = async (pageSize: number) => {
  registerPagination.value.page = 1
  registerPagination.value.pageSize = pageSize
  await reloadAll()
}

const changeRegisterSort = async (payload: { prop?: string; order?: string | null }) => {
  registerSort.value = {
    sortBy: payload.prop || undefined,
    sortOrder: payload.order === 'descending' ? 'desc' : payload.order === 'ascending' ? 'asc' : undefined,
  }
  registerPagination.value.page = 1
  await reloadAll()
}

const selectRegister = (register: ModbusRegister) => {
  selectedRegisterId.value = register.id
}

const openCreateGroup = () => {
  editingGroup.value = null
  defaultGroupParentId.value = ''
  groupDialogMode.value = 'create'
  groupDialogVisible.value = true
}

const openCreateChildGroup = (group: ModbusRegisterGroup | null) => {
  editingGroup.value = null
  defaultGroupParentId.value = group?.id || ''
  groupDialogMode.value = 'create'
  groupDialogVisible.value = true
}

const openEditGroup = (group: ModbusRegisterGroup) => {
  editingGroup.value = group
  defaultGroupParentId.value = ''
  groupDialogMode.value = 'edit'
  groupDialogVisible.value = true
}

const saveGroup = async (payload: Record<string, unknown>) => {
  saving.value = true
  try {
    if (groupDialogMode.value === 'edit' && editingGroup.value) {
      await dataAPI.updateModbusRegisterGroup(
        props.projectId,
        props.connection.id,
        editingGroup.value.id,
        {
          ...payload,
          hasParentId: true,
        },
      )
    } else {
      await dataAPI.createModbusRegisterGroup(props.projectId, props.connection.id, payload)
    }
    groupDialogVisible.value = false
    await reloadAll()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存寄存器组失败'))
  } finally {
    saving.value = false
  }
}

const removeGroup = async (group: ModbusRegisterGroup) => {
  await ElMessageBox.confirm(
    `删除寄存器组“${group.name}”？组内变量会移动到未分组。`,
    '删除寄存器组',
  )
  await dataAPI.deleteModbusRegisterGroup(props.projectId, props.connection.id, group.id)
  if (selectedGroupId.value === group.id) selectedGroupId.value = ''
  await reloadAll()
}

const openCreateRegister = () => {
  editingRegister.value = null
  registerDialogMode.value = 'create'
  registerDialogVisible.value = true
}

const openDuplicateRegister = (register: ModbusRegister) => {
  editingRegister.value = {
    ...register,
    id: '',
    name: `${register.name} 副本`,
    code: `${register.code}_copy`,
    datapointId: null,
    datapointPath: null,
    datapointStatus: null,
    lastValue: undefined,
    quality: undefined,
    lastUpdatedAt: null,
  }
  registerDialogMode.value = 'create'
  registerDialogVisible.value = true
}

const openEditRegister = (register: ModbusRegister) => {
  editingRegister.value = register
  registerDialogMode.value = 'edit'
  registerDialogVisible.value = true
}

const saveRegister = async (payload: Record<string, unknown>) => {
  saving.value = true
  try {
    const saved =
      registerDialogMode.value === 'edit' && editingRegister.value
        ? await dataAPI.updateModbusRegister(
            props.projectId,
            props.connection.id,
            editingRegister.value.id,
            payload,
          )
        : await dataAPI.createModbusRegister(props.projectId, props.connection.id, payload)
    registerDialogVisible.value = false
    await reloadAll()
    selectedRegisterId.value = saved?.data?.id || saved?.data?.data?.id || ''
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存变量失败'))
  } finally {
    saving.value = false
  }
}

const removeRegister = async (register: ModbusRegister) => {
  await ElMessageBox.confirm(`删除变量“${register.name}”？对应数据点将标记为失效。`, '删除变量')
  await dataAPI.deleteModbusRegister(props.projectId, props.connection.id, register.id)
  if (selectedRegisterId.value === register.id) selectedRegisterId.value = ''
  await reloadAll()
}

const importRegisters = async (rows: Array<Record<string, unknown>>) => {
  saving.value = true
  try {
    await dataAPI.batchImportModbusRegisters(props.projectId, props.connection.id, {
      groupId: selectedGroupId.value || null,
      registers: rows,
    })
    importVisible.value = false
    await reloadAll()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '导入变量失败'))
  } finally {
    saving.value = false
  }
}

const exportRegisters = async (format: string | number | object) => {
  loading.value = true
  try {
    const exportItems = await loadExportRegisters()
    const rows = buildRegisterExportRows(exportItems)
    if (rows.length === 0) {
      ElMessage.warning('当前筛选结果没有可导出的变量')
      return
    }
    const suffix = format === 'xlsx' ? 'xlsx' : 'csv'
    const filename = `modbus-variables-filtered.${suffix}`
    if (suffix === 'xlsx') {
      downloadXlsx(filename, [
        { name: '变量清单', headers: registerExportHeaders, rows },
        { name: '问题清单', headers: issueExportHeaders, rows: buildIssueExportRows(exportItems, validationIssues.value) },
      ])
      return
    }
    downloadCsv(filename, registerExportHeaders, rows)
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
    ElMessage.warning('请先连接 Modbus 开发态会话')
    return
  }
  try {
    const response = await dataAPI.pollModbusDevSession(
      props.projectId,
      props.connection.id,
      session.sessionId.value,
      {
        groupId: selectedGroupId.value || null,
      },
    )
    const data = unwrapData(response)
    previewRegisters.value = data.values || []
    previewDiagnostics.value = data.diagnostics || []
    previewVisible.value = true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '变量预览失败'))
  }
}

const runValidation = async () => {
  try {
    const response = await dataAPI.validateModbusModel(props.projectId, props.connection.id)
    const data = unwrapData(response)
    validationIssues.value = data.issues || []
    validationVisible.value = true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '建模校验失败'))
  }
}

const openReadPlan = async () => {
  await reloadEstimate()
  readPlanVisible.value = true
}

const locateRegister = (registerId: string) => {
  selectedRegisterId.value = registerId
  const register = registers.value.find((item) => item.id === registerId)
  selectedGroupId.value = register?.groupId || ''
  validationVisible.value = false
}

const openRegisterMenu = (event: MouseEvent, register: ModbusRegister) => {
  event.preventDefault()
  selectedRegisterId.value = register.id
  registerMenu.value = {
    visible: true,
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 210),
    register,
  }
}

const closeRegisterMenu = () => {
  registerMenu.value.visible = false
}

const runRegisterMenuAction = async (
  action: 'edit' | 'duplicate' | 'copy-info' | 'copy-path' | 'datapoint' | 'delete',
) => {
  const register = registerMenu.value.register
  closeRegisterMenu()
  if (!register) return
  if (action === 'edit') {
    openEditRegister(register)
    return
  }
  if (action === 'duplicate') {
    openDuplicateRegister(register)
    return
  }
  if (action === 'copy-info') {
    await copyText(
      [
        `name=${register.name}`,
        `code=${register.code}`,
        `address=${formatModbusAddress(register)}`,
        `type=${register.dataType}`,
        `datapoint=${register.datapointPath || ''}`,
      ].join('\n'),
      '变量信息已复制',
    )
    return
  }
  if (action === 'copy-path') {
    if (!register.datapointPath) {
      ElMessage.warning('当前变量还没有数据点 path')
      return
    }
    await copyText(register.datapointPath, '数据点 path 已复制')
    return
  }
  if (action === 'datapoint') {
    ElMessage.info(
      register.datapointPath ? `数据点：${register.datapointPath}` : '当前变量还没有数据点',
    )
    return
  }
  await removeRegister(register)
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

function formatModbusAddress(register: ModbusRegister) {
  return `${register.unitId}/${register.area}/${register.address} (protocol ${register.protocolAddress})`
}

const unitCount = computed(() => new Set(registers.value.map((item) => item.unitId)).size)

const deviceRedundancyText = computed(() => {
  const redundancy = config.value.redundancy
  if (!redundancy || redundancy.enabled === false) return '未配置'
  const count = Array.isArray(redundancy.endpoints) ? redundancy.endpoints.length : 0
  return count > 1 ? `主备优先级 · ${count} endpoint` : '待补备用路径'
})

const storageSummaryText = computed(() => {
  if (!storageSummary.value) return '未加载'
  if (storageSummary.value.enabledCount === 0) return '项目未配置历史归档'
  return `项目 ${storageSummary.value.enabledCount} 条启用策略，约 ${storageSummary.value.estimatedRowsPerDay} rows/day`
})

async function copyText(text: string, successMessage: string) {
  await navigator.clipboard.writeText(text)
  ElMessage.success(successMessage)
}

function issueSummaryForRegister(registerId: string) {
  return validationIssues.value
    .filter((issue) => issue.registerId === registerId)
    .map((issue) => `${issue.severity}:${issue.message}`)
    .join('; ')
}

function formatExportValue(value: unknown) {
  if (value === null || value === undefined || value === '') return ''
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

watch(registerKeyword, () => {
  if (searchTimer) window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    registerPagination.value.page = 1
    void reloadAll()
  }, 250)
})

watch(quickFilter, () => {
  registerPagination.value.page = 1
  void reloadAll()
})

onBeforeUnmount(() => {
  if (searchTimer) window.clearTimeout(searchTimer)
})

onMounted(() => {
  void reloadAll()
  void reloadStorageSummary()
})
</script>

<style scoped>
.modbus-workbench {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 264px minmax(0, 1fr) 292px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
  overflow: hidden;
}

.modbus-workbench__side {
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--dc-border);
  overflow: hidden;
  background: var(--dc-surface-muted);
}

.modbus-workbench__side :deep(.modbus-group-tree) {
  flex: 1;
  min-height: 0;
}

.modbus-workbench__connect-action {
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

.modbus-workbench__connect-label.is-hover {
  display: none;
}

.modbus-workbench__connect-action.is-connected:hover .modbus-workbench__connect-label.is-default {
  display: none;
}

.modbus-workbench__connect-action.is-connected:hover .modbus-workbench__connect-label.is-hover {
  display: inline;
}

.modbus-workbench__connect-action::before {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: currentColor;
  content: '';
}

.modbus-workbench__connect-action.is-connected {
  border-color: color-mix(in oklch, var(--dc-success) 32%, var(--dc-border));
  background: color-mix(in oklch, var(--dc-success) 12%, var(--dc-surface-raised));
  color: var(--dc-success);
}

.modbus-workbench__connect-action.is-connected:hover {
  border-color: rgba(220, 38, 38, 0.22);
  background: rgba(220, 38, 38, 0.08);
  color: #b91c1c;
}

.modbus-workbench__main {
  min-width: 0;
  min-height: 0;
  background: var(--dc-surface-raised);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modbus-workbench__bar {
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

.modbus-workbench__actions {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.modbus-workbench__icon-action {
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

.modbus-workbench__icon-action:hover:not(:disabled) {
  color: var(--dc-primary);
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
}

.modbus-workbench__icon-action:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.modbus-workbench__icon-action.is-primary {
  border-color: var(--dc-primary);
  background: var(--dc-primary);
  color: var(--dc-surface-raised);
}

.modbus-workbench__icon-action svg {
  width: 15px;
  height: 15px;
}

.modbus-workbench__search {
  min-width: 0;
}
.modbus-workbench__filters {
  min-height: 36px;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}
.modbus-workbench__filters button {
  height: 24px;
  padding: 0 9px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.modbus-workbench__filters button.is-active,
.modbus-workbench__filters button:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 32%, var(--dc-border));
  color: var(--dc-primary);
}
.modbus-workbench__menu-mask {
  position: fixed;
  inset: 0;
  z-index: 2100;
}
.modbus-workbench__context-menu {
  position: fixed;
  min-width: 154px;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}
.modbus-workbench__context-menu button {
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
.modbus-workbench__context-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}
.modbus-workbench__context-menu button.is-danger:hover {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}
.modbus-workbench__menu-icon {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}

@media (max-width: 1100px) {
  .modbus-workbench {
    grid-template-columns: 230px minmax(0, 1fr);
  }

  .modbus-workbench :deep(.modbus-inspector) {
    display: none;
  }
}
</style>
