import { afterEach, describe, expect, it, vi } from "vitest";
import { createEditorUiStore, getEditorUiStore } from "./editor-ui-store";
import zhCn from "element-plus/es/locale/lang/zh-cn";
import en from "element-plus/es/locale/lang/en";
import { i18n } from "@/i18n";
import { STORAGE_KEYS } from "@/constants";

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

  it("非法输入会回退到本地缓存和默认值", () => {
    stubStorage();
    store.theme = JSON.stringify("dark");

    const ui = createEditorUiStore();
    ui.initFromRuntime({ theme: "solarized", locale: "jp" });

    expect(ui.theme.value).toBe("dark");
    expect(ui.locale.value).toBe("zh");
    expect(ui.elementLocale.value).toBe(zhCn);
    expect(i18n.global.locale.value).toBe("zh");
  });

  it("切换主题时同步更新 DOM 与本地缓存", () => {
    stubStorage();

    const ui = createEditorUiStore();
    ui.initFromRuntime();
    ui.setTheme("dark");

    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    expect(store[STORAGE_KEYS.DESIGNER_THEME]).toBe(JSON.stringify("dark"));
    expect(store[STORAGE_KEYS.THEME]).toBeUndefined();
  });

  it("切换语言时同步更新本地缓存与 Element Plus locale", () => {
    stubStorage();

    const ui = createEditorUiStore();
    ui.initFromRuntime();
    ui.setLocale("en");

    expect(ui.locale.value).toBe("en");
    expect(ui.elementLocale.value).toBe(en);
    expect(i18n.global.locale.value).toBe("en");
    expect(store[STORAGE_KEYS.DESIGNER_LANGUAGE]).toBe(JSON.stringify("en"));
    expect(store[STORAGE_KEYS.LANGUAGE]).toBeUndefined();
  });

  it("无运行时输入时优先读取 designer 专属键，缺失时回退通用键", () => {
    stubStorage();
    store[STORAGE_KEYS.DESIGNER_THEME] = JSON.stringify("dark");
    store[STORAGE_KEYS.DESIGNER_LANGUAGE] = JSON.stringify("en");
    store[STORAGE_KEYS.THEME] = JSON.stringify("light");
    store[STORAGE_KEYS.LANGUAGE] = JSON.stringify("zh");

    const ui = createEditorUiStore();
    ui.initFromRuntime();

    expect(ui.theme.value).toBe("dark");
    expect(ui.locale.value).toBe("en");
    expect(store[STORAGE_KEYS.DESIGNER_THEME]).toBe(JSON.stringify("dark"));
    expect(store[STORAGE_KEYS.DESIGNER_LANGUAGE]).toBe(JSON.stringify("en"));
  });

  it("designer 专属键缺失时才会回退读取通用键", () => {
    vi.stubGlobal("localStorage", {
      getItem: (key: string) => {
        if (key === STORAGE_KEYS.DESIGNER_THEME || key === STORAGE_KEYS.DESIGNER_LANGUAGE) {
          return null;
        }
        if (key === STORAGE_KEYS.THEME) {
          return JSON.stringify("dark");
        }
        if (key === STORAGE_KEYS.LANGUAGE) {
          return JSON.stringify("en");
        }
        return null;
      },
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn(),
    });

    const ui = createEditorUiStore();
    ui.initFromRuntime();

    expect(ui.theme.value).toBe("dark");
    expect(ui.locale.value).toBe("en");
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
