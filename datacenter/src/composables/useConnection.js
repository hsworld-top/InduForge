/**
 * 连接管理 Composable
 * 处理连接的 CRUD 操作和状态管理
 */

import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import dataAPI from '@/api/data.api'

export function useConnection(projectId) {
  const connections = ref([])
  const selectedConnectionId = ref(null)
  const loading = ref(false)

  // 当前选中的连接
  const selectedConnection = computed(() => {
    return connections.value.find(c => c.id === selectedConnectionId.value) || null
  })

  // 关系型数据库连接列表
  const relationalConnections = computed(() => {
    return connections.value.filter(c => c.type === 'relational')
  })

  /**
   * 加载连接列表
   */
  const loadConnections = async () => {
    if (!projectId.value) return

    loading.value = true
    try {
      const response = await dataAPI.getConnections(projectId.value)
      if (response.success) {
        connections.value = response.data.connections || []
      }
    } catch (error) {
      console.error('加载连接列表失败:', error)
      ElMessage.error('加载连接列表失败：' + (error.response?.data?.message || error.message))
    } finally {
      loading.value = false
    }
  }

  /**
   * 创建连接
   */
  const createConnection = async (data) => {
    if (!projectId.value) return null

    try {
      const response = await dataAPI.createConnection(projectId.value, data)
      if (response.success) {
        ElMessage.success('连接创建成功')
        await loadConnections()
        return response.data
      }
    } catch (error) {
      ElMessage.error('创建连接失败：' + (error.response?.data?.message || error.message))
      throw error
    }
  }

  /**
   * 更新连接
   */
  const updateConnection = async (connectionId, data) => {
    if (!projectId.value) return null

    try {
      const response = await dataAPI.updateConnection(projectId.value, connectionId, data)
      if (response.success) {
        ElMessage.success('连接更新成功')
        await loadConnections()
        return response.data
      }
    } catch (error) {
      ElMessage.error('更新连接失败：' + (error.response?.data?.message || error.message))
      throw error
    }
  }

  /**
   * 删除连接
   */
  const deleteConnection = async (connectionId) => {
    if (!projectId.value) return false

    try {
      const response = await dataAPI.deleteConnection(projectId.value, connectionId)
      if (response.success) {
        ElMessage.success('连接删除成功')
        await loadConnections()
        
        // 如果删除的是当前选中的连接，清空选择
        if (selectedConnectionId.value === connectionId) {
          selectedConnectionId.value = null
        }
        return true
      }
    } catch (error) {
      ElMessage.error('删除连接失败：' + (error.response?.data?.message || error.message))
      throw error
    }
  }

  /**
   * 测试连接
   */
  const testConnection = async (type, config) => {
    if (!projectId.value) return false

    try {
      const response = await dataAPI.testConnection(projectId.value, { type, config })
      if (response.success) {
        ElMessage.success(response.message || '连接测试成功')
        return true
      }
      return false
    } catch (error) {
      ElMessage.error('连接测试失败：' + (error.response?.data?.message || error.message))
      throw error
    }
  }

  /**
   * 更新连接状态
   */
  const updateConnectionStatus = async (connectionId, status) => {
    if (!projectId.value) return null

    try {
      const response = await dataAPI.updateConnectionStatus(projectId.value, connectionId, status)
      if (response.success) {
        await loadConnections()
        return response.data
      }
    } catch (error) {
      ElMessage.error('更新连接状态失败：' + (error.response?.data?.message || error.message))
      throw error
    }
  }

  /**
   * 选择连接
   */
  const selectConnection = (connectionId) => {
    selectedConnectionId.value = connectionId
  }

  /**
   * 根据 ID 获取连接
   */
  const getConnectionById = (connectionId) => {
    return connections.value.find(c => c.id === connectionId) || null
  }

  return {
    connections,
    selectedConnectionId,
    selectedConnection,
    relationalConnections,
    loading,
    loadConnections,
    createConnection,
    updateConnection,
    deleteConnection,
    testConnection,
    updateConnectionStatus,
    selectConnection,
    getConnectionById,
  }
}
