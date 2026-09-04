import axios, {
  type AxiosError,
  type AxiosInstance,
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
} from 'axios'
import { ElMessage } from 'element-plus'
import { Storage } from '@/utils/storage'
import { STORAGE_KEYS } from '@/constants'
import type { ApiResponse } from '@/types/api'
import { getCachedAuthoringContext, notifyAuthoringStale } from './authoring-context'

type RequestInstance = AxiosInstance & {
  <T = unknown, D = unknown>(config: AxiosRequestConfig<D>): Promise<T>
  <T = unknown, D = unknown>(url: string, config?: AxiosRequestConfig<D>): Promise<T>
  request<T = unknown, D = unknown>(config: AxiosRequestConfig<D>): Promise<T>
  get<T = unknown, D = unknown>(url: string, config?: AxiosRequestConfig<D>): Promise<T>
  delete<T = unknown, D = unknown>(url: string, config?: AxiosRequestConfig<D>): Promise<T>
  head<T = unknown, D = unknown>(url: string, config?: AxiosRequestConfig<D>): Promise<T>
  options<T = unknown, D = unknown>(url: string, config?: AxiosRequestConfig<D>): Promise<T>
  post<T = unknown, D = unknown>(url: string, data?: D, config?: AxiosRequestConfig<D>): Promise<T>
  put<T = unknown, D = unknown>(url: string, data?: D, config?: AxiosRequestConfig<D>): Promise<T>
  patch<T = unknown, D = unknown>(url: string, data?: D, config?: AxiosRequestConfig<D>): Promise<T>
}

/** 调用方可选择自行呈现错误，认证刷新和登录跳转仍由统一层处理。 */
export type RequestConfig = AxiosRequestConfig & {
  skipErrorToast?: boolean
  /** 普通开发态写入所属工程，用于携带并发代次，优先级高于 URL 兼容解析。 */
  projectId?: string | number
}

/**
 * 需要读取下载响应头的请求配置。仍使用统一 request 实例，只跳过包络解包。
 */
export type RawResponseConfig = RequestConfig & {
  returnRawResponse?: boolean
}

// 创建 axios 实例
const request = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
}) as RequestInstance

type ErrorResponseData = {
  code?: number | string
  msg?: string
  reqId?: string
  errors?: Record<string, string[]>
  data?: unknown
  [key: string]: unknown
}

type ExtendedRequestConfig = InternalAxiosRequestConfig & {
  forcePermissionToast?: boolean
  skipPermissionToast?: boolean
  skipAuthRedirect?: boolean
  skipAuthRefresh?: boolean
  /** 业务页面会统一处理错误时，避免与全局拦截器重复提示。 */
  skipErrorToast?: boolean
  _authRetried?: boolean
  /** 下载等场景需要读取响应头时，保留 Axios 原始响应。 */
  returnRawResponse?: boolean
  projectId?: string | number
}

type ApiErrorMeta = {
  code?: number
  msg: string
  reqId?: string
  status?: number
}

const toNumericCode = (value: unknown): number | undefined => {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string' && /^\d+$/.test(value.trim())) {
    return Number.parseInt(value.trim(), 10)
  }
  return undefined
}

const isApiResponsePayload = (value: unknown): value is ApiResponse<unknown> => {
  if (!value || typeof value !== 'object') {
    return false
  }

  const payload = value as Record<string, unknown>
  const code = toNumericCode(payload.code)
  return code !== undefined && typeof payload.msg === 'string' && 'data' in payload
}

export class ApiBusinessError extends Error {
  code: number
  reqId?: string
  data: unknown
  status?: number
  isBusinessError = true

