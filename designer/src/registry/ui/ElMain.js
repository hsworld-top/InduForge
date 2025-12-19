import ElMainComponent from './ElMain.vue';

export default {
  type: 'ElMain',
  name: '主体',
  category: 'Element 组件',
  icon: 'layout',
  thumbnail: null,
  tags: ['main', 'layout'],
  description: 'Element Plus 主体',
  component: ElMainComponent,

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
