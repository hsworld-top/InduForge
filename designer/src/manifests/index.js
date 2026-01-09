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
 *   props: PropDefinition[]
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
  category: "基础",
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
  category: "基础",
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
  category: "基础",
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
  category: "表单",
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
  category: "表单",
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
  category: "表单",
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
