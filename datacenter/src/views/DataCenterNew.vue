<template>
  <DataCenterShell v-model:active-module="activeModule" :modules="datacenterModules">
    <template #actions>
      <div class="datacenter-side-actions">
        <button
          type="button"
          class="datacenter-side-action"
          title="数据契约检查"
          aria-label="数据契约检查"
          @click="showDataContractCheckDialog = true"
        >
          <IconTablerShieldCheck class="h-5 w-5" />
          <span>数据契约检查</span>
        </button>
      </div>
    </template>

    <DataPointWorkspace v-if="activeModule === 'datapoint' && projectId" :project-id="projectId" />

    <template v-else-if="activeModule === 'access-source'">
      <!-- v2 workbench 路由：tab === 'workbench' 时挂新容器 -->
      <AccessSourceWorkbench
        v-if="activeWorkbenchConnection && projectId"
        :connection="activeWorkbenchConnection"
        :project-id="String(projectId)"
        @back="handleWorkbenchBack"
        @update-connection="loadConnections"
      />

      <AccessSourceWorkspace
        v-else
        :connections="connections"
        :selected-connection-id="selectedConnectionId"
        :project-id="projectId"
        @create="openCreateConnectionDialog"
        @refresh="loadConnections"
        @edit="handleEditConnection"
      />

      <!-- 旧标签页能力入口，已被 v2 接管，保留代码备用。 -->
      <div v-if="false" class="data-center h-full flex overflow-hidden">
        <!-- 左侧连接面板 -->
        <ConnectionList
          ref="connectionListRef"
          :connections="connections"
          :selected-connection-id="selectedConnectionId"
          :project-id="projectId"
          @select="handleSelectConnection"
          @dblclick="handleConnectionDblClick"
          @contextmenu="handleConnectionContextMenu"
          @create="openCreateConnectionDialog"
          @refresh="loadConnections"
          @table-dblclick="handleTableDblClick"
          @table-contextmenu="handleTableContextMenu"
          @view-table-list="handleViewTableList"
          @query-dblclick="handleQueryDblClick"
          @query-contextmenu="handleQueryContextMenu"
          @query-deleted="handleQueryDeleted"
          @datapoint-open="handleOpenDataPointList"
          @calcunit-open="handleOpenCalcUnitList"
          @alarmunit-open="handleOpenAlarmUnitList"
        />

        <!-- 右侧内容区域 - 统一标签页系统 -->
        <div class="flex-1 flex flex-col bg-gray-50 dark:bg-gray-900 overflow-hidden">
          <template v-if="tabs.length > 0">
            <el-tabs
              ref="tabsRef"
              v-model="activeTabId"
              closable
              class="query-tabs flex-1 flex flex-col overflow-hidden"
              @tab-remove="handleCloseTab"
              @tab-click="handleTabClick"
            >
              <el-tab-pane
                v-for="tab in tabs"
                :key="tab.id"
                :name="tab.id"
                :closable="tab.closable"
                class="flex-1 flex flex-col overflow-hidden"
              >
                <template #label>
                  <span class="flex items-center">
                    <component :is="tab.icon" v-if="tab.icon" class="mr-1 w-4 h-4" />
                    <span>{{ tab.label }}</span>
                    <IconTablerAlertCircle
                      v-if="tab.modified"
                      class="ml-1 text-orange-500 w-3 h-3"
                    />
                  </span>
                </template>

                <!-- 表列表内容 -->
                <div v-if="tab.type === 'table-list'" class="flex-1 flex flex-col overflow-hidden">
                  <div
                    class="table-toolbar flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800"
                  >
                    <div class="text-sm text-gray-700 dark:text-gray-300">
                      {{ tab.connection.name }} - {{ t('tabs.tableList') }}
                    </div>
                    <div class="space-x-2">
                      <el-button
                        type="primary"
                        size="small"
                        @click="createNewQuery(tab.connection)"
                      >
                        <IconTablerPlus class="mr-1 w-4 h-4" />
                        {{ t('actions.newQuery') }}
                      </el-button>
                    </div>
                  </div>
                  <div class="flex-1 overflow-y-auto p-4 min-h-0">
                    <MysqlTableList
                      v-if="tab.connection.relationalConfig?.dbType === 'mysql'"
                      :connection-id="tab.connection.id"
                      @table-select="(tableName) => createQueryFromTable(tab.connection, tableName)"
                    />
                    <PostgresTableList
                      v-else-if="tab.connection.relationalConfig?.dbType === 'postgresql'"
                      :connection-id="tab.connection.id"
                      @table-select="(tableName) => createQueryFromTable(tab.connection, tableName)"
                    />
                    <SqlServerTableList
                      v-else-if="tab.connection.relationalConfig?.dbType === 'sqlserver'"
                      :connection-id="tab.connection.id"
                      @table-select="(tableName) => createQueryFromTable(tab.connection, tableName)"
                    />
                  </div>
                </div>

                <!-- 数据点列表 -->
                <div
                  v-else-if="tab.type === 'datapoints'"
                  class="flex-1 flex flex-col overflow-hidden"
                >
                  <DataPointList
                    :ref="(el) => setDataPointListRef(tab.id, el)"
                    :project-id="projectId"
                  />
                </div>

                <!-- 计算单元 -->
                <div
                  v-else-if="tab.type === 'calc-units'"
                  class="flex-1 flex flex-col overflow-hidden"
                >
                  <ComputeUnitPanel :project-id="projectId" />
                </div>

                <!-- 报警单元 -->
                <div
                  v-else-if="tab.type === 'alarm-units'"
                  class="flex-1 flex flex-col overflow-hidden"
                >
                  <AlarmWorkspace v-if="projectId" :project-id="projectId" />
                </div>

                <!-- SQL 查询编辑器 -->
                <div
                  v-else-if="tab.type === 'query'"
                  class="flex-1 flex flex-col overflow-y-auto p-4"
                >
                  <MysqlQueryEditor
                    v-if="tab.connection.relationalConfig?.dbType === 'mysql'"
                    :tab="tab"
                    @execute="handleQueryExecute"
                    @save="handleQuerySave"
                  />
                  <PostgresQueryEditor
                    v-else-if="tab.connection.relationalConfig?.dbType === 'postgresql'"
                    :tab="tab"
                    @execute="handleQueryExecute"
                    @save="handleQuerySave"
                  />
                  <SqlServerQueryEditor
                    v-else-if="tab.connection.relationalConfig?.dbType === 'sqlserver'"
                    :tab="tab"
                    @execute="handleQueryExecute"
                    @save="handleQuerySave"
                  />
                </div>
              </el-tab-pane>
            </el-tabs>
          </template>

          <!-- 默认提示 -->
          <div v-else class="h-full flex items-center justify-center text-gray-500">
            <div class="text-center">
              <IconTablerDatabase class="text-6xl mb-4 w-24 h-24 mx-auto" />
              <div class="text-lg">{{ t('states.emptyHint') }}</div>
            </div>
          </div>
        </div>
      </div>
    </template>

    <div v-else-if="activeModule === 'compute'" class="h-full overflow-hidden">
      <ComputeWorkspace
        v-if="projectId"
        :project-id="projectId"
        :selected-unit-id="selectedComputeUnitId"
      />
    </div>

    <div v-else-if="activeModule === 'alarm'" class="h-full overflow-hidden">
      <AlarmWorkspace v-if="projectId" :project-id="projectId" />
    </div>
  </DataCenterShell>

  <DataContractCheckDialog v-model="showDataContractCheckDialog" :project-id="projectId" />

  <!-- 连接右键菜单 -->
  <ConnectionContextMenu
    v-model:visible="showContextMenu"
    :position="contextMenuPosition"
    :connection="contextMenuConnection"
    @open="handleOpenConnection"
    @disconnect="handleDisconnectConnection"
    @view-details="handleViewConnectionDetails"
    @edit="handleEditConnection"
    @delete="handleDeleteConnection"
  />

  <!-- 表右键菜单 -->
  <TableContextMenu
    v-model:visible="showTableContextMenu"
    :position="tableContextMenuPosition"
    :connection="tableContextMenuConnection"
    :table="tableContextMenuTable"
    @view-structure="handleViewTableStructure"
    @query-table="handleQueryTable"
  />

  <!-- 查询右键菜单 -->
  <QueryContextMenu
    v-model:visible="showQueryContextMenu"
    :position="queryContextMenuPosition"
    :connection="queryContextMenuConnection"
    :query="queryContextMenuQuery"
    @view-details="handleViewQueryDetails"
    @open="handleQueryDblClick"
    @delete="handleDeleteQueryFromMenu"
  />

  <!-- 新建/编辑连接对话框 -->
  <ConnectionDialog
    v-model="showConnectionDialog"
    :mode="connectionDialogMode"
    :connection="currentConnection"
    :project-id="projectId"
    @submit="handleConnectionSubmit"
  />

  <!-- 查看连接详情对话框 -->
  <ConnectionDetailsDialog v-model="showDetailsDialog" :connection="currentConnection" />

  <!-- 表结构查看对话框 -->
  <TableStructureDialog
    v-model="showTableStructureDialog"
    :project-id="projectId"
    :connection-id="currentTableConnectionId"
    :table-name="currentTableName"
  />
