import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import { applyRuntimeRouteEffects, registerDesignerBeforeEachGuard } from "./index";
import {
  buildIdeRestoreUrl,
  createBootstrapRequest,
  resolveDesignerIdeOriginFromRuntime,
  resolveDesignerEntrypointPlan,
  shouldRedirectTopLevelToIde,
  shouldUseDebugMode,
} from "../runtime/host-bootstrap";
import { Storage } from "@/utils/storage";
import { getEditorUiStore } from "@/stores/editor-ui-store";
import { i18n } from "@/i18n";

describe("designer 入口规划", () => {
  afterEach(() => {
    localStorage.clear();
    vi.unstubAllEnvs();
  });

  it("正式入口顶层访问会带 handoffId 回跳 IDE", () => {
    const plan = resolveDesignerEntrypointPlan("http://designer.example/designer/?handoffId=handoff-1", {
      isTopLevelWindow: true,
      ideOrigin: "http://ide.example",
    });

    expect(plan.isDebugRoute).toBe(false);
    expect(plan.shouldRedirectToIde).toBe(true);
    expect(plan.shouldWaitForBootstrap).toBe(false);
    expect(plan.ideRedirectUrl).toBe("http://ide.example/?handoffId=handoff-1");
  });

  it("/designer/debug 保留独立调试模式，不会回跳 IDE", () => {
    const plan = resolveDesignerEntrypointPlan("http://designer.example/designer/debug?handoffId=handoff-1", {
      isTopLevelWindow: true,
      ideOrigin: "http://ide.example",
    });

    expect(plan.isDebugRoute).toBe(true);
    expect(plan.shouldRedirectToIde).toBe(false);
    expect(plan.shouldWaitForBootstrap).toBe(false);
    expect(plan.ideRedirectUrl).toBeNull();
  });

  it("runtime helper 会把正式入口、debug 例外与 handoff 恢复地址统一起来", () => {
    expect(shouldUseDebugMode("/designer/debug")).toBe(true);
    expect(shouldUseDebugMode("/designer/")).toBe(false);
    expect(shouldRedirectTopLevelToIde("/designer/", true)).toBe(true);
    expect(shouldRedirectTopLevelToIde("/designer/debug", true)).toBe(false);
    expect(shouldRedirectTopLevelToIde("/designer/", false)).toBe(false);
    expect(buildIdeRestoreUrl("handoff-restore", "http://ide.example")).toBe(
      "http://ide.example/?handoffId=handoff-restore",
    );

    expect(createBootstrapRequest("http://designer.example/designer/?handoffId=handoff-9")).toMatchObject({
      type: "APP_BOOTSTRAP_REQUEST",
      payload: {
        appType: "designer",
        handoffId: "handoff-9",
        pathname: "/designer/",
        search: "?handoffId=handoff-9",
      },
    });
  });

  it("开发态默认按 IDE 端口推导回跳 origin", () => {
    expect(
      resolveDesignerIdeOriginFromRuntime({
        currentUrl: "http://127.0.0.1:18603/designer/",
        isDev: true,
        devHost: "127.0.0.1",
        idePort: "18601",
      }),
    ).toBe("http://127.0.0.1:18601");
  });
});

describe("runtime route effects", () => {
  afterEach(() => {
    localStorage.clear();
    document.documentElement.className = "";
    document.documentElement.removeAttribute("data-theme");
    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "light", locale: "zh" });
    i18n.global.locale.value = "zh";
  });

  it("设计态路由不会从本地缓存恢复主题语言，而是保留当前 editorUi 状态", () => {
    Storage.setTheme("dark");
    Storage.setLanguage("en");

    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "dark", locale: "en" });

    applyRuntimeRouteEffects("/", "http://designer.example/designer/", { editorUi });

    expect(editorUi.theme.value).toBe("dark");
    expect(editorUi.locale.value).toBe("en");
    expect(i18n.global.locale.value).toBe("en");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
  });
});

