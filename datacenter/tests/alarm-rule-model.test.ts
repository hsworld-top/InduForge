import { describe, expect, test } from "vitest";

import {
  alarmContractFields,
  alarmRuntimeBoundaryNotes,
  alarmRuleSamples,
  alarmRuleTypes,
  alarmSeverities,
} from "../src/components/alarm/alarmRuleModel";

describe("alarm rule workspace metadata", () => {
  test("规则类型覆盖阈值、偏差、变化率和表达式", () => {
    expect(alarmRuleTypes.map((type) => type.id)).toEqual([
      "high",
      "low",
      "deviation",
      "rate",
      "expression",
    ]);
  });

  test("示例规则包含构建规则和运行态契约必需字段", () => {
    const sample = alarmRuleSamples[0];

    expect(sample.targetPath).toContain("mqtt.");
    expect(sample.qualityCondition).toContain("quality");
    expect(sample.onlineCondition).toContain("online");
    expect(sample.trigger).toContain("持续");
    expect(sample.recovery).toContain("持续");
    expect(sample.suppression.length).toBeGreaterThan(0);
    expect(sample.messageTemplate).toContain("{{");
    expect(sample.runtimeNodeContract.length).toBeGreaterThan(0);
  });

  test("严重级别按节点侧契约可排序等级定义", () => {
    expect(alarmSeverities.map((severity) => severity.level)).toEqual([
      1, 2, 3, 4,
    ]);
  });

  test("契约字段只描述发布给运行态节点的规则配置", () => {
    const contractKeys = alarmContractFields.map((field) => field.key);

    expect(contractKeys).toContain("targetPath");
    expect(contractKeys).toContain("qualityCondition");
    expect(contractKeys).toContain("ruleType");
    expect(contractKeys).toContain("recoveryPolicy");
    expect(contractKeys).toContain("suppressionPolicy");
    expect(contractKeys).toContain("messageTemplate");
  });

  test("边界文案明确排除数据中心运行态事件管理能力", () => {
    const boundaryText = alarmRuntimeBoundaryNotes.join("\n");

    expect(boundaryText).toContain("只负责构建规则");
    expect(boundaryText).toContain("运行态流程由节点侧负责");
    expect(boundaryText).toContain("事件确认");
    expect(boundaryText).toContain("不提供任何运行态事件管理模块");
    expect(boundaryText).not.toContain("事件时间线");
    expect(boundaryText).not.toContain("最近事件");
    expect(boundaryText).not.toContain("静默");
  });
});
