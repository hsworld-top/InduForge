/**
 * 节点样式计算 Composable
 *
 * 从 NodeRenderer 抽取的定位/布局样式逻辑，供递归渲染器复用。
 * 提供 resolveLayoutStyle、resolveContainerStyle、formatGridTemplate 等工具。
 *
 * @module ui/Canvas/composables/use-node-style
 */

import type { ComputedRef, CSSProperties, Ref } from "vue";
import type { ComponentNode } from "@/editor-core/document/types";
import { computed, onBeforeUnmount, onMounted, ref, watch, watchEffect } from "vue";
import {
  getDescriptor,
  isContainerType,
  resolveDescriptorContainerStyle,
} from "@/editor-core/descriptors/registry";
import { componentRegistry } from "@/editor-core";
import { normalizeStyleObject, resolveTextPropStyle } from "@/editor-core/utils/style-utils";
import { elevateStyleConfigPriority, replaceStyleConfigPlaceholders } from "../style-config-css";

type StyleValue = CSSProperties[keyof CSSProperties] | any;
type StyleMap = Record<string, StyleValue>;
type LooseRecord = Record<string, any>;

const EL_CONTAINER_REGION_PRESET_TOP_MAIN = "top-main";
const EL_CONTAINER_REGION_PRESET_ASIDE_FULL_HEIGHT = "aside-full-height";
const EL_CONTAINER_REGION_PRESET_ASIDE_BETWEEN = "aside-between";
const PANEL_CONTAINER_TYPES = new Set(["Tabs", "Collapse"]);
const AUTO_SIZE_PANEL_LAYOUT_TYPES = new Set(["HorizontalLayout", "VerticalLayout"]);
const EL_CONTAINER_SECTION_TYPES = new Set(["ElHeader", "ElAside", "ElMain", "ElFooter"]);

interface NodeStyleProps {
  readonly?: boolean;
  isRoot?: boolean;
}

interface CanvasPageLike {
  rootNodeId?: string | null;
}

interface NodeStyleHelpersDeps {
  doc: Ref<
    | {
        getNode?: (id: string) => ComponentNode | null | undefined;
        getParent?: (id: string) => ComponentNode | null | undefined;
      }
    | null
    | undefined
  >;
  currentPage: Ref<CanvasPageLike | null | undefined>;
  props?: NodeStyleProps;
}

/**
 * 解析区域布局预设
 * @param {unknown} value - 预设值
 * @returns {"top-main" | "aside-full-height" | "aside-between"}
 */
function resolveElContainerRegionPreset(
  value: unknown,
): "top-main" | "aside-full-height" | "aside-between" {
  if (
    value === EL_CONTAINER_REGION_PRESET_TOP_MAIN ||
    value === EL_CONTAINER_REGION_PRESET_ASIDE_FULL_HEIGHT ||
    value === EL_CONTAINER_REGION_PRESET_ASIDE_BETWEEN
  ) {
    return value;
  }
  return EL_CONTAINER_REGION_PRESET_ASIDE_BETWEEN;
}

/**
 * 是否明确的数值尺寸。
 * 面板内布局只有手动设置为数值或 px 时才固定高度，百分比/auto 仍按内容自适应。
 */
function isExplicitNumericSize(value: unknown): boolean {
  if (typeof value === "number") {
    return Number.isFinite(value);
  }
  if (typeof value !== "string") return false;
  const text = value.trim();
  return /^-?\d+(\.\d+)?(px)?$/i.test(text);
}

function isAutoSizeValue(value: unknown): boolean {
  return typeof value === "string" && value.trim().toLowerCase() === "auto";
}

function isDefaultPanelLayoutPadding(value: unknown): boolean {
  if (typeof value === "number") return value === 8;
  if (typeof value !== "string") return value === undefined || value === null;
  return value.trim().toLowerCase() === "8px";
}

function isDefaultPanelLayoutMinHeight(value: unknown): boolean {
  if (typeof value === "number") return value === 40;
  if (typeof value !== "string") return false;
  return value.trim().toLowerCase() === "40px";
}

/**
 * 折叠面板/选项卡内的水平、垂直布局默认跟随内容高度，避免旧数据中的 100% 高度撑出空白。
 */
function shouldAutoSizePanelLayout(
  currentNode: ComponentNode | null | undefined,
  parentNode: ComponentNode | null | undefined,
): boolean {
  return Boolean(
    currentNode?.type &&
    parentNode?.type &&
    AUTO_SIZE_PANEL_LAYOUT_TYPES.has(currentNode.type) &&
    PANEL_CONTAINER_TYPES.has(parentNode.type),
  );
}

function isElContainerSectionType(type: string | null | undefined): boolean {
  return Boolean(type && EL_CONTAINER_SECTION_TYPES.has(type));
}

function applyPanelLayoutAutoSize(
  style: StyleMap,
  customStyle: StyleMap,
  options: { editing: boolean; hasChildren?: boolean } = { editing: false },
): void {
  const hasFixedWidth = isExplicitNumericSize(customStyle.width);
  const hasFixedHeight = isExplicitNumericSize(customStyle.height);
  const hasFixedMinHeight = isExplicitNumericSize(customStyle.minHeight);
  const hasUserFixedMinHeight =
    hasFixedMinHeight && !isDefaultPanelLayoutMinHeight(customStyle.minHeight);
  if (!hasFixedWidth) {
    style.width = "100%";
    style.maxWidth = "100%";
  }
  style.boxSizing = "border-box";
  style.alignSelf = "stretch";
  style.flex = "0 0 auto";
  if (!hasUserFixedMinHeight) {
    style.minHeight = options.editing && !options.hasChildren ? "60px" : "0";
  }
  if (!hasFixedHeight) {
    style.height = "auto";
  }
  if (isDefaultPanelLayoutPadding(customStyle.padding)) {
    style.padding = "0";
  }
  if (!style.display) {
    style.display = "block";
  }
}

