import type { Ref } from "vue";

export type FixedSystemType = "home" | "login" | "logout";

interface BasicPageMetaLike {
  label: string;
  path: string;
}

interface CreatePageResultLike {
  id?: string;
  page?: {
    id?: string;
  };
}

interface CreateHomePageResultLike {
  ok?: boolean;
  pageId?: string;
  error?: {
    message?: string;
  };
}

interface EntryConfigLike {
  homePageId?: string | null;
  loginPageId?: string | null;
  logoutPageId?: string | null;
}

interface PageLike {
  id?: string | null;
  path?: string | null;
}

interface EditorStoreLike {
  createHomePage: (projectId: string) => Promise<CreateHomePageResultLike>;
  buildNewPageSchema: (input: { name: string; path: string }) => unknown;
  createPage: (payload: {
    name: string;
    type: "page";
    parentId: null;
    path?: string;
    schemaContent: unknown;
  }) => Promise<CreatePageResultLike | null | undefined>;
  saveEntryPatch: (patch: Record<string, unknown>) => Promise<void>;
  refreshPages: () => Promise<unknown>;
}

export interface CreateBasicPageActionContext {
  basicType: FixedSystemType;
  entryConfig: Ref<EntryConfigLike>;
  pages: Ref<PageLike[]>;
  projectId: Ref<string>;
  creatingBasicPageType: Ref<FixedSystemType | null>;
  editorStore: EditorStoreLike;
  getBasicPageMeta: (type: FixedSystemType) => BasicPageMetaLike;
  openCreatedPageTab: (pageId: string) => Promise<void>;
  showSuccess: (message: string) => void;
  showWarning: (message: string) => void;
  showError: (message: string) => void;
}

/**
 * 提取基础页面创建逻辑，便于页面树复用与单元测试。
 * 约束：基础页面仅允许通过菜单入口创建，并对重复触发做前端幂等保护。
 * @param {CreateBasicPageActionContext} ctx - 创建所需上下文
 * @returns {Promise<void>}
 */
export async function createBasicPageAction(ctx: CreateBasicPageActionContext): Promise<void> {
  if (ctx.creatingBasicPageType.value) {
    return;
  }

  ctx.creatingBasicPageType.value = ctx.basicType;
  try {
    if (ctx.basicType === "home") {
      await createHomeBasicPage(ctx);
      return;
    }

    await createFixedEntryPage(ctx);
  } finally {
    ctx.creatingBasicPageType.value = null;
  }
}

/**
 * 创建首页入口页。
 * @param {CreateBasicPageActionContext} ctx - 创建上下文
 * @returns {Promise<void>}
 */
async function createHomeBasicPage(ctx: CreateBasicPageActionContext): Promise<void> {
  if (ctx.entryConfig.value?.homePageId) {
    ctx.showWarning("首页已存在");
    return;
  }
  if (!ctx.projectId.value) {
    ctx.showError("缺少工程信息");
    return;
  }

  const result = await ctx.editorStore.createHomePage(ctx.projectId.value);
  if (!result?.ok || !result.pageId) {
    ctx.showError(result?.error?.message || "首页创建失败");
    return;
  }

  await ctx.openCreatedPageTab(result.pageId);
  ctx.showSuccess("首页创建成功");
}

/**
 * 创建登录页或登出页，并同步入口配置。
 * @param {CreateBasicPageActionContext} ctx - 创建上下文
 * @returns {Promise<void>}
 */
async function createFixedEntryPage(ctx: CreateBasicPageActionContext): Promise<void> {
  const meta = ctx.getBasicPageMeta(ctx.basicType);
  const entryKey = ctx.basicType === "login" ? "loginPageId" : "logoutPageId";
  const entryPageId = ctx.entryConfig.value?.[entryKey];
  const hasBoundEntryPage =
    typeof entryPageId === "string" &&
    ctx.pages.value.some((page) => page.id === entryPageId);
  const hasSamePathPage = ctx.pages.value.some((page) => page.path === meta.path);

  if (hasBoundEntryPage || hasSamePathPage) {
    ctx.showWarning(`${meta.label}已存在`);
    return;
  }

  try {
    const schemaContent = ctx.editorStore.buildNewPageSchema({
      name: meta.label,
      path: meta.path,
    });
    const result = await ctx.editorStore.createPage({
      name: meta.label,
      type: "page",
      parentId: null,
      path: meta.path,
      schemaContent,
    });
    const pageId = result?.id || result?.page?.id;
    if (!pageId) {
      throw new Error(`${meta.label}创建失败`);
    }

    try {
      await ctx.editorStore.saveEntryPatch({ [entryKey]: pageId });
    } catch (error) {
      const reason = error instanceof Error ? error.message : `${meta.label}入口绑定失败`;
      await ctx.openCreatedPageTab(pageId);
      ctx.showError(`${meta.label}已创建，但入口绑定失败：${reason}`);
      return;
    }

    await ctx.editorStore.refreshPages();
    await ctx.openCreatedPageTab(pageId);
    ctx.showSuccess(`${meta.label}创建成功`);
  } catch (error) {
    const message = error instanceof Error ? error.message : `${meta.label}创建失败`;
    ctx.showError(message);
  }
}
