import request from "@/utils/request";

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
    method: "get",
    params,
  });
};

/**
 * 创建数据连接
 * @param {string} projectId - 工程ID
 * @param {object} data - 连接数据
 */
export const createConnection = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/connections`,
    method: "post",
    data,
  });
};

export const testConnection = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/connections/test`,
    method: "post",
    data,
  });
};

export const getConnectionTables = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/tables`,
    method: "get",
  });
};

export const getTableData = (
  projectId,
  connectionId,
  tableName,
  params = {},
) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/tables/${tableName}/data`,
    method: "get",
    params,
  });
};

/**
 * 获取表结构信息
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 * @param {string} tableName - 表名
 */
export const getTableStructure = (projectId, connectionId, tableName) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/tables/${tableName}/structure`,
    method: "get",
  });
};

/**
 * 更新数据连接
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 * @param {object} data - 更新数据
 */
export const updateConnection = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}`,
    method: "put",
    data,
  });
};

/**
 * 删除数据连接
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 */
export const deleteConnection = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}`,
    method: "delete",
  });
};

/**
 * 更新连接状态
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 * @param {string} status - 状态 (connected, disconnected, error, unknown)
 */
export const updateConnectionStatus = (projectId, connectionId, status) => {
  return request({
    url: `/data/projects/${projectId}/connections/${connectionId}/status`,
    method: "patch",
    data: { status },
  });
};

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
    method: "get",
    params,
  });
};

/**
 * 创建数据查询
 * @param {string} projectId - 工程ID
 * @param {object} data - 查询数据
 */
export const createQuery = (projectId, data) => {
  return request({
    url: `/data/projects/${projectId}/queries`,
    method: "post",
    data,
  });
};

/**
 * 更新数据查询
 * @param {string} id - 查询ID
 * @param {object} data - 更新数据
 */
export const updateQuery = (id, data) => {
  return request({
    url: `/data/queries/${id}`,
    method: "put",
    data,
  });
};

/**
 * 删除数据查询
 * @param {string} id - 查询ID
 */
export const deleteQuery = (id) => {
  return request({
    url: `/data/queries/${id}`,
    method: "delete",
  });
};

/**
 * 执行数据查询
 * @param {string} id - 查询ID
 * @param {object} parameters - 查询参数
 */
export const executeQuery = (id, parameters = {}) => {
  return request({
    url: `/data/queries/${id}/execute`,
    method: "post",
    data: { parameters },
  });
};

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
    method: "post",
    data: { sql, parameters },
  });
};

/**
 * 保存数据查询
 * @param {string} id - 查询ID
 * @param {object} data - 查询数据
 */
export const saveQuery = (id, data) => {
  return request({
    url: `/data/queries/${id}`,
    method: "put",
    data,
  });
};

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
    method: "get",
    params,
  });
};

/**
 * 删除数据点
 * @param {string} projectId - 工程ID
 * @param {string} datapointId - 数据点ID
 */
export const deleteDataPoint = (projectId, datapointId) => {
  return request({
    url: `/data/projects/${projectId}/datapoints/${datapointId}`,
    method: "delete",
  });
};

/**
 * 批量删除失效数据点
 * @param {string} projectId - 工程ID
 * @param {string[]} datapointIds - 数据点ID列表
 */
export const deleteDataPointsBatch = (projectId, datapointIds) => {
  return request({
    url: `/data/projects/${projectId}/datapoints/delete-batch`,
    method: "post",
    data: {
      ids: datapointIds,
    },
  });
};

// ===========================================
// MQTT订阅相关API
// ===========================================

/**
 * 获取MQTT订阅列表
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 */
export const getMqttSubscriptions = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/connections/${connectionId}/subscriptions`,
    method: "get",
  });
};

/**
 * 获取单个MQTT订阅
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 */
export const getMqttSubscription = (projectId, subscriptionId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}`,
    method: "get",
  });
};

/**
 * 创建MQTT订阅
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 * @param {object} data - 订阅数据
 */
export const createMqttSubscription = (projectId, connectionId, data) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/connections/${connectionId}/subscriptions`,
    method: "post",
    data,
  });
};

/**
 * 更新MQTT订阅
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 * @param {object} data - 更新数据
 */
export const updateMqttSubscription = (projectId, subscriptionId, data) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}`,
    method: "put",
    data,
  });
};

/**
 * 删除MQTT订阅
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 */
export const deleteMqttSubscription = (projectId, subscriptionId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}`,
    method: "delete",
  });
};

/**
 * 切换MQTT订阅启用状态
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 */
export const toggleMqttSubscription = (projectId, subscriptionId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/toggle`,
    method: "patch",
  });
};

/**
 * 获取MQTT订阅的消息列表
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 */
export const getMqttSubscriptionMessages = (projectId, subscriptionId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/messages`,
    method: "get",
  });
};

