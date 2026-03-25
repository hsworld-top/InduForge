import { describe, expect, it } from "vitest";
import {
  getPageTreeOrderStorageKey,
  moveIdBefore,
  PAGE_TREE_ORDER_PREFIX,
  validatePageName,
} from "./page-tree-utils";

describe("page-tree-utils", () => {
  describe("validatePageName", () => {
    it("rejects empty", () => {
      expect(validatePageName("  ").valid).toBe(false);
    });

    it("rejects path-like characters", () => {
      expect(validatePageName("a/b").valid).toBe(false);
    });

    it("accepts normal names", () => {
      expect(validatePageName("页面一").valid).toBe(true);
    });
  });

  describe("moveIdBefore", () => {
    it("moves source before target", () => {
      expect(moveIdBefore(["a", "b", "c"], "c", "a")).toEqual(["c", "a", "b"]);
    });

    it("appends when target missing", () => {
      expect(moveIdBefore(["a", "b"], "c", "x")).toEqual(["a", "b", "c"]);
    });
  });

  describe("getPageTreeOrderStorageKey", () => {
    it("includes prefix and project id", () => {
      expect(getPageTreeOrderStorageKey("proj-1")).toBe(`${PAGE_TREE_ORDER_PREFIX}:proj-1`);
    });
  });
});
