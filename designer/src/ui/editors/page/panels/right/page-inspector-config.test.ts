import { describe, expect, it } from "vitest";
import { buildPageConfigPatch, normalizeWindowStyle } from "./page-inspector-config";

describe("page-inspector-config", () => {
  it("maps legacy normal window style to replace", () => {
    expect(normalizeWindowStyle("normal")).toBe("replace");
  });

  it("returns only allowed page config fields", () => {
    const patch = buildPageConfigPatch({
      description: "desc",
      width: 1440,
      height: 900,
      autoFit: true,
      lockAspectRatio: true,
      enableMinSize: true,
      windowStyle: "popup",
      permissionDesc: "2item",
      backgroundKind: "gradient",
      backgroundValue: "linear-gradient(#fff,#000)",
    });

    expect(patch).toEqual({
      description: "desc",
      width: 1440,
      height: 900,
      autoFit: true,
      lockAspectRatio: true,
      enableMinSize: true,
      windowStyle: "popup",
      permissionDesc: "2item",
      background: {
        kind: "gradient",
        value: "linear-gradient(#fff,#000)",
      },
    });

    expect("showGrid" in patch).toBe(false);
    expect("enableSnap" in patch).toBe(false);
    expect("windowWidth" in patch).toBe(false);
    expect("windowHeight" in patch).toBe(false);
    expect("x" in patch).toBe(false);
    expect("y" in patch).toBe(false);
  });

  it("does not persist fontAutoFit and disables runtime constraints when autoFit is false", () => {
    const patch = buildPageConfigPatch({
      description: "",
      width: 0,
      height: -1,
      autoFit: false,
      lockAspectRatio: true,
      enableMinSize: true,
      windowStyle: "normal",
      permissionDesc: "",
      backgroundKind: "color",
      backgroundValue: "",
    });

    expect(patch).toEqual({
      description: "",
      width: 1920,
      height: 1080,
      autoFit: false,
      lockAspectRatio: false,
      enableMinSize: false,
      windowStyle: "replace",
      permissionDesc: "0item",
      background: {
        kind: "color",
        value: "#ffffff",
      },
    });

    expect("fontAutoFit" in patch).toBe(false);
  });
});
