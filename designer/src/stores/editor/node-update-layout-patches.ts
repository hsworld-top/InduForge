/**
 * updateNode 流水线中的纯函数：尺寸解析与 El 布局相关 patch 裁剪。
 * 从 editor-store 拆出以降低单文件体积，逻辑与原先一致。
 */

import type { DocumentModel } from "@/editor-core/document/DocumentModel";
import type { ComponentNode } from "@/editor-core/document/types";

const SIZE_NUMBER_REGEX = /^[\d.]+$/;

export function parseSizeToNumber(value: unknown): number | undefined {
  if (value === null || value === undefined) return undefined;
  if (typeof value === "number" && Number.isFinite(value)) return value;
  const text = String(value).trim();
  if (!text || text === "auto") return undefined;
  if (text.endsWith("px")) {
    const num = Number.parseFloat(text.slice(0, -2));
    return Number.isFinite(num) ? num : undefined;
  }
  if (SIZE_NUMBER_REGEX.test(text)) {
    const num = Number.parseFloat(text);
    return Number.isFinite(num) ? num : undefined;
  }
  return undefined;
}

export function resolveElLayoutMinHeight(layoutProps: Record<string, unknown> | undefined): number {
  const props = layoutProps || {};
  if (!props) return 0;
  return 0;
}

export function syncAbsoluteSizePatch(
  node: ComponentNode | null,
  patch: Partial<ComponentNode>,
): Partial<ComponentNode> {
  if (!node || !patch?.style) return patch;
  if (node.positioning !== "absolute" || !node.absolutePos) return patch;

  const widthValue = parseSizeToNumber(patch.style.width);
  const heightValue = parseSizeToNumber(patch.style.height);
  if (widthValue === undefined && heightValue === undefined) return patch;

  const nextAbs = { ...(node.absolutePos || {}) };
  if (widthValue !== undefined) {
    nextAbs.w = Math.max(1, Math.round(widthValue));
  }
  if (heightValue !== undefined) {
    nextAbs.h = Math.max(1, Math.round(heightValue));
  }

  const baseLayoutItem = patch.layoutItem || node.layoutItem;
  let nextLayoutItem = baseLayoutItem;
  if (baseLayoutItem?.free?.abs) {
    nextLayoutItem = {
      ...(baseLayoutItem || {}),
      free: {
        ...(baseLayoutItem.free || {}),
        abs: {
          ...(baseLayoutItem.free.abs || {}),
          ...(widthValue !== undefined ? { w: nextAbs.w } : {}),
          ...(heightValue !== undefined ? { h: nextAbs.h } : {}),
        },
      },
    };
  }

  return {
    ...patch,
    absolutePos: nextAbs,
    ...(nextLayoutItem ? { layoutItem: nextLayoutItem } : {}),
  };
}

export function syncElLayoutMinHeightPatch(
  node: ComponentNode | null,
  patch: Partial<ComponentNode>,
  nextProps: Record<string, unknown>,
): Partial<ComponentNode> {
  if (!node || node.type !== "ElLayout") return patch;
  const nextPositioning = patch.positioning ?? node.positioning;
  const baseAbs = patch.absolutePos || node.absolutePos || node.layoutItem?.free?.abs;
  if (nextPositioning !== "absolute" || !baseAbs) return patch;
  const minHeight = resolveElLayoutMinHeight(nextProps);
  if (!minHeight) return patch;
  const currentHeight = Number(baseAbs.h) || 0;
  if (currentHeight >= minHeight) return patch;
  const nextAbs = { ...baseAbs, h: Math.max(1, minHeight) };
  const baseLayoutItem = patch.layoutItem || node.layoutItem;
  let nextLayoutItem = baseLayoutItem;
  if (baseLayoutItem?.free?.abs) {
    nextLayoutItem = {
      ...(baseLayoutItem || {}),
      free: {
        ...(baseLayoutItem.free || {}),
        abs: {
          ...(baseLayoutItem.free.abs || {}),
          h: nextAbs.h,
        },
      },
    };
  }
  return {
    ...patch,
    absolutePos: nextAbs,
    ...(nextLayoutItem ? { layoutItem: nextLayoutItem } : {}),
  };
}

