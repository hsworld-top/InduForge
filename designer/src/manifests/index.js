/**
 * 组件 Manifest 注册表
 * 定义组件的属性、样式和事件配置
 */

/**
 * 属性类型
 * @typedef {'string' | 'number' | 'boolean' | 'color' | 'enum' | 'object' | 'array'} PropType
 */

/**
 * 属性定义
 * @typedef {{
 *   name: string,
 *   type: PropType,
 *   label: string,
 *   group?: string,
 *   defaultValue?: any,
 *   options?: Array<{ label: string, value: any }>,
 *   min?: number,
 *   max?: number,
 *   step?: number,
 *   placeholder?: string
 * }} PropDefinition
 */

/**
 * 组件 Manifest
 * @typedef {{
 *   type: string,
 *   name: string,
 *   category: string,
 *   props: PropDefinition[],
 *   defaultSize?: { width: number, height: number }
 * }} ComponentManifest
 */

/** @type {Map<string, ComponentManifest>} */
const manifestRegistry = new Map();

/**
 * 注册组件 Manifest
 * @param {ComponentManifest} manifest - 组件 Manifest
 */
export function registerManifest(manifest) {
  if (!manifest?.type) {
    console.warn("Invalid manifest: missing type");
    return;
  }
  manifestRegistry.set(manifest.type, manifest);
}

/**
 * 获取组件 Manifest
 * @param {string} type - 组件类型
 * @returns {ComponentManifest | undefined}
 */
export function getManifest(type) {
  return manifestRegistry.get(type);
}

/**
 * 获取所有已注册的 Manifest
 * @returns {ComponentManifest[]}
 */
export function getAllManifests() {
  return Array.from(manifestRegistry.values());
}

/**
 * 按分类获取 Manifest
 * @param {string} category - 分类名称
 * @returns {ComponentManifest[]}
 */
export function getManifestsByCategory(category) {
  return getAllManifests().filter((m) => m.category === category);
}

// ============ 内置组件 Manifest ============

// FlexContainer 布局容器
registerManifest({
  type: "FlexContainer",
  name: "弹性容器",
  category: "布局",
  isContainer: true,
  props: [
    {
      name: "direction",
      type: "enum",
      label: "方向",
      group: "布局",
      defaultValue: "column",
      options: [
        { label: "垂直", value: "column" },
        { label: "水平", value: "row" },
        { label: "垂直反向", value: "column-reverse" },
        { label: "水平反向", value: "row-reverse" },
      ],
    },
    {
      name: "wrap",
      type: "enum",
      label: "换行",
      group: "布局",
      defaultValue: "nowrap",
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

// FreeContainer 自由容器
registerManifest({
  type: "FreeContainer",
  name: "自由容器",
  category: "布局",
  isContainer: true,
  props: [
    {
      name: "overflow",
      type: "enum",
      label: "溢出处理",
      group: "布局",
      defaultValue: "visible",
      options: [
        { label: "可见", value: "visible" },
        { label: "隐藏", value: "hidden" },
        { label: "滚动", value: "auto" },
        { label: "强制滚动", value: "scroll" },
      ],
    },
  ],
});

// GridContainer 网格容器
registerManifest({
  type: "GridContainer",
  name: "网格容器",
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
      max: 12,
    },
    {
      name: "rows",
      type: "number",
      label: "行数",
      group: "布局",
      defaultValue: 3,
      min: 1,
      max: 12,
    },
    {
      name: "gap",
      type: "number",
      label: "间距",
      group: "布局",
      defaultValue: 8,
      min: 0,
      max: 50,
    },
    {
      name: "columnTemplate",
      type: "string",
      label: "列模板",
      group: "高级",
      defaultValue: "1fr 1fr 1fr",
      placeholder: "例如: 1fr 1fr 1fr 或 repeat(3, 1fr)",
    },
    {
      name: "rowTemplate",
      type: "string",
      label: "行模板",
      group: "高级",
      defaultValue: "auto auto auto",
      placeholder: "例如: auto auto auto 或 repeat(3, auto)",
    },
  ],
});

// ResponsiveLayout 响应式布局
registerManifest({
  type: "ResponsiveLayout",
  name: "响应式布局",
  category: "布局",
  isContainer: true,
  defaultSize: { width: 360, height: 200 },
  props: [
    {
      name: "direction",
      type: "enum",
      label: "方向",
      group: "布局",
      defaultValue: "row",
      options: [
        { label: "垂直", value: "column" },
        { label: "水平", value: "row" },
        { label: "垂直反向", value: "column-reverse" },
        { label: "水平反向", value: "row-reverse" },
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
      defaultValue: 8,
      min: 0,
      max: 100,
    },
  ],
});

// ColumnLayout1 分栏*1
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
      defaultValue: 8,
      min: 0,
      max: 50,
    },
    {
      name: "columnTemplate",
      type: "string",
      label: "列模板",
      group: "高级",
      defaultValue: "1fr",
      placeholder: "例如: 1fr 或 repeat(1, 1fr)",
    },
    {
      name: "rowTemplate",
      type: "string",
      label: "行模板",
      group: "高级",
      defaultValue: "auto",
      placeholder: "例如: auto 或 repeat(1, auto)",
    },
  ],
});

