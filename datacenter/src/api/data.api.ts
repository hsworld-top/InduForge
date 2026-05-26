// @ts-nocheck
import request from '@/utils/request'
import {
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

export const createOpcuaConfig = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/opcua/configs`,
    method: 'post',
    data,
  })
}

export const createS7Config = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/s7/configs`,
    method: 'post',
    data,
  })
}

export const createModbusConfig = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/modbus/configs`,
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

export const getKafkaTopicGroups = async (projectId, connectionId) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/sources/${connectionId}/topic-groups`,
    method: 'get',
  })
  return KafkaTopicGroupListSchema.parse(res)
}

export const createKafkaTopicGroup = async (projectId, connectionId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/sources/${connectionId}/topic-groups`,
    method: 'post',
    data,
  })
  return KafkaTopicGroupSchema.parse(res)
}

export const updateKafkaTopicGroup = async (projectId, groupId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-groups/${groupId}`,
    method: 'put',
    data,
  })
  return KafkaTopicGroupSchema.parse(res)
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
  return KafkaTopicMappingListSchema.parse(res)
}

export const getKafkaTopicMapping = async (projectId, mappingId) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}`,
    method: 'get',
  })
  return KafkaTopicMappingSchema.parse(res)
}

export const createKafkaTopicMapping = async (projectId, connectionId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/sources/${connectionId}/topic-mappings`,
    method: 'post',
    data,
  })
  return KafkaTopicMappingSchema.parse(res)
}

export const updateKafkaTopicMapping = async (projectId, mappingId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}`,
    method: 'put',
    data,
  })
  return KafkaTopicMappingSchema.parse(res)
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
  return KafkaPreviewSchema.parse(res)
}

export const previewKafkaTopicMapping = async (projectId, mappingId, data = {}) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}/preview`,
    method: 'post',
    data,
  })
  return KafkaPreviewSchema.parse(res)
}

export const getKafkaFields = async (projectId, mappingId) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}/fields`,
    method: 'get',
  })
  return KafkaFieldListSchema.parse(res)
}

export const createKafkaField = async (projectId, mappingId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}/fields`,
    method: 'post',
    data,
  })
  return KafkaFieldSchema.parse(res)
}

export const createKafkaFieldsBatch = async (projectId, mappingId, fields) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}/fields/batch`,
    method: 'post',
    data: { fields },
  })
  return KafkaFieldListSchema.parse(res)
}

export const updateKafkaField = async (projectId, fieldId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/fields/${fieldId}`,
    method: 'put',
    data,
  })
  return KafkaFieldSchema.parse(res)
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
  return KafkaFieldSchema.parse(res)
}

const normalizeOpcuaListPayload = (payload, legacyKey = '') => {
  const data = payload?.data ?? payload ?? {}
  const list = Array.isArray(data.list)
    ? data.list
    : legacyKey && Array.isArray(data[legacyKey])
      ? data[legacyKey]
      : Array.isArray(data)
        ? data
        : []
  return { list, pagination: data.pagination }
}

const normalizeModbusListPayload = normalizeOpcuaListPayload
const normalizeS7ListPayload = normalizeOpcuaListPayload

export const getOpcuaNodeGroups = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/node-groups`,
    method: 'get',
  }).then((response) => ({
    ...response,
    data: normalizeOpcuaListPayload(response, 'groups'),
  }))
}

export const createOpcuaNodeGroup = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/node-groups`,
    method: 'post',
    data,
  })
}

export const updateOpcuaNodeGroup = (projectId, connectionId, groupId, data) => {
  return request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/node-groups/${groupId}`,
    method: 'put',
    data,
  })
}

export const deleteOpcuaNodeGroup = (projectId, connectionId, groupId) => {
  return request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/node-groups/${groupId}`,
    method: 'delete',
  })
}

export const getOpcuaNodes = (projectId, connectionId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/nodes`,
    method: 'get',
    params,
  }).then((response) => ({
    ...response,
    data: normalizeOpcuaListPayload(response, 'nodes'),
  }))
}

export const createOpcuaNode = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/nodes`,
    method: 'post',
    data,
  })
}