export function clampElContainerSizePatch(
  doc: DocumentModel | null | undefined,
  node: ComponentNode | null,
  patch: Partial<ComponentNode>,
): Partial<ComponentNode> {
  if (!node || node.type !== "ElContainer" || !patch?.style) return patch;

  const children = node.children || [];
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

  const props = (node.props || {}) as Record<string, unknown>;
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

  const nextStyle = { ...(patch.style || {}) };
  const widthValue = parseSizeToNumber(nextStyle.width);
  const heightValue = parseSizeToNumber(nextStyle.height);
  if (widthValue !== undefined && minWidth > 0) {
    nextStyle.width = `${Math.max(widthValue, minWidth)}px`;
  }
  if (heightValue !== undefined && minHeight > 0) {
    nextStyle.height = `${Math.max(heightValue, minHeight)}px`;
  }

  return { ...patch, style: nextStyle };
}

export function clampElColSpanPatch(
  doc: DocumentModel | null | undefined,
  node: ComponentNode | null,
  patch: Partial<ComponentNode>,
): Partial<ComponentNode> {
  if (!node || node.type !== "ElCol" || !patch?.props) return patch;
  if (!Object.hasOwn(patch.props, "span")) return patch;
  const parentNode = doc?.getParent?.(node.id);
  if (!parentNode || parentNode.type !== "ElLayoutRow") return patch;

  const colIds = (parentNode.children || []).filter((childId: string) => {
    const childNode = doc?.getNode?.(childId);
    return childNode?.type === "ElCol";
  });
  const anchorIndex = colIds.indexOf(node.id);
  const leftIds = anchorIndex >= 0 ? colIds.slice(0, anchorIndex) : [];
  const rightCount = anchorIndex >= 0 ? Math.max(0, colIds.length - anchorIndex - 1) : 0;
  const leftTotal = leftIds.reduce((sum: number, colId: string) => {
    const colNode = doc?.getNode?.(colId);
    const span = Number(colNode?.props?.span) || 0;
    return sum + Math.max(1, Math.min(24, span));
  }, 0);
  const maxSpan = Math.max(1, 24 - leftTotal - rightCount);
  const nextSpan = Number(patch.props.span) || 1;
  const clamped = Math.max(1, Math.min(maxSpan, nextSpan));
  if (clamped === nextSpan) return patch;
  return {
    ...patch,
    props: { ...(patch.props || {}), span: clamped },
  };
}

export function clampElColOffsetPatch(
  doc: DocumentModel | null | undefined,
  node: ComponentNode | null,
  patch: Partial<ComponentNode>,
): Partial<ComponentNode> {
  if (!node || node.type !== "ElCol" || !patch?.props) return patch;
  if (!Object.hasOwn(patch.props, "offset")) return patch;
  const parentNode = doc?.getParent?.(node.id);
  if (!parentNode || parentNode.type !== "ElLayoutRow") return patch;
  const colIds = (parentNode.children || []).filter((childId: string) => {
    const childNode = doc?.getNode?.(childId);
    return childNode?.type === "ElCol";
  });
  const anchorIndex = colIds.indexOf(node.id);
  if (anchorIndex < 0) return patch;

  const spans = colIds.map((colId: string) => {
    const colNode = doc?.getNode?.(colId);
    const span = Number(colNode?.props?.span) || 1;
    return Math.max(1, Math.min(24, span));
  });
  if (Object.hasOwn(patch.props, "span")) {
    const nextSpan = Number(patch.props.span) || 1;
    spans[anchorIndex] = Math.max(1, Math.min(24, nextSpan));
  }

  const offsets = colIds.map((colId: string) => {
    const colNode = doc?.getNode?.(colId);
    const offset = Number(colNode?.props?.offset) || 0;
    return Math.max(0, Math.min(24, offset));
  });
  const nextOffset = Math.max(0, Math.min(24, Number(patch.props.offset) || 0));
  offsets[anchorIndex] = nextOffset;

  const fixedSpanTotal = spans
    .slice(0, anchorIndex + 1)
    .reduce((sum: number, value: number) => sum + value, 0);
  const offsetOthers = offsets.reduce(
    (sum: number, value: number, index: number) => (index === anchorIndex ? sum : sum + value),
    0,
  );
  const rightCount = Math.max(0, colIds.length - anchorIndex - 1);
  const maxOffset = Math.max(0, 24 - fixedSpanTotal - offsetOthers - rightCount);
  const clampedOffset = Math.min(nextOffset, maxOffset);
  if (clampedOffset === nextOffset) return patch;
  return {
    ...patch,
    props: { ...(patch.props || {}), offset: clampedOffset },
  };
}

