/**
 * Serializer - 序列化器
 * 负责 Schema 的导入导出、版本迁移和差量补丁
 */

import { DocumentModel } from "./DocumentModel.js";

/**
 * @typedef {import('./types.js').ProjectSchema} ProjectSchema
 * @typedef {import('./types.js').Patch} Patch
 * @typedef {import('./types.js').PatchOp} PatchOp
 */

/**
 * 序列化器类
 */
export class Serializer {
  /**
   * 创建序列化器
   * @param {Object} [options] - 配置选项
   * @param {boolean} [options.prettyPrint=true] - 是否美化输出
   * @param {boolean} [options.autoMigrate=true] - 是否自动迁移
   */
  constructor(options = {}) {
    /** @type {boolean} */
    this._prettyPrint = options.prettyPrint !== false;

    /** @type {boolean} */
    this._autoMigrate = options.autoMigrate !== false;
  }

  // ==================== 导出 ====================

  /**
   * 导出文档模型为 JSON 字符串
   * @param {DocumentModel} doc - 文档模型
   * @returns {string}
   */
  export(doc) {
    const schema = this.exportToSchema(doc);
    return this._stringify(schema);
  }

  /**
   * 导出文档模型为 Schema 对象
   * @param {DocumentModel} doc - 文档模型
   * @returns {ProjectSchema}
   */
  exportToSchema(doc) {
    // 更新时间戳
    const schema = JSON.parse(JSON.stringify(doc.schema));
    schema.project.updatedAt = Date.now();
    return schema;
  }

  /**
   * 导出指定页面
   * @param {DocumentModel} doc - 文档模型
   * @param {string} pageId - 页面 ID
   * @returns {Object} 页面数据（包含页面节点和相关组件/图形）
   */
  exportPage(doc, pageId) {
    const page = doc.getPage(pageId);
    if (!page) {
      throw new Error(`页面不存在: ${pageId}`);
    }

    // 收集页面相关的所有节点
    const nodeIds = new Set();
    const collectNodes = (nodeId) => {
      nodeIds.add(nodeId);
      const node = doc.getNode(nodeId);
      if (node && node.children) {
        for (const childId of node.children) {
          collectNodes(childId);
        }
      }
    };
    collectNodes(page.rootNodeId);

    // 构建导出数据
    const nodesById = {};
    for (const nodeId of nodeIds) {
      const node = doc.getNode(nodeId);
      if (node) {
        nodesById[nodeId] = JSON.parse(JSON.stringify(node));
      }
    }

    // 收集图形
    const graphicsById = {};
    if (page.graphicsIds) {
      for (const graphicId of page.graphicsIds) {
        const graphic = doc.getGraphic(graphicId);
        if (graphic) {
          graphicsById[graphicId] = JSON.parse(JSON.stringify(graphic));
        }
      }
    }

    return {
      page: JSON.parse(JSON.stringify(page)),
      nodesById,
      graphicsById,
    };
  }

  // ==================== 导入 ====================

  /**
   * 从 JSON 字符串导入
   * @param {string} json - JSON 字符串
   * @returns {DocumentModel}
   */
  import(json) {
    const schema = this._parse(json);
    return this.importFromSchema(schema);
  }

  /**
   * 从 Schema 对象导入
   * @param {Object} schema - Schema 对象
   * @returns {DocumentModel}
   */
  importFromSchema(schema) {
    // 自动迁移
    const finalSchema = schema;
    // 版本迁移已禁用，直接使用原始 schema

    return new DocumentModel(finalSchema);
  }

  /**
   * 导入页面到现有文档
   * @param {DocumentModel} doc - 目标文档模型
   * @param {Object} pageData - 页面数据（来自 exportPage）
   * @param {Object} [options] - 导入选项
   * @param {boolean} [options.generateNewIds=true] - 是否生成新 ID
   * @returns {string} 导入后的页面 ID
   */
  importPage(doc, pageData, options = {}) {
    const generateNewIds = options.generateNewIds !== false;

    const { page, nodesById, graphicsById } = pageData;
    const idMap = new Map();

    // 生成新 ID 映射
    if (generateNewIds) {
      // 页面 ID
      const newPageId =
        "page_" + crypto.randomUUID().replace(/-/g, "").substring(0, 8);
      idMap.set(page.id, newPageId);
      page.id = newPageId;

      // 节点 ID
      for (const nodeId of Object.keys(nodesById)) {
        const newNodeId =
          "node_" + crypto.randomUUID().replace(/-/g, "").substring(0, 8);
        idMap.set(nodeId, newNodeId);
      }

      // 图形 ID
      for (const graphicId of Object.keys(graphicsById)) {
        const newGraphicId =
          "gfx_" + crypto.randomUUID().replace(/-/g, "").substring(0, 8);
        idMap.set(graphicId, newGraphicId);
      }

      // 更新引用
      page.rootNodeId = idMap.get(page.rootNodeId) || page.rootNodeId;
      page.graphicsIds = (page.graphicsIds || []).map(
        (id) => idMap.get(id) || id,
      );

      // 更新节点
      const newNodesById = {};
      for (const [oldId, node] of Object.entries(nodesById)) {
        const newId = idMap.get(oldId);
        node.id = newId;
        node.children = (node.children || []).map((id) => idMap.get(id) || id);
        newNodesById[newId] = node;
      }
      Object.assign(nodesById, newNodesById);
      for (const oldId of Object.keys(nodesById)) {
        if (!idMap.has(oldId)) continue;
        delete nodesById[oldId];
      }

      // 更新图形
      const newGraphicsById = {};
      for (const [oldId, graphic] of Object.entries(graphicsById)) {
        const newId = idMap.get(oldId);
        graphic.id = newId;
        newGraphicsById[newId] = graphic;
      }
      Object.assign(graphicsById, newGraphicsById);
      for (const oldId of Object.keys(graphicsById)) {
        if (!idMap.has(oldId)) continue;
        delete graphicsById[oldId];
      }
    }

    // 插入页面
    doc._insertPage(page);

    // 插入节点
    for (const node of Object.values(nodesById)) {
      doc._schema.nodesById[node.id] = node;
    }

    // 插入图形
    for (const graphic of Object.values(graphicsById)) {
      doc._schema.graphicsById[graphic.id] = graphic;
    }

    // 重建索引
    doc._rebuildIndexes();

    return page.id;
  }

