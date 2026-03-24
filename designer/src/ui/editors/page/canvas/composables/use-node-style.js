/**
 * 节点样式计算 Composable
 *
 * 从 NodeRenderer 抽取的定位/布局样式逻辑，供递归渲染器复用。
 * 提供 resolveLayoutStyle、resolveContainerStyle、formatGridTemplate 等工具。
 *
 * @module ui/Canvas/composables/use-node-style
 */

import { computed, ref, watch, watchEffect, onMounted, onBeforeUnmount } from "vue";
import { getDescriptor, isContainerType } from "@/components/descriptors/registry.ts";
import { resolveDescriptorContainerStyle } from "@/components/descriptors/registry.ts";
import { componentRegistry } from "@/editor-core";
import {
  normalizeStyleObject,
  resolveTextPropStyle,
} from "@/editor-core/utils/style-utils";

/**
 * 格式化 Grid 模板
 * @param {string | number} value - 模板配置
 * @returns {string}
 */
export function formatGridTemplate(value) {
  if (typeof value === "number") {
    return `repeat(${value}, minmax(0, 1fr))`;
  }
  return value;
}

/**
 * 创建依赖 doc / currentPage / props 的样式解析器
 * @param {{ doc: import('vue').Ref, currentPage: import('vue').Ref, props: { readonly?: boolean } }} deps
 * @returns {{ resolveLayoutStyle: Function, resolveContainerStyle: Function, isRootCanvasContainer: Function }}
 */
