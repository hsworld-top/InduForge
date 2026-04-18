/**
 * DownloadLink 下载组件 Manifest
 * 该组件保留给资产拖拽能力，不在默认物料区展示。
 */

import type { ComponentManifest } from "@/materials/manifests/manifest-registry";
import { registerManifest } from "@/materials/manifests/manifest-registry";

export const manifest: ComponentManifest = {
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
  ],
};

registerManifest(manifest);

export default manifest;

