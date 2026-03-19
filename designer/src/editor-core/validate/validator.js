/**
 * Schema 校验器
 * 提供 Schema 结构校验和完整性检查
 */

/**
 * @typedef {import('../document/types.js').ProjectSchema} ProjectSchema
 * @typedef {import('../document/types.js').ValidationResult} ValidationResult
 * @typedef {import('../document/types.js').ValidationError} ValidationError
 * @typedef {import('../document/types.js').ValidationWarning} ValidationWarning
 */

/**
 * Schema 校验器类
 */
export class Validator {
  /**
   * 创建校验器
   * @param {Object} [options] - 配置选项
   * @param {boolean} [options.strict=false] - 严格模式
   */
  constructor(options = {}) {
    /** @type {boolean} */
    this._strict = options.strict === true;
  }

  /**
   * 校验完整工程 Schema
   * @param {ProjectSchema} schema - 工程 Schema
   * @returns {ValidationResult} 校验结果
   */
  validate(schema) {
    /** @type {ValidationError[]} */
    const errors = [];
    /** @type {ValidationWarning[]} */
    const warnings = [];

    // 校验基础结构
    this._validateStructure(schema, errors);

    // 校验页面
    this._validatePages(schema, errors, warnings);

    // 校验节点引用完整性
    this._validateNodeReferences(schema, errors);

    // 校验图形引用完整性
    this._validateGraphicReferences(schema, errors);

    return {
      valid: errors.length === 0,
      errors,
      warnings,
    };
  }

  /**
   * 校验单个页面
   * @param {ProjectSchema} schema - 工程 Schema
   * @param {string} pageId - 页面 ID
   * @returns {ValidationResult} 校验结果
   */
  validatePage(schema, pageId) {
    /** @type {ValidationError[]} */
    const errors = [];
    /** @type {ValidationWarning[]} */
    const warnings = [];

    const page = schema.pagesById?.[pageId];
    if (!page) {
      errors.push({
        code: "PAGE_NOT_FOUND",
        message: `页面不存在: ${pageId}`,
        path: `pagesById.${pageId}`,
      });
      return { valid: false, errors, warnings };
    }

    // 校验页面根节点
    if (!schema.nodesById?.[page.rootNodeId]) {
      errors.push({
        code: "ROOT_NODE_MISSING",
        message: `页面根节点不存在: ${page.rootNodeId}`,
        path: `pagesById.${pageId}.rootNodeId`,
        nodeId: pageId,
      });
    }

    // 校验页面图形引用
    if (page.graphicsIds) {
      for (const graphicId of page.graphicsIds) {
        if (!schema.graphicsById?.[graphicId]) {
          errors.push({
            code: "GRAPHIC_NOT_FOUND",
            message: `图形不存在: ${graphicId}`,
            path: `pagesById.${pageId}.graphicsIds`,
            nodeId: pageId,
          });
        }
      }
    }

    return {
      valid: errors.length === 0,
      errors,
      warnings,
    };
  }

  /**
   * 校验单个节点
   * @param {import('../document/types.js').ComponentNode} node - 节点
   * @returns {ValidationResult} 校验结果
   */
  validateNode(node) {
    /** @type {ValidationError[]} */
    const errors = [];
    /** @type {ValidationWarning[]} */
    const warnings = [];

    if (!node.id) {
      errors.push({
        code: "NODE_ID_MISSING",
        message: "节点缺少 ID",
        path: "id",
      });
    }

    if (!node.type) {
      errors.push({
        code: "NODE_TYPE_MISSING",
        message: "节点缺少类型",
        path: "type",
        nodeId: node.id,
      });
    }

    // TODO: 校验 props、bindings、events 等

    return {
      valid: errors.length === 0,
      errors,
      warnings,
    };
  }

  // ==================== 私有方法 ====================

  /**
   * 校验基础结构
   * @param {ProjectSchema} schema
   * @param {ValidationError[]} errors
   * @private
   */
  _validateStructure(schema, errors) {
    if (!schema) {
      errors.push({
        code: "SCHEMA_EMPTY",
        message: "Schema 为空",
        path: "",
      });
      return;
    }

    if (!schema.project) {
      errors.push({
        code: "PROJECT_META_MISSING",
        message: "缺少工程元信息",
        path: "project",
      });
    }

    if (!schema.pagesById) {
      errors.push({
        code: "PAGES_MISSING",
        message: "缺少页面映射",
        path: "pagesById",
      });
    }

    if (!schema.nodesById) {
      errors.push({
        code: "NODES_MISSING",
        message: "缺少节点映射",
        path: "nodesById",
      });
    }
  }

  /**
   * 校验页面
   * @param {ProjectSchema} schema
   * @param {ValidationError[]} errors
   * @param {ValidationWarning[]} warnings
   * @private
   */
  _validatePages(schema, errors, warnings) {
    if (!schema.pagesById) return;

    const pageIds = Object.keys(schema.pagesById);

    if (pageIds.length === 0) {
      warnings.push({
        code: "NO_PAGES",
        message: "工程中没有页面",
        path: "pagesById",
      });
    }

    // 校验首页是否存在
    if (schema.entry?.homePageId) {
      if (!schema.pagesById[schema.entry.homePageId]) {
        errors.push({
          code: "HOME_PAGE_NOT_FOUND",
          message: `首页不存在: ${schema.entry.homePageId}`,
          path: "entry.homePageId",
        });
      }
    }

    // 校验每个页面
    for (const pageId of pageIds) {
      const result = this.validatePage(schema, pageId);
      errors.push(...result.errors);
      warnings.push(...result.warnings);
    }
  }

  /**
   * 校验节点引用完整性
   * @param {ProjectSchema} schema
   * @param {ValidationError[]} errors
   * @private
   */
  _validateNodeReferences(schema, errors) {
    if (!schema.nodesById) return;

    for (const [nodeId, node] of Object.entries(schema.nodesById)) {
      // 校验子节点引用
      if (node.children && Array.isArray(node.children)) {
        for (const childId of node.children) {
          if (!schema.nodesById[childId]) {
            errors.push({
              code: "CHILD_NODE_NOT_FOUND",
              message: `子节点不存在: ${childId}`,
              path: `nodesById.${nodeId}.children`,
              nodeId,
            });
          }
        }
      }
    }
  }

  /**
   * 校验图形引用完整性
   * @param {ProjectSchema} schema
   * @param {ValidationError[]} errors
   * @private
   */
  _validateGraphicReferences(schema, errors) {
    if (!schema.graphicsById) return;

    for (const [graphicId, graphic] of Object.entries(schema.graphicsById)) {
      // 校验符号引用
      if (graphic.type === "Canvas.Symbol" && graphic.props?.symbolId) {
        if (!schema.symbolsById?.[graphic.props.symbolId]) {
          errors.push({
            code: "SYMBOL_NOT_FOUND",
            message: `符号不存在: ${graphic.props.symbolId}`,
            path: `graphicsById.${graphicId}.props.symbolId`,
            nodeId: graphicId,
          });
        }
      }
    }
  }
}

export default Validator;
