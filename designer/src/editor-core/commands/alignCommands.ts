// @ts-nocheck
/**
 * 对齐、分布、等大小命令
 * 支持节点（node）和图形（graphic）两种元素类型
 */

import { Command } from "./Command.js";

/**
 * @typedef {import('../document/DocumentModel.js').DocumentModel} DocumentModel
 * @typedef {{ kind: 'node' | 'graphic', id: string }} SelectableElement
 * @typedef {{ x: number, y: number, width: number, height: number }} Bounds
 */

/**
 * 从文档中读取元素的位置和尺寸
 * 按优先级：absolutePos → layoutItem.free.abs → style.left/top/width/height
 * 跳过 positioning === "flow" 的节点（流式布局不支持自由对齐）
 * @param {DocumentModel} doc
 * @param {SelectableElement} element
 * @returns {Bounds | null}
 */
function getElementBounds(doc, element) {
  if (element.kind === "node") {
    const node = doc.getNode(element.id);
    if (!node) return null;
    // 流式布局节点不支持自由对齐
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
    // 回退到 style
    if (!node.style) return null;
    return {
      x: node.style.left || 0,
      y: node.style.top || 0,
      width: node.style.width || 100,
      height: node.style.height || 100,
    };
  }

  const graphic = doc.getGraphic(element.id);
  if (!graphic?.props) return null;
  return getGraphicBounds(graphic);
}

/**
 * 获取图形的包围盒
 * @param {Object} graphic
 * @returns {Bounds | null}
 */
