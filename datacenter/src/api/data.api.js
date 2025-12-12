import request from '@/utils/request'

// ===========================================
// 数据连接相关API
// ===========================================

/**
 * 获取数据连接列表
 * @param {string} projectId - 工程ID
 * @param {object} params - 查询参数
 */
export const getConnections = (projectId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/connections`,
    method: 'get',
    params
  })
}

/**
 * 创建数据连接
 * @param {string} projectId - 工程ID
 * @param {object} data - 连接数据
 */
export const createConnection = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/connections`,
    method: 'post',
    data
  })
}

export const testConnection = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/connections/test`,
    method: 'post',
    data
  })
}

export const getConnectionTables = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/tables`,
    method: 'get'
  })
}

export const getTableData = (projectId, connectionId, tableName, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/tables/${tableName}/data`,
    method: 'get',
    params
  })
}

/**
 * 更新数据连接
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 * @param {object} data - 更新数据
 */
export const updateConnection = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}`,
    method: 'put',
    data
  })
}

/**
 * 删除数据连接
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 */
export const deleteConnection = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}`,
    method: 'delete'
  })
}

/**
 * 更新连接状态
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 * @param {string} status - 状态 (connected, disconnected, error, unknown)
 */
export const updateConnectionStatus = (projectId, connectionId, status) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/status`,
    method: 'patch',
    data: { status }
  })
}


// ===========================================
// 数据查询相关API
// ===========================================

/**
 * 获取数据查询列表
 * @param {string} projectId - 工程ID
 * @param {object} params - 查询参数
 */
export const getQueries = (projectId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/queries`,
    method: 'get',
    params
  })
}

/**
 * 创建数据查询
 * @param {string} projectId - 工程ID
 * @param {object} data - 查询数据
 */
export const createQuery = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/queries`,
    method: 'post',
    data
  })
}

/**
 * 更新数据查询
 * @param {string} id - 查询ID
 * @param {object} data - 更新数据
 */
export const updateQuery = (id, data) => {
  return request({
    url: `/data/queries/${id}`,
    method: 'put',
    data
  })
}

/**
 * 删除数据查询
 * @param {string} id - 查询ID
 */
export const deleteQuery = (id) => {
  return request({
    url: `/data/queries/${id}`,
    method: 'delete'
  })
}

/**
 * 执行数据查询
 * @param {string} id - 查询ID
 * @param {object} parameters - 查询参数
 */
export const executeQuery = (id, parameters = {}) => {
  return request({
    url: `/data/queries/${id}/execute`,
    method: 'post',
    data: { parameters }
  })
}

/**
 * 直接执行SQL查询
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 * @param {string} sql - SQL语句
 * @param {array} parameters - 参数数组（可选）
 */
export const executeSql = (projectId, connectionId, sql, parameters = []) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/execute-sql`,
    method: 'post',
    data: { sql, parameters }
  })
}

/**
 * 保存数据查询
 * @param {string} id - 查询ID
 * @param {object} data - 查询数据
 */
export const saveQuery = (id, data) => {
  return request({
    url: `/data/queries/${id}`,
    method: 'put',
    data
  })
}

export default {
  getConnections,
  createConnection,
  testConnection,
  getConnectionTables,
  getTableData,
  updateConnection,
  deleteConnection,
  updateConnectionStatus,
  getQueries,
  createQuery,
  updateQuery,
  executeQuery,
  executeSql,
  deleteQuery,
}

