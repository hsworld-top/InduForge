import request from '@/utils/request'

type ApiId = string | number
type QueryParams = Record<string, unknown>
type DeleteOptions = {
  force?: boolean
}

export type ProjectOverviewSortField =
  | 'createdAt'
  | 'updatedAt'
  | 'lastDeployedAt'
  | 'runtimeStatus'
export type ProjectOverviewSortOrder = 'ASC' | 'DESC' | 'asc' | 'desc'
export type ProjectRuntimeMode = 'DEV' | 'RELEASE'
export type ProjectDeployStatus =
  | 'pending'
  | 'deploying'
  | 'running'
  | 'stopped'
  | 'error'
  | 'rollback'
export type ProjectVisibility = 'private' | 'internal'

export type ProjectListQueryParams = QueryParams & {
  page?: number
  limit?: number
  name?: string
  group?: string
  groupId?: string
  tag?: string | string[]
  tagId?: string | string[]
  tags?: string | string[]
  runtimeMode?: ProjectRuntimeMode | ProjectRuntimeMode[]
  deployStatus?: ProjectDeployStatus | ProjectDeployStatus[]
  visibility?: ProjectVisibility | ProjectVisibility[]
  createdBy?: string
  createdByName?: string
  sortBy?: ProjectOverviewSortField
  sortField?: ProjectOverviewSortField
  sortOrder?: ProjectOverviewSortOrder
  order?: ProjectOverviewSortOrder
}

export type ProjectLabelQueryParams = {
  keyword?: string
}

export type ProjectTagPayload = {
  name: string
  description?: string
  sortOrder?: number
}

export type ProjectTagUpdatePayload = Partial<ProjectTagPayload>

export type ProjectGroupPayload = {
  name: string
  description?: string
  sortOrder?: number
}

export type ProjectGroupUpdatePayload = Partial<ProjectGroupPayload>

export type ProjectTagBindingPayload = {
  tagIds: string[]
}

export type ProjectGroupBindingPayload = {
  groupId: string | null
}

export type ProjectUpdatePayload = {
  name?: string
  description?: string
  visibility?: ProjectVisibility
}

export type ProjectAuthoringContext = {
  projectId: string
  authoringEpoch: string
}

/**
 * 工程管理 API
 */
