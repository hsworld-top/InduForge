import { describe, expect, it } from "vitest";
import { canEditRuntimeConstraint, getPageInspectorSections } from "./page-inspector-sections";

describe("page-inspector-sections", () => {
  it("returns fixed section order", () => {
    const sections = getPageInspectorSections();
    expect(sections.map((section) => section.key)).toEqual(["basic", "visual", "runtime"]);
    expect(sections.map((section) => section.title)).toEqual(["基本", "视觉", "运行"]);
  });

  it("contains required runtime fields", () => {
    const runtimeSection = getPageInspectorSections().find((section) => section.key === "runtime");
    expect(runtimeSection?.fields).toEqual([
      "autoFit",
      "lockAspectRatio",
      "enableMinSize",
      "windowStyle",
      "permissionDesc",
    ]);
  });

  it("runtime constraint editability follows autoFit", () => {
    expect(canEditRuntimeConstraint(true)).toBe(true);
    expect(canEditRuntimeConstraint(false)).toBe(false);
  });
});
