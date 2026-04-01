import { describe, expect, it } from "vitest";
import { normalizeAssetExt, resolveAssetTypeLabel } from "./asset-meta";

describe("asset-meta", () => {
  it("应将 x-zip-compressed 归一为 zip", () => {
    expect(normalizeAssetExt("x-zip-compressed")).toBe("zip");
    expect(normalizeAssetExt("application/x-zip-compressed")).toBe("zip");
  });

  it("应输出统一大写的资源类型文案", () => {
    expect(resolveAssetTypeLabel({ ext: "jpeg" })).toBe("JPEG");
    expect(resolveAssetTypeLabel({ mimeType: "application/x-zip-compressed" })).toBe("ZIP");
    expect(resolveAssetTypeLabel({ type: "other" })).toBe("OTHER");
  });
});

