/**
 * 简单组件批量 Descriptor 注册
 *
 * 将 NodeRenderer 中 _builtinRenderTagMap 的组件迁移为 descriptor，
 * 仅需 renderTag（及少量 displayContent）的组件在此统一注册，
 * 复杂组件（Button、HorizontalLayout、VerticalLayout 等）在各自目录单独定义。
 *
 * @module components/descriptors/simple-descriptors
 */

import { registerDescriptor } from "./registry.ts";

/**
 * 规范化选项列表（用于 propsFilter）
 * @param {Array} source - 原始列表
 * @param {Array} fallback - 默认列表
 * @returns {Array}
 */
const normalizeOptions = (source, fallback) => {
  if (Array.isArray(source)) return source;
  return fallback || [];
};

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
const SIMPLE_DESCRIPTOR_MAP = {
  Text: {
    renderTag: (node, resolvedProps) => resolvedProps?.tag || "div",
    displayContent: (node, resolvedProps) => resolvedProps?.text ?? node?.label ?? "",
    propsFilter: (resolvedProps) => {
      const { text, ...elProps } = resolvedProps ?? {};
      return elProps;
    },
  },
  Input: { renderTag: "el-input" },
  Select: {
    renderTag: "el-select",
    propsFilter: (resolvedProps) => {
      const { options, ...elProps } = resolvedProps ?? {};
      return elProps;
    },
  },
  InputNumber: { renderTag: "el-input-number" },
  Switch: { renderTag: "el-switch" },
  Table: {
    renderTag: "el-table",
    renderKey: (node, ctx) => {
      const renderVersion = ctx?.tableRenderVersion ?? 0;
      const columnsSize = Array.isArray(node?.props?.columns)
        ? node.props.columns.length
        : 0;
      const dataSize = Array.isArray(node?.props?.data)
        ? node.props.data.length
        : 0;
      return `${node?.id ?? ""}-${columnsSize}-${dataSize}-${renderVersion}`;
    },
    propsFilter: (resolvedProps) => {
      const { columns, ...rest } = resolvedProps ?? {};
      const nextProps = { ...rest };
      nextProps.data = normalizeOptions(nextProps.data, fallbackTableData);
      return nextProps;
    },
  },
  BigDataTable: {
    renderTag: "el-table",
    renderKey: (node, ctx) => {
      const renderVersion = ctx?.tableRenderVersion ?? 0;
      const columnsSize = Array.isArray(node?.props?.columns)
        ? node.props.columns.length
        : 0;
      const dataSize = Array.isArray(node?.props?.data)
        ? node.props.data.length
        : 0;
      return `${node?.id ?? ""}-${columnsSize}-${dataSize}-${renderVersion}`;
    },
    propsFilter: (resolvedProps) => {
      const { columns, ...rest } = resolvedProps ?? {};
      const nextProps = { ...rest };
      nextProps.data = normalizeOptions(nextProps.data, fallbackBigTableData);
      return nextProps;
    },
  },
  Tree: { renderTag: "el-tree" },
  Transfer: { renderTag: "el-transfer" },
  Tag: {
    renderTag: "el-tag",
    displayContent: (node, resolvedProps) => resolvedProps?.text ?? node?.label ?? "标签",
    propsFilter: (resolvedProps) => {
      const { text, ...elProps } = resolvedProps ?? {};
      return elProps;
    },
  },
  Dropdown: {
    renderTag: "el-dropdown",
    propsFilter: (resolvedProps) => {
      const { label, items, ...elProps } = resolvedProps ?? {};
      return elProps;
    },
  },
  Menu: {
    renderTag: "el-menu",
    propsFilter: (resolvedProps) => {
      const { items, ...elProps } = resolvedProps ?? {};
      return elProps;
    },
  },
  Radio: {
    renderTag: "el-radio-group",
    propsFilter: (resolvedProps) => {
      const { options, ...elProps } = resolvedProps ?? {};
      return elProps;
    },
  },
  Checkbox: {
    renderTag: "el-checkbox-group",
    propsFilter: (resolvedProps) => {
      const { options, ...elProps } = resolvedProps ?? {};
      return elProps;
    },
  },
  Cascader: { renderTag: "el-cascader" },
  Image: { renderTag: "el-image" },
  Tabs: {
    renderTag: "el-tabs",
    isContainer: true,
    childLayout: "none",
    renderKey: (node, ctx) => {
      const resolved = ctx?.resolvedProps ?? {};
      const tabs = Array.isArray(node?.props?.tabs) ? node.props.tabs : [];
      const tabKey = tabs
        .map((item) => item?.name ?? item?.label ?? "")
        .join("|");
      const activeName =
        node?.props?.activeName ?? resolved.activeName ?? "";
      return `${node?.id ?? ""}-${tabKey}-${String(activeName)}`;
    },
    propsFilter: (resolvedProps) => {
      const { tabs, ...elProps } = resolvedProps ?? {};
      return elProps;
    },
  },
  Timeline: {
    renderTag: "el-timeline",
    propsFilter: (resolvedProps) => {
      const { items, ...elProps } = resolvedProps ?? {};
      return elProps;
    },
  },
  ImageCarousel: {
    renderTag: "el-carousel",
    propsFilter: (resolvedProps) => {
      const { items, ...rest } = resolvedProps ?? {};
      const nextProps = { ...rest };
      if (!nextProps.height) {
        nextProps.height = "160px";
      }
      return nextProps;
    },
  },
  CarouselComponent: {
    renderTag: "el-carousel",
    propsFilter: (resolvedProps) => {
      const { items, ...rest } = resolvedProps ?? {};
      const nextProps = { ...rest };
      if (!nextProps.height) {
        nextProps.height = "160px";
      }
      return nextProps;
    },
  },
  WebContainer: {
    renderTag: "el-card",
    displayContent: (node, resolvedProps) => resolvedProps?.url
      ? `网页容器: ${resolvedProps.url}`
      : "网页容器",
  },
  Steps: {
    renderTag: "el-steps",
    propsFilter: (resolvedProps) => {
      const { items, ...elProps } = resolvedProps ?? {};
      return elProps;
    },
  },
  Card: {
    renderTag: "el-card",
    displayContent: (node, resolvedProps) => resolvedProps?.content ?? node?.label ?? "卡片",
    propsFilter: (resolvedProps) => {
      const { title, content, ...elProps } = resolvedProps ?? {};
      return elProps;
    },
  },
  Pagination: { renderTag: "el-pagination" },
  Collapse: {
    renderTag: "el-collapse",
    propsFilter: (resolvedProps) => {
      const { items, ...elProps } = resolvedProps ?? {};
      return elProps;
    },
  },
  BusinessCard: {
    renderTag: "el-card",
    displayContent: (node, resolvedProps) => resolvedProps?.content ?? node?.label ?? "业务卡片",
    propsFilter: (resolvedProps) => {
      const { title, content, ...elProps } = resolvedProps ?? {};
      return elProps;
    },
  },
  Barcode: {
    renderTag: "el-card",
    displayContent: (node, resolvedProps) => resolvedProps?.value ?? "1234567890",
  },
  Slider: { renderTag: "el-slider" },
  Calendar: { renderTag: "el-calendar" },
  Signature: {
    renderTag: "el-input",
    propsFilter: (resolvedProps) => {
      const nextProps = { ...resolvedProps };
      nextProps.type = "textarea";
      if (!nextProps.rows) {
        nextProps.rows = 3;
      }
      return nextProps;
    },
  },
  // El 系列容器：Flex 布局
  ElContainer: {
    renderTag: "el-container",
    isContainer: true,
    childLayout: "flex",
    defaultSize: { width: 360, height: 240 },
    propsFilter: (resolvedProps) => {
      const { showHeader, showAside, showMain, showFooter, ...elProps } = resolvedProps ?? {};
      return elProps;
    },
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
    propsFilter: (resolvedProps) => {
      const { columns, rows, ...elProps } = resolvedProps ?? {};
      return elProps;
    },
  },
  ElLayoutRow: {
    renderTag: "el-row",
    isContainer: true,
    childLayout: "flex",
    acceptChildren: ["ElCol"],
    isMovable: false,
    propsFilter: (resolvedProps) => {
      const { columns, ...elProps } = resolvedProps ?? {};
      return elProps;
    },
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
  FlexContainer: { renderTag: "div", isContainer: true, childLayout: "flex", defaultSize: { width: 360, height: 200 } },
  ResponsiveLayout: { renderTag: "div", isContainer: true, childLayout: "flex" },
  // 布局容器：Grid 布局
  GridContainer: { renderTag: "div", isContainer: true, childLayout: "grid", defaultSize: { width: 360, height: 200 } },
  ColumnLayout1: { renderTag: "div", isContainer: true, childLayout: "grid" },
  ColumnLayout2: { renderTag: "div", isContainer: true, childLayout: "grid" },
  ColumnLayout4: { renderTag: "div", isContainer: true, childLayout: "grid" },
  // 布局容器：自由定位
  FreeContainer: { renderTag: "div", isContainer: true, childLayout: "free", defaultSize: { width: 360, height: 200 } },
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