</template>

<script setup lang="ts">
import { ref, computed, provide, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import IconTablerDatabase from '~icons/tabler/database'
import IconTablerTable from '~icons/tabler/table'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerAlertCircle from '~icons/tabler/alert-circle'
import IconTablerCalculator from '~icons/tabler/calculator'
import IconTablerBell from '~icons/tabler/bell'
import IconTablerShieldCheck from '~icons/tabler/shield-check'
import ConnectionList from '@/components/connection/ConnectionList.vue'
import ConnectionContextMenu from '@/components/connection/ConnectionContextMenu.vue'
import TableContextMenu from '@/components/connection/TableContextMenu.vue'
import QueryContextMenu from '@/components/connection/QueryContextMenu.vue'
import ConnectionDialog from '@/components/dialogs/ConnectionDialog.vue'
import ConnectionDetailsDialog from '@/components/dialogs/ConnectionDetailsDialog.vue'
import TableStructureDialog from '@/components/dialogs/TableStructureDialog.vue'
import MysqlTableList from '@/components/database/mysql/MysqlTableList.vue'
import MysqlQueryEditor from '@/components/database/mysql/MysqlQueryEditor.vue'
import PostgresTableList from '@/components/database/postgres/PostgresTableList.vue'
import PostgresQueryEditor from '@/components/database/postgres/PostgresQueryEditor.vue'
import SqlServerTableList from '@/components/database/sqlserver/SqlServerTableList.vue'
import SqlServerQueryEditor from '@/components/database/sqlserver/SqlServerQueryEditor.vue'
import AccessSourceWorkspace from '@/components/access-source/AccessSourceWorkspace.vue'
import AccessSourceWorkbench from '@/components/access-source/AccessSourceWorkbench.vue'
import DataPointList from '@/components/datapoint/DataPointList.vue'
import DataPointWorkspace from '@/components/datapoint/DataPointWorkspace.vue'
import AlarmWorkspace from '@/components/alarm/AlarmWorkspace.vue'
import ComputeUnitPanel from '@/components/compute/ComputeUnitPanel.vue'
import ComputeWorkspace from '@/components/compute/ComputeWorkspace.vue'
import DataContractCheckDialog from '@/components/contract/DataContractCheckDialog.vue'
import DataCenterShell from '@/components/layout/DataCenterShell.vue'
import { useConnection } from '@/composables/useConnection'
import { useConfirm } from '@/composables/useConfirm'
import { onBeforeRouteLeave } from 'vue-router'
import { datacenterModules, type DatacenterModuleId } from '@/config/datacenterModules'
import { type V2ModuleId, DEFAULT_MODULE } from '@/router/route-config'
import { datacenterLocale, t } from '@/i18n/runtime'
import { useProjectStore } from '@/stores/project.store'
import dataAPI from '@/api/data.api'
import dayjs from 'dayjs'
import { TIME_FORMAT } from '@/constants'
import { resolveDatacenterTabLabel } from '@/utils/tabTitle'
import { getApiErrorMessage } from '@/utils/request'
import { stripHandoffQuery } from '@/router/handoff-query'

// 从路由获取 project 信息
const route = useRoute()
const router = useRouter()
const projectStore = useProjectStore()

// ---- 路由同步：从 URL 读 module/objectId/tab ----
// 有效的 v2 模块 ID 集合，用于安全降级
const VALID_MODULES = new Set<V2ModuleId>(['datapoint', 'access-source', 'compute', 'alarm'])

/** 从路由 params 解析当前模块，无效时回退默认值 */
const resolveModuleFromRoute = (): DatacenterModuleId => {
  const m = route.params.module as string | undefined
  if (m && VALID_MODULES.has(m as V2ModuleId)) {
    return m as DatacenterModuleId
  }
  return DEFAULT_MODULE
}

const activeModule = ref<DatacenterModuleId>(resolveModuleFromRoute())
const activeObjectId = computed(() => String(route.params.objectId || ''))
const selectedComputeUnitId = computed(() =>
  route.params.module === 'compute' ? activeObjectId.value : '',
)

/** 监听路由变化，同步 activeModule（如从 NavRail 外部 push 路由时） */
watch(
  () => route.params.module,
  (m) => {
    const next = (
      m && VALID_MODULES.has(m as V2ModuleId) ? m : DEFAULT_MODULE
    ) as DatacenterModuleId
    if (activeModule.value !== next) {
      activeModule.value = next
    }
  },
)

/** 切换模块时更新 URL（由 NavRail 双向绑定触发） */
watch(activeModule, (newModule, oldModule) => {
  if (newModule === oldModule) return
  if (route.params.module === newModule) return
  // 切换模块时清空 objectId/tab，保留 query string
  router.replace({
    path: `/${newModule}`,
    query: stripHandoffQuery(route.query),
  })
})
const project = computed(() => {
  const projectId = route.query.pid || route.query.id || route.meta?.project?.id
  const tenantId = route.query.tenant || route.meta?.project?.tenantId
  if (projectId) {
    return { id: projectId, tenantId: tenantId }
  }
  return null
})

// 提供给子组件使用
const projectId = computed(() => project.value?.id)
provide('projectId', projectId)
// 使用 composable
const {
  connections,
  selectedConnectionId,
  loadConnections,
  resetConnections,
  createConnection,
  updateConnection,
  deleteConnection,
  updateConnectionStatus,
  selectConnection,
} = useConnection(projectId)

// 统一标签页管理
const tabs = ref([])
const activeTabId = ref('')
const tabsRef = ref(null)
const dataPointListRefs = ref(new Map()) // 存储数据点列表引用
let tabCounter = 0

const resolveTabLabel = (tab) => resolveDatacenterTabLabel(tab, t)

/**
 * 设置数据点列表引用
 */
const setDataPointListRef = (tabId, el) => {
  if (el) {
    dataPointListRefs.value.set(tabId, el)
  } else {
    dataPointListRefs.value.delete(tabId)
  }
}

// 对话框状态
const showDataContractCheckDialog = ref(false)
const showConnectionDialog = ref(false)
const showDetailsDialog = ref(false)
const showTableStructureDialog = ref(false)
const connectionDialogMode = ref('create')
const currentConnection = ref(null)
const currentTableConnectionId = ref('')
const currentTableName = ref('')

// 连接右键菜单状态
const showContextMenu = ref(false)
const contextMenuPosition = ref({ x: 0, y: 0 })
const contextMenuConnection = ref(null)

// 表右键菜单状态
const showTableContextMenu = ref(false)
const tableContextMenuPosition = ref({ x: 0, y: 0 })
const tableContextMenuConnection = ref(null)
const tableContextMenuTable = ref(null)

// 查询右键菜单状态
const showQueryContextMenu = ref(false)
const queryContextMenuPosition = ref({ x: 0, y: 0 })
const queryContextMenuConnection = ref(null)
const queryContextMenuQuery = ref(null)

// 引用
const connectionListRef = ref(null)

/**
 * 创建表列表标签页
 */
const openTableListTab = (connection) => {
  const tabId = `table-list-${connection.id}`
  const existingTab = tabs.value.find((t) => t.id === tabId)

  if (existingTab) {
    activeTabId.value = tabId
    return
  }

  const newTab = {
    id: tabId,
    type: 'table-list',
    labelKey: 'tabs.tableList',
    labelPrefix: connection.name,
    label: resolveTabLabel({
      labelKey: 'tabs.tableList',
      labelPrefix: connection.name,
    }),
    icon: IconTablerTable,
    closable: true,
    connection: connection,
  }

  tabs.value.push(newTab)
  activeTabId.value = tabId
}

/**
 * 打开数据点列表
 */
const handleOpenDataPointList = () => {
  const tabId = 'datapoints'
  const existingTab = tabs.value.find((t) => t.id === tabId)

  if (existingTab) {
    activeTabId.value = tabId
    const listRef = dataPointListRefs.value.get(tabId)
    if (listRef && typeof listRef.refresh === 'function') {
      listRef.refresh()
    }
    return
  }

  const newTab = {
    id: tabId,
    type: 'datapoints',
    labelKey: 'tabs.datapoints',
    label: resolveTabLabel({ labelKey: 'tabs.datapoints' }),
    icon: IconTablerDatabase,
    closable: true,
  }

  tabs.value.push(newTab)
  activeTabId.value = tabId

  nextTick(() => {
    const listRef = dataPointListRefs.value.get(tabId)
    if (listRef && typeof listRef.refresh === 'function') {
      listRef.refresh()
    }
  })
}

/**
 * 打开计算单元列表
 */
const handleOpenCalcUnitList = () => {
  const tabId = 'calc-units'
  const existingTab = tabs.value.find((t) => t.id === tabId)

  if (existingTab) {
    activeTabId.value = tabId
    return
  }

  const newTab = {
    id: tabId,
    type: 'calc-units',
    labelKey: 'tabs.calcUnits',
    label: resolveTabLabel({ labelKey: 'tabs.calcUnits' }),
    icon: IconTablerCalculator,
    closable: true,
  }

  tabs.value.push(newTab)
  activeTabId.value = tabId
}

/**
 * 打开报警单元列表
 */
const handleOpenAlarmUnitList = () => {
  const tabId = 'alarm-units'
  const existingTab = tabs.value.find((t) => t.id === tabId)

  if (existingTab) {
    activeTabId.value = tabId
    return
  }

  const newTab = {
    id: tabId,
    type: 'alarm-units',
    labelKey: 'tabs.alarmUnits',
    label: resolveTabLabel({ labelKey: 'tabs.alarmUnits' }),
    icon: IconTablerBell,
    closable: true,
  }

  tabs.value.push(newTab)
  activeTabId.value = tabId
}

/**
 * 创建新查询标签页
 */
const createNewQuery = (connection) => {
  tabCounter++
  const tabId = `query-${connection.id}-${tabCounter}`

  const newTab = {
    id: tabId,
    type: 'query',
    labelKey: 'tabs.queryCounter',
    labelPrefix: connection.name,
    labelParams: { index: tabCounter },
    label: resolveTabLabel({
      labelKey: 'tabs.queryCounter',
      labelPrefix: connection.name,
      labelParams: { index: tabCounter },
    }),
    closable: true,
    connection: connection,
    connectionId: connection.id,
    table: '',
    sql: 'SELECT * FROM ',
    result: null,
    executing: false,
    saving: false,
    modified: false,
    resultPage: 1,
    resultPageSize: 20,
    parameters: [],
    parametersExpanded: true,
  }

  tabs.value.push(newTab)
  activeTabId.value = tabId
}

/**
 * 从表双击创建查询标签页
 */
const createQueryFromTable = (connection, tableName) => {
  tabCounter++
  const tabId = `query-${connection.id}-${tabCounter}`

  const dbType = connection.relationalConfig?.dbType
  let sql = ''
  if (dbType === 'mysql') {
    sql = `SELECT * FROM \`${tableName}\` LIMIT 100`
  } else if (dbType === 'postgresql') {
    sql = `SELECT * FROM "${tableName}" LIMIT 100`
  } else if (dbType === 'sqlserver') {
    sql = `SELECT * FROM [${tableName}] ORDER BY (SELECT NULL) OFFSET 0 ROWS FETCH NEXT 100 ROWS ONLY`
  }

  const newTab = {
    id: tabId,
    type: 'query',
    label: `${connection.name} - ${tableName}`,
    closable: true,
    connection: connection,
    connectionId: connection.id,
    table: tableName,
    sql: sql,
    result: null,
    executing: false,
    saving: false,
    modified: false,
    resultPage: 1,
    resultPageSize: 20,
    parameters: [],
    parametersExpanded: true,
    autoExecute: true,
  }

  tabs.value.push(newTab)
  activeTabId.value = tabId
}

/**
 * 关闭标签页
 */
const handleCloseTab = (tabId) => {
  const index = tabs.value.findIndex((t) => t.id === tabId)
  if (index === -1) return

  const tab = tabs.value[index]

  tabs.value.splice(index, 1)

  // 如果关闭的是当前激活的标签页，切换到最后一个标签页
  if (activeTabId.value === tabId) {
    if (tabs.value.length > 0) {
      activeTabId.value = tabs.value[tabs.value.length - 1].id
    } else {
      activeTabId.value = ''
    }
  }
}

/**
 * 标签页点击事件
 */
const handleTabClick = (_tab) => {
  // 可以在这里添加额外的逻辑
}

// 添加标签栏滚轮滚动支持
const setupTabsWheelScroll = async () => {
  await nextTick()

  if (tabsRef.value) {
    const tabsElement = tabsRef.value.$el
    const header = tabsElement?.querySelector('.el-tabs__header')

    if (header) {
      const handleWheel = (e) => {
        // 查找 Element Plus 的左右导航按钮
        const prevBtn = header.querySelector('.el-tabs__nav-prev')
        const nextBtn = header.querySelector('.el-tabs__nav-next')

        if (!prevBtn || !nextBtn) return

        // 检查按钮是否可用（没有 disabled 类）
        const canScrollLeft = !prevBtn.classList.contains('is-disabled')
        const canScrollRight = !nextBtn.classList.contains('is-disabled')

        // 向上滚动 -> 触发左按钮，向下滚动 -> 触发右按钮
        if (e.deltaY < 0 && canScrollLeft) {
          e.preventDefault()
          prevBtn.click()
        } else if (e.deltaY > 0 && canScrollRight) {
          e.preventDefault()
          nextBtn.click()
        }
      }

      header.addEventListener('wheel', handleWheel, { passive: false })

      // 返回清理函数
      return () => {
        header.removeEventListener('wheel', handleWheel)
      }
    }
  }
  return null
}

let cleanupWheelScroll = null

// 监听标签页变化，重新设置滚轮事件
watch(
  () => tabs.value.length,
  async (newLength, oldLength) => {
    console.log('Tabs length changed:', oldLength, '->', newLength)

    if (newLength > 0 && oldLength === 0) {
      // 第一次添加标签页时设置滚轮事件
      console.log('Setting up wheel scroll for first tab')
      cleanupWheelScroll = await setupTabsWheelScroll()
    }
  },
  { immediate: false },
)

// ---- 草稿保护 ----
const { confirmDraftAction } = useConfirm()

/**
 * 子模块可通过 provide 注入此函数来注册草稿状态检测器。
 * 检测器返回 true 表示当前有未保存草稿。
 */
const draftCheckers = ref<Array<() => boolean>>([])

/**
 * 提供给子模块注册草稿检测器。
 * 子模块在自己的 setup 里调用：inject('registerDraftChecker')(fn)
 */
provide('registerDraftChecker', (checker: () => boolean) => {
  draftCheckers.value.push(checker)
  // 返回注销函数
  return () => {
    const idx = draftCheckers.value.indexOf(checker)
    if (idx !== -1) draftCheckers.value.splice(idx, 1)
  }
})

/** 检测是否有任意模块存在未保存草稿 */
const hasPendingDraft = (): boolean => {
  return draftCheckers.value.some((fn) => fn())
}

// 路由离开守卫：有草稿时弹三选一
onBeforeRouteLeave(async (_to, _from, next) => {
  if (!hasPendingDraft()) {
    next()
    return
  }

  const action = await confirmDraftAction()
  if (action === 'cancel') {
    // 用户取消，不离开
    next(false)
  } else if (action === 'keep') {
    // 用户选择保留草稿，正常离开（草稿已在模块内写入 localStorage）
    next()
  } else {
    // 用户丢弃，清除草稿（由子模块监听 clearDraft 事件自行处理）
    draftCheckers.value = []
    next()
  }
})

// 浏览器关闭 / 刷新时提示（不支持三选一，只能给标准提示）
const handleBeforeUnload = (e: BeforeUnloadEvent) => {
  if (hasPendingDraft()) {
    e.preventDefault()
    // 标准做法：设置 returnValue 触发浏览器默认提示
    e.returnValue = ''
  }
}

/** 同步 project.store：将路由/Storage 的上下文注入 store */
const syncProjectStore = () => {
  if (project.value?.id) {
    projectStore.bootstrap({
      projectId: String(project.value.id),
      tenantId: project.value.tenantId ? String(project.value.tenantId) : undefined,
    })
  }
}

// 组件挂载时：同步 project.store 并加载数据，挂载 beforeunload 监听
onMounted(() => {
  syncProjectStore()
  if (projectId.value) {
    loadConnections()
  }
  window.addEventListener('beforeunload', handleBeforeUnload)
})

// 路由中 project 变化时（如正式入口 bootstrap 完成后路由元数据更新）也同步
watch(
  () => project.value?.id,
  (newId, oldId) => {
    if (!newId) {
      resetConnections()
      return
    }
    syncProjectStore()
    if (newId !== oldId) {
      resetConnections()
      loadConnections()
    }
  },
)

// 组件卸载时清理
onBeforeUnmount(() => {
  if (cleanupWheelScroll) {
    cleanupWheelScroll()
  }
  // 移除 beforeunload 监听
  window.removeEventListener('beforeunload', handleBeforeUnload)
})

/**
 * 选择连接
 */
const handleSelectConnection = (connection) => {
  selectConnection(connection.id)
}

/* v2 workbench：当 tab === 'workbench' 且找到对应 connection 时激活 */
const activeWorkbenchConnection = computed(() => {
  if (route.params.tab !== 'workbench') return null
  const id = String(route.params.objectId || '')
  if (!id) return null
  return connections.value.find((c) => String(c.id) === id) ?? null
})

/* 从 workbench 返回接入源列表，保留筛选 query */
const handleWorkbenchBack = () => {
  const isDebug = route.path.startsWith('/debug/')
  const base = isDebug ? '/debug' : ''
  void router.push({ path: `${base}/access-source`, query: route.query })
}

/**
 * 双击连接 - 测试连接并切换展开/折叠
 */
const handleConnectionDblClick = async (connection) => {
  try {
    if (connection.type === 'relational' && connection.relationalConfig) {
      // 检查当前展开状态
      const currentState = connectionListRef.value?.getConnectionState(connection.id)
      const isCurrentlyExpanded = currentState?.expanded || false

      // 如果已经展开，则折叠
      if (isCurrentlyExpanded) {
        if (connectionListRef.value) {
          currentState.expanded = false
        }
        return
      }

      // 如果未展开，则测试连接并展开
      const config = connection.relationalConfig
      await dataAPI.testConnection(projectId.value, {
        type: connection.type,
        config: {
          dbType: config.dbType,
          host: config.host,
          port: config.port,
          database: config.database,
          username: config.username,
          password: config.password,
        },
      })

      await updateConnectionStatus(connection.id, 'connected')
      ElMessage({
        type: 'success',
        message: t('query.connectionTestSuccess'),
        offset: 60,
        duration: 3000,
      })

      // 展开连接并加载表列表和查询列表
      if (connectionListRef.value) {
        const state = connectionListRef.value.getConnectionState(connection.id)
        state.expanded = true

        // 并行加载表列表和查询列表
        const loadPromises = []

        if (state.tables.length === 0) {
          loadPromises.push(connectionListRef.value.loadTables(connection.id))
        }

        if (state.queries.length === 0) {
          loadPromises.push(connectionListRef.value.loadQueries(connection.id))
        }

        if (loadPromises.length > 0) {
          await Promise.all(loadPromises)
        }
      }
    }
  } catch (error) {
    await updateConnectionStatus(connection.id, 'error')
    ElMessage({
      type: 'error',
      message: t('query.connectionTestFailed', {
        message: getApiErrorMessage(error, '连接测试失败'),
      }),
      offset: 60,
      duration: 5000,
      showClose: true,
    })
  }
}

/**
 * 右键菜单
 */
const handleConnectionContextMenu = (event, connection) => {
  contextMenuConnection.value = connection
  contextMenuPosition.value = { x: event.clientX, y: event.clientY }
  showContextMenu.value = true
}

/**
 * 打开连接（右键菜单）
 */
const handleOpenConnection = async (connection) => {
  await handleConnectionDblClick(connection)
}

/**
 * 断开连接
 */
const handleDisconnectConnection = async (connection) => {
  try {
    await updateConnectionStatus(connection.id, 'disconnected')
    ElMessage({
      type: 'success',
      message: t('query.disconnected'),
      offset: 60,
      duration: 3000,
    })

    // 折叠连接树
    if (connectionListRef.value) {
      const state = connectionListRef.value.getConnectionState(connection.id)
      state.expanded = false
      state.tablesExpanded = true
    }
  } catch {
    ElMessage({
      type: 'error',
      message: t('query.disconnectFailed'),
      offset: 60,
      duration: 3000,
    })
  }
}

/**
 * 查看连接详情
 */
const handleViewConnectionDetails = (connection) => {
  currentConnection.value = connection
  showDetailsDialog.value = true
}

/**
 * 编辑连接
 */
const handleEditConnection = (connection) => {
  connectionDialogMode.value = 'edit'
  currentConnection.value = connection
  showConnectionDialog.value = true
}

/**
 * 删除连接
 */
const handleDeleteConnection = async (connection) => {
  try {
    await ElMessageBox.confirm(
      t('query.deleteConnectionConfirm', { name: connection.name }),
      t('query.deleteConnectionTitle'),
      {
        confirmButtonText: t('actions.delete'),
        cancelButtonText: t('actions.cancel'),
        type: 'warning',
      },
    )

    await deleteConnection(connection.id)
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除连接失败:', error)
    }
  }
}

