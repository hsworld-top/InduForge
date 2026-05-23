import { z } from 'zod'
import { IdSchema, TimeFieldSchema } from './common.schema'

// 计算单元语言类型
export const ComputeLangSchema = z.enum(['javascript', 'python', 'lua'])

export type ComputeLang = z.infer<typeof ComputeLangSchema>

// 计算单元状态
export const ComputeStatusSchema = z.enum(['idle', 'running', 'error', 'disabled', 'enabled'])

export type ComputeStatus = z.infer<typeof ComputeStatusSchema>

// 计算单元列表项
export const ComputeUnitSchema = z
  .object({
    id: IdSchema,
    name: z.string(),
    path: z.string().optional(),
    outputPath: z.string().optional(),
    lang: ComputeLangSchema.or(z.string()).optional(),
    status: ComputeStatusSchema.or(z.string()).optional(),
    description: z.string().optional().nullable(),
    folderId: IdSchema.optional().nullable(),
    createdAt: TimeFieldSchema,
    updatedAt: TimeFieldSchema,
  })
  .passthrough()

export type ComputeUnit = z.infer<typeof ComputeUnitSchema>

// 计算单元详情（含代码）
export const ComputeUnitDetailSchema = ComputeUnitSchema.extend({
  code: z.string().optional(),
  inputs: z.array(z.record(z.string(), z.unknown())).optional(),
  outputs: z.array(z.record(z.string(), z.unknown())).optional(),
  inputBindings: z.record(z.string(), z.unknown()).optional(),
  outputBindings: z.record(z.string(), z.unknown()).optional(),
  triggerType: z.string().optional(),
  triggerConfig: z.record(z.string(), z.unknown()).optional(),
  timeoutMs: z.number().optional(),
  isEnabled: z.boolean().optional(),
  dependencies: z.array(z.unknown()).optional(),
})

export type ComputeUnitDetail = z.infer<typeof ComputeUnitDetailSchema>

// 计算单元创建/更新参数
export const ComputeUnitSaveSchema = z
  .object({
    name: z.string(),
    lang: z.string().optional(),
    code: z.string().optional(),
    description: z.string().optional().nullable(),
    folderId: IdSchema.optional().nullable(),
    triggerType: z.string().optional(),
    triggerConfig: z.record(z.string(), z.unknown()).optional(),
    inputBindings: z.record(z.string(), z.unknown()).optional(),
    outputBindings: z.record(z.string(), z.unknown()).optional(),
    timeoutMs: z.number().optional(),
    isEnabled: z.boolean().optional(),
    dependencies: z.array(z.unknown()).optional(),
  })
  .passthrough()

export type ComputeUnitSave = z.infer<typeof ComputeUnitSaveSchema>

// 计算单元文件夹创建/更新参数
export const ComputeFolderSaveSchema = z
  .object({
    name: z.string(),
    parentId: IdSchema.optional().nullable(),
  })
  .passthrough()

export type ComputeFolderSave = z.infer<typeof ComputeFolderSaveSchema>

// 计算单元文件夹
export const ComputeFolderSchema = z
  .object({
    id: IdSchema,
    name: z.string(),
    parentId: IdSchema.optional().nullable(),
    children: z.array(z.lazy(() => ComputeFolderSchema)).optional(),
  })
  .passthrough()

export type ComputeFolder = z.infer<typeof ComputeFolderSchema>

export const ComputeDependencySchema = z
  .object({
    id: z.string(),
    name: z.string(),
    runtime: z.string(),
    version: z.string().optional(),
    description: z.string().optional(),
    status: z.string().optional(),
    importName: z.string().optional(),
  })
  .passthrough()

export type ComputeDependency = z.infer<typeof ComputeDependencySchema>

// 调试运行结果
export const ComputeRunResultSchema = z
  .object({
    status: z.string().optional(),
    durationMs: z.number().optional(),
    dryRun: z.boolean().optional(),
    output: z.unknown().optional(),
    sideEffects: z.array(z.unknown()).optional(),
    logs: z.array(z.string()).optional(),
    error: z.string().optional().nullable(),
    errorMessage: z.string().optional().nullable(),
    duration: z.number().optional(),
    startedAt: TimeFieldSchema.optional(),
    finishedAt: TimeFieldSchema.optional(),
  })
  .passthrough()

export type ComputeRunResult = z.infer<typeof ComputeRunResultSchema>

export const ComputeSyntaxDiagnosticSchema = z
  .object({
    severity: z.string(),
    message: z.string(),
    line: z.number(),
    column: z.number(),
    endLine: z.number().optional(),
    endColumn: z.number().optional(),
    source: z.string().optional(),
  })
  .passthrough()

export type ComputeSyntaxDiagnostic = z.infer<typeof ComputeSyntaxDiagnosticSchema>

export const ComputeSyntaxCheckResultSchema = z
  .object({
    diagnostics: z.array(ComputeSyntaxDiagnosticSchema).optional(),
  })
  .passthrough()

export type ComputeSyntaxCheckResult = z.infer<typeof ComputeSyntaxCheckResultSchema>
