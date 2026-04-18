/**
 * 组件描述符注册中心：渲染标签、容器行为、子项布局策略等元数据
 */

import type { Component } from "vue";

export interface DescriptorNode {
  id?: string;
  label?: string;
  props?: Record<string, unknown>;
  [key: string]: unknown;
}

export type DescriptorRenderTagFn = (
  node: DescriptorNode | undefined,
  resolvedProps: Record<string, unknown> | undefined,
) => string | Component | undefined;

export type DescriptorRenderTag = string | DescriptorRenderTagFn;

export interface ComponentDescriptor {
  renderTag?: DescriptorRenderTag;
  isContainer?: boolean;
  defaultStyle?: Record<string, string | number>;
  containerStyle?: (node: DescriptorNode) => Record<string, string | number>;
  childPositioning?: "absolute" | "flow" | null;
  childFlowLayout?: Record<string, unknown> | null;
  childStyle?: (parentType: string) => Record<string, string | number>;
  childResizable?: boolean;
  isMovable?: boolean;
  acceptChildren?: boolean | string[];
  maxChildren?: number | null;
  isRegion?: boolean;
  flexDirection?: "row" | "column" | null;
  displayContent?: (node: DescriptorNode, resolvedProps: Record<string, unknown>) => string | null;
  slots?: Record<string, unknown> | null;
  renderKey?: (node: DescriptorNode, context: Record<string, unknown>) => string;
  propsFilter?: (resolvedProps: Record<string, unknown>) => Record<string, unknown>;
  customRenderer?: Component | null;
  childLayout?: "flex" | "grid" | "free" | "none" | null;
  defaultSize?: { width: number; height: number } | null;
}

const FIXED_LAYOUT_SHELL_SLOT_TYPES = new Set(["ElHeader", "ElAside", "ElMain", "ElFooter"]);

const REGION_DESIGNER_HINTS: Record<string, string> = {
  ElHeader: "Header区域",
  ElAside: "Aside区域",
  ElMain: "Main区域",
  ElFooter: "Footer区域",
};

const _descriptors = new Map<string, ComponentDescriptor>();

export function registerDescriptor(type: string, descriptor: ComponentDescriptor): void {
  if (!type) {
    throw new Error("[ComponentRegistry] registerDescriptor: type 不能为空");
  }
  _descriptors.set(type, { ...descriptor });
}

export function getDescriptor(type: string): ComponentDescriptor | null {
  return _descriptors.get(type) ?? null;
}

export function hasDescriptor(type: string): boolean {
  return _descriptors.has(type);
}

export function getRenderTag(
  type: string,
  node?: DescriptorNode,
  resolvedProps?: Record<string, unknown>,
): string | Component {
  if (!type) return "div";
  const descriptor = _descriptors.get(type);
  if (!descriptor) {
    throw new Error(`[descriptors] 未注册组件类型: ${type}`);
  }
  const renderTag = descriptor.renderTag;
  if (!renderTag) {
    throw new Error(`[descriptors] 描述符缺少 renderTag: ${type}`);
  }
  if (typeof renderTag === "function") {
    const resolved = renderTag(node, resolvedProps);
    return resolved ?? "div";
  }
  return renderTag;
}

export function isContainerType(type: string): boolean {
  return _descriptors.get(type)?.isContainer ?? false;
}

export function getChildPositioning(parentType: string): "absolute" | "flow" | null {
  return _descriptors.get(parentType)?.childPositioning ?? null;
}

export function getChildFlowLayout(parentType: string): Record<string, unknown> | null {
  return _descriptors.get(parentType)?.childFlowLayout ?? null;
}

export function getChildStyle(parentType: string): Record<string, string | number> {
  const descriptor = _descriptors.get(parentType);
  if (!descriptor?.childStyle) return {};
  return descriptor.childStyle(parentType);
}

export function isChildResizable(parentType: string): boolean {
  const descriptor = _descriptors.get(parentType);
  if (!descriptor) return true;
  return descriptor.childResizable !== false;
}

export function resolveDescriptorContainerStyle(
  type: string,
  node: DescriptorNode,
): Record<string, string | number> | null {
  const descriptor = _descriptors.get(type);
  if (!descriptor?.containerStyle) return null;
  return descriptor.containerStyle(node);
}

export function getDisplayContent(
  type: string,
  node: DescriptorNode,
  resolvedProps?: Record<string, unknown>,
): string | null {
  const descriptor = _descriptors.get(type);
  if (!descriptor?.displayContent) return null;
  return descriptor.displayContent(node, resolvedProps ?? {});
}