/**
 * 打开创建连接对话框
 */
const openCreateConnectionDialog = () => {
  connectionDialogMode.value = 'create'
  currentConnection.value = null
  showConnectionDialog.value = true
}

/**
 * 提交连接（创建或更新）
 */
const handleConnectionSubmit = async (data) => {
  try {
    if (connectionDialogMode.value === 'create') {
      const protocolCreator = protocolCreateHandlers[data.type]
      if (protocolCreator) {
        await protocolCreator(projectId.value, {
          name: data.name,
          status: 'disconnected',
          ...data.config,
        })
        ElMessage({
          type: 'success',
          message: '接入源创建成功',
          offset: 60,
          duration: 3000,
        })
        await loadConnections()
      } else {
        await createConnection({
          name: data.name,
          type: data.type,
          config: data.config,
        })
      }
    } else {
      await updateConnection(currentConnection.value.id, {
        name: data.name,
        type: data.type,
        config: data.config,
      })
    }
    showConnectionDialog.value = false
  } catch (error) {
    console.error('保存连接失败:', error)
  }
}

const protocolCreateHandlers = {
  mqtt: dataAPI.createMqttConnection,
  kafka: dataAPI.createKafkaConfig,
  http: dataAPI.createHttpConfig,
  websocket: dataAPI.createWebSocketConfig,
  redis: dataAPI.createRedisConfig,
  opcua: dataAPI.createOpcuaConfig,
  s7: dataAPI.createS7Config,
  modbus: dataAPI.createModbusConfig,
  tdengine: dataAPI.createTdengineConfig,
}

