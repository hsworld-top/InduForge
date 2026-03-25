/**
 * 对齐、分布、等大小命令
 * 支持节点（node）和图形（graphic）两种元素类型
 */

import type { DocumentModel } from "../document/DocumentModel";
import type {
  AbsolutePosition,
  GraphicNode,
  GraphicProps,
  LayoutItem,
  SelectableElement,
} from "../document/types";
import { Command } from "./Command";

export interface Bounds {
  x: number;
  y: number;
  width: number;
  height: number;
}

type AlignType = "left" | "centerH" | "right" | "top" | "centerV" | "bottom";

/**
 * 从文档中读取元素的位置和尺寸
 * 按优先级：absolutePos → layoutItem.free.abs → style.left/top/width/height
 * 跳过 positioning === "flow" 的节点（流式布局不支持自由对齐）
 */
function getElementBounds(doc: DocumentModel, element: SelectableElement): Bounds | null {
  if (element.kind === "node") {
    const node = doc.getNode(element.id);
    if (!node) return null;
    if (node.positioning === "flow") return null;

    const absolutePos = node.absolutePos;
    const freeAbsLayout = node.layoutItem?.free?.abs;
    if (
      absolutePos &&
      (node.positioning === "absolute" ||
        Number.isFinite(absolutePos.x) ||
        Number.isFinite(absolutePos.y))
    ) {
      return {
        x: Number.isFinite(absolutePos.x) ? absolutePos.x : 0,
        y: Number.isFinite(absolutePos.y) ? absolutePos.y : 0,
        width: Number.isFinite(absolutePos.w) ? absolutePos.w : 100,
        height: Number.isFinite(absolutePos.h) ? absolutePos.h : 100,
      };
    }
    if (freeAbsLayout) {
      return {
        x: Number.isFinite(freeAbsLayout.x) ? freeAbsLayout.x : 0,
        y: Number.isFinite(freeAbsLayout.y) ? freeAbsLayout.y : 0,
        width: Number.isFinite(freeAbsLayout.w) ? freeAbsLayout.w : 100,
        height: Number.isFinite(freeAbsLayout.h) ? freeAbsLayout.h : 100,
      };
    }
    if (!node.style) return null;
    return {
      x: Number(node.style.left) || 0,
      y: Number(node.style.top) || 0,
      width: Number(node.style.width) || 100,
      height: Number(node.style.height) || 100,
    };
  }

  const graphic = doc.getGraphic(element.id);
  if (!graphic?.props) return null;
  return getGraphicBounds(graphic);
}

/** 获取图形的包围盒 */
function getGraphicBounds(graphic: GraphicNode): Bounds | null {
  const props = graphic.props as GraphicProps;
  switch (graphic.type) {
    case "Canvas.Rect":
      return {
        x: props.x || 0,
        y: props.y || 0,
        width: props.width || 0,
        height: props.height || 0,
      };
    case "Canvas.Circle": {
      const radius = props.radius || 0;
      return {
        x: (props.cx || 0) - radius,
        y: (props.cy || 0) - radius,
        width: radius * 2,
        height: radius * 2,
      };
    }
    case "Canvas.Ellipse": {
      const rx = props.rx || 0;
      const ry = props.ry || 0;
      return {
        x: (props.cx || 0) - rx,
        y: (props.cy || 0) - ry,
        width: rx * 2,
        height: ry * 2,
      };
    }
    case "Canvas.Line":
    case "Canvas.Polygon":
    case "Canvas.Pipe":
      if (props.points && props.points.length > 0) {
        const xs = props.points.map(([x]) => x);
        const ys = props.points.map(([, y]) => y);
        return {
          x: Math.min(...xs),
          y: Math.min(...ys),
          width: Math.max(...xs) - Math.min(...xs),
          height: Math.max(...ys) - Math.min(...ys),
        };
      }
      return null;
    case "Canvas.Text":
      return {
        x: props.x || 0,
        y: props.y || 0,
        width: 100,
        height: (props.fontSize || 14) * 1.5,
      };
    case "Canvas.Symbol":
      return {
        x: props.x || 0,
        y: props.y || 0,
        width: 50 * (props.scale || 1),
        height: 50 * (props.scale || 1),
      };
    default:
      return null;
  }
}

