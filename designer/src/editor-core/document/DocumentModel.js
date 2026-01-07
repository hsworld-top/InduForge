/**
 * DocumentModel - 规范化文档模型
 * 管理工程 Schema 的规范化存储、CRUD 操作和索引维护
 *
 * 设计原则：
 * - 规范化存储：pagesById + nodesById + graphicsById 扁平结构
 * - ID 引用：父子关系通过 ID 数组表达
 * - 索引维护：支持快速查询
 * - 变更事件：支持订阅变更
 */

import { EventEmitter } from "../utils/EventEmitter.js";
import {
  CURRENT_SCHEMA_VERSION,
  generateId,
  createEmptySchema,
  createPageNode,
  createComponentNode,
} from "./types.js";

/**
 * @typedef {import('./types.js').ProjectSchema} ProjectSchema
 * @typedef {import('./types.js').PageNode} PageNode
 * @typedef {import('./types.js').ComponentNode} ComponentNode
 * @typedef {import('./types.js').GraphicNode} GraphicNode
 * @typedef {import('./types.js').SymbolDef} SymbolDef
 * @typedef {import('./types.js').Change} Change
 * @typedef {import('./types.js').GraphicType} GraphicType
 */

/**
 * 文档模型类
 * 管理工程 Schema 的规范化存储
 */
export class DocumentModel extends EventEmitter {
  /**
   * 创建文档模型
   * @param {ProjectSchema} [schema] - 初始 Schema
   */
  constructor(schema) {
    super();

    // 初始化或使用提供的 Schema
    /** @type {ProjectSchema} */
    this._schema = schema || createEmptySchema();

    // 确保 Schema 结构完整
    this._ensureSchemaStructure();

    // 初始化索引
    /** @type {Map<string, string>} nodeId → parentId */
    this._parentIndex = new Map();

    /** @type {Map<string, Set<string>>} type → nodeIds */
    this._typeIndex = new Map();

    /** @type {Map<string, Set<string>>} datapointPath → nodeIds/graphicIds */
    this._bindingIndex = new Map();

    /** @type {Map<string, string>} graphicId → pageId */
    this._graphicPageIndex = new Map();

    // 构建索引
    this._rebuildIndexes();
  }

  // ==================== 属性访问器 ====================

  /**
   * 获取工程元信息
   * @returns {import('./types.js').ProjectMeta}
   */
  get project() {
    return this._schema.project;
  }

  /**
   * 获取安全声明
   * @returns {import('./types.js').SecurityDecl}
   */
  get securityDecl() {
    return this._schema.securityDecl;
  }

  /**
   * 获取入口配置
   * @returns {import('./types.js').EntryConfig}
   */
  get entry() {
    return this._schema.entry;
  }

  /**
   * 获取数据提供者配置
   * @returns {Record<string, import('./types.js').DataProvider>}
   */
  get dataProviders() {
    return this._schema.dataProviders;
  }

  /**
   * 获取变量配置
   * @returns {import('./types.js').VarsConfig}
   */
  get vars() {
    return this._schema.vars;
  }

  /**
   * 获取资源引用
   * @returns {Record<string, import('./types.js').AssetRef>}
   */
  get assetsById() {
    return this._schema.assetsById;
  }

  /**
   * 获取页面节点映射
   * @returns {Record<string, PageNode>}
   */
  get pagesById() {
    return this._schema.pagesById;
  }

  /**
   * 获取组件节点映射
   * @returns {Record<string, ComponentNode>}
   */
  get nodesById() {
    return this._schema.nodesById;
  }

  /**
   * 获取图形节点映射
   * @returns {Record<string, GraphicNode>}
   */
  get graphicsById() {
    return this._schema.graphicsById;
  }

  /**
   * 获取符号库映射
   * @returns {Record<string, SymbolDef>}
   */
  get symbolsById() {
    return this._schema.symbolsById;
  }

  /**
   * 获取完整 Schema
   * @returns {ProjectSchema}
   */
  get schema() {
    return this._schema;
  }

  /**
   * 更新入口配置（内部方法）
   * @param {Partial<import('./types.js').EntryConfig>} patch - 更新内容
   */
  _updateEntry(patch) {
    const oldValue = { ...this._schema.entry };
    this._schema.entry = { ...this._schema.entry, ...patch };
    this._emitChange({
      type: "update",
      target: "entry",
      oldValue,
      newValue: this._schema.entry,
    });
  }

