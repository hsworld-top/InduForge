/**
 * Tabs 选项卡布局组件 Descriptor
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
  renderTag: "el-tabs",
  isContainer: true,
  childPositioning: "flow",
  childLayout: "none",
  defaultSize: { width: 360, height: 200 },
  renderKey: (node, ctx) => {
    const resolved = (ctx?.resolvedProps ?? {}) as Record<string, unknown>;
    const tabs = Array.isArray(node?.props?.tabs) ? node.props.tabs : [];
    const tabKey = tabs.map((item) => item?.name ?? item?.label ?? "").join("|");
    const activeName = node?.props?.activeName ?? resolved.activeName ?? "";
    return `${node?.id ?? ""}-${tabKey}-${String(activeName)}`;
  },
  propsFilter: (resolvedProps) => omitKeys(resolvedProps, ["tabs"]),
};

export default descriptor;

