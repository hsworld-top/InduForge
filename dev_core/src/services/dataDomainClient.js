const AppError = require('../utils/AppError')
const ErrorCodes = require('../constants/errorCodes')

const DEFAULT_TIMEOUT_MS = Number(process.env.DATA_SERVICE_TIMEOUT_MS || 10000)
const DEFAULT_BASE_URL =
  process.env.DATA_SERVICE_BASE_URL || process.env.VITE_DATA_SERVICE_URL || 'http://127.0.0.1:18102'

/**
 * 读取对象中的首个已定义字段，兼容 data_service 当前返回的 PascalCase 字段，
 * 同时兼容 dev_core 历史导入包里的 camelCase 字段。
 * @param {object|null|undefined} source - 原始对象
 * @param {string[]} keys - 候选字段名
 * @returns {any}
 */
const pickField = (source, ...keys) => {
  if (!source || typeof source !== 'object') {
    return undefined
  }

  for (const key of keys) {
    if (Object.prototype.hasOwnProperty.call(source, key) && source[key] !== undefined) {
      return source[key]
    }
  }

  return undefined
}

/**
 * 统一转成数组，避免调用方反复判空。
 * @param {any} value - 可能的数组
 * @returns {Array}
 */
const asArray = (value) => (Array.isArray(value) ? value : [])

/**
 * 规范化连接记录，统一成 dev_core 侧稳定的 camelCase 结构。
 * @param {object} record - 原始连接记录
 * @returns {object}
 */
const normalizeConnectionRecord = (record) => ({
  id: pickField(record, 'id', 'ID'),
  projectId: pickField(record, 'projectId', 'ProjectID'),
  name: pickField(record, 'name', 'Name'),
  type: pickField(record, 'type', 'Type'),
  status: pickField(record, 'status', 'Status'),
  config: pickField(record, 'config', 'Config', 'metadata', 'Metadata') || {},
  createdAt: pickField(record, 'createdAt', 'CreatedAt'),
  updatedAt: pickField(record, 'updatedAt', 'UpdatedAt'),
})

/**
 * 规范化关系库配置记录。
 * @param {object} record - 原始关系库配置
 * @returns {object}
 */
const normalizeRelationalConfigRecord = (record) => ({
  connectionId: pickField(record, 'connectionId', 'ConnectionID'),
  dbType: pickField(record, 'dbType', 'DBType'),
  host: pickField(record, 'host', 'Host'),
  port: pickField(record, 'port', 'Port'),
  database: pickField(record, 'database', 'Database'),
  username: pickField(record, 'username', 'Username'),
  password: pickField(record, 'password', 'Password'),
  schema: pickField(record, 'schema', 'Schema') || null,
  charset: pickField(record, 'charset', 'Charset') || null,
  timezone: pickField(record, 'timezone', 'Timezone') || null,
  ssl: Boolean(pickField(record, 'ssl', 'SSL')),
  sslConfig: pickField(record, 'sslConfig', 'SSLConfig') || {},
})

/**
 * 规范化 MQTT 配置记录。
 * @param {object} record - 原始 MQTT 配置
 * @returns {object}
 */
const normalizeMqttConfigRecord = (record) => ({
  connectionId: pickField(record, 'connectionId', 'ConnectionID'),
  brokerUrl: pickField(record, 'brokerUrl', 'BrokerURL'),
  protocol: pickField(record, 'protocol', 'Protocol'),
  port: pickField(record, 'port', 'Port'),
  clientId: pickField(record, 'clientId', 'ClientID') || null,
  username: pickField(record, 'username', 'Username') || null,
  password: pickField(record, 'password', 'Password') || null,
  keepalive: pickField(record, 'keepalive', 'Keepalive'),
  cleanSession: Boolean(pickField(record, 'cleanSession', 'CleanSession')),
  qos: pickField(record, 'qos', 'QOS'),
  reconnectPeriod: pickField(record, 'reconnectPeriod', 'ReconnectPeriodMS'),
  connectTimeout: pickField(record, 'connectTimeout', 'ConnectTimeoutMS'),
  will: pickField(record, 'will', 'Will') || {},
  sslConfig: pickField(record, 'sslConfig', 'SSLConfig') || {},
})

