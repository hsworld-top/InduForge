import ElDropdownComponent from './ElDropdown.vue';

export default {
  type: 'ElDropdown',
  name: '下拉菜单',
  category: 'Element 组件',
  icon: 'arrow-down',
  thumbnail: null,
  tags: ['dropdown', 'nav'],
  description: 'Element Plus 下拉菜单',
  component: ElDropdownComponent,

  defaultProps: {
    trigger: 'click',
    text: '下拉菜单',
    items: [
      { command: 'A', label: '选项 A' },
      { command: 'B', label: '选项 B' },
      { command: 'C', label: '选项 C', divided: true },
    ],
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 180,
    height: 48,
    zIndex: 1,
  },

  propsSchema: {
    trigger: {
      type: 'enum',
      label: '触发方式',
      group: '组件属性',
      options: [
        { value: 'hover', label: '悬停' },
        { value: 'click', label: '点击' },
        { value: 'contextmenu', label: '右键' },
      ],
      default: 'click',
    },
    text: { type: 'string', label: '按钮文本', group: '组件属性', default: '下拉菜单' },
    items: {
      type: 'string',
      label: '菜单项 (JSON)',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify(
        [
          { command: 'A', label: '选项 A' },
          { command: 'B', label: '选项 B' },
          { command: 'C', label: '选项 C', divided: true },
        ],
        null,
        2,
      ),
    },
  },

  eventsSchema: {
    command: { label: '命令选择' },
    'visible-change': { label: '显示变化' },
  },

  container: false,
  version: '1.0.0',
};
