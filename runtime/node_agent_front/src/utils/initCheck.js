/**
 * NodeAgent 初始化状态检查工具
 * @description 检查节点是否已完成初始化配置
 */

const CONFIG_KEYS = {
  MODE: 'nodeagent_mode',
  NODE_ID: 'nodeagent_node_id',
  NODE_NAME: 'nodeagent_node_name',
  CENTER_URL: 'nodeagent_center_url',
  REGISTRATION_TOKEN: 'nodeagent_registration_token',
  APPROVAL_STATUS: 'nodeagent_approval_status',
  IP_ADDRESS: 'nodeagent_ip_address',
  PORT: 'nodeagent_port',
}

/**
 * 检查是否已完成初始化
 * @returns {boolean} 是否已初始化
 */
export function isInitialized() {
  const mode = localStorage.getItem(CONFIG_KEYS.MODE)
  
  if (!mode) {
    return false
  }

  // 离线模式只需要有 mode 和 node_name 即可
  if (mode === 'offline') {
    return !!localStorage.getItem(CONFIG_KEYS.NODE_NAME)
  }

  // 在线模式需要审批通过并获取到 token
  const approvalStatus = localStorage.getItem(CONFIG_KEYS.APPROVAL_STATUS)
  const registrationToken = localStorage.getItem(CONFIG_KEYS.REGISTRATION_TOKEN)
  const nodeId = localStorage.getItem(CONFIG_KEYS.NODE_ID)
  const centerUrl = localStorage.getItem(CONFIG_KEYS.CENTER_URL)

  return (
    approvalStatus === 'approved' &&
    registrationToken &&
    nodeId &&
    centerUrl
  )
}

/**
 * 获取节点配置
 * @returns {Object} 节点配置信息
 */
export function getNodeConfig() {
  return {
    mode: localStorage.getItem(CONFIG_KEYS.MODE),
    nodeId: localStorage.getItem(CONFIG_KEYS.NODE_ID),
    nodeName: localStorage.getItem(CONFIG_KEYS.NODE_NAME),
    centerUrl: localStorage.getItem(CONFIG_KEYS.CENTER_URL),
    registrationToken: localStorage.getItem(CONFIG_KEYS.REGISTRATION_TOKEN),
    approvalStatus: localStorage.getItem(CONFIG_KEYS.APPROVAL_STATUS),
    ipAddress: localStorage.getItem(CONFIG_KEYS.IP_ADDRESS),
    port: localStorage.getItem(CONFIG_KEYS.PORT),
  }
}

/**
 * 保存节点配置
 * @param {Object} config - 配置对象
 */
export function saveNodeConfig(config) {
  if (config.mode) localStorage.setItem(CONFIG_KEYS.MODE, config.mode)
  if (config.nodeId) localStorage.setItem(CONFIG_KEYS.NODE_ID, config.nodeId)
  if (config.nodeName) localStorage.setItem(CONFIG_KEYS.NODE_NAME, config.nodeName)
  if (config.centerUrl) localStorage.setItem(CONFIG_KEYS.CENTER_URL, config.centerUrl)
  if (config.registrationToken) localStorage.setItem(CONFIG_KEYS.REGISTRATION_TOKEN, config.registrationToken)
  if (config.approvalStatus) localStorage.setItem(CONFIG_KEYS.APPROVAL_STATUS, config.approvalStatus)
  if (config.ipAddress) localStorage.setItem(CONFIG_KEYS.IP_ADDRESS, config.ipAddress)
  if (config.port) localStorage.setItem(CONFIG_KEYS.PORT, config.port)
}

/**
 * 清除节点配置（重新初始化）
 */
export function clearNodeConfig() {
  Object.values(CONFIG_KEYS).forEach(key => {
    localStorage.removeItem(key)
  })
}

/**
 * 获取运维中心地址
 * @returns {string} 运维中心地址
 */
export function getCenterUrl() {
  return localStorage.getItem(CONFIG_KEYS.CENTER_URL) || ''
}

/**
 * 检查是否为在线模式
 * @returns {boolean} 是否在线模式
 */
export function isOnlineMode() {
  return localStorage.getItem(CONFIG_KEYS.MODE) === 'online'
}

/**
 * 检查是否为离线模式
 * @returns {boolean} 是否离线模式
 */
export function isOfflineMode() {
  return localStorage.getItem(CONFIG_KEYS.MODE) === 'offline'
}
