/**
 * Col Component - 列容器
 * 参考 OpenTiny 的行列布局
 */
export default {
  type: 'Col',
  name: '列容器',
  category: 'layout',
  icon: 'list',
  defaultProps: {
    span: 12,
    offset: 0,
  },
  defaultStyle: {
    position: 'relative',
    flex: '0 0 50%',
    maxWidth: '50%',
    padding: '10px',
    backgroundColor: 'transparent',
    border: '1px dashed #dcdfe6',
    minHeight: '50px',
  },
  propsSchema: {
    span: {
      type: 'number',
      label: '栅格占据列数',
      default: 12,
      min: 1,
      max: 24,
    },
    offset: {
      type: 'number',
      label: '栅格左侧间隔',
      default: 0,
      min: 0,
      max: 24,
    },
  },
}
