/**
 * 数据层：绑定工厂与辅助（类型与 JSDoc 原 types.js 对齐）
 */

import type {
  DatapointBinding,
  ExprBinding,
  TransformOp,
  VarBinding,
} from "../editor-core/document/types.ts";

export type DatapointStatus = "active" | "invalid" | "unknown";

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