export const projectAPI = {
  getAuthoringContext(id: ApiId) {
    return request.get(`/projects/${id}/authoring-context`, { projectId: id })
  },
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
  getProjects(params: ProjectListQueryParams = {}) {
    return request.get('/projects', {
      params,
      paramsSerializer: {
        indexes: null,
      },
    })
  },

  /**
   * 获取工程标签列表
   * @param {object} params - 查询参数
   * @param {string} params.keyword - 标签名称关键词
   * @returns {Promise} 标签列表
   */
  listProjectTags(params: ProjectLabelQueryParams = {}) {
    return request.get('/projects/tags', { params })
  },

  /**
   * 创建工程标签
   * @param {object} payload - 标签数据
   * @returns {Promise} 创建结果
   */
  createProjectTag(payload: ProjectTagPayload) {
    return request.post('/projects/tags', payload)
  },

  /**
   * 更新工程标签
   * @param {string} tagId - 标签 ID
   * @param {object} payload - 标签更新数据
   * @returns {Promise} 更新结果
   */
  updateProjectTag(tagId: ApiId, payload: ProjectTagUpdatePayload) {
    return request.put(`/projects/tags/${tagId}`, payload)
  },

  /**
   * 删除工程标签
   * @param {string} tagId - 标签 ID
   * @returns {Promise} 删除结果
   */
  deleteProjectTag(tagId: ApiId) {
    return request.delete(`/projects/tags/${tagId}`)
  },

  /**
   * 获取工程分组列表
   * @param {object} params - 查询参数
   * @param {string} params.keyword - 分组名称关键词
   * @returns {Promise} 分组列表
   */
  listProjectGroups(params: ProjectLabelQueryParams = {}) {
    return request.get('/projects/groups', { params })
  },

  /**
   * 创建工程分组
   * @param {object} payload - 分组数据
   * @returns {Promise} 创建结果
   */
  createProjectGroup(payload: ProjectGroupPayload) {
    return request.post('/projects/groups', payload)
  },

  /**
   * 更新工程分组
   * @param {string} groupId - 分组 ID
   * @param {object} payload - 分组更新数据
   * @returns {Promise} 更新结果
   */
  updateProjectGroup(groupId: ApiId, payload: ProjectGroupUpdatePayload) {
    return request.put(`/projects/groups/${groupId}`, payload)
  },

  /**
   * 删除工程分组
   * @param {string} groupId - 分组 ID
   * @returns {Promise} 删除结果
   */
  deleteProjectGroup(groupId: ApiId) {
    return request.delete(`/projects/groups/${groupId}`)
  },

  /**
   * 绑定工程标签集合（全量替换）
   * @param {string} id - 工程 ID
   * @param {object} payload - 标签绑定数据
   * @returns {Promise} 绑定结果
   */
  bindProjectTags(id: ApiId, payload: ProjectTagBindingPayload) {
    return request.put(`/projects/${id}/tags`, payload)
  },

  /**
   * 绑定工程分组（可传 null 清空分组）
   * @param {string} id - 工程 ID
   * @param {object} payload - 分组绑定数据
   * @returns {Promise} 绑定结果
   */
  bindProjectGroup(id: ApiId, payload: ProjectGroupBindingPayload) {
    return request.put(`/projects/${id}/group`, payload)
  },

  /**
   * 创建工程
   * @param {object} projectData - 工程数据
   * @param {string} projectData.name - 工程名称
   * @param {string} projectData.description - 描述
   * @param {string} projectData.colorTag - 颜色标签
   * @returns {Promise} 创建结果
   */
  createProject(projectData: Record<string, unknown>) {
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
  updateProject(id: ApiId, projectData: ProjectUpdatePayload & Record<string, unknown>) {
    return request.put(`/projects/${id}`, projectData)
  },

  /**
   * 删除工程
   * @param {string} id - 工程ID
   * @param {object} options - 删除选项
   * @param {boolean} options.force - 是否强制删除（系统管理员）
   * @returns {Promise} 删除结果
   */
  deleteProject(id: ApiId, options: DeleteOptions = {}) {
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
  getDeleteImpact(id: ApiId) {
    return request.get(`/projects/${id}/delete-impact`)
  },

  /**
   * 执行工程运维操作
   * @param {string} id - 工程ID
   * @param {string} operation - 操作类型 (start, stop, restart, deploy, backup)
   * @returns {Promise} 操作结果
   */
  performOperation(id: ApiId, operation: string) {
    return request.post(`/projects/${id}/operations/${operation}`)
  },

  /**
   * 导出工程
   * @param {string} id - 工程ID
   * @returns {Promise} 导出结果
   */
  exportProject(id: ApiId) {
    return request.get(`/projects/${id}/export`, { responseType: 'blob' })
  },

  /**
   * 导入工程
   * @param {object} payload - 导入数据
   * @param {string} [payload.name] - 新工程名称
   * @param {object} payload.payload - 工程导出内容
   * @returns {Promise} 导入结果
   */
  importProject(payload: Record<string, unknown>) {
    return request.post('/projects/import', payload)
  },

  /**
   * 获取工程运行态用户列表
   * @param {string} id - 工程 ID
   * @returns {Promise} 用户列表
   */
  listRuntimeUsers(id: ApiId) {
    return request.get(`/projects/${id}/runtime-users`)
  },

  /**
   * 创建工程运行态用户
   * @param {string} id - 工程 ID
   * @param {object} payload - 用户数据
   * @returns {Promise} 创建结果
   */
  createRuntimeUser(id: ApiId, payload: Record<string, unknown>) {
    return request.post(`/projects/${id}/runtime-users`, payload)
  },

  /**
   * 更新工程运行态用户状态
   * @param {string} id - 工程 ID
   * @param {string} userId - 运行态用户 ID
   * @param {object} payload - 状态数据
   * @returns {Promise} 更新结果
   */
  updateRuntimeUserStatus(id: ApiId, userId: ApiId, payload: Record<string, unknown>) {
    return request.patch(`/projects/${id}/runtime-users/${userId}/status`, payload)
  },

  /**
   * 删除工程运行态用户
   * @param {string} id - 工程 ID
   * @param {string} userId - 运行态用户 ID
   * @returns {Promise} 删除结果
   */
  deleteRuntimeUser(id: ApiId, userId: ApiId) {
    return request.delete(`/projects/${id}/runtime-users/${userId}`)
  },

  /**
   * 更新工程运行态用户角色绑定
   * @param {string} id - 工程 ID
   * @param {string} userId - 运行态用户 ID
   * @param {object} payload - 角色绑定数据
   * @returns {Promise} 更新结果
   */
  updateRuntimeUserRoles(id: ApiId, userId: ApiId, payload: Record<string, unknown>) {
    return request.put(`/projects/${id}/runtime-users/${userId}/roles`, payload)
  },

  /**
   * 重置工程运行态用户密码
   * @param {string} id - 工程 ID
   * @param {string} userId - 运行态用户 ID
   * @param {object} payload - 新密码数据
   * @returns {Promise} 重置结果
   */
  resetRuntimeUserPassword(id: ApiId, userId: ApiId, payload: Record<string, unknown>) {
    return request.post(`/projects/${id}/runtime-users/${userId}/reset-password`, payload)
  },

  /**
   * 获取工程运行态角色列表
   * @param {string} id - 工程 ID
   * @returns {Promise} 角色列表
   */
  listRuntimeRoles(id: ApiId) {
    return request.get(`/projects/${id}/runtime-roles`)
  },

  /**
   * 创建工程运行态角色
   * @param {string} id - 工程 ID
   * @param {object} payload - 角色数据
   * @returns {Promise} 创建结果
   */
  createRuntimeRole(id: ApiId, payload: Record<string, unknown>) {
    return request.post(`/projects/${id}/runtime-roles`, payload)
  },

  /**
   * 更新工程运行态角色
   * @param {string} id - 工程 ID
   * @param {string} roleId - 角色 ID
   * @param {object} payload - 角色数据
   * @returns {Promise} 更新结果
   */
  updateRuntimeRole(id: ApiId, roleId: ApiId, payload: Record<string, unknown>) {
    return request.put(`/projects/${id}/runtime-roles/${roleId}`, payload)
  },

  /**
   * 删除工程运行态角色
   * @param {string} id - 工程 ID
   * @param {string} roleId - 角色 ID
   * @returns {Promise} 删除结果
   */
  deleteRuntimeRole(id: ApiId, roleId: ApiId) {
    return request.delete(`/projects/${id}/runtime-roles/${roleId}`)
  },
}
