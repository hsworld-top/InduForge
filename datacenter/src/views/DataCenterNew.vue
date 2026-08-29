<template>
  <DataCenterShell v-model:active-module="navActiveModule" :modules="localizedModules">
    <template #actions>
      <div class="datacenter-side-actions">
        <button
          type="button"
          class="datacenter-side-action"
          :title="t('shell.contractCheck')"
          @click="showDataContractCheckDialog = true"
        >
          <IconTablerFileCheck class="h-5 w-5" /><span>{{ t('shell.contractCheck') }}</span>
        </button>
        <button
          type="button"
          class="datacenter-side-action"
          :class="{ 'is-primary': showCollectorAgentManager }"
          :title="t('shell.collectorAgent')"
          @click="showCollectorAgentManager = true"
        >
          <IconTablerDeviceDesktopCog class="h-5 w-5" /><span>{{ t('shell.collectorAgent') }}</span>
        </button>
      </div>
    </template>

    <CollectorAgentManager
      v-if="showCollectorAgentManager"
      @close="showCollectorAgentManager = false"
    />
    <DataPointWorkspace
      v-else-if="activeModule === 'datapoint' && projectId"
      :project-id="projectId"
    />
    <IndustrialCollectorWorkbench
      v-else-if="activeModule === 'industrial-collector' && projectId"
      :project-id="projectId"
      @navigate="handleCollectorNavigate"
    />
    <template v-else-if="activeModule === 'access-source'">
      <AccessSourceWorkbench
        v-if="activeWorkbenchConnection && projectId"
        :connection="activeWorkbenchConnection"
        :project-id="projectId"
        @back="handleWorkbenchBack"
        @update-connection="loadConnections"
      />
      <AccessSourceWorkspace
        v-else
        :connections="connections"
        :selected-connection-id="selectedConnectionId"
        :project-id="projectId"
        :loading="connectionLoading"
        :pagination="connectionPagination"
        @create="openCreateConnectionDialog"
        @refresh="loadConnections"
        @edit="handleEditConnection"
        @query-change="setConnectionQuery"
      />
    </template>
    <HistoryStorageWorkspace
      v-else-if="activeModule === 'history-storage' && projectId"
      :project-id="projectId"
      class="h-full overflow-hidden"
    />
    <ComputeWorkspace
      v-else-if="activeModule === 'compute' && projectId"
      :project-id="projectId"
      :selected-unit-id="selectedComputeUnitId"
      class="h-full overflow-hidden"
    />
    <AlarmWorkspace
      v-else-if="activeModule === 'alarm' && projectId"
      :project-id="projectId"
      :selected-item-id="selectedAlarmItemId"
      class="h-full overflow-hidden"
      @close-selected-item="handleCloseAlarmItem"
    />
  </DataCenterShell>

  <DataContractCheckDialog v-model="showDataContractCheckDialog" :project-id="projectId" />
  <ConnectionDialog
    ref="connectionDialogRef"
    v-model="showConnectionDialog"
    :mode="connectionDialogMode"
    :connection="currentConnection"
    :project-id="projectId"
    @submit="handleConnectionSubmit"
  />
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import IconTablerDeviceDesktopCog from '~icons/tabler/device-desktop-cog'
import IconTablerFileCheck from '~icons/tabler/file-check'
import AccessSourceWorkspace from '@/components/access-source/AccessSourceWorkspace.vue'
import AccessSourceWorkbench from '@/components/access-source/AccessSourceWorkbench.vue'
import AlarmWorkspace from '@/components/alarm/AlarmWorkspace.vue'
import CollectorAgentManager from '@/components/collector/CollectorAgentManager.vue'
import IndustrialCollectorWorkbench from '@/components/collector-workbench/IndustrialCollectorWorkbench.vue'
import ComputeWorkspace from '@/components/compute/ComputeWorkspace.vue'
import DataContractCheckDialog from '@/components/contract/DataContractCheckDialog.vue'
import DataPointWorkspace from '@/components/datapoint/DataPointWorkspace.vue'
import ConnectionDialog from '@/components/dialogs/ConnectionDialog.vue'
import DataCenterShell from '@/components/layout/DataCenterShell.vue'
import HistoryStorageWorkspace from '@/views/history-storage/HistoryStorageWorkspace.vue'
import dataAPI from '@/api/data.api'
import { getConnection, testSavedConnection } from '@/api/connection.api'
import { useConfirm } from '@/composables/useConfirm'
import { useConnection } from '@/composables/useConnection'
import { datacenterModules, type DatacenterModuleId } from '@/config/datacenterModules'
import { DEFAULT_MODULE, resolveDatacenterRouteBase, type V2ModuleId } from '@/router/route-config'
import { useProjectStore } from '@/stores/project.store'
import { getApiErrorMessage } from '@/utils/request'
import { t } from '@/i18n/runtime'

