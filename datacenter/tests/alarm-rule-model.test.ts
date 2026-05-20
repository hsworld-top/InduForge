import { describe, expect, test } from "vitest";

import {
  alarmRuleTypeOptions,
  createAlarmDraft,
  draftToAlarmSavePayload,
  toAlarmDraft,
} from "../src/components/alarm/alarmRuleModel";

describe("alarm rule model", () => {
  test("规则类型覆盖最终枚举", () => {
    expect(alarmRuleTypeOptions.map((item) => item.value)).toEqual([
      "H",
      "L",
      "HH",
      "LL",
      "deviation_high",
      "deviation_low",
      "rate_of_change",
      "cel",
    ]);
  });

  test("新建草稿序列化为最终保存 payload", () => {
    const draft = createAlarmDraft();
    draft.name = "温度高报";
    draft.targetPath = "metrics.temperature";
    draft.ruleType = "H";
    draft.condition.limit = 80;
    draft.severity = "major";

    expect(draftToAlarmSavePayload(draft)).toMatchObject({
      name: "温度高报",
      targetPath: "metrics.temperature",
      ruleType: "H",
      condition: { limit: 80 },
      severity: "major",
    });
  });

  test("详情转草稿保留脏状态为 false", () => {
    const draft = toAlarmDraft({
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

    expect(draft.dirty).toBe(false);
  });
});
