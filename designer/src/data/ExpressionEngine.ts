/**
 * ExpressionEngine - 表达式引擎
 * 解析和执行 {{ }} 语法的表达式
 */

import type { ExpressionContext, ExpressionResult } from "./types.ts";
import dayjs from "dayjs";

type AnyFn = (...args: any[]) => any;
type CompiledExpressionFn = (
  context: ExpressionContext,
  functions: Record<string, AnyFn>,
) => unknown;
interface ExpressionEngineOptions {
  strict?: boolean;
}

const FORMAT_TOKEN_RE = /%(\.\d+)?[sdf]/g;
const EXPRESSION_BLOCK_RE = /\{\{.+?\}\}/;
const TEMPLATE_EXPR_RE = /\{\{(.+?)\}\}/g;
const DATA_POINT_DEP_RE = /\$dp\[['"]([^'"]+)['"]\]|\$dp\.(\w+)/g;
const PAGE_VAR_DEP_RE = /\$vars\.(\w+)|\$vars\[['"]([^'"]+)['"]\]/g;
const GLOBAL_VAR_DEP_RE = /\$global\.(\w+)|\$global\[['"]([^'"]+)['"]\]/g;

function createDynamicFunction(
  ...args: [string, string, string]
): (...params: unknown[]) => unknown {
  // 这里是受控动态执行入口，用于表达式编译，不扩散到其他层
  // eslint-disable-next-line no-new-func
  return new Function(...args) as (...params: unknown[]) => unknown;
}

// ==================== 内置函数 ====================

/**
 * 内置函数注册表
 * @type {Record<string, (...args: any[]) => any>}
 */
const builtinFunctions: Record<string, AnyFn> = {
  // 数值函数
  abs: Math.abs,
  ceil: Math.ceil,
  floor: Math.floor,
  round: (value, decimals = 0) => {
    const factor = 10 ** decimals;
    return Math.round(value * factor) / factor;
  },
  min: Math.min,
  max: Math.max,
  clamp: (value, min, max) => Math.min(Math.max(value, min), max),
  pow: Math.pow,
  sqrt: Math.sqrt,

  // 格式化函数
  format: (value, template) => {
    if (!template) return String(value);
    return template.replace(FORMAT_TOKEN_RE, (match: string) => {
      if (match === "%s") return String(value);
      if (match === "%d") return String(Math.floor(Number(value)));
      if (match.startsWith("%.") && match.endsWith("f")) {
        const decimals = Number.parseInt(match.slice(2, -1), 10);
        return Number(value).toFixed(decimals);
      }
      return match;
    });
  },

  toFixed: (value, digits = 2) => Number(value).toFixed(digits),

  // 日期函数
  dateFormat: (value, formatStr = "YYYY-MM-DD HH:mm:ss") => {
    if (!value) return "";
    const parsed = dayjs(value);
    return parsed.isValid() ? parsed.format(formatStr) : String(value);
  },

  now: () => Date.now(),

  // 字符串函数
  concat: (...args) => args.join(""),
  substring: (str, start, end) => String(str).substring(start, end),
  replace: (str, search, replacement) => String(str).replace(search, replacement),
  toLowerCase: (str) => String(str).toLowerCase(),
  toUpperCase: (str) => String(str).toUpperCase(),
  trim: (str) => String(str).trim(),
  length: (value) => {
    if (Array.isArray(value)) return value.length;
    if (typeof value === "string") return value.length;
    return 0;
  },

  // 空值处理
  ifNull: (value: any, replacement: any) => value ?? replacement,

  ifEmpty: (value: any, replacement: any) =>
    value === null || value === undefined || value === "" ? replacement : value,

  // 类型转换
  toNumber: (value: any) => Number(value),
  toString: (value: any) => String(value),
  toBoolean: (value: any) => Boolean(value),

  // 数组函数
  first: (arr: any) => (Array.isArray(arr) ? arr[0] : arr),
  last: (arr: any) => (Array.isArray(arr) ? arr.at(-1) : arr),
  at: (arr: any, index: any) => (Array.isArray(arr) ? arr[index] : undefined),
  includes: (arr: any, value: any) => (Array.isArray(arr) ? arr.includes(value) : false),
  join: (arr: any, separator = ", ") => (Array.isArray(arr) ? arr.join(separator) : String(arr)),

  // 对象函数
  keys: (obj: any) => (obj && typeof obj === "object" ? Object.keys(obj) : []),
  values: (obj: any) => (obj && typeof obj === "object" ? Object.values(obj) : []),
  hasKey: (obj: any, key: any) => (obj && typeof obj === "object" ? key in obj : false),
  get: (obj: any, path: any, defaultValue: any) => {
    if (!obj || typeof obj !== "object") return defaultValue;
    const keys = String(path).split(".");
    let current = obj;
    for (const key of keys) {
      if (current === null || current === undefined) return defaultValue;
      current = current[key];
    }
    return current === undefined ? defaultValue : current;
  },

  // 条件函数
  iif: (condition: any, trueValue: any, falseValue: any) => (condition ? trueValue : falseValue),

  // 工业计算函数
  scale: (value: any, inMin: any, inMax: any, outMin: any, outMax: any) => {
    const ratio = (value - inMin) / (inMax - inMin);
    return outMin + ratio * (outMax - outMin);
  },

  // 状态映射
  statusText: (value: any, mapping: any) => {
    if (!mapping || typeof mapping !== "object") return String(value);
    return mapping[String(value)] ?? String(value);
  },
};

