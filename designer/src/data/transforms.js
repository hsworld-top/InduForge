/**
 * Transform 操作函数
 * 用于数据绑定的值转换
 */

import dayjs from "dayjs";

/**
 * @typedef {import('./types.js').TransformOp} TransformOp
 * @typedef {import('./types.js').TransformFunction} TransformFunction
 */

// ==================== 数值转换 ====================

/**
 * 保留小数位
 * @param {*} value - 输入值
 * @param {number} [digits=2] - 小数位数
 * @returns {string}
 */
export function toFixed(value, digits = 2) {
  const num = Number(value);
  if (Number.isNaN(num)) return String(value);
  return num.toFixed(digits);
}

/**
 * 四舍五入
 * @param {*} value - 输入值
 * @param {number} [decimals=0] - 小数位数
 * @returns {number}
 */
export function round(value, decimals = 0) {
  const num = Number(value);
  if (Number.isNaN(num)) return num;
  const factor = Math.pow(10, decimals);
  return Math.round(num * factor) / factor;
}

/**
 * 向下取整
 * @param {*} value - 输入值
 * @returns {number}
 */
export function floor(value) {
  const num = Number(value);
  if (Number.isNaN(num)) return num;
  return Math.floor(num);
}

/**
 * 向上取整
 * @param {*} value - 输入值
 * @returns {number}
 */
export function ceil(value) {
  const num = Number(value);
  if (Number.isNaN(num)) return num;
  return Math.ceil(num);
}

/**
 * 绝对值
 * @param {*} value - 输入值
 * @returns {number}
 */
export function abs(value) {
  const num = Number(value);
  if (Number.isNaN(num)) return num;
  return Math.abs(num);
}

/**
 * 限制范围
 * @param {*} value - 输入值
 * @param {number} min - 最小值
 * @param {number} max - 最大值
 * @returns {number}
 */
export function clamp(value, min, max) {
  const num = Number(value);
  if (Number.isNaN(num)) return num;
  return Math.min(Math.max(num, min), max);
}

/**
 * 百分比
 * @param {*} value - 输入值
 * @param {number} [decimals=0] - 小数位数
 * @returns {string}
 */
export function percent(value, decimals = 0) {
  const num = Number(value);
  if (Number.isNaN(num)) return String(value);
  return (num * 100).toFixed(decimals) + "%";
}

// ==================== 字符串转换 ====================

/**
 * 添加前缀
 * @param {*} value - 输入值
 * @param {string} prefix - 前缀
 * @returns {string}
 */
export function prefix(value, prefix) {
  return String(prefix) + String(value);
}

/**
 * 添加后缀
 * @param {*} value - 输入值
 * @param {string} suffix - 后缀
 * @returns {string}
 */
export function suffix(value, suffix) {
  return String(value) + String(suffix);
}

/**
 * 格式化模板
 * @param {*} value - 输入值
 * @param {string} template - 模板字符串，使用 %s 或 %.Nf 占位
 * @returns {string}
 */
export function format(value, template) {
  if (!template) return String(value);

  // 支持 printf 风格的格式化
  return template.replace(/%(\.\d+)?[sdf]/g, (match) => {
    if (match === "%s") {
      return String(value);
    }
    if (match === "%d") {
      return String(Math.floor(Number(value)));
    }
    if (match.startsWith("%.") && match.endsWith("f")) {
      const decimals = parseInt(match.slice(2, -1), 10);
      return Number(value).toFixed(decimals);
    }
    return match;
  });
}

/**
 * 截断字符串
 * @param {*} value - 输入值
 * @param {number} maxLength - 最大长度
 * @param {string} [ellipsis='...'] - 省略符
 * @returns {string}
 */
export function truncate(value, maxLength, ellipsis = "...") {
  const str = String(value);
  if (str.length <= maxLength) return str;
  return str.slice(0, maxLength - ellipsis.length) + ellipsis;
}

/**
 * 转大写
 * @param {*} value - 输入值
 * @returns {string}
 */
export function toUpperCase(value) {
  return String(value).toUpperCase();
}

/**
 * 转小写
 * @param {*} value - 输入值
 * @returns {string}
 */
export function toLowerCase(value) {
  return String(value).toLowerCase();
}

/**
 * 去除首尾空白
 * @param {*} value - 输入值
 * @returns {string}
 */
export function trim(value) {
  return String(value).trim();
}

// ==================== 日期时间转换 ====================

/**
 * 日期格式化
 * @param {*} value - 输入值（时间戳或日期字符串）
 * @param {string} [formatStr='YYYY-MM-DD HH:mm:ss'] - 格式字符串
 * @returns {string}
 */
export function dateFormat(value, formatStr = "YYYY-MM-DD HH:mm:ss") {
  if (!value) return "";
  const d = dayjs(value);
  if (!d.isValid()) return String(value);
  return d.format(formatStr);
}

/**
 * 相对时间
 * @param {*} value - 输入值（时间戳或日期字符串）
 * @returns {string}
 */
export function fromNow(value) {
  if (!value) return "";
  const d = dayjs(value);
  if (!d.isValid()) return String(value);

  const now = dayjs();
  const diffSeconds = now.diff(d, "second");

  if (diffSeconds < 60) return "刚刚";
  if (diffSeconds < 3600) return `${Math.floor(diffSeconds / 60)} 分钟前`;
  if (diffSeconds < 86400) return `${Math.floor(diffSeconds / 3600)} 小时前`;
  if (diffSeconds < 604800) return `${Math.floor(diffSeconds / 86400)} 天前`;

  return d.format("YYYY-MM-DD");
}

