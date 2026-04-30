/**
 * Button 按钮组件 Manifest
 * 属性定义、事件定义、默认值
 */

import type { ComponentManifest } from "@/materials/manifests/manifest-registry";
import { registerManifest } from "@/materials/manifests/manifest-registry";

export const manifest: ComponentManifest = {
  type: "Button",
  name: "按钮",
  category: "PC端组件",
  defaultStyle: { width: "auto", height: "auto" },
  props: [
    {
      name: "text",
      type: "string",
      label: "按钮文字",
      group: "内容",
      defaultValue: "按钮",
      bindable: true,
    },
    {
      name: "icon",
      type: "string",
      label: "图标",
      group: "内容",
      defaultValue: "",
      editor: "icon",
      placeholder: "选择或输入图标名",
      options: [
        { label: "图标-搜索", value: "Search" },
        { label: "图标-新增", value: "Plus" },
        { label: "图标-下载", value: "Download" },
        { label: "图标-上传", value: "Upload" },
        { label: "图标-删除", value: "Delete" },
        { label: "图标-编辑", value: "EditPen" },
        { label: "图标-刷新", value: "Refresh" },
        { label: "图标-关闭", value: "Close" },
        { label: "图标-文档", value: "Document" },
        { label: "图标-文件夹", value: "Folder" },
      ],
    },
    {
      name: "type",
      type: "enum",
      label: "按钮类型",
      group: "外观",
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
      name: "size",
      type: "enum",
      label: "视觉规格",
      group: "外观",
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
      group: "外观",
      defaultValue: false,
    },
    {
      name: "shape",
      type: "enum",
      label: "圆角样式",
      group: "外观",
      defaultValue: "default",
      options: [
        { label: "默认", value: "default" },
        { label: "按钮圆角选项", value: "round" },
        { label: "按钮圆形选项", value: "circle" },
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
      name: "loading",
      type: "boolean",
      label: "加载中",
      group: "状态",
      defaultValue: false,
    },
  ],
};

registerManifest(manifest);

export default manifest;
