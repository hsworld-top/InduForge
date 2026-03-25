/**
 * 文档 Schema 规范化：工程/页面结构、节点映射、布局修复
 */

import type { ExportedPagePayload } from "@/editor-core/document/Serializer";
import type { ComponentNode, PageNode, ProjectSchema } from "@/editor-core/document/types";
import { createComponentNode } from "@/editor-core/document/factory";
import { Serializer } from "@/editor-core/document/Serializer";
import { createEmptySchema, createPageNode } from "@/editor-core/document/types";

/** API / 序列化过程中的节点快照（弱于 ComponentNode，便于承接 JSON） */
type SchemaNode = Record<string, unknown> & {
  id: string;
  type?: string;
  children?: unknown;
  props?: Record<string, unknown>;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object";
}

/** 规范化后的节点表写入 ProjectSchema 时，仅此一处做 ComponentNode 断言 */
function toComponentNodesMap(nodes: Record<string, SchemaNode>): Record<string, ComponentNode> {
  return nodes as unknown as Record<string, ComponentNode>;
}

/** Serializer 导出页载荷 + 创建页时附带的 entry（与远端 updatePage 约定一致） */
export type PageSchemaCreatePayload = ExportedPagePayload & {
  entry: ProjectSchema["entry"];
};

type ProjectSchemaLike = Pick<ProjectSchema, "pagesById" | "nodesById" | "graphicsById" | "entry">;

interface PagePayloadLike {
  page?: PageNode | Record<string, unknown>;
  nodesById?: unknown;
  graphicsById?: unknown;
  vars?: unknown;
  entry?: unknown;
}

function isProjectSchemaLike(payload: unknown): payload is ProjectSchemaLike {
  return (
    isRecord(payload) &&
    isRecord(payload.pagesById) &&
    isRecord(payload.nodesById) &&
    isRecord(payload.graphicsById) &&
    isRecord(payload.entry)
  );
}

function isPagePayloadLike(payload: unknown): payload is PagePayloadLike {
  return isRecord(payload);
}

function toPageNodeLike(payload: Record<string, unknown>): PageNode {
  return payload as unknown as PageNode;
}

export function normalizePageList(payload: unknown): unknown[] {
  if (payload == null) {
    throw new Error("页面列表缺失：期望 pages 为数组");
  }
  if (!Array.isArray(payload)) {
    throw new TypeError("页面列表格式无效：期望数组（不再兼容 items/list 等历史字段）");
  }
  return payload;
}

export function normalizePageSchema(payload: unknown): unknown {
  if (!isRecord(payload)) return payload;
  if (Object.hasOwn(payload, "schema")) {
    const inner = payload.schema;
    if (inner != null && typeof inner === "object") return inner;
    throw new Error("页面详情 envelope 中 schema 无效或缺失");
  }
  return payload;
}

export function isProjectSchemaPayload(payload: unknown): boolean {
  return isProjectSchemaLike(payload);
}

export function normalizeNodesById(nodesById: unknown): {
  nodesById: Record<string, SchemaNode>;
  idMap: Map<string, string>;
} {
  const idMap = new Map<string, string>();

  if (Array.isArray(nodesById)) {
    const normalized: Record<string, SchemaNode> = {};
    for (const node of nodesById) {
      if (!node || typeof node !== "object" || !(node as SchemaNode).id) {
        throw new Error("nodesById 数组项必须为带 id 的对象");
      }
      const n = node as SchemaNode;
      if (Object.hasOwn(normalized, n.id)) {
        throw new Error(`nodesById 重复 id: ${n.id}`);
      }
      normalized[n.id] = n;
    }
    return { nodesById: normalized, idMap };
  }

  if (nodesById == null || typeof nodesById !== "object") {
    throw new Error("nodesById 必须为对象或数组");
  }

  const normalized: Record<string, SchemaNode> = {};
  for (const [key, node] of Object.entries(nodesById)) {
    if (!node || typeof node !== "object" || !(node as SchemaNode).id) {
      throw new Error(`nodesById 项无效: ${key}`);
    }
    const n = node as SchemaNode;
    if (key !== n.id) {
      throw new Error(`nodesById 键与 node.id 不一致: ${key} !== ${n.id}`);
    }
    normalized[n.id] = n;
  }
  return { nodesById: normalized, idMap };
}

export function resolveRootNodeId(
  page: { rootNodeId?: string } | null | undefined,
  nodesById: Record<string, unknown> | null | undefined,
): string {
  const rid = page?.rootNodeId;
  if (!rid || typeof rid !== "string") {
    throw new Error("页面缺少有效的 rootNodeId");
  }
  if (!nodesById?.[rid]) {
    throw new Error(`rootNodeId 在 nodesById 中不存在: ${rid}`);
  }
  return rid;
}

