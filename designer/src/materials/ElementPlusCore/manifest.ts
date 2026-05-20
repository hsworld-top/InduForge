import type { ComponentManifest } from "@/materials/manifests/manifest-registry";
import { registerManifest } from "@/materials/manifests/manifest-registry";

const commonSizeOptions = [
  { label: "大", value: "large" },
  { label: "默认", value: "default" },
  { label: "小", value: "small" },
];

const commonOptions = [
  { label: "选项一", value: "option1" },
  { label: "选项二", value: "option2" },
  { label: "选项三", value: "option3" },
];

const tableColumns = [
  { prop: "name", label: "名称", width: 120 },
  { prop: "status", label: "状态", width: 100 },
  { prop: "value", label: "数值" },
];

const tableData = [
  { name: "设备A", status: "运行", value: 86 },
  { name: "设备B", status: "停机", value: 12 },
  { name: "设备C", status: "待机", value: 45 },
];

export const inputManifest: ComponentManifest = {
  type: "Input",
  name: "输入框",
  category: "PC端组件",
  defaultStyle: { width: "220px", height: "auto" },
  defaultSize: { width: 220, height: 34 },
  props: [
    { name: "modelValue", type: "string", label: "默认值", group: "内容", defaultValue: "", bindable: true },
    { name: "placeholder", type: "string", label: "占位文本", group: "内容", defaultValue: "请输入", bindable: true },
    { name: "type", type: "enum", label: "输入类型", group: "内容", defaultValue: "text", options: [
      { label: "文本", value: "text" },
      { label: "多行文本", value: "textarea" },
      { label: "密码", value: "password" },
    ] },
    { name: "size", type: "enum", label: "视觉规格", group: "外观", defaultValue: "default", options: commonSizeOptions },
    { name: "clearable", type: "boolean", label: "可清空", group: "状态", defaultValue: true },
    { name: "disabled", type: "boolean", label: "禁用", group: "状态", defaultValue: false },
  ],
};

export const inputNumberManifest: ComponentManifest = {
  type: "InputNumber",
  name: "数字输入",
  category: "PC端组件",
  defaultStyle: { width: "160px", height: "auto" },
  defaultSize: { width: 160, height: 34 },
  props: [
    { name: "modelValue", type: "number", label: "默认值", group: "内容", defaultValue: 0, bindable: true },
    { name: "min", type: "number", label: "最小值", group: "数据", defaultValue: 0 },
    { name: "max", type: "number", label: "最大值", group: "数据", defaultValue: 100 },
    { name: "step", type: "number", label: "步长", group: "数据", defaultValue: 1, min: 1 },
    { name: "size", type: "enum", label: "视觉规格", group: "外观", defaultValue: "default", options: commonSizeOptions },
    { name: "disabled", type: "boolean", label: "禁用", group: "状态", defaultValue: false },
  ],
};

export const selectManifest: ComponentManifest = {
  type: "Select",
  name: "选择器",
  category: "PC端组件",
  defaultStyle: { width: "220px", height: "auto" },
  defaultSize: { width: 220, height: 34 },
  props: [
    { name: "modelValue", type: "string", label: "默认值", group: "内容", defaultValue: "option1", bindable: true },
    { name: "placeholder", type: "string", label: "占位文本", group: "内容", defaultValue: "请选择", bindable: true },
    { name: "options", type: "array", label: "选项", group: "数据", defaultValue: commonOptions },
    { name: "size", type: "enum", label: "视觉规格", group: "外观", defaultValue: "default", options: commonSizeOptions },
    { name: "clearable", type: "boolean", label: "可清空", group: "状态", defaultValue: true },
    { name: "disabled", type: "boolean", label: "禁用", group: "状态", defaultValue: false },
  ],
};

