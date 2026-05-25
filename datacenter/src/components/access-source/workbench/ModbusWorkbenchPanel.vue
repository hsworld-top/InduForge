<template>
  <section class="modbus-workbench">
    <header class="modbus-workbench__header">
      <WorkbenchSourceHeader
        :title="connection.name || '未命名 Modbus 接入源'"
        fallback-title="未命名 Modbus 接入源"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #actions>
          <el-button size="small" :icon="IconTablerPlugConnected" :loading="testing" @click="runConnectionTest">
            测试连接
          </el-button>
          <button type="button" class="workbench-source-header__icon-action" title="导入变量" aria-label="导入变量" @click="importVisible = true">
            <IconTablerUpload />
          </button>
          <button type="button" class="workbench-source-header__icon-action is-primary" title="新建变量" aria-label="新建变量" @click="openCreateRegister">
            <IconTablerPlus />
          </button>
          <button type="button" class="workbench-source-header__icon-action" title="变量预览" aria-label="变量预览" @click="openPreview">
            <IconTablerActivityHeartbeat />
          </button>
          <button type="button" class="workbench-source-header__icon-action" title="建模校验" aria-label="建模校验" @click="runValidation">
            <IconTablerChecklist />
          </button>
          <button type="button" class="workbench-source-header__icon-action" title="运行态读取预估" aria-label="运行态读取预估" @click="openReadPlan">
            <IconTablerRoute />
          </button>
          <button type="button" class="workbench-source-header__icon-action" title="刷新" aria-label="刷新" @click="reloadAll">
            <IconTablerRefresh />
          </button>
        </template>
      </WorkbenchSourceHeader>
    </header>

    <div class="modbus-workbench__body">
      <ModbusGroupTree
        :groups="groups"
        :registers="registers"
        :selected-group-id="selectedGroupId"
        @select="selectGroup"
        @create="openCreateGroup"
        @edit="openEditGroup"
        @delete="removeGroup"
      />

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
        :issues="validationIssues"
        :estimate="readPlanEstimate"
      />
    </div>

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
    <ModbusValidationDrawer v-model="validationVisible" :issues="validationIssues" @locate="locateRegister" />
    <ModbusReadPlanDialog v-model="readPlanVisible" :estimate="readPlanEstimate" />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'
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
  ModbusRegister,
  ModbusRegisterGroup,
  ModbusValidationIssue,
} from '@/components/modbus/types'
import IconTablerActivityHeartbeat from '~icons/tabler/activity-heartbeat'
import IconTablerChecklist from '~icons/tabler/checklist'
import IconTablerPlugConnected from '~icons/tabler/plug-connected'
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
const testing = ref(false)
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
const previewRegisters = ref<ModbusRegister[]>([])
const previewDiagnostics = ref<string[]>([])
const validationVisible = ref(false)
const validationIssues = ref<ModbusValidationIssue[]>([])
const readPlanVisible = ref(false)

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

const runConnectionTest = async () => {
  testing.value = true
  try {
    await dataAPI.testConnection(props.projectId, {
      type: 'modbus',
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
    const response = await dataAPI.previewModbusRegisters(props.projectId, props.connection.id, {
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
  grid-template-rows: auto minmax(0, 1fr);
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
  overflow: hidden;
}

.modbus-workbench__header {
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.modbus-workbench__body {
  min-height: 0;
  display: grid;
  grid-template-columns: 264px minmax(0, 1fr) 292px;
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
  min-height: 52px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--dc-border);
  background:
    linear-gradient(90deg, color-mix(in oklch, var(--dc-surface-subtle) 84%, var(--dc-primary) 16%), var(--dc-surface-subtle)),
    var(--dc-surface-subtle);
  display: grid;
  grid-template-columns: minmax(0, 1fr) 240px;
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

.modbus-workbench__search {
  min-width: 0;
}

@media (max-width: 1100px) {
  .modbus-workbench__body {
    grid-template-columns: 230px minmax(0, 1fr);
  }

  .modbus-workbench__body :deep(.modbus-inspector) {
    display: none;
  }
}
</style>