/**
 * 启动MQTT连接
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 */
export const startMqttConnection = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/connections/${connectionId}/start`,
    method: "post",
  });
};

/**
 * 停止MQTT连接
 * @param {string} projectId - 工程ID
 * @param {string} connectionId - 连接ID
 */
export const stopMqttConnection = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/connections/${connectionId}/stop`,
    method: "post",
  });
};

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
    method: "get",
    params,
  });
};

/**
 * 获取单个变量组
 * @param {string} groupId - 变量组ID
 */
export const getMqttTagGroup = (groupId) => {
  return request({
    url: `/data/mqtt/tag-groups/${groupId}`,
    method: "get",
  });
};

/**
 * 创建变量组
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 * @param {object} data - 变量组数据
 */
export const createMqttTagGroup = (projectId, subscriptionId, data) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/tag-groups`,
    method: "post",
    data,
  });
};

/**
 * 更新变量组
 * @param {string} groupId - 变量组ID
 * @param {object} data - 更新数据
 */
export const updateMqttTagGroup = (groupId, data) => {
  return request({
    url: `/data/mqtt/tag-groups/${groupId}`,
    method: "put",
    data,
  });
};

/**
 * 删除变量组
 * @param {string} groupId - 变量组ID
 */
export const deleteMqttTagGroup = (groupId) => {
  return request({
    url: `/data/mqtt/tag-groups/${groupId}`,
    method: "delete",
  });
};

/**
 * 更新变量组顺序
 * @param {array} groups - 变量组数组，包含id和order
 */
export const updateMqttTagGroupsOrder = (groups) => {
  return request({
    url: `/data/mqtt/tag-groups/order`,
    method: "put",
    data: { groups },
  });
};

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
    method: "get",
    params,
  });
};

/**
 * 获取项目的所有Tag
 * @param {string} projectId - 工程ID
 * @param {object} params - 查询参数
 */
export const getProjectMqttTags = (projectId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/tags`,
    method: "get",
    params,
  });
};

/**
 * 获取Tag详情
 * @param {string} tagId - Tag ID
 */
export const getMqttTag = (tagId) => {
  return request({
    url: `/data/mqtt/tags/${tagId}`,
    method: "get",
  });
};

/**
 * 创建Tag
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 * @param {object} data - Tag数据
 */
export const createMqttTag = (projectId, subscriptionId, data) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/tags`,
    method: "post",
    data,
  });
};

/**
 * 批量创建Tag
 * @param {string} projectId - 工程ID
 * @param {string} subscriptionId - 订阅ID
 * @param {array} tags - Tag数据数组
 */
export const createMqttTagsBatch = (projectId, subscriptionId, tags) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/tags/batch`,
    method: "post",
    data: { tags },
  });
};

/**
 * 更新Tag
 * @param {string} tagId - Tag ID
 * @param {object} data - 更新数据
 */
export const updateMqttTag = (tagId, data) => {
  return request({
    url: `/data/mqtt/tags/${tagId}`,
    method: "put",
    data,
  });
};

/**
 * 删除Tag
 * @param {string} tagId - Tag ID
 */
export const deleteMqttTag = (tagId) => {
  return request({
    url: `/data/mqtt/tags/${tagId}`,
    method: "delete",
  });
};

/**
 * 切换Tag启用状态
 * @param {string} tagId - Tag ID
 */
export const toggleMqttTag = (tagId) => {
  return request({
    url: `/data/mqtt/tags/${tagId}/toggle`,
    method: "patch",
  });
};

/**
 * 更新Tags顺序
 * @param {array} tagIds - Tag ID数组
 */
export const updateMqttTagsOrder = (tagIds) => {
  return request({
    url: `/data/mqtt/tags/order`,
    method: "put",
    data: { tagIds },
  });
};

/**
 * 获取Tag的当前值
 * @param {string} tagId - Tag ID
 */
export const getMqttTagValue = (tagId) => {
  return request({
    url: `/data/mqtt/tags/${tagId}/value`,
    method: "get",
  });
};

/**
 * 获取多个Tag的当前值
 * @param {array} tagIds - Tag ID数组
 */
export const getMqttTagValues = (tagIds) => {
  return request({
    url: `/data/mqtt/tags/values`,
    method: "post",
    data: { tagIds },
  });
};

export default {
  getConnections,
  createConnection,
  testConnection,
  getConnectionTables,
  getTableData,
  getTableStructure,
  updateConnection,
  deleteConnection,
  updateConnectionStatus,
  getQueries,
  createQuery,
  updateQuery,
  executeQuery,
  executeSql,
  deleteQuery,
  // 数据点相关
  getDataPoints,
  deleteDataPoint,
  deleteDataPointsBatch,
  // MQTT订阅相关
  getMqttSubscriptions,
  getMqttSubscription,
  createMqttSubscription,
  updateMqttSubscription,
  deleteMqttSubscription,
  toggleMqttSubscription,
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
  toggleMqttTag,
  updateMqttTagsOrder,
  getMqttTagValue,
  getMqttTagValues,
};