export const radioGroupManifest: ComponentManifest = {
  type: "Radio",
  name: "单选框",
  category: "PC端组件",
  defaultStyle: { width: "260px", height: "auto" },
  defaultSize: { width: 260, height: 34 },
  props: [
    { name: "modelValue", type: "string", label: "默认值", group: "内容", defaultValue: "option1", bindable: true },
    { name: "options", type: "array", label: "选项", group: "数据", defaultValue: commonOptions },
    { name: "buttonStyle", type: "boolean", label: "按钮样式", group: "外观", defaultValue: false },
    { name: "size", type: "enum", label: "视觉规格", group: "外观", defaultValue: "default", options: commonSizeOptions },
    { name: "disabled", type: "boolean", label: "禁用", group: "状态", defaultValue: false },
  ],
};

export const checkboxGroupManifest: ComponentManifest = {
  type: "Checkbox",
  name: "多选框",
  category: "PC端组件",
  defaultStyle: { width: "280px", height: "auto" },
  defaultSize: { width: 280, height: 34 },
  props: [
    { name: "modelValue", type: "array", label: "默认值", group: "内容", defaultValue: ["option1"], bindable: true },
    { name: "options", type: "array", label: "选项", group: "数据", defaultValue: commonOptions },
    { name: "buttonStyle", type: "boolean", label: "按钮样式", group: "外观", defaultValue: false },
    { name: "size", type: "enum", label: "视觉规格", group: "外观", defaultValue: "default", options: commonSizeOptions },
    { name: "disabled", type: "boolean", label: "禁用", group: "状态", defaultValue: false },
  ],
};

export const switchManifest: ComponentManifest = {
  type: "Switch",
  name: "开关",
  category: "PC端组件",
  defaultStyle: { width: "80px", height: "auto" },
  defaultSize: { width: 80, height: 32 },
  props: [
    { name: "modelValue", type: "boolean", label: "默认值", group: "内容", defaultValue: true, bindable: true },
    { name: "activeText", type: "string", label: "开启文本", group: "内容", defaultValue: "" },
    { name: "inactiveText", type: "string", label: "关闭文本", group: "内容", defaultValue: "" },
    { name: "disabled", type: "boolean", label: "禁用", group: "状态", defaultValue: false },
  ],
};

export const tableManifest: ComponentManifest = {
  type: "Table",
  name: "表格",
  category: "PC端组件",
  defaultStyle: { width: "420px", height: "180px" },
  defaultSize: { width: 420, height: 180 },
  props: [
    { name: "columns", type: "array", label: "列配置", group: "数据", defaultValue: tableColumns },
    { name: "data", type: "array", label: "表格数据", group: "数据", defaultValue: tableData, bindable: true },
    { name: "stripe", type: "boolean", label: "斑马纹", group: "外观", defaultValue: true },
    { name: "border", type: "boolean", label: "显示边框", group: "外观", defaultValue: true },
    { name: "size", type: "enum", label: "视觉规格", group: "外观", defaultValue: "default", options: commonSizeOptions },
  ],
};

export const paginationManifest: ComponentManifest = {
  type: "Pagination",
  name: "分页",
  category: "PC端组件",
  defaultStyle: { width: "420px", height: "36px" },
  defaultSize: { width: 420, height: 36 },
  props: [
    { name: "currentPage", type: "number", label: "当前页", group: "数据", defaultValue: 1, min: 1, bindable: true },
    { name: "pageSize", type: "number", label: "每页条数", group: "数据", defaultValue: 10, min: 1, bindable: true },
    { name: "total", type: "number", label: "总数", group: "数据", defaultValue: 100, min: 0, bindable: true },
    { name: "background", type: "boolean", label: "背景色", group: "外观", defaultValue: true },
    { name: "small", type: "boolean", label: "小型", group: "外观", defaultValue: false },
  ],
};

[
  inputManifest,
  inputNumberManifest,
  selectManifest,
  radioGroupManifest,
  checkboxGroupManifest,
  switchManifest,
  tableManifest,
  paginationManifest,
].forEach(registerManifest);
