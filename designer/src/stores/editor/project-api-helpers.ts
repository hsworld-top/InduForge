/**
 * 工程/页面与设置相关的 API 拉取辅助（从 editor-store 拆出）
 */

import type { PagesListPayload } from "@/types/api";
import { projectApi } from "@/services";
import { unwrapApiData } from "@/types/api";

export async function fetchProjectVariablesAndSettings(projectId: string): Promise<{
  varsResult: unknown;
  settingsResult: unknown;
}> {
  const results = await Promise.allSettled([
    projectApi.getProjectVariables(projectId),
    projectApi.getProjectSettings(projectId),
  ]);

  const varsResult = results[0].status === "fulfilled" ? unwrapApiData(results[0].value) : null;
  const settingsResult = results[1].status === "fulfilled" ? unwrapApiData(results[1].value) : null;

  return { varsResult, settingsResult };
}

export async function fetchPagesListPayload(projectId: string): Promise<PagesListPayload> {
  const pagesResponse = await projectApi.getPages(projectId);
  return unwrapApiData(pagesResponse) as PagesListPayload;
}
