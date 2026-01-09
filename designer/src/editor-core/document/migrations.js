/**
 * Schema 版本迁移
 * 支持从旧版本 Schema 迁移到新版本
 */

import { CURRENT_SCHEMA_VERSION, generateId } from "./types.js";

/**
 * 迁移函数映射
 * key: 源版本号
 * value: 迁移到下一版本的函数
 */
const migrations = {
  /**
   * 从 v1 迁移到 v2
   * v1: 嵌套 components 数组结构
   * v2: 规范化 pagesById + nodesById 结构
   */
  1: (schema) => {
    console.log("迁移 Schema: v1 -> v2");

    // 创建 v2 结构
    const v2Schema = {
      schemaVersion: 2,
      project: {
        projectId: schema.meta?.id || generateId("proj_"),
        name: schema.meta?.name || "迁移工程",
        uiTargets: ["pc"],
        createdAt: Date.now(),
        updatedAt: Date.now(),
      },
      securityDecl: {
        roles: schema.permissions?.roles || ["admin", "operator", "viewer"],
        mode: "nodeLocalAuth",
      },
      entry: {
        homePageId: "",
      },
      dataProviders: {},
      vars: {
        global: {},
        pages: {},
      },
      assetsById: {},
      pagesById: {},
      nodesById: {},
      graphicsById: {},
      symbolsById: {},
    };

    // 迁移变量
    if (schema.variables && typeof schema.variables === "object") {
      v2Schema.vars.global = migrateVariables(schema.variables);
    }
    if (
      schema.projectVariables &&
      typeof schema.projectVariables === "object"
    ) {
      Object.assign(
        v2Schema.vars.global,
        migrateVariables(schema.projectVariables)
      );
    }

    // 如果是单页面 Schema（旧版 page schema）
    if (schema.components && Array.isArray(schema.components)) {
      const pageId = schema.meta?.id || generateId("page_");
      const rootNodeId = generateId("node_root_");

      // 创建页面
      v2Schema.pagesById[pageId] = {
        id: pageId,
        name: schema.meta?.name || "页面",
        path: `/${pageId}`,
        target: "pc",
        logicalId: generateId("logic_"),
        isDefaultTarget: true,
        rootNodeId,
        graphicsIds: [],
        config: {
          width: schema.config?.width || 1920,
          height: schema.config?.height || 1080,
          fitMode: schema.config?.scaleMode || "contain",
          background: {
            kind: "color",
            value: schema.config?.backgroundColor || "#ffffff",
          },
        },
      };

      // 设置首页
      v2Schema.entry.homePageId = pageId;

      // 页面级变量
      if (schema.variables) {
        v2Schema.vars.pages[pageId] = migrateVariables(schema.variables);
      }

      // 创建根节点
      v2Schema.nodesById[rootNodeId] = {
        id: rootNodeId,
        type: "FlexContainer",
        label: "根容器",
        props: { direction: "column" },
        style: {
          width: "100%",
          height: "100%",
        },
        layoutItem: null,
        bindings: {},
        permissions: {},
        events: {},
        children: [],
      };

      // 迁移组件
      for (const component of schema.components) {
        migrateComponent(component, rootNodeId, v2Schema.nodesById);
        v2Schema.nodesById[rootNodeId].children.push(
          component.id || generateId("node_")
        );
      }
    }

    return v2Schema;
  },
};

/**
 * 迁移变量定义
 * @param {Object} variables - 旧版变量对象
 * @returns {Record<string, import('./types.js').VarDef>}
 */
function migrateVariables(variables) {
  const result = {};

  for (const [name, value] of Object.entries(variables)) {
    // 推断变量类型
    let type = "string";
    if (typeof value === "number") type = "number";
    else if (typeof value === "boolean") type = "boolean";
    else if (Array.isArray(value)) type = "array";
    else if (typeof value === "object" && value !== null) type = "object";

    result[name] = {
      type,
      default: value,
    };
  }

  return result;
}

/**
 * 迁移组件
 * @param {Object} component - 旧版组件
 * @param {string} parentId - 父节点 ID
 * @param {Record<string, import('./types.js').ComponentNode>} nodesById - 节点映射
 */
function migrateComponent(component, parentId, nodesById) {
  const nodeId = component.id || generateId("node_");

  // 创建节点
  /** @type {import('./types.js').ComponentNode} */
  const node = {
    id: nodeId,
    type: component.type || "Unknown",
    label: component.label || component.name || component.type,
    props: { ...component.props } || {},
    style: { ...component.style } || {},
    layoutItem: migrateLayoutItem(component),
    bindings: migrateBindings(component.bindings),
    permissions: component.permissions || {},
    events: migrateEvents(component.events),
    animations: component.animations || [],
    children: [],
    locked: component.locked || false,
    visible: component.visible !== false,
  };

  nodesById[nodeId] = node;

  // 递归迁移子组件
  if (component.children && Array.isArray(component.children)) {
    for (const child of component.children) {
      const childId = child.id || generateId("node_");
      migrateComponent({ ...child, id: childId }, nodeId, nodesById);
      node.children.push(childId);
    }
  }
}