export function ensureProjectSchemaStructure(
  schema: ProjectSchema | Record<string, unknown> | null | undefined,
): ProjectSchema | Record<string, unknown> | null | undefined {
  if (!isRecord(schema)) return schema;
  const s = schema;
  if (!s.pagesById || typeof s.pagesById !== "object") return schema;

  if (s.nodesById == null || typeof s.nodesById !== "object") {
    throw new Error("工程 Schema 缺少 nodesById");
  }
  if (s.graphicsById == null || typeof s.graphicsById !== "object") {
    throw new Error("工程 Schema 缺少 graphicsById");
  }
  if (s.entry == null || typeof s.entry !== "object") {
    throw new Error("工程 Schema 缺少 entry");
  }

  const { nodesById } = normalizeNodesById(s.nodesById);
  s.nodesById = toComponentNodesMap(nodesById);

  for (const page of Object.values(s.pagesById as Record<string, PageNode | null>)) {
    if (!page) continue;
    page.rootNodeId = resolveRootNodeId(page, nodesById);
  }

  return schema;
}

export function ensurePagePayloadId(
  payload: Record<string, unknown> | null | undefined,
  fallbackPageId: string,
): Record<string, unknown> | null | undefined {
  if (!isRecord(payload)) return payload;
  if (!fallbackPageId) return payload;

  const page = payload.page && isRecord(payload.page) ? payload.page : {};
  page.id = fallbackPageId;

  const { nodesById } = normalizeNodesById(payload.nodesById);
  page.rootNodeId = resolveRootNodeId(page as { rootNodeId?: string }, nodesById);

  payload.page = page;
  payload.nodesById = nodesById;
  return payload;
}

export function resolveProjectSchema(
  payload: unknown,
  projectId: string,
  fallbackPageId: string,
): ProjectSchema {
  const normalized = normalizePageSchema(payload);
  if (isProjectSchemaLike(normalized)) {
    return ensureProjectSchemaStructure(normalized) as ProjectSchema;
  }
  if (isPagePayloadLike(normalized)) {
    const ensured = ensurePagePayloadId(normalized as Record<string, unknown>, fallbackPageId);
    if (!ensured) {
      throw new Error("页面载荷为空：无法解析为工程 Schema");
    }
    return buildSchemaFromPagePayload(ensured, projectId);
  }
  throw new Error("无效的页面载荷：无法解析为工程 Schema");
}

const KNOWN_LAYOUT_TYPES = new Set([
  "FreeContainer",
  "ElLayout",
  "ElLayoutRow",
  "ElCol",
  "VerticalLayout",
  "HorizontalLayout",
]);

const LEGACY_LAYOUT_TYPES = new Set([
  "Container",
  "Row",
  "Col",
  "Elayout",
  "EILayout",
  "ElayoutRow",
  "EILayoutRow",
  "Elcol",
  "EICol",
]);

export function ensurePageRootNodes(
  schema: ProjectSchema | Record<string, unknown> | null | undefined,
): void {
  if (!isRecord(schema)) return;
  const s = schema;
  if (!s.pagesById || !s.nodesById) return;

  const pagesById = s.pagesById as Record<string, PageNode | null>;
  const nodesById = s.nodesById as Record<string, SchemaNode>;

  for (const page of Object.values(pagesById)) {
    if (!page) continue;
    const rootId = page.rootNodeId;
    if (!rootId) {
      throw new Error("页面缺少 rootNodeId");
    }
    const rootNode = nodesById[rootId];
    if (!rootNode) {
      throw new Error(`页面根节点不存在: ${rootId}`);
    }
    if (typeof rootNode.type !== "string" || !rootNode.type) {
      throw new Error("页面根节点类型非法");
    }
    if (!KNOWN_LAYOUT_TYPES.has(rootNode.type)) {
      throw new Error(`页面根类型不在允许列表: ${rootNode.type}`);
    }
  }
}

function assertLayoutStructureStrict(schema: ProjectSchema): void {
  const nodesById = schema.nodesById;
  if (!nodesById) return;

  for (const node of Object.values(nodesById)) {
    if (!node?.type) continue;
    const t = node.type;
    if (LEGACY_LAYOUT_TYPES.has(t)) {
      throw new Error(`不支持的遗留布局类型: ${t}`);
    }
    if (t === "ElLayout") {
      const children = Array.isArray(node.children) ? node.children : [];
      for (const cid of children) {
        const child = nodesById[cid];
        if (!child || child.type !== "ElLayoutRow") {
          throw new Error(`ElLayout ${node.id} 只能包含 ElLayoutRow 子节点`);
        }
      }
      const rows = node.props?.rows;
      if (rows != null && Number(rows) !== children.length) {
        throw new Error(`ElLayout ${node.id} 的 props.rows 与子行数量不一致`);
      }
    }
    if (t === "ElLayoutRow") {
      const children = Array.isArray(node.children) ? node.children : [];
      for (const cid of children) {
        const child = nodesById[cid];
        if (!child || child.type !== "ElCol") {
          throw new Error(`ElLayoutRow ${node.id} 只能包含 ElCol 子节点`);
        }
      }
      const columns = node.props?.columns;
      if (columns != null && Number(columns) !== children.length) {
        throw new Error(`ElLayoutRow ${node.id} 的 props.columns 与 ElCol 子节点数量不一致`);
      }
    }
  }
}

