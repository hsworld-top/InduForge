<template>
  <section class="s7-workbench">
    <aside class="s7-workbench__side">
      <WorkbenchSourceHeader
        :title="connection.name || '未命名 S7 接入源'"
        fallback-title="未命名 S7 接入源"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #status>
          <button
            type="button"
            class="s7-workbench__connect-action"
            :class="{ 'is-connected': session.connected.value }"
            :title="session.connected.value ? '断开 S7 开发态会话' : '连接 S7 开发态会话'"
            @click="toggleSession"
          >
            <span>{{ session.connected.value ? '已连接' : '连接' }}</span>
          </button>
        </template>
      </WorkbenchSourceHeader>
      <S7GroupTree
        :groups="groups"
        :selected-group-id="selectedGroupId"
        :total="variablePagination.total"
        @select="selectGroup"
        @create="openCreateGroup"
        @edit="openEditGroup"
        @delete="removeGroup"
      />
    </aside>
    <main class="s7-workbench__main">
      <div class="s7-workbench__bar">
        <div class="s7-workbench__actions">
          <button
            type="button"
            class="s7-workbench__icon-action"
            title="配置 PLC 类型"
            aria-label="配置 PLC 类型"
            @click="profileVisible = true"
          >
            <IconTablerCpu />
          </button>
          <button
            type="button"
            class="s7-workbench__icon-action"
            :title="importActionTitle"
            :aria-label="importActionTitle"
            @click="importVisible = true"
          >
            <IconTablerUpload />
          </button>
          <el-dropdown
            trigger="click"
            :disabled="filteredVariables.length === 0"
            @command="exportVariables"
          >
            <button
              type="button"
              class="s7-workbench__icon-action"
              :disabled="filteredVariables.length === 0"
              title="导出当前页变量"
              aria-label="导出当前页变量"
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
            class="s7-workbench__icon-action is-primary"
            title="新建变量"
            aria-label="新建变量"
            @click="openCreateVariable"
          >
            <IconTablerPlus />
          </button>
          <button
            type="button"
            class="s7-workbench__icon-action"
            :disabled="!session.connected.value"
            :title="readActionTitle"
            :aria-label="readActionTitle"
            @click="readCurrentScope"
          >
            <IconTablerBolt />
          </button>
          <button
            type="button"
            class="s7-workbench__icon-action"
            :disabled="!session.connected.value"
            :title="previewActionTitle"
            :aria-label="previewActionTitle"
            @click="openPreview"
          >
            <IconTablerActivityHeartbeat />
          </button>
          <button
            type="button"
            class="s7-workbench__icon-action"
            :title="validationActionTitle"
            :aria-label="validationActionTitle"
            @click="runValidation"
          >
            <IconTablerChecklist />
          </button>
          <button
            type="button"
            class="s7-workbench__icon-action"
            :title="readPlanActionTitle"
            :aria-label="readPlanActionTitle"
            @click="openReadPlan"
          >
            <IconTablerRoute />
          </button>
          <button
            type="button"
            class="s7-workbench__icon-action"
            title="刷新"
            aria-label="刷新"
            @click="reloadAll"
          >
            <IconTablerRefresh />
          </button>
        </div>
        <el-input
          v-model="keyword"
          class="s7-workbench__search"
          size="small"
          placeholder="搜索变量"
          clearable
        />
      </div>
      <S7VariableTable
        :variables="filteredVariables"
        :loading="loading"
        :selected-variable-id="selectedVariableId"
        :page="variablePagination.page"
        :page-size="variablePagination.pageSize"
        :total="variablePagination.total"
        @select="selectVariable"
        @edit="openEditVariable"
        @delete="removeVariable"
        @row-contextmenu="openVariableMenu"
        @page-change="changeVariablePage"
        @page-size-change="changeVariablePageSize"
      />
      <div class="s7-workbench__filters">
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
    <S7InspectorPanel
      :profile="profile"
      :group="currentGroup"
      :variable="selectedVariable"
      :variables="variables"
      :issues="scopedValidationIssues"
      :estimate="readPlanEstimate"
      :connection="connection"
      :project-id="projectId"
    />

    <S7ProfileDialog
      v-model="profileVisible"
      :profile="profile"
      :loading="saving"
      @submit="saveProfile"
    />
    <S7GroupDialog
      v-model="groupVisible"
      :mode="groupMode"
      :groups="groups"
      :group-value="editingGroup"
      :loading="saving"
      @submit="saveGroup"
    />
    <S7VariableDialog
      v-model="variableVisible"
      :mode="variableMode"
      :groups="groups"
      :variable="editingVariable"
      :default-group-id="selectedGroupId"
      :loading="saving"
      @submit="saveVariable"
    />
    <S7ImportDialog v-model="importVisible" :loading="saving" @submit="importVariables" />
    <S7PreviewDialog
      v-model="previewVisible"
      :values="previewValues"
      :diagnostics="previewDiagnostics"
    />
    <S7ValidationDrawer
      v-model="validationVisible"
      :issues="scopedValidationIssues"
      :scope-label="validationScopeLabel"
      @locate="locateVariable"
    />
    <S7ReadPlanDialog v-model="readPlanVisible" :estimate="readPlanEstimate" />
    <Teleport to="body">
      <div
        v-if="variableMenu.visible"
        class="s7-workbench__menu-mask"
        @click="closeVariableMenu"
        @contextmenu.prevent="closeVariableMenu"
      >
        <div
          class="s7-workbench__context-menu"
          :style="{ left: `${variableMenu.x}px`, top: `${variableMenu.y}px` }"
          @click.stop
        >
          <button type="button" @click="runVariableMenuAction('edit')">
            <IconTablerPencil class="s7-workbench__menu-icon" />
            <span>编辑</span>
          </button>
          <button type="button" @click="runVariableMenuAction('duplicate')">
            <IconTablerCopy class="s7-workbench__menu-icon" />
            <span>复制为新变量</span>
          </button>
          <button type="button" @click="runVariableMenuAction('copy-info')">
            <IconTablerClipboard class="s7-workbench__menu-icon" />
            <span>复制变量信息</span>
          </button>
          <button type="button" @click="runVariableMenuAction('copy-path')">
            <IconTablerRoute class="s7-workbench__menu-icon" />
            <span>复制数据点路径</span>
          </button>
          <button type="button" @click="runVariableMenuAction('datapoint')">
            <IconTablerExternalLink class="s7-workbench__menu-icon" />
            <span>查看数据点详情</span>
          </button>
          <button type="button" class="is-danger" @click="runVariableMenuAction('delete')">
            <IconTablerTrash class="s7-workbench__menu-icon" />
            <span>删除变量</span>
          </button>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'
