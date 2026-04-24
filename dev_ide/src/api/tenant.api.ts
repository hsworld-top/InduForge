import request from '@/utils/request'

type ApiId = string | number
type QueryParams = Record<string, unknown>
type TenantPayload = Record<string, unknown>

/**
 * 租户管理 API
 */
export const tenantAPI = {
  /**
   * 获取租户列表
   * @param {object} params - 查询参数
   * @param {number} params.page - 页码
   * @param {number} params.limit - 每页大小
   * @param {string} params.keyword - 搜索关键词
   * @param {string} params.status - 状态筛选
   * @returns {Promise} 租户列表
   */
  getTenants(params: QueryParams = {}) {
    return request.get('/tenants', { params })
  },

  /**
   * 获取租户详情
   * @param {number} id - 租户ID
   * @returns {Promise} 租户详情
   */
  getTenantById(id: ApiId) {
    return request.get(`/tenants/${id}`)
  },

  /**
   * 获取当前登录用户所属租户品牌信息。
   * @returns {Promise} 当前租户信息
   */
  getCurrentTenant() {
    return request.get('/tenants/current')
  },

  /**
   * 获取当前租户仪表盘共享便签列表。
   * @returns {Promise} 当前租户共享便签列表
   */
  getDashboardNotes() {
    return request.get('/tenants/current/dashboard-notes')
  },

  /**
   * 新增当前租户仪表盘共享便签。
   * @param {string} content - 便签正文，最多 2000 个字符
   * @returns {Promise} 新增后的共享便签
   */
  createDashboardNote(content: string) {
    return request.post('/tenants/current/dashboard-notes', { content })
  },

  /**
   * 更新当前租户仪表盘共享便签。
   * @param {string} noteId - 便签 ID
   * @param {string} content - 便签正文，最多 2000 个字符
   * @returns {Promise} 更新后的共享便签
   */
  updateDashboardNote(noteId: string, content: string) {
    return request.put(`/tenants/current/dashboard-notes/${noteId}`, { content })
  },

  /**
   * 删除当前租户仪表盘共享便签。
   * @param {string} noteId - 便签 ID
   * @returns {Promise} 删除结果
   */
  deleteDashboardNote(noteId: string) {
    return request.delete(`/tenants/current/dashboard-notes/${noteId}`)
  },

  /**
   * 创建租户
   * @param {object} tenantData - 租户数据
   * @param {string} tenantData.name - 租户名称
   * @param {string} tenantData.code - 租户代码
   * @param {string} tenantData.description - 描述
   * @param {string} tenantData.contactEmail - 联系邮箱
   * @param {string} tenantData.contactPhone - 联系电话
   * @param {string} tenantData.logoUrl - Logo URL
   * @param {string} tenantData.loginBackgroundUrl - 登录背景图片 URL
   * @param {object} tenantData.settings - 租户扩展配置
   * @returns {Promise} 创建结果
   */
  createTenant(tenantData: TenantPayload) {
    return request.post('/tenants', tenantData)
  },

  /**
   * 更新租户
   * @param {string} code - 租户代码
   * @param {object} tenantData - 租户数据
   * @returns {Promise} 更新结果
   */
  updateTenant(code: string, tenantData: TenantPayload) {
    return request.put(`/tenants/${code}`, tenantData)
  },

  /**
   * 删除租户
   * @param {number} id - 租户ID
   * @returns {Promise} 删除结果
   */
  deleteTenant(id: ApiId) {
    return request.delete(`/tenants/${id}`)
  },

  /**
   * 激活租户
   * @param {number} id - 租户ID
   * @returns {Promise} 激活结果
   */
  activateTenant(id: ApiId) {
    return request.post(`/tenants/${id}/activate`)
  },

  /**
   * 暂停租户
   * @param {number} id - 租户ID
   * @returns {Promise} 暂停结果
   */
  suspendTenant(id: ApiId) {
    return request.post(`/tenants/${id}/suspend`)
  },

  /**
   * 上传租户文件
   * @param {string} tenantId - 租户ID或代码
   * @param {string} type - 文件类型 ('logo' | 'background')
   * @param {FormData} formData - 文件数据
   * @returns {Promise} 上传结果
   */
  uploadFile(tenantId: ApiId, type: string, formData: FormData) {
    return request.post(`/tenants/${tenantId}/upload?type=${type}`, formData)
  },
}