/**
 * 规范化查询记录。
 * @param {object} record - 原始查询记录
 * @returns {object}
 */
const normalizeQueryRecord = (record) => ({
  id: pickField(record, 'id', 'ID'),
  projectId: pickField(record, 'projectId', 'ProjectID'),
  connectionId: pickField(record, 'connectionId', 'ConnectionID'),
  name: pickField(record, 'name', 'Name'),
  description: pickField(record, 'description', 'Description') || null,
  category: pickField(record, 'category', 'Category') || null,
  queryType: pickField(record, 'queryType', 'QueryType'),
  config: pickField(record, 'config', 'Config') || {},
  transformer: pickField(record, 'transformer', 'Transformer') || null,
  isEnabled: pickField(record, 'isEnabled', 'IsEnabled'),
  timeoutMs: pickField(record, 'timeoutMs', 'TimeoutMS'),
  cacheEnabled: Boolean(pickField(record, 'cacheEnabled', 'CacheEnabled')),
  cacheTtlSeconds: pickField(record, 'cacheTtlSeconds', 'CacheTtlSeconds'),
  createdBy: pickField(record, 'createdBy', 'CreatedBy') || null,
  updatedBy: pickField(record, 'updatedBy', 'UpdatedBy') || null,
  createdAt: pickField(record, 'createdAt', 'CreatedAt'),
  updatedAt: pickField(record, 'updatedAt', 'UpdatedAt'),
})

/**
 * 规范化 MQTT 订阅记录。
 * @param {object} record - 原始 MQTT 订阅
 * @returns {object}
 */
const normalizeMqttSubscriptionRecord = (record) => ({
  id: pickField(record, 'id', 'ID'),
  projectId: pickField(record, 'projectId', 'ProjectID'),
  connectionId: pickField(record, 'connectionId', 'ConnectionID'),
  name: pickField(record, 'name', 'Name'),
  topic: pickField(record, 'topic', 'Topic'),
  qos: pickField(record, 'qos', 'QOS'),
  description: pickField(record, 'description', 'Description') || null,
  isEnabled: pickField(record, 'isEnabled', 'IsEnabled'),
  messageRetention: pickField(record, 'messageRetention', 'MessageRetention'),
  createdAt: pickField(record, 'createdAt', 'CreatedAt'),
  updatedAt: pickField(record, 'updatedAt', 'UpdatedAt'),
})

/**
 * 规范化 MQTT 变量组记录。
 * @param {object} record - 原始 MQTT 变量组
 * @returns {object}
 */
const normalizeMqttTagGroupRecord = (record) => ({
  id: pickField(record, 'id', 'ID'),
  projectId: pickField(record, 'projectId', 'ProjectID'),
  subscriptionId: pickField(record, 'subscriptionId', 'SubscriptionID'),
  name: pickField(record, 'name', 'Name'),
  code: pickField(record, 'code', 'Code'),
  description: pickField(record, 'description', 'Description') || null,
  color: pickField(record, 'color', 'Color') || null,
  icon: pickField(record, 'icon', 'Icon') || null,
  order: pickField(record, 'order', 'Order'),
  createdAt: pickField(record, 'createdAt', 'CreatedAt'),
  updatedAt: pickField(record, 'updatedAt', 'UpdatedAt'),
})

/**
 * 规范化 MQTT 变量记录。
 * @param {object} record - 原始 MQTT 变量
 * @returns {object}
 */
