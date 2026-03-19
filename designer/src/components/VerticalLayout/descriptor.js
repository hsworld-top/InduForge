/**
 * VerticalLayout 垂直布局组件 Descriptor
 *
 * @type {import('../../components/registry.js').ComponentDescriptor}
 */
export const descriptor = {
  /** 渲染为 div，通过 CSS flex column 实现垂直布局 */
  renderTag: "div",

  isContainer: true,

  defaultStyle: {
    width: "100%",
    minHeight: "80px",
  },

  /**
   * 容器布局样式
   * @param {Object} node - 当前节点
   * @returns {Object}
   */
  containerStyle: (node) => {
    const style = {
      display: "flex",
      flexDirection: "column",
      flexWrap: "nowrap",
      justifyContent: node.props?.justify || "flex-start",
      alignItems: node.props?.align || "stretch",
      position: "relative",
      boxSizing: "border-box",
      width: "100%",
      minHeight: "80px",
    };
    if (node.props?.gap !== undefined) {
      style.gap = `${node.props.gap}px`;
    }
    return style;
  },

  acceptChildren: true,

  /** 子项定位模式：流式 */
  childPositioning: "flow",

  childFlowLayout: { grow: 1, shrink: 1, basis: "0%" },

  /**
   * 子项默认样式
   * @returns {Object}
   */
  childStyle: () => ({
    width: "100%",
    height: "100%",
    minWidth: "0",
    minHeight: "0",
  }),

  /** 子项不允许 resize */
  childResizable: false,

  isMovable: true,
};

export default descriptor;
