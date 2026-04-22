import 'axios'

declare module 'axios' {
  interface AxiosRequestConfig<D = unknown> {
    /**
     * 403 时是否强制弹出权限提示。
     */
    forcePermissionToast?: boolean
    /**
     * 403 时是否跳过权限提示。
     */
    skipPermissionToast?: boolean
    /**
     * 内部重试标记，仅供 request 刷新 token 场景使用。
     */
    _retry?: boolean
  }
}
