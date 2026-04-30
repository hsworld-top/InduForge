/**
 * Button 属性面板辅助逻辑。
 *
 * shape 是设计态更易理解的圆角配置项；渲染前需要映射为
 * Element Plus Button 能识别的 round/circle，避免把面板辅助字段透传出去。
 */

import type { Component } from "vue";
import IconEpClose from "~icons/ep/close";
import IconEpDelete from "~icons/ep/delete";
import IconEpDocument from "~icons/ep/document";
import IconEpDownload from "~icons/ep/download";
import IconEpEditPen from "~icons/ep/edit-pen";
import IconEpFolder from "~icons/ep/folder";
import IconEpPlus from "~icons/ep/plus";
import IconEpRefresh from "~icons/ep/refresh";
import IconEpSearch from "~icons/ep/search";
import IconEpUpload from "~icons/ep/upload";

export type ButtonShape = "default" | "round" | "circle";

interface LooseRecord extends Record<string, unknown> {}

const buttonIconMap: Record<string, Component> = {
  Close: IconEpClose,
  Delete: IconEpDelete,
  Document: IconEpDocument,
  Download: IconEpDownload,
  EditPen: IconEpEditPen,
  Folder: IconEpFolder,
  Plus: IconEpPlus,
  Refresh: IconEpRefresh,
  Search: IconEpSearch,
  Upload: IconEpUpload,
};

export function resolveButtonShapeProps(shape: unknown): Pick<LooseRecord, "round" | "circle"> {
  if (shape === "round") {
    return { round: true, circle: false };
  }
  if (shape === "circle") {
    return { round: false, circle: true };
  }
  return { round: false, circle: false };
}

export function normalizeButtonRenderProps(resolvedProps: LooseRecord = {}): LooseRecord {
  const { text: _text, shape, ...elProps } = resolvedProps;

  const normalizedProps = {
    ...elProps,
  };
  if (shape !== undefined && shape !== null && shape !== "") {
    Object.assign(normalizedProps, resolveButtonShapeProps(shape));
  }
  if (typeof normalizedProps.icon === "string") {
    const icon = resolveButtonIcon(normalizedProps.icon);
    if (icon) {
      normalizedProps.icon = icon;
    } else {
      delete normalizedProps.icon;
    }
  }
  return normalizedProps;
}

export function resolveButtonIcon(icon: string): Component | string | undefined {
  const text = String(icon || "").trim();
  if (!text) return undefined;
  const normalizedName = text
    .replace(/^el-icon-/i, "")
    .split(/[-_\s]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join("");
  return buttonIconMap[normalizedName] || text;
}
