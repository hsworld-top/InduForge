// @ts-nocheck
import axios from "axios";
import { ElMessage } from "element-plus";
import { Storage } from "@/utils/storage";
import { STORAGE_KEYS } from "@/constants";
import {
  buildIdeLoginUrl,
  postMessageToHost,
  resolveIdeOriginFromRuntime,
} from "../runtime/host-bootstrap";

/**
 * 统一响应契约：
 * 成功/业务失败均为 HTTP 2xx，响应体为 { code, msg, data, reqId }。
 * 其中仅 code === 0 视为成功，其余 code 统一按业务异常处理。
 */
const DEFAULT_BUSINESS_ERROR_CODE = 30000;

// 创建 axios 实例
const request = axios.create({
  baseURL: "/api/v1",
  timeout: 10000,
  headers: {
    "Content-Type": "application/json",
  },
});

// 刷新token的状态标志
let isRefreshing = false;
let failedQueue = [];

const toNumericCode = (value) => {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string" && /^\d+$/.test(value.trim())) {
    return Number.parseInt(value.trim(), 10);
  }
  return undefined;
};

const hasOwn = (target, key) =>
  Object.prototype.hasOwnProperty.call(target, key);

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
  if (!value || typeof value !== "object") {
    return false;
  }

  const payload = value;
  const code = toNumericCode(payload.code);
  if (code === undefined) {
    return false;
  }

  const hasStringMsg = typeof payload.msg === "string";
  const hasEnvelopeMarker =
    hasOwn(payload, "msg") || hasOwn(payload, "data") || hasOwn(payload, "reqId");

  return hasStringMsg || hasEnvelopeMarker;
};

export class ApiBusinessError extends Error {
  constructor(payload, status) {
    super(payload.msg || "请求失败");
    this.name = "ApiBusinessError";
    this.code = payload.code;
    this.reqId = payload.reqId;
    this.data = payload.data;
    this.status = status;
    this.isBusinessError = true;
  }
}

export const isApiBusinessError = (error) => {
  return (
    error instanceof ApiBusinessError ||
    (typeof error === "object" &&
      error !== null &&
      error.isBusinessError === true)
  );
};

const pickAxiosErrorData = (error) => {
  if (!error || typeof error !== "object" || !("response" in error)) {
    return undefined;
  }
  return error.response?.data;
};

const pickAxiosStatus = (error) => {
  if (!error || typeof error !== "object" || !("response" in error)) {
    return undefined;
  }
  const status = error.response?.status;
  return typeof status === "number" ? status : undefined;
};

/**
 * 统一提取 API 错误信息：
 * 1. 业务失败（2xx + code!=0）优先读取业务 code/msg/reqId
 * 2. 技术失败（4xx/5xx）读取 HTTP 响应中的 msg 与 status
 * 3. 兜底到 Error.message 或调用方传入的 fallback
 */
export const resolveApiError = (error, fallback = "请求失败") => {
  if (isApiBusinessError(error)) {
    return {
      code: error.code,
      msg: error.message || fallback,
      reqId: error.reqId,
      status: error.status,
    };
  }

  const data = pickAxiosErrorData(error);
  const status = pickAxiosStatus(error);
  const responseCode = toNumericCode(data?.code);
  const responseMsg = typeof data?.msg === "string" ? data.msg : "";
  const responseReqId = typeof data?.reqId === "string" ? data.reqId : undefined;
  const nativeMessage = error instanceof Error ? error.message : "";

  return {
    code: responseCode,
    msg: responseMsg || nativeMessage || fallback,
    reqId: responseReqId,
    status,
  };
};

export const getApiErrorMessage = (error, fallback = "请求失败") => {
  return resolveApiError(error, fallback).msg;
};

export const getApiErrorCode = (error) => {
  return resolveApiError(error).code;
};

export const getApiErrorReqId = (error) => {
  return resolveApiError(error).reqId;
};

/**
 * 处理失败的请求队列。
 * @param {Error|null} error - 错误
 * @param {string|null} token - 新令牌
 */
const processQueue = (error, token = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(token);
    }
  });

  failedQueue = [];
};

/**
 * 刷新访问令牌。
 * @param {string} refreshToken - 刷新令牌
 * @returns {Promise} 刷新结果
 */
const refreshAccessToken = (refreshToken) => {
  return axios.post("/api/v1/auth/refresh", { refreshToken });
};