const normalizeMqttTagRecord = (record) => ({
  id: pickField(record, 'id', 'ID'),
  projectId: pickField(record, 'projectId', 'ProjectID'),
  subscriptionId: pickField(record, 'subscriptionId', 'SubscriptionID'),
  groupId: pickField(record, 'groupId', 'GroupID') || null,
  name: pickField(record, 'name', 'Name'),
  code: pickField(record, 'code', 'Code'),
  description: pickField(record, 'description', 'Description') || null,
  dataType: pickField(record, 'dataType', 'DataType'),
  parseType: pickField(record, 'parseType', 'ParseType'),
  parseRule: pickField(record, 'parseRule', 'ParseRule'),
  defaultValue: pickField(record, 'defaultValue', 'DefaultValue') || null,
  unit: pickField(record, 'unit', 'Unit') || null,
  transform: pickField(record, 'transform', 'Transform') || null,
  validation: pickField(record, 'validation', 'Validation') || {},
  isEnabled: pickField(record, 'isEnabled', 'IsEnabled'),
  order: pickField(record, 'order', 'Order'),
  createdAt: pickField(record, 'createdAt', 'CreatedAt'),
  updatedAt: pickField(record, 'updatedAt', 'UpdatedAt'),
})

/**
 * 规范化数据点记录。
 * @param {object} record - 原始数据点
 * @returns {object}
 */
const normalizeDataPointRecord = (record) => ({
  id: pickField(record, 'id', 'ID'),
  projectId: pickField(record, 'projectId', 'ProjectID'),
  path: pickField(record, 'path', 'Path'),
  name: pickField(record, 'name', 'Name'),
  description: pickField(record, 'description', 'Description') || null,
  sourceType: pickField(record, 'sourceType', 'SourceType'),
  sourceId: pickField(record, 'sourceId', 'SourceID') || null,
  sourceConfig: pickField(record, 'sourceConfig', 'SourceConfig') || {},
  dataType: pickField(record, 'dataType', 'DataType'),
  unit: pickField(record, 'unit', 'Unit') || null,
  precisionNum: pickField(record, 'precisionNum', 'PrecisionNum') ?? null,
  defaultValue: pickField(record, 'defaultValue', 'DefaultValue') || null,
  minValue: pickField(record, 'minValue', 'MinValue') ?? null,
  maxValue: pickField(record, 'maxValue', 'MaxValue') ?? null,
  alarmLow: pickField(record, 'alarmLow', 'AlarmLow') ?? null,
  alarmHigh: pickField(record, 'alarmHigh', 'AlarmHigh') ?? null,
  tags: asArray(pickField(record, 'tags', 'Tags')),
  refreshMode: pickField(record, 'refreshMode', 'RefreshMode'),
  refreshIntervalMs: pickField(record, 'refreshIntervalMs', 'RefreshIntervalMS') ?? null,
  status: pickField(record, 'status', 'Status'),
  createdBy: pickField(record, 'createdBy', 'CreatedBy') || null,
  updatedBy: pickField(record, 'updatedBy', 'UpdatedBy') || null,
  createdAt: pickField(record, 'createdAt', 'CreatedAt'),
  updatedAt: pickField(record, 'updatedAt', 'UpdatedAt'),
})

/**
 * 规范化项目快照。
 * @param {object|null|undefined} snapshot - 原始快照响应
 * @returns {object}
 */
const normalizeProjectSnapshot = (snapshot) => ({
  connections: asArray(snapshot?.connections).map(normalizeConnectionRecord),
  relationalConfigs: asArray(snapshot?.relationalConfigs).map(normalizeRelationalConfigRecord),
  queries: asArray(snapshot?.queries).map(normalizeQueryRecord),
  mqttConfigs: asArray(snapshot?.mqttConfigs).map(normalizeMqttConfigRecord),
  mqttSubscriptions: asArray(snapshot?.mqttSubscriptions).map(normalizeMqttSubscriptionRecord),
  mqttTagGroups: asArray(snapshot?.mqttTagGroups).map(normalizeMqttTagGroupRecord),
  mqttTags: asArray(snapshot?.mqttTags).map(normalizeMqttTagRecord),
  datapoints: asArray(snapshot?.datapoints).map(normalizeDataPointRecord),
})

