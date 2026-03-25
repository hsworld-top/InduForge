import type { EntryConfigFromApi } from "./pages-sync-types";
import type { DocumentModel } from "@/editor-core/document/DocumentModel.ts";
import type { EntryConfig } from "@/editor-core/document/types";

/**
 * 将入口配置补丁应用到文档（空对象/无文档时跳过）
 */
export function applyEntryPatchIfPresent(
  doc: DocumentModel | null | undefined,
  patch: Partial<EntryConfig> | EntryConfigFromApi | null | undefined,
): void {
  if (!patch || !doc || Object.keys(patch).length === 0) return;
  doc._updateEntry(patch as Partial<EntryConfig>);
}
