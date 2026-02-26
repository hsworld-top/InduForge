import axios from 'axios'
import { ElMessage } from 'element-plus'
import { Storage } from '@/utils/storage'
import { authAPI } from '@/api/auth.api'
import { STORAGE_KEYS } from '@/constants'

// 创建 axios 实例
const request = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 刷新token的状态标志
let isRefreshing = false
let failedQueue = []

const clearAuthAndRedirectToLogin = () => {
  Storage.remove(STORAGE_KEYS.TOKEN)
  Storage.remove(STORAGE_KEYS.REFRESH_TOKEN)
  Storage.remove(STORAGE_KEYS.USER_INFO)
  Storage.remove(STORAGE_KEYS.TENANT_ID)
  window.location.href = '/login'
}

// 处理失败的请求队列
const processQueue = (error, token = null) => {
  failedQueue.forEach(prom => {
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
    // 添加认证 token
    const token = Storage.getToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }

    // 添加租户 ID
    const tenantId = Storage.getTenantId()
    if (tenantId) {
      config.headers['X-Tenant-ID'] = tenantId
    }

    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
request.interceptors.response.use(
  (response) => {
    return response.data
  },
  (error) => {
    const { response, config } = error
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

          if (!isRefreshing) {
            isRefreshing = true

            return authAPI.refreshToken(refreshToken)
              .then(result => {
                const accessToken = result.data?.accessToken || result.data?.token
                const newRefreshToken = result.data?.refreshToken

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
                config.headers.Authorization = `Bearer ${accessToken}`
                return request(config)
              })
              .catch(refreshError => {
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
                  config.headers.Authorization = `Bearer ${token}`
                  resolve(request(config))
                },
                reject
              })
            })
          }
        }
        case 403:
          ElMessage.error('没有权限访问此资源')
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
            ElMessage.error(data.message || '请求参数错误')
          }
          break
        case 500:
          ElMessage.error('服务器内部错误')
          break
        default:
          ElMessage.error(data.message || '请求失败')
      }
    } else {
      // 网络错误
      ElMessage.error('网络连接失败，请检查网络设置')
    }

    return Promise.reject(error)
  }
)

export default request
