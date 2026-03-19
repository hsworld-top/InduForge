/**
 * HorizontalLayout 水平布局组件 Descriptor
 * 渲染标签、容器样式、子项策略等元数据
 *
 * @type {import('../../components/registry.js').ComponentDescriptor}
 */
export const descriptor = {
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
   * @param {Object} node - 当前节点
   * @returns {Object} CSS 样式对象
   */
  containerStyle: (node) => {
    const style = {
      display: "flex",
      flexDirection: "row",
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

  /** 接受任意子组件 */
  acceptChildren: true,

  /** 子项定位模式：流式布局 */
  childPositioning: "flow",

  /** 子项默认 flowLayout：等分扩展 */
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

  /** 可自由拖拽移动 */
  isMovable: true,
};

export default descriptor;
