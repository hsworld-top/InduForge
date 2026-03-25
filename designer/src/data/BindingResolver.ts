/**
 * BindingResolver - 绑定解析器
 * 解析和执行数据绑定，支持三种模式：设计态、预览态、运行态
 */

import { EventEmitter } from "../editor-core/utils/EventEmitter.ts";
import { applyTransforms } from "./transforms.ts";
import { getBindingKind, isValidValue } from "./types.ts";

type BindingMode = "edit" | "preview" | "runtime";
type BindingCallback = ((value: any) => void) & { _binding?: any };

interface BindingResolverDependencies {
  dataService?: any;
  mockProvider?: any;
  varsStore?: any;
  expressionEngine?: any;
  diagnosticsStore?: any;
}

interface BindingResolverOptions {
  mode?: BindingMode;
  pageId?: string | null;
}

/**
 * 绑定解析器类
 */
export class BindingResolver extends EventEmitter {
  private _dataService: NonNullable<BindingResolverDependencies["dataService"]>;

  private _mockProvider: NonNullable<BindingResolverDependencies["mockProvider"]>;

  private _varsStore: NonNullable<BindingResolverDependencies["varsStore"]>;

  private _expressionEngine: NonNullable<BindingResolverDependencies["expressionEngine"]>;

  private _diagnosticsStore: NonNullable<BindingResolverDependencies["diagnosticsStore"]>;

  private _mode: BindingMode;

  private _currentPageId: string | null;

  private _bindingCallbacks: Map<string, Set<BindingCallback>>;

  private _unsubscribeFns: Map<string, () => void>;

  private _boundVarChangeHandler: (data: any) => void;

  /**
   * 创建绑定解析器
   * @param {BindingResolverDependencies} dependencies - 依赖项
   * @param {BindingResolverOptions} [options] - 配置选项
   */
  constructor(
    dependencies: BindingResolverDependencies = {},
    options: BindingResolverOptions = {},
  ) {
    super();

    this._dataService = dependencies.dataService || null;
    this._mockProvider = dependencies.mockProvider || null;
    this._varsStore = dependencies.varsStore || null;
    this._expressionEngine = dependencies.expressionEngine || null;
    this._diagnosticsStore = dependencies.diagnosticsStore || null;
    this._mode = options.mode || "edit";
    this._currentPageId = options.pageId || null;
    this._bindingCallbacks = new Map();
    this._unsubscribeFns = new Map();
    this._boundVarChangeHandler = this._handleVarChange.bind(this);

    if (this._varsStore) {
      this._varsStore.on("change", this._boundVarChangeHandler);
    }
  }

  /**
   * 设置数据模式
   * @param {BindingMode} mode - 模式
   */
  setMode(mode: BindingMode): void {
    if (this._mode !== mode) {
      this._mode = mode;
      this.emit("modeChange", { mode });
      this._refreshAllBindings();
    }
  }

  /**
   * 获取当前模式
   * @returns {BindingMode}
   */
  getMode(): BindingMode {
    return this._mode;
  }

  /**
   * 设置当前页面
   * @param {string | null} pageId - 页面 ID
   */
  setCurrentPage(pageId: string | null): void {
    this._currentPageId = pageId;
    this._varsStore?.setCurrentPage(pageId);
  }

  /**
   * 解析绑定并返回当前值
   * @param {any} binding - 绑定配置
   * @param {Record<string, unknown>} [context] - 额外上下文
   */
  resolve(binding: any, context: Record<string, unknown> = {}): any {
    const kind = getBindingKind(binding);
    switch (kind) {
      case "datapoint":
        return this._resolveDatapoint(binding, context);
      case "var":
        return this._resolveVar(binding, context);
      case "expr":
        return this._resolveExpr(binding, context);
      default:
        return { value: undefined, isLoading: false, error: `Unknown binding kind: ${kind}` };
    }
  }

