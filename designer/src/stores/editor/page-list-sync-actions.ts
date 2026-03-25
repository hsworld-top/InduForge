/**
 * 刷新工程页面列表并同步 entry 到当前文档（从 editor-store 拆出）
 */

import type { Ref, ShallowRef } from "vue";
import type { PageListEntry } from "./page-crud-actions";
import type { EntryConfigFromApi, PagesRefreshResult } from "./pages-sync-types";
import type { PagesListApi } from "./project-page-actions";
import type { DocumentModel } from "@/editor-core/document/DocumentModel.ts";
import { applyEntryPatchIfPresent } from "./entry-config-helpers";
import { fetchNormalizedPageList } from "./project-page-actions";

export async function refreshPagesForStore(input: {
  projectId: string;
  pages: Ref<PageListEntry[]>;
  entryConfig: Ref<EntryConfigFromApi>;
  doc: ShallowRef<DocumentModel | null>;
  projectApi: PagesListApi;
}): Promise<PagesRefreshResult> {
  if (!input.projectId) {
    input.pages.value = [];
    input.entryConfig.value = {} as EntryConfigFromApi;
    return { pages: [], entryConfig: {} as EntryConfigFromApi };
  }
  const { pages: pageList, entryConfig: newEntryConfig } = await fetchNormalizedPageList(
    input.projectId,
    input.projectApi,
  );
  input.pages.value = pageList;
  input.entryConfig.value = newEntryConfig;
  applyEntryPatchIfPresent(input.doc.value, newEntryConfig);
  return { pages: pageList, entryConfig: newEntryConfig };
}
