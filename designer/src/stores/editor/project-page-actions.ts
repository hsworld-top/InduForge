// @ts-nocheck
/**
 * 工程页面列表刷新（从 editor-store 拆出）
 */

import { unwrapApiData } from "@/types/api";
import { normalizePageList } from "./normalize-schema.js";

export type PagesListApi = {
  getPages: (projectId: string) => Promise<unknown>;
};

export async function fetchNormalizedPageList(
  projectId: string,
  api: PagesListApi,
): Promise<{
  pageList: ReturnType<typeof normalizePageList>;
  entryConfig: Record<string, unknown>;
  responseData: Record<string, unknown>;
}> {
  const pagesResponse = await api.getPages(projectId);
  const responseData = unwrapApiData(pagesResponse);
  if (!responseData || typeof responseData !== "object") {
    throw new Error("页面列表响应无效");
  }
  const rd = responseData as Record<string, unknown>;
  const pageList = normalizePageList(rd.pages);
  const ec = rd.entryConfig;
  const entryConfig =
    ec && typeof ec === "object" ? (ec as Record<string, unknown>) : {};
  return { pageList, entryConfig, responseData: rd };
}
