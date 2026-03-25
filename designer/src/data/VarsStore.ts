/**
 * VarsStore - 变量状态管理
 * 管理页面变量和全局变量
 */

import type { VarDefinition, VarsDefinitions } from "./types.ts";
import { EventEmitter } from "../editor-core/utils/EventEmitter.ts";

type VarsScope = "global" | "page";

interface VarsChangePayload {
  scope: VarsScope;
  name: string;
  value: unknown;
  oldValue: unknown;
  pageId?: string;
}

interface VarsImportState {
  global?: Record<string, unknown>;
  pages?: Record<string, Record<string, unknown>>;
}

/**
 * 变量存储类
 */
export class VarsStore extends EventEmitter {
  private _definitions: VarsDefinitions;

  private _globalVars: Map<string, unknown>;

  private _pageVars: Map<string, Map<string, unknown>>;

  private _currentPageId: string | null;

  /**
   * 创建变量存储
   * @param {VarsDefinitions} [definitions] - 变量定义
   */
  constructor(definitions: VarsDefinitions = { global: {}, pages: {} }) {
    super();

    this._definitions = definitions;
    this._globalVars = new Map();
    this._pageVars = new Map();
    this._currentPageId = null;

    this._initGlobalVars();
  }

  /**
   * 初始化全局变量
   * @private
   */
  private _initGlobalVars(): void {
    const globalDefs = this._definitions.global || {};

    for (const [name, def] of Object.entries(globalDefs)) {
      let value: unknown = def.default;

      if (def.persistent && typeof localStorage !== "undefined") {
        const storageKey = def.storageKey || `var_${name}`;
        try {
          const stored = localStorage.getItem(storageKey);
          if (stored !== null) {
            value = JSON.parse(stored);
          }
        } catch (error) {
          console.warn(`Failed to restore var "${name}" from localStorage:`, error);
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
  initPageVars(pageId: string, definitions?: Record<string, VarDefinition>): void {
    const defs = definitions || this._definitions.pages?.[pageId] || {};
    const vars = new Map<string, unknown>();

    for (const [name, def] of Object.entries(defs)) {
      vars.set(name, def.default);
    }

    this._pageVars.set(pageId, vars);

    if (definitions) {
      this._definitions.pages[pageId] = definitions;
    }

    this.emit("pageInit", { pageId });
  }

  /**
   * 清理页面变量
   * @param {string} pageId - 页面 ID
   */
  clearPageVars(pageId: string): void {
    this._pageVars.delete(pageId);
    this.emit("pageClear", { pageId });
  }

  /**
   * 设置当前页面
   * @param {string | null} pageId - 页面 ID
   */
  setCurrentPage(pageId: string | null): void {
    this._currentPageId = pageId;
  }

  /**
   * 获取变量值
   * @param {'global' | 'page'} scope - 作用域
   * @param {string} name - 变量名
   * @param {string} [pageId] - 页面 ID（scope 为 page 时必填）
   * @returns {*}
   */
  get(scope: VarsScope, name: string, pageId?: string): unknown {
    if (scope === "global") {
      return this._globalVars.get(name);
    }

    const targetPageId = pageId || this._currentPageId;
    if (!targetPageId) {
      console.warn(`VarsStore.get: No pageId specified for page var "${name}"`);
      return undefined;
    }
    return this._pageVars.get(targetPageId)?.get(name);
  }

  /**
   * 获取全局变量值
   * @param {string} name - 变量名
   * @returns {*}
   */
  getGlobal(name: string): unknown {
    return this._globalVars.get(name);
  }

  /**
   * 获取页面变量值
   * @param {string} name - 变量名
   * @param {string} [pageId] - 页面 ID
   * @returns {*}
   */
  getPage(name: string, pageId?: string): unknown {
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
  has(scope: VarsScope, name: string, pageId?: string): boolean {
    if (scope === "global") {
      return this._globalVars.has(name);
    }

    const targetPageId = pageId || this._currentPageId;
    if (!targetPageId) return false;
    return this._pageVars.get(targetPageId)?.has(name) ?? false;
  }

  /**
   * 设置变量值
   * @param {'global' | 'page'} scope - 作用域
   * @param {string} name - 变量名
   * @param {*} value - 值
   * @param {string} [pageId] - 页面 ID（scope 为 page 时可选）
   */
  set(scope: VarsScope, name: string, value: unknown, pageId?: string): void {
    const oldValue = this.get(scope, name, pageId);

    if (scope === "global") {
      this._globalVars.set(name, value);

      const def = this._definitions.global?.[name];
      if (def?.persistent && typeof localStorage !== "undefined") {
        const storageKey = def.storageKey || `var_${name}`;
        try {
          localStorage.setItem(storageKey, JSON.stringify(value));
        } catch (error) {
          console.warn(`Failed to persist var "${name}":`, error);
        }
      }
    } else {
      const targetPageId = pageId || this._currentPageId;
      if (!targetPageId) {
        console.warn(`VarsStore.set: No pageId specified for page var "${name}"`);
        return;
      }

      let pageVars = this._pageVars.get(targetPageId);
      if (!pageVars) {
        pageVars = new Map();
        this._pageVars.set(targetPageId, pageVars);
      }
      pageVars.set(name, value);
    }

    const changePayload: VarsChangePayload = {
      scope,
      name,
      value,
      oldValue,
    };
    if (scope === "page") {
      const resolvedPageId = pageId || this._currentPageId;
      if (resolvedPageId !== undefined && resolvedPageId !== null) {
        changePayload.pageId = resolvedPageId;
      }
    }
    this.emit("change", changePayload);
  }

  /**
   * 设置全局变量
   * @param {string} name - 变量名
   * @param {*} value - 值
   */
  setGlobal(name: string, value: unknown): void {
    this.set("global", name, value);
  }

  /**
   * 设置页面变量
   * @param {string} name - 变量名
   * @param {*} value - 值
   * @param {string} [pageId] - 页面 ID
   */
  setPage(name: string, value: unknown, pageId?: string): void {
    this.set("page", name, value, pageId);
  }

  /**
   * 批量设置变量
   * @param {Array<{scope: 'global' | 'page', name: string, value: *}>} vars - 变量列表
   * @param {string} [pageId] - 页面 ID
   */
  setMany(vars: Array<{ scope: VarsScope; name: string; value: unknown }>, pageId?: string): void {
    for (const { scope, name, value } of vars) {
      this.set(scope, name, value, pageId);
    }
  }

  /**
   * 重置变量为默认值
   * @param {'global' | 'page'} scope - 作用域
   * @param {string} name - 变量名
   * @param {string} [pageId] - 页面 ID
   */
  reset(scope: VarsScope, name: string, pageId?: string): void {
    let def: VarDefinition | undefined;
    if (scope === "global") {
      def = this._definitions.global?.[name];
    } else {
      const targetPageId = pageId || this._currentPageId;
      def = targetPageId ? this._definitions.pages?.[targetPageId]?.[name] : undefined;
    }

    if (def) {
      this.set(scope, name, def.default, pageId);
    }
  }

  /**
   * 重置所有页面变量
   * @param {string} [pageId] - 页面 ID
   */
  resetPageVars(pageId?: string): void {
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
  resetGlobalVars(): void {
    const defs = this._definitions.global || {};
    for (const [name, def] of Object.entries(defs)) {
      this.set("global", name, def.default);
    }
  }

  /**
   * 获取变量上下文（用于表达式求值）
   * @param {string} [pageId] - 页面 ID
   * @returns {VarsContext}
   */
  getContext(pageId?: string) {
    const targetPageId = pageId || this._currentPageId;
    const pageVars = targetPageId ? this._pageVars.get(targetPageId) : null;

    return {
      $vars: pageVars ? Object.fromEntries(pageVars) : {},
      $global: Object.fromEntries(this._globalVars),
    };
  }

  /**
   * 获取所有全局变量
   * @returns {Record<string, unknown>}
   */
  getAllGlobal(): Record<string, unknown> {
    return Object.fromEntries(this._globalVars);
  }

  /**
   * 获取所有页面变量
   * @param {string} [pageId] - 页面 ID
   * @returns {Record<string, unknown>}
   */
  getAllPage(pageId?: string): Record<string, unknown> {
    const targetPageId = pageId || this._currentPageId;
    if (!targetPageId) return {};
    const pageVars = this._pageVars.get(targetPageId);
    return pageVars ? Object.fromEntries(pageVars) : {};
  }

  /**
   * 获取变量定义
   * @param {'global' | 'page'} scope - 作用域
   * @param {string} name - 变量名
   * @param {string} [pageId] - 页面 ID
   * @returns {VarDefinition | undefined}
   */
  getDefinition(scope: VarsScope, name: string, pageId?: string): VarDefinition | undefined {
    if (scope === "global") {
      return this._definitions.global?.[name];
    }
    const targetPageId = pageId || this._currentPageId;
    if (!targetPageId) return undefined;
    return this._definitions.pages?.[targetPageId]?.[name];
  }

  /**
   * 获取所有全局变量定义
   * @returns {Record<string, VarDefinition>}
   */
  getGlobalDefinitions(): Record<string, VarDefinition> {
    return this._definitions.global || {};
  }

  /**
   * 获取页面变量定义
   * @param {string} [pageId] - 页面 ID
   * @returns {Record<string, VarDefinition>}
   */
  getPageDefinitions(pageId?: string): Record<string, VarDefinition> {
    const targetPageId = pageId || this._currentPageId;
    if (!targetPageId) return {};
    return this._definitions.pages?.[targetPageId] || {};
  }

  /**
   * 更新变量定义
   * @param {VarsDefinitions} definitions - 新的变量定义
   */
  updateDefinitions(definitions: VarsDefinitions): void {
    this._definitions = definitions;
    this._globalVars.clear();
    this._initGlobalVars();

    for (const pageId of this._pageVars.keys()) {
      const pageDefs = definitions.pages?.[pageId];
      if (pageDefs) {
        this.initPageVars(pageId, pageDefs);
      }
    }

    this.emit("definitionsUpdated");
  }

  /**
   * 导出当前状态
   * @returns {{global: Record<string, unknown>, pages: Record<string, Record<string, unknown>>}}
   */
  export() {
    const pages: Record<string, Record<string, unknown>> = {};
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
   * @param {{global?: Record<string, unknown>, pages?: Record<string, Record<string, unknown>>}} state - 状态
   */
  import(state: VarsImportState): void {
    if (state.global) {
      for (const [name, value] of Object.entries(state.global)) {
        this._globalVars.set(name, value);
      }
    }

    if (state.pages) {
      for (const [pageId, vars] of Object.entries(state.pages)) {
        const pageVars = new Map<string, unknown>();
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
  clear(): void {
    this._globalVars.clear();
    this._pageVars.clear();
    this._initGlobalVars();
    this.emit("cleared");
  }
}

export default VarsStore;
