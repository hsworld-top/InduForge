import ElColorPickerComponent from './ElColorPicker.vue';

export default {
  type: 'ElColorPicker',
  name: '取色器',
  category: 'Element 组件',
  icon: 'palette',
  thumbnail: null,
  tags: ['color', '表单'],
  description: 'Element Plus 取色器',
  component: ElColorPickerComponent,

  defaultProps: {
    modelValue: '#409EFF',
    showAlpha: false,
    predefine: ['#409EFF', '#67C23A', '#E6A23C', '#F56C6C'],
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 160,
    height: 40,
    zIndex: 1,
  },

  propsSchema: {
    modelValue: { type: 'string', label: '颜色', group: '组件属性', default: '#409EFF', format: 'color' },
    showAlpha: { type: 'boolean', label: 'Alpha 通道', group: '组件属性', default: false },
    predefine: {
      type: 'string',
      label: '预设颜色',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify(['#409EFF', '#67C23A', '#E6A23C', '#F56C6C'], null, 2),
    },
  },

  eventsSchema: {
    change: { label: '变更' },
    'active-change': { label: '悬浮改变' },
  },

  container: false,
  version: '1.0.0',
};
