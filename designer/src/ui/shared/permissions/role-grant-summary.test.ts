import { describe, expect, it } from "vitest";
import { summarizeRoleGrant } from "./role-grant-summary";

describe("role-grant-summary", () => {
  it("returns inherit summary when grant is empty", () => {
    expect(summarizeRoleGrant()).toBe("继承上级");
    expect(summarizeRoleGrant({})).toBe("继承上级");
  });

  it("summarizes allow deny and inherit counts", () => {
    expect(
      summarizeRoleGrant({
        allowRoles: ["admin", "operator"],
        denyRoles: ["guest"],
        inherit: false,
      }),
    ).toBe("允许 2 / 拒绝 1 / 不继承");
  });

  it("ignores empty role names and duplicate entries", () => {
    expect(
      summarizeRoleGrant({
        allowRoles: ["admin", "admin", " "],
        denyRoles: ["viewer", "", "viewer"],
      }),
    ).toBe("允许 1 / 拒绝 1");
  });
});
