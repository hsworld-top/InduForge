import { z } from "zod";
import { IdSchema, TimeFieldSchema } from "./common.schema";

// 契约检查状态
export const ContractCheckStatusSchema = z.enum([
  "pending",
  "running",
  "passed",
  "failed",
  "error",
]);

export type ContractCheckStatus = z.infer<typeof ContractCheckStatusSchema>;

// 单条检查项结果
export const ContractCheckItemSchema = z
  .object({
    id: IdSchema.optional(),
    name: z.string().optional(),
    status: ContractCheckStatusSchema.or(z.string()).optional(),
    message: z.string().optional().nullable(),
    detail: z.unknown().optional(),
  })
  .passthrough();

export type ContractCheckItem = z.infer<typeof ContractCheckItemSchema>;

// 一次契约检查结果
export const ContractCheckResultSchema = z
  .object({
    id: IdSchema.optional(),
    status: ContractCheckStatusSchema.or(z.string()).optional(),
    items: z.array(ContractCheckItemSchema).optional(),
    passCount: z.number().optional(),
    failCount: z.number().optional(),
    createdAt: TimeFieldSchema,
    finishedAt: TimeFieldSchema,
  })
  .passthrough();

export type ContractCheckResult = z.infer<typeof ContractCheckResultSchema>;
