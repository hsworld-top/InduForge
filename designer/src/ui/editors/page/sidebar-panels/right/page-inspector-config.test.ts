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
      runtimeAccessEnabled: true,
      runtimeAccessAllowedRoles: [
        { roleId: "role-1", roleCode: "PROJECT_ADMIN", roleName: "管理员" },
      ],
      runtimePermissionSchemes: [
        {
          id: "scheme-1",
          name: "查看",
          roleRefs: [{ roleId: "role-1", roleCode: "PROJECT_ADMIN", roleName: "管理员" }],
        },
      ],
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
        summary: "legacy-summary",
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
      runtimeAccess: {
        enabled: true,
        allowedRoles: [{ roleId: "role-1", roleCode: "PROJECT_ADMIN", roleName: "管理员" }],
        schemes: [
          {
            id: "scheme-1",
            name: "查看",
            roleRefs: [{ roleId: "role-1", roleCode: "PROJECT_ADMIN", roleName: "管理员" }],
          },
        ],
      },
    });

    expect("showGrid" in patch).toBe(false);
    expect("enableSnap" in patch).toBe(false);
    expect("windowWidth" in patch).toBe(false);
    expect("windowHeight" in patch).toBe(false);
    expect("x" in patch).toBe(false);
    expect("y" in patch).toBe(false);
  });

  it("does not persist fontAutoFit or legacy page permissions", () => {
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
      runtimeAccessEnabled: false,
      runtimeAccessAllowedRoles: [],
      runtimePermissionSchemes: [],
      cacheMode: "default",
      preloadMode: "lazy",
    });

    expect(patch.runtime?.permission?.summary).toBe("0item");
    expect(patch.permissionDesc).toBe("0item");
    expect(patch.windowStyle).toBe("replace");
    expect(patch.enableMinSize).toBe(false);
    expect("fontAutoFit" in patch).toBe(false);
    expect("runtimePermissions" in patch).toBe(false);
  });
});
