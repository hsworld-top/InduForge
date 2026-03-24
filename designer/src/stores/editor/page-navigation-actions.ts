/**
 * 页面切换 / 加载时的 schema 解析（从 editor-store 拆出）
 */

import type { DocumentModel } from "@/editor-core/document/DocumentModel.ts";
import type { EntryConfig, ProjectSchema } from "@/editor-core/document/types";
import {
  fetchResolvedProjectSchemaForPage,
  type PageContentApi,
  type ResolveProjectSchemaFn,
} from "./page-load-save-actions";

/** 将当前文档上的 entry 合并进刚解析出的 schema（切换页时保留内存中的入口状态） */
export function mergeExistingEntryIntoPageSchema(
  schema: ProjectSchema,
  existingEntry: EntryConfig | undefined | null,
): void {
  if (!existingEntry || typeof existingEntry !== "object") return;
  schema.entry = { ...schema.entry, ...existingEntry };
}

export type ResolvePageSchemaOnLoadArgs = {
  projectId: string;
  pageId: string;
  pageDrafts: Record<string, unknown>;
  doc: DocumentModel | null;
  projectApi: PageContentApi;
  resolveProjectSchema: ResolveProjectSchemaFn;
};

/**
 * 按草稿优先、否则拉取远端，解析为 ProjectSchema，并合并当前 doc.entry。
 */
export async function resolvePageSchemaOnLoad(
  args: ResolvePageSchemaOnLoadArgs,
): Promise<ProjectSchema> {
  const draft = args.pageDrafts[args.pageId];
  let nextSchema: ProjectSchema;
  if (draft) {
    nextSchema = args.resolveProjectSchema(
      draft,
      args.projectId,
      args.pageId,
    );
  } else {
    nextSchema = await fetchResolvedProjectSchemaForPage(
      args.projectApi,
      args.projectId,
      args.pageId,
      args.resolveProjectSchema,
    );
  }
  mergeExistingEntryIntoPageSchema(nextSchema, args.doc?.entry ?? null);
  return nextSchema;
}
