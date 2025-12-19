import ElTreeComponent from './ElTree.vue';

export default {
  type: 'ElTree',
  name: '树形控件',
  category: 'Element 组件',
  icon: 'tree',
  thumbnail: null,
  tags: ['tree', 'data'],
  description: 'Element Plus 树形控件',
  component: ElTreeComponent,

  defaultProps: {
    data: [
      { label: '一级 1', children: [{ label: '二级 1-1' }] },
      { label: '一级 2', children: [{ label: '二级 2-1' }] },
    ],
    treeProps: { children: 'children', label: 'label' },
    showCheckbox: false,
    highlightCurrent: false,
    expandOnClickNode: true,
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 240,
    height: 200,
    zIndex: 1,
  },

  propsSchema: {
    data: {
      type: 'string',
      label: '数据 (JSON)',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify(
        [
          { label: '一级 1', children: [{ label: '二级 1-1' }] },
          { label: '一级 2', children: [{ label: '二级 2-1' }] },
        ],
        null,
        2,
      ),
    },
    treeProps: {
      type: 'string',
      label: '字段映射 (JSON)',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify({ children: 'children', label: 'label' }, null, 2),
    },
    showCheckbox: { type: 'boolean', label: '多选', group: '组件属性', default: false },
    highlightCurrent: { type: 'boolean', label: '高亮当前', group: '组件属性', default: false },
    expandOnClickNode: { type: 'boolean', label: '点击展开', group: '组件属性', default: true },
  },

  eventsSchema: {
    'node-click': { label: '节点点击' },
    check: { label: '勾选变化' },
  },

  container: false,
  version: '1.0.0',
};
