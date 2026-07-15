// @ts-nocheck
import request from '@/utils/request'
import {
  HttpRequestGroupListSchema,
  HttpRequestGroupSchema,
  HttpRequestListSchema,
  HttpRequestSchema,
  HttpSendResponseSchema,
} from './schemas/http-workbench.schema'
import {
  WebSocketPreviewResponseSchema,
  WebSocketSessionGroupListSchema,
  WebSocketSessionGroupSchema,
  WebSocketSessionListSchema,
  WebSocketSessionSchema,
} from './schemas/websocket-workbench.schema'
import {
  KafkaFieldGroupListSchema,
  KafkaFieldGroupSchema,
  KafkaFieldListSchema,
  KafkaFieldSchema,
  KafkaPreviewSchema,
  KafkaTopicGroupListSchema,
  KafkaTopicGroupSchema,
  KafkaTopicMappingListSchema,
  KafkaTopicMappingSchema,
} from './schemas/kafka-workbench.schema'

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
    params,
  })
}

export const getConnection = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}`,
    method: 'get',
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
    data,
  })
}

export const createMqttConnection = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/connections`,
    method: 'post',
    data,
  })
}

export const testConnection = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/connections/test`,
    method: 'post',
    data,
  })
}

export const testMqttConnection = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/connections/test`,
    method: 'post',
    data,
  })
}

export const createKafkaConfig = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/kafka/configs`,
    method: 'post',
    data,
  })
}

export const createHttpConfig = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/http/configs`,
    method: 'post',
    data,
  })
}

export const createWebSocketConfig = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/websocket/configs`,
    method: 'post',
    data,
  })
}

export const createRedisConfig = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/redis/configs`,
    method: 'post',
    data,
  })
}

export const createTdengineConfig = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/tdengine/configs`,
    method: 'post',
    data,
  })
}

export const validateOpcdaContract = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/opcda/contracts/validate`,
    method: 'post',
    data,
  })
}

export const previewProtocol = (projectId, connectionId, data = {}) => {
  return request({
    url: `/data/projects/${projectId}/protocols/${connectionId}/preview`,
    method: 'post',
    data,
  })
}

const unwrapKafkaPayload = (response) => response?.data ?? response
const unwrapHTTPPayload = (response) => response?.data ?? response

export const getHttpRequestGroups = async (projectId, connectionId) => {
  const res = await request({
    url: `/data/projects/${projectId}/http/sources/${connectionId}/request-groups`,
    method: 'get',
  })
  return HttpRequestGroupListSchema.parse(unwrapHTTPPayload(res))
}

export const createHttpRequestGroup = async (projectId, connectionId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/http/sources/${connectionId}/request-groups`,
    method: 'post',
    data,
  })
  return HttpRequestGroupSchema.parse(unwrapHTTPPayload(res))
}

export const updateHttpRequestGroup = async (projectId, groupId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/http/request-groups/${groupId}`,
    method: 'put',
    data,
  })
  return HttpRequestGroupSchema.parse(unwrapHTTPPayload(res))
}

export const deleteHttpRequestGroup = (projectId, groupId) => {
  return request({
    url: `/data/projects/${projectId}/http/request-groups/${groupId}`,
    method: 'delete',
  })
}

export const getHttpRequests = async (projectId, connectionId, params = {}) => {
  const res = await request({
    url: `/data/projects/${projectId}/http/sources/${connectionId}/requests`,
    method: 'get',
    params,
  })
  return HttpRequestListSchema.parse(unwrapHTTPPayload(res))
}

export const getHttpRequest = async (projectId, requestId) => {
  const res = await request({
    url: `/data/projects/${projectId}/http/requests/${requestId}`,
    method: 'get',
  })
  return HttpRequestSchema.parse(unwrapHTTPPayload(res))
}

export const createHttpRequest = async (projectId, connectionId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/http/sources/${connectionId}/requests`,
    method: 'post',
    data,
  })
  return HttpRequestSchema.parse(unwrapHTTPPayload(res))
}

export const updateHttpRequest = async (projectId, requestId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/http/requests/${requestId}`,
    method: 'put',
    data,
  })
  return HttpRequestSchema.parse(unwrapHTTPPayload(res))
}

