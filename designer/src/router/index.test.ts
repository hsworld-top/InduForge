import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import { i18n } from "@/i18n";
import { STORAGE_KEYS } from "@/constants";
import { getEditorUiStore } from "@/stores/editor-ui-store";
import { applyRuntimeRouteEffects, registerDesignerBeforeEachGuard } from "./index";
import { Storage } from "@/utils/storage";

describe("router runtime effects", () => {
  beforeEach(() => {
    Storage.setToken("seed-token");
    Storage.setProjectId("seed-project");
    window.history.replaceState({}, "", "/designer/");
  });

  afterEach(() => {
    localStorage.clear();
    document.documentElement.className = "";
    document.documentElement.removeAttribute("data-theme");
    i18n.global.locale.value = "zh";

    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "light", locale: "zh" });
    vi.restoreAllMocks();
  });

  it("preview 路由不会同步 editorUi，并会清除主题副作用和清洗查询参数", () => {
    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "dark", locale: "en" });
    const initSpy = vi.spyOn(editorUi, "initFromRuntime");
    const replaceState = vi.fn();

    document.documentElement.classList.add("dark");
    document.documentElement.setAttribute("data-theme", "dark");

    applyRuntimeRouteEffects(
      "/preview",
      "http://localhost/designer/preview?theme=dark&locale=en&token=t1&refreshToken=rt1&id=p1",
      { editorUi, replaceState },
    );

    expect(initSpy).not.toHaveBeenCalled();
    expect(document.documentElement.classList.contains("dark")).toBe(false);
    expect(document.documentElement.hasAttribute("data-theme")).toBe(false);
    expect(replaceState).toHaveBeenCalledWith({}, "", "/designer/preview?id=p1");
  });

  it("编辑路由进入时优先保留 designer 专属主题与语言，而不是通用键", () => {
    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "light", locale: "zh" });
    const replaceState = vi.fn();

    localStorage.setItem(STORAGE_KEYS.DESIGNER_THEME, JSON.stringify("dark"));
    localStorage.setItem(STORAGE_KEYS.DESIGNER_LANGUAGE, JSON.stringify("en"));
    localStorage.setItem(STORAGE_KEYS.THEME, JSON.stringify("light"));
    localStorage.setItem(STORAGE_KEYS.LANGUAGE, JSON.stringify("zh"));

    applyRuntimeRouteEffects("/designer", "http://localhost/designer/?pid=p1", {
      editorUi,
      replaceState,
    });

    expect(editorUi.theme.value).toBe("dark");
    expect(editorUi.locale.value).toBe("en");
    expect(i18n.global.locale.value).toBe("en");
    expect(replaceState).not.toHaveBeenCalled();
  });

  it("preview 路由仍保持隔离，不会同步 editorUi 即使存在 designer 专属键", () => {
    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "light", locale: "zh" });
    const initSpy = vi.spyOn(editorUi, "initFromRuntime");
    const replaceState = vi.fn();

    localStorage.setItem(STORAGE_KEYS.DESIGNER_THEME, JSON.stringify("dark"));
    localStorage.setItem(STORAGE_KEYS.DESIGNER_LANGUAGE, JSON.stringify("en"));
    localStorage.setItem(STORAGE_KEYS.THEME, JSON.stringify("light"));
    localStorage.setItem(STORAGE_KEYS.LANGUAGE, JSON.stringify("zh"));

    applyRuntimeRouteEffects("/preview", "http://localhost/designer/preview?pid=p1", {
      editorUi,
      replaceState,
    });

    expect(initSpy).not.toHaveBeenCalled();
    expect(editorUi.theme.value).toBe("light");
    expect(editorUi.locale.value).toBe("zh");
    expect(i18n.global.locale.value).toBe("zh");
  });

  it("通过 beforeEach 导航到 preview 时会清洗 URL 且不同步 editorUi，导航到设计器路由时会消费运行时参数", async () => {
    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "light", locale: "zh" });

    const initSpy = vi.spyOn(editorUi, "initFromRuntime");
    const replaceStateSpy = vi.spyOn(window.history, "replaceState");
    const testRouter = createRouter({
      history: createMemoryHistory("/designer/"),
      routes: [
        {
          path: "/",
          name: "Designer",
          component: { template: "<div>designer</div>" },
          meta: { title: "设计器", requiresAuth: true },
        },
        {
          path: "/preview",
          name: "Preview",
          component: { template: "<div>preview</div>" },
          meta: { title: "预览", requiresAuth: true },
        },
      ],
    });
    const removeGuard = registerDesignerBeforeEachGuard(testRouter);

    document.documentElement.classList.add("dark");
    document.documentElement.setAttribute("data-theme", "dark");
    window.history.replaceState(
      {},
      "",
      "/designer/preview?theme=dark&locale=en&token=t1&refreshToken=rt1&id=p1",
    );

    await testRouter.push("/preview");

    expect(initSpy).not.toHaveBeenCalled();
    expect(document.documentElement.classList.contains("dark")).toBe(false);
    expect(document.documentElement.hasAttribute("data-theme")).toBe(false);
    expect(replaceStateSpy).toHaveBeenLastCalledWith({}, "", "/designer/preview?id=p1");
    expect(Storage.getToken()).toBe("t1");
    expect(Storage.getRefreshToken()).toBe("rt1");

    initSpy.mockClear();
    replaceStateSpy.mockClear();

    window.history.replaceState(
      {},
      "",
      "/designer/?theme=dark&locale=en&token=t2&refreshToken=rt2&pid=project-1&tenant=tenant-1",
    );

    await testRouter.push("/");

    expect(initSpy).toHaveBeenCalledWith({ theme: "dark", locale: "en" });
    expect(editorUi.theme.value).toBe("dark");
    expect(editorUi.locale.value).toBe("en");
    expect(replaceStateSpy).toHaveBeenLastCalledWith({}, "", "/designer/?pid=project-1&tenant=tenant-1");
    expect(Storage.getToken()).toBe("t2");
    expect(Storage.getRefreshToken()).toBe("rt2");

    removeGuard();
  });
});