type ConnectionRecord = Record<string, any> & {
  id: string
  type: string
  name: string
  enabled?: boolean
}

const route = useRoute()
const router = useRouter()
const projectStore = useProjectStore()
const VALID_MODULES = new Set<V2ModuleId>([
  'datapoint',
  'access-source',
  'industrial-collector',
  'history-storage',
  'compute',
  'alarm',
])
const moduleMessageKeys: Record<DatacenterModuleId, string> = {
  datapoint: 'datapoint',
  'access-source': 'accessSource',
  'industrial-collector': 'industrialCollector',
  'history-storage': 'historyStorage',
  compute: 'compute',
  alarm: 'alarm',
}
const localizedModules = computed(() =>
  datacenterModules.map((module) => {
    const key = moduleMessageKeys[module.id]
    return {
      ...module,
      label: t(`modules.${key}.label`),
      description: t(`modules.${key}.description`),
    }
  }),
)
const resolveModuleFromRoute = (): DatacenterModuleId => {
  const module = route.params.module as string | undefined
  return module && VALID_MODULES.has(module as V2ModuleId)
    ? (module as DatacenterModuleId)
    : DEFAULT_MODULE
}

const activeModule = ref<DatacenterModuleId>(resolveModuleFromRoute())
const showCollectorAgentManager = ref(false)
const navActiveModule = computed<DatacenterModuleId | null>({
  get: () => (showCollectorAgentManager.value ? null : activeModule.value),
  set: (module) => {
    if (!module) return
    void requestModuleChange(module)
  },
})
const activeObjectId = computed(() => String(route.params.objectId || ''))
const selectedComputeUnitId = computed(() =>
  route.params.module === 'compute' ? activeObjectId.value : '',
)
const selectedAlarmItemId = computed(() =>
  route.params.module === 'alarm' ? activeObjectId.value : '',
)
const project = computed(() => {
  const id = route.query.pid || route.query.id || route.meta?.project?.id
  const tenantId = route.query.tenant || route.meta?.project?.tenantId
  return id ? { id: String(id), tenantId: tenantId ? String(tenantId) : undefined } : null
})
const projectId = computed(() => project.value?.id || '')
provide('projectId', projectId)

const {
  connections,
  selectedConnectionId,
  loading: connectionLoading,
  connectionPagination,
  loadConnections,
  setConnectionQuery,
  resetConnections,
  createConnection,
  updateConnection,
} = useConnection(projectId, { paginated: true })
const showDataContractCheckDialog = ref(false)
const showConnectionDialog = ref(false)
const connectionDialogMode = ref<'create' | 'edit'>('create')
const currentConnection = ref<ConnectionRecord | null>(null)
const connectionDialogRef = ref<InstanceType<typeof ConnectionDialog> | null>(null)
const activeWorkbenchConnection = ref<ConnectionRecord | null>(null)
let workbenchConnectionLoadVersion = 0

const loadConnectionWorkspace = () =>
  setConnectionQuery({
    page: 1,
    pageSize: connectionPagination.value.pageSize,
    search: String(route.query.q || ''),
    typeGroup: String(route.query.type || 'all'),
  })
