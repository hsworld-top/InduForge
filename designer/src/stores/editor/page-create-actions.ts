/**
 * 创建页面/分组（从 editor-store 拆出）
 */

import type { Ref } from "vue";
import { unwrapApiData } from "@/types/api";
import type { PagesRefreshResult } from "./pages-sync-types";

export type CreatePagePayload = {
  name: string;
  type: string;
  parentId?: string | null;
  schemaContent?: unknown;
};

export type PageCreateProjectApi = {
  createPage: (pid: string, payload: CreatePagePayload) => Promise<unknown>;
  updatePage: (pid: string, pageId: string, payload: unknown) => Promise<unknown>;
};

export type PageCreateStoreContext = {
  projectId: Ref<string>;
  projectApi: PageCreateProjectApi;
  refreshPages: () => Promise<PagesRefreshResult>;
};

export async function createPageForStore(
  ctx: PageCreateStoreContext,
  payload: CreatePagePayload,
): Promise<unknown> {
  if (!ctx.projectId.value) {
    throw new Error("缺少工程信息");
  }

  const result = await ctx.projectApi.createPage(ctx.projectId.value, payload);
  const data = unwrapApiData(result) as
    | { id?: string; page?: { id?: string } }
    | null
    | undefined;
  const pageId = data?.id || data?.page?.id;

  if (payload?.schemaContent && pageId) {
    await ctx.projectApi.updatePage(
      ctx.projectId.value,
      pageId,
      payload.schemaContent,
    );
  }

  await ctx.refreshPages();
  return data;
}
