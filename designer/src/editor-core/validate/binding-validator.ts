/**
 * 绑定校验器
 */

import type {
  Binding,
  ComponentNode,
  DatapointBinding,
  ExprBinding,
  GraphicNode,
  VarBinding,
} from '../document/types.ts'

const EXPRESSION_OPEN_REGEX = /\{\{/g
const EXPRESSION_CLOSE_REGEX = /\}\}/g

export type DatapointStatus = 'active' | 'invalid' | 'unknown'

export interface BindingValidationError {
  code: string
  message: string
  nodeId: string
  propPath: string
  datapointPath?: string
}

export interface BindingValidationResult {
  valid: boolean
  errors: BindingValidationError[]
  warnings: BindingValidationError[]
}

export class BindingValidator {
  private _datapointStatusCache: Map<string, DatapointStatus>
  private _knownVariables: Set<string>

  constructor(_options: Record<string, unknown> = {}) {
    this._datapointStatusCache = new Map()
    this._knownVariables = new Set()
  }

  setDatapointStatus(statusMap: Map<string, DatapointStatus>): void {
    this._datapointStatusCache = statusMap
  }

  setKnownVariables(variables: Set<string>): void {
    this._knownVariables = variables
  }

  validateNodeBindings(node: ComponentNode | GraphicNode): BindingValidationResult {
    const errors: BindingValidationError[] = []
    const warnings: BindingValidationError[] = []

    if (!node.bindings) {
      return { valid: true, errors, warnings }
    }

    for (const [propPath, binding] of Object.entries(node.bindings)) {
      const result = this.validateBinding(binding as Binding, node.id, propPath)
      errors.push(...result.errors)
      warnings.push(...result.warnings)
    }

    return {
      valid: errors.length === 0,
      errors,
      warnings,
    }
  }

  validateBinding(
    binding: Binding | null | undefined,
    nodeId: string,
    propPath: string,
  ): BindingValidationResult {
    const errors: BindingValidationError[] = []
    const warnings: BindingValidationError[] = []

    if (!binding || !binding.kind) {
      errors.push({
        code: 'BINDING_INVALID',
        message: '绑定配置无效',
        nodeId,
        propPath,
      })
      return { valid: false, errors, warnings }
    }

    switch (binding.kind) {
      case 'datapoint':
        this._validateDatapointBinding(binding, nodeId, propPath, errors, warnings)
        break
      case 'var':
        this._validateVarBinding(binding, nodeId, propPath, errors, warnings)
        break
      case 'expr':
        this._validateExprBinding(binding, nodeId, propPath, errors, warnings)
        break
      default: {
        const k = (binding as { kind?: string }).kind
        errors.push({
          code: 'BINDING_KIND_UNKNOWN',
          message: `未知的绑定类型: ${k ?? ''}`,
          nodeId,
          propPath,
        })
      }
    }

    return {
      valid: errors.length === 0,
      errors,
      warnings,
    }
  }

  private _validateDatapointBinding(
    binding: DatapointBinding,
    nodeId: string,
    propPath: string,
    errors: BindingValidationError[],
    warnings: BindingValidationError[],
  ): void {
    if (!binding.path) {
      errors.push({
        code: 'DATAPOINT_PATH_MISSING',
        message: '数据点路径缺失',
        nodeId,
        propPath,
      })
      return
    }

    const status = this._datapointStatusCache.get(binding.path)
    if (status) {
      if (status === 'invalid') {
        errors.push({
          code: 'DATAPOINT_INVALID',
          message: `数据点已失效: ${binding.path}`,
          nodeId,
          propPath,
          datapointPath: binding.path,
        })
      } else if (status === 'unknown') {
        warnings.push({
          code: 'DATAPOINT_UNKNOWN',
          message: `数据点状态未知: ${binding.path}`,
          nodeId,
          propPath,
          datapointPath: binding.path,
        })
      }
    }

    if (binding.transform && Array.isArray(binding.transform)) {
      for (const op of binding.transform) {
        if (!op.op) {
          errors.push({
            code: 'TRANSFORM_OP_MISSING',
            message: 'transform 操作缺少 op 字段',
            nodeId,
            propPath,
          })
        }
      }
    }
  }

  private _validateVarBinding(
    binding: VarBinding,
    nodeId: string,
    propPath: string,
    errors: BindingValidationError[],
    warnings: BindingValidationError[],
  ): void {
    if (!binding.name) {
      errors.push({
        code: 'VAR_NAME_MISSING',
        message: '变量名缺失',
        nodeId,
        propPath,
      })
      return
    }

    if (!['page', 'global'].includes(binding.scope)) {
      errors.push({
        code: 'VAR_SCOPE_INVALID',
        message: `变量作用域无效: ${binding.scope}`,
        nodeId,
        propPath,
      })
    }

    const varKey = binding.scope === 'global' ? `global.${binding.name}` : binding.name
    if (this._knownVariables.size > 0 && !this._knownVariables.has(varKey)) {
      warnings.push({
        code: 'VAR_NOT_FOUND',
        message: `变量可能不存在: ${binding.name} (${binding.scope})`,
        nodeId,
        propPath,
      })
    }
  }

  private _validateExprBinding(
    binding: ExprBinding,
    nodeId: string,
    propPath: string,
    errors: BindingValidationError[],
    warnings: BindingValidationError[],
  ): void {
    if (!binding.expr) {
      errors.push({
        code: 'EXPR_MISSING',
        message: '表达式缺失',
        nodeId,
        propPath,
      })
      return
    }

    const expr = binding.expr

    if (!expr.includes('{{') || !expr.includes('}}')) {
      warnings.push({
        code: 'EXPR_FORMAT_WARNING',
        message: '表达式可能格式不正确，应使用 {{ }} 包裹',
        nodeId,
        propPath,
      })
    }

    const openCount = (expr.match(EXPRESSION_OPEN_REGEX) || []).length
    const closeCount = (expr.match(EXPRESSION_CLOSE_REGEX) || []).length
    if (openCount !== closeCount) {
      errors.push({
        code: 'EXPR_BRACKETS_MISMATCH',
        message: '表达式括号不匹配',
        nodeId,
        propPath,
      })
    }
  }
}

export default BindingValidator
