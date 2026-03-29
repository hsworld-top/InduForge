/**
 * 简单组件批量 Descriptor 注册
 *
 * 将 NodeRenderer 中 _builtinRenderTagMap 的组件迁移为 descriptor，
 * 仅需 renderTag（及少量 displayContent）的组件在此统一注册，
 * 复杂组件（Button、HorizontalLayout、VerticalLayout 等）在各自目录单独定义。
 */

import type { ComponentDescriptor, DescriptorNode } from "./registry";
import { registerDescriptor } from "./registry";

function normalizeOptions<T>(source: unknown, fallback: T[]): T[] {
  if (Array.isArray(source)) return source as T[];
  return fallback;
}

function omitResolvedKeys(
  resolvedProps: Record<string, unknown> | undefined,
  keys: string[],
): Record<string, unknown> {
  const o = { ...(resolvedProps ?? {}) };
  for (const k of keys) {
    delete o[k];
  }
  return o;
}

// Fallback 数据常量
const fallbackTableData = [
  { name: "张三", age: 28, address: "上海" },
  { name: "李四", age: 32, address: "北京" },
];
const fallbackBigTableData = [
  { name: "张三", value: 100 },
  { name: "李四", value: 200 },
];

/** 简单组件 type -> descriptor 映射（仅 renderTag，必要时带 displayContent） */
const SIMPLE_DESCRIPTOR_MAP: Record<string, ComponentDescriptor> = {
  Text: {
    renderTag: (
      _node: DescriptorNode | undefined,
      resolvedProps: Record<string, unknown> | undefined,
    ) => String(resolvedProps?.tag ?? "div"),
    displayContent: (node, resolvedProps) => String(resolvedProps?.text ?? node?.label ?? ""),
    propsFilter: (resolvedProps) => omitResolvedKeys(resolvedProps, ["text"]),
  },
  Input: { renderTag: "el-input" },
  Select: {
    renderTag: "el-select",
    propsFilter: (resolvedProps) => omitResolvedKeys(resolvedProps, ["options"]),
  },
  InputNumber: { renderTag: "el-input-number" },
  Switch: { renderTag: "el-switch" },
  Table: {
    renderTag: "el-table",
    renderKey: (node, ctx) => {
      const renderVersion = ctx?.tableRenderVersion ?? 0;
      const columnsSize = Array.isArray(node?.props?.columns) ? node.props.columns.length : 0;
      const dataSize = Array.isArray(node?.props?.data) ? node.props.data.length : 0;
      return `${node?.id ?? ""}-${columnsSize}-${dataSize}-${renderVersion}`;
    },
    propsFilter: (resolvedProps) => {
      const nextProps = omitResolvedKeys(resolvedProps, ["columns"]);
      nextProps.data = normalizeOptions(nextProps.data, fallbackTableData);
      return nextProps;
    },
  },
  BigDataTable: {
    renderTag: "el-table",
    renderKey: (node, ctx) => {
      const renderVersion = ctx?.tableRenderVersion ?? 0;
      const columnsSize = Array.isArray(node?.props?.columns) ? node.props.columns.length : 0;
      const dataSize = Array.isArray(node?.props?.data) ? node.props.data.length : 0;
      return `${node?.id ?? ""}-${columnsSize}-${dataSize}-${renderVersion}`;
    },
    propsFilter: (resolvedProps) => {
      const nextProps = omitResolvedKeys(resolvedProps, ["columns"]);
      nextProps.data = normalizeOptions(nextProps.data, fallbackBigTableData);
      return nextProps;
    },
  },
  Tree: { renderTag: "el-tree" },
  Transfer: { renderTag: "el-transfer" },
  Tag: {
    renderTag: "el-tag",
    displayContent: (node, resolvedProps) => String(resolvedProps?.text ?? node?.label ?? "标签"),
    propsFilter: (resolvedProps) => omitResolvedKeys(resolvedProps, ["text"]),
  },
  Dropdown: {
    renderTag: "el-dropdown",
    propsFilter: (resolvedProps) => omitResolvedKeys(resolvedProps, ["label", "items"]),
  },
  Menu: {
    renderTag: "el-menu",
    propsFilter: (resolvedProps) => omitResolvedKeys(resolvedProps, ["items"]),
  },
  Radio: {
    renderTag: "el-radio-group",
    propsFilter: (resolvedProps) => omitResolvedKeys(resolvedProps, ["options"]),
  },
  Checkbox: {
    renderTag: "el-checkbox-group",
    propsFilter: (resolvedProps) => omitResolvedKeys(resolvedProps, ["options"]),
  },
  Cascader: { renderTag: "el-cascader" },
  Image: { renderTag: "el-image" },
  Tabs: {
    renderTag: "el-tabs",
    isContainer: true,
    childPositioning: "flow",
    childLayout: "none",
    renderKey: (node, ctx) => {
      const resolved = (ctx?.resolvedProps ?? {}) as Record<string, unknown>;
      const tabs = Array.isArray(node?.props?.tabs) ? node.props.tabs : [];
      const tabKey = tabs.map((item) => item?.name ?? item?.label ?? "").join("|");
      const activeName = node?.props?.activeName ?? resolved.activeName ?? "";
      return `${node?.id ?? ""}-${tabKey}-${String(activeName)}`;
    },
    propsFilter: (resolvedProps) => omitResolvedKeys(resolvedProps, ["tabs"]),
  },
  Timeline: {
    renderTag: "el-timeline",
    propsFilter: (resolvedProps) => omitResolvedKeys(resolvedProps, ["items"]),
  },
  ImageCarousel: {
    renderTag: "el-carousel",
    propsFilter: (resolvedProps) => {
      const nextProps = omitResolvedKeys(resolvedProps, ["items"]);
      if (!nextProps.height) {
        nextProps.height = "160px";
      }
      return nextProps;
    },
  },
  CarouselComponent: {
    renderTag: "el-carousel",
    propsFilter: (resolvedProps) => {
      const nextProps = omitResolvedKeys(resolvedProps, ["items"]);
      if (!nextProps.height) {
        nextProps.height = "160px";
      }
      return nextProps;
    },
  },
  WebContainer: {
    renderTag: "el-card",
    displayContent: (_node, resolvedProps) =>
      resolvedProps?.url ? `网页容器: ${String(resolvedProps.url)}` : "网页容器",
  },
  Steps: {
    renderTag: "el-steps",
    propsFilter: (resolvedProps) => omitResolvedKeys(resolvedProps, ["items"]),
  },
  Card: {
    renderTag: "el-card",
    displayContent: (node, resolvedProps) =>
      String(resolvedProps?.content ?? node?.label ?? "卡片"),
    propsFilter: (resolvedProps) => omitResolvedKeys(resolvedProps, ["title", "content"]),
  },
  Pagination: { renderTag: "el-pagination" },
  Collapse: {
    renderTag: "el-collapse",
    isContainer: true,
    childPositioning: "flow",
    childLayout: "none",
    propsFilter: (resolvedProps) => omitResolvedKeys(resolvedProps, ["items"]),
  },
  BusinessCard: {
    renderTag: "el-card",
    displayContent: (node, resolvedProps) =>
      String(resolvedProps?.content ?? node?.label ?? "业务卡片"),
    propsFilter: (resolvedProps) => omitResolvedKeys(resolvedProps, ["title", "content"]),
  },
  Barcode: {
    renderTag: "el-card",
    displayContent: (_node, resolvedProps) => String(resolvedProps?.value ?? "1234567890"),
  },
  Slider: { renderTag: "el-slider" },
  Calendar: { renderTag: "el-calendar" },
  Signature: {
    renderTag: "el-input",
    propsFilter: (resolvedProps) => {
      const nextProps = { ...(resolvedProps ?? {}) };
      nextProps.type = "textarea";
      if (!nextProps.rows) {
        nextProps.rows = 3;
      }
      return nextProps;
    },
  },
  FormLayout: {
    renderTag: "div",
    isContainer: true,
    childLayout: "flex",
    childPositioning: "flow",
    childFlowLayout: { grow: 0, shrink: 0, basis: "auto" },
    defaultSize: { width: 360, height: 280 },
    containerStyle: (node: DescriptorNode) => {
      const props = node.props ?? {};
      const gap = Number(props.itemGap);
      return {
        display: "flex",
        flexDirection: "column",
        alignItems: "stretch",
        position: "relative",
        boxSizing: "border-box",
        width: "100%",
        minHeight: "120px",
        gap: `${Number.isFinite(gap) ? Math.max(0, gap) : 12}px`,
      };
    },
    childStyle: () => ({
      width: "100%",
      minWidth: "0",
    }),
  },
  // El 系列容器：Flex 布局
  ElContainer: {
    renderTag: "el-container",
    isContainer: true,
    childLayout: "flex",
    defaultSize: { width: 360, height: 240 },
    propsFilter: (resolvedProps) =>
      omitResolvedKeys(resolvedProps, [
        "regionPreset",
        "showHeader",
        "showAside",
        "showMain",
        "showFooter",
      ]),
  },
  ElHeader: {
    renderTag: "el-header",
    isContainer: true,
    childLayout: "flex",
    maxChildren: 1,
    isRegion: true,
    isMovable: false,
  },
  ElAside: {
    renderTag: "el-aside",
    isContainer: true,
    childLayout: "flex",
    maxChildren: 1,
    isRegion: true,
    isMovable: false,
  },
  ElMain: {
    renderTag: "el-main",
    isContainer: true,
    childLayout: "flex",
    maxChildren: 1,
    isRegion: true,
    isMovable: false,
  },
  ElFooter: {
    renderTag: "el-footer",
    isContainer: true,
    childLayout: "flex",
    maxChildren: 1,
    isRegion: true,
    isMovable: false,
  },
  ElLayout: {
    renderTag: "div",
    isContainer: true,
    childLayout: "flex",
    defaultSize: { width: 360, height: 200 },
    acceptChildren: ["ElLayoutRow"],
    propsFilter: (resolvedProps) => omitResolvedKeys(resolvedProps, ["columns", "rows"]),
  },
  ElLayoutRow: {
    renderTag: "el-row",
    isContainer: true,
    childLayout: "flex",
    acceptChildren: ["ElCol"],
    isMovable: false,
    propsFilter: (resolvedProps) => omitResolvedKeys(resolvedProps, ["columns"]),
  },
  ElCol: {
    renderTag: "el-col",
    isContainer: true,
    childLayout: "flex",
    maxChildren: 1,
    isRegion: true,
    isMovable: false,
  },
  // 布局容器：Flex 布局
  FlexContainer: {
    renderTag: "div",
    isContainer: true,
    childLayout: "flex",
    defaultSize: { width: 360, height: 200 },
  },
  ResponsiveLayout: {
    renderTag: "div",
    isContainer: true,
    childLayout: "flex",
  },
  // 布局容器：Grid 布局
  GridContainer: {
    renderTag: "div",
    isContainer: true,
    childLayout: "grid",
    defaultSize: { width: 360, height: 200 },
  },
  ColumnLayout1: { renderTag: "div", isContainer: true, childLayout: "grid" },
  ColumnLayout2: { renderTag: "div", isContainer: true, childLayout: "grid" },
  ColumnLayout4: { renderTag: "div", isContainer: true, childLayout: "grid" },
  // 布局容器：自由定位
  FreeContainer: {
    renderTag: "div",
    isContainer: true,
    childLayout: "free",
    defaultSize: { width: 360, height: 200 },
  },
};

/**
 * 注册所有简单组件描述符
 * 应在 registerAllDescriptors 中、各组件单独 registerDescriptor 之后调用，
 * 以便单独定义的 descriptor（如 Button、HorizontalLayout）优先，此处仅补全未覆盖类型
 */
export function registerSimpleDescriptors() {
  for (const [type, descriptor] of Object.entries(SIMPLE_DESCRIPTOR_MAP)) {
    registerDescriptor(type, descriptor);
  }
}

export default registerSimpleDescriptors;
