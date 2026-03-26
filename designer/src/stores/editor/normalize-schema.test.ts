import { describe, expect, it } from "vitest";
import { isProjectSchemaPayload, normalizePageList, normalizePageSchema } from "./normalize-schema";

const MISSING_PAGE_LIST_REGEX = /页面列表缺失/;
const EXPECT_ARRAY_REGEX = /期望数组/;

describe("normalize-schema", () => {
  describe("normalizePageList", () => {
    it("returns array as-is", () => {
      expect(normalizePageList([{ id: "a" }])).toEqual([{ id: "a" }]);
    });

    it("throws when null", () => {
      expect(() => normalizePageList(null)).toThrow(MISSING_PAGE_LIST_REGEX);
    });

    it("throws when not array", () => {
      expect(() => normalizePageList({})).toThrow(EXPECT_ARRAY_REGEX);
    });
  });

  describe("normalizePageSchema", () => {
    it("unwraps envelope with schema", () => {
      const inner = { pagesById: {}, nodesById: {}, graphicsById: {}, entry: {} };
      expect(normalizePageSchema({ schema: inner })).toBe(inner);
    });

    it("returns payload when no schema key", () => {
      const p = { foo: 1 };
      expect(normalizePageSchema(p)).toBe(p);
    });
  });

  describe("isProjectSchemaPayload", () => {
    it("is true for full project shape", () => {
      expect(
        isProjectSchemaPayload({
          pagesById: {},
          nodesById: {},
          graphicsById: {},
          entry: {},
        }),
      ).toBe(true);
    });

    it("is false for partial", () => {
      expect(isProjectSchemaPayload({ pagesById: {} })).toBe(false);
    });
  });
});
