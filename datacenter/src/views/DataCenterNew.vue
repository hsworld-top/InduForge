<template>
  <div class="data-center h-full flex overflow-hidden">
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
      @view-table-list="handleViewTableList"
      @query-dblclick="handleQueryDblClick"
      @query-deleted="handleQueryDeleted"
    />

    <!-- 右侧内容区域 -->
    <div class="flex-1 flex flex-col bg-gray-50 dark:bg-gray-900 overflow-hidden">
      <!-- MySQL 内容区域 -->
      <MysqlContent
        v-if="selectedConnection && selectedConnection.type === 'relational'"
        ref="mysqlContentRef"
        :connection-id="selectedConnection.id"
        :connection="selectedConnection"
        @query-execute="handleQueryExecute"
        @query-save="handleQuerySave"
      />

      <!-- 默认提示 -->
      <div v-else class="h-full flex items-center justify-center text-gray-500">
        <div class="text-center">
          <IconTablerDatabase class="text-6xl mb-4 w-24 h-24 mx-auto" />
          <div class="text-lg">请选择数据连接</div>
        </div>
      </div>
    </div>

    <!-- 右键菜单 -->
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

    <!-- 新建/编辑连接对话框 -->
    <ConnectionDialog
      v-model="showConnectionDialog"
      :mode="connectionDialogMode"
      :connection="currentConnection"
      @submit="handleConnectionSubmit"
      @test="handleConnectionTest"
    />

    <!-- 查看连接详情对话框 -->
    <ConnectionDetailsDialog
      v-model="showDetailsDialog"
      :connection="currentConnection"
    />
  </div>
</template>

<script setup>
import { ref, computed, provide, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import IconTablerDatabase from '~icons/tabler/database'
import ConnectionList from '@/components/connection/ConnectionList.vue'
import ConnectionContextMenu from '@/components/connection/ConnectionContextMenu.vue'
import ConnectionDialog from '@/components/dialogs/ConnectionDialog.vue'
import ConnectionDetailsDialog from '@/components/dialogs/ConnectionDetailsDialog.vue'
import MysqlContent from '@/components/database/mysql/MysqlContent.vue'
import { useConnection } from '@/composables/useConnection'
import dataAPI from '@/api/data.api'

// 从路由获取 project 信息
const route = useRoute()
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
  selectedConnection,
  loadConnections,
  createConnection,
  updateConnection,
  deleteConnection,
  testConnection,
  updateConnectionStatus,
  selectConnection,
} = useConnection(projectId)

// 对话框状态
const showConnectionDialog = ref(false)
const showDetailsDialog = ref(false)
const connectionDialogMode = ref('create')
const currentConnection = ref(null)

// 右键菜单状态
const showContextMenu = ref(false)
const contextMenuPosition = ref({ x: 0, y: 0 })
const contextMenuConnection = ref(null)

// 引用
const mysqlContentRef = ref(null)
const connectionListRef = ref(null)

// 组件挂载时加载数据
onMounted(() => {
  if (projectId.value) {
    loadConnections()
  }
})

/**
 * 选择连接
 */
const handleSelectConnection = (connection) => {
  selectConnection(connection.id)
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
      const response = await dataAPI.testConnection(projectId.value, {
        type: connection.type,
        config: {
          dbType: config.dbType,
          host: config.host,
          port: config.port,
          database: config.database,
          username: config.username,
          password: config.password
        }
      })

      if (response.success) {
        await updateConnectionStatus(connection.id, 'connected')
        ElMessage.success('连接测试成功')
        
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
    }
  } catch (error) {
    await updateConnectionStatus(connection.id, 'error')
    ElMessage.error('连接测试失败：' + (error.response?.data?.message || error.message))
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
    ElMessage.success('连接已断开')
    
    // 折叠连接树
    if (connectionListRef.value) {
      const state = connectionListRef.value.getConnectionState(connection.id)
      state.expanded = false
      state.tablesExpanded = true
    }
  } catch (error) {
    ElMessage.error('断开连接失败')
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
      `确定要删除连接 "${connection.name}" 吗？此操作不可恢复。`,
      '删除确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
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
      await createConnection({
        name: data.name,
        type: data.type,
        config: data.config
      })
    } else {
      await updateConnection(currentConnection.value.id, {
        name: data.name,
        type: data.type,
        config: data.config
      })
    }
    showConnectionDialog.value = false
  } catch (error) {
    console.error('保存连接失败:', error)
  }
}

