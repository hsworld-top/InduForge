import ElPaginationComponent from './ElPagination.vue';

export default {
  type: 'ElPagination',
  name: '分页',
  category: 'Element 组件',
  icon: 'more',
  thumbnail: null,
  tags: ['pagination'],
  description: 'Element Plus 分页',
  component: ElPaginationComponent,

  defaultProps: {
    currentPage: 1,
    pageSize: 10,
    total: 100,
    layout: 'prev, pager, next',
    small: false,
    background: false,
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 360,
    height: 48,
    zIndex: 1,
  },

  propsSchema: {
    currentPage: { type: 'number', label: '当前页', group: '组件属性', default: 1 },
    pageSize: { type: 'number', label: '每页数量', group: '组件属性', default: 10 },
    total: { type: 'number', label: '总数', group: '组件属性', default: 100 },
    layout: { type: 'string', label: '布局', group: '组件属性', default: 'prev, pager, next' },
    small: { type: 'boolean', label: '小型', group: '组件属性', default: false },
    background: { type: 'boolean', label: '背景', group: '组件属性', default: false },
  },

  eventsSchema: {
    'current-change': { label: '页码切换' },
    'size-change': { label: '页大小变化' },
    'prev-click': { label: '上一页点击' },
    'next-click': { label: '下一页点击' },
  },

  container: false,
  version: '1.0.0',
};
