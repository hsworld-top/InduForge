import ElMenuComponent from './ElMenu.vue';

export default {
  type: 'ElMenu',
  name: '菜单',
  category: 'Element 组件',
  icon: 'menu',
  thumbnail: null,
  tags: ['menu', 'nav'],
  description: 'Element Plus 导航菜单',
  component: ElMenuComponent,

  defaultProps: {
    defaultActive: '1',
    mode: 'horizontal',
    ellipsis: true,
    collapse: false,
    items: [
      { index: '1', label: '菜单一' },
      {
        index: '2',
        label: '子菜单',
        children: [
          { index: '2-1', label: '选项 1' },
          { index: '2-2', label: '选项 2' },
        ],
      },
      { index: '3', label: '菜单三' },
    ],
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 320,
    height: 72,
    zIndex: 1,
  },

  propsSchema: {
    defaultActive: { type: 'string', label: '默认激活', group: '组件属性', default: '1' },
    mode: {
      type: 'enum',
      label: '模式',
      group: '组件属性',
      options: [
        { value: 'horizontal', label: '水平' },
        { value: 'vertical', label: '垂直' },
      ],
      default: 'horizontal',
    },
    ellipsis: { type: 'boolean', label: '省略', group: '组件属性', default: true },
    collapse: { type: 'boolean', label: '折叠', group: '组件属性', default: false },
    items: {
      type: 'string',
      label: '菜单项 (JSON)',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify(
        [
          { index: '1', label: '菜单一' },
          {
            index: '2',
            label: '子菜单',
            children: [
              { index: '2-1', label: '选项 1' },
              { index: '2-2', label: '选项 2' },
            ],
          },
          { index: '3', label: '菜单三' },
        ],
        null,
        2,
      ),
    },
  },

  eventsSchema: {
    select: { label: '选择' },
    open: { label: '展开' },
    close: { label: '收起' },
  },

  container: false,
  version: '1.0.0',
};
