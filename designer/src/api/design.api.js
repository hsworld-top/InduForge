/**
 * Design API - 设计中心 API 模块
 * 提供页面管理的 API 调用
 * Requirements: 7.1, 7.2, 7.3, 7.4, 7.5
 */
import request from '@/utils/request'

/**
 * 设计中心 API
 */
export const designAPI = {
  /**
   * 获取项目的页面列表
   * Requirements: 7.1
   * @param {string} projectId - 项目ID
   * @returns {Promise<Array>} 页面列表，包含 id, name, type, parentId
   */
  getPages(projectId) {
    return request.get(`/design/projects/${projectId}/pages`)
  },

  /**
   * 获取单个页面的完整 Schema
   * Requirements: 7.2
   * @param {string} pageId - 页面ID
   * @returns {Promise<Object>} 完整的 Page Schema
   */
  getPage(pageId) {
    return request.get(`/design/pages/${pageId}`)
  },

  /**
   * 创建新页面
   * Requirements: 7.3
   * @param {string} projectId - 项目ID
   * @param {Object} data - 页面数据 { name, type, parentId, schemaContent }
   * @returns {Promise<Object>} 创建的页面数据
   */
  createPage(projectId, data) {
    return request.post(`/design/projects/${projectId}/pages`, data)
  },

  /**
   * 更新页面 Schema
   * Requirements: 7.4
   * @param {string} pageId - 页面ID
   * @param {Object} schema - 更新的 Page Schema
   * @returns {Promise<void>}
   */
  updatePage(pageId, schema) {
    return request.put(`/design/pages/${pageId}`, schema)
  },

  /**
   * 删除页面
   * Requirements: 7.5
   * @param {string} pageId - 页面ID
   * @returns {Promise<void>}
   */
  deletePage(pageId) {
    return request.delete(`/design/pages/${pageId}`)
  },

  /**
   * 重命名页面
   * @param {string} pageId - 页面ID
   * @param {string} name - 新名称
   * @returns {Promise<void>}
   */
  renamePage(pageId, name) {
    return request.patch(`/design/pages/${pageId}/rename`, { name })
  },
}

export default designAPI
