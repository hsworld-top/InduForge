import { describe, expect, test } from "vitest";

import {
  AlarmRuleSchema,
  AlarmTrialResultSchema,
} from "../src/api/schemas/alarm.schema";

describe("alarm schema", () => {
  test("解析最终报警规则", () => {
    const rule = AlarmRuleSchema.parse({
      id: "rule-1",
      projectId: "project-1",
      name: "温度高报",
      targetDatapointId: "dp-1",
      targetPath: "metrics.temperature",
      targetDataType: "number",
      ruleType: "H",
      condition: { limit: 80 },
      severity: "major",
      isEnabled: true,
      suppression: { enabled: false },
      messageTemplate: "",
      contract: {},
      createdAt: "2026-05-20T00:00:00Z",
      updatedAt: "2026-05-20T00:00:00Z",
    });

    expect(rule.ruleType).toBe("H");
  });

  test("解析试算输入不足状态", () => {
    const result = AlarmTrialResultSchema.parse({
      triggered: false,
      state: "insufficient_input",
      severity: "warning",
      ruleType: "deviation_high",
      targetPath: "metrics.temperature",
      diagnostics: {},
    });

    expect(result.state).toBe("insufficient_input");
  });
});
