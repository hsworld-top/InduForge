/**
 * NodeAgent 初始化状态检查工具
 * @description 检查节点是否已完成初始化配置
 */

import { nodeApi } from '@/api/nodeApi'

/**
 * 检查是否已完成初始化
 * @returns {boolean} 是否已初始化
 */
export async function isInitialized() {
  try {
    // 新版初始化判断优先使用 bootstrap 状态机接口
    const bootstrap = await nodeApi.getBootstrapStatus()
    if (typeof bootstrap?.isInitialized === 'boolean') {
      return bootstrap.isInitialized
    }

    // 兼容旧版本后端
    const config = await nodeApi.getServiceConfig()
    return !!config?.bootstrap?.isInitialized
  } catch (error) {
    console.error('获取服务配置失败:', error)
    return false
  }
}

/**
 * 获取节点配置
 * @returns {Object} 节点配置信息
 */
export async function getNodeConfig() {
  try {
    return await nodeApi.getServiceConfig()
  } catch (error) {
    console.error('获取服务配置失败:', error)
    return null
  }
}

/**
 * 获取运维中心地址
 * @returns {string} 运维中心地址
 */
export async function getCenterUrl() {
  const config = await getNodeConfig()
  return config?.online?.centerUrl || ''
}

/**
 * 检查是否为在线模式
 * @returns {boolean} 是否在线模式
 */
export async function isOnlineMode() {
  const config = await getNodeConfig()
  return config?.mode === 'online'
}

/**
 * 检查是否为离线模式
 * @returns {boolean} 是否离线模式
 */
export async function isOfflineMode() {
  const config = await getNodeConfig()
  return config?.mode === 'offline'
}
