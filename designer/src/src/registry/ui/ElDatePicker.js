import ElDatePickerComponent from './ElDatePicker.vue';

export default {
  type: 'ElDatePicker',
  name: '日期选择',
  category: 'Element 组件',
  icon: 'calendar',
  thumbnail: null,
  tags: ['date', '表单'],
  description: 'Element Plus 日期选择器',
  component: ElDatePickerComponent,

  defaultProps: {
    modelValue: '2025-01-01',
    type: 'date',
    placeholder: '选择日期',
    startPlaceholder: '开始日期',
    endPlaceholder: '结束日期',
    format: 'YYYY-MM-DD',
    valueFormat: 'YYYY-MM-DD',
    clearable: true,
    rangeSeparator: '-',
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 260,
    height: 40,
    zIndex: 1,
  },

  propsSchema: {
    modelValue: { type: 'string', label: '日期值', group: '组件属性', default: '2025-01-01' },
    type: {
      type: 'enum',
      label: '类型',
      group: '组件属性',
      options: [
        { value: 'date', label: '日期' },
        { value: 'datetime', label: '日期时间' },
        { value: 'daterange', label: '日期范围' },
        { value: 'datetimerange', label: '日期时间范围' },
      ],
      default: 'date',
    },
    placeholder: { type: 'string', label: '占位', group: '组件属性', default: '选择日期' },
    startPlaceholder: { type: 'string', label: '开始占位', group: '组件属性', default: '开始日期' },
    endPlaceholder: { type: 'string', label: '结束占位', group: '组件属性', default: '结束日期' },
    format: { type: 'string', label: '显示格式', group: '组件属性', default: 'YYYY-MM-DD' },
    valueFormat: { type: 'string', label: '值格式', group: '组件属性', default: 'YYYY-MM-DD' },
    clearable: { type: 'boolean', label: '可清空', group: '组件属性', default: true },
    rangeSeparator: { type: 'string', label: '范围分隔符', group: '组件属性', default: '-' },
  },

  eventsSchema: {
    change: { label: '变更' },
    blur: { label: '失去焦点' },
    focus: { label: '获得焦点' },
  },

  container: false,
  version: '1.0.0',
};
