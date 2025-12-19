import ElTransferComponent from './ElTransfer.vue';

export default {
  type: 'ElTransfer',
  name: '穿梭框',
  category: 'Element 组件',
  icon: 'arrow-right',
  thumbnail: null,
  tags: ['transfer', '表单'],
  description: 'Element Plus 穿梭框',
  component: ElTransferComponent,

  defaultProps: {
    modelValue: [1, 4],
    data: [
      { key: 1, label: '选项 1' },
      { key: 2, label: '选项 2' },
      { key: 3, label: '选项 3' },
      { key: 4, label: '选项 4' },
    ],
    titles: ['源列表', '目标列表'],
    filterable: true,
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 360,
    height: 220,
    zIndex: 1,
  },

  propsSchema: {
    modelValue: { type: 'string', label: '选中项 (JSON)', group: '组件属性', multiline: true, format: 'json', default: '[1,4]' },
    data: {
      type: 'string',
      label: '数据源 (JSON)',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify(
        [
          { key: 1, label: '选项 1' },
          { key: 2, label: '选项 2' },
          { key: 3, label: '选项 3' },
          { key: 4, label: '选项 4' },
        ],
        null,
        2,
      ),
    },
    titles: {
      type: 'string',
      label: '标题 (JSON)',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify(['源列表', '目标列表'], null, 2),
    },
    filterable: { type: 'boolean', label: '可搜索', group: '组件属性', default: true },
  },

  eventsSchema: {
    change: { label: '变更' },
  },

  container: false,
  version: '1.0.0',
};
