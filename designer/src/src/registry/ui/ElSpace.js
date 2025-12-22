import ElSpaceComponent from './ElSpace.vue';

export default {
  type: 'ElSpace',
  name: '间距',
  category: 'Element 组件',
  icon: 'layout',
  thumbnail: null,
  tags: ['space', 'layout'],
  description: 'Element Plus 间距容器',
  component: ElSpaceComponent,

  defaultProps: {
    wrap: true,
    size: 8,
    alignment: 'center',
    direction: 'horizontal',
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 240,
    height: 72,
    zIndex: 1,
  },

  propsSchema: {
    wrap: { type: 'boolean', label: '自动换行', group: '组件属性', default: true },
    size: { type: 'number', label: '间距', group: '组件属性', default: 8 },
    alignment: {
      type: 'enum',
      label: '对齐',
      group: '组件属性',
      options: [
        { value: 'start', label: '左/上' },
        { value: 'center', label: '居中' },
        { value: 'end', label: '右/下' },
        { value: 'baseline', label: '基线' },
      ],
      default: 'center',
    },
    direction: {
      type: 'enum',
      label: '方向',
      group: '组件属性',
      options: [
        { value: 'horizontal', label: '横向' },
        { value: 'vertical', label: '纵向' },
      ],
      default: 'horizontal',
    },
  },

  eventsSchema: {},
  container: false,
  version: '1.0.0',
};
