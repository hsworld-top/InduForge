import ElCollapseComponent from './ElCollapse.vue';

export default {
  type: 'ElCollapse',
  name: '折叠面板',
  category: 'Element 组件',
  icon: 'list',
  thumbnail: null,
  tags: ['collapse'],
  description: 'Element Plus 折叠面板',
  component: ElCollapseComponent,

  defaultProps: {
    modelValue: ['1'],
    items: [
      { name: '1', title: '面板 1', content: '内容 1' },
      { name: '2', title: '面板 2', content: '内容 2' },
    ],
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 260,
    height: 180,
    zIndex: 1,
  },

  propsSchema: {
    modelValue: { type: 'string', label: '默认展开', group: '组件属性', default: '["1"]', format: 'json' },
    items: {
      type: 'string',
      label: '面板列表',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify(
        [
          { name: '1', title: '面板 1', content: '内容 1' },
          { name: '2', title: '面板 2', content: '内容 2' },
        ],
        null,
        2,
      ),
    },
  },

  eventsSchema: {},
  container: false,
  version: '1.0.0',
};
