/**
 * 连接管理 Composable
 * 处理连接的 CRUD 操作和状态管理
 */

import { ref, computed, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  createConnection as createConnectionRequest,
  deleteConnection as deleteConnectionRequest,
  listConnections,
  testConnectionDraft,
  updateConnection as updateConnectionRequest,
} from '@/api/connection.api'
import type { Connection, ConnectionPage } from '@/api/schemas/connection.schema'
import { getApiErrorMessage } from '@/utils/request'

export function useConnection(
  projectId: Readonly<Ref<string | undefined>>,
  options: { paginated?: boolean } = {},
) {
  const connections = ref<Connection[]>([])
  const selectedConnectionId = ref<string | null>(null)
  const loading = ref(false)
  const paginated = options.paginated === true
  const connectionPagination = ref({ page: 1, pageSize: 10, total: 0, totalPages: 0 })
  const connectionQuery = ref({ search: '', typeGroup: 'all' })

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
    if (!projectId.value) {
      connections.value = []
      selectedConnectionId.value = null
      return
    }

    loading.value = true
    try {
      const params = paginated
        ? {
            page: connectionPagination.value.page,
            pageSize: connectionPagination.value.pageSize,
            search: connectionQuery.value.search || undefined,
            typeGroup:
              connectionQuery.value.typeGroup === 'all'
                ? undefined
                : connectionQuery.value.typeGroup,
          }
        : undefined
      const response = await listConnections(projectId.value, params)
      if (paginated) {
        const page = response as ConnectionPage
        connections.value = page.list
        connectionPagination.value = page.pagination
        if (
          connectionPagination.value.totalPages > 0 &&
          connectionPagination.value.page > connectionPagination.value.totalPages
        ) {
          connectionPagination.value.page = connectionPagination.value.totalPages
          return await loadConnections()
        }
      } else connections.value = response as Connection[]
    } catch (error) {
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
   * 工程上下文切换时必须清空上一工程的连接状态。
   * IDE 会缓存多个数据中心标签页，同源 localStorage 变化不能让旧列表继续显示。
   */
  const resetConnections = () => {
    connections.value = []
    selectedConnectionId.value = null
    connectionPagination.value = { page: 1, pageSize: 10, total: 0, totalPages: 0 }
    connectionQuery.value = { search: '', typeGroup: 'all' }
  }

  const setConnectionQuery = async ({
    page,
    pageSize,
    search,
    typeGroup,
  }: {
    page?: number
    pageSize?: number
    search?: string
    typeGroup?: string
  }) => {
    if (!paginated) return
    connectionPagination.value.page = Number(page || 1)
    connectionPagination.value.pageSize = Number(pageSize || connectionPagination.value.pageSize)
    connectionQuery.value = {
      search: String(search || ''),
      typeGroup: String(typeGroup || 'all'),
    }
    await loadConnections()
  }

  /**
   * 创建连接
   */
  const createConnection = async (data: Record<string, unknown>) => {
    if (!projectId.value) return null

    try {
      const response = await createConnectionRequest(projectId.value, data)
      ElMessage({
        type: 'success',
        message: '连接创建成功',
        offset: 60,
        duration: 3000,
      })
      await loadConnections()
      return response
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
  const updateConnection = async (connectionId: string, data: Record<string, unknown>) => {
    if (!projectId.value) return null

    try {
      const response = await updateConnectionRequest(projectId.value, connectionId, data)
      ElMessage({
        type: 'success',
        message: '连接更新成功',
        offset: 60,
        duration: 3000,
      })
      await loadConnections()
      return response
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
  const deleteConnection = async (connectionId: string) => {
    if (!projectId.value) return false

    try {
      await deleteConnectionRequest(projectId.value, connectionId)
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
  const testConnection = async (type: string, config: Record<string, unknown>) => {
    if (!projectId.value) return false

    try {
      const response = await testConnectionDraft(projectId.value, {
        type,
        config,
      })
      ElMessage({
        type: 'success',
        message: response.message || '连接测试成功',
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
   * 选择连接
   */
  const selectConnection = (connectionId: string | null) => {
    selectedConnectionId.value = connectionId
  }

  /**
   * 根据 ID 获取连接
   */
  const getConnectionById = (connectionId: string) => {
    return connections.value.find((c) => c.id === connectionId) || null
  }

  return {
    connections,
    selectedConnectionId,
    selectedConnection,
    relationalConnections,
    loading,
    connectionPagination,
    connectionQuery,
    loadConnections,
    setConnectionQuery,
    resetConnections,
    createConnection,
    updateConnection,
    deleteConnection,
    testConnection,
    selectConnection,
    getConnectionById,
  }
}
