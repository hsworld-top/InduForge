/**
 * VerticalLayout 垂直布局组件 Descriptor
 */

import type { ComponentDescriptor, DescriptorNode } from './registry'

export const descriptor: ComponentDescriptor = {
  /** 渲染为 div，通过 CSS flex column 实现垂直布局 */
  renderTag: 'div',

  isContainer: true,

  defaultStyle: {
    width: '100%',
    minHeight: '80px',
  },

  /**
   * 容器布局样式
   * @param {object} node - 当前节点
   * @returns {object}
   */
  containerStyle: (node: DescriptorNode) => {
    const props = node.props ?? {}
    const style: Record<string, string | number> = {
      display: 'flex',
      flexDirection: 'column',
      flexWrap: 'nowrap',
      justifyContent: String(props.justify ?? 'flex-start'),
      alignItems: String(props.align ?? 'stretch'),
      position: 'relative',
      boxSizing: 'border-box',
      width: '100%',
      minHeight: '80px',
    }
    if (props.gap !== undefined) {
      style.gap = `${props.gap}px`
    }
    if (props.showBorder === true) {
      style.border = '1px solid #dcdfe6'
      style.borderRadius = '4px'
    }
    return style
  },

  acceptChildren: true,

  /** 子项定位模式：流式 */
  childPositioning: 'flow',

  childFlowLayout: { grow: 1, shrink: 1, basis: '0%' },

  /**
   * 子项默认样式
   * @returns {object}
   */
  childStyle: () => ({
    width: '100%',
    height: '100%',
    minWidth: '0',
    minHeight: '0',
  }),

  /** 子项允许 resize */
  childResizable: true,

  isMovable: true,

  /** 子项布局类型：Flex 布局 */
  childLayout: 'flex',

  /** Flex 布局方向：垂直 */
  flexDirection: 'column',
}

export default descriptor