/**
 * 将认证失效显式回传宿主。
 * 宿主不在场时再回落到 IDE 登录页，避免 iframe 内部误跳成本地 `/login`。
 */
const handleLogout = () => {
  Storage.remove(STORAGE_KEYS.TOKEN);
  Storage.remove(STORAGE_KEYS.REFRESH_TOKEN);
  Storage.remove(STORAGE_KEYS.USER_INFO);
  Storage.remove(STORAGE_KEYS.TENANT_ID);
  Storage.removeProjectId();

  const posted = postMessageToHost({
    type: "AUTH_EXPIRED",
    app: "datacenter",
  });

  if (window.parent === window || !posted) {
    const ideOrigin = resolveIdeOriginFromRuntime({
      currentUrl: window.location.href,
      referrer: document.referrer,
    });
    window.location.href = buildIdeLoginUrl(window.location.href, ideOrigin);
  }
};

// 请求拦截器
request.interceptors.request.use(
  (config) => {
    // 添加认证 token
    const token = Storage.getToken();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }

    // 添加租户 ID
    const tenantId = Storage.getTenantId();
    if (tenantId) {
      config.headers["X-Tenant-ID"] = tenantId;
    }

    return config;
  },
  (error) => {
    return Promise.reject(error);
  },
);

// 响应拦截器
request.interceptors.response.use(
  (response) => {
    const responseData = response.data;
    if (isApiResponsePayload(responseData)) {
      const normalizedCode =
        toNumericCode(responseData.code) ?? DEFAULT_BUSINESS_ERROR_CODE;
      const normalizedMsg =
        typeof responseData.msg === "string" ? responseData.msg : "";
      const normalizedPayload = {
        code: normalizedCode,
        msg: normalizedMsg,
        data: responseData.data,
        reqId: responseData.reqId,
      };

      if (normalizedCode !== 0) {
        return Promise.reject(
          new ApiBusinessError(normalizedPayload, response.status),
        );
      }
      return normalizedPayload;
    }

    return responseData;
  },
  (error) => {
    const { response, config } = error;
    const requestUrl = config?.url || "";

    if (response) {
      const { status, data } = response;

      switch (status) {
        case 401: {
          // 刷新 token 失败时直接跳转登录页
          if (requestUrl.includes("/auth/refresh")) {
            handleLogout();
            return Promise.reject(error);
          }

          const refreshToken = Storage.getRefreshToken();
          if (!refreshToken) {
            handleLogout();
            return Promise.reject(error);
          }
          if (!config) {
            handleLogout();
            return Promise.reject(error);
          }

          if (!isRefreshing) {
            isRefreshing = true;

            return refreshAccessToken(refreshToken)
              .then((result) => {
                const payload = result?.data?.data || result?.data || {};
                const { accessToken, refreshToken: newRefreshToken } = payload;

                Storage.setToken(accessToken);
                if (newRefreshToken) {
                  Storage.setRefreshToken(newRefreshToken);
                }

                postMessageToHost({
                  type: "AUTH_REFRESHED",
                  app: "datacenter",
                  token: accessToken,
                  refreshToken: newRefreshToken || refreshToken,
                });

                processQueue(null, accessToken);

                config.headers = config.headers || {};
                config.headers.Authorization = `Bearer ${accessToken}`;
                return request(config);
              })
              .catch((refreshError) => {
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
                if (!token) {
                  reject(new Error("刷新令牌后未获取到 access token"));
                  return;
                }
                config.headers = config.headers || {};
                config.headers.Authorization = `Bearer ${token}`;
                resolve(request(config));
              },
              reject,
            });
          });
        }
        case 403:
          ElMessage.error("没有权限访问此资源");
          break;
        case 404:
          ElMessage.error("请求的资源不存在");
          break;
        case 422:
          // 验证错误
          if (data?.errors) {
            const errorMessages = Object.values(data.errors).flat();
            ElMessage.error(errorMessages.join("; "));
          } else {
            ElMessage.error(
              typeof data?.msg === "string" ? data.msg : "请求参数错误",
            );
          }
          break;
        case 500:
          // 服务器错误，不在拦截器中显示，让业务代码处理
          break;
        default:
          // 其他错误，不在拦截器中显示，让业务代码处理
          break;
      }
    } else {
      // 网络错误
      ElMessage.error("网络连接失败，请检查网络设置");
    }

    return Promise.reject(error);
  },
);

export default request;
