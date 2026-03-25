/**
 * 创建默认首页并同步远端（从 editor-store 拆出）
 */

import type { Ref } from "vue";
import type { EntryConfigFromApi, PagesRefreshResult } from "./pages-sync-types";
import type { ProjectSchema } from "@/editor-core/document/types";
import { unwrapApiData } from "@/types/api";

export interface HomePageProjectApi {
  /** 与 services projectApi 对齐；body 形状由调用方保证 */
  createPage: (pid: string, payload: any) => Promise<unknown>;
  updatePage: (pid: string, pageId: string, payload: unknown) => Promise<unknown>;
  updateEntryConfig: (pid: string, payload: unknown) => Promise<unknown>;
}

export interface CreateHomePageContext {
  entryConfig: Ref<EntryConfigFromApi>;
  currentPageId: Ref<string>;
  initEditor: (schema: ProjectSchema) => void;
  createBaseSchema: (projectId: string) => ProjectSchema;
  projectApi: HomePageProjectApi;
}

export async function createHomePageForStore(
  ctx: CreateHomePageContext,
  pid: string,
  refreshPages: () => Promise<PagesRefreshResult>,
): Promise<{ ok: true; pageId: string } | { ok: false; error: Error }> {
  try {
    const result = await ctx.projectApi.createPage(pid, {
      name: "首页",
      type: "page",
      parentId: null,
    });

    const created = unwrapApiData(result);
    if (!created || typeof created !== "object") {
      throw new Error("创建首页失败：API 返回异常");
    }

    const cr = created as { id?: string; page?: { id?: string } };
    const pageId = cr.id ?? cr.page?.id;
    if (!pageId) {
      throw new Error("创建首页失败：未获取到页面ID");
    }

    const schema = ctx.createBaseSchema(pid);
    const pageNode = Object.values(schema.pagesById)[0];
    if (!pageNode) {
      throw new Error("创建首页失败：无法获取页面节点");
    }

    const rootNode = schema.nodesById[pageNode.rootNodeId];

    delete schema.pagesById[pageNode.id];
    pageNode.id = pageId;
    schema.pagesById[pageId] = pageNode;
    schema.entry.homePageId = pageId;

    if (rootNode) {
      pageNode.rootNodeId = rootNode.id;
    }

    const pagePayload = {
      page: pageNode,
      nodesById: rootNode ? { [rootNode.id]: rootNode } : {},
      graphicsById: {},
    };
    await ctx.projectApi.updatePage(pid, pageId, pagePayload);

    const newEntryConfig = { homePageId: pageId };
    await ctx.projectApi.updateEntryConfig(pid, newEntryConfig);
    ctx.entryConfig.value = newEntryConfig;

    schema.pagesById = { [pageId]: pageNode };
    ctx.initEditor(schema);
    ctx.currentPageId.value = pageId;

    await refreshPages();
    return { ok: true, pageId };
  } catch (err) {
    console.error("创建首页失败:", err);
    return {
      ok: false,
      error: err instanceof Error ? err : new Error("创建首页失败"),
    };
  }
}
