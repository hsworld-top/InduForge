// @ts-nocheck
/**
 * 连接管理 Composable
 * 处理连接的 CRUD 操作和状态管理
 */

import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'

/**
 * 兼容 data_service 当前返回的两类连接列表包络：
 * 1. { data: { connections: [...] } }：前端旧假设
 * 2. { data: [...] }：后端当前实际返回
 *
 * 这里统一收敛成数组，避免 debug 链路在接口成功时仍被误判为“暂无连接”。
 */
export function normalizeConnectionsPayload(response) {
  if (Array.isArray(response?.data?.connections)) {
    return response.data.connections
  }

  if (Array.isArray(response?.data)) {
    return response.data
  }

  return []
}

export function useConnection(projectId) {
  const connections = ref([])
  const selectedConnectionId = ref(null)
  const loading = ref(false)

  // 当前选中的连接
  const selectedConnection = computed(() => {
    return connections.value.find((c) => c.id === selectedConnectionId.value) || null
  })

  // 关系型数据库连接列表
  const relationalConnections = computed(() => {
    return connections.value.filter((c) => c.type === 'relational')
  })

  /**
   * 加载连接列表
   */
  const loadConnections = async () => {
    if (!projectId.value) return

    loading.value = true
    try {
      const response = await dataAPI.getConnections(projectId.value)
      connections.value = normalizeConnectionsPayload(response)
    } catch (error) {
      console.error('加载连接列表失败:', error)
      ElMessage({
        type: 'error',
        message: '加载连接列表失败：' + getApiErrorMessage(error, '加载连接列表失败'),
        offset: 60,
        duration: 5000,
        showClose: true,
      })
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
      ElMessage({
        type: 'success',
        message: '连接创建成功',
        offset: 60,
        duration: 3000,
      })
      await loadConnections()
      return response.data
    } catch (error) {
      ElMessage({
        type: 'error',
        message: '创建连接失败：' + getApiErrorMessage(error, '创建连接失败'),
        offset: 60,
        duration: 5000,
        showClose: true,
      })
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
      ElMessage({
        type: 'success',
        message: '连接更新成功',
        offset: 60,
        duration: 3000,
      })
      await loadConnections()
      return response.data
    } catch (error) {
      ElMessage({
        type: 'error',
        message: '更新连接失败：' + getApiErrorMessage(error, '更新连接失败'),
        offset: 60,
        duration: 5000,
        showClose: true,
      })
      throw error
    }
  }

  /**
   * 删除连接
   */
  const deleteConnection = async (connectionId) => {
    if (!projectId.value) return false

    try {
      await dataAPI.deleteConnection(projectId.value, connectionId)
      ElMessage({
        type: 'success',
        message: '连接删除成功',
        offset: 60,
        duration: 3000,
      })
      await loadConnections()

      // 如果删除的是当前选中的连接，清空选择
      if (selectedConnectionId.value === connectionId) {
        selectedConnectionId.value = null
      }
      return true
    } catch (error) {
      ElMessage({
        type: 'error',
        message: '删除连接失败：' + getApiErrorMessage(error, '删除连接失败'),
        offset: 60,
        duration: 5000,
        showClose: true,
      })
      throw error
    }
  }

  /**
   * 测试连接
   */
  const testConnection = async (type, config) => {
    if (!projectId.value) return false

    try {
      const response = await dataAPI.testConnection(projectId.value, {
        type,
        config,
      })
      ElMessage({
        type: 'success',
        message: response.msg || '连接测试成功',
        offset: 60,
        duration: 3000,
      })
      return true
    } catch (error) {
      ElMessage({
        type: 'error',
        message: '连接测试失败：' + getApiErrorMessage(error, '连接测试失败'),
        offset: 60,
        duration: 5000,
        showClose: true,
      })
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
      await loadConnections()
      return response.data
    } catch (error) {
      ElMessage({
        type: 'error',
        message: '更新连接状态失败：' + getApiErrorMessage(error, '更新连接状态失败'),
        offset: 60,
        duration: 5000,
        showClose: true,
      })
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
    return connections.value.find((c) => c.id === connectionId) || null
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
