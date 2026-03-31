/**
 * 页面属性面板配置构建工具
 */
import type { BackgroundConfig, PageNode } from "@/editor-core/document/types";

export type WindowStyle = "popup" | "cover" | "replace";
export type BackgroundKind = BackgroundConfig["kind"];

export interface PageInspectorConfigInput {
  description: string;
  width: number;
  height: number;
  autoFit: boolean;
  lockAspectRatio: boolean;
  enableMinSize: boolean;
  windowStyle: string;
  permissionDesc: string;
  backgroundKind: BackgroundKind;
  backgroundValue: string;
}

export interface PageInspectorConfigPatch extends Partial<PageNode["config"]> {
  description?: string;
  lockAspectRatio?: boolean;
  enableMinSize?: boolean;
  windowStyle?: WindowStyle;
  permissionDesc?: string;
}

const DEFAULT_WIDTH = 1920;
const DEFAULT_HEIGHT = 1080;
const DEFAULT_BACKGROUND_COLOR = "#ffffff";
const DEFAULT_PERMISSION_DESC = "0item";

/**
 * 规范化窗口类型（兼容旧值 normal）
 * @param {unknown} value - 原始窗口类型
 * @returns {WindowStyle} 规范化后的窗口类型
 */
export function normalizeWindowStyle(value: unknown): WindowStyle {
  if (value === "popup" || value === "cover" || value === "replace") {
    return value;
  }
  if (value === "normal") {
    return "replace";
  }
  return "cover";
}

/**
 * 规范化背景类型
 * @param {unknown} value - 原始背景类型
 * @returns {BackgroundKind} 规范化后的背景类型
 */
function normalizeBackgroundKind(value: unknown): BackgroundKind {
  if (value === "color" || value === "image" || value === "gradient") {
    return value;
  }
  return "color";
}

/**
 * 归一化数字输入
 * @param {unknown} value - 原始数值
 * @param {number} fallback - 回退值
 * @returns {number} 归一化后的数字
 */
function normalizeNumber(value: unknown, fallback: number): number {
  const numeric = Number(value);
  return Number.isFinite(numeric) && numeric > 0 ? numeric : fallback;
}

/**
 * 构建页面配置补丁（仅包含保留字段）
 * @param {PageInspectorConfigInput} input - 表单输入
 * @returns {PageInspectorConfigPatch} 页面配置补丁
 */
export function buildPageConfigPatch(input: PageInspectorConfigInput): PageInspectorConfigPatch {
  const autoFit = Boolean(input.autoFit);
  return {
    description: String(input.description || ""),
    width: normalizeNumber(input.width, DEFAULT_WIDTH),
    height: normalizeNumber(input.height, DEFAULT_HEIGHT),
    autoFit,
    lockAspectRatio: autoFit && Boolean(input.lockAspectRatio),
    enableMinSize: autoFit && Boolean(input.enableMinSize),
    windowStyle: normalizeWindowStyle(input.windowStyle),
    permissionDesc: String(input.permissionDesc || DEFAULT_PERMISSION_DESC),
    background: {
      kind: normalizeBackgroundKind(input.backgroundKind),
      value: String(input.backgroundValue || DEFAULT_BACKGROUND_COLOR),
    },
  };
}