// ==================== 值映射 ====================

/**
 * 值映射
 * @param {*} value - 输入值
 * @param {Object} mapping - 映射对象
 * @param {*} [defaultValue] - 默认值
 * @returns {*}
 */
export function map(value, mapping, defaultValue) {
  if (!mapping || typeof mapping !== "object") return value;
  const key = String(value);
  return key in mapping ? mapping[key] : (defaultValue ?? value);
}

/**
 * 布尔映射
 * @param {*} value - 输入值
 * @param {*} trueValue - true 时的值
 * @param {*} falseValue - false 时的值
 * @returns {*}
 */
export function boolMap(value, trueValue, falseValue) {
  return Boolean(value) ? trueValue : falseValue;
}

/**
 * 范围映射
 * @param {*} value - 输入值
 * @param {Array<{min?: number, max?: number, value: *}>} ranges - 范围配置
 * @param {*} [defaultValue] - 默认值
 * @returns {*}
 */
export function rangeMap(value, ranges, defaultValue) {
  const num = Number(value);
  if (Number.isNaN(num) || !Array.isArray(ranges)) return defaultValue ?? value;

  for (const range of ranges) {
    const min = range.min ?? -Infinity;
    const max = range.max ?? Infinity;
    if (num >= min && num < max) {
      return range.value;
    }
  }

  return defaultValue ?? value;
}

// ==================== 空值处理 ====================

/**
 * 空值替换
 * @param {*} value - 输入值
 * @param {*} replacement - 替换值
 * @returns {*}
 */
export function ifNull(value, replacement) {
  return value === null || value === undefined ? replacement : value;
}

/**
 * 空值或空字符串替换
 * @param {*} value - 输入值
 * @param {*} replacement - 替换值
 * @returns {*}
 */
export function ifEmpty(value, replacement) {
  if (value === null || value === undefined || value === "") {
    return replacement;
  }
  return value;
}

/**
 * NaN 替换
 * @param {*} value - 输入值
 * @param {*} replacement - 替换值
 * @returns {*}
 */
export function ifNaN(value, replacement) {
  const num = Number(value);
  return Number.isNaN(num) ? replacement : num;
}

// ==================== 数组操作 ====================

/**
 * 获取数组长度
 * @param {*} value - 输入值
 * @returns {number}
 */
export function length(value) {
  if (Array.isArray(value)) return value.length;
  if (typeof value === "string") return value.length;
  return 0;
}

/**
 * 数组第一个元素
 * @param {*} value - 输入值
 * @returns {*}
 */
export function first(value) {
  if (Array.isArray(value)) return value[0];
  return value;
}

/**
 * 数组最后一个元素
 * @param {*} value - 输入值
 * @returns {*}
 */
export function last(value) {
  if (Array.isArray(value)) return value[value.length - 1];
  return value;
}

/**
 * 数组连接
 * @param {*} value - 输入值
 * @param {string} [separator=', '] - 分隔符
 * @returns {string}
 */
export function join(value, separator = ", ") {
  if (Array.isArray(value)) return value.join(separator);
  return String(value);
}

// ==================== Transform 注册表 ====================

/**
 * Transform 函数注册表
 * @type {Record<string, TransformFunction>}
 */
export const transformRegistry = {
  // 数值
  toFixed,
  round,
  floor,
  ceil,
  abs,
  clamp,
  percent,

  // 字符串
  prefix,
  suffix,
  format,
  truncate,
  toUpperCase,
  toLowerCase,
  trim,

  // 日期
  dateFormat,
  fromNow,

  // 映射
  map,
  boolMap,
  rangeMap,

  // 空值
  ifNull,
  ifEmpty,
  ifNaN,

  // 数组
  length,
  first,
  last,
  join,
};

/**
 * 注册自定义 Transform
 * @param {string} name - 名称
 * @param {TransformFunction} fn - 函数
 */
export function registerTransform(name, fn) {
  if (typeof fn !== "function") {
    throw new Error(`Transform "${name}" must be a function`);
  }
  transformRegistry[name] = fn;
}

/**
 * 执行单个 Transform 操作
 * @param {*} value - 输入值
 * @param {TransformOp} op - 操作
 * @returns {*}
 */
export function executeTransform(value, op) {
  const fn = transformRegistry[op.op];
  if (!fn) {
    console.warn(`Unknown transform: ${op.op}`);
    return value;
  }

  try {
    const args = op.args || [];
    return fn(value, ...args);
  } catch (error) {
    console.error(`Transform error (${op.op}):`, error);
    return value;
  }
}

/**
 * 执行 Transform 链
 * @param {*} value - 输入值
 * @param {TransformOp[]} transforms - Transform 操作链
 * @returns {*}
 */
export function applyTransforms(value, transforms) {
  if (!Array.isArray(transforms) || transforms.length === 0) {
    return value;
  }

  let result = value;
  for (const op of transforms) {
    result = executeTransform(result, op);
  }
  return result;
}

export default {
  transformRegistry,
  registerTransform,
  executeTransform,
  applyTransforms,
};

