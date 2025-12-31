/**
 * MQTT消息解析引擎
 * 支持多种解析类型：JSONPath、正则表达式、JavaScript脚本、固定值
 */

const { logger } = require("../utils/logger");
const { AppError } = require("../utils/AppError");
const { ErrorCodes } = require("../constants/errorCodes");

/**
 * JSONPath 简单实现
 * 支持基本路径如: $.data.temperature, $.sensors[0].value
 */
function evaluateJsonPath(obj, path) {
  if (!path || path === "$") return obj;

  // 移除开头的 $. 或 $
  let cleanPath = path.replace(/^\$\.?/, "");

  if (!cleanPath) return obj;

  // 分割路径并遍历
  const parts = cleanPath.split(/\.|\[|\]/).filter(Boolean);
  let result = obj;

  for (const part of parts) {
    if (result === null || result === undefined) {
      return undefined;
    }

    // 处理数组索引
    if (/^\d+$/.test(part)) {
      const index = parseInt(part, 10);
      if (Array.isArray(result) && index < result.length) {
        result = result[index];
      } else {
        return undefined;
      }
    } else {
      // 处理对象属性
      result = result[part];
    }
  }

  return result;
}

/**
 * 解析消息
 * @param {string} message - 原始消息内容
 * @param {Object} tag - Tag配置对象
 * @returns {Object} 解析结果 { success, value, error }
 */
function parseMessage(message, tag) {
  try {
    const { parseType, parseRule, dataType, defaultValue, transform } = tag;

    let rawValue;

    // 第一步：根据解析类型提取原始值
    switch (parseType) {
      case "jsonpath":
        rawValue = parseJsonPath(message, parseRule);
        break;

      case "regex":
        rawValue = parseRegex(message, parseRule);
        break;

      case "script":
        rawValue = parseScript(message, parseRule);
        break;

      case "fixed":
        rawValue = parseRule; // 固定值直接使用解析规则作为值
        break;

      default:
        throw new AppError(
          `不支持的解析类型: ${parseType}`,
          ErrorCodes.INVALID_INPUT
        );
    }

    // 如果解析失败且有默认值，使用默认值
    if (rawValue === undefined || rawValue === null) {
      if (defaultValue !== undefined && defaultValue !== null) {
        rawValue = defaultValue;
      } else {
        return {
          success: false,
          value: null,
          quality: "bad",
          error: "无法解析值且没有默认值",
        };
      }
    }

    // 第二步：类型转换
    let typedValue = convertType(rawValue, dataType);

    // 第三步：应用值转换函数（如果有）
    if (transform && transform.trim() !== "") {
      try {
        typedValue = applyTransform(typedValue, transform, message);
      } catch (err) {
        logger.warn(
          `[MessageParser] Transform function failed for tag ${tag.code}:`,
          err.message
        );
        // 转换失败时使用原值
      }
    }

    // 第四步：验证
    const validation = validateValue(typedValue, tag.validation);
    if (!validation.valid) {
      return {
        success: false,
        value: typedValue,
        quality: "uncertain",
        error: `验证失败: ${validation.error}`,
      };
    }

    return {
      success: true,
      value: typedValue,
      quality: "good",
      error: null,
    };
  } catch (error) {
    logger.error(
      `[MessageParser] Parse error for tag ${tag.code}:`,
      error.message
    );
    return {
      success: false,
      value: null,
      quality: "bad",
      error: error.message,
    };
  }
}

/**
 * 使用 JSONPath 解析
 */
function parseJsonPath(message, path) {
  try {
    // 尝试将消息解析为JSON
    const jsonObj =
      typeof message === "string" ? JSON.parse(message) : message;
    return evaluateJsonPath(jsonObj, path);
  } catch (error) {
    throw new Error(`JSONPath解析失败: ${error.message}`);
  }
}

/**
 * 使用正则表达式解析
 */
function parseRegex(message, pattern) {
  try {
    const regex = new RegExp(pattern);
    const match = message.match(regex);

    if (!match) {
      return undefined;
    }

    // 如果有捕获组，返回第一个捕获组；否则返回整个匹配
    return match[1] !== undefined ? match[1] : match[0];
  } catch (error) {
    throw new Error(`正则表达式解析失败: ${error.message}`);
  }
}

