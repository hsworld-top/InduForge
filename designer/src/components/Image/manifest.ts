/**
 * Image 图片组件 Manifest
 * 该组件保留给资产拖拽能力，不在默认物料区展示。
 */

import type { ComponentManifest } from "@/manifests/manifest-registry";
import { registerManifest } from "@/manifests/manifest-registry";

export const manifest: ComponentManifest = {
  type: "Image",
  name: "图片",
  category: "PC端组件",
  defaultSize: { width: 240, height: 160 },
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
      defaultValue: "contain",
      options: [
        { label: "填充", value: "fill" },
        { label: "包含", value: "contain" },
        { label: "覆盖", value: "cover" },
        { label: "不缩放", value: "none" },
        { label: "缩放", value: "scale-down" },
      ],
    },
  ],
};

registerManifest(manifest);

export default manifest;

