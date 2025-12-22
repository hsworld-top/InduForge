import ElTooltipComponent from './ElTooltip.vue';

export default {
  type: 'ElTooltip',
  name: '文字提示',
  category: 'Element 组件',
  icon: 'info',
  thumbnail: null,
  tags: ['tooltip', 'feedback'],
  description: 'Element Plus 文字提示',
  component: ElTooltipComponent,

  defaultProps: {
    content: '提示内容',
    placement: 'top',
    visible: true,
    effect: 'dark',
    showAfter: 0,
    hideAfter: 0,
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 120,
    height: 48,
    zIndex: 1,
  },

  propsSchema: {
    content: { type: 'string', label: '内容', group: '组件属性', default: '提示内容', multiline: true },
    placement: {
      type: 'enum',
      label: '位置',
      group: '组件属性',
      options: [
        { value: 'top', label: '上' },
        { value: 'top-start', label: '上-左' },
        { value: 'top-end', label: '上-右' },
        { value: 'bottom', label: '下' },
        { value: 'bottom-start', label: '下-左' },
        { value: 'bottom-end', label: '下-右' },
        { value: 'left', label: '左' },
        { value: 'left-start', label: '左-上' },
        { value: 'left-end', label: '左-下' },
        { value: 'right', label: '右' },
        { value: 'right-start', label: '右-上' },
        { value: 'right-end', label: '右-下' },
      ],
      default: 'top',
    },
    visible: { type: 'boolean', label: '默认可见', group: '组件属性', default: true },
    effect: {
      type: 'enum',
      label: '主题',
      group: '组件属性',
      options: [
        { value: 'dark', label: '暗' },
        { value: 'light', label: '亮' },
      ],
      default: 'dark',
    },
    showAfter: { type: 'number', label: '显示延迟(ms)', group: '组件属性', default: 0 },
    hideAfter: { type: 'number', label: '隐藏延迟(ms)', group: '组件属性', default: 0 },
  },

  eventsSchema: {},

  container: false,
  version: '1.0.0',
};
