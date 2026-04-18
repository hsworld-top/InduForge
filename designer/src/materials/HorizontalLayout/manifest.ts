/**
 * HorizontalLayout 水平布局组件 Manifest
 */

import type { ComponentManifest } from "@/materials/manifests/manifest-registry";
import { registerManifest } from "@/materials/manifests/manifest-registry";

export const manifest: ComponentManifest = {
  type: "HorizontalLayout",
  name: "水平布局",
  category: "布局",
  isContainer: true,
  defaultSize: { width: 400, height: 160 },
  props: [
    {
      name: "justify",
      type: "enum",
      label: "水平排列",
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
      label: "垂直对齐",
      group: "布局",
      defaultValue: "stretch",
      options: [
        { label: "拉伸", value: "stretch" },
        { label: "顶部", value: "flex-start" },
        { label: "居中", value: "center" },
        { label: "底部", value: "flex-end" },
        { label: "基线", value: "baseline" },
      ],
    },
    {
      name: "gap",
      type: "number",
      label: "间距",
      group: "布局",
      defaultValue: 10,
      min: 0,
      max: 100,
    },
  ],
};

registerManifest(manifest);

export default manifest;
