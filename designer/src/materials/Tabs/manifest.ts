/**
 * Tabs 选项卡布局组件 Manifest
 */

import type { ComponentManifest } from "@/materials/manifests/manifest-registry";
import { registerManifest } from "@/materials/manifests/manifest-registry";

export const manifest: ComponentManifest = {
  type: "Tabs",
  name: "选项卡布局",
  category: "布局",
  isContainer: true,
  defaultSize: { width: 360, height: 200 },
  props: [
    {
      name: "modelValue",
      type: "string",
      label: "当前激活",
      group: "状态",
      defaultValue: "tab1",
    },
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
      defaultValue: [
        { name: "tab1", label: "标签1", disabled: false, content: "" },
        { name: "tab2", label: "标签2", disabled: false, content: "" },
      ],
    },
  ],
};

registerManifest(manifest);

export default manifest;

