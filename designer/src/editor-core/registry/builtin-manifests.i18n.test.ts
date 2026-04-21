import { afterEach, describe, expect, it } from "vitest";
import { i18n } from "@/i18n";
import { getManifest } from "@/materials/manifests";
import { registerBuiltinComponents } from "./builtin-manifests";
import { componentRegistry } from "./component-registry";

describe("builtin-manifests i18n", () => {
  afterEach(() => {
    i18n.global.locale.value = "zh";
    componentRegistry.clear();
  });

  it("rebuilds component registry and manifest labels with locale changes", () => {
    registerBuiltinComponents();

    expect(componentRegistry.get("ElContainer")?.name).toBe("区域布局");
    expect(getManifest("HorizontalLayout")?.props.find((item) => item.name === "justify")?.label).toBe(
      "水平排列",
    );
    expect(getManifest("ElContainer")?.props.find((item) => item.name === "showHeader")?.group).toBe(
      "显示",
    );

    i18n.global.locale.value = "en";
    registerBuiltinComponents();

    expect(componentRegistry.get("ElContainer")?.name).toBe("Region Layout");
    expect(getManifest("HorizontalLayout")?.props.find((item) => item.name === "justify")?.label).toBe(
      "Horizontal Distribution",
    );
    expect(getManifest("ElContainer")?.props.find((item) => item.name === "showHeader")?.group).toBe(
      "Display",
    );
  });
});
