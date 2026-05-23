// @ts-nocheck
/**
 * MySQL 专用逻辑 Composable
 * 处理 MySQL 特定的功能和 SQL 方言
 */

import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { format } from 'sql-formatter'
import { validateSql, extractParameters, parseSql } from '@/utils/sqlParser'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'

export function useMysql(projectId, connectionId) {
  const tables = ref([])
  const loading = ref(false)
  const tableData = ref([])
  const tableColumns = ref([])
  const tablePagination = ref({
    page: 1,
    limit: 100,
    total: 0,
    totalPages: 0,
  })

  /**
   * 加载表列表
   */
  const loadTables = async () => {
    if (!projectId.value || !connectionId.value) return

    loading.value = true
    try {
      const response = await dataAPI.getConnectionTables(projectId.value, connectionId.value)
      tables.value = response.data?.tables || []
      return tables.value
    } catch (error) {
      ElMessage.error('加载表列表失败：' + getApiErrorMessage(error, '加载表列表失败'))
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
      const response = await dataAPI.getTableData(projectId.value, connectionId.value, tableName, {
        page,
        limit: tablePagination.value.limit,
      })

      tableColumns.value = response.data?.columns || []
      tableData.value = response.data?.rows || []

      const pagination = response.data?.pagination || {}
      tablePagination.value = {
        page: pagination.page || 1,
        limit: pagination.limit || 100,
        total: pagination.total || 0,
        totalPages: pagination.totalPages || 0,
      }

      return response.data
    } catch (error) {
      ElMessage.error('加载表数据失败：' + getApiErrorMessage(error, '加载表数据失败'))
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
        parameters,
      )

      return {
        columns: response.data?.columns || [],
        rows: response.data?.rows || [],
        rowCount: response.data?.rowCount || 0,
        executionTime: response.data?.executionTime || 0,
      }
    } catch (error) {
      ElMessage.error('执行SQL失败：' + getApiErrorMessage(error, '执行SQL失败'))
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
      totalPages: 0,
    }
  }

  /**
   * 格式化表名（MySQL 使用反引号）
   */
  const formatTableName = (tableName) => {
    return `\`${tableName}\``
  }

  /**
   * 格式化列名（MySQL 使用反引号）
   */
  const formatColumnName = (columnName) => {
    return `\`${columnName}\``
  }

  /**
   * 获取默认查询语句
   */
  const getDefaultSelectQuery = (tableName) => {
    return `SELECT * FROM \`${tableName}\` LIMIT 100`
  }

  /**
   * 格式化 SQL（MySQL 方言）
   */
  const formatSql = (sql) => {
    try {
      return format(sql, {
        language: 'mysql',
        tabWidth: 2,
        keywordCase: 'upper',
        linesBetweenQueries: 2,
      })
    } catch (error) {
      console.error('SQL 格式化失败:', error)
      return sql
    }
  }

  /**
   * 获取支持的函数列表
   */
  const getSupportedFunctions = () => {
    return [
      // 日期时间函数
      { name: 'NOW()', description: '返回当前日期和时间' },
      { name: 'CURDATE()', description: '返回当前日期' },
      { name: 'CURTIME()', description: '返回当前时间' },
      { name: 'DATE_FORMAT(date, format)', description: '格式化日期' },
      { name: 'DATE_ADD(date, INTERVAL value unit)', description: '日期加法' },
      { name: 'DATE_SUB(date, INTERVAL value unit)', description: '日期减法' },
      { name: 'DATEDIFF(date1, date2)', description: '计算日期差' },

      // 字符串函数
      { name: 'CONCAT(str1, str2, ...)', description: '连接字符串' },
      {
        name: 'CONCAT_WS(separator, str1, str2, ...)',
        description: '使用分隔符连接字符串',
      },
      { name: 'SUBSTRING(str, pos, len)', description: '提取子字符串' },
      { name: 'LENGTH(str)', description: '返回字符串长度' },
      { name: 'UPPER(str)', description: '转换为大写' },
      { name: 'LOWER(str)', description: '转换为小写' },
      { name: 'TRIM(str)', description: '去除首尾空格' },
      { name: 'REPLACE(str, from_str, to_str)', description: '替换字符串' },

      // 数值函数
      { name: 'ABS(x)', description: '返回绝对值' },
      { name: 'CEIL(x)', description: '向上取整' },
      { name: 'FLOOR(x)', description: '向下取整' },
      { name: 'ROUND(x, d)', description: '四舍五入' },
      { name: 'RAND()', description: '返回随机数' },

      // 聚合函数
      { name: 'COUNT(*)', description: '计数' },
      { name: 'SUM(column)', description: '求和' },
      { name: 'AVG(column)', description: '平均值' },
      { name: 'MAX(column)', description: '最大值' },
      { name: 'MIN(column)', description: '最小值' },

      // 条件函数
      {
        name: 'IF(condition, true_value, false_value)',
        description: '条件判断',
      },
      { name: 'IFNULL(expr, alt_value)', description: '空值处理' },
      { name: 'CASE WHEN ... THEN ... END', description: '多条件判断' },
    ]
  }

  /**
   * 获取常用查询模板
   */
  const getQueryTemplates = () => {
    return [
      {
        name: '基础查询',
        sql: 'SELECT * FROM `table_name` WHERE condition LIMIT 100',
      },
      {
        name: '分组统计',
        sql: 'SELECT column, COUNT(*) as count\nFROM `table_name`\nGROUP BY column\nORDER BY count DESC',
      },
      {
        name: '内连接',
        sql: 'SELECT t1.*, t2.*\nFROM `table1` t1\nINNER JOIN `table2` t2 ON t1.id = t2.table1_id',
      },
      {
        name: '左连接',
        sql: 'SELECT t1.*, t2.*\nFROM `table1` t1\nLEFT JOIN `table2` t2 ON t1.id = t2.table1_id',
      },
      {
        name: '子查询',
        sql: 'SELECT * FROM `table_name`\nWHERE column IN (\n  SELECT column FROM `other_table`\n)',
      },
    ]
  }

  /**
   * 验证 SQL 语法（使用 SQL 解析器）
   */
  const validateMysqlSql = (sql) => {
    if (!sql || !sql.trim()) {
      return {
        valid: false,
        errors: ['SQL 语句不能为空'],
        warnings: [],
      }
    }

    return validateSql(sql, 'mysql')
  }

  /**
   * 提取 SQL 参数
   */
  const extractSqlParameters = (sql) => {
    return extractParameters(sql)
  }

  /**
   * 解析 SQL 语句
   */
  const parseMysqlSql = (sql) => {
    return parseSql(sql, 'mysql')
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
    formatTableName,
    formatColumnName,
    getDefaultSelectQuery,
    formatSql,
    getSupportedFunctions,
    getQueryTemplates,
    validateSql: validateMysqlSql,
    extractSqlParameters,
    parseSql: parseMysqlSql,
  }
}