export const batchImportOpcuaNodes = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/nodes/batch-import`,
    method: 'post',
    data,
  })
}

export const updateOpcuaNode = (projectId, connectionId, nodeId, data) => {
  return request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/nodes/${nodeId}`,
    method: 'put',
    data,
  })
}

export const deleteOpcuaNode = (projectId, connectionId, nodeId) => {
  return request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/nodes/${nodeId}`,
    method: 'delete',
  })
}

export const validateOpcuaModel = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/validate-model`,
    method: 'post',
  })
}

export const previewOpcuaNodes = (projectId, connectionId, data = {}) => {
  return request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/preview`,
    method: 'post',
    data,
  })
}

export const createOpcuaDevSession = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/sessions`,
    method: 'post',
  })
}

export const closeOpcuaDevSession = (projectId, connectionId, sessionId) => {
  return request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/sessions/${sessionId}`,
    method: 'delete',
  })
}

export const browseOpcuaDevSession = (projectId, connectionId, sessionId) => {
  return request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/sessions/${sessionId}/browse`,
    method: 'get',
  })
}

export const readOpcuaDevSession = (projectId, connectionId, sessionId, data = {}) => {
  return request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/sessions/${sessionId}/read`,
    method: 'post',
    data,
  })
}

export const subscribeOpcuaDevSession = (projectId, connectionId, sessionId, data = {}) => {
  return request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/sessions/${sessionId}/subscribe`,
    method: 'post',
    data,
  })
}

export const getModbusRegisterGroups = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/register-groups`,
    method: 'get',
  }).then((response) => ({
    ...response,
    data: normalizeModbusListPayload(response, 'groups'),
  }))
}

export const createModbusRegisterGroup = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/register-groups`,
    method: 'post',
    data,
  })
}

export const updateModbusRegisterGroup = (projectId, connectionId, groupId, data) => {
  return request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/register-groups/${groupId}`,
    method: 'put',
    data,
  })
}

export const deleteModbusRegisterGroup = (projectId, connectionId, groupId) => {
  return request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/register-groups/${groupId}`,
    method: 'delete',
  })
}

export const getModbusRegisters = (projectId, connectionId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/registers`,
    method: 'get',
    params,
  }).then((response) => ({
    ...response,
    data: normalizeModbusListPayload(response, 'registers'),
  }))
}

export const createModbusRegister = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/registers`,
    method: 'post',
    data,
  })
}

export const batchImportModbusRegisters = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/registers/batch-import`,
    method: 'post',
    data,
  })
}

export const updateModbusRegister = (projectId, connectionId, registerId, data) => {
  return request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/registers/${registerId}`,
    method: 'put',
    data,
  })
}

export const deleteModbusRegister = (projectId, connectionId, registerId) => {
  return request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/registers/${registerId}`,
    method: 'delete',
  })
}

export const validateModbusModel = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/validate-model`,
    method: 'post',
  })
}

export const previewModbusRegisters = (projectId, connectionId, data = {}) => {
  return request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/preview`,
    method: 'post',
    data,
  })
}

export const getModbusReadPlanEstimate = (projectId, connectionId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/read-plan-estimate`,
    method: 'get',
    params,
  })
}

export const createModbusDevSession = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/sessions`,
    method: 'post',
  })
}

export const closeModbusDevSession = (projectId, connectionId, sessionId) => {
  return request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/sessions/${sessionId}`,
    method: 'delete',
  })
}

export const readModbusDevSession = (projectId, connectionId, sessionId, data = {}) => {
  return request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/sessions/${sessionId}/read`,
    method: 'post',
    data,
  })
}

export const pollModbusDevSession = (projectId, connectionId, sessionId, data = {}) => {
  return request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/sessions/${sessionId}/poll`,
    method: 'post',
    data,
  })
}

export const getS7Profile = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/profile`,
    method: 'get',
  })
}

export const updateS7Profile = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/profile`,
    method: 'put',
    data,
  })
}

export const getS7VariableGroups = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/variable-groups`,
    method: 'get',
  }).then((response) => ({
    ...response,
    data: normalizeS7ListPayload(response, 'groups'),
  }))
}

export const createS7VariableGroup = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/variable-groups`,
    method: 'post',
    data,
  })
}

