import { describe, expect, it } from "vitest";
import {
  buildPageConfigPatch,
  mergePageConfigPatch,
  normalizeWindowStyle,
} from "./page-inspector-config";

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
      pageViewPermission: {
        allowRoles: ["admin", " operator ", "admin"],
        denyRoles: ["guest"],
        inherit: false,
      },
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
    expect("permissionDesc" in patch).toBe(false);
  });

  it("does not persist fontAutoFit and clears page permissions when grant is empty", () => {
    const patch = buildPageConfigPatch({
      description: "",
      width: 0,
      height: -1,
      autoFit: false,
      lockAspectRatio: true,
      enableMinSize: true,
      windowStyle: "normal",
      pageViewPermission: {
        allowRoles: ["", " "],
        denyRoles: [],
      },
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
      background: {
        kind: "color",
        value: "#ffffff",
      },
    });

    expect("fontAutoFit" in patch).toBe(false);
    expect("runtimePermissions" in patch).toBe(false);
  });

  it("removes existing page view permissions from merged config when grant is cleared", () => {
    const patch = buildPageConfigPatch({
      description: "desc",
      width: 1440,
      height: 900,
      autoFit: true,
      lockAspectRatio: false,
      enableMinSize: false,
      windowStyle: "cover",
      pageViewPermission: undefined,
      backgroundKind: "color",
      backgroundValue: "#000000",
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
});
