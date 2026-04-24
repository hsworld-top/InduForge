// @ts-nocheck
import { beforeEach, test, vi } from "vitest";
import assert from "node:assert/strict";

import { resolveDatacenterDebugProjectMeta } from "../src/router/debug-project";

const { requestGetMock, storageGetTokenMock } = vi.hoisted(() => ({
  requestGetMock: vi.fn(),
  storageGetTokenMock: vi.fn(),
}));

vi.mock("../src/utils/request", () => ({
  default: {
    get: requestGetMock,
  },
}));

vi.mock("../src/utils/storage", () => ({
  Storage: {
    getToken: storageGetTokenMock,
  },
}));

import { debugProjectAPI } from "../src/api/debug-project.api";

beforeEach(() => {
  requestGetMock.mockReset();
  storageGetTokenMock.mockReset();
});

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

test("debug 默认工程解析失败时回退到本地调试工程", async () => {
  const writes = [];
  const project = await resolveDatacenterDebugProjectMeta({
    targetUrl: "http://datacenter.example/datacenter/debug",
    resolveDefaultDebugProject: async () => {
      throw new Error("访问令牌无效");
    },
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
    id: "test-project",
    tenantId: null,
  });
  assert.deepEqual(writes, [["projectId", "test-project"]]);
});

test("默认工程解析兼容 projects 接口的 data.list.projects 包络", async () => {
  storageGetTokenMock.mockReturnValue("debug-token");
  requestGetMock.mockResolvedValue({
    code: 0,
    msg: "操作成功",
    data: {
      list: {
        projects: [
          {
            id: "project-test",
            name: "test",
            tenantId: "tenant-test",
          },
          {
            id: "project-other",
            name: "test1",
            tenantId: "tenant-other",
          },
        ],
      },
    },
  });

  const project = await debugProjectAPI.resolveDefaultProjectByName();

  assert.deepEqual(project, {
    id: "project-test",
    name: "test",
    tenantId: "tenant-test",
  });
  assert.equal(requestGetMock.mock.calls.length, 1);
});
