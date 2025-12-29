import ElColComponent from './ElCol.vue';

export default {
  type: 'ElCol',
  name: '列',
  category: 'Element 组件',
  icon: 'grid',
  thumbnail: null,
  tags: ['col', 'layout'],
  description: 'Element Plus 栅格列',
  component: ElColComponent,

  defaultProps: {
    span: 12,
    offset: 0,
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 200,
    height: 64,
    zIndex: 1,
  },

  propsSchema: {
    span: { type: 'number', label: '栅格', group: '组件属性', default: 12 },
    offset: { type: 'number', label: '偏移', group: '组件属性', default: 0 },
  },

  eventsSchema: {},
  container: true,
  version: '1.0.0',
};
