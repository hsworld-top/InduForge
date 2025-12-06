/**
 * Row Component - 行容器
 * 参考 OpenTiny 的行列布局
 */
export default {
  type: 'Row',
  name: '行容器',
  category: 'layout',
  icon: 'menu',
  defaultProps: {
    gutter: 0,
    justify: 'start',
    align: 'top',
  },
  defaultStyle: {
    position: 'relative',
    left: 0,
    top: 0,
    width: 400,
    height: 100,
    display: 'flex',
    flexDirection: 'row',
    gap: '0px',
    padding: '10px',
    backgroundColor: 'transparent',
    border: '1px dashed #dcdfe6',
  },
  propsSchema: {
    gutter: {
      type: 'number',
      label: '栅格间隔',
      default: 0,
      min: 0,
      max: 100,
    },
    justify: {
      type: 'select',
      label: '水平排列',
      default: 'start',
      options: [
        { label: '左对齐', value: 'start' },
        { label: '居中', value: 'center' },
        { label: '右对齐', value: 'end' },
        { label: '两端对齐', value: 'space-between' },
        { label: '均匀分布', value: 'space-around' },
      ],
    },
    align: {
      type: 'select',
      label: '垂直对齐',
      default: 'top',
      options: [
        { label: '顶部', value: 'top' },
        { label: '居中', value: 'middle' },
        { label: '底部', value: 'bottom' },
      ],
    },
  },
}
