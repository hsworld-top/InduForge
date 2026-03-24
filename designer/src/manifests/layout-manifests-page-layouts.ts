/** 布局 manifest：响应式、网格、分栏、Tabs、Element 布局等页面级结构。在 H/V 包 manifest 之后加载。 */
import { registerManifest } from "./manifest-registry";

// ResponsiveLayout 响应式容器
registerManifest({
  type: "ResponsiveLayout",
  name: "响应式布局",
  category: "布局",
  isContainer: true,
  props: [
    {
      name: "direction",
      type: "enum",
      label: "方向",
      group: "布局",
      defaultValue: "row",
      options: [
        { label: "水平", value: "row" },
        { label: "垂直", value: "column" },
        { label: "水平反向", value: "row-reverse" },
        { label: "垂直反向", value: "column-reverse" },
      ],
    },
    {
      name: "wrap",
      type: "enum",
      label: "换行",
      group: "布局",
      defaultValue: "wrap",
      options: [
        { label: "不换行", value: "nowrap" },
        { label: "换行", value: "wrap" },
        { label: "反向换行", value: "wrap-reverse" },
      ],
    },
    {
      name: "justify",
      type: "enum",
      label: "主轴对齐",
      group: "布局",
      defaultValue: "flex-start",
      options: [
        { label: "起始", value: "flex-start" },
        { label: "居中", value: "center" },
        { label: "末尾", value: "flex-end" },
        { label: "两端", value: "space-between" },
        { label: "环绕", value: "space-around" },
        { label: "均匀", value: "space-evenly" },
      ],
    },
    {
      name: "align",
      type: "enum",
      label: "交叉轴对齐",
      group: "布局",
      defaultValue: "stretch",
      options: [
        { label: "拉伸", value: "stretch" },
        { label: "起始", value: "flex-start" },
        { label: "居中", value: "center" },
        { label: "末尾", value: "flex-end" },
        { label: "基线", value: "baseline" },
      ],
    },
    {
      name: "gap",
      type: "number",
      label: "间距",
      group: "布局",
      defaultValue: 0,
      min: 0,
      max: 100,
    },
  ],
});

// GridContainer 网格布局
registerManifest({
  type: "GridContainer",
  name: "网格布局",
  category: "布局",
  isContainer: true,
  props: [
    {
      name: "columns",
      type: "string",
      label: "列模板",
      group: "布局",
      defaultValue: "repeat(3, 1fr)",
    },
    {
      name: "rows",
      type: "string",
      label: "行模板",
      group: "布局",
      defaultValue: "repeat(2, 1fr)",
    },
    {
      name: "gap",
      type: "number",
      label: "间距",
      group: "布局",
      defaultValue: 0,
      min: 0,
      max: 100,
    },
  ],
});

// FreeContainer 自由布局
registerManifest({
  type: "FreeContainer",
  name: "自由布局",
  category: "布局",
  isContainer: true,
  props: [
    {
      name: "overflow",
      type: "enum",
      label: "溢出",
      group: "布局",
      defaultValue: "visible",
      options: [
        { label: "可见", value: "visible" },
        { label: "隐藏", value: "hidden" },
        { label: "滚动", value: "auto" },
      ],
    },
  ],
});

// ColumnLayout 分栏
registerManifest({
  type: "ColumnLayout1",
  name: "分栏*1",
  category: "布局",
  isContainer: true,
  defaultSize: { width: 360, height: 200 },
  props: [
    {
      name: "columns",
      type: "number",
      label: "列数",
      group: "布局",
      defaultValue: 1,
      min: 1,
      max: 12,
    },
    {
      name: "rows",
      type: "number",
      label: "行数",
      group: "布局",
      defaultValue: 1,
      min: 1,
      max: 12,
    },
    {
      name: "gap",
      type: "number",
      label: "间距",
      group: "布局",
      defaultValue: 0,
      min: 0,
      max: 100,
    },
  ],
});

registerManifest({
  type: "ColumnLayout2",
  name: "分栏*2",
  category: "布局",
  isContainer: true,
  defaultSize: { width: 360, height: 200 },
  props: [
    {
      name: "columns",
      type: "number",
      label: "列数",
      group: "布局",
      defaultValue: 2,
      min: 1,
      max: 12,
    },
    {
      name: "rows",
      type: "number",
      label: "行数",
      group: "布局",
      defaultValue: 1,
      min: 1,
      max: 12,
    },
    {
      name: "gap",
      type: "number",
      label: "间距",
      group: "布局",
      defaultValue: 0,
      min: 0,
      max: 100,
    },
  ],
});

registerManifest({
  type: "ColumnLayout4",
  name: "分栏*4",
  category: "布局",
  isContainer: true,
  defaultSize: { width: 360, height: 200 },
  props: [
    {
      name: "columns",
      type: "number",
      label: "列数",
      group: "布局",
      defaultValue: 4,
      min: 1,
      max: 12,
    },
    {
      name: "rows",
      type: "number",
      label: "行数",
      group: "布局",
      defaultValue: 1,
      min: 1,
      max: 12,
    },
    {
      name: "gap",
      type: "number",
      label: "间距",
      group: "布局",
      defaultValue: 0,
      min: 0,
      max: 100,
    },
  ],
});