  // ==================== 页面操作 ====================

  /**
   * 获取页面
   * @param {string} id - 页面 ID
   * @returns {PageNode | null}
   */
  getPage(id) {
    return this._schema.pagesById[id] || null;
  }

  /**
   * 获取所有页面
   * @returns {PageNode[]}
   */
  getAllPages() {
    return Object.values(this._schema.pagesById);
  }

  /**
   * 获取指定路由的所有页面视图
   * @param {string} path - 路由路径
   * @returns {PageNode[]}
   */
  getPagesByPath(path) {
    return this.getAllPages().filter((page) => page.path === path);
  }

  /**
   * 获取指定目标端的页面
   * @param {import('./types.js').UITarget} target - 目标端
   * @returns {PageNode[]}
   */
  getPagesByTarget(target) {
    return this.getAllPages().filter((page) => page.target === target);
  }

  // ==================== 组件节点操作 ====================

  /**
   * 获取节点
   * @param {string} id - 节点 ID
   * @returns {ComponentNode | null}
   */
  getNode(id) {
    return this._schema.nodesById[id] || null;
  }

  /**
   * 获取父节点
   * @param {string} nodeId - 节点 ID
   * @returns {ComponentNode | null}
   */
  getParent(nodeId) {
    const parentId = this._parentIndex.get(nodeId);
    if (!parentId) return null;
    return this.getNode(parentId);
  }

  /**
   * 获取节点所属的页面 ID
   * @param {string} nodeId - 节点 ID
   * @returns {string | null}
   */
  getNodePageId(nodeId) {
    // 向上查找根节点
    let currentId = nodeId;
    const visited = new Set();

    while (currentId && !visited.has(currentId)) {
      visited.add(currentId);
      const parentId = this._parentIndex.get(currentId);
      if (!parentId) {
        // 当前节点是根节点，查找它属于哪个页面
        for (const page of Object.values(this._schema.pagesById)) {
          if (page.rootNodeId === currentId) {
            return page.id;
          }
        }
        return null;
      }
      currentId = parentId;
    }
    return null;
  }

  /**
   * 获取子节点
   * @param {string} nodeId - 节点 ID
   * @returns {ComponentNode[]}
   */
  getChildren(nodeId) {
    const node = this.getNode(nodeId);
    if (!node || !node.children) return [];
    return node.children
      .map((childId) => this.getNode(childId))
      .filter((child) => child !== null);
  }

  /**
   * 获取所有祖先节点
   * @param {string} nodeId - 节点 ID
   * @returns {ComponentNode[]}
   */
  getAncestors(nodeId) {
    const ancestors = [];
    let currentId = this._parentIndex.get(nodeId);
    const visited = new Set();

    while (currentId && !visited.has(currentId)) {
      visited.add(currentId);
      const node = this.getNode(currentId);
      if (node) {
        ancestors.push(node);
      }
      currentId = this._parentIndex.get(currentId);
    }

    return ancestors;
  }

  /**
   * 获取所有后代节点
   * @param {string} nodeId - 节点 ID
   * @returns {ComponentNode[]}
   */
  getDescendants(nodeId) {
    const descendants = [];
    const node = this.getNode(nodeId);
    if (!node) return descendants;

    const stack = [...(node.children || [])];
    const visited = new Set();

    while (stack.length > 0) {
      const childId = stack.pop();
      if (visited.has(childId)) continue;
      visited.add(childId);

      const child = this.getNode(childId);
      if (child) {
        descendants.push(child);
        if (child.children) {
          stack.push(...child.children);
        }
      }
    }

    return descendants;
  }

  /**
   * 按类型查找节点
   * @param {string} type - 组件类型
   * @returns {ComponentNode[]}
   */
  findNodesByType(type) {
    const nodeIds = this._typeIndex.get(type);
    if (!nodeIds) return [];
    return Array.from(nodeIds)
      .map((id) => this.getNode(id))
      .filter((node) => node !== null);
  }

  /**
   * 按绑定路径查找节点
   * @param {string} datapointPath - 数据点路径
   * @returns {ComponentNode[]}
   */
  findNodesByBinding(datapointPath) {
    const elementIds = this._bindingIndex.get(datapointPath);
    if (!elementIds) return [];
    return Array.from(elementIds)
      .map((id) => this.getNode(id))
      .filter((node) => node !== null);
  }