export const deleteHttpRequest = (projectId, requestId) => {
  return request({
    url: `/data/projects/${projectId}/http/requests/${requestId}`,
    method: 'delete',
  })
}

export const sendHttpRequest = async (projectId, requestId) => {
  const res = await request({
    url: `/data/projects/${projectId}/http/requests/${requestId}/send`,
    method: 'post',
  })
  return HttpSendResponseSchema.parse(unwrapHTTPPayload(res))
}

export const getWebSocketSessionGroups = async (projectId, connectionId) => {
  const res = await request({
    url: `/data/projects/${projectId}/websocket/sources/${connectionId}/session-groups`,
    method: 'get',
  })
  return WebSocketSessionGroupListSchema.parse(unwrapHTTPPayload(res))
}

export const createWebSocketSessionGroup = async (projectId, connectionId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/websocket/sources/${connectionId}/session-groups`,
    method: 'post',
    data,
  })
  return WebSocketSessionGroupSchema.parse(unwrapHTTPPayload(res))
}

export const updateWebSocketSessionGroup = async (projectId, groupId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/websocket/session-groups/${groupId}`,
    method: 'put',
    data,
  })
  return WebSocketSessionGroupSchema.parse(unwrapHTTPPayload(res))
}

export const deleteWebSocketSessionGroup = (projectId, groupId) => {
  return request({
    url: `/data/projects/${projectId}/websocket/session-groups/${groupId}`,
    method: 'delete',
  })
}

export const getWebSocketSessions = async (projectId, connectionId, params = {}) => {
  const res = await request({
    url: `/data/projects/${projectId}/websocket/sources/${connectionId}/sessions`,
    method: 'get',
    params,
  })
  return WebSocketSessionListSchema.parse(unwrapHTTPPayload(res))
}

export const getWebSocketSession = async (projectId, sessionId) => {
  const res = await request({
    url: `/data/projects/${projectId}/websocket/sessions/${sessionId}`,
    method: 'get',
  })
  return WebSocketSessionSchema.parse(unwrapHTTPPayload(res))
}

export const createWebSocketSession = async (projectId, connectionId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/websocket/sources/${connectionId}/sessions`,
    method: 'post',
    data,
  })
  return WebSocketSessionSchema.parse(unwrapHTTPPayload(res))
}

export const updateWebSocketSession = async (projectId, sessionId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/websocket/sessions/${sessionId}`,
    method: 'put',
    data,
  })
  return WebSocketSessionSchema.parse(unwrapHTTPPayload(res))
}

export const deleteWebSocketSession = (projectId, sessionId) => {
  return request({
    url: `/data/projects/${projectId}/websocket/sessions/${sessionId}`,
    method: 'delete',
  })
}

export const connectWebSocketPreview = async (projectId, sessionId) => {
  const res = await request({
    url: `/data/projects/${projectId}/websocket/sessions/${sessionId}/connect-preview`,
    method: 'post',
  })
  return WebSocketPreviewResponseSchema.parse(unwrapHTTPPayload(res))
}

export const getKafkaTopicGroups = async (projectId, connectionId) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/sources/${connectionId}/topic-groups`,
    method: 'get',
  })
  return KafkaTopicGroupListSchema.parse(unwrapKafkaPayload(res))
}

export const createKafkaTopicGroup = async (projectId, connectionId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/sources/${connectionId}/topic-groups`,
    method: 'post',
    data,
  })
  return KafkaTopicGroupSchema.parse(unwrapKafkaPayload(res))
}

