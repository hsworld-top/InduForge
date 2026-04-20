import { afterEach, describe, expect, it, vi } from "vitest";
import { STORAGE_KEYS } from "@/constants";
import { Storage } from "./storage";

const store: Record<string, string> = {};

describe("storage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    Object.keys(store).forEach((k) => {
      delete store[k];
    });
  });

  it("get returns default when missing", () => {
    vi.stubGlobal("localStorage", {
      getItem: () => null,
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn(),
    });
    expect(Storage.get("missing", "d")).toBe("d");
  });

  it("set and get round-trip JSON", () => {
    vi.stubGlobal("localStorage", {
      getItem: (key: string) => store[key] ?? null,
      setItem: (key: string, value: string) => {
        store[key] = value;
      },
      removeItem: (key: string) => {
        delete store[key];
      },
      clear: () => {
        Object.keys(store).forEach((k) => {
          delete store[k];
        });
      },
    });
    Storage.set("k", { a: 1 });
    expect(Storage.get<{ a: number }>("k", null)).toEqual({ a: 1 });
  });

  it("theme and language helpers round-trip JSON values", () => {
    vi.stubGlobal("localStorage", {
      getItem: (key: string) => store[key] ?? null,
      setItem: (key: string, value: string) => {
        store[key] = value;
      },
      removeItem: (key: string) => {
        delete store[key];
      },
      clear: () => {
        Object.keys(store).forEach((k) => {
          delete store[k];
        });
      },
    });

    Storage.setTheme("dark");
    Storage.setLanguage("en");

    expect(Storage.getTheme()).toBe("dark");
    expect(Storage.getLanguage()).toBe("en");
    expect(store.theme).toBe(JSON.stringify("dark"));
    expect(store.language).toBe(JSON.stringify("en"));
  });

  it("designer 专属主题和语言键写入时不应污染通用键", () => {
    vi.stubGlobal("localStorage", {
      getItem: (key: string) => store[key] ?? null,
      setItem: (key: string, value: string) => {
        store[key] = value;
      },
      removeItem: (key: string) => {
        delete store[key];
      },
      clear: () => {
        Object.keys(store).forEach((k) => {
          delete store[k];
        });
      },
    });

    Storage.setDesignerTheme("dark");
    Storage.setDesignerLanguage("en");

    expect(store[STORAGE_KEYS.DESIGNER_THEME]).toBe(JSON.stringify("dark"));
    expect(store[STORAGE_KEYS.DESIGNER_LANGUAGE]).toBe(JSON.stringify("en"));
    expect(store[STORAGE_KEYS.THEME]).toBeUndefined();
    expect(store[STORAGE_KEYS.LANGUAGE]).toBeUndefined();
  });

  it("designer 专属主题和语言键优先读取，缺失时才回退通用键", () => {
    vi.stubGlobal("localStorage", {
      getItem: (key: string) => {
        if (key === STORAGE_KEYS.DESIGNER_THEME) {
          return null;
        }
        if (key === STORAGE_KEYS.DESIGNER_LANGUAGE) {
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

    expect(Storage.getDesignerTheme()).toBe("dark");
    expect(Storage.getDesignerLanguage()).toBe("en");
  });

  it("theme and language helpers fall back to defaults on invalid values", () => {
    vi.stubGlobal("localStorage", {
      getItem: (key: string) => {
        if (key === "theme") {
          return JSON.stringify("solarized");
        }
        if (key === "language") {
          return JSON.stringify("jp");
        }
        return null;
      },
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn(),
    });

    expect(Storage.getTheme()).toBe("light");
    expect(Storage.getLanguage()).toBe("zh");
  });
});
