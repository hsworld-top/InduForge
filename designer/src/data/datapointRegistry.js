/**
 * 数据点注册表
 * 管理数据点的注册、查询和状态追踪
 */

import { EventEmitter } from "../editor-core/utils/EventEmitter.js";

/**
 * @typedef {import('./types.js').DatapointStatus} DatapointStatus
 * @typedef {import('./types.js').DatapointStatusInfo} DatapointStatusInfo
 */

/**
 * @typedef {Object} DatapointMeta
 * 数据点元信息
 * @property {string} id - 数据点 ID
 * @property {string} path - 数据点路径
 * @property {string} provider - 数据提供者 ID
 * @property {string} [dataType] - 数据类型
 * @property {string} [unit] - 单位
 * @property {string} [description] - 描述
 * @property {DatapointStatus} [status] - 状态
 * @property {*} [lastValue] - 最后一次的值
 * @property {number} [lastUpdated] - 最后更新时间
 */

/**
 * 数据点注册表事件类型
 * @enum {string}
 */
export const DatapointRegistryEvents = {
  /** 数据点注册 */
  REGISTERED: "registered",
  /** 数据点注销 */
  UNREGISTERED: "unregistered",
  /** 数据点状态变化 */
  STATUS_CHANGED: "statusChanged",
  /** 数据点值更新 */
  VALUE_UPDATED: "valueUpdated",
};

/**
 * 数据点注册表类
 */
export class DatapointRegistry extends EventEmitter {
  constructor() {
    super();

    /** @type {Map<string, DatapointMeta>} */
    this._datapoints = new Map();

    /** @type {Map<string, Set<string>>} */
    this._byProvider = new Map();

    /** @type {Map<string, Set<string>>} */
    this._subscribers = new Map(); // datapointId -> Set<componentId>
  }

  /**
   * 注册数据点
   * @param {DatapointMeta} meta - 数据点元信息
   */
  register(meta) {
    if (!meta.id || !meta.path) {
      throw new Error("数据点元信息缺少必要字段 (id, path)");
    }

    this._datapoints.set(meta.id, {
      ...meta,
      status: meta.status || "unknown",
      lastUpdated: Date.now(),
    });

    // 更新提供者索引
    const provider = meta.provider || "default";
    if (!this._byProvider.has(provider)) {
      this._byProvider.set(provider, new Set());
    }
    this._byProvider.get(provider).add(meta.id);

    this.emit(DatapointRegistryEvents.REGISTERED, meta);
  }

  /**
   * 批量注册数据点
   * @param {DatapointMeta[]} metas - 数据点元信息列表
   */
  registerAll(metas) {
    for (const meta of metas) {
      this.register(meta);
    }
  }

  /**
   * 注销数据点
   * @param {string} id - 数据点 ID
   */
  unregister(id) {
    const meta = this._datapoints.get(id);
    if (meta) {
      // 从提供者索引中移除
      const providerSet = this._byProvider.get(meta.provider || "default");
      if (providerSet) {
        providerSet.delete(id);
      }

      this._datapoints.delete(id);
      this._subscribers.delete(id);

      this.emit(DatapointRegistryEvents.UNREGISTERED, { id });
    }
  }

  /**
   * 获取数据点元信息
   * @param {string} id - 数据点 ID
   * @returns {DatapointMeta | undefined}
   */
  get(id) {
    return this._datapoints.get(id);
  }

  /**
   * 通过路径获取数据点
   * @param {string} path - 数据点路径
   * @returns {DatapointMeta | undefined}
   */
  getByPath(path) {
    for (const meta of this._datapoints.values()) {
      if (meta.path === path) {
        return meta;
      }
    }
    return undefined;
  }

  /**
   * 检查数据点是否已注册
   * @param {string} id - 数据点 ID
   * @returns {boolean}
   */
  has(id) {
    return this._datapoints.has(id);
  }

  /**
   * 获取所有数据点
   * @returns {DatapointMeta[]}
   */
  getAll() {
    return Array.from(this._datapoints.values());
  }