function applyPosition(
  doc: DocumentModel,
  element: SelectableElement,
  newX: number,
  newY: number,
): void {
  if (element.kind === "node") {
    const node = doc.getNode(element.id);
    if (!node) return;
    if (node.positioning === "flow") return;

    const absolutePos = node.absolutePos;
    const freeAbsLayout = node.layoutItem?.free?.abs;
    if (
      absolutePos &&
      (node.positioning === "absolute" ||
        Number.isFinite(absolutePos.x) ||
        Number.isFinite(absolutePos.y))
    ) {
      doc._updateNode(element.id, {
        absolutePos: { ...absolutePos, x: newX, y: newY },
      });
      return;
    }
    if (freeAbsLayout) {
      const nextLayoutItem = (
        node.layoutItem ? JSON.parse(JSON.stringify(node.layoutItem)) : {}
      ) as LayoutItem;
      const prevFree = nextLayoutItem.free;
      const abs: AbsolutePosition = {
        x: newX,
        y: newY,
        w: Number.isFinite(freeAbsLayout.w) ? freeAbsLayout.w : 100,
        h: Number.isFinite(freeAbsLayout.h) ? freeAbsLayout.h : 100,
      };
      if (freeAbsLayout.z !== undefined) abs.z = freeAbsLayout.z;
      nextLayoutItem.free = {
        mode: prevFree?.mode ?? "abs",
        abs,
        ...(prevFree?.constraints ? { constraints: prevFree.constraints } : {}),
        ...(prevFree?.z !== undefined ? { z: prevFree.z } : {}),
      };
      doc._updateNode(element.id, { layoutItem: nextLayoutItem });
      return;
    }
    const newStyle = { ...node.style, left: newX, top: newY };
    doc._updateNode(element.id, { style: newStyle });
  } else {
    const graphic = doc.getGraphic(element.id);
    if (!graphic) return;
    const oldBounds = getGraphicBounds(graphic);
    if (!oldBounds) return;
    const dx = newX - oldBounds.x;
    const dy = newY - oldBounds.y;
    const newProps = JSON.parse(JSON.stringify(graphic.props)) as GraphicProps;
    if (typeof newProps.x === "number") newProps.x += dx;
    if (typeof newProps.y === "number") newProps.y += dy;
    if (typeof newProps.cx === "number") newProps.cx += dx;
    if (typeof newProps.cy === "number") newProps.cy += dy;
    if (Array.isArray(newProps.points)) {
      newProps.points = newProps.points.map(([x, y]) => [x + dx, y + dy]);
    }
    doc._updateGraphic(element.id, { props: newProps });
  }
}

function applySize(
  doc: DocumentModel,
  element: SelectableElement,
  newWidth: number | null,
  newHeight: number | null,
): void {
  if (element.kind === "node") {
    const node = doc.getNode(element.id);
    if (!node) return;
    if (node.positioning === "flow") return;

    const absolutePos = node.absolutePos;
    const freeAbsLayout = node.layoutItem?.free?.abs;
    if (
      absolutePos &&
      (node.positioning === "absolute" ||
        Number.isFinite(absolutePos.w) ||
        Number.isFinite(absolutePos.h))
    ) {
      const next = { ...absolutePos };
      if (newWidth !== null) next.w = newWidth;
      if (newHeight !== null) next.h = newHeight;
      doc._updateNode(element.id, { absolutePos: next });
      return;
    }
    if (freeAbsLayout) {
      const nextLayoutItem = (
        node.layoutItem ? JSON.parse(JSON.stringify(node.layoutItem)) : {}
      ) as LayoutItem;
      const prevFree = nextLayoutItem.free;
      const base = prevFree?.abs ?? freeAbsLayout;
      const abs: AbsolutePosition = {
        x: Number.isFinite(base.x) ? base.x : 0,
        y: Number.isFinite(base.y) ? base.y : 0,
        w: newWidth !== null ? newWidth : Number.isFinite(base.w) ? base.w : 100,
        h: newHeight !== null ? newHeight : Number.isFinite(base.h) ? base.h : 100,
      };
      if (base.z !== undefined) abs.z = base.z;
      nextLayoutItem.free = {
        mode: prevFree?.mode ?? "abs",
        abs,
        ...(prevFree?.constraints ? { constraints: prevFree.constraints } : {}),
        ...(prevFree?.z !== undefined ? { z: prevFree.z } : {}),
      };
      doc._updateNode(element.id, { layoutItem: nextLayoutItem });
      return;
    }
    const patch = { ...node.style };
    if (newWidth !== null) patch.width = newWidth;
    if (newHeight !== null) patch.height = newHeight;
    doc._updateNode(element.id, { style: patch });
  } else {
    const graphic = doc.getGraphic(element.id);
    if (!graphic) return;
    const newProps = JSON.parse(JSON.stringify(graphic.props)) as GraphicProps;
    if (graphic.type === "Canvas.Rect") {
      if (newWidth !== null) newProps.width = newWidth;
      if (newHeight !== null) newProps.height = newHeight;
    } else if (graphic.type === "Canvas.Circle") {
      const size = newWidth ?? newHeight;
      if (size !== null) newProps.radius = size / 2;
    } else if (graphic.type === "Canvas.Ellipse") {
      if (newWidth !== null) newProps.rx = newWidth / 2;
      if (newHeight !== null) newProps.ry = newHeight / 2;
    }
    doc._updateGraphic(element.id, { props: newProps });
  }
}

