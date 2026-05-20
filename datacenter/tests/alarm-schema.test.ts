import { describe, expect, test } from "vitest";

import {
  AlarmBulkSelectionSchema,
  AlarmPolicySaveSchema,
  AlarmPolicySchema,
  AlarmPolicyTreeSchema,
} from "../src/api/schemas/alarm.schema";

describe("alarm policy schema", () => {
  test("解析多条件报警策略", () => {
    const policy = AlarmPolicySchema.parse({
      id: "policy-1",
      projectId: "project-1",
      groupId: null,
      name: "温度策略",
      mode: "per_target",
      targets: [
        {
          datapointId: "dp-1",
          path: "metrics.temperature",
          dataType: "number",
        },
      ],
      inputs: [],
      derivedExpression: "",
      conditions: [
        {
          id: "c-h",
          type: "H",
          name: "高限",
          isEnabled: true,
          severity: "major",
          params: { limit: 80 },
        },
        {
          id: "c-l",
          type: "L",
          name: "低限",
          isEnabled: true,
          severity: "warning",
          params: { limit: 20 },
        },
      ],
      suppression: {},
      messageTemplate: "",
      isEnabled: true,
      effectiveEnabled: true,
      contract: {},
      createdAt: "2026-05-20T00:00:00Z",
      updatedAt: "2026-05-20T00:00:00Z",
    });

    expect(policy.conditions).toHaveLength(2);
  });

  test("保存 payload 拒绝旧 targetPath 字段", () => {
    expect(() =>
      AlarmPolicySaveSchema.parse({
        name: "策略",
        mode: "per_target",
        targets: [],
        inputs: [],
        derivedExpression: "",
        conditions: [],
        suppression: {},
        messageTemplate: "",
        isEnabled: true,
        targetPath: "metrics.temperature",
      }),
    ).toThrow();
  });

  test("解析筛选后全选选择状态", () => {
    const selection = AlarmBulkSelectionSchema.parse({
      mode: "filtered",
      filters: { search: "温度", enabled: true },
      excludePolicyIds: ["policy-2"],
    });

    expect(selection.mode).toBe("filtered");
  });

  test("解析根目录策略树", () => {
    const tree = AlarmPolicyTreeSchema.parse({
      groups: [],
      rootPolicies: [],
      policies: [],
      matchedPolicyCount: 0,
      totalPolicyCount: 0,
    });

    expect(tree.rootPolicies).toEqual([]);
  });
});