/**
 * 使用 JavaScript 脚本解析
 * 脚本中可以使用 message 变量访问原始消息
 */
function parseScript(message, script) {
  try {
    // 创建一个安全的沙箱环境
    const sandbox = {
      message,
      JSON,
      Math,
      Date,
      parseInt,
      parseFloat,
      String,
      Number,
      Boolean,
      Array,
      Object,
      console: {
        log: () => {}, // 禁用console.log以提高安全性
        warn: () => {},
        error: () => {},
      },
    };

    // 使用 Function 构造器执行脚本
    const scriptFunction = new Function(
      ...Object.keys(sandbox),
      `return (${script});`
    );
    const result = scriptFunction(...Object.values(sandbox));

    return result;
  } catch (error) {
    throw new Error(`脚本执行失败: ${error.message}`);
  }
}

/**
 * 类型转换
 */
function convertType(value, dataType) {
  if (value === null || value === undefined) {
    return null;
  }

  switch (dataType) {
    case "number":
      const num = Number(value);
      return isNaN(num) ? null : num;

    case "boolean":
      if (typeof value === "boolean") return value;
      if (typeof value === "string") {
        const lower = value.toLowerCase();
        if (lower === "true" || lower === "1") return true;
        if (lower === "false" || lower === "0") return false;
      }
      return Boolean(value);

    case "string":
      return String(value);

    case "object":
    case "array":
      if (typeof value === "string") {
        try {
          return JSON.parse(value);
        } catch {
          return value;
        }
      }
      return value;

    default:
      return value;
  }
}

/**
 * 应用值转换函数
 */
function applyTransform(value, transformScript, originalMessage) {
  try {
    const sandbox = {
      value,
      message: originalMessage,
      Math,
      Date,
      parseInt,
      parseFloat,
      String,
      Number,
      Boolean,
    };

    const transformFunction = new Function(
      ...Object.keys(sandbox),
      `return (${transformScript})(value);`
    );

    return transformFunction(...Object.values(sandbox));
  } catch (error) {
    throw new Error(`值转换失败: ${error.message}`);
  }
}

/**
 * 验证值
 */
function validateValue(value, validation) {
  if (!validation || typeof validation !== "object") {
    return { valid: true };
  }

  // 数值范围验证
  if (typeof value === "number") {
    if (validation.min !== undefined && value < validation.min) {
      return { valid: false, error: `值小于最小值 ${validation.min}` };
    }
    if (validation.max !== undefined && value > validation.max) {
      return { valid: false, error: `值大于最大值 ${validation.max}` };
    }
  }

  // 字符串长度验证
  if (typeof value === "string") {
    if (validation.minLength !== undefined && value.length < validation.minLength) {
      return {
        valid: false,
        error: `长度小于最小长度 ${validation.minLength}`,
      };
    }
    if (validation.maxLength !== undefined && value.length > validation.maxLength) {
      return {
        valid: false,
        error: `长度大于最大长度 ${validation.maxLength}`,
      };
    }

    // 正则表达式验证
    if (validation.pattern) {
      const regex = new RegExp(validation.pattern);
      if (!regex.test(value)) {
        return { valid: false, error: `不匹配模式 ${validation.pattern}` };
      }
    }
  }

  // 枚举值验证
  if (validation.enum && Array.isArray(validation.enum)) {
    if (!validation.enum.includes(value)) {
      return {
        valid: false,
        error: `值不在允许的枚举值中: ${validation.enum.join(", ")}`,
      };
    }
  }

  return { valid: true };
}

/**
 * 批量解析消息（用于一个消息对应多个Tag的场景）
 */
function parseMessageBatch(message, tags) {
  const results = {};

  for (const tag of tags) {
    if (!tag.isEnabled) {
      continue;
    }

    const result = parseMessage(message, tag);
    results[tag.code] = {
      tagId: tag.id,
      tagName: tag.name,
      ...result,
    };
  }

  return results;
}

module.exports = {
  parseMessage,
  parseMessageBatch,
  evaluateJsonPath, // 导出供测试使用
};