export const updateKafkaTopicGroup = async (projectId, groupId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-groups/${groupId}`,
    method: 'put',
    data,
  })
  return KafkaTopicGroupSchema.parse(unwrapKafkaPayload(res))
}

export const deleteKafkaTopicGroup = (projectId, groupId) => {
  return request({
    url: `/data/projects/${projectId}/kafka/topic-groups/${groupId}`,
    method: 'delete',
  })
}

export const getKafkaTopicMappings = async (projectId, connectionId) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/sources/${connectionId}/topic-mappings`,
    method: 'get',
  })
  return KafkaTopicMappingListSchema.parse(unwrapKafkaPayload(res))
}

export const getKafkaTopicMapping = async (projectId, mappingId) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}`,
    method: 'get',
  })
  return KafkaTopicMappingSchema.parse(unwrapKafkaPayload(res))
}

export const createKafkaTopicMapping = async (projectId, connectionId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/sources/${connectionId}/topic-mappings`,
    method: 'post',
    data,
  })
  return KafkaTopicMappingSchema.parse(unwrapKafkaPayload(res))
}

export const updateKafkaTopicMapping = async (projectId, mappingId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}`,
    method: 'put',
    data,
  })
  return KafkaTopicMappingSchema.parse(unwrapKafkaPayload(res))
}

export const deleteKafkaTopicMapping = (projectId, mappingId) => {
  return request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}`,
    method: 'delete',
  })
}

export const previewKafkaConnection = async (projectId, connectionId, data = {}) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/sources/${connectionId}/preview`,
    method: 'post',
    data,
  })
  return KafkaPreviewSchema.parse(unwrapKafkaPayload(res))
}

export const previewKafkaTopicMapping = async (projectId, mappingId, data = {}) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}/preview`,
    method: 'post',
    data,
  })
  return KafkaPreviewSchema.parse(unwrapKafkaPayload(res))
}

export const getKafkaFieldGroups = async (projectId, mappingId) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}/field-groups`,
    method: 'get',
  })
  return KafkaFieldGroupListSchema.parse(unwrapKafkaPayload(res))
}

export const createKafkaFieldGroup = async (projectId, mappingId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}/field-groups`,
    method: 'post',
    data,
  })
  return KafkaFieldGroupSchema.parse(unwrapKafkaPayload(res))
}

export const updateKafkaFieldGroup = async (projectId, groupId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/field-groups/${groupId}`,
    method: 'put',
    data,
  })
  return KafkaFieldGroupSchema.parse(unwrapKafkaPayload(res))
}

export const deleteKafkaFieldGroup = (projectId, groupId) => {
  return request({
    url: `/data/projects/${projectId}/kafka/field-groups/${groupId}`,
    method: 'delete',
  })
}

export const getKafkaFields = async (projectId, mappingId, params = {}) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}/fields`,
    method: 'get',
    params,
  })
  return KafkaFieldListSchema.parse(unwrapKafkaPayload(res))
}

export const createKafkaField = async (projectId, mappingId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}/fields`,
    method: 'post',
    data,
  })
  return KafkaFieldSchema.parse(unwrapKafkaPayload(res))
}

export const createKafkaFieldsBatch = async (projectId, mappingId, fields) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}/fields/batch`,
    method: 'post',
    data: { fields },
  })
  return KafkaFieldListSchema.parse(unwrapKafkaPayload(res))
}

export const updateKafkaField = async (projectId, fieldId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/fields/${fieldId}`,
    method: 'put',
    data,
  })
  return KafkaFieldSchema.parse(unwrapKafkaPayload(res))
}

export const deleteKafkaField = (projectId, fieldId) => {
  return request({
    url: `/data/projects/${projectId}/kafka/fields/${fieldId}`,
    method: 'delete',
  })
}

export const toggleKafkaField = async (projectId, fieldId, enabled) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/fields/${fieldId}/toggle`,
    method: 'patch',
    data: { enabled },
  })
  return KafkaFieldSchema.parse(unwrapKafkaPayload(res))
}

export const getRealtimeStoreKeys = (projectId, connectionId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/realtime-stores/${connectionId}/keys`,
    method: 'get',
    params,
  })
}

export const getRealtimeStoreKey = (projectId, connectionId, key) => {
  return request({
    url: `/data/projects/${projectId}/realtime-stores/${connectionId}/key`,
    method: 'get',
    params: { key },
  })
}

export const saveRealtimeStoreKey = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/realtime-stores/${connectionId}/key`,
    method: 'put',
    data,
  })
}

