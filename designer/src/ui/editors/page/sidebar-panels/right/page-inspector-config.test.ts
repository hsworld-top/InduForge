import { describe, expect, it } from "vitest";
import { buildPageConfigPatch, normalizeWindowStyle } from "./page-inspector-config";

describe("page-inspector-config", () => {
  it("maps legacy normal window style to replace", () => {
    expect(normalizeWindowStyle("normal")).toBe("replace");
  });

  it("builds grouped page config fields while keeping legacy compatibility fields", () => {
    const patch = buildPageConfigPatch({
      title: "",
      description: "desc",
      role: "normal",
      routeMode: "auto",
      routePath: "/demo",
      routeSlug: "demo",
      viewportPreset: "pc",
      width: 1440,
      height: 900,
      autoFit: true,
      lockAspectRatio: true,
      minWidth: 1280,
      minHeight: 720,
      overflowMode: "auto",
      backgroundType: "gradient",
      backgroundValue: "linear-gradient(#fff,#000)",
      backgroundSize: "contain",
      backgroundPosition: "top left",
      backgroundRepeat: "repeat-x",
      transitionType: "fade",
      openMode: "popup",
      popupWidth: 960,
      popupHeight: 540,
      popupCenter: true,
      popupMaskClosable: true,
      permissionSummary: "2item",
      cacheMode: "cache",
      preloadMode: "eager",
    });

    expect(patch.meta).toEqual({
      title: "",
      description: "desc",
    });
    expect(patch.route).toEqual({
      mode: "auto",
      path: "/demo",
      slug: "demo",
    });
    expect(patch.viewport).toEqual({
      preset: "pc",
      width: 1440,
      height: 900,
      autoFit: true,
      lockAspectRatio: true,
      minWidth: 1280,
      minHeight: 720,
      overflowMode: "auto",
    });
    expect(patch.runtime).toEqual({
      openMode: "popup",
      popup: {
        width: 960,
        height: 540,
        center: true,
        maskClosable: true,
      },
      permission: {
        summary: "2item",
      },
      cacheMode: "cache",
      preloadMode: "eager",
    });
    expect(patch.background).toEqual({
      kind: "gradient",
      value: "linear-gradient(#fff,#000)",
      size: "contain",
      position: "top left",
      repeat: "repeat-x",
    });
    expect(patch.transition).toEqual({
      type: "fade",
    });
    expect(patch.width).toBe(1440);
    expect(patch.height).toBe(900);
    expect(patch.autoFit).toBe(true);
    expect(patch.lockAspectRatio).toBe(true);
    expect(patch.windowStyle).toBe("popup");
    expect(patch.permissionDesc).toBe("2item");
    expect(patch.description).toBe("desc");

    expect("showGrid" in patch).toBe(false);
    expect("enableSnap" in patch).toBe(false);
    expect("windowWidth" in patch).toBe(false);
    expect("windowHeight" in patch).toBe(false);
    expect("x" in patch).toBe(false);
    expect("y" in patch).toBe(false);
  });

  it("does not persist fontAutoFit and disables runtime constraints when autoFit is false", () => {
    const patch = buildPageConfigPatch({
      title: "",
      description: "",
      role: "normal",
      routeMode: "auto",
      routePath: "/legacy",
      routeSlug: "legacy",
      viewportPreset: "custom",
      width: 0,
      height: -1,
      autoFit: false,
      lockAspectRatio: true,
      minWidth: 1280,
      minHeight: 720,
      overflowMode: "scroll",
      backgroundType: "color",
      backgroundValue: "",
      backgroundSize: "cover",
      backgroundPosition: "center",
      backgroundRepeat: "no-repeat",
      transitionType: "none",
      openMode: "normal",
      popupWidth: 0,
      popupHeight: 0,
      popupCenter: true,
      popupMaskClosable: true,
      permissionSummary: "",
      cacheMode: "default",
      preloadMode: "lazy",
    });

    expect(patch.viewport).toEqual({
      preset: "custom",
      width: 1920,
      height: 1080,
      autoFit: false,
      lockAspectRatio: false,
      minWidth: 0,
      minHeight: 0,
      overflowMode: "scroll",
    });
    expect(patch.runtime).toEqual({
      openMode: "replace",
      popup: {
        width: 960,
        height: 540,
        center: true,
        maskClosable: true,
      },
      permission: {
        summary: "0item",
      },
      cacheMode: "default",
      preloadMode: "lazy",
    });
    expect(patch.background).toEqual({
      kind: "color",
      value: "#ffffff",
      size: "cover",
      position: "center",
      repeat: "no-repeat",
    });
    expect(patch.enableMinSize).toBe(false);
    expect(patch.windowStyle).toBe("replace");

    expect("fontAutoFit" in patch).toBe(false);
  });
});
