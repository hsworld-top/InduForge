/**
 * 加载工程首屏（从 editor-store 拆出）
 */

import type { Ref, ShallowRef } from "vue";
import type { PageListEntry } from "./page-crud-actions";
import type { PageContentApi, ResolveProjectSchemaFn } from "./page-load-save-actions";
import type { PagesRefreshResult } from "./pages-sync-types";
import type { DocumentModel } from "@/editor-core/document/DocumentModel.ts";
import type { ProjectSchema } from "@/editor-core/document/types";
import { applyEntryPatchIfPresent } from "./entry-config-helpers";
import { fetchResolvedProjectSchemaForPage } from "./page-load-save-actions";

export interface LoadProjectStoreContext {
  projectId: Ref<string>;
  isLoading: Ref<boolean>;
  error: Ref<string>;
  doc: ShallowRef<DocumentModel | null>;
  currentPageId: Ref<string>;
  loadProjectSettings: () => Promise<void>;
  releasePageLock: () => Promise<void>;
  refreshPages: () => Promise<PagesRefreshResult>;
  createHomePage: (pid: string) => Promise<{ ok: boolean; pageId?: string; error?: Error }>;
  initEditor: (schema: ProjectSchema) => void;
  createBaseSchema: (projectId: string) => ProjectSchema;
  resolveLandingPageId: (
    pageList: PageListEntry[],
    entryConfig: { homePageId?: string } | null | undefined,
  ) => string | null;
  projectApi: PageContentApi;
  resolveProjectSchema: ResolveProjectSchemaFn;
}

export async function loadProjectForStore(
  id: string,
  ctx: LoadProjectStoreContext,
): Promise<{ ok: boolean; error?: Error }> {
  ctx.projectId.value = id || "";
  ctx.isLoading.value = true;
  ctx.error.value = "";

  try {
    await ctx.loadProjectSettings();
    await ctx.releasePageLock();
    const { pages: pageList, entryConfig: entryConfigResp } = await ctx.refreshPages();

    if (!pageList.length) {
      const homePageResult = await ctx.createHomePage(id);
      if (!homePageResult.ok) {
        ctx.initEditor(ctx.createBaseSchema(id));
      }
      return { ok: homePageResult.ok };
    }

    const targetPageId = ctx.resolveLandingPageId(pageList, entryConfigResp);

    if (!targetPageId) {
      const homePageResult = await ctx.createHomePage(id);
      if (!homePageResult.ok) {
        ctx.initEditor(ctx.createBaseSchema(id));
      }
      return { ok: homePageResult.ok };
    }

    const nextSchema = await fetchResolvedProjectSchemaForPage(
      ctx.projectApi,
      id,
      targetPageId,
      ctx.resolveProjectSchema,
    );
    ctx.initEditor(nextSchema);

    applyEntryPatchIfPresent(ctx.doc.value, entryConfigResp);

    ctx.currentPageId.value = targetPageId;
    return { ok: true };
  } catch (cause) {
    const nextError = cause instanceof Error ? cause : new Error("加载工程失败");
    ctx.error.value = nextError.message;
    ctx.initEditor(ctx.createBaseSchema(id));
    return { ok: false, error: nextError };
  } finally {
    ctx.isLoading.value = false;
  }
}

export type LoadProjectForStoreResult = Awaited<ReturnType<typeof loadProjectForStore>>;
