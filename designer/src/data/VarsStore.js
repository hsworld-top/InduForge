/**
 * VarsStore - 变量状态管理
 * 管理页面变量和全局变量
 */

import { EventEmitter } from "../editor-core/utils/EventEmitter.ts";

/**
 * @typedef {import('./types.js').VarDefinition} VarDefinition
 * @typedef {import('./types.js').VarsDefinitions} VarsDefinitions
 * @typedef {import('./types.js').VarsContext} VarsContext
 */

/**
 * 变量存储类
 */
export class VarsStore extends EventEmitter {
  /**
   * 创建变量存储
   * @param {VarsDefinitions} [definitions] - 变量定义
   */
  constructor(definitions = { global: {}, pages: {} }) {
    super();

    /** @type {VarsDefinitions} */
    this._definitions = definitions;

    /** @type {Map<string, *>} 全局变量值 */
    this._globalVars = new Map();

    /** @type {Map<string, Map<string, *>>} 页面变量值 pageId -> (name -> value) */
    this._pageVars = new Map();

    /** @type {string | null} 当前活动页面 ID */
    this._currentPageId = null;

    // 初始化全局变量
    this._initGlobalVars();
  }

  // ==================== 初始化 ====================

  /**
   * 初始化全局变量
   * @private
   */
  _initGlobalVars() {
    const globalDefs = this._definitions.global || {};

    for (const [name, def] of Object.entries(globalDefs)) {
      let value = def.default;

      // 从 localStorage 恢复持久化变量
      if (def.persistent && typeof localStorage !== "undefined") {
        const storageKey = def.storageKey || `var_${name}`;
        try {
          const stored = localStorage.getItem(storageKey);
          if (stored !== null) {
            value = JSON.parse(stored);
          }
        } catch (e) {
          console.warn(`Failed to restore var "${name}" from localStorage:`, e);
        }
      }

      this._globalVars.set(name, value);
    }
  }

  /**
   * 初始化页面变量
   * @param {string} pageId - 页面 ID
   * @param {Record<string, VarDefinition>} [definitions] - 页面变量定义
   */
  initPageVars(pageId, definitions) {
    const defs = definitions || this._definitions.pages?.[pageId] || {};
    const vars = new Map();

    for (const [name, def] of Object.entries(defs)) {
      vars.set(name, def.default);
    }

    this._pageVars.set(pageId, vars);

    // 保存页面变量定义
    if (definitions) {
      if (!this._definitions.pages) {
        this._definitions.pages = {};
      }
      this._definitions.pages[pageId] = definitions;
    }

    this.emit("pageInit", { pageId });
  }

  /**
   * 清理页面变量
   * @param {string} pageId - 页面 ID
   */
  clearPageVars(pageId) {
    this._pageVars.delete(pageId);
    this.emit("pageClear", { pageId });
  }

  /**
   * 设置当前页面
   * @param {string | null} pageId - 页面 ID
   */
  setCurrentPage(pageId) {
    this._currentPageId = pageId;
  }

  // ==================== 变量读取 ====================

  /**
   * 获取变量值
   * @param {'global' | 'page'} scope - 作用域
   * @param {string} name - 变量名
   * @param {string} [pageId] - 页面 ID（scope 为 page 时必填）
   * @returns {*}
   */
  get(scope, name, pageId) {
    if (scope === "global") {
      return this._globalVars.get(name);
    } else {
      const targetPageId = pageId || this._currentPageId;
      if (!targetPageId) {
        console.warn(
          `VarsStore.get: No pageId specified for page var "${name}"`,
        );
        return undefined;
      }
      return this._pageVars.get(targetPageId)?.get(name);
    }
  }

  /**
   * 获取全局变量值
   * @param {string} name - 变量名
   * @returns {*}
   */
  getGlobal(name) {
    return this._globalVars.get(name);
  }

