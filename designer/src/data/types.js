/**
 * 数据层类型定义
 * 数据绑定系统的核心类型
 */

// ==================== 数据点状态 ====================

/**
 * @typedef {'active' | 'invalid' | 'unknown'} DatapointStatus
 * 数据点状态
 * - active: 正常可用
 * - invalid: 失效（需人工处理）
 * - unknown: 未知（引用了不存在的路径）
 */

/**
 * @typedef {Object} DatapointStatusInfo
 * 数据点状态详情
 * @property {string} path - 数据点路径
 * @property {DatapointStatus} status - 状态
 * @property {string} [statusReason] - 状态原因
 * @property {string} [lastSeenAt] - 最后看到时间
 * @property {string} [dataType] - 数据类型
 */

// ==================== 变量系统 ====================

/**
 * @typedef {'string' | 'number' | 'boolean' | 'array' | 'object'} VarType
 * 变量类型
 */

/**
 * @typedef {Object} VarDefinition
 * 变量定义
 * @property {VarType} type - 变量类型
 * @property {*} default - 默认值
 * @property {string} [label] - 显示名称
 * @property {string} [description] - 描述
 * @property {*[]} [enum] - 枚举值
 * @property {number} [min] - 最小值（number）
 * @property {number} [max] - 最大值（number）
 * @property {number} [minLength] - 最小长度（string/array）
 * @property {number} [maxLength] - 最大长度（string/array）
 * @property {{type: string}} [items] - 数组元素类型
 * @property {boolean} [persistent] - 是否持久化到 localStorage
 * @property {string} [storageKey] - localStorage key
 */

/**
 * @typedef {Object} VarsDefinitions
 * 变量定义集合
 * @property {Record<string, VarDefinition>} global - 全局变量定义
 * @property {Record<string, Record<string, VarDefinition>>} pages - 页面变量定义
 */

/**
 * @typedef {Object} VarsContext
 * 变量上下文（用于表达式求值）
 * @property {Record<string, *>} $vars - 当前页面变量
 * @property {Record<string, *>} $global - 全局变量
 */

// ==================== 表达式引擎 ====================

/**
 * @typedef {Object} ExpressionContext
 * 表达式上下文
 * @property {Record<string, *>} $dp - 数据点值
 * @property {Record<string, *>} $vars - 页面变量
 * @property {Record<string, *>} $global - 全局变量
 * @property {Record<string, *>} $props - 组件属性
 * @property {*} [$event] - 事件对象（仅在事件处理中）
 * @property {*} [$item] - 循环当前项
 * @property {number} [$index] - 循环索引
 */

/**
 * @typedef {Object} ExpressionResult
 * 表达式求值结果
 * @property {boolean} success - 是否成功
 * @property {*} value - 结果值
 * @property {string} [error] - 错误信息
 * @property {string[]} [dependencies] - 依赖的数据点路径
 */

// ==================== Transform 操作 ====================

/**
 * @typedef {Object} TransformOp
 * 转换操作
 * @property {string} op - 操作名称
 * @property {*[]} [args] - 参数
 */

/**
 * @typedef {function(*,...*): *} TransformFunction
 * 转换函数
 */

// ==================== 数据服务 ====================

/**
 * @typedef {Object} DataServiceOptions
 * 数据服务配置
 * @property {string} [baseUrl] - API 基础路径
 * @property {string} [wsPath] - WebSocket 路径
 * @property {boolean} [autoReconnect] - 是否自动重连
 * @property {number} [reconnectDelay] - 重连延迟（毫秒）
 * @property {number} [maxReconnectAttempts] - 最大重连次数
 */

/**
 * @typedef {Object} SubscriptionOptions
 * 订阅选项
 * @property {boolean} [immediate] - 是否立即获取当前值
 * @property {number} [throttle] - 节流时间（毫秒）
 */

/**
 * @typedef {Object} DatapointValue
 * 数据点值
 * @property {string} path - 数据点路径
 * @property {*} value - 值
 * @property {number} [timestamp] - 时间戳
 * @property {string} [quality] - 质量标识
 */