// ==================== 表达式引擎类 ====================

/**
 * 表达式引擎
 */
export class ExpressionEngine {
  private _strict: boolean;

  private _functions: Record<string, AnyFn>;

  private _cache: Map<string, CompiledExpressionFn>;

  /**
   * 创建表达式引擎
   * @param {ExpressionEngineOptions} [options] - 配置选项
   */
  constructor(options: ExpressionEngineOptions = {}) {
    this._strict = options.strict ?? false;
    this._functions = { ...builtinFunctions };
    this._cache = new Map();
  }

  // ==================== 函数注册 ====================

  /**
   * 注册自定义函数
   * @param {string} name - 函数名
   * @param {Function} fn - 函数
   */
  registerFunction(name: string, fn: AnyFn): void {
    if (typeof fn !== "function") {
      throw new TypeError(`Function "${name}" must be a function`);
    }
    this._functions[name] = fn;
  }

  /**
   * 批量注册函数
   * @param {Record<string, Function>} functions - 函数映射
   */
  registerFunctions(functions: Record<string, AnyFn>): void {
    for (const [name, fn] of Object.entries(functions)) {
      this.registerFunction(name, fn);
    }
  }

  /**
   * 获取所有已注册的函数名
   * @returns {string[]}
   */
  getFunctionNames(): string[] {
    return Object.keys(this._functions);
  }

  // ==================== 表达式解析 ====================

  /**
   * 检查字符串是否包含表达式
   * @param {string} str - 输入字符串
   * @returns {boolean}
   */
  hasExpression(str: string): boolean {
    if (typeof str !== "string") return false;
    return EXPRESSION_BLOCK_RE.test(str);
  }

  /**
   * 提取表达式依赖的数据点路径
   * @param {string} expr - 表达式
   * @returns {string[]}
   */
  extractDependencies(expr: string): string[] {
    if (typeof expr !== "string") return [];

    const deps = new Set<string>();

    // 匹配 $dp['path'] 或 $dp["path"] 或 $dp.path
    for (let match = DATA_POINT_DEP_RE.exec(expr); match; match = DATA_POINT_DEP_RE.exec(expr)) {
      const path = match[1] || match[2];
      if (path) deps.add(path);
    }

    return Array.from(deps);
  }

  /**
   * 提取表达式依赖的变量名
   * @param {string} expr - 表达式
   * @returns {{page: string[], global: string[]}}
   */
  extractVarDependencies(expr: string): { page: string[]; global: string[] } {
    if (typeof expr !== "string") return { page: [], global: [] };

    const pageVars = new Set<string>();
    const globalVars = new Set<string>();

    // 匹配 $vars.name 或 $vars['name']
    for (let match = PAGE_VAR_DEP_RE.exec(expr); match; match = PAGE_VAR_DEP_RE.exec(expr)) {
      const name = match[1] || match[2];
      if (name) pageVars.add(name);
    }

    // 匹配 $global.name 或 $global['name']
    for (
      let globalMatch = GLOBAL_VAR_DEP_RE.exec(expr);
      globalMatch;
      globalMatch = GLOBAL_VAR_DEP_RE.exec(expr)
    ) {
      const name = globalMatch[1] || globalMatch[2];
      if (name) globalVars.add(name);
    }

    return {
      page: Array.from(pageVars),
      global: Array.from(globalVars),
    };
  }

  // ==================== 表达式求值 ====================

  /**
   * 执行表达式
   * @param {string} expr - 表达式字符串（带或不带 {{ }}）
   * @param {ExpressionContext} context - 上下文
   * @returns {ExpressionResult}
   */
  evaluate(
    expr: string,
    context: ExpressionContext = {},
  ): ExpressionResult & { dependencies?: string[] } {
    try {
      // 去除 {{ }}
      let cleanExpr = String(expr).trim();
      if (cleanExpr.startsWith("{{") && cleanExpr.endsWith("}}")) {
        cleanExpr = cleanExpr.slice(2, -2).trim();
      }

      // 空表达式
      if (!cleanExpr) {
        return { success: true, value: "" };
      }

      // 编译并执行
      const fn = this._compile(cleanExpr);
      const value = fn(context, this._functions);

      return {
        success: true,
        value,
        dependencies: this.extractDependencies(cleanExpr),
      };
    } catch (error: any) {
      return {
        success: false,
        value: undefined,
        error: error.message,
      };
    }
  }

