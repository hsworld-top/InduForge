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
            <span v-if="session.connected.value" class="modbus-workbench__connect-label is-hover">断开</span>
          </button>
        </template>
      </WorkbenchSourceHeader>
      <ModbusGroupTree
        :groups="groups"
        :registers="registers"
        :selected-group-id="selectedGroupId"
        @select="selectGroup"
        @create="openCreateGroup"
        @edit="openEditGroup"
        @delete="removeGroup"
      />
    </aside>

      <main class="modbus-workbench__main">
        <div class="modbus-workbench__bar">
          <div class="modbus-workbench__title">
            <div>
              <strong>{{ currentGroup?.name || '全部变量' }}</strong>
              <span>{{ filteredRegisters.length }} 个变量 · 自动同步 modbus.register 数据点</span>
            </div>
            <div class="modbus-workbench__metrics">
              <span><b>{{ readPlanEstimate.unitCount }}</b>从站</span>
              <span><b>{{ readPlanEstimate.readCount }}</b>读取</span>
              <span><b>{{ readPlanEstimate.readsPerSecond.toFixed(2) }}</b>reads/s</span>
            </div>
          </div>
          <div class="modbus-workbench__actions">
            <button type="button" class="modbus-workbench__icon-action" :title="importActionTitle" :aria-label="importActionTitle" @click="importVisible = true">
              <IconTablerUpload />
            </button>
            <button type="button" class="modbus-workbench__icon-action is-primary" title="新建变量" aria-label="新建变量" @click="openCreateRegister">
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
            <button type="button" class="modbus-workbench__icon-action" :title="validationActionTitle" :aria-label="validationActionTitle" @click="runValidation">
              <IconTablerChecklist />
            </button>
            <button type="button" class="modbus-workbench__icon-action" :title="readPlanActionTitle" :aria-label="readPlanActionTitle" @click="openReadPlan">
              <IconTablerRoute />
            </button>
            <button type="button" class="modbus-workbench__icon-action" title="刷新" aria-label="刷新" @click="reloadAll">
              <IconTablerRefresh />
            </button>
          </div>
          <el-input v-model="registerKeyword" class="modbus-workbench__search" size="small" placeholder="搜索变量" clearable />
        </div>
        <ModbusRegisterTable
          :registers="filteredRegisters"
          :loading="loading"
          :selected-register-id="selectedRegisterId"
          @select="selectRegister"
          @edit="openEditRegister"
          @delete="removeRegister"
        />
      </main>

      <ModbusInspectorPanel
        :group="currentGroup"
        :register="selectedRegister"
        :registers="registers"
        :issues="scopedValidationIssues"
        :estimate="readPlanEstimate"
      />

    <ModbusGroupDialog
      v-model="groupDialogVisible"
      :mode="groupDialogMode"
      :groups="groups"
      :group-value="editingGroup"
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
    <ModbusPreviewDialog v-model="previewVisible" :registers="previewRegisters" :diagnostics="previewDiagnostics" />
    <ModbusValidationDrawer
      v-model="validationVisible"
      :issues="scopedValidationIssues"
      :scope-label="validationScopeLabel"
      @locate="locateRegister"
    />
    <ModbusReadPlanDialog v-model="readPlanVisible" :estimate="readPlanEstimate" />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'
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
import type {
  ModbusReadPlanEstimate,
  ModbusReadValue,
  ModbusRegister,
  ModbusRegisterGroup,
  ModbusValidationIssue,
} from '@/components/modbus/types'
import IconTablerActivityHeartbeat from '~icons/tabler/activity-heartbeat'
import IconTablerChecklist from '~icons/tabler/checklist'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerRoute from '~icons/tabler/route'
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
const loading = ref(false)
const saving = ref(false)
const selectedGroupId = ref('')
const selectedRegisterId = ref('')
const registerKeyword = ref('')
const groupDialogVisible = ref(false)
const groupDialogMode = ref<'create' | 'edit'>('create')
const editingGroup = ref<ModbusRegisterGroup | null>(null)
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

const session = useProtocolDevSession({
  create: () => dataAPI.createModbusDevSession(props.projectId, props.connection.id),
  close: (sessionId: string) => dataAPI.closeModbusDevSession(props.projectId, props.connection.id, sessionId),
})