  /**
   * 获取页面变量值
   * @param {string} name - 变量名
   * @param {string} [pageId] - 页面 ID
   * @returns {*}
   */
  getPage(name, pageId) {
    const targetPageId = pageId || this._currentPageId;
    if (!targetPageId) return undefined;
    return this._pageVars.get(targetPageId)?.get(name);
  }

  /**
   * 检查变量是否存在
   * @param {'global' | 'page'} scope - 作用域
   * @param {string} name - 变量名
   * @param {string} [pageId] - 页面 ID
   * @returns {boolean}
   */
  has(scope, name, pageId) {
    if (scope === "global") {
      return this._globalVars.has(name);
    } else {
      const targetPageId = pageId || this._currentPageId;
      if (!targetPageId) return false;
      return this._pageVars.get(targetPageId)?.has(name) ?? false;
    }
  }

  // ==================== 变量写入 ====================

  /**
   * 设置变量值
   * @param {'global' | 'page'} scope - 作用域
   * @param {string} name - 变量名
   * @param {*} value - 值
   * @param {string} [pageId] - 页面 ID（scope 为 page 时可选）
   */
  set(scope, name, value, pageId) {
    const oldValue = this.get(scope, name, pageId);

    if (scope === "global") {
      this._globalVars.set(name, value);

      // 持久化
      const def = this._definitions.global?.[name];
      if (def?.persistent && typeof localStorage !== "undefined") {
        const storageKey = def.storageKey || `var_${name}`;
        try {
          localStorage.setItem(storageKey, JSON.stringify(value));
        } catch (e) {
          console.warn(`Failed to persist var "${name}":`, e);
        }
      }
    } else {
      const targetPageId = pageId || this._currentPageId;
      if (!targetPageId) {
        console.warn(
          `VarsStore.set: No pageId specified for page var "${name}"`,
        );
        return;
      }

      let pageVars = this._pageVars.get(targetPageId);
      if (!pageVars) {
        pageVars = new Map();
        this._pageVars.set(targetPageId, pageVars);
      }
      pageVars.set(name, value);
    }

    // 触发变更事件
    this.emit("change", {
      scope,
      name,
      value,
      oldValue,
      pageId: scope === "page" ? pageId || this._currentPageId : undefined,
    });
  }

  /**
   * 设置全局变量
   * @param {string} name - 变量名
   * @param {*} value - 值
   */
  setGlobal(name, value) {
    this.set("global", name, value);
  }

  /**
   * 设置页面变量
   * @param {string} name - 变量名
   * @param {*} value - 值
   * @param {string} [pageId] - 页面 ID
   */
  setPage(name, value, pageId) {
    this.set("page", name, value, pageId);
  }

  /**
   * 批量设置变量
   * @param {Array<{scope: 'global' | 'page', name: string, value: *}>} vars - 变量列表
   * @param {string} [pageId] - 页面 ID
   */
  setMany(vars, pageId) {
    for (const { scope, name, value } of vars) {
      this.set(scope, name, value, pageId);
    }
  }

  // ==================== 变量重置 ====================

  /**
   * 重置变量为默认值
   * @param {'global' | 'page'} scope - 作用域
   * @param {string} name - 变量名
   * @param {string} [pageId] - 页面 ID
   */
  reset(scope, name, pageId) {
    let def;
    if (scope === "global") {
      def = this._definitions.global?.[name];
    } else {
      const targetPageId = pageId || this._currentPageId;
      def = this._definitions.pages?.[targetPageId]?.[name];
    }

    if (def) {
      this.set(scope, name, def.default, pageId);
    }
  }

  /**
   * 重置所有页面变量
   * @param {string} [pageId] - 页面 ID
   */
  resetPageVars(pageId) {
    const targetPageId = pageId || this._currentPageId;
    if (!targetPageId) return;

    const defs = this._definitions.pages?.[targetPageId] || {};
    for (const [name, def] of Object.entries(defs)) {
      this.set("page", name, def.default, targetPageId);
    }
  }

  /**
   * 重置所有全局变量
   */
  resetGlobalVars() {
    const defs = this._definitions.global || {};
    for (const [name, def] of Object.entries(defs)) {
      this.set("global", name, def.default);
    }
  }

