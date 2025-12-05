import axios from 'axios'
import { ElMessage } from 'element-plus'
import { Storage } from '@/utils/storage'
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

    if (response) {
      const { status, data } = response

      switch (status) {
        case 401:
          // 401 错误，清除本地存储并跳转到登录页
          Storage.remove(STORAGE_KEYS.TOKEN)
          Storage.remove(STORAGE_KEYS.REFRESH_TOKEN)
          Storage.remove(STORAGE_KEYS.USER_INFO)
          Storage.remove(STORAGE_KEYS.TENANT_ID)
          // 跳转到主应用的登录页（不是 /datacenter/login）
          window.location.href = '/login'
          return Promise.reject(error)
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

