import ElDrawerComponent from './ElDrawer.vue';

export default {
  type: 'ElDrawer',
  name: '抽屉',
  category: 'Element 组件',
  icon: 'sidebar',
  thumbnail: null,
  tags: ['drawer', 'feedback'],
  description: 'Element Plus 抽屉',
  component: ElDrawerComponent,

  defaultProps: {
    modelValue: true,
    title: '抽屉',
    size: '320px',
    withHeader: true,
    modal: false,
    appendToBody: false,
    content: '抽屉内容',
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 320,
    height: 240,
    zIndex: 1,
  },

  propsSchema: {
    modelValue: { type: 'boolean', label: '默认可见', group: '组件属性', default: true },
    title: { type: 'string', label: '标题', group: '组件属性', default: '抽屉' },
    size: { type: 'string', label: '尺寸', group: '组件属性', default: '320px' },
    withHeader: { type: 'boolean', label: '显示头部', group: '组件属性', default: true },
    modal: { type: 'boolean', label: '遮罩', group: '组件属性', default: false },
    appendToBody: { type: 'boolean', label: '挂载到 body', group: '组件属性', default: false },
    content: { type: 'string', label: '内容', group: '组件属性', default: '抽屉内容', multiline: true },
  },

  eventsSchema: {
    open: { label: '打开' },
    close: { label: '关闭' },
  },

  container: true,
  version: '1.0.0',
};