// ==================== 绑定解析 ====================

/**
 * @typedef {'edit' | 'preview' | 'runtime'} DataMode
 * 数据模式
 * - edit: 设计态，使用 Mock 数据
 * - preview: 预览态，调用开发系统 API
 * - runtime: 运行态，连接节点侧数据源
 */

/**
 * @typedef {Object} BindingResolverOptions
 * 绑定解析器配置
 * @property {DataMode} mode - 数据模式
 * @property {string} [pageId] - 当前页面 ID
 */

/**
 * @typedef {Object} ResolvedBinding
 * 解析后的绑定
 * @property {*} value - 当前值
 * @property {boolean} isLoading - 是否加载中
 * @property {string} [error] - 错误信息
 * @property {DatapointStatus} [status] - 数据点状态（仅 datapoint 绑定）
 * @property {function(): void} [unsubscribe] - 取消订阅函数
 */

// ==================== 诊断信息 ====================

/**
 * @typedef {Object} DiagnosticInfo
 * 诊断信息
 * @property {string} path - 数据点路径
 * @property {DatapointStatus} status - 状态
 * @property {string} [reason] - 原因
 * @property {string[]} [affectedNodeIds] - 受影响的节点 ID
 * @property {number} [lastCheckedAt] - 最后检查时间
 */

/**
 * @typedef {Object} DiagnosticsSummary
 * 诊断摘要
 * @property {number} total - 总数据点数
 * @property {number} active - 正常数量
 * @property {number} invalid - 失效数量
 * @property {number} unknown - 未知数量
 * @property {DiagnosticInfo[]} issues - 问题列表
 */

// ==================== 工厂函数 ====================

/**
 * 创建默认变量定义
 * @param {VarType} type - 变量类型
 * @param {Object} [options] - 额外选项
 * @returns {VarDefinition}
 */
export function createVarDefinition(type, options = {}) {
  const defaults = {
    string: "",
    number: 0,
    boolean: false,
    array: [],
    object: {},
  };

  return {
    type,
    default: options.default ?? defaults[type],
    ...options,
  };
}

/**
 * 创建数据点绑定
 * @param {string} path - 数据点路径
 * @param {Object} [options] - 额外选项
 * @returns {import('../editor-core/types.js').DatapointBinding}
 */
export function createDatapointBinding(path, options = {}) {
  return {
    kind: "datapoint",
    provider: options.provider || "dc_main",
    datapointId: options.datapointId || "",
    path,
    transform: options.transform || [],
    fallback: options.fallback,
    designMock: options.designMock,
  };
}

/**
 * 创建变量绑定
 * @param {'page' | 'global'} scope - 作用域
 * @param {string} name - 变量名
 * @param {Object} [options] - 额外选项
 * @returns {import('../editor-core/types.js').VarBinding}
 */
export function createVarBinding(scope, name, options = {}) {
  return {
    kind: "var",
    scope,
    name,
    transform: options.transform || [],
    fallback: options.fallback,
  };
}

/**
 * 创建表达式绑定
 * @param {string} expr - 表达式
 * @param {Object} [options] - 额外选项
 * @returns {import('../editor-core/types.js').ExprBinding}
 */
export function createExprBinding(expr, options = {}) {
  return {
    kind: "expr",
    expr,
    fallback: options.fallback,
  };
}

/**
 * 判断绑定类型
 * @param {Object} binding - 绑定对象
 * @returns {'datapoint' | 'var' | 'expr' | null}
 */
export function getBindingKind(binding) {
  if (!binding || typeof binding !== "object") return null;
  return binding.kind || null;
}

/**
 * 检查值是否为有效数据
 * @param {*} value - 值
 * @returns {boolean}
 */
export function isValidValue(value) {
  return value !== undefined && value !== null && !Number.isNaN(value);
}

export default {
  createVarDefinition,
  createDatapointBinding,
  createVarBinding,
  createExprBinding,
  getBindingKind,
  isValidValue,
};
