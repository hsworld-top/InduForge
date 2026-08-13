/**
 * HTTP 请求封装（designer）
 *
 * 统一响应契约：
 * - 成功/业务失败均为 HTTP 2xx，包络为 { code, msg, data, reqId }
 * - 仅 code===0 视为成功，其余 code 统一抛业务异常
 */

import type {
  AxiosError,
  AxiosInstance,
  AxiosRequestConfig,
  InternalAxiosRequestConfig,
} from 'axios'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { getCurrentTenantId } from '@/runtime/wujie-context'

const DEFAULT_BUSINESS_ERROR_CODE = 30000
const DIGITS_ONLY_RE = /^\d+$/

type ErrorResponseData = {
  code?: number | string
  msg?: string
  reqId?: string
  errors?: Record<string, string[]>
  data?: unknown
  [key: string]: unknown
}

export interface ApiResponsePayload<T = unknown> {
  code: number
  msg: string
  data?: T
  reqId?: string
}

export interface ApiErrorMeta {
  code?: number | undefined
  msg: string
  reqId?: string | undefined
  status?: number | undefined
  data?: unknown
}

export interface UnwrappedHttpClient {
  get: <T = unknown>(url: string, config?: AxiosRequestConfig) => Promise<T>
  post: <T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig) => Promise<T>
  put: <T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig) => Promise<T>
  patch: <T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig) => Promise<T>
  delete: <T = unknown>(url: string, config?: AxiosRequestConfig) => Promise<T>
  interceptors: AxiosInstance['interceptors']
  defaults: AxiosInstance['defaults']
}

/** Element Plus 的 ElMessage 选项类型在部分 TS 配置下过窄，此处收窄为运行时实际用法 */
function notifyRequestError(message: string): void {
  ;(ElMessage as unknown as (opts: { type: 'error'; message: string }) => void)({
    type: 'error',
    message,
  })
}

function hasOwn(target: Record<string, unknown>, key: string): boolean {
  return Object.prototype.hasOwnProperty.call(target, key)
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' ? (value as Record<string, unknown>) : null
}

function toNumericCode(value: unknown): number | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string' && DIGITS_ONLY_RE.test(value.trim())) {
    return Number.parseInt(value.trim(), 10)
  }
  return undefined
}

/**
 * 包络识别规则：
 * 1. 必须能解析出 code；
 * 2. 且包含 msg 或包络标识字段（data/reqId/msg 任一显式存在）。
 */
function isApiResponsePayload(
  value: unknown,
): value is Partial<ApiResponsePayload<unknown>> & Record<string, unknown> {
  if (!value || typeof value !== 'object') {
    return false
  }

  const payload = value as Record<string, unknown>
  const code = toNumericCode(payload.code)
  if (code === undefined) {
    return false
  }

  const hasStringMsg = typeof payload.msg === 'string'
  const hasEnvelopeMarker =
    hasOwn(payload, 'msg') || hasOwn(payload, 'data') || hasOwn(payload, 'reqId')

  return hasStringMsg || hasEnvelopeMarker
}

function normalizeApiResponsePayload<T = unknown>(value: unknown): ApiResponsePayload<T> | null {
  if (!isApiResponsePayload(value)) {
    return null
  }

  const normalizedCode = toNumericCode(value.code) ?? DEFAULT_BUSINESS_ERROR_CODE
  const normalizedMsg = typeof value.msg === 'string' ? value.msg : ''
  const normalizedReqId = typeof value.reqId === 'string' ? value.reqId : undefined

  return {
    code: normalizedCode,
    msg: normalizedMsg,
    data: value.data as T,
    ...(normalizedReqId ? { reqId: normalizedReqId } : {}),
  }
}

export class ApiBusinessError extends Error {
  code: number
  reqId: string | undefined
  data: unknown
  status: number | undefined
  isBusinessError = true

  constructor(payload: ApiResponsePayload<unknown>, status?: number) {
    super(payload.msg || '请求失败')
    this.name = 'ApiBusinessError'
    this.code = payload.code
    this.reqId = payload.reqId
    this.data = payload.data
    this.status = status
  }
}