const config = computed(() => props.connection.config || {})
const endpointText = computed(() => {
  if (config.value.mode === 'rtu') return String(config.value.serialConfig?.port || 'RTU 串口未配置')
  return `${config.value.host || '未配置 host'}:${config.value.port || 502}`
})
const sourceMetaRows = computed(() => [
  { label: '类型', value: 'Modbus' },
  { label: '端点', value: endpointText.value },
  { label: '默认从站', value: String(config.value.slaveId ?? 1) },
])
const currentGroup = computed(() => groups.value.find((group) => group.id === selectedGroupId.value) || null)
const selectedRegister = computed(() => registers.value.find((item) => item.id === selectedRegisterId.value) || null)
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
    const ids = new Set(registers.value.filter((item) => item.groupId === selectedGroupId.value).map((item) => item.id))
    return validationIssues.value.filter((item) => item.registerId && ids.has(item.registerId))
  }
  return validationIssues.value
})
const filteredRegisters = computed(() => {
  const text = registerKeyword.value.trim().toLowerCase()
  return registers.value.filter((item) => {
    const inGroup = !selectedGroupId.value || item.groupId === selectedGroupId.value
    const matched =
      !text ||
      [item.name, item.code, item.datapointPath || '', item.area, item.address].some((value) =>
        String(value || '').toLowerCase().includes(text),
      )
    return inGroup && matched
  })
})

const unwrapList = <T,>(response: any): T[] => response?.data?.list || response?.data?.data?.list || []
const unwrapData = (response: any) => response?.data?.data || response?.data || {}

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
      dataAPI.getModbusRegisters(props.projectId, props.connection.id),
    ])
    groups.value = unwrapList<ModbusRegisterGroup>(groupResponse)
    registers.value = unwrapList<ModbusRegister>(registerResponse)
    await reloadEstimate()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载 Modbus 建模数据失败'))
  } finally {
    loading.value = false
  }
}

const selectGroup = async (groupId: string) => {
  selectedGroupId.value = groupId
  selectedRegisterId.value = ''
  await reloadEstimate()
}

const selectRegister = (register: ModbusRegister) => {
  selectedRegisterId.value = register.id
}

const openCreateGroup = () => {
  editingGroup.value = null
  groupDialogMode.value = 'create'
  groupDialogVisible.value = true
}

const openEditGroup = (group: ModbusRegisterGroup) => {
  editingGroup.value = group
  groupDialogMode.value = 'edit'
  groupDialogVisible.value = true
}

const saveGroup = async (payload: Record<string, unknown>) => {
  saving.value = true
  try {
    if (groupDialogMode.value === 'edit' && editingGroup.value) {
      await dataAPI.updateModbusRegisterGroup(props.projectId, props.connection.id, editingGroup.value.id, {
        ...payload,
        hasParentId: true,
      })
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
  await ElMessageBox.confirm(`删除寄存器组“${group.name}”？组内变量会移动到未分组。`, '删除寄存器组')
  await dataAPI.deleteModbusRegisterGroup(props.projectId, props.connection.id, group.id)
  if (selectedGroupId.value === group.id) selectedGroupId.value = ''
  await reloadAll()
}

const openCreateRegister = () => {
  editingRegister.value = null
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
        ? await dataAPI.updateModbusRegister(props.projectId, props.connection.id, editingRegister.value.id, payload)
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
    const response = await dataAPI.pollModbusDevSession(props.projectId, props.connection.id, session.sessionId.value, {
      groupId: selectedGroupId.value || null,
    })
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

onMounted(reloadAll)
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
  grid-template-columns: minmax(0, 1fr) auto 240px;
  align-items: center;
  gap: 12px;
}

.modbus-workbench__title {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 12px;
}

.modbus-workbench__title > div:first-child {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.modbus-workbench__bar strong {
  color: var(--dc-text);
  font-size: 14px;
  line-height: 18px;
}

.modbus-workbench__bar span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.modbus-workbench__metrics {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.modbus-workbench__metrics span {
  height: 26px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font-size: 11px;
  font-weight: 700;
}

.modbus-workbench__metrics b {
  color: var(--dc-primary);
  font-size: 12px;
}

.modbus-workbench__actions {
  display: flex;
  align-items: center;
  gap: 6px;
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

@media (max-width: 1100px) {
  .modbus-workbench {
    grid-template-columns: 230px minmax(0, 1fr);
  }

  .modbus-workbench :deep(.modbus-inspector) {
    display: none;
  }
}
</style>