// ColumnLayout2 分栏*2
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
      defaultValue: 8,
      min: 0,
      max: 50,
    },
    {
      name: "columnTemplate",
      type: "string",
      label: "列模板",
      group: "高级",
      defaultValue: "1fr 1fr",
      placeholder: "例如: 1fr 1fr 或 repeat(2, 1fr)",
    },
    {
      name: "rowTemplate",
      type: "string",
      label: "行模板",
      group: "高级",
      defaultValue: "auto",
      placeholder: "例如: auto 或 repeat(1, auto)",
    },
  ],
});

// ColumnLayout4 分栏*4
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
      defaultValue: 8,
      min: 0,
      max: 50,
    },
    {
      name: "columnTemplate",
      type: "string",
      label: "列模板",
      group: "高级",
      defaultValue: "1fr 1fr 1fr 1fr",
      placeholder: "例如: 1fr 1fr 1fr 1fr 或 repeat(4, 1fr)",
    },
    {
      name: "rowTemplate",
      type: "string",
      label: "行模板",
      group: "高级",
      defaultValue: "auto",
      placeholder: "例如: auto 或 repeat(1, auto)",
    },
  ],
});

// Diagram2D 2D流程图组件
registerManifest({
  type: "ElContainer",
  name: "Container布局",
  category: "布局",
  isContainer: true,
  props: [
    {
      name: "showHeader",
      type: "boolean",
      label: "el-header",
      group: "区域",
      defaultValue: true,
    },
    {
      name: "headerHeight",
      type: "string",
      label: "Header高度",
      group: "区域",
      defaultValue: "60px",
    },
    {
      name: "showAside",
      type: "boolean",
      label: "el-aside",
      group: "区域",
      defaultValue: true,
    },
    {
      name: "asideWidth",
      type: "string",
      label: "Aside宽度",
      group: "区域",
      defaultValue: "200px",
    },
    {
      name: "showMain",
      type: "boolean",
      label: "el-main",
      group: "区域",
      defaultValue: true,
    },
    {
      name: "showFooter",
      type: "boolean",
      label: "el-footer",
      group: "区域",
      defaultValue: true,
    },
    {
      name: "footerHeight",
      type: "string",
      label: "Footer高度",
      group: "区域",
      defaultValue: "60px",
    },
  ],
});

// Element Plus Header 容器
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

// Element Plus Aside 容器
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

// Element Plus Main 容器
registerManifest({
  type: "ElMain",
  name: "Main",
  category: "布局",
  isContainer: true,
  props: [],
});

