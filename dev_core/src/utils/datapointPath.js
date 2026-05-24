/**
 * 数据点路径规范化工具
 * 负责对路径段进行统一处理，保证路径可用于索引与订阅
 */

/**
 * 规范化单个路径段
 * @param {string} segment - 原始路径段
 * @returns {string} 规范化后的路径段
 */
function normalizeSegment(segment) {
  if (segment === null || segment === undefined) {
    return ''
  }

  const raw = String(segment).trim()
  if (!raw) {
    return ''
  }

  return raw
    .replace(/[./\\]+/g, '_')
    .replace(/\s+/g, '_')
    .replace(/[^\w\u4e00-\u9fa5-]+/g, '_')
    .replace(/_+/g, '_')
    .replace(/^_+|_+$/g, '')
}

/**
 * 规范化完整路径
 * @param {string} path - 原始路径
 * @returns {string} 规范化后的路径
 */
function normalizePath(path) {
  if (!path) {
    return ''
  }

  const segments = String(path)
    .split('.')
    .map((segment) => normalizeSegment(segment))
    .filter(Boolean)

  return segments.join('.')
}

module.exports = {
  normalizeSegment,
  normalizePath,
}
