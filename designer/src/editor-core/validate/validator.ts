/**
 * Schema 校验器
 */

import type { ComponentNode, GraphicNode, ProjectSchema } from '../document/types.ts'

export interface ValidationError {
  code: string
  message: string
  path: string
  nodeId?: string
}

export interface ValidationWarning {
  code: string
  message: string
  path: string
  nodeId?: string
}

export interface ValidationResult {
  valid: boolean
  errors: ValidationError[]
  warnings: ValidationWarning[]
}

export class Validator {
  private _strict: boolean

  constructor(options: { strict?: boolean } = {}) {
    this._strict = options.strict === true
  }

  validate(schema: ProjectSchema | null | undefined): ValidationResult {
    void this._strict
    const errors: ValidationError[] = []
    const warnings: ValidationWarning[] = []

    this._validateStructure(schema, errors)
    this._validatePages(schema, errors, warnings)
    this._validateNodeReferences(schema, errors)
    this._validateGraphicReferences(schema, errors)

    return {
      valid: errors.length === 0,
      errors,
      warnings,
    }
  }

  validatePage(schema: ProjectSchema, pageId: string): ValidationResult {
    const errors: ValidationError[] = []
    const warnings: ValidationWarning[] = []

    const page = schema.pagesById?.[pageId]
    if (!page) {
      errors.push({
        code: 'PAGE_NOT_FOUND',
        message: `页面不存在: ${pageId}`,
        path: `pagesById.${pageId}`,
      })
      return { valid: false, errors, warnings }
    }

    if (!schema.nodesById?.[page.rootNodeId]) {
      errors.push({
        code: 'ROOT_NODE_MISSING',
        message: `页面根节点不存在: ${page.rootNodeId}`,
        path: `pagesById.${pageId}.rootNodeId`,
        nodeId: pageId,
      })
    }

    if (page.graphicsIds) {
      for (const graphicId of page.graphicsIds) {
        if (!schema.graphicsById?.[graphicId]) {
          errors.push({
            code: 'GRAPHIC_NOT_FOUND',
            message: `图形不存在: ${graphicId}`,
            path: `pagesById.${pageId}.graphicsIds`,
            nodeId: pageId,
          })
        }
      }
    }

    return {
      valid: errors.length === 0,
      errors,
      warnings,
    }
  }

  validateNode(node: ComponentNode): ValidationResult {
    const errors: ValidationError[] = []
    const warnings: ValidationWarning[] = []

    if (!node.id) {
      errors.push({
        code: 'NODE_ID_MISSING',
        message: '节点缺少 ID',
        path: 'id',
      })
    }

    if (!node.type) {
      errors.push({
        code: 'NODE_TYPE_MISSING',
        message: '节点缺少类型',
        path: 'type',
        nodeId: node.id,
      })
    }

    return {
      valid: errors.length === 0,
      errors,
      warnings,
    }
  }

  private _validateStructure(
    schema: ProjectSchema | null | undefined,
    errors: ValidationError[],
  ): void {
    if (!schema) {
      errors.push({
        code: 'SCHEMA_EMPTY',
        message: 'Schema 为空',
        path: '',
      })
      return
    }

    if (!schema.project) {
      errors.push({
        code: 'PROJECT_META_MISSING',
        message: '缺少工程元信息',
        path: 'project',
      })
    }

    if (!schema.pagesById) {
      errors.push({
        code: 'PAGES_MISSING',
        message: '缺少页面映射',
        path: 'pagesById',
      })
    }

    if (!schema.nodesById) {
      errors.push({
        code: 'NODES_MISSING',
        message: '缺少节点映射',
        path: 'nodesById',
      })
    }
  }

  private _validatePages(
    schema: ProjectSchema | null | undefined,
    errors: ValidationError[],
    warnings: ValidationWarning[],
  ): void {
    if (!schema?.pagesById) return

    const pageIds = Object.keys(schema.pagesById)

    if (pageIds.length === 0) {
      warnings.push({
        code: 'NO_PAGES',
        message: '工程中没有页面',
        path: 'pagesById',
      })
    }

    if (schema.entry?.homePageId) {
      if (!schema.pagesById[schema.entry.homePageId]) {
        errors.push({
          code: 'HOME_PAGE_NOT_FOUND',
          message: `首页不存在: ${schema.entry.homePageId}`,
          path: 'entry.homePageId',
        })
      }
    }

    for (const pageId of pageIds) {
      const result = this.validatePage(schema, pageId)
      errors.push(...result.errors)
      warnings.push(...result.warnings)
    }
  }

  private _validateNodeReferences(
    schema: ProjectSchema | null | undefined,
    errors: ValidationError[],
  ): void {
    if (!schema?.nodesById) return

    for (const [nodeId, node] of Object.entries(schema.nodesById)) {
      if (node.children && Array.isArray(node.children)) {
        for (const childId of node.children) {
          if (!schema.nodesById[childId]) {
            errors.push({
              code: 'CHILD_NODE_NOT_FOUND',
              message: `子节点不存在: ${childId}`,
              path: `nodesById.${nodeId}.children`,
              nodeId,
            })
          }
        }
      }
    }
  }

  private _validateGraphicReferences(
    schema: ProjectSchema | null | undefined,
    errors: ValidationError[],
  ): void {
    if (!schema?.graphicsById) return

    for (const [graphicId, graphic] of Object.entries(schema.graphicsById) as [
      string,
      GraphicNode,
    ][]) {
      if (graphic.type === 'Canvas.Symbol' && graphic.props && 'symbolId' in graphic.props) {
        const symbolId = (graphic.props as { symbolId?: string }).symbolId
        if (symbolId && !schema.symbolsById?.[symbolId]) {
          errors.push({
            code: 'SYMBOL_NOT_FOUND',
            message: `符号不存在: ${symbolId}`,
            path: `graphicsById.${graphicId}.props.symbolId`,
            nodeId: graphicId,
          })
        }
      }
    }
  }
}

export default Validator