  /**
   * 解析模板字符串（支持混合文本和表达式）
   * @param {string} template - 模板字符串，如 "温度: {{ $dp['temp'] }}℃"
   * @param {ExpressionContext} context - 上下文
   * @returns {ExpressionResult}
   */
  evaluateTemplate(
    template: string,
    context: ExpressionContext = {},
  ): ExpressionResult & { dependencies?: string[] } {
    if (typeof template !== "string") {
      return { success: true, value: template };
    }

    // 如果不包含表达式，直接返回
    if (!this.hasExpression(template)) {
      return { success: true, value: template };
    }

    try {
      const allDeps: string[] = [];

      // 替换所有 {{ expr }}
      const result = template.replace(TEMPLATE_EXPR_RE, (match, expr) => {
        const evalResult = this.evaluate(expr.trim(), context);
        if (!evalResult.success) {
          throw new Error(evalResult.error);
        }
        if (evalResult.dependencies) {
          allDeps.push(...evalResult.dependencies);
        }
        return String(evalResult.value ?? "");
      });

      return {
        success: true,
        value: result,
        dependencies: [...new Set(allDeps)],
      };
    } catch (error: any) {
      return {
        success: false,
        value: template,
        error: error.message,
      };
    }
  }

  /**
   * 编译表达式为可执行函数
   * @param {string} expr - 表达式
   * @returns {Function}
   * @private
   */
  _compile(expr: string): CompiledExpressionFn {
    // 检查缓存
    if (this._cache.has(expr)) {
      return this._cache.get(expr)!;
    }

    // 构建函数体
    // 安全处理：创建一个受限的执行环境
    const funcBody = `
      "use strict";
      const { $dp = {}, $vars = {}, $global = {}, $props = {}, $event, $item, $index } = ctx;
      const state = ctx.state ?? $global.state ?? $global;
      const fns = funcs;
      
      // 注入函数到作用域
      ${Object.keys(this._functions)
        .map((name) => `const ${name} = fns.${name};`)
        .join("\n")}
      
      return (${expr});
    `;

    try {
      const fn = createDynamicFunction("ctx", "funcs", funcBody);
      const compiled = fn as unknown as CompiledExpressionFn;

      // 缓存编译结果
      this._cache.set(expr, compiled);

      return compiled;
    } catch (error: any) {
      throw new Error(`表达式编译失败: ${error.message}`);
    }
  }

  /**
   * 清除编译缓存
   */
  clearCache(): void {
    this._cache.clear();
  }

  // ==================== 便捷方法 ====================

  /**
   * 简单求值（返回值或 undefined）
   * @param {string} expr - 表达式
   * @param {ExpressionContext} context - 上下文
   * @returns {*}
   */
  eval(expr: string, context: ExpressionContext = {}): unknown {
    const result = this.evaluate(expr, context);
    return result.success ? result.value : undefined;
  }

  /**
   * 求值模板（返回字符串或原值）
   * @param {string} template - 模板
   * @param {ExpressionContext} context - 上下文
   * @returns {string}
   */
  evalTemplate(template: string, context: ExpressionContext = {}): unknown {
    const result = this.evaluateTemplate(template, context);
    return result.success ? result.value : template;
  }

  /**
   * 安全求值（失败时返回 fallback）
   * @param {string} expr - 表达式
   * @param {ExpressionContext} context - 上下文
   * @param {*} fallback - 降级值
   * @returns {*}
   */
  safeEval(expr: string, context: ExpressionContext = {}, fallback = undefined): unknown {
    const result = this.evaluate(expr, context);
    return result.success ? result.value : fallback;
  }

  /**
   * 批量求值
   * @param {Record<string, string>} expressions - 表达式映射
   * @param {ExpressionContext} context - 上下文
   * @returns {Record<string, *>}
   */
  evaluateMany(
    expressions: Record<string, string>,
    context: ExpressionContext = {},
  ): Record<string, unknown> {
    const results: Record<string, unknown> = {};
    for (const [key, expr] of Object.entries(expressions)) {
      results[key] = this.eval(expr, context);
    }
    return results;
  }

  // ==================== 校验 ====================

  /**
   * 校验表达式语法
   * @param {string} expr - 表达式
   * @returns {{valid: boolean, error?: string}}
   */
  validate(expr: string): { valid: boolean; error?: string } {
    try {
      let cleanExpr = String(expr).trim();
      if (cleanExpr.startsWith("{{") && cleanExpr.endsWith("}}")) {
        cleanExpr = cleanExpr.slice(2, -2).trim();
      }

      // 尝试编译
      this._compile(cleanExpr);
      return { valid: true };
    } catch (error: any) {
      return { valid: false, error: error.message };
    }
  }
}

/**
 * 默认表达式引擎实例
 */
export const defaultEngine = new ExpressionEngine();

/**
 * 快捷函数：执行表达式
 * @param {string} expr - 表达式
 * @param {ExpressionContext} context - 上下文
 * @returns {*}
 */
export function evaluate(expr: string, context: ExpressionContext = {}): unknown {
  return defaultEngine.eval(expr, context);
}

/**
 * 快捷函数：执行模板
 * @param {string} template - 模板
 * @param {ExpressionContext} context - 上下文
 * @returns {string}
 */
export function evaluateTemplate(template: string, context: ExpressionContext = {}): unknown {
  return defaultEngine.evalTemplate(template, context);
}

export default ExpressionEngine;
