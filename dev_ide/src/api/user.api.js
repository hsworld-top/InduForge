import request from '@/utils/request'

/**
 * 用户管理 API
 */
export const userAPI = {
  /**
   * 获取用户列表
   * @param {object} params - 查询参数
   * @param {number} params.page - 页码
   * @param {number} params.limit - 每页大小
   * @param {string} params.username - 用户名搜索
   * @param {string} params.role - 角色筛选
   * @param {string} params.status - 状态筛选
   * @returns {Promise} 用户列表
   */
  getUsers(params = {}) {
    return request.get('/users', { params })
  },

  /**
   * 创建用户
   * @param {object} userData - 用户数据
   * @param {string} userData.username - 用户名
   * @param {string} userData.password - 密码
   * @param {string} userData.email - 邮箱
   * @param {string} userData.fullName - 真实姓名
   * @param {string} userData.role - 角色
   * @param {string} userData.tenantId - 租户ID
   * @returns {Promise} 创建结果
   */
  createUser(userData) {
    return request.post('/users', userData)
  },

  /**
   * 更新用户
   * @param {string} id - 用户ID
   * @param {object} userData - 用户数据
   * @param {string} userData.email - 邮箱
   * @param {string} userData.fullName - 真实姓名
   * @param {string} userData.role - 角色
   * @param {string} userData.status - 状态
   * @returns {Promise} 更新结果
   */
  updateUser(id, userData) {
    return request.put(`/users/${id}`, userData)
  },

  /**
   * 修改用户密码
   * @param {string} id - 用户ID
   * @param {string} newPassword - 新密码
   * @returns {Promise} 修改结果
   */
  updatePassword(id, newPassword) {
    return request.put(`/users/${id}/password`, { newPassword })
  },

  /**
   * 删除用户
   * @param {string} id - 用户ID
   * @returns {Promise} 删除结果
   */
  deleteUser(id) {
    return request.delete(`/users/${id}`)
  },
}