export function createNodeStyleHelpers(deps) {
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
  function isRootCanvasContainer(currentNode) {
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
  function resolveLayoutStyle(currentNode, isRoot) {
    if (isRoot) {
      return {
        position: "relative",
        width: "100%",
        height: "100%",
        boxSizing: "border-box",
      };
    }

    const style = {};
    const parentNode = getDoc()?.getParent?.(currentNode.id);
    if (
      parentNode?.type === "ElHeader" ||
      parentNode?.type === "ElAside" ||
      parentNode?.type === "ElMain" ||
      parentNode?.type === "ElFooter"
    ) {
      return {
        position: "relative",
        width: "100%",
        height: "100%",
        flexGrow: 1,
        flexShrink: 1,
        alignSelf: "stretch",
        justifySelf: "stretch",
      };
    }

    if (isRootCanvasContainer(currentNode)) {
      const zIndex =
        currentNode.absolutePos?.z ?? currentNode.layoutItem?.free?.abs?.z;
      return {
        position: "absolute",
        left: 0,
        top: 0,
        width: "100%",
        height: "100%",
        boxSizing: "border-box",
        ...(zIndex !== undefined ? { zIndex } : {}),
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
        const flow = currentNode.flowLayout;
        if (
          flow.grow !== undefined ||
          flow.shrink !== undefined ||
          flow.basis !== undefined
        ) {
          style.flexGrow = flow.grow ?? 0;
          style.flexShrink = flow.shrink ?? 1;
          style.flexBasis = flow.basis ?? "auto";
          if (flow.alignSelf) style.alignSelf = flow.alignSelf;
        }
        if (flow.row !== undefined) {
          const rowSpan = flow.rowSpan || 1;
          style.gridRow = `${flow.row} / span ${rowSpan}`;
        }
        if (flow.col !== undefined) {
          const colSpan = flow.colSpan || 1;
          style.gridColumn = `${flow.col} / span ${colSpan}`;
        }
      }
      if (parentNode && getDescriptor(parentNode.type)?.childFlowLayout) {
        const parentDescriptor = getDescriptor(parentNode.type);
        const fl = parentDescriptor.childFlowLayout;
        style.position = "relative";
        style.flexGrow = currentNode.flowLayout?.grow ?? fl.grow ?? 1;
        style.flexShrink = currentNode.flowLayout?.shrink ?? fl.shrink ?? 1;
        style.flexBasis = currentNode.flowLayout?.basis ?? fl.basis ?? "0%";
        style.alignSelf = style.alignSelf ?? "stretch";
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
      } else if (
        parentNode?.type === "ElHeader" ||
        parentNode?.type === "ElAside" ||
        parentNode?.type === "ElMain" ||
        parentNode?.type === "ElFooter"
      ) {
        style.width = "100%";
        style.height = "100%";
        style.flexGrow = 1;
        style.flexShrink = 1;
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
      if (grid.row !== undefined) {
        style.gridRow = `${grid.row} / span ${grid.rowSpan || 1}`;
      }
      if (grid.col !== undefined) {
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
  function resolveContainerStyle(currentNode, baseStyle) {
    const style = {};
    const manifest = componentRegistry.get(currentNode.type);
    const descriptorContainerStyle = resolveDescriptorContainerStyle(
      currentNode.type,
      currentNode,
    );
    if (descriptorContainerStyle) {
      return { ...descriptorContainerStyle };
    }

    if (
      currentNode.type === "FlexContainer" ||
      currentNode.type === "ResponsiveLayout"
    ) {
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
      const hasBody = hasAside || hasMain;
      const hasTwoCols = hasAside && hasMain;
      const containerProps = currentNode.props || {};
      const headerHeight =
        containerProps.headerHeight || headerNode?.props?.height || "60px";
      const footerHeight =
        containerProps.footerHeight || footerNode?.props?.height || "60px";
      const asideWidth =
        containerProps.asideWidth || asideNode?.props?.width || "200px";

      style.display = "grid";
      style.position = "relative";
      style.gridTemplateColumns = hasTwoCols ? `${asideWidth} 1fr` : "1fr";
      const rows = [];
      const areas = [];
      if (hasHeader) {
        rows.push(headerHeight);
        areas.push(hasTwoCols ? '"header header"' : '"header"');
      }
      if (hasBody) {
        rows.push("1fr");
        if (hasTwoCols) areas.push('"aside main"');
        else if (hasAside) areas.push('"aside"');
        else areas.push('"main"');
      }
      if (hasFooter) {
        rows.push(footerHeight);
        areas.push(hasTwoCols ? '"footer footer"' : '"footer"');
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
        rawHeight === undefined || rawHeight === null
          ? ""
          : String(rawHeight).trim();
      const hasFixedHeight =
        normalizedHeight !== "" && normalizedHeight !== "auto";
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
      const rowGap = props.readonly
        ? 0
        : Math.max(0, Number(currentNode.props?.gutter) || 0);
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
    nodeRef,
    docRef,
    docVersionRef,
    resolvedNodePropsRef,
    isContainerRef,
    isMovableRef,
    layoutStyleRef,
    props = {},
  ) {
    return computed(() => {
      docVersionRef.value;
      if (!nodeRef.value) return {};
      const containerStyle = resolveContainerStyle(nodeRef.value, {});
      const customStyle = normalizeStyleObject(nodeRef.value.style || {});
      const textStyle =
        nodeRef.value.type === "Text"
          ? normalizeStyleObject(
              resolveTextPropStyle(resolvedNodePropsRef.value || {}),
            )
          : {};
      const parentNode = docRef.value?.getParent?.(nodeRef.value.id);
      const style = {
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
        } else if (
          nodeRef.value.type === "ElLayoutRow" ||
          nodeRef.value.type === "ElCol"
        ) {
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
      if (
        parentNode?.type === "ElHeader" ||
        parentNode?.type === "ElAside" ||
        parentNode?.type === "ElMain" ||
        parentNode?.type === "ElFooter"
      ) {
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
      if (parentNode?.type === "Tabs") {
        style.width = "100%";
        style.height = "100%";
        style.minHeight = "100%";
        style.alignSelf = "stretch";
        style.flex = "1 1 auto";
        if (!style.display) {
          style.display = "block";
        }
      }
      if (parentNode?.type === "ElCol" && isMovableRef.value) {
        style.width = "100%";
        if (isContainerRef.value) {
          style.height = "100%";
        } else if (!style.height) {
          style.height = "auto";
        }
        const rowParent = docRef.value?.getParent?.(parentNode.id);
        const rowHeight =
          rowParent?.type === "ElLayoutRow" ? rowParent.style?.height : undefined;
        const normalizedRowHeight =
          rowHeight === undefined || rowHeight === null
            ? ""
            : String(rowHeight).trim();
        const hasFixedRowHeight =
          normalizedRowHeight !== "" && normalizedRowHeight !== "auto";
        if (hasFixedRowHeight && (!style.height || style.height === "auto")) {
          style.height = "100%";
        }
      }
      if (parentNode?.type === "ElCol") {
        const rowParent = docRef.value?.getParent?.(parentNode.id);
        const rowHeight =
          rowParent?.type === "ElLayoutRow" ? rowParent.style?.height : undefined;
        const normalizedRowHeight =
          rowHeight === undefined || rowHeight === null
            ? ""
            : String(rowHeight).trim();
        const hasFixedRowHeight =
          normalizedRowHeight !== "" && normalizedRowHeight !== "auto";
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
      if (
        !style.width &&
        (parentNode?.type === "ElHeader" ||
          parentNode?.type === "ElAside" ||
          parentNode?.type === "ElMain" ||
          parentNode?.type === "ElFooter")
      ) {
        style.width = "100%";
      }
      if (
        !style.height &&
        (parentNode?.type === "ElHeader" ||
          parentNode?.type === "ElAside" ||
          parentNode?.type === "ElMain" ||
          parentNode?.type === "ElFooter")
      ) {
        style.height = "100%";
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
          const childStyle = parentDescriptor.childStyle(parentNode.type) || {};
          if (childStyle.width && !style.width) style.width = childStyle.width;
          if (childStyle.height && !style.height) style.height = childStyle.height;
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
    nodeRef,
    docRef,
    useComponentWrapperRef,
    wrapperStyleRef,
    contentStyleRef,
    resolvedPropsRef,
    props = {},
  ) {
    return computed(() => {
      if (!useComponentWrapperRef.value) return {};
      const style = { ...wrapperStyleRef.value, ...contentStyleRef.value };
      const type = nodeRef.value?.type;
      const parentNode = docRef.value?.getParent?.(nodeRef.value?.id);
      const customStyle = normalizeStyleObject(nodeRef.value?.style || {});
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
        const columns = Math.max(
          1,
          Math.min(24, Number(nodeRef.value?.props?.columns) || 1),
        );
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
      if (
        type === "ElLayoutRow" &&
        parentNode?.type === "ElLayout" &&
        isEditingMode
      ) {
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
          parentNode?.style?.height !== undefined
            ? parentNode.style.height
            : undefined;
        const normalizedRowHeight =
          rowHeight === undefined || rowHeight === null
            ? ""
            : String(rowHeight).trim();
        const hasFixedRowHeight =
          normalizedRowHeight !== "" && normalizedRowHeight !== "auto";
        if (hasFixedRowHeight) {
          style.height = "100%";
        }
        const colProps = resolvedPropsRef.value || nodeRef.value?.props || {};
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
            colContentInsetX > 0
              ? `calc(${clampedHalf} + ${colContentInsetX}px)`
              : clampedHalf;
          style.paddingRight =
            colContentInsetX > 0
              ? `calc(${clampedHalf} + ${colContentInsetX}px)`
              : clampedHalf;
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
  function createStyleElementSync(nodeRef, styleConfigCssRef) {
    const styleElementRef = ref(null);

    const syncStyleElement = () => {
      if (typeof document === "undefined") return;
      const nodeId = nodeRef.value?.id;
      const cssText = styleConfigCssRef.value;

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
        styleEl = document.querySelector(`style[data-style-node="${nodeId}"]`);
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
      styleConfigCssRef.value;
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
    nodeRef,
    docRef,
    layoutStyleRef,
    isMovableRef,
    isContainerRef,
  ) {
    return computed(() => {
      if (!nodeRef.value) return {};
      const style = { ...layoutStyleRef.value };
      const customStyle = normalizeStyleObject(nodeRef.value.style || {});
      if (customStyle.width && !style.width) {
        style.width = customStyle.width;
      }
      if (customStyle.height && !style.height) {
        style.height = customStyle.height;
      }
      const parentNode = docRef.value?.getParent?.(nodeRef.value.id);
      const hasCustomWidth = Boolean(customStyle.width);
      const hasCustomHeight = Boolean(customStyle.height);
      if (
        nodeRef.value.type === "ElCol" &&
        parentNode?.type === "ElLayoutRow" &&
        !style.height
      ) {
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
        style.width = "100%";
        style.height = "100%";
        if (style.position === "absolute") {
          style.left = "0";
          style.top = "0";
        }
      }
      if (isContainerRef.value && isMovableRef.value) {
        if (!hasCustomWidth && !style.width) style.width = "100%";
        if (!hasCustomHeight && !style.height) style.height = "100%";
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
