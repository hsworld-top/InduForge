/**
 * Button 按钮组件 Descriptor
 */

import type { ComponentDescriptor } from "./registry";
import { normalizeButtonRenderProps } from "./button-props";

export const descriptor: ComponentDescriptor = {
  /** 渲染为 Element Plus el-button */
  renderTag: "el-button",

  /** 非容器 */
  isContainer: false,

  /** 组件默认样式 */
  defaultStyle: {},

  /** 不接受子组件 */
  acceptChildren: false,

  /** 可自由拖拽移动 */
  isMovable: true,

  /** 子项 resize 策略：不涉及（非容器） */
  childResizable: true,

  /** 显示内容生成函数 */
  displayContent: (node, resolvedProps) => String(resolvedProps?.text ?? node?.label ?? "按钮"),

  /** 过滤/转换传给 renderTag 的 props */
  propsFilter: (resolvedProps) => normalizeButtonRenderProps(resolvedProps ?? {}),

  /** 默认尺寸 */
  defaultSize: { width: 120, height: 36 },
};

export default descriptor;