export function normalizeLayoutSchema(
  schema: ProjectSchema | null | undefined,
): ProjectSchema | null | undefined {
  if (!schema || typeof schema !== "object") return schema;
  if (!schema.nodesById) return schema;

  ensurePageRootNodes(schema);
  assertLayoutStructureStrict(schema);
  return schema;
}

export function applyPageNamePath(
  schema: Record<string, unknown> | null | undefined,
  pageId: string,
  name: string,
  path: string,
): Record<string, unknown> | null | undefined {
  if (!isRecord(schema)) return schema;
  if (schema.page) {
    const pg = isRecord(schema.page) ? schema.page : {};
    return {
      ...schema,
      page: {
        ...pg,
        name,
        path,
      },
    };
  }
  const pagesById = schema.pagesById as Record<string, Record<string, unknown>> | undefined;
  if (pagesById && pagesById[pageId]) {
    return {
      ...schema,
      pagesById: {
        ...pagesById,
        [pageId]: {
          ...pagesById[pageId],
          name,
          path,
        },
      },
    };
  }
  return schema;
}

export function createBaseSchema(projectId: string): ProjectSchema {
  const schema = createEmptySchema({
    projectId,
    name: projectId ? `工程 ${projectId}` : "新工程",
  });

  const page = createPageNode({
    name: "首页",
    path: "/",
  });

  const rootNode = createComponentNode("FreeContainer", {
    label: "画布",
    props: {},
    style: {
      width: "100%",
      height: "100%",
    },
  });
  rootNode.id = page.rootNodeId;

  schema.pagesById[page.id] = page;
  schema.nodesById[rootNode.id] = rootNode;
  schema.entry.homePageId = page.id;

  return schema;
}

export function buildSchemaFromPagePayload(
  payload: Record<string, unknown>,
  projectId: string,
): ProjectSchema {
  const schema = createEmptySchema({
    projectId,
    name: projectId ? `工程 ${projectId}` : "新工程",
  });

  const pageSource = isRecord(payload.page) ? payload.page : payload;
  const page = createPageNode(toPageNodeLike(pageSource));
  if (payload.vars && isRecord(payload.vars)) {
    const vars = payload.vars;
    const nextPages = vars.pages && isRecord(vars.pages) ? vars.pages : {};
    schema.vars = {
      ...schema.vars,
      ...vars,
      pages: {
        ...(schema.vars?.pages || {}),
        ...nextPages,
      },
    } as ProjectSchema["vars"];
  }
  if (payload.entry && isRecord(payload.entry)) {
    schema.entry = {
      ...schema.entry,
      ...(payload.entry as Partial<ProjectSchema["entry"]>),
    };
  }
  const { nodesById } = normalizeNodesById(payload.nodesById);
  if (payload.graphicsById == null || typeof payload.graphicsById !== "object") {
    throw new Error("页面载荷缺少 graphicsById 对象");
  }
  const graphicsById = payload.graphicsById as ProjectSchema["graphicsById"];

  page.rootNodeId = resolveRootNodeId(page, nodesById);

  schema.pagesById[page.id] = page;
  schema.nodesById = toComponentNodesMap(nodesById);
  schema.graphicsById = graphicsById;
  schema.entry.homePageId = page.id;

  return schema;
}

export function createPageSchemaPayload(page: {
  id: string;
  name: string;
  path: string;
  projectId?: string;
  projectName?: string;
}): PageSchemaCreatePayload {
  const schema = createEmptySchema({
    projectId: page.projectId || "",
    name: page.projectName || "工程",
  });
  const pageNode = createPageNode({
    id: page.id,
    name: page.name,
    path: page.path,
  });
  const rootNode = createComponentNode("FreeContainer", {
    label: "画布",
    props: {},
    style: {
      width: "100%",
      height: "100%",
    },
  });
  rootNode.id = pageNode.rootNodeId;

  schema.pagesById[pageNode.id] = pageNode;
  schema.nodesById[rootNode.id] = rootNode;
  schema.entry = { ...schema.entry };

  const tempSerializer = new Serializer();
  const tempDoc = tempSerializer.importFromSchema(schema);
  const exported = tempSerializer.exportPage(tempDoc, pageNode.id);
  return { ...exported, entry: schema.entry };
}

export function buildNewPageSchema(pageInfo: { name: string; path: string }): {
  page: PageNode;
  nodesById: Record<string, ComponentNode>;
  graphicsById: Record<string, never>;
} {
  const pageNode = createPageNode({
    name: pageInfo.name,
    path: pageInfo.path,
  });

  const rootNode = createComponentNode("FreeContainer", {
    label: "画布",
    props: {},
    style: {
      width: "100%",
      height: "100%",
    },
  });
  rootNode.id = pageNode.rootNodeId;

  return {
    page: pageNode,
    nodesById: { [rootNode.id]: rootNode },
    graphicsById: {},
  };
}
