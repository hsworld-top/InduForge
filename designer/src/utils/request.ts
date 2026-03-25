/**
 * HTTP 请求封装
 *
 * 响应拦截器返回 `response.data`，故方法泛型 `T` 表示解包后的业务数据类型。
 */

import type {
  AxiosInstance,
  AxiosRequestConfig,
  AxiosResponse,
  InternalAxiosRequestConfig,
} from "axios";
import axios from "axios";
import { ElMessage } from "element-plus";
import { STORAGE_KEYS } from "@/constants";
import { Storage } from "@/utils/storage";

/** Element Plus 的 ElMessage 选项类型在部分 TS 配置下过窄，此处收窄为运行时实际用法 */
function notifyRequestError(message: string): void {
  (ElMessage as unknown as (opts: { type: "error"; message: string }) => void)({
    type: "error",
    message,
  });
}

export interface UnwrappedHttpClient {
  get: <T = unknown>(url: string, config?: AxiosRequestConfig) => Promise<T>;
  post: <T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig) => Promise<T>;
  put: <T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig) => Promise<T>;
  patch: <T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig) => Promise<T>;
  delete: <T = unknown>(url: string, config?: AxiosRequestConfig) => Promise<T>;
  interceptors: AxiosInstance["interceptors"];
  defaults: AxiosInstance["defaults"];
}

interface QueueItem {
  resolve: (token: string | null) => void;
  reject: (err: unknown) => void;
}

const requestCore = axios.create({
  baseURL: "/api/v1",
  timeout: 30000,
  headers: {
    "Content-Type": "application/json",
  },
});

let isRefreshing = false;
let failedQueue: QueueItem[] = [];

function processQueue(error: unknown, token: string | null = null): void {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(token);
    }
  });
  failedQueue = [];
}

function refreshAccessToken(refreshToken: string) {
  return axios.post("/api/v1/auth/refresh", { refreshToken });
}

function handleLogout(): void {
  Storage.remove(STORAGE_KEYS.TOKEN);
  Storage.remove(STORAGE_KEYS.REFRESH_TOKEN);
  Storage.remove(STORAGE_KEYS.USER_INFO);
  Storage.remove(STORAGE_KEYS.TENANT_ID);
  Storage.remove(STORAGE_KEYS.PROJECT_ID);

  if (window.parent !== window) {
    window.parent.postMessage({ type: "AUTH_EXPIRED" }, "*");
  }
}

requestCore.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = Storage.getToken();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    const tenantId = Storage.getTenantId();
    if (tenantId) {
      config.headers["X-Tenant-ID"] = tenantId;
    }
    return config;
  },
  (error) => Promise.reject(error),
);

requestCore.interceptors.response.use(
  (response) => response.data,
  (error) => {
    const { response, config } = error;

    if (response) {
      const { status, data } = response;

      switch (status) {
        case 401: {
          if (config?.url?.includes("/auth/refresh")) {
            handleLogout();
            return Promise.reject(error);
          }

          const refreshToken = Storage.getRefreshToken();
          if (!refreshToken) {
            handleLogout();
            return Promise.reject(error);
          }

          if (!isRefreshing) {
            isRefreshing = true;

            return refreshAccessToken(refreshToken)
              .then(
                (
                  result: AxiosResponse<{ data?: Record<string, string> } & Record<string, string>>,
                ) => {
                  const body = result?.data as { data?: Record<string, string> } & Record<
                    string,
                    string
                  >;
                  const payload = body?.data || body || {};
                  const accessToken = payload.accessToken as string | undefined;
                  const newRefreshToken = payload.refreshToken as string | undefined;

                  if (accessToken) Storage.setToken(accessToken);
                  if (newRefreshToken) Storage.setRefreshToken(newRefreshToken);

                  processQueue(null, accessToken ?? null);

                  if (config?.headers && accessToken) {
                    config.headers.Authorization = `Bearer ${accessToken}`;
                  }
                  return requestCore(config!);
                },
              )
              .catch((refreshError: unknown) => {
                processQueue(refreshError, null);
                handleLogout();
                return Promise.reject(refreshError);
              })
              .finally(() => {
                isRefreshing = false;
              });
          }

          return new Promise((resolve, reject) => {
            failedQueue.push({
              resolve: (token) => {
                if (config?.headers && token) {
                  config.headers.Authorization = `Bearer ${token}`;
                }
                resolve(requestCore(config!));
              },
              reject,
            });
          });
        }
        case 403:
          notifyRequestError("没有权限访问此资源");
          break;
        case 404:
          notifyRequestError("请求的资源不存在");
          break;
        case 422: {
          const body = data as { errors?: Record<string, string[]>; message?: string };
          if (body?.errors) {
            const errorMessages = Object.values(body.errors).flat();
            notifyRequestError(errorMessages.join("; "));
          } else {
            notifyRequestError(body?.message || "请求参数错误");
          }
          break;
        }
        default:
          break;
      }
    } else {
      notifyRequestError("网络连接失败，请检查网络设置");
    }

    return Promise.reject(error);
  },
);

const request = requestCore as unknown as UnwrappedHttpClient;
export default request;
