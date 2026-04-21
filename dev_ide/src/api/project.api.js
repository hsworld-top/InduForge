import request from '@/utils/request'

/**
 * 工程管理 API
 */
export const projectAPI = {
  /**
   * 获取工程列表
   * @param {object} params - 查询参数
   * @param {number} params.page - 页码
   * @param {number} params.limit - 每页大小
   * @param {string} params.name - 工程名称搜索
   * @param {string} params.status - 状态筛选
   * @param {string} params.priority - 优先级筛选
   * @returns {Promise} 工程列表
   */
  getProjects(params = {}) {
    return request.get('/projects', { params })
  },

  /**
   * 创建工程
   * @param {object} projectData - 工程数据
   * @param {string} projectData.name - 工程名称
   * @param {string} projectData.description - 描述
   * @param {string} projectData.colorTag - 颜色标签
   * @returns {Promise} 创建结果
   */
  createProject(projectData) {
    return request.post('/projects', projectData)
  },

  /**
   * 更新工程
   * @param {string} id - 工程ID
   * @param {object} projectData - 工程数据
   * @param {string} projectData.name - 工程名称
   * @param {string} projectData.description - 描述
   * @param {string} projectData.colorTag - 颜色标签
   * @returns {Promise} 更新结果
   */
  updateProject(id, projectData) {
    return request.put(`/projects/${id}`, projectData)
  },

  /**
   * 删除工程
   * @param {string} id - 工程ID
   * @param {object} options - 删除选项
   * @param {boolean} options.force - 是否强制删除（系统管理员）
   * @returns {Promise} 删除结果
   */
  deleteProject(id, options = {}) {
    return request.delete(`/projects/${id}`, {
      data: {
        force: Boolean(options.force),
      },
    })
  },

  /**
   * 获取删除工程影响评估
   * @param {string} id - 工程ID
   * @returns {Promise} 影响评估结果
   */
  getDeleteImpact(id) {
    return request.get(`/projects/${id}/delete-impact`)
  },

  /**
   * 执行工程运维操作
   * @param {string} id - 工程ID
   * @param {string} operation - 操作类型 (start, stop, restart, deploy, backup)
   * @returns {Promise} 操作结果
   */
  performOperation(id, operation) {
    return request.post(`/projects/${id}/operations/${operation}`)
  },

  /**
   * 导出工程
   * @param {string} id - 工程ID
   * @returns {Promise} 导出结果
   */
  exportProject(id) {
    return request.get(`/projects/${id}/export`, { responseType: 'blob' })
  },

  /**
   * 导入工程
   * @param {object} payload - 导入数据
   * @param {string} [payload.name] - 新工程名称
   * @param {object} payload.payload - 工程导出内容
   * @returns {Promise} 导入结果
   */
  importProject(payload) {
    return request.post('/projects/import', payload)
  },

  /**
   * 获取工程运行态用户列表
   * @param {string} id - 工程 ID
   * @returns {Promise} 用户列表
   */
  listRuntimeUsers(id) {
    return request.get(`/projects/${id}/runtime-users`)
  },

  /**
   * 创建工程运行态用户
   * @param {string} id - 工程 ID
   * @param {object} payload - 用户数据
   * @returns {Promise} 创建结果
   */
  createRuntimeUser(id, payload) {
    return request.post(`/projects/${id}/runtime-users`, payload)
  },

  /**
   * 更新工程运行态用户
   * @param {string} id - 工程 ID
   * @param {string} userId - 运行态用户 ID
   * @param {object} payload - 用户数据
   * @returns {Promise} 更新结果
   */
  updateRuntimeUser(id, userId, payload) {
    return request.put(`/projects/${id}/runtime-users/${userId}`, payload)
  },

  /**
   * 更新工程运行态用户状态
   * @param {string} id - 工程 ID
   * @param {string} userId - 运行态用户 ID
   * @param {object} payload - 状态数据
   * @returns {Promise} 更新结果
   */
  updateRuntimeUserStatus(id, userId, payload) {
    return request.put(`/projects/${id}/runtime-users/${userId}/status`, payload)
  },

  /**
   * 重置工程运行态用户密码
   * @param {string} id - 工程 ID
   * @param {string} userId - 运行态用户 ID
   * @param {object} payload - 新密码数据
   * @returns {Promise} 重置结果
   */
  resetRuntimeUserPassword(id, userId, payload) {
    return request.post(`/projects/${id}/runtime-users/${userId}/reset-password`, payload)
  },

  /**
   * 获取工程运行态角色列表
   * @param {string} id - 工程 ID
   * @returns {Promise} 角色列表
   */
  listRuntimeRoles(id) {
    return request.get(`/projects/${id}/runtime-roles`)
  },

  /**
   * 创建工程运行态角色
   * @param {string} id - 工程 ID
   * @param {object} payload - 角色数据
   * @returns {Promise} 创建结果
   */
  createRuntimeRole(id, payload) {
    return request.post(`/projects/${id}/runtime-roles`, payload)
  },

  /**
   * 更新工程运行态角色
   * @param {string} id - 工程 ID
   * @param {string} roleId - 角色 ID
   * @param {object} payload - 角色数据
   * @returns {Promise} 更新结果
   */
  updateRuntimeRole(id, roleId, payload) {
    return request.put(`/projects/${id}/runtime-roles/${roleId}`, payload)
  },

  /**
   * 删除工程运行态角色
   * @param {string} id - 工程 ID
   * @param {string} roleId - 角色 ID
   * @returns {Promise} 删除结果
   */
  deleteRuntimeRole(id, roleId) {
    return request.delete(`/projects/${id}/runtime-roles/${roleId}`)
  },
}
