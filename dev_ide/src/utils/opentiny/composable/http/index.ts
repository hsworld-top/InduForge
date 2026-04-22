import { HttpService } from '@opentiny/tiny-engine'
import { useBroadcastChannel } from '@vueuse/core'
import { constants } from '@opentiny/tiny-engine-utils'

const LOGIN_EXPIRED_CODE = 401
const { BROADCAST_CHANNEL } = constants

const { post: globalNotify } = useBroadcastChannel({ name: BROADCAST_CHANNEL.Notify })

const getVsCodeBridge = (): unknown => (window as Window & { vscodeBridge?: unknown }).vscodeBridge

const showError = (url: string | undefined, message: string | undefined) => {
  globalNotify({
    type: 'error',
    title: '接口报错',
    message: `报错接口: ${url} \n报错信息: ${message ?? ''}`,
  })
}

const preRequest = (config: Record<string, any>) => {
  const isDevelopEnv = import.meta.env.MODE?.includes('dev')

  if (isDevelopEnv && typeof config.url === 'string' && config.url.match(/\/generate\//)) {
    config.baseURL = ''
  }

  if (getVsCodeBridge()) {
    config.baseURL = ''
  }

  return config
}

const preResponse = (res: Record<string, any>) => {
  if (res.data?.error) {
    showError(res.config?.url, res?.data?.error?.message)

    return Promise.reject(res.data.error)
  }

  return res.data?.data || res.data
}

const errorResponse = (error: Record<string, any>) => {
  // 用户信息失效时，弹窗提示登录
  const { response } = error

  if (response?.status === LOGIN_EXPIRED_CODE) {
    // vscode 插件环境弹出输入框提示登录
    if (getVsCodeBridge()) {
      return Promise.resolve(true)
    }
  }

  showError(error.config?.url, error?.message)

  return response?.data.error ? Promise.reject(response.data.error) : Promise.reject(error.message)
}

const getConfig = (env: ImportMetaEnv = import.meta.env) => {
  const baseURL = env.VITE_ORIGIN
  // 仅在本地开发时，启用 withCredentials
  const dev = env.MODE?.includes('dev')
  // 获取租户 id
  const getTenant = () => new URLSearchParams(location.search).get('tenant')

  return {
    baseURL,
    withCredentials: dev,
    headers: {
      ...(dev && { 'x-lowcode-mode': 'develop' }),
      'x-lowcode-org': getTenant(),
    },
  }
}

const customizeHttpService = () => {
  const options = {
    axiosConfig: getConfig(),
    interceptors: {
      request: [preRequest],
      response: [[preResponse, errorResponse]],
    },
  }

  HttpService.apis.setOptions(options)

  return HttpService
}

export default customizeHttpService()