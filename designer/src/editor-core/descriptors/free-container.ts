/**
 * FreeContainer 页面根自由布局容器 Descriptor
 */

import type { ComponentDescriptor } from "./registry";

export const descriptor: ComponentDescriptor = {
  renderTag: "div",
  isContainer: true,
  childPositioning: "absolute",
  childLayout: "free",
  defaultSize: { width: 360, height: 200 },
};

export default descriptor;