  // ==================== Canvas 图形操作 ====================

  /**
   * 获取图形
   * @param {string} id - 图形 ID
   * @returns {GraphicNode | null}
   */
  getGraphic(id) {
    return this._schema.graphicsById[id] || null;
  }

  /**
   * 获取图形所属页面 ID
   * @param {string} graphicId - 图形 ID
   * @returns {string | null}
   */
  getGraphicPageId(graphicId) {
    return this._graphicPageIndex.get(graphicId) || null;
  }

  /**
   * 获取页面的所有图形
   * @param {string} pageId - 页面 ID
   * @returns {GraphicNode[]}
   */
  getGraphicsByPage(pageId) {
    const page = this.getPage(pageId);
    if (!page || !page.graphicsIds) return [];
    return page.graphicsIds
      .map((id) => this.getGraphic(id))
      .filter((graphic) => graphic !== null)
      .sort((a, b) => (a.z || 0) - (b.z || 0));
  }

  /**
   * 按类型获取图形
   * @param {GraphicType} type - 图形类型
   * @returns {GraphicNode[]}
   */
  getGraphicsByType(type) {
    return Object.values(this._schema.graphicsById).filter(
      (graphic) => graphic.type === type
    );
  }

  /**
   * 按绑定路径查找图形
   * @param {string} datapointPath - 数据点路径
   * @returns {GraphicNode[]}
   */
  findGraphicsByBinding(datapointPath) {
    const elementIds = this._bindingIndex.get(datapointPath);
    if (!elementIds) return [];
    return Array.from(elementIds)
      .map((id) => this.getGraphic(id))
      .filter((graphic) => graphic !== null);
  }

  // ==================== 符号库操作 ====================

  /**
   * 获取符号
   * @param {string} id - 符号 ID
   * @returns {SymbolDef | null}
   */
  getSymbol(id) {
    return this._schema.symbolsById[id] || null;
  }

  /**
   * 获取所有符号
   * @returns {SymbolDef[]}
   */
  getAllSymbols() {
    return Object.values(this._schema.symbolsById);
  }

  /**
   * 按分类获取符号
   * @param {string} category - 符号分类
   * @returns {SymbolDef[]}
   */
  getSymbolsByCategory(category) {
    return this.getAllSymbols().filter(
      (symbol) => symbol.category === category
    );
  }

  // ==================== 混合查询 ====================

  /**
   * 获取元素（节点或图形）
   * @param {string} id - 元素 ID
   * @returns {ComponentNode | GraphicNode | null}
   */
  getElement(id) {
    return this.getNode(id) || this.getGraphic(id);
  }

  /**
   * 按绑定路径查找所有元素（节点 + 图形）
   * @param {string} datapointPath - 数据点路径
   * @returns {(ComponentNode | GraphicNode)[]}
   */
  findElementsByBinding(datapointPath) {
    const elementIds = this._bindingIndex.get(datapointPath);
    if (!elementIds) return [];
    return Array.from(elementIds)
      .map((id) => this.getElement(id))
      .filter((element) => element !== null);
  }

  // ==================== 内部变更操作（由 Command 调用）====================

  /**
   * 插入页面（内部方法）
   * @param {PageNode} page - 页面节点
   */
  _insertPage(page) {
    this._schema.pagesById[page.id] = page;

    // 如果页面有图形，更新索引
    if (page.graphicsIds) {
      for (const graphicId of page.graphicsIds) {
        this._graphicPageIndex.set(graphicId, page.id);
      }
    }

    // 触发变更事件
    this._emitChange({
      type: "insert",
      target: "page",
      id: page.id,
      newValue: page,
    });
  }

  /**
   * 删除页面（内部方法）
   * @param {string} pageId - 页面 ID
   * @returns {PageNode | null} 被删除的页面
   */
  _removePage(pageId) {
    const page = this._schema.pagesById[pageId];
    if (!page) return null;

    // 清理图形索引
    if (page.graphicsIds) {
      for (const graphicId of page.graphicsIds) {
        this._graphicPageIndex.delete(graphicId);
      }
    }

    delete this._schema.pagesById[pageId];

    // 触发变更事件
    this._emitChange({
      type: "remove",
      target: "page",
      id: pageId,
      oldValue: page,
    });

    return page;
  }

