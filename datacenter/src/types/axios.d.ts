import 'axios'

declare module 'axios' {
  interface AxiosRequestConfig<_D = unknown> {
    /** 普通开发态写入所属工程，用于携带并发代次。 */
    projectId?: string | number
  }
}
