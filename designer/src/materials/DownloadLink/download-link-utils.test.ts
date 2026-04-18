import { describe, expect, it } from "vitest";
import {
  buildDownloadLinkStyle,
  resolveDownloadLinkConfig,
  shouldTriggerByEvent,
} from "./download-link-utils";

describe("download-link-utils", () => {
  it("应解析默认配置（双击触发）", () => {
    const config = resolveDownloadLinkConfig({
      text: "资料.zip",
      href: "/api/file/1",
    });

    expect(config.actionMode).toBe("download");
    expect(config.displayMode).toBe("button");
    expect(config.triggerMode).toBe("double");
    expect(config.downloadFileName).toBe("资料.zip");
  });

  it("应按按钮模式生成样式", () => {
    const config = resolveDownloadLinkConfig({
      displayMode: "button",
      textColor: "#ffffff",
      backgroundColor: "#1677ff",
      borderColor: "#1677ff",
      borderRadius: 8,
      paddingX: 16,
      paddingY: 10,
    });
    const style = buildDownloadLinkStyle(config);

    expect(style.border).toContain("#1677ff");
    expect(style.backgroundColor).toBe("#1677ff");
    expect(style.padding).toBe("10px 16px");
    expect(style.textDecoration).toBe("none");
  });

  it("应按链接模式生成下划线样式", () => {
    const config = resolveDownloadLinkConfig({
      displayMode: "link",
      underline: true,
      textColor: "#2563eb",
    });
    const style = buildDownloadLinkStyle(config);

    expect(style.backgroundColor).toBe("transparent");
    expect(style.border).toBe("none");
    expect(style.textDecoration).toBe("underline");
  });

  it("应按触发模式判定事件", () => {
    expect(shouldTriggerByEvent("single", "click")).toBe(true);
    expect(shouldTriggerByEvent("single", "dblclick")).toBe(false);
    expect(shouldTriggerByEvent("double", "click")).toBe(false);
    expect(shouldTriggerByEvent("double", "dblclick")).toBe(true);
  });
});
