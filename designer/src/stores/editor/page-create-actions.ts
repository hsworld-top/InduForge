/**
 * 创建页面/分组（从 editor-store 拆出）
 */

import type { Ref } from "vue";
import type { PagesRefreshResult } from "./pages-sync-types";
import { unwrapApiData } from "@/types/api";

export interface CreatePagePayload {
  name: string;
  type: string;
  parentId?: string | null;
  path?: string;
  schemaContent?: unknown;
}

export interface PageCreateProjectApi {
  createPage: (pid: string, payload: CreatePagePayload) => Promise<unknown>;
  updatePage: (pid: string, pageId: string, payload: unknown) => Promise<unknown>;
}

export interface PageCreateStoreContext {
  projectId: Ref<string>;
  projectApi: PageCreateProjectApi;
  refreshPages: () => Promise<PagesRefreshResult>;
}

/** createPage API 解包后的常见形状（与 unwrapApiData 一致） */
export type CreatePageForStoreResult = { id?: string; page?: { id?: string } } | null | undefined;

export async function createPageForStore(
  ctx: PageCreateStoreContext,
  payload: CreatePagePayload,
): Promise<CreatePageForStoreResult> {
  if (!ctx.projectId.value) {
    throw new Error("缺少工程信息");
  }

  const result = await ctx.projectApi.createPage(ctx.projectId.value, payload);
  const data = unwrapApiData(result) as CreatePageForStoreResult;
  const pageId = data?.id || data?.page?.id;

  if (payload?.schemaContent && pageId) {
    await ctx.projectApi.updatePage(ctx.projectId.value, pageId, payload.schemaContent);
  }

  await ctx.refreshPages();
  return data;
}