/** 对齐元素命令 */
export class AlignElementsCommand extends Command {
  private _elements!: SelectableElement[];
  private _alignType!: AlignType;
  private _oldBounds!: Map<string, Bounds>;

  get type(): string {
    return "AlignElements";
  }

  constructor(elements: SelectableElement[], alignType: AlignType) {
    super();
    this._elements = elements;
    this._alignType = alignType;
    this._oldBounds = new Map();
  }

  execute(doc: DocumentModel): void {
    const boundsMap = new Map<string, Bounds>();
    for (const el of this._elements) {
      const bounds = getElementBounds(doc, el);
      if (bounds) {
        boundsMap.set(el.id, bounds);
        this._oldBounds.set(el.id, { ...bounds });
      }
    }
    if (boundsMap.size < 2) return;

    const allBounds = [...boundsMap.values()];
    let target: number;

    switch (this._alignType) {
      case "left":
        target = Math.min(...allBounds.map((b) => b.x));
        for (const el of this._elements) {
          const b = boundsMap.get(el.id);
          if (b) applyPosition(doc, el, target, b.y);
        }
        break;
      case "right":
        target = Math.max(...allBounds.map((b) => b.x + b.width));
        for (const el of this._elements) {
          const b = boundsMap.get(el.id);
          if (b) applyPosition(doc, el, target - b.width, b.y);
        }
        break;
      case "centerH":
        target =
          Math.min(...allBounds.map((b) => b.x)) +
          (Math.max(...allBounds.map((b) => b.x + b.width)) -
            Math.min(...allBounds.map((b) => b.x))) /
            2;
        for (const el of this._elements) {
          const b = boundsMap.get(el.id);
          if (b) applyPosition(doc, el, target - b.width / 2, b.y);
        }
        break;
      case "top":
        target = Math.min(...allBounds.map((b) => b.y));
        for (const el of this._elements) {
          const b = boundsMap.get(el.id);
          if (b) applyPosition(doc, el, b.x, target);
        }
        break;
      case "bottom":
        target = Math.max(...allBounds.map((b) => b.y + b.height));
        for (const el of this._elements) {
          const b = boundsMap.get(el.id);
          if (b) applyPosition(doc, el, b.x, target - b.height);
        }
        break;
      case "centerV":
        target =
          Math.min(...allBounds.map((b) => b.y)) +
          (Math.max(...allBounds.map((b) => b.y + b.height)) -
            Math.min(...allBounds.map((b) => b.y))) /
            2;
        for (const el of this._elements) {
          const b = boundsMap.get(el.id);
          if (b) applyPosition(doc, el, b.x, target - b.height / 2);
        }
        break;
    }
  }

  undo(doc: DocumentModel): void {
    for (const el of this._elements) {
      const old = this._oldBounds.get(el.id);
      if (old) applyPosition(doc, el, old.x, old.y);
    }
  }

  getDescription(): string {
    const labels: Record<AlignType, string> = {
      left: "左对齐",
      right: "右对齐",
      centerH: "水平居中",
      top: "顶对齐",
      bottom: "底对齐",
      centerV: "垂直居中",
    };
    return labels[this._alignType] || "对齐";
  }
}