import { downloadCsv, downloadXlsx } from '@/utils/tabular-file'
import { useProtocolDevSession } from './useProtocolDevSession'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import S7GroupDialog from '@/components/s7/S7GroupDialog.vue'
import S7GroupTree from '@/components/s7/S7GroupTree.vue'
import S7ImportDialog from '@/components/s7/S7ImportDialog.vue'
import S7InspectorPanel from '@/components/s7/S7InspectorPanel.vue'
import S7PreviewDialog from '@/components/s7/S7PreviewDialog.vue'
import S7ProfileDialog from '@/components/s7/S7ProfileDialog.vue'
import S7ReadPlanDialog from '@/components/s7/S7ReadPlanDialog.vue'
import S7ValidationDrawer from '@/components/s7/S7ValidationDrawer.vue'
import S7VariableDialog from '@/components/s7/S7VariableDialog.vue'
import S7VariableTable from '@/components/s7/S7VariableTable.vue'
import type {
  S7Profile,
  S7ReadPlanEstimate,
  S7ReadValue,
  S7ValidationIssue,
  S7Variable,
  S7VariableGroup,
} from '@/components/s7/types'
import IconTablerActivityHeartbeat from '~icons/tabler/activity-heartbeat'
import IconTablerBolt from '~icons/tabler/bolt'
import IconTablerChecklist from '~icons/tabler/checklist'
import IconTablerClipboard from '~icons/tabler/clipboard'
import IconTablerCopy from '~icons/tabler/copy'
import IconTablerCpu from '~icons/tabler/cpu'
import IconTablerDownload from '~icons/tabler/download'
import IconTablerExternalLink from '~icons/tabler/external-link'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerRoute from '~icons/tabler/route'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerUpload from '~icons/tabler/upload'