/**
 * 表双击 - 创建查询
 */
const handleTableDblClick = (connection, table) => {
  createQueryFromTable(connection, table.name)
}

/**
 * 表右键菜单
 */
const handleTableContextMenu = (event, connection, table) => {
  tableContextMenuConnection.value = connection
  tableContextMenuTable.value = table
  tableContextMenuPosition.value = { x: event.clientX, y: event.clientY }
  showTableContextMenu.value = true
}

/**
 * 查看表结构
 */
const handleViewTableStructure = (connection, table) => {
  currentTableConnectionId.value = connection.id
  currentTableName.value = table.name
  showTableStructureDialog.value = true
}

/**
 * 查询表数据（从右键菜单）
 */
const handleQueryTable = (connection, table) => {
  handleTableDblClick(connection, table)
}

/**
 * 查看表列表
 */
const handleViewTableList = (connection) => {
  openTableListTab(connection)
}

/**
 * 执行查询
 */
const handleQueryExecute = async (tab) => {
  // 标记正在执行
  tab.executing = true

  try {
    const parameters = tab.parameters.map((p) => p.value || '')
    console.log('执行查询 - SQL:', tab.sql)
    console.log('执行查询 - 参数对象:', tab.parameters)
    console.log('执行查询 - 参数值:', parameters)

    const response = await dataAPI.executeSql(
      projectId.value,
      tab.connectionId,
      tab.sql,
      parameters,
    )

    // 更新结果
    tab.result = {
      columns: response.data?.columns || [],
      rows: response.data?.rows || [],
      rowCount: response.data?.rowCount || 0,
      executionTime: response.data?.executionTime || 0,
    }
    tab.resultPage = 1
    ElMessage({
      type: 'success',
      message: t('query.executeSuccess', { count: tab.result.rowCount }),
      offset: 60,
      duration: 3000,
    })
  } catch (error) {
    const errorMsg = getApiErrorMessage(error, '执行查询失败')
    ElMessage({
      type: 'error',
      message: errorMsg,
      offset: 60,
      duration: 5000,
      showClose: true,
    })
  } finally {
    // 确保在所有情况下都重置执行状态
    tab.executing = false
  }
}

