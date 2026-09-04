import axios from 'axios'
import { ElMessage } from 'element-plus'
import {
  getCurrentAuthoringEpoch,
  getCurrentProjectId,
  getCurrentTenantId,
  getMicroAppContext,
} from '@/runtime/wujie-context'

/**
 * 统一响应契约：
 * 成功/业务失败均为 HTTP 2xx，响应体为 { code, msg, data, reqId }。
 * 其中仅 code === 0 视为成功，其余 code 统一按业务异常处理。
 */
const DEFAULT_BUSINESS_ERROR_CODE = 30000

// 创建 axios 实例
const request = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
})

const toNumericCode = (value) => {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string' && /^\d+$/.test(value.trim())) {
    return Number.parseInt(value.trim(), 10)
  }
  return undefined
}

const hasOwn = (target, key) => Object.prototype.hasOwnProperty.call(target, key)

/**
 * 业务响应包络识别规则：
 * 1. 必须能解析出 code；
 * 2. 需要具备 msg，或至少具备统一包络中的显式字段（data/reqId/msg 键）。
 *
 * 说明：
 * - 后端可能省略 data，因此不能把 data 作为必须项；
 * - 一旦识别为业务包络，code != 0 必须抛出 ApiBusinessError，避免业务失败被吞掉。
 */
const isApiResponsePayload = (value) => {
  if (!value || typeof value !== 'object') {
    return false
  }

  const payload = value
  const code = toNumericCode(payload.code)
  if (code === undefined) {
    return false
  }

  const hasStringMsg = typeof payload.msg === 'string'
  const hasEnvelopeMarker =
    hasOwn(payload, 'msg') || hasOwn(payload, 'data') || hasOwn(payload, 'reqId')

  return hasStringMsg || hasEnvelopeMarker
}

export class ApiBusinessError extends Error {
  code: number
  reqId?: string
  data: unknown
  status?: number
  isBusinessError: true

  constructor(
    payload: { code: number; msg?: string; reqId?: string; data?: unknown },
    status?: number,
  ) {
    super(payload.msg || '请求失败')
    this.name = 'ApiBusinessError'
    this.code = payload.code
    this.reqId = payload.reqId
    this.data = payload.data
    this.status = status
    this.isBusinessError = true
  }
}

export const isApiBusinessError = (error) => {
  return (
    error instanceof ApiBusinessError ||
    (typeof error === 'object' && error !== null && error.isBusinessError === true)
  )
}

const pickAxiosErrorData = (error) => {
  if (!error || typeof error !== 'object' || !('response' in error)) {
    return undefined
  }
  return error.response?.data
}

const pickAxiosStatus = (error) => {
  if (!error || typeof error !== 'object' || !('response' in error)) {
    return undefined
  }
  const status = error.response?.status
  return typeof status === 'number' ? status : undefined
}

/**
 * 统一提取 API 错误信息：
 * 1. 业务失败（2xx + code!=0）优先读取业务 code/msg/reqId
 * 2. 技术失败（4xx/5xx）读取 HTTP 响应中的 msg 与 status
 * 3. 兜底到 Error.message 或调用方传入的 fallback
 */
export const resolveApiError = (error, fallback = '请求失败') => {
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

export const getApiErrorMessage = (error, fallback = '请求失败') => {
  return resolveApiError(error, fallback).msg
}

export const getApiErrorCode = (error) => {
  return resolveApiError(error).code
}

export const getApiErrorReqId = (error) => {
  return resolveApiError(error).reqId
}

let standaloneRefreshPromise: Promise<boolean> | null = null

const refreshStandaloneSession = (): Promise<boolean> => {
  if (standaloneRefreshPromise) return standaloneRefreshPromise
  standaloneRefreshPromise = fetch('/api/v1/auth/refresh', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: '{}',
  }).then(async (response) => {
    const payload = await response.json().catch(() => null)
    return response.ok && toNumericCode(payload?.code) === 0
  })
  standaloneRefreshPromise = standaloneRefreshPromise
    .catch(() => false)
    .finally(() => {
      standaloneRefreshPromise = null
    })
  return standaloneRefreshPromise
}

const refreshSession = (): Promise<boolean> => {
  const hostRefresh = getMicroAppContext()?.onRefreshAuth
  return hostRefresh ? hostRefresh() : refreshStandaloneSession()
}

const handleAuthExpired = () => {
  const hostHandler = getMicroAppContext()?.onAuthExpired
  if (hostHandler) {
    hostHandler()
    return
  }
  window.location.assign('/login')
}

const isMutation = (method) =>
  ['post', 'put', 'patch', 'delete'].includes(String(method || 'get').toLowerCase())

const resolveProjectId = (config) => {
  const explicit = String(config?.projectId || '').trim()
  if (explicit) return explicit
  return getCurrentProjectId() || ''
}

const staleDetail = (error) => {
  if (error?.response?.status !== 409 || !isMutation(error?.config?.method)) return null
  const data = error.response?.data?.data
  if (!data || typeof data !== 'object' || data.action !== 'reload') return null
  const projectId = String(data.projectId || resolveProjectId(error.config)).trim()
  if (!projectId) return null
  return {
    projectId,
    currentAuthoringEpoch: data.currentAuthoringEpoch
      ? String(data.currentAuthoringEpoch)
      : undefined,
    action: 'reload' as const,
  }
}

// 请求拦截器
request.interceptors.request.use(
  (config) => {
    const tenantId = getCurrentTenantId()
    if (tenantId) {
      config.headers['X-Tenant-ID'] = tenantId
    }
    if (isMutation(config.method)) {
      const projectId = resolveProjectId(config)
      const contextProjectId = getCurrentProjectId()
      const epoch = projectId && projectId === contextProjectId ? getCurrentAuthoringEpoch() : null
      if (epoch) config.headers['X-InduForge-Authoring-Epoch'] = epoch
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
      const normalizedCode = toNumericCode(responseData.code) ?? DEFAULT_BUSINESS_ERROR_CODE
      const normalizedMsg = typeof responseData.msg === 'string' ? responseData.msg : ''
      const normalizedPayload = {
        code: normalizedCode,
        msg: normalizedMsg,
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
  async (error) => {
    const { response } = error
    const config = error.config
    const authoringStale = staleDetail(error)
    if (authoringStale) {
      getMicroAppContext()?.onAuthoringStale?.(authoringStale)
      return Promise.reject(error)
    }

    if (response) {
      const { status, data } = response

      switch (status) {
        case 401: {
          if (
            config &&
            !config._authRetried &&
            !String(config.url || '').includes('/auth/refresh')
          ) {
            const refreshed = await refreshSession()
            if (refreshed) {
              config._authRetried = true
              return request(config)
            }
          }
          handleAuthExpired()
          break
        }
        case 403:
          ElMessage.error('没有权限访问此资源')
          break
        case 404:
          ElMessage.error('请求的资源不存在')
          break
        case 422:
          // 验证错误
          if (data?.errors) {
            const errorMessages = Object.values(data.errors).flat()
            ElMessage.error(errorMessages.join('; '))
          } else {
            ElMessage.error(typeof data?.msg === 'string' ? data.msg : '请求参数错误')
          }
          break
        case 500:
          // 服务器错误，不在拦截器中显示，让业务代码处理
          break
        default:
          // 其他错误，不在拦截器中显示，让业务代码处理
          break
      }
    } else {
      // 网络错误
      ElMessage.error('网络连接失败，请检查网络设置')
    }

    return Promise.reject(error)
  },
)

export default request
