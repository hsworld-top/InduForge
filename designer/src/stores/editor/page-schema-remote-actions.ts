/**
 * 远端更新页面 Schema（从 editor-store 拆出）
 */

import type { Ref, ShallowRef } from "vue";
import type { DocumentModel } from "@/editor-core/document/DocumentModel";
import type { ExportedPagePayload, Serializer } from "@/editor-core/document/Serializer";

export interface PageSchemaRemoteProjectApi {
  updatePage: (pid: string, pageId: string, payload: unknown) => Promise<unknown>;
}

export interface PageSchemaRemoteStoreContext {
  projectId: Ref<string>;
  doc: ShallowRef<DocumentModel | null>;
  serializer: ShallowRef<Serializer>;
  projectApi: PageSchemaRemoteProjectApi;
}

export async function updatePageSchemaForStore(
  ctx: PageSchemaRemoteStoreContext,
  pageId: string,
  schema?: ExportedPagePayload,
): Promise<void> {
  if (!ctx.projectId.value) {
    throw new Error("缺少工程信息");
  }
  if (!pageId) {
    throw new Error("缺少页面信息");
  }
  if (!schema && !ctx.doc.value) {
    throw new Error("缺少文档");
  }

  let payload: ExportedPagePayload;
  if (schema) {
    payload = schema;
  } else {
    const doc = ctx.doc.value;
    if (!doc) {
      throw new Error("缺少文档");
    }
    payload = ctx.serializer.value.exportPage(doc, pageId);
  }
  await ctx.projectApi.updatePage(ctx.projectId.value, pageId, payload);
}
