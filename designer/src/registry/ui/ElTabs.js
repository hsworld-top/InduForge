import ElTabsComponent from './ElTabs.vue';

export default {
  type: 'ElTabs',
  name: '选项卡',
  category: 'Element 组件',
  icon: 'tabs',
  thumbnail: null,
  tags: ['tabs', 'nav'],
  description: 'Element Plus 选项卡',
  component: ElTabsComponent,

  defaultProps: {
    modelValue: '1',
    type: '',
    tabPosition: 'top',
    closable: false,
    addable: false,
    editable: false,
    stretch: false,
    panes: [
      { label: 'Tab 1', name: '1', content: 'Tab 1 内容' },
      { label: 'Tab 2', name: '2', content: 'Tab 2 内容' },
    ],
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 300,
    height: 180,
    zIndex: 1,
  },

  propsSchema: {
    modelValue: { type: 'string', label: '当前选中', group: '组件属性', default: '1' },
    type: {
      type: 'enum',
      label: '类型',
      group: '组件属性',
      options: [
        { value: '', label: '默认' },
        { value: 'card', label: '卡片' },
        { value: 'border-card', label: '卡片边框' },
      ],
      default: '',
    },
    tabPosition: {
      type: 'enum',
      label: '选项卡位置',
      group: '组件属性',
      options: [
        { value: 'top', label: '顶部' },
        { value: 'right', label: '右侧' },
        { value: 'bottom', label: '底部' },
        { value: 'left', label: '左侧' },
      ],
      default: 'top',
    },
    closable: { type: 'boolean', label: '可关闭', group: '组件属性', default: false },
    addable: { type: 'boolean', label: '可新增', group: '组件属性', default: false },
    editable: { type: 'boolean', label: '可编辑', group: '组件属性', default: false },
    stretch: { type: 'boolean', label: '拉伸充满', group: '组件属性', default: false },
    panes: {
      type: 'string',
      label: '选项卡列表 (JSON)',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify(
        [
          { label: 'Tab 1', name: '1', content: 'Tab 1 内容' },
          { label: 'Tab 2', name: '2', content: 'Tab 2 内容' },
        ],
        null,
        2,
      ),
    },
  },

  eventsSchema: {
    'tab-click': { label: '标签点击' },
    'tab-remove': { label: '移除标签' },
    'tab-add': { label: '新增标签' },
    edit: { label: '编辑标签' },
    'tab-change': { label: '切换' },
  },

  container: true,
  version: '1.0.0',
};
