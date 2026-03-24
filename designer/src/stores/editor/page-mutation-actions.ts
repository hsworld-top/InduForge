/**
 * 页面删除 / 移动 / 重命名（从 editor-store 拆出）
 */

import type { Ref, ShallowRef } from "vue";
import type { DocumentModel } from "@/editor-core/document/DocumentModel";
import type { History } from "@/editor-core/commands/History";
import { UpdatePageCommand } from "@/editor-core/commands/pageCommands";
import type { ProjectSchema } from "@/editor-core/document/types";
import { mapPagesAfterRename, type PageListEntry } from "./page-crud-actions";
import type { PagesRefreshResult } from "./pages-sync-types";

export type PageMutationProjectApi = {
  deletePage: (
    pid: string,
    pageId: string,
    mode?: "single" | "folder-only" | "cascade",
  ) => Promise<unknown>;
  movePageToGroup: (
    pid: string,
    pageId: string,
    targetGroupId: string | null,
    path?: string,
  ) => Promise<unknown>;
  renamePage: (
    pid: string,
    pageId: string,
    name: string,
    path?: string,
  ) => Promise<unknown>;
};

export type PageMutationStoreContext = {
  projectId: Ref<string>;
  currentPageId: Ref<string>;
  pages: Ref<PageListEntry[]>;
  doc: ShallowRef<DocumentModel | null>;
  history: ShallowRef<History | null>;
  projectApi: PageMutationProjectApi;
  refreshPages: () => Promise<PagesRefreshResult>;
  loadPage: (pageId: string) => Promise<{ ok: boolean; error?: Error }>;
  initEditor: (schema: ProjectSchema) => void;
  createBaseSchema: (projectId: string) => ProjectSchema;
};

export async function deletePageForStore(
  ctx: PageMutationStoreContext,
  pageId: string,
  mode?: "single" | "folder-only" | "cascade",
): Promise<void> {
  if (!ctx.projectId.value) {
    throw new Error("缺少工程信息");
  }
  if (!pageId) {
    throw new Error("缺少页面信息");
  }

  await ctx.projectApi.deletePage(ctx.projectId.value, pageId, mode);
  const { pages: pageList, entryConfig: entryConfigResp } =
    await ctx.refreshPages();

  if (
    ctx.currentPageId.value &&
    pageList.some((p) => p.id === ctx.currentPageId.value)
  ) {
    return;
  }

  const homeId = entryConfigResp?.homePageId;
  const nextId =
    homeId && pageList.some((p) => p.id === homeId)
      ? homeId
      : pageList.find((p) => p.type === "page")?.id;

  if (nextId) {
    await ctx.loadPage(nextId);
    return;
  }

  ctx.initEditor(ctx.createBaseSchema(ctx.projectId.value));
  ctx.currentPageId.value = "";
}

export async function movePageToGroupForStore(
  ctx: Pick<
    PageMutationStoreContext,
    "projectId" | "projectApi" | "refreshPages"
  >,
  pageId: string,
  targetGroupId: string | null,
  path?: string,
): Promise<void> {
  if (!ctx.projectId.value) {
    throw new Error("缺少工程信息");
  }
  if (!pageId) {
    throw new Error("缺少页面信息");
  }

  await ctx.projectApi.movePageToGroup(
    ctx.projectId.value,
    pageId,
    targetGroupId,
    path,
  );
  await ctx.refreshPages();
}

export async function renamePageForStore(
  ctx: PageMutationStoreContext,
  pageId: string,
  name: string,
  path?: string,
): Promise<void> {
  if (!ctx.projectId.value) {
    throw new Error("缺少工程信息");
  }
  if (!pageId) {
    throw new Error("缺少页面信息");
  }

  await ctx.projectApi.renamePage(
    ctx.projectId.value,
    pageId,
    name,
    path,
  );

  ctx.pages.value = mapPagesAfterRename(ctx.pages.value, pageId, name, path);

  if (ctx.doc.value && ctx.currentPageId.value === pageId && ctx.history.value) {
    const patch: { name: string; path?: string } = { name };
    if (path !== undefined) {
      patch.path = path;
    }
    ctx.history.value.execute(new UpdatePageCommand(pageId, patch));
  }
}
