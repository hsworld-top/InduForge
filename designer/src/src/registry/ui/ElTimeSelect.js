import ElTimeSelectComponent from './ElTimeSelect.vue';

export default {
  type: 'ElTimeSelect',
  name: '时间下拉',
  category: 'Element 组件',
  icon: 'time',
  thumbnail: null,
  tags: ['time', '表单'],
  description: 'Element Plus 时间下拉选择',
  component: ElTimeSelectComponent,

  defaultProps: {
    modelValue: '10:00',
    start: '08:30',
    end: '18:30',
    step: '00:30',
    placeholder: '选择时间',
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
    modelValue: { type: 'string', label: '时间', group: '组件属性', default: '10:00' },
    start: { type: 'string', label: '开始时间', group: '组件属性', default: '08:30' },
    end: { type: 'string', label: '结束时间', group: '组件属性', default: '18:30' },
    step: { type: 'string', label: '步长', group: '组件属性', default: '00:30' },
    placeholder: { type: 'string', label: '占位', group: '组件属性', default: '选择时间' },
  },

  eventsSchema: {
    change: { label: '变更' },
    blur: { label: '失去焦点' },
    focus: { label: '获得焦点' },
  },

  container: false,
  version: '1.0.0',
};
