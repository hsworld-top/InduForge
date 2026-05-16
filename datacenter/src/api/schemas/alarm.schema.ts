import { z } from "zod";
import { IdSchema, TimeFieldSchema } from "./common.schema";

// 报警级别
export const AlarmLevelSchema = z.enum([
  "critical",
  "major",
  "minor",
  "warning",
  "info",
]);

export type AlarmLevel = z.infer<typeof AlarmLevelSchema>;

// 报警规则状态
export const AlarmRuleStatusSchema = z.enum(["enabled", "disabled"]);

export type AlarmRuleStatus = z.infer<typeof AlarmRuleStatusSchema>;

// 报警规则
export const AlarmRuleSchema = z
  .object({
    id: IdSchema,
    name: z.string(),
    level: AlarmLevelSchema.or(z.string()).optional(),
    status: AlarmRuleStatusSchema.or(z.string()).optional(),
    description: z.string().optional().nullable(),
    condition: z.record(z.string(), z.unknown()).optional(),
    notifyChannels: z.array(z.string()).optional(),
    createdAt: TimeFieldSchema,
    updatedAt: TimeFieldSchema,
  })
  .passthrough();

export type AlarmRule = z.infer<typeof AlarmRuleSchema>;

// 报警规则创建/更新参数
export const AlarmRuleSaveSchema = z
  .object({
    name: z.string(),
    level: z.string().optional(),
    condition: z.record(z.string(), z.unknown()).optional(),
    description: z.string().optional().nullable(),
    notifyChannels: z.array(z.string()).optional(),
  })
  .passthrough();

export type AlarmRuleSave = z.infer<typeof AlarmRuleSaveSchema>;

// 报警试算结果
export const AlarmTrialResultSchema = z
  .object({
    triggered: z.boolean().optional(),
    message: z.string().optional().nullable(),
    matchedDatapoints: z.array(z.string()).optional(),
    evaluatedAt: TimeFieldSchema,
  })
  .passthrough();

export type AlarmTrialResult = z.infer<typeof AlarmTrialResultSchema>;