export const renameRealtimeStoreKey = (projectId, connectionId, key, data) => {
  return request({
    url: `/data/projects/${projectId}/realtime-stores/${connectionId}/key/rename`,
    method: 'patch',
    params: { key },
    data,
  })
}

export const deleteRealtimeStoreKey = (projectId, connectionId, key) => {
  return request({
    url: `/data/projects/${projectId}/realtime-stores/${connectionId}/key`,
    method: 'delete',
    params: { key },
  })
}

export const createRealtimeStoreKeyDatapoint = (projectId, connectionId, key, data) => {
  return request({
    url: `/data/projects/${projectId}/realtime-stores/${connectionId}/key/datapoint`,
    method: 'post',
    params: { key },
    data,
  })
}

export const batchCreateRealtimeStoreKeyDatapoints = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/realtime-stores/${connectionId}/keys/datapoints`,
    method: 'post',
    data,
  })
}

export const getConnectionTables = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/tables`,
    method: 'get',
  })
}

export const createConnectionTable = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/tables`,
    method: 'post',
    data,
  })
}

export const updateConnectionTableStructure = (projectId, connectionId, tableName, data) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/tables/${encodeURIComponent(tableName)}/structure`,
    method: 'patch',
    data,
  })
}

export const renameConnectionTable = (projectId, connectionId, tableName, data) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/tables/${encodeURIComponent(tableName)}`,
    method: 'put',
    data,
  })
}

export const deleteConnectionTable = (projectId, connectionId, tableName) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/tables/${encodeURIComponent(tableName)}`,
    method: 'delete',
  })
}

export const getTableData = (projectId, connectionId, tableName, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/tables/${encodeURIComponent(tableName)}/data`,
    method: 'get',
    params,
  })
}

export const getWorkbenchGroups = (projectId, connectionId, scope) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/workbench-groups`,
    method: 'get',
    params: { scope },
  })
}

export const createWorkbenchGroup = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/workbench-groups`,
    method: 'post',
    data,
  })
}

export const updateWorkbenchGroup = (projectId, groupId, data) => {
  return request({
    url: `/data/projects/${projectId}/workbench-groups/${groupId}`,
    method: 'put',
    data,
  })
}

export const deleteWorkbenchGroup = (projectId, groupId) => {
  return request({
    url: `/data/projects/${projectId}/workbench-groups/${groupId}`,
    method: 'delete',
  })
}

export const moveQueryToWorkbenchGroup = (projectId, queryId, groupId) => {
  return request({
    url: `/data/projects/${projectId}/queries/${queryId}/group`,
    method: 'patch',
    data: { groupId },
  })
}

export const getTableGroupMembers = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/table-group-members`,
    method: 'get',
  })
}

export const moveTableToWorkbenchGroup = (projectId, connectionId, tableName, groupId) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/tables/${encodeURIComponent(tableName)}/group`,
    method: 'patch',
    data: { groupId },
  })
}

/**
 * 获取表结构信息
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 * @param {string} tableName - 表名
 */
export const getTableStructure = (projectId, connectionId, tableName) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/tables/${encodeURIComponent(tableName)}/structure`,
    method: 'get',
  })
}

export const executeBuiltinRelationSql = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/execute-sql`,
    method: 'post',
    data,
  })
}

export const queryBuiltinTimeseries = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/execute-sql`,
    method: 'post',
    data,
  })
}

export const sampleBuiltinTimeseries = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/builtin/timeseries/sample`,
    method: 'post',
    data,
  })
}

export const publishMqttMessage = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/connections/${connectionId}/publish`,
    method: 'post',
    data,
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
    data,
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
    method: 'delete',
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
    data: { status },
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
    params,
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
    data,
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
    data,
  })
}

/**
 * 删除数据查询
 * @param {string} id - 查询ID
 */
