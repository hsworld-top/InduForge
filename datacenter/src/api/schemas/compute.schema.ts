import { z } from 'zod'
import { IdSchema, TimeFieldSchema } from './common.schema'
import { CanonicalDataPointTypeSchema } from './source-output.schema'

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

export const ComputeOutputInputSchema = z.object({
  id: IdSchema.optional(),
  key: z.string().min(1).max(100),
  name: z.string().min(1).max(100),
  path: z.string().min(1).max(255),
  dataType: CanonicalDataPointTypeSchema,
  unit: z.string().optional().nullable(),
  precisionNum: z.number().int().nonnegative().optional().nullable(),
  nullPolicy: z.enum(['error', 'skip']).default('error'),
  description: z.string().optional().nullable(),
})

export const ComputeOutputSchema = ComputeOutputInputSchema.extend({
  id: IdSchema,
  datapointId: IdSchema,
  sortOrder: z.number().int().nonnegative().default(0),
})

export type ComputeOutputInput = z.infer<typeof ComputeOutputInputSchema>
export type ComputeOutput = z.infer<typeof ComputeOutputSchema>

// 计算单元详情（含代码）
export const ComputeUnitDetailSchema = ComputeUnitSchema.extend({
  code: z.string().optional(),
  inputs: z.array(z.record(z.string(), z.unknown())).optional(),
  outputs: z.array(ComputeOutputSchema).min(1),
  inputBindings: z.record(z.string(), z.unknown()).optional(),
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
    // 新建弹窗允许省略输出，由后端生成标准 result 输出；编辑详情仍要求至少一个输出。
    outputs: z.array(ComputeOutputInputSchema).default([]),
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
    path: z.string().optional(),
    hasChildren: z.boolean().default(false),
    unitCount: z.number().int().nonnegative().default(0),
  })
  .passthrough()

export type ComputeFolder = z.infer<typeof ComputeFolderSchema>

export const ComputeFolderPageSchema = z
  .object({
    list: z.array(ComputeFolderSchema).default([]),
    pagination: z.object({
      page: z.number().int().positive(),
      pageSize: z.number().int().positive(),
      total: z.number().int().nonnegative(),
      totalPages: z.number().int().nonnegative(),
    }),
  })
  .passthrough()

export type ComputeFolderPage = z.infer<typeof ComputeFolderPageSchema>

export const ComputeDependencySchema = z
  .object({
    id: z.string(),
    name: z.string(),
    runtime: z.string(),
    version: z.string().optional(),
    description: z.string().optional(),
    status: z.string().optional(),
    importName: z.string().optional(),
    referenceCount: z.number().optional().default(0),
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

export const ComputeCapabilitiesSchema = z.object({
  sandboxStatus: z.enum(['available', 'unavailable']),
  sandboxReason: z.string().optional().default(''),
  languages: z.array(z.object({ language: z.string(), version: z.string() })),
  sdk: z.array(z.string()),
  dependencies: z.array(z.object({ language: z.string(), name: z.string(), version: z.string() })),
  triggerTypes: z.array(z.string()),
  limits: z.object({
    maxExecutionTimeMs: z.number(),
    maxInputBytes: z.number(),
    maxOutputBytes: z.number(),
    maxLogBytes: z.number(),
  }),
})

export type ComputeCapabilities = z.infer<typeof ComputeCapabilitiesSchema>
