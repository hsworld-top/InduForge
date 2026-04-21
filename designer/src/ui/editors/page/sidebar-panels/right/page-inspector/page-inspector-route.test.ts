import { describe, expect, it } from "vitest";
import { resolveRouteByRole } from "./page-inspector-route";

describe("page-inspector-route", () => {
  it("keeps system page routes fixed by role", () => {
    expect(
      resolveRouteByRole({
        role: "home",
        name: "首页",
      }),
    ).toEqual({
      routePath: "/",
      routeSlug: "",
      routeMode: "auto",
    });

    expect(
      resolveRouteByRole({
        role: "login",
        name: "登录",
      }),
    ).toEqual({
      routePath: "/login",
      routeSlug: "login",
      routeMode: "auto",
    });
  });

  it("preserves manual route path when route mode is manual", () => {
    expect(
      resolveRouteByRole({
        role: "normal",
        name: "报表中心",
        routeMode: "manual",
        routePath: "/custom/report-center",
        routeSlug: "report-center",
        parentRoutePath: "/reports",
      }),
    ).toEqual({
      routePath: "/custom/report-center",
      routeSlug: "report-center",
      routeMode: "manual",
    });
  });

  it("builds business page routes from parent route path and slug", () => {
    expect(
      resolveRouteByRole({
        role: "normal",
        name: "生产总览",
        parentRoutePath: "/reports",
      }),
    ).toEqual({
      routePath: "/reports/生产总览",
      routeSlug: "生产总览",
      routeMode: "auto",
    });
  });
});
