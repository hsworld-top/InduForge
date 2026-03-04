import request from '@/utils/request'

/**
 * 系统日志 API
 */
export const logAPI = {
  /**
   * 获取系统日志列表
   * @param {object} params - 查询参数
   * @param {number} params.page - 页码
   * @param {number} params.size - 每页大小
   * @param {string} params.level - 日志级别 (ERROR, WARN, INFO, DEBUG)
   * @param {string} params.keyword - 搜索关键词
   * @param {string} params.startDate - 开始日期
   * @param {string} params.endDate - 结束日期
   * @param {string} params.userId - 用户ID
   * @returns {Promise} 日志列表
   */
  getLogs(params = {}) {
    return request.get('/logs', { params })
  },

  /**
   * 获取日志详情
   * @param {number} id - 日志ID
   * @returns {Promise} 日志详情
   */
  getLogById(id) {
    return request.get(`/logs/${id}`)
  },

  /**
   * 导出日志
   * @param {object} params - 导出参数
   * @returns {Promise} 导出结果
   */
  exportLogs(params = {}) {
    return request.get('/logs/export', {
      params,
      responseType: 'blob',
    })
  },

  /**
   * 删除旧日志
   * @param {string} beforeDate - 删除此日期之前的日志
   * @returns {Promise} 删除结果
   */
  deleteOldLogs(beforeDate) {
    return request.delete('/logs', { params: { beforeDate } })
  },

  /**
   * 获取日志统计信息
   * @param {object} params - 统计参数
   * @param {string} params.startDate - 开始日期
   * @param {string} params.endDate - 结束日期
   * @returns {Promise} 统计信息
   */
  getLogStats(params = {}) {
    return request.get('/logs/stats', { params })
  },

  /**
   * 获取仪表盘最近活动（租户维度）
   * @param {object} params - 查询参数
   * @param {number} params.limit - 数量限制
   * @returns {Promise} 最近活动
   */
  getRecentActivities(params = {}) {
    return request.get('/logs/recent-activities', { params })
  },
}