export const updateS7VariableGroup = (projectId, connectionId, groupId, data) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/variable-groups/${groupId}`,
    method: 'put',
    data,
  })
}

export const deleteS7VariableGroup = (projectId, connectionId, groupId) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/variable-groups/${groupId}`,
    method: 'delete',
  })
}

export const getS7Variables = (projectId, connectionId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/variables`,
    method: 'get',
    params,
  }).then((response) => ({
    ...response,
    data: normalizeS7ListPayload(response, 'variables'),
  }))
}

export const createS7Variable = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/variables`,
    method: 'post',
    data,
  })
}

export const batchImportS7Variables = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/variables/batch-import`,
    method: 'post',
    data,
  })
}

export const updateS7Variable = (projectId, connectionId, variableId, data) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/variables/${variableId}`,
    method: 'put',
    data,
  })
}

export const deleteS7Variable = (projectId, connectionId, variableId) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/variables/${variableId}`,
    method: 'delete',
  })
}

export const validateS7Model = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/validate-model`,
    method: 'post',
  })
}

export const previewS7Variables = (projectId, connectionId, data = {}) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/preview`,
    method: 'post',
    data,
  })
}

export const getS7ReadPlanEstimate = (projectId, connectionId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/read-plan-estimate`,
    method: 'get',
    params,
  })
}

export const createS7DevSession = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/sessions`,
    method: 'post',
  })
}

export const closeS7DevSession = (projectId, connectionId, sessionId) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/sessions/${sessionId}`,
    method: 'delete',
  })
}

export const readS7DevSession = (projectId, connectionId, sessionId, data = {}) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/sessions/${sessionId}/read`,
    method: 'post',
    data,
  })
}

export const pollS7DevSession = (projectId, connectionId, sessionId, data = {}) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/sessions/${sessionId}/poll`,
    method: 'post',
    data,
  })
}

export const stopS7DevSessionPoll = (projectId, connectionId, sessionId) => {
  return request({
    url: `/data/projects/${projectId}/s7/${connectionId}/sessions/${sessionId}/poll`,
    method: 'delete',
  })
}

export const getRedisKeys = (projectId, connectionId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/redis/keys`,
    method: 'get',
    params,
  })
}

export const getRedisValue = (projectId, connectionId, key) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/redis/value`,
    method: 'get',
    params: { key },
  })
}

export const executeRedisCommand = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/redis/command`,
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

export const getBuiltinRealtimeKeys = (projectId, connectionId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/realtime/keys`,
    method: 'get',
    params,
  })
}

export const setBuiltinRealtimeKey = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/realtime/keys`,
    method: 'post',
    data,
  })
}

export const getBuiltinRealtimeKey = (projectId, connectionId, key) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/realtime/keys/${encodeURIComponent(key)}`,
    method: 'get',
  })
}

export const deleteBuiltinRealtimeKey = (projectId, connectionId, key) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/realtime/keys/${encodeURIComponent(key)}`,
    method: 'delete',
  })
}

export const publishBuiltinMessage = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/message/publish`,
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

export const getBuiltinMessageTopics = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/message/topics`,
    method: 'get',
  })
}

export const createBuiltinMessageTopic = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/message/topics`,
    method: 'post',
    data,
  })
}

export const getBuiltinMessageVariables = (projectId, connectionId, topicId) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/message/topics/${topicId}/variables`,
    method: 'get',
  })
}

export const createBuiltinMessageVariable = (projectId, connectionId, topicId, data) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/message/topics/${topicId}/variables`,
    method: 'post',
    data,
  })
}

export const createBuiltinMessagePreviewSession = (projectId, data = {}) => {
  return request({
    url: `/data/projects/${projectId}/builtin/message/preview-session`,
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
// MQTT 变量组相关API
// ===========================================

/**
 * 获取订阅的所有变量组
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 * @param {object} params - 查询参数
 */
export const getMqttTagGroups = (projectId, subscriptionId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/tag-groups`,
    method: 'get',
    params,
  }).then((response) => ({
    ...response,
    data: normalizeMqttListPayload(response, 'groups'),
  }))
}

/**
 * 获取单个变量组
 * @param {string} groupId - 变量组ID
 */
export const getMqttTagGroup = (projectId, groupId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/tag-groups/${groupId}`,
    method: 'get',
  })
}

/**
 * 创建变量组
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 * @param {object} data - 变量组数据
 */
export const createMqttTagGroup = (projectId, subscriptionId, data) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/tag-groups`,
    method: 'post',
    data,
  })
}

/**
 * 更新变量组
 * @param {string} groupId - 变量组ID
 * @param {object} data - 更新数据
 */
export const updateMqttTagGroup = (projectId, groupId, data) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/tag-groups/${groupId}`,
    method: 'put',
    data,
  })
}