/**
 * 格式化 Grid 模板
 * @param {string | number} value - 模板配置
 * @returns {string}
 */
export function formatGridTemplate(value: unknown): string {
  if (typeof value === "number") {
    return `repeat(${value}, minmax(0, 1fr))`;
  }
  return typeof value === "string" ? value : "";
}

/**
 * 创建依赖 doc / currentPage / props 的样式解析器
 * @param {{ doc: import('vue').Ref, currentPage: import('vue').Ref, props: { readonly?: boolean } }} deps
 * @returns {{ resolveLayoutStyle: Function, resolveContainerStyle: Function, isRootCanvasContainer: Function }}
 */
export function createNodeStyleHelpers(deps: NodeStyleHelpersDeps) {
  const { doc, currentPage, props = {} } = deps;

  function getDoc() {
    return doc?.value ?? null;
  }

  function getCurrentPage() {
    return currentPage?.value ?? null;
  }

  /**
   * 是否根画布下的 ElContainer
   */
  function isRootCanvasContainer(currentNode: ComponentNode | null | undefined): boolean {
    if (!currentNode || currentNode.type !== "ElContainer") return false;
    const rootId = getCurrentPage()?.rootNodeId;
    if (!rootId) return false;
    const parentNode = getDoc()?.getParent?.(currentNode.id);
    return parentNode?.id === rootId;
  }

  /**
   * 将布局配置转换为样式（支持新旧两种架构）
   * @param {import('@/editor-core').ComponentNode} currentNode
   * @param {boolean} isRoot
   * @returns {Record<string, any>}
   */
  function resolveLayoutStyle(currentNode: ComponentNode, isRoot: boolean): StyleMap {
    if (isRoot) {
      return {
        position: "relative",
        width: "100%",
        height: "100%",
        boxSizing: "border-box",
      };
    }

    const style: StyleMap = {};
    const parentNode = getDoc()?.getParent?.(currentNode.id);
    if (isElContainerSectionType(parentNode?.type)) {
      const shouldFillRegion = isContainerType(currentNode.type);
      return {
        position: "relative",
        width: shouldFillRegion ? "100%" : "auto",
        height: shouldFillRegion ? "100%" : "auto",
        flexGrow: shouldFillRegion ? 1 : 0,
        flexShrink: shouldFillRegion ? 1 : 0,
        flexBasis: shouldFillRegion ? "0%" : "auto",
        alignSelf: shouldFillRegion ? "stretch" : "flex-start",
        justifySelf: shouldFillRegion ? "stretch" : "start",
      };
    }

    if (currentNode.positioning === "absolute" && currentNode.absolutePos) {
      const pos = currentNode.absolutePos;
      style.position = "absolute";
      style.left = `${pos.x ?? 0}px`;
      style.top = `${pos.y ?? 0}px`;
      if (pos.w !== undefined) style.width = `${pos.w}px`;
      if (pos.h !== undefined) style.height = `${pos.h}px`;
      if (pos.z !== undefined) style.zIndex = pos.z;
      return style;
    }

    if (currentNode.positioning === "flow") {
      if (currentNode.flowLayout) {
        const flow = currentNode.flowLayout as LooseRecord;
        const useExplicitGridPlacement = parentNode?.type !== "FormLayout";
        if (flow.grow !== undefined || flow.shrink !== undefined || flow.basis !== undefined) {
          style.flexGrow = flow.grow ?? 0;
          style.flexShrink = flow.shrink ?? 1;
          style.flexBasis = flow.basis ?? "auto";
          if (flow.alignSelf) style.alignSelf = flow.alignSelf;
        }
        if (useExplicitGridPlacement && flow.row !== undefined) {
          const rowSpan = flow.rowSpan || 1;
          style.gridRow = `${flow.row} / span ${rowSpan}`;
        }
        if (useExplicitGridPlacement && flow.col !== undefined) {
          const colSpan = flow.colSpan || 1;
          style.gridColumn = `${flow.col} / span ${colSpan}`;
        }
      }
      if (parentNode && getDescriptor(parentNode.type)?.childFlowLayout) {
        const parentDescriptor = getDescriptor(parentNode.type);
        const fl = parentDescriptor?.childFlowLayout as LooseRecord | null | undefined;
        const currentFlow = currentNode.flowLayout as LooseRecord | null | undefined;
        const customStyle = normalizeStyleObject(currentNode.style || {}) as StyleMap;
        const flexDirection = parentDescriptor?.flexDirection;
        const fixedMainSize =
          flexDirection === "row"
            ? customStyle.width
            : flexDirection === "column"
              ? customStyle.height
              : undefined;
        const fixedCrossSize =
          flexDirection === "row"
            ? customStyle.height
            : flexDirection === "column"
              ? customStyle.width
              : undefined;
        const hasFixedMainSize = isExplicitNumericSize(fixedMainSize);
        const hasFixedCrossSize = isExplicitNumericSize(fixedCrossSize);
        style.position = "relative";
        if (hasFixedMainSize) {
          style.flexGrow = 0;
          style.flexShrink = 0;
          style.flexBasis = fixedMainSize;
        } else {
          style.flexGrow = currentFlow?.grow ?? fl?.grow ?? 1;
          style.flexShrink = currentFlow?.shrink ?? fl?.shrink ?? 1;
          style.flexBasis = currentFlow?.basis ?? fl?.basis ?? "0%";
        }
        const configuredAlignSelf = currentFlow?.alignSelf ?? fl?.alignSelf;
        if (configuredAlignSelf) {
          style.alignSelf = configuredAlignSelf;
        } else if (hasFixedCrossSize) {
          style.alignSelf = "flex-start";
        } else {
          style.alignSelf = style.alignSelf ?? "stretch";
        }
        style.minWidth = style.minWidth ?? "0";
        style.minHeight = style.minHeight ?? "0";
        return style;
      }
      style.position = "relative";
      style.display = "block";
      if (parentNode?.type === "ElCol") {
        style.width = "100%";
        style.alignSelf = "stretch";
      }
      if (parentNode?.type === "ElContainer") {
        if (currentNode.type === "ElHeader") style.gridArea = "header";
        else if (currentNode.type === "ElAside") style.gridArea = "aside";
        else if (currentNode.type === "ElMain") style.gridArea = "main";
        else if (currentNode.type === "ElFooter") style.gridArea = "footer";
        style.width = "100%";
        style.height = "100%";
        style.alignSelf = "stretch";
        style.justifySelf = "stretch";
      } else if (isElContainerSectionType(parentNode?.type)) {
        const shouldFillRegion = isContainerType(currentNode.type);
        style.width = shouldFillRegion ? "100%" : "auto";
        style.height = shouldFillRegion ? "100%" : "auto";
        style.flexGrow = shouldFillRegion ? 1 : 0;
        style.flexShrink = shouldFillRegion ? 1 : 0;
        style.flexBasis = shouldFillRegion ? "0%" : "auto";
        style.alignSelf = shouldFillRegion ? "stretch" : "flex-start";
      }
      return style;
    }

    if (currentNode.layoutItem?.free) {
      const free = currentNode.layoutItem.free;
      if (free.mode === "abs" && free.abs) {
        const abs = free.abs;
        style.position = "absolute";
        style.left = `${abs.x ?? 0}px`;
        style.top = `${abs.y ?? 0}px`;
        if (abs.w !== undefined) style.width = `${abs.w}px`;
        if (abs.h !== undefined) style.height = `${abs.h}px`;
        if (abs.z !== undefined) style.zIndex = abs.z;
      }
      if (free.mode === "constraints" && free.constraints) {
        const constraints = free.constraints;
        style.position = "absolute";
        if (constraints.left !== undefined) style.left = `${constraints.left}px`;
        if (constraints.right !== undefined) style.right = `${constraints.right}px`;
        if (constraints.top !== undefined) style.top = `${constraints.top}px`;
        if (constraints.bottom !== undefined) style.bottom = `${constraints.bottom}px`;
        if (constraints.width !== undefined) style.width = `${constraints.width}px`;
        if (constraints.height !== undefined) style.height = `${constraints.height}px`;
      }
    }
    if (currentNode.layoutItem?.flex) {
      const flex = currentNode.layoutItem.flex;
      style.flexGrow = flex.grow ?? 0;
      style.flexShrink = flex.shrink ?? 1;
      style.flexBasis = flex.basis ?? "auto";
      if (flex.alignSelf) style.alignSelf = flex.alignSelf;
    }
    if (currentNode.layoutItem?.grid) {
      const grid = currentNode.layoutItem.grid;
      const parentNode = getDoc()?.getParent?.(currentNode.id);
      const useExplicitGridPlacement = parentNode?.type !== "FormLayout";
      if (useExplicitGridPlacement && grid.row !== undefined) {
        style.gridRow = `${grid.row} / span ${grid.rowSpan || 1}`;
      }
      if (useExplicitGridPlacement && grid.col !== undefined) {
        style.gridColumn = `${grid.col} / span ${grid.colSpan || 1}`;
      }
    }
    return style;
  }

  /**
   * 解析容器布局样式
   * @param {import('@/editor-core').ComponentNode} currentNode
   * @param {Record<string, any>} baseStyle
   * @returns {Record<string, any>}
   */
  function resolveContainerStyle(currentNode: ComponentNode, baseStyle: StyleMap): StyleMap {
    const style: StyleMap = {};
    const manifest = componentRegistry.get(currentNode.type);
    const descriptorContainerStyle = resolveDescriptorContainerStyle(
      currentNode.type,
      currentNode as any,
    );
    if (descriptorContainerStyle) {
      return { ...descriptorContainerStyle };
    }

    if (currentNode.type === "FlexContainer" || currentNode.type === "ResponsiveLayout") {
      style.display = "flex";
      style.flexDirection = currentNode.props?.direction || "row";
      style.flexWrap = currentNode.props?.wrap || "nowrap";
      style.justifyContent = currentNode.props?.justify || "flex-start";
      style.alignItems = currentNode.props?.align || "stretch";
      style.position = "relative";
      if (currentNode.props?.gap !== undefined) {
        style.gap = `${currentNode.props.gap}px`;
      }
    } else if (
      currentNode.type === "GridContainer" ||
      currentNode.type === "ColumnLayout1" ||
      currentNode.type === "ColumnLayout2" ||
      currentNode.type === "ColumnLayout4"
    ) {
      style.display = "grid";
      style.position = "relative";
      if (currentNode.props?.columns) {
        style.gridTemplateColumns = formatGridTemplate(currentNode.props.columns);
      }
      if (currentNode.props?.rows) {
        style.gridTemplateRows = formatGridTemplate(currentNode.props.rows);
      }
      if (currentNode.props?.gap !== undefined) {
        style.gap = currentNode.props.gap;
      }
    } else if (currentNode.type === "FreeContainer") {
      style.display = "flex";
      style.flexDirection = "column";
      style.flexWrap = "nowrap";
      style.position = "relative";
      if (currentNode.props?.overflow) {
        style.overflow = currentNode.props.overflow;
      }
    } else if (currentNode.type === "ElContainer") {
      const docModel = getDoc();
      const children = currentNode.children || [];
      let headerNode = null;
      let footerNode = null;
      let asideNode = null;
      let mainNode = null;
      for (const childId of children) {
        const childNode = docModel?.getNode?.(childId);
        if (!childNode) continue;
        if (childNode.type === "ElHeader" && !headerNode) headerNode = childNode;
        if (childNode.type === "ElFooter" && !footerNode) footerNode = childNode;
        if (childNode.type === "ElAside" && !asideNode) asideNode = childNode;
        if (childNode.type === "ElMain" && !mainNode) mainNode = childNode;
      }
      const hasHeader = Boolean(headerNode);
      const hasFooter = Boolean(footerNode);
      const hasAside = Boolean(asideNode);
      const hasMain = Boolean(mainNode);
      const containerProps = (currentNode.props || {}) as LooseRecord;
      const headerHeight = containerProps.headerHeight || headerNode?.props?.height || "60px";
      const footerHeight = containerProps.footerHeight || footerNode?.props?.height || "60px";
      const asideWidth = containerProps.asideWidth || asideNode?.props?.width || "200px";
      const regionPreset = resolveElContainerRegionPreset(containerProps.regionPreset);

      style.display = "grid";
      style.position = "relative";

      const rows: string[] = [];
      const areas: string[] = [];

      if (regionPreset === EL_CONTAINER_REGION_PRESET_TOP_MAIN) {
        style.gridTemplateColumns = "1fr";
        if (hasHeader) {
          rows.push(headerHeight);
          areas.push('"header"');
        }
        if (hasMain) {
          rows.push("1fr");
          areas.push('"main"');
        } else if (hasAside) {
          rows.push("1fr");
          areas.push('"aside"');
        }
        if (hasFooter) {
          rows.push(footerHeight);
          areas.push('"footer"');
        }
      } else if (regionPreset === EL_CONTAINER_REGION_PRESET_ASIDE_FULL_HEIGHT) {
        style.gridTemplateColumns = hasAside ? `${asideWidth} 1fr` : "1fr";
        if (hasAside) {
          if (hasHeader) {
            rows.push(headerHeight);
            areas.push('"aside header"');
          }
          if (hasMain) {
            rows.push("1fr");
            areas.push('"aside main"');
          }
          if (hasFooter) {
            rows.push(footerHeight);
            areas.push('"aside footer"');
          }
          if (!rows.length) {
            rows.push("1fr");
            areas.push('"aside main"');
          }
        } else {
          if (hasHeader) {
            rows.push(headerHeight);
            areas.push('"header"');
          }
          if (hasMain) {
            rows.push("1fr");
            areas.push('"main"');
          }
          if (hasFooter) {
            rows.push(footerHeight);
            areas.push('"footer"');
          }
        }
      } else {
        const hasTwoCols = hasAside && hasMain;
        style.gridTemplateColumns = hasTwoCols ? `${asideWidth} 1fr` : "1fr";
        if (hasHeader) {
          rows.push(headerHeight);
          areas.push(hasTwoCols ? '"header header"' : '"header"');
        }
        if (hasAside || hasMain) {
          rows.push("1fr");
          if (hasTwoCols) areas.push('"aside main"');
          else if (hasAside) areas.push('"aside"');
          else areas.push('"main"');
        }
        if (hasFooter) {
          rows.push(footerHeight);
          areas.push(hasTwoCols ? '"footer footer"' : '"footer"');
        }
      }

      if (rows.length === 0) {
        rows.push("1fr");
        areas.push('"main"');
      }
      style.gridTemplateRows = rows.join(" ");
      style.gridTemplateAreas = areas.join(" ");
    } else if (currentNode.type === "Tabs") {
      style.display = "flex";
      style.flexDirection = "column";
      style.position = "relative";
      style.overflow = "hidden";
    } else if (currentNode.type === "ElHeader") {
      const parentNode = getDoc()?.getParent?.(currentNode.id);
      const headerHeight =
        parentNode?.type === "ElContainer"
          ? parentNode.props?.headerHeight
          : currentNode.props?.height;
      style.display = "flex";
      style.flexDirection = "column";
      style.alignItems = "stretch";
      style.width = "100%";
      style.height = headerHeight || "60px";
      style.padding = "0";
      style.overflow = "hidden";
      style.position = "relative";
    } else if (currentNode.type === "ElFooter") {
      const parentNode = getDoc()?.getParent?.(currentNode.id);
      const footerHeight =
        parentNode?.type === "ElContainer"
          ? parentNode.props?.footerHeight
          : currentNode.props?.height;
      style.display = "flex";
      style.flexDirection = "column";
      style.alignItems = "stretch";
      style.width = "100%";
      style.height = footerHeight || "60px";
      style.padding = "0";
      style.overflow = "hidden";
      style.position = "relative";
    } else if (currentNode.type === "ElAside") {
      const parentNode = getDoc()?.getParent?.(currentNode.id);
      const asideWidth =
        parentNode?.type === "ElContainer"
          ? parentNode.props?.asideWidth
          : currentNode.props?.width;
      style.display = "flex";
      style.flexDirection = "column";
      style.alignItems = "stretch";
      style.width = asideWidth || "200px";
      style.height = "100%";
      style.padding = "0";
      style.overflow = "hidden";
      style.position = "relative";
    } else if (currentNode.type === "ElMain") {
      style.display = "flex";
      style.flexDirection = "column";
      style.alignItems = "stretch";
      style.width = "100%";
      style.height = "100%";
      style.padding = "0";
      style.overflow = "hidden";
      style.position = "relative";
    } else if (currentNode.type === "ElLayoutRow") {
      const gutter = Math.max(0, Number(currentNode.props?.gutter) || 0);
      const halfGutter = gutter ? gutter / 2 : 0;
      const parentNode = getDoc()?.getParent?.(currentNode.id);
      const rawHeight = currentNode.style?.height;
      const normalizedHeight =
        rawHeight === undefined || rawHeight === null ? "" : String(rawHeight).trim();
      const hasFixedHeight = normalizedHeight !== "" && normalizedHeight !== "auto";
      style.display = "flex";
      style.flexWrap = "wrap";
      style.alignItems = "stretch";
      if (parentNode?.type === "ElLayout") {
        if (hasFixedHeight) {
          style.flexGrow = 0;
          style.flexShrink = 0;
        } else {
          style.flexGrow = 1;
          style.flexShrink = 1;
          style.flexBasis = "0%";
        }
      }
      if (halfGutter) {
        style.marginLeft = `${-halfGutter}px`;
        style.marginRight = `${-halfGutter}px`;
      }
      style.width = "100%";
      style.position = "relative";
      style.boxSizing = "border-box";
    } else if (currentNode.type === "ElCol") {
      const parentNode = getDoc()?.getParent?.(currentNode.id);
      const gutter = Math.max(0, Number(parentNode?.props?.gutter) || 0);
      const halfGutter = gutter ? gutter / 2 : 0;
      if (halfGutter) {
        style.paddingLeft = `${halfGutter}px`;
        style.paddingRight = `${halfGutter}px`;
      }
      style.display = "flex";
      style.flexDirection = "column";
      style.alignItems = "stretch";
      style.width = "100%";
      style.position = "relative";
      style.boxSizing = "border-box";
    } else if (currentNode.type === "ElLayout") {
      const rowGap = props.readonly ? 0 : Math.max(0, Number(currentNode.props?.gutter) || 0);
      const rawPadding = currentNode.props?.padding;
      const layoutPadding = props.readonly
        ? "0px"
        : typeof rawPadding === "number" && Number.isFinite(rawPadding)
          ? `${rawPadding}px`
          : rawPadding !== undefined
            ? String(rawPadding)
            : "0px";
      style.display = "flex";
      style.flexDirection = "column";
      style.alignItems = "stretch";
      style.rowGap = `${rowGap}px`;
      style.padding = layoutPadding;
      style.width = "100%";
      style.height = "100%";
      style.overflow = "hidden";
    }

    void manifest;
    const isContainer = isContainerType(currentNode.type);
    if (isContainer && !baseStyle?.position) {
      style.position = style.position ?? "relative";
    }
    if (isContainer && currentNode.positioning === "absolute") {
      if (style.width === undefined) style.width = "100%";
      if (style.height === undefined) style.height = "100%";
    }
    return style;
  }

  /**
   * 创建 contentStyle computed
   * @param {import('vue').Ref} nodeRef - 节点 ref
   * @param {import('vue').Ref} docRef - 文档 ref
   * @param {import('vue').Ref} docVersionRef - 文档版本 ref
   * @param {import('vue').Ref} resolvedNodePropsRef - 已解析的节点属性 ref
   * @param {import('vue').Ref} isContainerRef - 是否为容器 ref
   * @param {import('vue').Ref} isMovableRef - 是否可移动 ref
   * @param {import('vue').Ref} layoutStyleRef - 布局样式 ref
   * @param {{ readonly?: boolean, isRoot?: boolean }} props - 组件 props
   * @returns {import('vue').ComputedRef<Record<string, any>>}
   */
  function createContentStyle(
    nodeRef: Ref<ComponentNode | null | undefined>,
    docRef: Ref<NodeStyleHelpersDeps["doc"]["value"]>,
    docVersionRef: Ref<number>,
    resolvedNodePropsRef: Ref<Record<string, unknown>>,
    isContainerRef: Ref<boolean>,
    isMovableRef: Ref<boolean>,
    layoutStyleRef: Ref<StyleMap>,
    props: NodeStyleProps = {},
  ): ComputedRef<StyleMap> {
    return computed(() => {
      void docVersionRef.value;
      if (!nodeRef.value) return {};
      const containerStyle = resolveContainerStyle(nodeRef.value, {});
      const customStyle = normalizeStyleObject(nodeRef.value.style || {}) as StyleMap;
      const textStyle =
        nodeRef.value.type === "Text"
          ? (normalizeStyleObject(
              resolveTextPropStyle(resolvedNodePropsRef.value || {}),
            ) as StyleMap)
          : {};
      const parentNode = docRef.value?.getParent?.(nodeRef.value.id);
      const style: StyleMap = {
        ...containerStyle,
        ...textStyle,
        ...customStyle,
      };
      if (!style.overflow && !isContainerRef.value) {
        style.overflow = "hidden";
      }
      if (nodeRef.value.type === "ElLayoutRow") {
        style.overflow = "visible";
      }
      if (!props.readonly) {
        if (nodeRef.value.type === "ElLayout") {
          style.overflow = "hidden";
        } else if (nodeRef.value.type === "ElLayoutRow" || nodeRef.value.type === "ElCol") {
          style.overflow = "visible";
        }
      }
      if (nodeRef.value.type === "ElLayout") {
        if (props.readonly) {
          style.padding = "0";
          style.gap = "0";
          style.columnGap = "0";
          style.rowGap = "0";
        } else {
          const rawPadding = nodeRef.value.props?.padding;
          const paddingValue =
            typeof rawPadding === "number" && Number.isFinite(rawPadding)
              ? `${rawPadding}px`
              : rawPadding !== undefined
                ? String(rawPadding)
                : "0px";
          style.padding = paddingValue;
        }
        delete style.minHeight;
      }
      if (isElContainerSectionType(parentNode?.type)) {
        style.overflow = "hidden";
      }
      if (nodeRef.value.type === "ElCol" && parentNode?.type === "ElLayoutRow") {
        style.height = "100%";
      }
      if (parentNode?.type === "ElCol") {
        style.width = "100%";
        style.height = "100%";
        style.minHeight = "100%";
        style.alignSelf = "stretch";
        if (!style.display) {
          style.display = "block";
        }
      }
      const isAutoPanelLayout = shouldAutoSizePanelLayout(nodeRef.value, parentNode);
      if (parentNode?.type === "Tabs") {
        if (isAutoPanelLayout) {
          applyPanelLayoutAutoSize(style, customStyle, {
            editing: !props.readonly,
            hasChildren: Boolean(nodeRef.value.children?.length),
          });
        } else {
          style.width = "100%";
          style.height = "100%";
          style.minHeight = "100%";
          style.alignSelf = "stretch";
          style.flex = "1 1 auto";
          if (!style.display) {
            style.display = "block";
          }
        }
      }
      if (parentNode?.type === "Collapse" && isAutoPanelLayout) {
        applyPanelLayoutAutoSize(style, customStyle, {
          editing: !props.readonly,
          hasChildren: Boolean(nodeRef.value.children?.length),
        });
      }
      if (parentNode?.type === "ElCol" && isMovableRef.value) {
        style.width = "100%";
        if (isContainerRef.value) {
          style.height = "100%";
        } else if (!style.height) {
          style.height = "auto";
        }
        const rowParent = docRef.value?.getParent?.(parentNode.id);
        const rowHeight = rowParent?.type === "ElLayoutRow" ? rowParent.style?.height : undefined;
        const normalizedRowHeight =
          rowHeight === undefined || rowHeight === null ? "" : String(rowHeight).trim();
        const hasFixedRowHeight = normalizedRowHeight !== "" && normalizedRowHeight !== "auto";
        if (hasFixedRowHeight && (!style.height || style.height === "auto")) {
          style.height = "100%";
        }
      }
      if (parentNode?.type === "ElCol") {
        const rowParent = docRef.value?.getParent?.(parentNode.id);
        const rowHeight = rowParent?.type === "ElLayoutRow" ? rowParent.style?.height : undefined;
        const normalizedRowHeight =
          rowHeight === undefined || rowHeight === null ? "" : String(rowHeight).trim();
        const hasFixedRowHeight = normalizedRowHeight !== "" && normalizedRowHeight !== "auto";
        if (hasFixedRowHeight) {
          style.height = "100%";
        }
      }
      if (
        nodeRef.value.type === "ElContainer" ||
        nodeRef.value.type === "ElHeader" ||
        nodeRef.value.type === "ElAside" ||
        nodeRef.value.type === "ElMain" ||
        nodeRef.value.type === "ElFooter"
      ) {
        style.padding = "0";
      }
      if (
        nodeRef.value.type === "ElHeader" ||
        nodeRef.value.type === "ElAside" ||
        nodeRef.value.type === "ElMain" ||
        nodeRef.value.type === "ElFooter"
      ) {
        style.overflow = "hidden";
      }
      if (parentNode?.type === "ElContainer") {
        if (nodeRef.value.type === "ElHeader") {
          style.width = "100%";
          style.height = parentNode.props?.headerHeight || "60px";
        } else if (nodeRef.value.type === "ElFooter") {
          style.width = "100%";
          style.height = parentNode.props?.footerHeight || "60px";
        } else if (nodeRef.value.type === "ElAside") {
          style.width = parentNode.props?.asideWidth || "200px";
          style.height = "100%";
        }
      }
      if (!style.width && isElContainerSectionType(parentNode?.type)) {
        style.width = isContainerRef.value ? "100%" : "auto";
      }
      if (!style.height && isElContainerSectionType(parentNode?.type)) {
        style.height = isContainerRef.value ? "100%" : "auto";
      }
      // 绝对定位时外层框由 absolutePos 定尺寸（选择框），内层内容需填满框体，避免「框比按钮大」的空白
      if (props.isRoot || layoutStyleRef.value.position === "absolute") {
        style.width = "100%";
        style.height = "100%";
        if (!isContainerRef.value) {
          style.display = "block";
          style.boxSizing = "border-box";
        }
      }
      if (parentNode && nodeRef.value.positioning === "flow") {
        const parentDescriptor = getDescriptor(parentNode.type);
        if (parentDescriptor?.childStyle) {
          const childStyle = (parentDescriptor.childStyle(parentNode.type) || {}) as LooseRecord;
          if (childStyle.width && (!style.width || isAutoSizeValue(style.width))) {
            style.width = childStyle.width;
          }
          if (childStyle.height && (!style.height || isAutoSizeValue(style.height))) {
            style.height = childStyle.height;
          }
          if (childStyle.minWidth && !style.minWidth) {
            style.minWidth = childStyle.minWidth;
          }
          if (childStyle.minHeight && !style.minHeight) {
            style.minHeight = childStyle.minHeight;
          }
        }
      }
      return style;
    });
  }

  /**
   * 创建 wrapperComponentStyle computed
   * @param {import('vue').Ref} nodeRef - 节点 ref
   * @param {import('vue').Ref} docRef - 文档 ref
   * @param {import('vue').Ref} useComponentWrapperRef - 是否使用组件包装器 ref
   * @param {import('vue').Ref} wrapperStyleRef - 包装器样式 ref
   * @param {import('vue').Ref} contentStyleRef - 内容样式 ref
   * @param {import('vue').Ref} resolvedPropsRef - 已解析的属性 ref
   * @param {{ readonly?: boolean }} props - 组件 props
   * @returns {import('vue').ComputedRef<Record<string, any>>}
   */
  function createWrapperComponentStyle(
    nodeRef: Ref<ComponentNode | null | undefined>,
    docRef: Ref<NodeStyleHelpersDeps["doc"]["value"]>,
    useComponentWrapperRef: Ref<boolean>,
    wrapperStyleRef: Ref<StyleMap>,
    contentStyleRef: Ref<StyleMap>,
    resolvedPropsRef: Ref<Record<string, unknown>>,
    props: NodeStyleProps = {},
  ): ComputedRef<StyleMap> {
    return computed(() => {
      if (!useComponentWrapperRef.value) return {};
      const style: StyleMap = { ...wrapperStyleRef.value, ...contentStyleRef.value };
      const type = nodeRef.value?.type;
      const nodeId = nodeRef.value?.id;
      const parentNode = nodeId ? docRef.value?.getParent?.(nodeId) : null;
      const customStyle = normalizeStyleObject(nodeRef.value?.style || {}) as StyleMap;
      const hasCustomHeight = Boolean(customStyle.height);
      const isEditingMode = !props.readonly;
      const layoutSelectInset = isEditingMode ? 6 : 0;
      const rowGuidePaddingX = isEditingMode ? 6 : 0;
      const rowGuidePaddingY = isEditingMode ? 8 : 0;
      const colContentInsetX = isEditingMode ? 10 : 0;
      const colContentInsetY = isEditingMode ? 12 : 0;
      if (type === "ElLayoutRow" || type === "ElCol") {
        delete style.display;
        style["--layout-select-inset"] = `${layoutSelectInset}px`;
      }
      if (type === "ElLayoutRow" && parentNode?.type === "ElLayout") {
        style.flex = hasCustomHeight ? "0 0 auto" : "1 1 0";
        style.width = "100%";
        style.minHeight = "0";
        style.maxHeight = hasCustomHeight ? "none" : "100%";
        style.alignItems = "stretch";
        style.overflow = "visible";
        style.columnGap = "0";
        style.gap = "0";
        style.padding = "0";
        style.paddingTop = "0";
        style.paddingRight = "0";
        style.paddingBottom = "0";
        style.paddingLeft = "0";
      }
      if (type === "ElLayoutRow") {
        const gutter = props.readonly ? 0 : Number(nodeRef.value?.props?.gutter) || 0;
        const columns = Math.max(1, Math.min(24, Number(nodeRef.value?.props?.columns) || 1));
        style["--row-gutter"] = `${Math.max(0, gutter)}px`;
        style["--row-columns"] = String(columns);
        style.paddingLeft = "0";
        style.paddingRight = "0";
        if (!gutter) {
          style["--el-row-gutter"] = "0px";
          style.paddingTop = "0";
          style.paddingBottom = "0";
          style.marginLeft = "0";
          style.marginRight = "0";
          style.columnGap = "0px";
          style.gap = "0px";
          style.margin = "0";
          style.padding = "0";
        }
        if (gutter > 0) {
          style.marginLeft = "0";
          style.marginRight = "0";
        }
        style.boxSizing = "border-box";
        style.flexWrap = "nowrap";
        style.overflow = "visible";
        if (isEditingMode) {
          style.paddingTop = `${rowGuidePaddingY}px`;
          style.paddingBottom = `${rowGuidePaddingY}px`;
          style.paddingLeft = `${rowGuidePaddingX}px`;
          style.paddingRight = `${rowGuidePaddingX}px`;
        }
      }
      if (type === "ElLayoutRow" && parentNode?.type === "ElLayout" && isEditingMode) {
        style.marginTop = "6px";
        style.marginBottom = "6px";
      }
      if (type === "ElCol" && parentNode?.type === "ElLayoutRow") {
        delete style.flexGrow;
        delete style.flexShrink;
        delete style.flexBasis;
        style.alignSelf = "stretch";
        style.boxSizing = "border-box";
        style.maxHeight = "100%";
        style.overflow = "visible";
        style.minHeight = "0";
        style.padding = "0";
        style.height = "100%";
        const rowHeight =
          parentNode?.style?.height !== undefined ? parentNode.style.height : undefined;
        const normalizedRowHeight =
          rowHeight === undefined || rowHeight === null ? "" : String(rowHeight).trim();
        const hasFixedRowHeight = normalizedRowHeight !== "" && normalizedRowHeight !== "auto";
        if (hasFixedRowHeight) {
          style.height = "100%";
        }
        const colProps = (resolvedPropsRef.value || nodeRef.value?.props || {}) as LooseRecord;
        const gutter = props.readonly ? 0 : Number(parentNode.props?.gutter) || 0;
        if (!gutter) {
          style.paddingLeft = `${colContentInsetX}px`;
          style.paddingRight = `${colContentInsetX}px`;
          style["--col-gutter-x"] = "0px";
          style.paddingTop = `${colContentInsetY}px`;
          style.paddingBottom = `${colContentInsetY}px`;
          style.marginLeft = "0";
          style.marginRight = "0";
          style.margin = "0";
        } else {
          const halfGutter = "calc(var(--row-gutter) / 2)";
          const halfCol = "calc(100% / var(--row-columns) / 2)";
          const clampedHalf = `min(${halfGutter}, ${halfCol})`;
          style.paddingLeft =
            colContentInsetX > 0 ? `calc(${clampedHalf} + ${colContentInsetX}px)` : clampedHalf;
          style.paddingRight =
            colContentInsetX > 0 ? `calc(${clampedHalf} + ${colContentInsetX}px)` : clampedHalf;
          style["--col-gutter-x"] = clampedHalf;
          style.paddingTop = `${colContentInsetY}px`;
          style.paddingBottom = `${colContentInsetY}px`;
        }
        const offset = Math.max(0, Math.min(24, Number(colProps?.offset) || 0));
        if (offset > 0) {
          style.marginLeft = `${(offset / 24) * 100}%`;
        }
        const span = Math.max(1, Math.min(24, Number(colProps?.span) || 24));
        const cols = (parentNode?.children || []).filter((childId) => {
          const childNode = docRef.value?.getNode?.(childId);
          return childNode?.type === "ElCol";
        });
        const colCount = Math.max(1, cols.length);
        const spans = cols.map((colId) => {
          const colNode = docRef.value?.getNode?.(colId);
          const colSpan = Number(colNode?.props?.span);
          if (Number.isFinite(colSpan) && colSpan > 0) {
            return Math.max(1, Math.min(24, colSpan));
          }
          return Math.max(1, Math.floor(24 / colCount));
        });
        const totalSpan = Math.max(
          1,
          spans.reduce((sum, value) => sum + value, 0),
        );
        const percent = `${(span / totalSpan) * 100}%`;
        style.flex = `0 0 ${percent}`;
        style.maxWidth = percent;
        style.width = percent;
        const push = Math.max(0, Math.min(24, Number(colProps?.push) || 0));
        const pull = Math.max(0, Math.min(24, Number(colProps?.pull) || 0));
        const shift = push - pull;
        if (shift !== 0) {
          style.position = "relative";
          style.left = `${(shift / 24) * 100}%`;
        }
      }
      return style;
    });
  }

  /**
   * 创建样式注入生命周期管理
   * @param {import('vue').Ref} nodeRef - 节点 ref
   * @param {import('vue').ComputedRef<string>} styleConfigCssRef - 样式配置 CSS ref
   * @returns {{ styleElementRef: import('vue').Ref<HTMLStyleElement | null> }}
   */
  function createStyleElementSync(
    nodeRef: Ref<ComponentNode | null | undefined>,
    styleConfigCssRef: ComputedRef<string>,
  ) {
    const styleElementRef = ref<HTMLStyleElement | null>(null);

    const syncStyleElement = () => {
      if (typeof document === "undefined") return;
      const nodeId = nodeRef.value?.id;
      const cssText = elevateStyleConfigPriority(
        replaceStyleConfigPlaceholders(styleConfigCssRef.value, {
          nodeId: nodeId || null,
        }),
      );

      if (!nodeId || !cssText) {
        if (styleElementRef.value?.parentNode) {
          styleElementRef.value.parentNode.removeChild(styleElementRef.value);
        }
        styleElementRef.value = null;
        return;
      }

      let styleEl = styleElementRef.value;
      if (!styleEl || styleEl.getAttribute("data-style-node") !== nodeId) {
        if (styleEl?.parentNode) {
          styleEl.parentNode.removeChild(styleEl);
        }
        styleEl = document.querySelector<HTMLStyleElement>(`style[data-style-node="${nodeId}"]`);
        if (!styleEl) {
          styleEl = document.createElement("style");
          styleEl.setAttribute("data-style-node", nodeId);
          document.head.appendChild(styleEl);
        }
        styleElementRef.value = styleEl;
      }

      if (styleEl.textContent !== cssText) {
        styleEl.textContent = cssText;
      }
    };

    watch(
      [() => nodeRef.value?.id, styleConfigCssRef],
      () => {
        syncStyleElement();
      },
      { immediate: true },
    );

    watchEffect(() => {
      void styleConfigCssRef.value;
      syncStyleElement();
    });

    onMounted(() => {
      syncStyleElement();
    });

    onBeforeUnmount(() => {
      if (styleElementRef.value?.parentNode) {
        styleElementRef.value.parentNode.removeChild(styleElementRef.value);
      }
      styleElementRef.value = null;
    });

    return { styleElementRef };
  }

  /**
   * 创建 wrapperStyle computed
   * @param {import('vue').Ref} nodeRef - 节点 ref
   * @param {import('vue').Ref} docRef - 文档 ref
   * @param {import('vue').Ref} layoutStyleRef - 布局样式 ref
   * @param {import('vue').Ref} isMovableRef - 是否可移动 ref
   * @param {import('vue').Ref} isContainerRef - 是否为容器 ref
   * @returns {import('vue').ComputedRef<Record<string, any>>}
   */
  function createWrapperStyle(
    nodeRef: Ref<ComponentNode | null | undefined>,
    docRef: Ref<NodeStyleHelpersDeps["doc"]["value"]>,
    layoutStyleRef: Ref<StyleMap>,
    isMovableRef: Ref<boolean>,
    isContainerRef: Ref<boolean>,
  ): ComputedRef<StyleMap> {
    return computed(() => {
      if (!nodeRef.value) return {};
      const style: StyleMap = { ...layoutStyleRef.value };
      const customStyle = normalizeStyleObject(nodeRef.value.style || {}) as StyleMap;
      if (customStyle.width && !style.width) {
        style.width = customStyle.width;
      }
      if (customStyle.height && !style.height) {
        style.height = customStyle.height;
      }
      // 图层操作写入 style.zIndex，必须作用在外层节点上，避免流式布局因调整 children 顺序而改变位置。
      if (customStyle.zIndex !== undefined && customStyle.zIndex !== null) {
        style.zIndex = customStyle.zIndex;
      }
      const parentNode = docRef.value?.getParent?.(nodeRef.value.id);
      const hasCustomWidth = Boolean(customStyle.width);
      const hasCustomHeight = Boolean(customStyle.height);
      if (nodeRef.value.type === "ElCol" && parentNode?.type === "ElLayoutRow" && !style.height) {
        style.height = "auto";
      }
      if (parentNode?.type === "ElCol" && isMovableRef.value) {
        style.width = "100%";
        style.height = "100%";
        if (style.position === "absolute") {
          style.left = "0";
          style.top = "0";
        }
      }
      if (parentNode?.type === "Tabs") {
        const wasAbsolute = style.position === "absolute";
        const shouldStretchPanelChild = Boolean(isContainerRef.value);
        style.position = "relative";
        style.width = hasCustomWidth ? style.width : shouldStretchPanelChild ? "100%" : "auto";
        style.height = hasCustomHeight ? style.height : "auto";
        style.flex = "0 0 auto";
        if (wasAbsolute) {
          delete style.left;
          delete style.top;
        }
      }
      if (isContainerRef.value && isMovableRef.value) {
        if (!hasCustomWidth && !style.width) style.width = "100%";
        if (!hasCustomHeight && !style.height) style.height = "100%";
      }
      if (shouldAutoSizePanelLayout(nodeRef.value, parentNode)) {
        const hasUserFixedMinHeight =
          isExplicitNumericSize(customStyle.minHeight) &&
          !isDefaultPanelLayoutMinHeight(customStyle.minHeight);
        if (!hasCustomWidth) {
          style.width = "100%";
        }
        if (!hasCustomHeight) {
          style.height = "auto";
        }
        if (!hasUserFixedMinHeight) {
          style.minHeight =
            !props.readonly && !nodeRef.value.children?.length ? "60px" : "0";
        }
        style.flex = "0 0 auto";
      }
      return style;
    });
  }

  return {
    resolveLayoutStyle,
    resolveContainerStyle,
    formatGridTemplate,
    isRootCanvasContainer,
    createContentStyle,
    createWrapperComponentStyle,
    createStyleElementSync,
    createWrapperStyle,
  };
}

export default { formatGridTemplate, createNodeStyleHelpers };
