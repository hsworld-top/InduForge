/**
 * Link Component - 链接
 * 参考 OpenTiny 的链接组件
 */
export default {
  type: 'Link',
  name: '链接',
  category: 'basic',
  icon: 'link',
  defaultProps: {
    text: '链接文本',
    href: '#',
    target: '_self',
    underline: true,
    type: 'default',
  },
  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    fontSize: '14px',
    color: '#5e7ce0',
    cursor: 'pointer',
  },
  propsSchema: {
    text: {
      type: 'string',
      label: '链接文本',
      default: '链接文本',
    },
    href: {
      type: 'string',
      label: '链接地址',
      default: '#',
    },
    target: {
      type: 'select',
      label: '打开方式',
      default: '_self',
      options: [
        { label: '当前窗口', value: '_self' },
        { label: '新窗口', value: '_blank' },
      ],
    },
    underline: {
      type: 'boolean',
      label: '下划线',
      default: true,
    },
    type: {
      type: 'select',
      label: '类型',
      default: 'default',
      options: [
        { label: '默认', value: 'default' },
        { label: '主要', value: 'primary' },
        { label: '成功', value: 'success' },
        { label: '警告', value: 'warning' },
        { label: '危险', value: 'danger' },
      ],
    },
  },
}
