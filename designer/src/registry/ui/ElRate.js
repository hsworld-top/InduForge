import ElRateComponent from './ElRate.vue';

export default {
  type: 'ElRate',
  name: '评分',
  category: 'Element 组件',
  icon: 'star',
  thumbnail: null,
  tags: ['rate', '表单'],
  description: 'Element Plus 评分',
  component: ElRateComponent,

  defaultProps: {
    modelValue: 4,
    max: 5,
    allowHalf: false,
    showScore: false,
    texts: [],
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 200,
    height: 48,
    zIndex: 1,
  },

  propsSchema: {
    modelValue: { type: 'number', label: '分值', group: '组件属性', default: 4 },
    max: { type: 'number', label: '最大分', group: '组件属性', default: 5 },
    allowHalf: { type: 'boolean', label: '允许半星', group: '组件属性', default: false },
    showScore: { type: 'boolean', label: '显示分值', group: '组件属性', default: false },
    texts: { type: 'string', label: '文字提示 (JSON)', group: '组件属性', multiline: true, format: 'json', default: '[]' },
  },

  eventsSchema: {
    change: { label: '变更' },
  },

  container: false,
  version: '1.0.0',
};