/**
 * 规范化 artifact 协议对象（kafka/http/websocket/redis）。
 * @param {object} record - 原始协议对象
 * @returns {object}
 */
const normalizeArtifactProtocolRecord = (record) => ({
  id: pickField(record, 'id', 'ID'),
  name: pickField(record, 'name', 'Name'),
  type: pickField(record, 'type', 'Type'),
  status: pickField(record, 'status', 'Status'),
  config: pickField(record, 'config', 'Config') || {},
})

/**
 * 规范化 artifact 中的 MQTT 连接对象。
 * @param {object} record - 原始 MQTT 连接对象
 * @returns {object}
 */
const normalizeArtifactMqttConnectionRecord = (record) => ({
  id: pickField(record, 'id', 'ID', 'connectionId', 'ConnectionID'),
  name: pickField(record, 'name', 'Name') || '',
  type: pickField(record, 'type', 'Type') || 'mqtt',
  status: pickField(record, 'status', 'Status') || '',
  brokerUrl: pickField(record, 'brokerUrl', 'BrokerURL') || '',
  protocol: pickField(record, 'protocol', 'Protocol') || 'mqtt',
  port: pickField(record, 'port', 'Port'),
  clientId: pickField(record, 'clientId', 'ClientID') || null,
  username: pickField(record, 'username', 'Username') || null,
  password: pickField(record, 'password', 'Password') || null,
  keepalive: pickField(record, 'keepalive', 'Keepalive'),
  cleanSession: Boolean(pickField(record, 'cleanSession', 'CleanSession')),
  qos: pickField(record, 'qos', 'QOS'),
  reconnectPeriod: pickField(record, 'reconnectPeriod', 'ReconnectPeriodMS'),
  connectTimeout: pickField(record, 'connectTimeout', 'ConnectTimeoutMS'),
  will: pickField(record, 'will', 'Will') || {},
  sslConfig: pickField(record, 'sslConfig', 'SSLConfig') || {},
})

/**
 * 规范化项目 artifact v1。
 * @param {object|null|undefined} artifact - 原始 artifact 响应
 * @returns {object}
 */
const normalizeProjectArtifact = (artifact) => ({
  version: pickField(artifact, 'version', 'Version'),
  projectId: pickField(artifact, 'projectId', 'ProjectID'),
  generatedAt: pickField(artifact, 'generatedAt', 'GeneratedAt'),
  connections: asArray(artifact?.connections).map(normalizeConnectionRecord),
  queries: asArray(artifact?.queries).map(normalizeQueryRecord),
  datapoints: asArray(artifact?.datapoints).map(normalizeDataPointRecord),
  mqtt: {
    connections: asArray(artifact?.mqtt?.connections).map(normalizeArtifactMqttConnectionRecord),
    subscriptions: asArray(artifact?.mqtt?.subscriptions).map(normalizeMqttSubscriptionRecord),
    tagGroups: asArray(artifact?.mqtt?.tagGroups).map(normalizeMqttTagGroupRecord),
    tags: asArray(artifact?.mqtt?.tags).map(normalizeMqttTagRecord),
  },
  protocols: {
    kafka: asArray(artifact?.protocols?.kafka).map(normalizeArtifactProtocolRecord),
    http: asArray(artifact?.protocols?.http).map(normalizeArtifactProtocolRecord),
    websocket: asArray(artifact?.protocols?.websocket).map(normalizeArtifactProtocolRecord),
    redis: asArray(artifact?.protocols?.redis).map(normalizeArtifactProtocolRecord),
  },
})

/**
 * data_service 客户端，负责 dev_core 与正式数据域服务交互。
 */