type AccessSourceConnection = {
  id: string
  name?: string
  type?: string
  status?: string
  config?: Record<string, any>
}
const props = defineProps<{ connection: AccessSourceConnection; projectId: string }>()
defineEmits<{ (event: 'back'): void }>()

const emptyEstimate = (): S7ReadPlanEstimate => ({
  variableCount: 0,
  blockCount: 0,
  totalReadBytes: 0,
  readsPerSecond: 0,
  estimatedCycleMs: 0,
  largestBlockBytes: 0,
  fragmentedGroupCount: 0,
  plans: [],
  diagnostics: [],
})
const profile = ref<S7Profile | null>(null)
const groups = ref<S7VariableGroup[]>([])
const variables = ref<S7Variable[]>([])
const readPlanEstimate = ref<S7ReadPlanEstimate>(emptyEstimate())
const variablePagination = ref({ page: 1, pageSize: 20, total: 0, totalPages: 0 })
const validationIssues = ref<S7ValidationIssue[]>([])
const loading = ref(false)
const saving = ref(false)
const selectedGroupId = ref('')
const selectedVariableId = ref('')
const keyword = ref('')
const profileVisible = ref(false)
const groupVisible = ref(false)
const groupMode = ref<'create' | 'edit'>('create')
const editingGroup = ref<S7VariableGroup | null>(null)
const variableVisible = ref(false)
const variableMode = ref<'create' | 'edit'>('create')
const editingVariable = ref<S7Variable | null>(null)
const importVisible = ref(false)
const previewVisible = ref(false)
const previewValues = ref<S7ReadValue[]>([])
const previewDiagnostics = ref<string[]>([])
const validationVisible = ref(false)
const readPlanVisible = ref(false)
const quickFilter = ref('all')
const variableMenu = ref({
  visible: false,
  x: 0,
  y: 0,
  variable: null as S7Variable | null,
})
const quickFilters = [
  { label: '全部', value: 'all' },
  { label: '异常', value: 'issue' },
  { label: '未同步', value: 'datapoint' },
  { label: '可写', value: 'writable' },
  { label: '已停用', value: 'disabled' },
]

const session = useProtocolDevSession({
  create: () => dataAPI.createS7DevSession(props.projectId, props.connection.id),
  close: (sessionId: string) =>
    dataAPI.closeS7DevSession(props.projectId, props.connection.id, sessionId),
})

