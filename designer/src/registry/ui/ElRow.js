import ElRowComponent from './ElRow.vue';

export default {
  type: 'ElRow',
  name: '行',
  category: 'Element 组件',
  icon: 'grid',
  thumbnail: null,
  tags: ['row', 'layout'],
  description: 'Element Plus 栅格行',
  component: ElRowComponent,

  defaultProps: {
    gutter: 12,
    justify: 'start',
    align: 'top',
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 320,
    height: 96,
    zIndex: 1,
  },

  propsSchema: {
    gutter: { type: 'number', label: '间距', group: '组件属性', default: 12 },
    justify: {
      type: 'enum',
      label: '对齐',
      group: '组件属性',
      options: [
        { value: 'start', label: '左' },
        { value: 'center', label: '居中' },
        { value: 'end', label: '右' },
        { value: 'space-around', label: '两侧留白' },
        { value: 'space-between', label: '两端对齐' },
        { value: 'space-evenly', label: '等分' },
      ],
      default: 'start',
    },
    align: {
      type: 'enum',
      label: '垂直对齐',
      group: '组件属性',
      options: [
        { value: 'top', label: '顶部' },
        { value: 'middle', label: '居中' },
        { value: 'bottom', label: '底部' },
      ],
      default: 'top',
    },
  },

  eventsSchema: {},
  container: true,
  version: '1.0.0',
};