  /**
   * 按提供者获取数据点
   * @param {string} provider - 提供者 ID
   * @returns {DatapointMeta[]}
   */
  getByProvider(provider) {
    const ids = this._byProvider.get(provider);
    if (!ids) return [];

    return Array.from(ids)
      .map((id) => this._datapoints.get(id))
      .filter(Boolean);
  }

  /**
   * 更新数据点状态
   * @param {string} id - 数据点 ID
   * @param {DatapointStatus} status - 新状态
   * @param {string} [reason] - 状态变更原因
   */
  updateStatus(id, status, reason) {
    const meta = this._datapoints.get(id);
    if (meta) {
      const oldStatus = meta.status;
      meta.status = status;
      meta.lastUpdated = Date.now();

      if (oldStatus !== status) {
        this.emit(DatapointRegistryEvents.STATUS_CHANGED, {
          id,
          path: meta.path,
          oldStatus,
          newStatus: status,
          reason,
        });
      }
    }
  }

  /**
   * 更新数据点值
   * @param {string} id - 数据点 ID
   * @param {*} value - 新值
   */
  updateValue(id, value) {
    const meta = this._datapoints.get(id);
    if (meta) {
      meta.lastValue = value;
      meta.lastUpdated = Date.now();

      // 如果之前是 unknown 状态，更新为 active
      if (meta.status === "unknown") {
        meta.status = "active";
      }

      this.emit(DatapointRegistryEvents.VALUE_UPDATED, {
        id,
        path: meta.path,
        value,
      });
    }
  }

  /**
   * 订阅数据点（组件级订阅追踪）
   * @param {string} datapointId - 数据点 ID
   * @param {string} componentId - 组件 ID
   */
  subscribe(datapointId, componentId) {
    if (!this._subscribers.has(datapointId)) {
      this._subscribers.set(datapointId, new Set());
    }
    this._subscribers.get(datapointId).add(componentId);
  }

  /**
   * 取消订阅数据点
   * @param {string} datapointId - 数据点 ID
   * @param {string} componentId - 组件 ID
   */
  unsubscribe(datapointId, componentId) {
    const subs = this._subscribers.get(datapointId);
    if (subs) {
      subs.delete(componentId);
    }
  }

  /**
   * 获取数据点的订阅者
   * @param {string} datapointId - 数据点 ID
   * @returns {string[]} 组件 ID 列表
   */
  getSubscribers(datapointId) {
    const subs = this._subscribers.get(datapointId);
    return subs ? Array.from(subs) : [];
  }

  /**
   * 获取组件订阅的所有数据点
   * @param {string} componentId - 组件 ID
   * @returns {string[]} 数据点 ID 列表
   */
  getComponentSubscriptions(componentId) {
    const result = [];
    for (const [datapointId, subs] of this._subscribers) {
      if (subs.has(componentId)) {
        result.push(datapointId);
      }
    }
    return result;
  }

  /**
   * 清除组件的所有订阅
   * @param {string} componentId - 组件 ID
   */
  clearComponentSubscriptions(componentId) {
    for (const subs of this._subscribers.values()) {
      subs.delete(componentId);
    }
  }

  /**
   * 搜索数据点
   * @param {string} keyword - 关键词
   * @returns {DatapointMeta[]}
   */
  search(keyword) {
    const lowerKeyword = keyword.toLowerCase();
    return this.getAll().filter(
      (m) =>
        m.path.toLowerCase().includes(lowerKeyword) ||
        m.description?.toLowerCase().includes(lowerKeyword),
    );
  }

  /**
   * 获取所有失效的数据点
   * @returns {DatapointMeta[]}
   */
  getInvalidDatapoints() {
    return this.getAll().filter((m) => m.status === "invalid");
  }

  /**
   * 获取所有未知状态的数据点
   * @returns {DatapointMeta[]}
   */
  getUnknownDatapoints() {
    return this.getAll().filter((m) => m.status === "unknown");
  }

  /**
   * 清空注册表
   */
  clear() {
    this._datapoints.clear();
    this._byProvider.clear();
    this._subscribers.clear();
  }
}

// 导出单例
export const datapointRegistry = new DatapointRegistry();

export default DatapointRegistry;
