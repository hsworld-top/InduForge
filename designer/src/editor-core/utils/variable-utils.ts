/**
 * 变量工具函数：全局变量值的归一化与默认值映射。
 *
 * @module editor-core/utils/variable-utils
 */

/** 变量定义最小型（画布 / 预览 / store 共用） */
export interface GlobalVariableDetail {
  type?: string;
  default?: unknown;
  [key: string]: unknown;
}

const REGEXP_LITERAL_RE = /^\/(.*)\/([gimsuy]*)$/;
const compileDynamicFunction = (source: string): ((...args: unknown[]) => unknown) => {
  // 这里是受控动态执行入口，保留 Function 仅用于兼容用户自定义函数序列化
  // eslint-disable-next-line no-new-func
  return new Function(source) as (...args: unknown[]) => unknown;
};

/**
 * 归一化全局变量值（function / set / map / regexp 等序列化形态）
 */
export function normalizeGlobalValue(detail: unknown): unknown {
  const d = detail as GlobalVariableDetail | null | undefined;
  const type = d?.type;
  const raw = d?.default;
  if (type === "function") {
    if (typeof raw === "function") return raw;
    if (typeof raw === "string") {
      const text = raw.trim();
      if (!text) return () => undefined;
      try {
        if (
          text.startsWith("function") ||
          text.startsWith("async function") ||
          text.startsWith("(") ||
          text.startsWith("async (") ||
          text.startsWith("async(")
        ) {
          // eslint-disable-next-line no-new-func
          return new Function(`return (${text});`)() as unknown;
        }
        return compileDynamicFunction(text);
      } catch {
        return () => undefined;
      }
    }
    return () => undefined;
  }
  if (type === "set") {
    if (raw instanceof Set) return raw;
    if (Array.isArray(raw)) return new Set(raw);
    if (typeof raw === "string") {
      try {
        const parsed = JSON.parse(raw) as unknown;
        return new Set(Array.isArray(parsed) ? parsed : []);
      } catch {
        return new Set();
      }
    }
    return new Set();
  }
  if (type === "map") {
    if (raw instanceof Map) return raw;
    if (Array.isArray(raw)) return new Map(raw as Iterable<readonly [unknown, unknown]>);
    if (raw && typeof raw === "object")
      return new Map(Object.entries(raw as Record<string, unknown>));
    if (typeof raw === "string") {
      try {
        const parsed = JSON.parse(raw) as unknown;
        if (Array.isArray(parsed)) return new Map(parsed as Iterable<readonly [unknown, unknown]>);
        if (parsed && typeof parsed === "object") {
          return new Map(Object.entries(parsed as Record<string, unknown>));
        }
      } catch {
        return new Map();
      }
    }
    return new Map();
  }
  if (type === "regexp") {
    if (raw instanceof RegExp) return raw;
    if (typeof raw === "string") {
      try {
        const match = raw.match(REGEXP_LITERAL_RE);
        if (match) return new RegExp(match[1] ?? "", match[2] ?? "");
        return new RegExp(raw);
      } catch {
        return null;
      }
    }
  }
  return raw ?? null;
}

/**
 * 由变量定义表构建「名 → 归一化默认值」
 */
export function buildVarValuesFromDefinitions(
  definitions: Record<string, unknown> | null | undefined,
): Record<string, unknown> {
  const result: Record<string, unknown> = {};
  if (!definitions || typeof definitions !== "object") return result;
  for (const [name, detail] of Object.entries(definitions)) {
    result[name] = normalizeGlobalValue(detail);
  }
  return result;
}
