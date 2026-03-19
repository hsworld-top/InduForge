/**
 * Button 按钮组件 Manifest
 * 属性定义、事件定义、默认值
 */

/** @type {import('../../manifests/index.js').ComponentManifest} */
export const manifest = {
  type: "Button",
  name: "按钮",
  category: "PC端组件",
  props: [
    // ── 属性分组（对应旧版"属性"区块） ──────────────────────
    {
      name: "disabled",
      type: "boolean",
      label: "disabled",
      group: "属性",
      defaultValue: false,
    },
    {
      name: "plain",
      type: "boolean",
      label: "plain",
      group: "属性",
      defaultValue: false,
    },
    {
      name: "round",
      type: "boolean",
      label: "round",
      group: "属性",
      defaultValue: false,
    },
    {
      name: "circle",
      type: "boolean",
      label: "circle",
      group: "属性",
      defaultValue: false,
    },
    {
      name: "loading",
      type: "boolean",
      label: "loading",
      group: "属性",
      defaultValue: false,
    },
    {
      name: "type",
      type: "enum",
      label: "type",
      group: "属性",
      defaultValue: "primary",
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
      name: "text",
      type: "string",
      label: "text",
      group: "属性",
      defaultValue: "button",
      bindable: true,
    },
    {
      name: "icon",
      type: "string",
      label: "icon",
      group: "属性",
      defaultValue: "",
      placeholder: "如: el-icon-search",
    },
    // ── 尺寸策略分组 ──────────────────────────────────────────
    {
      name: "horizontalPolicy",
      type: "enum",
      label: "水平策略",
      group: "尺寸策略",
      defaultValue: "expanding",
      options: [
        { label: "固定", value: "fixed" },
        { label: "可延展", value: "expanding" },
      ],
    },
    {
      name: "fixedWidth",
      type: "number",
      label: "宽度",
      group: "尺寸策略",
      defaultValue: 90,
      min: 10,
      max: 2000,
    },
    {
      name: "verticalPolicy",
      type: "enum",
      label: "垂直策略",
      group: "尺寸策略",
      defaultValue: "expanding",
      options: [
        { label: "固定", value: "fixed" },
        { label: "可延展", value: "expanding" },
      ],
    },
    {
      name: "fixedHeight",
      type: "number",
      label: "高度",
      group: "尺寸策略",
      defaultValue: 106,
      min: 10,
      max: 2000,
    },
    // ── 安全策略分组 ──────────────────────────────────────────
    {
      name: "safetyControl",
      type: "boolean",
      label: "是否添加权限控制",
      group: "安全策略",
      defaultValue: false,
    },
    {
      name: "safetyDesc",
      type: "string",
      label: "权限描述",
      group: "安全策略",
      defaultValue: "",
      placeholder: "请输入权限描述",
    },
  ],
};

export default manifest;
