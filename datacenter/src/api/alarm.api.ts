import request from "@/utils/request";
import { listResponseSchema } from "./schemas/common.schema";
import {
  AlarmRuleSchema,
  AlarmRuleSaveSchema,
  AlarmTrialResultSchema,
  type AlarmRule,
  type AlarmRuleSave,
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

/**
 * 获取报警规则列表。
 * 后端接口待实现，404 由 useApiError 兜底。
 */
export async function getAlarmRules(
  projectId: string,
  params: Record<string, unknown> = {},
): Promise<AlarmListResp> {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-rules`,
    method: "get",
    params,
  });
  return alarmListSchema.parse(res);
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
  return AlarmRuleSchema.parse(res);
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
  return AlarmRuleSchema.parse(res);
}

/** 更新报警规则 */
export async function updateAlarmRule(
  projectId: string,
  ruleId: string,
  data: Partial<AlarmRuleSave>,
): Promise<AlarmRule> {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-rules/${ruleId}`,
    method: "put",
    data,
  });
  return AlarmRuleSchema.parse(res);
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

/**
 * 试算报警规则（模拟触发评估）。
 * 后端接口待实现。
 */
export async function trialAlarmRule(
  projectId: string,
  ruleId: string,
  context: Record<string, unknown> = {},
): Promise<AlarmTrialResult> {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-rules/${ruleId}/trial`,
    method: "post",
    data: context,
  });
  return AlarmTrialResultSchema.parse(res);
}
