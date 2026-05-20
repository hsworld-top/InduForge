import request from "@/utils/request";
import { listResponseSchema } from "./schemas/common.schema";
import {
  AlarmContractSchema,
  AlarmDraftValidationSchema,
  AlarmRuleSchema,
  AlarmRuleSaveSchema,
  AlarmRuleUpdateSchema,
  AlarmTrialPayloadSchema,
  AlarmTrialResultSchema,
  type AlarmContract,
  type AlarmDraftValidation,
  type AlarmRule,
  type AlarmRuleSave,
  type AlarmRuleUpdate,
  type AlarmTrialPayload,
  type AlarmTrialResult,
} from "./schemas/alarm.schema";

const alarmListSchema = listResponseSchema(AlarmRuleSchema);

type AlarmListResp = {
  list: AlarmRule[];
  pagination?: {
    page?: number;
    pageSize?: number;
    total?: number;
  };
};

const unwrapData = (value: unknown) => {
  if (
    value &&
    typeof value === "object" &&
    "code" in value &&
    "data" in value
  ) {
    return (value as { data?: unknown }).data;
  }
  return value;
};

/** 获取报警规则列表 */
export async function getAlarmRules(
  projectId: string,
  params: Record<string, unknown> = {},
): Promise<AlarmListResp> {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-rules`,
    method: "get",
    params,
  });
  return alarmListSchema.parse(unwrapData(res));
}

/** 获取报警规则详情 */
export async function getAlarmRule(
  projectId: string,
  ruleId: string,
): Promise<AlarmRule> {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-rules/${ruleId}`,
    method: "get",
  });
  return AlarmRuleSchema.parse(unwrapData(res));
}

/** 创建报警规则 */
export async function createAlarmRule(
  projectId: string,
  data: AlarmRuleSave,
): Promise<AlarmRule> {
  const body = AlarmRuleSaveSchema.parse(data);
  const res = await request({
    url: `/data/projects/${projectId}/alarm-rules`,
    method: "post",
    data: body,
  });
  return AlarmRuleSchema.parse(unwrapData(res));
}

/** 更新报警规则 */
export async function updateAlarmRule(
  projectId: string,
  ruleId: string,
  data: AlarmRuleUpdate,
): Promise<AlarmRule> {
  const body = AlarmRuleUpdateSchema.parse(data);
  const res = await request({
    url: `/data/projects/${projectId}/alarm-rules/${ruleId}`,
    method: "put",
    data: body,
  });
  return AlarmRuleSchema.parse(unwrapData(res));
}

/** 启停报警规则 */
export async function toggleAlarmRule(
  projectId: string,
  ruleId: string,
  isEnabled: boolean,
): Promise<AlarmRule> {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-rules/${ruleId}/enabled`,
    method: "patch",
    data: { isEnabled },
  });
  return AlarmRuleSchema.parse(unwrapData(res));
}

/** 删除报警规则 */
export async function deleteAlarmRule(
  projectId: string,
  ruleId: string,
): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/alarm-rules/${ruleId}`,
    method: "delete",
  });
}

/** 试算报警规则 */
export async function testAlarmRule(
  projectId: string,
  ruleId: string,
  payload: AlarmTrialPayload = {},
): Promise<AlarmTrialResult> {
  const body = AlarmTrialPayloadSchema.parse(payload);
  const res = await request({
    url: `/data/projects/${projectId}/alarm-rules/${ruleId}/test`,
    method: "post",
    data: body,
  });
  return AlarmTrialResultSchema.parse(unwrapData(res));
}

/** 获取报警规则契约 */
export async function getAlarmRuleContract(
  projectId: string,
  ruleId: string,
): Promise<AlarmContract> {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-rules/${ruleId}/contract`,
    method: "get",
  });
  return AlarmContractSchema.parse(unwrapData(res));
}

/** 校验未保存草稿 */
export async function validateAlarmRuleDraft(
  projectId: string,
  data: AlarmRuleSave,
): Promise<AlarmDraftValidation> {
  const body = AlarmRuleSaveSchema.parse(data);
  const res = await request({
    url: `/data/projects/${projectId}/alarm-rules/validate-draft`,
    method: "post",
    data: body,
  });
  return AlarmDraftValidationSchema.parse(unwrapData(res));
}
