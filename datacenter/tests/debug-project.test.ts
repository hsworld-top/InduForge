// @ts-nocheck
import { test } from "vitest";
import assert from "node:assert/strict";

import { resolveDatacenterDebugProjectMeta } from "../src/router/debug-project";

test("debug 入口未带 pid 时默认解析名称为 test 的工程", async () => {
  const writes = [];
  const project = await resolveDatacenterDebugProjectMeta({
    targetUrl: "http://datacenter.example/datacenter/debug?handoff=handoff-1",
    resolveDefaultDebugProject: async () => ({
      id: "project-test",
      tenantId: "tenant-test",
    }),
    getStoredProjectId: () => null,
    getStoredTenantId: () => null,
    setProjectId: (value) => {
      writes.push(["projectId", value]);
    },
    setTenantId: (value) => {
      writes.push(["tenantId", value]);
    },
  });

  assert.deepEqual(project, {
    id: "project-test",
    tenantId: "tenant-test",
  });
  assert.deepEqual(writes, [
    ["projectId", "project-test"],
    ["tenantId", "tenant-test"],
  ]);
});
