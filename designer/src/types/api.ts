/**
 * 统一 API 响应：包络形态为 `{ success: boolean, data: T, message? }`
 * axios 拦截器已对成功包络解包；此处供仍收到完整包络或需类型断言的调用方使用。
 */

export interface ApiSuccessBody<T = unknown> {
  success: true;
  data: T;
  message?: string;
  code?: number | string;
}

export interface ApiFailureBody {
  success: false;
  message?: string;
  code?: number | string;
  errors?: Record<string, unknown>;
  data?: unknown;
}

export type ApiResponse<T> = ApiSuccessBody<T> | ApiFailureBody;

export interface AssetFolder {
  id: string;
  name: string;
  parentId?: string | null;
}

export interface AssetItem {
  id: string;
  name?: string;
  originalName?: string;
  folderId?: string | null;
  url?: string;
  src?: string;
  path?: string;
  type?: string;
  mimeType?: string;
  size?: number | string | null;
  fileSize?: number | string | null;
  file_size?: number | string | null;
  length?: number | string | null;
  bytes?: number | string | null;
  thumbnailUrl?: string;
  metadata?: {
    size?: number | string | null;
    fileSize?: number | string | null;
    length?: number | string | null;
  };
}

export interface AssetFoldersPayload {
  folders: AssetFolder[];
}

export interface AssetsPayload {
  assets: AssetItem[];
}

export interface PagesListPayload {
  pages: unknown[];
  entryConfig?: Record<string, unknown>;
}

/** GET .../connections 业务体 */
export interface ConnectionsPayload {
  connections: unknown[];
}

/** GET .../datapoints 分页列表业务体 */
export interface DataPointsListPayload {
  datapoints: unknown[];
  pagination?: Record<string, unknown>;
}

function isRecord(v: unknown): v is Record<string, unknown> {
  return v !== null && typeof v === "object";
}

/**
 * 从包络或未包络的 payload 中取出业务数据。
 * - `{ success: false, data?, message? }` → 抛错
 * - `{ success: true, data: T }` 或其它含 `success`+`data` 的成功包络 → `T`
 * - 其它原样返回
 */
export function unwrapApiData<T = unknown>(payload: unknown): T {
  if (isRecord(payload) && "success" in payload && "data" in payload) {
    if (payload["success"] === false) {
      const msg =
        typeof payload["message"] === "string"
          ? payload["message"]
          : "请求失败";
      throw new Error(msg);
    }
    return payload["data"] as T;
  }
  return payload as T;
}
