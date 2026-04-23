// @ts-nocheck
import { test } from "vitest";
import assert from "node:assert/strict";

import { normalizeConnectionsPayload } from "../src/composables/useConnection";

test("连接列表兼容后端直接返回 data 数组的包络", () => {
  const payload = normalizeConnectionsPayload({
    data: [
      {
        id: "conn-1",
        name: "test1",
        type: "relational",
      },
    ],
  });

  assert.deepEqual(payload, [
    {
      id: "conn-1",
      name: "test1",
      type: "relational",
    },
  ]);
});

test("连接列表继续兼容前端旧的 data.connections 包络", () => {
  const payload = normalizeConnectionsPayload({
    data: {
      connections: [
        {
          id: "conn-legacy",
          name: "legacy",
          type: "mqtt",
        },
      ],
    },
  });

  assert.deepEqual(payload, [
    {
      id: "conn-legacy",
      name: "legacy",
      type: "mqtt",
    },
  ]);
});