  /**
   * 更新页面（内部方法）
   * @param {string} pageId - 页面 ID
   * @param {Partial<PageNode>} patch - 更新内容
   */
  _updatePage(pageId, patch) {
    const page = this._schema.pagesById[pageId];
    if (!page) return;

    const oldValue = { ...page };

    // 更新图形索引
    if (patch.graphicsIds) {
      // 清除旧索引
      if (page.graphicsIds) {
        for (const graphicId of page.graphicsIds) {
          this._graphicPageIndex.delete(graphicId);
        }
      }
      // 建立新索引
      for (const graphicId of patch.graphicsIds) {
        this._graphicPageIndex.set(graphicId, pageId);
      }
    }

    Object.assign(page, patch);

    // 触发变更事件
    this._emitChange({
      type: "update",
      target: "page",
      id: pageId,
      oldValue,
      newValue: page,
    });
  }

  /**
   * 插入节点（内部方法）
   * @param {string} parentId - 父节点 ID
   * @param {number} index - 插入位置
   * @param {ComponentNode} node - 节点
   */
  _insertNode(parentId, index, node) {
    // 添加到 nodesById
    this._schema.nodesById[node.id] = node;

    // 更新父节点的 children
    const parent = this.getNode(parentId);
    if (parent) {
      if (!parent.children) {
        parent.children = [];
      }
      const insertIndex = Math.min(Math.max(0, index), parent.children.length);
      parent.children.splice(insertIndex, 0, node.id);
    }

    // 更新索引
    this._parentIndex.set(node.id, parentId);
    this._addToTypeIndex(node);
    this._addToBindingIndex(node);

    // 递归添加子节点索引
    if (node.children) {
      for (const childId of node.children) {
        const child = this._schema.nodesById[childId];
        if (child) {
          this._parentIndex.set(childId, node.id);
          this._addToTypeIndex(child);
          this._addToBindingIndex(child);
        }
      }
    }

    // 触发变更事件
    this._emitChange({
      type: "insert",
      target: "node",
      id: node.id,
      parentId,
      index,
      newValue: node,
    });
  }

  /**
   * 删除节点（内部方法）
   * @param {string} nodeId - 节点 ID
   * @returns {ComponentNode | null} 被删除的节点
   */
  _removeNode(nodeId) {
    const node = this._schema.nodesById[nodeId];
    if (!node) return null;

    // 获取父节点并从 children 中移除
    const parentId = this._parentIndex.get(nodeId);
    if (parentId) {
      const parent = this.getNode(parentId);
      if (parent && parent.children) {
        const index = parent.children.indexOf(nodeId);
        if (index !== -1) {
          parent.children.splice(index, 1);
        }
      }
    }

    // 递归删除子节点
    if (node.children) {
      for (const childId of [...node.children]) {
        this._removeNode(childId);
      }
    }

    // 清理索引
    this._parentIndex.delete(nodeId);
    this._removeFromTypeIndex(node);
    this._removeFromBindingIndex(node);

    // 从 nodesById 中删除
    delete this._schema.nodesById[nodeId];

    // 触发变更事件
    this._emitChange({
      type: "remove",
      target: "node",
      id: nodeId,
      parentId,
      oldValue: node,
    });

    return node;
  }

  /**
   * 更新节点（内部方法）
   * @param {string} nodeId - 节点 ID
   * @param {Partial<ComponentNode>} patch - 更新内容
   */
  _updateNode(nodeId, patch) {
    const node = this._schema.nodesById[nodeId];
    if (!node) return;

    const oldValue = { ...node };

    // 如果更新了 bindings，需要更新绑定索引
    if (patch.bindings) {
      this._removeFromBindingIndex(node);
    }

    // 如果更新了 type，需要更新类型索引
    if (patch.type && patch.type !== node.type) {
      this._removeFromTypeIndex(node);
    }

    // 应用更新
    Object.assign(node, patch);

    // 重建索引
    if (patch.bindings) {
      this._addToBindingIndex(node);
    }
    if (patch.type) {
      this._addToTypeIndex(node);
    }

    // 触发变更事件
    this._emitChange({
      type: "update",
      target: "node",
      id: nodeId,
      oldValue,
      newValue: node,
    });
  }

