import ElInputNumberComponent from './ElInputNumber.vue';

export default {
  type: 'ElInputNumber',
  name: '计数器',
  category: 'Element 组件',
  icon: 'numbers',
  thumbnail: null,
  tags: ['input-number', '表单'],
  description: 'Element Plus 计数器',
  component: ElInputNumberComponent,

  defaultProps: {
    modelValue: 1,
    min: 0,
    max: 10,
    step: 1,
    precision: null,
    controls: true,
    disabled: false,
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 200,
    height: 40,
    zIndex: 1,
  },

  propsSchema: {
    modelValue: { type: 'number', label: '当前值', group: '组件属性', default: 1 },
    min: { type: 'number', label: '最小值', group: '组件属性', default: 0 },
    max: { type: 'number', label: '最大值', group: '组件属性', default: 10 },
    step: { type: 'number', label: '步长', group: '组件属性', default: 1 },
    precision: { type: 'number', label: '精度', group: '组件属性', default: null },
    controls: { type: 'boolean', label: '显示控制按钮', group: '组件属性', default: true },
    disabled: { type: 'boolean', label: '禁用', group: '组件属性', default: false },
  },

  eventsSchema: {
    change: { label: '变更' },
    input: { label: '输入' },
    focus: { label: '获得焦点' },
    blur: { label: '失去焦点' },
  },

  container: false,
  version: '1.0.0',
};
