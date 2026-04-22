import { describe, expect, it } from "vitest";
import type { Action, RolePermission } from "@/editor-core/document/types";
import { updateNodeActionPermission } from "./property-panel-action-permissions";

describe("property-panel-action-permissions", () => {
  it("writes permissions back to the targeted action", () => {
    const permission: RolePermission = {
      allowRoles: ["admin"],
      denyRoles: ["guest"],
      inherit: false,
    };
    const nextEvents = updateNodeActionPermission(
      {
        click: [
          { type: "navigate", config: { to: "/dashboard" } } satisfies Action,
          { type: "notify", config: { message: "done" } } satisfies Action,
        ],
      },
      "click",
      1,
      permission,
    );

    expect(nextEvents.click?.[0]).toEqual({
      type: "navigate",
      config: { to: "/dashboard" },
    });
    expect(nextEvents.click?.[1]).toEqual({
      type: "notify",
      config: { message: "done" },
      permissions: permission,
    });
  });

  it("removes permissions from the targeted action when cleared", () => {
    const nextEvents = updateNodeActionPermission(
      {
        click: [
          {
            type: "notify",
            config: { message: "done" },
            permissions: {
              allowRoles: ["admin"],
              inherit: false,
            },
          } satisfies Action,
        ],
      },
      "click",
      0,
      undefined,
    );

    expect(nextEvents.click?.[0]).toEqual({
      type: "notify",
      config: { message: "done" },
    });
    expect("permissions" in (nextEvents.click?.[0] || {})).toBe(false);
  });
});
