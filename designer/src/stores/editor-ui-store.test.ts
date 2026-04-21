import { afterEach, describe, expect, it, vi } from "vitest";
import { createEditorUiStore, getEditorUiStore } from "./editor-ui-store";
import zhCn from "element-plus/es/locale/lang/zh-cn";
import en from "element-plus/es/locale/lang/en";
import { i18n } from "@/i18n";
const store: Record<string, string> = {};

function stubStorage() {
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => store[key] ?? null,
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      Object.keys(store).forEach((key) => delete store[key]);
    },
  });
}

describe("editor-ui-store", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    document.documentElement.className = "";
    document.documentElement.removeAttribute("data-theme");
    i18n.global.locale.value = "zh";
    Object.keys(store).forEach((key) => delete store[key]);
  });

  it("优先使用 URL 中的 theme 和 locale", () => {
    stubStorage();
    store.theme = JSON.stringify("light");
    store.language = JSON.stringify("zh");

    const ui = createEditorUiStore();
    ui.initFromRuntime({ theme: "dark", locale: "en" });

    expect(ui.theme.value).toBe("dark");
    expect(ui.locale.value).toBe("en");
    expect(ui.elementLocale.value).toBe(en);
    expect(i18n.global.locale.value).toBe("en");
  });

  it("非法输入会固定回退到默认值", () => {
    stubStorage();
    store.theme = JSON.stringify("dark");
    store.language = JSON.stringify("en");

    const ui = createEditorUiStore();
    ui.initFromRuntime({ theme: "solarized", locale: "jp" });

    expect(ui.theme.value).toBe("light");
    expect(ui.locale.value).toBe("zh");
    expect(ui.elementLocale.value).toBe(zhCn);
    expect(i18n.global.locale.value).toBe("zh");
  });

  it("切换主题时只更新 DOM，不写 designer 专属缓存", () => {
    stubStorage();

    const ui = createEditorUiStore();
    ui.initFromRuntime();
    ui.setTheme("dark");

    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    expect(Object.keys(store)).toEqual([]);
  });

  it("切换语言时不写 designer 专属缓存，但会更新 Element Plus locale", () => {
    stubStorage();

    const ui = createEditorUiStore();
    ui.initFromRuntime();
    ui.setLocale("en");

    expect(ui.locale.value).toBe("en");
    expect(ui.elementLocale.value).toBe(en);
    expect(i18n.global.locale.value).toBe("en");
    expect(Object.keys(store)).toEqual([]);
  });

  it("无运行时输入时固定使用默认 light + zh，而不是读取本地缓存", () => {
    stubStorage();
    store.theme = JSON.stringify("dark");
    store.language = JSON.stringify("en");
    store.designer_theme = JSON.stringify("dark");
    store.designer_language = JSON.stringify("en");

    const ui = createEditorUiStore();
    ui.initFromRuntime();

    expect(ui.theme.value).toBe("light");
    expect(ui.locale.value).toBe("zh");
  });

  it("多次获取时返回同一状态源", () => {
    stubStorage();

    const first = createEditorUiStore();
    const second = getEditorUiStore();

    expect(first).toBe(second);
    expect(first.theme).toBe(second.theme);
    expect(first.locale).toBe(second.locale);

    first.setTheme("dark");
    first.setLocale("en");

    expect(second.theme.value).toBe("dark");
    expect(second.locale.value).toBe("en");
    expect(i18n.global.locale.value).toBe("en");
  });
});
