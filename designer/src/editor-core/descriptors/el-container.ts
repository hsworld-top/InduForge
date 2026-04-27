/**
 * ElContainer 区域布局及子区域 Descriptor
 */

import type { ComponentDescriptor } from "./registry";

function omitKeys(
  resolvedProps: Record<string, unknown> | undefined,
  keys: string[],
): Record<string, unknown> {
  const next = { ...(resolvedProps ?? {}) };
  for (const key of keys) {
    delete next[key];
  }
  return next;
}

export const descriptor: ComponentDescriptor = {
  renderTag: "el-container",
  isContainer: true,
  childLayout: "flex",
  defaultSize: { width: 360, height: 240 },
  acceptChildren: ["ElHeader", "ElAside", "ElMain", "ElFooter"],
  propsFilter: (resolvedProps) =>
    omitKeys(resolvedProps, ["regionPreset", "showHeader", "showAside", "showMain", "showFooter"]),
};

export const headerDescriptor: ComponentDescriptor = {
  renderTag: "el-header",
  isContainer: true,
  childLayout: "flex",
  childPositioning: "flow",
  maxChildren: 1,
  isRegion: true,
  isMovable: false,
};

export const asideDescriptor: ComponentDescriptor = {
  renderTag: "el-aside",
  isContainer: true,
  childLayout: "flex",
  childPositioning: "flow",
  maxChildren: 1,
  isRegion: true,
  isMovable: false,
};

export const mainDescriptor: ComponentDescriptor = {
  renderTag: "el-main",
  isContainer: true,
  childLayout: "flex",
  childPositioning: "flow",
  maxChildren: 1,
  isRegion: true,
  isMovable: false,
};

export const footerDescriptor: ComponentDescriptor = {
  renderTag: "el-footer",
  isContainer: true,
  childLayout: "flex",
  childPositioning: "flow",
  maxChildren: 1,
  isRegion: true,
  isMovable: false,
};

export default descriptor;
