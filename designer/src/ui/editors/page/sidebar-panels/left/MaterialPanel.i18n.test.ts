import { afterEach, describe, expect, it } from "vitest";
import { i18n } from "@/i18n";

describe("Material panel i18n messages", () => {
  afterEach(() => {
    i18n.global.locale.value = "zh";
  });

  it("提供物料区与画布模式的中英文文案", () => {
    expect(i18n.global.t("materialPanel.components")).toBe("组件");
    expect(i18n.global.t("materialPanel.resources")).toBe("资源");
    expect(i18n.global.t("materialPanel.canvasTools")).toBe("Canvas 绘图工具");

    i18n.global.locale.value = "en";

    expect(i18n.global.t("materialPanel.components")).toBe("Components");
    expect(i18n.global.t("materialPanel.resources")).toBe("Resources");
    expect(i18n.global.t("materialPanel.canvasTools")).toBe("Canvas Drawing Tools");
  });
});
