/**
 * FreeContainer 页面根自由布局容器 Manifest
 * 仅用于根容器能力，不在物料区展示。
 */

import type { ComponentManifest } from "@/manifests/manifest-registry";
import { registerManifest } from "@/manifests/manifest-registry";

export const manifest: ComponentManifest = {
  type: "FreeContainer",
  name: "页面根布局",
  category: "布局",
  isContainer: true,
  defaultSize: { width: 360, height: 200 },
  props: [],
};

registerManifest(manifest);

export default manifest;