const config = computed(() => props.connection.config || {})
const sourceMetaRows = computed(() => [
  { label: '类型', value: 'Siemens S7' },
  {
    label: '端点',
    value: profile.value
      ? `${profile.value.host}:${profile.value.port}`
      : `${config.value.host || '未配置 host'}:${config.value.port || 102}`,
  },
  { label: 'PLC', value: profile.value?.plcFamily || '未确认' },
  {
    label: 'Rack/Slot',
    value: profile.value ? `${profile.value.rack}/${profile.value.slot}` : '-',
  },
])
const currentGroup = computed(
  () => groups.value.find((group) => group.id === selectedGroupId.value) || null,
)
const selectedVariable = computed(
  () => variables.value.find((item) => item.id === selectedVariableId.value) || null,
)
const importActionTitle = computed(() =>
  currentGroup.value ? `导入到「${currentGroup.value.name}」` : '导入到未分组',
)
const readActionTitle = computed(() =>
  session.connected.value
    ? currentGroup.value
      ? `读取「${currentGroup.value.name}」最近值`
      : '读取全部变量最近值'
    : '连接后可读取最近值',
)
const previewActionTitle = computed(() =>
  session.connected.value
    ? currentGroup.value
      ? `预览「${currentGroup.value.name}」变量`
      : '预览全部变量'
    : '连接后可预览当前分组变量',
)
const validationActionTitle = computed(() =>
  selectedVariable.value
    ? `校验当前变量：${selectedVariable.value.name}`
    : currentGroup.value
      ? `校验当前分组：${currentGroup.value.name}`
      : '校验全部变量',
)
const readPlanActionTitle = computed(() =>
  currentGroup.value ? `当前分组读取计划：${currentGroup.value.name}` : '全部变量读取计划',
)
const filteredVariables = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  return variables.value.filter((item) => {
    const inGroup = !selectedGroupId.value || item.groupId === selectedGroupId.value
    const matchedFilter = matchQuickFilter(item)
    const matched =
      !text ||
      [
        item.name,
        item.code,
        item.normalizedAddress,
        item.addressText,
        item.datapointPath || '',
      ].some((value) =>
        String(value || '')
          .toLowerCase()
          .includes(text),
      )
    return inGroup && matchedFilter && matched
  })
})
const validationScopeLabel = computed(() =>
  selectedVariable.value
    ? `当前变量：${selectedVariable.value.name}`
    : currentGroup.value
      ? `当前分组：${currentGroup.value.name}`
      : '全部变量',
)
const scopedValidationIssues = computed(() => {
  if (selectedVariableId.value)
    return validationIssues.value.filter((item) => item.variableId === selectedVariableId.value)
  if (selectedGroupId.value) {
    const ids = new Set(
      variables.value
        .filter((item) => item.groupId === selectedGroupId.value)
        .map((item) => item.id),
    )
    return validationIssues.value.filter((item) => item.variableId && ids.has(item.variableId))
  }
  return validationIssues.value
})
const variableExportHeaders = [
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
const variableExportRows = computed(() =>
  filteredVariables.value.map((variable) => ({
    变量名: variable.name,
    Code: variable.code,
    分组: groupPathOf(variable.groupId),
    '协议地址 / NodeId': variable.normalizedAddress || variable.addressText,
    数据类型: variable.dataType,
    采集周期: `${variable.pollIntervalMs}ms`,
    发布能力: 'read',
    '数据点 path': variable.datapointPath || '',
    状态: variable.status || '',
    最近开发态读取值: formatExportValue(variable.lastValue),
    质量: variable.quality || '',
    诊断问题摘要: issueSummaryForVariable(variable.id),
  })),
)
const issueExportRows = computed(() =>
  scopedValidationIssues.value.map((issue) => {
    const variable = issue.variableId
      ? variables.value.find((item) => item.id === issue.variableId)
      : null
    return {
      严重级别: issue.severity,
      问题类型: issue.code,
      变量名: issue.variableName || variable?.name || '',
      Code: variable?.code || '',
      分组: groupPathOf(issue.groupId || variable?.groupId),
      定位字段: issue.variableId ? '变量' : issue.groupId ? '分组' : '连接',
      问题说明: issue.message,
      建议处理: '按诊断提示修正建模后重新校验',
    }
  }),
)

const unwrapList = <T,>(response: any): T[] =>
  response?.data?.list || response?.data?.data?.list || []
const unwrapData = (response: any) => response?.data?.data || response?.data || {}
const unwrapPagination = (response: any) =>
  response?.data?.pagination || response?.data?.data?.pagination
const loadProfile = async () => {
  const response = await dataAPI.getS7Profile(props.projectId, props.connection.id)
  profile.value = unwrapData(response) as S7Profile
  if (!profile.value?.configured) profileVisible.value = true
}
const reloadEstimate = async () => {
  const response = await dataAPI.getS7ReadPlanEstimate(props.projectId, props.connection.id, {
    groupId: selectedGroupId.value || undefined,
  })
  readPlanEstimate.value = { ...emptyEstimate(), ...unwrapData(response) }
}
const reloadAll = async () => {
  loading.value = true
  try {
    const [profileResponse, groupResponse, variableResponse] = await Promise.all([
      dataAPI.getS7Profile(props.projectId, props.connection.id),
      dataAPI.getS7VariableGroups(props.projectId, props.connection.id),
      dataAPI.getS7Variables(props.projectId, props.connection.id, {
        groupId: selectedGroupId.value || undefined,
        page: variablePagination.value.page,
        pageSize: variablePagination.value.pageSize,
      }),
    ])
    profile.value = unwrapData(profileResponse) as S7Profile
    groups.value = unwrapList<S7VariableGroup>(groupResponse)
    variables.value = unwrapList<S7Variable>(variableResponse)
    variablePagination.value = {
      ...variablePagination.value,
      ...unwrapPagination(variableResponse),
    }
    await reloadEstimate()
    if (!profile.value?.configured) profileVisible.value = true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载 S7 建模数据失败'))
  } finally {
    loading.value = false
  }
}
const selectGroup = async (groupId: string) => {
  selectedGroupId.value = groupId
  selectedVariableId.value = ''
  variablePagination.value.page = 1
  await reloadAll()
}
const changeVariablePage = async (page: number) => {
  variablePagination.value.page = page
  await reloadAll()
}
const changeVariablePageSize = async (pageSize: number) => {
  variablePagination.value.page = 1
  variablePagination.value.pageSize = pageSize
  await reloadAll()
}
const selectVariable = (variable: S7Variable) => {
  selectedVariableId.value = variable.id
}
const openCreateGroup = () => {
  editingGroup.value = null
  groupMode.value = 'create'
  groupVisible.value = true
}
const openEditGroup = (group: S7VariableGroup) => {
  editingGroup.value = group
  groupMode.value = 'edit'
  groupVisible.value = true
}
const saveProfile = async (payload: Record<string, unknown>) => {
  saving.value = true
  try {
    await dataAPI.updateS7Profile(props.projectId, props.connection.id, payload)
    profileVisible.value = false
    await loadProfile()
    await reloadEstimate()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存 PLC 档案失败'))
  } finally {
    saving.value = false
  }
}
const saveGroup = async (payload: Record<string, unknown>) => {
  saving.value = true
  try {
    if (groupMode.value === 'edit' && editingGroup.value)
      await dataAPI.updateS7VariableGroup(
        props.projectId,
        props.connection.id,
        editingGroup.value.id,
        { ...payload, hasParentId: true },
      )
    else await dataAPI.createS7VariableGroup(props.projectId, props.connection.id, payload)
    groupVisible.value = false
    await reloadAll()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存变量组失败'))
  } finally {
    saving.value = false
  }
}
const removeGroup = async (group: S7VariableGroup) => {
  await ElMessageBox.confirm(`删除变量组“${group.name}”？组内变量会移动到未分组。`, '删除变量组')
  await dataAPI.deleteS7VariableGroup(props.projectId, props.connection.id, group.id)
  if (selectedGroupId.value === group.id) selectedGroupId.value = ''
  await reloadAll()
}
const openCreateVariable = () => {
  editingVariable.value = null
  variableMode.value = 'create'
  variableVisible.value = true
}
const openDuplicateVariable = (variable: S7Variable) => {
  editingVariable.value = {
    ...variable,
    id: '',
    name: `${variable.name} 副本`,
    code: `${variable.code}_copy`,
    datapointId: null,
    datapointPath: null,
    datapointStatus: null,
    lastValue: undefined,
    quality: undefined,
    lastUpdatedAt: null,
  }
  variableMode.value = 'create'
  variableVisible.value = true
}
const openEditVariable = (variable: S7Variable) => {
  editingVariable.value = variable
  variableMode.value = 'edit'
  variableVisible.value = true
}
const saveVariable = async (payload: Record<string, unknown>) => {
  saving.value = true
  try {
    const saved =
      variableMode.value === 'edit' && editingVariable.value
        ? await dataAPI.updateS7Variable(
            props.projectId,
            props.connection.id,
            editingVariable.value.id,
            payload,
          )
        : await dataAPI.createS7Variable(props.projectId, props.connection.id, payload)
    variableVisible.value = false
    await reloadAll()
    selectedVariableId.value = saved?.data?.id || saved?.data?.data?.id || ''
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存变量失败'))
  } finally {
    saving.value = false
  }
}
const removeVariable = async (variable: S7Variable) => {
  await ElMessageBox.confirm(`删除变量“${variable.name}”？对应数据点将标记为失效。`, '删除变量')
  await dataAPI.deleteS7Variable(props.projectId, props.connection.id, variable.id)
  if (selectedVariableId.value === variable.id) selectedVariableId.value = ''
  await reloadAll()
}
const importVariables = async (rows: Array<Record<string, unknown>>) => {
  saving.value = true
  try {
    await dataAPI.batchImportS7Variables(props.projectId, props.connection.id, {
      groupId: selectedGroupId.value || null,
      variables: rows,
    })
    importVisible.value = false
    await reloadAll()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '导入变量失败'))
  } finally {
    saving.value = false
  }
}
const exportVariables = (format: string | number | object) => {
  if (variableExportRows.value.length === 0) {
    ElMessage.warning('当前页没有可导出的变量')
    return
  }
  const suffix = format === 'xlsx' ? 'xlsx' : 'csv'
  const filename = `s7-variables-current-page.${suffix}`
  if (suffix === 'xlsx') {
    downloadXlsx(filename, [
      { name: '变量清单', headers: variableExportHeaders, rows: variableExportRows.value },
      { name: '问题清单', headers: issueExportHeaders, rows: issueExportRows.value },
    ])
    return
  }
  downloadCsv(filename, variableExportHeaders, variableExportRows.value)
}
const toggleSession = () => {
  if (session.connected.value) void session.disconnect()
  else void session.connect()
}
const applyReadValues = (values: S7ReadValue[]) => {
  const valueMap = new Map(values.map((item) => [item.variableId, item]))
  variables.value = variables.value.map((variable) => {
    const next = valueMap.get(variable.id)
    return next
      ? { ...variable, lastValue: next.value, quality: next.quality, lastUpdatedAt: next.timestamp }
      : variable
  })
}
const readCurrentScope = async () => {
  if (!session.connected.value || !session.sessionId.value)
    return ElMessage.warning('请先连接 S7 开发态会话')
  const response = await dataAPI.readS7DevSession(
    props.projectId,
    props.connection.id,
    session.sessionId.value,
    { groupId: selectedGroupId.value || null },
  )
  const values = unwrapData(response).values || []
  applyReadValues(values)
}
const openPreview = async () => {
  if (!session.connected.value || !session.sessionId.value)
    return ElMessage.warning('请先连接 S7 开发态会话')
  const response = await dataAPI.pollS7DevSession(
    props.projectId,
    props.connection.id,
    session.sessionId.value,
    { groupId: selectedGroupId.value || null },
  )
  const data = unwrapData(response)
  previewValues.value = data.values || []
  previewDiagnostics.value = data.diagnostics || []
  applyReadValues(previewValues.value)
  previewVisible.value = true
}
const runValidation = async () => {
  const response = await dataAPI.validateS7Model(props.projectId, props.connection.id)
  validationIssues.value = unwrapData(response).issues || []
  validationVisible.value = true
}
const openReadPlan = async () => {
  await reloadEstimate()
  readPlanVisible.value = true
}
const locateVariable = (variableId: string) => {
  selectedVariableId.value = variableId
  selectedGroupId.value = variables.value.find((item) => item.id === variableId)?.groupId || ''
  validationVisible.value = false
}

