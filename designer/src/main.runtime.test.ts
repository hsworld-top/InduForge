import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { i18n } from "./i18n";
import { createRuntimeMessageHandler, resolveTrustedMessageSources } from "./main";
import { getEditorUiStore } from "./stores/editor-ui-store";
import {
  initializeDesignerHostBootstrap,
  resetHostBootstrapSessionForTests,
  waitForHostBootstrap,
} from "./runtime/host-bootstrap";
import { Storage } from "@/utils/storage";

describe("main runtime message handler", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.className = "";
    document.documentElement.removeAttribute("data-theme");

    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "light", locale: "zh" });
    i18n.global.locale.value = "zh";
    resetHostBootstrapSessionForTests();
  });

  afterEach(() => {
    localStorage.clear();
    document.documentElement.className = "";
    document.documentElement.removeAttribute("data-theme");

    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "light", locale: "zh" });
    i18n.global.locale.value = "zh";
    resetHostBootstrapSessionForTests();
    vi.useRealTimers();
  });

  it("APP_BOOTSTRAP_RESPONSE 会写入 token、projectId、theme 与 locale", () => {
    const editorUi = getEditorUiStore();
    const trustedParent = {} as Window;
    const handler = createRuntimeMessageHandler({
      editorUi,
      i18n,
      getTrustedOriginSet: () => new Set(["http://localhost:18601"]),
      getTrustedSources: () => [trustedParent],
    });

    handler(
      new MessageEvent("message", {
        data: {
          type: "APP_BOOTSTRAP_RESPONSE",
          payload: {
            token: "bootstrap-token",
            refreshToken: "bootstrap-refresh",
            projectId: "project-1",
            tenantId: "tenant-1",
            theme: "dark",
            locale: "en",
          },
        },
        origin: "http://localhost:18601",
        source: trustedParent,
      }),
    );

    expect(Storage.getToken()).toBe("bootstrap-token");
    expect(Storage.getRefreshToken()).toBe("bootstrap-refresh");
    expect(Storage.getProjectId()).toBe("project-1");
    expect(Storage.getTenantId()).toBe("tenant-1");
    expect(editorUi.theme.value).toBe("dark");
    expect(editorUi.locale.value).toBe("en");
    expect(i18n.global.locale.value).toBe("en");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
  });

  it("无宿主消息时不会从本地缓存恢复主题与语言", () => {
    Storage.setTheme("dark");
    Storage.setLanguage("en");

    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime();

    expect(editorUi.theme.value).toBe("light");
    expect(editorUi.locale.value).toBe("zh");
    expect(i18n.global.locale.value).toBe("zh");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("AUTH_REFRESHED 会更新 token 与 refreshToken", () => {
    const editorUi = getEditorUiStore();
    const trustedParent = {} as Window;
    const handler = createRuntimeMessageHandler({
      editorUi,
      i18n,
      getTrustedOriginSet: () => new Set(["http://localhost:18601"]),
      getTrustedSources: () => [trustedParent],
    });

    Storage.setToken("old-token");
    Storage.setRefreshToken("old-refresh");

    handler(
      new MessageEvent("message", {
        data: {
          type: "AUTH_REFRESHED",
          token: "next-token",
          refreshToken: "next-refresh",
        },
        origin: "http://localhost:18601",
        source: trustedParent,
      }),
    );

    expect(Storage.getToken()).toBe("next-token");
    expect(Storage.getRefreshToken()).toBe("next-refresh");
  });

  it("非可信 origin 或 source 的消息会被忽略", () => {
    const editorUi = getEditorUiStore();
    const trustedParent = {} as Window;
    const handler = createRuntimeMessageHandler({
      editorUi,
      i18n,
      getTrustedOriginSet: () => new Set(["http://localhost:18601"]),
      getTrustedSources: () => [trustedParent],
    });

    handler(
      new MessageEvent("message", {
        data: {
          type: "APP_BOOTSTRAP_RESPONSE",
          payload: {
            token: "bad-token",
            projectId: "bad-project",
            theme: "dark",
            locale: "en",
          },
        },
        origin: "http://evil.example",
        source: trustedParent,
      }),
    );
    handler(
      new MessageEvent("message", {
        data: {
          type: "AUTH_REFRESHED",
          token: "bad-token-2",
        },
        origin: "http://localhost:18601",
        source: {} as Window,
      }),
    );

    expect(Storage.getToken()).toBeNull();
    expect(Storage.getProjectId()).toBeNull();
    expect(editorUi.theme.value).toBe("light");
    expect(editorUi.locale.value).toBe("zh");
    expect(i18n.global.locale.value).toBe("zh");
  });

  it("仍支持可信宿主下发的主题与语言更新", () => {
    const editorUi = getEditorUiStore();
    const trustedParent = {} as Window;
    const handler = createRuntimeMessageHandler({
      editorUi,
      i18n,
      getTrustedOriginSet: () => new Set(["http://localhost:18601"]),
      getTrustedSources: () => [trustedParent],
    });

    handler(
      new MessageEvent("message", {
        data: { type: "THEME_UPDATE", theme: "dark" },
        origin: "http://localhost:18601",
        source: trustedParent,
      }),
    );
    handler(
      new MessageEvent("message", {
        data: { type: "LOCALE_UPDATE", locale: "en" },
        origin: "http://localhost:18601",
        source: trustedParent,
      }),
    );

    expect(editorUi.theme.value).toBe("dark");
    expect(editorUi.locale.value).toBe("en");
    expect(i18n.global.locale.value).toBe("en");
  });

  it("preview 路由仍接收认证更新，但不会消费编辑器主题同步", () => {
    const editorUi = getEditorUiStore();
    const trustedParent = {} as Window;
    const handler = createRuntimeMessageHandler({
      editorUi,
      i18n,
      getPathname: () => "/preview",
      getTrustedOriginSet: () => new Set(["http://localhost:18601"]),
      getTrustedSources: () => [trustedParent],
    });

    handler(
      new MessageEvent("message", {
        data: {
          type: "AUTH_REFRESHED",
          token: "preview-token",
          refreshToken: "preview-refresh",
        },
        origin: "http://localhost:18601",
        source: trustedParent,
      }),
    );
    handler(
      new MessageEvent("message", {
        data: { type: "THEME_UPDATE", theme: "dark" },
        origin: "http://localhost:18601",
        source: trustedParent,
      }),
    );

    expect(Storage.getToken()).toBe("preview-token");
    expect(Storage.getRefreshToken()).toBe("preview-refresh");
    expect(editorUi.theme.value).toBe("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("preview 路由会接收认证类 bootstrap，但不会被 bootstrap 改写主题与语言", () => {
    const editorUi = getEditorUiStore();
    const trustedParent = {} as Window;
    const handler = createRuntimeMessageHandler({
      editorUi,
      i18n,
      getPathname: () => "/preview",
      getTrustedOriginSet: () => new Set(["http://localhost:18601"]),
      getTrustedSources: () => [trustedParent],
    });

    handler(
      new MessageEvent("message", {
        data: {
          type: "APP_BOOTSTRAP_RESPONSE",
          payload: {
            token: "preview-bootstrap-token",
            refreshToken: "preview-bootstrap-refresh",
            projectId: "preview-project",
            tenantId: "preview-tenant",
            theme: "dark",
            locale: "en",
          },
        },
        origin: "http://localhost:18601",
        source: trustedParent,
      }),
    );

    expect(Storage.getToken()).toBe("preview-bootstrap-token");
    expect(Storage.getRefreshToken()).toBe("preview-bootstrap-refresh");
    expect(Storage.getProjectId()).toBe("preview-project");
    expect(Storage.getTenantId()).toBe("preview-tenant");
    expect(editorUi.theme.value).toBe("light");
    expect(editorUi.locale.value).toBe("zh");
    expect(i18n.global.locale.value).toBe("zh");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
    expect(document.documentElement.getAttribute("data-theme")).not.toBe("dark");
  });

  it("bootstrap 响应 requestId 不匹配时会忽略该消息，并在超时后失败收敛", async () => {
    vi.useFakeTimers();

    const editorUi = getEditorUiStore();
    const trustedParent = {
      postMessage: vi.fn(),
    } as unknown as Window;

    initializeDesignerHostBootstrap({
      currentUrl: "http://designer.example/designer/?handoffId=handoff-mismatch",
      isTopLevelWindow: false,
      parentWindow: trustedParent,
      referrer: "http://localhost:18601/dashboard",
      selfWindow: window,
    });

    const bootstrapPromise = waitForHostBootstrap();
    const handler = createRuntimeMessageHandler({
      editorUi,
      i18n,
      getTrustedOriginSet: () => new Set(["http://localhost:18601"]),
      getTrustedSources: () => [trustedParent],
    });

    handler(
      new MessageEvent("message", {
        data: {
          type: "APP_BOOTSTRAP_RESPONSE",
          requestId: "wrong-request-id",
          payload: {
            token: "ignored-token",
            projectId: "ignored-project",
            theme: "dark",
            locale: "en",
          },
        },
        origin: "http://localhost:18601",
        source: trustedParent,
      }),
    );

    await vi.runAllTimersAsync();

    await expect(bootstrapPromise).resolves.toBe(false);
    expect(Storage.getToken()).toBeNull();
    expect(Storage.getProjectId()).toBeNull();
    expect(editorUi.theme.value).toBe("light");
    expect(editorUi.locale.value).toBe("zh");
  });

  it("嵌入场景下可信来源集合包含当前页 origin 与父页面 origin", () => {
    const trustedOrigins = resolveTrustedMessageSources({
      locationOrigin: "http://localhost:18603",
      referrer: "http://localhost:18601/dashboard",
    });

    expect([...trustedOrigins]).toEqual(["http://localhost:18603", "http://localhost:18601"]);
  });
});
