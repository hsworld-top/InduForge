/**
 * HorizontalLayout 水平布局组件 Descriptor
 */

import type { ComponentDescriptor, DescriptorNode } from "./registry";

export const descriptor: ComponentDescriptor = {
  /** 渲染为 div，通过 CSS flex 实现水平布局 */
  renderTag: "div",

  /** 是容器 */
  isContainer: true,

  /** 容器默认样式 */
  defaultStyle: {
    width: "100%",
    minHeight: "80px",
  },

  /**
   * 容器布局样式生成函数
   * @param {object} node - 当前节点
   * @returns {object} CSS 样式对象
   */
  containerStyle: (node: DescriptorNode) => {
    const props = node.props ?? {};
    const style: Record<string, string | number> = {
      display: "flex",
      flexDirection: "row",
      flexWrap: "nowrap",
      justifyContent: String(props.justify ?? "flex-start"),
      alignItems: String(props.align ?? "stretch"),
      position: "relative",
      boxSizing: "border-box",
      width: "100%",
      minHeight: "80px",
    };
    if (props.gap !== undefined) {
      style.gap = `${props.gap}px`;
    }
    if (props.showBorder === true) {
      style.border = "1px solid #dcdfe6";
      style.borderRadius = "4px";
    }
    return style;
  },

  /** 接受任意子组件 */
  acceptChildren: true,

  /** 子项定位模式：流式布局 */
  childPositioning: "flow",

  /** 子项默认 flowLayout：等分扩展 */
  childFlowLayout: { grow: 1, shrink: 1, basis: "0%" },

  /**
   * 子项默认样式
   * @returns {object}
   */
  childStyle: () => ({
    width: "100%",
    height: "100%",
    minWidth: "0",
    minHeight: "0",
  }),

  /** 子项允许 resize */
  childResizable: true,

  /** 可自由拖拽移动 */
  isMovable: true,

  /** 子项布局类型：Flex 布局 */
  childLayout: "flex",

  /** Flex 布局方向：水平 */
  flexDirection: "row",
};

export default descriptor;
