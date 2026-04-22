import { describe, expect, it } from "vitest";
import {
  buildPageConfigPatch,
  hydratePageInspectorForm,
  mergePageConfigPatch,
  normalizeWindowStyle,
} from "./page-inspector-config";

describe("page-inspector-config", () => {
  it("maps legacy normal window style to replace", () => {
    expect(normalizeWindowStyle("normal")).toBe("replace");
  });

  it("returns only allowed page config fields", () => {
    const patch = buildPageConfigPatch({
      title: "title",
      description: "desc",
      role: "normal",
      routeMode: "manual",
      routePath: "/custom",
      routeSlug: "custom",
      viewportPreset: "pc",
      width: 1440,
      height: 900,
      autoFit: true,
      lockAspectRatio: true,
      minWidth: 320,
      minHeight: 240,
      overflowMode: "hidden",
      backgroundType: "gradient",
      backgroundValue: "linear-gradient(#fff,#000)",
      backgroundSize: "cover",
      backgroundPosition: "center",
      backgroundRepeat: "no-repeat",
      transitionType: "fade",
      openMode: "popup",
      popupWidth: 800,
      popupHeight: 500,
      popupCenter: true,
      popupMaskClosable: false,
      permissionSummary: "legacy-summary",
      pageViewPermission: {
        allowRoles: ["admin", " operator ", "admin"],
        denyRoles: ["guest"],
        inherit: false,
      },
      cacheMode: "cache",
      preloadMode: "eager",
    });

    expect(patch.meta).toEqual({
      title: "title",
      description: "desc",
    });
    expect(patch.route).toEqual({
      mode: "manual",
      path: "/custom",
      slug: "custom",
    });
    expect(patch.viewport).toEqual({
      preset: "pc",
      width: 1440,
      height: 900,
      autoFit: true,
      lockAspectRatio: true,
      minWidth: 320,
      minHeight: 240,
      overflowMode: "hidden",
    });
    expect(patch.runtime).toEqual({
      openMode: "popup",
      popup: {
        width: 800,
        height: 500,
        center: true,
        maskClosable: false,
      },
      permission: {
        summary: "已配置页面访问权限",
      },
      cacheMode: "cache",
      preloadMode: "eager",
    });
    expect(patch).toMatchObject({
      description: "desc",
      width: 1440,
      height: 900,
      autoFit: true,
      lockAspectRatio: true,
      enableMinSize: true,
      windowStyle: "popup",
      background: {
        kind: "gradient",
        value: "linear-gradient(#fff,#000)",
      },
      runtimePermissions: {
        pageView: {
          allowRoles: ["admin", "operator"],
          denyRoles: ["guest"],
          inherit: false,
        },
      },
    });

    expect("showGrid" in patch).toBe(false);
    expect("enableSnap" in patch).toBe(false);
    expect("windowWidth" in patch).toBe(false);
    expect("windowHeight" in patch).toBe(false);
    expect("x" in patch).toBe(false);
    expect("y" in patch).toBe(false);
  });

  it("does not persist fontAutoFit and clears page permissions when grant is empty", () => {
    const patch = buildPageConfigPatch({
      title: "",
      description: "",
      role: "normal",
      routeMode: "auto",
      routePath: "",
      routeSlug: "",
      viewportPreset: "pc",
      width: 0,
      height: -1,
      autoFit: false,
      lockAspectRatio: true,
      minWidth: 100,
      minHeight: 200,
      overflowMode: "auto",
      backgroundType: "color",
      backgroundValue: "",
      backgroundSize: "cover",
      backgroundPosition: "",
      backgroundRepeat: "no-repeat",
      transitionType: "none",
      openMode: "normal",
      popupWidth: 0,
      popupHeight: 0,
      popupCenter: true,
      popupMaskClosable: true,
      permissionSummary: "",
      pageViewPermission: {
        allowRoles: ["", " "],
        denyRoles: [],
      },
      cacheMode: "default",
      preloadMode: "lazy",
    });

    expect(patch.runtimePermissions).toBeUndefined();
    expect(patch.runtime?.permission?.summary).toBe("0item");
    expect(patch.permissionDesc).toBe("0item");
    expect(patch.windowStyle).toBe("replace");
    expect(patch.enableMinSize).toBe(false);
    expect("fontAutoFit" in patch).toBe(false);
    expect("runtimePermissions" in patch).toBe(false);
  });

  it("removes existing page view permissions from merged config when grant is cleared", () => {
    const patch = buildPageConfigPatch({
      title: "title",
      description: "desc",
      role: "normal",
      routeMode: "auto",
      routePath: "/page",
      routeSlug: "page",
      viewportPreset: "pc",
      width: 1440,
      height: 900,
      autoFit: true,
      lockAspectRatio: false,
      minWidth: 0,
      minHeight: 0,
      overflowMode: "auto",
      backgroundType: "color",
      backgroundValue: "#000000",
      backgroundSize: "cover",
      backgroundPosition: "center",
      backgroundRepeat: "no-repeat",
      transitionType: "none",
      openMode: "cover",
      popupWidth: 960,
      popupHeight: 540,
      popupCenter: true,
      popupMaskClosable: true,
      permissionSummary: "0item",
      pageViewPermission: undefined,
      cacheMode: "default",
      preloadMode: "lazy",
    });

    const merged = mergePageConfigPatch(
      {
        width: 1920,
        height: 1080,
        runtimePermissions: {
          pageView: {
            allowRoles: ["admin"],
            denyRoles: ["guest"],
            inherit: false,
          },
        },
      },
      patch,
      { clearPageViewPermission: true },
    );

    expect(merged.runtimePermissions).toBeUndefined();
  });

  it("hydrates page permission from runtimePermissions.pageView", () => {
    const form = hydratePageInspectorForm({
      name: "页面",
      role: "normal",
      routePath: "/legacy",
      config: {
        runtimePermissions: {
          pageView: {
            allowRoles: ["admin", "operator"],
            denyRoles: ["guest"],
            inherit: false,
          },
        },
      },
    });

    expect(form.pageViewPermission).toEqual({
      allowRoles: ["admin", "operator"],
      denyRoles: ["guest"],
      inherit: false,
    });
    expect(form.permissionSummary).toBe("已配置页面访问权限");
  });
});
