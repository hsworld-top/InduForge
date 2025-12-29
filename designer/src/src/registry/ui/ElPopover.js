import ElPopoverComponent from './ElPopover.vue';

export default {
  type: 'ElPopover',
  name: '气泡卡片',
  category: 'Element 组件',
  icon: 'info',
  thumbnail: null,
  tags: ['popover', 'feedback'],
  description: 'Element Plus 气泡卡片',
  component: ElPopoverComponent,

  defaultProps: {
    trigger: 'click',
    width: 200,
    placement: 'top',
    visible: true,
    title: '标题',
    content: '气泡卡片内容',
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 180,
    height: 48,
    zIndex: 1,
  },

  propsSchema: {
    trigger: {
      type: 'enum',
      label: '触发方式',
      group: '组件属性',
      options: [
        { value: 'hover', label: '悬停' },
        { value: 'click', label: '点击' },
        { value: 'focus', label: '聚焦' },
      ],
      default: 'click',
    },
    width: { type: 'number', label: '宽度', group: '组件属性', default: 200 },
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
        { value: 'right', label: '右' },
      ],
      default: 'top',
    },
    visible: { type: 'boolean', label: '默认可见', group: '组件属性', default: true },
    title: { type: 'string', label: '标题', group: '组件属性', default: '标题' },
    content: { type: 'string', label: '内容', group: '组件属性', default: '气泡卡片内容', multiline: true },
  },

  eventsSchema: {},
  container: false,
  version: '1.0.0',
};
