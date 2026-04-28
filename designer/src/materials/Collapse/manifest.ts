/**
 * Collapse 折叠布局组件 Manifest
 */

import type { ComponentManifest } from "@/materials/manifests/manifest-registry";
import { registerManifest } from "@/materials/manifests/manifest-registry";

export const manifest: ComponentManifest = {
  type: "Collapse",
  name: "折叠面板布局",
  category: "布局",
  isContainer: true,
  defaultSize: { width: 360, height: 200 },
  props: [
    {
      name: "modelValue",
      type: "array",
      label: "默认展开",
      group: "状态",
      defaultValue: ["1"],
    },
    {
      name: "accordion",
      type: "boolean",
      label: "手风琴",
      group: "状态",
      defaultValue: false,
    },
    {
      name: "items",
      type: "array",
      label: "面板",
      group: "数据",
      defaultValue: [
        { name: "1", title: "面板1", content: "内容1" },
        { name: "2", title: "面板2", content: "内容2" },
      ],
    },
  ],
};

registerManifest(manifest);

export default manifest;

