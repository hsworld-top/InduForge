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
      display: "flex",
      flexDirection: "column",
      alignItems: "stretch",
      position: "relative",
      boxSizing: "border-box",
      width: "100%",
      minHeight: "120px",
      gap: `${Number.isFinite(gap) ? Math.max(0, gap) : 12}px`,
    };
  },
  childStyle: () => ({
    width: "100%",
    minWidth: "0",
  }),
};

export default descriptor;

