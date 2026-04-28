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
    const style: Record<string, string | number> = {
      display: "grid",
      gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
      alignItems: "start",
      position: "relative",
      boxSizing: "border-box",
      width: "100%",
      minHeight: "120px",
      gap: `${Number.isFinite(gap) ? Math.max(0, gap) : 0}px`,
    };
    if (props.showBorder === true) {
      style.border = "1px solid #dcdfe6";
      style.borderRadius = "4px";
    }
    return style;
  },
  childStyle: () => ({
    minWidth: "0",
    width: "100%",
  }),
};

export default descriptor;
