import ElSliderComponent from './ElSlider.vue';

export default {
  type: 'ElSlider',
  name: '滑块',
  category: 'Element 组件',
  icon: 'slider',
  thumbnail: null,
  tags: ['slider', '表单'],
  description: 'Element Plus 滑块',
  component: ElSliderComponent,

  defaultProps: {
    modelValue: 30,
    min: 0,
    max: 100,
    step: 1,
    range: false,
    showStops: false,
    disabled: false,
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 260,
    height: 48,
    zIndex: 1,
  },

  propsSchema: {
    modelValue: { type: 'number', label: '当前值', group: '组件属性', default: 30 },
    min: { type: 'number', label: '最小值', group: '组件属性', default: 0 },
    max: { type: 'number', label: '最大值', group: '组件属性', default: 100 },
    step: { type: 'number', label: '步长', group: '组件属性', default: 1 },
    range: { type: 'boolean', label: '范围', group: '组件属性', default: false },
    showStops: { type: 'boolean', label: '显示间断点', group: '组件属性', default: false },
    disabled: { type: 'boolean', label: '禁用', group: '组件属性', default: false },
  },

  eventsSchema: {
    change: { label: '变更' },
    input: { label: '输入' },
  },

  container: false,
  version: '1.0.0',
};
