/** 展示类 manifest：图片、媒体与大量 PC 端展示组件。由 index 在 Button 自注册之后加载。 */
import { registerManifest } from "./manifest-registry";

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

// DownloadLink 下载链接组件
registerManifest({
  type: "DownloadLink",
  name: "下载组件",
  category: "PC端组件",
  defaultSize: { width: 220, height: 32 },
  props: [
    {
      name: "text",
      type: "string",
      label: "文案",
      group: "基础",
      defaultValue: "下载文件",
      placeholder: "请输入链接文本",
    },
    {
      name: "href",
      type: "string",
      label: "下载地址",
      group: "基础",
      defaultValue: "",
      placeholder: "请输入资源 URL",
    },
    {
      name: "displayMode",
      type: "enum",
      label: "展示形态",
      group: "样式",
      defaultValue: "button",
      options: [
        { label: "按钮", value: "button" },
        { label: "链接", value: "link" },
      ],
    },
    {
      name: "actionMode",
      type: "enum",
      label: "点击行为",
      group: "行为",
      defaultValue: "download",
      options: [
        { label: "下载", value: "download" },
        { label: "打开", value: "open" },
        { label: "无动作", value: "none" },
      ],
    },
    {
      name: "triggerMode",
      type: "enum",
      label: "触发方式",
      group: "行为",
      defaultValue: "double",
      options: [
        { label: "单击", value: "single" },
        { label: "双击", value: "double" },
      ],
    },
    {
      name: "target",
      type: "enum",
      label: "打开方式",
      group: "行为",
      defaultValue: "_blank",
      options: [
        { label: "新窗口", value: "_blank" },
        { label: "当前窗口", value: "_self" },
      ],
    },
    {
      name: "downloadFileName",
      type: "string",
      label: "下载文件名",
      group: "行为",
      defaultValue: "",
      placeholder: "留空时使用文案",
    },
    {
      name: "disabled",
      type: "boolean",
      label: "禁用",
      group: "状态",
      defaultValue: false,
    },
    {
      name: "fontSize",
      type: "number",
      label: "字体大小",
      group: "样式",
      defaultValue: 14,
    },
    {
      name: "fontWeight",
      type: "enum",
      label: "字重",
      group: "样式",
      defaultValue: 500,
      options: [
        { label: "常规", value: 400 },
        { label: "中等", value: 500 },
        { label: "加粗", value: 600 },
      ],
    },
    {
      name: "textColor",
      type: "color",
      label: "文字颜色",
      group: "样式",
      defaultValue: "#2563eb",
    },
    {
      name: "backgroundColor",
      type: "color",
      label: "背景颜色",
      group: "样式",
      defaultValue: "#ffffff",
    },
    {
      name: "borderColor",
      type: "color",
      label: "边框颜色",
      group: "样式",
      defaultValue: "#2563eb",
    },
    {
      name: "borderRadius",
      type: "number",
      label: "圆角",
      group: "样式",
      defaultValue: 6,
    },
    {
      name: "paddingX",
      type: "number",
      label: "水平内边距",
      group: "样式",
      defaultValue: 12,
    },
    {
      name: "paddingY",
      type: "number",
      label: "垂直内边距",
      group: "样式",
      defaultValue: 6,
    },
    {
      name: "underline",
      type: "boolean",
      label: "链接下划线",
      group: "样式",
      defaultValue: true,
    },
  ],
});

// Tabs 标签页组件
registerManifest({
  type: "Tabs",
  name: "选项卡布局",
  category: "布局",
  isContainer: true,
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
      name: "closable",
      type: "boolean",
      label: "可关闭",
      group: "功能",
      defaultValue: false,
    },
    {
      name: "tabPosition",
      type: "enum",
      label: "标签位置",
      group: "样式",
      defaultValue: "top",
      options: [
        { label: "上", value: "top" },
        { label: "右", value: "right" },
        { label: "下", value: "bottom" },
        { label: "左", value: "left" },
      ],
    },
    {
      name: "stretch",
      type: "boolean",
      label: "宽度自撑",
      group: "样式",
      defaultValue: false,
    },
    {
      name: "tabs",
      type: "array",
      label: "标签页",
      group: "数据",
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
      defaultValue: [{ label: "内容一" }, { label: "内容二" }],
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
      defaultValue: [{ title: "步骤一" }, { title: "步骤二" }, { title: "步骤三" }],
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
  name: "折叠面板布局",
  category: "布局",
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
