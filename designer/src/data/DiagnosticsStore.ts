// @ts-nocheck — 待 data/types 迁 TS 后补全 DatapointStatus 等类型
/**
 * DiagnosticsStore - 数据点状态追踪
 * 管理和追踪所有数据点的状态
 */

import { EventEmitter } from "../editor-core/utils/EventEmitter.ts";

/**
 * @typedef {import('./types.js').DatapointStatus} DatapointStatus
 * @typedef {import('./types.js').DatapointStatusInfo} DatapointStatusInfo
 * @typedef {import('./types.js').DiagnosticInfo} DiagnosticInfo
 * @typedef {import('./types.js').DiagnosticsSummary} DiagnosticsSummary
 */

/**
 * 诊断存储类
 */
export class DiagnosticsStore extends EventEmitter {
  /**
   * 创建诊断存储
   * @param {object} [options] - 配置选项
   * @param {string} [options.apiBaseUrl] - API 基础路径
   * @param {number} [options.cacheExpireTime] - 缓存过期时间（毫秒）
   * @param {number} [options.batchSize] - 批量查询大小
   */
  constructor(options = {}) {
    super();

    /** @type {string} */
    this._apiBaseUrl = options.apiBaseUrl || "/api/v1";

    /** @type {number} */
    this._cacheExpireTime = options.cacheExpireTime ?? 30000;

    /** @type {number} */
    this._batchSize = options.batchSize ?? 50;

    /** @type {Map<string, {info: DatapointStatusInfo, timestamp: number}>} 状态缓存 */
    this._cache = new Map();

    /** @type {Map<string, Set<string>>} 节点到数据点的映射 */
    this._nodeBindings = new Map();

    /** @type {Map<string, Set<string>>} 数据点到节点的映射 */
    this._datapointNodes = new Map();

    /** @type {Set<string>} 待查询队列 */
    this._pendingPaths = new Set();

    /** @type {number | null} 批量查询定时器 */
    this._batchTimer = null;

    /** @type {number} 批量查询延迟 */
    this._batchDelay = 100;
  }

  // ==================== 状态查询 ====================

  /**
   * 获取数据点状态
   * @param {string} path - 数据点路径
   * @param {boolean} [forceRefresh] - 是否强制刷新
   * @returns {Promise<DatapointStatusInfo>}
   */
  async getStatus(path, forceRefresh = false) {
    // 检查缓存
    if (!forceRefresh) {
      const cached = this._getFromCache(path);
      if (cached) return cached;
    }

    // 添加到待查询队列并触发批量查询
    this._addToPending(path);

    // 等待查询完成
    return new Promise((resolve) => {
      const handler = (data) => {
        if (data.path === path) {
          this.off("statusUpdated", handler);
          resolve(data);
        }
      };
      this.on("statusUpdated", handler);

      // 超时处理
      setTimeout(() => {
        this.off("statusUpdated", handler);
        resolve(this._getFromCache(path) || this._createUnknownStatus(path));
      }, 5000);
    });
  }

  /**
   * 批量获取数据点状态
   * @param {string[]} paths - 数据点路径数组
   * @param {boolean} [forceRefresh] - 是否强制刷新
   * @returns {Promise<Map<string, DatapointStatusInfo>>}
   */
  async getStatusBatch(paths, forceRefresh = false) {
    const results = new Map();
    const needFetch = [];

    // 检查缓存
    for (const path of paths) {
      if (!forceRefresh) {
        const cached = this._getFromCache(path);
        if (cached) {
          results.set(path, cached);
          continue;
        }
      }
      needFetch.push(path);
    }

    // 如果有需要查询的
    if (needFetch.length > 0) {
      await this._fetchStatusBatch(needFetch);

      // 从缓存获取结果
      for (const path of needFetch) {
        const cached = this._getFromCache(path);
        results.set(path, cached || this._createUnknownStatus(path));
      }
    }

    return results;
  }

  /**
   * 从缓存获取状态
   * @param {string} path - 数据点路径
   * @returns {DatapointStatusInfo | null}
   * @private
   */
  _getFromCache(path) {
    const cached = this._cache.get(path);
    if (!cached) return null;

    // 检查是否过期
    if (Date.now() - cached.timestamp > this._cacheExpireTime) {
      this._cache.delete(path);
      return null;
    }

    return cached.info;
  }

