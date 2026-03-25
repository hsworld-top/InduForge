/**
 * Transform 操作函数
 * 用于数据绑定的值转换
 */

import dayjs from "dayjs";

export interface TransformOp {
  op: string;
  args?: unknown[];
}

export type TransformFunction = (value: unknown, ...args: unknown[]) => unknown;

export function toFixed(value: unknown, digits = 2): string {
  const num = Number(value);
  if (Number.isNaN(num)) return String(value);
  return num.toFixed(digits);
}

export function round(value: unknown, decimals = 0): number {
  const num = Number(value);
  if (Number.isNaN(num)) return num;
  const factor = 10 ** decimals;
  return Math.round(num * factor) / factor;
}

export function floor(value: unknown): number {
  const num = Number(value);
  if (Number.isNaN(num)) return num;
  return Math.floor(num);
}

export function ceil(value: unknown): number {
  const num = Number(value);
  if (Number.isNaN(num)) return num;
  return Math.ceil(num);
}

export function abs(value: unknown): number {
  const num = Number(value);
  if (Number.isNaN(num)) return num;
  return Math.abs(num);
}

export function clamp(value: unknown, min: number, max: number): number {
  const num = Number(value);
  if (Number.isNaN(num)) return num;
  return Math.min(Math.max(num, min), max);
}

export function percent(value: unknown, decimals = 0): string {
  const num = Number(value);
  if (Number.isNaN(num)) return String(value);
  return `${(num * 100).toFixed(decimals)}%`;
}

export function prefix(value: unknown, prefixText: string): string {
  return String(prefixText) + String(value);
}

export function suffix(value: unknown, suffixText: string): string {
  return String(value) + String(suffixText);
}

export function format(value: unknown, template: string): string {
  if (!template) return String(value);

  return template.replace(/%(\.\d+)?[sdf]/g, (match) => {
    if (match === "%s") {
      return String(value);
    }
    if (match === "%d") {
      return String(Math.floor(Number(value)));
    }
    if (match.startsWith("%.") && match.endsWith("f")) {
      const decimals = Number.parseInt(match.slice(2, -1), 10);
      return Number(value).toFixed(decimals);
    }
    return match;
  });
}

export function truncate(value: unknown, maxLength: number, ellipsis = "..."): string {
  const str = String(value);
  if (str.length <= maxLength) return str;
  return str.slice(0, maxLength - ellipsis.length) + ellipsis;
}

export function toUpperCase(value: unknown): string {
  return String(value).toUpperCase();
}

export function toLowerCase(value: unknown): string {
  return String(value).toLowerCase();
}

export function trim(value: unknown): string {
  return String(value).trim();
}

export function dateFormat(value: unknown, formatStr = "YYYY-MM-DD HH:mm:ss"): string {
  if (!value) return "";
  const parsed = dayjs(value as string | number | Date);
  if (!parsed.isValid()) return String(value);
  return parsed.format(formatStr);
}

export function fromNow(value: unknown): string {
  if (!value) return "";
  const parsed = dayjs(value as string | number | Date);
  if (!parsed.isValid()) return String(value);

  const now = dayjs();
  const diffSeconds = now.diff(parsed, "second");

  if (diffSeconds < 60) return "刚刚";
  if (diffSeconds < 3600) return `${Math.floor(diffSeconds / 60)} 分钟前`;
  if (diffSeconds < 86400) return `${Math.floor(diffSeconds / 3600)} 小时前`;
  if (diffSeconds < 604800) return `${Math.floor(diffSeconds / 86400)} 天前`;

  return parsed.format("YYYY-MM-DD");
}

export function map(
  value: unknown,
  mapping: Record<string, unknown> | null | undefined,
  defaultValue?: unknown,
): unknown {
  if (!mapping || typeof mapping !== "object") return value;
  const key = String(value);
  return key in mapping ? mapping[key] : (defaultValue ?? value);
}

export function boolMap(value: unknown, trueValue: unknown, falseValue: unknown): unknown {
  return value ? trueValue : falseValue;
}

export function rangeMap(
  value: unknown,
  ranges: Array<{ min?: number; max?: number; value: unknown }> | null | undefined,
  defaultValue?: unknown,
): unknown {
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

export function ifNull(value: unknown, replacement: unknown): unknown {
  return value ?? replacement;
}

export function ifEmpty(value: unknown, replacement: unknown): unknown {
  if (value === null || value === undefined || value === "") {
    return replacement;
  }
  return value;
}

export function ifNaN(value: unknown, replacement: unknown): unknown {
  const num = Number(value);
  return Number.isNaN(num) ? replacement : num;
}

export function length(value: unknown): number {
  if (Array.isArray(value)) return value.length;
  if (typeof value === "string") return value.length;
  return 0;
}

export function first(value: unknown): unknown {
  if (Array.isArray(value)) return value[0];
  return value;
}

export function last(value: unknown): unknown {
  if (Array.isArray(value)) return value.at(-1);
  return value;
}

export function join(value: unknown, separator = ", "): string {
  if (Array.isArray(value)) return value.join(separator);
  return String(value);
}

/** 具体实现参数比 `unknown` 更窄；运行时由 executeTransform 传入 JSON 侧参数，统一按 TransformFunction 注册 */
export const transformRegistry = {
  toFixed,
  round,
  floor,
  ceil,
  abs,
  clamp,
  percent,
  prefix,
  suffix,
  format,
  truncate,
  toUpperCase,
  toLowerCase,
  trim,
  dateFormat,
  fromNow,
  map,
  boolMap,
  rangeMap,
  ifNull,
  ifEmpty,
  ifNaN,
  length,
  first,
  last,
  join,
} as Record<string, TransformFunction>;

export function registerTransform(name: string, fn: TransformFunction): void {
  if (typeof fn !== "function") {
    throw new TypeError(`Transform "${name}" must be a function`);
  }
  transformRegistry[name] = fn;
}

export function executeTransform(value: unknown, op: TransformOp): unknown {
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

export function applyTransforms(
  value: unknown,
  transforms: TransformOp[] | null | undefined,
): unknown {
  if (!Array.isArray(transforms) || transforms.length === 0) {
    return value;
  }

  let result: unknown = value;
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
