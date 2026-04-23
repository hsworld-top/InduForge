/**
 * 通用 API 返回结构。
 */
export interface ApiResponse<T = unknown> {
  code: number
  msg: string
  data: T
  reqId?: string
}

/**
 * 分页查询参数。
 */
export interface PaginationQuery {
  page?: number
  pageSize?: number
}

/**
 * 分页返回结构。
 */
export interface PaginatedResult<T = unknown> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

/**
 * 通用对象字典类型。
 */
export type ApiRecord = Record<string, unknown>
