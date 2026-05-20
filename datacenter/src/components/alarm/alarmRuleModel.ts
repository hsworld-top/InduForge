import type {
  AlarmRule,
  AlarmRuleSave,
  AlarmRuleType,
  AlarmSeverity,
} from "@/api/schemas/alarm.schema";

export type AlarmCondition = Record<string, unknown>;
export type AlarmSuppression = Record<string, unknown>;

export type AlarmRuleDraft = {
  id?: string;
  projectId?: string;
  name: string;
  description?: string | null;
  targetDatapointId?: string;
  targetPath: string;
  targetName?: string | null;
  targetDataType?: string;
  ruleType: AlarmRuleType;
  condition: AlarmCondition;
  severity: AlarmSeverity;
  isEnabled: boolean;
  suppression: AlarmSuppression;
  messageTemplate: string;
  contract: Record<string, unknown>;
  dirty: boolean;
};

export const alarmRuleTypeOptions = [
  { value: "H", label: "H 高限" },
  { value: "L", label: "L 低限" },
  { value: "HH", label: "HH 高高限" },
  { value: "LL", label: "LL 低低限" },
  { value: "deviation_high", label: "大偏差" },
  { value: "deviation_low", label: "小偏差" },
  { value: "rate_of_change", label: "变化率" },
  { value: "cel", label: "CEL" },
] as const;

export const alarmSeverityOptions = [
  { value: "info", label: "提示" },
  { value: "warning", label: "警告" },
  { value: "major", label: "重要" },
  { value: "critical", label: "紧急" },
] as const;

export function createAlarmDraft(): AlarmRuleDraft {
  return {
    name: "",
    description: "",
    targetPath: "",
    ruleType: "H",
    condition: {},
    severity: "warning",
    isEnabled: true,
    suppression: { enabled: false },
    messageTemplate: "",
    contract: {},
    dirty: true,
  };
}

export function toAlarmDraft(rule: AlarmRule): AlarmRuleDraft {
  return {
    id: rule.id,
    projectId: rule.projectId,
    name: rule.name,
    description: rule.description ?? "",
    targetDatapointId: rule.targetDatapointId,
    targetPath: rule.targetPath,
    targetName: rule.targetName ?? "",
    targetDataType: rule.targetDataType,
    ruleType: rule.ruleType,
    condition: { ...rule.condition },
    severity: rule.severity,
    isEnabled: rule.isEnabled,
    suppression: { ...rule.suppression },
    messageTemplate: rule.messageTemplate,
    contract: { ...rule.contract },
    dirty: false,
  };
}

export function draftToAlarmSavePayload(draft: AlarmRuleDraft): AlarmRuleSave {
  return {
    name: draft.name,
    description: draft.description ?? "",
    targetPath: draft.targetPath,
    ruleType: draft.ruleType,
    condition: { ...draft.condition },
    severity: draft.severity,
    isEnabled: draft.isEnabled,
    suppression: { ...draft.suppression },
    messageTemplate: draft.messageTemplate,
  };
}
