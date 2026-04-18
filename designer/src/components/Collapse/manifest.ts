/**
 * Collapse 折叠布局组件 Manifest
 */

import type { ComponentManifest } from "@/manifests/manifest-registry";
import { registerManifest } from "@/manifests/manifest-registry";

export const manifest: ComponentManifest = {
  type: "Collapse",
  name: "折叠面板布局",
  category: "布局",
  isContainer: true,
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
};

registerManifest(manifest);

export default manifest;