export function getRenderKey(
  type: string,
  node: DescriptorNode,
  context?: Record<string, unknown>,
): string {
  const descriptor = _descriptors.get(type);
  if (!descriptor?.renderKey) return (node?.id as string | undefined) ?? "";
  return descriptor.renderKey(node, context ?? {});
}

export function isTableLikeType(_type: string): boolean {
  return false;
}

export function isNodeDesignerMovable(type: string): boolean {
  if (!type) return true;
  const descriptor = _descriptors.get(type);
  if (descriptor && descriptor.isMovable === false) return false;
  return !FIXED_LAYOUT_SHELL_SLOT_TYPES.has(type);
}

export function usesComponentWrapper(_type: string): boolean {
  return false;
}

export function usesLegacyFlexDirectionProps(_type: string): boolean {
  return false;
}

export function getRegionDesignerHint(type: string): string {
  return REGION_DESIGNER_HINTS[type] ?? "区域";
}

export function getDesignerNodeLayoutClasses(
  type: string,
  ctx: { tabPosition?: string; parentGutter?: number } = {},
): string[] {
  if (!type) return [];
  const out: string[] = [];
  if (type === "HorizontalLayout" || type === "VerticalLayout") {
    out.push("layout-container-visible");
  }
  if (type === "Tabs") {
    out.push("tabs-container");
    const position = ctx.tabPosition ?? "top";
    out.push(`tabs-pos-${position}`);
  }
  return out;
}

export function getPropsFilter(
  type: string,
  resolvedProps?: Record<string, unknown>,
): Record<string, unknown> {
  const descriptor = _descriptors.get(type);
  if (!descriptor?.propsFilter) return resolvedProps ?? {};
  return descriptor.propsFilter(resolvedProps ?? {});
}

export function getCustomRenderer(type: string): Component | null {
  return _descriptors.get(type)?.customRenderer ?? null;
}

export function getDefaultSize(type: string): { width: number; height: number } | null {
  const descriptorSize = _descriptors.get(type)?.defaultSize;
  if (
    descriptorSize &&
    typeof descriptorSize.width === "number" &&
    typeof descriptorSize.height === "number"
  ) {
    return descriptorSize;
  }
  return null;
}

export function getChildLayout(type: string): "flex" | "grid" | "free" | "none" | null {
  return _descriptors.get(type)?.childLayout ?? null;
}

export function isFlexContainer(type: string): boolean {
  return getChildLayout(type) === "flex";
}

export function isLayoutContainerType(type: string): boolean {
  const layout = getChildLayout(type);
  return layout === "flex" || layout === "grid" || layout === "free";
}

export function isLayoutType(type: string): boolean {
  return isLayoutContainerType(type) || type === "ElHeader" || type === "ElAside" || type === "ElMain" || type === "ElFooter";
}

export function canAcceptChildByDescriptor(
  parentType: string,
  childType: string,
  currentChildCount = 0,
): boolean {
  const descriptor = _descriptors.get(parentType);
  if (!descriptor) return false;

  if (typeof descriptor.maxChildren === "number" && currentChildCount >= descriptor.maxChildren) {
    return false;
  }

  const acceptChildren = descriptor.acceptChildren;
  if (acceptChildren === false) return false;
  if (acceptChildren === true) return true;
  if (Array.isArray(acceptChildren)) return acceptChildren.includes(childType);
  return true;
}

export function isRegionType(type: string): boolean {
  return _descriptors.get(type)?.isRegion === true;
}

export function getFlexDirection(type: string): "row" | "column" | null {
  return _descriptors.get(type)?.flexDirection ?? null;
}

export default {
  registerDescriptor,
  getDescriptor,
  hasDescriptor,
  getRenderTag,
  isContainerType,
  getChildPositioning,
  getChildFlowLayout,
  getChildStyle,
  isChildResizable,
  resolveDescriptorContainerStyle,
  getDisplayContent,
  getRenderKey,
  getPropsFilter,
  getCustomRenderer,
  getDefaultSize,
  getChildLayout,
  isFlexContainer,
  isLayoutContainerType,
  isLayoutType,
  canAcceptChildByDescriptor,
  isRegionType,
  getFlexDirection,
  isTableLikeType,
  isNodeDesignerMovable,
  usesComponentWrapper,
  usesLegacyFlexDirectionProps,
  getRegionDesignerHint,
  getDesignerNodeLayoutClasses,
};
