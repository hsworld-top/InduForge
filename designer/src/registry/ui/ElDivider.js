import ElDividerComponent from './ElDivider.vue';

export default {
  type: 'ElDivider',
  name: '分割线',
  category: 'Element 组件',
  icon: 'minus',
  thumbnail: null,
  tags: ['divider'],
  description: 'Element Plus 分割线',
  component: ElDividerComponent,

  defaultProps: {
    direction: 'horizontal',
    contentPosition: 'center',
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 240,
    height: 32,
    zIndex: 1,
  },

  propsSchema: {
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
    contentPosition: {
      type: 'enum',
      label: '内容位置',
      group: '组件属性',
      options: [
        { value: 'left', label: '左' },
        { value: 'center', label: '中' },
        { value: 'right', label: '右' },
      ],
      default: 'center',
    },
  },

  eventsSchema: {},
  container: false,
  version: '1.0.0',
};
