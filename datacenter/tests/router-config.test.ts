// @ts-nocheck
import { test } from "vitest";
import assert from "node:assert/strict";

import { createDatacenterRoutes } from "../src/router/route-config";

test("生产态不会注册 datacenter debug 路由", () => {
  const DataCenterComponent = { name: "DataCenterStub" };
  const routes = createDatacenterRoutes({
    DataCenterComponent,
    enableDebugRoute: false,
  });

  assert.deepEqual(routes, [
    {
      path: "/",
      name: "datacenter",
      component: DataCenterComponent,
      meta: {
        titleKey: "route.datacenter",
        requiresAuth: true,
      },
    },
  ]);
});
