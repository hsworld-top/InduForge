/**
 * 画布主组件内部共用类型（插入线、画布配置等），不对外当稳定 API。
 */

import type { ComponentNode } from "@/editor-core/document/types";

/** 插入线方向（与模板 class 一致） */
export type CanvasInsertOrientation = "vertical" | "horizontal";

export interface CanvasInsertLineStyle {
  orientation: CanvasInsertOrientation;
  offset: number;
}

export interface CanvasInsertLineBox {
  left: number;
  top: number;
  width: number;
  height: number;
}

/** ElLayoutRow 列插入解析结果 */
export interface CanvasRowInsertTarget {
  rowNode: ComponentNode;
  index: number;
  lineBox: CanvasInsertLineBox | null;
  insertLine: CanvasInsertLineStyle | null;
}

/** ElLayout 行插入解析结果 */
export type CanvasLayoutInsertTarget =
  | { layoutNode: ComponentNode; index: number }
  | {
      layoutNode: ComponentNode;
      index: number;
      lineBox: CanvasInsertLineBox;
      insertLine: CanvasInsertLineStyle;
    };

/** 设计画布页 config 在壳层可能含扁平背景字段 */
export interface DesignCanvasPageConfig {
  width?: number;
  height?: number;
  showGrid?: boolean;
  enableSnap?: boolean;
  background?: { kind?: string; value?: string } | null;
  backgroundColor?: string;
  backgroundImage?: string;
  backgroundSize?: string;
  backgroundRepeat?: string;
  backgroundPosition?: string;
  [key: string]: unknown;
}
