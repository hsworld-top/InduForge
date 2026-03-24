import { registerManifest } from "./manifest-registry";

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
