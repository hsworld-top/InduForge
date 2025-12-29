import ElSkeletonComponent from './ElSkeleton.vue';

export default {
  type: 'ElSkeleton',
  name: '骨架屏',
  category: 'Element 组件',
  icon: 'loading',
  thumbnail: null,
  tags: ['skeleton'],
  description: 'Element Plus 骨架屏',
  component: ElSkeletonComponent,

  defaultProps: {
    loading: true,
    rows: 3,
    animated: true,
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 320,
    height: 120,
    zIndex: 1,
  },

  propsSchema: {
    loading: { type: 'boolean', label: '加载中', group: '组件属性', default: true },
    rows: { type: 'number', label: '行数', group: '组件属性', default: 3 },
    animated: { type: 'boolean', label: '动画', group: '组件属性', default: true },
  },

  eventsSchema: {},
  container: false,
  version: '1.0.0',
};
