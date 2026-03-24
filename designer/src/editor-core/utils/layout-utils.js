/**
 * 布局工具函数
 *
 * 从 NodeRenderer 抽取的布局/尺寸计算辅助函数，供 composable 和组件复用。
 *
 * @module editor-core/utils/layout-utils
 */

/**
 * 解析节点自由布局信息
 * @param {import('@/editor-core').ComponentNode} currentNode - 当前节点
 * @param {HTMLElement | null} nodeElement - 节点 DOM 元素
 * @returns {{ x: number, y: number, w: number, h: number, z: number }}
 */
export function resolveAbsoluteLayout(currentNode, nodeElement) {
  const fallbackAbs = currentNode.layoutItem?.free?.abs || {};
  const useAbsolute =
    currentNode.positioning === "absolute" ||
    currentNode.layoutItem?.free?.mode === "abs";
  const absolutePos = useAbsolute ? currentNode.absolutePos || fallbackAbs : {};
  const rect = nodeElement?.getBoundingClientRect?.();
  const width = Number.isFinite(absolutePos.w)
    ? absolutePos.w
    : (rect?.width ?? 120);
  const height = Number.isFinite(absolutePos.h)
    ? absolutePos.h
    : (rect?.height ?? 40);

  let x = Number.isFinite(absolutePos.x) ? absolutePos.x : 0;
  let y = Number.isFinite(absolutePos.y) ? absolutePos.y : 0;

  if (!Number.isFinite(absolutePos.x) || !Number.isFinite(absolutePos.y)) {
    const parentElement =
      nodeElement?.parentElement?.closest?.("[data-node-id]");
    const parentRect = parentElement?.getBoundingClientRect?.();
    if (rect && parentRect) {
      x = rect.left - parentRect.left;
      y = rect.top - parentRect.top;
    }
  }

  return {
    x: Math.round(x),
    y: Math.round(y),
    w: Math.max(1, Math.round(width)),
    h: Math.max(1, Math.round(height)),
    z: Number.isFinite(absolutePos.z) ? absolutePos.z : 1,
  };
}

/**
 * 生成流式布局下的尺寸样式（清理绝对布局尺寸）
 * @param {Record<string, any> | undefined} currentStyle - 当前样式
 * @returns {Record<string, any>}
 */
export function buildFlowResetStyle(currentStyle) {
  const nextStyle = { ...(currentStyle || {}) };
  delete nextStyle.width;
  delete nextStyle.height;
  return nextStyle;
}

/**
 * 解析尺寸为像素值
 * @param {string | number | undefined | null} value - 尺寸值
 * @returns {number | undefined}
 */
export function parseSizeToNumber(value) {
  if (value === null || value === undefined) return undefined;
  if (typeof value === "number" && Number.isFinite(value)) return value;
  const text = String(value).trim();
  if (!text || text === "auto") return undefined;
  if (text.endsWith("px")) {
    const num = Number.parseFloat(text.slice(0, -2));
    return Number.isFinite(num) ? num : undefined;
  }
  if (/^[\d.]+$/.test(text)) {
    const num = Number.parseFloat(text);
    return Number.isFinite(num) ? num : undefined;
  }
  return undefined;
}

/**
 * 计算 Layout 布局最小高度（当前不强制最小高度）
 * @param {import('@/editor-core').ComponentNode | null} layoutNode - Layout 节点
 * @returns {number}
 */
export function resolveElLayoutMinHeight(layoutNode) {
  if (!layoutNode || layoutNode.type !== "ElLayout") return 0;
  return 0;
}

/**
 * 计算容器最小尺寸，避免小于内部区域
 * @param {import('@/editor-core').ComponentNode | null} containerNode - 容器节点
 * @param {import('@/editor-core').Document} doc - 文档实例
 * @returns {{ width: number, height: number } | null}
 */
export function resolveElContainerMinSize(containerNode, doc) {
  if (!containerNode || containerNode.type !== "ElContainer") return null;
  const children = containerNode.children || [];
  let hasHeader = false;
  let hasFooter = false;
  let hasAside = false;
  let hasMain = false;
  for (const childId of children) {
    const childNode = doc?.getNode?.(childId);
    if (!childNode) continue;
    if (childNode.type === "ElHeader") hasHeader = true;
    if (childNode.type === "ElFooter") hasFooter = true;
    if (childNode.type === "ElAside") hasAside = true;
    if (childNode.type === "ElMain") hasMain = true;
  }

  const props = containerNode.props || {};
  if (typeof props.showHeader === "boolean") hasHeader = props.showHeader;
  if (typeof props.showFooter === "boolean") hasFooter = props.showFooter;
  if (typeof props.showAside === "boolean") hasAside = props.showAside;
  if (typeof props.showMain === "boolean") hasMain = props.showMain;

  const headerHeight = parseSizeToNumber(props.headerHeight) ?? 60;
  const footerHeight = parseSizeToNumber(props.footerHeight) ?? 60;
  const asideWidth = parseSizeToNumber(props.asideWidth) ?? 200;
  const minBodySize = 40;

  const hasBody = hasAside || hasMain;
  let minWidth = 0;
  if (hasAside && hasMain) {
    minWidth = asideWidth + minBodySize;
  } else if (hasAside) {
    minWidth = asideWidth;
  } else if (hasMain) {
    minWidth = minBodySize;
  }

  let minHeight = 0;
  if (hasHeader) minHeight += headerHeight;
  if (hasFooter) minHeight += footerHeight;
  if (hasBody) minHeight += minBodySize;

  if (minWidth <= 0 && minHeight <= 0) return null;
  return { width: minWidth, height: minHeight };
}

