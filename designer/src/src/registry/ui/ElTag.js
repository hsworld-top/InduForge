import ElTagComponent from './ElTag.vue';

export default {
  type: 'ElTag',
  name: '标签',
  category: 'Element 组件',
  icon: 'tag',
  thumbnail: null,
  tags: ['tag'],
  description: 'Element Plus 标签',
  component: ElTagComponent,

  defaultProps: {
    text: '标签',
    type: 'success',
    effect: 'light',
    closable: false,
    hit: false,
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 120,
    height: 32,
    zIndex: 1,
  },

  propsSchema: {
    text: { type: 'string', label: '文本', group: '组件属性', default: '标签' },
    type: {
      type: 'enum',
      label: '类型',
      group: '组件属性',
      options: [
        { value: 'success', label: '成功' },
        { value: 'info', label: '信息' },
        { value: 'warning', label: '警告' },
        { value: 'danger', label: '危险' },
      ],
      default: 'success',
    },
    effect: {
      type: 'enum',
      label: '主题',
      group: '组件属性',
      options: [
        { value: 'light', label: '浅色' },
        { value: 'dark', label: '深色' },
        { value: 'plain', label: '朴素' },
      ],
      default: 'light',
    },
    closable: { type: 'boolean', label: '可关闭', group: '组件属性', default: false },
    hit: { type: 'boolean', label: '有边框', group: '组件属性', default: false },
  },

  eventsSchema: {},
  container: false,
  version: '1.0.0',
};
