import axios, {
  type AxiosError,
  type AxiosInstance,
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
} from 'axios'
import { ElMessage } from 'element-plus'
import { Storage } from '@/utils/storage'
import { authAPI, type AuthTokenPayload } from '@/api/auth.api'
import { STORAGE_KEYS } from '@/constants'
import type { ApiResponse } from '@/types/api'

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

// 创建 axios 实例
const request = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
}) as RequestInstance

// 刷新token的状态标志
let isRefreshing = false
type QueueEntry = {
  resolve: (token: string | null) => void
  reject: (error: unknown) => void
}
let failedQueue: QueueEntry[] = []

type ErrorResponseData = {
  code?: number | string
  msg?: string
  reqId?: string
  errors?: Record<string, string[]>
  data?: unknown
  [key: string]: unknown
}

type ExtendedRequestConfig = InternalAxiosRequestConfig & {
  _retry?: boolean
  forcePermissionToast?: boolean
  skipPermissionToast?: boolean
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
  return error instanceof ApiBusinessError || (
    typeof error === 'object' &&
    error !== null &&
    (error as { isBusinessError?: unknown }).isBusinessError === true
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

const clearAuthAndRedirectToLogin = () => {
  Storage.remove(STORAGE_KEYS.TOKEN)
  Storage.remove(STORAGE_KEYS.REFRESH_TOKEN)
  Storage.remove(STORAGE_KEYS.USER_INFO)
  Storage.remove(STORAGE_KEYS.TENANT_ID)
  window.location.href = '/login'
}

// 处理失败的请求队列
const processQueue = (error: unknown, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error)
    } else {
      prom.resolve(token)
    }
  })

  failedQueue = []
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

    // 添加认证 token
    const token = Storage.getToken()
    if (token) {
      ;(config.headers as Record<string, string>).Authorization = `Bearer ${token}`
    }

    // 添加租户 ID
    const tenantId = Storage.getTenantId()
    if (tenantId) {
      ;(config.headers as Record<string, string>)['X-Tenant-ID'] = tenantId
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
        return Promise.reject(new ApiBusinessError(normalizedPayload, response.status))
      }
      return normalizedPayload
    }
    return responseData
  },
  (error: AxiosError<ErrorResponseData>) => {
    const response = error.response
    const config = error.config as ExtendedRequestConfig | undefined
    const requestUrl = config?.url || ''

    if (response) {
      const { status, data } = response

      switch (status) {
        case 401: {
          if (config?._retry) {
            clearAuthAndRedirectToLogin()
            return Promise.reject(error)
          }

          // 检查是否是登录接口的401错误（用户名密码错误）
          if (requestUrl.includes('/auth/login')) {
            // 登录接口的401错误不重定向，直接返回错误让页面处理
            return Promise.reject(error)
          }

          // 检查是否是刷新token接口的401错误
          if (requestUrl.includes('/auth/refresh')) {
            // 刷新token失败，跳转到登录页
            clearAuthAndRedirectToLogin()
            return Promise.reject(error)
          }

          // 尝试刷新token
          const refreshToken = Storage.getRefreshToken()
          if (!refreshToken) {
            // 没有refresh token，直接跳转登录页
            clearAuthAndRedirectToLogin()
            return Promise.reject(error)
          }
          if (!config) {
            // 缺失原始请求配置时无法重放，请求按未登录处理。
            clearAuthAndRedirectToLogin()
            return Promise.reject(error)
          }

          if (!isRefreshing) {
            isRefreshing = true

            return authAPI
              .refreshToken(refreshToken)
              .then(
                (
                  result: ApiResponse<AuthTokenPayload> | { data: ApiResponse<AuthTokenPayload> },
                ) => {
                  const refreshResult = 'code' in result ? result : result.data
                  const accessToken = refreshResult.data?.accessToken || refreshResult.data?.token
                  const newRefreshToken = refreshResult.data?.refreshToken

                  // 更新存储的token
                  if (!accessToken) {
                    throw new Error('刷新令牌响应缺少 accessToken/token 字段')
                  }
                  Storage.setToken(accessToken)
                  if (newRefreshToken) {
                    Storage.setRefreshToken(newRefreshToken)
                  }

                  // 处理队列中的请求
                  processQueue(null, accessToken)

                  // 重新发起原始请求
                  config._retry = true
                  config.headers = config.headers || {}
                  ;(config.headers as Record<string, string>).Authorization =
                    `Bearer ${accessToken}`
                  return request(config)
                },
              )
              .catch((refreshError: unknown) => {
                // 刷新失败，跳转到登录页
                processQueue(refreshError, null)
                clearAuthAndRedirectToLogin()
                return Promise.reject(refreshError)
              })
              .finally(() => {
                isRefreshing = false
              })
          } else {
            // 如果正在刷新，将请求加入队列
            return new Promise((resolve, reject) => {
              failedQueue.push({
                resolve: (token) => {
                  if (!token) {
                    reject(new Error('刷新令牌后未获取到 access token'))
                    return
                  }
                  config.headers = config.headers || {}
                  ;(config.headers as Record<string, string>).Authorization = `Bearer ${token}`
                  resolve(request(config))
                },
                reject,
              })
            })
          }
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