export function clampElColShiftPatch(
  doc: DocumentModel | null | undefined,
  node: ComponentNode | null,
  patch: Partial<ComponentNode>,
): Partial<ComponentNode> {
  if (!node || node.type !== "ElCol" || !patch?.props) return patch;
  const hasPush = Object.hasOwn(patch.props, "push");
  const hasPull = Object.hasOwn(patch.props, "pull");
  if (!hasPush && !hasPull) return patch;
  const parentNode = doc?.getParent?.(node.id);
  if (!parentNode || parentNode.type !== "ElLayoutRow") return patch;

  const mergedProps = {
    ...(node.props || {}),
    ...(patch.props || {}),
  } as Record<string, unknown>;
  const span = Math.max(1, Math.min(24, Number(mergedProps.span) || 1));
  const offset = Math.max(0, Math.min(24, Number(mergedProps.offset) || 0));
  const prevPush = Math.max(0, Math.min(24, Number(node.props?.push) || 0));
  const prevPull = Math.max(0, Math.min(24, Number(node.props?.pull) || 0));
  const push = Math.max(0, Math.min(24, Number(mergedProps.push) || 0));
  const pull = Math.max(0, Math.min(24, Number(mergedProps.pull) || 0));
  const colIds = (parentNode.children || []).filter((childId: string) => {
    const childNode = doc?.getNode?.(childId);
    return childNode?.type === "ElCol";
  });
  const colIndex = colIds.indexOf(node.id);
  let leftEdge = offset;
  if (colIndex > 0) {
    leftEdge = colIds.slice(0, colIndex).reduce((sum: number, colId: string) => {
      const colNode = doc?.getNode?.(colId);
      if (!colNode) return sum;
      const colSpan = Number(colNode.props?.span) || 1;
      const colOffset = Number(colNode.props?.offset) || 0;
      return sum + Math.max(1, Math.min(24, colSpan)) + Math.max(0, Math.min(24, colOffset));
    }, 0);
    leftEdge += offset;
  }
  const minShift = -leftEdge;
  const maxShift = 24 - leftEdge - span;
  const desiredShift = push - pull;
  const clampedShift = Math.min(maxShift, Math.max(minShift, desiredShift));

  const changedPush = hasPush && push !== prevPush;
  const changedPull = hasPull && pull !== prevPull;
  let nextPush = push;
  let nextPull = pull;
  if (changedPush && !changedPull) {
    nextPush = Math.max(0, clampedShift + pull);
    nextPull = pull;
  } else if (changedPull && !changedPush) {
    nextPull = Math.max(0, push - clampedShift);
    nextPush = push;
  } else if (changedPush && changedPull) {
    nextPush = Math.max(0, clampedShift + pull);
    nextPull = pull;
  } else {
    return patch;
  }

  if (nextPush === push && nextPull === pull) return patch;
  return {
    ...patch,
    props: { ...(patch.props || {}), push: nextPush, pull: nextPull },
  };
}

export function clampElLayoutRowColumnsPatch(
  node: ComponentNode | null,
  patch: Partial<ComponentNode>,
): Partial<ComponentNode> {
  if (!node || node.type !== "ElLayoutRow" || !patch?.props) return patch;
  if (!Object.hasOwn(patch.props, "columns")) return patch;
  const raw = Number(patch.props.columns);
  if (!Number.isFinite(raw)) return patch;
  const clamped = Math.max(1, Math.min(24, raw));
  if (clamped === raw) return patch;
  return {
    ...patch,
    props: { ...(patch.props || {}), columns: clamped },
  };
}
