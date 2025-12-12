/**
 * 数据库通用逻辑 Composable
 * 提供所有数据库类型共享的基础功能
 */

import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import dataAPI from '@/api/data.api'

export function useDatabase(projectId, connectionId) {
  const tables = ref([])
  const loading = ref(false)
  const tableData = ref([])
  const tableColumns = ref([])
  const tablePagination = ref({
    page: 1,
    limit: 100,
    total: 0,
    totalPages: 0
  })

  /**
   * 加载表列表
   */
  const loadTables = async () => {
    if (!projectId.value || !connectionId.value) return

    loading.value = true
    try {
      const response = await dataAPI.getConnectionTables(projectId.value, connectionId.value)
      if (response.success) {
        tables.value = response.data.tables || []
        return tables.value
      }
    } catch (error) {
      ElMessage.error('加载表列表失败：' + (error.response?.data?.message || error.message))
      throw error
    } finally {
      loading.value = false
    }
  }

  /**
   * 加载表数据
   */
  const loadTableData = async (tableName, page = 1) => {
    if (!projectId.value || !connectionId.value || !tableName) return

    loading.value = true
    try {
      const response = await dataAPI.getTableData(
        projectId.value,
        connectionId.value,
        tableName,
        { page, limit: tablePagination.value.limit }
      )
      
      if (response.success) {
        tableColumns.value = response.data.columns || []
        tableData.value = response.data.rows || []
        
        const pagination = response.data.pagination || {}
        tablePagination.value = {
          page: pagination.page || 1,
          limit: pagination.limit || 100,
          total: pagination.total || 0,
          totalPages: pagination.totalPages || 0
        }
        
        return response.data
      }
    } catch (error) {
      ElMessage.error('加载表数据失败：' + (error.response?.data?.message || error.message))
      throw error
    } finally {
      loading.value = false
    }
  }

  /**
   * 执行 SQL 查询
   */
  const executeSql = async (sql, parameters = []) => {
    if (!projectId.value || !connectionId.value) return null

    try {
      const response = await dataAPI.executeSql(
        projectId.value,
        connectionId.value,
        sql,
        parameters
      )
      
      if (response.success) {
        return {
          columns: response.data.columns || [],
          rows: response.data.rows || [],
          rowCount: response.data.rowCount || 0,
          executionTime: response.executionTime || 0
        }
      }
    } catch (error) {
      ElMessage.error('执行SQL失败：' + (error.response?.data?.message || error.message))
      throw error
    }
  }

  /**
   * 重置表数据
   */
  const resetTableData = () => {
    tableData.value = []
    tableColumns.value = []
    tablePagination.value = {
      page: 1,
      limit: 100,
      total: 0,
      totalPages: 0
    }
  }

  return {
    tables,
    loading,
    tableData,
    tableColumns,
    tablePagination,
    loadTables,
    loadTableData,
    executeSql,
    resetTableData,
  }
}