/**
 * 查询双击 - 打开查询
 */
const handleQueryDblClick = (connection, query) => {
  tabCounter++
  const tabId = `saved-query-${query.id}`

  // 检查是否已经打开
  const existingTab = tabs.value.find((t) => t.id === tabId)
  if (existingTab) {
    activeTabId.value = tabId
    return
  }

  const newTab = {
    id: tabId,
    type: 'query',
    label: `${connection.name} - ${query.name}`,
    closable: true,
    connection: connection,
    queryId: query.id,
    connectionId: connection.id,
    table: '',
    sql: query.config?.sql || '',
    result: null,
    executing: false,
    saving: false,
    modified: false,
    resultPage: 1,
    resultPageSize: 20,
    parameters: (query.config?.parameters || []).map((p) => ({
      name: p.name,
      type: p.type || 'string',
      value: p.default || '',
    })),
    parametersExpanded: true,
  }

  tabs.value.push(newTab)
  activeTabId.value = tabId
}

/**
 * 查询删除后的处理
 */
const handleQueryDeleted = (query) => {
  // 关闭对应的标签页
  const tabId = `saved-query-${query.id}`
  const index = tabs.value.findIndex((t) => t.id === tabId)
  if (index > -1) {
    handleCloseTab(tabId)
  }
}

