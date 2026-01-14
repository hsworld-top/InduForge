/**
 * 数据中心 API 服务
 * 提供数据连接、查询、数据点相关的 API 调用
 */

import request from "@/utils/request";

/**
 * 数据中心 API
 */
export const datacenterApi = {
  /**
   * 获取数据连接列表
   * @param {string} projectId - 工程ID
   * @param {Object} [params] - 查询参数
   * @returns {Promise<Array>} 连接列表
   */
  getConnections(projectId, params = {}) {
    return request.get(`/data/projects/${projectId}/connections`, { params });
  },

  /**
   * 获取数据查询列表
   * @param {string} projectId - 项目ID
   * @param {Object} [params] - 查询参数
   * @returns {Promise<Array>} 查询列表
   */
  getQueries(projectId, params = {}) {
    return request.get(`/data/projects/${projectId}/queries`, { params });
  },

  /**
   * 获取数据点列表（兼容旧路径）
   * @param {string} projectId - 项目ID
   * @param {string} connectionId - 连接ID
   * @returns {Promise<Array>} 数据点列表
   */
  getDatapoints(projectId, connectionId) {
    return request.get(
      `/data/projects/${projectId}/connections/${connectionId}/datapoints`
    );
  },

  /**
   * 获取数据点列表
   * @param {string} projectId - 项目ID
   * @param {Object} [params] - 查询参数
   * @returns {Promise<Object>} 数据点列表
   */
  getDataPoints(projectId, params = {}) {
    return request.get(`/data/projects/${projectId}/datapoints`, { params });
  },

  /**
   * 获取数据点状态
   * @param {string} projectId - 项目ID
   * @param {string[]} datapointIds - 数据点ID列表
   * @returns {Promise<Object>} 数据点状态映射
   */
  getDatapointStatus(projectId, datapointIds) {
    return request.post(`/data/projects/${projectId}/datapoints/status`, {
      datapointIds,
    });
  },

  /**
   * 获取数据点实时值
   * @param {string} projectId - 项目ID
   * @param {string[]} datapointIds - 数据点ID列表
   * @returns {Promise<Object>} 数据点值映射
   */
  getDatapointValues(projectId, datapointIds) {
    return request.post(`/data/projects/${projectId}/datapoints/values`, {
      datapointIds,
    });
  },

  /**
   * 写入数据点值
   * @param {string} projectId - 项目ID
   * @param {string} datapointId - 数据点ID
   * @param {*} value - 要写入的值
   * @returns {Promise<void>}
   */
  writeDatapointValue(projectId, datapointId, value) {
    return request.post(
      `/data/projects/${projectId}/datapoints/${datapointId}/write`,
      { value }
    );
  },
};

export default datacenterApi;