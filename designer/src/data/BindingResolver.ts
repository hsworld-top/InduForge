// @ts-nocheck — 依赖仍为 data/*.js；待 types/transforms 等迁 TS 后移除并补全类型
/**
 * BindingResolver - 绑定解析器
 * 解析和执行数据绑定，支持三种模式：设计态、预览态、运行态
 */

import { EventEmitter } from "../editor-core/utils/EventEmitter.ts";
import { applyTransforms } from "./transforms.ts";
import { getBindingKind, isValidValue } from "./types.ts";

/**
 * 绑定解析器类
 */
export class BindingResolver extends EventEmitter {
  /**
   * @param {object} dependencies
   * @param {object} [options]
   */
  constructor(dependencies = {}, options = {}) {
    super();

    /** @type {import('./DataService.ts').DataService | null} */
    this._dataService = dependencies.dataService || null;

    /** @type {import('./MockDataProvider.js').default | import('./MockDataProvider.js').MockDataProvider | null} */
    this._mockProvider = dependencies.mockProvider || null;

    /** @type {import('./VarsStore.ts').VarsStore | null} */
    this._varsStore = dependencies.varsStore || null;

    /** @type {import('./ExpressionEngine.ts').ExpressionEngine | null} */
    this._expressionEngine = dependencies.expressionEngine || null;

    /** @type {import('./DiagnosticsStore.ts').DiagnosticsStore | null} */
    this._diagnosticsStore = dependencies.diagnosticsStore || null;

    /** @type {'edit' | 'preview' | 'runtime'} */
    this._mode = options.mode || "edit";

    /** @type {string | null} */
    this._currentPageId = options.pageId || null;

    /** @type {Map<string, Set<Function>>} */
    this._bindingCallbacks = new Map();

    /** @type {Map<string, Function>} */
    this._unsubscribeFns = new Map();

    if (this._varsStore) {
      this._varsStore.on("change", this._handleVarChange.bind(this));
    }
  }

  // ==================== 模式管理 ====================

  /**
   * 设置数据模式
   * @param {DataMode} mode - 模式
   */
  setMode(mode) {
    if (this._mode !== mode) {
      this._mode = mode;
      this.emit("modeChange", { mode });

      // 重新解析所有绑定
      this._refreshAllBindings();
    }
  }

  /**
   * 获取当前模式
   * @returns {DataMode}
   */
  getMode() {
    return this._mode;
  }

  /**
   * 设置当前页面
   * @param {string | null} pageId - 页面 ID
   */
  setCurrentPage(pageId) {
    this._currentPageId = pageId;
    if (this._varsStore) {
      this._varsStore.setCurrentPage(pageId);
    }
  }

  // ==================== 绑定解析 ====================

  /**
   * 解析绑定并返回当前值
   * @param {Binding} binding - 绑定配置
   * @param {object} [context] - 额外上下文
   * @returns {ResolvedBinding}
   */
  resolve(binding, context = {}) {
    const kind = getBindingKind(binding);

    switch (kind) {
      case "datapoint":
        return this._resolveDatapoint(binding, context);
      case "var":
        return this._resolveVar(binding, context);
      case "expr":
        return this._resolveExpr(binding, context);
      default:
        return {
          value: undefined,
          isLoading: false,
          error: `Unknown binding kind: ${kind}`,
        };
    }
  }

  /**
   * 解析数据点绑定
   * @param {import('../editor-core/types.js').DatapointBinding} binding - 绑定
   * @param {object} context - 上下文
   * @returns {ResolvedBinding}
   * @private
   */
  _resolveDatapoint(binding, context) {
    const { path, transform, fallback, designMock } = binding;

    // 设计态使用 Mock 数据
    if (this._mode === "edit") {
      const mockValue = this._mockProvider
        ? this._mockProvider.getValue(path, binding)
        : designMock;

      const value = isValidValue(mockValue)
        ? applyTransforms(mockValue, transform || [])
        : fallback;

      return {
        value,
        isLoading: false,
        status: "active",
      };
    }

    // 预览态/运行态使用真实数据
    if (!this._dataService) {
      return {
        value: fallback,
        isLoading: false,
        error: "DataService not available",
      };
    }

    // 获取缓存值
    const cachedValue = this._dataService.getValue(path);

    if (cachedValue !== undefined) {
      const value = applyTransforms(cachedValue, transform || []);
      return {
        value,
        isLoading: false,
        status: this._diagnosticsStore
          ? this._diagnosticsStore._getFromCache(path)?.status
          : "active",
      };
    }

    // 值还未加载
    return {
      value: fallback,
      isLoading: true,
    };
  }

