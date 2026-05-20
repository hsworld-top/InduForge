import { z } from "zod";
import { TimeFieldSchema } from "./common.schema";

const ObjectRecordSchema = z.record(z.string(), z.unknown());

export const AlarmRuleTypeSchema = z.enum([
  "H",
  "L",
  "HH",
  "LL",
  "deviation_high",
  "deviation_low",
  "rate_of_change",
  "cel",
]);

export type AlarmRuleType = z.infer<typeof AlarmRuleTypeSchema>;

export const AlarmSeveritySchema = z.enum([
  "info",
  "warning",
  "major",
  "critical",
]);

export type AlarmSeverity = z.infer<typeof AlarmSeveritySchema>;

export const AlarmTrialStateSchema = z.enum([
  "triggered",
  "not_triggered",
  "insufficient_input",
]);

export type AlarmTrialState = z.infer<typeof AlarmTrialStateSchema>;

export const AlarmRuleSchema = z
  .object({
    id: z.string(),
    projectId: z.string(),
    name: z.string(),
    description: z.string().optional().nullable(),
    targetDatapointId: z.string(),
    targetPath: z.string(),
    targetName: z.string().optional().nullable(),
    targetDataType: z.string(),
    ruleType: AlarmRuleTypeSchema,
    condition: ObjectRecordSchema,
    severity: AlarmSeveritySchema,
    isEnabled: z.boolean(),
    suppression: ObjectRecordSchema,
    messageTemplate: z.string(),
    contract: ObjectRecordSchema,
    createdAt: TimeFieldSchema,
    updatedAt: TimeFieldSchema,
  })
  .passthrough();

export type AlarmRule = z.infer<typeof AlarmRuleSchema>;

export const AlarmRuleSaveSchema = z
  .object({
    name: z.string(),
    description: z.string().optional().nullable(),
    targetPath: z.string(),
    ruleType: AlarmRuleTypeSchema,
    condition: ObjectRecordSchema,
    severity: AlarmSeveritySchema,
    isEnabled: z.boolean(),
    suppression: ObjectRecordSchema,
    messageTemplate: z.string(),
  })
  .passthrough();

export type AlarmRuleSave = z.infer<typeof AlarmRuleSaveSchema>;

export const AlarmRuleUpdateSchema = AlarmRuleSaveSchema.partial();

export type AlarmRuleUpdate = z.infer<typeof AlarmRuleUpdateSchema>;

export const AlarmTrialPayloadSchema = z
  .object({
    value: z.unknown().optional(),
    timestamp: z.string().optional(),
    context: ObjectRecordSchema.optional(),
  })
  .passthrough();

export type AlarmTrialPayload = z.infer<typeof AlarmTrialPayloadSchema>;

export const AlarmTrialResultSchema = z
  .object({
    triggered: z.boolean(),
    state: AlarmTrialStateSchema,
    severity: AlarmSeveritySchema,
    ruleType: AlarmRuleTypeSchema,
    targetPath: z.string(),
    message: z.string().optional().nullable(),
    diagnostics: ObjectRecordSchema,
  })
  .passthrough();

export type AlarmTrialResult = z.infer<typeof AlarmTrialResultSchema>;

export const AlarmDraftValidationSchema = z
  .object({
    valid: z.boolean(),
    errors: z.array(z.unknown()).default([]),
    target: ObjectRecordSchema.optional(),
    contract: ObjectRecordSchema.optional(),
  })
  .passthrough();

export type AlarmDraftValidation = z.infer<typeof AlarmDraftValidationSchema>;

export const AlarmContractSchema = ObjectRecordSchema;

export type AlarmContract = z.infer<typeof AlarmContractSchema>;