  /**
   * 移动节点（内部方法）
   * @param {string} nodeId - 节点 ID
   * @param {string} newParentId - 新父节点 ID
   * @param {number} newIndex - 新位置索引
   */
  _moveNode(nodeId, newParentId, newIndex) {
    const node = this.getNode(nodeId);
    if (!node) return;

    const oldParentId = this._parentIndex.get(nodeId);

    // 从旧父节点移除
    if (oldParentId) {
      const oldParent = this.getNode(oldParentId);
      if (oldParent && oldParent.children) {
        const index = oldParent.children.indexOf(nodeId);
        if (index !== -1) {
          oldParent.children.splice(index, 1);
        }
      }
    }

    // 添加到新父节点
    const newParent = this.getNode(newParentId);
    if (newParent) {
      if (!newParent.children) {
        newParent.children = [];
      }
      const insertIndex = Math.min(
        Math.max(0, newIndex),
        newParent.children.length
      );
      newParent.children.splice(insertIndex, 0, nodeId);
    }

    // 更新父索引
    this._parentIndex.set(nodeId, newParentId);

    // 触发变更事件
    this._emitChange({
      type: "move",
      target: "node",
      id: nodeId,
      oldValue: { parentId: oldParentId },
      newValue: { parentId: newParentId, index: newIndex },
    });
  }

  /**
   * 插入图形（内部方法）
   * @param {string} pageId - 页面 ID
   * @param {GraphicNode} graphic - 图形节点
   */
  _insertGraphic(pageId, graphic) {
    // 添加到 graphicsById
    this._schema.graphicsById[graphic.id] = graphic;

    // 更新页面的 graphicsIds
    const page = this.getPage(pageId);
    if (page) {
      if (!page.graphicsIds) {
        page.graphicsIds = [];
      }
      page.graphicsIds.push(graphic.id);
    }

    // 更新索引
    this._graphicPageIndex.set(graphic.id, pageId);
    this._addToBindingIndex(graphic);

    // 触发变更事件
    this._emitChange({
      type: "insert",
      target: "graphic",
      id: graphic.id,
      parentId: pageId,
      newValue: graphic,
    });
  }

  /**
   * 删除图形（内部方法）
   * @param {string} graphicId - 图形 ID
   * @returns {GraphicNode | null} 被删除的图形
   */
  _removeGraphic(graphicId) {
    const graphic = this._schema.graphicsById[graphicId];
    if (!graphic) return null;

    // 获取所属页面
    const pageId = this._graphicPageIndex.get(graphicId);
    if (pageId) {
      const page = this.getPage(pageId);
      if (page && page.graphicsIds) {
        const index = page.graphicsIds.indexOf(graphicId);
        if (index !== -1) {
          page.graphicsIds.splice(index, 1);
        }
      }
    }

    // 清理索引
    this._graphicPageIndex.delete(graphicId);
    this._removeFromBindingIndex(graphic);

    // 从 graphicsById 中删除
    delete this._schema.graphicsById[graphicId];

    // 触发变更事件
    this._emitChange({
      type: "remove",
      target: "graphic",
      id: graphicId,
      parentId: pageId,
      oldValue: graphic,
    });

    return graphic;
  }

  /**
   * 更新图形（内部方法）
   * @param {string} graphicId - 图形 ID
   * @param {Partial<GraphicNode>} patch - 更新内容
   */
  _updateGraphic(graphicId, patch) {
    const graphic = this._schema.graphicsById[graphicId];
    if (!graphic) return;

    const oldValue = { ...graphic };

    // 如果更新了 bindings，需要更新绑定索引
    if (patch.bindings) {
      this._removeFromBindingIndex(graphic);
    }

    // 应用更新
    Object.assign(graphic, patch);

    // 重建绑定索引
    if (patch.bindings) {
      this._addToBindingIndex(graphic);
    }

    // 触发变更事件
    this._emitChange({
      type: "update",
      target: "graphic",
      id: graphicId,
      oldValue,
      newValue: graphic,
    });
  }

  /**
   * 插入符号（内部方法）
   * @param {SymbolDef} symbol - 符号定义
   */
  _insertSymbol(symbol) {
    this._schema.symbolsById[symbol.id] = symbol;

    // 触发变更事件
    this._emitChange({
      type: "insert",
      target: "symbol",
      id: symbol.id,
      newValue: symbol,
    });
  }

  /**
   * 删除符号（内部方法）
   * @param {string} symbolId - 符号 ID
   * @returns {SymbolDef | null} 被删除的符号
   */
  _removeSymbol(symbolId) {
    const symbol = this._schema.symbolsById[symbolId];
    if (!symbol) return null;

    delete this._schema.symbolsById[symbolId];

    // 触发变更事件
    this._emitChange({
      type: "remove",
      target: "symbol",
      id: symbolId,
      oldValue: symbol,
    });

    return symbol;
  }

