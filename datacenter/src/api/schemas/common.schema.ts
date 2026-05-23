import { z } from 'zod'

// 数据中心 v2 统一响应包络与通用模型 schema。
// 第一版策略：可选字段保守 .optional() / .nullable()，关键字段（id/path/name）必填。

export const PaginationSchema = z
  .object({
    page: z.number().int().nonnegative().optional(),
    pageSize: z.number().int().positive().optional(),
    total: z.number().int().nonnegative().optional(),
  })
  .passthrough()

export type Pagination = z.infer<typeof PaginationSchema>

/**
 * 列表响应通用结构：{ list, pagination }。
 * 后端旧接口可能用 `data.datapoints` / `data.items` 等命名，由 API 适配层把它们映射到 list。
 */
export const listResponseSchema = <T extends z.ZodTypeAny>(itemSchema: T) =>
  z
    .object({
      list: z.array(itemSchema).default([]),
      pagination: PaginationSchema.optional(),
    })
    .passthrough()

/**
 * 业务包络。request.ts 已经在拦截器里解包了一次 { code, msg, data, reqId }，
 * 这里只校验 data 部分；如果后端没解包成功直接抛 ApiBusinessError，不到这里。
 */
export const RuntimeGrantSchema = z
  .object({
    inherit: z.boolean().optional(),
    allowRoles: z.array(z.string()).optional(),
    denyRoles: z.array(z.string()).optional(),
  })
  .passthrough()

export type RuntimeGrant = z.infer<typeof RuntimeGrantSchema>

export const TimeFieldSchema = z.string().optional().nullable()

/**
 * 通用 ID 校验：允许字符串与数字混合的项目 / 对象 ID。
 */
export const IdSchema = z.union([z.string(), z.number()])

export type IdValue = z.infer<typeof IdSchema>
