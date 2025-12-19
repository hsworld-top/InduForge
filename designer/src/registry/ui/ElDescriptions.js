import ElDescriptionsComponent from './ElDescriptions.vue';

export default {
  type: 'ElDescriptions',
  name: '描述列表',
  category: 'Element 组件',
  icon: 'list',
  thumbnail: null,
  tags: ['descriptions'],
  description: 'Element Plus 描述列表',
  component: ElDescriptionsComponent,

  defaultProps: {
    title: '信息',
    border: true,
    column: 2,
    items: [
      { label: '用户名', value: 'admin' },
      { label: '角色', value: 'Editor' },
      { label: '邮箱', value: 'admin@example.com' },
    ],
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 360,
    height: 200,
    zIndex: 1,
  },

  propsSchema: {
    title: { type: 'string', label: '标题', group: '组件属性', default: '信息' },
    border: { type: 'boolean', label: '边框', group: '组件属性', default: true },
    column: { type: 'number', label: '列数', group: '组件属性', default: 2 },
    items: {
      type: 'string',
      label: '条目 (JSON)',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify(
        [
          { label: '用户名', value: 'admin' },
          { label: '角色', value: 'Editor' },
          { label: '邮箱', value: 'admin@example.com' },
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
