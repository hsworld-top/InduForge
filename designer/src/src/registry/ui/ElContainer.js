import ElContainerComponent from './ElContainer.vue';

export default {
  type: 'ElContainer',
  name: '容器',
  category: 'Element 组件',
  icon: 'layout',
  thumbnail: null,
  tags: ['container', 'layout'],
  description: 'Element Plus 容器布局',
  component: ElContainerComponent,

  defaultProps: {},

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 320,
    height: 200,
    zIndex: 1,
  },

  propsSchema: {},
  eventsSchema: {},
  container: true,
  version: '1.0.0',
};
