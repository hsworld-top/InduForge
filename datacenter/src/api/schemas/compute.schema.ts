import { z } from "zod";
import { IdSchema, TimeFieldSchema } from "./common.schema";

// 计算单元语言类型
export const ComputeLangSchema = z.enum(["javascript", "python", "lua"]);

export type ComputeLang = z.infer<typeof ComputeLangSchema>;

// 计算单元状态
export const ComputeStatusSchema = z.enum([
  "idle",
  "running",
  "error",
  "disabled",
]);

export type ComputeStatus = z.infer<typeof ComputeStatusSchema>;

// 计算单元列表项
export const ComputeUnitSchema = z
  .object({
    id: IdSchema,
    name: z.string(),
    path: z.string().optional(),
    lang: ComputeLangSchema.or(z.string()).optional(),
    status: ComputeStatusSchema.or(z.string()).optional(),
    description: z.string().optional().nullable(),
    folderId: IdSchema.optional().nullable(),
    createdAt: TimeFieldSchema,
    updatedAt: TimeFieldSchema,
  })
  .passthrough();

export type ComputeUnit = z.infer<typeof ComputeUnitSchema>;

// 计算单元详情（含代码）
export const ComputeUnitDetailSchema = ComputeUnitSchema.extend({
  code: z.string().optional(),
  inputs: z.array(z.record(z.string(), z.unknown())).optional(),
  outputs: z.array(z.record(z.string(), z.unknown())).optional(),
});

export type ComputeUnitDetail = z.infer<typeof ComputeUnitDetailSchema>;

// 计算单元创建/更新参数
export const ComputeUnitSaveSchema = z
  .object({
    name: z.string(),
    lang: z.string().optional(),
    code: z.string().optional(),
    description: z.string().optional().nullable(),
    folderId: IdSchema.optional().nullable(),
  })
  .passthrough();

export type ComputeUnitSave = z.infer<typeof ComputeUnitSaveSchema>;

// 计算单元文件夹
export const ComputeFolderSchema = z
  .object({
    id: IdSchema,
    name: z.string(),
    parentId: IdSchema.optional().nullable(),
    children: z.array(z.lazy(() => ComputeFolderSchema)).optional(),
  })
  .passthrough();

export type ComputeFolder = z.infer<typeof ComputeFolderSchema>;

// 调试运行结果
export const ComputeRunResultSchema = z
  .object({
    output: z.unknown().optional(),
    logs: z.array(z.string()).optional(),
    error: z.string().optional().nullable(),
    duration: z.number().optional(),
  })
  .passthrough();

export type ComputeRunResult = z.infer<typeof ComputeRunResultSchema>;
