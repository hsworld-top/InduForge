import ElFooterComponent from './ElFooter.vue';

export default {
  type: 'ElFooter',
  name: '底部',
  category: 'Element 组件',
  icon: 'layout',
  thumbnail: null,
  tags: ['footer', 'layout'],
  description: 'Element Plus 底部',
  component: ElFooterComponent,

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