  private _resolveDatapoint(binding: any, _context: Record<string, unknown>): any {
    const { path, transform, fallback, designMock } = binding;
    if (this._mode === "edit") {
      const mockValue = this._mockProvider
        ? this._mockProvider.getValue(path, binding)
        : designMock;
      const value = isValidValue(mockValue)
        ? applyTransforms(mockValue, transform || [])
        : fallback;
      return { value, isLoading: false, status: "active" };
    }

    if (!this._dataService) {
      return { value: fallback, isLoading: false, error: "DataService not available" };
    }

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

    return { value: fallback, isLoading: true };
  }

  private _resolveVar(binding: any, _context: Record<string, unknown>): any {
    const { scope, name, transform, fallback } = binding;
    if (!this._varsStore) {
      return { value: fallback, isLoading: false, error: "VarsStore not available" };
    }

    const rawValue = this._varsStore.get(scope, name, this._currentPageId || undefined);
    if (rawValue !== undefined) {
      return { value: applyTransforms(rawValue, transform || []), isLoading: false };
    }

    return { value: fallback, isLoading: false };
  }

  private _resolveExpr(binding: any, context: Record<string, unknown>): any {
    const { expr, fallback } = binding;
    if (!this._expressionEngine) {
      return { value: fallback, isLoading: false, error: "ExpressionEngine not available" };
    }

    const exprContext = this._buildExpressionContext(context);
    const result = this._expressionEngine.evaluate(expr, exprContext);
    if (result.success) {
      return { value: result.value, isLoading: false };
    }
    return { value: fallback, isLoading: false, error: result.error };
  }

  private _buildExpressionContext(additionalContext: Record<string, unknown> = {}) {
    const $dp: Record<string, unknown> = {};
    if (this._mode === "edit" && this._mockProvider) {
      const mockProvider = this._mockProvider;
      return new Proxy($dp, {
        get(_target, prop) {
          return mockProvider.getValue(String(prop));
        },
      });
    }

    if (this._dataService) {
      const dataService = this._dataService;
      Object.assign(
        $dp,
        Object.fromEntries(
          dataService.getValues(
            dataService._valueCache ? Array.from(dataService._valueCache.keys()) : [],
          ),
        ),
      );
    }

    const varsContext = this._varsStore
      ? this._varsStore.getContext(this._currentPageId)
      : { $vars: {}, $global: {} };

    return {
      $dp,
      $vars: varsContext.$vars,
      $global: varsContext.$global,
      $props: (additionalContext.$props as Record<string, unknown>) || {},
      $event: additionalContext.$event,
      $item: additionalContext.$item,
      $index: additionalContext.$index,
    };
  }

