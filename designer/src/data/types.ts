/**
 * 数据层：绑定工厂与辅助（类型与 JSDoc 原 types.js 对齐）
 */

import type {
  Binding,
  DatapointBinding,
  ExprBinding,
  PageNode,
  TransformOp,
  VarBinding,
} from "../editor-core/document/types.ts";

/**
 * 统一 API 包络（HTTP 2xx 场景）：只有 code===0 表示成功。
 * data 在部分接口可省略（例如纯写操作），因此保持可选。
 */
export interface ApiEnvelope<T = unknown> {
  code: number;
  msg: string;
  data?: T;
  reqId?: string;
}

export interface ApiErrorMeta {
  code?: number | undefined;
  msg: string;
  reqId?: string | undefined;
  status?: number | undefined;
  data?: unknown;
  isBusinessError?: boolean;
}

export const DEFAULT_API_ERROR_CODE = 30000;

const DIGITS_ONLY_RE = /^\d+$/;

function hasOwn(target: Record<string, unknown>, key: string): boolean {
  return Object.prototype.hasOwnProperty.call(target, key);
}

export function toApiNumericCode(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string" && DIGITS_ONLY_RE.test(value.trim())) {
    return Number.parseInt(value.trim(), 10);
  }
  return undefined;
}

/**
 * 包络识别策略：
 * 1. 需可解析 code；
 * 2. 需包含 msg 或包络标识字段（data/reqId/msg 任一显式存在）。
 */
export function isApiEnvelopePayload(
  value: unknown,
): value is Partial<ApiEnvelope<unknown>> & Record<string, unknown> {
  if (!value || typeof value !== "object") {
    return false;
  }

  const payload = value as Record<string, unknown>;
  const code = toApiNumericCode(payload.code);
  if (code === undefined) {
    return false;
  }

  const hasStringMsg = typeof payload.msg === "string";
  const hasEnvelopeMarker =
    hasOwn(payload, "msg") || hasOwn(payload, "data") || hasOwn(payload, "reqId");

  return hasStringMsg || hasEnvelopeMarker;
}

/**
 * fetch/业务层统一错误模型，兼容：
 * - 业务失败（2xx + code!=0）
 * - 技术失败（4xx/5xx 或网络异常）
 */
export class ApiRequestError extends Error {
  code: number | undefined;
  reqId: string | undefined;
  status: number | undefined;
  data: unknown;
  isBusinessError: boolean;

  constructor(meta: ApiErrorMeta) {
    super(meta.msg || "请求失败");
    this.name = "ApiRequestError";
    this.code = meta.code;
    this.reqId = meta.reqId;
    this.status = meta.status;
    this.data = meta.data;
    this.isBusinessError = meta.isBusinessError === true;
  }
}

export function normalizeApiEnvelope<T = unknown>(payload: unknown): ApiEnvelope<T> | null {
  if (!isApiEnvelopePayload(payload)) {
    return null;
  }

  const normalizedCode = toApiNumericCode(payload.code) ?? DEFAULT_API_ERROR_CODE;
  const normalizedMsg = typeof payload.msg === "string" ? payload.msg : "";
  const normalizedReqId = typeof payload.reqId === "string" ? payload.reqId : undefined;

  return {
    code: normalizedCode,
    msg: normalizedMsg,
    data: payload.data as T,
    ...(normalizedReqId ? { reqId: normalizedReqId } : {}),
  };
}

export type DatapointStatus = "active" | "invalid" | "unknown";

export interface DatapointStatusInfo {
  path: string;
  status: DatapointStatus;
  dataType: string;
  statusReason?: string;
  [key: string]: unknown;
}

export interface DiagnosticInfo {
  path: string;
  status: DatapointStatus;
  statusReason?: string;
  dataType?: string;
  nodeId?: string;
  bindingCount?: number;
  [key: string]: unknown;
}

export interface DiagnosticsSummary {
  total: number;
  active: number;
  invalid: number;
  unknown: number;
  [key: string]: unknown;
}

export type VarType = "string" | "number" | "boolean" | "array" | "object";

export interface VarDefinition {
  type: VarType;
  default?: unknown;
  label?: string;
  description?: string;
  enum?: unknown[];
  min?: number;
  max?: number;
  minLength?: number;
  maxLength?: number;
  items?: { type: string };
  persistent?: boolean;
  storageKey?: string;
  [key: string]: unknown;
}

export interface VarsDefinitions {
  global: Record<string, VarDefinition>;
  pages: Record<string, Record<string, VarDefinition>>;
}

export interface VarsContext {
  scope: "global" | "page";
  name: string;
  pageId?: string | null;
}

export type DataMode = "edit" | "preview" | "runtime";

export interface ExpressionContext {
  $dp?: Record<string, unknown>;
  $vars?: Record<string, unknown>;
  $global?: Record<string, unknown>;
  $page?: PageNode | null;
  [key: string]: unknown;
}

export interface ExpressionResult {
  success: boolean;
  value?: unknown;
  error?: string;
}

export interface ResolvedBinding {
  value: unknown;
  isLoading: boolean;
  status?: DatapointStatus;
  error?: string;
}

export type BindingValue = Binding;

export function createVarDefinition(
  type: VarType,
  options: Partial<VarDefinition> = {},
): VarDefinition {
  const defaults: Record<VarType, unknown> = {
    string: "",
    number: 0,
    boolean: false,
    array: [],
    object: {},
  };

  return {
    type,
    default: options.default ?? defaults[type],
    ...options,
  } as VarDefinition;
}

export interface DatapointBindingOptions {
  provider?: string;
  datapointId?: string;
  transform?: TransformOp[];
  fallback?: unknown;
  designMock?: unknown;
}

export function createDatapointBinding(
  path: string,
  options: DatapointBindingOptions = {},
): DatapointBinding {
  return {
    kind: "datapoint",
    provider: options.provider || "dc_main",
    datapointId: options.datapointId || "",
    path,
    transform: options.transform || [],
    fallback: options.fallback,
    designMock: options.designMock,
  };
}

export interface VarBindingOptions {
  transform?: TransformOp[];
  fallback?: unknown;
}

export function createVarBinding(
  scope: "page" | "global",
  name: string,
  options: VarBindingOptions = {},
): VarBinding {
  return {
    kind: "var",
    scope,
    name,
    transform: options.transform || [],
    fallback: options.fallback,
  };
}

export interface ExprBindingOptions {
  fallback?: unknown;
}

export function createExprBinding(expr: string, options: ExprBindingOptions = {}): ExprBinding {
  return {
    kind: "expr",
    expr,
    fallback: options.fallback,
  };
}

export function getBindingKind(binding: unknown): string | null {
  if (!binding || typeof binding !== "object") return null;
  const k = (binding as { kind?: string }).kind;
  return k ?? null;
}

export function isValidValue(value: unknown): boolean {
  return value !== undefined && value !== null && !Number.isNaN(value);
}

export default {
  createVarDefinition,
  createDatapointBinding,
  createVarBinding,
  createExprBinding,
  getBindingKind,
  isValidValue,
};
