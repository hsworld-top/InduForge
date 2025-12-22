import ElAsideComponent from './ElAside.vue';

export default {
  type: 'ElAside',
  name: '侧边栏',
  category: 'Element 组件',
  icon: 'layout',
  thumbnail: null,
  tags: ['aside', 'layout'],
  description: 'Element Plus 侧边栏',
  component: ElAsideComponent,

  defaultProps: {
    width: '160px',
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 180,
    height: 200,
    zIndex: 1,
  },

  propsSchema: {
    width: { type: 'string', label: '宽度', group: '组件属性', default: '160px' },
  },

  eventsSchema: {},
  container: true,
  version: '1.0.0',
};
