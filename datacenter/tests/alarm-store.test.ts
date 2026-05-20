import { beforeEach, describe, expect, test, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import type { AlarmRule } from "../src/api/schemas/alarm.schema";

const alarmRule: AlarmRule = {
  id: "rule-1",
  projectId: "project-1",
  name: "温度高报",
  description: "",
  targetDatapointId: "dp-1",
  targetPath: "metrics.temperature",
  targetDataType: "number",
  ruleType: "H",
  condition: { limit: 80 },
  severity: "major",
  isEnabled: true,
  suppression: { enabled: false },
  messageTemplate: "",
  contract: { version: 1 },
  createdAt: "",
  updatedAt: "",
};

const getAlarmRules = vi.fn(async () => ({
  list: [alarmRule],
  pagination: { total: 1 },
}));
const getAlarmRule = vi.fn(async () => alarmRule);
const createAlarmRule = vi.fn(async () => ({ ...alarmRule, id: "rule-2" }));
const updateAlarmRule = vi.fn(async () => ({ ...alarmRule, name: "温度高报2" }));
const deleteAlarmRule = vi.fn(async () => undefined);
const toggleAlarmRule = vi.fn(async () => ({ ...alarmRule, isEnabled: false }));
const testAlarmRule = vi.fn(async () => ({
  triggered: true,
  state: "triggered",
  severity: "major",
  ruleType: "H",
  targetPath: "metrics.temperature",
  diagnostics: {},
}));
const getAlarmRuleContract = vi.fn(async () => ({ version: 1, ruleId: "rule-1" }));
const validateAlarmRuleDraft = vi.fn(async () => ({
  valid: true,
  errors: [],
  contract: { version: 1 },
}));

vi.mock("../src/api/alarm.api", () => ({
  getAlarmRules,
  getAlarmRule,
  createAlarmRule,
  updateAlarmRule,
  deleteAlarmRule,
  toggleAlarmRule,
  testAlarmRule,
  getAlarmRuleContract,
  validateAlarmRuleDraft,
}));

describe("alarm store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  test("拉取列表并记录 total", async () => {
    const { useAlarmStore } = await import("../src/stores/alarm.store");
    const store = useAlarmStore();

    await store.fetchList("project-1", { page: 1 });

    expect(getAlarmRules).toHaveBeenCalledWith("project-1", { page: 1 });
    expect(store.total).toBe(1);
    expect(store.list[0].id).toBe("rule-1");
    expect(store.listError).toBe("");
  });

  test("列表失败时清空列表并记录错误", async () => {
    getAlarmRules.mockRejectedValueOnce(new Error("接口不可用"));
    const { useAlarmStore } = await import("../src/stores/alarm.store");
    const store = useAlarmStore();

    await expect(store.fetchList("project-1")).rejects.toThrow("接口不可用");

    expect(store.list).toEqual([]);
    expect(store.total).toBe(0);
    expect(store.listError).toBe("接口不可用");
    expect(store.loading).toBe(false);
  });

  test("打开详情并写入 editing", async () => {
    const { useAlarmStore } = await import("../src/stores/alarm.store");
    const store = useAlarmStore();

    const detail = await store.openForEdit("project-1", "rule-1");

    expect(getAlarmRule).toHaveBeenCalledWith("project-1", "rule-1");
    expect(detail.id).toBe("rule-1");
    expect(store.editing?.id).toBe("rule-1");
    expect(store.detailError).toBe("");
  });

  test("创建规则后插入列表并设为当前编辑", async () => {
    const { useAlarmStore } = await import("../src/stores/alarm.store");
    const store = useAlarmStore();

    const created = await store.createRule("project-1", {
      name: "温度高报",
      targetPath: "metrics.temperature",
      ruleType: "H",
      condition: { limit: 80 },
      severity: "major",
      isEnabled: true,
      suppression: { enabled: false },
      messageTemplate: "",
    });

    expect(createAlarmRule).toHaveBeenCalled();
    expect(created.id).toBe("rule-2");
    expect(store.list[0].id).toBe("rule-2");
    expect(store.editing?.id).toBe("rule-2");
  });

  test("保存规则后同步详情和列表", async () => {
    const { useAlarmStore } = await import("../src/stores/alarm.store");
    const store = useAlarmStore();
    store.list = [alarmRule];

    const saved = await store.saveRule("project-1", "rule-1", { name: "温度高报2" });

    expect(updateAlarmRule).toHaveBeenCalledWith("project-1", "rule-1", {
      name: "温度高报2",
    });
    expect(saved.name).toBe("温度高报2");
    expect(store.editing?.name).toBe("温度高报2");
    expect(store.list[0].name).toBe("温度高报2");
  });

  test("删除当前规则后清理详情和列表", async () => {
    const { useAlarmStore } = await import("../src/stores/alarm.store");
    const store = useAlarmStore();
    store.list = [alarmRule];
    store.total = 1;
    store.editing = alarmRule;

    await store.removeRule("project-1", "rule-1");

    expect(deleteAlarmRule).toHaveBeenCalledWith("project-1", "rule-1");
    expect(store.list).toEqual([]);
    expect(store.total).toBe(0);
    expect(store.editing).toBeNull();
  });

  test("启停规则后同步列表", async () => {
    const { useAlarmStore } = await import("../src/stores/alarm.store");
    const store = useAlarmStore();
    store.list = [alarmRule];

    const rule = await store.setRuleEnabled("project-1", "rule-1", false);

    expect(toggleAlarmRule).toHaveBeenCalledWith("project-1", "rule-1", false);
    expect(rule.isEnabled).toBe(false);
    expect(store.list[0].isEnabled).toBe(false);
  });

  test("试算、契约和草稿校验写入独立状态", async () => {
    const { useAlarmStore } = await import("../src/stores/alarm.store");
    const store = useAlarmStore();
    const payload = { value: 90 };
    const draft = {
      name: "温度高报",
      targetPath: "metrics.temperature",
      ruleType: "H" as const,
      condition: { limit: 80 },
      severity: "major" as const,
      isEnabled: true,
      suppression: { enabled: false },
      messageTemplate: "",
    };

    await store.runTrial("project-1", "rule-1", payload);
    await store.fetchContract("project-1", "rule-1");
    const validation = await store.validateDraft("project-1", draft);

    expect(testAlarmRule).toHaveBeenCalledWith("project-1", "rule-1", payload);
    expect(store.trial.result?.state).toBe("triggered");
    expect(store.trial.running).toBe(false);
    expect(store.contract.data?.ruleId).toBe("rule-1");
    expect(validation.valid).toBe(true);
    expect(store.validation?.valid).toBe(true);
  });
});