  /**
   * 解析变量绑定
   * @param {import('../editor-core/types.js').VarBinding} binding - 绑定
   * @param {object} context - 上下文
   * @returns {ResolvedBinding}
   * @private
   */
  _resolveVar(binding, context) {
    const { scope, name, transform, fallback } = binding;

    if (!this._varsStore) {
      return {
        value: fallback,
        isLoading: false,
        error: "VarsStore not available",
      };
    }

    const rawValue = this._varsStore.get(scope, name, this._currentPageId);

    if (rawValue !== undefined) {
      const value = applyTransforms(rawValue, transform || []);
      return {
        value,
        isLoading: false,
      };
    }

    return {
      value: fallback,
      isLoading: false,
    };
  }

  /**
   * 解析表达式绑定
   * @param {import('../editor-core/types.js').ExprBinding} binding - 绑定
   * @param {object} context - 上下文
   * @returns {ResolvedBinding}
   * @private
   */
  _resolveExpr(binding, context) {
    const { expr, fallback } = binding;

    if (!this._expressionEngine) {
      return {
        value: fallback,
        isLoading: false,
        error: "ExpressionEngine not available",
      };
    }

    // 构建表达式上下文
    const exprContext = this._buildExpressionContext(context);

    // 执行表达式
    const result = this._expressionEngine.evaluate(expr, exprContext);

    if (result.success) {
      return {
        value: result.value,
        isLoading: false,
      };
    }

    return {
      value: fallback,
      isLoading: false,
      error: result.error,
    };
  }

  /**
   * 构建表达式上下文
   * @param {object} additionalContext - 额外上下文
   * @returns {ExpressionContext}
   * @private
   */
  _buildExpressionContext(additionalContext = {}) {
    // 获取数据点值
    const $dp = {};
    if (this._mode === "edit" && this._mockProvider) {
      // 设计态使用代理获取 Mock 值
      const mockProvider = this._mockProvider;
      return new Proxy($dp, {
        get(target, prop) {
          return mockProvider.getValue(String(prop));
        },
      });
    } else if (this._dataService) {
      // 预览态/运行态从缓存获取
      const dataService = this._dataService;
      Object.assign(
        $dp,
        Object.fromEntries(dataService.getValues(Array.from(dataService._valueCache.keys()))),
      );
    }

    // 获取变量上下文
    const varsContext = this._varsStore
      ? this._varsStore.getContext(this._currentPageId)
      : { $vars: {}, $global: {} };

    return {
      $dp,
      $vars: varsContext.$vars,
      $global: varsContext.$global,
      $props: additionalContext.$props || {},
      $event: additionalContext.$event,
      $item: additionalContext.$item,
      $index: additionalContext.$index,
    };
  }

  // ==================== 响应式订阅 ====================

  /**
   * 订阅绑定值变化
   * @param {string} bindingId - 绑定标识（用于管理订阅）
   * @param {Binding} binding - 绑定配置
   * @param {function(ResolvedBinding): void} callback - 回调函数
   * @param {object} [context] - 额外上下文
   * @returns {function(): void} 取消订阅函数
   */
  subscribe(bindingId, binding, callback, context = {}) {
    const kind = getBindingKind(binding);

    // 先立即解析一次
    const resolved = this.resolve(binding, context);
    callback(resolved);

    // 根据绑定类型设置订阅
    switch (kind) {
      case "datapoint":
        return this._subscribeDatapoint(bindingId, binding, callback, context);
      case "var":
        return this._subscribeVar(bindingId, binding, callback, context);
      case "expr":
        return this._subscribeExpr(bindingId, binding, callback, context);
      default:
        return () => {};
    }
  }