export const isApiBusinessError = (error: unknown): error is ApiBusinessError => {
  return (
    error instanceof ApiBusinessError ||
    (typeof error === 'object' &&
      error !== null &&
      (error as { isBusinessError?: unknown }).isBusinessError === true)
  )
}

const pickAxiosErrorData = (error: unknown): ErrorResponseData | undefined => {
  if (!error || typeof error !== 'object' || !('response' in error)) {
    return undefined
  }
  return (error as { response?: { data?: ErrorResponseData } }).response?.data
}

const pickAxiosStatus = (error: unknown): number | undefined => {
  if (!error || typeof error !== 'object' || !('response' in error)) {
    return undefined
  }
  const status = (error as { response?: { status?: number } }).response?.status
  return typeof status === 'number' ? status : undefined
}

/**
 * 统一提取 API 错误：
 * - 业务失败（2xx + code!=0）返回业务 code/msg/reqId/data
 * - 技术失败（4xx/5xx）返回 status 与响应 msg（若有）
 */
export const resolveApiError = (error: unknown, fallback = '请求失败'): ApiErrorMeta => {
  if (isApiBusinessError(error)) {
    return {
      code: error.code,
      msg: error.message || fallback,
      reqId: error.reqId,
      status: error.status,
      data: error.data,
    }
  }

  const data = pickAxiosErrorData(error)
  const status = pickAxiosStatus(error)
  const responseCode = toNumericCode(data?.code)
  const responseMsg = typeof data?.msg === 'string' ? data.msg : ''
  const responseReqId = typeof data?.reqId === 'string' ? data.reqId : undefined
  const nativeMessage = error instanceof Error ? error.message : ''

  return {
    code: responseCode,
    msg: responseMsg || nativeMessage || fallback,
    reqId: responseReqId,
    status,
    data: data?.data,
  }
}

export const getApiErrorMessage = (error: unknown, fallback = '请求失败'): string => {
  return resolveApiError(error, fallback).msg
}

export const getApiErrorCode = (error: unknown): number | undefined => {
  return resolveApiError(error).code
}

export const getApiErrorReqId = (error: unknown): string | undefined => {
  return resolveApiError(error).reqId
}

export const getApiErrorData = (error: unknown): unknown => {
  return resolveApiError(error).data
}

const requestCore = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
})

function handleLogout(): void {
  window.location.assign('/login')
}

requestCore.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const tenantId = getCurrentTenantId()
    if (tenantId) {
      config.headers['X-Tenant-ID'] = tenantId
    }
    return config
  },
  (error) => Promise.reject(error),
)

requestCore.interceptors.response.use(
  (response) => {
    const normalizedPayload = normalizeApiResponsePayload(response.data)
    if (normalizedPayload) {
      if (normalizedPayload.code !== 0) {
        const error = new ApiBusinessError(normalizedPayload, response.status)
        return Promise.reject(error)
      }
      return normalizedPayload
    }
    return response.data
  },
  (error: AxiosError<ErrorResponseData>) => {
    const response = error.response
    if (response) {
      const { status, data } = response

      switch (status) {
        case 401: {
          handleLogout()
          break
        }
        case 403:
          notifyRequestError(typeof data?.msg === 'string' ? data.msg : '没有权限访问此资源')
          break
        case 404:
          notifyRequestError(typeof data?.msg === 'string' ? data.msg : '请求的资源不存在')
          break
        case 422:
          if (data?.errors) {
            const errorMessages = Object.values(data.errors).flat()
            notifyRequestError(errorMessages.join('; '))
          } else {
            notifyRequestError(typeof data?.msg === 'string' ? data.msg : '请求参数错误')
          }
          break
        default:
          if (typeof data?.msg === 'string' && data.msg) {
            notifyRequestError(data.msg)
          }
          break
      }
    } else {
      notifyRequestError('网络连接失败，请检查网络设置')
    }

    return Promise.reject(error)
  },
)

const request = requestCore as unknown as UnwrappedHttpClient
export default request
