/**
 * FormLayout 表单布局组件 Manifest
 */

import type { ComponentManifest } from "@/manifests/manifest-registry";
import { registerManifest } from "@/manifests/manifest-registry";

export const manifest: ComponentManifest = {
  type: "FormLayout",
  name: "表单布局",
  category: "布局",
  isContainer: true,
  defaultSize: { width: 360, height: 280 },
  props: [
    {
      name: "itemGap",
      type: "number",
      label: "表单项间距",
      group: "布局",
      defaultValue: 12,
      min: 0,
      max: 100,
    },
  ],
};

registerManifest(manifest);

export default manifest;