/**
 * 查询右键菜单
 */
const handleQueryContextMenu = (event, connection, query) => {
  queryContextMenuConnection.value = connection
  queryContextMenuQuery.value = query
  queryContextMenuPosition.value = { x: event.clientX, y: event.clientY }
  showQueryContextMenu.value = true
}

/**
 * 查看查询详情
 */
const handleViewQueryDetails = (connection, query) => {
  const createdAt = dayjs(query.createdAt)
  const updatedAt = dayjs(query.updatedAt)
  ElMessageBox.alert(
    `
    <div style="text-align: left;">
      <p><strong>${t('query.name')}：</strong>${query.name}</p>
      <p><strong>${t('query.connection')}：</strong>${connection.name}</p>
      <p><strong>${t('query.type')}：</strong>${query.queryType || 'SQL'}</p>
      <p><strong>${t('query.enabled')}：</strong>${query.isEnabled ? t('common.yes') : t('common.no')}</p>
      <p><strong>${t('query.timeout')}：</strong>${query.timeout || 60000}ms</p>
      <p><strong>${t('query.cache')}：</strong>${query.cacheEnabled ? t('common.enabled') : t('common.disabled')}</p>
      ${query.cacheEnabled ? `<p><strong>${t('query.cacheTtl')}：</strong>${query.cacheTtl}${t('common.seconds')}</p>` : ''}
      <p><strong>${t('connection.createdAt')}：</strong>${createdAt.isValid() ? createdAt.format(TIME_FORMAT) : '-'}</p>
      <p><strong>${t('connection.updatedAt')}：</strong>${updatedAt.isValid() ? updatedAt.format(TIME_FORMAT) : '-'}</p>
    </div>
    `,
    t('query.detailsTitle'),
    {
      dangerouslyUseHTMLString: true,
      confirmButtonText: t('actions.close'),
    },
  )
}