// Element Plus Footer 容器
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
      label: "间距",
      group: "布局",
      defaultValue: 0,
      min: 0,
      max: 100,
    },
    {
      name: "tag",
      type: "string",
      label: "标签",
      group: "布局",
      defaultValue: "div",
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

// Diagram2D 2D流程图组件
registerManifest({
  type: "Diagram2D",
  name: "2D流程图",
  category: "绘图",
  props: [
    {
      name: "diagramId",
      type: "string",
      label: "绘图ID",
      group: "内部",
      defaultValue: "",
    },
    {
      name: "showGrid",
      type: "boolean",
      label: "显示网格",
      group: "显示",
      defaultValue: true,
    },
    {
      name: "gridSize",
      type: "number",
      label: "网格大小",
      group: "显示",
      defaultValue: 10,
      min: 5,
      max: 50,
    },
    {
      name: "background",
      type: "color",
      label: "背景色",
      group: "样式",
      defaultValue: "#ffffff",
    },
    {
      name: "snapToGrid",
      type: "boolean",
      label: "吸附网格",
      group: "编辑",
      defaultValue: true,
    },
  ],
});

// Text 文本组件
registerManifest({
  type: "Text",
  name: "文本",
  category: "PC端组件",
  props: [
    {
      name: "text",
      type: "string",
      label: "文本内容",
      group: "基础",
      defaultValue: "文本内容",
      placeholder: "请输入文本",
    },
    {
      name: "tag",
      type: "enum",
      label: "标签类型",
      group: "基础",
      defaultValue: "span",
      options: [
        { label: "span", value: "span" },
        { label: "div", value: "div" },
        { label: "p", value: "p" },
        { label: "h1", value: "h1" },
        { label: "h2", value: "h2" },
        { label: "h3", value: "h3" },
        { label: "h4", value: "h4" },
        { label: "h5", value: "h5" },
        { label: "h6", value: "h6" },
      ],
    },
    {
      name: "fontSize",
      type: "number",
      label: "字号(px)",
      group: "字体",
      defaultValue: 14,
      min: 8,
      max: 200,
      step: 1,
    },
    {
      name: "fontWeight",
      type: "enum",
      label: "字重",
      group: "字体",
      defaultValue: "normal",
      options: [
        { label: "100-细", value: "100" },
        { label: "300-较细", value: "300" },
        { label: "正常", value: "normal" },
        { label: "500-中等", value: "500" },
        { label: "粗体", value: "bold" },
        { label: "900-最粗", value: "900" },
      ],
    },
    {
      name: "fontFamily",
      type: "string",
      label: "字体",
      group: "字体",
      defaultValue: "",
      placeholder: "如: Arial, 微软雅黑",
    },
    {
      name: "color",
      type: "color",
      label: "文字颜色",
      group: "颜色",
      defaultValue: "#333333",
    },
    {
      name: "textAlign",
      type: "enum",
      label: "对齐方式",
      group: "布局",
      defaultValue: "left",
      options: [
        { label: "左对齐", value: "left" },
        { label: "居中", value: "center" },
        { label: "右对齐", value: "right" },
        { label: "两端对齐", value: "justify" },
      ],
    },
    {
      name: "lineHeight",
      type: "number",
      label: "行高",
      group: "布局",
      defaultValue: 1.5,
      min: 0.5,
      max: 5,
      step: 0.1,
    },
    {
      name: "letterSpacing",
      type: "number",
      label: "字间距(px)",
      group: "布局",
      defaultValue: 0,
      min: -5,
      max: 20,
      step: 0.5,
    },
    {
      name: "truncate",
      type: "boolean",
      label: "单行省略",
      group: "溢出",
      defaultValue: false,
    },
    {
      name: "lineClamp",
      type: "number",
      label: "最大行数",
      group: "溢出",
      defaultValue: 0,
      min: 0,
      max: 20,
      placeholder: "0表示不限制",
    },
    {
      name: "wordBreak",
      type: "enum",
      label: "换行规则",
      group: "溢出",
      defaultValue: "normal",
      options: [
        { label: "正常", value: "normal" },
        { label: "断词", value: "break-all" },
        { label: "保持完整", value: "keep-all" },
      ],
    },
  ],
});

// Button 按钮组件（ElementPlus）
registerManifest({
  type: "Button",
  name: "按钮",
  category: "PC端组件",
  props: [
    {
      name: "text",
      type: "string",
      label: "文本",
      group: "基础",
      defaultValue: "按钮",
    },
    {
      name: "type",
      type: "enum",
      label: "类型",
      group: "基础",
      defaultValue: "default",
      options: [
        { label: "默认", value: "default" },
        { label: "主要", value: "primary" },
        { label: "成功", value: "success" },
        { label: "警告", value: "warning" },
        { label: "危险", value: "danger" },
        { label: "信息", value: "info" },
        { label: "文本", value: "text" },
      ],
    },
    {
      name: "size",
      type: "enum",
      label: "尺寸",
      group: "基础",
      defaultValue: "default",
      options: [
        { label: "大", value: "large" },
        { label: "默认", value: "default" },
        { label: "小", value: "small" },
      ],
    },
    {
      name: "plain",
      type: "boolean",
      label: "朴素按钮",
      group: "样式",
      defaultValue: false,
    },
    {
      name: "round",
      type: "boolean",
      label: "圆角按钮",
      group: "样式",
      defaultValue: false,
    },
    {
      name: "circle",
      type: "boolean",
      label: "圆形按钮",
      group: "样式",
      defaultValue: false,
    },
    {
      name: "link",
      type: "boolean",
      label: "链接按钮",
      group: "样式",
      defaultValue: false,
    },
    {
      name: "disabled",
      type: "boolean",
      label: "禁用",
      group: "状态",
      defaultValue: false,
    },
    {
      name: "loading",
      type: "boolean",
      label: "加载中",
      group: "状态",
      defaultValue: false,
    },
    {
      name: "icon",
      type: "string",
      label: "图标",
      group: "图标",
      defaultValue: "",
      placeholder: "如: el-icon-search",
    },
    {
      name: "iconPlacement",
      type: "enum",
      label: "图标位置",
      group: "图标",
      defaultValue: "left",
      options: [
        { label: "左侧", value: "left" },
        { label: "右侧", value: "right" },
      ],
    },
  ],
});

// Image 图片组件
registerManifest({
  type: "Image",
  name: "图片",
  category: "PC端组件",
  props: [
    {
      name: "src",
      type: "string",
      label: "图片地址",
      group: "基础",
      defaultValue: "",
      placeholder: "请输入图片 URL",
    },
    {
      name: "alt",
      type: "string",
      label: "替代文本",
      group: "基础",
      defaultValue: "",
    },
    {
      name: "fit",
      type: "enum",
      label: "填充模式",
      group: "样式",
      defaultValue: "cover",
      options: [
        { label: "填充", value: "fill" },
        { label: "包含", value: "contain" },
        { label: "覆盖", value: "cover" },
        { label: "无", value: "none" },
        { label: "缩小", value: "scale-down" },
      ],
    },
  ],
});

// Input 输入框组件
registerManifest({
  type: "Input",
  name: "输入框",
  category: "PC端组件",
  props: [
    {
      name: "placeholder",
      type: "string",
      label: "占位符",
      group: "基础",
      defaultValue: "请输入",
    },
    {
      name: "type",
      type: "enum",
      label: "类型",
      group: "基础",
      defaultValue: "text",
      options: [
        { label: "文本", value: "text" },
        { label: "密码", value: "password" },
        { label: "数字", value: "number" },
        { label: "邮箱", value: "email" },
        { label: "多行", value: "textarea" },
      ],
    },
    {
      name: "disabled",
      type: "boolean",
      label: "禁用",
      group: "状态",
      defaultValue: false,
    },
    {
      name: "readonly",
      type: "boolean",
      label: "只读",
      group: "状态",
      defaultValue: false,
    },
    {
      name: "clearable",
      type: "boolean",
      label: "可清空",
      group: "功能",
      defaultValue: false,
    },
  ],
});

// Select 选择器组件
registerManifest({
  type: "Select",
  name: "选择器",
  category: "PC端组件",
  props: [
    {
      name: "placeholder",
      type: "string",
      label: "占位符",
      group: "基础",
      defaultValue: "请选择",
    },
    {
      name: "multiple",
      type: "boolean",
      label: "多选",
      group: "功能",
      defaultValue: false,
    },
    {
      name: "clearable",
      type: "boolean",
      label: "可清空",
      group: "功能",
      defaultValue: false,
    },
    {
      name: "filterable",
      type: "boolean",
      label: "可搜索",
      group: "功能",
      defaultValue: false,
    },
    {
      name: "options",
      type: "array",
      label: "选项",
      group: "数据",
      defaultValue: [
        { label: "选项一", value: "option1" },
        { label: "选项二", value: "option2" },
      ],
    },
    {
      name: "disabled",
      type: "boolean",
      label: "禁用",
      group: "状态",
      defaultValue: false,
    },
  ],
});
// Switch 开关组件
registerManifest({
  type: "Switch",
  name: "开关",
  category: "PC端组件",
  props: [
    {
      name: "activeText",
      type: "string",
      label: "开启文本",
      group: "基础",
      defaultValue: "",
    },
    {
      name: "inactiveText",
      type: "string",
      label: "关闭文本",
      group: "基础",
      defaultValue: "",
    },
    {
      name: "disabled",
      type: "boolean",
      label: "禁用",
      group: "状态",
      defaultValue: false,
    },
  ],
});

// Table 表格组件
registerManifest({
  type: "Table",
  name: "表格",
  category: "PC端组件",
  defaultSize: { width: 360, height: 200 },
  props: [
    {
      name: "data",
      type: "array",
      label: "数据",
      group: "数据",
      defaultValue: [
        { name: "张三", age: 28, address: "上海" },
        { name: "李四", age: 32, address: "北京" },
      ],
    },
    {
      name: "columns",
      type: "array",
      label: "列",
      group: "数据",
      defaultValue: [
        { label: "姓名", prop: "name" },
        { label: "年龄", prop: "age" },
        { label: "地址", prop: "address" },
      ],
    },
    {
      name: "stripe",
      type: "boolean",
      label: "斑马纹",
      group: "样式",
      defaultValue: true,
    },
    {
      name: "border",
      type: "boolean",
      label: "边框",
      group: "样式",
      defaultValue: true,
    },
  ],
});

// Tree 树组件
registerManifest({
  type: "Tree",
  name: "树",
  category: "PC端组件",
  defaultSize: { width: 240, height: 200 },
  props: [
    {
      name: "data",
      type: "array",
      label: "数据",
      group: "数据",
      defaultValue: [
        {
          label: "一级 1",
          children: [{ label: "二级 1-1" }, { label: "二级 1-2" }],
        },
        {
          label: "一级 2",
          children: [{ label: "二级 2-1" }, { label: "二级 2-2" }],
        },
      ],
    },
    {
      name: "showCheckbox",
      type: "boolean",
      label: "显示复选框",
      group: "样式",
      defaultValue: false,
    },
    {
      name: "defaultExpandAll",
      type: "boolean",
      label: "默认展开",
      group: "样式",
      defaultValue: true,
    },
  ],
});

// Dropdown 下拉菜单组件
registerManifest({
  type: "Dropdown",
  name: "下拉菜单",
  category: "PC端组件",
  defaultSize: { width: 120, height: 32 },
  props: [
    {
      name: "label",
      type: "string",
      label: "按钮文字",
      group: "基础",
      defaultValue: "更多",
    },
    {
      name: "trigger",
      type: "enum",
      label: "触发方式",
      group: "行为",
      defaultValue: "click",
      options: [
        { label: "点击", value: "click" },
        { label: "悬停", value: "hover" },
      ],
    },
    {
      name: "items",
      type: "array",
      label: "菜单项",
      group: "数据",
      defaultValue: [
        { label: "操作一", value: "action1" },
        { label: "操作二", value: "action2" },
      ],
    },
  ],
});

// Menu 导航菜单组件
registerManifest({
  type: "Menu",
  name: "导航菜单",
  category: "PC端组件",
  defaultSize: { width: 240, height: 120 },
  props: [
    {
      name: "mode",
      type: "enum",
      label: "模式",
      group: "布局",
      defaultValue: "vertical",
      options: [
        { label: "垂直", value: "vertical" },
        { label: "水平", value: "horizontal" },
      ],
    },
    {
      name: "defaultActive",
      type: "string",
      label: "默认激活",
      group: "状态",
      defaultValue: "1",
    },
    {
      name: "items",
      type: "array",
      label: "菜单项",
      group: "数据",
      defaultValue: [
        { index: "1", label: "菜单一" },
        { index: "2", label: "菜单二" },
        { index: "3", label: "菜单三" },
      ],
    },
  ],
});

// Radio 单选框组件
registerManifest({
  type: "Radio",
  name: "单选框",
  category: "PC端组件",
  defaultSize: { width: 200, height: 32 },
  props: [
    {
      name: "modelValue",
      type: "string",
      label: "当前值",
      group: "数据",
      defaultValue: "option1",
    },
    {
      name: "options",
      type: "array",
      label: "选项",
      group: "数据",
      defaultValue: [
        { label: "选项一", value: "option1" },
        { label: "选项二", value: "option2" },
      ],
    },
  ],
});

// Checkbox 多选框组件
registerManifest({
  type: "Checkbox",
  name: "多选框",
  category: "PC端组件",
  defaultSize: { width: 200, height: 32 },
  props: [
    {
      name: "modelValue",
      type: "array",
      label: "当前值",
      group: "数据",
      defaultValue: ["option1"],
    },
    {
      name: "options",
      type: "array",
      label: "选项",
      group: "数据",
      defaultValue: [
        { label: "选项一", value: "option1" },
        { label: "选项二", value: "option2" },
      ],
    },
  ],
});

// Cascader 级联选择器组件
registerManifest({
  type: "Cascader",
  name: "级联选择器",
  category: "PC端组件",
  defaultSize: { width: 220, height: 32 },
  props: [
    {
      name: "options",
      type: "array",
      label: "选项",
      group: "数据",
      defaultValue: [
        {
          label: "一级 1",
          value: "1",
          children: [
            { label: "二级 1-1", value: "1-1" },
            { label: "二级 1-2", value: "1-2" },
          ],
        },
        {
          label: "一级 2",
          value: "2",
          children: [
            { label: "二级 2-1", value: "2-1" },
            { label: "二级 2-2", value: "2-2" },
          ],
        },
      ],
    },
    {
      name: "placeholder",
      type: "string",
      label: "占位符",
      group: "基础",
      defaultValue: "请选择",
    },
    {
      name: "clearable",
      type: "boolean",
      label: "可清空",
      group: "功能",
      defaultValue: false,
    },
  ],
});

// Tabs 标签页组件
registerManifest({
  type: "Tabs",
  name: "标签页",
  category: "PC端组件",
  defaultSize: { width: 360, height: 200 },
  props: [
    {
      name: "activeName",
      type: "string",
      label: "默认激活",
      group: "状态",
      defaultValue: "tab1",
    },
    {
      name: "type",
      type: "enum",
      label: "风格",
      group: "样式",
      defaultValue: "card",
      options: [
        { label: "默认", value: "" },
        { label: "卡片", value: "card" },
        { label: "边框卡片", value: "border-card" },
      ],
    },
    {
      name: "tabs",
      type: "array",
      label: "标签页",
      group: "数据",
      defaultValue: [
        { name: "tab1", label: "标签一", content: "内容一" },
        { name: "tab2", label: "标签二", content: "内容二" },
        { name: "tab3", label: "标签三", content: "内容三" },
      ],
    },
  ],
});

// Transfer 穿梭框组件
registerManifest({
  type: "Transfer",
  name: "穿梭框",
  category: "PC端组件",
  defaultSize: { width: 360, height: 200 },
  props: [
    {
      name: "data",
      type: "array",
      label: "数据",
      group: "数据",
      defaultValue: [
        { key: "1", label: "选项一" },
        { key: "2", label: "选项二" },
        { key: "3", label: "选项三" },
      ],
    },
    {
      name: "titles",
      type: "array",
      label: "标题",
      group: "显示",
      defaultValue: ["待选", "已选"],
    },
  ],
});

// Tag 标签组件
registerManifest({
  type: "Tag",
  name: "标签",
  category: "PC端组件",
  defaultSize: { width: 120, height: 32 },
  props: [
    {
      name: "text",
      type: "string",
      label: "文本",
      group: "基础",
      defaultValue: "标签",
    },
    {
      name: "type",
      type: "enum",
      label: "类型",
      group: "样式",
      defaultValue: "primary",
      options: [
        { label: "默认", value: "" },
        { label: "主要", value: "primary" },
        { label: "成功", value: "success" },
        { label: "警告", value: "warning" },
        { label: "危险", value: "danger" },
      ],
    },
  ],
});

// InputNumber 计数器组件
registerManifest({
  type: "InputNumber",
  name: "计数器",
  category: "PC端组件",
  defaultSize: { width: 160, height: 32 },
  props: [
    {
      name: "modelValue",
      type: "number",
      label: "当前值",
      group: "数据",
      defaultValue: 1,
    },
    {
      name: "min",
      type: "number",
      label: "最小值",
      group: "数据",
      defaultValue: 0,
    },
    {
      name: "max",
      type: "number",
      label: "最大值",
      group: "数据",
      defaultValue: 10,
    },
  ],
});

// Timeline 时间线组件
registerManifest({
  type: "Timeline",
  name: "时间线",
  category: "PC端组件",
  defaultSize: { width: 240, height: 200 },
  props: [
    {
      name: "items",
      type: "array",
      label: "节点",
      group: "数据",
      defaultValue: [
        { label: "步骤一", timestamp: "2024-01-01" },
        { label: "步骤二", timestamp: "2024-01-02" },
        { label: "步骤三", timestamp: "2024-01-03" },
      ],
    },
  ],
});

// ImageCarousel 图片轮播组件
registerManifest({
  type: "ImageCarousel",
  name: "图片轮播",
  category: "PC端组件",
  defaultSize: { width: 360, height: 200 },
  props: [
    {
      name: "height",
      type: "string",
      label: "高度",
      group: "样式",
      defaultValue: "160px",
    },
    {
      name: "items",
      type: "array",
      label: "图片",
      group: "数据",
      defaultValue: [
        { src: "", label: "轮播一" },
        { src: "", label: "轮播二" },
      ],
    },
  ],
});

// CarouselComponent 轮播组件
registerManifest({
  type: "CarouselComponent",
  name: "轮播组件",
  category: "PC端组件",
  defaultSize: { width: 360, height: 200 },
  props: [
    {
      name: "height",
      type: "string",
      label: "高度",
      group: "样式",
      defaultValue: "160px",
    },
    {
      name: "items",
      type: "array",
      label: "内容",
      group: "数据",
      defaultValue: [
        { label: "内容一" },
        { label: "内容二" },
      ],
    },
  ],
});

// WebContainer 网页容器组件
registerManifest({
  type: "WebContainer",
  name: "网页容器",
  category: "PC端组件",
  defaultSize: { width: 360, height: 200 },
  props: [
    {
      name: "url",
      type: "string",
      label: "URL",
      group: "基础",
      defaultValue: "",
    },
  ],
});

// Steps 步骤条组件
registerManifest({
  type: "Steps",
  name: "步骤条",
  category: "PC端组件",
  defaultSize: { width: 360, height: 120 },
  props: [
    {
      name: "active",
      type: "number",
      label: "当前步",
      group: "状态",
      defaultValue: 1,
    },
    {
      name: "items",
      type: "array",
      label: "步骤",
      group: "数据",
      defaultValue: [
        { title: "步骤一" },
        { title: "步骤二" },
        { title: "步骤三" },
      ],
    },
  ],
});

// Card 卡片组件
registerManifest({
  type: "Card",
  name: "卡片",
  category: "PC端组件",
  defaultSize: { width: 240, height: 140 },
  props: [
    {
      name: "title",
      type: "string",
      label: "标题",
      group: "基础",
      defaultValue: "卡片标题",
    },
    {
      name: "content",
      type: "string",
      label: "内容",
      group: "基础",
      defaultValue: "卡片内容",
    },
  ],
});

// Pagination 分页组件
registerManifest({
  type: "Pagination",
  name: "分页",
  category: "PC端组件",
  defaultSize: { width: 360, height: 40 },
  props: [
    {
      name: "currentPage",
      type: "number",
      label: "当前页",
      group: "数据",
      defaultValue: 1,
    },
    {
      name: "pageSize",
      type: "number",
      label: "每页条数",
      group: "数据",
      defaultValue: 10,
    },
    {
      name: "total",
      type: "number",
      label: "总数",
      group: "数据",
      defaultValue: 100,
    },
    {
      name: "layout",
      type: "string",
      label: "布局",
      group: "显示",
      defaultValue: "prev, pager, next",
    },
  ],
});

// Collapse 折叠面板组件
registerManifest({
  type: "Collapse",
  name: "折叠面板",
  category: "PC端组件",
  defaultSize: { width: 360, height: 200 },
  props: [
    {
      name: "items",
      type: "array",
      label: "面板",
      group: "数据",
      defaultValue: [
        { name: "1", title: "面板一", content: "内容一" },
        { name: "2", title: "面板二", content: "内容二" },
      ],
    },
  ],
});

// BigDataTable 大数据表格组件
registerManifest({
  type: "BigDataTable",
  name: "大数据表格",
  category: "PC端组件",
  defaultSize: { width: 360, height: 200 },
  props: [
    {
      name: "data",
      type: "array",
      label: "数据",
      group: "数据",
      defaultValue: [
        { name: "张三", value: 100 },
        { name: "李四", value: 200 },
      ],
    },
    {
      name: "columns",
      type: "array",
      label: "列",
      group: "数据",
      defaultValue: [
        { label: "名称", prop: "name" },
        { label: "数值", prop: "value" },
      ],
    },
  ],
});

// BusinessCard 业务卡片组件
registerManifest({
  type: "BusinessCard",
  name: "业务卡片",
  category: "PC端组件",
  defaultSize: { width: 240, height: 140 },
  props: [
    {
      name: "title",
      type: "string",
      label: "标题",
      group: "基础",
      defaultValue: "业务卡片",
    },
    {
      name: "content",
      type: "string",
      label: "内容",
      group: "基础",
      defaultValue: "指标说明",
    },
  ],
});

// Barcode 条形码组件
registerManifest({
  type: "Barcode",
  name: "条形码",
  category: "PC端组件",
  defaultSize: { width: 200, height: 80 },
  props: [
    {
      name: "value",
      type: "string",
      label: "内容",
      group: "基础",
      defaultValue: "1234567890",
    },
  ],
});

// Slider 滑块组件
registerManifest({
  type: "Slider",
  name: "滑块",
  category: "PC端组件",
  defaultSize: { width: 240, height: 32 },
  props: [
    {
      name: "modelValue",
      type: "number",
      label: "当前值",
      group: "数据",
      defaultValue: 30,
    },
    {
      name: "min",
      type: "number",
      label: "最小值",
      group: "数据",
      defaultValue: 0,
    },
    {
      name: "max",
      type: "number",
      label: "最大值",
      group: "数据",
      defaultValue: 100,
    },
  ],
});

// Calendar 日历组件
registerManifest({
  type: "Calendar",
  name: "日历",
  category: "PC端组件",
  defaultSize: { width: 360, height: 260 },
  props: [],
});

// Signature 电子签名组件
registerManifest({
  type: "Signature",
  name: "电子签名",
  category: "PC端组件",
  defaultSize: { width: 360, height: 120 },
  props: [
    {
      name: "placeholder",
      type: "string",
      label: "占位符",
      group: "基础",
      defaultValue: "请签名",
    },
  ],
});

// ============ Canvas 图形 Manifest ============

// Canvas.Rect 矩形
registerManifest({
  type: "Canvas.Rect",
  name: "矩形",
  category: "图形",
  props: [
    {
      name: "cornerRadius",
      type: "number",
      label: "圆角",
      group: "基础",
      defaultValue: 0,
      min: 0,
      max: 100,
    },
  ],
});

// Canvas.Circle 圆形
registerManifest({
  type: "Canvas.Circle",
  name: "圆形",
  category: "图形",
  props: [
    {
      name: "radius",
      type: "number",
      label: "半径",
      group: "基础",
      defaultValue: 50,
      min: 1,
      max: 1000,
    },
  ],
});

// Canvas.Line 线段
registerManifest({
  type: "Canvas.Line",
  name: "线段",
  category: "图形",
  props: [
    {
      name: "tension",
      type: "number",
      label: "张力",
      group: "基础",
      defaultValue: 0,
      min: 0,
      max: 1,
      step: 0.1,
    },
    {
      name: "closed",
      type: "boolean",
      label: "闭合",
      group: "基础",
      defaultValue: false,
    },
  ],
});

// Canvas.Pipe 管道
registerManifest({
  type: "Canvas.Pipe",
  name: "管道",
  category: "图形",
  props: [
    {
      name: "flowSpeed",
      type: "number",
      label: "流速",
      group: "动画",
      defaultValue: 0,
      min: 0,
      max: 100,
    },
    {
      name: "flowDirection",
      type: "enum",
      label: "流向",
      group: "动画",
      defaultValue: "forward",
      options: [
        { label: "正向", value: "forward" },
        { label: "反向", value: "backward" },
      ],
    },
    {
      name: "pipeWidth",
      type: "number",
      label: "管道宽度",
      group: "基础",
      defaultValue: 10,
      min: 1,
      max: 100,
    },
  ],
});

// Canvas.Text 文字标注
registerManifest({
  type: "Canvas.Text",
  name: "文字标注",
  category: "图形",
  props: [
    {
      name: "text",
      type: "string",
      label: "文本",
      group: "基础",
      defaultValue: "文字",
    },
    {
      name: "fontSize",
      type: "number",
      label: "字号",
      group: "样式",
      defaultValue: 14,
      min: 8,
      max: 200,
    },
    {
      name: "fontFamily",
      type: "enum",
      label: "字体",
      group: "样式",
      defaultValue: "system-ui",
      options: [
        { label: "系统默认", value: "system-ui" },
        { label: "思源黑体", value: "Source Han Sans SC" },
        { label: "等宽字体", value: "monospace" },
        { label: "数码字体", value: "DSEG7 Classic" },
      ],
    },
    {
      name: "align",
      type: "enum",
      label: "对齐",
      group: "样式",
      defaultValue: "left",
      options: [
        { label: "左对齐", value: "left" },
        { label: "居中", value: "center" },
        { label: "右对齐", value: "right" },
      ],
    },
  ],
});

export default {
  registerManifest,
  getManifest,
  getAllManifests,
  getManifestsByCategory,
};