  /**
   * 订阅数据点变化
   * @param {string} bindingId - 绑定标识
   * @param {import('../editor-core/types.js').DatapointBinding} binding - 绑定
   * @param {function} callback - 回调
   * @param {object} context - 上下文
   * @returns {function(): void}
   * @private
   */
  _subscribeDatapoint(bindingId, binding, callback, context) {
    const { path, transform, fallback } = binding;

    // 设计态订阅 Mock 提供者
    if (this._mode === "edit" && this._mockProvider) {
      const unsubscribe = this._mockProvider.subscribe(path, (data) => {
        const value = isValidValue(data.value)
          ? applyTransforms(data.value, transform || [])
          : fallback;
        callback({
          value,
          isLoading: false,
          status: "active",
        });
      });

      this._unsubscribeFns.set(bindingId, unsubscribe);
      return () => {
        unsubscribe();
        this._unsubscribeFns.delete(bindingId);
      };
    }

    // 预览态/运行态订阅数据服务
    if (this._dataService) {
      const unsubscribe = this._dataService.subscribe(path, (data) => {
        const value = isValidValue(data.value)
          ? applyTransforms(data.value, transform || [])
          : fallback;
        callback({
          value,
          isLoading: false,
          status: this._diagnosticsStore
            ? this._diagnosticsStore._getFromCache(path)?.status
            : "active",
        });
      });

      this._unsubscribeFns.set(bindingId, unsubscribe);
      return () => {
        unsubscribe();
        this._unsubscribeFns.delete(bindingId);
      };
    }

    return () => {};
  }

  /**
   * 订阅变量变化
   * @param {string} bindingId - 绑定标识
   * @param {import('../editor-core/types.js').VarBinding} binding - 绑定
   * @param {function} callback - 回调
   * @param {object} context - 上下文
   * @returns {function(): void}
   * @private
   */
  _subscribeVar(bindingId, binding, callback, context) {
    const { scope, name, transform, fallback } = binding;

    // 存储回调
    const key = `${scope}:${name}`;
    if (!this._bindingCallbacks.has(key)) {
      this._bindingCallbacks.set(key, new Set());
    }
    this._bindingCallbacks.get(key).add(callback);

    // 保存绑定信息供后续调用
    callback._binding = binding;

    return () => {
      this._bindingCallbacks.get(key)?.delete(callback);
    };
  }

  /**
   * 订阅表达式变化
   * @param {string} bindingId - 绑定标识
   * @param {import('../editor-core/types.js').ExprBinding} binding - 绑定
   * @param {function} callback - 回调
   * @param {object} context - 上下文
   * @returns {function(): void}
   * @private
   */
  _subscribeExpr(bindingId, binding, callback, context) {
    const { expr, fallback } = binding;

    if (!this._expressionEngine) return () => {};

    // 提取依赖
    const dpDeps = this._expressionEngine.extractDependencies(expr);
    const varDeps = this._expressionEngine.extractVarDependencies(expr);

    const unsubscribes = [];

    // 订阅数据点依赖
    for (const path of dpDeps) {
      if (this._mode === "edit" && this._mockProvider) {
        unsubscribes.push(
          this._mockProvider.subscribe(path, () => {
            const resolved = this._resolveExpr(binding, context);
            callback(resolved);
          }),
        );
      } else if (this._dataService) {
        unsubscribes.push(
          this._dataService.subscribe(path, () => {
            const resolved = this._resolveExpr(binding, context);
            callback(resolved);
          }),
        );
      }
    }

    // 订阅变量依赖
    const varChangeHandler = (data) => {
      const isPageVar = data.scope === "page" && varDeps.page.includes(data.name);
      const isGlobalVar = data.scope === "global" && varDeps.global.includes(data.name);

      if (isPageVar || isGlobalVar) {
        const resolved = this._resolveExpr(binding, context);
        callback(resolved);
      }
    };

    if (this._varsStore && (varDeps.page.length > 0 || varDeps.global.length > 0)) {
      this._varsStore.on("change", varChangeHandler);
      unsubscribes.push(() => {
        this._varsStore.off("change", varChangeHandler);
      });
    }

    return () => {
      unsubscribes.forEach((unsub) => unsub());
    };
  }

  /**
   * 处理变量变化
   * @param {object} data - 变化数据
   * @private
   */
  _handleVarChange(data) {
    const { scope, name, value } = data;
    const key = `${scope}:${name}`;

    const callbacks = this._bindingCallbacks.get(key);
    if (!callbacks) return;

    for (const callback of callbacks) {
      const binding = callback._binding;
      if (!binding) continue;

      const { transform, fallback } = binding;
      const transformedValue = isValidValue(value)
        ? applyTransforms(value, transform || [])
        : fallback;

      callback({
        value: transformedValue,
        isLoading: false,
      });
    }
  }

