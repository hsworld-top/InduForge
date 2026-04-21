import { afterEach, describe, expect, it } from "vitest";
import { i18n } from "@/i18n";
import { canEditRuntimeConstraint, getPageInspectorSections } from "./page-inspector-sections";

describe("page-inspector-sections", () => {
  afterEach(() => {
    i18n.global.locale.value = "zh";
  });

  it("returns localized section titles", () => {
    expect(getPageInspectorSections().map((section) => section.key)).toEqual([
      "basic",
      "visual",
      "runtime",
    ]);
    expect(getPageInspectorSections().map((section) => section.title)).toEqual([
      "基本",
      "视觉",
      "运行",
    ]);

    i18n.global.locale.value = "en";

    expect(getPageInspectorSections().map((section) => section.title)).toEqual([
      "Basic",
      "Visual",
      "Runtime",
    ]);
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
