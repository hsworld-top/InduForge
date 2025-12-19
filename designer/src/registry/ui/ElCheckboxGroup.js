import ElCheckboxGroupComponent from './ElCheckboxGroup.vue';

export default {
  type: 'ElCheckboxGroup',
  name: '多选组',
  category: 'Element 组件',
  icon: 'checkbox',
  thumbnail: null,
  tags: ['checkbox', 'form'],
  description: 'Element Plus 多选组',
  component: ElCheckboxGroupComponent,

  defaultProps: {
    modelValue: ['A'],
    size: '',
    disabled: false,
    options: [
      { label: '选项 A', value: 'A' },
      { label: '选项 B', value: 'B' },
      { label: '选项 C', value: 'C' },
    ],
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 220,
    height: 60,
    zIndex: 1,
  },

  propsSchema: {
    modelValue: { type: 'string', label: '选中值 (JSON)', group: '组件属性', default: '["A"]', format: 'json' },
    size: {
      type: 'enum',
      label: '尺寸',
      group: '组件属性',
      options: [
        { value: 'large', label: '大' },
        { value: 'default', label: '默认' },
        { value: 'small', label: '小' },
        { value: '', label: '继承' },
      ],
      default: '',
    },
    disabled: { type: 'boolean', label: '禁用', group: '组件属性', default: false },
    options: {
      type: 'string',
      label: '选项 (JSON)',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify(
        [
          { label: '选项 A', value: 'A' },
          { label: '选项 B', value: 'B' },
          { label: '选项 C', value: 'C' },
        ],
        null,
        2,
      ),
    },
  },

  eventsSchema: {
    change: { label: '变更' },
  },

  container: false,
  version: '1.0.0',
};