function getGraphicBounds(graphic) {
  const props = graphic.props;
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
      if (props.points?.length > 0) {
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

/**
 * 将位置变更应用到元素
 * 按优先级写入：absolutePos → layoutItem.free.abs → style.left/top
 * @param {DocumentModel} doc
 * @param {SelectableElement} element
 * @param {number} newX
 * @param {number} newY
 */
function applyPosition(doc, element, newX, newY) {
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
      const nextLayoutItem = JSON.parse(JSON.stringify(node.layoutItem || {}));
      if (!nextLayoutItem.free) nextLayoutItem.free = {};
      if (!nextLayoutItem.free.abs) nextLayoutItem.free.abs = {};
      nextLayoutItem.free.abs = {
        ...nextLayoutItem.free.abs,
        x: newX,
        y: newY,
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
    const newProps = JSON.parse(JSON.stringify(graphic.props));
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

/**
 * 将尺寸变更应用到元素
 * 按优先级写入：absolutePos.w/h → layoutItem.free.abs.w/h → style.width/height
 * @param {DocumentModel} doc
 * @param {SelectableElement} element
 * @param {number | null} newWidth
 * @param {number | null} newHeight
 */
function applySize(doc, element, newWidth, newHeight) {
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
      const nextLayoutItem = JSON.parse(JSON.stringify(node.layoutItem || {}));
      if (!nextLayoutItem.free) nextLayoutItem.free = {};
      if (!nextLayoutItem.free.abs) nextLayoutItem.free.abs = {};
      const nextAbs = { ...nextLayoutItem.free.abs };
      if (newWidth !== null) nextAbs.w = newWidth;
      if (newHeight !== null) nextAbs.h = newHeight;
      nextLayoutItem.free.abs = nextAbs;
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
    const newProps = JSON.parse(JSON.stringify(graphic.props));
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

/**
 * 对齐元素命令
 */
export class AlignElementsCommand extends Command {
  get type() {
    return "AlignElements";
  }

  /**
   * @param {SelectableElement[]} elements - 选中元素列表
   * @param {'left' | 'centerH' | 'right' | 'top' | 'centerV' | 'bottom'} alignType
   */
  constructor(elements, alignType) {
    super();
    this._elements = elements;
    this._alignType = alignType;
    /** @type {Map<string, Bounds>} */
    this._oldBounds = new Map();
  }

  execute(doc) {
    const boundsMap = new Map();
    for (const el of this._elements) {
      const bounds = getElementBounds(doc, el);
      if (bounds) {
        boundsMap.set(el.id, bounds);
        this._oldBounds.set(el.id, { ...bounds });
      }
    }
    if (boundsMap.size < 2) return;

    const allBounds = [...boundsMap.values()];
    let target;

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

  undo(doc) {
    for (const el of this._elements) {
      const old = this._oldBounds.get(el.id);
      if (old) applyPosition(doc, el, old.x, old.y);
    }
  }

  getDescription() {
    const labels = {
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

/**
 * 分布元素命令
 */
export class DistributeElementsCommand extends Command {
  get type() {
    return "DistributeElements";
  }

  /**
   * @param {SelectableElement[]} elements - 至少 3 个元素
   * @param {'horizontal' | 'vertical'} direction
   */
  constructor(elements, direction) {
    super();
    this._elements = elements;
    this._direction = direction;
    /** @type {Map<string, Bounds>} */
    this._oldBounds = new Map();
  }

  execute(doc) {
    const items = [];
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
      const totalWidth = items.reduce((s, it) => s + it.bounds.width, 0);
      const rangeStart = items[0].bounds.x;
      const rangeEnd =
        items[items.length - 1].bounds.x + items[items.length - 1].bounds.width;
      const gap = (rangeEnd - rangeStart - totalWidth) / (items.length - 1);
      let currentX = rangeStart;
      for (const it of items) {
        applyPosition(doc, it.el, currentX, it.bounds.y);
        currentX += it.bounds.width + gap;
      }
    } else {
      items.sort((a, b) => a.bounds.y - b.bounds.y);
      const totalHeight = items.reduce((s, it) => s + it.bounds.height, 0);
      const rangeStart = items[0].bounds.y;
      const rangeEnd =
        items[items.length - 1].bounds.y +
        items[items.length - 1].bounds.height;
      const gap = (rangeEnd - rangeStart - totalHeight) / (items.length - 1);
      let currentY = rangeStart;
      for (const it of items) {
        applyPosition(doc, it.el, it.bounds.x, currentY);
        currentY += it.bounds.height + gap;
      }
    }
  }

  undo(doc) {
    for (const el of this._elements) {
      const old = this._oldBounds.get(el.id);
      if (old) applyPosition(doc, el, old.x, old.y);
    }
  }

  getDescription() {
    return this._direction === "horizontal" ? "水平等距分布" : "垂直等距分布";
  }
}

/**
 * 统一尺寸命令
 */
export class MatchSizeCommand extends Command {
  get type() {
    return "MatchSize";
  }

  /**
   * @param {SelectableElement[]} elements
   * @param {'width' | 'height' | 'both'} mode
   * @param {string} referenceId - 基准元素 ID（主选中元素）
   */
  constructor(elements, mode, referenceId) {
    super();
    this._elements = elements;
    this._mode = mode;
    this._referenceId = referenceId;
    /** @type {Map<string, Bounds>} */
    this._oldBounds = new Map();
  }

  execute(doc) {
    let refBounds = null;
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

      const newW =
        this._mode === "width" || this._mode === "both"
          ? refBounds.width
          : null;
      const newH =
        this._mode === "height" || this._mode === "both"
          ? refBounds.height
          : null;
      applySize(doc, el, newW, newH);
    }
  }

  undo(doc) {
    for (const el of this._elements) {
      if (el.id === this._referenceId) continue;
      const old = this._oldBounds.get(el.id);
      if (!old) continue;
      applySize(doc, el, old.width, old.height);
    }
  }

  getDescription() {
    const labels = { width: "等宽", height: "等高", both: "等大小" };
    return labels[this._mode] || "统一尺寸";
  }
}

export default {
  AlignElementsCommand,
  DistributeElementsCommand,
  MatchSizeCommand,
};
