/**
 * Divider Component - 分割线
 * 参考 OpenTiny 的分割线组件
 */
export default {
  type: 'Divider',
  name: '分割线',
  category: 'basic',
  icon: 'minus',
  defaultProps: {
    direction: 'horizontal',
    contentPosition: 'center',
    text: '',
  },
  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 300,
    height: 1,
    backgroundColor: '#dcdfe6',
    margin: '16px 0',
  },
  propsSchema: {
    direction: {
      type: 'select',
      label: '方向',
      default: 'horizontal',
      options: [
        { label: '水平', value: 'horizontal' },
        { label: '垂直', value: 'vertical' },
      ],
    },
    contentPosition: {
      type: 'select',
      label: '文字位置',
      default: 'center',
      options: [
        { label: '左侧', value: 'left' },
        { label: '居中', value: 'center' },
        { label: '右侧', value: 'right' },
      ],
    },
    text: {
      type: 'string',
      label: '文字内容',
      default: '',
    },
  },
}
