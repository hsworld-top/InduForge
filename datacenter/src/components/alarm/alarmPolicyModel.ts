import type {
  AlarmBulkSelection,
  AlarmCondition,
  AlarmInputRef,
  AlarmPolicy,
  AlarmPolicyMode,
  AlarmPolicySave,
  AlarmTargetRef,
} from "@/api/schemas/alarm.schema";

export type AlarmPolicyDraft = {
  id?: string;
  groupId: string | null;
  name: string;
  description: string;
  mode: AlarmPolicyMode;
  targets: AlarmTargetRef[];
  inputs: AlarmInputRef[];
  derivedExpression: string;
  conditions: AlarmCondition[];
  suppression: Record<string, unknown>;
  messageTemplate: string;
  isEnabled: boolean;
  dirty: boolean;
};

export function createDefaultAlarmPolicyDraft(): AlarmPolicyDraft {
  return {
    groupId: null,
    name: "",
    description: "",
    mode: "per_target",
    targets: [],
    inputs: [],
    derivedExpression: "",
    conditions: [],
    suppression: { enabled: false },
    messageTemplate: "",
    isEnabled: true,
    dirty: false,
  };
}

export function toAlarmPolicyDraft(policy: AlarmPolicy): AlarmPolicyDraft {
  return {
    id: policy.id,
    groupId: policy.groupId ?? null,
    name: policy.name,
    description: policy.description ?? "",
    mode: policy.mode,
    targets: policy.targets.map((target) => ({ ...target })),
    inputs: policy.inputs.map((input) => ({ ...input })),
    derivedExpression: policy.derivedExpression,
    conditions: cloneConditions(policy.conditions),
    suppression: { ...policy.suppression },
    messageTemplate: policy.messageTemplate,
    isEnabled: policy.isEnabled,
    dirty: false,
  };
}

export function draftToAlarmPolicySavePayload(
  draft: AlarmPolicyDraft,
): AlarmPolicySave {
  return {
    groupId: draft.groupId,
    name: draft.name.trim(),
    description: draft.description.trim() || null,
    mode: draft.mode,
    targets: draft.targets,
    inputs: draft.inputs,
    derivedExpression: draft.derivedExpression.trim(),
    conditions: cloneConditions(draft.conditions),
    suppression: { ...draft.suppression },
    messageTemplate: draft.messageTemplate.trim(),
    isEnabled: draft.isEnabled,
  };
}

export function makeIdSelection(policyIds: string[] = []): AlarmBulkSelection {
  return { mode: "ids", policyIds };
}

export function makeFilteredSelection(
  filters: Record<string, unknown>,
): AlarmBulkSelection {
  return { mode: "filtered", filters, excludePolicyIds: [] };
}

export function togglePolicyInSelection(
  selection: AlarmBulkSelection,
  policyId: string,
  selected: boolean,
): AlarmBulkSelection {
  if (selection.mode === "filtered") {
    const excluded = new Set(selection.excludePolicyIds);
    if (selected) {
      excluded.delete(policyId);
    } else {
      excluded.add(policyId);
    }
    return { ...selection, excludePolicyIds: [...excluded] };
  }

  const ids = new Set(selection.policyIds);
  if (selected) {
    ids.add(policyId);
  } else {
    ids.delete(policyId);
  }
  return { mode: "ids", policyIds: [...ids] };
}

export function createThresholdCondition(
  type: AlarmCondition["type"] = "H",
): AlarmCondition {
  const nameMap: Record<AlarmCondition["type"], string> = {
    HH: "高高限",
    H: "高限",
    L: "低限",
    LL: "低低限",
    deviation_high: "高偏差",
    deviation_low: "低偏差",
    rate_of_change: "变化率",
    cel: "自定义",
  };
  return {
    id: crypto.randomUUID(),
    type,
    name: nameMap[type],
    isEnabled: true,
    severity: "warning",
    params:
      type === "cel"
        ? { expression: "value > 0" }
        : type === "rate_of_change"
          ? { limit: 0, windowMs: 60000, direction: "up" }
          : { limit: 0, hysteresis: 0, durationMs: 0 },
  };
}

function cloneConditions(conditions: AlarmCondition[]): AlarmCondition[] {
  return conditions.map((condition) => ({
    ...condition,
    params: { ...condition.params },
  }));
}
