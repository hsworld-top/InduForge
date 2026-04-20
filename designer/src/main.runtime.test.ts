import { afterEach, describe, expect, it } from "vitest";
import { i18n } from "./i18n";
import { createRuntimeMessageHandler, resolveTrustedMessageSources } from "./main";
import { getEditorUiStore } from "./stores/editor-ui-store";

describe("main runtime message handler", () => {
  afterEach(() => {
    localStorage.clear();
    document.documentElement.className = "";
    document.documentElement.removeAttribute("data-theme");

    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "light", locale: "zh" });
    i18n.global.locale.value = "zh";
  });

  it("LOCALE_UPDATE 能同步共享 editor UI 状态与 i18n locale", () => {
    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "light", locale: "zh" });
    const trustedParent = {} as Window;
    const handler = createRuntimeMessageHandler({
      editorUi,
      i18n,
      getTrustedOriginSet: () => new Set(["http://localhost:18601"]),
      getTrustedSources: () => [trustedParent],
    });

    handler(
      new MessageEvent("message", {
        data: { type: "LOCALE_UPDATE", locale: "en" },
        origin: "http://localhost:18601",
        source: trustedParent,
      }),
    );

    expect(editorUi.locale.value).toBe("en");
    expect(i18n.global.locale.value).toBe("en");
  });

  it("THEME_UPDATE 能同步共享 editor UI 状态与 DOM 主题", () => {
    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "light", locale: "zh" });
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

    expect(editorUi.theme.value).toBe("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
  });

  it("可信来源消息会通过，非法来源会被忽略", () => {
    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "light", locale: "zh" });
    const trustedParent = {} as Window;
    const handler = createRuntimeMessageHandler({
      editorUi,
      i18n,
      getTrustedOriginSet: () => new Set(["http://localhost:18601"]),
      getTrustedSources: () => [trustedParent],
    });

    handler(
      new MessageEvent("message", {
        data: { type: "LOCALE_UPDATE", locale: "en" },
        origin: "http://evil.example",
        source: trustedParent,
      }),
    );
    handler(
      new MessageEvent("message", {
        data: { type: "LOCALE_UPDATE", locale: "en" },
        origin: "http://localhost:18601",
        source: {} as Window,
      }),
    );
    handler(
      new MessageEvent("message", {
        data: { type: "LOCALE_UPDATE", locale: "en" },
        origin: "http://localhost:18601",
        source: trustedParent,
      }),
    );

    expect(editorUi.locale.value).toBe("en");
    expect(i18n.global.locale.value).toBe("en");
  });

  it("非法消息会被忽略", () => {
    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "light", locale: "zh" });
    const trustedParent = {} as Window;
    const handler = createRuntimeMessageHandler({
      editorUi,
      i18n,
      getTrustedOriginSet: () => new Set(["http://localhost:18601"]),
      getTrustedSources: () => [trustedParent],
    });

    handler(
      new MessageEvent("message", {
        data: { type: "LOCALE_UPDATE", locale: "jp" },
        origin: "http://localhost:18601",
        source: trustedParent,
      }),
    );
    handler(
      new MessageEvent("message", {
        data: { type: "UNKNOWN", theme: "dark" },
        origin: "http://localhost:18601",
        source: trustedParent,
      }),
    );

    expect(editorUi.locale.value).toBe("zh");
    expect(editorUi.theme.value).toBe("light");
    expect(i18n.global.locale.value).toBe("zh");
  });

  it("非法 THEME_UPDATE 载荷会被忽略", () => {
    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "light", locale: "zh" });
    const trustedParent = {} as Window;
    const handler = createRuntimeMessageHandler({
      editorUi,
      i18n,
      getTrustedOriginSet: () => new Set(["http://localhost:18601"]),
      getTrustedSources: () => [trustedParent],
    });

    handler(
      new MessageEvent("message", {
        data: { type: "THEME_UPDATE", theme: "solarized" },
        origin: "http://localhost:18601",
        source: trustedParent,
      }),
    );

    expect(editorUi.theme.value).toBe("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
    expect(document.documentElement.getAttribute("data-theme")).toBe("light");
  });

  it("非对象消息会被忽略", () => {
    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "light", locale: "zh" });
    const trustedParent = {} as Window;
    const handler = createRuntimeMessageHandler({
      editorUi,
      i18n,
      getTrustedOriginSet: () => new Set(["http://localhost:18601"]),
      getTrustedSources: () => [trustedParent],
    });

    handler(
      new MessageEvent("message", {
        data: "THEME_UPDATE",
        origin: "http://localhost:18601",
        source: trustedParent,
      }),
    );
    handler(
      new MessageEvent("message", {
        data: null,
        origin: "http://localhost:18601",
        source: trustedParent,
      }),
    );

    expect(editorUi.theme.value).toBe("light");
    expect(editorUi.locale.value).toBe("zh");
    expect(i18n.global.locale.value).toBe("zh");
  });

  it("preview 路由会忽略宿主主题与语言同步消息", () => {
    const editorUi = getEditorUiStore();
    editorUi.initFromRuntime({ theme: "light", locale: "zh" });
    const trustedParent = {} as Window;
    const handler = createRuntimeMessageHandler({
      editorUi,
      i18n,
      getPathname: () => "/designer/preview",
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

    expect(editorUi.theme.value).toBe("light");
    expect(editorUi.locale.value).toBe("zh");
    expect(i18n.global.locale.value).toBe("zh");
  });

  it("嵌入场景下可信消息来源会包含当前 origin 与父页面 origin", () => {
    const trustedOrigins = resolveTrustedMessageSources({
      locationOrigin: "http://localhost:18603",
      referrer: "http://localhost:18601/dashboard",
    });

    expect([...trustedOrigins]).toEqual(["http://localhost:18603", "http://localhost:18601"]);
  });
});