  /**
   * 创建未知状态
   * @param {string} path - 数据点路径
   * @returns {DatapointStatusInfo}
   * @private
   */
  _createUnknownStatus(path) {
    return {
      path,
      status: "unknown",
      statusReason: "数据点未找到或未配置",
    };
  }

  // ==================== 批量查询 ====================

  /**
   * 添加到待查询队列
   * @param {string} path - 数据点路径
   * @private
   */
  _addToPending(path) {
    this._pendingPaths.add(path);
    this._scheduleBatchFetch();
  }

  /**
   * 调度批量查询
   * @private
   */
  _scheduleBatchFetch() {
    if (this._batchTimer) return;

    this._batchTimer = setTimeout(async () => {
      this._batchTimer = null;

      if (this._pendingPaths.size === 0) return;

      // 取出待查询路径
      const paths = Array.from(this._pendingPaths);
      this._pendingPaths.clear();

      // 分批查询
      for (let i = 0; i < paths.length; i += this._batchSize) {
        const batch = paths.slice(i, i + this._batchSize);
        await this._fetchStatusBatch(batch);
      }
    }, this._batchDelay);
  }

  /**
   * 批量查询状态（API 调用）
   * @param {string[]} paths - 数据点路径数组
   * @private
   */
  async _fetchStatusBatch(paths) {
    try {
      const response = await fetch(`${this._apiBaseUrl}/datapoints/status`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ paths }),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      const result = await response.json();

      if (result.success && result.data) {
        // 更新缓存
        for (const info of result.data) {
          this._updateCache(info);
        }
      }
    } catch (error) {
      console.error("Failed to fetch datapoint status:", error);

      // 查询失败时，将路径标记为 unknown
      for (const path of paths) {
        this._updateCache({
          path,
          status: "unknown",
          statusReason: `查询失败: ${error.message}`,
        });
      }
    }
  }

  /**
   * 更新缓存
   * @param {DatapointStatusInfo} info - 状态信息
   * @private
   */
  _updateCache(info) {
    const oldInfo = this._cache.get(info.path)?.info;

    this._cache.set(info.path, {
      info,
      timestamp: Date.now(),
    });

    // 如果状态变化，触发事件
    if (!oldInfo || oldInfo.status !== info.status) {
      this.emit("statusUpdated", info);
      this.emit("statusChange", {
        path: info.path,
        oldStatus: oldInfo?.status,
        newStatus: info.status,
        info,
      });
    }
  }

  // ==================== 绑定追踪 ====================

  /**
   * 注册节点绑定
   * @param {string} nodeId - 节点 ID
   * @param {string[]} datapointPaths - 数据点路径数组
   */
  registerNodeBindings(nodeId, datapointPaths) {
    // 清除旧绑定
    this.unregisterNodeBindings(nodeId);

    // 注册新绑定
    this._nodeBindings.set(nodeId, new Set(datapointPaths));

    for (const path of datapointPaths) {
      if (!this._datapointNodes.has(path)) {
        this._datapointNodes.set(path, new Set());
      }
      this._datapointNodes.get(path).add(nodeId);
    }
  }

  /**
   * 取消注册节点绑定
   * @param {string} nodeId - 节点 ID
   */
  unregisterNodeBindings(nodeId) {
    const paths = this._nodeBindings.get(nodeId);
    if (!paths) return;

    for (const path of paths) {
      this._datapointNodes.get(path)?.delete(nodeId);
    }

    this._nodeBindings.delete(nodeId);
  }

  /**
   * 获取节点绑定的数据点
   * @param {string} nodeId - 节点 ID
   * @returns {string[]}
   */
  getNodeBindings(nodeId) {
    const paths = this._nodeBindings.get(nodeId);
    return paths ? Array.from(paths) : [];
  }

  /**
   * 获取使用某数据点的节点
   * @param {string} path - 数据点路径
   * @returns {string[]}
   */
  getDatapointNodes(path) {
    const nodes = this._datapointNodes.get(path);
    return nodes ? Array.from(nodes) : [];
  }

  // ==================== 诊断报告 ====================

  /**
   * 获取诊断摘要
   * @returns {DiagnosticsSummary}
   */
  getSummary() {
    const issues = [];
    let active = 0;
    let invalid = 0;
    let unknown = 0;

    for (const [path, cached] of this._cache.entries()) {
      const { info } = cached;
      switch (info.status) {
        case "active":
          active++;
          break;
        case "invalid":
          invalid++;
          issues.push({
            path,
            status: info.status,
            reason: info.statusReason,
            affectedNodeIds: this.getDatapointNodes(path),
            lastCheckedAt: cached.timestamp,
          });
          break;
        case "unknown":
          unknown++;
          issues.push({
            path,
            status: info.status,
            reason: info.statusReason,
            affectedNodeIds: this.getDatapointNodes(path),
            lastCheckedAt: cached.timestamp,
          });
          break;
      }
    }

    return {
      total: this._cache.size,
      active,
      invalid,
      unknown,
      issues,
    };
  }

  /**
   * 获取所有问题数据点
   * @returns {DiagnosticInfo[]}
   */
  getIssues() {
    const issues = [];

    for (const [path, cached] of this._cache.entries()) {
      const { info } = cached;
      if (info.status !== "active") {
        issues.push({
          path,
          status: info.status,
          reason: info.statusReason,
          affectedNodeIds: this.getDatapointNodes(path),
          lastCheckedAt: cached.timestamp,
        });
      }
    }

    return issues;
  }

  /**
   * 获取特定状态的数据点
   * @param {DatapointStatus} status - 状态
   * @returns {string[]}
   */
  getPathsByStatus(status) {
    const paths = [];
    for (const [path, cached] of this._cache.entries()) {
      if (cached.info.status === status) {
        paths.push(path);
      }
    }
    return paths;
  }

  // ==================== 手动状态管理 ====================

  /**
   * 手动设置状态（用于 WebSocket 推送）
   * @param {DatapointStatusInfo} info - 状态信息
   */
  setStatus(info) {
    this._updateCache(info);
  }

  /**
   * 批量设置状态
   * @param {DatapointStatusInfo[]} infos - 状态信息数组
   */
  setStatusBatch(infos) {
    for (const info of infos) {
      this._updateCache(info);
    }
  }

  /**
   * 标记数据点为失效
   * @param {string} path - 数据点路径
   * @param {string} [reason] - 原因
   */
  markInvalid(path, reason) {
    this._updateCache({
      path,
      status: "invalid",
      statusReason: reason || "手动标记为失效",
    });
  }

  /**
   * 标记数据点为活跃
   * @param {string} path - 数据点路径
   */
  markActive(path) {
    this._updateCache({
      path,
      status: "active",
    });
  }

  // ==================== 缓存管理 ====================

  /**
   * 清除缓存
   */
  clearCache() {
    this._cache.clear();
    this.emit("cacheCleared");
  }

  /**
   * 清除过期缓存
   */
  clearExpiredCache() {
    const now = Date.now();
    for (const [path, cached] of this._cache.entries()) {
      if (now - cached.timestamp > this._cacheExpireTime) {
        this._cache.delete(path);
      }
    }
  }

  /**
   * 刷新所有缓存
   * @returns {Promise<void>}
   */
  async refreshAll() {
    const paths = Array.from(this._cache.keys());
    if (paths.length === 0) return;

    // 分批刷新
    for (let i = 0; i < paths.length; i += this._batchSize) {
      const batch = paths.slice(i, i + this._batchSize);
      await this._fetchStatusBatch(batch);
    }
  }

  // ==================== 销毁 ====================

  /**
   * 销毁
   */
  destroy() {
    if (this._batchTimer) {
      clearTimeout(this._batchTimer);
      this._batchTimer = null;
    }

    this._cache.clear();
    this._nodeBindings.clear();
    this._datapointNodes.clear();
    this._pendingPaths.clear();

    this.removeAllListeners();
  }
}

/**
 * 创建诊断存储实例
 * @param {object} [options] - 配置选项
 * @returns {DiagnosticsStore}
 */
export function createDiagnosticsStore(options) {
  return new DiagnosticsStore(options);
}

export default DiagnosticsStore;
