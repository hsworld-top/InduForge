/**
 * HTTP 请求封装
 */

import axios, {
  type AxiosError,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
} from "axios";
import { ElMessage } from "element-plus";
import { Storage } from "@/utils/storage";
import { STORAGE_KEYS } from "@/constants";

function isRecord(v: unknown): v is Record<string, unknown> {
  return v !== null && typeof v === "object";
}

/** 包络 `{ success, data }`：失败在拦截器层拒绝，成功则只返回 data */
function unwrapEnvelopeOrPass(body: unknown): unknown {
  if (!isRecord(body) || !("success" in body) || !("data" in body)) {
    return body;
  }
  if (body["success"] === false) {
    const msg =
      typeof body["message"] === "string" ? body["message"] : "请求失败";
    throw new Error(msg);
  }
  return body["data"];
}

type QueueProm = {
  resolve: (token: string) => void;
  reject: (err: unknown) => void;
};

const request = axios.create({
  baseURL: "/api/v1",
  timeout: 30000,
  headers: {
    "Content-Type": "application/json",
  },
});

let isRefreshing = false;
let failedQueue: QueueProm[] = [];

const processQueue = (error: unknown, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else if (token) {
      prom.resolve(token);
    }
  });
  failedQueue = [];
};

const refreshAccessToken = (refreshToken: string) => {
  return axios.post<{ data?: { accessToken?: string; refreshToken?: string } }>(
    "/api/v1/auth/refresh",
    { refreshToken },
  );
};

const handleLogout = () => {
  Storage.remove(STORAGE_KEYS.TOKEN);
  Storage.remove(STORAGE_KEYS.REFRESH_TOKEN);
  Storage.remove(STORAGE_KEYS.USER_INFO);
  Storage.remove(STORAGE_KEYS.TENANT_ID);
  Storage.remove(STORAGE_KEYS.PROJECT_ID);

  if (window.parent !== window) {
    window.parent.postMessage({ type: "AUTH_EXPIRED" }, "*");
  }
};

request.interceptors.request.use(
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
  (error: AxiosError) => Promise.reject(error),
);

request.interceptors.response.use(
  (response: AxiosResponse) => {
    try {
      // 约定：对外 Promise 直接 resolve 业务体（与历史行为一致），与 Axios 默认类型不同故断言
      return unwrapEnvelopeOrPass(response.data) as unknown as AxiosResponse;
    } catch (err) {
      const msg = err instanceof Error ? err.message : "请求失败";
      ElMessage.error(msg as never);
      return Promise.reject(err instanceof Error ? err : new Error(msg));
    }
  },
  (error: AxiosError) => {
    const response = error.response;
    const config = error.config;

    if (response && config) {
      const { status, data } = response as {
        status: number;
        data: { errors?: Record<string, unknown>; message?: string };
      };

      switch (status) {
        case 401: {
          if (config.url?.includes("/auth/refresh")) {
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
              .then((result) => {
                const payload =
                  (result?.data as { data?: unknown } | undefined)?.data ??
                  result?.data ??
                  {};
                const typed = payload as {
                  accessToken?: string;
                  refreshToken?: string;
                };
                const { accessToken, refreshToken: newRefreshToken } = typed;
                if (!accessToken) {
                  throw new Error("刷新令牌响应缺少 accessToken");
                }

                Storage.setToken(accessToken);
                if (newRefreshToken) {
                  Storage.setRefreshToken(newRefreshToken);
                }

                processQueue(null, accessToken);

                config.headers.Authorization = `Bearer ${accessToken}`;
                return request(config);
              })
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
              resolve: (token: string) => {
                config.headers.Authorization = `Bearer ${token}`;
                resolve(request(config));
              },
              reject,
            });
          });
        }
        case 403:
          ElMessage.error("没有权限访问此资源" as never);
          break;
        case 404:
          ElMessage.error("请求的资源不存在" as never);
          break;
        case 422:
          if (data.errors) {
            const errorMessages = Object.values(data.errors).flat();
            ElMessage.error(errorMessages.join("; ") as never);
          } else {
            ElMessage.error((data.message || "请求参数错误") as never);
          }
          break;
        case 500:
          break;
        default:
          break;
      }
    } else {
      ElMessage.error("网络连接失败，请检查网络设置" as never);
    }

    return Promise.reject(error);
  },
);

export default request;