/**
 * 从右键菜单删除查询
 */
const handleDeleteQueryFromMenu = async (connection, query) => {
  try {
    await ElMessageBox.confirm(
      t('query.deleteQueryConfirm', { name: query.name }),
      t('query.deleteConnectionTitle'),
      {
        confirmButtonText: t('actions.delete'),
        cancelButtonText: t('actions.cancel'),
        type: 'warning',
      },
    )

    await dataAPI.deleteQuery(query.id)
    ElMessage.success(t('query.deleted'))

    // 从连接树中移除
    if (connectionListRef.value) {
      const state = connectionListRef.value.getConnectionState(connection.id)
      const index = state.queries.findIndex((q) => q.id === query.id)
      if (index > -1) {
        state.queries.splice(index, 1)
      }
    }

    // 关闭对应的标签页
    handleQueryDeleted(query)
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(
        t('query.deleteQueryFailed', {
          message: getApiErrorMessage(error, '删除查询失败'),
        }),
      )
    }
  }
}

/**
 * 保存查询
 */
const handleQuerySave = async (tab) => {
  const defaultName = tab.labelKey === 'tabs.queryCounter' ? '' : tab.label?.split(' - ')[1] || ''

  // 使用自定义的 beforeClose 来控制对话框关闭
  const messageBoxInstance = ElMessageBox.prompt(t('query.promptName'), t('query.saveTitle'), {
    confirmButtonText: t('actions.save'),
    cancelButtonText: t('actions.cancel'),
    inputPattern: /^.{2,100}$/,
    inputErrorMessage: t('query.nameLengthError'),
    inputValue: defaultName,
    beforeClose: async (action, instance, done) => {
      if (action === 'confirm') {
        const queryName = instance.inputValue

        // 验证输入
        if (!queryName || queryName.length < 2 || queryName.length > 100) {
          instance.editorErrorMessage = t('query.nameLengthError')
          return
        }

        // 显示加载状态
        instance.confirmButtonLoading = true
        tab.saving = true

        try {
          const parameters = tab.parameters.map((param, index) => ({
            name: `param${index + 1}`,
            type: param.type || 'string',
            required: false,
            default: param.value || null,
          }))

          const response = await dataAPI.createQuery(projectId.value, {
            name: queryName,
            connectionId: tab.connectionId,
            queryType: 'sql',
            config: {
              sql: tab.sql,
              parameters: parameters,
            },
          })

          ElMessage({
            type: 'success',
            message: t('query.saveSuccess'),
            offset: 60,
            duration: 3000,
          })
          tab.label = `${tab.connection.name} - ${queryName}`
          tab.labelKey = null
          tab.labelPrefix = null
          tab.labelParams = null
          tab.queryId = response.data?.id
          tab.modified = false

          // 刷新左侧查询列表
          if (connectionListRef.value) {
            await connectionListRef.value.loadQueries(tab.connectionId)
          }

          // 关闭对话框
          done()
        } catch (error) {
          // 保存失败，显示错误但不关闭对话框
          const errorMsg = getApiErrorMessage(error, '保存查询失败')
          instance.editorErrorMessage = errorMsg
        } finally {
          instance.confirmButtonLoading = false
          tab.saving = false
        }
      } else {
        // 用户取消
        tab.saving = false
        done()
      }
    },
  })

  // 捕获用户直接关闭对话框的情况
  messageBoxInstance.catch(() => {
    tab.saving = false
  })
}

