/**
 * NodeAgent 注册相关 API
 * @description 用于初始化向导的注册和审批查询接口
 */
import axios from 'axios'

/**
 * 登录获取访问令牌
 * @param {Object} data - 注册数据
 * @param {string} data.centerUrl - 运维中心地址
 * @param {string} data.username - 用户名
 * @param {string} data.password - 密码
 * @param {string} data.tenantCode - 租户代码（多租户必填）
 * @returns {Promise<Object>} 登录结果
 */
export async function loginWithAuth(data) {
  const { centerUrl, ...requestData } = data
  
  try {
    const response = await axios.post(
      `/api/v1/center/login`,
      { centerUrl, ...requestData },
      {
        timeout: 10000,
      }
    )
    
    if (!response.data?.success) {
      throw new Error(response.data?.message || '登录失败')
    }

    return response.data
  } catch (error) {
    throw new Error(error.response?.data?.message || error.response?.data?.error || error.message || '登录请求失败')
  }
}

/**
 * 提交节点注册申请（用户名密码模式）
 * @param {Object} data - 注册数据
 * @param {string} data.centerUrl - 运维中心地址
 * @param {string} data.username - 用户名
 * @param {string} data.password - 密码
 * @param {string} data.tenantCode - 租户代码
 * @param {string} data.nodeName - 节点名称
 * @param {string} data.nodeDescription - 节点描述
 * @param {string} data.ipAddress - IP地址
 * @param {number} data.port - 端口
 * @param {string} data.agentVersion - Agent版本
 * @returns {Promise<Object>} 注册结果
 */
export async function registerNodeWithToken(data) {
  const { centerUrl, username, password, tenantCode, nodeName, nodeDescription, ipAddress, port, agentVersion } = data

  try {
    const response = await axios.post(
      `/api/v1/center/register`,
      {
        centerUrl,
        username,
        password,
        tenantCode,
        nodeName,
        nodeDescription,
        agentVersion,
        ipAddress,
        port,
      },
      {
        timeout: 10000,
      }
    )

    if (!response.data?.success) {
      throw new Error(response.data?.message || '注册失败')
    }

    return response.data
  } catch (error) {
    throw new Error(error.response?.data?.message || error.response?.data?.error || error.message || '注册请求失败')
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
      `/api/v1/center/approval-status`,
      {
        params: { centerUrl, nodeId },
        timeout: 5000,
      }
    )
    
    return response.data
  } catch (error) {
    const err = new Error(error.response?.data?.error || error.message || '查询审批状态失败')
    err.status = error.response?.status
    throw err
  }
}

/**
 * 测试运维中心连接
 * @param {string} centerUrl - 运维中心地址
 * @returns {Promise<boolean>} 是否可连接
 */
export async function testCenterConnection(centerUrl) {
  try {
    const response = await axios.get(`/api/v1/center/health`, {
      params: { centerUrl },
      timeout: 3000,
    })
    return response.status === 200
  } catch (error) {
    return false
  }
}