  // ==================== 索引维护 ====================

  /**
   * 确保 Schema 结构完整
   * @private
   */
  _ensureSchemaStructure() {
    const s = this._schema;
    if (!s.pagesById) s.pagesById = {};
    if (!s.nodesById) s.nodesById = {};
    if (!s.graphicsById) s.graphicsById = {};
    if (!s.symbolsById) s.symbolsById = {};
    if (!s.assetsById) s.assetsById = {};
    if (!s.dataProviders) s.dataProviders = {};
    if (!s.vars) s.vars = { global: {}, pages: {} };
  }

  /**
   * 重建所有索引
   * @private
   */
  _rebuildIndexes() {
    this._parentIndex.clear();
    this._typeIndex.clear();
    this._bindingIndex.clear();
    this._graphicPageIndex.clear();

    // 构建节点索引
    for (const node of Object.values(this._schema.nodesById)) {
      this._addToTypeIndex(node);
      this._addToBindingIndex(node);

      // 构建父子关系索引
      if (node.children) {
        for (const childId of node.children) {
          this._parentIndex.set(childId, node.id);
        }
      }
    }

    // 构建图形页面索引
    for (const page of Object.values(this._schema.pagesById)) {
      if (page.graphicsIds) {
        for (const graphicId of page.graphicsIds) {
          this._graphicPageIndex.set(graphicId, page.id);
        }
      }
    }

    // 构建图形绑定索引
    for (const graphic of Object.values(this._schema.graphicsById)) {
      this._addToBindingIndex(graphic);
    }
  }

  /**
   * 添加到类型索引
   * @param {ComponentNode} node - 节点
   * @private
   */
  _addToTypeIndex(node) {
    if (!this._typeIndex.has(node.type)) {
      this._typeIndex.set(node.type, new Set());
    }
    this._typeIndex.get(node.type).add(node.id);
  }

  /**
   * 从类型索引移除
   * @param {ComponentNode} node - 节点
   * @private
   */
  _removeFromTypeIndex(node) {
    const typeSet = this._typeIndex.get(node.type);
    if (typeSet) {
      typeSet.delete(node.id);
      if (typeSet.size === 0) {
        this._typeIndex.delete(node.type);
      }
    }
  }

  /**
   * 添加到绑定索引
   * @param {ComponentNode | GraphicNode} element - 元素
   * @private
   */
  _addToBindingIndex(element) {
    if (!element.bindings) return;

    for (const binding of Object.values(element.bindings)) {
      if (binding && binding.kind === "datapoint" && binding.path) {
        if (!this._bindingIndex.has(binding.path)) {
          this._bindingIndex.set(binding.path, new Set());
        }
        this._bindingIndex.get(binding.path).add(element.id);
      }
    }
  }

  /**
   * 从绑定索引移除
   * @param {ComponentNode | GraphicNode} element - 元素
   * @private
   */
  _removeFromBindingIndex(element) {
    if (!element.bindings) return;

    for (const binding of Object.values(element.bindings)) {
      if (binding && binding.kind === "datapoint" && binding.path) {
        const bindingSet = this._bindingIndex.get(binding.path);
        if (bindingSet) {
          bindingSet.delete(element.id);
          if (bindingSet.size === 0) {
            this._bindingIndex.delete(binding.path);
          }
        }
      }
    }
  }

  /**
   * 触发变更事件
   * @param {Change} change - 变更记录
   * @private
   */
  _emitChange(change) {
    this.emit("change", [change]);
  }

  // ==================== 序列化 ====================

  /**
   * 导出为 JSON 字符串
   * @returns {string}
   */
  toJSON() {
    return JSON.stringify(this._schema, null, 2);
  }

  /**
   * 从 JSON 字符串创建 DocumentModel
   * @param {string} json - JSON 字符串
   * @returns {DocumentModel}
   */
  static fromJSON(json) {
    const schema = JSON.parse(json);
    return new DocumentModel(schema);
  }

  /**
   * 克隆文档模型
   * @returns {DocumentModel}
   */
  clone() {
    const schemaClone = JSON.parse(JSON.stringify(this._schema));
    return new DocumentModel(schemaClone);
  }
}

export default DocumentModel;
