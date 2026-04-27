/**
 * FormLayout 表单布局组件 Descriptor
 */

import type { ComponentDescriptor, DescriptorNode } from "./registry";

export const descriptor: ComponentDescriptor = {
  renderTag: "div",
  isContainer: true,
  childLayout: "flex",
  childPositioning: "flow",
  childFlowLayout: { grow: 0, shrink: 0, basis: "auto" },
  defaultSize: { width: 360, height: 280 },
  containerStyle: (node: DescriptorNode) => {
    const props = node.props ?? {};
    const gap = Number(props.itemGap);
    return {
      display: "grid",
      gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
      alignItems: "start",
      position: "relative",
      boxSizing: "border-box",
      width: "100%",
      minHeight: "120px",
      gap: `${Number.isFinite(gap) ? Math.max(0, gap) : 12}px`,
    };
  },
  childStyle: () => ({
    minWidth: "0",
    width: "100%",
  }),
};

export default descriptor;
