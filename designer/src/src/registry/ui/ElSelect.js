import ElSelectComponent from './ElSelect.vue';

export default {
  type: 'ElSelect',
  name: '选择器',
  category: 'Element 组件',
  icon: 'select',
  thumbnail: null,
  tags: ['select', '表单'],
  description: 'Element Plus 选择器',
  component: ElSelectComponent,

  defaultProps: {
    modelValue: '',
    placeholder: '请选择',
    multiple: false,
    clearable: true,
    filterable: true,
    options: [
      { label: '选项一', value: '1' },
      { label: '选项二', value: '2' },
      { label: '选项三', value: '3' },
    ],
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
    modelValue: { type: 'string', label: '选中值', group: '组件属性', default: '' },
    placeholder: { type: 'string', label: '占位', group: '组件属性', default: '请选择' },
    multiple: { type: 'boolean', label: '多选', group: '组件属性', default: false },
    clearable: { type: 'boolean', label: '可清空', group: '组件属性', default: true },
    filterable: { type: 'boolean', label: '可搜索', group: '组件属性', default: true },
    options: {
      type: 'string',
      label: '选项',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify(
        [
          { label: '选项一', value: '1' },
          { label: '选项二', value: '2' },
          { label: '选项三', value: '3' },
        ],
        null,
        2,
      ),
      description: '数组 { label, value }',
    },
  },

  eventsSchema: {
    change: { label: '变更' },
    'visible-change': { label: '下拉显示变化' },
    'remove-tag': { label: '移除标签' },
    clear: { label: '清空' },
    blur: { label: '失去焦点' },
    focus: { label: '获得焦点' },
  },

  container: false,
  version: '1.0.0',
};