class DataDomainClient {
  constructor(options = {}) {
    this.baseUrl = String(options.baseUrl || DEFAULT_BASE_URL).replace(/\/+$/, '')
    this.timeoutMs = Number(options.timeoutMs || DEFAULT_TIMEOUT_MS)
    this.serviceToken = options.serviceToken || process.env.DATA_SERVICE_BEARER_TOKEN || ''
  }

  buildUrl(pathname) {
    return `${this.baseUrl}${pathname}`
  }

  buildAuthorizationHeader(authorization) {
    return authorization || (this.serviceToken ? `Bearer ${this.serviceToken}` : '')
  }

  async request(pathname, options = {}) {
    const controller = new AbortController()
    const timeout = setTimeout(() => controller.abort(), this.timeoutMs)

    try {
      const headers = {
        Accept: 'application/json',
        ...(options.body ? { 'Content-Type': 'application/json' } : {}),
        ...(options.headers || {}),
      }

      const authorization = this.buildAuthorizationHeader(options.authorization)
      if (authorization) {
        headers.Authorization = authorization
      }

      const response = await fetch(this.buildUrl(pathname), {
        method: options.method || 'GET',
        headers,
        body: options.body ? JSON.stringify(options.body) : undefined,
        signal: controller.signal,
      })

      const contentType = response.headers.get('content-type') || ''
      const payload = contentType.includes('application/json')
        ? await response.json()
        : await response.text()

      if (!response.ok) {
        throw new AppError(ErrorCodes.EXTERNAL_SERVICE_ERROR, 502, {
          message:
            payload?.message || `data_service 请求失败: ${response.status} ${response.statusText}`,
        })
      }

      if (payload && typeof payload === 'object' && payload.success === false) {
        throw new AppError(ErrorCodes.EXTERNAL_SERVICE_ERROR, 502, {
          message: payload.message || 'data_service 返回业务失败',
        })
      }

      return payload?.data !== undefined ? payload.data : payload
    } catch (error) {
      if (error instanceof AppError) {
        throw error
      }

      if (error?.name === 'AbortError') {
        throw new AppError(ErrorCodes.EXTERNAL_SERVICE_ERROR, 504, {
          message: `data_service 请求超时（>${this.timeoutMs}ms）`,
        })
      }

      throw new AppError(ErrorCodes.EXTERNAL_SERVICE_ERROR, 502, {
        message: error?.message || 'data_service 请求异常',
      })
    } finally {
      clearTimeout(timeout)
    }
  }

  /**
   * 获取项目数据域快照。
   * @param {string} projectId - 工程ID
   * @param {string} [authorization] - Bearer Token
   * @returns {Promise<object>}
   */
  async getProjectSnapshot(projectId, authorization) {
    const snapshot = await this.request(`/api/v1/data/projects/${projectId}/snapshot`, {
      authorization,
    })
    return normalizeProjectSnapshot(snapshot)
  }

  /**
   * 获取项目 artifact v1。
   * @param {string} projectId - 工程ID
   * @param {string} [authorization] - Bearer Token
   * @returns {Promise<object>}
   */
  async getProjectArtifact(projectId, authorization) {
    const artifact = await this.request(`/api/v1/data/projects/${projectId}/artifact`, {
      authorization,
    })
    return normalizeProjectArtifact(artifact)
  }

  /**
   * 用新快照覆盖项目数据域数据。
   * @param {string} projectId - 工程ID
   * @param {object} snapshot - 快照内容
   * @param {string} [authorization] - Bearer Token
   * @returns {Promise<object>}
   */
  async replaceProjectSnapshot(projectId, snapshot, authorization) {
    return this.request(`/api/v1/data/projects/${projectId}/snapshot`, {
      method: 'PUT',
      authorization,
      body: snapshot,
    })
  }
}

module.exports = {
  DataDomainClient,
  dataDomainClient: new DataDomainClient(),
  normalizeProjectSnapshot,
  normalizeProjectArtifact,
}
