/**
 * 绑定校验器
 * 提供数据绑定的校验功能，包括数据点状态检查
 */

/**
 * @typedef {import('../document/types.js').Binding} Binding
 * @typedef {import('../document/types.js').ComponentNode} ComponentNode
 * @typedef {import('../document/types.js').GraphicNode} GraphicNode
 * @typedef {import('../../data/types.js').DatapointStatus} DatapointStatus
 */

/**
 * @typedef {Object} BindingValidationError
 * 绑定校验错误
 * @property {string} code - 错误代码
 * @property {string} message - 错误信息
 * @property {string} nodeId - 节点 ID
 * @property {string} propPath - 属性路径
 * @property {string} [datapointPath] - 数据点路径
 */

/**
 * @typedef {Object} BindingValidationResult
 * 绑定校验结果
 * @property {boolean} valid - 是否有效
 * @property {BindingValidationError[]} errors - 错误列表
 * @property {BindingValidationError[]} warnings - 警告列表
 */

/**
 * 绑定校验器类
 */
export class BindingValidator {
  /**
   * 创建绑定校验器
   * @param {Object} [options] - 配置选项
   */
  constructor(options = {}) {
    /** @type {Map<string, DatapointStatus>} */
    this._datapointStatusCache = new Map();

    /** @type {Set<string>} */
    this._knownVariables = new Set();
  }

  /**
   * 设置数据点状态缓存
   * @param {Map<string, DatapointStatus>} statusMap - 数据点状态映射
   */
  setDatapointStatus(statusMap) {
    this._datapointStatusCache = statusMap;
  }

  /**
   * 设置已知变量列表
   * @param {Set<string>} variables - 变量名集合
   */
  setKnownVariables(variables) {
    this._knownVariables = variables;
  }

  /**
   * 校验节点的所有绑定
   * @param {ComponentNode | GraphicNode} node - 节点
   * @returns {BindingValidationResult} 校验结果
   */
  validateNodeBindings(node) {
    /** @type {BindingValidationError[]} */
    const errors = [];
    /** @type {BindingValidationError[]} */
    const warnings = [];

    if (!node.bindings) {
      return { valid: true, errors, warnings };
    }

    for (const [propPath, binding] of Object.entries(node.bindings)) {
      const result = this.validateBinding(binding, node.id, propPath);
      errors.push(...result.errors);
      warnings.push(...result.warnings);
    }

    return {
      valid: errors.length === 0,
      errors,
      warnings,
    };
  }

  /**
   * 校验单个绑定
   * @param {Binding} binding - 绑定配置
   * @param {string} nodeId - 节点 ID
   * @param {string} propPath - 属性路径
   * @returns {BindingValidationResult} 校验结果
   */
  validateBinding(binding, nodeId, propPath) {
    /** @type {BindingValidationError[]} */
    const errors = [];
    /** @type {BindingValidationError[]} */
    const warnings = [];

    if (!binding || !binding.kind) {
      errors.push({
        code: "BINDING_INVALID",
        message: "绑定配置无效",
        nodeId,
        propPath,
      });
      return { valid: false, errors, warnings };
    }

    switch (binding.kind) {
      case "datapoint":
        this._validateDatapointBinding(binding, nodeId, propPath, errors, warnings);
        break;
      case "var":
        this._validateVarBinding(binding, nodeId, propPath, errors, warnings);
        break;
      case "expr":
        this._validateExprBinding(binding, nodeId, propPath, errors, warnings);
        break;
      default:
        errors.push({
          code: "BINDING_KIND_UNKNOWN",
          message: `未知的绑定类型: ${binding.kind}`,
          nodeId,
          propPath,
        });
    }

    return {
      valid: errors.length === 0,
      errors,
      warnings,
    };
  }

  // ==================== 私有方法 ====================

  /**
   * 校验数据点绑定
   * @param {import('../document/types.js').DatapointBinding} binding
   * @param {string} nodeId
   * @param {string} propPath
   * @param {BindingValidationError[]} errors
   * @param {BindingValidationError[]} warnings
   * @private
   */
  _validateDatapointBinding(binding, nodeId, propPath, errors, warnings) {
    if (!binding.path) {
      errors.push({
        code: "DATAPOINT_PATH_MISSING",
        message: "数据点路径缺失",
        nodeId,
        propPath,
      });
      return;
    }

    // 检查数据点状态
    const status = this._datapointStatusCache.get(binding.path);
    if (status) {
      if (status === "invalid") {
        errors.push({
          code: "DATAPOINT_INVALID",
          message: `数据点已失效: ${binding.path}`,
          nodeId,
          propPath,
          datapointPath: binding.path,
        });
      } else if (status === "unknown") {
        warnings.push({
          code: "DATAPOINT_UNKNOWN",
          message: `数据点状态未知: ${binding.path}`,
          nodeId,
          propPath,
          datapointPath: binding.path,
        });
      }
    }

    // 校验 transform 操作
    if (binding.transform && Array.isArray(binding.transform)) {
      for (const op of binding.transform) {
        if (!op.op) {
          errors.push({
            code: "TRANSFORM_OP_MISSING",
            message: "transform 操作缺少 op 字段",
            nodeId,
            propPath,
          });
        }
      }
    }
  }

  /**
   * 校验变量绑定
   * @param {import('../document/types.js').VarBinding} binding
   * @param {string} nodeId
   * @param {string} propPath
   * @param {BindingValidationError[]} errors
   * @param {BindingValidationError[]} warnings
   * @private
   */
  _validateVarBinding(binding, nodeId, propPath, errors, warnings) {
    if (!binding.name) {
      errors.push({
        code: "VAR_NAME_MISSING",
        message: "变量名缺失",
        nodeId,
        propPath,
      });
      return;
    }

    if (!["page", "global"].includes(binding.scope)) {
      errors.push({
        code: "VAR_SCOPE_INVALID",
        message: `变量作用域无效: ${binding.scope}`,
        nodeId,
        propPath,
      });
    }

    // 检查变量是否存在
    const varKey =
      binding.scope === "global" ? `global.${binding.name}` : binding.name;
    if (this._knownVariables.size > 0 && !this._knownVariables.has(varKey)) {
      warnings.push({
        code: "VAR_NOT_FOUND",
        message: `变量可能不存在: ${binding.name} (${binding.scope})`,
        nodeId,
        propPath,
      });
    }
  }

  /**
   * 校验表达式绑定
   * @param {import('../document/types.js').ExprBinding} binding
   * @param {string} nodeId
   * @param {string} propPath
   * @param {BindingValidationError[]} errors
   * @param {BindingValidationError[]} warnings
   * @private
   */
  _validateExprBinding(binding, nodeId, propPath, errors, warnings) {
    if (!binding.expr) {
      errors.push({
        code: "EXPR_MISSING",
        message: "表达式缺失",
        nodeId,
        propPath,
      });
      return;
    }

    // 简单的语法检查
    const expr = binding.expr;

    // 检查是否包含表达式标记
    if (!expr.includes("{{") || !expr.includes("}}")) {
      warnings.push({
        code: "EXPR_FORMAT_WARNING",
        message: "表达式可能格式不正确，应使用 {{ }} 包裹",
        nodeId,
        propPath,
      });
    }

    // 检查括号匹配
    const openCount = (expr.match(/\{\{/g) || []).length;
    const closeCount = (expr.match(/\}\}/g) || []).length;
    if (openCount !== closeCount) {
      errors.push({
        code: "EXPR_BRACKETS_MISMATCH",
        message: "表达式括号不匹配",
        nodeId,
        propPath,
      });
    }
  }
}

export default BindingValidator;