/** 分布元素命令 */
export class DistributeElementsCommand extends Command {
  private _elements!: SelectableElement[];
  private _direction!: "horizontal" | "vertical";
  private _oldBounds!: Map<string, Bounds>;

  get type(): string {
    return "DistributeElements";
  }

  constructor(elements: SelectableElement[], direction: "horizontal" | "vertical") {
    super();
    this._elements = elements;
    this._direction = direction;
    this._oldBounds = new Map();
  }

  execute(doc: DocumentModel): void {
    const items: { el: SelectableElement; bounds: Bounds }[] = [];
    for (const el of this._elements) {
      const bounds = getElementBounds(doc, el);
      if (bounds) {
        this._oldBounds.set(el.id, { ...bounds });
        items.push({ el, bounds });
      }
    }
    if (items.length < 3) return;

    if (this._direction === "horizontal") {
      items.sort((a, b) => a.bounds.x - b.bounds.x);
      const firstH = items[0];
      const lastH = items.at(-1);
      if (!firstH || !lastH) return;
      const totalWidth = items.reduce((s, it) => s + it.bounds.width, 0);
      const rangeStart = firstH.bounds.x;
      const rangeEnd = lastH.bounds.x + lastH.bounds.width;
      const gap = (rangeEnd - rangeStart - totalWidth) / (items.length - 1);
      let currentX = rangeStart;
      for (const it of items) {
        applyPosition(doc, it.el, currentX, it.bounds.y);
        currentX += it.bounds.width + gap;
      }
    } else {
      items.sort((a, b) => a.bounds.y - b.bounds.y);
      const firstV = items[0];
      const lastV = items.at(-1);
      if (!firstV || !lastV) return;
      const totalHeight = items.reduce((s, it) => s + it.bounds.height, 0);
      const rangeStart = firstV.bounds.y;
      const rangeEnd = lastV.bounds.y + lastV.bounds.height;
      const gap = (rangeEnd - rangeStart - totalHeight) / (items.length - 1);
      let currentY = rangeStart;
      for (const it of items) {
        applyPosition(doc, it.el, it.bounds.x, currentY);
        currentY += it.bounds.height + gap;
      }
    }
  }

  undo(doc: DocumentModel): void {
    for (const el of this._elements) {
      const old = this._oldBounds.get(el.id);
      if (old) applyPosition(doc, el, old.x, old.y);
    }
  }

  getDescription(): string {
    return this._direction === "horizontal" ? "水平等距分布" : "垂直等距分布";
  }
}

/** 统一尺寸命令 */
export class MatchSizeCommand extends Command {
  private _elements!: SelectableElement[];
  private _mode!: "width" | "height" | "both";
  private _referenceId!: string;
  private _oldBounds!: Map<string, Bounds>;

  get type(): string {
    return "MatchSize";
  }

  constructor(
    elements: SelectableElement[],
    mode: "width" | "height" | "both",
    referenceId: string,
  ) {
    super();
    this._elements = elements;
    this._mode = mode;
    this._referenceId = referenceId;
    this._oldBounds = new Map();
  }

  execute(doc: DocumentModel): void {
    let refBounds: Bounds | null = null;
    for (const el of this._elements) {
      const bounds = getElementBounds(doc, el);
      if (bounds) {
        this._oldBounds.set(el.id, { ...bounds });
        if (el.id === this._referenceId) refBounds = bounds;
      }
    }
    if (!refBounds) return;

    for (const el of this._elements) {
      if (el.id === this._referenceId) continue;
      const b = this._oldBounds.get(el.id);
      if (!b) continue;

      const newW = this._mode === "width" || this._mode === "both" ? refBounds.width : null;
      const newH = this._mode === "height" || this._mode === "both" ? refBounds.height : null;
      applySize(doc, el, newW, newH);
    }
  }

  undo(doc: DocumentModel): void {
    for (const el of this._elements) {
      if (el.id === this._referenceId) continue;
      const old = this._oldBounds.get(el.id);
      if (!old) continue;
      applySize(doc, el, old.width, old.height);
    }
  }

  getDescription(): string {
    const labels: Record<"width" | "height" | "both", string> = {
      width: "等宽",
      height: "等高",
      both: "等大小",
    };
    return labels[this._mode] || "统一尺寸";
  }
}

export default {
  AlignElementsCommand,
  DistributeElementsCommand,
  MatchSizeCommand,
};