watch(datacenterLocale, () => {
  tabs.value = tabs.value.map((tab) => ({
    ...tab,
    label: resolveTabLabel(tab),
  }))
})
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

.dark .datacenter-side-action {
  color: rgba(226, 232, 240, 0.76);
}

.dark .datacenter-side-action:hover {
  border-color: rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.12);
  color: #f8fafc;
}

.dark .datacenter-side-action span {
  border-color: rgba(255, 255, 255, 0.12);
  background: #1e293b;
  color: #f8fafc;
}

.data-center {
  min-height: 400px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.module-card {
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.query-tabs :deep(.el-tabs__header) {
  margin: 0;
  flex-shrink: 0;
  background: var(--dc-surface-subtle);
  border-bottom: 1px solid var(--dc-border);
}

.query-tabs :deep(.el-tabs__nav-wrap::after) {
  height: 1px;
  background-color: var(--dc-border);
}

.query-tabs :deep(.el-tabs__item) {
  border: none;
  background: transparent;
  color: var(--dc-text-secondary);
}

.query-tabs :deep(.el-tabs__item.is-active) {
  color: var(--dc-primary);
  font-weight: 600;
}

.query-tabs :deep(.el-tabs__active-bar) {
  height: 2px;
  background-color: var(--dc-primary);
}

.query-tabs :deep(.el-tabs__nav) {
  border: none;
}

.query-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.query-tabs :deep(.el-tab-pane) {
  height: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
</style>