  /**
   * 刷新所有绑定
   * @private
   */
  _refreshAllBindings() {
    // 在模式切换时，需要取消所有现有订阅并重新订阅
    for (const [bindingId, unsubscribe] of this._unsubscribeFns.entries()) {
      unsubscribe();
    }
    this._unsubscribeFns.clear();

    this.emit("bindingsRefreshed");
  }

  // ==================== 批量解析 ====================

  /**
   * 批量解析绑定
   * @param {Record<string, Binding>} bindings - 绑定映射
   * @param {object} [context] - 额外上下文
   * @returns {Record<string, ResolvedBinding>}
   */
  resolveMany(bindings, context = {}) {
    const results = {};
    for (const [key, binding] of Object.entries(bindings)) {
      results[key] = this.resolve(binding, context);
    }
    return results;
  }

  /**
   * 解析组件的所有绑定
   * @param {Record<string, Binding>} bindings - 组件的绑定配置
   * @param {Record<string, *>} props - 组件的静态属性
   * @param {object} [context] - 额外上下文
   * @returns {Record<string, *>} 合并后的属性值
   */
  resolveComponentBindings(bindings, props = {}, context = {}) {
    const result = { ...props };

    for (const [propKey, binding] of Object.entries(bindings)) {
      const resolved = this.resolve(binding, { ...context, $props: props });
      // 只在绑定成功且有值时覆盖
      if (!resolved.error && resolved.value !== undefined) {
        result[propKey] = resolved.value;
      }
    }

    return result;
  }

  // ==================== 工具方法 ====================

  /**
   * 检查绑定是否有效
   * @param {Binding} binding - 绑定
   * @returns {{valid: boolean, error?: string}}
   */
  validate(binding) {
    const kind = getBindingKind(binding);

    switch (kind) {
      case "datapoint":
        if (!binding.path) {
          return { valid: false, error: "数据点路径不能为空" };
        }
        break;

      case "var":
        if (!binding.name) {
          return { valid: false, error: "变量名不能为空" };
        }
        if (!["page", "global"].includes(binding.scope)) {
          return { valid: false, error: "无效的变量作用域" };
        }
        break;

      case "expr":
        if (!binding.expr) {
          return { valid: false, error: "表达式不能为空" };
        }
        if (this._expressionEngine) {
          const result = this._expressionEngine.validate(binding.expr);
          if (!result.valid) {
            return result;
          }
        }
        break;

      default:
        return { valid: false, error: "未知的绑定类型" };
    }

    return { valid: true };
  }

  /**
   * 获取绑定的依赖路径
   * @param {Binding} binding - 绑定
   * @returns {string[]}
   */
  getDependencies(binding) {
    const kind = getBindingKind(binding);

    switch (kind) {
      case "datapoint":
        return [binding.path];

      case "var":
        return [`${binding.scope}:${binding.name}`];

      case "expr":
        if (this._expressionEngine) {
          const dpDeps = this._expressionEngine.extractDependencies(binding.expr);
          const varDeps = this._expressionEngine.extractVarDependencies(binding.expr);
          return [
            ...dpDeps,
            ...varDeps.page.map((n) => `page:${n}`),
            ...varDeps.global.map((n) => `global:${n}`),
          ];
        }
        return [];

      default:
        return [];
    }
  }

  // ==================== 销毁 ====================

  /**
   * 销毁解析器
   */
  destroy() {
    // 取消所有订阅
    for (const unsubscribe of this._unsubscribeFns.values()) {
      unsubscribe();
    }
    this._unsubscribeFns.clear();
    this._bindingCallbacks.clear();

    // 移除变量监听
    if (this._varsStore) {
      this._varsStore.off("change", this._handleVarChange);
    }

    this.removeAllListeners();
  }
}

/**
 * 创建绑定解析器实例
 * @param {object} dependencies - 依赖项
 * @param {BindingResolverOptions} [options] - 配置选项
 * @returns {BindingResolver}
 */
export function createBindingResolver(dependencies, options) {
  return new BindingResolver(dependencies, options);
}

export default BindingResolver;