const handleCloseAlarmItem = () => {
  const params = { ...route.params }
  delete params.objectId
  delete params.tab
  void router.replace({ name: route.name || 'datacenter', params, query: route.query })
}
const handleCollectorNavigate = (payload: { module: 'datapoint'; objectId: string }) => {
  void router.push({
    path: `${resolveDatacenterRouteBase(route.path)}/${payload.module}/${payload.objectId}`,
    query: route.query,
  })
}

watch(
  () => route.params.module,
  (module) => {
    const next = (
      module && VALID_MODULES.has(module as V2ModuleId) ? module : DEFAULT_MODULE
    ) as DatacenterModuleId
    if (activeModule.value !== next) activeModule.value = next
  },
)
watch(activeModule, (newModule, oldModule) => {
  showCollectorAgentManager.value = false
  if (newModule === oldModule || route.params.module === newModule) return
  void router.replace({
    path: `${resolveDatacenterRouteBase(route.path)}/${newModule}`,
    query: route.query,
  })
})
watch(
  [
    () => route.params.module,
    () => route.params.tab,
    () => route.params.objectId,
    () => projectId.value,
    () => connections.value.map((connection) => connection.id).join(','),
  ],
  async ([module, tab, objectId, currentProjectId]) => {
    const loadVersion = ++workbenchConnectionLoadVersion
    if (module !== 'access-source' || tab !== 'workbench' || !objectId || !currentProjectId) {
      activeWorkbenchConnection.value = null
      return
    }
    const connectionId = String(objectId)
    const currentPageConnection = connections.value.find(
      (connection) => String(connection.id) === connectionId,
    ) as ConnectionRecord | undefined
    if (currentPageConnection) {
      activeWorkbenchConnection.value = currentPageConnection
      return
    }
    try {
      const response = (await getConnection(
        String(currentProjectId),
        connectionId,
      )) as ConnectionRecord
      if (loadVersion === workbenchConnectionLoadVersion) activeWorkbenchConnection.value = response
    } catch (error) {
      if (loadVersion === workbenchConnectionLoadVersion) {
        activeWorkbenchConnection.value = null
        ElMessage.error(getApiErrorMessage(error, t('shell.loadSourceDetailFailed')))
      }
    }
  },
  { immediate: true },
)

const handleWorkbenchBack = () =>
  void router.push({
    path: `${resolveDatacenterRouteBase(route.path)}/access-source`,
    query: route.query,
  })
const openCreateConnectionDialog = () => {
  connectionDialogMode.value = 'create'
  currentConnection.value = null
  showConnectionDialog.value = true
}
const handleEditConnection = (connection: ConnectionRecord) => {
  connectionDialogMode.value = 'edit'
  currentConnection.value = connection
  showConnectionDialog.value = true
}

const protocolCreateHandlers: Record<string, (projectId: string, data: any) => Promise<unknown>> = {
  mqtt: dataAPI.createMqttConnection,
  kafka: dataAPI.createKafkaConfig,
  http: dataAPI.createHttpConfig,
  websocket: dataAPI.createWebSocketConfig,
  redis: dataAPI.createRedisConfig,
  tdengine: dataAPI.createTdengineConfig,
}
const protocolUpdateHandlers: Record<
  string,
  (projectId: string, connectionId: string, data: any) => Promise<unknown>
> = {
  mqtt: dataAPI.updateMqttConnection,
  kafka: dataAPI.updateKafkaConfig,
  http: dataAPI.updateHttpConfig,
  websocket: dataAPI.updateWebSocketConfig,
  redis: dataAPI.updateRedisConfig,
  tdengine: dataAPI.updateTdengineConfig,
}
const handleConnectionSubmit = async (data: {
  name: string
  type: string
  config: any
  tested?: boolean
}) => {
  if (!projectId.value) return
  try {
    if (connectionDialogMode.value === 'create') {
      const createProtocol = protocolCreateHandlers[data.type]
      if (createProtocol) {
        const response: any = await createProtocol(projectId.value, {
          name: data.name,
          enabled: true,
          ...data.config,
        })
        const connectionId = String(response?.data?.id || response?.id || '')
        if (data.tested && connectionId) {
          await testSavedConnection(projectId.value, connectionId)
        }
        await loadConnections()
        ElMessage.success(t('shell.sourceCreateSuccess'))
      } else await createConnection({ name: data.name, type: data.type, config: data.config })
    } else if (currentConnection.value) {
      const updateProtocol = protocolUpdateHandlers[data.type]
      if (updateProtocol) {
        await updateProtocol(projectId.value, currentConnection.value.id, {
          name: data.name,
          enabled: currentConnection.value.enabled ?? true,
          ...data.config,
        })
        await loadConnections()
        ElMessage.success(t('shell.sourceUpdateSuccess'))
      } else await updateConnection(currentConnection.value.id, data)
    }
    connectionDialogRef.value?.closeSilently?.()
    showConnectionDialog.value = false
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('shell.sourceSaveFailed')))
  }
}