/**
 * 删除变量组
 * @param {string} groupId - 变量组ID
 */
export const deleteMqttTagGroup = (projectId, groupId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/tag-groups/${groupId}`,
    method: 'delete',
  })
}

/**
 * 更新变量组顺序
 * @param {array} groups - 变量组数组，包含id和order
 */
export const updateMqttTagGroupsOrder = (projectId, groups) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/tag-groups/order`,
    method: 'put',
    data: { groups },
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
export const getMqttTagValues = (projectId, tagIds) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/tags/values`,
    method: 'post',
    data: { tagIds },
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
  createConnection,
  createMqttConnection,
  testConnection,
  testMqttConnection,
  createKafkaConfig,
  createHttpConfig,
  createWebSocketConfig,
  createRedisConfig,
  getRedisKeys,
  getRedisValue,
  executeRedisCommand,
  createOpcuaConfig,
  createS7Config,
  createModbusConfig,
  createTdengineConfig,
  validateOpcdaContract,
  previewProtocol,
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
  createKafkaField,
  createKafkaFieldsBatch,
  updateKafkaField,
  deleteKafkaField,
  toggleKafkaField,
  getOpcuaNodeGroups,
  createOpcuaNodeGroup,
  updateOpcuaNodeGroup,
  deleteOpcuaNodeGroup,
  getOpcuaNodes,
  createOpcuaNode,
  batchImportOpcuaNodes,
  updateOpcuaNode,
  deleteOpcuaNode,
  validateOpcuaModel,
  previewOpcuaNodes,
  createOpcuaDevSession,
  closeOpcuaDevSession,
  browseOpcuaDevSession,
  readOpcuaDevSession,
  subscribeOpcuaDevSession,
  getModbusRegisterGroups,
  createModbusRegisterGroup,
  updateModbusRegisterGroup,
  deleteModbusRegisterGroup,
  getModbusRegisters,
  createModbusRegister,
  batchImportModbusRegisters,
  updateModbusRegister,
  deleteModbusRegister,
  validateModbusModel,
  previewModbusRegisters,
  getModbusReadPlanEstimate,
  createModbusDevSession,
  closeModbusDevSession,
  readModbusDevSession,
  pollModbusDevSession,
  getS7Profile,
  updateS7Profile,
  getS7VariableGroups,
  createS7VariableGroup,
  updateS7VariableGroup,
  deleteS7VariableGroup,
  getS7Variables,
  createS7Variable,
  batchImportS7Variables,
  updateS7Variable,
  deleteS7Variable,
  validateS7Model,
  previewS7Variables,
  getS7ReadPlanEstimate,
  createS7DevSession,
  closeS7DevSession,
  readS7DevSession,
  pollS7DevSession,
  stopS7DevSessionPoll,
  getConnectionTables,
  createConnectionTable,
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
  setBuiltinRealtimeKey,
  getBuiltinRealtimeKey,
  deleteBuiltinRealtimeKey,
  publishBuiltinMessage,
  getBuiltinMessageTopics,
  createBuiltinMessageTopic,
  getBuiltinMessageVariables,
  createBuiltinMessageVariable,
  publishMqttMessage,
  createBuiltinMessagePreviewSession,
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
  updateDatapointRuntimePermissions,
  deleteDataPoint,
  deleteDataPointsBatch,
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
  deleteMqttSubscription,
  getMqttSubscriptionMessages,
  startMqttConnection,
  stopMqttConnection,
  // MQTT 变量组相关
  getMqttTagGroups,
  getMqttTagGroup,
  createMqttTagGroup,
  updateMqttTagGroup,
  deleteMqttTagGroup,
  updateMqttTagGroupsOrder,
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