// Element Plus Container
registerManifest({
  type: "ElContainer",
  name: "Container容器",
  category: "布局",
  isContainer: true,
  props: [
    {
      name: "showHeader",
      type: "boolean",
      label: "Header区域",
      group: "显示",
      defaultValue: true,
    },
    {
      name: "showAside",
      type: "boolean",
      label: "Aside区域",
      group: "显示",
      defaultValue: true,
    },
    {
      name: "showMain",
      type: "boolean",
      label: "Main区域",
      group: "显示",
      defaultValue: true,
    },
    {
      name: "showFooter",
      type: "boolean",
      label: "Footer区域",
      group: "显示",
      defaultValue: true,
    },
    {
      name: "headerHeight",
      type: "string",
      label: "Header高度",
      group: "布局",
      defaultValue: "60px",
    },
    {
      name: "asideWidth",
      type: "string",
      label: "Aside宽度",
      group: "布局",
      defaultValue: "200px",
    },
    {
      name: "footerHeight",
      type: "string",
      label: "Footer高度",
      group: "布局",
      defaultValue: "60px",
    },
  ],
});

registerManifest({
  type: "ElHeader",
  name: "Header",
  category: "布局",
  isContainer: true,
  props: [
    {
      name: "height",
      type: "string",
      label: "高度",
      group: "布局",
      defaultValue: "60px",
    },
  ],
});

registerManifest({
  type: "ElAside",
  name: "Aside",
  category: "布局",
  isContainer: true,
  props: [
    {
      name: "width",
      type: "string",
      label: "宽度",
      group: "布局",
      defaultValue: "200px",
    },
  ],
});

registerManifest({
  type: "ElMain",
  name: "Main",
  category: "布局",
  isContainer: true,
  props: [],
});

registerManifest({
  type: "ElFooter",
  name: "Footer",
  category: "布局",
  isContainer: true,
  props: [
    {
      name: "height",
      type: "string",
      label: "高度",
      group: "布局",
      defaultValue: "60px",
    },
  ],
});

// Element Plus Layout 布局
registerManifest({
  type: "ElLayout",
  name: "Layout布局",
  category: "布局",
  isContainer: true,
  props: [
    {
      name: "rows",
      type: "number",
      label: "行数",
      group: "布局",
      defaultValue: 1,
      min: 1,
      max: 24,
    },
    {
      name: "gutter",
      type: "number",
      label: "行间距",
      group: "布局",
      defaultValue: 12,
      min: 0,
      max: 100,
    },
    {
      name: "padding",
      type: "number",
      label: "内边距",
      group: "布局",
      defaultValue: 8,
      min: 0,
      max: 200,
    },
  ],
});

// Element Plus Layout 行容器
registerManifest({
  type: "ElLayoutRow",
  name: "行",
  category: "布局",
  isContainer: true,
  props: [
    {
      name: "columns",
      type: "number",
      label: "列数",
      group: "布局",
      defaultValue: 3,
      min: 1,
      max: 24,
    },
    {
      name: "gutter",
      type: "number",
      label: "列间距",
      group: "布局",
      defaultValue: 12,
      min: 0,
      max: 100,
    },
    {
      name: "xs",
      type: "object",
      label: "XS (<768px)",
      group: "响应式",
      placeholder: '4 或 {"span":4,"offset":4}',
    },
    {
      name: "sm",
      type: "object",
      label: "SM (≥768px)",
      group: "响应式",
      placeholder: '4 或 {"span":4,"offset":4}',
    },
    {
      name: "md",
      type: "object",
      label: "MD (≥992px)",
      group: "响应式",
      placeholder: '4 或 {"span":4,"offset":4}',
    },
    {
      name: "lg",
      type: "object",
      label: "LG (≥1200px)",
      group: "响应式",
      placeholder: '4 或 {"span":4,"offset":4}',
    },
    {
      name: "xl",
      type: "object",
      label: "XL (≥1920px)",
      group: "响应式",
      placeholder: '4 或 {"span":4,"offset":4}',
    },
  ],
});

// Element Plus Col 容器
registerManifest({
  type: "ElCol",
  name: "Col",
  category: "布局",
  isContainer: true,
  props: [
    {
      name: "span",
      type: "number",
      label: "栅格",
      group: "布局",
      defaultValue: 24,
      min: 1,
      max: 24,
    },
    {
      name: "offset",
      type: "number",
      label: "偏移",
      group: "布局",
      defaultValue: 0,
      min: 0,
      max: 24,
    },
    {
      name: "push",
      type: "number",
      label: "向右移动",
      group: "布局",
      defaultValue: 0,
      min: 0,
      max: 24,
    },
    {
      name: "pull",
      type: "number",
      label: "向左移动",
      group: "布局",
      defaultValue: 0,
      min: 0,
      max: 24,
    },
    {
      name: "tag",
      type: "string",
      label: "标签",
      group: "布局",
      defaultValue: "div",
    },
  ],
});