export const deleteQuery = (id) => {
  return request({
    url: `/data/queries/${id}`,
    method: 'delete',
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
    data: { parameters },
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
    data: { sql, parameters },
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
    data,
  })
}

// ===========================================
// 数据点相关API
// ===========================================

/**
 * 获取数据点列表
 * @param {string} projectId - 工程ID
 * @param {object} params - 查询参数
 */
export const getDataPoints = (projectId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/datapoints`,
    method: 'get',
    params,
  })
}

export const getDataPoint = (projectId, datapointId) => {
  return request({
    url: `/data/projects/${projectId}/datapoints/${datapointId}`,
    method: 'get',
  })
}

export const getDataPointStatuses = (projectId, data = {}) => {
  return request({
    url: `/data/projects/${projectId}/datapoints/status`,
    method: 'post',
    data,
  })
}

/**
 * 更新数据点
 * @param {string} projectId - 工程ID
 * @param {string} datapointId - 数据点ID
 * @param {object} data - 数据点更新数据
 */
export const updateDataPoint = (projectId, datapointId, data) => {
  return request({
    url: `/data/projects/${projectId}/datapoints/${datapointId}`,
    method: 'put',
    data,
  })
}

/**
 * 更新数据点运行态权限
 * @param {string} projectId - 工程ID
 * @param {string} datapointId - 数据点ID
 * @param {object} data - 运行态权限数据
 */
export const updateDatapointRuntimePermissions = (projectId, datapointId, data) => {
  return request({
    url: `/data/projects/${projectId}/datapoints/${datapointId}/runtime-permissions`,
    method: 'put',
    data,
  })
}

/**
 * 删除数据点
 * @param {string} projectId - 工程ID
 * @param {string} datapointId - 数据点ID
 */
export const deleteDataPoint = (projectId, datapointId) => {
  return request({
    url: `/data/projects/${projectId}/datapoints/${datapointId}`,
    method: 'delete',
  })
}

/**
 * 批量删除失效数据点
 * @param {string} projectId - 工程ID
 * @param {string[]} datapointIds - 数据点ID列表
 */
export const deleteDataPointsBatch = (projectId, datapointIds) => {
  return request({
    url: `/data/projects/${projectId}/datapoints/delete-batch`,
    method: 'post',
    data: {
      ids: datapointIds,
    },
  })
}

export const deleteDataPointsByFilter = (projectId, filter = {}) => {
  return request({
    url: `/data/projects/${projectId}/datapoints/delete-by-filter`,
    method: 'post',
    data: { filter },
  })
}

export const appendDataPointTagsByFilter = (projectId, filter = {}, tags = []) => {
  return request({
    url: `/data/projects/${projectId}/datapoints/tags-by-filter`,
    method: 'post',
    data: { filter, tags },
  })
}

// ===========================================
// Preview Session 相关API
// ===========================================

/**
 * 创建 preview session
 * @param {string} projectId - 工程 ID
 * @returns {Promise} preview session 响应
 */
export const createPreviewSession = (projectId) => {
  return request({
    url: `/data/projects/${projectId}/preview/sessions`,
    method: 'post',
    data: {},
  })
}

/**
 * 续期 preview session
 * @param {string} sessionId - preview session ID
 * @returns {Promise} 续期结果
 */
export const heartbeatPreviewSession = (sessionId) => {
  return request({
    url: `/data/preview/sessions/${sessionId}/heartbeat`,
    method: 'post',
  })
}

/**
 * 关闭 preview session
 * @param {string} sessionId - preview session ID
 * @returns {Promise} 关闭结果
 */
export const deletePreviewSession = (sessionId) => {
  return request({
    url: `/data/preview/sessions/${sessionId}`,
    method: 'delete',
  })
}

// ===========================================
// MQTT订阅相关API
// ===========================================

const normalizeMqttListPayload = (payload, legacyKey = '') => {
  const data = payload?.data ?? payload ?? {}
  const list = Array.isArray(data.list)
    ? data.list
    : legacyKey && Array.isArray(data[legacyKey])
      ? data[legacyKey]
      : Array.isArray(data)
        ? data
        : []
  const pagination = data.pagination || {
    page: 1,
    pageSize: list.length,
    total: list.length,
    totalPages: list.length > 0 ? 1 : 0,
  }
  return { list, pagination }
}

/**
 * 获取MQTT订阅列表
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 */
export const getMqttSubscriptions = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/connections/${connectionId}/subscriptions`,
    method: 'get',
  }).then((response) => ({
    ...response,
    data: normalizeMqttListPayload(response, 'subscriptions'),
  }))
}