/**
 * 迁移布局配置
 * @param {Object} component - 组件
 * @returns {import('./types.js').LayoutItem | null}
 */
function migrateLayoutItem(component) {
  const style = component.style || {};

  // 检查是否有绝对定位
  if (
    style.position === "absolute" ||
    style.left !== undefined ||
    style.top !== undefined
  ) {
    return {
      free: {
        mode: "abs",
        abs: {
          x: parseFloat(style.left) || 0,
          y: parseFloat(style.top) || 0,
          w: parseFloat(style.width) || 100,
          h: parseFloat(style.height) || 100,
          z: style.zIndex || 0,
        },
      },
    };
  }

  // 检查 flex 配置
  if (style.flex || style.flexGrow || style.flexShrink) {
    return {
      flex: {
        grow: style.flexGrow || 0,
        shrink: style.flexShrink || 1,
        basis: style.flexBasis || "auto",
        alignSelf: style.alignSelf,
      },
    };
  }

  return null;
}

/**
 * 迁移绑定配置
 * @param {Object} bindings - 旧版绑定
 * @returns {Record<string, import('./types.js').Binding>}
 */
function migrateBindings(bindings) {
  if (!bindings || typeof bindings !== "object") return {};

  const result = {};

  for (const [key, binding] of Object.entries(bindings)) {
    if (!binding) continue;

    // 如果已经是新格式
    if (binding.kind) {
      result[key] = binding;
      continue;
    }

    // 旧格式：{ source: 'dataSource', path: 'xxx' }
    if (binding.source && binding.path) {
      result[key] = {
        kind: "datapoint",
        provider: "dc_main",
        datapointId: generateId("dp_"),
        path: binding.path,
        transform: binding.transform || [],
        fallback: binding.default,
      };
    }
    // 旧格式：表达式字符串
    else if (typeof binding === "string" && binding.includes("{{")) {
      result[key] = {
        kind: "expr",
        expr: binding,
      };
    }
    // 旧格式：变量引用
    else if (binding.variable) {
      result[key] = {
        kind: "var",
        scope: binding.scope || "page",
        name: binding.variable,
      };
    }
  }

  return result;
}

/**
 * 迁移事件配置
 * @param {Object} events - 旧版事件
 * @returns {Record<string, import('./types.js').Action[]>}
 */
function migrateEvents(events) {
  if (!events || typeof events !== "object") return {};

  const result = {};

  for (const [eventName, handlers] of Object.entries(events)) {
    if (!handlers) continue;

    // 如果是字符串（脚本），转换为动作
    if (typeof handlers === "string") {
      result[eventName] = [
        {
          type: "script",
          config: { code: handlers },
        },
      ];
      continue;
    }

    // 如果是数组，保持不变
    if (Array.isArray(handlers)) {
      result[eventName] = handlers;
      continue;
    }

    // 如果是对象（单个动作），包装为数组
    result[eventName] = [handlers];
  }

  return result;
}

/**
 * 执行 Schema 迁移
 * @param {Object} schema - 原始 Schema
 * @returns {import('./types.js').ProjectSchema} 迁移后的 Schema
 */
export function migrate(schema) {
  if (!schema) {
    throw new Error("Schema 不能为空");
  }

  let currentVersion = schema.schemaVersion ?? schema.version ?? 1;
  let currentSchema = schema;

  // 逐版本迁移
  while (currentVersion < CURRENT_SCHEMA_VERSION) {
    const migrateFn = migrations[currentVersion];
    if (!migrateFn) {
      throw new Error(`不支持从版本 ${currentVersion} 迁移`);
    }

    currentSchema = migrateFn(currentSchema);
    currentVersion++;
  }

  return currentSchema;
}

/**
 * 检查 Schema 版本
 * @param {Object} schema - Schema
 * @returns {number} 版本号
 */
export function getSchemaVersion(schema) {
  return schema.schemaVersion ?? schema.version ?? 1;
}

/**
 * 检查是否需要迁移
 * @param {Object} schema - Schema
 * @returns {boolean}
 */
export function needsMigration(schema) {
  // 版本迁移已禁用，所有 schema 当作初版处理
  return false;
}

export default {
  migrate,
  getSchemaVersion,
  needsMigration,
  CURRENT_SCHEMA_VERSION,
};