  // ==================== 差量补丁 ====================

  /**
   * 生成差量补丁
   * @param {ProjectSchema} oldSchema - 旧 Schema
   * @param {ProjectSchema} newSchema - 新 Schema
   * @returns {Patch}
   */
  generatePatch(oldSchema, newSchema) {
    const ops = [];
    this._diffObject(oldSchema, newSchema, "", ops);

    return {
      ops,
      timestamp: Date.now(),
    };
  }

  /**
   * 应用差量补丁
   * @param {ProjectSchema} schema - 原 Schema
   * @param {Patch} patch - 差量补丁
   * @returns {ProjectSchema} 应用补丁后的 Schema
   */
  applyPatch(schema, patch) {
    const result = JSON.parse(JSON.stringify(schema));

    for (const op of patch.ops) {
      this._applyOp(result, op);
    }

    return result;
  }

  /**
   * 合并多个补丁
   * @param {Patch[]} patches - 补丁列表
   * @returns {Patch}
   */
  mergePatches(patches) {
    const allOps = [];
    for (const patch of patches) {
      allOps.push(...patch.ops);
    }

    // TODO: 优化合并逻辑（移除冗余操作）

    return {
      ops: allOps,
      timestamp: Date.now(),
    };
  }

  // ==================== 私有方法 ====================

  /**
   * 序列化为 JSON 字符串
   * @param {Object} obj
   * @returns {string}
   * @private
   */
  _stringify(obj) {
    return JSON.stringify(obj, null, this._prettyPrint ? 2 : 0);
  }

  /**
   * 解析 JSON 字符串
   * @param {string} json
   * @returns {Object}
   * @private
   */
  _parse(json) {
    try {
      return JSON.parse(json);
    } catch (error) {
      throw new Error(`JSON 解析失败: ${error.message}`);
    }
  }

  /**
   * 递归对比对象
   * @param {*} oldVal
   * @param {*} newVal
   * @param {string} path
   * @param {PatchOp[]} ops
   * @private
   */
  _diffObject(oldVal, newVal, path, ops) {
    // 类型不同或值不同
    if (typeof oldVal !== typeof newVal) {
      if (oldVal === undefined) {
        ops.push({ op: "add", path, value: newVal });
      } else if (newVal === undefined) {
        ops.push({ op: "remove", path });
      } else {
        ops.push({ op: "replace", path, value: newVal });
      }
      return;
    }

    // 基本类型
    if (typeof oldVal !== "object" || oldVal === null) {
      if (oldVal !== newVal) {
        ops.push({ op: "replace", path, value: newVal });
      }
      return;
    }

    // 数组
    if (Array.isArray(oldVal)) {
      if (!Array.isArray(newVal)) {
        ops.push({ op: "replace", path, value: newVal });
        return;
      }

      // 简化处理：如果数组不同，直接替换
      if (JSON.stringify(oldVal) !== JSON.stringify(newVal)) {
        ops.push({ op: "replace", path, value: newVal });
      }
      return;
    }

    // 对象
    const allKeys = new Set([...Object.keys(oldVal), ...Object.keys(newVal)]);
    for (const key of allKeys) {
      const childPath = path ? `${path}.${key}` : key;
      this._diffObject(oldVal[key], newVal[key], childPath, ops);
    }
  }

  /**
   * 应用单个操作
   * @param {Object} obj
   * @param {PatchOp} op
   * @private
   */
  _applyOp(obj, op) {
    const pathParts = op.path.split(".").filter(Boolean);
    const lastKey = pathParts.pop();

    if (!lastKey) {
      // 根路径操作
      if (op.op === "replace") {
        Object.assign(obj, op.value);
      }
      return;
    }

    // 定位到父对象
    let target = obj;
    for (const part of pathParts) {
      if (target[part] === undefined) {
        target[part] = {};
      }
      target = target[part];
    }

    // 执行操作
    switch (op.op) {
      case "add":
      case "replace":
        target[lastKey] = op.value;
        break;
      case "remove":
        delete target[lastKey];
        break;
    }
  }
}

export default Serializer;