type DraftGuard = {
  isDirty: () => boolean
  save?: () => Promise<boolean>
  discard?: () => void
}
const { confirmSaveDraftAction } = useConfirm()
const draftGuards = ref<DraftGuard[]>([])
provide('registerDraftChecker', (checker: (() => boolean) | DraftGuard) => {
  const guard = typeof checker === 'function' ? { isDirty: checker } : checker
  draftGuards.value.push(guard)
  return () => {
    const index = draftGuards.value.indexOf(guard)
    if (index !== -1) draftGuards.value.splice(index, 1)
  }
})
const pendingDraftGuards = () => draftGuards.value.filter((guard) => guard.isDirty())
const hasPendingDraft = () => pendingDraftGuards().length > 0

async function resolvePendingDrafts() {
  const guards = pendingDraftGuards()
  if (!guards.length) return true
  const action = await confirmSaveDraftAction()
  if (action === 'cancel') return false
  if (action === 'discard') {
    guards.forEach((guard) => guard.discard?.())
    return true
  }
  for (const guard of guards) {
    if (!guard.save || !(await guard.save())) return false
  }
  return true
}

async function requestModuleChange(module: DatacenterModuleId) {
  if (module === activeModule.value) return
  if (!(await resolvePendingDrafts())) return
  showCollectorAgentManager.value = false
  activeModule.value = module
}

onBeforeRouteLeave(async (_to, _from, next) => {
  next(await resolvePendingDrafts())
})
const handleBeforeUnload = (event: BeforeUnloadEvent) => {
  if (!hasPendingDraft()) return
  event.preventDefault()
  event.returnValue = ''
}
const syncProjectStore = () => {
  if (project.value?.id) projectStore.bootstrap(project.value)
}
onMounted(() => {
  syncProjectStore()
  if (projectId.value) void loadConnectionWorkspace()
  window.addEventListener('beforeunload', handleBeforeUnload)
})
watch(
  () => project.value?.id,
  (newId, oldId) => {
    if (!newId) return resetConnections()
    syncProjectStore()
    if (newId !== oldId) {
      resetConnections()
      void loadConnectionWorkspace()
    }
  },
)
onBeforeUnmount(() => window.removeEventListener('beforeunload', handleBeforeUnload))
</script>

<style scoped>
.datacenter-side-actions {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.datacenter-side-action {
  position: relative;
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  transition:
    background-color 0.18s ease,
    border-color 0.18s ease,
    color 0.18s ease,
    transform 0.18s ease;
}
.datacenter-side-action:hover {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
  transform: translateY(-1px);
}
.datacenter-side-action.is-primary {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.datacenter-side-action span {
  position: absolute;
  z-index: 10;
  left: calc(100% + 8px);
  top: 50%;
  max-width: 140px;
  padding: 6px 9px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-popover);
  color: var(--dc-text);
  font-size: 12px;
  font-weight: 600;
  line-height: 1;
  opacity: 0;
  pointer-events: none;
  transform: translate(4px, -50%);
  transition:
    opacity 0.16s ease,
    transform 0.16s ease;
  white-space: nowrap;
}
.datacenter-side-action:hover span,
.datacenter-side-action:focus-visible span {
  opacity: 1;
  transform: translate(0, -50%);
}
</style>
