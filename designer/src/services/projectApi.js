/**
 * 项目 API 服务
 * 提供页面管理的 API 调用
 */

import request from "@/utils/request";

/**
 * 项目 API
 */
export const projectApi = {
  /**
   * 获取项目的页面列表
   * @param {string} projectId - 项目ID
   * @returns {Promise<Array>} 页面列表
   */
  getPages(projectId) {
    return request.get(`/design/projects/${projectId}/pages`);
  },

  /**
   * 获取单个页面的完整 Schema
   * @param {string} projectId - 项目ID
   * @param {string} pageId - 页面ID
   * @returns {Promise<Object>} 完整的 Page Schema
   */
  getPage(projectId, pageId) {
    return request.get(`/design/projects/${projectId}/pages/${pageId}`);
  },

  /**
   * 创建新页面
   * @param {string} projectId - 项目ID
   * @param {Object} data - 页面数据 { name, type, parentId, schemaContent }
   * @returns {Promise<Object>} 创建的页面数据
   */
  createPage(projectId, data) {
    return request.post(`/design/projects/${projectId}/pages`, data);
  },

  /**
   * 更新页面 Schema
   * @param {string} projectId - 项目ID
   * @param {string} pageId - 页面ID
   * @param {Object} schema - 更新的 Page Schema
   * @returns {Promise<void>}
   */
  updatePage(projectId, pageId, schema) {
    return request.put(`/design/projects/${projectId}/pages/${pageId}`, {
      schema,
    });
  },

  /**
   * 删除页面
   * @param {string} projectId - 项目ID
   * @param {string} pageId - 页面ID
   * @returns {Promise<void>}
   */
  deletePage(projectId, pageId) {
    return request.delete(`/design/projects/${projectId}/pages/${pageId}`);
  },

  /**
   * 重命名页面
   * @param {string} projectId - 项目ID
   * @param {string} pageId - 页面ID
   * @param {string} name - 新名称
   * @returns {Promise<void>}
   */
  renamePage(projectId, pageId, name) {
    return request.patch(
      `/design/projects/${projectId}/pages/${pageId}/rename`,
      { name }
    );
  },

  /**
   * 移动页面到指定页面组
   * @param {string} projectId - 项目ID
   * @param {string} pageId - 页面ID
   * @param {string|null} targetGroupId - 目标页面组ID
   * @returns {Promise<void>}
   */
  movePageToGroup(projectId, pageId, targetGroupId) {
    return request.patch(`/design/projects/${projectId}/pages/${pageId}/move`, {
      parentId: targetGroupId,
    });
  },

  /**
   * 获取工程级别全局变量
   * @param {string} projectId - 项目ID
   * @returns {Promise<Object>}
   */
  getProjectVariables(projectId) {
    return request.get(`/design/projects/${projectId}/variables`);
  },

  /**
   * 更新工程级别全局变量
   * @param {string} projectId - 项目ID
   * @param {Object} variables - 变量对象
   * @returns {Promise<Object>}
   */
  updateProjectVariables(projectId, variables) {
    return request.put(`/design/projects/${projectId}/variables`, { variables });
  },
};

export default projectApi;

