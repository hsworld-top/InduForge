import ElTimePickerComponent from './ElTimePicker.vue';

export default {
  type: 'ElTimePicker',
  name: '时间选择',
  category: 'Element 组件',
  icon: 'time',
  thumbnail: null,
  tags: ['time', '表单'],
  description: 'Element Plus 时间选择器',
  component: ElTimePickerComponent,

  defaultProps: {
    modelValue: '12:00:00',
    isRange: false,
    startPlaceholder: '开始时间',
    endPlaceholder: '结束时间',
    placeholder: '选择时间',
    format: 'HH:mm:ss',
    valueFormat: 'HH:mm:ss',
    arrowControl: true,
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 240,
    height: 40,
    zIndex: 1,
  },

  propsSchema: {
    modelValue: { type: 'string', label: '时间值', group: '组件属性', default: '12:00:00' },
    isRange: { type: 'boolean', label: '范围选择', group: '组件属性', default: false },
    startPlaceholder: { type: 'string', label: '开始占位', group: '组件属性', default: '开始时间' },
    endPlaceholder: { type: 'string', label: '结束占位', group: '组件属性', default: '结束时间' },
    placeholder: { type: 'string', label: '占位', group: '组件属性', default: '选择时间' },
    format: { type: 'string', label: '显示格式', group: '组件属性', default: 'HH:mm:ss' },
    valueFormat: { type: 'string', label: '值格式', group: '组件属性', default: 'HH:mm:ss' },
    arrowControl: { type: 'boolean', label: '箭头控制', group: '组件属性', default: true },
  },

  eventsSchema: {
    change: { label: '变更' },
    blur: { label: '失去焦点' },
    focus: { label: '获得焦点' },
  },

  container: false,
  version: '1.0.0',
};