describe("router beforeEach bootstrap", () => {
  beforeEach(() => {
    localStorage.clear();
    window.history.replaceState({}, "", "/designer/");
  });

  afterEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
    vi.unstubAllEnvs();
  });

  it("正式入口会先等待宿主 bootstrap，再继续鉴权与项目恢复", async () => {
    const waitForBootstrap = vi.fn(async () => {
      Storage.setToken("bootstrap-token");
      Storage.setProjectId("bootstrap-project");
      Storage.setTenantId("tenant-1");
      return true;
    });
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
          path: "/debug",
          name: "DesignerDebug",
          component: { template: "<div>debug</div>" },
          meta: { title: "设计器调试", requiresAuth: false },
        },
      ],
    });
    const removeGuard = registerDesignerBeforeEachGuard(testRouter, {
      waitForBootstrap,
      navigateToUrl: vi.fn(),
      getIdeOrigin: () => "http://ide.example",
      getCurrentUrl: () => "http://designer.example/designer/",
      isTopLevelWindow: () => false,
    });

    await testRouter.push("/");

    expect(waitForBootstrap).toHaveBeenCalledTimes(1);
    expect(testRouter.currentRoute.value.meta.project).toEqual({
      id: "bootstrap-project",
      tenantId: "tenant-1",
    });

    removeGuard();
  });

  it("正式入口等待 bootstrap 失败后不会卡住，而是回到现有登录兜底", async () => {
    const waitForBootstrap = vi.fn(async () => false);
    const navigateToUrl = vi.fn();
    const testRouter = createRouter({
      history: createMemoryHistory("/designer/"),
      routes: [
        {
          path: "/",
          name: "Designer",
          component: { template: "<div>designer</div>" },
          meta: { title: "设计器", requiresAuth: true },
        },
      ],
    });
    const removeGuard = registerDesignerBeforeEachGuard(testRouter, {
      waitForBootstrap,
      navigateToUrl,
      getIdeOrigin: () => "http://ide.example",
      getCurrentUrl: () => "http://designer.example/designer/?handoffId=handoff-timeout",
      isTopLevelWindow: () => false,
    });

    await testRouter.push("/");

    expect(waitForBootstrap).toHaveBeenCalledTimes(1);
    expect(navigateToUrl).toHaveBeenCalledWith(
      "http://ide.example/login?redirect=http%3A%2F%2Fdesigner.example%2Fdesigner%2F%3FhandoffId%3Dhandoff-timeout",
    );

    removeGuard();
  });

  it("嵌入正式入口如果已存在 token，不会等待 bootstrap 且可直接进入", async () => {
    Storage.setToken("cached-token");
    Storage.setProjectId("cached-project");
    Storage.setTenantId("tenant-cached");

    const waitForBootstrap = vi.fn(async () => true);
    const navigateToUrl = vi.fn();
    const testRouter = createRouter({
      history: createMemoryHistory("/designer/"),
      routes: [
        {
          path: "/",
          name: "Designer",
          component: { template: "<div>designer</div>" },
          meta: { title: "设计器", requiresAuth: true },
        },
      ],
    });
    const removeGuard = registerDesignerBeforeEachGuard(testRouter, {
      waitForBootstrap,
      navigateToUrl,
      getIdeOrigin: () => "http://ide.example",
      getCurrentUrl: () => "http://designer.example/designer/?handoffId=handoff-cache",
      isTopLevelWindow: () => false,
    });

    await testRouter.push("/");

    expect(waitForBootstrap).not.toHaveBeenCalled();
    expect(navigateToUrl).not.toHaveBeenCalled();
    expect(testRouter.currentRoute.value.meta.project).toEqual({
      id: "cached-project",
      tenantId: "tenant-cached",
    });

    removeGuard();
  });

  it("正式入口顶层访问会直接回跳 IDE，不等待 bootstrap", async () => {
    const waitForBootstrap = vi.fn(async () => true);
    const navigateToUrl = vi.fn();
    const testRouter = createRouter({
      history: createMemoryHistory("/designer/"),
      routes: [
        {
          path: "/",
          name: "Designer",
          component: { template: "<div>designer</div>" },
          meta: { title: "设计器", requiresAuth: true },
        },
      ],
    });
    const removeGuard = registerDesignerBeforeEachGuard(testRouter, {
      waitForBootstrap,
      navigateToUrl,
      getIdeOrigin: () => "http://ide.example",
      getCurrentUrl: () => "http://designer.example/designer/?handoffId=handoff-top",
      isTopLevelWindow: () => true,
    });

    await testRouter.push("/");

    expect(waitForBootstrap).not.toHaveBeenCalled();
    expect(navigateToUrl).toHaveBeenCalledWith("http://ide.example/?handoffId=handoff-top");

    removeGuard();
  });

  it("正式入口顶层访问在未注入 getIdeOrigin 时仍会回跳到配置的 IDE origin", async () => {
    vi.stubEnv("VITE_IDE_ORIGIN", "http://ide.example");

    const waitForBootstrap = vi.fn(async () => true);
    const navigateToUrl = vi.fn();
    const testRouter = createRouter({
      history: createMemoryHistory("/designer/"),
      routes: [
        {
          path: "/",
          name: "Designer",
          component: { template: "<div>designer</div>" },
          meta: { title: "设计器", requiresAuth: true },
        },
      ],
    });
    const removeGuard = registerDesignerBeforeEachGuard(testRouter, {
      waitForBootstrap,
      navigateToUrl,
      getCurrentUrl: () => "http://designer.example/designer/?handoffId=handoff-runtime",
      isTopLevelWindow: () => true,
    });

    await testRouter.push("/");

    expect(waitForBootstrap).not.toHaveBeenCalled();
    expect(navigateToUrl).toHaveBeenCalledWith("http://ide.example/?handoffId=handoff-runtime");

    removeGuard();
  });

  it("debug 路由不会等待 bootstrap，允许独立进入", async () => {
    const waitForBootstrap = vi.fn(async () => true);
    const navigateToUrl = vi.fn();
    const testRouter = createRouter({
      history: createMemoryHistory("/designer/"),
      routes: [
        {
          path: "/debug",
          name: "DesignerDebug",
          component: { template: "<div>debug</div>" },
          meta: { title: "设计器调试", requiresAuth: false },
        },
      ],
    });
    const removeGuard = registerDesignerBeforeEachGuard(testRouter, {
      waitForBootstrap,
      navigateToUrl,
      getIdeOrigin: () => "http://ide.example",
      getCurrentUrl: () => "http://designer.example/designer/debug?handoffId=handoff-2",
      isTopLevelWindow: () => true,
    });

    await testRouter.push("/debug");

    expect(waitForBootstrap).not.toHaveBeenCalled();
    expect(navigateToUrl).not.toHaveBeenCalled();
    expect(testRouter.currentRoute.value.name).toBe("DesignerDebug");

    removeGuard();
  });

  it("程序化导航到 debug 时会按目标路由判定入口模式，不受旧 href 干扰", async () => {
    const waitForBootstrap = vi.fn(async () => true);
    const navigateToUrl = vi.fn();
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
          path: "/debug",
          name: "DesignerDebug",
          component: { template: "<div>debug</div>" },
          meta: { title: "设计器调试", requiresAuth: false },
        },
      ],
    });
    const removeGuard = registerDesignerBeforeEachGuard(testRouter, {
      waitForBootstrap,
      navigateToUrl,
      getIdeOrigin: () => "http://ide.example",
      getCurrentUrl: () => "http://designer.example/designer/?handoffId=handoff-stale",
      isTopLevelWindow: () => true,
    });

    await testRouter.push("/debug");

    expect(waitForBootstrap).not.toHaveBeenCalled();
    expect(navigateToUrl).not.toHaveBeenCalled();
    expect(testRouter.currentRoute.value.name).toBe("DesignerDebug");

    removeGuard();
  });
});