/**
 * 获取MQTT订阅分组树
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 */
export const getMqttSubscriptionGroups = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/connections/${connectionId}/subscription-groups`,
    method: 'get',
  }).then((response) => ({
    ...response,
    data: normalizeMqttListPayload(response, 'groups'),
  }))
}

/**
 * 创建MQTT订阅分组
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 * @param {object} data - 分组数据
 */
export const createMqttSubscriptionGroup = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/connections/${connectionId}/subscription-groups`,
    method: 'post',
    data,
  })
}

/**
 * 更新MQTT订阅分组
 * @param {string} projectId - 工程ID
 * @param {string} groupId - 分组ID
 * @param {object} data - 更新数据
 */
export const updateMqttSubscriptionGroup = (projectId, groupId, data) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscription-groups/${groupId}`,
    method: 'put',
    data,
  })
}

/**
 * 删除MQTT订阅分组
 * @param {string} projectId - 工程ID
 * @param {string} groupId - 分组ID
 */
export const deleteMqttSubscriptionGroup = (projectId, groupId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscription-groups/${groupId}`,
    method: 'delete',
  })
}

/**
 * 获取单个MQTT订阅
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 */
export const getMqttSubscription = (projectId, subscriptionId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}`,
    method: 'get',
  })
}

/**
 * 创建MQTT订阅
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 * @param {object} data - 订阅数据
 */
export const createMqttSubscription = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/connections/${connectionId}/subscriptions`,
    method: 'post',
    data,
  })
}

/**
 * 更新MQTT订阅
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 * @param {object} data - 更新数据
 */
export const updateMqttSubscription = (projectId, subscriptionId, data) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}`,
    method: 'put',
    data,
  })
}

/**
 * 保存MQTT订阅默认批量解析规则
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 * @param {object} rule - 默认批量解析规则
 */
export const updateMqttSubscriptionDefaultBatchParseRule = (projectId, subscriptionId, rule) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/default-batch-parse-rule`,
    method: 'put',
    data: { rule },
  })
}

/**
 * 删除MQTT订阅
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 */
export const deleteMqttSubscription = (projectId, subscriptionId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}`,
    method: 'delete',
  })
}

/**
 * 获取MQTT订阅的消息列表
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 */
export const getMqttSubscriptionMessages = (projectId, subscriptionId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/messages`,
    method: 'get',
    params,
  }).then((response) => ({
    ...response,
    data: normalizeMqttListPayload(response, 'messages'),
  }))
}

/**
 * 清空MQTT订阅的消息缓存
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 */
export const clearMqttSubscriptionMessages = (projectId, subscriptionId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/messages`,
    method: 'delete',
  })
}

/**
 * 启动MQTT连接
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 */
export const startMqttConnection = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/connections/${connectionId}/start`,
    method: 'post',
  })
}

/**
 * 停止MQTT连接
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 */
export const stopMqttConnection = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/connections/${connectionId}/stop`,
    method: 'post',
  })
}

// ===========================================
// MQTT Tag (变量) 相关API
// ===========================================

/**
 * 获取订阅的所有Tag
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 * @param {object} params - 查询参数
 */
export const getMqttTags = (projectId, subscriptionId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/tags`,
    method: 'get',
    params,
  }).then((response) => ({
    ...response,
    data: normalizeMqttListPayload(response, 'tags'),
  }))
}