/**
 * 根据容器高度缩放 Header/Footer 尺寸
 * @param {import('@/editor-core').ComponentNode} containerNode - 容器节点
 * @param {number} baseWidth - 原始宽度
 * @param {number} nextWidth - 目标宽度
 * @param {number} baseHeight - 原始高度
 * @param {number} nextHeight - 目标高度
 * @param {{ headerHeight?: number, footerHeight?: number, asideWidth?: number }} baseSectionSizes - 基础区域尺寸
 * @returns {Record<string, string> | null}
 */
export function buildContainerSectionSizePatch(
  containerNode,
  baseWidth,
  nextWidth,
  baseHeight,
  nextHeight,
  baseSectionSizes,
) {
  if (!containerNode || containerNode.type !== "ElContainer") return null;
  const scaleX =
    Number.isFinite(baseWidth) && baseWidth > 0 && Number.isFinite(nextWidth)
      ? nextWidth / baseWidth
      : 1;
  const scaleY =
    Number.isFinite(baseHeight) && baseHeight > 0 && Number.isFinite(nextHeight)
      ? nextHeight / baseHeight
      : 1;
  const hasScaleX = Number.isFinite(scaleX) && Math.abs(scaleX - 1) >= 0.001;
  const hasScaleY = Number.isFinite(scaleY) && Math.abs(scaleY - 1) >= 0.001;
  if (!hasScaleX && !hasScaleY) return null;
  const props = containerNode.props || {};
  const minSectionSize = 40;
  const patch = {};

  if (hasScaleY && props.showHeader !== false) {
    const headerHeight = baseSectionSizes?.headerHeight;
    if (Number.isFinite(headerHeight)) {
      patch.headerHeight = `${Math.max(
        minSectionSize,
        Math.round(headerHeight * scaleY),
      )}px`;
    }
  }
  if (hasScaleY && props.showFooter !== false) {
    const footerHeight = baseSectionSizes?.footerHeight;
    if (Number.isFinite(footerHeight)) {
      patch.footerHeight = `${Math.max(
        minSectionSize,
        Math.round(footerHeight * scaleY),
      )}px`;
    }
  }
  if (hasScaleX && props.showAside !== false) {
    const asideWidth = baseSectionSizes?.asideWidth;
    if (Number.isFinite(asideWidth)) {
      patch.asideWidth = `${Math.max(
        minSectionSize,
        Math.round(asideWidth * scaleX),
      )}px`;
    }
  }

  return Object.keys(patch).length > 0 ? patch : null;
}

/**
 * 按容器尺寸限制区域大小，避免超出容器
 * @param {import('@/editor-core').ComponentNode} containerNode - 容器节点
 * @param {number} width - 容器宽度
 * @param {number} height - 容器高度
 * @returns {Record<string, string> | null}
 */
export function clampElContainerPropsBySize(containerNode, width, height) {
  if (!containerNode || containerNode.type !== "ElContainer") return null;
  if (!Number.isFinite(width) || !Number.isFinite(height)) return null;
  if (width <= 0 || height <= 0) return null;
  const props = containerNode.props || {};
  const hasHeader = props.showHeader !== false;
  const hasFooter = props.showFooter !== false;
  const hasAside = props.showAside !== false;
  const hasMain = props.showMain !== false;
  const hasBody = hasAside || hasMain;
  const minBodySize = 40;
  const minSectionSize = 40;
  const headerHeight = parseSizeToNumber(props.headerHeight) ?? 60;
  const footerHeight = parseSizeToNumber(props.footerHeight) ?? 60;
  let asideWidth = parseSizeToNumber(props.asideWidth) ?? 200;
  const patch = {};

  if (hasAside) {
    const maxAside = Math.max(0, width - (hasMain ? minBodySize : 0));
    if (Number.isFinite(maxAside)) {
      asideWidth = Math.min(asideWidth, maxAside);
      asideWidth = Math.max(minSectionSize, asideWidth);
      const nextAside = `${Math.round(asideWidth)}px`;
      if (nextAside !== props.asideWidth) {
        patch.asideWidth = nextAside;
      }
    }
  }

  if (hasHeader || hasFooter) {
    const available = Math.max(0, height - (hasBody ? minBodySize : 0));
    let nextHeader = hasHeader ? headerHeight : 0;
    let nextFooter = hasFooter ? footerHeight : 0;
    const total = nextHeader + nextFooter;
    if (total > available && total > 0) {
      const scale = available / total;
      nextHeader = Math.max(minSectionSize, Math.round(nextHeader * scale));
      nextFooter = Math.max(minSectionSize, Math.round(nextFooter * scale));
    } else {
      if (hasHeader) nextHeader = Math.min(nextHeader, available);
      if (hasFooter) nextFooter = Math.min(nextFooter, available);
    }
    if (hasHeader) {
      const nextHeaderText = `${nextHeader}px`;
      if (nextHeaderText !== props.headerHeight) {
        patch.headerHeight = nextHeaderText;
      }
    }
    if (hasFooter) {
      const nextFooterText = `${nextFooter}px`;
      if (nextFooterText !== props.footerHeight) {
        patch.footerHeight = nextFooterText;
      }
    }
  }

  return Object.keys(patch).length > 0 ? patch : null;
}

/**
 * 解析 ElContainer 的 ElMain 子节点
 * @param {import('@/editor-core').Document} doc - 文档实例
 * @param {import('@/editor-core').ComponentNode | null} container - 容器节点
 * @returns {import('@/editor-core').ComponentNode | null}
 */
export function resolveElContainerMain(doc, container) {
  if (!container || container.type !== "ElContainer") return null;
  const mainChildId = (container.children || []).find((childId) => {
    const childNode = doc?.getNode?.(childId);
    return childNode?.type === "ElMain";
  });
  return mainChildId ? doc?.getNode?.(mainChildId) || null : null;
}