const openVariableMenu = (event: MouseEvent, variable: S7Variable) => {
  event.preventDefault()
  selectedVariableId.value = variable.id
  variableMenu.value = {
    visible: true,
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 210),
    variable,
  }
}

const closeVariableMenu = () => {
  variableMenu.value.visible = false
}

const runVariableMenuAction = async (
  action: 'edit' | 'duplicate' | 'copy-info' | 'copy-path' | 'datapoint' | 'delete',
) => {
  const variable = variableMenu.value.variable
  closeVariableMenu()
  if (!variable) return
  if (action === 'edit') {
    openEditVariable(variable)
    return
  }
  if (action === 'duplicate') {
    openDuplicateVariable(variable)
    return
  }
  if (action === 'copy-info') {
    await copyText(
      [
        `name=${variable.name}`,
        `code=${variable.code}`,
        `address=${variable.normalizedAddress || variable.addressText}`,
        `type=${variable.dataType}`,
        `datapoint=${variable.datapointPath || ''}`,
      ].join('\n'),
      '变量信息已复制',
    )
    return
  }
  if (action === 'copy-path') {
    if (!variable.datapointPath) {
      ElMessage.warning('当前变量还没有数据点 path')
      return
    }
    await copyText(variable.datapointPath, '数据点 path 已复制')
    return
  }
  if (action === 'datapoint') {
    ElMessage.info(
      variable.datapointPath ? `数据点：${variable.datapointPath}` : '当前变量还没有数据点',
    )
    return
  }
  await removeVariable(variable)
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

function matchQuickFilter(variable: S7Variable) {
  if (quickFilter.value === 'issue')
    return validationIssues.value.some((issue) => issue.variableId === variable.id)
  if (quickFilter.value === 'datapoint')
    return !variable.datapointPath || ['invalid', 'error'].includes(variable.datapointStatus || '')
  if (quickFilter.value === 'writable')
    return variable.area === 'Q' || variable.area === 'M' || variable.area === 'DB'
  if (quickFilter.value === 'disabled') return variable.status !== 'active'
  return true
}

async function copyText(text: string, successMessage: string) {
  await navigator.clipboard.writeText(text)
  ElMessage.success(successMessage)
}

function issueSummaryForVariable(variableId: string) {
  return validationIssues.value
    .filter((issue) => issue.variableId === variableId)
    .map((issue) => `${issue.severity}:${issue.message}`)
    .join('; ')
}

function formatExportValue(value: unknown) {
  if (value === null || value === undefined || value === '') return ''
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

onMounted(reloadAll)
</script>

<style scoped>
.s7-workbench {
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
.s7-workbench__side {
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
  overflow: hidden;
}
.s7-workbench__side :deep(.s7-group-tree) {
  flex: 1;
  min-height: 0;
}
.s7-workbench__connect-action {
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
}
.s7-workbench__connect-action::before {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: currentColor;
  content: '';
}
.s7-workbench__connect-action.is-connected {
  border-color: color-mix(in oklch, var(--dc-success) 32%, var(--dc-border));
  background: color-mix(in oklch, var(--dc-success) 12%, var(--dc-surface-raised));
  color: var(--dc-success);
}
.s7-workbench__main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
  overflow: hidden;
}
.s7-workbench__bar {
  min-height: 46px;
  padding: 7px 10px 7px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
  display: grid;
  grid-template-columns: auto minmax(180px, 260px);
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.s7-workbench__actions {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.s7-workbench__icon-action {
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
.s7-workbench__icon-action:hover:not(:disabled) {
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
  color: var(--dc-primary);
}
.s7-workbench__icon-action:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}
.s7-workbench__icon-action.is-primary {
  border-color: var(--dc-primary);
  background: var(--dc-primary);
  color: var(--dc-surface-raised);
}
.s7-workbench__icon-action svg {
  width: 15px;
  height: 15px;
}
.s7-workbench__search {
  min-width: 0;
}
.s7-workbench__filters {
  min-height: 36px;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}
.s7-workbench__filters button {
  height: 24px;
  padding: 0 9px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.s7-workbench__filters button.is-active,
.s7-workbench__filters button:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 32%, var(--dc-border));
  color: var(--dc-primary);
}
.s7-workbench__menu-mask {
  position: fixed;
  inset: 0;
  z-index: 2100;
}
.s7-workbench__context-menu {
  position: fixed;
  min-width: 154px;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}
.s7-workbench__context-menu button {
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
.s7-workbench__context-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}
.s7-workbench__context-menu button.is-danger:hover {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}
.s7-workbench__menu-icon {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}
@media (max-width: 1100px) {
  .s7-workbench {
    grid-template-columns: 230px minmax(0, 1fr);
  }
  .s7-workbench :deep(.s7-inspector) {
    display: none;
  }
}
</style>