/**
 * 获取项目的所有Tag
 * @param {string} projectId - 工程ID
 * @param {object} params - 查询参数
 */
export const getProjectMqttTags = (projectId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/tags`,
    method: 'get',
    params,
  }).then((response) => ({
    ...response,
    data: normalizeMqttListPayload(response, 'tags'),
  }))
}

/**
 * 获取Tag详情
 * @param {string} tagId - Tag ID
 */
export const getMqttTag = (projectId, tagId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/tags/${tagId}`,
    method: 'get',
  })
}

/**
 * 创建Tag
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 * @param {object} data - Tag数据
 */
export const createMqttTag = (projectId, subscriptionId, data) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/tags`,
    method: 'post',
    data,
  })
}

/**
 * 批量创建Tag
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 * @param {array} tags - Tag数据数组
 */
export const createMqttTagsBatch = (projectId, subscriptionId, tags) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/tags/batch`,
    method: 'post',
    data: { tags },
  })
}

/**
 * 更新Tag
 * @param {string} tagId - Tag ID
 * @param {object} data - 更新数据
 */
export const updateMqttTag = (projectId, tagId, data) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/tags/${tagId}`,
    method: 'put',
    data,
  })
}

/**
 * 删除Tag
 * @param {string} tagId - Tag ID
 */
export const deleteMqttTag = (projectId, tagId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/tags/${tagId}`,
    method: 'delete',
  })
}

/**
 * 批量删除Tag
 * @param {string} projectId - 工程ID
 * @param {string[]} tagIds - Tag ID数组
 */
export const deleteMqttTagsBatch = (projectId, tagIds) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/tags/delete-batch`,
    method: 'post',
    data: { tagIds },
  })
}

/**
 * 按订阅筛选条件批量删除Tag
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 * @param {object} filters - 当前筛选条件
 */
export const deleteMqttTagsByFilter = (projectId, subscriptionId, filters = {}) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/tags/delete-filtered`,
    method: 'post',
    data: filters,
  })
}

/**
 * 更新Tags顺序
 * @param {array} tagIds - Tag ID数组
 */
export const updateMqttTagsOrder = (projectId, tagIds) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/tags/order`,
    method: 'put',
    data: { tagIds },
  })
}

/**
 * 获取Tag的当前值
 * @param {string} tagId - Tag ID
 */
export const getMqttTagValue = (projectId, tagId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/tags/${tagId}/value`,
    method: 'get',
  })
}

/**
 * 获取多个Tag的当前值
 * @param {array} tagIds - Tag ID数组
 */
export const getMqttTagValues = (projectId, tagIds, options = {}) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/tags/values`,
    method: 'post',
    data: { tagIds, ...options },
  })
}

// ===========================================
// Compute 鐩稿叧 API
// ===========================================

/**
 * 鍒涘缓璁＄畻鍗曞厓
 * @param {string} projectId - 宸ョ▼ID
 * @param {object} data - 璁＄畻鍗曞厓鏁版嵁
 */
export const createComputeUnit = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/compute-units`,
    method: 'post',
    data,
  })
}

/**
 * 鎵ц璁＄畻鍗曞厓
 * @param {string} projectId - 宸ョ▼ID
 * @param {string} id - 璁＄畻鍗曞厓ID
 * @param {object} input - 杈撳叆鍙傛暟
 */
export const runComputeUnit = (projectId, id, input = {}) => {
  return request({
    url: `/data/projects/${projectId}/compute-units/${id}/run`,
    method: 'post',
    data: { input },
  })
}

/**
 * 璋冭瘯璁＄畻鍗曞厓
 * @param {string} projectId - 宸ョ▼ID
 * @param {string} id - 璁＄畻鍗曞厓ID
 * @param {object} input - 杈撳叆鍙傛暟
 */
export const debugComputeUnit = (projectId, id, input = {}) => {
  return request({
    url: `/data/projects/${projectId}/compute-units/${id}/debug`,
    method: 'post',
    data: { input },
  })
}

