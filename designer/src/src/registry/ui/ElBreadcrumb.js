import ElBreadcrumbComponent from './ElBreadcrumb.vue';

export default {
  type: 'ElBreadcrumb',
  name: '面包屑',
  category: 'Element 组件',
  icon: 'more',
  thumbnail: null,
  tags: ['breadcrumb', 'nav'],
  description: 'Element Plus 面包屑',
  component: ElBreadcrumbComponent,

  defaultProps: {
    separator: '/',
    separatorClass: '',
    items: ['首页', '列表', '详情'],
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
    separator: { type: 'string', label: '分隔符', group: '组件属性', default: '/' },
    separatorClass: { type: 'string', label: '分隔符类名', group: '组件属性', default: '' },
    items: {
      type: 'string',
      label: '节点',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify(['首页', '列表', '详情'], null, 2),
    },
  },

  eventsSchema: {},
  container: false,
  version: '1.0.0',
};
