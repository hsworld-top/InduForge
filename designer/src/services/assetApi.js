/**
 * 资源 API 服务
 * 提供工程内资源（图片、字体等）的文件夹与文件管理
 */

import request from "@/utils/request";

export const assetApi = {
  /**
   * 获取资源文件夹列表
   * @param {string} projectId - 工程 ID
   * @returns {Promise<Array>}
   */
  getFolders(projectId) {
    return request.get(`/design/projects/${projectId}/asset-folders`);
  },
  /**
   * 创建资源文件夹
   * @param {string} projectId - 工程 ID
   * @param {Object} data - 文件夹数据（如 name、parentId）
   * @returns {Promise<Object>}
   */
  createFolder(projectId, data) {
    return request.post(`/design/projects/${projectId}/asset-folders`, data);
  },
  /**
   * 重命名资源文件夹
   * @param {string} projectId - 工程 ID
   * @param {string} folderId - 文件夹 ID
   * @param {Object} payload - 更新数据（如 name）
   * @returns {Promise<void>}
   */
  renameFolder(projectId, folderId, payload = {}) {
    return request.patch(
      `/design/projects/${projectId}/asset-folders/${folderId}`,
      payload,
    );
  },
  /**
   * 删除资源文件夹
   * @param {string} projectId - 工程 ID
   * @param {string} folderId - 文件夹 ID
   * @returns {Promise<void>}
   */
  deleteFolder(projectId, folderId) {
    return request.delete(
      `/design/projects/${projectId}/asset-folders/${folderId}`,
    );
  },
  /**
   * 更新资源元信息
   * @param {string} projectId - 工程 ID
   * @param {string} assetId - 资源 ID
   * @param {Object} payload - 更新数据
   * @returns {Promise<void>}
   */
  updateAsset(projectId, assetId, payload = {}) {
    return request.patch(
      `/design/projects/${projectId}/assets/${assetId}`,
      payload,
    );
  },
  /**
   * 复制资源
   * @param {string} projectId - 工程 ID
   * @param {string} assetId - 资源 ID
   * @param {Object} payload - 复制参数（如 folderId）
   * @returns {Promise<Object>}
   */
  copyAsset(projectId, assetId, payload = {}) {
    return request.post(
      `/design/projects/${projectId}/assets/${assetId}/copy`,
      payload,
    );
  },
  /**
   * 获取资源列表
   * @param {string} projectId - 工程 ID
   * @param {Object} params - 查询参数（如 folderId、type）
   * @returns {Promise<Object>}
   */
  getAssets(projectId, params = {}) {
    return request.get(`/design/projects/${projectId}/assets`, { params });
  },
  /**
   * 上传资源文件
   * @param {string} projectId - 工程 ID
   * @param {File[]} files - 文件列表
   * @param {string} [folderId] - 目标文件夹 ID
   * @param {Object} [options] - 选项（如 conflictStrategy）
   * @param {Function} [onProgress] - 上传进度回调
   * @returns {Promise<Object>}
   */
  uploadAssets(projectId, files, folderId, options = {}, onProgress) {
    const form = new FormData();
    (files || []).forEach((file) => form.append("files", file));
    if (folderId) {
      form.append("folderId", folderId);
    }
    if (options?.conflictStrategy) {
      form.append("conflictStrategy", options.conflictStrategy);
    }
    return request.post(`/design/projects/${projectId}/assets`, form, {
      headers: {
        "Content-Type": "multipart/form-data",
      },
      onUploadProgress: onProgress,
    });
  },
  /**
   * 删除资源
   * @param {string} projectId - 工程 ID
   * @param {string} assetId - 资源 ID
   * @returns {Promise<void>}
   */
  deleteAsset(projectId, assetId) {
    return request.delete(`/design/projects/${projectId}/assets/${assetId}`);
  },
};

export default assetApi;