export default {
  getConnections,
  getConnection,
  createConnection,
  createMqttConnection,
  testConnection,
  testMqttConnection,
  createKafkaConfig,
  createHttpConfig,
  createWebSocketConfig,
  createRedisConfig,
  getRealtimeStoreKeys,
  getRealtimeStoreKey,
  saveRealtimeStoreKey,
  renameRealtimeStoreKey,
  deleteRealtimeStoreKey,
  createRealtimeStoreKeyDatapoint,
  batchCreateRealtimeStoreKeyDatapoints,
  createTdengineConfig,
  validateOpcdaContract,
  previewProtocol,
  getHttpRequestGroups,
  createHttpRequestGroup,
  updateHttpRequestGroup,
  deleteHttpRequestGroup,
  getHttpRequests,
  getHttpRequest,
  createHttpRequest,
  updateHttpRequest,
  deleteHttpRequest,
  sendHttpRequest,
  getWebSocketSessionGroups,
  createWebSocketSessionGroup,
  updateWebSocketSessionGroup,
  deleteWebSocketSessionGroup,
  getWebSocketSessions,
  getWebSocketSession,
  createWebSocketSession,
  updateWebSocketSession,
  deleteWebSocketSession,
  connectWebSocketPreview,
  getKafkaTopicGroups,
  createKafkaTopicGroup,
  updateKafkaTopicGroup,
  deleteKafkaTopicGroup,
  getKafkaTopicMappings,
  getKafkaTopicMapping,
  createKafkaTopicMapping,
  updateKafkaTopicMapping,
  deleteKafkaTopicMapping,
  previewKafkaConnection,
  previewKafkaTopicMapping,
  getKafkaFields,
  getKafkaFieldGroups,
  createKafkaFieldGroup,
  updateKafkaFieldGroup,
  deleteKafkaFieldGroup,
  createKafkaField,
  createKafkaFieldsBatch,
  updateKafkaField,
  deleteKafkaField,
  toggleKafkaField,
  getConnectionTables,
  createConnectionTable,
  updateConnectionTableStructure,
  renameConnectionTable,
  deleteConnectionTable,
  getTableData,
  getTableStructure,
  getWorkbenchGroups,
  createWorkbenchGroup,
  updateWorkbenchGroup,
  deleteWorkbenchGroup,
  moveQueryToWorkbenchGroup,
  getTableGroupMembers,
  moveTableToWorkbenchGroup,
  executeBuiltinRelationSql,
  queryBuiltinTimeseries,
  sampleBuiltinTimeseries,
  publishMqttMessage,
  updateConnection,
  deleteConnection,
  updateConnectionStatus,
  getQueries,
  createQuery,
  updateQuery,
  updateDataPoint,
  executeQuery,
  executeSql,
  deleteQuery,
  // 数据点相关
  getDataPoints,
  getDataPoint,
  getDataPointStatuses,
  updateDatapointRuntimePermissions,
  deleteDataPoint,
  deleteDataPointsBatch,
  deleteDataPointsByFilter,
  appendDataPointTagsByFilter,
  createPreviewSession,
  heartbeatPreviewSession,
  deletePreviewSession,
  // MQTT订阅相关
  getMqttSubscriptions,
  getMqttSubscriptionGroups,
  createMqttSubscriptionGroup,
  updateMqttSubscriptionGroup,
  deleteMqttSubscriptionGroup,
  getMqttSubscription,
  createMqttSubscription,
  updateMqttSubscription,
  updateMqttSubscriptionDefaultBatchParseRule,
  deleteMqttSubscription,
  getMqttSubscriptionMessages,
  clearMqttSubscriptionMessages,
  startMqttConnection,
  stopMqttConnection,
  // MQTT Tag相关
  getMqttTags,
  getProjectMqttTags,
  getMqttTag,
  createMqttTag,
  createMqttTagsBatch,
  updateMqttTag,
  deleteMqttTag,
  updateMqttTagsOrder,
  getMqttTagValue,
  getMqttTagValues,
  createComputeUnit,
  runComputeUnit,
  debugComputeUnit,
}
