import { afterEach, describe, expect, it, vi } from "vitest";
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
});