  subscribe(
    bindingId: string,
    binding: any,
    callback: BindingCallback,
    context: Record<string, unknown> = {},
  ): () => void {
    const kind = getBindingKind(binding);
    const resolved = this.resolve(binding, context);
    callback(resolved);

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

  private _subscribeDatapoint(
    bindingId: string,
    binding: any,
    callback: BindingCallback,
    _context: Record<string, unknown>,
  ): () => void {
    const { path, transform, fallback } = binding;
    if (this._mode === "edit" && this._mockProvider) {
      const unsubscribe = this._mockProvider.subscribe(path, (data: any) => {
        const value = isValidValue(data.value)
          ? applyTransforms(data.value, transform || [])
          : fallback;
        callback({ value, isLoading: false, status: "active" });
      });
      this._unsubscribeFns.set(bindingId, unsubscribe);
      return () => {
        unsubscribe();
        this._unsubscribeFns.delete(bindingId);
      };
    }

    if (this._dataService) {
      const unsubscribe = this._dataService.subscribe(path, (data: any) => {
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

  private _subscribeVar(
    bindingId: string,
    binding: any,
    callback: BindingCallback,
    _context: Record<string, unknown>,
  ): () => void {
    const { scope, name } = binding;
    const key = `${scope}:${name}`;
    if (!this._bindingCallbacks.has(key)) {
      this._bindingCallbacks.set(key, new Set());
    }
    this._bindingCallbacks.get(key)?.add(callback);
    callback._binding = binding;

    return () => {
      this._bindingCallbacks.get(key)?.delete(callback);
    };
  }

  private _subscribeExpr(
    bindingId: string,
    binding: any,
    callback: BindingCallback,
    context: Record<string, unknown>,
  ): () => void {
    const { expr } = binding;
    if (!this._expressionEngine) return () => {};

    const dpDeps = this._expressionEngine.extractDependencies(expr);
    const varDeps = this._expressionEngine.extractVarDependencies(expr);
    const unsubscribes: Array<() => void> = [];

    for (const path of dpDeps) {
      if (this._mode === "edit" && this._mockProvider) {
        unsubscribes.push(
          this._mockProvider.subscribe(path, () => callback(this._resolveExpr(binding, context))),
        );
      } else if (this._dataService) {
        unsubscribes.push(
          this._dataService.subscribe(path, () => callback(this._resolveExpr(binding, context))),
        );
      }
    }

    const varChangeHandler = (data: any) => {
      const isPageVar = data.scope === "page" && varDeps.page.includes(data.name);
      const isGlobalVar = data.scope === "global" && varDeps.global.includes(data.name);
      if (isPageVar || isGlobalVar) {
        callback(this._resolveExpr(binding, context));
      }
    };

    if (this._varsStore && (varDeps.page.length > 0 || varDeps.global.length > 0)) {
      this._varsStore.on("change", varChangeHandler);
      unsubscribes.push(() => {
        this._varsStore?.off("change", varChangeHandler);
      });
    }

    return () => {
      unsubscribes.forEach((unsub) => unsub());
    };
  }

  private _handleVarChange(data: any): void {
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
      callback({ value: transformedValue, isLoading: false });
    }
  }

  private _refreshAllBindings(): void {
    for (const unsubscribe of this._unsubscribeFns.values()) {
      unsubscribe();
    }
    this._unsubscribeFns.clear();
    this.emit("bindingsRefreshed");
  }

  resolveMany(bindings: Record<string, any>, context: Record<string, unknown> = {}) {
    const results: Record<string, unknown> = {};
    for (const [key, binding] of Object.entries(bindings)) {
      results[key] = this.resolve(binding, context);
    }
    return results;
  }

  resolveComponentBindings(
    bindings: Record<string, any>,
    props: Record<string, unknown> = {},
    context: Record<string, unknown> = {},
  ) {
    const result = { ...props };
    for (const [propKey, binding] of Object.entries(bindings)) {
      const resolved = this.resolve(binding, { ...context, $props: props });
      if (!resolved.error && resolved.value !== undefined) {
        result[propKey] = resolved.value;
      }
    }
    return result;
  }

  validate(binding: any): { valid: boolean; error?: string } {
    const kind = getBindingKind(binding);
    switch (kind) {
      case "datapoint":
        if (!binding.path) return { valid: false, error: "数据点路径不能为空" };
        break;
      case "var":
        if (!binding.name) return { valid: false, error: "变量名不能为空" };
        if (!["page", "global"].includes(binding.scope))
          return { valid: false, error: "无效的变量作用域" };
        break;
      case "expr":
        if (!binding.expr) return { valid: false, error: "表达式不能为空" };
        if (this._expressionEngine) {
          const result = this._expressionEngine.validate(binding.expr);
          if (!result.valid) return result;
        }
        break;
      default:
        return { valid: false, error: "未知的绑定类型" };
    }
    return { valid: true };
  }

  getDependencies(binding: any): string[] {
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
            ...varDeps.page.map((n: any) => `page:${n}`),
            ...varDeps.global.map((n: any) => `global:${n}`),
          ];
        }
        return [];
      default:
        return [];
    }
  }

  destroy(): void {
    for (const unsubscribe of this._unsubscribeFns.values()) unsubscribe();
    this._unsubscribeFns.clear();
    this._bindingCallbacks.clear();
    this._varsStore?.off("change", this._boundVarChangeHandler);
    this.removeAllListeners();
  }
}

/**
 * 创建绑定解析器实例
 * @param {BindingResolverDependencies} dependencies - 依赖项
 * @param {BindingResolverOptions} [options] - 配置选项
 * @returns {BindingResolver}
 */
export function createBindingResolver(
  dependencies: BindingResolverDependencies,
  options?: BindingResolverOptions,
): BindingResolver {
  return new BindingResolver(dependencies, options);
}

export default BindingResolver;
