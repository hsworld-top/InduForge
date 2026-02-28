import request from "@/utils/request";

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
  getTenants(params = {}) {
    return request.get("/tenants", { params });
  },

  /**
   * 获取租户详情
   * @param {number} id - 租户ID
   * @returns {Promise} 租户详情
   */
  getTenantById(id) {
    return request.get(`/tenants/${id}`);
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
  createTenant(tenantData) {
    return request.post("/tenants", tenantData);
  },

  /**
   * 更新租户
   * @param {string} code - 租户代码
   * @param {object} tenantData - 租户数据
   * @returns {Promise} 更新结果
   */
  updateTenant(code, tenantData) {
    return request.put(`/tenants/${code}`, tenantData);
  },

  /**
   * 删除租户
   * @param {number} id - 租户ID
   * @returns {Promise} 删除结果
   */
  deleteTenant(id) {
    return request.delete(`/tenants/${id}`);
  },

  /**
   * 激活租户
   * @param {number} id - 租户ID
   * @returns {Promise} 激活结果
   */
  activateTenant(id) {
    return request.post(`/tenants/${id}/activate`);
  },

  /**
   * 暂停租户
   * @param {number} id - 租户ID
   * @returns {Promise} 暂停结果
   */
  suspendTenant(id) {
    return request.post(`/tenants/${id}/suspend`);
  },

  /**
   * 上传租户文件
   * @param {string} tenantId - 租户ID或代码
   * @param {string} type - 文件类型 ('logo' | 'background')
   * @param {FormData} formData - 文件数据
   * @returns {Promise} 上传结果
   */
  uploadFile(tenantId, type, formData) {
    return request.post(`/tenants/${tenantId}/upload?type=${type}`, formData);
  },
};
