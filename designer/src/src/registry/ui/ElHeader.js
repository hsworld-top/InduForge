import ElHeaderComponent from './ElHeader.vue';

export default {
  type: 'ElHeader',
  name: '头部',
  category: 'Element 组件',
  icon: 'layout',
  thumbnail: null,
  tags: ['header', 'layout'],
  description: 'Element Plus 头部',
  component: ElHeaderComponent,

  defaultProps: {
    height: '48px',
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 320,
    height: 48,
    zIndex: 1,
  },

  propsSchema: {
    height: { type: 'string', label: '高度', group: '组件属性', default: '48px' },
  },

  eventsSchema: {},
  container: false,
  version: '1.0.0',
};