/**
 * 测试连接
 */
const handleConnectionTest = async (data) => {
  try {
    await testConnection(data.type, data.config)
  } catch (error) {
    console.error('测试连接失败:', error)
  }
}

/**
 * 表双击 - 创建查询
 */
const handleTableDblClick = (connection, table) => {
  selectConnection(connection.id)
  if (mysqlContentRef.value) {
    mysqlContentRef.value.createQueryFromTable(table.name)
  }
}

/**
 * 查看表列表
 */
const handleViewTableList = (connection) => {
  selectConnection(connection.id)
  // 切换到表列表标签页
  if (mysqlContentRef.value) {
    mysqlContentRef.value.showTableList()
  }
}

/**
 * 执行查询
 */
const handleQueryExecute = async (tab) => {
  // 标记正在执行
  tab.executing = true
  
  try {
    const parameters = tab.parameters.map(p => p.value || '')
    const response = await dataAPI.executeSql(
      projectId.value,
      tab.connectionId,
      tab.sql,
      parameters
    )

    if (response.success) {
      // 更新结果
      tab.result = {
        columns: response.data.columns || [],
        rows: response.data.rows || [],
        rowCount: response.data.rowCount || 0,
        executionTime: response.executionTime || 0
      }
      tab.resultPage = 1
      ElMessage.success(`查询执行成功，返回 ${tab.result.rowCount} 行`)
    }
  } catch (error) {
    ElMessage.error('执行SQL失败：' + (error.response?.data?.message || error.message))
  } finally {
    // 确保在所有情况下都重置执行状态
    tab.executing = false
  }
}

/**
 * 查询双击 - 打开查询
 */
const handleQueryDblClick = (connection, query) => {
  selectConnection(connection.id)
  if (mysqlContentRef.value) {
    mysqlContentRef.value.openQuery(query)
  }
}

/**
 * 查询删除后的处理
 */
const handleQueryDeleted = (query) => {
  // 如果当前打开的标签页是被删除的查询，关闭它
  if (mysqlContentRef.value) {
    mysqlContentRef.value.closeQueryTab(query.id)
  }
}

/**
 * 保存查询
 */
const handleQuerySave = async (tab) => {
  try {
    const { value: queryName } = await ElMessageBox.prompt('请输入查询名称', '保存查询', {
      confirmButtonText: '保存',
      cancelButtonText: '取消',
      inputPattern: /^.{2,100}$/,
      inputErrorMessage: '查询名称长度在 2 到 100 个字符',
      inputValue: tab.name.startsWith('查询') ? '' : tab.name
    })

    tab.saving = true

    const parameters = tab.parameters.map((param, index) => ({
      name: `param${index + 1}`,
      type: param.type || 'string',
      required: false,
      default: param.value || null
    }))

    const response = await dataAPI.createQuery(projectId.value, {
      name: queryName,
      connectionId: tab.connectionId,
      queryType: 'sql',
      config: {
        sql: tab.sql,
        parameters: parameters
      }
    })

    if (response.success) {
      ElMessage.success('查询保存成功')
      tab.name = queryName
      tab.queryId = response.data.id
      tab.modified = false
      
      // 刷新左侧查询列表
      if (connectionListRef.value) {
        await connectionListRef.value.loadQueries(tab.connectionId)
      }
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('保存查询失败：' + (error.response?.data?.message || error.message))
    }
  } finally {
    tab.saving = false
  }
}
</script>

<style scoped>
.data-center {
  min-height: 400px;
}
</style>