  // ==================== 上下文获取 ====================

  /**
   * 获取变量上下文（用于表达式求值）
   * @param {string} [pageId] - 页面 ID
   * @returns {VarsContext}
   */
  getContext(pageId) {
    const targetPageId = pageId || this._currentPageId;
    const pageVars = targetPageId ? this._pageVars.get(targetPageId) : null;

    return {
      $vars: pageVars ? Object.fromEntries(pageVars) : {},
      $global: Object.fromEntries(this._globalVars),
    };
  }

  /**
   * 获取所有全局变量
   * @returns {Record<string, *>}
   */
  getAllGlobal() {
    return Object.fromEntries(this._globalVars);
  }

  /**
   * 获取所有页面变量
   * @param {string} [pageId] - 页面 ID
   * @returns {Record<string, *>}
   */
  getAllPage(pageId) {
    const targetPageId = pageId || this._currentPageId;
    if (!targetPageId) return {};
    const pageVars = this._pageVars.get(targetPageId);
    return pageVars ? Object.fromEntries(pageVars) : {};
  }

  // ==================== 定义管理 ====================

  /**
   * 获取变量定义
   * @param {'global' | 'page'} scope - 作用域
   * @param {string} name - 变量名
   * @param {string} [pageId] - 页面 ID
   * @returns {VarDefinition | undefined}
   */
  getDefinition(scope, name, pageId) {
    if (scope === "global") {
      return this._definitions.global?.[name];
    } else {
      const targetPageId = pageId || this._currentPageId;
      return this._definitions.pages?.[targetPageId]?.[name];
    }
  }

  /**
   * 获取所有全局变量定义
   * @returns {Record<string, VarDefinition>}
   */
  getGlobalDefinitions() {
    return this._definitions.global || {};
  }

  /**
   * 获取页面变量定义
   * @param {string} [pageId] - 页面 ID
   * @returns {Record<string, VarDefinition>}
   */
  getPageDefinitions(pageId) {
    const targetPageId = pageId || this._currentPageId;
    if (!targetPageId) return {};
    return this._definitions.pages?.[targetPageId] || {};
  }

  /**
   * 更新变量定义
   * @param {VarsDefinitions} definitions - 新的变量定义
   */
  updateDefinitions(definitions) {
    this._definitions = definitions;

    // 重新初始化全局变量
    this._globalVars.clear();
    this._initGlobalVars();

    // 重新初始化已加载的页面变量
    for (const pageId of this._pageVars.keys()) {
      const pageDefs = definitions.pages?.[pageId];
      if (pageDefs) {
        this.initPageVars(pageId, pageDefs);
      }
    }

    this.emit("definitionsUpdated");
  }

  // ==================== 序列化 ====================

  /**
   * 导出当前状态
   * @returns {{global: Record<string, *>, pages: Record<string, Record<string, *>>}}
   */
  export() {
    const pages = {};
    for (const [pageId, vars] of this._pageVars.entries()) {
      pages[pageId] = Object.fromEntries(vars);
    }

    return {
      global: Object.fromEntries(this._globalVars),
      pages,
    };
  }

  /**
   * 导入状态
   * @param {{global?: Record<string, *>, pages?: Record<string, Record<string, *>>}} state - 状态
   */
  import(state) {
    if (state.global) {
      for (const [name, value] of Object.entries(state.global)) {
        this._globalVars.set(name, value);
      }
    }

    if (state.pages) {
      for (const [pageId, vars] of Object.entries(state.pages)) {
        const pageVars = new Map();
        for (const [name, value] of Object.entries(vars)) {
          pageVars.set(name, value);
        }
        this._pageVars.set(pageId, pageVars);
      }
    }

    this.emit("imported");
  }

  /**
   * 清空所有变量
   */
  clear() {
    this._globalVars.clear();
    this._pageVars.clear();
    this._initGlobalVars();
    this.emit("cleared");
  }
}

export default VarsStore;
