import ElCascaderComponent from './ElCascader.vue';

export default {
  type: 'ElCascader',
  name: '级联选择',
  category: 'Element 组件',
  icon: 'tree',
  thumbnail: null,
  tags: ['cascader', '表单'],
  description: 'Element Plus 级联选择',
  component: ElCascaderComponent,

  defaultProps: {
    modelValue: [],
    options: [
      { label: '一级 1', value: '1', children: [{ label: '二级 1-1', value: '1-1' }] },
      { label: '一级 2', value: '2', children: [{ label: '二级 2-1', value: '2-1' }] },
    ],
    cascaderProps: { checkStrictly: false, multiple: false },
    clearable: true,
    filterable: true,
    placeholder: '请选择',
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
    modelValue: { type: 'string', label: '选中值', group: '组件属性', default: '[]', format: 'json', multiline: true },
    options: {
      type: 'string',
      label: '选项 (JSON)',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify(
        [
          { label: '一级 1', value: '1', children: [{ label: '二级 1-1', value: '1-1' }] },
          { label: '一级 2', value: '2', children: [{ label: '二级 2-1', value: '2-1' }] },
        ],
        null,
        2,
      ),
      description: '树形选项',
    },
    cascaderProps: {
      type: 'string',
      label: '选项配置',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify({ checkStrictly: false, multiple: false }, null, 2),
      description: '如 { checkStrictly, multiple }',
    },
    clearable: { type: 'boolean', label: '可清空', group: '组件属性', default: true },
    filterable: { type: 'boolean', label: '可搜索', group: '组件属性', default: true },
    placeholder: { type: 'string', label: '占位', group: '组件属性', default: '请选择' },
  },

  eventsSchema: {
    change: { label: '变更' },
    'expand-change': { label: '展开变化' },
    blur: { label: '失去焦点' },
    focus: { label: '获得焦点' },
    'visible-change': { label: '下拉显示变化' },
  },

  container: false,
  version: '1.0.0',
};
