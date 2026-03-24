/**
 * 当前页面与入口配置的持久化（从 editor-store 拆出）
 */

import type { Ref, ShallowRef } from "vue";
import type { DocumentModel } from "@/editor-core/document/DocumentModel.ts";
import type { Serializer } from "@/editor-core/document/Serializer.ts";
import { mergePageVariablesIntoPayload } from "./page-load-save-actions";

export type EntryPersistProjectApi = {
  updateEntryConfig: (pid: string, payload: unknown) => Promise<unknown>;
};

export type PagePersistProjectApi = {
  updatePage: (
    pid: string,
    pageId: string,
    payload: unknown,
  ) => Promise<unknown>;
};

export type PersistEntryConfigContext = {
  projectId: Ref<string>;
  doc: ShallowRef<DocumentModel | null>;
  entryConfig: Ref<Record<string, unknown>>;
  projectApi: EntryPersistProjectApi;
};

export type SaveCurrentPageContext = {
  projectId: Ref<string>;
  currentPageId: Ref<string>;
  doc: ShallowRef<DocumentModel | null>;
  serializer: ShallowRef<Serializer>;
  pageDrafts: Ref<Record<string, unknown>>;
  projectApi: PagePersistProjectApi;
};

export type SavePageDraftContext = {
  doc: ShallowRef<DocumentModel | null>;
  serializer: ShallowRef<Serializer>;
  pageDrafts: Ref<Record<string, unknown>>;
};

export async function persistEntryConfigForStore(
  ctx: PersistEntryConfigContext,
): Promise<void> {
  const pid = ctx.projectId.value;
  if (!pid) return;
  const raw = ctx.doc.value?.entry || ctx.entryConfig.value || {};
  const payload =
    raw && typeof raw === "object"
      ? ({ ...raw } as Record<string, unknown>)
      : {};
  await ctx.projectApi.updateEntryConfig(pid, payload);
  ctx.entryConfig.value = { ...payload };
}

export async function saveCurrentPageForStore(
  ctx: SaveCurrentPageContext,
): Promise<void> {
  const pid = ctx.projectId.value;
  const pageId = ctx.currentPageId.value;
  const doc = ctx.doc.value;
  const serializer = ctx.serializer.value;
  if (!doc) {
    throw new Error("缺少文档");
  }
  const payload = serializer.exportPage(doc, pageId);
  mergePageVariablesIntoPayload(
    payload as unknown as Record<string, unknown>,
    pageId,
    doc.schema?.vars?.pages?.[pageId],
  );
  await ctx.projectApi.updatePage(pid, pageId, payload);
  if (ctx.pageDrafts.value[pageId]) {
    const nextDrafts = { ...(ctx.pageDrafts.value || {}) };
    delete nextDrafts[pageId];
    ctx.pageDrafts.value = nextDrafts;
  }
}

export function savePageDraftForStore(
  ctx: SavePageDraftContext,
  pageId: string,
): void {
  if (!ctx.doc.value || !pageId) return;
  try {
    const payload = ctx.serializer.value.exportPage(ctx.doc.value, pageId);
    ctx.pageDrafts.value = { ...(ctx.pageDrafts.value || {}), [pageId]: payload };
  } catch {
    // ignore
  }
}
