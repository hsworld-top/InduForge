/**
 * Collapse 折叠布局组件 Descriptor
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
  renderTag: "el-collapse",
  isContainer: true,
  childPositioning: "flow",
  childLayout: "none",
  defaultSize: { width: 360, height: 200 },
  propsFilter: (resolvedProps) => omitKeys(resolvedProps, ["items"]),
};

export default descriptor;

