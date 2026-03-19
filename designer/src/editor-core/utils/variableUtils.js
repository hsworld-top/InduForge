/**
 * 变量工具函数
 *
 * 提供全局变量值的归一化和构建功能。
 *
 * @module editor-core/utils/variableUtils
 */

/**
 * 归一化全局变量值
 * @param {Object} detail - 变量详情对象，包含 type 和 default
 * @returns {*} 归一化后的值
 */
export function normalizeGlobalValue(detail) {
  const type = detail?.type;
  const raw = detail?.default;
  if (type === "function") {
    if (typeof raw === "function") return raw;
    if (typeof raw === "string") {
      const text = raw.trim();
      if (!text) return () => undefined;
      try {
        // Treat as function expression or async function expression
        if (
          text.startsWith("function") ||
          text.startsWith("async function") ||
          text.startsWith("(") ||
          text.startsWith("async (") ||
          text.startsWith("async(")
        ) {
          return new Function(`return (${text});`)();
        }
        // Treat as function body
        return new Function(text);
      } catch (error) {
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
        const parsed = JSON.parse(raw);
        return new Set(Array.isArray(parsed) ? parsed : []);
      } catch (error) {
        return new Set();
      }
    }
    return new Set();
  }
  if (type === "map") {
    if (raw instanceof Map) return raw;
    if (Array.isArray(raw)) return new Map(raw);
    if (raw && typeof raw === "object") return new Map(Object.entries(raw));
    if (typeof raw === "string") {
      try {
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed)) return new Map(parsed);
        if (parsed && typeof parsed === "object") {
          return new Map(Object.entries(parsed));
        }
      } catch (error) {
        return new Map();
      }
    }
    return new Map();
  }
  if (type === "regexp") {
    if (raw instanceof RegExp) return raw;
    if (typeof raw === "string") {
      try {
        const match = raw.match(/^\/(.*)\/([gimsuy]*)$/);
        if (match) return new RegExp(match[1], match[2]);
        return new RegExp(raw);
      } catch (error) {
        return null;
      }
    }
  }
  return raw ?? null;
}

/**
 * 构建变量默认值映射
 * @param {Record<string, { default?: any }>} definitions - 变量定义
 * @returns {Record<string, any>}
 */
export function buildVarValuesFromDefinitions(definitions) {
  const result = {};
  if (!definitions || typeof definitions !== "object") return result;
  Object.entries(definitions).forEach(([name, detail]) => {
    result[name] = normalizeGlobalValue(detail);
  });
  return result;
}
