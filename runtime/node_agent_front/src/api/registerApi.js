/**
 * NodeAgent 注册相关 API
 * @description 用于初始化向导的注册和审批查询接口
 */
import axios from 'axios'

/**
 * 提交带用户认证的注册申请
 * @param {Object} data - 注册数据
 * @param {string} data.centerUrl - 运维中心地址
 * @param {string} data.username - 用户名
 * @param {string} data.password - 密码
 * @param {string} data.nodeName - 节点名称
 * @param {string} data.nodeDescription - 节点描述
 * @param {string} data.ipAddress - IP地址
 * @param {number} data.port - 端口
 * @param {string} data.agentVersion - Agent版本
 * @param {string} data.mode - 节点模式
 * @returns {Promise<Object>} 注册结果
 */
export async function registerWithAuth(data) {
  const { centerUrl, ...requestData } = data
  
  try {
    const response = await axios.post(
      `${centerUrl}/api/v1/node-register/register-with-auth`,
      requestData,
      {
        timeout: 10000,
      }
    )
    
    return response.data
  } catch (error) {
    throw new Error(error.response?.data?.error || error.message || '注册请求失败')
  }
}

/**
 * 查询审批状态
 * @param {string} centerUrl - 运维中心地址
 * @param {string} nodeId - 节点ID
 * @returns {Promise<Object>} 审批状态
 */
export async function checkApprovalStatus(centerUrl, nodeId) {
  try {
    const response = await axios.get(
      `${centerUrl}/api/v1/node-register/${nodeId}/approval-status`,
      {
        timeout: 5000,
      }
    )
    
    return response.data
  } catch (error) {
    throw new Error(error.response?.data?.error || error.message || '查询审批状态失败')
  }
}

/**
 * 测试运维中心连接
 * @param {string} centerUrl - 运维中心地址
 * @returns {Promise<boolean>} 是否可连接
 */
export async function testCenterConnection(centerUrl) {
  try {
    const response = await axios.get(`${centerUrl}/health`, {
      timeout: 3000,
    })
    return response.status === 200
  } catch (error) {
    return false
  }
}