  constructor(payload: ApiResponse<unknown>, status?: number) {
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

export const resolveApiError = (error: unknown, fallback = '请求失败'): ApiErrorMeta => {
  if (isApiBusinessError(error)) {
    return {
      code: error.code,
      msg: error.message || fallback,
      reqId: error.reqId,
      status: error.status,
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

export const clearAuthAndRedirectToLogin = () => {
  Storage.remove(STORAGE_KEYS.USER_INFO)
  Storage.remove(STORAGE_KEYS.TENANT_ID)
  window.location.href = '/login'
}

let refreshPromise: Promise<boolean> | null = null

/**
 * 刷新 HttpOnly Cookie 会话。所有宿主请求和 Wujie 子应用共用这一 Promise，
 * 避免多个 401 同时轮换同一个 Refresh Token。
 */
export const refreshSession = (): Promise<boolean> => {
  if (refreshPromise) return refreshPromise

  refreshPromise = fetch('/api/v1/auth/refresh', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: '{}',
  }).then(async (response) => {
    const payload = (await response.json().catch(() => null)) as Record<string, unknown> | null
    return response.ok && toNumericCode(payload?.code) === 0
  })
  refreshPromise = refreshPromise
    .catch(() => false)
    .finally(() => {
      refreshPromise = null
    })

  return refreshPromise
}

const isRefreshRequest = (config?: ExtendedRequestConfig): boolean =>
  String(config?.url || '').includes('/auth/refresh')

const isMutation = (method?: string): boolean =>
  ['post', 'put', 'patch', 'delete'].includes(String(method || 'get').toLowerCase())

const resolveProjectId = (config?: ExtendedRequestConfig): string => {
  const explicit = String(config?.projectId ?? '').trim()
  if (explicit) return explicit
  const match = String(config?.url || '').match(/\/projects\/([^/?#]+)/)
  return match?.[1] ? decodeURIComponent(match[1]) : ''
}

const resolveStaleDetail = (
  config: ExtendedRequestConfig | undefined,
  status: number | undefined,
  payload: ErrorResponseData | undefined,
) => {
  if (status !== 409 || !isMutation(config?.method)) return null
  const data = payload?.data
  if (!data || typeof data !== 'object' || (data as Record<string, unknown>).action !== 'reload') {
    return null
  }
  const record = data as Record<string, unknown>
  const projectId = String(record.projectId || resolveProjectId(config)).trim()
  if (!projectId) return null
  const current = String(record.currentAuthoringEpoch || '').trim()
  return {
    projectId,
    currentAuthoringEpoch: current || undefined,
    action: 'reload' as const,
  }
}

// 请求拦截器
request.interceptors.request.use(
  (config) => {
    // FormData 请求必须移除默认 JSON Content-Type，让浏览器自动带 boundary
    if (config.data instanceof window.FormData) {
      if (typeof config.headers?.delete === 'function') {
        config.headers.delete('Content-Type')
      } else if (config.headers) {
        delete config.headers['Content-Type']
      }
    }

    // 添加租户 ID
    const tenantId = Storage.getTenantId()
    if (tenantId) {
      ;(config.headers as Record<string, string>)['X-Tenant-ID'] = tenantId
    }

    if (isMutation(config.method)) {
      const projectId = resolveProjectId(config as ExtendedRequestConfig)
      const context = projectId ? getCachedAuthoringContext(projectId) : null
      if (context) {
        ;(config.headers as Record<string, string>)['X-InduForge-Authoring-Epoch'] =
          context.authoringEpoch
      }
    }

    return config
  },
  (error) => {
    return Promise.reject(error)
  },
)

// 响应拦截器
request.interceptors.response.use(
  (response) => {
    if ((response.config as ExtendedRequestConfig | undefined)?.returnRawResponse) {
      return response
    }
    const responseData = response.data
    if (isApiResponsePayload(responseData)) {
      const normalizedCode = toNumericCode(responseData.code) ?? 30000
      const normalizedPayload: ApiResponse<unknown> = {
        code: normalizedCode,
        msg: responseData.msg,
        data: responseData.data,
        reqId: responseData.reqId,
      }
      if (normalizedCode !== 0) {
        const error = new ApiBusinessError(normalizedPayload, response.status)
        return Promise.reject(error)
      }
      return normalizedPayload
    }
    return responseData
  },
  async (error: AxiosError<ErrorResponseData>) => {
    const response = error.response
    const config = error.config as ExtendedRequestConfig | undefined

    const staleDetail = resolveStaleDetail(config, response?.status, response?.data)
    if (staleDetail) {
      notifyAuthoringStale(staleDetail)
      return Promise.reject(error)
    }

    // 登录会话仍需由统一层续租或跳转；其余运维请求交给业务层展示一次明确错误。
    if (config?.skipErrorToast && (!response || response.status !== 401)) {
      return Promise.reject(error)
    }

    if (response) {
      const { status, data } = response

      switch (status) {
        case 401: {
          if (
            config &&
            !config.skipAuthRefresh &&
            !config._authRetried &&
            !isRefreshRequest(config)
          ) {
            const refreshed = await refreshSession()
            if (refreshed) {
              config._authRetried = true
              return request.request(config)
            }
          }
          if (!config?.skipAuthRedirect) {
            clearAuthAndRedirectToLogin()
          }
          break
        }
        case 403:
          // 默认不弹全局 403 提示，避免与业务层 catch 中的错误提示重复。
          // 如需全局提示，可在请求配置中显式传 forcePermissionToast: true。
          if (config?.forcePermissionToast && !config?.skipPermissionToast) {
            ElMessage.error(typeof data?.msg === 'string' ? data.msg : '没有权限访问此资源')
          }
          break
        case 404:
          ElMessage.error('请求的资源不存在')
          break
        case 422:
          // 验证错误
          if (data.errors) {
            const errorMessages = Object.values(data.errors).flat()
            ElMessage.error(errorMessages.join('; '))
          } else {
            ElMessage.error(typeof data?.msg === 'string' ? data.msg : '请求参数错误')
          }
          break
        case 500:
          ElMessage.error('服务器内部错误')
          break
        default:
          ElMessage.error(typeof data?.msg === 'string' ? data.msg : '请求失败')
      }
    } else {
      // 网络错误
      ElMessage.error('网络连接失败，请检查网络设置')
    }

    return Promise.reject(error)
  },
)

export default request
