// @ts-nocheck
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request, { getApiErrorMessage } from '../utils/request'

/**
 * MQTT 连接管理 Composable
 */
export function useMqttConnection(projectId) {
  const connections = ref([])
  const loading = ref(false)
  const pagination = ref({
    page: 1,
    pageSize: 50,
    total: 0,
    totalPages: 0,
  })

  /**
   * 获取连接列表
   */
  const fetchConnections = async (page = 1) => {
    try {
      loading.value = true
      const response = await request.get(`/data/projects/${projectId}/mqtt/connections`, {
        params: {
          page,
          pageSize: pagination.value.pageSize,
        },
      })

      connections.value = response.data?.connections || []
      Object.assign(pagination.value, response.data?.pagination || {})
    } catch (error) {
      console.error('获取 MQTT 连接列表失败:', error)
      ElMessage.error(getApiErrorMessage(error, '获取连接列表失败'))
      throw error
    } finally {
      loading.value = false
    }
  }

  /**
   * 获取单个连接
   */
  const fetchConnection = async (connectionId) => {
    try {
      const response = await request.get(
        `/data/projects/${projectId}/mqtt/connections/${connectionId}`,
      )

      return response.data
    } catch (error) {
      console.error('获取 MQTT 连接失败:', error)
      ElMessage.error(getApiErrorMessage(error, '获取连接失败'))
      throw error
    }
  }

  /**
   * 创建连接
   */
  const createConnection = async (connectionData) => {
    try {
      loading.value = true
      const response = await request.post(
        `/data/projects/${projectId}/mqtt/connections`,
        connectionData,
      )

      ElMessage.success('MQTT 连接创建成功')
      await fetchConnections(pagination.value.page)
      return response.data
    } catch (error) {
      console.error('创建 MQTT 连接失败:', error)
      ElMessage.error(getApiErrorMessage(error, '创建连接失败'))
      throw error
    } finally {
      loading.value = false
    }
  }

  /**
   * 更新连接
   */
  const updateConnection = async (connectionId, connectionData) => {
    try {
      loading.value = true
      const response = await request.put(
        `/data/projects/${projectId}/mqtt/connections/${connectionId}`,
        connectionData,
      )

      ElMessage.success('MQTT 连接更新成功')
      await fetchConnections(pagination.value.page)
      return response.data
    } catch (error) {
      console.error('更新 MQTT 连接失败:', error)
      ElMessage.error(getApiErrorMessage(error, '更新连接失败'))
      throw error
    } finally {
      loading.value = false
    }
  }

  /**
   * 删除连接
   */
  const deleteConnection = async (connectionId) => {
    try {
      await ElMessageBox.confirm('确定要删除这个 MQTT 连接吗？删除后将无法恢复。', '确认删除', {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消',
      })

      loading.value = true
      await request.delete(`/data/projects/${projectId}/mqtt/connections/${connectionId}`)

      ElMessage.success('MQTT 连接删除成功')
      await fetchConnections(pagination.value.page)
    } catch (error) {
      if (error !== 'cancel') {
        console.error('删除 MQTT 连接失败:', error)
        ElMessage.error(getApiErrorMessage(error, '删除连接失败'))
        throw error
      }
    } finally {
      loading.value = false
    }
  }

  /**
   * 测试连接
   */
  const testConnection = async (connectionData) => {
    try {
      await request.post(`/data/projects/${projectId}/mqtt/connections/test`, connectionData)

      ElMessage.success('MQTT 连接测试成功')
      return true
    } catch (error) {
      console.error('测试 MQTT 连接失败:', error)
      ElMessage.error(getApiErrorMessage(error, '连接测试失败'))
      throw error
    }
  }

  /**
   * 启动连接
   */
  const startConnection = async (connectionId) => {
    try {
      loading.value = true
      const response = await request.post(
        `/data/projects/${projectId}/mqtt/connections/${connectionId}/start`,
      )

      ElMessage.success('MQTT 连接已启动')
      await fetchConnections(pagination.value.page)
      return response.data
    } catch (error) {
      console.error('启动 MQTT 连接失败:', error)
      ElMessage.error(getApiErrorMessage(error, '启动连接失败'))
      throw error
    } finally {
      loading.value = false
    }
  }

  /**
   * 停止连接
   */
  const stopConnection = async (connectionId) => {
    try {
      loading.value = true
      const response = await request.post(
        `/data/projects/${projectId}/mqtt/connections/${connectionId}/stop`,
      )

      ElMessage.success('MQTT 连接已停止')
      await fetchConnections(pagination.value.page)
      return response.data
    } catch (error) {
      console.error('停止 MQTT 连接失败:', error)
      ElMessage.error(getApiErrorMessage(error, '停止连接失败'))
      throw error
    } finally {
      loading.value = false
    }
  }

  /**
   * 获取连接状态
   */
  const getConnectionStatus = async (connectionId) => {
    try {
      const response = await request.get(
        `/data/projects/${projectId}/mqtt/connections/${connectionId}/status`,
      )

      return response.data
    } catch (error) {
      console.error('获取连接状态失败:', error)
      throw error
    }
  }

  /**
   * 翻页
   */
  const handlePageChange = (page) => {
    fetchConnections(page)
  }

  // 计算属性
  const hasConnections = computed(() => connections.value.length > 0)
  const activeConnections = computed(() =>
    connections.value.filter((conn) => conn.runtimeStatus?.connected),
  )
  const inactiveConnections = computed(() =>
    connections.value.filter((conn) => !conn.runtimeStatus?.connected),
  )

  return {
    // 状态
    connections,
    loading,
    pagination,

    // 计算属性
    hasConnections,
    activeConnections,
    inactiveConnections,

    // 方法
    fetchConnections,
    fetchConnection,
    createConnection,
    updateConnection,
    deleteConnection,
    testConnection,
    startConnection,
    stopConnection,
    getConnectionStatus,
    handlePageChange,
  }
}
