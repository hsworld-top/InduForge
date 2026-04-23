import { describe, expect, it, vi } from "vitest";
import { ref } from "vue";
import { createBaseSchema } from "./normalize-schema";
import {
  createHomePageForStore,
  type CreateHomePageContext,
} from "./home-page-actions";

describe("home-page-actions", () => {
  function buildContext(overrides: Partial<CreateHomePageContext> = {}) {
    const createPage = vi.fn().mockResolvedValue({
      code: 0,
      msg: "ok",
      data: { id: "page-1" },
    });
    const updatePage = vi.fn().mockResolvedValue(undefined);
    const updateEntryConfig = vi.fn().mockResolvedValue(undefined);
    const deletePage = vi.fn().mockResolvedValue(undefined);

    return {
      ctx: {
        entryConfig: ref({}),
        currentPageId: ref(""),
        initEditor: vi.fn(),
        createBaseSchema,
        projectApi: {
          createPage,
          updatePage,
          updateEntryConfig,
          deletePage,
        } as unknown as CreateHomePageContext["projectApi"],
        canCommit: () => true,
        ...overrides,
      },
      createPage,
      updatePage,
      updateEntryConfig,
      deletePage,
    };
  }

  type ExtendedHomePageContext = CreateHomePageContext & {
    shouldRollbackStaleRemote?: () => boolean;
  };

  it("请求在创建页面后过期时会回滚远端页面并停止后续写入", async () => {
    let active = true;
    const { ctx, createPage, updatePage, updateEntryConfig, deletePage } = buildContext();
    createPage.mockImplementation(async () => {
      active = false;
      return {
        code: 0,
        msg: "ok",
        data: { id: "page-1" },
      };
    });
    ctx.canCommit = () => active;

    const result = await createHomePageForStore(ctx, "project-1", vi.fn());

    expect(result).toEqual({ ok: true, pageId: "page-1" });
    expect(updatePage).not.toHaveBeenCalled();
    expect(updateEntryConfig).not.toHaveBeenCalled();
    expect(deletePage).toHaveBeenCalledWith("project-1", "page-1", "single");
    expect(ctx.initEditor).not.toHaveBeenCalled();
    expect(ctx.currentPageId.value).toBe("");
    expect(ctx.entryConfig.value).toEqual({});
  });

  it("请求在入口配置写入后过期时会恢复入口配置并回滚首页", async () => {
    let active = true;
    const refreshPages = vi.fn();
    const { ctx, updatePage, updateEntryConfig, deletePage } = buildContext({
      entryConfig: ref({ homePageId: "old-home" }),
    });
    updateEntryConfig.mockImplementation(async () => {
      if (active) {
        active = false;
      }
      return undefined;
    });
    ctx.canCommit = () => active;

    const result = await createHomePageForStore(ctx, "project-1", refreshPages);

    expect(result).toEqual({ ok: true, pageId: "page-1" });
    expect(updatePage).toHaveBeenCalledTimes(1);
    expect(updateEntryConfig).toHaveBeenNthCalledWith(1, "project-1", { homePageId: "page-1" });
    expect(updateEntryConfig).toHaveBeenNthCalledWith(2, "project-1", { homePageId: "old-home" });
    expect(deletePage).toHaveBeenCalledWith("project-1", "page-1", "single");
    expect(refreshPages).not.toHaveBeenCalled();
    expect(ctx.initEditor).not.toHaveBeenCalled();
    expect(ctx.currentPageId.value).toBe("");
    expect(ctx.entryConfig.value).toEqual({ homePageId: "old-home" });
  });

  it("同一工程重入导致旧请求过期时不会回滚后续请求可复用的远端首页", async () => {
    let active = true;
    const refreshPages = vi.fn();
    const { ctx, updateEntryConfig, deletePage } = buildContext({
      entryConfig: ref({ homePageId: "old-home" }),
    });
    const extendedCtx = ctx as ExtendedHomePageContext;
    extendedCtx.shouldRollbackStaleRemote = () => false;
    updateEntryConfig.mockImplementation(async () => {
      if (active) {
        active = false;
      }
      return undefined;
    });
    extendedCtx.canCommit = () => active;

    const result = await createHomePageForStore(extendedCtx, "project-1", refreshPages);

    expect(result).toEqual({ ok: true, pageId: "page-1" });
    expect(updateEntryConfig).toHaveBeenCalledTimes(1);
    expect(updateEntryConfig).toHaveBeenCalledWith("project-1", { homePageId: "page-1" });
    expect(deletePage).not.toHaveBeenCalled();
    expect(refreshPages).not.toHaveBeenCalled();
    expect(extendedCtx.initEditor).not.toHaveBeenCalled();
    expect(extendedCtx.currentPageId.value).toBe("");
    expect(extendedCtx.entryConfig.value).toEqual({ homePageId: "old-home" });
  });
});
