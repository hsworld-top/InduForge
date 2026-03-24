/**
 * 页面列表拉取与创建页 API 辅助（从 editor-store 拆出）
 */

import { unwrapApiData } from "@/types/api";
import { projectApi } from "@/services";
import type { CreatePageBody } from "@/services/projectApi";
import { fetchPagesListPayload } from "./project-api-helpers";

function normalizePageListPayload(payload: unknown): unknown[] {
  if (payload == null) return [];
  if (Array.isArray(payload)) return payload;
  throw new Error(
    "页面列表格式无效：期望数组（不再兼容 items/list 等历史字段）",
  );
}

export async function loadNormalizedPageList(projectId: string): Promise<{
  pageList: unknown[];
  entryConfig: Record<string, unknown>;
}> {
  const responseData = await fetchPagesListPayload(projectId);
  const pageList = normalizePageListPayload(responseData.pages);
  const entryConfig =
    responseData.entryConfig && typeof responseData.entryConfig === "object"
      ? (responseData.entryConfig as Record<string, unknown>)
      : {};
  return { pageList, entryConfig };
}

/**
 * 创建页面；若 body 含 schemaContent 则在拿到 id 后写入页面 schema
 */
export async function createPageWithOptionalSchema(
  projectId: string,
  payload: CreatePageBody,
): Promise<unknown> {
  const result = await projectApi.createPage(projectId, payload);
  const data = unwrapApiData(result) as Record<string, unknown> | null;
  if (!data || typeof data !== "object" || data.id == null || data.id === "") {
    throw new Error("创建页面失败：API 返回缺少 id");
  }
  const pageId = String(data.id);
  if (payload.schemaContent && pageId) {
    await projectApi.updatePage(projectId, pageId, payload.schemaContent);
  }
  return data;
}
