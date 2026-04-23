/**
 * 统一 API 响应：包络形态为 `{ code, msg, data, reqId }`
 * 其中仅 code===0 视为成功，其他 code 代表业务失败。
 */

export interface ApiResponse<T = unknown> {
  code: number;
  msg: string;
  data?: T;
  reqId?: string;
  errors?: Record<string, unknown>;
}

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

const DIGITS_ONLY_RE = /^\d+$/;

function toNumericCode(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string" && DIGITS_ONLY_RE.test(value.trim())) {
    return Number.parseInt(value.trim(), 10);
  }
  return undefined;
}

/**
 * 从包络或未包络的 payload 中取出业务数据。
 * - `{ code!=0, msg }` → 抛错
 * - `{ code===0, data }` → 返回 data
 * - 其它原样返回
 */
export function unwrapApiData<T = unknown>(payload: unknown): T {
  if (isRecord(payload) && "code" in payload && ("msg" in payload || "data" in payload)) {
    const code = toNumericCode(payload.code);
    if (code !== undefined && code !== 0) {
      const msg = typeof payload.msg === "string" ? payload.msg : "请求失败";
      throw new Error(msg);
    }
    return payload.data as T;
  }
  return payload as T;
}
